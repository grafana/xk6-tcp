package tcp

import (
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"

	"github.com/grafana/sobek"
	"go.k6.io/k6/js/promises"
)

var (
	errNoActiveConnection = errors.New("no active connection")
	errWrite0Bytes        = errors.New("write returned 0 bytes written")
)

type writeOptions struct {
	Encoding string
	Tags     map[string]string
}

func (s *socket) write(data sobek.Value, opts *writeOptions) error {
	dataBytes, opts, err := s.writePrepare(data, opts)
	if err != nil {
		if err := s.handleError(err, "write", addToTagSet(s.currentTags(), opts.Tags)); err != nil {
			return err
		}

		return nil
	}

	return s.writeExecute(dataBytes, opts)
}

func (s *socket) writeAsync(data sobek.Value, opts *writeOptions) (*sobek.Promise, error) {
	dataBytes, opts, err := s.writePrepare(data, opts)
	if err != nil {
		return nil, err
	}

	promise, resolve, reject := promises.New(s.vu)

	go func() {
		err := s.writeExecute(dataBytes, opts)
		if err != nil {
			reject(err)

			return
		}

		resolve(sobek.Undefined())
	}()

	return promise, nil
}

func (s *socket) writePrepare(input sobek.Value, opts *writeOptions) ([]byte, *writeOptions, error) {
	if opts == nil {
		opts = &writeOptions{}
	}

	data, err := stringOrArrayBuffer(input, s.vu.Runtime())
	if err != nil {
		return nil, opts, err
	}

	return data, opts, nil
}

func (s *socket) writeExecute(data []byte, opts *writeOptions) error {
	// Copy connection under lock to avoid holding the mutex during blocking I/O.
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()

	if conn == nil {
		return errNoActiveConnection
	}

	s.log.Debug("Writing data to TCP socket")

	var result error

	// Write all data in fragments until complete or error occurs
	var totalWritten int
	for totalWritten < len(data) {
		n, err := conn.Write(data[totalWritten:])
		if err != nil {
			result = fmt.Errorf("failed to write data after %d bytes: %w", totalWritten, err)

			break
		}

		if n == 0 {
			result = fmt.Errorf("%w: after %d bytes", errWrite0Bytes, totalWritten)

			break
		}

		totalWritten += n

		atomic.AddInt64(&s.totalWritten, int64(n))

		s.log.WithField("bytes_written", n).WithField("total_written", totalWritten).Debug("TCP write fragment completed")
	}

	if result != nil {
		s.log.WithError(result).Error("TCP write failed")

		// Track partial write failures separately
		if totalWritten > 0 && totalWritten < len(data) {
			s.addCounterMetrics(s.metrics.tcpPartialWrites, addToTagSet(s.currentTags(), opts.Tags))
		}
	}

	s.addCounterMetrics(s.metrics.tcpWrites, addToTagSet(s.currentTags(), opts.Tags))

	return result
}

func stringOrArrayBuffer(input sobek.Value, runtime *sobek.Runtime) ([]byte, error) {
	var data []byte

	switch input.ExportType() {
	case reflect.TypeFor[string]():
		var str string

		if err := runtime.ExportTo(input, &str); err != nil {
			return nil, err
		}

		data = []byte(str)

	case reflect.TypeFor[[]byte]():
		if err := runtime.ExportTo(input, &data); err != nil {
			return nil, err
		}

	case reflect.TypeFor[sobek.ArrayBuffer]():
		var ab sobek.ArrayBuffer

		if err := runtime.ExportTo(input, &ab); err != nil {
			return nil, err
		}

		data = ab.Bytes()

	default:
		return nil, fmt.Errorf("%w: String or ArrayBuffer expected", errInvalidType)
	}

	return data, nil
}
