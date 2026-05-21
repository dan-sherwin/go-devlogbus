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
	callRPC        CallRPCFunc
	settings       *settings
	handler        *handler
	output         io.Writer
	queueSize      int
	publishTimeout time.Duration
	setup          bool
}

var defaultRuntime = newRuntimeState()

func Setup(options SetupOptions) {
	defaultRuntime.setupRuntime(options)
}

func WithHandler(handlers []slog.Handler, level slog.Leveler) []slog.Handler {
	return append(handlers, defaultRuntime.handlerForLevel(level))
}

func currentStatus() (Status, error) {
	var status Status
	if err := defaultRuntime.call("Status", EmptyArgs{}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func enable(endpoint string) (Status, error) {
	if strings.TrimSpace(endpoint) != "" {
		return configure(config{Enabled: true, Endpoint: endpoint})
	}
	var status Status
	if err := defaultRuntime.call("Enable", EmptyArgs{}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func disable() (Status, error) {
	var status Status
	if err := defaultRuntime.call("Disable", EmptyArgs{}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func setEndpoint(endpoint string) (Status, error) {
	var status Status
	if err := defaultRuntime.call("SetEndpoint", EndpointArgs{Endpoint: endpoint}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func configure(config config) (Status, error) {
	var status Status
	if err := defaultRuntime.call("Configure", ConfigureArgs{Enabled: config.Enabled, Endpoint: config.Endpoint}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func newRuntimeState() *runtimeState {
	return &runtimeState{
		source:   "unknown",
		rpcName:  defaultRPCName,
		settings: newSettings(),
		output:   os.Stdout,
	}
}

func (s *runtimeState) setupRuntime(options SetupOptions) {
	source := strings.TrimSpace(options.Source)
	if source == "" {
		source = "unknown"
	}
	rpcName := strings.TrimSpace(options.RPCName)
	if rpcName == "" {
		rpcName = defaultRPCName
	}
	output := options.Output
	if output == nil {
		output = os.Stdout
	}

	s.mu.Lock()
	s.source = source
	s.rpcName = rpcName
	s.callRPC = options.CallRPC
	s.output = output
	s.queueSize = options.QueueSize
	s.publishTimeout = options.PublishTimeout
	setup := s.setup
	settings := s.settings
	s.setup = true
	s.mu.Unlock()

	if setup {
		return
	}

	settings.register()
	if options.RegisterRPC != nil {
		options.RegisterRPC(rpcName, newRPCReceiver(settings, !options.DisableRPCPersistence))
	}
}

func (s *runtimeState) handlerForLevel(level slog.Leveler) *handler {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handler == nil {
		s.handler = s.settings.newHandler(handlerOptions{
			Source:         s.source,
			Level:          level,
			QueueSize:      s.queueSize,
			PublishTimeout: s.publishTimeout,
		})
		return s.handler
	}
	s.handler.setLevel(level)
	_ = s.settings.bind(s.handler)
	return s.handler
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
		rpcName = defaultRPCName
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
