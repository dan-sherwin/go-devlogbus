package godevlogbus

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

type RegisterRPCFunc func(name string, receiver any)
type CallRPCFunc func(serviceMethod string, args any, reply any) error

type SetupOptions struct {
	Source                string
	RPCName               string
	RegisterRPC           RegisterRPCFunc
	CallRPC               CallRPCFunc
	Output                io.Writer
	QueueSize             int
	PublishTimeout        time.Duration
	DisableRPCPersistence bool
}

type runtimeState struct {
	mu             sync.Mutex
	source         string
	rpcName        string
	registerRPC    RegisterRPCFunc
	callRPC        CallRPCFunc
	settings       *Settings
	handler        *Handler
	output         io.Writer
	queueSize      int
	publishTimeout time.Duration
	persistRPC     bool
	setup          bool
}

var defaultRuntime = newRuntimeState()

func Setup(options SetupOptions) {
	defaultRuntime.setupRuntime(options)
}

func WithHandler(handlers []slog.Handler, level slog.Leveler) []slog.Handler {
	return append(handlers, RuntimeHandler(level))
}

func RuntimeHandler(level slog.Leveler) *Handler {
	return defaultRuntime.handlerForLevel(level)
}

func RuntimeSettings() *Settings {
	return defaultRuntime.currentSettings()
}

func RPCName() string {
	return defaultRuntime.currentRPCName()
}

func CurrentStatus() (Status, error) {
	var status Status
	if err := defaultRuntime.call("Status", EmptyArgs{}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func Enable(endpoint string) (Status, error) {
	if strings.TrimSpace(endpoint) != "" {
		return Configure(Config{Enabled: true, Endpoint: endpoint})
	}
	var status Status
	if err := defaultRuntime.call("Enable", EmptyArgs{}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func Disable() (Status, error) {
	var status Status
	if err := defaultRuntime.call("Disable", EmptyArgs{}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func SetEndpoint(endpoint string) (Status, error) {
	var status Status
	if err := defaultRuntime.call("SetEndpoint", EndpointArgs{Endpoint: endpoint}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func Configure(config Config) (Status, error) {
	var status Status
	if err := defaultRuntime.call("Configure", ConfigureArgs{Enabled: config.Enabled, Endpoint: config.Endpoint}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func newRuntimeState() *runtimeState {
	return &runtimeState{
		source:     "unknown",
		rpcName:    DefaultRPCName,
		settings:   NewSettings(),
		output:     os.Stdout,
		persistRPC: true,
	}
}

func (s *runtimeState) setupRuntime(options SetupOptions) {
	source := strings.TrimSpace(options.Source)
	if source == "" {
		source = "unknown"
	}
	rpcName := strings.TrimSpace(options.RPCName)
	if rpcName == "" {
		rpcName = DefaultRPCName
	}
	output := options.Output
	if output == nil {
		output = os.Stdout
	}

	s.mu.Lock()
	s.source = source
	s.rpcName = rpcName
	s.registerRPC = options.RegisterRPC
	s.callRPC = options.CallRPC
	s.output = output
	s.queueSize = options.QueueSize
	s.publishTimeout = options.PublishTimeout
	s.persistRPC = !options.DisableRPCPersistence
	setup := s.setup
	settings := s.settings
	registerRPC := s.registerRPC
	persistRPC := s.persistRPC
	s.setup = true
	s.mu.Unlock()

	if setup {
		return
	}

	settings.Register()
	if registerRPC != nil {
		receiver := NewRPCReceiver(settings, nil)
		receiver.Persist = persistRPC
		registerRPC(rpcName, receiver)
	}
}

func (s *runtimeState) handlerForLevel(level slog.Leveler) *Handler {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handler == nil {
		s.handler = s.settings.NewHandler(Options{
			Source:         s.source,
			Level:          level,
			QueueSize:      s.queueSize,
			PublishTimeout: s.publishTimeout,
		})
		return s.handler
	}
	s.handler.SetLevel(level)
	_ = s.settings.Bind(s.handler)
	return s.handler
}

func (s *runtimeState) currentSettings() *Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

func (s *runtimeState) currentRPCName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(s.rpcName) == "" {
		return DefaultRPCName
	}
	return s.rpcName
}

func (s *runtimeState) call(method string, args any, reply any) error {
	s.mu.Lock()
	callRPC := s.callRPC
	rpcName := s.rpcName
	s.mu.Unlock()

	if callRPC == nil {
		return errors.New("devlogbus RPC caller is not configured")
	}
	if strings.TrimSpace(rpcName) == "" {
		rpcName = DefaultRPCName
	}
	return callRPC(rpcName+"."+method, args, reply)
}

func runtimeWriter() io.Writer {
	defaultRuntime.mu.Lock()
	defer defaultRuntime.mu.Unlock()
	if defaultRuntime.output == nil {
		return os.Stdout
	}
	return defaultRuntime.output
}
