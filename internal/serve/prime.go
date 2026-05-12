package serve

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/codescan"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// BuildPrimeContext gathers session context from discovered specs and returns
// a formatted summary suitable for MCP Instructions. It performs no index
// rebuild—just spec discovery and formatting.
func BuildPrimeContext(heroDir, projectRoot string, includeKnowledge bool) (string, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return "", fmt.Errorf("prime: discover specs: %w", err)
	}

	projectName := filepath.Base(projectRoot)

	// Partition specs into buckets.
	var delivering, inReview, planning []*spec.Spec
	var conventions, decisions, tripwires []*spec.Spec
	staleCount := 0
	unclaimedDelivering := 0
	staleCutoff := time.Now().AddDate(0, 0, -14)

	for _, s := range specs {
		switch {
		case s.Type == spec.TypeConvention && (s.Status == spec.StatusActive || s.Status == spec.StatusDraft):
			conventions = append(conventions, s)
		case s.Type == spec.TypeDecision:
			decisions = append(decisions, s)
		case s.Type == spec.TypeTripwire && s.Status == spec.StatusActive:
			tripwires = append(tripwires, s)
		}

		switch s.Status {
		case spec.StatusDelivering:
			delivering = append(delivering, s)
			if s.ClaimedBy == "" {
				unclaimedDelivering++
			}
		case spec.StatusInReview:
			inReview = append(inReview, s)
		case spec.StatusPlanning:
			planning = append(planning, s)
		}

		if s.IsWorkSpec() && s.IsInFlight() && !s.ModifiedAt.IsZero() && s.ModifiedAt.Before(staleCutoff) {
			staleCount++
		}
	}

	var b strings.Builder

	fmt.Fprintf(&b, "Hero project: %s\n", projectName)

	// Active Work
	activeSpecs := concat(delivering, inReview, planning)
	if len(activeSpecs) > 0 {
		b.WriteString("\n## Active Work\n")
		for _, s := range activeSpecs {
			claimed := ""
			if s.ClaimedBy != "" {
				claimed = fmt.Sprintf(" [claimed by: %s]", s.ClaimedBy)
			}
			fmt.Fprintf(&b, "- %s (%s, %s): %s%s\n", s.Slug, s.Type, s.Status, s.Title, claimed)
		}
	}

	// Tripwires — always included, not gated by includeKnowledge.
	// These are forbidden-option guardrails that must survive context
	// compaction. Short and high-signal.
	if len(tripwires) > 0 {
		b.WriteString("\n## Tripwires (Do Not Violate)\n")
		for _, s := range tripwires {
			sev := s.Severity
			if sev == "" {
				sev = "high"
			}
			constraint := s.Sections["constraint"]
			instead := s.Sections["instead"]
			if constraint != "" {
				fmt.Fprintf(&b, "- **%s** [%s]: %s", s.Slug, sev, constraint)
				if instead != "" {
					fmt.Fprintf(&b, " Instead: %s", instead)
				}
				b.WriteString("\n")
			} else {
				fmt.Fprintf(&b, "- **%s** [%s]: %s\n", s.Slug, sev, s.Title)
			}
		}
	}

	// Conventions
	if includeKnowledge && len(conventions) > 0 {
		b.WriteString("\n## Conventions\n")
		for _, s := range conventions {
			fmt.Fprintf(&b, "- %s: %s\n", s.Slug, s.Title)
		}
	}

	// Decisions
	if includeKnowledge && len(decisions) > 0 {
		b.WriteString("\n## Decisions\n")
		for _, s := range decisions {
			fmt.Fprintf(&b, "- %s (%s): %s\n", s.Slug, s.Status, s.Title)
		}
	}

	// Watch For
	var warnings []string
	if staleCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d stale specs", staleCount))
	}
	if unclaimedDelivering > 0 {
		warnings = append(warnings, fmt.Sprintf("%d unclaimed delivering specs", unclaimedDelivering))
	}
	if len(warnings) > 0 {
		b.WriteString("\n## Watch For\n")
		fmt.Fprintf(&b, "- %s\n", strings.Join(warnings, ", "))
	}

	b.WriteString("\nUse hero_context, hero_search, hero_read_spec, hero_code tools for detailed information.\n")

	// Code structure summary (if available)
	if cfg, err := config.Load(projectRoot); err == nil && !cfg.CodeScan.IsDisabled() {
		codeDir := cfg.CodeDir(projectRoot)
		if overview, err := codescan.GetOverview(codeDir); err == nil && overview != "" {
			// Extract just the package count and top-level summary, not the full overview
			lines := strings.Split(overview, "\n")
			var summaryLines []string
			for _, line := range lines {
				if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "**") {
					summaryLines = append(summaryLines, line)
				}
				if len(summaryLines) >= 5 {
					break
				}
			}
			if len(summaryLines) > 0 {
				b.WriteString("\n## Code Structure\n")
				for _, line := range summaryLines {
					b.WriteString(line + "\n")
				}
				b.WriteString("\nUse hero_code tool for symbol search, package details, dependency graph, and hot files.\n")
			}
		}
	}

	return b.String(), nil
}

// concat joins multiple spec slices into one.
func concat(slices ...[]*spec.Spec) []*spec.Spec {
	var out []*spec.Spec
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}

// PrimeContextJSON is the structured JSON representation of prime context.
type PrimeContextJSON struct {
	Project     string              `json:"project"`
	ActiveWork  []PrimeSpecJSON     `json:"active_work"`
	Tripwires   []PrimeTripwireJSON `json:"tripwires,omitempty"`
	Conventions []PrimeSpecJSON     `json:"conventions,omitempty"`
	Decisions   []PrimeSpecJSON     `json:"decisions,omitempty"`
	Warnings    []string            `json:"warnings,omitempty"`
}

// PrimeSpecJSON is a single spec entry in the prime context JSON output.
type PrimeSpecJSON struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	ClaimedBy string `json:"claimed_by,omitempty"`
	TrackerID string `json:"tracker_id,omitempty"`
}

// PrimeTripwireJSON is a tripwire entry in the prime context JSON output.
type PrimeTripwireJSON struct {
	Slug       string   `json:"slug"`
	Title      string   `json:"title"`
	Severity   string   `json:"severity"`
	Constraint string   `json:"constraint"`
	Instead    string   `json:"instead,omitempty"`
	Triggers   []string `json:"triggers,omitempty"`
}

// BuildPrimeContextJSON gathers session context and returns a structured
// representation suitable for JSON serialization.
func BuildPrimeContextJSON(heroDir, projectRoot string, includeKnowledge bool) (*PrimeContextJSON, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("prime: discover specs: %w", err)
	}

	projectName := filepath.Base(projectRoot)
	staleCutoff := time.Now().AddDate(0, 0, -14)

	result := &PrimeContextJSON{
		Project: projectName,
	}

	staleCount := 0
	unclaimedDelivering := 0

	for _, s := range specs {
		js := PrimeSpecJSON{
			Slug:      s.Slug,
			Title:     s.Title,
			Type:      string(s.Type),
			Status:    string(s.Status),
			ClaimedBy: s.ClaimedBy,
			TrackerID: s.TrackerID,
		}

		switch {
		case s.Type == spec.TypeTripwire && s.Status == spec.StatusActive:
			sev := s.Severity
			if sev == "" {
				sev = "high"
			}
			result.Tripwires = append(result.Tripwires, PrimeTripwireJSON{
				Slug:       s.Slug,
				Title:      s.Title,
				Severity:   sev,
				Constraint: s.Sections["constraint"],
				Instead:    s.Sections["instead"],
				Triggers:   s.Triggers,
			})
		case s.Type == spec.TypeConvention && includeKnowledge && (s.Status == spec.StatusActive || s.Status == spec.StatusDraft):
			result.Conventions = append(result.Conventions, js)
		case s.Type == spec.TypeDecision && includeKnowledge:
			result.Decisions = append(result.Decisions, js)
		}

		if s.IsInFlight() {
			result.ActiveWork = append(result.ActiveWork, js)
			if s.Status == spec.StatusDelivering && s.ClaimedBy == "" {
				unclaimedDelivering++
			}
		}

		if s.IsWorkSpec() && s.IsInFlight() && !s.ModifiedAt.IsZero() && s.ModifiedAt.Before(staleCutoff) {
			staleCount++
		}
	}

	if staleCount > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%d stale specs", staleCount))
	}
	if unclaimedDelivering > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%d unclaimed delivering specs", unclaimedDelivering))
	}

	return result, nil
}
