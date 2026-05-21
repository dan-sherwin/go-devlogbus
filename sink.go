package godevlogbus

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/dan-sherwin/devlogbus/pkg/protocol"
)

type sink struct {
	endpoint       endpoint
	queue          chan protocol.Record
	publishTimeout time.Duration
	onError        func(error)
	onSuccess      func()
	done           chan struct{}
	stopped        chan struct{}
	closeOnce      sync.Once
}

func newSink(endpoint endpoint, queueSize int, publishTimeout time.Duration, onError func(error), onSuccess func()) *sink {
	if queueSize <= 0 {
		queueSize = 256
	}
	return &sink{
		endpoint:       endpoint,
		queue:          make(chan protocol.Record, queueSize),
		publishTimeout: publishTimeout,
		onError:        onError,
		onSuccess:      onSuccess,
		done:           make(chan struct{}),
		stopped:        make(chan struct{}),
	}
}

func (s *sink) start() {
	go s.run()
}

func (s *sink) enqueue(record protocol.Record) bool {
	select {
	case s.queue <- record:
		return true
	case <-s.done:
		return false
	default:
		return false
	}
}

func (s *sink) close() {
	s.closeOnce.Do(func() {
		close(s.done)
		<-s.stopped
	})
}

func (s *sink) run() {
	defer close(s.stopped)

	var conn net.Conn
	var encoder *json.Encoder
	defer func() {
		if conn != nil {
			_ = conn.Close()
		}
	}()

	for {
		select {
		case <-s.done:
			return
		case record := <-s.queue:
			var cancel context.CancelFunc
			ctx := context.Background()
			if s.publishTimeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, s.publishTimeout)
			} else {
				ctx, cancel = context.WithCancel(ctx)
			}

			if conn == nil {
				next, err := dialEndpoint(ctx, s.endpoint)
				if err != nil {
					cancel()
					s.markError(err)
					continue
				}
				conn = next
				encoder = json.NewEncoder(next)
			}

			if err := publishRecord(ctx, conn, encoder, record); err != nil {
				_ = conn.Close()
				conn = nil
				encoder = nil
				cancel()
				s.markError(err)
				continue
			}
			cancel()
			s.markSuccess()
		}
	}
}

func (s *sink) markError(err error) {
	if s.onError != nil {
		s.onError(err)
	}
}

func (s *sink) markSuccess() {
	if s.onSuccess != nil {
		s.onSuccess()
	}
}

func dialEndpoint(ctx context.Context, endpoint endpoint) (net.Conn, error) {
	var dialer net.Dialer
	return dialer.DialContext(ctx, endpoint.Network, endpoint.Address)
}

func publishRecord(ctx context.Context, conn net.Conn, encoder *json.Encoder, record protocol.Record) error {
	if conn == nil || encoder == nil {
		return errors.New("devlogbus publisher is closed")
	}
	if record.Time.IsZero() {
		return errors.New("record time is required")
	}
	record.Level = protocol.NormalizeLevel(record.Level)
	if err := record.Validate(); err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetWriteDeadline(deadline)
		defer func() { _ = conn.SetWriteDeadline(time.Time{}) }()
	}
	return encoder.Encode(protocol.Envelope{Type: protocol.MessageTypeLog, Record: &record})
}
