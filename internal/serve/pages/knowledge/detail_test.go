package knowledge

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeKnowledgeEntry seeds .hero/knowledge/notes/<slug>.md under dir.
func writeKnowledgeEntry(t *testing.T, dir, slug, content string) string {
	t.Helper()
	notesDir := filepath.Join(dir, "knowledge", "notes")
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(notesDir, slug+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write entry: %v", err)
	}
	return path
}

func TestEntryDetail_RendersRealEntry(t *testing.T) {
	heroDir := t.TempDir()
	writeKnowledgeEntry(t, heroDir, "my-note",
		"---\ntitle: My Note\ntype: note\nstatus: active\ncreated: 2026-05-01\nrelates-to: [other-note]\n---\n\n# My Note\n\nFirst paragraph with **bold** and a `code` span.\n\n- bullet one\n- bullet two\n")

	r := newTestRouter(t)
	if err := Register(r, Deps{HeroDir: heroDir, UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/knowledge/my-note")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	mustHave := []string{
		`id="knowledge-entry-detail"`,
		`My Note`,                  // title in hero
		`<strong>bold</strong>`,    // markdown rendered
		`<code>code</code>`,        // inline code
		`<li>bullet one</li>`,      // bullet list
		`other-note`,               // related entry chip
	}
	for _, want := range mustHave {
		if !strings.Contains(s, want) {
			t.Errorf("response missing %q", want)
		}
	}
}

func TestEntryDetail_UnknownSlug_Returns404(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{HeroDir: t.TempDir(), UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/knowledge/does-not-exist")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// TestSlugRoute_DoesNotShadowNamedRoutes asserts that the
// /knowledge/{slug} wildcard does not pre-empt the specifically
// registered /knowledge/search, /why, /staleness, /recent, /write
// routes. Go 1.22 ServeMux gives precedence to specifically-registered
// patterns over wildcards — this test guards that contract.
func TestSlugRoute_DoesNotShadowNamedRoutes(t *testing.T) {
	heroDir := t.TempDir()
	// Seed an entry whose slug collides with a named sub-route so a
	// regression would actually exercise the collision.
	writeKnowledgeEntry(t, heroDir, "search", "# Search\n\nBody.\n")
	writeKnowledgeEntry(t, heroDir, "why", "# Why\n\nBody.\n")

	r := newTestRouter(t)
	if err := Register(r, Deps{HeroDir: heroDir, UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	cases := []struct {
		path    string
		marker  string // expected on the named-route page
		notFrom string // would only appear if slug detail fired
	}{
		{"/knowledge/search", "knowledge-search-stub", "knowledge-entry-detail"},
		{"/knowledge/why", "knowledge-provenance", "knowledge-entry-detail"},
		{"/knowledge/staleness", "knowledge-staleness", "knowledge-entry-detail"},
		{"/knowledge/recent", "knowledge-recent-stub", "knowledge-entry-detail"},
		{"/knowledge/write", "knowledge-write-stub", "knowledge-entry-detail"},
	}
	for _, tc := range cases {
		resp, err := http.Get(srv.URL + tc.path)
		if err != nil {
			t.Errorf("GET %s: %v", tc.path, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		s := string(body)
		if !strings.Contains(s, tc.marker) {
			t.Errorf("GET %s: missing named-route marker %q", tc.path, tc.marker)
		}
		if strings.Contains(s, tc.notFrom) {
			t.Errorf("GET %s: detail handler fired (found %q) — named route should win", tc.path, tc.notFrom)
		}
	}
}

func TestChatInput_RendersOnKnowledgeHome(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{HeroDir: t.TempDir(), UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/knowledge")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), `data-chat-input-variant="inline"`) {
		t.Errorf("/knowledge missing inline chat-input")
	}
}
