package serve

import (
	"sync"
	"time"
)

// pendingRemoveQueue is an in-memory queue of pending project removals
// that fire after a grace window unless cancelled. Phase 4 of
// hero-serve-project-section: each enqueue starts a timer goroutine that
// either calls onCommit (deadline elapsed) or returns silently
// (cancelled). Entries do NOT survive a daemon restart — that's
// intentional safety; a process restart drops any unconfirmed removal.
//
// Safe for concurrent use. The default constructor uses time.Now and
// time.After; tests inject a fake clock via newPendingRemoveQueueClock.
type pendingRemoveQueue struct {
	mu      sync.Mutex
	entries map[string]*pendingRemoveEntry

	// now returns the current time. Swappable for tests.
	now func() time.Time

	// after returns a channel that delivers a value after d. Swappable
	// for tests so deadlines can be triggered deterministically.
	after func(d time.Duration) <-chan time.Time
}

// pendingRemoveEntry is one in-flight pending removal.
type pendingRemoveEntry struct {
	Slug     string
	Deadline time.Time
	cancel   chan struct{}
}

func newPendingRemoveQueue() *pendingRemoveQueue {
	return &pendingRemoveQueue{
		entries: make(map[string]*pendingRemoveEntry),
		now:     time.Now,
		after:   func(d time.Duration) <-chan time.Time { return time.After(d) },
	}
}

// Enqueue registers a pending-remove for slug that fires onCommit after
// deadline unless Cancel is called first. If a pending-remove already
// exists for slug, the existing one is cancelled and replaced.
//
// Returns the absolute deadline as observed by the queue's clock so the
// caller can surface it to the client.
func (q *pendingRemoveQueue) Enqueue(slug string, after time.Duration, onCommit func() error) time.Time {
	q.mu.Lock()
	// Cancel any existing entry for this slug.
	if existing, ok := q.entries[slug]; ok {
		close(existing.cancel)
		delete(q.entries, slug)
	}
	deadline := q.now().Add(after)
	entry := &pendingRemoveEntry{
		Slug:     slug,
		Deadline: deadline,
		cancel:   make(chan struct{}),
	}
	q.entries[slug] = entry
	cancelCh := entry.cancel
	timerCh := q.after(after)
	q.mu.Unlock()

	go func() {
		select {
		case <-timerCh:
			// Deadline elapsed. Re-check under lock: a concurrent Cancel
			// could have closed cancelCh and deleted the entry between
			// the timer firing and us acquiring the lock. Treat absence
			// from the map as "cancelled" so onCommit doesn't run.
			q.mu.Lock()
			cur, ok := q.entries[slug]
			if !ok || cur != entry {
				q.mu.Unlock()
				return
			}
			delete(q.entries, slug)
			q.mu.Unlock()
			// Also drain a cancel that may have arrived simultaneously.
			select {
			case <-cancelCh:
				return
			default:
			}
			if onCommit != nil {
				_ = onCommit()
			}
		case <-cancelCh:
			// Cancelled — onCommit never runs.
		}
	}()
	return deadline
}

// Cancel removes any pending-remove for slug. No-op when nothing is
// pending (or when the entry already fired). Always safe to call.
func (q *pendingRemoveQueue) Cancel(slug string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	entry, ok := q.entries[slug]
	if !ok {
		return false
	}
	close(entry.cancel)
	delete(q.entries, slug)
	return true
}

// Pending reports whether a removal is currently pending for slug.
func (q *pendingRemoveQueue) Pending(slug string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	_, ok := q.entries[slug]
	return ok
}
