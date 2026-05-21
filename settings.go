package godevlogbus

import (
	"strconv"
	"strings"
	"sync"

	app_settings "github.com/dan-sherwin/go-app-settings"
)

type settings struct {
	mu         sync.RWMutex
	Enabled    bool
	Endpoint   string
	handler    *handler
	registered bool
}

func newSettings() *settings {
	return &settings{
		Enabled:  defaultEnabled(),
		Endpoint: defaultEndpoint(),
	}
}

func (s *settings) register() {
	s.mu.Lock()
	if s.registered {
		s.mu.Unlock()
		return
	}
	s.registered = true
	s.mu.Unlock()

	app_settings.RegisterSetting(&app_settings.Setting{
		Name:        settingEnabled,
		Description: "Publish logs to DevLogBus",
		GetFunc: func() string {
			return strconv.FormatBool(s.config().Enabled)
		},
		SetFunc: func(value string) error {
			enabled, err := strconv.ParseBool(value)
			if err != nil {
				return err
			}
			return s.setEnabled(enabled)
		},
	})
	app_settings.RegisterSetting(&app_settings.Setting{
		Name:        settingEndpoint,
		Description: "DevLogBus endpoint; accepts a Unix socket path, unix:/path.sock, tcp://host:port, or host:port",
		GetFunc: func() string {
			return s.config().Endpoint
		},
		SetFunc: func(value string) error {
			return s.setEndpoint(value)
		},
	})
}

func (s *settings) bind(handler *handler) error {
	s.mu.Lock()
	s.handler = handler
	config := s.configLocked()
	s.mu.Unlock()
	if handler == nil {
		return nil
	}
	return handler.configure(config)
}

func (s *settings) newHandler(options handlerOptions) *handler {
	config := s.config()
	options.Enabled = config.Enabled
	options.Endpoint = config.Endpoint
	handler := newHandler(options)
	_ = s.bind(handler)
	return handler
}

func (s *settings) config() config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configLocked()
}

func (s *settings) status() Status {
	s.mu.RLock()
	handler := s.handler
	s.mu.RUnlock()
	if handler != nil {
		return handler.status()
	}
	config := s.config()
	endpoint, err := parseEndpoint(config.Endpoint)
	status := Status{Enabled: config.Enabled}
	if err != nil {
		status.Endpoint = config.Endpoint
		status.LastError = err.Error()
		return status
	}
	status.Endpoint = endpoint.String()
	return status
}

func (s *settings) configure(config config) error {
	if strings.TrimSpace(config.Endpoint) == "" {
		config.Endpoint = defaultEndpoint()
	}
	if _, err := parseEndpoint(config.Endpoint); err != nil {
		return err
	}

	s.mu.Lock()
	s.Enabled = config.Enabled
	s.Endpoint = strings.TrimSpace(config.Endpoint)
	handler := s.handler
	s.mu.Unlock()

	if handler != nil {
		return handler.configure(config)
	}
	return nil
}

func (s *settings) setEnabled(enabled bool) error {
	config := s.config()
	config.Enabled = enabled
	return s.configure(config)
}

func (s *settings) setEndpoint(endpoint string) error {
	config := s.config()
	config.Endpoint = endpoint
	return s.configure(config)
}

func (s *settings) configLocked() config {
	endpoint := strings.TrimSpace(s.Endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint()
	}
	return config{Enabled: s.Enabled, Endpoint: endpoint}
}
