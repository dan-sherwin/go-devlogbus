package godevlogbus

import "testing"

func TestDefaultEndpointUsesStableDevLogBusSocket(t *testing.T) {
	if got := defaultEndpoint(); got != "/tmp/devlogbus/devlogbus.sock" {
		t.Fatalf("defaultEndpoint = %q, want stable DevLogBus socket", got)
	}
}

func TestParseEndpointDefaultsToStableSocket(t *testing.T) {
	endpoint, err := parseEndpoint("")
	if err != nil {
		t.Fatalf("parseEndpoint returned error: %v", err)
	}
	if endpoint.Network != networkUnix {
		t.Fatalf("network = %q, want %q", endpoint.Network, networkUnix)
	}
	if endpoint.Address != "/tmp/devlogbus/devlogbus.sock" {
		t.Fatalf("address = %q, want stable DevLogBus socket", endpoint.Address)
	}
}

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		network string
		address string
		display string
	}{
		{
			name:    "absolute unix path",
			input:   "/tmp/devlogbus/devlogbus.sock",
			network: networkUnix,
			address: "/tmp/devlogbus/devlogbus.sock",
			display: "unix:/tmp/devlogbus/devlogbus.sock",
		},
		{
			name:    "unix scheme",
			input:   "unix:/tmp/devlogbus/devlogbus.sock",
			network: networkUnix,
			address: "/tmp/devlogbus/devlogbus.sock",
			display: "unix:/tmp/devlogbus/devlogbus.sock",
		},
		{
			name:    "tcp scheme",
			input:   "tcp://127.0.0.1:7422",
			network: networkTCP,
			address: "127.0.0.1:7422",
			display: "tcp://127.0.0.1:7422",
		},
		{
			name:    "tcp address",
			input:   "prod-debug-host:7422",
			network: networkTCP,
			address: "prod-debug-host:7422",
			display: "tcp://prod-debug-host:7422",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint, err := parseEndpoint(tt.input)
			if err != nil {
				t.Fatalf("parseEndpoint returned error: %v", err)
			}
			if endpoint.Network != tt.network {
				t.Fatalf("network = %q, want %q", endpoint.Network, tt.network)
			}
			if endpoint.Address != tt.address {
				t.Fatalf("address = %q, want %q", endpoint.Address, tt.address)
			}
			if endpoint.String() != tt.display {
				t.Fatalf("display = %q, want %q", endpoint.String(), tt.display)
			}
		})
	}
}

func TestParseEndpointRejectsTCPWithoutPort(t *testing.T) {
	if _, err := parseEndpoint("tcp://127.0.0.1"); err == nil {
		t.Fatal("expected tcp endpoint without port to fail")
	}
}
