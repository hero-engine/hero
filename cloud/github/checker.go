package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// GovernanceMode controls how strictly the check enforces spec linkage.
type GovernanceMode string

const (
	ModeAdvisory    GovernanceMode = "advisory"    // post comment, pass check
	ModeEnforcement GovernanceMode = "enforcement" // fail check if no spec
	ModeDisabled    GovernanceMode = "disabled"    // do nothing
)

// CheckResult is the outcome of checking a PR for spec linkage.
type CheckResult struct {
	SpecSlugs          []string          // spec slugs found in PR
	HasSpec            bool              // whether any spec was linked
	BranchSpec         string            // spec slug inferred from branch name (hero/deliver/*)
	BodySpecs          []string          // spec slugs found in PR body
	TitleSpecs         []string          // spec slugs found in PR title
	Summary            string            // human-readable summary
	Conclusion         string            // "success", "failure", "neutral"
	ComplianceMatches  []ConventionMatch // conventions triggered by changed files
	ScopeDrift         *ScopeDriftResult // scope drift detection result (nil if not checked)
}

// specRefPattern matches Hero spec references in PR text.
// Matches: spec:my-feature, hero:my-feature, [hero:my-feature]
var specRefPattern = regexp.MustCompile(`(?:spec|hero)[:\s]+([a-zA-Z0-9_-]+(?:/[a-zA-Z0-9_-]+)?)`)

// branchSpecPattern matches branch names created by hero deliver --async.
var branchSpecPattern = regexp.MustCompile(`^hero/deliver/(.+)$`)

// CheckPR analyzes a pull request for spec linkage.
func CheckPR(pr *PullRequestPayload, mode GovernanceMode) *CheckResult {
	result := &CheckResult{}

	// 1. Check branch name (hero/deliver/<slug>)
	if m := branchSpecPattern.FindStringSubmatch(pr.Head.Ref); len(m) == 2 {
		result.BranchSpec = m[1]
		result.SpecSlugs = append(result.SpecSlugs, m[1])
	}

	// 2. Check PR title for spec references
	for _, m := range specRefPattern.FindAllStringSubmatch(pr.Title, -1) {
		if len(m) == 2 {
			result.TitleSpecs = append(result.TitleSpecs, m[1])
			result.SpecSlugs = appendUnique(result.SpecSlugs, m[1])
		}
	}

	// 3. Check PR body for spec references
	if pr.Body != "" {
		for _, m := range specRefPattern.FindAllStringSubmatch(pr.Body, -1) {
			if len(m) == 2 {
				result.BodySpecs = append(result.BodySpecs, m[1])
				result.SpecSlugs = appendUnique(result.SpecSlugs, m[1])
			}
		}
	}

	result.HasSpec = len(result.SpecSlugs) > 0

	// Build summary and conclusion
	if result.HasSpec {
		slugList := strings.Join(result.SpecSlugs, ", ")
		result.Summary = fmt.Sprintf("PR linked to spec(s): %s", slugList)
		result.Conclusion = "success"
	} else {
		switch mode {
		case ModeEnforcement:
			result.Summary = "No Hero spec linked to this PR. All changes must trace to an approved spec.\n\n" +
				"Link a spec by:\n" +
				"- Using `hero deliver --async <slug>` (creates branch automatically)\n" +
				"- Adding `spec:<slug>` to the PR title or body\n" +
				"- Creating the PR from a `hero/deliver/<slug>` branch"
			result.Conclusion = "failure"
		default:
			result.Summary = "No Hero spec linked to this PR.\n\n" +
				"Consider linking a spec for traceability: add `spec:<slug>` to the PR body."
			result.Conclusion = "neutral"
		}
	}

	return result
}

// CreateCheckRun creates a GitHub check run on a PR's head commit.
func CreateCheckRun(app *App, installationID int64, repo string, headSHA string, result *CheckResult) error {
	token, err := app.InstallationToken(installationID)
	if err != nil {
		return fmt.Errorf("getting installation token: %w", err)
	}

	title := "Hero: Spec Linkage"
	if result.HasSpec {
		title = fmt.Sprintf("Hero: Linked to %s", strings.Join(result.SpecSlugs, ", "))
	}

	body := map[string]interface{}{
		"name":        "hero/spec-check",
		"head_sha":    headSHA,
		"status":      "completed",
		"conclusion":  result.Conclusion,
		"completed_at": time.Now().UTC().Format(time.RFC3339),
		"output": map[string]string{
			"title":   title,
			"summary": result.Summary,
		},
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/check-runs", repo)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("creating check run: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// PostComment posts an advisory comment on a PR.
func PostComment(app *App, installationID int64, repo string, prNumber int, body string) error {
	token, err := app.InstallationToken(installationID)
	if err != nil {
		return fmt.Errorf("getting installation token: %w", err)
	}

	payload, _ := json.Marshal(map[string]string{"body": body})

	url := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/comments", repo, prNumber)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("posting comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

// FormatComplianceSummary appends compliance information to the check result summary.
func FormatComplianceSummary(result *CheckResult) {
	var parts []string

	if len(result.ComplianceMatches) > 0 {
		parts = append(parts, "\n### Applicable Conventions\n")
		for _, m := range result.ComplianceMatches {
			files := strings.Join(m.MatchedFiles, ", ")
			parts = append(parts, fmt.Sprintf("- **%s** (`%s`): %d file(s) — %s",
				m.Convention.Title, m.Convention.Slug, len(m.MatchedFiles), files))
		}
	}

	if result.ScopeDrift != nil && result.ScopeDrift.HasDrift {
		parts = append(parts, "\n### Scope Drift Warning\n")
		parts = append(parts, fmt.Sprintf("Spec `%s` declares scope: %s",
			result.ScopeDrift.SpecSlug, strings.Join(result.ScopeDrift.SpecScope, ", ")))
		parts = append(parts, fmt.Sprintf("\nFiles outside declared scope (%d):", len(result.ScopeDrift.DriftFiles)))
		for _, f := range result.ScopeDrift.DriftFiles {
			parts = append(parts, fmt.Sprintf("- `%s`", f))
		}
	}

	if len(parts) > 0 {
		result.Summary += "\n" + strings.Join(parts, "\n")
	}
}
