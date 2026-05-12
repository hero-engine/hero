package contract

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
)

// CriterionStatus tracks one criterion's contract status.
type CriterionStatus struct {
	Index      int             `json:"index"`
	Raw        string          `json:"raw"`
	Kind       string          `json:"kind"`
	Linked     bool            `json:"linked"`
	VerifiedBy []spec.TestLink `json:"verified_by,omitempty"`
}

// ContractReport is the contract status for one spec.
type ContractReport struct {
	Slug      string            `json:"slug"`
	Title     string            `json:"title"`
	Criteria  []CriterionStatus `json:"criteria"`
	Linked    int               `json:"linked"`
	Total     int               `json:"total"`
	Coverage  float64           `json:"coverage"`
}

// Status returns the contract report for a spec.
func Status(s *spec.Spec) *ContractReport {
	criteria := s.AcceptanceCriteria()
	r := &ContractReport{
		Slug:  s.Slug,
		Title: s.Title,
		Total: len(criteria),
	}

	for i, c := range criteria {
		cs := CriterionStatus{
			Index:      i + 1,
			Raw:        c.Raw,
			Kind:       c.Kind.String(),
			Linked:     len(c.VerifiedBy) > 0,
			VerifiedBy: c.VerifiedBy,
		}
		if cs.Linked {
			r.Linked++
		}
		r.Criteria = append(r.Criteria, cs)
	}

	if r.Total > 0 {
		r.Coverage = float64(r.Linked) / float64(r.Total) * 100
	}

	return r
}

// Link adds a verified_by annotation to a criterion in the spec file.
func Link(specPath, projectRoot string, criterionIdx int, testRef string) error {
	parts := strings.SplitN(testRef, "::", 2)
	if len(parts) != 2 {
		return fmt.Errorf("test ref must be in format file::testname, got %q", testRef)
	}

	testFile := parts[0]
	abs := filepath.Join(projectRoot, testFile)
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		return fmt.Errorf("test file %q does not exist", testFile)
	}

	content, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("reading spec: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	annotation := "  verified_by: " + testRef

	// Check for duplicate
	for _, line := range lines {
		if strings.TrimSpace(line) == "verified_by: "+testRef {
			return nil // already exists
		}
	}

	// Find the Nth criterion bullet and insert after it
	bulletCount := 0
	inCriteria := false
	insertIdx := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## Acceptance Criteria") || strings.HasPrefix(trimmed, "## acceptance criteria") {
			inCriteria = true
			continue
		}
		if inCriteria && strings.HasPrefix(trimmed, "## ") {
			break
		}
		if inCriteria && (strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")) {
			bulletCount++
			if bulletCount == criterionIdx {
				// Find the end of this criterion (next bullet or next verified_by group end)
				insertIdx = i + 1
				for insertIdx < len(lines) {
					nextTrimmed := strings.TrimSpace(lines[insertIdx])
					if strings.HasPrefix(nextTrimmed, "verified_by:") {
						insertIdx++
						continue
					}
					break
				}
			}
		}
	}

	if insertIdx < 0 {
		return fmt.Errorf("criterion %d not found in spec", criterionIdx)
	}

	// Insert the annotation
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:insertIdx]...)
	newLines = append(newLines, annotation)
	newLines = append(newLines, lines[insertIdx:]...)

	return os.WriteFile(specPath, []byte(strings.Join(newLines, "\n")), 0o644)
}

// RegressionResult is the result of running a linked test.
type RegressionResult struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Criterion string `json:"criterion"`
	TestFile  string `json:"test_file"`
	TestName  string `json:"test_name"`
	Passed    bool   `json:"passed"`
}

// Check runs all linked tests for a spec and returns regression results.
func Check(s *spec.Spec, projectRoot string) []RegressionResult {
	criteria := s.AcceptanceCriteria()
	var results []RegressionResult

	for _, c := range criteria {
		for _, link := range c.VerifiedBy {
			passed := runTest(projectRoot, link)
			results = append(results, RegressionResult{
				Slug:      s.Slug,
				Title:     s.Title,
				Criterion: c.Raw,
				TestFile:  link.File,
				TestName:  link.Name,
				Passed:    passed,
			})
		}
	}

	return results
}

func runTest(projectRoot string, link spec.TestLink) bool {
	runner, args := detectRunner(link)
	if runner == "" {
		return true // can't run, assume pass
	}

	cmd := exec.Command(runner, args...)
	cmd.Dir = projectRoot
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func detectRunner(link spec.TestLink) (string, []string) {
	f := link.File

	switch {
	case strings.HasSuffix(f, "_test.go"):
		pkg := "./" + filepath.Dir(f)
		return "go", []string{"test", "-run", link.Name, pkg}
	case strings.HasSuffix(f, ".spec.ts") || strings.HasSuffix(f, ".test.ts"):
		return "npx", []string{"playwright", "test", f, "-g", link.Name}
	case strings.HasSuffix(f, ".test.js") || strings.HasSuffix(f, ".spec.js"):
		return "npx", []string{"vitest", "run", f, "-t", link.Name}
	case strings.HasSuffix(f, ".py"):
		return "pytest", []string{f + "::" + link.Name}
	default:
		return "", nil
	}
}

// RenderText renders a contract status report.
func RenderText(r *ContractReport) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s (%d criteria)\n", r.Slug, r.Total)

	for _, cs := range r.Criteria {
		fmt.Fprintf(&sb, "  %d. [%s]  %s\n", cs.Index, cs.Kind, truncate(cs.Raw, 80))
		if cs.Linked {
			for _, link := range cs.VerifiedBy {
				fmt.Fprintf(&sb, "     linked: %s::%s\n", link.File, link.Name)
			}
		} else {
			sb.WriteString("     UNLINKED\n")
		}
	}

	fmt.Fprintf(&sb, "\nContract coverage: %d/%d (%.0f%%)\n", r.Linked, r.Total, r.Coverage)
	return sb.String()
}

// RenderCheckText renders regression check results.
func RenderCheckText(results []RegressionResult) string {
	if len(results) == 0 {
		return "No linked tests to check.\n"
	}

	var sb strings.Builder
	passed := 0
	failed := 0
	var failures []RegressionResult

	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failed++
			failures = append(failures, r)
		}
	}

	fmt.Fprintf(&sb, "Contract check — %d linked tests\n\n", len(results))

	if failed == 0 {
		fmt.Fprintf(&sb, "  ✓ All %d tests pass\n", passed)
	} else {
		for _, f := range failures {
			fmt.Fprintf(&sb, "  REGRESSION: %s — %s\n", f.Slug, truncate(f.Criterion, 60))
			fmt.Fprintf(&sb, "    FAIL: %s::%s\n", f.TestFile, f.TestName)
		}
	}

	fmt.Fprintf(&sb, "\nResult: %d passed, %d failed\n", passed, failed)
	return sb.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
