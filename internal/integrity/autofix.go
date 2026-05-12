package integrity

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// FixAction describes one proposed status frontmatter rewrite.
// Generated from a Report; applied with ApplyFix.
type FixAction struct {
	Slug       string
	Path       string
	OldStatus  spec.Status
	NewStatus  spec.Status
	Verdict    Verdict
	Reason     string // human-readable evidence ("3/7 ACs failing")
	Skipped    bool   // true when the verdict has no recommended downgrade
	SkipReason string // why we skipped (verified, unverifiable, etc.)
}

// PlanFixes turns a Report into the list of FixActions an --auto-fix
// run would apply. Unverifiable specs are skipped (no concrete
// evidence to act on); verified specs are skipped (nothing to fix).
// Lying and partial verdicts produce concrete proposals.
func PlanFixes(report *Report) []FixAction {
	if report == nil {
		return nil
	}
	out := make([]FixAction, 0, len(report.Findings))
	for _, f := range report.Findings {
		newStatus := SuggestStatus(f.Verdict)
		if newStatus == "" {
			out = append(out, FixAction{
				Slug:       f.Slug,
				Path:       f.Path,
				Verdict:    f.Verdict,
				Skipped:    true,
				SkipReason: skipReasonFor(f.Verdict),
			})
			continue
		}
		out = append(out, FixAction{
			Slug:      f.Slug,
			Path:      f.Path,
			OldStatus: spec.StatusCompleted,
			NewStatus: newStatus,
			Verdict:   f.Verdict,
			Reason:    formatReason(f),
		})
	}
	return out
}

func skipReasonFor(v Verdict) string {
	switch v {
	case VerdictVerified:
		return "all ACs passing — nothing to fix"
	case VerdictUnverifiable:
		return "no Criterion nodes — graph cannot judge either way"
	}
	return "no recommended downgrade"
}

func formatReason(f Finding) string {
	parts := []string{}
	if f.Failing > 0 {
		parts = append(parts, fmt.Sprintf("%d failing", f.Failing))
	}
	if f.Regressed > 0 {
		parts = append(parts, fmt.Sprintf("%d regressed", f.Regressed))
	}
	if f.ProposedOrOpen > 0 {
		parts = append(parts, fmt.Sprintf("%d open", f.ProposedOrOpen))
	}
	if f.Passing > 0 {
		parts = append(parts, fmt.Sprintf("%d passing", f.Passing))
	}
	return fmt.Sprintf("%s (of %d ACs)", strings.Join(parts, ", "), f.Total)
}

// ApplyFix rewrites the spec file's frontmatter status line in place.
// Adds an `auto_downgraded` annotation with timestamp + evidence so
// the rewrite is auditable. Idempotent — running on a file already at
// the target status is a no-op.
//
// The rewrite is line-level: we replace just the `status:` line and
// inject `auto_downgraded:` immediately after it. Other frontmatter
// fields (title, tags, relations, …) are preserved verbatim — no
// re-encoding pass — so the diff stays minimal and the file stays
// faithful to the human's formatting.
func ApplyFix(action FixAction) error {
	if action.Skipped {
		return nil
	}
	data, err := os.ReadFile(action.Path)
	if err != nil {
		return fmt.Errorf("read %s: %w", action.Path, err)
	}
	rewritten, changed, err := rewriteFrontmatterStatus(data, string(action.NewStatus), action.Reason)
	if err != nil {
		return fmt.Errorf("rewrite %s: %w", action.Path, err)
	}
	if !changed {
		return nil
	}
	if err := os.WriteFile(action.Path, rewritten, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", action.Path, err)
	}
	return nil
}

// rewriteFrontmatterStatus is the byte-level rewrite. Pure function
// so tests don't need filesystem.
//
// Returns (newContent, changed, err). `changed` is false when the
// status line is already at the target value (idempotent re-run).
func rewriteFrontmatterStatus(data []byte, newStatus, reason string) ([]byte, bool, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var (
		inFront     bool
		seenOpen    bool
		statusLine  int = -1
		closeIdx    int = -1
		statusValue string
	)

	// Split lines so we can inject after the status line.
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, false, err
	}

	// Find frontmatter range and existing status line.
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !seenOpen {
			if trimmed == "---" {
				seenOpen = true
				inFront = true
				continue
			}
			// File doesn't start with frontmatter — leave untouched.
			break
		}
		if inFront && trimmed == "---" {
			closeIdx = i
			inFront = false
			break
		}
		if inFront && strings.HasPrefix(trimmed, "status:") {
			statusLine = i
			statusValue = strings.TrimSpace(strings.TrimPrefix(trimmed, "status:"))
		}
	}

	if !seenOpen || closeIdx < 0 {
		// No frontmatter at all — nothing to rewrite.
		return data, false, nil
	}
	if statusLine < 0 {
		return nil, false, fmt.Errorf("no `status:` line in frontmatter")
	}
	if statusValue == newStatus {
		// Already at target — idempotent no-op.
		return data, false, nil
	}

	// Build replacement: status line at the new value, plus an
	// auto_downgraded annotation immediately after it. If a previous
	// auto_downgraded line already exists, replace it; otherwise
	// insert a new one.
	stamp := time.Now().UTC().Format("2006-01-02")
	annotation := fmt.Sprintf("auto_downgraded: \"%s by hero check status: %s\"", stamp, reason)

	newLines := make([]string, 0, len(lines)+1)
	for i, line := range lines {
		switch {
		case i == statusLine:
			newLines = append(newLines, fmt.Sprintf("status: %s", newStatus))
			// Look ahead — if next line is already auto_downgraded,
			// replace it; otherwise insert.
			if i+1 < closeIdx && strings.HasPrefix(strings.TrimSpace(lines[i+1]), "auto_downgraded:") {
				newLines = append(newLines, annotation)
				// Skip the original auto_downgraded line.
				lines[i+1] = ""
				continue
			}
			newLines = append(newLines, annotation)
		case line == "" && i == statusLine+1 && len(newLines) > 1 && strings.HasPrefix(newLines[len(newLines)-2], "auto_downgraded:"):
			// We replaced the original auto_downgraded line — skip it.
			continue
		default:
			newLines = append(newLines, line)
		}
	}

	// Preserve trailing newline if original had one.
	result := strings.Join(newLines, "\n")
	if bytes.HasSuffix(data, []byte("\n")) && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return []byte(result), true, nil
}
