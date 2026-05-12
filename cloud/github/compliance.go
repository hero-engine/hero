package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

// PRFile represents a file changed in a pull request.
type PRFile struct {
	Filename string `json:"filename"`
	Status   string `json:"status"` // added, removed, modified, renamed
	Patch    string `json:"patch"`
}

// GetPRFiles fetches the list of changed files for a PR.
func GetPRFiles(app *App, installationID int64, repo string, prNumber int) ([]PRFile, error) {
	token, err := app.InstallationToken(installationID)
	if err != nil {
		return nil, fmt.Errorf("getting installation token: %w", err)
	}

	var allFiles []PRFile
	page := 1

	for {
		url := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d/files?per_page=100&page=%d", repo, prNumber, page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching PR files: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("github returned %d: %s", resp.StatusCode, string(body))
		}

		var files []PRFile
		if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
			return nil, fmt.Errorf("decoding files: %w", err)
		}

		allFiles = append(allFiles, files...)
		if len(files) < 100 {
			break
		}
		page++
	}

	return allFiles, nil
}

// Convention represents a convention with its scope globs.
type Convention struct {
	Slug   string   `json:"slug"`
	Title  string   `json:"title"`
	Scope  []string `json:"scope"`  // glob patterns (e.g. "src/api/**")
	Status string   `json:"status"` // draft, active
}

// ConventionMatch represents a convention that applies to changed files.
type ConventionMatch struct {
	Convention   Convention `json:"convention"`
	MatchedFiles []string   `json:"matched_files"` // which PR files triggered the match
}

// MatchConventions finds which conventions apply to the changed files.
func MatchConventions(conventions []Convention, changedFiles []string) []ConventionMatch {
	var matches []ConventionMatch

	for _, conv := range conventions {
		if conv.Status != "active" {
			continue
		}

		var matched []string
		for _, file := range changedFiles {
			for _, pattern := range conv.Scope {
				if matchGlob(pattern, file) {
					matched = append(matched, file)
					break
				}
			}
		}

		if len(matched) > 0 {
			matches = append(matches, ConventionMatch{
				Convention:   conv,
				MatchedFiles: matched,
			})
		}
	}

	return matches
}

// ScopeDriftResult reports files changed outside the spec's declared scope.
type ScopeDriftResult struct {
	SpecSlug     string   `json:"spec_slug"`
	SpecScope    []string `json:"spec_scope"`    // declared file scope from spec
	InScopeFiles []string `json:"in_scope_files"`
	DriftFiles   []string `json:"drift_files"`   // files outside declared scope
	HasDrift     bool     `json:"has_drift"`
}

// DetectScopeDrift checks if PR files extend beyond a spec's declared scope.
func DetectScopeDrift(specScope []string, changedFiles []string) *ScopeDriftResult {
	result := &ScopeDriftResult{
		SpecScope: specScope,
	}

	if len(specScope) == 0 {
		// No scope declared — can't detect drift
		result.InScopeFiles = changedFiles
		return result
	}

	for _, file := range changedFiles {
		inScope := false
		for _, pattern := range specScope {
			if matchGlob(pattern, file) {
				inScope = true
				break
			}
		}
		if inScope {
			result.InScopeFiles = append(result.InScopeFiles, file)
		} else {
			result.DriftFiles = append(result.DriftFiles, file)
		}
	}

	result.HasDrift = len(result.DriftFiles) > 0
	return result
}

// matchGlob matches a file path against a glob pattern.
// Supports: * (single segment), ** (any depth), ? (single char).
func matchGlob(pattern, path string) bool {
	// Normalize separators
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)

	// Handle ** patterns by expanding to regex-like matching
	if strings.Contains(pattern, "**") {
		return matchDoublestar(pattern, path)
	}

	// Use filepath.Match for simple patterns
	matched, _ := filepath.Match(pattern, path)
	if matched {
		return true
	}

	// Also try matching against just the filename for patterns without /
	if !strings.Contains(pattern, "/") {
		matched, _ = filepath.Match(pattern, filepath.Base(path))
		return matched
	}

	return false
}

// matchDoublestar handles ** glob patterns.
func matchDoublestar(pattern, path string) bool {
	patParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	return matchParts(patParts, pathParts)
}

func matchParts(patParts, pathParts []string) bool {
	if len(patParts) == 0 {
		return len(pathParts) == 0
	}

	if patParts[0] == "**" {
		// ** matches zero or more path segments
		rest := patParts[1:]
		for i := 0; i <= len(pathParts); i++ {
			if matchParts(rest, pathParts[i:]) {
				return true
			}
		}
		return false
	}

	if len(pathParts) == 0 {
		return false
	}

	matched, _ := filepath.Match(patParts[0], pathParts[0])
	if !matched {
		return false
	}

	return matchParts(patParts[1:], pathParts[1:])
}
