package shell

// kitchen-sink — dev-time fragment showcase at /_kitchen-sink.
//
// Renders one example of every shared shell fragment using realistic
// dummy data so designers and engineers can review the fragment
// library in a browser without waiting on a home spec. Always
// registered; the URL is obscure enough to avoid stumbled-upon
// surfacing.

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
)

func (r *Router) handleKitchenSink(w http.ResponseWriter, req *http.Request) {
	subnav := &SubNav{
		Tabs: []SubNavTab{
			{Label: "Sessions", Href: "#", Active: true, Badge: "3"},
			{Label: "Proposals", Href: "#", Badge: "1"},
			{Label: "Scheduled", Href: "#"},
			{Label: "Drift", Href: "#", Variant: "amber", Badge: "2"},
			{Label: "Cloud sync", Href: "#", Variant: "locked", LockMeta: "Requires cloud edition"},
		},
	}

	hero := PageHero{
		Eyebrow: template.HTML("kitchen-sink · shared fragment showcase"),
		Title:   "Kitchen sink",
		Subhead: template.HTML("One of every <strong>shared shell fragment</strong> in one place. Update fixtures here when a fragment template changes."),
		Actions: []PageHeroAction{
			{Kind: "primary", Label: "Primary action", Href: "#"},
			{Kind: "ghost", Label: "Ghost action", Href: "#"},
			{Kind: "chip", Label: "Solo", Chip: "Solo"},
		},
	}

	strip := MetricStrip{
		AllLink: "#",
		Tabs: []MetricTab{
			{
				Slug: "sprint", Label: "This sprint", Active: true,
				Tiles: []MetricTile{
					{Value: template.HTML(`9<span class="unit">/ 14</span>`), Label: "specs done · Sprint 17"},
					{Value: template.HTML("2"), Label: "days remaining", Footer: template.HTML(`<div class="metric-sub">ends Wed May 20</div>`)},
					{Value: template.HTML("2"), Accent: "warn", Label: "specs flagged at risk"},
					{Value: template.HTML(`3<span class="unit">/ 4</span>`), Label: "your committed specs"},
				},
			},
			{
				Slug: "week", Label: "My week",
				Tiles: []MetricTile{
					{Value: template.HTML("5"), Label: "specs shipped this week"},
					{Value: template.HTML("47"), Label: "commits authored"},
					{Value: template.HTML(`84<span class="unit">%</span>`), Label: "agent assist"},
					{Value: template.HTML(`22<span class="unit"> sessions</span>`), Label: "≈18h Hero-active"},
				},
			},
		},
	}

	chat := ChatInput{
		Variant:     "hero",
		Placeholder: "Describe what you want to work on…",
		Context: []ChatContextChip{
			{Kind: "page", Label: "page: /now"},
			{Kind: "spec", Label: "spec: hero-surface-shell"},
		},
	}

	empty := EmptyState{
		Headline:      "Hero needs an adapter to send chat",
		Body:          template.HTML("Install <code>hero-code</code> or another chat adapter to enable streaming. The shell will mount whichever adapter you wire up."),
		PrimaryAction: EmptyStateAction{Label: "Install hero-code", Href: "https://heroengine.ai/docs/adapters"},
		GhostAction:   EmptyStateAction{Label: "Read more", Href: "#"},
		FootNote:      "Once an adapter is connected the chat input becomes live everywhere.",
	}

	content := func(w io.Writer) error {
		// page-hero
		if err := r.tmpl.ExecuteTemplate(w, "page-hero", hero); err != nil {
			return fmt.Errorf("page-hero: %w", err)
		}
		if err := r.tmpl.ExecuteTemplate(w, "tabbed-metric-strip", strip); err != nil {
			return fmt.Errorf("tabbed-metric-strip: %w", err)
		}
		// chat-input variants
		if _, err := io.WriteString(w, `<section><h2 class="section-title">chat-input variants</h2>`); err != nil {
			return err
		}
		for _, v := range []string{"hero", "overlay", "inline"} {
			c := chat
			c.Variant = v
			if _, err := io.WriteString(w, `<div style="margin:24px 0;">`); err != nil {
				return err
			}
			if err := r.tmpl.ExecuteTemplate(w, "chat-input", c); err != nil {
				return fmt.Errorf("chat-input %s: %w", v, err)
			}
			if _, err := io.WriteString(w, `</div>`); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, `</section>`); err != nil {
			return err
		}
		// empty-state
		if err := r.tmpl.ExecuteTemplate(w, "empty-state-notice", empty); err != nil {
			return fmt.Errorf("empty-state-notice: %w", err)
		}
		return nil
	}

	page := Page{
		ActiveHome: "", // no tab — dev route
		PageTitle:  "Kitchen sink · Hero",
		SubNav:     subnav,
		Content:    content,
	}
	if err := r.RenderPage(w, req, page); err != nil {
		http.Error(w, "render kitchen-sink: "+err.Error(), http.StatusInternalServerError)
	}
}
