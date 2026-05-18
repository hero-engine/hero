package shell

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/serve/edition"
	"github.com/hero-engine/hero/internal/serve/session"
)

// stubHandler is a tiny handler that records that it was invoked.
type stubHandler struct{ hit bool }

func (h *stubHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.hit = true
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "ok")
}

func newTestRouter(t *testing.T, ed edition.Edition) (*Router, *session.Store) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "shell-sessions.db")
	store, err := session.Open(dbPath)
	if err != nil {
		t.Fatalf("open session store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return New(ed, store, "hero", "main", "Ben Wheeler", "test"), store
}

func TestRegisterHome_FiltersByEdition(t *testing.T) {
	r, _ := newTestRouter(t, edition.CE)

	// Home gated to cloud + enterprise — should be silently dropped.
	gated := Home{
		Slug:     "people",
		Label:    "People",
		Href:     "/people",
		Render:   func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) },
		Editions: []string{"cloud", "enterprise"},
	}
	if err := r.RegisterHome(gated); err != nil {
		t.Fatalf("register gated home: %v", err)
	}

	// Ungated home must register.
	open := Home{
		Slug:   "now",
		Label:  "Now",
		Href:   "/now",
		Render: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) },
	}
	if err := r.RegisterHome(open); err != nil {
		t.Fatalf("register open home: %v", err)
	}

	// Direct hit on gated URL is 404.
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/people")
	if err != nil {
		t.Fatalf("get /people: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("gated /people status = %d, want 404", resp.StatusCode)
	}

	// Open URL works.
	resp, err = http.Get(srv.URL + "/now")
	if err != nil {
		t.Fatalf("get /now: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("open /now status = %d, want 200", resp.StatusCode)
	}

	// Chrome built from request shows only the open tab.
	chrome := r.buildChrome(httptest.NewRequest("GET", "/now", nil), "")
	if len(chrome.Tabs) != 1 {
		t.Fatalf("tabs count = %d, want 1 (gated home filtered)", len(chrome.Tabs))
	}
	if chrome.Tabs[0].Slug != "now" {
		t.Fatalf("tab slug = %q, want now", chrome.Tabs[0].Slug)
	}
}

func TestTabActivation_MatchesPrefix(t *testing.T) {
	r, _ := newTestRouter(t, edition.Local)
	for _, h := range []Home{
		{Slug: "now", Label: "Now", Href: "/now", Render: nopHandler},
		{Slug: "work", Label: "Work", Href: "/work", Render: nopHandler},
	} {
		if err := r.RegisterHome(h); err != nil {
			t.Fatalf("register %s: %v", h.Slug, err)
		}
	}

	cases := []struct {
		path       string
		activeWant string
	}{
		{"/now", "now"},
		{"/work", "work"},
		{"/work/spec/foo", "work"},
		{"/knowledge", ""},
	}
	for _, c := range cases {
		req := httptest.NewRequest("GET", c.path, nil)
		chrome := r.buildChrome(req, "")
		got := ""
		for _, tab := range chrome.Tabs {
			if tab.Active {
				got = tab.Slug
			}
		}
		if got != c.activeWant {
			t.Errorf("path %q active = %q, want %q", c.path, got, c.activeWant)
		}
	}
}

func TestRootRedirect_FallsBackToNow(t *testing.T) {
	r, _ := newTestRouter(t, edition.Local)
	if err := r.RegisterHome(Home{
		Slug: "now", Label: "Now", Href: "/now", Render: nopHandler,
	}); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(r.Handler())
	defer srv.Close()
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("get /: %v", err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("/ status = %d, want 302", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/now" {
		t.Fatalf("Location = %q, want /now", loc)
	}
}

func TestRootRedirect_UsesRecordedLastHome(t *testing.T) {
	r, store := newTestRouter(t, edition.Local)
	for _, h := range []Home{
		{Slug: "now", Label: "Now", Href: "/now", Render: nopHandler},
		{Slug: "work", Label: "Work", Href: "/work", Render: nopHandler},
	} {
		if err := r.RegisterHome(h); err != nil {
			t.Fatal(err)
		}
	}

	// Pre-seed last home for the test user.
	uid := "test-user"
	if err := store.SetLastHome(uid, "work"); err != nil {
		t.Fatalf("seed last home: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Hero-User", uid)
	rr := httptest.NewRecorder()
	r.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/work" {
		t.Fatalf("Location = %q, want /work", loc)
	}
}

func TestRenderPage_WritesLastHomeOnHomeRoot(t *testing.T) {
	r, store := newTestRouter(t, edition.Local)
	uid := "test-user"

	rendered := false
	home := Home{
		Slug:  "now",
		Label: "Now",
		Href:  "/now",
		Render: func(w http.ResponseWriter, req *http.Request) {
			rendered = true
			page := Page{
				ActiveHome: "now",
				PageTitle:  "Now · Hero",
				Content: func(out io.Writer) error {
					_, err := io.WriteString(out, `<p>now</p>`)
					return err
				},
			}
			if err := r.RenderPage(w, req, page); err != nil {
				t.Errorf("render page: %v", err)
			}
		},
	}
	if err := r.RegisterHome(home); err != nil {
		t.Fatal(err)
	}

	// Home-root render: SetLastHome should be called.
	req := httptest.NewRequest("GET", "/now", nil)
	req.Header.Set("X-Hero-User", uid)
	rr := httptest.NewRecorder()
	r.Handler().ServeHTTP(rr, req)
	if !rendered {
		t.Fatalf("home handler not invoked")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `<nav class="topnav">`) {
		t.Errorf("rendered body missing topnav; got %s", body[:min(200, len(body))])
	}
	last, ok := store.LastHome(uid)
	if !ok || last != "now" {
		t.Errorf("last home = (%q, %v), want (now, true)", last, ok)
	}
}

func TestRenderPage_DoesNotWriteLastHomeFromItem(t *testing.T) {
	r, store := newTestRouter(t, edition.Local)
	uid := "test-user-2"

	rendered := false
	home := Home{
		Slug:   "work",
		Label:  "Work",
		Href:   "/work",
		Render: nopHandler,
		Items: []ItemRoute{{
			Pattern: "/work/spec/",
			Render: func(w http.ResponseWriter, req *http.Request) {
				rendered = true
				// Item handler renders WITHOUT going through the
				// home-root wrapper that records last_home.
				page := Page{
					ActiveHome: "work",
					PageTitle:  "Spec · Hero",
					Content:    func(out io.Writer) error { return nil },
				}
				if err := r.RenderPage(w, req, page); err != nil {
					t.Errorf("render page: %v", err)
				}
			},
		}},
	}
	if err := r.RegisterHome(home); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/work/spec/foo", nil)
	req.Header.Set("X-Hero-User", uid)
	rr := httptest.NewRecorder()
	r.Handler().ServeHTTP(rr, req)
	if !rendered {
		t.Fatalf("item handler not invoked")
	}
	if _, ok := store.LastHome(uid); ok {
		t.Errorf("last home should be unrecorded after per-item render")
	}
}

func TestRegisterHome_DuplicatePatternPanics(t *testing.T) {
	r, _ := newTestRouter(t, edition.Local)
	if err := r.RegisterHome(Home{
		Slug: "now", Label: "Now", Href: "/now", Render: nopHandler,
	}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatalf("expected panic on duplicate route pattern")
		}
	}()
	_ = r.RegisterHome(Home{
		Slug: "other", Label: "Other", Href: "/now", Render: nopHandler,
	})
}

func TestRegisterHome_DuplicateItemPatternPanics(t *testing.T) {
	r, _ := newTestRouter(t, edition.Local)
	if err := r.RegisterHome(Home{
		Slug: "work", Label: "Work", Href: "/work", Render: nopHandler,
		Items: []ItemRoute{{Pattern: "/work/spec/{slug}", Render: nopHandler}},
	}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatalf("expected panic on duplicate item route pattern")
		}
	}()
	_ = r.RegisterHome(Home{
		Slug: "knowledge", Label: "Knowledge", Href: "/knowledge", Render: nopHandler,
		Items: []ItemRoute{{Pattern: "/work/spec/{slug}", Render: nopHandler}},
	})
}

func TestInitials(t *testing.T) {
	cases := map[string]string{
		"":            "??",
		"Ben Wheeler": "BW",
		"alice":       "AL",
		"a":           "A",
		"three word name": "TW",
	}
	for in, want := range cases {
		if got := initials(in); got != want {
			t.Errorf("initials(%q) = %q, want %q", in, got, want)
		}
	}
}

func nopHandler(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

// Helper invoked by the kitchen-sink test to confirm the route renders
// without error in the default registration.
func TestKitchenSink_Renders(t *testing.T) {
	r, _ := newTestRouter(t, edition.Local)
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/_kitchen-sink")
	if err != nil {
		t.Fatalf("get kitchen-sink: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("kitchen-sink status = %d, want 200; body: %s", resp.StatusCode, body)
	}
}

// Ensure all stub homes register and render successfully end-to-end.
func TestStubHomes_AllRender(t *testing.T) {
	r, _ := newTestRouter(t, edition.Local)
	RegisterStubHomes(r)
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	for _, slug := range []string{"now", "work", "knowledge", "agents", "people"} {
		resp, err := http.Get(fmt.Sprintf("%s/%s", srv.URL, slug))
		if err != nil {
			t.Fatalf("get /%s: %v", slug, err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("stub /%s status = %d, want 200; body: %s", slug, resp.StatusCode, body)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), `<nav class="topnav">`) {
			t.Errorf("stub /%s body missing topnav", slug)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
