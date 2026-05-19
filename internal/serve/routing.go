package serve

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/hero-engine/hero/internal/serve/shell"
)

// projectRouterCache lazily builds a shell.Router per project slug.
// One router instance per project lets the homes capture project-
// specific Deps via closures (matching the existing pattern) without
// rebuilding the router on every request.
//
// AllProjectsSlug is the reserved slug that routes to the cross-
// project aggregate views (`/p/all/...`). Item 7/8 work fills in the
// aggregate renderers; this phase wires the routing seam.
type projectRouterCache struct {
	mu      sync.Mutex
	routers map[string]*shell.Router
	build   func(pc *ProjectContext) *shell.Router
}

// AllProjectsSlug is the reserved URL slug for the cross-project
// aggregate view. Reserved so a real project can't register under this
// name and collide.
const AllProjectsSlug = "all"

// legacyPagePaths are the bare page URLs that existed before the
// /p/<slug>/<page> rewrite. Requests at these exact paths redirect to
// the default project's namespaced URL so existing bookmarks keep
// working.
var legacyPagePaths = []string{"/now", "/work", "/knowledge", "/people", "/agents", "/project"}

// ActiveProjectCookie is the cookie name client JS writes when the user
// switches projects via the top-nav dropdown. Server-side fallbacks for
// the default project read this when present.
const ActiveProjectCookie = "hero_active_project"

func newProjectRouterCache(build func(pc *ProjectContext) *shell.Router) *projectRouterCache {
	return &projectRouterCache{
		routers: map[string]*shell.Router{},
		build:   build,
	}
}

func (c *projectRouterCache) get(pc *ProjectContext) *shell.Router {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.routers[pc.Slug]; ok {
		return r
	}
	r := c.build(pc)
	c.routers[pc.Slug] = r
	return r
}

// projectHandler returns the http.Handler that dispatches requests
// under /p/<slug>/... to a project-scoped shell router. Unknown slugs
// 404 with a list of registered projects. The reserved slug "all"
// returns an aggregate-view stub — items 7 and 8 fill in the per-page
// aggregate renderers.
func (s *Server) projectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip /p/ prefix and split the slug.
		path := strings.TrimPrefix(r.URL.Path, "/p/")
		if path == "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		slug, rest := splitFirst(path, "/")
		if slug == "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		if slug == AllProjectsSlug {
			s.allProjectsHandler(w, r, rest)
			return
		}

		pc := s.GetProject(slug)
		if pc == nil {
			s.renderUnknownProject(w, r, slug)
			return
		}

		// Rewrite the request so the per-project shell router sees the
		// inner path (e.g. /now) instead of /p/<slug>/now.
		inner := "/" + rest
		if inner == "/" {
			// /p/<slug>/ → redirect to the project's default landing
			// page (/now).
			http.Redirect(w, r, "/p/"+slug+"/now", http.StatusFound)
			return
		}

		// Build a shallow-copied request with the rewritten path so the
		// inner shell router can be reused unchanged. We don't mutate r
		// in place — http handlers further up the chain may still hold
		// a reference.
		r2 := r.Clone(r.Context())
		newURL := *r.URL
		newURL.Path = inner
		newURL.RawPath = ""
		r2.URL = &newURL

		router := s.projectRouterCacheRouter(pc)
		router.Handler().ServeHTTP(w, r2)
	})
}

// ensure net/url is imported (used by inner request rewriting).
var _ = url.Parse

// resolveDefaultProjectSlug picks the project slug to land on when a
// request arrives without one. Precedence:
//
//  1. The hero_active_project cookie (set by client JS on selection).
//  2. The slug matching the daemon's primary project root (the one the
//     daemon was launched from), since "open the dashboard" from a
//     project's terminal usually means "open this project."
//  3. The first registered project in alphabetical order.
func (s *Server) resolveDefaultProjectSlug(r *http.Request) string {
	if c, err := r.Cookie(ActiveProjectCookie); err == nil && c.Value != "" {
		if s.GetProject(c.Value) != nil || c.Value == AllProjectsSlug {
			return c.Value
		}
	}
	// Primary project (the daemon's launching directory).
	if s.projectRoot != "" {
		s.mu.RLock()
		for slug, pc := range s.projects {
			if pc.Path == s.projectRoot {
				s.mu.RUnlock()
				return slug
			}
		}
		s.mu.RUnlock()
	}
	// Alphabetical first.
	slugs := s.Projects()
	if len(slugs) == 0 {
		return ""
	}
	sort.Strings(slugs)
	return slugs[0]
}

// renderUnknownProject writes a friendly 404 listing the registered
// project slugs so users can spot a typo without crawling docs.
func (s *Server) renderUnknownProject(w http.ResponseWriter, r *http.Request, badSlug string) {
	slugs := s.Projects()
	sort.Strings(slugs)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "unknown project %q\n\nregistered projects:\n", badSlug)
	for _, sl := range slugs {
		fmt.Fprintf(w, "  - %s\n", sl)
	}
	if len(slugs) == 0 {
		fmt.Fprintln(w, "  (none — register one with `hero serve --add .`)")
	}
}

// allProjectsHandler is the stub for the cross-project aggregate
// views. Routing accepts /p/all/<page> and renders a placeholder so
// items 7 and 8 can fill in per-page aggregate renderers without
// touching this file. The shape of MultiProject Deps is the seam those
// items consume.
func (s *Server) allProjectsHandler(w http.ResponseWriter, r *http.Request, rest string) {
	if rest == "" {
		http.Redirect(w, r, "/p/all/now", http.StatusFound)
		return
	}
	page, _ := splitFirst(rest, "/")
	// People aggregate is intentionally absent — the spec calls for an
	// empty-state prompt because team membership semantics aren't
	// settled. Items 7/8 wire this through the page's existing empty-
	// state renderer; for now we render plain text.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if page == "people" {
		fmt.Fprintf(w, `<!doctype html><meta charset=utf-8><title>All projects · People</title>`+
			`<h1>People &amp; ROI</h1><p>Pick a project — cross-project team views are coming.</p>`+
			`<p><a href="/p/all/now">Back to All projects · Now</a></p>`)
		return
	}
	fmt.Fprintf(w, `<!doctype html><meta charset=utf-8><title>All projects · %s</title>`+
		`<h1>All projects · %s</h1>`+
		`<p>Cross-project %s view — coming in the dashboard-redesign and project-section work.</p>`+
		`<p>%d project(s) registered.</p>`+
		`<p><a href="/">Back</a></p>`,
		page, page, page, s.ProjectCount())
}
