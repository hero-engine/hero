package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/feed"
	"github.com/spf13/cobra"
)

var (
	eventSlug    string
	eventSession string
	eventAgent   string
)

var eventCmd = &cobra.Command{
	Use:   "events <type> <message>",
	Short: "Log an event to the cross-session activity feed",
	Long: `Appends a significant event to .hero/events.log so other agents and
sessions can see what happened.

Valid types: spec_created, spec_updated, files_modified, decision_made,
blocker_hit, delivery_complete

Examples:
  hero event decision_made "Chose Redis over Postgres for session cache"
  hero event files_modified "Updated api/export.go" --slug csv-export
  hero event blocker_hit "Auth test failures" --slug csv-export`,
	Args: cobra.ExactArgs(2),
	RunE: runEvent,
}

func init() {
	eventCmd.Flags().StringVar(&eventSlug, "slug", "", "associate with a spec")
	eventCmd.Flags().StringVar(&eventSession, "session", "", "session ID (defaults to HERO_SESSION)")
	eventCmd.Flags().StringVar(&eventAgent, "agent", "", "agent identity (defaults to HERO_AGENT or hostname)")
}

func runEvent(cmd *cobra.Command, args []string) error {
	eventType, message := args[0], args[1]

	if !feed.IsValidType(eventType) {
		return fmt.Errorf("unknown event type %q\nValid types: %s", eventType, strings.Join(feed.ValidTypes, ", "))
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logPath := filepath.Join(cfg.HeroDir(projectRoot), "events.log")

	agent := eventAgent
	if agent == "" {
		agent = os.Getenv("HERO_AGENT")
	}
	if agent == "" {
		agent = "human/" + gitUserName()
	}

	session := eventSession
	if session == "" {
		session = os.Getenv("HERO_SESSION")
	}

	evt := feed.FeedEvent{
		Type:       eventType,
		Agent:      agent,
		Session:    session,
		Slug:       eventSlug,
		Message:    message,
		Subproject: resolveActiveScope(projectRoot, cfg.HeroDir(projectRoot)),
	}

	if err := feed.AppendEvent(logPath, evt); err != nil {
		return err
	}

	fmt.Printf("Logged %s event\n", eventType)
	return nil
}
