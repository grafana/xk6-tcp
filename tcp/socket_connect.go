package tcp

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/grafana/sobek"
	"go.k6.io/k6/js/promises"
)

const defaultHost = "localhost"

type connectOptions struct {
	Port int               `js:"port"`
	Host string            `js:"host"`
	TLS  bool              `js:"tls"`
	Tags map[string]string `js:"tags"`
}

var errNoTLSConfig = errors.New("TLS requested but no TLS config available")

func (co *connectOptions) address() string {
	return net.JoinHostPort(co.Host, strconv.Itoa(co.Port))
}

func (s *socket) connect(portOrOptions sobek.Value, hostOrEmpty sobek.Value) (*sobek.Object, error) {
	err := s.connectPrepare(portOrOptions, hostOrEmpty)
	if err != nil {
		if e := s.handleError(err, "connect", s.tags()); e != nil {
			return s.this, e
		}

		return s.this, nil
	}

	err = s.connectExecute()
	if err != nil {
		if e := s.handleError(err, "connect", s.tags()); e != nil {
			return s.this, e
		}
	}

	return s.this, nil
}

func (s *socket) connectAsync(portOrOptions sobek.Value, hostOrEmpty sobek.Value) (*sobek.Promise, error) {
	promise, resolve, reject := promises.New(s.vu)

	if err := s.connectPrepare(portOrOptions, hostOrEmpty); err != nil {
		tcpErr := s.handleError(err, "connect", s.tags())
		if tcpErr == nil {
			tcpErr = newTCPError(err, "connect")
		}

		reject(tcpErr)

		return promise, nil
	}

	go func() {
		if err := s.connectExecute(); err != nil {
			tcpErr := s.handleError(err, "connect", s.tags())
			if tcpErr == nil {
				tcpErr = newTCPError(err, "connect")
			}

			reject(tcpErr)

			return
		}

		resolve(nil)
	}()

	return promise, nil
}

func (s *socket) connectPrepare(portOrOptions sobek.Value, hostOrEmpty sobek.Value) error {
	var opts *connectOptions

	switch portOrOptions.ExportType() {
	case reflect.TypeFor[int64](), reflect.TypeFor[string]():
		opts = &connectOptions{
			Port: int(portOrOptions.ToInteger()),
			Host: defaultHost,
		}

	case reflect.TypeFor[map[string]any]():
		if err := s.vu.Runtime().ExportTo(portOrOptions, &opts); err != nil {
			return err
		}

		if len(opts.Host) == 0 {
			opts.Host = defaultHost
		}

		hostOrEmpty = nil

	default:
		return fmt.Errorf("%w: expected integer or object", errInvalidType)
	}

	if hostOrEmpty != nil && !sobek.IsUndefined(hostOrEmpty) && !sobek.IsNull(hostOrEmpty) {
		opts.Host = hostOrEmpty.String()
	}

	s.connectOpts = opts

	return nil
}

func (s *socket) connectExecute() error {
	s.mu.Lock()

	s.log.WithField("address", s.connectOpts.address()).Debug("Connecting to TCP server")

	tags := s.tags()

	s.state = socketStateOpening

	err := s.addDurationMetricsFor(s.metrics.tcpResolving, tags, s.resolve)
	if err != nil {
		s.state = socketStateDisconnected
		s.mu.Unlock()

		return err
	}

	err = s.addDurationMetricsFor(s.metrics.tcpConnecting, tags, s.dial)
	if err != nil {
		s.state = socketStateDisconnected
		s.mu.Unlock()

		return err
	}

	// Release mutex before firing events and starting read goroutine.
	s.mu.Unlock()

	// Queue connect before the read goroutine can enqueue data.
	s.fire("connect")

	go s.read()

	s.addCounterMetrics(s.metrics.tcpSockets, tags)

	return nil
}

func (s *socket) resolve() error {
	ip, port, err := s.vu.State().GetAddrResolver().ResolveAddr(s.connectOpts.address())
	if err != nil {
		return err
	}

	s.endpoints.remoteIP = ip.String()
	s.endpoints.remotePort = port
	s.endpoints.remoteAddr = net.JoinHostPort(s.endpoints.remoteIP, strconv.Itoa(s.connectOpts.Port))

	return nil
}

func (s *socket) dial() error {
	conn, err := s.vu.State().Dialer.DialContext(s.vu.Context(), "tcp", s.endpoints.remoteAddr)
	if err != nil {
		return err
	}

	// Wrap with TLS if enabled
	if s.connectOpts.TLS {
		tlsConn, err := s.wrapTLS(conn)
		if err != nil {
			return err
		}

		conn = tlsConn
	}

	localAddr := conn.LocalAddr()
	if tcpAddr, ok := localAddr.(*net.TCPAddr); ok {
		s.endpoints.localIP = tcpAddr.IP.String()
		s.endpoints.localPort = tcpAddr.Port
	}

	s.state = socketStateOpen
	s.conn = conn
	s.connectTime = time.Now()

	// Set read deadline if timeout is configured
	if s.timeout > 0 {
		if err := conn.SetReadDeadline(s.connectTime.Add(s.timeout)); err != nil {
			return err
		}
	}

	return nil
}

func (s *socket) wrapTLS(conn net.Conn) (*tls.Conn, error) {
	if tlsConfig := s.vu.State().TLSConfig; tlsConfig != nil {
		// Clone the TLS config to avoid modifying the shared config
		tlsConfigCopy := tlsConfig.Clone()

		// Set ServerName for SNI if not already set
		if tlsConfigCopy.ServerName == "" {
			tlsConfigCopy.ServerName = s.connectOpts.Host
		}

		// Force HTTP/1.1 to avoid HTTP/2 binary frames
		// This makes raw TCP responses more readable for testing
		tlsConfigCopy.NextProtos = []string{"http/1.1"}

		tlsConn := tls.Client(conn, tlsConfigCopy)

		// Perform TLS handshake
		if err := tlsConn.HandshakeContext(s.vu.Context()); err != nil {
			_ = conn.Close()

			return nil, fmt.Errorf("TLS handshake failed: %w", err)
		}

		s.log.WithField("address", s.endpoints.remoteAddr).Debug("TLS handshake completed")

		return tlsConn, nil
	}

	_ = conn.Close()

	return nil, errNoTLSConfig
}

// destroyWithError is the JS-facing destroy. If an error value is provided,
// the error event is fired (best-effort) before the socket is destroyed.
func (s *socket) destroyWithError(errVal sobek.Value) *sobek.Object {
	if errVal != nil && !sobek.IsUndefined(errVal) && !sobek.IsNull(errVal) {
		s.fire("error", errVal)
	}

	s.destroy()

	return s.this
}

// destroy closes the connection and cleans up resources.
// Safe to call multiple times - cleanup happens exactly once.
func (s *socket) destroy() {
	s.destroyOnce.Do(func() {
		var closeCall func() error
		if call, ok := s.eventCall(nil, "close"); ok {
			closeCall = call
		}

		// Close connection and update state.
		s.mu.Lock()
		s.state = socketStateDestroyed
		conn := s.conn
		s.conn = nil
		duration := time.Since(s.connectTime)
		tags := s.tags()
		s.mu.Unlock()

		// Close connection outside lock.
		if conn != nil {
			_ = conn.Close()

			s.addDurationMetrics(duration, s.metrics.tcpDuration, tags)
		}

		s.closeDispatch(closeCall)

		// Cancel context to signal loops to stop after draining accepted events.
		s.cancel()
	})
}
