package snapshot

import (
	"strings"

	"github.com/hero-engine/hero/internal/spec"
)

// ReleaseResolver implements the priority chain for resolving a
// spec's release target. The chain (highest priority first):
//   1. Explicit `release_target:` on the spec frontmatter
//   2. Explicit `release_target:` on the parent initiative frontmatter
//   3. Tracker integration (Jira fixVersion / GitHub milestone / Linear cycle)
//   4. Git-tag heuristic (nearest later tag matching pattern)
//   5. None — caller treats the surface column as hidden
//
// Pure-function: ResolveRelease takes everything it needs as input.
type ReleaseResolver struct {
	// InitiativeReleases maps initiative slug → release_target name.
	// Built by the caller from frontmatter parsing.
	InitiativeReleases map[string]string

	// TrackerReleases maps spec slug → tracker-resolved release name.
	// Built by the caller from the tracker adapter; empty when
	// tracker integration is not configured.
	TrackerReleases map[string]string

	// GitTagReleases maps spec slug → git-tag-heuristic release name.
	// Built by the caller by scanning tags newer than the spec's
	// last-touched commit. Empty when no tag pattern is configured
	// or no tags match.
	GitTagReleases map[string]string
}

// ReleaseResolution is the output of ResolveRelease for one spec.
type ReleaseResolution struct {
	Target string // release target name; "" means none resolved
	Source string // "frontmatter" | "initiative" | "tracker" | "git-tag" | ""
}

// ResolveRelease walks the priority chain for a single spec. The
// spec's parent-initiative slug is looked up via the spec's
// Relations (kind == "parent").
func (r ReleaseResolver) ResolveRelease(s *spec.Spec) ReleaseResolution {
	if s == nil {
		return ReleaseResolution{}
	}

	// 1. Explicit on the spec itself.
	if rt := releaseTargetFromSpec(s); rt != "" {
		return ReleaseResolution{Target: rt, Source: "frontmatter"}
	}

	// 2. Parent initiative cascade.
	for _, rel := range s.Relations {
		if rel.Kind != "parent" && rel.Kind != "child-of" {
			continue
		}
		if rt, ok := r.InitiativeReleases[rel.Target]; ok && rt != "" {
			return ReleaseResolution{Target: rt, Source: "initiative"}
		}
	}

	// 3. Tracker integration.
	if rt, ok := r.TrackerReleases[s.Slug]; ok && rt != "" {
		return ReleaseResolution{Target: rt, Source: "tracker"}
	}

	// 4. Git-tag heuristic.
	if rt, ok := r.GitTagReleases[s.Slug]; ok && rt != "" {
		return ReleaseResolution{Target: rt, Source: "git-tag"}
	}

	// 5. None.
	return ReleaseResolution{}
}

// releaseTargetFromSpec extracts the release_target field. Prefers
// the parsed Spec.ReleaseTarget field; falls back to a raw-frontmatter
// scan when working with Spec records produced before the field was
// added (forward-compatible).
func releaseTargetFromSpec(s *spec.Spec) string {
	if s == nil {
		return ""
	}
	if s.ReleaseTarget != "" {
		return s.ReleaseTarget
	}
	if s.RawContent == "" {
		return ""
	}
	// Trim frontmatter delimiters and read line-by-line.
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
		if strings.HasPrefix(line, "release_target:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "release_target:"))
			val = strings.Trim(val, "\"'")
			return val
		}
	}
	return ""
}
