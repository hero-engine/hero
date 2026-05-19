package serve

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newMultiProjectServer constructs a Server with two registered
// projects, both pointed at minimal .hero workspaces. Reuses the same
// test helpers as server_test.go.
func newMultiProjectServer(t *testing.T) *Server {
	t.Helper()
	heroDir, projectRoot := setupTestWorkspace(t)
	srv := NewServer(ServerConfig{
		HeroDir:     heroDir,
		ProjectRoot: projectRoot,
		Version:     "test",
		Port:        7437,
		AutoWatch:   false,
		UIEnabled:   true,
	})
	// Add a second project so the /p/<slug>/... routes have somewhere
	// to dispatch to and the selector has a real choice.
	heroDir2, projectRoot2 := setupTestWorkspace(t)
	_ = srv.AddProject("project-beta", projectRoot2, heroDir2, false)
	return srv
}

func TestProjectHandler_UnknownSlug404(t *testing.T) {
	srv := newMultiProjectServer(t)
	req := httptest.NewRequest(http.MethodGet, "/p/no-such-slug/now", nil)
	rr := httptest.NewRecorder()
	srv.projectHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "unknown project") {
		t.Errorf("body should mention 'unknown project': %s", body)
	}
	if !strings.Contains(body, "registered projects:") {
		t.Errorf("body should list registered projects: %s", body)
	}
}

func TestProjectHandler_AllProjectsRoute(t *testing.T) {
	srv := newMultiProjectServer(t)
	req := httptest.NewRequest(http.MethodGet, "/p/all/now", nil)
	rr := httptest.NewRecorder()
	srv.projectHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "All projects") {
		t.Errorf("body should contain 'All projects': %s", rr.Body.String())
	}
}

func TestProjectHandler_AllProjectsPeopleEmptyState(t *testing.T) {
	srv := newMultiProjectServer(t)
	req := httptest.NewRequest(http.MethodGet, "/p/all/people", nil)
	rr := httptest.NewRecorder()
	srv.projectHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Pick a project") {
		t.Errorf("people aggregate should render the pick-a-project empty state: %s", rr.Body.String())
	}
}

func TestProjectHandler_SlugRootRedirectsToNow(t *testing.T) {
	srv := newMultiProjectServer(t)
	// /p/<slug>/ (trailing slash) should redirect to /p/<slug>/now.
	slugs := srv.Projects()
	if len(slugs) == 0 {
		t.Fatal("no projects in server")
	}
	slug := slugs[0]

	req := httptest.NewRequest(http.MethodGet, "/p/"+slug+"/", nil)
	rr := httptest.NewRecorder()
	srv.projectHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rr.Code)
	}
	loc := rr.Header().Get("Location")
	want := "/p/" + slug + "/now"
	if loc != want {
		t.Errorf("Location = %q, want %q", loc, want)
	}
}

func TestResolveDefaultProjectSlug_CookiePrecedence(t *testing.T) {
	srv := newMultiProjectServer(t)
	// Cookie naming a registered project wins.
	req := httptest.NewRequest(http.MethodGet, "/now", nil)
	req.AddCookie(&http.Cookie{Name: ActiveProjectCookie, Value: "project-beta"})

	got := srv.resolveDefaultProjectSlug(req)
	if got != "project-beta" {
		t.Errorf("cookie precedence: got %q, want project-beta", got)
	}
}

func TestResolveDefaultProjectSlug_AllProjectsCookie(t *testing.T) {
	srv := newMultiProjectServer(t)
	req := httptest.NewRequest(http.MethodGet, "/now", nil)
	req.AddCookie(&http.Cookie{Name: ActiveProjectCookie, Value: AllProjectsSlug})

	got := srv.resolveDefaultProjectSlug(req)
	if got != AllProjectsSlug {
		t.Errorf("all-projects cookie: got %q, want %q", got, AllProjectsSlug)
	}
}

func TestResolveDefaultProjectSlug_BogusCookieFallsThrough(t *testing.T) {
	srv := newMultiProjectServer(t)
	req := httptest.NewRequest(http.MethodGet, "/now", nil)
	req.AddCookie(&http.Cookie{Name: ActiveProjectCookie, Value: "no-such-project"})

	got := srv.resolveDefaultProjectSlug(req)
	if got == "no-such-project" {
		t.Errorf("bogus cookie should not be returned; got %q", got)
	}
	// Should be one of the registered projects.
	found := false
	for _, s := range srv.Projects() {
		if s == got {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("fallback returned unregistered slug: %q", got)
	}
}

func TestCurrentPageFromRequest(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/p/foo/now", "now"},
		{"/p/foo/work", "work"},
		{"/now", "now"},
		{"/work/spec/something", "work"},
		{"/", "now"},
		{"/p/foo/", "now"},
		{"/p/foo", "now"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		got := currentPageFromRequest(req, "foo")
		if got != c.want {
			t.Errorf("currentPageFromRequest(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestProjectSelectorFor_OptionsIncludeAllAndProjects(t *testing.T) {
	srv := newMultiProjectServer(t)
	probe := srv.projectSelectorFor("hero")
	req := httptest.NewRequest(http.MethodGet, "/p/hero/now", nil)
	sel := probe(req)

	if sel.Active != "hero" {
		t.Errorf("Active = %q, want hero", sel.Active)
	}
	if len(sel.Options) < 2 {
		t.Fatalf("Options len = %d, want at least 2", len(sel.Options))
	}
	if sel.Options[0].Slug != AllProjectsSlug {
		t.Errorf("first option = %q, want %q", sel.Options[0].Slug, AllProjectsSlug)
	}
	if sel.Options[0].Label != "All projects" {
		t.Errorf("first option label = %q, want 'All projects'", sel.Options[0].Label)
	}

	// Other options should include the two registered projects.
	hasBeta := false
	for _, o := range sel.Options[1:] {
		if o.Slug == "project-beta" {
			hasBeta = true
		}
	}
	if !hasBeta {
		t.Errorf("selector options missing project-beta: %+v", sel.Options)
	}
}
