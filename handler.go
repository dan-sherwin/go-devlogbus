package godevlogbus

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/dan-sherwin/devlogbus/pkg/protocol"
)

type Options struct {
	Source         string
	Level          slog.Leveler
	Enabled        bool
	Endpoint       string
	QueueSize      int
	PublishTimeout time.Duration
}

type Handler struct {
	state  *handlerState
	attrs  map[string]any
	groups []string
}

type handlerState struct {
	mu             sync.RWMutex
	source         string
	level          slog.Leveler
	enabled        bool
	endpoint       Endpoint
	queueSize      int
	publishTimeout time.Duration
	sink           *sink
	generation     uint64
	lastError      string
}

func New(options Options) *Handler {
	if strings.TrimSpace(options.Source) == "" {
		options.Source = "unknown"
	}
	if options.Level == nil {
		options.Level = slog.LevelDebug
	}
	if options.QueueSize <= 0 {
		options.QueueSize = 256
	}
	if options.PublishTimeout <= 0 {
		options.PublishTimeout = 250 * time.Millisecond
	}

	state := &handlerState{
		source:         strings.TrimSpace(options.Source),
		level:          options.Level,
		queueSize:      options.QueueSize,
		publishTimeout: options.PublishTimeout,
	}
	handler := &Handler{state: state, attrs: map[string]any{}}
	if err := handler.Configure(Config{Enabled: options.Enabled, Endpoint: options.Endpoint}); err != nil {
		state.setError(err)
	}
	return handler
}

func (h *Handler) Configure(config Config) error {
	endpoint, err := ParseEndpoint(config.Endpoint)
	if err != nil {
		h.state.setError(err)
		return err
	}
	h.state.configure(config.Enabled, endpoint)
	return nil
}

func (h *Handler) SetLevel(level slog.Leveler) {
	if level == nil {
		level = slog.LevelDebug
	}
	h.state.mu.Lock()
	defer h.state.mu.Unlock()
	h.state.level = level
}

func (h *Handler) Close() {
	h.state.configure(false, h.state.currentEndpoint())
}

func (h *Handler) Status() Status {
	return h.state.status()
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	enabled, currentLevel, currentSink := h.state.enabledSnapshot()
	return enabled && currentSink != nil && level >= currentLevel.Level()
}

func (h *Handler) Handle(_ context.Context, record slog.Record) error {
	snapshot := h.state.handleSnapshot(record.Level)
	if !snapshot.enabled || snapshot.sink == nil {
		return nil
	}

	out := protocol.Record{
		Time:    record.Time,
		Level:   record.Level.String(),
		Source:  snapshot.source,
		Message: record.Message,
		Attrs:   copyAttrs(h.attrs),
	}
	record.Attrs(func(attr slog.Attr) bool {
		addAttr(out.Attrs, h.groups, attr)
		return true
	})
	if len(out.Attrs) == 0 {
		out.Attrs = nil
	}
	snapshot.sink.enqueue(out)
	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = copyAttrs(h.attrs)
	for _, attr := range attrs {
		addAttr(clone.attrs, h.groups, attr)
	}
	return &clone
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	clone.groups = append(append([]string{}, h.groups...), name)
	return &clone
}

type handleSnapshot struct {
	enabled bool
	source  string
	sink    *sink
}

func (s *handlerState) configure(enabled bool, endpoint Endpoint) {
	s.mu.Lock()

	oldSink := s.sink
	reuseSink := enabled && oldSink != nil && s.endpoint.Network == endpoint.Network && s.endpoint.Address == endpoint.Address
	if reuseSink {
		s.enabled = true
		s.endpoint = endpoint
		s.lastError = ""
		s.mu.Unlock()
		return
	}

	var nextSink *sink
	if enabled {
		nextSink = newSink(endpoint, s.queueSize, s.publishTimeout, s.setError, s.clearError)
		nextSink.start()
	}
	s.enabled = enabled
	s.endpoint = endpoint
	s.sink = nextSink
	s.generation++
	s.lastError = ""
	s.mu.Unlock()

	if oldSink != nil {
		oldSink.close()
	}
}

func (s *handlerState) currentEndpoint() Endpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.endpoint.Raw == "" {
		endpoint, _ := ParseEndpoint("")
		return endpoint
	}
	return s.endpoint
}

func (s *handlerState) enabledSnapshot() (bool, slog.Leveler, *sink) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled, s.level, s.sink
}

func (s *handlerState) handleSnapshot(level slog.Level) handleSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.enabled || s.sink == nil || level < s.level.Level() {
		return handleSnapshot{}
	}
	return handleSnapshot{enabled: true, source: s.source, sink: s.sink}
}

func (s *handlerState) status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Status{
		Enabled:        s.enabled,
		Endpoint:       s.endpoint.Raw,
		Network:        s.endpoint.Network,
		Address:        s.endpoint.Address,
		SocketPath:     s.endpoint.SocketPath,
		Source:         s.source,
		QueueSize:      s.queueSize,
		PublishTimeout: s.publishTimeout.String(),
		Generation:     s.generation,
		LastError:      s.lastError,
	}
}

func (s *handlerState) setError(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = err.Error()
}

func (s *handlerState) clearError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = ""
}
