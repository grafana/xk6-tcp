package tcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnqueueDispatchPreservesFIFO(t *testing.T) {
	t.Parallel()

	mod := newTestModuleInstance(t)
	s := newSocket(mod.log, mod.vu, mod.metrics)

	var got []int

	for _, n := range []int{1, 2, 3} {
		value := n

		require.True(t, s.enqueueDispatch(func() error {
			got = append(got, value)

			return nil
		}))
	}

	for range 3 {
		call, ok := s.nextDispatch(context.Background())
		require.True(t, ok)
		require.NoError(t, call())
	}

	require.Equal(t, []int{1, 2, 3}, got)
}

func TestCloseDispatchAppendsFinalAndRejectsLaterEvents(t *testing.T) {
	t.Parallel()

	mod := newTestModuleInstance(t)
	s := newSocket(mod.log, mod.vu, mod.metrics)

	var got []int

	require.True(t, s.enqueueDispatch(func() error {
		got = append(got, 1)

		return nil
	}))

	s.closeDispatch(func() error {
		got = append(got, 2)

		return nil
	})

	require.False(t, s.enqueueDispatch(func() error {
		got = append(got, 3)

		return nil
	}))

	for range 2 {
		call, ok := s.nextDispatch(context.Background())
		require.True(t, ok)
		require.NoError(t, call())
	}

	call, ok := s.nextDispatch(context.Background())
	require.False(t, ok)
	require.Nil(t, call)
	require.Equal(t, []int{1, 2}, got)
}

func TestNextDispatchDrainsQueuedEventsAfterCancel(t *testing.T) {
	t.Parallel()

	mod := newTestModuleInstance(t)
	s := newSocket(mod.log, mod.vu, mod.metrics)

	var got []int

	for _, n := range []int{1, 2} {
		value := n

		require.True(t, s.enqueueDispatch(func() error {
			got = append(got, value)

			return nil
		}))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for range 2 {
		call, ok := s.nextDispatch(ctx)
		require.True(t, ok)
		require.NoError(t, call())
	}

	call, ok := s.nextDispatch(ctx)
	require.False(t, ok)
	require.Nil(t, call)
	require.True(t, s.dispatchClosed)
	require.Equal(t, []int{1, 2}, got)
}
