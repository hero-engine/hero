package work

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSpec seeds .hero/<bucket>/<slug>/spec.md under heroDir. Bucket
// is one of "specs", "planning/features", "planning/bugs",
// "planning/initiatives".
func writeSpec(t *testing.T, heroDir, bucket, slug, content string) {
	t.Helper()
	dir := filepath.Join(heroDir, bucket, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestSpecDetail_RendersFromSpecsBucket(t *testing.T) {
	heroDir := t.TempDir()
	writeSpec(t, heroDir, "specs", "shipped-feature",
		"---\ntitle: Shipped Feature\ntype: feature\nstatus: completed\nhorizon: now\n---\n\n# Shipped Feature\n\nBody with **emphasis** and a `code` span.\n")

	r := newTestRouter(t)
	if err := Register(r, Deps{HeroDir: heroDir, UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/work/spec/shipped-feature")
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
		`id="work-spec-detail"`,
		`Shipped Feature`,
		`<strong>emphasis</strong>`,
		`<code>code</code>`,
	}
	for _, want := range mustHave {
		if !strings.Contains(s, want) {
			t.Errorf("response missing %q", want)
		}
	}
}

func TestSpecDetail_RendersFromPlanningBucket(t *testing.T) {
	heroDir := t.TempDir()
	writeSpec(t, heroDir, "planning/features", "in-flight",
		"---\ntitle: In Flight\ntype: feature\nstatus: delivering\n---\n\n# In Flight\n\nPlanned work.\n")

	r := newTestRouter(t)
	if err := Register(r, Deps{HeroDir: heroDir, UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/work/spec/in-flight")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestSpecDetail_UnknownSlug_Returns404(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{HeroDir: t.TempDir(), UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/work/spec/does-not-exist")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// TestSlugRoute_DoesNotShadowSiblings ensures `/work/spec/{slug}`
// (which is a *separate* branch from /work/kanban etc.) doesn't fall
// through to the kanban / graph / blocked stubs — and conversely
// those still hit their own handlers.
func TestSlugRoute_DoesNotShadowSiblings(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{HeroDir: t.TempDir(), UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	cases := []struct {
		path     string
		mustHave string
	}{
		{"/work/kanban", "kanban-stub"},
		{"/work/graph", "graph-stub"},
		{"/work/blocked", "work-blocked"},
	}
	for _, tc := range cases {
		resp, err := http.Get(srv.URL + tc.path)
		if err != nil {
			t.Errorf("GET %s: %v", tc.path, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(body), tc.mustHave) {
			t.Errorf("GET %s: missing %q (named route should win, not spec detail)", tc.path, tc.mustHave)
		}
		if strings.Contains(string(body), `id="work-spec-detail"`) {
			t.Errorf("GET %s: spec-detail handler fired — named route should win", tc.path)
		}
	}
}

func TestWorkFilterUI_RmFiltersOnlyNoViewToolbar(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{HeroDir: t.TempDir(), UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/work")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	s := string(body)
	if !strings.Contains(s, `class="rm-filters"`) {
		t.Errorf("/work missing rm-filters row")
	}
	if strings.Contains(s, `class="view-toolbar"`) {
		t.Errorf("/work still rendering legacy view-toolbar — should be gone in polish-v2")
	}
	if strings.Contains(s, `id="work-toolbar"`) {
		t.Errorf("/work still rendering work-toolbar id — view-toolbar should be removed")
	}
}

func TestChatInput_RendersOnWorkHome(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{HeroDir: t.TempDir(), UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/work")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), `data-chat-input-variant="inline"`) {
		t.Errorf("/work missing inline chat-input")
	}
}
