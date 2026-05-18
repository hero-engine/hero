package shell

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

// fragmentRender executes a named template against a struct and
// returns the rendered HTML.
func fragmentRender(t *testing.T, name string, data any) string {
	t.Helper()
	tmpl := loadTemplates()
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		t.Fatalf("execute %s: %v", name, err)
	}
	return buf.String()
}

func TestRender_PageHero(t *testing.T) {
	out := fragmentRender(t, "page-hero", PageHero{
		Eyebrow: template.HTML("hero · main · solo edition"),
		Title:   "Now",
		Subhead: template.HTML("<strong>2 need your input</strong>"),
		Actions: []PageHeroAction{
			{Kind: "primary", Label: "Open Inbox", Href: "#"},
			{Kind: "ghost", Label: "What changed", Href: "#"},
			{Kind: "chip", Label: "Solo", Chip: "Solo"},
		},
	})
	for _, want := range []string{
		`class="page-hero"`,
		`class="eyebrow"`,
		`class="page-title">Now</h1>`,
		`<strong>2 need your input</strong>`,
		`btn btn-primary`,
		`btn btn-ghost`,
		`chip chip-solo`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("page-hero missing %q in:\n%s", want, out)
		}
	}
}

func TestRender_TabbedMetricStrip(t *testing.T) {
	out := fragmentRender(t, "tabbed-metric-strip", MetricStrip{
		AllLink: "/metrics",
		Tabs: []MetricTab{
			{Slug: "sprint", Label: "This sprint", Active: true,
				Tiles: []MetricTile{
					{Value: template.HTML(`9<span class="unit">/ 14</span>`), Label: "specs done"},
				}},
			{Slug: "week", Label: "My week",
				Tiles: []MetricTile{
					{Value: template.HTML("5"), Label: "shipped"},
				}},
		},
	})
	for _, want := range []string{
		`class="metric-strip-section"`,
		`data-metric-tab="sprint"`,
		`data-metric-tab="week"`,
		`data-metric-pane="sprint"`,
		`data-metric-pane="week"`,
		`aria-selected="true"`,
		`aria-selected="false"`,
		`hidden`, // inactive pane
		`9<span class="unit">/ 14</span>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("tabbed-metric-strip missing %q in:\n%s", want, out)
		}
	}
}

func TestRender_SubNav(t *testing.T) {
	out := fragmentRender(t, "sub-nav", &SubNav{
		Tabs: []SubNavTab{
			{Label: "Sessions", Href: "/agents/sessions", Active: true, Badge: "3"},
			{Label: "Proposals", Href: "/agents/proposals"},
			{Label: "Drift", Href: "/agents/drift", Variant: "amber"},
			{Label: "Cloud sync", Href: "#", Variant: "locked", LockMeta: "Requires cloud edition"},
		},
	})
	for _, want := range []string{
		`class="subnav"`,
		`subnav-tab active`,
		`subnav-tab amber`,
		`subnav-tab`, // locked variant
		`title="Requires cloud edition"`,
		`Sessions<`,
		`class="badge">3</span>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("sub-nav missing %q in:\n%s", want, out)
		}
	}
}

func TestRender_Footer(t *testing.T) {
	out := fragmentRender(t, "footer", Footer{
		Workspace: "hero",
		Version:   "0.18.2",
		Edition:   "local",
	})
	for _, want := range []string{
		`class="shell-footer"`,
		`Hero v0.18.2`,
		`workspace hero`,
		`local edition`,
		`>Docs</a>`,
		`>GitHub</a>`,
		`>⌘K</a>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("footer missing %q in:\n%s", want, out)
		}
	}
}

func TestRender_ChatInput(t *testing.T) {
	for _, variant := range []string{"hero", "overlay", "inline"} {
		out := fragmentRender(t, "chat-input", ChatInput{
			Variant:     variant,
			Placeholder: "describe what to do…",
			Context: []ChatContextChip{
				{Kind: "page", Label: "page: /now"},
			},
		})
		for _, want := range []string{
			`class="chat-input ` + variant + `"`,
			`data-command-bar-input`,
			`placeholder="describe what to do…"`,
			`data-chip-kind="page"`,
		} {
			if !strings.Contains(out, want) {
				t.Errorf("chat-input/%s missing %q in:\n%s", variant, want, out)
			}
		}
	}
}

func TestRender_EmptyStateNotice(t *testing.T) {
	out := fragmentRender(t, "empty-state-notice", EmptyState{
		Headline:      "Hero needs an adapter",
		Body:          template.HTML("Install <code>hero-code</code> first."),
		PrimaryAction: EmptyStateAction{Label: "Install", Href: "/install"},
		GhostAction:   EmptyStateAction{Label: "Learn more", Href: "/docs"},
		FootNote:      "Then chat goes live.",
	})
	for _, want := range []string{
		`class="empty-state-notice"`,
		`class="headline">Hero needs an adapter</h3>`,
		`Install <code>hero-code</code> first.`,
		`btn-primary">Install</a>`,
		`btn-ghost">Learn more</a>`,
		`class="footnote">Then chat goes live.</div>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("empty-state-notice missing %q in:\n%s", want, out)
		}
	}
}

func TestRender_TopNav(t *testing.T) {
	out := fragmentRender(t, "top-nav", Chrome{
		Workspace:    "hero",
		Branch:       "main",
		UserName:     "Ben Wheeler",
		UserInitials: "BW",
		Tabs: []ChromeTab{
			{Slug: "now", Label: "Now", Href: "/now", Active: true},
			{Slug: "work", Label: "Work", Href: "/work", HasCount: true, Count: 18},
		},
	})
	for _, want := range []string{
		`class="topnav"`,
		`class="brand"`,
		`<span class="workspace">hero</span>`,
		`class="nav-tab active"`,
		`Work<span class="count">18</span>`,
		`data-command-bar-trigger`,
		`<div class="avatar" title="Ben Wheeler">BW</div>`,
		`>main`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("top-nav missing %q in:\n%s", want, out)
		}
	}
}
