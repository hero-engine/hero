package tracker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestGitHub_SearchConformance asserts each SearchQuery field maps onto
// the native GitHub list-endpoint params, with IssueType and Priority
// mapped onto the type-label / priority-label conventions.
func TestGitHub_SearchConformance(t *testing.T) {
	var gotURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer srv.Close()
	g, _ := newGitHub("acme/widgets", "tok", srv.URL)

	_, err := g.Search(SearchQuery{
		Status:    "closed",
		Priority:  "High",
		Assignee:  "alice",
		Labels:    []string{"area::checkout"},
		IssueType: "Bug",
		OrderBy:   "created ASC",
		Limit:     40,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	qp := gotURL.Query()
	if qp.Get("state") != "closed" {
		t.Errorf("state = %q, want closed", qp.Get("state"))
	}
	if qp.Get("assignee") != "alice" {
		t.Errorf("assignee = %q, want alice", qp.Get("assignee"))
	}
	if qp.Get("sort") != "created" || qp.Get("direction") != "asc" {
		t.Errorf("sort/direction = %q/%q, want created/asc", qp.Get("sort"), qp.Get("direction"))
	}
	labels := qp.Get("labels")
	for _, want := range []string{"area::checkout", "bug", "priority::high"} {
		if !strings.Contains(labels, want) {
			t.Errorf("labels %q missing %q", labels, want)
		}
	}
}

// TestGitHub_SearchUnassigned maps the unassigned sentinel to
// assignee=none.
func TestGitHub_SearchUnassigned(t *testing.T) {
	var gotURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer srv.Close()
	g, _ := newGitHub("acme/widgets", "tok", srv.URL)
	if _, err := g.Search(SearchQuery{Assignee: "unassigned"}); err != nil {
		t.Fatal(err)
	}
	if gotURL.Query().Get("assignee") != "none" {
		t.Errorf("assignee = %q, want none", gotURL.Query().Get("assignee"))
	}
}

// TestGitHub_TypeInference classifies GitHub issues by label (no native
// type), consistent with the other adapters.
func TestGitHub_TypeInference(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"number": 1, "title": "crash", "state": "open",
				"labels": []map[string]string{{"name": "bug"}}},
			{"number": 2, "title": "roadmap", "state": "open",
				"labels": []map[string]string{{"name": "acme-type::initiative"}}},
			{"number": 3, "title": "just a story", "state": "open",
				"labels": []map[string]string{{"name": "area::x"}}},
		})
	}))
	defer srv.Close()
	g, _ := newGitHub("acme/widgets", "tok", srv.URL)
	issues, err := g.Search(SearchQuery{})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"1": "bug", "2": "initiative", "3": "story"}
	for _, iss := range issues {
		if iss.IssueType != want[iss.ID] {
			t.Errorf("#%s: IssueType = %q, want %q", iss.ID, iss.IssueType, want[iss.ID])
		}
	}
}

// TestListIssues_ClassifiesTypes asserts the broad ListIssues path
// (used by a no-config plain import) sets Issue.IssueType from labels on
// all three label-based adapters — so a plain import classifies
// bug/epic/feature rather than importing everything as feature.
func TestListIssues_ClassifiesTypes(t *testing.T) {
	t.Run("linear", func(t *testing.T) {
		nodes := []interface{}{
			linearNode("ENG-1", "crash", "Bug"),
			linearNode("ENG-2", "roadmap", "acme-type::initiative"),
			linearNode("ENG-3", "plain", "team::x"),
		}
		srv := linearIssuesServer(t, nodes, nil)
		defer srv.Close()
		l, _ := newLinear("ENG", "tok", srv.URL)
		issues, err := l.ListIssues("", 10)
		if err != nil {
			t.Fatal(err)
		}
		assertTypes(t, issues, map[string]string{"ENG-1": "bug", "ENG-2": "initiative", "ENG-3": "story"})
	})

	t.Run("github", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"number": 1, "title": "crash", "state": "open",
					"labels": []map[string]string{{"name": "bug"}}},
				{"number": 2, "title": "plain", "state": "open",
					"labels": []map[string]string{{"name": "area::x"}}},
			})
		}))
		defer srv.Close()
		g, _ := newGitHub("acme/widgets", "tok", srv.URL)
		issues, err := g.ListIssues("", 10)
		if err != nil {
			t.Fatal(err)
		}
		assertTypes(t, issues, map[string]string{"1": "bug", "2": "story"})
	})

	t.Run("gitlab", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]gitLabIssue{
				{IID: 1, Title: "crash", State: "opened", Labels: []string{"type::bug"}},
				{IID: 2, Title: "plain", State: "opened", Labels: []string{"team::x"}},
			})
		}))
		defer srv.Close()
		g, _ := newGitLab("group/proj", "tok", srv.URL)
		issues, err := g.ListIssues("", 10)
		if err != nil {
			t.Fatal(err)
		}
		assertTypes(t, issues, map[string]string{"1": "bug", "2": "story"})
	})
}

func assertTypes(t *testing.T, issues []Issue, want map[string]string) {
	t.Helper()
	if len(issues) != len(want) {
		t.Fatalf("got %d issues, want %d", len(issues), len(want))
	}
	for _, iss := range issues {
		if iss.IssueType != want[iss.ID] {
			t.Errorf("%s: IssueType = %q, want %q", iss.ID, iss.IssueType, want[iss.ID])
		}
	}
}

// TestGitLab_SearchConformance asserts each SearchQuery field maps onto
// the native GitLab list params, with IssueType and Priority mapped onto
// the type-label / priority-label conventions.
func TestGitLab_SearchConformance(t *testing.T) {
	var gotURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL
		json.NewEncoder(w).Encode([]gitLabIssue{})
	}))
	defer srv.Close()
	g, _ := newGitLab("group/proj", "tok", srv.URL)

	_, err := g.Search(SearchQuery{
		Status:    "closed",
		Priority:  "critical",
		Assignee:  "bob",
		Labels:    []string{"team::payments"},
		IssueType: "Bug",
		OrderBy:   "updated DESC",
		Limit:     15,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	qp := gotURL.Query()
	if qp.Get("state") != "closed" {
		t.Errorf("state = %q, want closed", qp.Get("state"))
	}
	if qp.Get("assignee_username") != "bob" {
		t.Errorf("assignee_username = %q, want bob", qp.Get("assignee_username"))
	}
	if qp.Get("order_by") != "updated_at" {
		t.Errorf("order_by = %q, want updated_at", qp.Get("order_by"))
	}
	labels := qp.Get("labels")
	for _, want := range []string{"team::payments", "bug", "priority::critical"} {
		if !strings.Contains(labels, want) {
			t.Errorf("labels %q missing %q", labels, want)
		}
	}
}

// TestGitLab_SearchUnassigned maps the unassigned sentinel to
// assignee_id=None.
func TestGitLab_SearchUnassigned(t *testing.T) {
	var gotURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL
		json.NewEncoder(w).Encode([]gitLabIssue{})
	}))
	defer srv.Close()
	g, _ := newGitLab("group/proj", "tok", srv.URL)
	if _, err := g.Search(SearchQuery{Assignee: "none"}); err != nil {
		t.Fatal(err)
	}
	if gotURL.Query().Get("assignee_id") != "None" {
		t.Errorf("assignee_id = %q, want None", gotURL.Query().Get("assignee_id"))
	}
}

// TestGitLab_TypeInferenceEpicLabel classifies GitLab issues by label
// for epic/initiative parity with the other adapters (native issue_type
// only covers issue/incident/task).
func TestGitLab_TypeInferenceEpicLabel(t *testing.T) {
	cases := []struct {
		gitlabType string
		labels     []string
		want       string
	}{
		{"issue", []string{"type::bug"}, "bug"},
		{"incident", nil, "bug"},
		{"issue", []string{"epic"}, "epic"},
		{"issue", []string{"acme-type::initiative"}, "initiative"},
		{"issue", []string{"team::x"}, "story"},
	}
	for _, c := range cases {
		if got := heroIssueType(c.gitlabType, c.labels); got != c.want {
			t.Errorf("heroIssueType(%q, %v) = %q, want %q", c.gitlabType, c.labels, got, c.want)
		}
	}
}

// TestTypeFromLabels covers the shared recognition set directly.
func TestTypeFromLabels(t *testing.T) {
	cases := []struct {
		labels []string
		want   string
	}{
		{[]string{"Bug"}, "bug"},
		{[]string{"type::bug"}, "bug"},
		{[]string{"kind/bug"}, "bug"},
		{[]string{"acme-type::epic"}, "epic"},
		{[]string{"epic"}, "epic"},
		{[]string{"acme-type::initiative"}, "initiative"},
		{[]string{"priority::p0", "team::x"}, ""},
		{nil, ""},
	}
	for _, c := range cases {
		if got := typeFromLabels(c.labels); got != c.want {
			t.Errorf("typeFromLabels(%v) = %q, want %q", c.labels, got, c.want)
		}
	}
}
