package cli

import (
	"io"
	"testing"
	"time"
)

// next-projection-stale-graph (found while delivering): a hook whose stdin
// is an open pipe that never sends anything — a git pre-commit run from an
// IDE or agent tool — must not block. `hero next checkpoint` used to wait
// on it forever.
func TestResolveSessionContextDoesNotBlockOnSilentStdin(t *testing.T) {
	r, w := io.Pipe()
	defer w.Close()
	done := make(chan payloadContext, 1)
	go func() { done <- resolveSessionContext(r, "override") }()
	select {
	case ctx := <-done:
		if ctx.SessionID != "override" {
			t.Errorf("SessionID = %q, want the override", ctx.SessionID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("resolveSessionContext blocked on a silent, open stdin")
	}
}

// A payload written promptly is still read and parsed.
func TestResolveSessionContextReadsPromptPayload(t *testing.T) {
	r, w := io.Pipe()
	go func() {
		w.Write([]byte(`{"session_id":"s-1","transcript_path":"/tmp/t.jsonl"}`))
	}()
	ctx := resolveSessionContext(r, "")
	if ctx.SessionID != "s-1" || ctx.TranscriptPath != "/tmp/t.jsonl" {
		t.Errorf("ctx = %+v", ctx)
	}
}
