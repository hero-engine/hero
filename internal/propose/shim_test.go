package propose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeIngestServer captures every ingest request so tests can assert
// what the shim forwarded.
type fakeIngestServer struct {
	mu       sync.Mutex
	received []*Envelope
}

func (f *fakeIngestServer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Path: /api/{project}/sessions/{session_id}/proposals/ingest
		if !strings.HasSuffix(r.URL.Path, "/proposals/ingest") {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var env Envelope
		if err := json.Unmarshal(body, &env); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.received = append(f.received, &env)
		f.mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `{"proposal_id":"%s"}`, env.ProposalID)
	})
}

func TestScanAndPost_ForwardsValidEnvelope(t *testing.T) {
	fake := &fakeIngestServer{}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	envelope := `{"schema_version":"1.0","proposal_id":"p-1","batch_id":"b-1","session_id":"sess","agent":"story-writer","target":{"spec_slug":"csv-export","anchor":{"kind":"section","value":"ac","position":"append"}},"content":{"format":"markdown","body":"x"}}`

	input := strings.Join([]string{
		"some chatter from the agent",
		HeroProposalPrefix + envelope,
		"more chatter",
	}, "\n") + "\n"

	var passthrough bytes.Buffer
	err := ScanAndPost(context.Background(), strings.NewReader(input), ShimConfig{
		DaemonURL:   srv.URL,
		Project:     "hero",
		SessionID:   "sess",
		HTTPClient:  srv.Client(),
		PassThrough: &passthrough,
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.received) != 1 {
		t.Fatalf("received %d envelopes, want 1", len(fake.received))
	}
	if fake.received[0].ProposalID != "p-1" {
		t.Errorf("proposal_id = %q", fake.received[0].ProposalID)
	}

	// Non-proposal lines passed through.
	out := passthrough.String()
	if !strings.Contains(out, "some chatter") || !strings.Contains(out, "more chatter") {
		t.Errorf("passthrough = %q, missing chatter lines", out)
	}
	if strings.Contains(out, HeroProposalPrefix) {
		t.Errorf("passthrough leaked proposal prefix line: %q", out)
	}
}

func TestScanAndPost_DropsInvalidEnvelope(t *testing.T) {
	fake := &fakeIngestServer{}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	input := HeroProposalPrefix + `{"schema_version":"1.0"}` + "\n"
	var errOut bytes.Buffer
	err := ScanAndPost(context.Background(), strings.NewReader(input), ShimConfig{
		DaemonURL:  srv.URL,
		Project:    "hero",
		SessionID:  "sess",
		HTTPClient: srv.Client(),
		ErrorLog:   &errOut,
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.received) != 0 {
		t.Errorf("daemon received %d invalid envelopes; want 0", len(fake.received))
	}
	if !strings.Contains(errOut.String(), "invalid envelope") {
		t.Errorf("error log missing invalid-envelope notice: %q", errOut.String())
	}
}

func TestScanAndPost_BackfillsSessionID(t *testing.T) {
	fake := &fakeIngestServer{}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	// Envelope omits session_id; shim should fill it from cfg.
	envelope := `{"schema_version":"1.0","proposal_id":"p-x","batch_id":"b-x","session_id":"sess-from-shim","agent":"a","target":{"spec_slug":"s","anchor":{"kind":"section","value":"ac"}},"content":{"format":"markdown","body":"x"}}`
	input := HeroProposalPrefix + envelope + "\n"
	err := ScanAndPost(context.Background(), strings.NewReader(input), ShimConfig{
		DaemonURL:  srv.URL,
		Project:    "hero",
		SessionID:  "sess-from-shim",
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.received[0].SessionID != "sess-from-shim" {
		t.Errorf("session_id = %q", fake.received[0].SessionID)
	}
}
