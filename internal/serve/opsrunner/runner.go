package opsrunner

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

// ringCap is how many recent output lines we keep per job for late
// subscriber backfill. ~200 lines feels right for a several-minute op.
const ringCap = 200

// stderrTailCap is the upper bound on stderr bytes retained for error
// reporting. Matches the spec — last 200 bytes of stderr are shown on
// failure.
const stderrTailCap = 200

// keepaliveInterval is how long Stream waits without real output before
// it emits an SSE comment frame to keep proxies from closing the
// connection. Exposed as a var for tests to override.
var keepaliveInterval = 15 * time.Second

// nowFn is the clock used for StartedAt and keepalive timers. Tests
// swap it to inject a deterministic clock.
var nowFn = time.Now

// Runner is the public type. Server.Run constructs one per daemon and
// shares it across per-project + aggregate handlers.
//
// A Runner is safe for concurrent use.
type Runner struct {
	// parentCtx scopes every subprocess. When parentCtx is cancelled
	// (daemon shutdown), every in-flight subprocess receives SIGKILL via
	// exec.CommandContext.
	parentCtx context.Context

	// binaryPath is the absolute path to the `hero` binary the runner
	// invokes. Resolved at construction via os.Executable() so the
	// subprocess always invokes the same binary as the running daemon.
	binaryPath string

	registry *registry

	// resolverErr is non-nil when os.Executable() failed at
	// construction. Surfaces on Start.
	resolverErr error
}

// New constructs a Runner. The parent ctx scopes every subprocess —
// callers usually pass the server's shutdown ctx so daemon shutdown
// reaps subprocesses cleanly.
//
// The `hero` binary path is resolved via os.Executable(). If that
// fails, every Start call will return an error — the runner does NOT
// silently fall back to PATH lookup.
func New(parent context.Context) *Runner {
	if parent == nil {
		parent = context.Background()
	}
	r := &Runner{parentCtx: parent, registry: newRegistry()}
	path, err := executablePath()
	if err != nil {
		r.resolverErr = fmt.Errorf("opsrunner: resolve hero binary: %w", err)
	}
	r.binaryPath = path
	return r
}

// executablePath is a var so tests can stub it.
var executablePath = func() (string, error) { return os.Executable() }

// Start spawns a subprocess for slug+verb in projectRoot. Returns the
// job id, a started flag (false when an existing in-flight job for the
// same slug+verb was returned), and any error.
//
// The subprocess outlives the request context — the runner uses its
// own parentCtx so a browser disconnect mid-request does NOT kill the
// running op.
func (r *Runner) Start(ctx context.Context, slug, projectRoot, verb string) (jobID string, started bool, err error) {
	if r == nil {
		return "", false, errors.New("opsrunner: nil runner")
	}
	if r.resolverErr != nil {
		return "", false, r.resolverErr
	}
	if slug == "" {
		return "", false, errors.New("opsrunner: empty slug")
	}
	args, ok := Verbs[verb]
	if !ok {
		return "", false, fmt.Errorf("opsrunner: verb %q not in allowlist", verb)
	}
	_ = ctx // request-scoped ctx is intentionally unused — subprocesses live on parentCtx.

	job, isNew := r.registry.getOrCreate(slug, verb, func() *Job {
		return &Job{
			ID:        newJobID(),
			Slug:      slug,
			Verb:      verb,
			StartedAt: nowFn().UTC(),
			done:      make(chan struct{}),
			ring:      newRingBuffer(ringCap),
		}
	})
	if !isNew {
		return job.ID, false, nil
	}

	// Build the subprocess.
	cmd := exec.CommandContext(r.parentCtx, r.binaryPath, args...)
	// Run the child in its own process group and kill the whole group on
	// ctx cancel — otherwise the `#!/bin/sh` wrapper's grandchildren leak,
	// hold the stdout/stderr pipes open, and stall the waiter goroutine.
	setupProcessGroup(cmd)
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(), "HERO_OPSRUNNER=1")

	stdout, errOut := cmd.StdoutPipe()
	if errOut != nil {
		r.registry.release(slug, verb, job.ID)
		close(job.done)
		return "", false, fmt.Errorf("opsrunner: stdout pipe: %w", errOut)
	}
	stderr, errOut := cmd.StderrPipe()
	if errOut != nil {
		r.registry.release(slug, verb, job.ID)
		close(job.done)
		return "", false, fmt.Errorf("opsrunner: stderr pipe: %w", errOut)
	}

	if errOut := cmd.Start(); errOut != nil {
		r.registry.release(slug, verb, job.ID)
		close(job.done)
		return "", false, fmt.Errorf("opsrunner: start: %w", errOut)
	}
	job.cmd = cmd

	// Fan out stdout + stderr concurrently. The waiter goroutine joins
	// both readers before calling cmd.Wait() and closing job.done.
	var ioWG sync.WaitGroup
	ioWG.Add(2)
	go r.pump(job, "stdout", stdout, &ioWG, nil)

	stderrTail := &tailBuffer{cap: stderrTailCap}
	go r.pump(job, "stderr", stderr, &ioWG, stderrTail)

	go func() {
		ioWG.Wait()
		waitErr := cmd.Wait()
		exitCode := 0
		if waitErr != nil {
			if ee, ok := waitErr.(*exec.ExitError); ok {
				exitCode = ee.ExitCode()
			} else {
				exitCode = -1
			}
		}
		job.setExit(exitCode, stderrTail.bytes())
		close(job.done)
		// We intentionally do NOT delete the registry slot here. The
		// registry's lookup() already filters out finished jobs, so
		// dedup-by-slug+verb works correctly on subsequent Start calls
		// (getOrCreate sees the done channel closed and overwrites with
		// a fresh Job). Retaining the entry lets findByID locate the
		// finished job so a slow client connecting after exit can still
		// backfill the ring + receive the exit event.
	}()

	return job.ID, true, nil
}

// Wait blocks until the job identified by slug+jobID has exited (or
// the supplied ctx is cancelled), then returns its exit code. Used by
// the healthcache to chain a cache update onto a `hero check --json`
// subprocess that the existing opsrunner dispatched.
//
// Returns -1 + ctx.Err() when ctx is cancelled before the job finishes.
// Returns an error when the job id is unknown (most likely because the
// caller passed a stale id after the registry slot was reused).
func (r *Runner) Wait(ctx context.Context, slug, jobID string) (int, error) {
	if r == nil {
		return -1, errors.New("opsrunner: nil runner")
	}
	job := r.registry.findByID(slug, jobID)
	if job == nil {
		return -1, fmt.Errorf("opsrunner: job %q not found", jobID)
	}
	select {
	case <-job.done:
		return job.ExitCode(), nil
	case <-ctx.Done():
		return -1, ctx.Err()
	}
}

// Lookup reports the in-flight job id for slug+verb. Used by the
// operations-section loader at page render so the template can mark
// the verb's button as already-in-flight.
func (r *Runner) Lookup(slug, verb string) (jobID string, ok bool) {
	if r == nil {
		return "", false
	}
	job := r.registry.lookup(slug, verb)
	if job == nil {
		return "", false
	}
	return job.ID, true
}

// Stream writes SSE frames to w until the subprocess exits or ctx is
// cancelled. It emits backfill from the ring buffer first, then live
// frames, then a final {"type":"exit","code":N} event followed by
// closing the writer.
//
// Caller is responsible for setting SSE response headers BEFORE calling
// Stream — see the api.go handler.
func (r *Runner) Stream(ctx context.Context, slug, jobID string, w http.ResponseWriter) error {
	if r == nil {
		return errors.New("opsrunner: nil runner")
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		return errors.New("opsrunner: streaming not supported")
	}
	job := r.registry.findByID(slug, jobID)
	if job == nil {
		return fmt.Errorf("opsrunner: job %q not found", jobID)
	}

	subID, lineCh := job.subscribe(64)
	defer job.unsubscribe(subID)

	// Backfill from the ring before live frames. A late subscriber
	// joining 10 seconds in still sees the recent context.
	for _, line := range job.snapshotRing() {
		if !writeLine(w, flusher, line) {
			return nil
		}
	}

	lastWrite := nowFn()
	ticker := time.NewTicker(keepaliveInterval / 3)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-job.done:
			// Drain any lines still in flight before emitting the exit
			// event. Non-blocking — we just take what's queued.
			for drained := true; drained; {
				select {
				case line, ok := <-lineCh:
					if !ok {
						drained = false
						break
					}
					if !writeLine(w, flusher, line) {
						return nil
					}
				default:
					drained = false
				}
			}
			writeExit(w, flusher, job.ExitCode(), job.Stderr())
			return nil
		case line, ok := <-lineCh:
			if !ok {
				// Channel closed without job.done being closed — defensive,
				// emit exit anyway.
				writeExit(w, flusher, job.ExitCode(), job.Stderr())
				return nil
			}
			if !writeLine(w, flusher, line) {
				return nil
			}
			lastWrite = nowFn()
		case <-ticker.C:
			if nowFn().Sub(lastWrite) >= keepaliveInterval {
				if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
					return nil
				}
				flusher.Flush()
				lastWrite = nowFn()
			}
		}
	}
}

// pump reads lines from r and emits them on the job's fan-out. The
// scanner buffer is bumped to 1MiB so long log lines don't fail.
func (run *Runner) pump(job *Job, stream string, src io.Reader, wg *sync.WaitGroup, tail *tailBuffer) {
	defer wg.Done()
	sc := bufio.NewScanner(src)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		text := sc.Text()
		job.emit(outputLine{Stream: stream, Text: text, At: nowFn()})
		if tail != nil {
			tail.write(text)
		}
	}
}

// writeLine encodes a single output line as an SSE "progress" event.
// Returns false if the write failed (client disconnected) so the
// caller exits cleanly.
func writeLine(w http.ResponseWriter, flusher http.Flusher, line outputLine) bool {
	payload, err := json.Marshal(map[string]interface{}{
		"type":   "progress",
		"stream": line.Stream,
		"text":   line.Text,
		"at":     line.At.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return true // skip a malformed frame rather than tear down the stream.
	}
	if _, err := fmt.Fprintf(w, "event: progress\ndata: %s\n\n", payload); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

// writeExit emits the final exit event and flushes. The runner closes
// the stream by returning from Stream after this call.
func writeExit(w http.ResponseWriter, flusher http.Flusher, code int, stderr []byte) {
	payload, _ := json.Marshal(map[string]interface{}{
		"type":   "exit",
		"code":   code,
		"stderr": string(stderr),
	})
	fmt.Fprintf(w, "event: exit\ndata: %s\n\n", payload)
	flusher.Flush()
}

// tailBuffer accumulates the trailing cap bytes of input. Used to
// capture the last 200 bytes of stderr for error reporting.
type tailBuffer struct {
	mu   sync.Mutex
	cap  int
	data []byte
}

func (t *tailBuffer) write(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data = append(t.data, line...)
	t.data = append(t.data, '\n')
	if len(t.data) > t.cap {
		t.data = t.data[len(t.data)-t.cap:]
	}
}

func (t *tailBuffer) bytes() []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]byte, len(t.data))
	copy(out, t.data)
	return out
}
