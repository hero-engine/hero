package shell

// Stub home registrations for Now / Work / Knowledge / Agents /
// People. Each stub renders a minimal placeholder so the top-nav tabs
// route somewhere end-to-end. When each home spec delivers, that
// home's package replaces the stub by registering its own Home with
// the same Slug + Href.

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
)

// RegisterStubHomes registers a placeholder for each of the five
// top-nav homes. Safe to call once at startup.
func RegisterStubHomes(r *Router) {
	for _, s := range stubSpecs() {
		stub := s
		home := Home{
			Slug:  stub.slug,
			Label: stub.label,
			Href:  stub.href,
			Render: func(w http.ResponseWriter, req *http.Request) {
				renderStub(r, w, req, stub)
			},
		}
		if err := r.RegisterHome(home); err != nil {
			fmt.Fprintf(stderr(), "shell: register stub home %s: %v\n", stub.slug, err)
		}
	}
}

type stubSpec struct {
	slug     string
	label    string
	href     string
	title    string
	specSlug string
	specHref string
}

func stubSpecs() []stubSpec {
	return []stubSpec{
		{slug: "now", label: "Now", href: "/now", title: "Now",
			specSlug: "hero-now-home",
			specHref: "/.hero/planning/features/hero-now-home/spec.md"},
		{slug: "work", label: "Work", href: "/work", title: "Work",
			specSlug: "hero-work-home",
			specHref: "/.hero/planning/features/hero-work-home/spec.md"},
		{slug: "knowledge", label: "Knowledge", href: "/knowledge", title: "Knowledge",
			specSlug: "hero-knowledge-home",
			specHref: "/.hero/planning/features/hero-knowledge-home/spec.md"},
		{slug: "agents", label: "Agents", href: "/agents", title: "Agents",
			specSlug: "hero-agents-home",
			specHref: "/.hero/planning/features/hero-agents-home/spec.md"},
		{slug: "people", label: "People", href: "/people", title: "People",
			specSlug: "hero-people-and-roi-home",
			specHref: "/.hero/planning/features/hero-people-and-roi-home/spec.md"},
	}
}

func renderStub(r *Router, w http.ResponseWriter, req *http.Request, s stubSpec) {
	hero := PageHero{
		Eyebrow: template.HTML("hero · stub · this surface is placeholder content"),
		Title:   s.title,
		Subhead: template.HTML("Coming soon — content delivered by the <code>" + s.specSlug + "</code> spec."),
	}

	strip := MetricStrip{
		Tabs: []MetricTab{
			{
				Slug: "overview", Label: "Overview", Active: true,
				Tiles: []MetricTile{
					{Value: template.HTML("—"), Label: "metric 1"},
					{Value: template.HTML("—"), Label: "metric 2"},
					{Value: template.HTML("—"), Label: "metric 3"},
					{Value: template.HTML("—"), Label: "metric 4"},
				},
			},
		},
	}

	empty := EmptyState{
		Headline:      s.title + " home is not yet implemented",
		Body:          template.HTML("This page is a stub registered by the shell so the top-nav routing works end-to-end. The real content arrives when the <code>" + s.specSlug + "</code> spec lands."),
		PrimaryAction: EmptyStateAction{Label: "Open spec", Href: s.specHref},
		GhostAction:   EmptyStateAction{Label: "Back to /", Href: "/"},
	}

	content := func(out io.Writer) error {
		if err := r.tmpl.ExecuteTemplate(out, "page-hero", hero); err != nil {
			return err
		}
		if err := r.tmpl.ExecuteTemplate(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		return r.tmpl.ExecuteTemplate(out, "empty-state-notice", empty)
	}

	page := Page{
		ActiveHome: s.slug,
		PageTitle:  s.title + " · Hero",
		Content:    content,
	}
	if err := r.RenderPage(w, req, page); err != nil {
		http.Error(w, "render "+s.slug+" stub: "+err.Error(), http.StatusInternalServerError)
	}
}

// stderr returns os.Stderr but is overridable for tests.
var stderr = func() io.Writer { return defaultStderr }
