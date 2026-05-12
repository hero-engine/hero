package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MigrationPlan summarizes what would happen if a nested .hero/
// workspace were converted to a satellite of the root workspace.
//
// This is intentionally a planning-only structure for now. Phase 4 of
// the spec calls for the actual move (specs / knowledge / events
// migration), but doing it without an interactive user watching the
// outcome on real corpus data is a bad default. The plan format here is
// the contract — when the move is wired up, it will execute exactly
// the actions described in this plan.
type MigrationPlan struct {
	// NestedHeroDir is the absolute path to the nested .hero/ folder
	// that would be removed.
	NestedHeroDir string
	// SatellitePath is the relative path (forward-slash) of the
	// subproject under the root.
	SatellitePath string
	// Scope is the canonical scope identifier the migrated artifacts
	// will receive.
	Scope string
	// SpecsToMove lists planning files that would be moved into the
	// root .hero/planning/ tree.
	SpecsToMove []string
	// KnowledgeToMove lists knowledge files that would be moved.
	KnowledgeToMove []string
	// EventsToAppend is the path to events.log that would be appended
	// to root events.log with a migrated_from annotation.
	EventsToAppend string
	// FilesIgnored lists files inside the nested .hero/ that the
	// migration does NOT understand and would not move automatically
	// (e.g. unknown structures). Surfaced so the user can review.
	FilesIgnored []string
}

// PlanMigration inspects a nested .hero/ folder and produces a migration
// plan. It does NOT modify anything. The caller decides what to do with
// the plan (display it, confirm, then execute via ApplyMigration once
// implemented).
func PlanMigration(rootDir, nestedRel string) (*MigrationPlan, error) {
	rootAbs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}
	satAbs := filepath.Join(rootAbs, filepath.FromSlash(nestedRel))
	nestedHero := filepath.Join(satAbs, ".hero")
	if info, err := os.Stat(nestedHero); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("no nested .hero/ at %s", satAbs)
	}

	plan := &MigrationPlan{
		NestedHeroDir: nestedHero,
		SatellitePath: nestedRel,
		Scope:         nestedRel,
	}

	// Scan planning/ for spec.md files.
	planningDir := filepath.Join(nestedHero, "planning")
	if _, err := os.Stat(planningDir); err == nil {
		_ = filepath.WalkDir(planningDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
				return nil
			}
			plan.SpecsToMove = append(plan.SpecsToMove, path)
			return nil
		})
	}

	// Scan knowledge/ for files.
	knowledgeDir := filepath.Join(nestedHero, "knowledge")
	if _, err := os.Stat(knowledgeDir); err == nil {
		_ = filepath.WalkDir(knowledgeDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			plan.KnowledgeToMove = append(plan.KnowledgeToMove, path)
			return nil
		})
	}

	// Events log.
	eventsPath := filepath.Join(nestedHero, "events.log")
	if _, err := os.Stat(eventsPath); err == nil {
		plan.EventsToAppend = eventsPath
	}

	// Anything else inside .hero/ is flagged.
	known := map[string]bool{
		"planning":   true,
		"knowledge":  true,
		"events.log": true,
		"specs":      true,
		"hero.json":  true,
		"version.json": true,
		"context":    true,
		"conventions": true,
		"decisions":  true,
		"index.db":   true,
		"graph.db":   true,
		"mocks":      true,
		"NEXT.md":    true,
		"next":       true,
	}
	entries, _ := os.ReadDir(nestedHero)
	for _, e := range entries {
		if !known[e.Name()] {
			plan.FilesIgnored = append(plan.FilesIgnored, filepath.Join(nestedHero, e.Name()))
		}
	}

	sort.Strings(plan.SpecsToMove)
	sort.Strings(plan.KnowledgeToMove)
	sort.Strings(plan.FilesIgnored)
	return plan, nil
}

// FormatMigrationPlan renders a plan as a human-readable preview.
func FormatMigrationPlan(p *MigrationPlan) string {
	if p == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Migration plan for nested workspace at %s\n", p.SatellitePath))
	sb.WriteString(fmt.Sprintf("  Scope: %s\n\n", p.Scope))
	sb.WriteString(fmt.Sprintf("  %d spec(s) would be moved into root .hero/planning/ with scope frontmatter:\n", len(p.SpecsToMove)))
	for _, s := range p.SpecsToMove {
		sb.WriteString(fmt.Sprintf("    - %s\n", s))
	}
	sb.WriteString(fmt.Sprintf("  %d knowledge file(s) would be moved:\n", len(p.KnowledgeToMove)))
	for _, k := range p.KnowledgeToMove {
		sb.WriteString(fmt.Sprintf("    - %s\n", k))
	}
	if p.EventsToAppend != "" {
		sb.WriteString(fmt.Sprintf("  events.log would be appended to root events.log with migrated_from=%s\n", p.SatellitePath))
	}
	if len(p.FilesIgnored) > 0 {
		sb.WriteString(fmt.Sprintf("  %d file(s) inside nested .hero/ are not recognized and will NOT move automatically:\n", len(p.FilesIgnored)))
		for _, f := range p.FilesIgnored {
			sb.WriteString(fmt.Sprintf("    - %s\n", f))
		}
	}
	sb.WriteString("\n  After migration: nested .hero/ removed, satellite materialized,\n")
	sb.WriteString("  scope auto-stamped on artifacts, root graph re-indexed.\n")
	return sb.String()
}
