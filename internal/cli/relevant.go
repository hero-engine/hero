package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	contractpeering "github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/mission"
	"github.com/hero-engine/hero/internal/peering"
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
	relevantFiles            []string
	relevantPeers            []string
	relevantSurface          string
	relevantExcludeSuperseded bool
)

func init() {
	relevantCmd.Flags().StringSliceVar(&relevantFiles, "files", nil, "file paths to check for context (alternative to positional args)")
	relevantCmd.Flags().StringSliceVar(&relevantPeers, "peer", nil, "include peer-surface conventions from this peer alias (repeatable)")
	relevantCmd.Flags().StringVar(&relevantSurface, "surface", "", "filter peer conventions to those tagged with this surface (e.g. http-response)")
	relevantCmd.Flags().BoolVar(&relevantExcludeSuperseded, "exclude-superseded", false, "omit superseded specs from the nudge (default: surface them with a redirect marker)")
}

func runRelevant(cmd *cobra.Command, args []string) error {
	files := append([]string{}, relevantFiles...)
	files = append(files, args...)
	if len(files) == 0 && len(relevantPeers) == 0 {
		return fmt.Errorf("no files specified — pass paths positionally, via --files, or use --peer to request peer-surface conventions")
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	nudgeLevel := cfg.Team.NudgeLevel
	if nudgeLevel == "off" && len(relevantPeers) == 0 {
		// --peer is an explicit ask: honor it even when nudge is off.
		return nil
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		// Nudge runs in pre-commit hooks and ambient watch contexts —
		// stay silent on no-workspace so it doesn't pollute output.
		return nil
	}

	// Resolve peer conventions first so we can decide whether the
	// command produced *any* output at the end.
	peerConventions, peerErr := loadPeerConventions(cfg, projectRoot, relevantPeers, relevantSurface)
	if peerErr != nil {
		// Peer resolution errors are user-actionable — surface them.
		return peerErr
	}

	var result *index.NudgeResult
	if len(files) > 0 {
		ret, err := retrieval.New(heroDir)
		if err == nil {
			defer ret.Close()
			if r, err := ret.NudgeFiles(files); err == nil {
				result = r
			}
		}
	}

	// --exclude-superseded drops superseded entries from RelatedSpecs
	// entirely. Default keeps them with a redirect annotation so the
	// agent learns "this is the old answer; follow <slug>."
	if relevantExcludeSuperseded && result != nil {
		filtered := result.RelatedSpecs[:0]
		for _, r := range result.RelatedSpecs {
			if r.SupersededBy == "" {
				filtered = append(filtered, r)
			}
		}
		result.RelatedSpecs = filtered
		result.HasPastWork = len(result.RelatedSpecs) > 0
	}

	localEmpty := result == nil || result.IsEmpty()
	if localEmpty && len(peerConventions) == 0 {
		return nil
	}

	if m, _ := mission.LoadFile(heroDir); m != nil {
		if line := mission.Preamble(m); line != "" {
			fmt.Println(line)
			fmt.Println()
		}
	}

	if !localEmpty {
		switch nudgeLevel {
		case "assertive":
			printAssertiveNudge(result)
		default:
			printGentleNudge(result)
		}
	}

	if len(peerConventions) > 0 {
		printPeerConventions(peerConventions, relevantSurface)
	}

	return nil
}

// peerConventionGroup bundles peer-surface conventions read from a
// single peer's manifest, keyed by display alias.
type peerConventionGroup struct {
	Alias    string
	PeerID   string
	Entries  []contractpeering.ConventionEntry
}

// loadPeerConventions reads each requested peer's manifest, optionally
// filters by surface, and returns one group per peer. Returns a clear
// error when a peer alias is unconfigured or its manifest is missing —
// the user asked for this signal explicitly and a silent miss would be
// misleading.
func loadPeerConventions(cfg config.Config, projectRoot string, peers []string, surface string) ([]peerConventionGroup, error) {
	if len(peers) == 0 {
		return nil, nil
	}
	// De-duplicate aliases while preserving order.
	seen := map[string]bool{}
	var ordered []string
	for _, p := range peers {
		if seen[p] {
			continue
		}
		seen[p] = true
		ordered = append(ordered, p)
	}

	var out []peerConventionGroup
	for _, alias := range ordered {
		peerPath, err := cfg.ResolveRepoPath(projectRoot, alias)
		if err != nil {
			return nil, fmt.Errorf("peer %q: %w", alias, err)
		}
		manifest, err := peering.ReadPeerManifest(peerPath, cfg.Folder)
		if err != nil {
			return nil, fmt.Errorf("peer %q: %w", alias, err)
		}
		entries := manifest.Conventions
		if surface != "" {
			entries = peering.FilterConventionsBySurface(entries, surface)
		}
		// Stable display order.
		sort.Slice(entries, func(i, j int) bool { return entries[i].Slug < entries[j].Slug })
		out = append(out, peerConventionGroup{
			Alias:   alias,
			PeerID:  manifest.Repo.PeerID,
			Entries: entries,
		})
	}
	return out, nil
}

// printPeerConventions renders the peer-surface block. Distinct from
// the local nudge formatting so the boundary is obvious to a reader.
func printPeerConventions(groups []peerConventionGroup, surface string) {
	fmt.Println("---")
	if surface != "" {
		fmt.Printf("**Hero** — peer-surface conventions (surface: %s):\n", surface)
	} else {
		fmt.Println("**Hero** — peer-surface conventions:")
	}
	fmt.Println()
	for _, g := range groups {
		fmt.Printf("Peer `%s` (peer_id %s):\n", g.Alias, g.PeerID)
		if len(g.Entries) == 0 {
			fmt.Println("  (no peer-surface conventions match)")
			continue
		}
		for _, e := range g.Entries {
			surf := ""
			if len(e.Surface) > 0 {
				surf = " [" + strings.Join(e.Surface, ",") + "]"
			}
			fmt.Printf("  - %s — %s%s\n    path: %s\n", e.Slug, e.Title, surf, e.Path)
		}
	}
	fmt.Println("---")
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
	fmt.Printf("Run `%s <paths>` for full context, or use `/design` to create a spec for this work.\n", cliHintByID("context-imports-files"))
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
			line := fmt.Sprintf("- **%s**: %s", r.Slug, r.Title)
			if r.SupersededBy != "" {
				line += fmt.Sprintf(" [SUPERSEDED by %s — follow %s instead]", r.SupersededBy, r.SupersededBy)
			} else if r.Status == "superseded" {
				line += " [SUPERSEDED — replacement unknown]"
			}
			fmt.Println(line)
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
