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
	networkUnix     = "unix"
	networkTCP      = "tcp"
	settingEnabled  = "devlogbus_enabled"
	settingEndpoint = "devlogbus_endpoint"
	defaultRPCName  = "DevLogBus"
)

type endpoint struct {
	Network string
	Address string
}

func defaultEnabled() bool {
	return runtime.GOOS == "darwin"
}

func defaultEndpoint() string {
	return devlogbusclient.DefaultSocketPath()
}

func parseEndpoint(raw string) (endpoint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = defaultEndpoint()
	}

	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "unix://"):
		address := raw[len("unix://"):]
		if address == "" {
			return endpoint{}, fmt.Errorf("unix devlogbus endpoint requires a socket path")
		}
		return unixEndpoint(address), nil
	case strings.HasPrefix(lower, "unix:"):
		address := raw[len("unix:"):]
		if address == "" {
			return endpoint{}, fmt.Errorf("unix devlogbus endpoint requires a socket path")
		}
		return unixEndpoint(address), nil
	case strings.HasPrefix(lower, "tcp://"):
		parsed, err := url.Parse(raw)
		if err != nil {
			return endpoint{}, fmt.Errorf("parse tcp devlogbus endpoint: %w", err)
		}
		if parsed.Host == "" {
			return endpoint{}, fmt.Errorf("tcp devlogbus endpoint requires host:port")
		}
		return tcpEndpoint(parsed.Host)
	case strings.HasPrefix(lower, "tcp:"):
		address := raw[len("tcp:"):]
		if address == "" {
			return endpoint{}, fmt.Errorf("tcp devlogbus endpoint requires host:port")
		}
		return tcpEndpoint(address)
	}

	if looksLikeTCPAddress(raw) {
		return tcpEndpoint(raw)
	}
	return unixEndpoint(raw), nil
}

func unixEndpoint(address string) endpoint {
	address = filepath.Clean(address)
	return endpoint{
		Network: networkUnix,
		Address: address,
	}
}

func tcpEndpoint(address string) (endpoint, error) {
	if _, _, err := net.SplitHostPort(address); err != nil {
		return endpoint{}, fmt.Errorf("tcp devlogbus endpoint must be host:port: %w", err)
	}
	return endpoint{
		Network: networkTCP,
		Address: address,
	}, nil
}

func looksLikeTCPAddress(raw string) bool {
	if _, _, err := net.SplitHostPort(raw); err == nil {
		return true
	}
	return false
}

func (e endpoint) String() string {
	switch e.Network {
	case networkUnix:
		if e.Address == "" {
			return ""
		}
		return "unix:" + e.Address
	case networkTCP:
		if e.Address == "" {
			return ""
		}
		return "tcp://" + e.Address
	default:
		return e.Address
	}
}
