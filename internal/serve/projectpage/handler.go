package projectpage

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"

	"github.com/hero-engine/hero/internal/serve/projectpage/data"
	"github.com/hero-engine/hero/internal/serve/shell"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Register installs the Project section page on the shell router.
//
// The page mounts at `/project` per the dashboard-project-url-scheme
// decision. In the multi-project routing model, this same page also
// renders at `/p/<slug>/project` — the shell router is one-per-project
// and rewrites the inner path so /p/<slug>/project arrives here as
// /project.
func Register(r *shell.Router, deps Deps) error {
	tmpl, err := loadTemplates()
	if err != nil {
		return fmt.Errorf("projectpage: load templates: %w", err)
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
		Render: h.handle,
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
	return template.New("").ParseFS(sub, "*.html")
}

// pageData composes every section's typed result into the outer
// template input. Each field aligns with the partial it feeds.
type pageData struct {
	Identity   data.Identity
	Health     data.Health
	Operations data.Operations
	Stack      data.Stack
	Registry   data.Registry
	Peers      data.Peers
	Trackers   data.Trackers
	Knowledge  data.Knowledge
	Config     data.Config
}

func (h *handler) handle(w http.ResponseWriter, req *http.Request) {
	d := h.resolveDeps(req)
	pd := h.buildPageData(d)

	subhead := h.composeSubhead(d, pd)
	hero := shell.PageHero{
		Eyebrow: template.HTML("Project · read-only"),
		Title:   pd.Identity.Name,
		Subhead: template.HTML(subhead),
	}

	content := func(out io.Writer) error {
		if err := h.router.RenderFragment(out, "page-hero", hero); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "page.html", pd)
	}

	page := shell.Page{
		ActiveHome: "project",
		PageTitle:  pd.Identity.Name + " · Project · Hero",
		Content:    content,
		HeadExtra:  template.HTML(projectStyles + projectScript(d.Slug)),
	}
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "projectpage: render: "+err.Error(), http.StatusInternalServerError)
	}
}

// resolveDeps returns the per-request Deps. In multi-project mode the
// shell router carries the per-project Deps already; in single-project
// mode (the /project fallback) Deps.Slug may be empty and we fall back
// to the project root's basename.
func (h *handler) resolveDeps(_ *http.Request) Deps {
	d := h.deps
	if d.Slug == "" && d.ProjectRoot != "" {
		d.Slug = filepath.Base(d.ProjectRoot)
	}
	return d
}

func (h *handler) buildPageData(d Deps) pageData {
	var regView *data.RegistryEntryView
	if d.RegistryEntry != nil {
		regView = &data.RegistryEntryView{
			Path:         d.RegistryEntry.Path,
			RegisteredAt: d.RegistryEntry.RegisteredAt,
		}
	}
	return pageData{
		Identity: data.LoadIdentity(data.IdentityInputs{
			ProjectRoot: d.ProjectRoot, HeroDir: d.HeroDir, Slug: d.Slug,
		}),
		Health: data.LoadHealth(data.HealthInputs{HeroDir: d.HeroDir}),
		Operations: data.LoadOperations(data.OperationsInputs{
			Slug:      d.Slug,
			Lookup:    d.OpsRunner,
			Available: d.OpsRunner != nil,
		}),
		Stack: data.LoadStack(data.StackInputs{
			ProjectRoot: d.ProjectRoot, HeroDir: d.HeroDir,
		}),
		Registry: data.LoadRegistry(data.RegistryInputs{
			Slug: d.Slug, Entry: regView, IsDefaultProject: d.IsDefaultProject,
		}),
		Peers: data.LoadPeers(data.PeersInputs{
			ProjectRoot: d.ProjectRoot, HeroDir: d.HeroDir,
		}),
		Trackers: data.LoadTrackers(data.TrackersInputs{
			ProjectRoot: d.ProjectRoot, HeroDir: d.HeroDir,
		}),
		Knowledge: data.LoadKnowledge(data.KnowledgeInputs{HeroDir: d.HeroDir}),
		Config: data.LoadConfig(data.ConfigInputs{
			ProjectRoot: d.ProjectRoot, HeroDir: d.HeroDir,
		}),
	}
}

// composeSubhead builds the small page-hero subhead summarising the
// section state — "12 specs · 4 conventions · health: never" kind of
// thing. All values are HTML-escaped through template.HTMLEscapeString.
func (h *handler) composeSubhead(d Deps, pd pageData) string {
	specPart := fmt.Sprintf("<strong>%d specs</strong>", pd.Identity.SpecCount)
	convPart := fmt.Sprintf("<strong>%d conventions</strong>", pd.Stack.ActiveConventions)
	healthPart := "<strong>health: never</strong>"
	if pd.Health.HasArtifact {
		if pd.Health.AllClear {
			healthPart = "<strong>health: all clear</strong>"
		} else {
			healthPart = fmt.Sprintf("<strong>health: %s</strong>", template.HTMLEscapeString(pd.Health.CapturedAtPretty))
		}
	}
	_ = d
	return specPart + ` <span class="dot-sep">·</span> ` + convPart + ` <span class="dot-sep">·</span> ` + healthPart
}

const projectStyles = `<style>
.project-section { border: 1px solid var(--border); border-radius: 6px; margin: 12px 0; }
.project-section-head { display: flex; align-items: center; gap: 12px; padding: 10px 14px; border-bottom: 1px solid var(--border); }
.project-section-head h2 { margin: 0; font-size: 1rem; }
.project-section-head .project-section-meta { color: var(--ink-muted); font-size: 0.85rem; }
.project-section-head .project-section-meta.muted { color: var(--ink-muted); font-style: italic; }
.project-section-head .project-section-link { margin-left: auto; }
.project-section-head .project-section-toggle { margin-left: auto; cursor: pointer; background: none; border: 1px solid var(--border); border-radius: 4px; padding: 2px 8px; font-size: 0.8rem; }
.project-section-head[data-default-collapsed="true"] .project-section-link { margin-left: 0; }
.project-section-body { padding: 14px; }
.project-meta { display: grid; grid-template-columns: auto 1fr; column-gap: 16px; row-gap: 4px; margin: 0; }
.project-meta dt { font-weight: 600; }
.project-meta dd { margin: 0; }
.project-empty { color: var(--ink-muted); font-style: italic; margin: 0; }
.project-peers { width: 100%; border-collapse: collapse; }
.project-peers th, .project-peers td { padding: 6px 8px; text-align: left; border-bottom: 1px solid var(--border); }
.project-peers .muted { color: var(--ink-muted); }
.project-health-clear { color: var(--success); font-weight: 600; margin: 0; }
.project-health-rows { list-style: none; padding: 0; margin: 0; }
.project-health-row { padding: 6px 0; border-bottom: 1px solid var(--border); display: flex; gap: 12px; }
.project-health-row.status-fail .status { color: var(--danger); }
.project-health-row.status-warn .status { color: var(--warn); }
.project-config-paths { font-size: 0.85rem; color: var(--ink-muted); margin: 0 0 8px 0; }
.project-config-paths .muted { margin-left: 8px; }
.project-config-body { background: var(--bg-soft); padding: 10px; border-radius: 4px; overflow-x: auto; font-size: 0.85rem; }
.project-tracker-optin p { margin: 4px 0; }
</style>`

// projectScript renders the inline collapse-toggle JS with the slug
// baked into the localStorage key. Tiny — full file lives at
// internal/serve/projectpage/static/project.js for editing convenience
// but is inlined here so we don't fan out a /static/projectpage/
// route in Phase 1.
func projectScript(slug string) string {
	safeSlug := template.JSEscapeString(slug)
	return `<script>(function(){
  var slug=` + jsonString(safeSlug) + `;
  var heads=document.querySelectorAll('.project-section-head');
  heads.forEach(function(head){
    var section=head.closest('.project-section');
    if(!section) return;
    var name=section.getAttribute('data-section')||'';
    var key='hero-projectpage:'+slug+':'+name;
    var body=section.querySelector('.project-section-body');
    var btn=head.querySelector('.project-section-toggle');
    var defaultCollapsed=(head.getAttribute('data-default-collapsed')==='true');
    var stored=null;
    try{stored=localStorage.getItem(key);}catch(e){}
    var collapsed=(stored===null)?defaultCollapsed:(stored==='1');
    apply();
    if(btn){
      btn.addEventListener('click',function(){
        collapsed=!collapsed;
        try{localStorage.setItem(key, collapsed?'1':'0');}catch(e){}
        apply();
      });
    }
    function apply(){
      if(!body) return;
      if(collapsed){body.setAttribute('hidden','');}else{body.removeAttribute('hidden');}
      if(btn){btn.setAttribute('aria-expanded', collapsed?'false':'true');btn.textContent=collapsed?'expand':'collapse';}
    }
  });
})();</script>`
}

// jsonString wraps a string for safe JS embedding. We avoid pulling in
// encoding/json for a single-string case to keep the handler import
// surface small.
func jsonString(s string) string {
	return "\"" + s + "\""
}
