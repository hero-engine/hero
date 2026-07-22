package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	evidencecontract "github.com/hero-engine/hero/contracts/trackerevidence"
	"github.com/hero-engine/hero/internal/tracker"
)

type evidenceLoadServiceMock struct {
	request   evidencecontract.Request
	status    evidencecontract.Status
	evidence  *tracker.IssueEvidence
	readCalls int
}

func (mock *evidenceLoadServiceMock) Load(_ context.Context, request evidencecontract.Request) evidencecontract.Status {
	mock.request = request
	return mock.status
}

func (mock *evidenceLoadServiceMock) ReadSnapshot(evidencecontract.Status) (*tracker.IssueEvidence, error) {
	mock.readCalls++
	if mock.evidence == nil {
		return nil, errors.New("missing test evidence")
	}
	return mock.evidence, nil
}

func withEvidenceLoadService(t *testing.T, mock evidenceLoadService) {
	t.Helper()
	original := newEvidenceLoadService
	newEvidenceLoadService = func(string) evidenceLoadService { return mock }
	t.Cleanup(func() {
		newEvidenceLoadService = original
		evidenceNoAttachments = false
		evidenceStatusOnly = false
		evidenceForce = false
		syncIntegration = ""
	})
}

func TestSyncEvidence_DefaultOutputPreservesIssueEvidenceShape(t *testing.T) {
	newTestEnv(t)
	mock := &evidenceLoadServiceMock{
		status:   evidencecontract.Status{Version: evidencecontract.Version, Status: evidencecontract.StateCurrent, EvidencePath: ".hero/private/evidence.json"},
		evidence: &tracker.IssueEvidence{Tracker: "jira", IssueID: "MORPH-297", Comments: []tracker.EvidenceComment{}, Attachments: []tracker.EvidenceAttachment{}},
	}
	withEvidenceLoadService(t, mock)
	out, err := runCmd("sync", "evidence", "morph-297")
	if err != nil {
		t.Fatal(err)
	}
	var evidence tracker.IssueEvidence
	if err := json.Unmarshal([]byte(out), &evidence); err != nil || evidence.Tracker != "jira" || evidence.IssueID != "MORPH-297" {
		t.Fatalf("default output lost compatibility: err=%v output=%s", err, out)
	}
	if mock.readCalls != 1 || !mock.request.AttachmentsEnabled() || mock.request.SpecSlug != "morph-297" {
		t.Fatalf("request=%+v readCalls=%d", mock.request, mock.readCalls)
	}
}

func TestSyncEvidence_StatusUsesSharedContractAndFlags(t *testing.T) {
	newTestEnv(t)
	want := evidencecontract.Status{Version: evidencecontract.Version, Status: evidencecontract.StateRefreshed, Provider: "jira", SpecSlug: "morph-297", IssueID: "MORPH-297"}
	mock := &evidenceLoadServiceMock{status: want}
	withEvidenceLoadService(t, mock)
	out, err := runCmd("sync", "evidence", "morph-297", "--status", "--force", "--no-attachments", "--integration", "jira-main")
	if err != nil {
		t.Fatal(err)
	}
	var got evidencecontract.Status
	if err := json.Unmarshal([]byte(out), &got); err != nil || got.Version != evidencecontract.Version || got.Status != evidencecontract.StateRefreshed {
		t.Fatalf("status output mismatch: err=%v output=%s", err, out)
	}
	if mock.request.AttachmentsEnabled() || !mock.request.ForceRefresh || mock.request.ConnectionID != "jira-main" || mock.readCalls != 0 {
		t.Fatalf("request=%+v readCalls=%d", mock.request, mock.readCalls)
	}
}

func TestSyncEvidence_DefaultOutputRejectsStructuredFailure(t *testing.T) {
	newTestEnv(t)
	mock := &evidenceLoadServiceMock{status: evidencecontract.Status{
		Version: evidencecontract.Version, Status: evidencecontract.StateUnsupported,
		Error: &evidencecontract.Error{Code: evidencecontract.ErrorUnsupportedProvider, Message: "tracker provider does not support full evidence"},
	}}
	withEvidenceLoadService(t, mock)
	if _, err := runCmd("sync", "evidence", "morph-297"); err == nil {
		t.Fatal("default evidence output accepted an unsupported result")
	}
}

func TestSyncEvidence_JiraSidecarEndToEnd(t *testing.T) {
	env := newTestEnv(t)
	var updated atomic.Value
	updated.Store("2026-07-21T12:00:00.123-0600")
	var evidenceCalls, commentCalls, attachmentCalls atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if !strings.HasPrefix(request.Header.Get("Authorization"), "Basic ") {
			t.Errorf("Jira credential was not injected")
		}
		w.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/rest/api/3/field":
			fmt.Fprint(w, `[]`)
		case "/rest/api/3/issue/MORPH-297":
			nativeUpdated := updated.Load().(string)
			if request.URL.Query().Get("fields") == "*all" {
				evidenceCalls.Add(1)
			}
			fmt.Fprintf(w, `{"key":"MORPH-297","fields":{"summary":"Evidence E2E","description":"full body","created":"2026-07-20T10:00:00.000-0600","updated":%q,"status":{"name":"Open"},"attachment":[{"id":"9","filename":"private.png","content":%q}]},"names":{"summary":"Summary"},"changelog":{"histories":[]}}`, nativeUpdated, server.URL+"/attachment/9")
		case "/rest/api/3/issue/MORPH-297/comment":
			commentCalls.Add(1)
			fmt.Fprint(w, `{"startAt":0,"maxResults":100,"total":1,"comments":[{"id":"1","body":"full comment"}]}`)
		case "/attachment/9":
			attachmentCalls.Add(1)
			fmt.Fprint(w, "private attachment bytes")
		default:
			http.NotFound(w, request)
		}
	}))
	defer server.Close()

	shared := fmt.Sprintf(`{"folder":".hero","integrations":{"connections":{"jira-main":{"provider":"jira","settings":{"project":"MORPH","base_url":%q,"user_email":"fixture@example.com"}}}}}`, server.URL)
	local := `{"integrations":{"connections":{"jira-main":{"auth":{"token":"EVIDENCE-E2E-CANARY"}}}}}`
	if err := os.WriteFile(filepath.Join(env.heroDir, "hero.json"), []byte(shared), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.heroDir, "hero.local.json"), []byte(local), 0o600); err != nil {
		t.Fatal(err)
	}
	specDir := filepath.Join(env.heroDir, "planning", "bugs", "morph-297")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte("---\ntitle: Evidence E2E\nslug: morph-297\ntype: bug\nstatus: planning\ntracker_id: MORPH-297\n---\n# Evidence E2E\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	evidenceStatusOnly = false
	firstJSON, err := runCmd("sync", "evidence", "morph-297", "--status")
	if err != nil {
		t.Fatal(err)
	}
	var first evidencecontract.Status
	if err := json.Unmarshal([]byte(firstJSON), &first); err != nil || first.Status != evidencecontract.StateFetched {
		t.Fatalf("first status=%+v err=%v output=%s", first, err, firstJSON)
	}
	evidenceStatusOnly = false
	secondJSON, err := runCmd("sync", "evidence", "morph-297", "--status")
	if err != nil {
		t.Fatal(err)
	}
	var second evidencecontract.Status
	if err := json.Unmarshal([]byte(secondJSON), &second); err != nil || second.Status != evidencecontract.StateCurrent {
		t.Fatalf("second status=%+v err=%v output=%s", second, err, secondJSON)
	}
	if evidenceCalls.Load() != 1 || commentCalls.Load() != 1 || attachmentCalls.Load() != 1 {
		t.Fatalf("cache hit repeated full Jira work: evidence=%d comments=%d attachments=%d", evidenceCalls.Load(), commentCalls.Load(), attachmentCalls.Load())
	}

	updated.Store("2026-07-21T13:00:00.000-0600")
	evidenceStatusOnly = false
	refreshedJSON, err := runCmd("sync", "evidence", "morph-297", "--status")
	if err != nil {
		t.Fatal(err)
	}
	var refreshed evidencecontract.Status
	if err := json.Unmarshal([]byte(refreshedJSON), &refreshed); err != nil || refreshed.Status != evidencecontract.StateRefreshed {
		t.Fatalf("refreshed status=%+v err=%v output=%s", refreshed, err, refreshedJSON)
	}

	evidenceStatusOnly = false
	legacyJSON, err := runCmd("sync", "evidence", "morph-297")
	if err != nil {
		t.Fatal(err)
	}
	var evidence tracker.IssueEvidence
	if err := json.Unmarshal([]byte(legacyJSON), &evidence); err != nil || evidence.IssueID != "MORPH-297" || len(evidence.Comments) != 1 || len(evidence.Attachments) != 1 {
		t.Fatalf("legacy evidence=%+v err=%v output=%s", evidence, err, legacyJSON)
	}
	if strings.Contains(firstJSON+secondJSON+refreshedJSON, "EVIDENCE-E2E-CANARY") {
		t.Fatal("status output exposed the configured credential")
	}
}
