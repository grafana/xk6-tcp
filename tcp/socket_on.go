package tcp

import (
	"fmt"

	"github.com/grafana/sobek"
	"go.k6.io/k6/metrics"
)

var events = map[string]struct{}{ //nolint:gochecknoglobals
	"connect": {},
	"data":    {},
	"close":   {},
	"error":   {},
	"timeout": {},
}

func (s *socket) on(event string, handler sobek.Callable) *sobek.Object {
	if _, ok := events[event]; !ok {
		s.log.WithField("event", event).Warn("Unknown event type")

		return s.this
	}

	if _, ok := s.handlers.Load(event); ok {
		s.log.WithField("event", event).Warn("Event handler already registered, overriding")
	}

	s.log.WithField("event", event).Debug("Event handler registered")

	s.handlers.Store(event, handler)

	return s.this
}

// fire queues an event to be fired in the VU's event loop.
// Args are converted to sobek.Value inside the event loop to avoid race conditions.
func (s *socket) fire(event string, args ...any) bool {
	return s.fireAndCleanup(nil, event, args...)
}

func (s *socket) eventCall(cleanup func(), event string, args ...any) (func() error, bool) {
	f, ok := s.handlers.Load(event)
	if !ok {
		return nil, false
	}

	fn, ok := f.(sobek.Callable)
	if !ok {
		return nil, false
	}

	return func() error {
		if cleanup != nil {
			defer cleanup()
		}

		s.log.WithField("event", event).Debug("Firing event handler")

		sobekArgs := make([]sobek.Value, len(args))
		for i, arg := range args {
			sobekArgs[i] = s.vu.Runtime().ToValue(arg)
		}

		_, err := fn(sobek.Undefined(), sobekArgs...)

		return err
	}, true
}

// fireAndCleanup fires an event with a cleanup callback.
// Args are converted to sobek.Value inside the event loop to avoid race conditions.
func (s *socket) fireAndCleanup(cleanup func(), event string, args ...any) bool {
	call, ok := s.eventCall(cleanup, event, args...)
	if !ok {
		if cleanup != nil {
			cleanup()
		}

		return false
	}

	s.log.WithField("event", event).Debug("Queuing event handler")

	if !s.enqueueDispatch(call) {
		s.log.WithField("event", event).Debug("Socket closed, skipping event")

		if cleanup != nil {
			cleanup()
		}

		return false
	}

	return true
}

func (s *socket) handleError(err error, method string, ts *metrics.TagSet) error {
	s.log.WithField("error", err).WithField("method", method).Error("Handling TCP error")

	s.addErrorMetrics(ts)

	wrapped := newTCPError(err, method)

	if s.fire("error", wrapped) {
		return nil
	}

	return wrapped
}

// TCPError represents an error that occurred during a TCP operation.
type TCPError struct { //nolint:revive
	Name    string
	Method  string
	Message string
}

func newTCPError(err error, method string) *TCPError {
	return &TCPError{
		Name:    "TCPError",
		Method:  method,
		Message: err.Error(),
	}
}

func (e *TCPError) Error() string {
	return fmt.Sprintf("TCP error during %s: %v", e.Method, e.Message)
}
