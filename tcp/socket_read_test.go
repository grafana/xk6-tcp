package tcp

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.k6.io/k6/js/modulestest"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/metrics"
)

// newRunningModuleInstance creates a module instance and transitions the VU
// from Init phase to running phase so that s.vu.State() is non-nil.
// This is required for any socket code that calls addErrorMetrics or currentTags.
func newRunningModuleInstance(t *testing.T) (*module, *modulestest.Runtime) {
	t.Helper()

	runtime := modulestest.NewRuntime(t)
	root := new(rootModule)
	moduleInstance := root.NewModuleInstance(runtime.VU)

	mod, ok := moduleInstance.(*module)
	if !ok {
		t.Fatalf("failed to assert module instance")
	}

	registry := runtime.VU.InitEnvField.TestPreInitState.Registry
	runtime.MoveToVUContext(&lib.State{
		Samples: make(chan metrics.SampleContainer, 1000),
		Tags:    lib.NewVUStateTags(registry.RootTagSet()),
	})

	return mod, runtime
}

type stubConn struct {
	net.Conn
	readErr error
}

func (c *stubConn) Read(_ []byte) (int, error)        { return 0, c.readErr }
func (c *stubConn) SetReadDeadline(_ time.Time) error { return nil }
func (c *stubConn) Close() error                      { return nil }

// timeoutError satisfies net.Error with Timeout() == true.
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

func TestReadLoopStepFatalErrorReturnsFalse(t *testing.T) {
	t.Parallel()

	mod, _ := newRunningModuleInstance(t)
	s := newSocket(mod.log, mod.vu, mod.metrics)
	_, cancel := context.WithCancel(mod.vu.Context())
	s.cancel = cancel
	s.state = socketStateOpen

	conn := &stubConn{readErr: errors.New("connection reset by peer")}
	require.False(t, s.readLoopStep(conn, 0))
}

func TestReadLoopStepTimeoutReturnsTrue(t *testing.T) {
	t.Parallel()

	mod := newTestModuleInstance(t)
	s := newSocket(mod.log, mod.vu, mod.metrics)
	_, cancel := context.WithCancel(mod.vu.Context())
	s.cancel = cancel
	s.state = socketStateOpen

	conn := &stubConn{readErr: &net.OpError{Op: "read", Err: &timeoutError{}}}
	require.True(t, s.readLoopStep(conn, 0))
}

func TestFatalReadErrorDestroysSocket(t *testing.T) {
	t.Parallel()

	mod, _ := newRunningModuleInstance(t)
	s := newSocket(mod.log, mod.vu, mod.metrics)
	s.socketOpts = new(socketOptions) // not set when bypassing the JS constructor

	ctx, cancel := context.WithCancel(mod.vu.Context())
	s.cancel = cancel

	go s.loop(ctx)

	s.state = socketStateOpen
	s.conn = &stubConn{readErr: errors.New("connection reset by peer")}

	go s.read()

	require.Eventually(t, func() bool {
		s.mu.RLock()
		state := s.state
		s.mu.RUnlock()
		return state == socketStateDestroyed
	}, time.Second, time.Millisecond)
}
