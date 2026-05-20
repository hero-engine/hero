package opsrunner

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// outputLine is the shape fanned out to live subscribers and stored in
// the ring buffer. Stream is "stdout" or "stderr"; Text is the raw
// line (no trailing newline).
type outputLine struct {
	Stream string
	Text   string
	At     time.Time
}

// Job is a single in-flight (or recently-finished) subprocess. It is
// owned by the Runner; callers receive Job pointers via Lookup only
// for read-only access.
//
// The Done channel is closed by the writer goroutine after the
// subprocess exits and ExitCode/Stderr have been populated. Subscribers
// MUST select on Done as well as the line channel so they exit cleanly
// when the subprocess finishes.
type Job struct {
	ID        string
	Slug      string
	Verb      string
	StartedAt time.Time

	// done is closed once the subprocess exits and ExitCode is set.
	done chan struct{}

	// cmd is retained so cancellation tests can inspect ProcessState
	// after the run completes.
	cmd *exec.Cmd

	// mu guards the mutable fields below.
	mu       sync.Mutex
	exitCode int
	stderr   []byte
	ring     *ringBuffer
	subs     map[uint64]chan outputLine
	nextSub  uint64
}

// ExitCode returns the subprocess exit code. Only valid after Done has
// been closed; reading earlier returns 0.
func (j *Job) ExitCode() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.exitCode
}

// Stderr returns the last ~200 bytes of stderr captured by the runner.
// Only meaningful after Done has been closed.
func (j *Job) Stderr() []byte {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := make([]byte, len(j.stderr))
	copy(out, j.stderr)
	return out
}

// Done returns a channel that is closed when the subprocess exits.
func (j *Job) Done() <-chan struct{} { return j.done }

// snapshotRing returns a copy of the current ring buffer contents for a
// late subscriber's backfill.
func (j *Job) snapshotRing() []outputLine {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.ring == nil {
		return nil
	}
	return j.ring.snapshot()
}

// subscribe attaches a new live subscriber and returns its id and channel.
func (j *Job) subscribe(buf int) (uint64, chan outputLine) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.subs == nil {
		j.subs = make(map[uint64]chan outputLine)
	}
	id := j.nextSub
	j.nextSub++
	ch := make(chan outputLine, buf)
	j.subs[id] = ch
	return id, ch
}

// unsubscribe removes a live subscriber. Safe to call after the writer
// goroutine has closed all channels.
func (j *Job) unsubscribe(id uint64) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if ch, ok := j.subs[id]; ok {
		delete(j.subs, id)
		close(ch)
	}
}

// emit fans a line out to every live subscriber and appends it to the
// ring. Non-blocking: a slow subscriber drops the line rather than
// stalling the writer goroutine.
func (j *Job) emit(line outputLine) {
	j.mu.Lock()
	if j.ring != nil {
		j.ring.push(line)
	}
	for _, ch := range j.subs {
		select {
		case ch <- line:
		default:
		}
	}
	j.mu.Unlock()
}

// closeSubs closes every live subscriber channel after the subprocess
// has exited and the exit event has been emitted. The writer goroutine
// calls this exactly once.
func (j *Job) closeSubs() {
	j.mu.Lock()
	defer j.mu.Unlock()
	for id, ch := range j.subs {
		close(ch)
		delete(j.subs, id)
	}
}

// setExit records the exit code + stderr tail.
func (j *Job) setExit(code int, stderr []byte) {
	j.mu.Lock()
	j.exitCode = code
	j.stderr = stderr
	j.mu.Unlock()
}

// ringBuffer is a tiny FIFO of outputLine. We keep a fixed cap so a
// late subscriber gets at most cap lines of backfill.
type ringBuffer struct {
	cap   int
	lines []outputLine
}

func newRingBuffer(cap int) *ringBuffer {
	return &ringBuffer{cap: cap}
}

func (rb *ringBuffer) push(line outputLine) {
	if len(rb.lines) >= rb.cap {
		// shift left by one — cheap at this size (cap=~200).
		copy(rb.lines, rb.lines[1:])
		rb.lines[len(rb.lines)-1] = line
		return
	}
	rb.lines = append(rb.lines, line)
}

func (rb *ringBuffer) snapshot() []outputLine {
	out := make([]outputLine, len(rb.lines))
	copy(out, rb.lines)
	return out
}

// registry is the in-memory job table keyed by "<slug>:<verb>".
// Lookups + dedup go through this struct; the Runner is just a thin
// orchestration layer over it.
type registry struct {
	mu   sync.Mutex
	jobs map[string]*Job
}

func newRegistry() *registry {
	return &registry{jobs: make(map[string]*Job)}
}

func registryKey(slug, verb string) string {
	return slug + ":" + verb
}

// getOrCreate returns (existingJob, false) when a job is already
// in-flight for slug+verb. When no in-flight job exists, it inserts
// the supplied factory result and returns (newJob, true).
//
// The factory is invoked under the registry lock — it must not block.
func (r *registry) getOrCreate(slug, verb string, factory func() *Job) (*Job, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := registryKey(slug, verb)
	if existing, ok := r.jobs[key]; ok {
		select {
		case <-existing.done:
			// Job already finished — fall through and create a new one.
		default:
			return existing, false
		}
	}
	job := factory()
	r.jobs[key] = job
	return job, true
}

// lookup returns the in-flight job for slug+verb, or nil. Finished
// jobs are NOT returned — only currently-running ones.
func (r *registry) lookup(slug, verb string) *Job {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[registryKey(slug, verb)]
	if !ok {
		return nil
	}
	select {
	case <-job.done:
		return nil
	default:
		return job
	}
}

// findByID returns the job with the given ID for slug. Returns finished
// jobs too — Stream needs to be able to read the ring buffer + exit
// code for a job that finished between the POST and the GET.
func (r *registry) findByID(slug, id string) *Job {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, job := range r.jobs {
		if job.Slug == slug && job.ID == id {
			return job
		}
	}
	return nil
}

// release removes the registry slot for slug+verb. Called by the
// writer goroutine after the subprocess exits and the final SSE event
// has been queued, so a fresh start can immediately spawn a new job.
//
// The Job struct itself stays alive (the runner retains a reference
// via the goroutine closure) so any subscriber still in the middle of
// Stream() can drain the ring buffer.
func (r *registry) release(slug, verb, id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := registryKey(slug, verb)
	if existing, ok := r.jobs[key]; ok && existing.ID == id {
		delete(r.jobs, key)
	}
}

// newJobID returns a ULID-like identifier — timestamp prefix for
// lexicographic sortability plus 8 bytes of randomness. Same shape as
// internal/peering/peercall.go::newCallID. Self-contained here so the
// opsrunner package has no internal-package dependencies.
func newJobID() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	ts := time.Now().UTC().UnixNano()
	return fmt.Sprintf("%016x%s", ts, hex.EncodeToString(buf[:]))
}
