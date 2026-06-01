// Package sizing computes effort estimates and detects size drift on
// specs. It is the shared backend for:
//
//   - `hero estimate` / `hero size --check` (internal/cli)
//   - `hero check` size-drift summary lines (internal/cli)
//   - `hero_warnings` MCP tool size-drift entries (internal/serve)
//
// Keeping these in one package avoids dual-truth drift between the
// CLI and MCP surfaces. See spec spec-size-and-promotion-nudge.
package sizing

import (
	"math"
	"strings"

	"github.com/hero-engine/hero/internal/snapshot"
	"github.com/hero-engine/hero/internal/spec"
)

// Effort tier constants — mirror internal/cli/cost.go. These strings
// are load-bearing: the calibration cache and hero_velocity history
// persist them literally, so do NOT rename.
const (
	EffortTrivial = "trivial"
	EffortSmall   = "small"
	EffortMedium  = "medium"
	EffortLarge   = "large"
	EffortXLarge  = "x-large"
	EffortGiant   = "giant"
)

// Estimate is the per-spec output of the effort heuristic.
type Estimate struct {
	Slug         string
	Title        string
	Type         spec.Type
	Points       float64
	Bucket       string
	Declared     string
	Drift        bool
	FileCount    int
	SectionCount int
	DependsCount int
	WordCount    int
}

// Calibration holds historical project averages used to scale raw
// estimates. Zero value is fine — HasHistory=false short-circuits the
// calibration blend.
type Calibration struct {
	CompletedCount int
	AvgFiles       float64
	AvgWords       float64
	AvgSections    float64
	HasHistory     bool
}

// Calibrate computes project-level averages from completed work specs.
// Knowledge/other types are ignored so they don't pollute the signal.
func Calibrate(specs []*spec.Spec) Calibration {
	var cal Calibration
	var totalFiles, totalWords, totalSections int
	for _, s := range specs {
		if s == nil || s.Status != spec.StatusCompleted || !s.IsWorkSpec() {
			continue
		}
		cal.CompletedCount++
		totalFiles += len(s.FilesTouched)
		totalSections += len(s.Sections)
		wc := 0
		for _, content := range s.Sections {
			wc += countWords(content)
		}
		totalWords += wc
	}
	if cal.CompletedCount > 0 {
		cal.HasHistory = true
		cal.AvgFiles = float64(totalFiles) / float64(cal.CompletedCount)
		cal.AvgWords = float64(totalWords) / float64(cal.CompletedCount)
		cal.AvgSections = float64(totalSections) / float64(cal.CompletedCount)
	}
	return cal
}

// EstimateSpec produces the computed bucket + declared-vs-computed
// drift signal for a single spec. Empty declared is a normal state
// (drift=false).
func EstimateSpec(s *spec.Spec, cal Calibration) Estimate {
	est := Estimate{
		Slug:         s.Slug,
		Title:        s.Title,
		Type:         s.Type,
		FileCount:    len(s.FilesTouched),
		SectionCount: len(s.Sections),
		DependsCount: countDependencies(s),
	}
	for _, content := range s.Sections {
		est.WordCount += countWords(content)
	}

	filePoints := float64(est.FileCount) * 1.5
	sectionPoints := float64(est.SectionCount) * 0.5
	depPoints := float64(est.DependsCount) * 2.0

	wordPoints := 0.0
	switch {
	case est.WordCount > 500:
		wordPoints = 2.0
	case est.WordCount > 200:
		wordPoints = 1.0
	case est.WordCount > 50:
		wordPoints = 0.5
	}

	typeMultiplier := 1.0
	switch s.Type {
	case spec.TypeBug:
		typeMultiplier = 0.7
	case spec.TypeInitiative:
		typeMultiplier = 2.0
	}

	raw := (filePoints + sectionPoints + depPoints + wordPoints) * typeMultiplier
	if raw < 1.0 && (est.FileCount > 0 || est.WordCount > 10) {
		raw = 1.0
	}
	if cal.HasHistory && cal.AvgFiles > 0 && est.FileCount > 0 {
		relativeSize := float64(est.FileCount) / cal.AvgFiles
		calibratedPoints := relativeSize * 5.0
		raw = raw*0.6 + calibratedPoints*0.4
	}

	est.Points = math.Round(raw*10) / 10
	est.Bucket = BucketFromPoints(est.Points)
	est.Declared = s.Size
	if est.Declared != "" && est.Declared != est.Bucket {
		est.Drift = true
	}
	return est
}

// BucketFromPoints maps a points value to a tier on the 6-tier
// ladder. Thresholds must stay in sync with
// internal/snapshot/rollup.go's containerBucketFromPoints.
func BucketFromPoints(points float64) string {
	switch {
	case points < 2:
		return EffortTrivial
	case points < 5:
		return EffortSmall
	case points < 10:
		return EffortMedium
	case points < 20:
		return EffortLarge
	case points < 40:
		return EffortXLarge
	default:
		return EffortGiant
	}
}

// CollectDrift walks a corpus and returns the two drift flavors:
// leaf (declared-vs-computed on feature/bug/enhancement specs that
// carry a declared `size:`) and container (declared-vs-rollup on
// initiatives). The split mirrors `hero size --check` and is reused
// by `hero check` summary lines and the `hero_warnings` MCP tool.
func CollectDrift(specs []*spec.Spec) (leaf []Estimate, container []snapshot.ContainerDriftReport) {
	calibration := Calibrate(specs)
	sizeFn := func(c *spec.Spec) string {
		return EstimateSpec(c, calibration).Bucket
	}
	parentMap := snapshot.BuildParentMap(specs)

	for _, s := range specs {
		if s == nil {
			continue
		}
		if s.Type == spec.TypeInitiative {
			children := parentMap[s.Slug]
			if d := snapshot.ContainerDrift(s, children, sizeFn); d != nil {
				container = append(container, *d)
			}
			continue
		}
		if s.Size == "" {
			continue
		}
		est := EstimateSpec(s, calibration)
		if est.Drift {
			leaf = append(leaf, est)
		}
	}
	return leaf, container
}

// TrackerCapability is the small projection of tracker state that the
// spec-sizing skill needs to pick a nudge intensity. The skill
// regimes are:
//
//   - tracker not configured                           — most aggressive
//   - tracker configured, hierarchy unsupported        — most aggressive
//   - tracker configured, hierarchy supported          — less aggressive
//
// See domains/engineering/skills/spec-sizing/SKILL.md "Tracker-aware
// tuning" for the full table.
type TrackerCapability struct {
	// Configured reports whether a tracker is wired up at all
	// (`hero.json: tracker.type != "none"`).
	Configured bool
	// Type is the configured tracker type ("jira", "linear",
	// "github", "none"). Empty when Configured is false.
	Type string
	// SupportsHierarchy mirrors the adapter's SupportsHierarchy()
	// method. Always false when Configured is false.
	SupportsHierarchy bool
}

// NudgeRegime returns the short label the skill uses to describe the
// current intensity ("most-aggressive" / "less-aggressive"). Keeping
// it as a string rather than an enum keeps the surface humane for
// CLI output.
func (c TrackerCapability) NudgeRegime() string {
	if !c.Configured || !c.SupportsHierarchy {
		return "most-aggressive"
	}
	return "less-aggressive"
}

func countDependencies(s *spec.Spec) int {
	count := 0
	for _, r := range s.Relations {
		if r.Kind == "depends-on" || r.Kind == "parent" {
			count++
		}
	}
	return count
}

func countWords(s string) int {
	return len(strings.Fields(s))
}
