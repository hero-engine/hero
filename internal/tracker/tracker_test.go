package tracker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// --- Factory tests ---

func TestNew_NoConfig(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestNew_NoneType(t *testing.T) {
	cfg := &config.TrackerConfig{Type: "none"}
	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for type=none")
	}
}

func TestNew_EmptyType(t *testing.T) {
	cfg := &config.TrackerConfig{Type: ""}
	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestNew_UnknownType(t *testing.T) {
	t.Setenv("TEST_TOKEN_UNK", "tok")
	cfg := &config.TrackerConfig{Type: "trello", TokenEnv: "TEST_TOKEN_UNK", Project: "x"}
	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestNew_MissingTokenEnv(t *testing.T) {
	cfg := &config.TrackerConfig{Type: "github", Project: "owner/repo", TokenEnv: ""}
	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for empty token_env")
	}
}

func TestNew_UnsetEnvVar(t *testing.T) {
	os.Unsetenv("HERO_TEST_MISSING_TOKEN")
	cfg := &config.TrackerConfig{Type: "github", Project: "owner/repo", TokenEnv: "HERO_TEST_MISSING_TOKEN"}
	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for unset env var")
	}
}

func TestNew_GitHub(t *testing.T) {
	t.Setenv("TEST_GH_TOKEN", "ghp_test123")
	cfg := &config.TrackerConfig{Type: "github", Project: "acme/widgets", TokenEnv: "TEST_GH_TOKEN"}
	tr, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Name() != "github" {
		t.Errorf("Name() = %q, want %q", tr.Name(), "github")
	}
}

func TestNew_Jira(t *testing.T) {
	t.Setenv("TEST_JIRA_TOKEN", "jira_test")
	cfg := &config.TrackerConfig{
		Type:     "jira",
		Project:  "PROJ",
		TokenEnv: "TEST_JIRA_TOKEN",
		BaseURL:  "https://test.atlassian.net",
	}
	tr, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Name() != "jira" {
		t.Errorf("Name() = %q, want %q", tr.Name(), "jira")
	}
}

func TestNew_Linear(t *testing.T) {
	t.Setenv("TEST_LINEAR_TOKEN", "lin_test")
	cfg := &config.TrackerConfig{Type: "linear", Project: "ENG", TokenEnv: "TEST_LINEAR_TOKEN"}
	tr, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Name() != "linear" {
		t.Errorf("Name() = %q, want %q", tr.Name(), "linear")
	}
}

// --- GitHub provider validation ---

func TestGitHub_InvalidProject(t *testing.T) {
	_, err := newGitHub("no-slash", "tok", "")
	if err == nil {
		t.Fatal("expected error for missing slash")
	}
}

func TestGitHub_EmptyOwner(t *testing.T) {
	_, err := newGitHub("/repo", "tok", "")
	if err == nil {
		t.Fatal("expected error for empty owner")
	}
}

func TestGitHub_EmptyRepo(t *testing.T) {
	_, err := newGitHub("owner/", "tok", "")
	if err == nil {
		t.Fatal("expected error for empty repo")
	}
}

// --- Jira provider validation ---

func TestJira_EmptyProject(t *testing.T) {
	_, err := newJira("", "tok", "user@example.com", "https://test.atlassian.net")
	if err == nil {
		t.Fatal("expected error for empty project")
	}
}

func TestJira_EmptyBaseURL(t *testing.T) {
	_, err := newJira("PROJ", "tok", "user@example.com", "")
	if err == nil {
		t.Fatal("expected error for empty base_url")
	}
}

func TestJira_BasicAuth_WhenEmailProvided(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When userEmail is set, Jira should use Basic Auth (email:token)
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected Basic Auth header, but none found")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if user != "user@example.com" {
			t.Errorf("basic auth user = %q, want %q", user, "user@example.com")
		}
		if pass != "test-token" {
			t.Errorf("basic auth pass = %q, want %q", pass, "test-token")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"key": "PROJ-1",
			"fields": map[string]interface{}{
				"summary": "Test",
				"status":  map[string]string{"name": "Open"},
			},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	_, err := j.GetIssue("PROJ-1")
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
}

func TestJira_BearerAuth_WhenNoEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When userEmail is empty, Jira should use Bearer token (for Server/DC PAT)
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("auth header = %q, want %q", auth, "Bearer test-token")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"key": "PROJ-1",
			"fields": map[string]interface{}{
				"summary": "Test",
				"status":  map[string]string{"name": "Open"},
			},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "", srv.URL)
	_, err := j.GetIssue("PROJ-1")
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
}

// --- Linear provider validation ---

func TestLinear_EmptyProject(t *testing.T) {
	_, err := newLinear("", "tok", "")
	if err == nil {
		t.Fatal("expected error for empty project")
	}
}

// --- StatusLabel ---

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		status spec.Status
		want   string
	}{
		{spec.StatusPlanning, "Planning"},
		{spec.StatusInReview, "In Review"},
		{spec.StatusDelivering, "Delivering"},
		{spec.StatusCompleted, "Completed"},
		{spec.StatusDraft, "Draft"},
		{spec.StatusActive, "Active"},
		{spec.StatusProposed, "Proposed"},
		{spec.StatusAccepted, "Accepted"},
		{spec.StatusSuperseded, "Superseded"},
		{spec.Status("custom"), "custom"},
	}

	for _, tt := range tests {
		got := StatusLabel(tt.status)
		if got != tt.want {
			t.Errorf("StatusLabel(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

// --- IssueBody ---

func TestIssueBody(t *testing.T) {
	s := &spec.Spec{
		Slug:   "csv-export",
		Title:  "CSV Export",
		Type:   spec.TypeFeature,
		Status: spec.StatusPlanning,
		Sections: map[string]string{
			"goal":     "Export data to CSV",
			"approach": "Use encoding/csv",
		},
	}

	body := IssueBody(s)

	checks := []string{
		"**Spec:** csv-export",
		"**Type:** feature",
		"**Status:** Planning",
		"## Goal",
		"Export data to CSV",
		"## Approach",
		"Use encoding/csv",
		"Hero",
	}

	for _, check := range checks {
		if !containsString(body, check) {
			t.Errorf("IssueBody missing %q", check)
		}
	}
}

// --- GitHub integration tests with httptest ---

func testSpec() *spec.Spec {
	return &spec.Spec{
		Slug:       "csv-export",
		Title:      "CSV Export",
		Type:       spec.TypeFeature,
		Status:     spec.StatusPlanning,
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
		Sections: map[string]string{
			"goal": "Export data to CSV format",
		},
	}
}

func TestGitHub_CreateIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repos/acme/widgets/issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		if _, ok := payload["title"]; !ok {
			t.Error("payload missing title")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"number":   42,
			"html_url": "https://github.com/acme/widgets/issues/42",
		})
	}))
	defer srv.Close()

	g, err := newGitHub("acme/widgets", "test-token", srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	id, err := g.CreateIssue(testSpec())
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	if id != "42" {
		t.Errorf("id = %q, want %q", id, "42")
	}
}

func TestGitHub_CreateIssue_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"message":"Validation Failed"}`))
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "test-token", srv.URL)
	_, err := g.CreateIssue(testSpec())
	if err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestGitHub_UpdateStatus(t *testing.T) {
	commentPosted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/repos/acme/widgets/issues/42/comments":
			commentPosted = true
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "test-token", srv.URL)
	if err := g.UpdateStatus("42", spec.StatusDelivering); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}
	if !commentPosted {
		t.Error("expected comment to be posted")
	}
}

func TestGitHub_UpdateStatus_Completed(t *testing.T) {
	closed := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
		case r.Method == "PATCH" && r.URL.Path == "/repos/acme/widgets/issues/42":
			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["state"] == "closed" {
				closed = true
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "test-token", srv.URL)
	if err := g.UpdateStatus("42", spec.StatusCompleted); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}
	if !closed {
		t.Error("expected issue to be closed on completed status")
	}
}

func TestGitHub_GetIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/widgets/issues/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"number":   42,
			"title":    "Test Issue",
			"state":    "open",
			"html_url": "https://github.com/acme/widgets/issues/42",
		})
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "test-token", srv.URL)
	issue, err := g.GetIssue("42")
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	if issue.ID != "42" {
		t.Errorf("ID = %q, want %q", issue.ID, "42")
	}
	if issue.Title != "Test Issue" {
		t.Errorf("Title = %q, want %q", issue.Title, "Test Issue")
	}
	if issue.Status != "open" {
		t.Errorf("Status = %q, want %q", issue.Status, "open")
	}
}

func TestGitHub_GetIssue_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "test-token", srv.URL)
	_, err := g.GetIssue("999")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

// --- Jira integration tests with httptest ---

func TestJira_CreateIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/rest/api/3/issue" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}

		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		fields, ok := payload["fields"].(map[string]interface{})
		if !ok {
			t.Error("missing fields in payload")
		}
		proj, ok := fields["project"].(map[string]interface{})
		if !ok || proj["key"] != "PROJ" {
			t.Error("missing or wrong project key")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"key": "PROJ-42"})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	id, err := j.CreateIssue(testSpec())
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	if id != "PROJ-42" {
		t.Errorf("id = %q, want %q", id, "PROJ-42")
	}
}

func TestJira_UpdateStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-42/comment" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	if err := j.UpdateStatus("PROJ-42", spec.StatusDelivering); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}
}

func TestJira_GetIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"key": "PROJ-42",
			"fields": map[string]interface{}{
				"summary": "Test Jira Issue",
				"status": map[string]interface{}{
					"name": "In Progress",
				},
			},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	issue, err := j.GetIssue("PROJ-42")
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	if issue.ID != "PROJ-42" {
		t.Errorf("ID = %q, want %q", issue.ID, "PROJ-42")
	}
	if issue.Title != "Test Jira Issue" {
		t.Errorf("Title = %q, want %q", issue.Title, "Test Jira Issue")
	}
	if issue.Status != "In Progress" {
		t.Errorf("Status = %q, want %q", issue.Status, "In Progress")
	}
}

func TestJira_GetIssue_ADFDescription(t *testing.T) {
	// v3 returns description as ADF (Atlassian Document Format)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"key": "PROJ-99",
			"fields": map[string]interface{}{
				"summary": "ADF Description Test",
				"status":  map[string]interface{}{"name": "Open"},
				"description": map[string]interface{}{
					"type":    "doc",
					"version": 1,
					"content": []interface{}{
						map[string]interface{}{
							"type": "paragraph",
							"content": []interface{}{
								map[string]interface{}{"type": "text", "text": "First paragraph."},
							},
						},
						map[string]interface{}{
							"type": "paragraph",
							"content": []interface{}{
								map[string]interface{}{"type": "text", "text": "Second paragraph."},
							},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	issue, err := j.GetIssue("PROJ-99")
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	if issue.Description != "First paragraph.\n\nSecond paragraph." {
		t.Errorf("Description = %q, want %q", issue.Description, "First paragraph.\n\nSecond paragraph.")
	}
}

func TestJira_GetIssue_PlainTextDescription(t *testing.T) {
	// v2 (or some configs) returns description as plain string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"key": "PROJ-100",
			"fields": map[string]interface{}{
				"summary":     "Plain text test",
				"status":      map[string]interface{}{"name": "Open"},
				"description": "This is a plain text description",
			},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	issue, err := j.GetIssue("PROJ-100")
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	if issue.Description != "This is a plain text description" {
		t.Errorf("Description = %q, want %q", issue.Description, "This is a plain text description")
	}
}

func TestJira_IssueType_Mapping(t *testing.T) {
	tests := []struct {
		specType spec.Type
		want     string
	}{
		{spec.TypeBug, "Bug"},
		{spec.TypeFeature, "Story"},
		{spec.TypeInitiative, "Epic"},
		{spec.TypeConvention, "Task"},
		{spec.TypeDecision, "Task"},
	}
	for _, tt := range tests {
		got := jiraIssueType(tt.specType)
		if got != tt.want {
			t.Errorf("jiraIssueType(%q) = %q, want %q", tt.specType, got, tt.want)
		}
	}
}

// --- Linear integration tests with httptest ---

func TestLinear_CreateIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)

		query, ok := payload["query"].(string)
		if !ok {
			t.Error("missing query in GraphQL payload")
		}
		if !containsString(query, "issueCreate") {
			t.Error("expected issueCreate mutation")
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"issueCreate": map[string]interface{}{
					"success": true,
					"issue": map[string]interface{}{
						"id":         "uuid-123",
						"identifier": "ENG-42",
						"url":        "https://linear.app/team/ENG-42",
					},
				},
			},
		})
	}))
	defer srv.Close()

	l, _ := newLinear("ENG", "test-token", srv.URL)
	id, err := l.CreateIssue(testSpec())
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	if id != "ENG-42" {
		t.Errorf("id = %q, want %q", id, "ENG-42")
	}
}

func TestLinear_CreateIssue_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"issueCreate": map[string]interface{}{
					"success": false,
				},
			},
		})
	}))
	defer srv.Close()

	l, _ := newLinear("ENG", "test-token", srv.URL)
	_, err := l.CreateIssue(testSpec())
	if err == nil {
		t.Fatal("expected error for success=false")
	}
}

func TestLinear_CreateIssue_GraphQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]interface{}{
				{"message": "Something went wrong"},
			},
		})
	}))
	defer srv.Close()

	l, _ := newLinear("ENG", "test-token", srv.URL)
	_, err := l.CreateIssue(testSpec())
	if err == nil {
		t.Fatal("expected error for GraphQL error response")
	}
}

func TestLinear_GetIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"issue": map[string]interface{}{
					"id":         "uuid-123",
					"identifier": "ENG-42",
					"title":      "Test Linear Issue",
					"url":        "https://linear.app/team/ENG-42",
					"state": map[string]interface{}{
						"name": "In Progress",
					},
				},
			},
		})
	}))
	defer srv.Close()

	l, _ := newLinear("ENG", "test-token", srv.URL)
	issue, err := l.GetIssue("ENG-42")
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	if issue.ID != "ENG-42" {
		t.Errorf("ID = %q, want %q", issue.ID, "ENG-42")
	}
	if issue.Title != "Test Linear Issue" {
		t.Errorf("Title = %q, want %q", issue.Title, "Test Linear Issue")
	}
}

func TestLinear_UpdateStatus(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		query := fmt.Sprintf("%v", payload["query"])

		if containsString(query, "commentCreate") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"commentCreate": map[string]interface{}{
						"success": true,
					},
				},
			})
			return
		}

		// resolveIssueID query
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"issue": map[string]interface{}{
					"id": "uuid-123",
				},
			},
		})
	}))
	defer srv.Close()

	l, _ := newLinear("ENG", "test-token", srv.URL)
	if err := l.UpdateStatus("ENG-42", spec.StatusDelivering); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}
	if callCount < 2 {
		t.Errorf("expected at least 2 API calls (resolve + comment), got %d", callCount)
	}
}

// --- Helpers ---

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && searchString(s, substr))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- truncateDescription ---

func TestTruncateDescription_Short(t *testing.T) {
	input := "short text"
	got := truncateDescription(input, 100)
	if got != input {
		t.Errorf("truncateDescription(%q, 100) = %q, want %q", input, got, input)
	}
}

func TestTruncateDescription_Long(t *testing.T) {
	input := "this is a longer description that should be truncated at a word boundary somewhere"
	got := truncateDescription(input, 40)
	if len(got) > 43 { // 40 + "..."
		t.Errorf("truncateDescription too long: len=%d, got %q", len(got), got)
	}
	if !containsString(got, "...") {
		t.Errorf("truncateDescription should end with ..., got %q", got)
	}
	// Should not cut mid-word — the truncated part (before "...") should end at a space boundary
	withoutEllipsis := got[:len(got)-3]
	if len(withoutEllipsis) > 0 && withoutEllipsis[len(withoutEllipsis)-1] == ' ' {
		t.Errorf("truncateDescription should not end with trailing space before ..., got %q", got)
	}
}

func TestTruncateDescription_Empty(t *testing.T) {
	got := truncateDescription("", 100)
	if got != "" {
		t.Errorf("truncateDescription(\"\", 100) = %q, want %q", got, "")
	}
}

// --- SearchQueryFromConfig ---

func TestSearchQueryFromConfig_NilFilter(t *testing.T) {
	q := SearchQueryFromConfig(nil, 50)
	if q.Limit != 50 {
		t.Errorf("Limit = %d, want 50", q.Limit)
	}
	if q.RawQuery != "" {
		t.Errorf("RawQuery = %q, want empty", q.RawQuery)
	}
	if q.FilterID != "" {
		t.Errorf("FilterID = %q, want empty", q.FilterID)
	}
}

func TestSearchQueryFromConfig_Populated(t *testing.T) {
	f := &config.ImportFilter{
		JQL:       "project = TEST",
		FilterID:  "12345",
		IssueType: "Bug",
		Assignee:  "alice",
		Labels:    []string{"urgent", "backend"},
		Status:    "Open",
		Priority:  "High",
		OrderBy:   "updated DESC",
	}

	q := SearchQueryFromConfig(f, 25)

	if q.RawQuery != "project = TEST" {
		t.Errorf("RawQuery = %q, want %q", q.RawQuery, "project = TEST")
	}
	if q.FilterID != "12345" {
		t.Errorf("FilterID = %q, want %q", q.FilterID, "12345")
	}
	if q.IssueType != "Bug" {
		t.Errorf("IssueType = %q, want %q", q.IssueType, "Bug")
	}
	if q.Assignee != "alice" {
		t.Errorf("Assignee = %q, want %q", q.Assignee, "alice")
	}
	if len(q.Labels) != 2 || q.Labels[0] != "urgent" || q.Labels[1] != "backend" {
		t.Errorf("Labels = %v, want [urgent backend]", q.Labels)
	}
	if q.Status != "Open" {
		t.Errorf("Status = %q, want %q", q.Status, "Open")
	}
	if q.Priority != "High" {
		t.Errorf("Priority = %q, want %q", q.Priority, "High")
	}
	if q.OrderBy != "updated DESC" {
		t.Errorf("OrderBy = %q, want %q", q.OrderBy, "updated DESC")
	}
	if q.Limit != 25 {
		t.Errorf("Limit = %d, want 25", q.Limit)
	}
}

// --- Jira Search integration tests ---

func TestJira_Search_RawJQL(t *testing.T) {
	var capturedJQL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedJQL = r.URL.Query().Get("jql")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-10",
					"fields": map[string]interface{}{
						"summary": "Raw JQL Result",
						"status":  map[string]interface{}{"name": "Open"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	issues, err := j.Search(SearchQuery{RawQuery: "project = PROJ AND status = Open"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if capturedJQL != "project = PROJ AND status = Open" {
		t.Errorf("JQL = %q, want %q", capturedJQL, "project = PROJ AND status = Open")
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if issues[0].ID != "PROJ-10" {
		t.Errorf("ID = %q, want %q", issues[0].ID, "PROJ-10")
	}
	if issues[0].Title != "Raw JQL Result" {
		t.Errorf("Title = %q, want %q", issues[0].Title, "Raw JQL Result")
	}
}

func TestJira_Search_RawJQL_AutoProjectScope(t *testing.T) {
	// Raw JQL without a project clause should auto-prepend project = PROJ
	var capturedJQL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedJQL = r.URL.Query().Get("jql")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-50",
					"fields": map[string]interface{}{
						"summary": "Auto-scoped",
						"status":  map[string]interface{}{"name": "Open"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	_, err := j.Search(SearchQuery{RawQuery: "assignee = EMPTY AND status != Done"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	want := "project = PROJ AND (assignee = EMPTY AND status != Done)"
	if capturedJQL != want {
		t.Errorf("JQL = %q, want %q", capturedJQL, want)
	}
}

func TestJira_Search_RawJQL_ExplicitProjectNotDuplicated(t *testing.T) {
	// Raw JQL that already contains a project clause should NOT get project prepended
	var capturedJQL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedJQL = r.URL.Query().Get("jql")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	_, err := j.Search(SearchQuery{RawQuery: "project = OTHER AND status = Open"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	// Should pass through as-is since it already has project
	if capturedJQL != "project = OTHER AND status = Open" {
		t.Errorf("JQL = %q, want %q", capturedJQL, "project = OTHER AND status = Open")
	}
}

func TestJira_Search_FilterID(t *testing.T) {
	var capturedJQL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedJQL = r.URL.Query().Get("jql")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-20",
					"fields": map[string]interface{}{
						"summary": "Filter Result",
						"status":  map[string]interface{}{"name": "To Do"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	issues, err := j.Search(SearchQuery{FilterID: "54321", OrderBy: "priority DESC"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if !containsString(capturedJQL, "filter = 54321") {
		t.Errorf("JQL should contain filter ID, got %q", capturedJQL)
	}
	if !containsString(capturedJQL, "ORDER BY priority DESC") {
		t.Errorf("JQL should contain ORDER BY, got %q", capturedJQL)
	}
	if len(issues) != 1 || issues[0].ID != "PROJ-20" {
		t.Errorf("unexpected issues: %+v", issues)
	}
}

func TestJira_Search_FieldFilters_UnassignedAssignee(t *testing.T) {
	var capturedJQL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedJQL = r.URL.Query().Get("jql")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-30",
					"fields": map[string]interface{}{
						"summary":   "Unassigned Bug",
						"status":    map[string]interface{}{"name": "New"},
						"issuetype": map[string]interface{}{"name": "Bug"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	issues, err := j.Search(SearchQuery{
		IssueType: "Bug",
		Assignee:  "unassigned",
		Priority:  "High",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if !containsString(capturedJQL, "project = PROJ") {
		t.Errorf("JQL should scope to project, got %q", capturedJQL)
	}
	if !containsString(capturedJQL, "assignee = EMPTY") {
		t.Errorf("JQL should have assignee = EMPTY for 'unassigned', got %q", capturedJQL)
	}
	if !containsString(capturedJQL, `issuetype = "Bug"`) {
		t.Errorf("JQL should filter by issue type, got %q", capturedJQL)
	}
	if !containsString(capturedJQL, `priority = "High"`) {
		t.Errorf("JQL should filter by priority, got %q", capturedJQL)
	}
	// No status was specified, so no status= clause should be added
	// (but statusCategory != Done is always present)
	if containsString(capturedJQL, `status = "`) {
		t.Errorf("JQL should not have a status= clause when none specified, got %q", capturedJQL)
	}
	if !containsString(capturedJQL, "statusCategory != Done") {
		t.Errorf("JQL should always exclude done status category, got %q", capturedJQL)
	}
	if !containsString(capturedJQL, "ORDER BY created DESC") {
		t.Errorf("JQL should have default ORDER BY, got %q", capturedJQL)
	}
	if len(issues) != 1 || issues[0].ID != "PROJ-30" {
		t.Errorf("unexpected issues: %+v", issues)
	}
}

func TestJira_Search_MixedFields(t *testing.T) {
	var capturedJQL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedJQL = r.URL.Query().Get("jql")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-40",
					"fields": map[string]interface{}{
						"summary": "Mixed Fields",
						"status":  map[string]interface{}{"name": "In Progress"},
						"labels":  []string{"frontend", "urgent"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	issues, err := j.Search(SearchQuery{
		Assignee: "bob@example.com",
		Labels:   []string{"frontend", "urgent"},
		Status:   "In Progress",
		OrderBy:  "updated ASC",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if !containsString(capturedJQL, `assignee = "bob@example.com"`) {
		t.Errorf("JQL should have assignee filter, got %q", capturedJQL)
	}
	if !containsString(capturedJQL, `labels = "frontend"`) {
		t.Errorf("JQL should have frontend label, got %q", capturedJQL)
	}
	if !containsString(capturedJQL, `labels = "urgent"`) {
		t.Errorf("JQL should have urgent label, got %q", capturedJQL)
	}
	if !containsString(capturedJQL, `status = "In Progress"`) {
		t.Errorf("JQL should have status filter, got %q", capturedJQL)
	}
	if !containsString(capturedJQL, "ORDER BY updated ASC") {
		t.Errorf("JQL should have custom ORDER BY, got %q", capturedJQL)
	}
	if len(issues) != 1 || issues[0].ID != "PROJ-40" {
		t.Errorf("unexpected issues: %+v", issues)
	}
}

// --- GitHub Search integration tests ---

func TestGitHub_Search_AssigneeNone(t *testing.T) {
	var capturedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"number":     1,
				"title":      "Unassigned Issue",
				"state":      "open",
				"html_url":   "https://github.com/acme/widgets/issues/1",
				"body":       "Some body text",
				"created_at": "2024-01-15T10:00:00Z",
				"user":       map[string]interface{}{"login": "reporter1"},
				"assignee":   nil,
				"labels":     []map[string]interface{}{},
			},
		})
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "test-token", srv.URL)
	issues, err := g.Search(SearchQuery{Assignee: "none"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if !containsString(capturedURL, "assignee=none") {
		t.Errorf("URL should contain assignee=none, got %q", capturedURL)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if issues[0].ID != "1" {
		t.Errorf("ID = %q, want %q", issues[0].ID, "1")
	}
	if issues[0].Title != "Unassigned Issue" {
		t.Errorf("Title = %q, want %q", issues[0].Title, "Unassigned Issue")
	}
	if issues[0].Reporter != "reporter1" {
		t.Errorf("Reporter = %q, want %q", issues[0].Reporter, "reporter1")
	}
}

func TestGitHub_Search_Labels(t *testing.T) {
	var capturedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"number":     5,
				"title":      "Labeled Issue",
				"state":      "open",
				"html_url":   "https://github.com/acme/widgets/issues/5",
				"body":       "Bug description here",
				"created_at": "2024-02-01T12:00:00Z",
				"user":       map[string]interface{}{"login": "dev1"},
				"assignee":   map[string]interface{}{"login": "dev2"},
				"labels":     []map[string]interface{}{{"name": "bug"}, {"name": "urgent"}},
			},
		})
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "test-token", srv.URL)
	issues, err := g.Search(SearchQuery{Labels: []string{"bug", "urgent"}})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if !containsString(capturedURL, "labels=bug,urgent") {
		t.Errorf("URL should contain labels=bug,urgent, got %q", capturedURL)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if issues[0].Assignee != "dev2" {
		t.Errorf("Assignee = %q, want %q", issues[0].Assignee, "dev2")
	}
	if len(issues[0].Labels) != 2 {
		t.Errorf("Labels count = %d, want 2", len(issues[0].Labels))
	}
}

func TestGitHub_Search_RawQuery(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"number":     99,
					"title":      "Search Result",
					"state":      "open",
					"html_url":   "https://github.com/acme/widgets/issues/99",
					"body":       "Found via search",
					"created_at": "2024-03-01T08:00:00Z",
					"user":       map[string]interface{}{"login": "searcher"},
					"assignee":   nil,
					"labels":     []map[string]interface{}{{"name": "search-hit"}},
				},
			},
		})
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "test-token", srv.URL)
	issues, err := g.Search(SearchQuery{RawQuery: "is:issue is:open label:search-hit"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if capturedPath != "/search/issues" {
		t.Errorf("should use search API, got path %q", capturedPath)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if issues[0].ID != "99" {
		t.Errorf("ID = %q, want %q", issues[0].ID, "99")
	}
	if issues[0].Title != "Search Result" {
		t.Errorf("Title = %q, want %q", issues[0].Title, "Search Result")
	}
	if issues[0].Reporter != "searcher" {
		t.Errorf("Reporter = %q, want %q", issues[0].Reporter, "searcher")
	}
	if len(issues[0].Labels) != 1 || issues[0].Labels[0] != "search-hit" {
		t.Errorf("Labels = %v, want [search-hit]", issues[0].Labels)
	}
}

// --- Jira pagination tests ---

func TestJira_Search_Pagination(t *testing.T) {
	// Simulate 3 pages of results with nextPageToken pagination.
	pageCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageCount++
		token := r.URL.Query().Get("nextPageToken")

		var issues []map[string]interface{}
		var nextToken string

		switch token {
		case "":
			// First page
			issues = []map[string]interface{}{
				{"key": "PROJ-1", "fields": map[string]interface{}{"summary": "Issue 1", "status": map[string]string{"name": "Open"}}},
				{"key": "PROJ-2", "fields": map[string]interface{}{"summary": "Issue 2", "status": map[string]string{"name": "Open"}}},
			}
			nextToken = "page2token"
		case "page2token":
			// Second page
			issues = []map[string]interface{}{
				{"key": "PROJ-3", "fields": map[string]interface{}{"summary": "Issue 3", "status": map[string]string{"name": "Open"}}},
				{"key": "PROJ-4", "fields": map[string]interface{}{"summary": "Issue 4", "status": map[string]string{"name": "Open"}}},
			}
			nextToken = "page3token"
		case "page3token":
			// Third (final) page
			issues = []map[string]interface{}{
				{"key": "PROJ-5", "fields": map[string]interface{}{"summary": "Issue 5", "status": map[string]string{"name": "Open"}}},
			}
			nextToken = "" // no more pages
		}

		resp := map[string]interface{}{"issues": issues}
		if nextToken != "" {
			resp["nextPageToken"] = nextToken
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	j.fieldDiscoveryDone = true // skip field discovery HTTP call in test
	issues, err := j.Search(SearchQuery{RawQuery: "project = PROJ", Limit: 500})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(issues) != 5 {
		t.Errorf("got %d issues, want 5 (across 3 pages)", len(issues))
	}
	if pageCount != 3 {
		t.Errorf("made %d requests, want 3 (one per page)", pageCount)
	}
}

func TestJira_Search_PaginationRespectsLimit(t *testing.T) {
	// With a limit of 3, should stop after 2 pages even though more exist.
	pageCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageCount++
		token := r.URL.Query().Get("nextPageToken")

		var issues []map[string]interface{}
		nextToken := ""

		switch token {
		case "":
			issues = []map[string]interface{}{
				{"key": "PROJ-1", "fields": map[string]interface{}{"summary": "Issue 1", "status": map[string]string{"name": "Open"}}},
				{"key": "PROJ-2", "fields": map[string]interface{}{"summary": "Issue 2", "status": map[string]string{"name": "Open"}}},
			}
			nextToken = "page2token"
		case "page2token":
			issues = []map[string]interface{}{
				{"key": "PROJ-3", "fields": map[string]interface{}{"summary": "Issue 3", "status": map[string]string{"name": "Open"}}},
				{"key": "PROJ-4", "fields": map[string]interface{}{"summary": "Issue 4", "status": map[string]string{"name": "Open"}}},
			}
			nextToken = "page3token"
		case "page3token":
			issues = []map[string]interface{}{
				{"key": "PROJ-5", "fields": map[string]interface{}{"summary": "Issue 5", "status": map[string]string{"name": "Open"}}},
			}
		}

		resp := map[string]interface{}{"issues": issues}
		if nextToken != "" {
			resp["nextPageToken"] = nextToken
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "test-token", "user@example.com", srv.URL)
	j.fieldDiscoveryDone = true // skip field discovery HTTP call in test
	issues, err := j.Search(SearchQuery{RawQuery: "project = PROJ", Limit: 3})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(issues) > 3 {
		t.Errorf("got %d issues, want at most 3 (limit should be respected)", len(issues))
	}
	if pageCount > 2 {
		t.Errorf("made %d requests, want at most 2 (should stop when limit reached)", pageCount)
	}
}

// --- Sprint Loader: configurable custom fields ---

func TestSprintLoader_DefaultStoryPointsField(t *testing.T) {
	// With nil JiraConfig, story points should use default "customfield_10016".
	loader, err := newJiraSprintLoader("PROJ", "tok", "user@example.com", "https://test.atlassian.net", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := loader.storyPointsField(); got != "customfield_10016" {
		t.Errorf("storyPointsField() = %q, want %q", got, "customfield_10016")
	}
}

func TestSprintLoader_ConfiguredStoryPointsField(t *testing.T) {
	jiraCfg := &config.JiraConfig{StoryPointsField: "customfield_10028"}
	loader, err := newJiraSprintLoader("PROJ", "tok", "user@example.com", "https://test.atlassian.net", jiraCfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := loader.storyPointsField(); got != "customfield_10028" {
		t.Errorf("storyPointsField() = %q, want %q", got, "customfield_10028")
	}
}

func TestSprintLoader_NoDefaultAcceptanceCriteriaField(t *testing.T) {
	// With nil JiraConfig, acceptance criteria field should be empty (not imported).
	loader, err := newJiraSprintLoader("PROJ", "tok", "user@example.com", "https://test.atlassian.net", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := loader.acceptanceCriteriaField(); got != "" {
		t.Errorf("acceptanceCriteriaField() = %q, want empty", got)
	}
}

func TestSprintLoader_ConfiguredAcceptanceCriteriaField(t *testing.T) {
	jiraCfg := &config.JiraConfig{AcceptanceCriteriaField: "customfield_10037"}
	loader, err := newJiraSprintLoader("PROJ", "tok", "user@example.com", "https://test.atlassian.net", jiraCfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := loader.acceptanceCriteriaField(); got != "customfield_10037" {
		t.Errorf("acceptanceCriteriaField() = %q, want %q", got, "customfield_10037")
	}
}

func TestSprintLoader_CustomFieldsInResponse(t *testing.T) {
	// Test that configurable custom field IDs are correctly parsed from the API response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the fields parameter includes our custom field IDs
		fields := r.URL.Query().Get("fields")
		if !containsString(fields, "customfield_10099") {
			t.Errorf("fields param should contain story points field, got %q", fields)
		}
		if !containsString(fields, "customfield_10077") {
			t.Errorf("fields param should contain acceptance criteria field, got %q", fields)
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-1",
					"fields": map[string]interface{}{
						"summary":           "Test Issue",
						"description":       nil,
						"issuetype":         map[string]interface{}{"name": "Story"},
						"status":            map[string]interface{}{"name": "To Do"},
						"priority":          map[string]interface{}{"name": "Medium"},
						"assignee":          nil,
						"reporter":          nil,
						"labels":            []interface{}{},
						"components":        []interface{}{},
						"fixVersions":       []interface{}{},
						"parent":            nil,
						"issuelinks":        []interface{}{},
						"customfield_10099": 5.0,
						"customfield_10077": "Given X, When Y, Then Z",
					},
				},
			},
		})
	}))
	defer srv.Close()

	jiraCfg := &config.JiraConfig{
		StoryPointsField:        "customfield_10099",
		AcceptanceCriteriaField: "customfield_10077",
	}
	loader, _ := newJiraSprintLoader("PROJ", "tok", "user@example.com", srv.URL, jiraCfg, "")
	loader.fieldDiscoveryDone = true // skip field discovery HTTP call in test
	items, err := loader.loadSprintItems("42", "Sprint 1")
	if err != nil {
		t.Fatalf("loadSprintItems failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].StoryPoints != 5.0 {
		t.Errorf("StoryPoints = %v, want 5.0", items[0].StoryPoints)
	}
	if items[0].AcceptanceCriteria != "Given X, When Y, Then Z" {
		t.Errorf("AcceptanceCriteria = %q, want %q", items[0].AcceptanceCriteria, "Given X, When Y, Then Z")
	}
}

func TestSprintLoader_NoAcceptanceCriteriaWhenUnconfigured(t *testing.T) {
	// When no acceptance criteria field is configured, it should not be imported
	// even if customfield_10016 (the old hardcoded field) has data.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-2",
					"fields": map[string]interface{}{
						"summary":           "No AC configured",
						"description":       nil,
						"issuetype":         map[string]interface{}{"name": "Story"},
						"status":            map[string]interface{}{"name": "Open"},
						"priority":          map[string]interface{}{"name": "Low"},
						"assignee":          nil,
						"reporter":          nil,
						"labels":            []interface{}{},
						"components":        []interface{}{},
						"fixVersions":       []interface{}{},
						"parent":            nil,
						"issuelinks":        []interface{}{},
						"customfield_10016": 3.0, // This is story points (the default)
					},
				},
			},
		})
	}))
	defer srv.Close()

	// nil JiraConfig = use defaults
	loader, _ := newJiraSprintLoader("PROJ", "tok", "user@example.com", srv.URL, nil, "")
	loader.fieldDiscoveryDone = true // skip field discovery HTTP call in test
	items, err := loader.loadSprintItems("42", "Sprint 1")
	if err != nil {
		t.Fatalf("loadSprintItems failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	// customfield_10016 should be parsed as story points (the default), not acceptance criteria
	if items[0].StoryPoints != 3.0 {
		t.Errorf("StoryPoints = %v, want 3.0", items[0].StoryPoints)
	}
	if items[0].AcceptanceCriteria != "" {
		t.Errorf("AcceptanceCriteria = %q, want empty (no AC field configured)", items[0].AcceptanceCriteria)
	}
}

func TestSprintLoader_ConsistentWithJiraTracker(t *testing.T) {
	// Verify that the sprint loader and jira tracker use the same default for story points.
	j, _ := newJira("PROJ", "tok", "user@example.com", "https://test.atlassian.net")
	loader, _ := newJiraSprintLoader("PROJ", "tok", "user@example.com", "https://test.atlassian.net", nil, "")

	jiraDefault := j.storyPointsField()
	loaderDefault := loader.storyPointsField()

	if jiraDefault != loaderDefault {
		t.Errorf("jira.storyPointsField() = %q, sprint loader = %q — these must match", jiraDefault, loaderDefault)
	}
}

// ---------------------------------------------------------------------------
// Custom field discovery, caching, and parsing tests
// ---------------------------------------------------------------------------

func TestParseCustomFieldValue_PlainString(t *testing.T) {
	raw := json.RawMessage(`"Critical"`)
	got := parseCustomFieldValue(raw)
	if got != "critical" {
		t.Errorf("parseCustomFieldValue(string) = %q, want %q", got, "critical")
	}
}

func TestParseCustomFieldValue_SelectObject(t *testing.T) {
	raw := json.RawMessage(`{"value": "Major", "id": "10100"}`)
	got := parseCustomFieldValue(raw)
	if got != "major" {
		t.Errorf("parseCustomFieldValue(select) = %q, want %q", got, "major")
	}
}

func TestParseCustomFieldValue_NameObject(t *testing.T) {
	// Some custom fields use "name" instead of "value"
	raw := json.RawMessage(`{"name": "Blocker"}`)
	got := parseCustomFieldValue(raw)
	if got != "blocker" {
		t.Errorf("parseCustomFieldValue(name obj) = %q, want %q", got, "blocker")
	}
}

func TestParseCustomFieldValue_Null(t *testing.T) {
	raw := json.RawMessage(`null`)
	got := parseCustomFieldValue(raw)
	if got != "" {
		t.Errorf("parseCustomFieldValue(null) = %q, want empty", got)
	}
}

func TestParseCustomFieldValue_EmptyString(t *testing.T) {
	raw := json.RawMessage(`""`)
	got := parseCustomFieldValue(raw)
	if got != "" {
		t.Errorf("parseCustomFieldValue(empty) = %q, want empty", got)
	}
}

func TestFieldDiscovery_SeverityLikeNames(t *testing.T) {
	// Server returns a list of fields; discovery should find the first severity-like match.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			json.NewEncoder(w).Encode([]map[string]string{
				{"id": "summary", "name": "Summary"},
				{"id": "priority", "name": "Priority"},
				{"id": "customfield_10200", "name": "Criticality"},
				{"id": "customfield_10300", "name": "Severity"},
				{"id": "customfield_10400", "name": "Impact"},
			})
			return
		}
		// Search endpoint
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-1",
					"fields": map[string]interface{}{
						"summary":           "Test",
						"status":            map[string]string{"name": "Open"},
						"priority":          map[string]string{"name": "High"},
						"customfield_10300": map[string]string{"value": "Critical"},
						"customfield_10200": map[string]string{"value": "P1"},
						"customfield_10400": map[string]string{"value": "Customer-facing"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "tok", "user@example.com", srv.URL)
	issues, err := j.ListIssues("", 10)
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}

	issue := issues[0]

	// Priority should be independent (from built-in field).
	if issue.Priority != "high" {
		t.Errorf("Priority = %q, want %q", issue.Priority, "high")
	}

	// Severity convenience field should be the first severity-like match.
	// "severity" beats "criticality" beats "impact" in the priority list.
	if issue.Severity != "critical" {
		t.Errorf("Severity = %q, want %q", issue.Severity, "critical")
	}

	// All three should appear in CustomFields.
	if issue.CustomFields["severity"] != "critical" {
		t.Errorf("CustomFields[severity] = %q, want %q", issue.CustomFields["severity"], "critical")
	}
	if issue.CustomFields["criticality"] != "p1" {
		t.Errorf("CustomFields[criticality] = %q, want %q", issue.CustomFields["criticality"], "p1")
	}
	if issue.CustomFields["impact"] != "customer-facing" {
		t.Errorf("CustomFields[impact] = %q, want %q", issue.CustomFields["impact"], "customer-facing")
	}
}

func TestFieldDiscovery_NoSeverityField(t *testing.T) {
	// When no severity-like fields exist, CustomFields and Severity should be empty.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			json.NewEncoder(w).Encode([]map[string]string{
				{"id": "summary", "name": "Summary"},
				{"id": "priority", "name": "Priority"},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-1",
					"fields": map[string]interface{}{
						"summary":  "Test",
						"status":   map[string]string{"name": "Open"},
						"priority": map[string]string{"name": "Medium"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "tok", "user@example.com", srv.URL)
	issues, err := j.ListIssues("", 10)
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}

	if issues[0].Severity != "" {
		t.Errorf("Severity = %q, want empty (no severity field exists)", issues[0].Severity)
	}
	if len(issues[0].CustomFields) != 0 {
		t.Errorf("CustomFields = %v, want empty", issues[0].CustomFields)
	}
	if issues[0].Priority != "medium" {
		t.Errorf("Priority = %q, want %q", issues[0].Priority, "medium")
	}
}

func TestFieldDiscovery_ConfigOverridesAutoDiscovery(t *testing.T) {
	// When JiraConfig.CustomFields has an ID, it should be used without discovery.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			// Return empty — config should take precedence
			json.NewEncoder(w).Encode([]map[string]string{})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-1",
					"fields": map[string]interface{}{
						"summary":           "Test",
						"status":            map[string]string{"name": "Open"},
						"priority":          map[string]string{"name": "Low"},
						"customfield_99999": map[string]string{"value": "SEV-1"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	jiraCfg := &config.JiraConfig{
		CustomFields: map[string]string{
			"Severity": "customfield_99999",
		},
	}
	j, _ := newJiraWithConfig("PROJ", "tok", "user@example.com", srv.URL, jiraCfg, "")
	issues, err := j.ListIssues("", 10)
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}

	if issues[0].Severity != "sev-1" {
		t.Errorf("Severity = %q, want %q", issues[0].Severity, "sev-1")
	}
	if issues[0].CustomFields["severity"] != "sev-1" {
		t.Errorf("CustomFields[severity] = %q, want %q", issues[0].CustomFields["severity"], "sev-1")
	}
}

func TestFieldDiscovery_LegacySeverityField(t *testing.T) {
	// Legacy SeverityField config should still work (backward compat).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			json.NewEncoder(w).Encode([]map[string]string{})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-1",
					"fields": map[string]interface{}{
						"summary":           "Test",
						"status":            map[string]string{"name": "Open"},
						"priority":          nil,
						"customfield_10100": "Major",
					},
				},
			},
		})
	}))
	defer srv.Close()

	jiraCfg := &config.JiraConfig{
		SeverityField: "customfield_10100",
	}
	j, _ := newJiraWithConfig("PROJ", "tok", "user@example.com", srv.URL, jiraCfg, "")
	issues, err := j.ListIssues("", 10)
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}

	if issues[0].Severity != "major" {
		t.Errorf("Severity = %q, want %q", issues[0].Severity, "major")
	}
	// Priority should be empty (it was null), NOT overwritten by severity.
	if issues[0].Priority != "" {
		t.Errorf("Priority = %q, want empty (null in API, no fallback)", issues[0].Priority)
	}
}

func TestFieldCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	fields := map[string]string{
		"severity":    "customfield_10100",
		"criticality": "customfield_10200",
	}
	saveFieldCache(dir, fields, "TEST")

	loaded := loadFieldCache(dir)
	if loaded == nil {
		t.Fatal("loadFieldCache returned nil after save")
	}
	if loaded.Fields["severity"] != "customfield_10100" {
		t.Errorf("cached severity = %q, want %q", loaded.Fields["severity"], "customfield_10100")
	}
	if loaded.Fields["criticality"] != "customfield_10200" {
		t.Errorf("cached criticality = %q, want %q", loaded.Fields["criticality"], "customfield_10200")
	}
	if loaded.DiscoveredAt == "" {
		t.Error("DiscoveredAt should be set")
	}
}

func TestFieldCache_MissingFile(t *testing.T) {
	dir := t.TempDir()
	loaded := loadFieldCache(dir)
	if loaded != nil {
		t.Errorf("loadFieldCache should return nil for missing file, got %+v", loaded)
	}
}

func TestFieldCache_EmptyFields(t *testing.T) {
	// saveFieldCache with empty map should still write (to persist AutoDiscoveryDone).
	dir := t.TempDir()
	saveFieldCache(dir, map[string]string{}, "TEST")
	loaded := loadFieldCache(dir)
	if loaded == nil {
		t.Fatal("loadFieldCache should return cache even with empty fields")
	}
	if !loaded.AutoDiscoveryDone {
		t.Error("AutoDiscoveryDone should be true after save")
	}
	if len(loaded.Fields) != 0 {
		t.Errorf("Fields should be empty, got %v", loaded.Fields)
	}
}

func TestFieldDiscovery_CachePersistsAndSkipsHTTP(t *testing.T) {
	// First call discovers and caches. Second call uses cache (no HTTP).
	discoveryCallCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			discoveryCallCount++
			json.NewEncoder(w).Encode([]map[string]string{
				{"id": "customfield_10300", "name": "Severity"},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-1",
					"fields": map[string]interface{}{
						"summary":           "Test",
						"status":            map[string]string{"name": "Open"},
						"priority":          map[string]string{"name": "High"},
						"customfield_10300": "Critical",
					},
				},
			},
		})
	}))
	defer srv.Close()

	cacheDir := t.TempDir()

	// First jira instance: should discover and cache.
	j1, _ := newJiraWithConfig("PROJ", "tok", "user@example.com", srv.URL, nil, cacheDir)
	issues1, err := j1.ListIssues("", 10)
	if err != nil {
		t.Fatalf("first ListIssues failed: %v", err)
	}
	if discoveryCallCount != 1 {
		t.Errorf("first run: discovery calls = %d, want 1", discoveryCallCount)
	}
	if issues1[0].Severity != "critical" {
		t.Errorf("first run: Severity = %q, want %q", issues1[0].Severity, "critical")
	}

	// Second jira instance: should use cache, no discovery HTTP call.
	j2, _ := newJiraWithConfig("PROJ", "tok", "user@example.com", srv.URL, nil, cacheDir)
	issues2, err := j2.ListIssues("", 10)
	if err != nil {
		t.Fatalf("second ListIssues failed: %v", err)
	}
	if discoveryCallCount != 1 {
		t.Errorf("second run: discovery calls = %d, want 1 (should use cache)", discoveryCallCount)
	}
	if issues2[0].Severity != "critical" {
		t.Errorf("second run: Severity = %q, want %q", issues2[0].Severity, "critical")
	}
}

func TestCustomFields_PriorityAndSeverityIndependent(t *testing.T) {
	// Both priority and severity should be populated independently — no fallback.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			json.NewEncoder(w).Encode([]map[string]string{
				{"id": "customfield_10300", "name": "Severity"},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-1",
					"fields": map[string]interface{}{
						"summary":           "Both fields set",
						"status":            map[string]string{"name": "Open"},
						"priority":          map[string]string{"name": "Low"},
						"customfield_10300": map[string]string{"value": "Critical"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "tok", "user@example.com", srv.URL)
	issues, err := j.ListIssues("", 10)
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}

	issue := issues[0]
	if issue.Priority != "low" {
		t.Errorf("Priority = %q, want %q", issue.Priority, "low")
	}
	if issue.Severity != "critical" {
		t.Errorf("Severity = %q, want %q", issue.Severity, "critical")
	}
	// They're different — that's the point. Low priority but critical severity.
}

func TestJira_AddComment(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/comment") {
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"12345"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := &config.TrackerConfig{Type: "jira", Project: "PROJ", Token: "tok", UserEmail: "a@b.com", BaseURL: srv.URL}
	tr, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.AddComment("PROJ-1", "Hello world"); err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}
	if len(gotBody) == 0 {
		t.Fatal("no request body received")
	}
	// Verify it's ADF format (Jira Cloud API v3)
	if !strings.Contains(string(gotBody), `"type":"doc"`) {
		t.Errorf("expected ADF body, got: %s", gotBody)
	}
}

func TestJira_AttachFile(t *testing.T) {
	var gotContentType string
	var gotNoCheck string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/attachments") {
			gotContentType = r.Header.Get("Content-Type")
			gotNoCheck = r.Header.Get("X-Atlassian-Token")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"id":"99"}]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// Create temp file
	tmp, err := os.CreateTemp("", "hero-test-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("# Test content\n")
	tmp.Close()

	cfg := &config.TrackerConfig{Type: "jira", Project: "PROJ", Token: "tok", UserEmail: "a@b.com", BaseURL: srv.URL}
	tr, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.AttachFile("PROJ-1", tmp.Name(), "diagnosis-PROJ-1.md"); err != nil {
		t.Fatalf("AttachFile failed: %v", err)
	}
	if !strings.Contains(gotContentType, "multipart/form-data") {
		t.Errorf("expected multipart content type, got: %s", gotContentType)
	}
	if gotNoCheck != "no-check" {
		t.Errorf("expected X-Atlassian-Token: no-check, got: %s", gotNoCheck)
	}
}

func TestGitHub_AddComment(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/comments") {
			json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":1}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := &config.TrackerConfig{Type: "github", Project: "owner/repo", Token: "tok", BaseURL: srv.URL}
	tr, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.AddComment("42", "Test comment"); err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}
	if gotBody["body"] != "Test comment" {
		t.Errorf("comment body = %q, want %q", gotBody["body"], "Test comment")
	}
}

func TestGitHub_AttachFile_FallsBackToComment(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/comments") {
			json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":1}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	tmp, err := os.CreateTemp("", "hero-test-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("# File content\n")
	tmp.Close()

	cfg := &config.TrackerConfig{Type: "github", Project: "owner/repo", Token: "tok", BaseURL: srv.URL}
	tr, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.AttachFile("42", tmp.Name(), "test.md"); err != nil {
		t.Fatalf("AttachFile failed: %v", err)
	}
	if !strings.Contains(gotBody["body"], "**Attached: test.md**") {
		t.Errorf("expected attachment header in comment, got: %s", gotBody["body"])
	}
	if !strings.Contains(gotBody["body"], "# File content") {
		t.Errorf("expected file content in comment, got: %s", gotBody["body"])
	}
}
