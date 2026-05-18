// Package project hosts the Project home — the project-shape rollup
// at GET /project. The home renders the surfaces table, active
// initiatives, recently-completed work, what's next, open risks,
// and (when archives exist) a timeline strip of dated archives.
//
// Per the project-snapshot spec, archive bodies render ONLY at the
// dedicated /project/snapshots/<date> route. The timeline strip on
// /project shows date + trigger + label only — never archive body
// content. See snapshot.archive containment invariants.
package project

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/serve/shell"
	"github.com/hero-engine/hero/internal/snapshot"
	"github.com/hero-engine/hero/internal/spec"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Deps is the small, injectable bundle of state the Project handler
// reads. Mirrors the other home Deps shapes so wiring stays
// symmetric.
type Deps struct {
	// ProjectRoot is the absolute path to the project being served.
	ProjectRoot string
	// HeroDir is the absolute path to .hero/ inside ProjectRoot.
	HeroDir string
	// Workspace is the human label shown in the eyebrow / footer chrome.
	Workspace string
	// Branch is the current git branch ("" when not in a git checkout).
	Branch string
	// UserName is the display name used in personalization strings.
	UserName string
}

// Register installs the Project home on the shell router. The
// archive view (/project/snapshots/<date>) lives under this home as
// an item route so the surface containment story is "one home, one
// place archives can render."
func Register(r *shell.Router, deps Deps) error {
	tmpl, err := loadTemplates()
	if err != nil {
		return fmt.Errorf("project: load templates: %w", err)
	}
	h := &handler{
		router: r,
		tmpl:   tmpl,
		deps:   deps,
	}
	return r.RegisterHome(shell.Home{
		Slug:   "project",
		Label:  "Project",
		Href:   "/project",
		Render: h.renderHome,
		Items: []shell.ItemRoute{
			{Pattern: "GET /project/surface/{id...}", Render: h.renderSurface},
			{Pattern: "GET /project/snapshots/{date}", Render: h.renderArchive},
		},
	})
}

type handler struct {
	router *shell.Router
	tmpl   *template.Template
	deps   Deps
}

func loadTemplates() (*template.Template, error) {
	sub, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		return nil, err
	}
	return template.New("").Funcs(funcMap()).ParseFS(sub, "*.html")
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"join": strings.Join,
	}
}

// pageData is the top-level rendering context for /project.
type pageData struct {
	Snapshot   *snapshot.Snapshot
	Archives   []snapshot.ArchiveRecord
	HasArchive bool
}

func (h *handler) renderHome(w http.ResponseWriter, req *http.Request) {
	snap, err := h.buildSnapshot()
	if err != nil {
		http.Error(w, "project snapshot: "+err.Error(), http.StatusInternalServerError)
		return
	}
	archives, _ := snapshot.List(h.deps.HeroDir)

	hero := shell.PageHero{
		Eyebrow: template.HTML("Project shape · projected from graph"),
		Title:   "Project",
		Subhead: template.HTML(fmt.Sprintf("%d surfaces · %d specs · %s",
			len(snap.Surfaces), snap.SourceNodes, snap.GeneratedAt.Format("2006-01-02 15:04"))),
	}

	content := func(out io.Writer) error {
		if err := h.router.RenderFragment(out, "page-hero", hero); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "project-home", pageData{
			Snapshot:   snap,
			Archives:   archives,
			HasArchive: len(archives) > 0,
		})
	}

	page := shell.Page{
		ActiveHome: "project",
		PageTitle:  "Project · Hero",
		Content:    content,
	}
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "render project home: "+err.Error(), http.StatusInternalServerError)
	}
}

func (h *handler) renderSurface(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	if id == "" {
		http.NotFound(w, req)
		return
	}
	snap, err := h.buildSnapshot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var found *snapshot.Surface
	for i := range snap.Surfaces {
		if snap.Surfaces[i].ID == id {
			found = &snap.Surfaces[i]
			break
		}
	}
	if found == nil {
		http.NotFound(w, req)
		return
	}
	var specs []spec.Spec
	for _, a := range snap.Assignments {
		if a.SurfaceID == id && a.Spec != nil {
			specs = append(specs, *a.Spec)
		}
	}

	hero := shell.PageHero{
		Eyebrow: template.HTML("Surface detail"),
		Title:   id,
		Subhead: template.HTML(fmt.Sprintf("Stage: <strong>%s</strong> · %d specs",
			found.Stage, len(specs))),
	}

	content := func(out io.Writer) error {
		if err := h.router.RenderFragment(out, "page-hero", hero); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "surface-detail", map[string]any{
			"Surface": found,
			"Specs":   specs,
		})
	}

	page := shell.Page{
		ActiveHome: "project",
		PageTitle:  id + " · Project · Hero",
		Content:    content,
		Breadcrumb: &shell.PageBreadcrumb{
			Crumbs: []shell.BreadcrumbCrumb{
				{Label: "Project", Href: "/project"},
				{Label: id, Current: true},
			},
		},
	}
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderArchive serves the dedicated archive read view. This is the
// ONLY route that exposes archive body content; the timeline strip
// on /project (and every other listing) renders only metadata.
func (h *handler) renderArchive(w http.ResponseWriter, req *http.Request) {
	date := req.PathValue("date")
	if date == "" {
		http.NotFound(w, req)
		return
	}
	rec, err := snapshot.FindArchive(h.deps.HeroDir, date)
	if err != nil || rec == nil {
		http.NotFound(w, req)
		return
	}

	hero := shell.PageHero{
		Eyebrow: template.HTML("Historical archive · read-only"),
		Title:   rec.Date,
		Subhead: template.HTML(fmt.Sprintf("Trigger: <strong>%s</strong>%s · git_commit: <code>%s</code>",
			rec.Trigger,
			labelSpan(rec.Label),
			rec.GitCommit)),
	}

	content := func(out io.Writer) error {
		if err := h.router.RenderFragment(out, "page-hero", hero); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "archive", map[string]any{
			"Archive": rec,
			"Body":    template.HTML(htmlEscape(rec.Body)),
			"Filename": filepath.Base(rec.Path),
		})
	}

	page := shell.Page{
		ActiveHome: "project",
		PageTitle:  rec.Date + " · Archive · Project · Hero",
		Content:    content,
		Breadcrumb: &shell.PageBreadcrumb{
			Crumbs: []shell.BreadcrumbCrumb{
				{Label: "Project", Href: "/project"},
				{Label: "Archives", Href: "/project"},
				{Label: rec.Date, Current: true},
			},
		},
	}
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func labelSpan(label string) string {
	if label == "" {
		return ""
	}
	return fmt.Sprintf(" · label: <strong>%s</strong>", template.HTMLEscapeString(label))
}

func htmlEscape(s string) string {
	// Render as preformatted text inside <pre>. Escape only the
	// dangerous chars so the body reads cleanly.
	return template.HTMLEscapeString(s)
}

func (h *handler) buildSnapshot() (*snapshot.Snapshot, error) {
	allSpecs, _ := spec.Discover(h.deps.HeroDir)
	override, _ := snapshot.LoadOverride(h.deps.HeroDir)
	return snapshot.Build(snapshot.BuildOptions{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
		ProjectName: filepath.Base(h.deps.ProjectRoot),
	}, allSpecs, override, nil)
}
