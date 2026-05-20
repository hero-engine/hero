package projectpage

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
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
	Danger     data.Danger

	// MissingPath is true when the project's registered root does not
	// exist on disk at render time. Drives the top-of-page deregister
	// banner. Only set on /p/<slug>/project URLs — the single-project
	// /project fallback (whose path is the daemon's cwd) never sets
	// this, so the banner doesn't fire for the daemon's home project
	// just because cwd changed. Phase 4 of hero-serve-project-section.
	MissingPath bool

	// Slug is the project slug, surfaced into the missing-path banner
	// markup so the deregister button can post to the right URL.
	Slug string
}

func (h *handler) handle(w http.ResponseWriter, req *http.Request) {
	d := h.resolveDeps(req)
	pd := h.buildPageData(d, req)

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

func (h *handler) buildPageData(d Deps, req *http.Request) pageData {
	var regView *data.RegistryEntryView
	if d.RegistryEntry != nil {
		regView = &data.RegistryEntryView{
			Path:         d.RegistryEntry.Path,
			RegisteredAt: d.RegistryEntry.RegisteredAt,
		}
	}
	registry := data.LoadRegistry(data.RegistryInputs{
		Slug: d.Slug, Entry: regView, IsDefaultProject: d.IsDefaultProject,
	})
	pd := pageData{
		Slug: d.Slug,
		Identity: data.LoadIdentity(data.IdentityInputs{
			ProjectRoot: d.ProjectRoot, HeroDir: d.HeroDir, Slug: d.Slug,
		}),
		Health: data.LoadHealth(data.HealthInputs{HeroDir: d.HeroDir, Slug: d.Slug, Cache: d.HealthCache}),
		Operations: data.LoadOperations(data.OperationsInputs{
			Slug:      d.Slug,
			Lookup:    d.OpsRunner,
			Available: d.OpsRunner != nil,
		}),
		Stack: data.LoadStack(data.StackInputs{
			ProjectRoot: d.ProjectRoot, HeroDir: d.HeroDir,
		}),
		Registry: registry,
		Peers: data.LoadPeers(data.PeersInputs{
			ProjectRoot: d.ProjectRoot, HeroDir: d.HeroDir, Slug: d.Slug, Cache: d.PeerCache,
		}),
		Trackers: data.LoadTrackers(data.TrackersInputs{
			ProjectRoot: d.ProjectRoot, HeroDir: d.HeroDir,
		}),
		Knowledge: data.LoadKnowledge(data.KnowledgeInputs{HeroDir: d.HeroDir}),
		Config: data.LoadConfig(data.ConfigInputs{
			ProjectRoot: d.ProjectRoot, HeroDir: d.HeroDir,
		}),
		Danger: data.LoadDanger(data.DangerInputs{
			Slug:       d.Slug,
			Registered: registry.Registered,
		}),
	}
	pd.MissingPath = h.shouldShowMissingPathBanner(d, req)
	return pd
}

// shouldShowMissingPathBanner reports whether the missing-path banner
// should fire for this request. The banner renders only when the
// project is being served through a per-project route AND its
// registered root no longer exists on disk. The bare /project fallback
// (daemon's cwd) is intentionally exempt — banner-on-cwd is bad UX.
func (h *handler) shouldShowMissingPathBanner(d Deps, _ *http.Request) bool {
	if d.IsFallbackProject {
		return false
	}
	if d.ProjectRoot == "" {
		return false
	}
	if _, err := os.Stat(d.ProjectRoot); err != nil {
		if os.IsNotExist(err) {
			return true
		}
	}
	return false
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
.project-missing-path-banner { background: var(--bg-soft); border: 1px solid var(--danger); border-radius: 4px; padding: 10px 14px; margin: 12px 0; display: flex; align-items: center; gap: 12px; }
.project-missing-path-banner button { margin-left: auto; }
.project-registry-actions { margin: 12px 0 0 0; display: flex; align-items: center; gap: 12px; }
.project-registry-remove { background: var(--bg-soft); border: 1px solid var(--danger); color: var(--danger); padding: 4px 10px; border-radius: 4px; cursor: pointer; }
.project-registry-remove[disabled] { opacity: 0.5; cursor: not-allowed; }
.project-section-danger .project-section-head { background: var(--bg-soft); }
.project-danger-verbs { list-style: none; padding: 0; margin: 0; }
.project-danger-verb { padding: 10px 0; border-bottom: 1px solid var(--border); }
.project-danger-verb:last-child { border-bottom: none; }
.project-danger-head { display: flex; flex-direction: column; gap: 2px; margin-bottom: 6px; }
.project-danger-form { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.project-danger-form label { display: flex; align-items: center; gap: 6px; font-size: 0.9rem; }
.project-danger-confirm { padding: 4px 8px; border: 1px solid var(--border); border-radius: 4px; font-family: monospace; min-width: 220px; }
.project-danger-submit { padding: 4px 10px; border: 1px solid var(--danger); background: var(--bg-soft); color: var(--danger); border-radius: 4px; cursor: pointer; }
.project-danger-submit[disabled] { opacity: 0.5; cursor: not-allowed; }
.hero-undo-toast { position: fixed; top: 12px; left: 50%; transform: translateX(-50%); background: var(--bg-soft); border: 1px solid var(--border); border-radius: 4px; padding: 8px 14px; box-shadow: 0 2px 8px rgba(0,0,0,0.15); z-index: 1000; display: flex; align-items: center; gap: 10px; }
.hero-undo-btn { background: none; border: 1px solid var(--border); padding: 2px 8px; border-radius: 4px; cursor: pointer; }
.project-health-stale { background: var(--bg-soft); color: var(--warn); font-size: 0.75rem; border: 1px solid var(--warn); border-radius: 4px; padding: 1px 6px; margin-left: 6px; }
.project-health-refresh, .project-peer-probe { background: none; border: 1px solid var(--border); border-radius: 4px; padding: 2px 8px; font-size: 0.8rem; cursor: pointer; margin-left: 8px; }
.project-health-refresh[aria-busy="true"], .project-peer-probe[aria-busy="true"] { opacity: 0.6; cursor: progress; }
.project-health-output { background: var(--bg-soft); padding: 8px; border-radius: 4px; font-size: 0.8rem; max-height: 240px; overflow: auto; margin: 0 0 10px 0; }
</style>`

// projectScript renders the inline collapse-toggle JS with the slug
// baked into the localStorage key. Tiny — full file lives at
// internal/serve/projectpage/static/project.js for editing convenience
// but is inlined here so we don't fan out a /static/projectpage/
// route in Phase 1.
//
// Phase 4 of hero-serve-project-section appended the destructive-ops
// wiring: registry-remove with undo toast, Danger Zone typed-confirm
// gate, and missing-path banner deregister.
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

  // ---- Phase 4: registry-remove with 5-second undo toast ----
  function postJSON(url){
    return fetch(url,{method:'POST',headers:{'Accept':'application/json'}});
  }
  function showUndoToast(slugVal){
    // Remove any prior toast.
    var existing=document.getElementById('hero-undo-toast');
    if(existing) existing.parentNode.removeChild(existing);
    var toast=document.createElement('div');
    toast.id='hero-undo-toast';
    toast.setAttribute('role','status');
    toast.className='hero-undo-toast';
    var msg=document.createElement('span');
    msg.textContent='Removing '+slugVal+' in ';
    var counter=document.createElement('strong');
    counter.textContent='5';
    var undoBtn=document.createElement('button');
    undoBtn.type='button';
    undoBtn.className='hero-undo-btn';
    undoBtn.textContent='Undo';
    msg.appendChild(counter);
    msg.appendChild(document.createTextNode('s. '));
    toast.appendChild(msg);
    toast.appendChild(undoBtn);
    document.body.appendChild(toast);
    var remaining=5;
    var cancelled=false;
    var timer=setInterval(function(){
      remaining-=1;
      if(remaining<=0){
        clearInterval(timer);
        if(!cancelled){
          toast.textContent=slugVal+' removed.';
          setTimeout(function(){window.location.reload();},400);
        }
        return;
      }
      counter.textContent=String(remaining);
    },1000);
    undoBtn.addEventListener('click',function(){
      cancelled=true;
      clearInterval(timer);
      postJSON('/api/'+encodeURIComponent(slugVal)+'/registry/remove/undo').finally(function(){
        toast.textContent='Cancelled.';
        setTimeout(function(){if(toast.parentNode) toast.parentNode.removeChild(toast);},800);
      });
    });
  }
  function bindRemoveButton(btn){
    if(!btn||btn.dataset.removeBound==='1') return;
    btn.dataset.removeBound='1';
    btn.addEventListener('click',function(ev){
      ev.preventDefault();
      var slugVal=btn.getAttribute('data-slug')||slug;
      btn.disabled=true;
      postJSON('/api/'+encodeURIComponent(slugVal)+'/registry/remove').then(function(resp){
        if(!resp.ok) throw new Error('remove failed: '+resp.status);
        showUndoToast(slugVal);
      }).catch(function(err){
        btn.disabled=false;
        console.error('registry remove failed',err);
      });
    });
  }
  Array.prototype.forEach.call(
    document.querySelectorAll('.project-registry-remove'),
    bindRemoveButton
  );

  // ---- Phase 5: Health "Refresh now" + Peers "Probe" wiring ----
  function fetchAndReplaceHealth(){
    return fetch('/api/'+encodeURIComponent(slug)+'/health',{headers:{'Accept':'application/json'}})
      .then(function(resp){return resp.ok?resp.json():null;})
      .then(function(payload){if(!payload) return; renderHealthSummary(payload);});
  }
  function renderHealthSummary(payload){
    var section=document.querySelector('section[data-section="health"]');
    if(!section) return;
    var meta=section.querySelector('.project-section-head .project-section-meta');
    if(meta){
      if(payload.captured_at){
        meta.textContent='as of '+payload.captured_at;
        meta.classList.remove('muted');
        meta.title=payload.captured_at;
      }
    }
    // Toggle stale chip.
    var head=section.querySelector('.project-section-head');
    if(head){
      var existing=head.querySelector('.project-health-stale');
      if(payload.stale){
        if(!existing){
          var chip=document.createElement('span');
          chip.className='project-health-stale';
          chip.textContent='stale';
          if(meta) meta.insertAdjacentElement('afterend',chip);
        }
      } else if(existing){
        existing.parentNode.removeChild(existing);
      }
    }
  }
  Array.prototype.forEach.call(
    document.querySelectorAll('[data-action="refresh-health"]'),
    function(btn){
      btn.addEventListener('click',function(){
        if(btn.hasAttribute('disabled')) return;
        btn.setAttribute('disabled','');
        btn.setAttribute('aria-busy','true');
        var section=btn.closest('section[data-section="health"]');
        var out=section?section.querySelector('.project-health-output'):null;
        if(out){out.textContent='';out.removeAttribute('hidden');}
        fetch('/api/'+encodeURIComponent(slug)+'/health/refresh',{method:'POST'})
          .then(function(resp){
            if(!resp.ok) throw new Error('refresh failed: '+resp.status);
            return resp.json();
          })
          .then(function(body){
            if(!body||!body.job_id) throw new Error('missing job_id');
            var es=new EventSource('/api/'+encodeURIComponent(slug)+'/ops/'+encodeURIComponent(body.job_id)+'/stream');
            es.addEventListener('progress',function(ev){
              if(!out) return;
              try{var f=JSON.parse(ev.data); if(f&&f.text){out.textContent+=f.text+'\n'; out.scrollTop=out.scrollHeight;}}catch(e){}
            });
            es.addEventListener('exit',function(){
              es.close();
              // Tiny grace window so the background-goroutine cache
              // update lands before we re-read /health.
              setTimeout(function(){
                fetchAndReplaceHealth().finally(function(){
                  btn.removeAttribute('disabled');
                  btn.removeAttribute('aria-busy');
                });
              },150);
            });
            es.addEventListener('error',function(){
              if(es.readyState===EventSource.CLOSED){
                btn.removeAttribute('disabled');
                btn.removeAttribute('aria-busy');
              }
            });
          })
          .catch(function(err){
            console.error('health refresh failed',err);
            btn.removeAttribute('disabled');
            btn.removeAttribute('aria-busy');
          });
      });
    }
  );

  Array.prototype.forEach.call(
    document.querySelectorAll('[data-action="probe-peer"]'),
    function(btn){
      btn.addEventListener('click',function(){
        if(btn.hasAttribute('disabled')) return;
        var alias=btn.getAttribute('data-alias')||'';
        if(!alias) return;
        btn.setAttribute('disabled','');
        btn.setAttribute('aria-busy','true');
        fetch('/api/'+encodeURIComponent(slug)+'/peers/'+encodeURIComponent(alias)+'/probe',{method:'POST'})
          .then(function(resp){
            if(!resp.ok) throw new Error('probe failed: '+resp.status);
            return resp.json();
          })
          .then(function(payload){
            var row=btn.closest('tr');
            if(!row) return;
            var reachCell=row.querySelector('.cell-reachable');
            if(reachCell) reachCell.textContent=payload.reachable?'yes':'no';
            var probeCell=row.querySelector('.cell-probe');
            if(probeCell){
              var when=new Date(payload.timestamp);
              probeCell.textContent='just now';
              probeCell.setAttribute('title',when.toLocaleString());
            }
          })
          .catch(function(err){console.error('peer probe failed',err);})
          .finally(function(){
            btn.removeAttribute('disabled');
            btn.removeAttribute('aria-busy');
          });
      });
    }
  );

  // ---- Phase 4: Danger Zone typed-confirm gate ----
  Array.prototype.forEach.call(
    document.querySelectorAll('.project-danger-form'),
    function(form){
      var input=form.querySelector('.project-danger-confirm');
      var btn=form.querySelector('.project-danger-submit');
      var target=form.getAttribute('data-slug')||'';
      var endpoint=form.getAttribute('data-endpoint')||'';
      if(!input||!btn) return;
      input.addEventListener('input',function(){
        btn.disabled=(input.value!==target);
      });
      btn.addEventListener('click',function(ev){
        ev.preventDefault();
        if(input.value!==target) return;
        btn.disabled=true;
        postJSON(endpoint).then(function(resp){
          if(!resp.ok) throw new Error('action failed: '+resp.status);
          showUndoToast(target);
        }).catch(function(err){
          btn.disabled=false;
          console.error('danger action failed',err);
        });
      });
    }
  );
})();</script>`
}

// jsonString wraps a string for safe JS embedding. We avoid pulling in
// encoding/json for a single-string case to keep the handler import
// surface small.
func jsonString(s string) string {
	return "\"" + s + "\""
}
