package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestQueuePreservesOrder(t *testing.T) {
	q := NewQueue(16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)
	defer q.Close()

	const n = 50
	var mu sync.Mutex
	var got []int
	done := make(chan struct{})
	for i := 0; i < n; i++ {
		i := i
		require.True(t, q.Enqueue(func(context.Context) {
			mu.Lock()
			got = append(got, i)
			mu.Unlock()
		}))
	}
	require.True(t, q.Enqueue(func(context.Context) { close(done) }))
	<-done
	require.Len(t, got, n)
	for i := range got {
		require.Equal(t, i, got[i])
	}
}

func TestQueueContinuesAfterTaskError(t *testing.T) {
	q := NewQueue(4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)
	defer q.Close()

	// Tasks handle their own errors inline; the queue only guarantees the
	// next task still runs.
	errs := make(chan error, 1)
	require.True(t, q.Enqueue(func(context.Context) {
		errs <- errors.New("boom")
	}))
	done := make(chan struct{})
	require.True(t, q.Enqueue(func(context.Context) { close(done) }))
	require.ErrorContains(t, <-errs, "boom")
	<-done
}

func TestQueueEnqueueAfterClose(t *testing.T) {
	q := NewQueue(4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)
	require.True(t, q.Enqueue(func(context.Context) {}))
	q.Close()
	require.False(t, q.Enqueue(func(context.Context) {}))
	q.Close() // idempotent
}

func TestQueueStopsOnCancel(t *testing.T) {
	q := NewQueue(4)
	ctx, cancel := context.WithCancel(context.Background())
	q.Start(ctx)
	cancel()
	time.Sleep(50 * time.Millisecond) // let the drain observe the cancel

	ran := make(chan struct{}, 1)
	require.True(t, q.Enqueue(func(context.Context) { ran <- struct{}{} }))
	select {
	case <-ran:
		t.Fatal("task ran after ctx cancel")
	case <-time.After(100 * time.Millisecond):
	}
	q.Close()
}
