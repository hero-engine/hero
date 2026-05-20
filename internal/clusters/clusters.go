// Package clusters detects "work clusters" — recurring shapes in a
// project's recent activity that an operator should notice. It extends
// the knowledge-flywheel pattern-detection thesis (which clusters
// captured notes / decisions) to the spec graph, file paths, and the
// events log so the Now and Work dashboards can surface "Hero noticed
// you're spending time on X" themes.
//
// The detector is intentionally conservative: a cluster surfaces only
// when MinItems or more items share a signal. Bad clusters are worse
// than no clusters, so the threshold biases toward false negatives.
// Below the threshold the detector returns no clusters rather than
// emitting a low-confidence header.
//
// This package is leaf-free: it depends only on the spec types so it
// can run inside the page-data fetchers without dragging in serve / cli
// internals.
package clusters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
)

// MinItems is the default confidence threshold — a cluster surfaces
// only when this many items share its signal. Per the spec:
// "Require minimum 3 related items in the window before surfacing a
// theme." Callers can override via Detector.MinItems.
const MinItems = 3

// MaxClusters caps how many clusters the detector returns. Anything
// beyond this is overload; the UI is meant to surface the top themes,
// not exhaustive analytics.
const MaxClusters = 5

// Cluster is one detected work cluster, ready to render.
//
// Kind names the signal that grouped it (file-path prefix vs.
// decision-tag co-occurrence). Items is bounded to the top three by
// the detector — the spec calls for "cluster label, item count, top-3
// items, and a why-this-clustered 1-line summary." Full items live
// inside ItemCount only as a number.
type Cluster struct {
	Kind     string // "path-prefix" | "tag"
	Label    string // e.g. "internal/serve/" or "auth"
	ItemCount int   // total items matching this cluster
	Items    []Item // top 3 items (most recent first)
	Why      string // 1-line "why this clustered" summary

	// Project tags a cluster with the project slug it came from when
	// running in aggregate mode (Detector.Aggregate=true). Empty in
	// single-project mode.
	Project string
}

// Item is one entry inside a cluster. Slug is the spec slug the item
// derives from (empty for non-spec items); Title is a display label.
type Item struct {
	Slug    string
	Title   string
	Path    string // optional path that contributed to the cluster signal
	Project string // populated in aggregate mode
}

// Input is the per-detector input bundle. Specs is the candidate spec
// set; the detector reads file-path mentions, tags, and slugs off
// each. Empty input yields no clusters.
type Input struct {
	// Specs is the candidate set. Pre-filter to the window the caller
	// cares about (e.g. last 7 days by ModifiedAt) before passing in.
	Specs []*spec.Spec
	// Project tags items in aggregate mode. Empty in single-project
	// mode.
	Project string
}

// Detector configures the cluster pass. The zero value is safe and
// uses MinItems / MaxClusters defaults.
type Detector struct {
	// MinItems, when non-zero, overrides the package-level MinItems
	// constant. Lower values surface more clusters at lower confidence.
	MinItems int
	// MaxClusters, when non-zero, overrides MaxClusters.
	MaxClusters int
	// PathDepth controls how many leading path segments form a
	// path-prefix cluster label. Default 2 — e.g. "internal/serve/" is
	// surfaced rather than "internal/serve/pages/now/" — so clusters
	// are coarse enough to actually cluster.
	PathDepth int
	// Aggregate signals that input items may come from multiple
	// projects (each Input pre-tags its Project); cluster outputs
	// preserve that tag.
	Aggregate bool
}

// Detect runs the cluster pass over one or more inputs and returns
// the ranked top clusters. Aggregate-mode callers pass one Input per
// project; single-project callers pass exactly one Input.
//
// Ranking: clusters with more items come first; ties break by label
// asc for stability.
func (d Detector) Detect(inputs ...Input) []Cluster {
	min := d.MinItems
	if min <= 0 {
		min = MinItems
	}
	maxC := d.MaxClusters
	if maxC <= 0 {
		maxC = MaxClusters
	}
	depth := d.PathDepth
	if depth <= 0 {
		depth = 2
	}

	pathHits := map[string][]Item{}
	tagHits := map[string][]Item{}

	for _, in := range inputs {
		for _, s := range in.Specs {
			if s == nil {
				continue
			}
			item := Item{
				Slug:    s.Slug,
				Title:   firstNonEmpty(s.Title, s.Slug),
				Project: in.Project,
			}

			// path-prefix signal — read the spec's Changes section
			// (file paths) plus the spec file's own location.
			for _, p := range pathsFromSpec(s) {
				prefix := pathPrefix(p, depth)
				if prefix == "" {
					continue
				}
				it := item
				it.Path = p
				pathHits[prefix] = append(pathHits[prefix], it)
			}

			// tag co-occurrence — every tag is a potential cluster
			// label. We deliberately don't promote single-tag specs
			// without a co-occurrence partner; the count threshold
			// filters those out below.
			for _, tag := range s.Tags {
				tag = strings.TrimSpace(tag)
				if tag == "" {
					continue
				}
				tagHits[tag] = append(tagHits[tag], item)
			}
		}
	}

	clusters := make([]Cluster, 0)
	for prefix, items := range pathHits {
		if len(items) < min {
			continue
		}
		uniq := dedupItemsBySlug(items)
		if len(uniq) < min {
			continue
		}
		clusters = append(clusters, Cluster{
			Kind:      "path-prefix",
			Label:     prefix,
			ItemCount: len(uniq),
			Items:     topN(uniq, 3),
			Why:       fmt.Sprintf("%d specs touched files under %s recently.", len(uniq), prefix),
			Project:   firstProject(uniq),
		})
	}
	for tag, items := range tagHits {
		if len(items) < min {
			continue
		}
		uniq := dedupItemsBySlug(items)
		if len(uniq) < min {
			continue
		}
		clusters = append(clusters, Cluster{
			Kind:      "tag",
			Label:     tag,
			ItemCount: len(uniq),
			Items:     topN(uniq, 3),
			Why:       fmt.Sprintf("%d specs share the tag %q.", len(uniq), tag),
			Project:   firstProject(uniq),
		})
	}

	sort.SliceStable(clusters, func(i, j int) bool {
		if clusters[i].ItemCount != clusters[j].ItemCount {
			return clusters[i].ItemCount > clusters[j].ItemCount
		}
		return clusters[i].Label < clusters[j].Label
	})
	if len(clusters) > maxC {
		clusters = clusters[:maxC]
	}
	return clusters
}

// pathsFromSpec extracts file paths from a spec for cluster signal.
// We read the FilesTouched slice (already populated by the spec
// loader from the spec's Changes section) — the spec's own location
// under .hero/planning/ is deliberately NOT included because every
// spec lives there and would over-cluster on the workspace's own
// directory layout.
func pathsFromSpec(s *spec.Spec) []string {
	if s == nil {
		return nil
	}
	return s.FilesTouched
}

// pathPrefix returns the first `depth` segments of a path. e.g.
// pathPrefix("internal/serve/pages/now/x.go", 2) → "internal/serve/".
// Returns "" when the path has fewer than depth segments (we don't
// cluster on too-short paths to avoid clusters like "internal/" that
// over-match).
func pathPrefix(p string, depth int) string {
	p = strings.TrimPrefix(p, "./")
	parts := strings.Split(p, "/")
	if len(parts) < depth+1 {
		// +1 because the last segment is usually the filename.
		return ""
	}
	return strings.Join(parts[:depth], "/") + "/"
}

// dedupItemsBySlug collapses duplicate items sharing the same slug —
// a spec that contributes multiple paths only counts once toward the
// cluster's item total. Items without a slug are kept as-is.
func dedupItemsBySlug(items []Item) []Item {
	seen := map[string]int{}
	out := make([]Item, 0, len(items))
	for _, it := range items {
		if it.Slug == "" {
			out = append(out, it)
			continue
		}
		if _, ok := seen[it.Slug]; ok {
			continue
		}
		seen[it.Slug] = len(out)
		out = append(out, it)
	}
	return out
}

func topN(items []Item, n int) []Item {
	if n <= 0 || len(items) == 0 {
		return nil
	}
	if len(items) <= n {
		out := make([]Item, len(items))
		copy(out, items)
		return out
	}
	out := make([]Item, n)
	copy(out, items[:n])
	return out
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func firstProject(items []Item) string {
	for _, it := range items {
		if it.Project != "" {
			return it.Project
		}
	}
	return ""
}
