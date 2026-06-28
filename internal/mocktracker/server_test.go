package mocktracker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/tracker"
)

// newTestServer starts a multi-mode mock server seeded with the embedded
// fixture, returning its base URL. Cleanup is registered via t.Cleanup
// (AC-11).
func newTestServer(t *testing.T) string {
	t.Helper()
	srv, err := NewServer(context.Background(), Options{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv)
	t.Cleanup(func() {
		hs.Close()
		srv.Close()
	})
	return hs.URL
}

func githubCfg(base string) *config.TrackerConfig {
	return &config.TrackerConfig{Type: "github", Project: "acme/widgets", Token: "tok", BaseURL: base + "/github"}
}
func gitlabCfg(base string) *config.TrackerConfig {
	return &config.TrackerConfig{Type: "gitlab", Project: "group/proj", Token: "tok", BaseURL: base + "/gitlab"}
}
func jiraCfg(base string) *config.TrackerConfig {
	return &config.TrackerConfig{Type: "jira", Project: "ACME", Token: "tok", UserEmail: "u@acme.test", BaseURL: base + "/jira"}
}
func linearCfg(base string) *config.TrackerConfig {
	return &config.TrackerConfig{Type: "linear", Project: "ACME", Token: "tok", BaseURL: base + "/linear/graphql"}
}

// AC-2: GitHub round-trip through the real adapter.
func TestRoundTrip_GitHub(t *testing.T) {
	base := newTestServer(t)
	tk, err := tracker.New(githubCfg(base))
	if err != nil {
		t.Fatal(err)
	}

	// import: open issues (5 of 6 — ACME-106 is closed)
	open, err := tk.ListIssues("", 100)
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(open) != 5 {
		t.Errorf("open issues = %d, want 5", len(open))
	}

	// field round-trip on issue 101
	before, err := tk.GetFields("101")
	if err != nil {
		t.Fatalf("GetFields: %v", err)
	}
	if before["title"].Str != "Add Apple Pay sheet" {
		t.Errorf("title = %q", before["title"].Str)
	}
	if err := tk.UpdateFields("101", map[string]tracker.Value{"title": tracker.StringValue("Changed Title")}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}
	after, _ := tk.GetFields("101")
	if after["title"].Str != "Changed Title" {
		t.Errorf("after push title = %q, want Changed Title", after["title"].Str)
	}
}

// AC-5: GitLab round-trip through the real adapter.
func TestRoundTrip_GitLab(t *testing.T) {
	base := newTestServer(t)
	tk, err := tracker.New(gitlabCfg(base))
	if err != nil {
		t.Fatal(err)
	}

	issues, err := tk.ListIssues("", 100)
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 5 {
		t.Errorf("open issues = %d, want 5", len(issues))
	}
	// type + epic linkage projected
	var found bool
	for _, is := range issues {
		if is.ID == "103" {
			found = true
			if is.IssueType != "bug" {
				t.Errorf("ACME-103 type = %q, want bug", is.IssueType)
			}
			if is.EpicKey == "" {
				t.Errorf("ACME-103 missing epic linkage")
			}
		}
	}
	if !found {
		t.Error("issue 103 not returned")
	}

	if err := tk.UpdateFields("101", map[string]tracker.Value{"title": tracker.StringValue("GL Changed")}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}
	after, _ := tk.GetFields("101")
	if after["title"].Str != "GL Changed" {
		t.Errorf("after push title = %q, want GL Changed", after["title"].Str)
	}

	// AC-5: status close
	if err := tk.UpdateStatus("102", "completed"); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	got, _ := tk.GetIssue("102")
	if got.Status != "closed" {
		t.Errorf("status after close = %q, want closed", got.Status)
	}
}

// AC-3: Jira round-trip through the real adapter.
func TestRoundTrip_Jira(t *testing.T) {
	base := newTestServer(t)
	tk, err := tracker.New(jiraCfg(base))
	if err != nil {
		t.Fatal(err)
	}
	issues, err := tk.ListIssues("", 100)
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) < 1 {
		t.Fatalf("expected issues, got %d", len(issues))
	}
	before, err := tk.GetFields("ACME-101")
	if err != nil {
		t.Fatalf("GetFields: %v", err)
	}
	if before["title"].Str != "Add Apple Pay sheet" {
		t.Errorf("title = %q", before["title"].Str)
	}
	if err := tk.UpdateFields("ACME-101", map[string]tracker.Value{"title": tracker.StringValue("Jira Changed")}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}
	after, _ := tk.GetFields("ACME-101")
	if after["title"].Str != "Jira Changed" {
		t.Errorf("after push title = %q, want Jira Changed", after["title"].Str)
	}
}

// AC-4: Linear round-trip through the real adapter.
func TestRoundTrip_Linear(t *testing.T) {
	base := newTestServer(t)
	tk, err := tracker.New(linearCfg(base))
	if err != nil {
		t.Fatal(err)
	}
	issues, err := tk.ListIssues("", 100)
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) < 1 {
		t.Fatalf("expected issues, got %d", len(issues))
	}
	before, err := tk.GetFields("ACME-101")
	if err != nil {
		t.Fatalf("GetFields: %v", err)
	}
	if before["title"].Str != "Add Apple Pay sheet" {
		t.Errorf("title = %q", before["title"].Str)
	}
	if err := tk.UpdateFields("ACME-101", map[string]tracker.Value{"title": tracker.StringValue("Linear Changed")}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}
	after, _ := tk.GetFields("ACME-101")
	if after["title"].Str != "Linear Changed" {
		t.Errorf("after push title = %q, want Linear Changed", after["title"].Str)
	}
}

// AC-6: pagination to exhaustion (server contract, github mode). per_page=2
// over 6 issues → 3 pages, Link followed, no dupes, no misses.
func TestPagination_GitHub(t *testing.T) {
	base := newTestServer(t)
	next := base + "/github/repos/acme/widgets/issues?state=all&per_page=2"
	seen := map[float64]bool{}
	pages := 0
	for next != "" {
		req, _ := http.NewRequest("GET", next, nil)
		req.Header.Set("Authorization", "Bearer tok")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var page []map[string]any
		json.NewDecoder(resp.Body).Decode(&page)
		link := resp.Header.Get("Link")
		resp.Body.Close()
		for _, is := range page {
			num := is["number"].(float64)
			if seen[num] {
				t.Errorf("duplicate issue number %v", num)
			}
			seen[num] = true
		}
		pages++
		next = parseNextLink(link)
		if pages > 10 {
			t.Fatal("pagination did not terminate")
		}
	}
	if len(seen) != 6 {
		t.Errorf("saw %d issues across pages, want 6", len(seen))
	}
	if pages != 3 {
		t.Errorf("pages = %d, want 3", pages)
	}
}

func parseNextLink(header string) string {
	for _, part := range strings.Split(header, ",") {
		segs := strings.Split(strings.TrimSpace(part), ";")
		if len(segs) < 2 {
			continue
		}
		if strings.TrimSpace(segs[1]) == `rel="next"` {
			return strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(segs[0]), "<"), ">")
		}
	}
	return ""
}

// AC-7: 429 backoff — inject one 429 on the PATCH path, then a field push
// succeeds after the adapter's single retry.
func TestAdmin_Inject429(t *testing.T) {
	base := newTestServer(t)
	adminPost(t, base, "/__admin/inject-429", map[string]any{
		"path_glob": "/github/repos/*/issues/*", "retry_after_seconds": 0, "count": 1,
	})
	tk, _ := tracker.New(githubCfg(base))
	if err := tk.UpdateFields("101", map[string]tracker.Value{"title": tracker.StringValue("After 429")}); err != nil {
		t.Fatalf("expected success after one retry, got %v", err)
	}
	after, _ := tk.GetFields("101")
	if after["title"].Str != "After 429" {
		t.Errorf("title = %q, want After 429", after["title"].Str)
	}
}

// AC-8: drift — mutate a title out-of-band, observe it on the next read.
func TestAdmin_MutateDrift(t *testing.T) {
	base := newTestServer(t)
	tk, _ := tracker.New(githubCfg(base))

	before, _ := tk.GetFields("101")
	if before["title"].Str != "Add Apple Pay sheet" {
		t.Fatalf("unexpected starting title %q", before["title"].Str)
	}
	adminPost(t, base, "/__admin/mutate", map[string]any{
		"id": "ACME-101", "field": "title", "value": "Mutated Externally",
	})
	after, _ := tk.GetFields("101")
	if after["title"].Str != "Mutated Externally" {
		t.Errorf("after mutate title = %q, want Mutated Externally", after["title"].Str)
	}
}

// AC-9: id rotation — github/gitlab (IID-based) follow the new IID;
// jira/linear (key-based) keep working by global id.
func TestAdmin_RotateIDs(t *testing.T) {
	base := newTestServer(t)
	gh, _ := tracker.New(githubCfg(base))

	// Before: addressable by IID 101.
	if _, err := gh.GetIssue("101"); err != nil {
		t.Fatalf("pre-rotate GetIssue(101): %v", err)
	}
	adminPost(t, base, "/__admin/rotate-ids", map[string]any{"id": "ACME-101", "new_id": "7777"})

	if _, err := gh.GetIssue("101"); err == nil {
		t.Error("expected old IID 101 to 404 after rotation")
	}
	if _, err := gh.GetIssue("7777"); err != nil {
		t.Errorf("post-rotate GetIssue(7777): %v", err)
	}

	// Key-based: jira still finds it by global id (no-op rotation).
	jr, _ := tracker.New(jiraCfg(base))
	adminPost(t, base, "/__admin/rotate-ids", map[string]any{"id": "ACME-102", "new_id": "ACME-102"})
	if _, err := jr.GetIssue("ACME-102"); err != nil {
		t.Errorf("jira GetIssue(ACME-102) after rotate: %v", err)
	}
}

// AC-12 guard at the source: the 404-for-everything-else contract.
func TestUnknownEndpoint404(t *testing.T) {
	base := newTestServer(t)
	req, _ := http.NewRequest("GET", base+"/github/repos/acme/widgets/pulls", nil)
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown endpoint status = %d, want 404", resp.StatusCode)
	}
}

// Auth: empty token rejected.
func TestAuthRequired(t *testing.T) {
	base := newTestServer(t)
	resp, err := http.Get(base + "/github/repos/acme/widgets/issues")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("no-token status = %d, want 401", resp.StatusCode)
	}
}

func adminPost(t *testing.T, base, path string, body map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", base+path, strings.NewReader(string(b)))
	req.Header.Set("Authorization", "Bearer admin")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("admin %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		t.Fatalf("admin %s status %d: %s", path, resp.StatusCode, msg)
	}
}
