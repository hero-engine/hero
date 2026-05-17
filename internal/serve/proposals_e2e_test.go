package serve

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/propose"
)

// TestE2E_ShimToDaemonRoundTrip simulates the full inline-propose
// path: agent stdout → shim parser → daemon ingest endpoint → event
// bus subscriber. Confirms the lifecycle log line is generated when
// the batch drains.
func TestE2E_ShimToDaemonRoundTrip(t *testing.T) {
	heroDir, projectRoot := setupTestWorkspace(t)
	slug := filepath.Base(projectRoot)
	srv := NewServer(ServerConfig{
		HeroDir:     heroDir,
		ProjectRoot: projectRoot,
		Version:     "test",
		Port:        0,
		AutoWatch:   false,
	})

	// Stand up the API on httptest so the shim can POST to a real URL.
	httpSrv := httptest.NewServer(srv.api.Handler())
	defer httpSrv.Close()

	id, ch := srv.Bus().Subscribe(32)
	defer srv.Bus().Unsubscribe(id)

	// Agent "stdout" — two proposals in one batch plus chatter.
	stdout := strings.Join([]string{
		"[turn 1] thinking...",
		propose.HeroProposalPrefix + `{"schema_version":"1.0","proposal_id":"p-1","batch_id":"b-1","session_id":"sess-e2e","agent":"story-writer","target":{"spec_slug":"csv-export","anchor":{"kind":"section","value":"ac1","position":"append"}},"content":{"format":"markdown","body":"- AC one"}}`,
		"[turn 1] emitted proposal 1",
		propose.HeroProposalPrefix + `{"schema_version":"1.0","proposal_id":"p-2","batch_id":"b-1","session_id":"sess-e2e","agent":"story-writer","target":{"spec_slug":"csv-export","anchor":{"kind":"section","value":"ac2","position":"append"}},"content":{"format":"markdown","body":"- AC two"}}`,
		"[turn 1] done",
	}, "\n") + "\n"

	var pass strings.Builder
	err := propose.ScanAndPost(context.Background(), strings.NewReader(stdout), propose.ShimConfig{
		DaemonURL:   httpSrv.URL,
		Project:     slug,
		SessionID:   "sess-e2e",
		HTTPClient:  httpSrv.Client(),
		PassThrough: &pass,
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	// Expect two proposal_emitted events.
	emitted := 0
	timeout := time.After(2 * time.Second)
	for emitted < 2 {
		select {
		case ev := <-ch:
			if ev.Type == EventProposalEmitted {
				emitted++
			}
		case <-timeout:
			t.Fatalf("only %d/2 proposal_emitted events arrived", emitted)
		}
	}

	// Chatter forwarded.
	if !strings.Contains(pass.String(), "thinking") {
		t.Errorf("passthrough missing chatter: %q", pass.String())
	}

	// Now accept one and reject the other; verify the lifecycle records.
	acceptURL := httpSrv.URL + "/api/" + slug + "/sessions/sess-e2e/proposals/p-1/accept"
	resp, err := httpSrv.Client().Post(acceptURL, "application/json", strings.NewReader(`{"by":"user"}`))
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	resp.Body.Close()

	rejectURL := httpSrv.URL + "/api/" + slug + "/sessions/sess-e2e/proposals/p-2/reject"
	resp, err = httpSrv.Client().Post(rejectURL, "application/json", strings.NewReader(`{"by":"user","reason":"redundant"}`))
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	resp.Body.Close()

	// Verify events surfaced.
	sawAccepted, sawRejected := false, false
	timeout = time.After(2 * time.Second)
	for !sawAccepted || !sawRejected {
		select {
		case ev := <-ch:
			switch ev.Type {
			case EventProposalAccepted:
				sawAccepted = true
			case EventProposalRejected:
				sawRejected = true
			}
		case <-timeout:
			t.Fatalf("missing events: accepted=%v rejected=%v", sawAccepted, sawRejected)
		}
	}
}
