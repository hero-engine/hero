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
	backfillCompletedAtDryRun bool
	backfillCompletedAtQuiet  bool
)

// backfillCompletedAtCmd derives the `completed_at:` frontmatter field
// for historical specs that were completed before Hero started writing
// the stamp. It walks every completed work spec, asks git for the most
// recent commit time touching the spec file, and writes that timestamp
// into the frontmatter. Already-stamped specs and specs with no git
// history are reported and skipped. Re-runnable.
var backfillCompletedAtCmd = &cobra.Command{
	Use:   "backfill-completed-at",
	Short: "Stamp historical specs with completed_at from git log",
	Long: `One-shot backfiller for the completed_at: frontmatter field.

Walks every spec under .hero/specs/ and .hero/planning/, filters to
status: completed with completed_at missing, then runs
'git log -1 --format=%aI -- <spec-path>' to find the most recent
commit time touching the file and stamps it. Specs that have no git
history (uncommitted) are reported but not stamped — backfill never
synthesizes a timestamp.

Re-runnable: already-stamped specs are skipped. Use --dry-run to see
which specs would be touched without writing.`,
	RunE: runBackfillCompletedAt,
}

func init() {
	backfillCompletedAtCmd.Flags().BoolVar(&backfillCompletedAtDryRun, "dry-run", false,
		"preview what would be stamped without writing")
	backfillCompletedAtCmd.Flags().BoolVar(&backfillCompletedAtQuiet, "quiet", false,
		"suppress per-spec output, print summary only")
}

func runBackfillCompletedAt(cmd *cobra.Command, args []string) error {
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

	var stamped, skipped, noGit int
	for _, s := range specs {
		if s.Status != spec.StatusCompleted {
			continue
		}
		if !s.CompletedAt.IsZero() {
			skipped++
			if !backfillCompletedAtQuiet {
				fmt.Printf("skip   %s (already stamped: %s)\n", s.Slug,
					s.CompletedAt.UTC().Format(time.RFC3339))
			}
			continue
		}

		ts, ok := gitCommittedAt(projectRoot, s.Path)
		if !ok {
			noGit++
			if !backfillCompletedAtQuiet {
				fmt.Printf("no-git %s (no git history)\n", s.Slug)
			}
			continue
		}

		if backfillCompletedAtDryRun {
			stamped++
			if !backfillCompletedAtQuiet {
				fmt.Printf("would  %s -> %s\n", s.Slug, ts.Format(time.RFC3339))
			}
			continue
		}

		if err := writeBackfillStamp(s.Path, ts); err != nil {
			fmt.Fprintf(os.Stderr, "warn   %s: %v\n", s.Slug, err)
			continue
		}
		stamped++
		if !backfillCompletedAtQuiet {
			fmt.Printf("stamp  %s -> %s\n", s.Slug, ts.Format(time.RFC3339))
		}
	}

	if !backfillCompletedAtDryRun && stamped > 0 {
		if _, err := index.Rebuild(heroDir); err != nil {
			fmt.Fprintf(os.Stderr, "warn: re-index failed: %v\n", err)
		}
	}

	fmt.Printf("Stamped: %d, Skipped (already stamped): %d, No git history: %d\n",
		stamped, skipped, noGit)
	if backfillCompletedAtDryRun {
		fmt.Println("(dry-run — no files written)")
	}
	return nil
}

// gitCommittedAt asks git for the author-date of the most recent commit
// that touched specPath, in RFC 3339 form. Returns (time, true) on
// success; (zero, false) when git returns empty (no history) or errors.
// Empty output is the "uncommitted file" case — we report it rather
// than stamping a synthesized time.
func gitCommittedAt(projectRoot, specPath string) (time.Time, bool) {
	cmd := exec.Command("git", "log", "-1", "--format=%aI", "--", specPath)
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, false
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

// writeBackfillStamp sets the completed_at frontmatter field to the
// supplied historical timestamp. Uses SetFrontmatterField directly
// (not StampCompletedAt) because the latter would substitute the
// current time.
func writeBackfillStamp(path string, ts time.Time) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	updated := spec.SetFrontmatterField(string(data), "completed_at",
		ts.Format(time.RFC3339))
	return os.WriteFile(path, []byte(updated), 0o644)
}
