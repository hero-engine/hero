package serve

import (
	"sync/atomic"
	"testing"
	"time"
)

// newTestPendingQueue builds a queue whose timer channel is controlled
// by the returned trigger func. Calling trigger() unblocks the after
// channel and lets the deadline path run.
func newTestPendingQueue(t *testing.T) (*pendingRemoveQueue, func()) {
	t.Helper()
	ch := make(chan time.Time, 1)
	q := newPendingRemoveQueue()
	q.after = func(d time.Duration) <-chan time.Time { return ch }
	trigger := func() {
		select {
		case ch <- time.Now():
		default:
		}
	}
	return q, trigger
}

func TestPendingRemove_ElapsedFiresOnCommit(t *testing.T) {
	q, trigger := newTestPendingQueue(t)
	var called int32
	q.Enqueue("foo", 5*time.Second, func() error {
		atomic.AddInt32(&called, 1)
		return nil
	})
	trigger()
	// Wait briefly for the goroutine to run.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&called) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&called); got != 1 {
		t.Fatalf("onCommit called %d times, want 1", got)
	}
	if q.Pending("foo") {
		t.Errorf("entry should be cleared after commit")
	}
}

func TestPendingRemove_CancelPreventsCommit(t *testing.T) {
	q, trigger := newTestPendingQueue(t)
	var called int32
	q.Enqueue("foo", 5*time.Second, func() error {
		atomic.AddInt32(&called, 1)
		return nil
	})
	if !q.Cancel("foo") {
		t.Fatalf("Cancel should return true when entry exists")
	}
	trigger() // late timer fire — entry already gone
	time.Sleep(50 * time.Millisecond)
	if got := atomic.LoadInt32(&called); got != 0 {
		t.Fatalf("onCommit called %d times after Cancel, want 0", got)
	}
}

func TestPendingRemove_CancelOnUnknownSlugIsNoop(t *testing.T) {
	q, _ := newTestPendingQueue(t)
	if q.Cancel("nothing") {
		t.Errorf("Cancel of unknown slug should return false")
	}
}

func TestPendingRemove_DoubleCancelIsNoop(t *testing.T) {
	q, _ := newTestPendingQueue(t)
	q.Enqueue("foo", 5*time.Second, func() error { return nil })
	if !q.Cancel("foo") {
		t.Errorf("first cancel should succeed")
	}
	if q.Cancel("foo") {
		t.Errorf("second cancel should be a no-op")
	}
}

func TestPendingRemove_ReEnqueueCancelsPrevious(t *testing.T) {
	// Use the real time.After: re-enqueue must close the previous entry's
	// cancel channel so its goroutine exits without committing.
	q := newPendingRemoveQueue()

	first := make(chan struct{}, 1)
	q.Enqueue("foo", 100*time.Millisecond, func() error {
		first <- struct{}{}
		return nil
	})

	var second int32
	q.Enqueue("foo", 5*time.Second, func() error {
		atomic.AddInt32(&second, 1)
		return nil
	})

	// The first entry's goroutine must NOT fire onCommit even after its
	// timer would have elapsed.
	select {
	case <-first:
		t.Fatalf("first onCommit ran after re-enqueue")
	case <-time.After(300 * time.Millisecond):
		// good — first entry was cancelled by the re-enqueue.
	}
	if atomic.LoadInt32(&second) != 0 {
		t.Fatalf("second onCommit ran prematurely")
	}
}
