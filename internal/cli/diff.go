package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var diffBase string

var diffCmd = &cobra.Command{
	Use:   "diff <spec-path>",
	Short: "Compare spec's planned changes vs actual git diff",
	Long: `Shows which files from the spec's Changes section have actually been
modified in git, and which git-changed files are not mentioned in the spec.

This helps verify that implementation matches the spec's plan.`,
	Args: cobra.ExactArgs(1),
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().StringVar(&diffBase, "base", "HEAD", "git base ref to diff against (default HEAD)")
}

func runDiff(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	specPath := args[0]

	s, err := spec.ParseFile(specPath)
	if err != nil {
		return fmt.Errorf("parsing spec %s: %w", specPath, err)
	}

	if len(s.FilesTouched) == 0 {
		fmt.Println("No files listed in spec's Changes section.")
		return nil
	}

	// Get git diff file list
	gitFiles, err := gitDiffFiles(projectRoot, diffBase)
	if err != nil {
		return fmt.Errorf("running git diff: %w", err)
	}

	// Build sets for comparison
	specFiles := make(map[string]bool)
	for _, f := range s.FilesTouched {
		specFiles[f] = true
	}

	gitFileSet := make(map[string]bool)
	for _, f := range gitFiles {
		gitFileSet[f] = true
	}

	// Categorize
	var matched, specOnly, gitOnly []string

	for _, f := range s.FilesTouched {
		if gitFileSet[f] {
			matched = append(matched, f)
		} else {
			specOnly = append(specOnly, f)
		}
	}

	for _, f := range gitFiles {
		if !specFiles[f] {
			gitOnly = append(gitOnly, f)
		}
	}

	sort.Strings(matched)
	sort.Strings(specOnly)
	sort.Strings(gitOnly)

	// Print results
	fmt.Printf("Spec: %s (%s)\n", s.Slug, s.Title)
	fmt.Printf("Base: %s\n\n", diffBase)

	if len(matched) > 0 {
		fmt.Printf("Matched (%d) — in spec and git diff:\n", len(matched))
		for _, f := range matched {
			fmt.Printf("  + %s\n", f)
		}
	}

	if len(specOnly) > 0 {
		fmt.Printf("\nSpec only (%d) — planned but not yet changed:\n", len(specOnly))
		for _, f := range specOnly {
			fmt.Printf("  - %s\n", f)
		}
	}

	if len(gitOnly) > 0 {
		fmt.Printf("\nGit only (%d) — changed but not in spec:\n", len(gitOnly))
		for _, f := range gitOnly {
			fmt.Printf("  ? %s\n", f)
		}
	}

	if len(specOnly) == 0 && len(gitOnly) == 0 {
		fmt.Println("\nAll changes align with spec.")
	}

	return nil
}

// gitDiffFiles returns the list of files changed relative to the base ref.
func gitDiffFiles(projectRoot, base string) ([]string, error) {
	// First try diff against base ref (for committed changes)
	cmd := exec.Command("git", "diff", "--name-only", base)
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		// If base ref doesn't exist, try just showing uncommitted changes
		cmd = exec.Command("git", "diff", "--name-only")
		cmd.Dir = projectRoot
		out, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git diff failed: %w", err)
		}
	}

	// Also include staged changes
	stagedCmd := exec.Command("git", "diff", "--name-only", "--cached")
	stagedCmd.Dir = projectRoot
	stagedOut, _ := stagedCmd.Output()

	// Also include untracked files
	untrackedCmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	untrackedCmd.Dir = projectRoot
	untrackedOut, _ := untrackedCmd.Output()

	// Merge all file lists
	seen := make(map[string]bool)
	var files []string

	for _, output := range [][]byte{out, stagedOut, untrackedOut} {
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !seen[line] {
				// Normalize to project-root-relative path
				absPath := filepath.Join(projectRoot, line)
				relPath, err := filepath.Rel(projectRoot, absPath)
				if err != nil {
					relPath = line
				}
				if !seen[relPath] {
					seen[relPath] = true
					files = append(files, relPath)
				}
			}
		}
	}

	return files, nil
}
