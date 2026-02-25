package tcp

import (
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
	port int
	host string
	Tags map[string]string
}

func (co *connectOptions) address() string {
	return net.JoinHostPort(co.host, strconv.Itoa(co.port))
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
	err := s.connectPrepare(portOrOptions, hostOrEmpty)
	if err != nil {
		if e := s.handleError(err, "connect", s.tags()); e != nil {
			return nil, e
		}

		return &sobek.Promise{}, nil
	}

	promise, resolve, reject := promises.New(s.vu)

	go func() {
		if err := s.connectExecute(); err != nil {
			if e := s.handleError(err, "connect", s.tags()); e != nil {
				reject(e)

				return
			}
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
			port: int(portOrOptions.ToInteger()),
			host: defaultHost,
		}

	case reflect.TypeFor[map[string]any]():
		if err := s.vu.Runtime().ExportTo(portOrOptions, &opts); err != nil {
			return err
		}

		if len(opts.host) == 0 {
			opts.host = defaultHost
		}

		hostOrEmpty = nil

	default:
		return fmt.Errorf("%w: expected integer or object", errInvalidType)
	}

	if hostOrEmpty != nil && !sobek.IsUndefined(hostOrEmpty) && !sobek.IsNull(hostOrEmpty) {
		opts.host = hostOrEmpty.String()
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

	// Release mutex before firing events and starting read goroutine
	s.mu.Unlock()

	// Start read goroutine
	go s.read()

	// Fire connect event
	s.fire("connect")

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
	s.endpoints.remoteAddr = net.JoinHostPort(s.endpoints.remoteIP, strconv.Itoa(s.connectOpts.port))

	return nil
}

func (s *socket) dial() error {
	conn, err := s.vu.State().Dialer.DialContext(s.vu.Context(), "tcp", s.endpoints.remoteAddr)
	if err != nil {
		return err
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

// destroy closes the connection and cleans up resources.
// Safe to call multiple times - cleanup happens exactly once.
func (s *socket) destroy() {
	s.destroyOnce.Do(func() {
		// Cancel context to signal loops to stop
		s.cancel()

		// Close connection and update state
		s.mu.Lock()
		s.state = socketStateDestroyed
		conn := s.conn
		s.conn = nil
		duration := time.Since(s.connectTime)
		tags := s.tags()
		s.mu.Unlock()

		// Close connection outside lock
		if conn != nil {
			_ = conn.Close()

			s.addDurationMetrics(duration, s.metrics.tcpDuration, tags)
		}

		// Fire close event
		s.fire("close")
	})
}
