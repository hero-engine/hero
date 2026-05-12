package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/mission"
	"github.com/hero-engine/hero/internal/retrieval"
	"github.com/spf13/cobra"
)

var relevantCmd = &cobra.Command{
	Use:   "relevant [files...]",
	Short: "Surface what's relevant to the files you're touching",
	Long: `Checks the spec corpus and graph for conventions, past work, and
in-flight specs relevant to the given files. Returns a lightweight,
focused message agents can include when a developer is working
without an explicit spec — so the model knows which conventions
apply, which prior decisions matter, and who else is actively
working in this area.

Files can be passed positionally (preferred) or via --files for
backward compatibility:

  hero relevant src/foo.go src/bar.go
  hero relevant --files src/foo.go,src/bar.go

The verbosity level (off, gentle, assertive) is controlled by
team.nudge_level in hero.json. When called interactively (no --quiet)
and no relevant context exists, the command prints a one-line message
so users know it succeeded.`,
	RunE: runRelevant,
}

var (
	relevantFiles []string
)

func init() {
	relevantCmd.Flags().StringSliceVar(&relevantFiles, "files", nil, "file paths to check for context (alternative to positional args)")
}

func runRelevant(cmd *cobra.Command, args []string) error {
	files := append([]string{}, relevantFiles...)
	files = append(files, args...)
	if len(files) == 0 {
		return fmt.Errorf("no files specified — pass paths positionally or via --files")
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	nudgeLevel := cfg.Team.NudgeLevel
	if nudgeLevel == "off" {
		return nil // silently do nothing — explicitly disabled
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		// Nudge runs in pre-commit hooks and ambient watch contexts —
		// stay silent on no-workspace so it doesn't pollute output.
		return nil
	}

	ret, err := retrieval.New(heroDir)
	if err != nil {
		// Index unavailable — silent, non-fatal.
		return nil
	}
	defer ret.Close()

	result, err := ret.NudgeFiles(files)
	if err != nil {
		// Retrieval error — silent, non-fatal.
		return nil
	}

	if result.IsEmpty() {
		// No relevant context — silent, the whole point of nudge.
		return nil
	}

	if m, _ := mission.LoadFile(heroDir); m != nil {
		if line := mission.Preamble(m); line != "" {
			fmt.Println(line)
			fmt.Println()
		}
	}

	switch nudgeLevel {
	case "gentle":
		printGentleNudge(result)
	case "assertive":
		printAssertiveNudge(result)
	default:
		printGentleNudge(result)
	}

	return nil
}

func printGentleNudge(result *index.NudgeResult) {
	fmt.Println("---")
	fmt.Println("**Hero** — relevant context exists for the files you're touching:")
	fmt.Println()

	if result.HasConventions {
		names := make([]string, len(result.Conventions))
		for i, c := range result.Conventions {
			names[i] = c.Slug
		}
		fmt.Printf("- Conventions: %s\n", strings.Join(names, ", "))
	}

	if result.HasPastWork {
		fmt.Printf("- %d past spec(s) touched these files\n", len(result.RelatedSpecs))
	}

	if result.HasPending {
		names := make([]string, len(result.PendingSpecs))
		for i, p := range result.PendingSpecs {
			names[i] = fmt.Sprintf("%s (%s)", p.Slug, string(p.Status))
		}
		fmt.Printf("- In-flight specs: %s\n", strings.Join(names, ", "))
	}

	fmt.Println()
	fmt.Println("Run `hero context --files <paths>` for full context, or use `/design` to create a spec for this work.")
	fmt.Println("---")
}

func printAssertiveNudge(result *index.NudgeResult) {
	fmt.Println("---")
	fmt.Println("## Hero — context awareness")
	fmt.Println()
	fmt.Println("You're working on files that have relevant context in the spec corpus.")
	fmt.Println("Working from a spec (`/design`) helps the team maintain cohesion and")
	fmt.Println("gives you conventions, past decisions, and known risks automatically.")
	fmt.Println()

	if result.HasConventions {
		fmt.Println("### Conventions that apply")
		for _, c := range result.Conventions {
			fmt.Printf("- **%s**: %s (path: %s)\n", c.Slug, c.Title, c.Path)
		}
		fmt.Println()
		fmt.Println("These conventions should be followed for consistency. Read the full convention spec if the name alone is not sufficient guidance.")
		fmt.Println()
	}

	if result.HasPastWork {
		fmt.Println("### Past work in this area")
		for _, r := range result.RelatedSpecs {
			fmt.Printf("- **%s**: %s\n", r.Slug, r.Title)
		}
		fmt.Println()
		fmt.Println("Review past specs to understand design rationale and avoid undoing previous work.")
		fmt.Println()
	}

	if result.HasPending {
		fmt.Println("### In-flight specs touching these files")
		for _, p := range result.PendingSpecs {
			fmt.Printf("- **%s** (%s): %s\n", p.Slug, string(p.Status), p.Title)
		}
		fmt.Println()
		fmt.Println("**Warning**: Another spec is actively being worked on in this area. Coordinate to avoid conflicts.")
		fmt.Println()
	}

	fmt.Println("### Recommendation")
	fmt.Println()
	fmt.Println("Use `/design` to create a spec for this work. This gives you:")
	fmt.Println("- Automatic context injection during delivery")
	fmt.Println("- Convention compliance")
	fmt.Println("- Conflict detection with other in-flight work")
	fmt.Println("- A record that becomes institutional knowledge for the team")
	fmt.Println("---")
}
