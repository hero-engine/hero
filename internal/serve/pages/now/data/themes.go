package data

import (
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/clusters"
	"github.com/hero-engine/hero/internal/spec"
)

// ThemesInputs is the per-request input bundle for the Hero-noticed
// section on the Now page.
type ThemesInputs struct {
	HeroDir string
	// Aggregate, when non-empty, runs cluster detection across the
	// listed projects and tags each item with its project slug.
	Aggregate []ActivityProject
	// Window scopes the input set to recent activity. Defaults to a
	// rolling 7-day window when zero (matches the activity feed
	// default).
	Window Window
}

// Themes is the section payload. Empty Clusters means no theme reached
// the confidence threshold — the template MUST render nothing in that
// case (no empty header), matching the spec acceptance criterion.
type Themes struct {
	Clusters  []ThemeCluster
	Aggregate bool
}

// ThemeCluster is the page-facing shape of one cluster. Mirrors
// clusters.Cluster but with the link computed for the page-context
// URL prefix.
type ThemeCluster struct {
	Kind      string
	Label     string
	ItemCount int
	Why       string
	Project   string
	Items     []ThemeItem
}

// ThemeItem is one top-3 item under a cluster.
type ThemeItem struct {
	Slug    string
	Title   string
	Project string
	Href    string
}

// LoadThemes runs the work-cluster detector over the active window
// and returns the ranked top clusters. Returns Themes{} (with no
// clusters) when nothing reaches the confidence threshold — callers
// should treat that as "render nothing."
func LoadThemes(in ThemesInputs) Themes {
	win := in.Window
	if win == "" {
		win = WindowWeek
	}
	cutoff := win.Since(time.Now())

	det := clusters.Detector{Aggregate: in.Aggregate != nil}

	var inputs []clusters.Input
	if in.Aggregate != nil {
		for _, p := range in.Aggregate {
			specs := specsModifiedSince(p.HeroDir, cutoff)
			inputs = append(inputs, clusters.Input{Project: p.Slug, Specs: specs})
		}
	} else {
		inputs = append(inputs, clusters.Input{Specs: specsModifiedSince(in.HeroDir, cutoff)})
	}

	raw := det.Detect(inputs...)
	out := Themes{Aggregate: in.Aggregate != nil}
	for _, c := range raw {
		tc := ThemeCluster{
			Kind:      c.Kind,
			Label:     c.Label,
			ItemCount: c.ItemCount,
			Why:       c.Why,
			Project:   c.Project,
		}
		for _, it := range c.Items {
			href := "/work/spec/" + it.Slug
			if it.Project != "" {
				href = "/p/" + it.Project + "/work/spec/" + it.Slug
			}
			tc.Items = append(tc.Items, ThemeItem{
				Slug:    it.Slug,
				Title:   it.Title,
				Project: it.Project,
				Href:    href,
			})
		}
		out.Clusters = append(out.Clusters, tc)
	}
	return out
}

// isNonWorkType reports whether a spec type carries low signal for
// work-cluster detection (notes, contexts, rules, etc.). Decisions and
// conventions count as work output and pass through.
func isNonWorkType(t spec.Type) bool {
	switch t {
	case spec.TypeNote, spec.TypeContext, spec.TypeTripwire,
		spec.TypeExternal, spec.TypeRule:
		return true
	}
	return false
}

// specsModifiedSince filters spec discovery to specs whose ModifiedAt
// is at-or-after cutoff. A zero cutoff returns all specs.
func specsModifiedSince(heroDir string, cutoff time.Time) []*spec.Spec {
	if heroDir == "" {
		return nil
	}
	all, err := spec.Discover(heroDir)
	if err != nil {
		return nil
	}
	out := make([]*spec.Spec, 0, len(all))
	for _, s := range all {
		if s == nil {
			continue
		}
		// Zero cutoff disables the time filter; non-zero cutoffs drop
		// specs older than the window.
		if !cutoff.IsZero() && s.ModifiedAt.Before(cutoff) {
			continue
		}
		// Drop spec types that don't carry useful work-cluster signal.
		// Decisions and conventions stay (they're work output) but
		// notes/contexts/tripwires are dropped — clustering on those
		// produces noise.
		if isNonWorkType(s.Type) {
			continue
		}
		// Skip empty-slug stragglers — they'd dedup wrong.
		if strings.TrimSpace(s.Slug) == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}
