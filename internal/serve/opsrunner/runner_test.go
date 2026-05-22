package opsrunner

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// withFakeBinary writes a shell script to tempdir, points the runner's
// executablePath resolver at it, and registers a cleanup to restore.
// The script content is interpolated into the body verbatim.
func withFakeBinary(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("opsrunner fake-binary tests use POSIX shell scripts")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "hero")
	body := "#!/bin/sh\n" + script + "\n"
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	prev := executablePath
	executablePath = func() (string, error) { return path, nil }
	t.Cleanup(func() { executablePath = prev })
	return path
}

// drainStream reads from an httptest.ResponseRecorder body via the
// Stream call's writes — since we run Stream synchronously against the
// recorder, the body is fully populated by the time Stream returns.
func drainStream(t *testing.T, runner *Runner, slug, id string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	if err := runner.Stream(context.Background(), slug, id, rec); err != nil {
		t.Fatalf("Stream err: %v", err)
	}
	return rec.Body.String()
}

func TestRunner_Start_AllowlistEnforcement(t *testing.T) {
	withFakeBinary(t, `echo ok`)
	r := New(context.Background())
	if _, _, err := r.Start(context.Background(), "p", t.TempDir(), "rm-rf"); err == nil {
		t.Fatal("expected error for non-allowlisted verb")
	}
}

func TestRunner_Start_Dedup(t *testing.T) {
	// Sleep keeps the first invocation alive long enough for the second
	// Start to land while it's still in flight.
	withFakeBinary(t, `sleep 2; echo done`)
	r := New(context.Background())

	id1, fresh1, err := r.Start(context.Background(), "p", t.TempDir(), "re-scan")
	if err != nil || !fresh1 {
		t.Fatalf("first Start err=%v fresh=%v", err, fresh1)
	}
	id2, fresh2, err := r.Start(context.Background(), "p", t.TempDir(), "re-scan")
	if err != nil {
		t.Fatal(err)
	}
	if fresh2 {
		t.Fatal("second Start should NOT be fresh")
	}
	if id1 != id2 {
		t.Fatalf("dedup returned different ids: %q vs %q", id1, id2)
	}
}

func TestRunner_Stream_EmitsExitEvent(t *testing.T) {
	withFakeBinary(t, `echo "hello from fake hero"; echo "world"; exit 0`)
	r := New(context.Background())
	id, _, err := r.Start(context.Background(), "p", t.TempDir(), "re-scan")
	if err != nil {
		t.Fatal(err)
	}
	// Wait for subprocess to finish so Stream sees a finished job and
	// can drain ring + emit exit deterministically.
	job := r.registry.findByID("p", id)
	select {
	case <-job.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("subprocess didn't exit in time")
	}

	body := drainStream(t, r, "p", id)
	if !strings.Contains(body, "event: progress") {
		t.Errorf("missing progress event:\n%s", body)
	}
	if !strings.Contains(body, "hello from fake hero") {
		t.Errorf("missing stdout text in body:\n%s", body)
	}
	if !strings.Contains(body, "event: exit") {
		t.Errorf("missing exit event:\n%s", body)
	}
	if !strings.Contains(body, `"code":0`) {
		t.Errorf("missing exit code 0 in body:\n%s", body)
	}
}

func TestRunner_Stream_NonZeroExit(t *testing.T) {
	withFakeBinary(t, `echo "trouble" 1>&2; exit 3`)
	r := New(context.Background())
	id, _, err := r.Start(context.Background(), "p", t.TempDir(), "re-scan")
	if err != nil {
		t.Fatal(err)
	}
	job := r.registry.findByID("p", id)
	<-job.Done()
	if got := job.ExitCode(); got != 3 {
		t.Errorf("ExitCode = %d, want 3", got)
	}
	if !strings.Contains(string(job.Stderr()), "trouble") {
		t.Errorf("Stderr missing 'trouble': %q", string(job.Stderr()))
	}

	body := drainStream(t, r, "p", id)
	if !strings.Contains(body, `"code":3`) {
		t.Errorf("missing exit code 3:\n%s", body)
	}
}

func TestRunner_Lookup_AfterFinish(t *testing.T) {
	withFakeBinary(t, `echo done`)
	r := New(context.Background())
	id, _, err := r.Start(context.Background(), "p", t.TempDir(), "re-scan")
	if err != nil {
		t.Fatal(err)
	}
	job := r.registry.findByID("p", id)
	<-job.Done()

	// After finish, Lookup must return ok=false so the loader doesn't
	// surface a stale "in flight" state.
	if _, ok := r.Lookup("p", "re-scan"); ok {
		t.Error("Lookup should return ok=false after job finished")
	}
}

func TestRunner_Lookup_InFlight(t *testing.T) {
	withFakeBinary(t, `sleep 2; echo done`)
	r := New(context.Background())
	id, _, err := r.Start(context.Background(), "p", t.TempDir(), "re-scan")
	if err != nil {
		t.Fatal(err)
	}
	gotID, ok := r.Lookup("p", "re-scan")
	if !ok || gotID != id {
		t.Errorf("Lookup = %q, %v; want %q, true", gotID, ok, id)
	}
}

func TestRunner_ParentContextCancel_KillsSubprocess(t *testing.T) {
	withFakeBinary(t, `sleep 30; echo done`)
	parent, cancel := context.WithCancel(context.Background())
	r := New(parent)
	id, _, err := r.Start(context.Background(), "p", t.TempDir(), "re-scan")
	if err != nil {
		t.Fatal(err)
	}
	job := r.registry.findByID("p", id)
	cancel()

	select {
	case <-job.Done():
	case <-time.After(15 * time.Second):
		t.Fatal("subprocess did not die after parent ctx cancel")
	}
	if job.cmd == nil || job.cmd.ProcessState == nil {
		t.Fatal("ProcessState nil after wait")
	}
	if job.cmd.ProcessState.Success() {
		// Killed subprocess returns non-zero / signal status — anything
		// non-success is fine; we just don't want it claiming success.
		t.Error("expected non-success ProcessState after kill")
	}
}

func TestRunner_ClientDisconnect_DoesNotKillSubprocess(t *testing.T) {
	withFakeBinary(t, `for i in 1 2 3 4 5; do echo "line $i"; sleep 0.2; done; exit 0`)
	r := New(context.Background())
	id, _, err := r.Start(context.Background(), "p", t.TempDir(), "re-scan")
	if err != nil {
		t.Fatal(err)
	}
	job := r.registry.findByID("p", id)

	// Subscribe and then bail out immediately by cancelling the stream's ctx.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	rec := httptest.NewRecorder()
	_ = r.Stream(ctx, "p", id, rec)

	// Subprocess MUST still finish on its own.
	select {
	case <-job.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("subprocess died with client; it should outlive the disconnect")
	}
	if job.ExitCode() != 0 {
		t.Errorf("ExitCode = %d, want 0 (subprocess ran to completion)", job.ExitCode())
	}
}

func TestRunner_Keepalive(t *testing.T) {
	// Force keepalive to a small interval and use a clock that jumps
	// past it on every read so the first tick is past-due.
	prevInterval := keepaliveInterval
	prevNow := nowFn
	keepaliveInterval = 50 * time.Millisecond
	t.Cleanup(func() { keepaliveInterval = prevInterval; nowFn = prevNow })

	withFakeBinary(t, `sleep 1; exit 0`)
	r := New(context.Background())
	id, _, err := r.Start(context.Background(), "p", t.TempDir(), "re-scan")
	if err != nil {
		t.Fatal(err)
	}
	job := r.registry.findByID("p", id)
	// Wait briefly so the subprocess is running but quiet (sleep 1).
	time.Sleep(200 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	rec := httptest.NewRecorder()
	done := make(chan error, 1)
	go func() { done <- r.Stream(ctx, "p", id, rec) }()

	// Wait long enough that at least one keepalive tick fires while the
	// subprocess is still sleeping (no real output to reset lastWrite).
	time.Sleep(300 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()
	if !strings.Contains(body, ": keepalive") {
		t.Errorf("expected keepalive frame; body:\n%s", body)
	}
	// Cleanup — wait for the job to finish so test doesn't leak goroutines.
	select {
	case <-job.Done():
	case <-time.After(3 * time.Second):
	}
}

func TestRunner_FinishedJob_LookupAndFindByID(t *testing.T) {
	withFakeBinary(t, `echo done; exit 0`)
	r := New(context.Background())
	id, _, err := r.Start(context.Background(), "p", t.TempDir(), "re-scan")
	if err != nil {
		t.Fatal(err)
	}
	job := r.registry.findByID("p", id)
	<-job.Done()

	// Lookup(slug, verb) filters out finished jobs so the page loader
	// doesn't surface a stale "in flight" badge.
	if _, ok := r.Lookup("p", "re-scan"); ok {
		t.Error("Lookup should drop finished jobs")
	}
	// findByID still locates the finished job so Stream can backfill +
	// emit the exit event for a late subscriber.
	if r.registry.findByID("p", id) == nil {
		t.Error("findByID should still return finished job")
	}

	// A second Start for the same slug+verb spawns a fresh job
	// (the dedup path only short-circuits while the prior is in flight).
	id2, fresh, err := r.Start(context.Background(), "p", t.TempDir(), "re-scan")
	if err != nil {
		t.Fatal(err)
	}
	if !fresh {
		t.Error("expected fresh job after prior finished")
	}
	if id2 == id {
		t.Error("expected a new id for the second run")
	}
}

// keep import alive in case future tests need fmt.
var _ = fmt.Sprintf
