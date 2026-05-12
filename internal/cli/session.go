package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/sessions"
	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage reasoning log sessions",
}

// ---------- session start ----------

var (
	sessionStartName  string
	sessionStartAgent string
)

var sessionStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a new reasoning session",
	RunE:  runSessionStart,
}

func init() {
	sessionStartCmd.Flags().StringVar(&sessionStartName, "name", "", "human-readable session name")
	sessionStartCmd.Flags().StringVar(&sessionStartAgent, "agent", "", "agent identity")

	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionEndCmd)
	sessionCmd.AddCommand(sessionLogCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionReplayCmd)
	sessionCmd.AddCommand(sessionDistillCmd)
	sessionCmd.AddCommand(sessionPruneCmd)
}

func runSessionStart(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)

	agent := resolveSessionAgent(sessionStartAgent, cfg)

	sess, err := sessions.Start(heroDir, sessionStartName, agent)
	if err != nil {
		return fmt.Errorf("starting session: %w", err)
	}

	fmt.Println(sess.ID)
	return nil
}

// ---------- session end ----------

var sessionEndCmd = &cobra.Command{
	Use:   "end [<id>]",
	Short: "End the current or specified session",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSessionEnd,
}

func runSessionEnd(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)

	id := ""
	if len(args) > 0 {
		id = args[0]
	} else {
		id = os.Getenv("HERO_SESSION")
	}

	if id == "" {
		// Try to end the most recent open session
		sess, err := sessions.Load(heroDir, "")
		if err != nil {
			return fmt.Errorf("no session to end: %w", err)
		}
		id = sess.ID
	}

	if err := sessions.End(heroDir, id); err != nil {
		return err
	}

	fmt.Printf("Session %s ended.\n", id)
	return nil
}

// ---------- session log ----------

var sessionLogCmd = &cobra.Command{
	Use:   "log [<id>]",
	Short: "Show events from a session",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSessionLog,
}

func runSessionLog(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)

	id := ""
	if len(args) > 0 {
		id = args[0]
	} else {
		id = os.Getenv("HERO_SESSION")
	}

	if id == "" {
		sess, err := sessions.Load(heroDir, "")
		if err != nil {
			return fmt.Errorf("no session found: %w", err)
		}
		id = sess.ID
	}

	events, err := sessions.ReadEvents(heroDir, id)
	if err != nil {
		return fmt.Errorf("reading events: %w", err)
	}
	if len(events) == 0 {
		fmt.Printf("No events in session %s.\n", id)
		return nil
	}

	fmt.Printf("Events for session %s:\n\n", id)
	for _, evt := range events {
		t, _ := evt["t"].(string)
		evtType, _ := evt["event"].(string)
		fmt.Printf("  [%s] %s", t, evtType)
		// Print any extra fields
		for k, v := range evt {
			if k == "t" || k == "event" || k == "session" {
				continue
			}
			fmt.Printf("  %s=%v", k, v)
		}
		fmt.Println()
	}
	return nil
}

// ---------- session list ----------

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	RunE:  runSessionList,
}

func runSessionList(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)

	list, err := sessions.List(heroDir)
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}
	if len(list) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	fmt.Printf("%-18s  %-20s  %-20s  %-10s  %s\n", "ID", "Name", "Agent", "Started", "Status")
	fmt.Println(strings.Repeat("─", 80))
	for _, s := range list {
		name := s.Name
		if name == "" {
			name = "-"
		}
		agent := s.Agent
		if agent == "" {
			agent = "-"
		}
		status := "open"
		if s.End != nil {
			status = "ended"
		}
		fmt.Printf("%-18s  %-20s  %-20s  %-10s  %s\n",
			s.ID, truncate(name, 20), truncate(agent, 20),
			s.Start.Format("2006-01-02 15:04"), status)
	}
	return nil
}

// ---------- session replay ----------

var sessionReplayCmd = &cobra.Command{
	Use:   "replay <id>",
	Short: "Render a full session summary",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionReplay,
}

func runSessionReplay(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)

	output, err := sessions.Replay(heroDir, args[0])
	if err != nil {
		return err
	}
	fmt.Print(output)
	return nil
}

// ---------- session distill ----------

var sessionDistillCmd = &cobra.Command{
	Use:   "distill <id>",
	Short: "Suggest knowledge entries from a session",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionDistill,
}

func runSessionDistill(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)

	output, err := sessions.Distill(heroDir, args[0])
	if err != nil {
		return err
	}
	fmt.Print(output)
	return nil
}

// ---------- session prune ----------

var sessionPruneDays int

var sessionPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune sessions older than N days",
	RunE:  runSessionPrune,
}

func init() {
	sessionPruneCmd.Flags().IntVar(&sessionPruneDays, "days", 30, "prune sessions older than this many days")
}

func runSessionPrune(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)

	days := sessionPruneDays
	if cfg.Sessions != nil && cfg.Sessions.RetentionDays > 0 && !cmd.Flags().Changed("days") {
		days = cfg.Sessions.RetentionDays
	}

	pruned, err := sessions.Prune(heroDir, days)
	if err != nil {
		return fmt.Errorf("pruning sessions: %w", err)
	}

	if pruned == 0 {
		fmt.Println("No sessions to prune.")
	} else {
		fmt.Printf("Pruned %d session(s).\n", pruned)
	}
	return nil
}

// ---------- helpers ----------

// resolveSessionAgent determines agent identity:
// 1. agentFlag if non-empty
// 2. HERO_AGENT env var
// 3. cfg.Tracking.DefaultAgent
// 4. "human/<git-user>"
func resolveSessionAgent(agentFlag string, cfg config.Config) string {
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

// gitUserName returns the git user.name, lowercased with spaces replaced by hyphens.
func gitUserName() string {
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return "unknown"
	}
	name := strings.TrimSpace(string(out))
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	if name == "" {
		return "unknown"
	}
	return name
}

// truncate shortens s to max chars, adding "…" if truncated.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
