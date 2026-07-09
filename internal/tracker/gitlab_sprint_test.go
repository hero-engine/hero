package tracker

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// AC-9: sprint loading returns the iteration's SprintItems.
func TestGitLabSprint_LoadIteration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/groups/") && strings.HasSuffix(r.URL.Path, "/iterations"):
			w.Write([]byte(`[
				{"id":10,"iid":1,"title":"Sprint 3","state":2,"start_date":"2026-06-01","due_date":"2026-06-14"},
				{"id":11,"iid":2,"title":"Sprint 4","state":1}
			]`))
		case strings.Contains(r.URL.Path, "/issues"):
			if r.URL.Query().Get("iteration_id") != "10" {
				t.Errorf("expected iteration_id=10, got %q", r.URL.Query().Get("iteration_id"))
			}
			w.Write([]byte(`[
				{"iid":42,"title":"Add Apple Pay","state":"opened","issue_type":"issue","weight":3,"labels":["priority::high"],"assignee":{"username":"alice"}}
			]`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	l, err := newGitLabSprintLoader("mygroup/proj", "tok", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	items, info, err := l.LoadIteration("") // current
	if err != nil {
		t.Fatalf("LoadIteration failed: %v", err)
	}
	if info.Name != "Sprint 3" || info.State != "active" {
		t.Errorf("info = %+v, want current Sprint 3", info)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].ID != "42" || items[0].StoryPoints != 3 || items[0].Assignee != "alice" {
		t.Errorf("item = %+v, unexpected", items[0])
	}
}

// AC-8/AC-9: iterations 403 (open-source tier) → clear error, not empty.
func TestGitLabSprint_PremiumDegradation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	l, _ := newGitLabSprintLoader("mygroup/proj", "tok", srv.URL)
	_, _, err := l.LoadIteration("")
	if err == nil {
		t.Fatal("expected error when iterations are not exposed")
	}
	if !strings.Contains(err.Error(), "iterations") {
		t.Errorf("error should mention iterations, got %v", err)
	}
}

func TestGitLabGroupOf(t *testing.T) {
	cases := map[string]string{
		"mygroup/proj":     "mygroup",
		"a/b/c":            "a/b",
		"123":              "",
		"bareproject":      "",
	}
	for in, want := range cases {
		if got := gitlabGroupOf(in); got != want {
			t.Errorf("gitlabGroupOf(%q) = %q, want %q", in, got, want)
		}
	}
}
