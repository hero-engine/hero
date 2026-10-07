package spec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// AuditResult holds the parsed delivery audit report metadata.
type AuditResult struct {
	Found   bool
	Path    string
	Verdict string // "SHIP" or "HOLD"
	Surface string // "clean" or "noteworthy"

	// ReportSlug is the spec slug named in the report's own title line
	// ("# Delivery audit — {slug}"), when present.
	ReportSlug string
	// SlugMismatch is true when a report was found on disk at the
	// expected location, but its title line names a different spec than
	// the one being checked. This happens when a spec doesn't own its
	// directory (a flat-named spec sharing a folder with initiative
	// siblings — see isDiscoverableFlatSpec) and a sibling's leftover
	// report is sitting under the generic delivery-audit.md filename.
	// A mismatched report never satisfies the gate: Found stays false.
	SlugMismatch bool
	// Stale is true when the report predates the spec's own most recent
	// modification — i.e. the spec was touched again after the audit was
	// written, so the report cannot speak to the current delivery. A
	// stale report never satisfies the gate: Found stays false.
	Stale bool
}

// FindAuditReport looks for the delivery audit report belonging to spec s
// and parses the verdict and surface from its header.
//
// A spec that owns its directory (spec.md or requirements.md at the root
// of its own folder) is checked at the conventional delivery-audit.md
// path. A flat-named spec that shares its directory with siblings (an
// initiative's children stored as sibling <slug>.md files) is checked
// first at a slug-scoped <slug>-delivery-audit.md path, falling back to
// the generic filename for older reports.
//
// Either way, a report found on disk only satisfies the gate when its own
// title line names this spec's slug and its mtime is not older than the
// spec file's own mtime — both defend against a directory-sharing
// collision silently reusing a sibling's (or an earlier delivery's)
// leftover report.
func FindAuditReport(s *Spec) AuditResult {
	if s == nil || s.Path == "" {
		return AuditResult{}
	}

	specDir := filepath.Dir(s.Path)

	if !ownsSpecDir(s.Path) && s.Slug != "" {
		slugPath := filepath.Join(specDir, s.Slug+"-delivery-audit.md")
		if result, exists := loadAuditReport(slugPath, s.Slug, auditCutoff(s), s.Path); exists {
			return result
		}
	}

	auditPath := filepath.Join(specDir, "delivery-audit.md")
	if result, exists := loadAuditReport(auditPath, s.Slug, auditCutoff(s), s.Path); exists {
		return result
	}

	return AuditResult{}
}

// ownsSpecDir reports whether the spec file at path is the sole occupant
// of its directory (the canonical spec.md / requirements.md layout), as
// opposed to a flat <slug>.md file sharing its directory with an
// initiative's spec.md and other children. Mirrors the basename check in
// slugFromPath / isDiscoverableFlatSpec.
func ownsSpecDir(path string) bool {
	base := filepath.Base(path)
	return base == "spec.md" || base == "requirements.md"
}

// loadAuditReport reads and validates the audit report at path against
// the spec it's meant to satisfy. The second return value is false only
// when no file exists at path — a stale or slug-mismatched report is
// still returned (exists=true) so the caller can surface why the gate
// isn't satisfied instead of silently trying another location.
func loadAuditReport(path, expectedSlug string, specModTime time.Time, specPath string) (AuditResult, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return AuditResult{}, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return AuditResult{}, false
	}

	result := parseAuditHeader(string(data))
	result.Path = path
	result.ReportSlug = parseAuditSlug(string(data))

	if expectedSlug != "" && result.ReportSlug != "" && result.ReportSlug != expectedSlug {
		result.SlugMismatch = true
		return result, true
	}

	if !specModTime.IsZero() && info.ModTime().Before(specModTime) && !committedAuditIsCurrent(specPath, path) {
		result.Stale = true
		return result, true
	}

	result.Found = true
	return result, true
}

// FindAuditReportInDir looks for delivery-audit.md in the given directory.
// Unlike FindAuditReport, this has no spec identity to validate against —
// callers that know which spec they expect should use FindAuditReport.
func FindAuditReportInDir(dir string) AuditResult {
	auditPath := filepath.Join(dir, "delivery-audit.md")

	data, err := os.ReadFile(auditPath)
	if err != nil {
		return AuditResult{}
	}

	result := parseAuditHeader(string(data))
	result.Found = true
	result.Path = auditPath
	result.ReportSlug = parseAuditSlug(string(data))
	return result
}

// parseAuditHeader extracts Verdict and Surface from the audit report header.
// Matches patterns like:
//
//	**Verdict:** SHIP
//	**Surface:** clean
func parseAuditHeader(content string) AuditResult {
	var result AuditResult

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		// Match **Verdict:** VALUE or Verdict: VALUE
		if v, ok := extractHeaderValue(trimmed, "verdict"); ok {
			result.Verdict = strings.ToUpper(v)
		}

		// Match **Surface:** VALUE or Surface: VALUE
		if v, ok := extractHeaderValue(trimmed, "surface"); ok {
			result.Surface = strings.ToLower(v)
		}

		// Stop scanning after both are found or after too many lines
		if result.Verdict != "" && result.Surface != "" {
			break
		}
	}

	return result
}

// parseAuditSlug extracts the spec slug named in the report's title line,
// e.g. "# Delivery audit — some-slug" -> "some-slug". Returns "" when no
// title line in the expected format is found (older or hand-written
// reports), in which case identity is not checked — FindAuditReport
// treats an empty ReportSlug as "unknown, not a mismatch" rather than
// rejecting reports that predate this convention.
func parseAuditSlug(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.Contains(strings.ToLower(trimmed), "delivery audit") {
			continue
		}
		// Canonical separator is an em dash: "# Delivery audit — {slug}".
		if idx := strings.Index(trimmed, "—"); idx != -1 {
			return strings.TrimSpace(trimmed[idx+len("—"):])
		}
		// Fallback for reports authored with a plain hyphen separator.
		if idx := strings.Index(trimmed, " - "); idx != -1 {
			return strings.TrimSpace(trimmed[idx+len(" - "):])
		}
		return ""
	}
	return ""
}

// extractHeaderValue matches lines like "**Key:** value" or "Key: value"
// and returns the value for the given key (case-insensitive match).
func extractHeaderValue(line, key string) (string, bool) {
	lower := strings.ToLower(line)
	keyLower := strings.ToLower(key)

	// Try "**Key:** value" pattern
	patterns := []string{
		"**" + keyLower + ":** ",
		"**" + keyLower + ":**",
		keyLower + ": ",
		keyLower + ":",
	}

	for _, pattern := range patterns {
		idx := strings.Index(lower, pattern)
		if idx == -1 {
			continue
		}
		// Extract value after the pattern
		after := strings.TrimSpace(line[idx+len(pattern):])
		if after == "" {
			continue
		}
		// Clean up: remove trailing bold markers, backticks
		after = strings.TrimRight(after, "* `")
		after = strings.TrimSpace(after)
		if after != "" {
			return after, true
		}
	}

	return "", false
}

// auditCutoff is the time an audit report must not predate. An archived
// spec (under .hero/specs/) has none: `hero spec verify` rewrites and moves
// the spec after its audit, so an archived spec is always newer than its
// report. A spec merely marked completed while still in planning/ keeps the
// check — that is exactly the hand-flipped status this gate must catch.
func auditCutoff(s *Spec) time.Time {
	if s.IsFinished() && isArchivedPath(s.Path) {
		return time.Time{}
	}
	return s.ModifiedAt
}

// isArchivedPath reports whether a spec file lives in the workspace's
// archive (<hero folder>/specs/...), whatever the hero folder is named. The
// hero folder is the nearest ancestor that is a workspace (holds hero.json
// or is named .hero) and whose child on the path is planning/ or specs/, so
// a repo beneath a "planning" or "specs" directory, a spec folder with
// either name, a hero.json further up, or a stray hero.json inside a spec
// folder cannot misplace it.
func isArchivedPath(path string) bool {
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		if rel, err := filepath.Rel(dir, path); err == nil {
			first, _, _ := strings.Cut(filepath.ToSlash(rel), "/")
			if first == "specs" || first == "planning" {
				_, err := os.Stat(filepath.Join(dir, "hero.json"))
				if err == nil || filepath.Base(dir) == ".hero" {
					return first == "specs"
				}
			}
		}
		if filepath.Dir(dir) == dir {
			return false
		}
	}
}

// committedAuditIsCurrent settles an mtime-based "stale" verdict with git.
// File mtimes are rewritten in arbitrary order by `git checkout` and fresh
// clones, so when both files are committed and unmodified, their last
// commit times decide: the audit is current unless the spec was committed
// after it. A spec and audit last changed in the same commit are accepted
// as one reviewed unit (deliveries commit them together, which is the
// common clone case this exists for). Uncommitted or untracked files keep
// the mtime verdict.
func committedAuditIsCurrent(specPath, auditPath string) bool {
	if specPath == "" {
		return false
	}
	dir := filepath.Dir(specPath)
	status, err := exec.Command("git", "-C", dir, "status", "--porcelain", "--", specPath, auditPath).Output()
	if err != nil || strings.TrimSpace(string(status)) != "" {
		return false
	}
	specAt, ok1 := lastCommitTime(dir, specPath)
	auditAt, ok2 := lastCommitTime(dir, auditPath)
	return ok1 && ok2 && auditAt >= specAt
}

func lastCommitTime(dir, path string) (int64, bool) {
	out, err := exec.Command("git", "-C", dir, "log", "-1", "--format=%ct", "--", path).Output()
	if err != nil {
		return 0, false
	}
	t, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	return t, err == nil
}
