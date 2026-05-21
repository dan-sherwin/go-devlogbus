package godevlogbus

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestRuntimeWithHandlerReusesBoundHandler(t *testing.T) {
	resetRuntimeForTest(t)
	if err := RuntimeSettings().Configure(Config{Enabled: false, Endpoint: "127.0.0.1:7422"}); err != nil {
		t.Fatalf("configure settings: %v", err)
	}
	Setup(SetupOptions{Source: "runtime-test"})
	handler := RuntimeHandler(slog.LevelDebug)
	if handler == nil {
		t.Fatal("expected handler")
	}
	status := handler.Status()
	if status.Source != "runtime-test" {
		t.Fatalf("source = %q, want runtime-test", status.Source)
	}
	if status.Network != NetworkTCP || status.Address != "127.0.0.1:7422" {
		t.Fatalf("unexpected endpoint status: %#v", status)
	}

	handlers := WithHandler([]slog.Handler{}, slog.LevelInfo)
	if len(handlers) != 1 {
		t.Fatalf("handlers len = %d, want 1", len(handlers))
	}
	if got := RuntimeHandler(slog.LevelWarn); got != handler {
		t.Fatal("expected runtime to reuse handler")
	}
}

func TestRuntimeCommandUsesConfiguredRPCCaller(t *testing.T) {
	resetRuntimeForTest(t)
	var output bytes.Buffer
	var calledMethod string
	Setup(SetupOptions{
		Source: "runtime-test",
		Output: &output,
		CallRPC: func(serviceMethod string, args any, reply any) error {
			calledMethod = serviceMethod
			status := reply.(*Status)
			*status = Status{
				Enabled:    true,
				Endpoint:   "127.0.0.1:7422",
				Network:    NetworkTCP,
				Address:    "127.0.0.1:7422",
				Source:     "runtime-test",
				Generation: 2,
			}
			return nil
		},
	})

	if err := (&StatusCommand{}).Run(); err != nil {
		t.Fatalf("status command: %v", err)
	}
	if calledMethod != DefaultRPCName+".Status" {
		t.Fatalf("called method = %q", calledMethod)
	}
	if !strings.Contains(output.String(), "Endpoint:        127.0.0.1:7422") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

func resetRuntimeForTest(t *testing.T) {
	t.Helper()
	previous := defaultRuntime
	defaultRuntime = newRuntimeState()
	t.Cleanup(func() {
		defaultRuntime = previous
	})
}
