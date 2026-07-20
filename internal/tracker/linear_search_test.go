package tracker

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// linearNode builds a Linear issue node with the given identifier, title,
// and labels for the mock issues connection response.
func linearNode(identifier, title string, labels ...string) map[string]interface{} {
	labelNodes := make([]interface{}, len(labels))
	for i, l := range labels {
		labelNodes[i] = map[string]interface{}{"name": l}
	}
	return map[string]interface{}{
		"identifier": identifier,
		"title":      title,
		"url":        "https://linear.app/team/" + identifier,
		"createdAt":  "2026-07-19T10:20:30.123Z",
		"updatedAt":  "2026-07-20T11:22:33.456Z",
		"state":      map[string]interface{}{"name": "Todo"},
		"labels":     map[string]interface{}{"nodes": labelNodes},
	}
}

// linearIssuesServer returns a test server that answers the Search
// issues connection query with the given nodes and records the last
// GraphQL query string it saw (for conformance assertions).
func linearIssuesServer(t *testing.T, nodes []interface{}, lastQuery *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload struct {
			Query string `json:"query"`
		}
		_ = json.Unmarshal(body, &payload)
		if lastQuery != nil {
			*lastQuery = payload.Query
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"team": map[string]interface{}{
					"issues": map[string]interface{}{"nodes": nodes},
				},
			},
		})
	}))
}

// TestLinear_TypeInference asserts that Linear issues are classified as
// bug/epic/initiative/feature from their labels (Linear has no native
// issue type), consistent with the seeder label conventions.
func TestLinear_TypeInference(t *testing.T) {
	nodes := []interface{}{
		linearNode("ENG-1", "A crash", "Bug", "priority::p0"),
		linearNode("ENG-2", "type-scoped bug", "type::bug"),
		linearNode("ENG-3", "An epic", "epic"),
		linearNode("ENG-4", "acme epic", "acme-type::epic"),
		linearNode("ENG-5", "An initiative", "acme-type::initiative"),
		linearNode("ENG-6", "A plain story", "team::payments"),
		linearNode("ENG-7", "No labels"),
	}
	srv := linearIssuesServer(t, nodes, nil)
	defer srv.Close()

	l, _ := newLinear("ENG", "tok", srv.URL)
	issues, err := l.Search(SearchQuery{Limit: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	want := map[string]string{
		"ENG-1": "bug",
		"ENG-2": "bug",
		"ENG-3": "epic",
		"ENG-4": "epic",
		"ENG-5": "initiative",
		"ENG-6": "story",
		"ENG-7": "story",
	}
	if len(issues) != len(want) {
		t.Fatalf("got %d issues, want %d", len(issues), len(want))
	}
	for _, iss := range issues {
		if got := iss.IssueType; got != want[iss.ID] {
			t.Errorf("%s: IssueType = %q, want %q", iss.ID, got, want[iss.ID])
		}
		if iss.CreatedAt != "2026-07-19T10:20:30.123Z" || iss.UpdatedAt != "2026-07-20T11:22:33.456Z" {
			t.Errorf("%s: timestamps = %q/%q, want native Linear values", iss.ID, iss.CreatedAt, iss.UpdatedAt)
		}
	}
}

// TestLinear_SearchConformance asserts each SearchQuery field the
// adapter claims to support is translated to the native GraphQL query.
func TestLinear_SearchConformance(t *testing.T) {
	var q string
	srv := linearIssuesServer(t, []interface{}{}, &q)
	defer srv.Close()
	l, _ := newLinear("ENG", "tok", srv.URL)

	_, err := l.Search(SearchQuery{
		Status:    "In Progress",
		Priority:  "high",
		Assignee:  "alice",
		Labels:    []string{"team::payments"},
		IssueType: "Bug",
		OrderBy:   "updated DESC",
		Limit:     25,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	checks := map[string]string{
		"status→state name":      `state: { name: { eqIgnoreCase: "In Progress" } }`,
		"priority→numeric high":  `priority: { eq: 2 }`,
		"assignee→displayName":   `assignee: { displayName: { eq: "alice" } }`,
		"issue_type→bug label":   `"bug"`,
		"user label present":     `"team::payments"`,
		"orderBy→updatedAt":      `orderBy: updatedAt`,
		"created activity field": "createdAt",
		"updated activity field": "updatedAt",
	}
	for name, frag := range checks {
		if !strings.Contains(q, frag) {
			t.Errorf("%s: query missing %q\n---\n%s", name, frag, q)
		}
	}
}

// TestLinear_SearchStatusAll drops the default state exclusion when
// Status is "all".
func TestLinear_SearchStatusAll(t *testing.T) {
	var q string
	srv := linearIssuesServer(t, []interface{}{}, &q)
	defer srv.Close()
	l, _ := newLinear("ENG", "tok", srv.URL)
	if _, err := l.Search(SearchQuery{Status: "all"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(q, "completed") {
		t.Errorf("status=all should drop the completed/canceled exclusion, got:\n%s", q)
	}
}
