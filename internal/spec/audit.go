package spec

import (
	"os"
	"path/filepath"
	"strings"
)

// AuditResult holds the parsed delivery audit report metadata.
type AuditResult struct {
	Found   bool
	Path    string
	Verdict string // "SHIP" or "HOLD"
	Surface string // "clean" or "noteworthy"
}

// FindAuditReport looks for a delivery-audit.md file in the spec's
// directory and parses the verdict and surface from the header.
// Checks both planning and specs directories.
func FindAuditReport(s *Spec) AuditResult {
	if s == nil || s.Path == "" {
		return AuditResult{}
	}

	specDir := filepath.Dir(s.Path)
	auditPath := filepath.Join(specDir, "delivery-audit.md")

	data, err := os.ReadFile(auditPath)
	if err != nil {
		return AuditResult{}
	}

	result := parseAuditHeader(string(data))
	result.Found = true
	result.Path = auditPath
	return result
}

// FindAuditReportInDir looks for delivery-audit.md in the given directory.
func FindAuditReportInDir(dir string) AuditResult {
	auditPath := filepath.Join(dir, "delivery-audit.md")

	data, err := os.ReadFile(auditPath)
	if err != nil {
		return AuditResult{}
	}

	result := parseAuditHeader(string(data))
	result.Found = true
	result.Path = auditPath
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
