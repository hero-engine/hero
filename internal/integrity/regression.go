package integrity

import (
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/acceptance"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/spec"
)

// RegressionDowngrade describes one auto-downgrade that fired (or
// would fire) because an AC went red on a previously-completed spec.
type RegressionDowngrade struct {
	Slug         string      // parent spec slug
	Path         string      // spec.md path
	OldStatus    spec.Status // typically StatusCompleted
	NewStatus    spec.Status // typically StatusRegressed
	RegressedACs []string    // failing/regressed AC keys
}

// AutoDowngradeRegressions scans the graph for Criterion nodes whose
// current status is failing/regressed and downgrades any parent spec
// whose frontmatter still claims completed.
//
// The bridge logic that closes spec-status-integrity AC-6: a regressed
// AC on a "shipped" spec is structural drift; this function turns the
// frontmatter back into the truth.
//
// Returns the list of downgrades performed (or planned, when dryRun
// is true). Idempotent: re-running on a spec already at StatusRegressed
// is a no-op.
func AutoDowngradeRegressions(specs []*spec.Spec, store *graph.Store, dryRun bool) ([]RegressionDowngrade, error) {
	if store == nil {
		return nil, fmt.Errorf("integrity: nil store")
	}

	// Index specs by slug for fast parent lookup.
	bySlug := make(map[string]*spec.Spec, len(specs))
	for _, s := range specs {
		if s == nil || s.Slug == "" {
			continue
		}
		bySlug[s.Slug] = s
	}

	failing, err := acceptance.FailingAcrossCorpus(store)
	if err != nil {
		return nil, fmt.Errorf("query failing criteria: %w", err)
	}

	// Group failing ACs by parent spec.
	byParent := map[string][]string{}
	for _, c := range failing {
		if c.Parent == "" {
			continue
		}
		byParent[c.Parent] = append(byParent[c.Parent], c.Key)
	}

	out := make([]RegressionDowngrade, 0, len(byParent))
	for parent, acs := range byParent {
		s, ok := bySlug[parent]
		if !ok {
			continue
		}
		// Only downgrade specs that currently claim completed. A spec
		// already at StatusRegressed (or any other state) is left alone.
		if s.Status != spec.StatusCompleted {
			continue
		}
		dg := RegressionDowngrade{
			Slug:         s.Slug,
			Path:         s.Path,
			OldStatus:    s.Status,
			NewStatus:    spec.StatusRegressed,
			RegressedACs: acs,
		}
		out = append(out, dg)
		if dryRun {
			continue
		}
		if err := applyRegressionDowngrade(dg); err != nil {
			return out, fmt.Errorf("downgrade %s: %w", s.Slug, err)
		}
	}
	return out, nil
}

// applyRegressionDowngrade rewrites a spec's frontmatter status to
// StatusRegressed and stamps an auto_downgraded annotation listing
// the AC keys that triggered it.
func applyRegressionDowngrade(dg RegressionDowngrade) error {
	data, err := os.ReadFile(dg.Path)
	if err != nil {
		return err
	}
	reason := fmt.Sprintf("auto-downgrade: AC regression in %s",
		strings.Join(dg.RegressedACs, ", "))
	rewritten, changed, err := rewriteFrontmatterStatus(data, string(dg.NewStatus), reason)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	return os.WriteFile(dg.Path, rewritten, 0o644)
}
