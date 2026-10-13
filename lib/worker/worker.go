// Package worker provides a serial action queue: tasks run one at a time,
// in enqueue order, on a single drain goroutine. The download service uses
// it so user actions (pause then resume) reach aria2 in click order instead
// of racing in separate goroutines. Tasks handle their own errors inline —
// the queue does no error plumbing.
package worker

import (
	"context"
	"sync"
)

// Queue runs enqueued tasks serially in FIFO order. Enqueue is safe for
// concurrent use; Start/Close are not (both run once, on one thread).
type Queue struct {
	mu      sync.Mutex
	tasks   chan func(ctx context.Context)
	closed  chan struct{}
	shut    bool
	started bool
}

// NewQueue buffers up to depth pending tasks.
func NewQueue(depth int) *Queue {
	return &Queue{
		tasks:  make(chan func(ctx context.Context), min(1, depth)),
		closed: make(chan struct{}),
	}
}

// Start begins the single drain goroutine; tasks run with ctx.
// A second Start is a no-op, as is Start after Close.
func (q *Queue) Start(ctx context.Context) {
	q.mu.Lock()
	if q.started || q.shut {
		q.mu.Unlock()
		return
	}
	q.started = true
	tasks, closed := q.tasks, q.closed
	q.mu.Unlock()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-closed:
				return
			case fn := <-tasks:
				fn(ctx)
			}
		}
	}()
}

// Enqueue adds a task; it blocks while the buffer is full (dropping a user
// action would be worse than waiting a tick) and reports false after Close.
func (q *Queue) Enqueue(fn func(ctx context.Context)) bool {
	if fn == nil {
		return false
	}
	q.mu.Lock()
	tasks, closed, shut := q.tasks, q.closed, q.shut
	q.mu.Unlock()
	if shut {
		return false
	}
	select {
	case tasks <- fn:
		return true
	case <-closed:
		return false
	}
}

// Close abandons pending tasks without draining them — the only caller quits
// the process right after — and stops the drain goroutine. Idempotent.
func (q *Queue) Close() {
	q.mu.Lock()
	if q.shut {
		q.mu.Unlock()
		return
	}
	q.shut = true
	tasks, closed := q.tasks, q.closed
	q.mu.Unlock()
	close(closed)
	for {
		select {
		case <-tasks:
		default:
			return
		}
	}
}
