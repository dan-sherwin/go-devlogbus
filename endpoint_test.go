package godevlogbus

import "testing"

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		network    string
		address    string
		socketPath string
	}{
		{
			name:       "absolute unix path",
			input:      "/tmp/devlogbus/devlogbus.sock",
			network:    "unix",
			address:    "/tmp/devlogbus/devlogbus.sock",
			socketPath: "/tmp/devlogbus/devlogbus.sock",
		},
		{
			name:       "unix scheme",
			input:      "unix:/tmp/devlogbus/devlogbus.sock",
			network:    "unix",
			address:    "/tmp/devlogbus/devlogbus.sock",
			socketPath: "/tmp/devlogbus/devlogbus.sock",
		},
		{
			name:    "tcp scheme",
			input:   "tcp://127.0.0.1:7422",
			network: "tcp",
			address: "127.0.0.1:7422",
		},
		{
			name:    "tcp address",
			input:   "prod-debug-host:7422",
			network: "tcp",
			address: "prod-debug-host:7422",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint, err := ParseEndpoint(tt.input)
			if err != nil {
				t.Fatalf("ParseEndpoint returned error: %v", err)
			}
			if endpoint.Network != tt.network {
				t.Fatalf("network = %q, want %q", endpoint.Network, tt.network)
			}
			if endpoint.Address != tt.address {
				t.Fatalf("address = %q, want %q", endpoint.Address, tt.address)
			}
			if endpoint.SocketPath != tt.socketPath {
				t.Fatalf("socket path = %q, want %q", endpoint.SocketPath, tt.socketPath)
			}
		})
	}
}

func TestParseEndpointRejectsTCPWithoutPort(t *testing.T) {
	if _, err := ParseEndpoint("tcp://127.0.0.1"); err == nil {
		t.Fatal("expected tcp endpoint without port to fail")
	}
}
