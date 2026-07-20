package tracker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

// --- construction ---

func TestNewGitLab_RequiresBaseURL(t *testing.T) {
	_, err := newGitLab("group/proj", "tok", "")
	if err == nil {
		t.Fatal("expected error for missing base_url")
	}
	if !strings.Contains(err.Error(), "base_url") {
		t.Errorf("error should mention base_url, got %v", err)
	}
}

func TestNewGitLab_RequiresProject(t *testing.T) {
	_, err := newGitLab("", "tok", "https://gitlab.com")
	if err == nil {
		t.Fatal("expected error for missing project")
	}
}

func TestNewGitLab_SatisfiesTracker(t *testing.T) {
	g, err := newGitLab("group/proj", "tok", "https://gitlab.com")
	if err != nil {
		t.Fatal(err)
	}
	var _ Tracker = g // compile-time, but also assert it's non-nil
	if g.Name() != "gitlab" {
		t.Errorf("Name() = %q, want gitlab", g.Name())
	}
	if !g.SupportsHierarchy() {
		t.Error("gitlab should support hierarchy")
	}
}

// --- CreateIssue ---

func TestGitLab_CreateIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/api/v4/projects/group%2Fproj/issues" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		if r.Header.Get("PRIVATE-TOKEN") != "test-token" {
			t.Errorf("unexpected token header: %s", r.Header.Get("PRIVATE-TOKEN"))
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(gitLabIssue{ID: 1001, IID: 42, WebURL: "https://gitlab.com/group/proj/-/issues/42"})
	}))
	defer srv.Close()

	g, err := newGitLab("group/proj", "test-token", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	id, err := g.CreateIssue(testSpec())
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	if id != "42" {
		t.Errorf("id = %q, want 42", id)
	}
}

func TestGitLab_CreateIssue_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"bad"}`))
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "test-token", srv.URL)
	if _, err := g.CreateIssue(testSpec()); err == nil {
		t.Fatal("expected error for API error response")
	}
}

// --- GetIssue & type/priority projection ---

func TestGitLab_GetIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/v4/projects/group%2Fproj/issues/42" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		w.Write([]byte(`{
			"id": 1001, "iid": 42, "title": "Add Apple Pay", "state": "opened",
			"web_url": "https://gitlab.com/group/proj/-/issues/42",
			"labels": ["priority::high", "type::bug"],
			"issue_type": "issue", "weight": 3,
			"created_at": "2026-07-19T10:20:30.123Z",
			"updated_at": "2026-07-20T11:22:33.456Z",
			"assignee": {"username": "alice"},
			"epic": {"id": 7, "iid": 3, "title": "Express Pay"},
			"milestone": {"id": 5, "title": "v1.0"}
		}`))
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "test-token", srv.URL)
	issue, err := g.GetIssue("42")
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	if issue.ID != "42" {
		t.Errorf("ID = %q, want 42", issue.ID)
	}
	if issue.IssueType != "bug" {
		t.Errorf("IssueType = %q, want bug (type::bug label)", issue.IssueType)
	}
	if issue.Priority != "high" {
		t.Errorf("Priority = %q, want high", issue.Priority)
	}
	if issue.Assignee != "alice" {
		t.Errorf("Assignee = %q, want alice", issue.Assignee)
	}
	if issue.CustomFields["gitlab_id"] != "1001" {
		t.Errorf("gitlab_id = %q, want 1001", issue.CustomFields["gitlab_id"])
	}
	if issue.CustomFields["gitlab_milestone"] != "v1.0" {
		t.Errorf("gitlab_milestone = %q, want v1.0", issue.CustomFields["gitlab_milestone"])
	}
	if issue.EpicKey != "3" {
		t.Errorf("EpicKey = %q, want 3", issue.EpicKey)
	}
	if issue.CreatedAt != "2026-07-19T10:20:30.123Z" || issue.UpdatedAt != "2026-07-20T11:22:33.456Z" {
		t.Errorf("timestamps = %q/%q, want native GitLab values", issue.CreatedAt, issue.UpdatedAt)
	}
}

func TestGitLab_GetIssue_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"404 Not Found"}`))
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "test-token", srv.URL)
	if _, err := g.GetIssue("999"); err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestGitLab_IssueTypeMapping(t *testing.T) {
	cases := []struct {
		gitlabType string
		labels     []string
		want       string
	}{
		{"incident", nil, "bug"},
		{"issue", []string{"kind/bug"}, "bug"},
		{"issue", nil, "story"},
		{"task", nil, "story"},
		{"test_case", nil, "task"},
	}
	for _, c := range cases {
		if got := heroIssueType(c.gitlabType, c.labels); got != c.want {
			t.Errorf("heroIssueType(%q,%v) = %q, want %q", c.gitlabType, c.labels, got, c.want)
		}
	}
}

// --- UpdateStatus ---

func TestGitLab_UpdateStatus_Completed(t *testing.T) {
	noteposted, closed := false, false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/notes"):
			noteposted = true
			w.WriteHeader(http.StatusCreated)
		case r.Method == "PUT":
			var p map[string]string
			json.NewDecoder(r.Body).Decode(&p)
			if p["state_event"] == "close" {
				closed = true
			}
			json.NewEncoder(w).Encode(gitLabIssue{IID: 42})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "test-token", srv.URL)
	if err := g.UpdateStatus("42", spec.StatusCompleted); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}
	if !noteposted {
		t.Error("expected status note to be posted")
	}
	if !closed {
		t.Error("expected issue to be closed on completed status")
	}
}

// --- ListIssues + pagination to exhaustion (AC-10) ---

func TestGitLab_ListIssues_PaginationToExhaustion(t *testing.T) {
	const total = 250
	const perPage = 100
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			fmt.Sscanf(p, "%d", &page)
		}
		start := (page - 1) * perPage
		end := start + perPage
		if end > total {
			end = total
		}
		var batch []gitLabIssue
		for i := start; i < end; i++ {
			batch = append(batch, gitLabIssue{
				ID:        1000 + i,
				IID:       i + 1,
				Title:     fmt.Sprintf("Issue %d", i+1),
				State:     "opened",
				CreatedAt: "2026-07-19T10:20:30.123Z",
				UpdatedAt: "2026-07-20T11:22:33.456Z",
			})
		}
		if end < total {
			nextURL := fmt.Sprintf("<%s://%s%s?page=%d>; rel=\"next\"", schemeOf(r), r.Host, r.URL.Path, page+1)
			w.Header().Set("Link", nextURL)
		}
		json.NewEncoder(w).Encode(batch)
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "test-token", srv.URL)
	issues, err := g.ListIssues("", 1000) // limit above total → fetch all
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}
	if len(issues) != total {
		t.Errorf("got %d issues, want %d (pagination not exhausted)", len(issues), total)
	}
	// No duplicates, contiguous IIDs.
	seen := map[string]bool{}
	for _, is := range issues {
		if seen[is.ID] {
			t.Errorf("duplicate issue %s", is.ID)
		}
		seen[is.ID] = true
		if is.CreatedAt != "2026-07-19T10:20:30.123Z" || is.UpdatedAt != "2026-07-20T11:22:33.456Z" {
			t.Errorf("%s timestamps = %q/%q, want native GitLab values", is.ID, is.CreatedAt, is.UpdatedAt)
		}
	}
}

func TestGitLab_SearchProjectsActivityTimestamps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]gitLabIssue{{
			ID:        1001,
			IID:       42,
			Title:     "Activity",
			State:     "opened",
			CreatedAt: "2026-07-19T10:20:30.123Z",
			UpdatedAt: "2026-07-20T11:22:33.456Z",
		}})
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "test-token", srv.URL)
	issues, err := g.Search(SearchQuery{RawQuery: "activity", Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("Search returned %d issues, want 1", len(issues))
	}
	if issues[0].CreatedAt != "2026-07-19T10:20:30.123Z" || issues[0].UpdatedAt != "2026-07-20T11:22:33.456Z" {
		t.Errorf("timestamps = %q/%q, want native GitLab values", issues[0].CreatedAt, issues[0].UpdatedAt)
	}
}

func TestGitLab_ListIssues_SinglePageNoLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No Link header at all — self-managed fallback: single page.
		json.NewEncoder(w).Encode([]gitLabIssue{{IID: 1, Title: "Only"}, {IID: 2, Title: "Two"}})
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "test-token", srv.URL)
	issues, err := g.ListIssues("", 100)
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}
	if len(issues) != 2 {
		t.Errorf("got %d issues, want 2", len(issues))
	}
}

// --- Auth errors (AC-10: 401, 403) ---

func TestGitLab_GetFields_AuthError(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte(`{"message":"denied"}`))
		}))
		g, _ := newGitLab("group/proj", "bad", srv.URL)
		_, err := g.GetFields("42")
		srv.Close()
		if err == nil {
			t.Fatalf("status %d: expected error", status)
		}
		fe, ok := err.(*FieldError)
		if !ok {
			t.Fatalf("status %d: expected *FieldError, got %T", status, err)
		}
		if fe.Kind != FieldErrorAuth {
			t.Errorf("status %d: Kind = %v, want FieldErrorAuth", status, fe.Kind)
		}
	}
}

// --- ListEpics premium degradation (AC-8) ---

func TestGitLab_ListEpics_Available(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/groups/group/epics" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		parent := 9
		json.NewEncoder(w).Encode([]gitLabEpic{
			{ID: 1, IID: 1, Title: "Frictionless Checkout", State: "opened"},
			{ID: 2, IID: 2, Title: "Sub Epic", State: "opened", ParentID: &parent},
		})
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "tok", srv.URL)
	epics, available, err := g.ListEpics("group")
	if err != nil {
		t.Fatalf("ListEpics failed: %v", err)
	}
	if !available {
		t.Fatal("expected epics available")
	}
	if len(epics) != 2 {
		t.Fatalf("got %d epics, want 2", len(epics))
	}
	if epics[0].IssueType != "initiative" {
		t.Errorf("top epic type = %q, want initiative", epics[0].IssueType)
	}
	if epics[1].IssueType != "epic" {
		t.Errorf("nested epic type = %q, want epic", epics[1].IssueType)
	}
}

func TestGitLab_ListEpics_PremiumDegradation(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusNotFound} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		g, _ := newGitLab("group/proj", "tok", srv.URL)
		epics, available, err := g.ListEpics("group")
		srv.Close()
		if err != nil {
			t.Fatalf("status %d: expected graceful degradation, got error %v", status, err)
		}
		if available {
			t.Errorf("status %d: expected available=false", status)
		}
		if epics != nil {
			t.Errorf("status %d: expected nil epics", status)
		}
	}
}

func schemeOf(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
