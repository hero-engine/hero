// Package shell hosts the slim hero serve web-app chrome: top nav,
// page router, shared page-fragment templates, edition gating, and the
// /-redirect to a user's last-visited home. Each top-level "home" (Now,
// Work, Knowledge, Agents, People) registers itself via RegisterHome
// and owns its own handlers and templates; the shell wires routes and
// composes the outer layout.
//
// The shell is deliberately small. It does NOT host chat dispatch,
// adapters, agent runners, or home content — those belong to other
// specs. See: hero-surface-shell.
package shell

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hero-engine/hero/internal/serve/edition"
	"github.com/hero-engine/hero/internal/serve/session"
)

// Router composes the shell's mux: top-nav-bearing home routes, the
// /-redirect, the kitchen-sink dev route, and static-asset serving.
// Per-request: it builds Chrome, calls the active home's handler, and
// records last-visited home in the session store on home-root renders.
type Router struct {
	edition   edition.Edition
	store     *session.Store
	workspace string
	branch    string
	userName  string
	version   string

	tmpl *template.Template

	mu           sync.RWMutex
	homes        []Home // in registration order
	adapterProbe func() AdapterState
	// projectSelectorProbe returns the dropdown data for a given
	// request. nil hides the selector. The probe runs on every page
	// render — keep it cheap (registry read, no I/O).
	projectSelectorProbe func(*http.Request) ProjectSelector
}

// New constructs a Router for the active edition. store may be nil —
// the router will still serve pages and silently skip session writes.
func New(ed edition.Edition, store *session.Store, workspace, branch, userName, version string) *Router {
	if workspace == "" {
		workspace = "hero"
	}
	if userName == "" {
		userName = "you"
	}

	r := &Router{
		edition:   ed,
		store:     store,
		workspace: workspace,
		branch:    branch,
		userName:  userName,
		version:   version,
	}
	r.tmpl = loadTemplates()
	return r
}

// loadTemplates parses every .html under the embedded templates/
// directory into a single template set so cross-template references
// (e.g. {{ template "top-nav" . }} from page-layout.html) resolve.
func loadTemplates() *template.Template {
	t, err := template.ParseFS(templatesFS(), "*.html")
	if err != nil {
		// Bundled at compile time; a parse failure here is a bug.
		panic("shell: parse templates: " + err.Error())
	}
	return t
}

// RegisterHome adds a home to the router. Returns nil and silently
// drops the home if the active edition is not in Home.Editions.
// Panics if any of the home's route patterns collide with a previously
// registered pattern — duplicate routes are a startup-time bug.
func (r *Router) RegisterHome(h Home) error {
	if h.Slug == "" {
		return fmt.Errorf("shell: home slug required")
	}
	if h.Href == "" {
		return fmt.Errorf("shell: home %q href required", h.Slug)
	}
	if h.Render == nil {
		return fmt.Errorf("shell: home %q render required", h.Slug)
	}
	if !edition.Allowed(r.edition, h.Editions) {
		return nil // gated out — silent skip
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Duplicate-pattern check across all already-registered homes.
	seen := map[string]string{}
	for _, existing := range r.homes {
		seen[existing.Href] = existing.Slug
		for _, it := range existing.Items {
			seen[it.Pattern] = existing.Slug
		}
	}
	if owner, ok := seen[h.Href]; ok {
		panic(fmt.Sprintf("shell: duplicate route pattern %q (already registered by home %q)", h.Href, owner))
	}
	for _, it := range h.Items {
		if owner, ok := seen[it.Pattern]; ok {
			panic(fmt.Sprintf("shell: duplicate route pattern %q (already registered by home %q)", it.Pattern, owner))
		}
		seen[it.Pattern] = h.Slug
	}

	r.homes = append(r.homes, h)
	return nil
}

// Handler returns the composed http.Handler for everything the shell
// owns: the / redirect, every registered home root and item route,
// the kitchen-sink dev route, and static assets at /static/shell/.
func (r *Router) Handler() http.Handler {
	mux := http.NewServeMux()

	r.mu.RLock()
	homes := append([]Home(nil), r.homes...)
	r.mu.RUnlock()

	// / → redirect to last-visited home, default /now.
	mux.HandleFunc("/", r.handleRoot)

	// Each home: root + items.
	for _, h := range homes {
		hh := h // capture for closures
		mux.HandleFunc(hh.Href, r.wrapHomeRoot(hh))
		for _, it := range hh.Items {
			itc := it
			pattern := itc.Pattern
			// Strip optional method prefix from registration log;
			// http.ServeMux understands "GET /foo" patterns natively.
			mux.HandleFunc(pattern, itc.Render)
		}
	}

	// Kitchen-sink — always on; URL is obscure enough to avoid users.
	mux.HandleFunc("/_kitchen-sink", r.handleKitchenSink)

	return mux
}

// handleRoot redirects / to the user's last-visited home, falling
// back to /now. It does NOT match any other path; ServeMux's "/"
// pattern is greedy so we 404 anything we don't recognize.
func (r *Router) handleRoot(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	target := "/now"
	if r.store != nil {
		uid := session.UserID(req)
		if slug, ok := r.store.LastHome(uid); ok && slug != "" {
			// Find a registered home with this slug.
			r.mu.RLock()
			for _, h := range r.homes {
				if h.Slug == slug {
					target = h.Href
					break
				}
			}
			r.mu.RUnlock()
		}
	}
	http.Redirect(w, req, target, http.StatusFound)
}

// wrapHomeRoot wraps a home's root handler with the after-render
// last-home write into the session store.
func (r *Router) wrapHomeRoot(h Home) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Only treat exact-Href requests as home-root so per-item
		// children that happen to share the prefix (mounted under
		// different patterns above) are not double-counted here.
		if req.URL.Path != h.Href {
			http.NotFound(w, req)
			return
		}
		h.Render(w, req)
		if r.store != nil {
			uid := session.UserID(req)
			if err := r.store.SetLastHome(uid, h.Slug); err != nil {
				fmt.Fprintf(os.Stderr, "shell: set last home (user=%s home=%s): %v\n", uid, h.Slug, err)
			}
		}
	}
}

// RenderFragment writes a shell-owned shared fragment (e.g.
// "page-hero", "tabbed-metric-strip", "chat-input",
// "empty-state-notice") into w using the supplied data. Homes use this
// to compose their own page bodies without duplicating the fragment
// markup.
func (r *Router) RenderFragment(w io.Writer, name string, data any) error {
	if r == nil || r.tmpl == nil {
		return fmt.Errorf("shell: router not initialized")
	}
	return r.tmpl.ExecuteTemplate(w, name, data)
}

// RenderPage writes the full HTML document for a home page. Homes
// call this from their own handlers.
func (r *Router) RenderPage(w http.ResponseWriter, req *http.Request, p Page) error {
	if p.Content == nil {
		return fmt.Errorf("shell: page content required")
	}

	// Render the home's content into a buffer first so we can inline
	// it into the layout template without losing escape semantics.
	var body bytes.Buffer
	if err := p.Content(&body); err != nil {
		return fmt.Errorf("shell: render content: %w", err)
	}

	data := struct {
		PageTitle   string
		HeadExtra   template.HTML
		Chrome      Chrome
		SubNav      *SubNav
		Breadcrumb  *PageBreadcrumb
		Footer      Footer
		ContentHTML template.HTML
	}{
		PageTitle:   p.PageTitle,
		HeadExtra:   p.HeadExtra,
		Chrome:      r.buildChrome(req, p.ActiveHome),
		SubNav:      p.SubNav,
		Breadcrumb:  p.Breadcrumb,
		Footer:      r.buildFooter(),
		ContentHTML: template.HTML(body.String()),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := r.tmpl.ExecuteTemplate(w, "page-layout.html", data); err != nil {
		return fmt.Errorf("shell: execute layout: %w", err)
	}
	return nil
}

// buildChrome assembles the top-nav data for one request, including
// any badge counts (each invoked with a 50ms deadline).
func (r *Router) buildChrome(req *http.Request, activeSlug string) Chrome {
	r.mu.RLock()
	homes := append([]Home(nil), r.homes...)
	r.mu.RUnlock()

	tabs := make([]ChromeTab, 0, len(homes))
	for _, h := range homes {
		tab := ChromeTab{
			Slug:   h.Slug,
			Label:  h.Label,
			Href:   h.Href,
			Active: tabActive(req, h, activeSlug),
		}
		if h.Count != nil {
			if n, ok := fetchCountWithTimeout(req, h.Count, 50*time.Millisecond); ok {
				tab.HasCount = true
				tab.Count = n
			}
		}
		tabs = append(tabs, tab)
	}

	chrome := Chrome{
		Workspace:    r.workspace,
		Branch:       r.branch,
		UserName:     r.userName,
		UserInitials: initials(r.userName),
		Tabs:         tabs,
		Adapter:      r.resolveAdapter(),
	}
	r.mu.RLock()
	probe := r.projectSelectorProbe
	r.mu.RUnlock()
	if probe != nil {
		chrome.ProjectSelector = probe(req)
		// Rewrite tab hrefs to /p/<slug>/<page> so navigation stays
		// inside the active project. Without this, clicking a tab
		// would jump back to the legacy /<page> route and bounce
		// through the default-project redirect on every click.
		if chrome.ProjectSelector.Active != "" {
			prefix := "/p/" + chrome.ProjectSelector.Active
			for i, t := range chrome.Tabs {
				if strings.HasPrefix(t.Href, "/") && !strings.HasPrefix(t.Href, "/p/") {
					chrome.Tabs[i].Href = prefix + t.Href
				}
			}
		}
	}
	return chrome
}

// SetProjectSelectorProbe wires the per-request project-selector data
// provider into the router. Setting nil hides the selector. Safe to
// call before or after RegisterHome.
func (r *Router) SetProjectSelectorProbe(probe func(*http.Request) ProjectSelector) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.projectSelectorProbe = probe
}

// SetAdapterProbe wires a live chat-adapter probe into the router. The
// returned AdapterState drives the top-nav adapter chip on every page
// render. Setting nil reverts the chip to the muted "no adapter"
// default. Safe to call before or after RegisterHome.
//
// The probe runs synchronously on every request; keep it cheap (a
// registry lookup, no I/O). Tests can inject a static probe.
func (r *Router) SetAdapterProbe(probe func() AdapterState) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapterProbe = probe
}

// resolveAdapter calls the registered probe (if any) and returns the
// current adapter state. Returns a zero AdapterState (disconnected,
// empty DisplayName) when no probe is registered — the original
// "muted no-adapter chip" default.
func (r *Router) resolveAdapter() AdapterState {
	r.mu.RLock()
	probe := r.adapterProbe
	r.mu.RUnlock()
	if probe == nil {
		return AdapterState{}
	}
	return probe()
}

// buildFooter assembles the footer data.
func (r *Router) buildFooter() Footer {
	return Footer{
		Workspace: r.workspace,
		Version:   r.version,
		Edition:   string(r.edition),
	}
}

// tabActive decides whether a home tab should render as active.
// Active wins if either (a) the home renders us, ActiveHome matches,
// or (b) the request URL is rooted at the home's Href.
func tabActive(req *http.Request, h Home, activeSlug string) bool {
	if activeSlug != "" {
		return activeSlug == h.Slug
	}
	if req == nil || req.URL == nil {
		return false
	}
	p := req.URL.Path
	if p == h.Href {
		return true
	}
	// Prefix match — /work/spec/foo highlights /work.
	if strings.HasPrefix(p, h.Href+"/") {
		return true
	}
	return false
}

// fetchCountWithTimeout invokes a count fetcher with a deadline. On
// timeout it logs a warning and returns (_, false) so the tab renders
// unbadged.
func fetchCountWithTimeout(req *http.Request, fn func(*http.Request) (int, bool), d time.Duration) (int, bool) {
	type result struct {
		n  int
		ok bool
	}
	ctx, cancel := context.WithTimeout(req.Context(), d)
	defer cancel()
	ch := make(chan result, 1)
	go func() {
		n, ok := fn(req)
		ch <- result{n, ok}
	}()
	select {
	case r := <-ch:
		return r.n, r.ok
	case <-ctx.Done():
		fmt.Fprintf(os.Stderr, "shell: count fetcher timed out after %s\n", d)
		return 0, false
	}
}

// initials returns up to two uppercase letters from the user's name.
func initials(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "??"
	}
	fields := strings.Fields(name)
	if len(fields) == 1 {
		s := strings.ToUpper(fields[0])
		if len(s) >= 2 {
			return s[:2]
		}
		return s
	}
	out := []byte{}
	for _, f := range fields {
		if len(f) == 0 {
			continue
		}
		out = append(out, strings.ToUpper(f)[:1]...)
		if len(out) >= 2 {
			break
		}
	}
	return string(out)
}
