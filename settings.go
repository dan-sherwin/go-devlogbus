package godevlogbus

import (
	"strconv"
	"strings"
	"sync"

	app_settings "github.com/dan-sherwin/go-app-settings"
)

type Settings struct {
	mu               sync.RWMutex
	Enabled          bool
	Endpoint         string
	LegacySocketPath string
	handler          *Handler
	registered       bool
}

type SettingsRegistrationOptions struct {
	RegisterLegacySocketPath bool
}

func NewSettings() *Settings {
	defaultEndpoint := DefaultEndpoint()
	return &Settings{
		Enabled:          DefaultEnabled(),
		Endpoint:         defaultEndpoint,
		LegacySocketPath: defaultEndpoint,
	}
}

func (s *Settings) Register() {
	s.RegisterWithOptions(SettingsRegistrationOptions{RegisterLegacySocketPath: true})
}

func (s *Settings) RegisterWithOptions(options SettingsRegistrationOptions) {
	s.mu.Lock()
	if s.registered {
		s.mu.Unlock()
		return
	}
	s.registered = true
	s.mu.Unlock()

	app_settings.RegisterSetting(&app_settings.Setting{
		Name:        SettingEnabled,
		Description: "Publish logs to DevLogBus",
		GetFunc: func() string {
			return strconv.FormatBool(s.Config().Enabled)
		},
		SetFunc: func(value string) error {
			enabled, err := strconv.ParseBool(value)
			if err != nil {
				return err
			}
			return s.SetEnabled(enabled)
		},
	})
	app_settings.RegisterSetting(&app_settings.Setting{
		Name:        SettingEndpoint,
		Description: "DevLogBus endpoint; accepts a Unix socket path, unix:/path.sock, tcp://host:port, or host:port",
		GetFunc: func() string {
			return s.Config().Endpoint
		},
		SetFunc: func(value string) error {
			return s.SetEndpoint(value)
		},
	})
	if !options.RegisterLegacySocketPath {
		return
	}
	app_settings.RegisterSetting(&app_settings.Setting{
		Name:        LegacySettingSocketPath,
		Description: "Deprecated DevLogBus Unix socket path setting; use devlogbus_endpoint",
		GetFunc: func() string {
			s.mu.RLock()
			defer s.mu.RUnlock()
			return s.LegacySocketPath
		},
		SetFunc: func(value string) error {
			return s.SetLegacySocketPath(value)
		},
	})
}

func (s *Settings) Bind(handler *Handler) error {
	s.mu.Lock()
	s.handler = handler
	config := s.configLocked()
	s.mu.Unlock()
	if handler == nil {
		return nil
	}
	return handler.Configure(config)
}

func (s *Settings) NewHandler(options Options) *Handler {
	config := s.Config()
	options.Enabled = config.Enabled
	options.Endpoint = config.Endpoint
	handler := New(options)
	_ = s.Bind(handler)
	return handler
}

func (s *Settings) Config() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configLocked()
}

func (s *Settings) Status() Status {
	s.mu.RLock()
	handler := s.handler
	s.mu.RUnlock()
	if handler != nil {
		return handler.Status()
	}
	config := s.Config()
	endpoint, err := ParseEndpoint(config.Endpoint)
	status := Status{Enabled: config.Enabled, Endpoint: config.Endpoint}
	if err != nil {
		status.LastError = err.Error()
		return status
	}
	status.Network = endpoint.Network
	status.Address = endpoint.Address
	status.SocketPath = endpoint.SocketPath
	return status
}

func (s *Settings) Configure(config Config) error {
	if strings.TrimSpace(config.Endpoint) == "" {
		config.Endpoint = DefaultEndpoint()
	}
	if _, err := ParseEndpoint(config.Endpoint); err != nil {
		return err
	}

	s.mu.Lock()
	s.Enabled = config.Enabled
	s.Endpoint = strings.TrimSpace(config.Endpoint)
	s.LegacySocketPath = s.Endpoint
	handler := s.handler
	s.mu.Unlock()

	if handler != nil {
		return handler.Configure(config)
	}
	return nil
}

func (s *Settings) SetEnabled(enabled bool) error {
	config := s.Config()
	config.Enabled = enabled
	return s.Configure(config)
}

func (s *Settings) SetEndpoint(endpoint string) error {
	config := s.Config()
	config.Endpoint = endpoint
	return s.Configure(config)
}

func (s *Settings) SetLegacySocketPath(socketPath string) error {
	s.mu.Lock()
	s.LegacySocketPath = strings.TrimSpace(socketPath)
	s.mu.Unlock()
	return s.SetEndpoint(socketPath)
}

func (s *Settings) configLocked() Config {
	endpoint := strings.TrimSpace(s.Endpoint)
	if endpoint == "" {
		endpoint = strings.TrimSpace(s.LegacySocketPath)
	}
	if endpoint == "" {
		endpoint = DefaultEndpoint()
	}
	return Config{Enabled: s.Enabled, Endpoint: endpoint}
}
