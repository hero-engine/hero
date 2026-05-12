package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/tracking"
	"github.com/spf13/cobra"
)

var (
	velocityJSON  bool
	velocitySince string
	velocityAgent string
)

var velocityCmd = &cobra.Command{
	Use:   "velocity",
	Short: "Show agent contribution velocity metrics",
	RunE:  runVelocity,
}

func init() {
	velocityCmd.Flags().BoolVar(&velocityJSON, "json", false, "output as JSON")
	velocityCmd.Flags().StringVar(&velocitySince, "since", "", "limit to events since date (YYYY-MM-DD) or duration (e.g. 30d)")
	velocityCmd.Flags().StringVar(&velocityAgent, "agent", "", "filter output to a specific agent")
}

func runVelocity(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)

	eventsLogPath := filepath.Join(heroDir, "events.log")

	events, err := tracking.ReadEvents(eventsLogPath)
	if err != nil {
		return fmt.Errorf("reading events: %w", err)
	}

	// Parse since
	since, sinceLabel, err := parseSince(velocitySince)
	if err != nil {
		return fmt.Errorf("invalid --since value: %w", err)
	}

	velocities := tracking.CalcVelocity(events, since)

	// Filter by agent if requested
	if velocityAgent != "" {
		var filtered []tracking.AgentVelocity
		for _, v := range velocities {
			if v.Agent == velocityAgent {
				filtered = append(filtered, v)
			}
		}
		velocities = filtered
	}

	if velocityJSON {
		data, err := json.MarshalIndent(velocities, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Plaintext output
	if sinceLabel == "" {
		sinceLabel = "all time"
	}
	fmt.Printf("Agent Velocity — %s\n\n", sinceLabel)

	if len(velocities) == 0 {
		fmt.Println("  No completed specs found in this window.")
		return nil
	}

	for _, v := range velocities {
		fmt.Printf("  %-30s  %2d specs done    avg %.1f days/spec\n",
			v.Agent, v.SpecsDone, v.AvgDays)
	}

	// Global fastest / slowest
	var fastest, slowest *tracking.AgentVelocity
	// We need to find globally fastest/slowest across all agents using raw events
	fastestDays, slowestDays := -1.0, -1.0
	type slugAgent struct {
		slug  string
		agent string
		days  float64
	}
	var completions []slugAgent
	type claimKey struct{ slug, agent string }
	claimTimes := make(map[claimKey]time.Time)
	for _, evt := range events {
		if !since.IsZero() && evt.At.Before(since) {
			continue
		}
		key := claimKey{evt.Slug, evt.Agent}
		if evt.Event == "claimed" {
			if _, ok := claimTimes[key]; !ok {
				claimTimes[key] = evt.At
			}
		}
		if evt.Event == "completed" {
			if ct, ok := claimTimes[key]; ok {
				days := evt.At.Sub(ct).Hours() / 24.0
				if days < 0 {
					days = 0
				}
				completions = append(completions, slugAgent{evt.Slug, evt.Agent, days})
			}
		}
	}

	for _, c := range completions {
		if fastestDays < 0 || c.days < fastestDays {
			fastestDays = c.days
			fastest = &tracking.AgentVelocity{FastestSlug: c.slug, Agent: c.agent, AvgDays: c.days}
		}
		if slowestDays < 0 || c.days > slowestDays {
			slowestDays = c.days
			slowest = &tracking.AgentVelocity{SlowestSlug: c.slug, Agent: c.agent, AvgDays: c.days}
		}
	}

	fmt.Println()
	if fastest != nil {
		fmt.Printf("Fastest spec: %s (%.1f days, %s)\n", fastest.FastestSlug, fastestDays, fastest.Agent)
	}
	if slowest != nil {
		fmt.Printf("Slowest spec: %s (%.1f days, %s)\n", slowest.SlowestSlug, slowestDays, slowest.Agent)
	}

	activeClaims := tracking.ActiveClaims(events)
	fmt.Printf("Currently claimed: %d specs\n", len(activeClaims))

	return nil
}

// parseSince parses a --since string into a time.Time.
// Supports YYYY-MM-DD or Nd (e.g. "30d").
// Returns zero time for empty string (all time).
func parseSince(s string) (time.Time, string, error) {
	if s == "" {
		return time.Time{}, "Last 30 days", nil
	}

	// Try Nd format
	if strings.HasSuffix(s, "d") {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			t := time.Now().AddDate(0, 0, -days)
			return t, fmt.Sprintf("Last %d days", days), nil
		}
	}

	// Try YYYY-MM-DD
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("expected YYYY-MM-DD or Nd format, got %q", s)
	}
	return t, fmt.Sprintf("Since %s", s), nil
}

// eventsLogPath returns the absolute path to .hero/events.log.
func eventsLogPath(heroDir string) string {
	return filepath.Join(heroDir, "events.log")
}


