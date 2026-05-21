package godevlogbus

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestRuntimeWithHandlerReusesBoundHandler(t *testing.T) {
	resetRuntimeForTest(t)
	if err := defaultRuntime.settings.configure(config{Enabled: false, Endpoint: "127.0.0.1:7422"}); err != nil {
		t.Fatalf("configure settings: %v", err)
	}
	Setup(SetupOptions{Source: "runtime-test"})
	handler := defaultRuntime.handlerForLevel(slog.LevelDebug)
	if handler == nil {
		t.Fatal("expected handler")
	}
	status := handler.status()
	if status.Source != "runtime-test" {
		t.Fatalf("source = %q, want runtime-test", status.Source)
	}
	if status.Endpoint != "tcp://127.0.0.1:7422" {
		t.Fatalf("unexpected endpoint status: %#v", status)
	}

	handlers := WithHandler([]slog.Handler{}, slog.LevelInfo)
	if len(handlers) != 1 {
		t.Fatalf("handlers len = %d, want 1", len(handlers))
	}
	if got := defaultRuntime.handlerForLevel(slog.LevelWarn); got != handler {
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
				Endpoint:   "tcp://127.0.0.1:7422",
				Source:     "runtime-test",
				Generation: 2,
			}
			return nil
		},
	})

	if err := (&statusCommand{}).Run(); err != nil {
		t.Fatalf("status command: %v", err)
	}
	if calledMethod != defaultRPCName+".Status" {
		t.Fatalf("called method = %q", calledMethod)
	}
	if !strings.Contains(output.String(), "Endpoint:   tcp://127.0.0.1:7422") {
		t.Fatalf("unexpected output: %s", output.String())
	}
	if strings.Contains(output.String(), "Network:") || strings.Contains(output.String(), "Address:") || strings.Contains(output.String(), "Socket Path:") {
		t.Fatalf("status output leaked transport internals: %s", output.String())
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
