package snapshot

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// BuildOptions tunes Build. All fields have sensible zero-value
// defaults.
type BuildOptions struct {
	ProjectRoot string
	HeroDir     string
	ProjectName string
	Mission     string

	// Now is injected for deterministic tests; defaults to time.Now().
	Now time.Time

	// RecentCutoff caps how far back "Recently completed" reaches.
	// Default: 14 days.
	RecentCutoff time.Duration
	// NextN caps "Next up". Default: 5.
	NextN int
	// RecentN caps "Recently completed". Default: 12.
	RecentN int
	// StaleCutoff defines how long an in-flight spec can sit without
	// activity before being flagged. Default: 14 days.
	StaleCutoff time.Duration
	// AgedBugCutoff defines how long a bug can be open before being
	// flagged. Default: 21 days.
	AgedBugCutoff time.Duration

	// Resolver contains tracker / git-tag release data. Nil-safe: when
	// nil the chain falls through to "frontmatter / initiative" only.
	Resolver *ReleaseResolver

	// ScopeTagToSurface, if provided, lets release_targets declared on
	// a surface filter which specs count toward that surface's release
	// rollup. Empty map means every spec on the surface counts.
	ScopeTagToSurface map[string]string
}

// Build assembles a Snapshot from disk + graph inputs. The expensive
// IO (spec discovery, override file load) lives here; rendering is
// pure-function on the returned struct.
//
// allSpecs is the canonical spec corpus, normally produced by
// spec.Discover. shippedTags is the set of release-tag names found
// in the local git checkout (empty when unavailable).
func Build(opts BuildOptions, allSpecs []*spec.Spec, override SurfacesOverride, shippedTags map[string]bool) (*Snapshot, error) {
	start := time.Now()
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.RecentCutoff == 0 {
		opts.RecentCutoff = 14 * 24 * time.Hour
	}
	if opts.NextN == 0 {
		opts.NextN = 5
	}
	if opts.RecentN == 0 {
		opts.RecentN = 12
	}
	if opts.StaleCutoff == 0 {
		opts.StaleCutoff = 14 * 24 * time.Hour
	}
	if opts.AgedBugCutoff == 0 {
		opts.AgedBugCutoff = 21 * 24 * time.Hour
	}

	// 1. Detect surfaces, apply override layer.
	rs, _ := ScanRepo(opts.ProjectRoot)
	detected := Detect(rs)
	surfaces := Merge(detected, override)
	surfaceByID := map[string]int{}
	for i, s := range surfaces {
		surfaceByID[s.ID] = i
	}

	// 2. Build initiative-release map for the resolver.
	initiativeReleases := map[string]string{}
	for _, s := range allSpecs {
		if s.Type == spec.TypeInitiative {
			if rt := releaseTargetFromSpec(s); rt != "" {
				initiativeReleases[s.Slug] = rt
			}
		}
	}
	resolver := ReleaseResolver{
		InitiativeReleases: initiativeReleases,
	}
	if opts.Resolver != nil {
		if opts.Resolver.TrackerReleases != nil {
			resolver.TrackerReleases = opts.Resolver.TrackerReleases
		}
		if opts.Resolver.GitTagReleases != nil {
			resolver.GitTagReleases = opts.Resolver.GitTagReleases
		}
	}

	// 3. Assign each spec to a surface; resolve release.
	assignments := make([]SpecAssignment, 0, len(allSpecs))
	unassignedCount := 0
	for _, s := range allSpecs {
		if s == nil {
			continue
		}
		// Knowledge entries don't surface on the snapshot.
		if s.IsKnowledge() {
			continue
		}
		sa := SpecAssignment{Spec: s}
		// Explicit frontmatter wins.
		declared := surfaceFromSpec(s)
		if declared != "" {
			sa.SurfaceID = declared
			sa.Inferred = false
		} else {
			sa.SurfaceID = inferSurfaceFromPaths(s, surfaces)
			sa.Inferred = true
		}
		if sa.SurfaceID == "" {
			sa.SurfaceID = UnassignedSurfaceID
			unassignedCount++
		}
		res := resolver.ResolveRelease(s)
		sa.ReleaseTarget = res.Target
		sa.ReleaseSource = res.Source
		assignments = append(assignments, sa)
	}

	// 4. Per-surface stage derivation. The unassigned bucket gets no
	//    stage.
	specsPerSurface := map[string][]SpecAssignment{}
	for _, sa := range assignments {
		specsPerSurface[sa.SurfaceID] = append(specsPerSurface[sa.SurfaceID], sa)
	}

	hasReleaseSignal := false
	for i, s := range surfaces {
		if s.StagePinned {
			continue
		}
		in := StageInputs{
			SpecsByStatus: map[spec.Status]int{},
		}
		for _, sa := range specsPerSurface[s.ID] {
			in.SpecsByStatus[sa.Spec.Status]++
			if sa.Spec.Status == spec.StatusCompleted && sa.Spec.ModifiedAt.After(in.LastTouched) {
				in.LastTouched = sa.Spec.ModifiedAt
			}
			if sa.ReleaseTarget != "" {
				in.ReleaseTotal++
				hasReleaseSignal = true
				if sa.Spec.Status == spec.StatusCompleted {
					in.ReleaseDone++
				}
			}
		}
		if shippedTags != nil {
			// Heuristic: any tag at all signals a published release.
			for range shippedTags {
				in.HasShippedTag = true
				break
			}
		}
		surfaces[i].Stage = DeriveStage(in)
	}

	// 5. Rollups: active initiatives, recently completed, next up, blockers.
	initiatives := rollupInitiatives(allSpecs, assignments)
	recent := rollupRecent(assignments, opts.Now, opts.RecentCutoff, opts.RecentN)
	nextUp := rollupNext(assignments, opts.NextN)
	bySlug := map[string]*spec.Spec{}
	for _, s := range allSpecs {
		if s != nil {
			bySlug[s.Slug] = s
		}
	}
	blockers := rollupBlockers(allSpecs, bySlug)
	stale := rollupStale(assignments, opts.Now, opts.StaleCutoff)
	agedBugs := rollupAgedBugs(allSpecs, opts.Now, opts.AgedBugCutoff)

	// 6. Health metadata.
	inferred := 0
	overridden := 0
	for _, s := range surfaces {
		switch s.Source {
		case "inferred":
			inferred++
		case "override", "added":
			overridden++
		}
	}
	overrideEdited := time.Time{}
	if override.Path != "" {
		// File timestamp lookup; ignore errors.
		if info, err := osStat(override.Path); err == nil {
			overrideEdited = info.ModTime()
		}
	}

	out := &Snapshot{
		ProjectName:          opts.ProjectName,
		Mission:              opts.Mission,
		GeneratedAt:          opts.Now,
		Surfaces:             surfaces,
		Assignments:          assignments,
		Initiatives:          initiatives,
		RecentlyDone:         recent,
		NextUp:               nextUp,
		Blockers:             blockers,
		StaleInFlight:        stale,
		AgedBugs:             agedBugs,
		UnassignedCount:      unassignedCount,
		InferredCount:        inferred,
		OverrideAppliedCount: overridden,
		OverrideEditedAt:     overrideEdited,
		SourceNodes:          len(allSpecs),
		HasReleaseSignal:     hasReleaseSignal,
		GenerationMillis:     time.Since(start).Milliseconds(),
	}
	return out, nil
}

// surfaceFromSpec returns the spec's `surface:` frontmatter value.
// Prefers the parsed Spec.Surface field (post-Phase-10) and falls
// back to a raw-frontmatter scan for forward compatibility with
// in-memory Spec records produced before the field existed.
func surfaceFromSpec(s *spec.Spec) string {
	if s == nil {
		return ""
	}
	if s.Surface != "" {
		return s.Surface
	}
	if s.RawContent == "" {
		return ""
	}
	body := s.RawContent
	if !strings.HasPrefix(body, "---") {
		return ""
	}
	rest := strings.TrimPrefix(body, "---")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return ""
	}
	fm := rest[:end]
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "surface:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "surface:"))
			return strings.Trim(val, "\"'")
		}
	}
	return ""
}

// inferSurfaceFromPaths picks the surface whose declared paths best
// cover the spec's FilesTouched. Longest-prefix-match wins; ties
// resolved alphabetically by surface id.
func inferSurfaceFromPaths(s *spec.Spec, surfaces []Surface) string {
	if s == nil || len(s.FilesTouched) == 0 {
		return ""
	}
	bestSurface := ""
	bestLen := 0
	for _, srf := range surfaces {
		for _, p := range srf.Paths {
			p = strings.TrimSuffix(p, "/")
			if p == "" {
				continue
			}
			for _, touched := range s.FilesTouched {
				touched = filepath.ToSlash(touched)
				if strings.HasPrefix(touched, p+"/") || touched == p {
					if len(p) > bestLen {
						bestLen = len(p)
						bestSurface = srf.ID
					}
				}
			}
		}
	}
	return bestSurface
}

// normalizeParentTarget converts a parent reference to a slug.
// Handles both slug format ("hero-cli") and relative-path format
// ("../../initiatives/hero-cli/spec.md").
func normalizeParentTarget(target string) string {
	if !strings.Contains(target, "/") {
		return target
	}
	// Path format — extract the directory name containing spec.md.
	dir := filepath.Dir(target)
	return filepath.Base(dir)
}

func rollupInitiatives(allSpecs []*spec.Spec, assignments []SpecAssignment) []Initiative {
	// Build a map of initiative-slug → child specs.
	parentTo := map[string][]*spec.Spec{}
	for _, s := range allSpecs {
		if s == nil {
			continue
		}
		for _, rel := range s.Relations {
			if rel.Kind == "parent" || rel.Kind == "child-of" {
				target := normalizeParentTarget(rel.Target)
				parentTo[target] = append(parentTo[target], s)
			}
		}
	}

	// Surface mapping for assignments.
	surfaceForSlug := map[string]string{}
	for _, a := range assignments {
		if a.Spec != nil {
			surfaceForSlug[a.Spec.Slug] = a.SurfaceID
		}
	}

	var out []Initiative
	for _, s := range allSpecs {
		if s == nil || s.Type != spec.TypeInitiative {
			continue
		}
		children := parentTo[s.Slug]
		init := Initiative{
			Slug:   s.Slug,
			Title:  s.Title,
			Status: s.Status,
		}
		surfaceSet := map[string]bool{}
		for _, c := range children {
			init.Total++
			if c.Status == spec.StatusCompleted {
				init.Done++
			}
			if c.Status == spec.StatusDelivering {
				init.InFlight = append(init.InFlight, c.Slug)
			}
			if sid, ok := surfaceForSlug[c.Slug]; ok && sid != "" && sid != UnassignedSurfaceID {
				surfaceSet[sid] = true
			}
		}
		for sid := range surfaceSet {
			init.Surfaces = append(init.Surfaces, sid)
		}
		sort.Strings(init.Surfaces)
		sort.Strings(init.InFlight)
		if s.Status == spec.StatusCompleted {
			init.CompletedAt = s.ModifiedAt
		}
		out = append(out, init)
	}
	// Active first, then completed; alpha within group.
	sort.Slice(out, func(i, j int) bool {
		ai := out[i].Status == spec.StatusCompleted
		aj := out[j].Status == spec.StatusCompleted
		if ai != aj {
			return !ai
		}
		return out[i].Slug < out[j].Slug
	})
	return out
}

func rollupRecent(assignments []SpecAssignment, now time.Time, cutoff time.Duration, limit int) []RecentItem {
	var items []RecentItem
	threshold := now.Add(-cutoff)
	for _, a := range assignments {
		if a.Spec == nil || a.Spec.Status != spec.StatusCompleted {
			continue
		}
		if a.Spec.ModifiedAt.Before(threshold) {
			continue
		}
		items = append(items, RecentItem{
			SurfaceID:   a.SurfaceID,
			Slug:        a.Spec.Slug,
			Title:       a.Spec.Title,
			CompletedAt: a.Spec.ModifiedAt,
			Type:        a.Spec.Type,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CompletedAt.After(items[j].CompletedAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func rollupNext(assignments []SpecAssignment, limit int) []NextItem {
	var items []NextItem
	for _, a := range assignments {
		if a.Spec == nil {
			continue
		}
		st := a.Spec.Status
		if st != spec.StatusDelivering && st != spec.StatusPlanning && st != spec.StatusInReview {
			continue
		}
		items = append(items, NextItem{
			SurfaceID: a.SurfaceID,
			Slug:      a.Spec.Slug,
			Title:     a.Spec.Title,
			Status:    a.Spec.Status,
			Priority:  a.Spec.Priority,
			Horizon:   a.Spec.Horizon,
		})
	}
	// Sort by status (delivering first), priority, then alpha.
	sort.Slice(items, func(i, j int) bool {
		si := statusOrder(items[i].Status)
		sj := statusOrder(items[j].Status)
		if si != sj {
			return si < sj
		}
		pi := priorityOrder(items[i].Priority)
		pj := priorityOrder(items[j].Priority)
		if pi != pj {
			return pi < pj
		}
		return items[i].Slug < items[j].Slug
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func statusOrder(s spec.Status) int {
	switch s {
	case spec.StatusDelivering:
		return 0
	case spec.StatusInReview:
		return 1
	case spec.StatusPlanning:
		return 2
	}
	return 3
}

func priorityOrder(p string) int {
	switch strings.ToLower(p) {
	case "p0", "critical":
		return 0
	case "p1", "high":
		return 1
	case "p2", "medium":
		return 2
	case "p3", "low":
		return 3
	}
	return 4
}

func rollupBlockers(allSpecs []*spec.Spec, bySlug map[string]*spec.Spec) []Blocker {
	var out []Blocker
	for _, s := range allSpecs {
		if s == nil {
			continue
		}
		if !s.IsWorkSpec() {
			continue
		}
		if !spec.IsBlocked(s, bySlug) {
			continue
		}
		var waits []string
		for _, rel := range s.Relations {
			if rel.Kind != "depends-on" {
				continue
			}
			dep, ok := bySlug[rel.Target]
			if !ok {
				waits = append(waits, rel.Target+" (missing)")
				continue
			}
			if dep.Status != spec.StatusCompleted {
				waits = append(waits, rel.Target)
			}
		}
		if len(waits) == 0 {
			continue
		}
		out = append(out, Blocker{
			Slug:    s.Slug,
			Title:   s.Title,
			WaitsOn: waits,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}

func rollupStale(assignments []SpecAssignment, now time.Time, cutoff time.Duration) []StaleItem {
	var out []StaleItem
	for _, a := range assignments {
		if a.Spec == nil || a.Spec.Status != spec.StatusDelivering {
			continue
		}
		age := now.Sub(a.Spec.ModifiedAt)
		if age < cutoff {
			continue
		}
		out = append(out, StaleItem{
			Slug:      a.Spec.Slug,
			Title:     a.Spec.Title,
			StaleDays: int(age.Hours() / 24),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StaleDays > out[j].StaleDays })
	return out
}

func rollupAgedBugs(allSpecs []*spec.Spec, now time.Time, cutoff time.Duration) []AgedBug {
	var out []AgedBug
	for _, s := range allSpecs {
		if s == nil || s.Type != spec.TypeBug {
			continue
		}
		if s.Status == spec.StatusCompleted {
			continue
		}
		age := now.Sub(s.CreatedAt)
		if age < cutoff {
			continue
		}
		out = append(out, AgedBug{
			Slug:     s.Slug,
			Title:    s.Title,
			AgeDays:  int(age.Hours() / 24),
			Severity: s.Severity,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AgeDays > out[j].AgeDays })
	return out
}
