package tcp

import "context"

func (s *socket) enqueueDispatch(call func() error) bool {
	s.dispatchMu.Lock()
	if s.dispatchClosed {
		s.dispatchMu.Unlock()

		return false
	}

	wasEmpty := len(s.dispatchQueue) == 0
	s.dispatchQueue = append(s.dispatchQueue, call)
	s.dispatchMu.Unlock()

	if wasEmpty {
		select {
		case s.dispatchWake <- struct{}{}:
		default:
		}
	}

	return true
}

func (s *socket) closeDispatch(final func() error) {
	s.dispatchMu.Lock()
	if s.dispatchClosed {
		s.dispatchMu.Unlock()

		return
	}

	if final != nil {
		s.dispatchQueue = append(s.dispatchQueue, final)
	}

	s.dispatchClosed = true
	s.dispatchMu.Unlock()

	select {
	case s.dispatchWake <- struct{}{}:
	default:
	}
}

func (s *socket) nextDispatch(ctx context.Context) (func() error, bool) {
	for {
		s.dispatchMu.Lock()

		if len(s.dispatchQueue) > 0 {
			call := s.dispatchQueue[0]
			s.dispatchQueue = s.dispatchQueue[1:]
			s.dispatchMu.Unlock()

			return call, true
		}

		if s.dispatchClosed {
			s.dispatchMu.Unlock()

			return nil, false
		}

		s.dispatchMu.Unlock()

		select {
		case <-s.dispatchWake:
		case <-ctx.Done():
			s.dispatchMu.Lock()
			s.dispatchClosed = true
			s.dispatchMu.Unlock()
		}
	}
}
