package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/managed"
)

// integrity.go — the install-integrity oracle: "is the install on disk what
// install would produce?"
//
// CheckIntegrity re-renders the managed body install would write right now
// (same defaultSections, same Options shape) and compares it against the
// body on disk in each installed target's native instruction file. It is
// READ-ONLY BY CONSTRUCTION: no managed.Writer.Write, no os.WriteFile —
// `hero check` reports, `hero install` / `hero upgrade` repair.
//
// Comparison is body-vs-body (managed.FindManagedRegion(...).Body against
// managed.Writer.RenderBody), never region-vs-region: the `v=` marker stamp
// records the writing binary's version and changes on every version bump,
// so including it would false-positive on 100% of upgrades.

// IntegrityKind classifies an install-integrity finding.
type IntegrityKind int

const (
	// IntegrityDamaged — the managed region is absent, or expected
	// sections are missing from it. Agents in this harness are running
	// cold; this is the failing-bucket signal.
	IntegrityDamaged IntegrityKind = iota
	// IntegrityStale — every expected section is present but the body
	// differs from what this binary would install (post-upgrade drift,
	// or a hand-edit inside the markers).
	IntegrityStale
)

// IntegrityFinding describes one damaged or stale root instruction file.
type IntegrityFinding struct {
	// Target is the installed target the finding is reported for. When
	// several installed targets share one instruction file (all
	// non-claude targets read AGENTS.md), this is the first of them in
	// stable resolution order — repairing with it regenerates the file
	// (and auto-sync refreshes the siblings).
	Target Target
	// File is the instruction file's base name ("AGENTS.md" / "CLAUDE.md").
	File string
	Kind IntegrityKind
	// MissingSections holds the expected section titles absent from the
	// on-disk managed body. Populated only for the sections-missing
	// damaged case; nil when the file or region is absent entirely.
	MissingSections []string
	// RepairCmd is the exact command that regenerates the file.
	RepairCmd string
}

// UnionTargets merges several target slices into one, deduping while
// preserving first-seen order (persisted set first, then probe extras).
// Shared by CheckIntegrity here and resolveUpgradeTargets in internal/cli
// so check and upgrade agree on what "installed" means.
func UnionTargets(sets ...[]Target) []Target {
	seen := map[Target]bool{}
	var out []Target
	for _, set := range sets {
		for _, t := range set {
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	return out
}

// CheckIntegrity compares each installed target's native instruction file
// against the managed body install would produce right now and returns one
// finding per damaged or stale file. A healthy install returns nil.
//
// base supplies the content source (ContentFS/SourceDir), Domain, and any
// body override — the caller must construct it the same way the install
// path does, so check resolves the same pack-body chain link install did.
// Target, Mode, and TargetDir are set here per installed target.
//
// The installed-target set is the union of PreviouslyInstalledTargets
// (persisted install-state.json, gitignored and therefore machine-local)
// and InferInstalledTargets (filesystem probe) — mirroring
// resolveUpgradeTargets. Targets never installed produce no findings, even
// when their instruction file exists on disk.
//
// Installed targets are grouped by instruction file: every non-claude
// target reads AGENTS.md, and multi-target installs are last-writer-wins
// (auto-sync), so the on-disk body legitimately matches exactly one of the
// group's renderings (codex appends a Codex-only subsection). A file is
// clean when its body matches ANY installed target's rendering for it —
// per-target strict equality would false-positive on every healthy
// codex-plus-sibling install.
func CheckIntegrity(projectRoot string, base Options) ([]IntegrityFinding, error) {
	if projectRoot == "" {
		return nil, nil
	}
	targets := UnionTargets(
		PreviouslyInstalledTargets(projectRoot),
		InferInstalledTargets(projectRoot),
	)
	if len(targets) == 0 {
		return nil, nil
	}

	// Group installed targets by the root instruction file they read,
	// preserving resolution order. Routing goes through
	// nativeInstructionFile — the single source of truth for the mapping.
	type fileGroup struct {
		file    string
		targets []Target
	}
	var groups []fileGroup
	groupIdx := map[string]int{}
	for _, t := range targets {
		name := nativeInstructionFile(t)
		if i, ok := groupIdx[name]; ok {
			groups[i].targets = append(groups[i].targets, t)
			continue
		}
		groupIdx[name] = len(groups)
		groups = append(groups, fileGroup{file: name, targets: []Target{t}})
	}

	var findings []IntegrityFinding
	for _, g := range groups {
		f, err := checkInstructionFileIntegrity(projectRoot, base, g.file, g.targets)
		if err != nil {
			return nil, err
		}
		if f != nil {
			findings = append(findings, *f)
		}
	}
	return findings, nil
}

// checkInstructionFileIntegrity checks one instruction file against the
// renderings of the installed targets that read it. Returns nil when the
// file is healthy.
func checkInstructionFileIntegrity(projectRoot string, base Options, file string, group []Target) (*IntegrityFinding, error) {
	path := filepath.Join(projectRoot, file)
	rep := group[0]
	repairCmd := fmt.Sprintf("hero install project . --target %s", rep)

	// Re-render the body each installed target's install would produce
	// (no filesystem writes). Expected section titles come from the first
	// rendering — the section list is identical across the group; only
	// body bytes differ (the Codex-only subsection is inside a section).
	var wants []string
	var expectedTitles []string
	for i, t := range group {
		opts := base
		opts.Target = t
		opts.Mode = ModeProject
		opts.TargetDir = projectRoot
		sections := defaultSections(opts, path)
		ctx := managed.Context{
			File:        path,
			HeroVersion: opts.heroVersion(),
			ProjectDir:  projectRoot,
		}
		want, err := managed.Writer{File: path, Sections: sections}.RenderBody(ctx)
		if err != nil {
			return nil, fmt.Errorf("render expected body for %s (%s): %w", file, t, err)
		}
		wants = append(wants, want)
		if i == 0 {
			for _, sec := range sections {
				body, err := sec.Render(ctx)
				if err != nil {
					return nil, fmt.Errorf("render section %s for %s: %w", sec.SectionID(), file, err)
				}
				// Match renderBody's skip: empty sections emit no heading.
				if strings.Trim(body, "\n") == "" {
					continue
				}
				if title := sec.SectionTitle(); title != "" {
					expectedTitles = append(expectedTitles, title)
				}
			}
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return &IntegrityFinding{Target: rep, File: file, Kind: IntegrityDamaged, RepairCmd: repairCmd}, nil
	}

	region := managed.FindManagedRegion(string(data))
	if !region.Present {
		return &IntegrityFinding{Target: rep, File: file, Kind: IntegrityDamaged, RepairCmd: repairCmd}, nil
	}

	var missing []string
	for _, title := range expectedTitles {
		if !bodyHasSectionHeading(region.Body, title) {
			missing = append(missing, title)
		}
	}
	if len(missing) > 0 {
		return &IntegrityFinding{Target: rep, File: file, Kind: IntegrityDamaged, MissingSections: missing, RepairCmd: repairCmd}, nil
	}

	for _, want := range wants {
		if region.Body == want {
			return nil, nil
		}
	}
	return &IntegrityFinding{Target: rep, File: file, Kind: IntegrityStale, RepairCmd: repairCmd}, nil
}

// bodyHasSectionHeading reports whether the managed body carries the H2
// heading the orchestrator emits for a section title (line-anchored, so a
// title appearing as prose or as a longer heading's prefix doesn't count).
func bodyHasSectionHeading(body, title string) bool {
	heading := "## " + title
	if body == heading || strings.HasPrefix(body, heading+"\n") {
		return true
	}
	return strings.Contains(body, "\n"+heading+"\n") || strings.HasSuffix(body, "\n"+heading)
}
