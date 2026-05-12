package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracking"
	"github.com/spf13/cobra"
)

// ---------- claim ----------

var (
	claimAgentFlag string
	claimRelease   bool
	claimComplete  bool
)

var claimCmd = &cobra.Command{
	Use:   "claim <spec-slug>",
	Short: "Claim a spec for delivery",
	Long: `Marks a spec as claimed by an agent. Use --release to release the claim or
--complete to mark the spec as done. Other developers will see the claim
in hero status and hero search output.`,
	Args: cobra.ExactArgs(1),
	RunE: runClaimNew,
}

func init() {
	claimCmd.Flags().StringVar(&claimAgentFlag, "agent", "", "agent identity (default: resolved from env/config/git)")
	claimCmd.Flags().BoolVar(&claimRelease, "release", false, "release the claim on this spec")
	claimCmd.Flags().BoolVar(&claimComplete, "complete", false, "mark the spec as completed")
}

// ---------- unclaim (kept for backwards compatibility) ----------

var unclaimCmd = &cobra.Command{
	Use:    "unclaim <spec-slug>",
	Short:  "Release a claim on a spec",
	Long:   "Removes the claim on a spec so others can pick it up.",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	RunE:   runUnclaim,
}

// ---------- claims (list) ----------

var (
	claimsAgentFilter string
	claimsStale       bool
)

var claimsCmd = &cobra.Command{
	Use:   "claims",
	Short: "List active spec claims",
	RunE:  runClaims,
}

func init() {
	claimsCmd.Flags().StringVar(&claimsAgentFilter, "agent", "", "filter by agent")
	claimsCmd.Flags().BoolVar(&claimsStale, "stale", false, "show only stale claims")
}

// ---------- implementations ----------

func runClaimNew(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	slug := args[0]
	agent := resolveAgent(claimAgentFlag, cfg)
	logPath := eventsLogPath(heroDir)
	now := time.Now()

	// Determine action
	action := "claim"
	if claimRelease {
		action = "release"
	} else if claimComplete {
		action = "complete"
	}

	// Find spec file
	specPath, err := findSpecBySlug(heroDir, slug)
	if err != nil {
		// Don't fail if spec file not found — still update index
		specPath = ""
	}

	switch action {
	case "claim":
		// Update spec frontmatter
		if specPath != "" {
			if ferr := tracking.UpdateSpecFrontmatter(specPath, "claim", agent, now); ferr != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not update spec frontmatter: %v\n", ferr)
			}
		}

		// Update index
		if idx, idxErr := index.Open(heroDir); idxErr == nil {
			if cerr := idx.Claim(slug, agent); cerr != nil {
				idx.Close()
				return cerr
			}
			idx.Close()
		}

		// Append event
		_ = tracking.AppendEvent(logPath, tracking.ClaimEvent{
			Event: "claimed",
			Slug:  slug,
			Agent: agent,
			At:    now,
		})

		fmt.Printf("Claimed %s for %s\n", slug, agent)

	case "release":
		// Update spec frontmatter
		if specPath != "" {
			if ferr := tracking.UpdateSpecFrontmatter(specPath, "release", agent, now); ferr != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not update spec frontmatter: %v\n", ferr)
			}
		}

		// Update index
		if idx, idxErr := index.Open(heroDir); idxErr == nil {
			_ = idx.Unclaim(slug)
			idx.Close()
		}

		// Append event
		_ = tracking.AppendEvent(logPath, tracking.ClaimEvent{
			Event: "released",
			Slug:  slug,
			Agent: agent,
			At:    now,
		})

		fmt.Printf("Released claim on %s\n", slug)

	case "complete":
		// Calculate duration from claimed_at
		durMins := 0
		events, _ := tracking.ReadEvents(logPath)
		for i := len(events) - 1; i >= 0; i-- {
			evt := events[i]
			if evt.Slug == slug && evt.Agent == agent && evt.Event == "claimed" {
				durMins = int(now.Sub(evt.At).Minutes())
				break
			}
		}

		// Update spec frontmatter
		if specPath != "" {
			if ferr := tracking.UpdateSpecFrontmatter(specPath, "complete", agent, now); ferr != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not update spec frontmatter: %v\n", ferr)
			}
		}

		// Update index
		if idx, idxErr := index.Open(heroDir); idxErr == nil {
			_ = idx.Unclaim(slug)
			idx.Close()
		}

		// Append event
		_ = tracking.AppendEvent(logPath, tracking.ClaimEvent{
			Event:           "completed",
			Slug:            slug,
			Agent:           agent,
			At:              now,
			DurationMinutes: durMins,
		})

		fmt.Printf("Completed %s (agent: %s", slug, agent)
		if durMins > 0 {
			fmt.Printf(", %d min", durMins)
		}
		fmt.Println(")")
	}

	return nil
}

func runUnclaim(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	slug := args[0]
	agent := resolveAgent("", cfg)
	logPath := eventsLogPath(heroDir)
	now := time.Now()

	// Update frontmatter
	if specPath, err := findSpecBySlug(heroDir, slug); err == nil {
		if ferr := tracking.UpdateSpecFrontmatter(specPath, "release", agent, now); ferr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not update spec frontmatter: %v\n", ferr)
		}
	}

	// Update index
	idx, err := index.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()
	if err := idx.Unclaim(slug); err != nil {
		return err
	}

	// Append event
	_ = tracking.AppendEvent(logPath, tracking.ClaimEvent{
		Event: "released",
		Slug:  slug,
		Agent: agent,
		At:    now,
	})

	fmt.Printf("Released claim on %s\n", slug)
	return nil
}

func runClaims(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	logPath := eventsLogPath(heroDir)
	events, err := tracking.ReadEvents(logPath)
	if err != nil {
		return fmt.Errorf("reading events: %w", err)
	}

	staleDays := cfg.Tracking.StaleClaimDaysOrDefault()

	var claims []tracking.ClaimEvent
	if claimsStale {
		claims = tracking.StaleClaims(events, staleDays)
	} else {
		claims = tracking.ActiveClaims(events)
	}

	// Filter by agent
	if claimsAgentFilter != "" {
		var filtered []tracking.ClaimEvent
		for _, c := range claims {
			if c.Agent == claimsAgentFilter {
				filtered = append(filtered, c)
			}
		}
		claims = filtered
	}

	if len(claims) == 0 {
		if claimsStale {
			fmt.Println("No stale claims found.")
		} else {
			fmt.Println("No active claims.")
		}
		return nil
	}

	label := "Active claims"
	if claimsStale {
		label = fmt.Sprintf("Stale claims (>%d days)", staleDays)
	}
	fmt.Printf("%s:\n\n", label)
	fmt.Printf("  %-30s  %-25s  %s\n", "Slug", "Agent", "Claimed at")
	fmt.Printf("  %s\n", strings.Repeat("─", 70))
	for _, c := range claims {
		fmt.Printf("  %-30s  %-25s  %s\n",
			c.Slug, c.Agent, c.At.Format("2006-01-02 15:04"))
	}
	return nil
}

// resolveAgent determines agent identity:
// 1. agentFlag if non-empty
// 2. HERO_AGENT env var
// 3. cfg.Tracking.DefaultAgent
// 4. "human/<git-user>"
func resolveAgent(agentFlag string, cfg config.Config) string {
	if agentFlag != "" {
		return agentFlag
	}
	if v := os.Getenv("HERO_AGENT"); v != "" {
		return v
	}
	if cfg.Tracking != nil && cfg.Tracking.DefaultAgent != "" {
		return cfg.Tracking.DefaultAgent
	}
	return "human/" + gitUserName()
}

// findSpecBySlug searches for a spec.md file matching the given slug under heroDir.
func findSpecBySlug(heroDir, slug string) (string, error) {
	// Search in planning/ and specs/ subdirectories
	for _, base := range []string{"planning", "specs"} {
		// Try direct path: <base>/<type>/<slug>/spec.md and <base>/<slug>/spec.md
		dirs := []string{
			filepath.Join(heroDir, base, slug, "spec.md"),
		}
		// Also search all subdirectories
		for _, candidate := range dirs {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}

		// Walk base directory looking for <slug>/spec.md
		baseDir := filepath.Join(heroDir, base)
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			// Check direct child
			candidate := filepath.Join(baseDir, entry.Name(), slug, "spec.md")
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
			// Also check if entry.Name() IS the slug
			if entry.Name() == slug {
				candidate := filepath.Join(baseDir, entry.Name(), "spec.md")
				if _, err := os.Stat(candidate); err == nil {
					return candidate, nil
				}
			}
		}
	}

	// Walk entire heroDir looking for spec with matching slug in frontmatter or path
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return "", fmt.Errorf("spec %q not found", slug)
	}
	for _, s := range specs {
		if s.Slug == slug {
			return s.Path, nil
		}
	}

	return "", fmt.Errorf("spec %q not found", slug)
}
