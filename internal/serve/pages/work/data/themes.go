package data

import (
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/clusters"
	"github.com/hero-engine/hero/internal/spec"
)

// ThemesInputs is the per-request input bundle for the Hero-noticed
// themes row on the Work page. Window scopes the input set so the
// row reflects whichever metric tab is currently active. Zero window
// falls back to a rolling 7-day window — the "This week" default.
type ThemesInputs struct {
	HeroDir string
	Window  time.Duration
	// Aggregate, when non-empty, runs cluster detection across the
	// listed projects and tags each item with its project slug.
	Aggregate []ThemesProject
}

// ThemesProject is one project contributing to a /p/all/ themes
// render. Mirrors the shape used elsewhere (work.AggregateProject,
// now/data.ActivityProject) but lives here so the data package
// builds without a back-import to the page wrapper.
type ThemesProject struct {
	Slug    string
	Path    string
	HeroDir string
}

// Themes is the section payload. Empty Clusters means no theme
// reached the confidence threshold — the template renders nothing in
// that case, matching the spec's "no empty header" rule.
type Themes struct {
	Clusters  []ThemeCluster
	Aggregate bool
}

// ThemeCluster mirrors clusters.Cluster with Work-page links.
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

// LoadThemes runs the work-cluster detector scoped to the active
// window and returns ranked top clusters. Returns Themes{} (no
// clusters) when nothing reaches the threshold — caller renders
// nothing in that case.
func LoadThemes(in ThemesInputs) Themes {
	win := in.Window
	if win <= 0 {
		win = 7 * 24 * time.Hour
	}
	cutoff := time.Now().Add(-win)

	det := clusters.Detector{Aggregate: in.Aggregate != nil}

	var inputs []clusters.Input
	if in.Aggregate != nil {
		for _, p := range in.Aggregate {
			specs := workSpecsModifiedSince(p.HeroDir, cutoff)
			inputs = append(inputs, clusters.Input{Project: p.Slug, Specs: specs})
		}
	} else {
		inputs = append(inputs, clusters.Input{Specs: workSpecsModifiedSince(in.HeroDir, cutoff)})
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

func workSpecsModifiedSince(heroDir string, cutoff time.Time) []*spec.Spec {
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
		if !cutoff.IsZero() && s.ModifiedAt.Before(cutoff) {
			continue
		}
		if isNonWorkSpecType(s.Type) {
			continue
		}
		if strings.TrimSpace(s.Slug) == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func isNonWorkSpecType(t spec.Type) bool {
	switch t {
	case spec.TypeNote, spec.TypeContext, spec.TypeTripwire,
		spec.TypeExternal, spec.TypeRule, spec.TypeExplainer:
		return true
	}
	return false
}
