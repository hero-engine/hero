package cli

import (
	"fmt"
	"path/filepath"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/feed"
	"github.com/spf13/cobra"
)

var (
	feedSince      string
	feedType       string
	feedSlug       string
	feedAgent      string
	feedLimit      int
	feedFormat     string
	feedSubproject string
)

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Show recent activity across all sessions",
	Long: `Displays the cross-session activity feed — significant events logged by
all agents working in this repo. Newest events first.

Examples:
  hero feed                       # last 20 events
  hero feed --since 1h            # events from the last hour
  hero feed --type decision_made  # filter by type
  hero feed --slug csv-export     # filter by spec`,
	RunE: runFeed,
}

func init() {
	feedCmd.Flags().StringVar(&feedSince, "since", "", "filter: duration (1h, 30m) or RFC 3339 timestamp")
	feedCmd.Flags().StringVar(&feedType, "type", "", "filter by event type")
	feedCmd.Flags().StringVar(&feedSlug, "slug", "", "filter by spec slug")
	feedCmd.Flags().StringVar(&feedAgent, "agent", "", "filter by agent identity")
	feedCmd.Flags().IntVar(&feedLimit, "limit", 20, "max events to show")
	feedCmd.Flags().StringVar(&feedFormat, "format", "", "output format: text (default), json")
	feedCmd.Flags().StringVar(&feedSubproject, "subproject", "", "filter by subproject scope (e.g. engines/mlx); 'all' disables. Default: active scope from cwd")
}

func runFeed(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logPath := filepath.Join(cfg.HeroDir(projectRoot), "events.log")

	filter := feed.Filter{Limit: feedLimit}

	if feedSince != "" {
		t, err := feed.ParseSince(feedSince)
		if err != nil {
			return err
		}
		filter.Since = t
	}
	filter.Type = feedType
	filter.Slug = feedSlug
	filter.Agent = feedAgent
	filter.Subproject = resolveSubprojectFilter(feedSubproject)
	maybePrintScopeHint(cmd.ErrOrStderr(), feedSubproject, filter.Subproject)

	events, err := feed.ReadEvents(logPath, filter)
	if err != nil {
		return err
	}

	if feedFormat == "json" {
		out, err := feed.FormatJSON(events)
		if err != nil {
			return err
		}
		fmt.Println(out)
	} else {
		fmt.Print(feed.FormatText(events))
	}

	return nil
}
