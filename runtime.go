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

type RuntimeOptions struct {
	Source                string
	RPCName               string
	RegisterRPC           RegisterRPCFunc
	CallRPC               CallRPCFunc
	Settings              *Settings
	Output                io.Writer
	QueueSize             int
	PublishTimeout        time.Duration
	DisableRPCPersistence bool
}

type Runtime struct {
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
	registered     bool
}

var (
	defaultRuntimeMu sync.RWMutex
	defaultRuntime   *Runtime
)

func NewRuntime(options RuntimeOptions) *Runtime {
	source := strings.TrimSpace(options.Source)
	if source == "" {
		source = "unknown"
	}
	rpcName := strings.TrimSpace(options.RPCName)
	if rpcName == "" {
		rpcName = DefaultRPCName
	}
	settings := options.Settings
	if settings == nil {
		settings = NewSettings()
	}
	output := options.Output
	if output == nil {
		output = os.Stdout
	}
	runtime := &Runtime{
		source:         source,
		rpcName:        rpcName,
		registerRPC:    options.RegisterRPC,
		callRPC:        options.CallRPC,
		settings:       settings,
		output:         output,
		queueSize:      options.QueueSize,
		publishTimeout: options.PublishTimeout,
		persistRPC:     !options.DisableRPCPersistence,
	}
	SetDefaultRuntime(runtime)
	return runtime
}

func SetDefaultRuntime(runtime *Runtime) {
	defaultRuntimeMu.Lock()
	defer defaultRuntimeMu.Unlock()
	defaultRuntime = runtime
}

func DefaultRuntime() *Runtime {
	defaultRuntimeMu.RLock()
	defer defaultRuntimeMu.RUnlock()
	return defaultRuntime
}

func (r *Runtime) Register() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.registered {
		r.mu.Unlock()
		return
	}
	r.registered = true
	settings := r.settings
	rpcName := r.rpcName
	registerRPC := r.registerRPC
	persistRPC := r.persistRPC
	r.mu.Unlock()

	settings.Register()
	if registerRPC != nil {
		receiver := NewRPCReceiver(settings, nil)
		receiver.Persist = persistRPC
		registerRPC(rpcName, receiver)
	}
	SetDefaultRuntime(r)
}

func (r *Runtime) WithHandler(handlers []slog.Handler, level slog.Leveler) []slog.Handler {
	return append(handlers, r.Handler(level))
}

func (r *Runtime) Handler(level slog.Leveler) *Handler {
	if r == nil {
		return New(Options{Level: level})
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.handler == nil {
		r.handler = r.settings.NewHandler(Options{
			Source:         r.source,
			Level:          level,
			QueueSize:      r.queueSize,
			PublishTimeout: r.publishTimeout,
		})
		return r.handler
	}
	r.handler.SetLevel(level)
	_ = r.settings.Bind(r.handler)
	return r.handler
}

func (r *Runtime) Settings() *Settings {
	if r == nil {
		return nil
	}
	return r.settings
}

func (r *Runtime) RPCName() string {
	if r == nil || strings.TrimSpace(r.rpcName) == "" {
		return DefaultRPCName
	}
	return r.rpcName
}

func (r *Runtime) Status() (Status, error) {
	var status Status
	if err := r.call("Status", EmptyArgs{}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func (r *Runtime) Enable(endpoint string) (Status, error) {
	if strings.TrimSpace(endpoint) != "" {
		return r.Configure(Config{Enabled: true, Endpoint: endpoint})
	}
	var status Status
	if err := r.call("Enable", EmptyArgs{}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func (r *Runtime) Disable() (Status, error) {
	var status Status
	if err := r.call("Disable", EmptyArgs{}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func (r *Runtime) SetEndpoint(endpoint string) (Status, error) {
	var status Status
	if err := r.call("SetEndpoint", EndpointArgs{Endpoint: endpoint}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func (r *Runtime) Configure(config Config) (Status, error) {
	var status Status
	if err := r.call("Configure", ConfigureArgs{Enabled: config.Enabled, Endpoint: config.Endpoint}, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func (r *Runtime) call(method string, args any, reply any) error {
	if r == nil {
		return errors.New("devlogbus runtime is nil")
	}
	if r.callRPC == nil {
		return errors.New("devlogbus runtime RPC caller is not configured")
	}
	return r.callRPC(r.RPCName()+"."+method, args, reply)
}

func (r *Runtime) writer() io.Writer {
	if r == nil || r.output == nil {
		return os.Stdout
	}
	return r.output
}
