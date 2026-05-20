package projectpage

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"

	"github.com/hero-engine/hero/internal/serve/projectpage/data"
	"github.com/hero-engine/hero/internal/serve/shell"
)

// AggregateDeps is the per-request bundle the cross-project /p/all/project
// page needs. The aggregate handler is a sibling of the Phase 1 per-
// project handler — it lives in the same package so the shared template
// set, styles, and section partials are reused.
//
// Projects is the live registry snapshot (built by the server on every
// request). DaemonSnapshot is the same underlying view the /api/status
// endpoint emits — passed directly so the loader does not have to make
// an HTTP round-trip.
type AggregateDeps struct {
	Projects       []data.DirectoryProject
	DaemonSnapshot func() *data.DaemonOpsSnapshot
}

// RegisterAggregate installs the cross-project /p/all/project page on
// the aggregate shell router. The aggregate router is built per request
// by Server.buildAggregateProjectRouter so the Projects slice is always
// fresh.
func RegisterAggregate(r *shell.Router, deps AggregateDeps) error {
	tmpl, err := loadTemplates()
	if err != nil {
		return fmt.Errorf("projectpage: load aggregate templates: %w", err)
	}
	h := &aggregateHandler{router: r, tmpl: tmpl, deps: deps}
	return r.RegisterHome(shell.Home{
		Slug:   "project",
		Label:  "Project",
		Href:   "/project",
		Render: h.handle,
	})
}

type aggregateHandler struct {
	router *shell.Router
	tmpl   *template.Template
	deps   AggregateDeps
}

// aggregatePageData composes all four sections for the cross-project
// view.
type aggregatePageData struct {
	Directory    data.Directory
	DaemonOps    data.DaemonOps
	HealthRollup data.HealthRollup
	PeersMap     data.PeersMap
}

func (h *aggregateHandler) handle(w http.ResponseWriter, req *http.Request) {
	pd := h.buildPageData()
	hero := shell.PageHero{
		Eyebrow: template.HTML("All projects · operator view"),
		Title:   "All projects",
		Subhead: template.HTML(h.composeSubhead(pd)),
	}
	content := func(out io.Writer) error {
		if err := h.router.RenderFragment(out, "page-hero", hero); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "page_all.html", pd)
	}
	page := shell.Page{
		ActiveHome: "project",
		PageTitle:  "All projects · Project · Hero",
		Content:    content,
		HeadExtra:  template.HTML(projectStyles + aggregateStyles + aggregateScript()),
	}
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "projectpage: aggregate render: "+err.Error(), http.StatusInternalServerError)
	}
}

// buildPageData fans out the four section loaders. Each loader is
// already isolation-safe (per-project failure folds into a degraded
// row); we still wrap loader invocations in a defer/recover layer at
// this seam so a panic in any one loader cannot poison the whole page.
func (h *aggregateHandler) buildPageData() (pd aggregatePageData) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(httpStderr(), "projectpage: aggregate loader panic: %v\n", r)
		}
	}()

	pd.Directory = safeLoadDirectory(h.deps.Projects)
	pd.HealthRollup = safeLoadHealthRollup(h.deps.Projects)
	pd.PeersMap = safeLoadPeersMap(h.deps.Projects)
	var snap *data.DaemonOpsSnapshot
	if h.deps.DaemonSnapshot != nil {
		snap = h.deps.DaemonSnapshot()
	}
	pd.DaemonOps = data.LoadDaemonOps(data.DaemonOpsInputs{Snapshot: snap})
	return pd
}

// composeSubhead summarises the page state in one line.
func (h *aggregateHandler) composeSubhead(pd aggregatePageData) string {
	count := len(pd.Directory.Rows)
	colour := pd.HealthRollup.OverallColor
	if colour == "" {
		colour = "unknown"
	}
	return fmt.Sprintf("<strong>%d projects</strong> <span class=\"dot-sep\">·</span> health: <strong>%s</strong>",
		count, template.HTMLEscapeString(colour))
}

// safeLoadDirectory wraps LoadDirectory in a recover() so a panic in
// any per-project sub-loader degrades to an empty result instead of
// 500ing the whole page. The per-row loader itself already folds path
// failures into degraded rows; this is the belt to the suspenders.
func safeLoadDirectory(projects []data.DirectoryProject) (out data.Directory) {
	defer func() { _ = recover() }()
	return data.LoadDirectory(data.DirectoryInputs{Projects: projects})
}

func safeLoadHealthRollup(projects []data.DirectoryProject) (out data.HealthRollup) {
	defer func() { _ = recover() }()
	return data.LoadHealthRollup(data.HealthRollupInputs{Projects: projects})
}

func safeLoadPeersMap(projects []data.DirectoryProject) (out data.PeersMap) {
	defer func() { _ = recover() }()
	return data.LoadPeersMap(data.PeersMapInputs{Projects: projects})
}

// httpStderr returns the stderr writer for the panic log. Wrapped so
// tests can stub it.
var httpStderr = func() io.Writer { return os.Stderr }

//go:embed static/project_all.js
var aggregateJS string

func aggregateScript() string {
	return "<script>" + aggregateJS + "</script>"
}

// aggregateStyles adds the small, page-local styles the aggregate page
// uses on top of projectStyles. Kept inline so the page does not depend
// on a /static/projectpage/ route.
const aggregateStyles = `<style>
.health-dot { font-size: 0.85rem; line-height: 1; }
.health-dot.health-green { color: var(--success); }
.health-dot.health-yellow { color: var(--warn); }
.health-dot.health-red { color: var(--danger); }
.health-dot.health-unknown { color: var(--ink-muted); }
.project-directory-controls { margin-bottom: 8px; }
.project-directory-filter { padding: 4px 8px; border: 1px solid var(--border); border-radius: 4px; min-width: 220px; }
.project-directory-degraded { background: var(--bg-soft); }
.project-directory th { user-select: none; }
.project-health-drill { margin-top: 12px; }
.project-health-drill summary { cursor: pointer; padding: 4px 0; color: var(--ink-muted); }
</style>`
