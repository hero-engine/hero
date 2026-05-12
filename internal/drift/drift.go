package drift

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/spec"
)

// Severity levels for drift signals.
const (
	SeverityWarning   = "warning"
	SeverityViolation = "violation"
)

// Signal is a single drift observation.
type Signal struct {
	Kind     string `json:"kind"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Detail   string `json:"detail,omitempty"`
}

// CriterionStatus tracks whether a single acceptance criterion has related code.
type CriterionStatus struct {
	Raw       string `json:"raw"`
	Addressed bool   `json:"addressed"`
	Detail    string `json:"detail,omitempty"`
}

// ConventionWarning flags a convention that governs changed files.
type ConventionWarning struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

// Report is the drift analysis for a single spec.
type Report struct {
	Slug        string              `json:"slug"`
	Title       string              `json:"title"`
	Status      string              `json:"status"`
	ClaimedBy   string              `json:"claimed_by,omitempty"`
	Criteria    []CriterionStatus   `json:"criteria,omitempty"`
	Signals     []Signal            `json:"signals"`
	Conventions []ConventionWarning `json:"conventions,omitempty"`
	ExitCode    int                 `json:"exit_code"`
}

// Analyze runs drift detection for a single spec against the project root.
// sinceRef is an optional git ref; if empty, uses the spec's creation date context.
func Analyze(s *spec.Spec, projectRoot string, sinceRef string) *Report {
	return AnalyzeWithIndex(s, projectRoot, sinceRef, nil)
}

// AnalyzeWithIndex runs drift detection with optional convention checking via the index.
func AnalyzeWithIndex(s *spec.Spec, projectRoot string, sinceRef string, idx *index.DB) *Report {
	r := &Report{
		Slug:      s.Slug,
		Title:     s.Title,
		Status:    string(s.Status),
		ClaimedBy: s.ClaimedBy,
	}

	changedFiles := changedFileSet(projectRoot, sinceRef)

	checkMissingFiles(r, s, projectRoot)
	checkRenamedFiles(r, s, projectRoot)
	checkCriteria(r, s, projectRoot, changedFiles)
	checkBoundaries(r, s, projectRoot, changedFiles)
	checkConventions(r, s, idx)
	if idx != nil && len(r.Conventions) > 0 {
		heroDir := filepath.Dir(filepath.Dir(filepath.Dir(s.Path)))
		CheckArchitecturalDrift(r, projectRoot, heroDir, idx)
	}

	r.ExitCode = exitCode(r.Signals)
	return r
}

// AnalyzeAll runs drift on every delivering spec under heroDir.
func AnalyzeAll(heroDir, projectRoot, sinceRef string) ([]*Report, error) {
	return AnalyzeAllWithIndex(heroDir, projectRoot, sinceRef, nil)
}

// AnalyzeAllWithIndex runs drift on every delivering spec with convention checking.
func AnalyzeAllWithIndex(heroDir, projectRoot, sinceRef string, idx *index.DB) ([]*Report, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}

	var reports []*Report
	for _, s := range specs {
		if s.Status != spec.StatusDelivering {
			continue
		}
		reports = append(reports, AnalyzeWithIndex(s, projectRoot, sinceRef, idx))
	}
	return reports, nil
}

// AnalyzeInitiative runs drift on every child spec of the given initiative.
func AnalyzeInitiative(heroDir, projectRoot, initiative, sinceRef string) ([]*Report, error) {
	return AnalyzeInitiativeWithIndex(heroDir, projectRoot, initiative, sinceRef, nil)
}

// AnalyzeInitiativeWithIndex runs drift on initiative children with convention checking.
func AnalyzeInitiativeWithIndex(heroDir, projectRoot, initiative, sinceRef string, idx *index.DB) ([]*Report, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}

	children := make(map[string]bool)
	for _, s := range specs {
		for _, rel := range s.Relations {
			if rel.Kind == "parent" && rel.Target == initiative {
				children[s.Slug] = true
			}
		}
	}

	var reports []*Report
	for _, s := range specs {
		if !children[s.Slug] {
			continue
		}
		reports = append(reports, AnalyzeWithIndex(s, projectRoot, sinceRef, idx))
	}
	return reports, nil
}

func checkMissingFiles(r *Report, s *spec.Spec, projectRoot string) {
	for _, f := range s.FilesTouched {
		abs := filepath.Join(projectRoot, f)
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			r.Signals = append(r.Signals, Signal{
				Kind:     "missing_file",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("Spec lists `%s` in Changes but file does not exist", f),
			})
		}
	}
}

func checkRenamedFiles(r *Report, s *spec.Spec, projectRoot string) {
	for _, f := range s.FilesTouched {
		abs := filepath.Join(projectRoot, f)
		if _, err := os.Stat(abs); err == nil {
			continue
		}
		newName := detectRename(projectRoot, f)
		if newName != "" {
			r.Signals = append(r.Signals, Signal{
				Kind:     "renamed_file",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("Spec lists `%s` but it was renamed to `%s`", f, newName),
				Detail:   newName,
			})
		}
	}
}

func checkCriteria(r *Report, s *spec.Spec, projectRoot string, changedFiles map[string]bool) {
	criteria := s.AcceptanceCriteria()
	if len(criteria) == 0 {
		return
	}

	for _, c := range criteria {
		cs := CriterionStatus{Raw: c.Raw}

		var textToSearch string
		if c.Kind.IsEARS() {
			textToSearch = c.Behavior
		} else {
			textToSearch = c.Raw
		}

		keywords := extractKeywords(textToSearch)
		if len(keywords) == 0 {
			cs.Addressed = true
			cs.Detail = "no extractable keywords"
			r.Criteria = append(r.Criteria, cs)
			continue
		}

		matched := keywordsInFiles(projectRoot, keywords, changedFiles)
		if matched {
			cs.Addressed = true
		} else {
			cs.Detail = fmt.Sprintf("no occurrences of %s in changed files", strings.Join(keywords, ", "))
			r.Signals = append(r.Signals, Signal{
				Kind:     "unaddressed_criterion",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("Criterion looks unaddressed: %q", c.Raw),
				Detail:   cs.Detail,
			})
		}
		r.Criteria = append(r.Criteria, cs)
	}
}

func checkBoundaries(r *Report, s *spec.Spec, projectRoot string, changedFiles map[string]bool) {
	boundaries, ok := s.Sections["boundaries"]
	if !ok {
		return
	}

	negatives := extractNegativePaths(boundaries)
	for _, neg := range negatives {
		for changed := range changedFiles {
			if pathMatches(changed, neg) {
				r.Signals = append(r.Signals, Signal{
					Kind:     "boundary_violation",
					Severity: SeverityViolation,
					Message:  fmt.Sprintf("Boundary possibly crossed: spec says does not touch `%s`, but `%s` was modified", neg, changed),
				})
			}
		}
	}
}

func checkConventions(r *Report, s *spec.Spec, idx *index.DB) {
	if idx == nil || len(s.FilesTouched) == 0 {
		return
	}

	results, err := idx.FindConventionsForFiles(s.FilesTouched)
	if err != nil || len(results) == 0 {
		return
	}

	seen := make(map[string]bool)
	for _, conv := range results {
		if seen[conv.Slug] {
			continue
		}
		seen[conv.Slug] = true
		r.Conventions = append(r.Conventions, ConventionWarning{
			Slug:  conv.Slug,
			Title: conv.Title,
		})
	}
}

// CheckArchitecturalDrift loads convention specs and checks if changed files
// violate architectural rules (e.g., "must not import X", "never call Y directly").
// Reads convention content from disk and adds violation signals to the report.
func CheckArchitecturalDrift(r *Report, projectRoot string, heroDir string, idx *index.DB) {
	if idx == nil || len(r.Conventions) == 0 {
		return
	}

	convSpecs, err := spec.Discover(heroDir)
	if err != nil {
		return
	}

	for _, conv := range r.Conventions {
		for _, cs := range convSpecs {
			if cs.Slug != conv.Slug {
				continue
			}
			rules := extractArchRules(cs.RawContent)
			for _, rule := range rules {
				checkArchViolation(r, projectRoot, rule, conv.Slug)
			}
		}
	}
}

type archRule struct {
	Verb   string
	Object string
}

func extractArchRules(content string) []archRule {
	clean := strings.ReplaceAll(content, "**", "")
	var rules []archRule
	archRe := regexp.MustCompile(`(?i)(?:must\s+not|should\s+not|never)\s+(import|call|depend\s+on|access)\s+[` + "`" + `"]?([^` + "`" + `"\n]+)[` + "`" + `"]?`)
	for _, match := range archRe.FindAllStringSubmatch(clean, -1) {
		if len(match) > 2 {
			rules = append(rules, archRule{
				Verb:   strings.TrimSpace(match[1]),
				Object: strings.TrimSpace(match[2]),
			})
		}
	}
	return rules
}

func checkArchViolation(r *Report, projectRoot string, rule archRule, convSlug string) {
	target := strings.ToLower(rule.Object)
	// Check if any changed file in the project contains the forbidden pattern
	for changed := range changedFileSet(projectRoot, "") {
		abs := filepath.Join(projectRoot, changed)
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		content := strings.ToLower(string(data))
		if strings.Contains(content, target) {
			r.Signals = append(r.Signals, Signal{
				Kind:     "arch_violation",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("Convention %q: must not %s %q — found in %s", convSlug, rule.Verb, rule.Object, changed),
			})
			break
		}
	}
}

func changedFileSet(projectRoot, sinceRef string) map[string]bool {
	set := make(map[string]bool)

	if sinceRef != "" {
		cmd := exec.Command("git", "-C", projectRoot, "diff", "--name-only", sinceRef+"..HEAD")
		if out, err := cmd.Output(); err == nil {
			for _, line := range splitLines(string(out)) {
				set[line] = true
			}
		}
		return set
	}

	// Default: diff from merge-base with default branch
	mbCmd := exec.Command("git", "-C", projectRoot, "merge-base", "HEAD", "HEAD~20")
	mbOut, _ := mbCmd.Output()
	base := strings.TrimSpace(string(mbOut))

	if base == "" {
		// Fallback: all tracked files
		cmd := exec.Command("git", "-C", projectRoot, "ls-files")
		if out, err := cmd.Output(); err == nil {
			for _, line := range splitLines(string(out)) {
				set[line] = true
			}
		}
		return set
	}

	cmd := exec.Command("git", "-C", projectRoot, "diff", "--name-only", base+"..HEAD")
	if out, err := cmd.Output(); err == nil {
		for _, line := range splitLines(string(out)) {
			set[line] = true
		}
	}
	return set
}

func detectRename(projectRoot, oldPath string) string {
	cmd := exec.Command("git", "-C", projectRoot, "log", "--follow", "--diff-filter=R", "--pretty=format:", "--name-only", "-1", "--", oldPath)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := splitLines(string(out))
	if len(lines) > 0 && lines[0] != oldPath {
		return lines[0]
	}
	return ""
}

var identifierRe = regexp.MustCompile(`(?:--|/)?[a-zA-Z_][a-zA-Z0-9_./\-]*(?:\.[a-zA-Z]+)?`)

// ExtractKeywords extracts identifier-shaped tokens from text, filtered by stop words.
// Exported for reuse by the coverage package.
func ExtractKeywords(text string) []string {
	return extractKeywords(text)
}

func extractKeywords(text string) []string {
	var keywords []string
	seen := make(map[string]bool)

	for _, match := range identifierRe.FindAllString(text, -1) {
		lower := strings.ToLower(match)
		// Skip very short or common English words
		if len(match) < 3 || isStopWord(lower) {
			continue
		}
		if !seen[lower] {
			seen[lower] = true
			keywords = append(keywords, match)
		}
	}
	return keywords
}

func isStopWord(w string) bool {
	stops := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "that": true,
		"this": true, "from": true, "not": true, "are": true, "was": true,
		"were": true, "been": true, "have": true, "has": true, "had": true,
		"will": true, "shall": true, "should": true, "would": true, "could": true,
		"can": true, "may": true, "must": true, "all": true, "each": true,
		"any": true, "both": true, "into": true, "when": true, "where": true,
		"which": true, "while": true, "then": true, "than": true, "also": true,
		"only": true, "does": true, "system": true, "spec": true, "file": true,
	}
	return stops[w]
}

func keywordsInFiles(projectRoot string, keywords []string, changedFiles map[string]bool) bool {
	for _, kw := range keywords {
		found := false
		for f := range changedFiles {
			abs := filepath.Join(projectRoot, f)
			data, err := os.ReadFile(abs)
			if err != nil {
				continue
			}
			if strings.Contains(strings.ToLower(string(data)), strings.ToLower(kw)) {
				found = true
				break
			}
		}
		if found {
			return true
		}
	}
	return false
}

func extractNegativePaths(boundaries string) []string {
	// Strip markdown bold markers before matching
	clean := strings.ReplaceAll(boundaries, "**", "")
	var paths []string
	negRe := regexp.MustCompile("(?i)(?:does\\s+not|must\\s+not|never|shall\\s+not)\\s+(?:touch|modify|change|alter|edit|update)\\s+[`\"]?([^`\"\\n]+)[`\"]?")

	for _, match := range negRe.FindAllStringSubmatch(clean, -1) {
		if len(match) > 1 {
			p := strings.TrimSpace(match[1])
			p = strings.Trim(p, "`\"")
			if p != "" {
				paths = append(paths, p)
			}
		}
	}
	return paths
}

func pathMatches(changed, pattern string) bool {
	if strings.Contains(changed, pattern) {
		return true
	}
	matched, _ := filepath.Match(pattern, changed)
	return matched
}

// CheckCrossRepo adds drift signals for cross-repo dependencies.
// resolver may be nil (graceful skip).
func CheckCrossRepo(r *Report, s *spec.Spec, resolver *spec.CrossRepoResolver) {
	if resolver == nil {
		return
	}

	refs := spec.CrossRepoRelations(s)
	if len(refs) == 0 {
		return
	}

	for _, ref := range refs {
		upstream, err := resolver.Resolve(ref)
		if err != nil {
			r.Signals = append(r.Signals, Signal{
				Kind:     "cross_repo_missing",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("cross-repo dependency %s/%s could not be resolved", ref.Repo, ref.Slug),
				Detail:   err.Error(),
			})
			continue
		}

		// Check if upstream was modified more recently than local spec
		if !upstream.ModifiedAt.IsZero() && !s.ModifiedAt.IsZero() && upstream.ModifiedAt.After(s.ModifiedAt) {
			r.Signals = append(r.Signals, Signal{
				Kind:     "upstream_modified",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("%s/%s was modified after this spec — review upstream changes", ref.Repo, ref.Slug),
				Detail:   fmt.Sprintf("upstream: %s, local: %s", upstream.ModifiedAt.Format("2006-01-02"), s.ModifiedAt.Format("2006-01-02")),
			})
		}

		// Check if upstream status changed to something concerning
		switch upstream.Status {
		case spec.StatusCompleted:
			// Completed is fine — the dependency was shipped
		case "superseded":
			r.Signals = append(r.Signals, Signal{
				Kind:     "upstream_superseded",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("%s/%s was superseded — check if this spec needs updating", ref.Repo, ref.Slug),
			})
		}
	}
}

func exitCode(signals []Signal) int {
	code := 0
	for _, s := range signals {
		if s.Severity == SeverityViolation && code < 2 {
			code = 2
		} else if s.Severity == SeverityWarning && code < 1 {
			code = 1
		}
	}
	return code
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// RenderText renders a human-readable drift report.
func RenderText(reports []*Report) string {
	if len(reports) == 0 {
		return "No specs to analyze.\n"
	}

	var sb strings.Builder
	for i, r := range reports {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "%s (status: %s", r.Slug, r.Status)
		if r.ClaimedBy != "" {
			fmt.Fprintf(&sb, ", claimed by %s", r.ClaimedBy)
		}
		sb.WriteString(")\n")

		if len(r.Criteria) > 0 {
			addressed := 0
			for _, c := range r.Criteria {
				if c.Addressed {
					addressed++
				}
			}
			fmt.Fprintf(&sb, "  ✓ %d/%d acceptance criteria have related code changes\n", addressed, len(r.Criteria))

			for _, c := range r.Criteria {
				if !c.Addressed {
					fmt.Fprintf(&sb, "  ⚠ 1 criterion looks unaddressed:\n")
					fmt.Fprintf(&sb, "      %q\n", c.Raw)
					if c.Detail != "" {
						fmt.Fprintf(&sb, "      → %s\n", c.Detail)
					}
				}
			}
		}

		for _, s := range r.Signals {
			if s.Kind == "unaddressed_criterion" {
				continue // already rendered above
			}
			icon := "⚠"
			if s.Severity == SeverityViolation {
				icon = "✗"
			}
			fmt.Fprintf(&sb, "  %s %s\n", icon, s.Message)
		}

		if len(r.Conventions) > 0 {
			fmt.Fprintf(&sb, "  📋 %d convention(s) govern changed files:\n", len(r.Conventions))
			for _, c := range r.Conventions {
				fmt.Fprintf(&sb, "      %s — %s\n", c.Slug, c.Title)
			}
		}

		if len(r.Signals) == 0 {
			sb.WriteString("  ✓ No drift detected\n")
		}

		hasFileSignal := false
		for _, s := range r.Signals {
			if s.Kind == "missing_file" || s.Kind == "renamed_file" {
				hasFileSignal = true
				break
			}
		}
		if !hasFileSignal && len(r.Criteria) == 0 {
			sb.WriteString("  ✓ All listed files in ## Changes exist\n")
		}
	}
	return sb.String()
}

// RenderJSON renders all reports as JSON.
func RenderJSON(reports []*Report) (string, error) {
	data, err := json.MarshalIndent(reports, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// AggregateExitCode returns the highest exit code across all reports.
func AggregateExitCode(reports []*Report) int {
	code := 0
	for _, r := range reports {
		if r.ExitCode > code {
			code = r.ExitCode
		}
	}
	return code
}

// DriftSummaries returns a pulse-friendly list of specs that have drift warnings.
func DriftSummaries(heroDir, projectRoot string) []DriftSummary {
	reports, err := AnalyzeAll(heroDir, projectRoot, "")
	if err != nil {
		return nil
	}
	var summaries []DriftSummary
	for _, r := range reports {
		if len(r.Signals) > 0 {
			summaries = append(summaries, DriftSummary{
				Slug:     r.Slug,
				Title:    r.Title,
				Warnings: len(r.Signals),
				HasViolation: r.ExitCode == 2,
			})
		}
	}
	return summaries
}

// DriftSummary is a compact representation for pulse integration.
type DriftSummary struct {
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Warnings     int    `json:"warnings"`
	HasViolation bool   `json:"has_violation"`
}
