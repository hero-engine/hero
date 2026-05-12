package cli

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/spf13/cobra"
)

var suggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest specs for undocumented high-churn areas",
	Long: `Analyzes git churn to find files with heavy recent activity but no
spec coverage — surfaces areas where work is happening without
documentation, ranked by churn intensity.`,
	RunE: runSuggest,
}

var (
	suggestSince string
	suggestTop   int
)

func init() {
	suggestCmd.Flags().StringVar(&suggestSince, "since", "30d", "time window for churn analysis (e.g. 30d, 90d)")
	suggestCmd.Flags().IntVar(&suggestTop, "top", 10, "number of suggestions to show")
}

type churnEntry struct {
	Path    string
	Commits int
	Covered bool
	Specs   []string
}

func runSuggest(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Parse since duration
	sinceArg := suggestSince
	if len(sinceArg) >= 2 {
		numStr := sinceArg[:len(sinceArg)-1]
		unit := sinceArg[len(sinceArg)-1]
		var days int
		fmt.Sscanf(numStr, "%d", &days)
		switch unit {
		case 'd':
			// already days
		case 'w':
			days *= 7
		default:
			days = 30
		}
		sinceArg = fmt.Sprintf("%d days ago", days)
	}

	// Get file churn from git
	gitCmd := exec.Command("git", "-C", projectRoot, "log",
		"--since="+sinceArg,
		"--pretty=format:", "--name-only", "--no-merges")
	out, err := gitCmd.Output()
	if err != nil {
		return fmt.Errorf("reading git log: %w", err)
	}

	// Count commits per file
	counts := make(map[string]int)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ".hero/") || strings.HasPrefix(line, ".") {
			continue
		}
		counts[line]++
	}

	// Check spec coverage via index
	idx, err := index.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	var entries []churnEntry
	for path, count := range counts {
		if count < 3 {
			continue
		}
		e := churnEntry{Path: path, Commits: count}
		results, searchErr := idx.SearchByFile(path)
		if searchErr == nil && len(results) > 0 {
			e.Covered = true
			for _, r := range results {
				e.Specs = append(e.Specs, r.Slug)
			}
		}
		entries = append(entries, e)
	}

	// Sort by commits descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Commits > entries[j].Commits
	})

	// Filter to uncovered only
	var uncovered []churnEntry
	for _, e := range entries {
		if !e.Covered {
			uncovered = append(uncovered, e)
		}
	}

	if len(uncovered) == 0 {
		fmt.Println("No high-churn areas without spec coverage found.")
		return nil
	}

	limit := suggestTop
	if limit > len(uncovered) {
		limit = len(uncovered)
	}

	fmt.Printf("High-churn files with no spec coverage (top %d):\n\n", limit)
	for i, e := range uncovered[:limit] {
		fmt.Printf("  %d. %s (%d commits)\n", i+1, e.Path, e.Commits)
	}

	fmt.Printf("\nThese files have heavy recent activity but no spec references them.\n")
	fmt.Printf("Consider running `/design` to create specs for these areas.\n")

	return nil
}
