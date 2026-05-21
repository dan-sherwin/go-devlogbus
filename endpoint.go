package godevlogbus

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"

	devlogbusclient "github.com/dan-sherwin/devlogbus/pkg/client"
)

const (
	NetworkUnix             = "unix"
	NetworkTCP              = "tcp"
	SettingEnabled          = "devlogbus_enabled"
	SettingEndpoint         = "devlogbus_endpoint"
	LegacySettingSocketPath = "devlogbus_socket_path"
	DefaultRPCName          = "DevLogBus"
)

type Endpoint struct {
	Raw        string
	Network    string
	Address    string
	SocketPath string
}

func DefaultEnabled() bool {
	return runtime.GOOS == "darwin"
}

func DefaultEndpoint() string {
	return devlogbusclient.DefaultSocketPath()
}

func ParseEndpoint(raw string) (Endpoint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = DefaultEndpoint()
	}

	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "unix://"):
		address := strings.TrimPrefix(raw, raw[:len("unix://")])
		if address == "" {
			return Endpoint{}, fmt.Errorf("unix devlogbus endpoint requires a socket path")
		}
		return unixEndpoint(raw, address), nil
	case strings.HasPrefix(lower, "unix:"):
		address := strings.TrimPrefix(raw, raw[:len("unix:")])
		if address == "" {
			return Endpoint{}, fmt.Errorf("unix devlogbus endpoint requires a socket path")
		}
		return unixEndpoint(raw, address), nil
	case strings.HasPrefix(lower, "tcp://"):
		parsed, err := url.Parse(raw)
		if err != nil {
			return Endpoint{}, fmt.Errorf("parse tcp devlogbus endpoint: %w", err)
		}
		if parsed.Host == "" {
			return Endpoint{}, fmt.Errorf("tcp devlogbus endpoint requires host:port")
		}
		return tcpEndpoint(raw, parsed.Host)
	case strings.HasPrefix(lower, "tcp:"):
		address := strings.TrimPrefix(raw, raw[:len("tcp:")])
		if address == "" {
			return Endpoint{}, fmt.Errorf("tcp devlogbus endpoint requires host:port")
		}
		return tcpEndpoint(raw, address)
	}

	if looksLikeTCPAddress(raw) {
		return tcpEndpoint(raw, raw)
	}
	return unixEndpoint(raw, raw), nil
}

func unixEndpoint(raw, address string) Endpoint {
	address = filepath.Clean(address)
	return Endpoint{
		Raw:        raw,
		Network:    NetworkUnix,
		Address:    address,
		SocketPath: address,
	}
}

func tcpEndpoint(raw, address string) (Endpoint, error) {
	if _, _, err := net.SplitHostPort(address); err != nil {
		return Endpoint{}, fmt.Errorf("tcp devlogbus endpoint must be host:port: %w", err)
	}
	return Endpoint{
		Raw:     raw,
		Network: NetworkTCP,
		Address: address,
	}, nil
}

func looksLikeTCPAddress(raw string) bool {
	if _, _, err := net.SplitHostPort(raw); err == nil {
		return true
	}
	return false
}
