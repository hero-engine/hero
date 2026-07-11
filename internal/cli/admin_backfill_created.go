package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var (
	backfillCreatedDryRun bool
	backfillCreatedQuiet  bool
)

// backfillCreatedCmd derives the `created:` frontmatter field for specs that
// were authored before Hero reliably stamped it. It walks every work spec (and
// initiative) whose CreatedAt came from file mtime rather than an authored
// `created:`, asks git for the oldest commit touching the file, and stamps that
// date. Uncommitted specs are stamped with today's date (a just-authored file's
// creation date IS today). Already-stamped specs are skipped. Re-runnable.
var backfillCreatedCmd = &cobra.Command{
	Use:   "backfill-created",
	Short: "Stamp specs with created: from their first git commit",
	Long: `One-shot backfiller for the created: frontmatter field.

Walks every work spec under .hero/specs/ and .hero/planning/, filters to
those with no authored created: field, then runs
'git log --follow --reverse --format=%aI -- <spec-path>' to find the OLDEST
commit touching the file and stamps that date. Specs with no git history
(uncommitted) are stamped with today's date, since for a just-authored file
today is the true creation date.

Re-runnable: specs that already carry created: are skipped. Use --dry-run to
see what would be stamped without writing.`,
	RunE: runBackfillCreated,
}

func init() {
	backfillCreatedCmd.Flags().BoolVar(&backfillCreatedDryRun, "dry-run", false,
		"preview what would be stamped without writing")
	backfillCreatedCmd.Flags().BoolVar(&backfillCreatedQuiet, "quiet", false,
		"suppress per-spec output, print summary only")
}

func runBackfillCreated(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	var stamped, skipped int
	for _, s := range specs {
		if !s.IsWorkSpec() && s.Type != spec.TypeInitiative {
			continue
		}
		if s.CreatedFromFrontmatter {
			skipped++
			if !backfillCreatedQuiet {
				fmt.Printf("skip   %s (already stamped: %s)\n", s.Slug,
					s.CreatedAt.Format("2006-01-02"))
			}
			continue
		}

		ts := createdDate(projectRoot, s.Path)

		if backfillCreatedDryRun {
			stamped++
			if !backfillCreatedQuiet {
				fmt.Printf("would  %s -> %s\n", s.Slug, ts.Format("2006-01-02"))
			}
			continue
		}

		if err := writeCreatedStamp(s.Path, ts); err != nil {
			fmt.Fprintf(os.Stderr, "warn   %s: %v\n", s.Slug, err)
			continue
		}
		stamped++
		if !backfillCreatedQuiet {
			fmt.Printf("stamp  %s -> %s\n", s.Slug, ts.Format("2006-01-02"))
		}
	}

	if !backfillCreatedDryRun && stamped > 0 {
		if _, err := index.Rebuild(heroDir); err != nil {
			fmt.Fprintf(os.Stderr, "warn: re-index failed: %v\n", err)
		}
	}

	fmt.Printf("Stamped: %d, Skipped (already stamped): %d\n", stamped, skipped)
	if backfillCreatedDryRun {
		fmt.Println("(dry-run — no files written)")
	}
	return nil
}

// workSpecsMissingCreated returns work specs (and initiatives) whose CreatedAt
// fell back to file mtime rather than an authored `created:` field — the set
// `hero check` reports and `--reconcile` / `backfill-created` stamps. Best
// effort: a discovery error yields an empty slice (check reports "pass").
func workSpecsMissingCreated(heroDir string) []*spec.Spec {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil
	}
	var out []*spec.Spec
	for _, s := range specs {
		if !s.IsWorkSpec() && s.Type != spec.TypeInitiative {
			continue
		}
		if !s.CreatedFromFrontmatter {
			out = append(out, s)
		}
	}
	return out
}

// createdDate resolves a spec's creation date: the author-date of the oldest
// commit that touched the file (following renames), truncated to date. Falls
// back to today's date when the file has no git history — for a just-authored
// spec, today IS the creation date, so synthesizing here is correct (unlike
// completed_at, which refuses to synthesize a completion time).
func createdDate(projectRoot, specPath string) time.Time {
	if ts, ok := gitFirstCommitDate(projectRoot, specPath); ok {
		return ts
	}
	return time.Now().UTC().Truncate(24 * time.Hour)
}

// gitFirstCommitDate asks git for the author-date of the OLDEST commit that
// touched specPath (following renames), in RFC 3339 form. Returns (time, true)
// on success; (zero, false) when git returns empty (no history) or errors.
func gitFirstCommitDate(projectRoot, specPath string) (time.Time, bool) {
	cmd := exec.Command("git", "log", "--follow", "--reverse", "--format=%aI", "--", specPath)
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, false
	}
	lines := strings.Fields(strings.TrimSpace(string(out)))
	if len(lines) == 0 {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, lines[0])
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

// writeCreatedStamp sets the created: frontmatter field to the supplied date
// (YYYY-MM-DD), matching the authored frontmatter convention.
func writeCreatedStamp(path string, ts time.Time) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	updated := spec.SetFrontmatterField(string(data), "created", ts.Format("2006-01-02"))
	return os.WriteFile(path, []byte(updated), 0o644)
}
