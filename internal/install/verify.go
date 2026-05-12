package install

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// verify.go — single-source-install P4 verify command.
//
// `hero verify-install` audits the on-disk install state against what
// `hero install` is supposed to produce. It's an inspection-only
// command (read-only — no filesystem changes). Three primary uses:
//
//   1. Spot check after install: did everything land as expected?
//   2. Ongoing health: re-run on CI / `hero check` to catch drift
//      that crept in (Cline manual edits, Windows-no-symlinks user
//      editing rendered copies, etc.).
//   3. Pre-flight: confirm a clean baseline before a release or a
//      migration.
//
// Check categories:
//
//   - **expected_symlink**: a harness content dir is a regular directory
//     where P2 would have a symlink to canonical. Drift risk — content
//     can desync from canonical. Severity: warning.
//
//   - **broken_symlink**: a symlink whose target doesn't exist. The
//     harness sees nothing through it. Severity: error.
//
//   - **wrong_symlink_target**: a symlink that points somewhere other
//     than the resolved canonical path for that kind. Could be a
//     legitimate user choice or stale state. Severity: warning.
//
//   - **symlink_escape**: a symlink whose target resolves outside the
//     project root. Security boundary — Hero shouldn't be linking at
//     arbitrary filesystem paths. Severity: error.
//
//   - **drifted_rendered**: rendered-copy mode (no symlinks) — a file
//     in a harness dir has different content from the canonical file
//     of the same name. Drift. Severity: warning.
//
//   - **missing_canonical**: a canonical directory doesn't exist even
//     though some harness target's symlink points at it. The harness
//     dir is broken. Severity: error.
//
//   - **mixed_mode**: across detected targets, some are in symlink mode
//     and some are in rendered mode. Usually fine (e.g., Cline forces
//     rendered) but worth flagging. Severity: info.

// VerificationIssue describes a single finding from the verify pass.
type VerificationIssue struct {
	Severity string `json:"severity"` // "error" | "warning" | "info"
	Code     string `json:"code"`     // one of the codes documented above
	Path     string `json:"path"`     // the affected filesystem path (relative to project root where possible)
	Message  string `json:"message"`  // one-line summary
	Detail   string `json:"detail,omitempty"` // optional multi-line diagnostic
}

// VerificationReport is the result of a verify pass.
type VerificationReport struct {
	TargetDir       string              `json:"target_dir"`
	DetectedTargets []Target            `json:"detected_targets"`
	Issues          []VerificationIssue `json:"issues"`
	// Clean is true when the report contains no error- or
	// warning-severity issues.
	Clean bool `json:"clean"`
}

// HasErrors reports whether at least one error-severity issue exists.
func (r *VerificationReport) HasErrors() bool {
	for _, i := range r.Issues {
		if i.Severity == "error" {
			return true
		}
	}
	return false
}

// StringReport produces a human-readable summary suitable for CLI
// output. Sorts issues by severity (errors first, then warnings,
// then info) and groups by code.
func (r *VerificationReport) StringReport() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Verifying install at %s\n", r.TargetDir))
	sb.WriteString(fmt.Sprintf("Detected targets: %v\n\n", targetNames(r.DetectedTargets)))

	if len(r.Issues) == 0 {
		sb.WriteString("✓ no issues found.\n")
		return sb.String()
	}

	// Stable severity order: error, warning, info.
	severityOrder := map[string]int{"error": 0, "warning": 1, "info": 2}
	issues := make([]VerificationIssue, len(r.Issues))
	copy(issues, r.Issues)
	sort.SliceStable(issues, func(i, j int) bool {
		if severityOrder[issues[i].Severity] != severityOrder[issues[j].Severity] {
			return severityOrder[issues[i].Severity] < severityOrder[issues[j].Severity]
		}
		return issues[i].Code < issues[j].Code
	})

	for _, issue := range issues {
		sb.WriteString(fmt.Sprintf("%s [%s] %s: %s\n",
			severityIcon(issue.Severity), issue.Severity, issue.Code, issue.Message))
		if issue.Path != "" {
			sb.WriteString(fmt.Sprintf("    path: %s\n", issue.Path))
		}
		if issue.Detail != "" {
			for _, line := range strings.Split(issue.Detail, "\n") {
				sb.WriteString("    " + line + "\n")
			}
		}
	}

	if r.HasErrors() {
		sb.WriteString("\nResult: install has errors — run `hero install project " + r.TargetDir + " --migrate` to recover.\n")
	}
	return sb.String()
}

func severityIcon(s string) string {
	switch s {
	case "error":
		return "✗"
	case "warning":
		return "!"
	default:
		return "·"
	}
}

// RunVerify audits the install state at targetDir and returns a
// report. Read-only: no filesystem modifications.
func RunVerify(targetDir string) (*VerificationReport, error) {
	abs, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("resolving target path: %w", err)
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("target directory does not exist: %s", abs)
	}

	report := &VerificationReport{
		TargetDir:       abs,
		DetectedTargets: DetectInstalledTargets(abs),
	}

	// Resolve canonical paths for this project — what each harness's
	// content dir SHOULD point at.
	agentsCanonical, commandsCanonical, skillsCanonical, err := ResolveCanonicalDirs(abs)
	if err != nil {
		// Configured external path missing — that's its own issue.
		report.Issues = append(report.Issues, VerificationIssue{
			Severity: "error",
			Code:     "missing_canonical",
			Path:     abs,
			Message:  "configured content path missing",
			Detail:   err.Error(),
		})
		report.Clean = false
		return report, nil
	}

	canonicalByKind := map[string]string{
		"agents":   agentsCanonical,
		"commands": commandsCanonical,
		"skills":   skillsCanonical,
	}

	// For each detected target × kind, check the harness dir against
	// the resolved canonical.
	modesSeen := map[string]bool{}
	for _, target := range report.DetectedTargets {
		layout := LayoutFor(target)
		if layout == nil {
			continue
		}
		for _, kind := range []string{"agents", "commands", "skills"} {
			harnessDir := filepath.Join(abs, layout.SubDir, kind)
			canonical := canonicalByKind[kind]

			mode, issues := verifyContentDir(abs, harnessDir, canonical, kind, target)
			if mode != "" {
				modesSeen[mode] = true
			}
			report.Issues = append(report.Issues, issues...)
		}
	}

	// Mixed-mode signal (info, not an error).
	if len(modesSeen) > 1 {
		modes := make([]string, 0, len(modesSeen))
		for m := range modesSeen {
			modes = append(modes, m)
		}
		sort.Strings(modes)
		report.Issues = append(report.Issues, VerificationIssue{
			Severity: "info",
			Code:     "mixed_mode",
			Message:  fmt.Sprintf("install uses mixed modes across targets: %s", strings.Join(modes, ", ")),
			Detail:   "this is fine when harnesses have different capability constraints (e.g., Cline always rendered), but worth knowing.",
		})
	}

	// Determine cleanliness.
	hasIssue := false
	for _, i := range report.Issues {
		if i.Severity == "error" || i.Severity == "warning" {
			hasIssue = true
			break
		}
	}
	report.Clean = !hasIssue

	return report, nil
}

// verifyContentDir audits a single harness content directory against
// canonical. Returns the detected install mode ("symlink", "rendered",
// "missing", or empty when no decision can be made) plus any issues.
func verifyContentDir(projectRoot, harnessDir, canonical, kind string, target Target) (string, []VerificationIssue) {
	rel := relPath(projectRoot, harnessDir)

	info, err := os.Lstat(harnessDir)
	if err != nil {
		if os.IsNotExist(err) {
			// The target has SOME content dirs but not this kind —
			// fine (e.g., target without skills concept). No issue.
			return "", nil
		}
		return "", []VerificationIssue{{
			Severity: "error",
			Code:     "stat_failed",
			Path:     rel,
			Message:  fmt.Sprintf("cannot stat %s: %v", rel, err),
		}}
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return verifySymlink(projectRoot, harnessDir, canonical, kind)
	}

	if info.IsDir() {
		return verifyRenderedDir(projectRoot, harnessDir, canonical, kind, target)
	}

	return "", []VerificationIssue{{
		Severity: "error",
		Code:     "unexpected_file_type",
		Path:     rel,
		Message:  fmt.Sprintf("%s is neither a directory nor a symlink (mode %s)", rel, info.Mode()),
	}}
}

// verifySymlink checks a content-dir symlink: target resolves, points
// inside the project, and matches the canonical path.
func verifySymlink(projectRoot, harnessDir, canonical, kind string) (string, []VerificationIssue) {
	rel := relPath(projectRoot, harnessDir)
	var issues []VerificationIssue

	linkTarget, err := os.Readlink(harnessDir)
	if err != nil {
		issues = append(issues, VerificationIssue{
			Severity: "error",
			Code:     "broken_symlink",
			Path:     rel,
			Message:  fmt.Sprintf("cannot read symlink target: %v", err),
		})
		return "symlink", issues
	}

	// Resolve the symlink target to an absolute path.
	resolved := linkTarget
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(filepath.Dir(harnessDir), linkTarget)
	}
	resolved = filepath.Clean(resolved)

	// Check the resolved path exists.
	if _, err := os.Stat(resolved); err != nil {
		issues = append(issues, VerificationIssue{
			Severity: "error",
			Code:     "broken_symlink",
			Path:     rel,
			Message:  fmt.Sprintf("symlink target does not exist: %s", linkTarget),
			Detail:   fmt.Sprintf("resolved to: %s", resolved),
		})
		return "symlink", issues
	}

	// Check the resolved path is inside the project root (security).
	projAbs, _ := filepath.Abs(projectRoot)
	if !strings.HasPrefix(resolved+string(filepath.Separator), projAbs+string(filepath.Separator)) && resolved != projAbs {
		issues = append(issues, VerificationIssue{
			Severity: "error",
			Code:     "symlink_escape",
			Path:     rel,
			Message:  "symlink target resolves outside the project root",
			Detail:   fmt.Sprintf("link target: %s\nresolved: %s\nproject root: %s", linkTarget, resolved, projAbs),
		})
		return "symlink", issues
	}

	// Check the target matches the configured canonical path.
	canonicalClean := filepath.Clean(canonical)
	if resolved != canonicalClean {
		issues = append(issues, VerificationIssue{
			Severity: "warning",
			Code:     "wrong_symlink_target",
			Path:     rel,
			Message:  fmt.Sprintf("symlink points at %s, expected canonical %s", resolved, canonicalClean),
			Detail:   "this may be a stale install state, or an intentional user-configured override — re-run `hero install --force` to sync, or update hero.json's content.* paths if intentional.",
		})
	}

	return "symlink", issues
}

// verifyRenderedDir checks a rendered (regular directory) harness
// content dir against canonical. Two findings possible:
//
//   - expected_symlink: P2 default is symlink; a regular dir is drift
//     risk. Warning.
//   - drifted_rendered: files in the rendered dir differ from
//     canonical. Warning (per file).
//
// When canonical doesn't exist at all (project never installed under
// P2), we don't have a baseline to compare against and just return
// "rendered" without per-file checks.
func verifyRenderedDir(projectRoot, harnessDir, canonical, kind string, target Target) (string, []VerificationIssue) {
	rel := relPath(projectRoot, harnessDir)
	var issues []VerificationIssue

	issues = append(issues, VerificationIssue{
		Severity: "warning",
		Code:     "expected_symlink",
		Path:     rel,
		Message:  fmt.Sprintf("%s is a rendered directory, expected a symlink to canonical", rel),
		Detail:   fmt.Sprintf("re-run `hero install project %s --target %s --force` to migrate to symlink layout, or `hero install project %s --migrate` to migrate all detected targets.", projectRoot, target, projectRoot),
	})

	// If canonical doesn't exist, we can't compare content.
	if _, err := os.Stat(canonical); err != nil {
		return "rendered", issues
	}

	// Compare per-file content between harnessDir and canonical.
	driftIssues := compareRenderedAgainstCanonical(projectRoot, harnessDir, canonical, kind)
	issues = append(issues, driftIssues...)

	return "rendered", issues
}

// compareRenderedAgainstCanonical hashes each file in harnessDir and
// compares to the same-named file under canonical. Reports drift per
// file with a short content-hash diff in the detail.
func compareRenderedAgainstCanonical(projectRoot, harnessDir, canonical, kind string) []VerificationIssue {
	var issues []VerificationIssue

	walk := func(dir string) (map[string]string, error) {
		hashes := map[string]string{}
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(dir, path)
			sum := sha256.Sum256(data)
			hashes[rel] = hex.EncodeToString(sum[:])[:8]
			return nil
		})
		return hashes, err
	}

	harnessHashes, _ := walk(harnessDir)
	canonicalHashes, _ := walk(canonical)

	for relName, harnessHash := range harnessHashes {
		canonicalHash, exists := canonicalHashes[relName]
		if !exists {
			issues = append(issues, VerificationIssue{
				Severity: "warning",
				Code:     "drifted_rendered",
				Path:     filepath.Join(relPath(projectRoot, harnessDir), relName),
				Message:  fmt.Sprintf("rendered file has no counterpart in canonical %s", canonical),
				Detail:   fmt.Sprintf("hash: %s", harnessHash),
			})
			continue
		}
		if harnessHash != canonicalHash {
			issues = append(issues, VerificationIssue{
				Severity: "warning",
				Code:     "drifted_rendered",
				Path:     filepath.Join(relPath(projectRoot, harnessDir), relName),
				Message:  "rendered file differs from canonical",
				Detail:   fmt.Sprintf("rendered: %s  canonical: %s", harnessHash, canonicalHash),
			})
		}
	}

	return issues
}

// relPath returns path relative to base if base contains path,
// otherwise the absolute path. Used for human-readable issue paths.
func relPath(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return rel
}
