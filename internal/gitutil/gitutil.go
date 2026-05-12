// Package gitutil provides Git helper functions for status reconciliation.
// All functions shell out to git and gracefully return empty results if git
// is unavailable or the directory is not a git repository.
package gitutil

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// IsRepo returns true if dir is inside a git working tree.
func IsRepo(dir string) bool {
	cmd := git(dir, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// DefaultBranch returns the name of the default branch (main, master, etc.).
// Falls back to "main" if detection fails.
func DefaultBranch(dir string) string {
	// Try symbolic-ref of origin/HEAD first
	cmd := git(dir, "symbolic-ref", "refs/remotes/origin/HEAD")
	out, err := cmd.Output()
	if err == nil {
		ref := strings.TrimSpace(string(out))
		// refs/remotes/origin/main → main
		if parts := strings.Split(ref, "/"); len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	// Fall back: check if main or master branch exists
	for _, branch := range []string{"main", "master"} {
		cmd = git(dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
		if err := cmd.Run(); err == nil {
			return branch
		}
	}

	return "main"
}

// CurrentBranch returns the current branch name, or "" if detached/unavailable.
func CurrentBranch(dir string) string {
	cmd := git(dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		return "" // detached
	}
	return branch
}

// FilesChangedOnBranch returns file paths that have commits on the current
// branch but not on the base branch. This shows what work has been done on
// a feature branch.
func FilesChangedOnBranch(dir, base string) []string {
	// Find the merge-base (where the branch diverged)
	mbCmd := git(dir, "merge-base", base, "HEAD")
	mbOut, err := mbCmd.Output()
	if err != nil {
		return nil
	}
	mergeBase := strings.TrimSpace(string(mbOut))
	if mergeBase == "" {
		return nil
	}

	// Get files changed since the merge-base
	cmd := git(dir, "diff", "--name-only", mergeBase+"..HEAD")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	return splitLines(string(out))
}

// FilesChangedUncommitted returns files with staged, unstaged, or untracked changes.
func FilesChangedUncommitted(dir string) []string {
	seen := make(map[string]bool)
	var result []string

	add := func(paths []string) {
		for _, p := range paths {
			if !seen[p] {
				seen[p] = true
				result = append(result, p)
			}
		}
	}

	// Unstaged changes
	cmd := git(dir, "diff", "--name-only")
	if out, err := cmd.Output(); err == nil {
		add(splitLines(string(out)))
	}

	// Staged changes
	cmd = git(dir, "diff", "--name-only", "--cached")
	if out, err := cmd.Output(); err == nil {
		add(splitLines(string(out)))
	}

	// Untracked files
	cmd = git(dir, "ls-files", "--others", "--exclude-standard")
	if out, err := cmd.Output(); err == nil {
		add(splitLines(string(out)))
	}

	return result
}

// AllChangedFiles returns the union of branch changes and uncommitted changes.
// This gives the complete picture of what files have been worked on.
func AllChangedFiles(dir string) []string {
	seen := make(map[string]bool)
	var result []string

	add := func(paths []string) {
		for _, p := range paths {
			if !seen[p] {
				seen[p] = true
				result = append(result, p)
			}
		}
	}

	defaultBranch := DefaultBranch(dir)
	currentBranch := CurrentBranch(dir)

	// If on a feature branch, get files changed on the branch
	if currentBranch != "" && currentBranch != defaultBranch {
		add(FilesChangedOnBranch(dir, defaultBranch))
	}

	// Also include uncommitted changes (works regardless of branch)
	add(FilesChangedUncommitted(dir))

	return result
}

// NormalizeFilePath converts a spec FilesTouched path to a form that can be
// compared against git output. Git returns paths relative to the repo root.
// Spec files may have leading ./ or be relative to the project root.
func NormalizeFilePath(projectRoot, path string) string {
	// If absolute, make relative to project root
	if filepath.IsAbs(path) {
		if rel, err := filepath.Rel(projectRoot, path); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	// Strip leading ./
	path = strings.TrimPrefix(path, "./")
	return filepath.ToSlash(path)
}

// RepoKey returns a stable identifier for the repository rooted at dir.
// It derives "owner/repo" from the git remote origin URL so that two
// developers cloning the same repo to different directory names share the
// same key. Falls back to filepath.Base(dir) if no remote is configured or
// the directory is not a git repo.
func RepoKey(dir string) string {
	cmd := git(dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return filepath.Base(dir)
	}
	return normalizeRemoteURL(strings.TrimSpace(string(out)), dir)
}

// normalizeRemoteURL converts a git remote URL to an "owner/repo" slug.
// Handles SSH (git@github.com:owner/repo.git), HTTPS, and local path forms.
func normalizeRemoteURL(url, fallbackDir string) string {
	url = strings.TrimSuffix(url, ".git")

	// Local path: starts with / . or ~ — use the base directory name.
	if strings.HasPrefix(url, "/") || strings.HasPrefix(url, ".") || strings.HasPrefix(url, "~") {
		return filepath.Base(url)
	}

	// SSH: git@github.com:owner/repo
	if idx := strings.Index(url, ":"); idx != -1 && !strings.HasPrefix(url, "http") {
		url = url[idx+1:]
	} else {
		// HTTPS: https://github.com/owner/repo — strip scheme + host
		stripped := strings.TrimPrefix(url, "https://")
		stripped = strings.TrimPrefix(stripped, "http://")
		if slash := strings.Index(stripped, "/"); slash != -1 {
			url = stripped[slash+1:]
		}
	}

	if url == "" {
		return filepath.Base(fallbackDir)
	}
	return url
}

// git creates an exec.Cmd for a git subcommand in the given directory.
func git(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd
}

// splitLines splits output into non-empty lines.
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
