package serve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/propose"
)

// proposalAPITestEnv returns an API + the project slug it's seeded with.
func proposalAPITestEnv(t *testing.T) (*API, string) {
	t.Helper()
	heroDir, projectRoot := setupTestWorkspace(t)
	slug := filepath.Base(projectRoot)
	srv := NewServer(ServerConfig{
		HeroDir:     heroDir,
		ProjectRoot: projectRoot,
		Version:     "test",
		Port:        0,
		AutoWatch:   false,
	})
	return srv.api, slug
}

func ingestEnvelope(t *testing.T, api *API, slug, sessionID string, env *propose.Envelope) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	url := fmt.Sprintf("/api/%s/sessions/%s/proposals/ingest", slug, sessionID)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	return rr
}

func newEnv(proposalID, batchID, agent, anchor string) *propose.Envelope {
	return &propose.Envelope{
		SchemaVersion: propose.SchemaVersion,
		ProposalID:    proposalID,
		BatchID:       batchID,
		SessionID:     "sess-test",
		Agent:         agent,
		Target: propose.Target{
			SpecSlug: "csv-export",
			Anchor: propose.Anchor{
				Kind:     propose.AnchorSection,
				Value:    anchor,
				Position: propose.PositionAppend,
			},
		},
		Content: propose.Content{
			Format: propose.FormatMarkdown,
			Body:   "- THE SYSTEM SHALL …",
		},
	}
}

func TestAPI_ProposalIngest_RoundTrip(t *testing.T) {
	api, slug := proposalAPITestEnv(t)

	// Subscribe to the bus to confirm SSE-equivalent publish happens.
	id, ch := api.bus.Subscribe(8)
	defer api.bus.Unsubscribe(id)

	env := newEnv("p-1", "b-1", "story-writer", "acceptance_criteria")
	rr := ingestEnvelope(t, api, slug, "sess-test", env)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("ingest status = %d body=%s", rr.Code, rr.Body.String())
	}

	select {
	case ev := <-ch:
		if ev.Type != EventProposalEmitted {
			t.Errorf("event type = %q, want %q", ev.Type, EventProposalEmitted)
		}
		if ev.SessionID != "sess-test" {
			t.Errorf("session_id = %q", ev.SessionID)
		}
		if ev.Project != slug {
			t.Errorf("project = %q, want %q", ev.Project, slug)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive proposal_emitted event")
	}

	// List and confirm it's there.
	listURL := fmt.Sprintf("/api/%s/sessions/sess-test/proposals", slug)
	req := httptest.NewRequest(http.MethodGet, listURL, nil)
	rr = httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", rr.Code, rr.Body.String())
	}
	var listResp struct {
		Proposals []propose.Envelope `json:"proposals"`
		Count     int                `json:"count"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if listResp.Count != 1 {
		t.Fatalf("list count = %d, want 1", listResp.Count)
	}
	if listResp.Proposals[0].ProposalID != "p-1" {
		t.Errorf("proposal_id = %q", listResp.Proposals[0].ProposalID)
	}
}

func TestAPI_ProposalIngest_BadJSON(t *testing.T) {
	api, slug := proposalAPITestEnv(t)
	url := fmt.Sprintf("/api/%s/sessions/sess/proposals/ingest", slug)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader([]byte("not json")))
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestAPI_ProposalIngest_SessionMismatch(t *testing.T) {
	api, slug := proposalAPITestEnv(t)
	env := newEnv("p-1", "b-1", "agent", "ac")
	env.SessionID = "different-sess"

	rr := ingestEnvelope(t, api, slug, "sess-test", env)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 on session mismatch", rr.Code)
	}
}

func TestAPI_ProposalIngest_BackfillsSessionID(t *testing.T) {
	api, slug := proposalAPITestEnv(t)
	env := newEnv("p-1", "b-1", "agent", "ac")
	env.SessionID = "" // ingest endpoint must backfill from URL
	rr := ingestEnvelope(t, api, slug, "sess-test", env)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAPI_ProposalAcceptLifecycle(t *testing.T) {
	api, slug := proposalAPITestEnv(t)
	id, ch := api.bus.Subscribe(8)
	defer api.bus.Unsubscribe(id)

	ingestEnvelope(t, api, slug, "sess-test", newEnv("p-1", "b-1", "story-writer", "ac"))
	<-ch // drain emitted

	url := fmt.Sprintf("/api/%s/sessions/sess-test/proposals/p-1/accept", slug)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(`{"by":"user"}`)))
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("accept status = %d body=%s", rr.Code, rr.Body.String())
	}

	select {
	case ev := <-ch:
		if ev.Type != EventProposalAccepted {
			t.Errorf("event type = %q, want accepted", ev.Type)
		}
		rec, ok := ev.Payload.(*propose.LifecycleRecord)
		if !ok {
			t.Fatalf("payload type = %T, want *LifecycleRecord", ev.Payload)
		}
		if rec.By != "user" {
			t.Errorf("by = %q", rec.By)
		}
	case <-time.After(time.Second):
		t.Fatal("no accepted event")
	}
}

func TestAPI_ProposalEditAcceptCarriesBody(t *testing.T) {
	api, slug := proposalAPITestEnv(t)
	id, ch := api.bus.Subscribe(8)
	defer api.bus.Unsubscribe(id)

	ingestEnvelope(t, api, slug, "sess-test", newEnv("p-1", "b-1", "story-writer", "ac"))
	<-ch

	url := fmt.Sprintf("/api/%s/sessions/sess-test/proposals/p-1/edit-accept", slug)
	body := []byte(`{"by":"user","edited_body":"refined body"}`)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("edit-accept status = %d body=%s", rr.Code, rr.Body.String())
	}

	ev := <-ch
	rec := ev.Payload.(*propose.LifecycleRecord)
	if rec.EditedBody != "refined body" {
		t.Errorf("edited_body = %q", rec.EditedBody)
	}
}

func TestAPI_ProposalEditAccept_RequiresBody(t *testing.T) {
	api, slug := proposalAPITestEnv(t)
	ingestEnvelope(t, api, slug, "sess-test", newEnv("p-1", "b-1", "story-writer", "ac"))

	url := fmt.Sprintf("/api/%s/sessions/sess-test/proposals/p-1/edit-accept", slug)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(`{"by":"user"}`)))
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 when edited_body missing", rr.Code)
	}
}

func TestAPI_ProposalReject(t *testing.T) {
	api, slug := proposalAPITestEnv(t)
	id, ch := api.bus.Subscribe(8)
	defer api.bus.Unsubscribe(id)

	ingestEnvelope(t, api, slug, "sess-test", newEnv("p-1", "b-1", "story-writer", "ac"))
	<-ch

	url := fmt.Sprintf("/api/%s/sessions/sess-test/proposals/p-1/reject", slug)
	body := []byte(`{"by":"user","reason":"off-topic"}`)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("reject status = %d body=%s", rr.Code, rr.Body.String())
	}
	ev := <-ch
	if ev.Type != EventProposalRejected {
		t.Errorf("type = %q", ev.Type)
	}
}

func TestAPI_ProposalBulkAccept(t *testing.T) {
	api, slug := proposalAPITestEnv(t)
	id, ch := api.bus.Subscribe(32)
	defer api.bus.Unsubscribe(id)

	// Seed 3 proposals sharing batch_id.
	for i, anchor := range []string{"ac1", "ac2", "ac3"} {
		ingestEnvelope(t, api, slug, "sess-test",
			newEnv(fmt.Sprintf("p-%d", i), "b-shared", "story-writer", anchor))
	}
	// Drain 3 emitted events.
	for i := 0; i < 3; i++ {
		<-ch
	}

	url := fmt.Sprintf("/api/%s/sessions/sess-test/proposals/bulk/accept", slug)
	body := []byte(`{"by":"user","batch_id":"b-shared"}`)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("bulk status = %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Applied []string `json:"applied"`
		Count   int      `json:"count"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Count != 3 {
		t.Errorf("applied count = %d, want 3", resp.Count)
	}

	// Three accepted events should arrive.
	got := 0
	timeout := time.After(time.Second)
	for got < 3 {
		select {
		case ev := <-ch:
			if ev.Type == EventProposalAccepted {
				got++
			}
		case <-timeout:
			t.Fatalf("only %d/3 accepted events arrived", got)
		}
	}
}

func TestAPI_ProposalBulk_EditAcceptRejected(t *testing.T) {
	api, slug := proposalAPITestEnv(t)
	ingestEnvelope(t, api, slug, "sess-test", newEnv("p-1", "b-1", "story-writer", "ac"))

	url := fmt.Sprintf("/api/%s/sessions/sess-test/proposals/bulk/edit-accept", slug)
	body := []byte(`{"batch_id":"b-1"}`)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for bulk edit-accept", rr.Code)
	}
}

func TestAPI_ProposalPerAnchorReplacement_OnEndpoint(t *testing.T) {
	api, slug := proposalAPITestEnv(t)

	rr := ingestEnvelope(t, api, slug, "sess-test", newEnv("p-1", "b-1", "story-writer", "ac"))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("first ingest status = %d", rr.Code)
	}
	rr = ingestEnvelope(t, api, slug, "sess-test", newEnv("p-2", "b-2", "story-writer", "ac"))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("second ingest status = %d", rr.Code)
	}
	var resp struct {
		ProposalID string `json:"proposal_id"`
		ReplacedID string `json:"replaced_id"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.ReplacedID != "p-1" {
		t.Errorf("replaced_id = %q, want p-1", resp.ReplacedID)
	}

	// Only the latest should be listed.
	listURL := fmt.Sprintf("/api/%s/sessions/sess-test/proposals", slug)
	req := httptest.NewRequest(http.MethodGet, listURL, nil)
	rr = httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	var listResp struct {
		Count int `json:"count"`
	}
	json.NewDecoder(rr.Body).Decode(&listResp)
	if listResp.Count != 1 {
		t.Errorf("list count = %d, want 1 after replacement", listResp.Count)
	}
}

func TestAPI_ProposalAccept_NotFound(t *testing.T) {
	api, slug := proposalAPITestEnv(t)
	url := fmt.Sprintf("/api/%s/sessions/sess-test/proposals/nope/accept", slug)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestAPI_ProposalList_FiltersBySpecAndAgent(t *testing.T) {
	api, slug := proposalAPITestEnv(t)
	ingestEnvelope(t, api, slug, "sess-test", newEnv("p-1", "b-1", "story-writer", "ac"))
	// Different agent on a different anchor — must not be filtered out by anchor-replace.
	other := newEnv("p-2", "b-2", "prd-author", "other-section")
	ingestEnvelope(t, api, slug, "sess-test", other)

	// Filter by agent.
	url := fmt.Sprintf("/api/%s/sessions/sess-test/proposals?agent=story-writer", slug)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	var resp struct {
		Count int `json:"count"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Count != 1 {
		t.Errorf("filtered count = %d, want 1", resp.Count)
	}
}

func TestAPI_ProposalAction_UnknownAction(t *testing.T) {
	api, slug := proposalAPITestEnv(t)
	ingestEnvelope(t, api, slug, "sess-test", newEnv("p-1", "b-1", "agent", "ac"))

	url := fmt.Sprintf("/api/%s/sessions/sess-test/proposals/p-1/levitate", slug)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}
