package godevlogbus

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/dan-sherwin/devlogbus/pkg/protocol"
)

func TestHandlerDisabledDoesNotPublish(t *testing.T) {
	listener, records := startTCPBroker(t)
	defer listener.Close()

	handler := New(Options{
		Source:         "test-service",
		Level:          slog.LevelDebug,
		Enabled:        false,
		Endpoint:       listener.Addr().String(),
		PublishTimeout: 25 * time.Millisecond,
	})
	defer handler.Close()

	logger := slog.New(handler)
	logger.Info("disabled message")

	select {
	case record := <-records:
		t.Fatalf("unexpected record while disabled: %#v", record)
	case <-time.After(75 * time.Millisecond):
	}
}

func TestHandlerCanEnableAndSwitchEndpoint(t *testing.T) {
	firstListener, firstRecords := startTCPBroker(t)
	defer firstListener.Close()
	secondListener, secondRecords := startTCPBroker(t)
	defer secondListener.Close()

	handler := New(Options{
		Source:         "test-service",
		Level:          slog.LevelDebug,
		Enabled:        false,
		Endpoint:       firstListener.Addr().String(),
		PublishTimeout: 100 * time.Millisecond,
	})
	defer handler.Close()

	logger := slog.New(handler.WithAttrs([]slog.Attr{slog.String("app", "unit-test")}))
	if err := handler.Configure(Config{Enabled: true, Endpoint: firstListener.Addr().String()}); err != nil {
		t.Fatalf("enable first endpoint: %v", err)
	}
	logger.Info("first message", slog.String("phase", "one"))
	firstRecord := waitRecord(t, firstRecords)
	if firstRecord.Message != "first message" {
		t.Fatalf("first message = %q", firstRecord.Message)
	}
	if firstRecord.Attrs["app"] != "unit-test" {
		t.Fatalf("expected WithAttrs app to be published, got %#v", firstRecord.Attrs)
	}
	if firstRecord.Attrs["phase"] != "one" {
		t.Fatalf("expected record attr phase to be published, got %#v", firstRecord.Attrs)
	}

	if err := handler.Configure(Config{Enabled: true, Endpoint: secondListener.Addr().String()}); err != nil {
		t.Fatalf("switch endpoint: %v", err)
	}
	logger.Info("second message")
	secondRecord := waitRecord(t, secondRecords)
	if secondRecord.Message != "second message" {
		t.Fatalf("second message = %q", secondRecord.Message)
	}

	select {
	case record := <-firstRecords:
		t.Fatalf("unexpected record on old endpoint after switch: %#v", record)
	case <-time.After(75 * time.Millisecond):
	}
}

func TestRPCReceiverConfiguresRuntimeWithoutPersistence(t *testing.T) {
	settings := NewSettings()
	handler := settings.NewHandler(Options{
		Source:         "rpc-test",
		Level:          slog.LevelDebug,
		PublishTimeout: 25 * time.Millisecond,
	})
	defer handler.Close()

	receiver := &RPCReceiver{Settings: settings, Handler: handler, Persist: false}
	var status Status
	if err := receiver.Disable(EmptyArgs{}, &status); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if status.Enabled {
		t.Fatal("expected disabled status")
	}

	if err := receiver.Configure(ConfigureArgs{Enabled: true, Endpoint: "127.0.0.1:7422"}, &status); err != nil {
		t.Fatalf("configure: %v", err)
	}
	if !status.Enabled {
		t.Fatal("expected enabled status")
	}
	if status.Network != "tcp" || status.Address != "127.0.0.1:7422" {
		t.Fatalf("unexpected endpoint status: %#v", status)
	}
}

func startTCPBroker(t *testing.T) (net.Listener, <-chan protocol.Record) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	records := make(chan protocol.Record, 8)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				decoder := json.NewDecoder(conn)
				for {
					var envelope protocol.Envelope
					if err := decoder.Decode(&envelope); err != nil {
						return
					}
					if envelope.Type == protocol.MessageTypeLog && envelope.Record != nil {
						records <- *envelope.Record
					}
				}
			}()
		}
	}()

	return listener, records
}

func waitRecord(t *testing.T, records <-chan protocol.Record) protocol.Record {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	select {
	case record := <-records:
		return record
	case <-ctx.Done():
		t.Fatal("timed out waiting for record")
		return protocol.Record{}
	}
}
