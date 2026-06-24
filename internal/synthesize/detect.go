package synthesize

import (
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
)

// Candidate is an inferred or explicit cluster of completed specs that looks
// like one shipped feature — a synthesis candidate. It carries the confidence
// and the signals that fired so a downstream surface (the trust handshake,
// #4) can explain why the cluster was proposed.
type Candidate struct {
	Slugs      []string
	OutSlug    string
	Title      string
	Confidence float64
	Signals    []string
}

const (
	detectThreshold = 0.5
	// hubFileThreshold: a file touched by more than this many specs is a
	// shared "hub" (e.g. internal/cli/root.go) and is ignored for clustering
	// — it is noise, not a co-feature signal. Without this guard, common
	// files transitively chain nearly every completed spec into one blob.
	hubFileThreshold = 5
	// maxInferredCluster backstops runaway inferred clusters: a feature
	// spanning more specs than this needs an explicit boundary, not a guess.
	maxInferredCluster = 12
)

// Detect finds candidate clusters of completed specs worth an explainer. It
// draws on two sources: explicit (an initiative whose materialized children
// are all completed) and inferred (connected components over the relation
// graph + file-overlap among completed work specs). Structural signals
// dominate the score; time/author are deliberately not used (delivered
// observation: same-window != same feature). Clusters already covered by a
// current explainer are omitted (defer to amendment, #5).
func Detect(heroDir string) ([]Candidate, error) {
	all, err := spec.Discover(heroDir)
	if err != nil {
		return nil, err
	}
	bySlug := make(map[string]*spec.Spec, len(all))
	for _, s := range all {
		bySlug[s.Slug] = s
	}
	covered := coveredSlugs(all)

	var candidates []Candidate
	claimed := map[string]bool{} // slugs already in an explicit candidate

	// --- Explicit: initiative with all materialized children completed. ---
	for _, s := range all {
		if s.Type != spec.TypeInitiative {
			continue
		}
		children := childSlugs(s, all)
		if len(children) == 0 {
			continue
		}
		var done []string
		allComplete := true
		for _, c := range children {
			cs := bySlug[c]
			if cs == nil || cs.Status != spec.StatusCompleted {
				allComplete = false // unmaterialized or open → not shippable yet
				continue
			}
			done = append(done, c)
		}
		if !allComplete || len(done) < 2 || fullyCovered(done, covered) {
			continue
		}
		sort.Strings(done)
		for _, c := range done {
			claimed[c] = true
		}
		candidates = append(candidates, Candidate{
			Slugs:      done,
			OutSlug:    s.Slug,
			Title:      strings.Trim(s.Title, `"`),
			Confidence: 0.95,
			Signals:    []string{"initiative-complete"},
		})
	}

	// --- Inferred: connected components among completed work specs. ---
	var work []*spec.Spec
	for _, s := range all {
		if s.Status == spec.StatusCompleted && (s.Type == spec.TypeFeature || s.Type == spec.TypeBug) && !claimed[s.Slug] {
			work = append(work, s)
		}
	}
	for _, comp := range components(work, bySlug) {
		if len(comp) < 2 {
			continue
		}
		if len(comp) > maxInferredCluster {
			continue // too broad to be one feature — needs an explicit boundary
		}
		slugs := make([]string, 0, len(comp))
		for _, s := range comp {
			slugs = append(slugs, s.Slug)
		}
		sort.Strings(slugs)
		if fullyCovered(slugs, covered) {
			continue
		}
		conf, signals := score(comp)
		if conf < detectThreshold {
			continue
		}
		dominant := comp[0]
		candidates = append(candidates, Candidate{
			Slugs:      slugs,
			OutSlug:    dominant.Slug,
			Title:      strings.Trim(dominant.Title, `"`),
			Confidence: conf,
			Signals:    signals,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Confidence > candidates[j].Confidence
	})
	return candidates, nil
}

// coveredSlugs returns the set of spec slugs already described by some
// existing explainer (via its synthesized_from provenance).
func coveredSlugs(all []*spec.Spec) map[string]bool {
	covered := map[string]bool{}
	for _, s := range all {
		if s.Type == spec.TypeExplainer {
			for _, src := range s.SynthesizedFrom {
				covered[src] = true
			}
		}
	}
	return covered
}

func fullyCovered(slugs []string, covered map[string]bool) bool {
	for _, s := range slugs {
		if !covered[s] {
			return false
		}
	}
	return true
}

// childSlugs returns the declared children of an initiative — both the
// `child:` relations on the initiative and any spec declaring it as `parent`.
func childSlugs(initiative *spec.Spec, all []*spec.Spec) []string {
	seen := map[string]bool{}
	var out []string
	add := func(slug string) {
		if slug != "" && !seen[slug] {
			seen[slug] = true
			out = append(out, slug)
		}
	}
	for _, r := range initiative.Relations {
		if r.Kind == "child" {
			add(r.Target)
		}
	}
	for _, s := range all {
		for _, r := range s.Relations {
			if r.Kind == "parent" && r.Target == initiative.Slug {
				add(s.Slug)
			}
		}
	}
	return out
}

// components groups work specs into connected components, where two specs are
// connected if they share a relation edge (either direction), a parent, or a
// touched file. Union-find over the candidate set.
func components(work []*spec.Spec, bySlug map[string]*spec.Spec) [][]*spec.Spec {
	idx := make(map[string]int, len(work))
	for i, s := range work {
		idx[s.Slug] = i
	}
	parent := make([]int, len(work))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}
		return x
	}
	union := func(a, b int) { parent[find(a)] = find(b) }

	// Relation edges among the work set.
	for i, s := range work {
		for _, r := range s.Relations {
			if j, ok := idx[r.Target]; ok {
				union(i, j)
			}
		}
	}
	// Shared parent (same initiative/parent target).
	parentOf := map[string][]int{}
	for i, s := range work {
		for _, r := range s.Relations {
			if r.Kind == "parent" {
				parentOf[r.Target] = append(parentOf[r.Target], i)
			}
		}
	}
	for _, members := range parentOf {
		for k := 1; k < len(members); k++ {
			union(members[0], members[k])
		}
	}
	// File overlap — but skip hub files touched by many specs, which would
	// otherwise transitively chain unrelated features into one blob.
	fileOwners := map[string][]int{}
	for i, s := range work {
		for _, f := range s.FilesTouched {
			fileOwners[f] = append(fileOwners[f], i)
		}
	}
	for _, owners := range fileOwners {
		if len(owners) > hubFileThreshold {
			continue
		}
		for k := 1; k < len(owners); k++ {
			union(owners[0], owners[k])
		}
	}

	groups := map[int][]*spec.Spec{}
	for i, s := range work {
		root := find(i)
		groups[root] = append(groups[root], s)
	}
	out := make([][]*spec.Spec, 0, len(groups))
	for _, g := range groups {
		out = append(out, g)
	}
	return out
}

// score weights the structural signals present in a component. Time and
// author are intentionally excluded — same-window is not same-feature.
func score(comp []*spec.Spec) (float64, []string) {
	conf := 0.3 // base for being one connected component
	var signals []string

	if sharedParent(comp) {
		conf += 0.3
		signals = append(signals, "shared-parent")
	}
	if relationEdges(comp) {
		conf += 0.25
		signals = append(signals, "relation-edges")
	}
	if fileOverlap(comp) {
		conf += 0.2
		signals = append(signals, "co-touched-files")
	}
	if sharedTags(comp) {
		conf += 0.1
		signals = append(signals, "shared-tags")
	}
	if conf > 0.95 {
		conf = 0.95
	}
	if len(signals) == 0 {
		signals = []string{"connected-component"}
	}
	return conf, signals
}

func sharedParent(comp []*spec.Spec) bool {
	counts := map[string]int{}
	for _, s := range comp {
		for _, r := range s.Relations {
			if r.Kind == "parent" {
				counts[r.Target]++
			}
		}
	}
	for _, c := range counts {
		if c >= 2 {
			return true
		}
	}
	return false
}

func relationEdges(comp []*spec.Spec) bool {
	in := map[string]bool{}
	for _, s := range comp {
		in[s.Slug] = true
	}
	for _, s := range comp {
		for _, r := range s.Relations {
			if in[r.Target] && r.Kind != "parent" {
				return true
			}
		}
	}
	return false
}

func fileOverlap(comp []*spec.Spec) bool {
	counts := map[string]int{}
	for _, s := range comp {
		seen := map[string]bool{}
		for _, f := range s.FilesTouched {
			if !seen[f] {
				seen[f] = true
				counts[f]++
			}
		}
	}
	for _, c := range counts {
		if c >= 2 {
			return true
		}
	}
	return false
}

func sharedTags(comp []*spec.Spec) bool {
	counts := map[string]int{}
	for _, s := range comp {
		for _, t := range s.Tags {
			counts[t]++
		}
	}
	for _, c := range counts {
		if c >= 2 {
			return true
		}
	}
	return false
}
