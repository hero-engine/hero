// Package cost provides effort calibration by comparing estimated vs actual
// delivery signals from the completed spec corpus and git history.
package cost

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

const (
	CalibrationFile      = "calibration.json"
	DefaultMinSpecs      = 10
	DefaultStaleDays     = 30
	MinTypeSpecsForRatio = 3
)

// Calibration is the top-level calibration data.
type Calibration struct {
	Generated   time.Time           `json:"generated"`
	SpecCount   int                 `json:"spec_count"`
	GlobalRatio float64             `json:"global_ratio"`
	ByType      map[string]TypeCal  `json:"by_type"`
	Entries     []CalibrationEntry  `json:"entries"`
}

// TypeCal holds per-type calibration metrics.
type TypeCal struct {
	Count      int     `json:"count"`
	AvgRatio   float64 `json:"avg_ratio"`
	AvgDays    float64 `json:"avg_days"`
	AvgCommits float64 `json:"avg_commits"`
}

// CalibrationEntry is one completed spec's calibration data.
type CalibrationEntry struct {
	Slug            string        `json:"slug"`
	Type            string        `json:"type"`
	EstimatedPoints float64       `json:"estimated_points"`
	ActualSignals   ActualSignals `json:"actual_signals"`
	ActualPoints    float64       `json:"actual_points"`
	Ratio           float64       `json:"ratio"`
}

// ActualSignals are the delivery signals measured from git/events.
type ActualSignals struct {
	DaysElapsed   float64 `json:"days_elapsed"`
	Commits       int     `json:"commits"`
	FilesChanged  int     `json:"files_changed"`
	CriteriaCount int     `json:"criteria_count"`
}

// LoadCalibration reads calibration.json from the knowledge directory.
func LoadCalibration(knowledgeDir string) (*Calibration, error) {
	path := filepath.Join(knowledgeDir, CalibrationFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cal Calibration
	if err := json.Unmarshal(data, &cal); err != nil {
		return nil, err
	}
	return &cal, nil
}

// SaveCalibration writes calibration.json.
func SaveCalibration(knowledgeDir string, cal *Calibration) error {
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cal, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(knowledgeDir, CalibrationFile), data, 0o644)
}

// BuildCalibration analyzes all completed specs against git history to
// produce calibration data. estimator is a function that computes raw
// estimated points for a spec (reusing the existing heuristic).
func BuildCalibration(heroDir, projectRoot string, estimator func(*spec.Spec) float64) (*Calibration, error) {
	allSpecs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, err
	}

	cal := &Calibration{
		Generated: time.Now().UTC(),
		ByType:    make(map[string]TypeCal),
	}

	var totalRatio float64
	for _, s := range allSpecs {
		if s.Status != spec.StatusCompleted || !s.IsWorkSpec() {
			continue
		}

		estimated := estimator(s)
		if estimated <= 0 {
			continue
		}

		signals := measureActual(s, projectRoot)
		actual := deriveActualPoints(signals)
		ratio := actual / estimated

		cal.Entries = append(cal.Entries, CalibrationEntry{
			Slug:            s.Slug,
			Type:            string(s.Type),
			EstimatedPoints: estimated,
			ActualSignals:   signals,
			ActualPoints:    actual,
			Ratio:           ratio,
		})
		totalRatio += ratio

		// Accumulate per-type
		tc := cal.ByType[string(s.Type)]
		tc.Count++
		tc.AvgRatio += ratio
		tc.AvgDays += signals.DaysElapsed
		tc.AvgCommits += float64(signals.Commits)
		cal.ByType[string(s.Type)] = tc
	}

	cal.SpecCount = len(cal.Entries)
	if cal.SpecCount > 0 {
		cal.GlobalRatio = totalRatio / float64(cal.SpecCount)
	}

	// Average out per-type accumulators
	for t, tc := range cal.ByType {
		if tc.Count > 0 {
			tc.AvgRatio /= float64(tc.Count)
			tc.AvgDays /= float64(tc.Count)
			tc.AvgCommits /= float64(tc.Count)
			cal.ByType[t] = tc
		}
	}

	return cal, nil
}

// measureActual extracts delivery signals from git and spec metadata.
func measureActual(s *spec.Spec, projectRoot string) ActualSignals {
	var signals ActualSignals

	// Days elapsed: created to last modified
	if !s.CreatedAt.IsZero() && !s.ModifiedAt.IsZero() {
		days := s.ModifiedAt.Sub(s.CreatedAt).Hours() / 24
		if days < 0 {
			days = 0
		}
		signals.DaysElapsed = days
	}

	// Commit count for files in Changes section
	if len(s.FilesTouched) > 0 {
		signals.Commits = countCommitsForFiles(projectRoot, s.FilesTouched)
		signals.FilesChanged = len(s.FilesTouched)
	}

	// Criteria count
	signals.CriteriaCount = len(s.AcceptanceCriteria())

	return signals
}

// deriveActualPoints converts raw signals into a single score.
func deriveActualPoints(s ActualSignals) float64 {
	// Weighted combination mirroring the estimation heuristic's scale
	points := 0.0
	points += s.DaysElapsed * 2.0          // 2 points per day
	points += float64(s.Commits) * 0.5     // 0.5 points per commit
	points += float64(s.FilesChanged) * 1.0 // 1 point per file
	points += float64(s.CriteriaCount) * 0.5 // 0.5 per criterion
	return points
}

// countCommitsForFiles counts unique commits touching any of the listed files.
func countCommitsForFiles(projectRoot string, files []string) int {
	if len(files) == 0 {
		return 0
	}

	args := []string{"-C", projectRoot, "log", "--oneline", "--"}
	args = append(args, files...)

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}

	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// TypeRatio returns the calibration ratio for a spec type.
// Falls back to global ratio if the type has insufficient data.
func (c *Calibration) TypeRatio(specType string) float64 {
	if tc, ok := c.ByType[specType]; ok && tc.Count >= MinTypeSpecsForRatio {
		return tc.AvgRatio
	}
	return c.GlobalRatio
}

// IsStale returns true if calibration data is older than maxDays.
func (c *Calibration) IsStale(maxDays int) bool {
	return time.Since(c.Generated).Hours()/24 > float64(maxDays)
}

// FormatHistory produces a human-readable calibration summary.
func FormatHistory(cal *Calibration) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Calibration data (%d completed specs)\n", cal.SpecCount)
	fmt.Fprintf(&b, "─────────────────────────────────────\n")

	for _, t := range []string{"feature", "bug", "initiative"} {
		tc, ok := cal.ByType[t]
		if !ok || tc.Count == 0 {
			continue
		}
		label := t + " specs"
		if t == "bug" {
			label = "bug fixes"
		}
		fmt.Fprintf(&b, "%-16s  %.2fx estimated (%d specs, avg %.1f days, avg %.0f commits)\n",
			label+":", tc.AvgRatio, tc.Count, tc.AvgDays, tc.AvgCommits)
	}

	fmt.Fprintf(&b, "\nOverall:          %.2fx estimated\n", cal.GlobalRatio)

	// Plain-English summary
	if cal.GlobalRatio > 1.1 {
		pct := (cal.GlobalRatio - 1.0) * 100
		fmt.Fprintf(&b, "\nSpecs in this repo take ~%.0f%% longer than raw estimates suggest.\n", pct)
	} else if cal.GlobalRatio < 0.9 {
		pct := (1.0 - cal.GlobalRatio) * 100
		fmt.Fprintf(&b, "\nRaw estimates are ~%.0f%% too pessimistic for this repo.\n", pct)
	} else {
		fmt.Fprintf(&b, "\nEstimates are well-calibrated for this repo.\n")
	}

	return b.String()
}

// FormatJSON returns calibration data as JSON.
func FormatJSON(cal *Calibration) (string, error) {
	data, err := json.MarshalIndent(cal, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// needed to avoid unused import in some builds
var _ = strconv.Atoi
