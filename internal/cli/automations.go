package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/automations"
	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

var automationsCmd = &cobra.Command{
	Use:   "automate",
	Short: "Manage event-driven automations",
	Long: `Configure and inspect automations that trigger agent work from
external events (Jira bugs, GitHub webhooks, schedules).

Subcommands:
  hero automations              — list configured automations
  hero automations test <name>  — dry-run against sample data
  hero automations log          — recent automation activity`,
	RunE: runAutomationsList,
}

var automationsTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Dry-run an automation against sample data",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutomationsTest,
}

var automationsLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Show recent automation activity",
	RunE:  runAutomationsLog,
}

var automationsLogLimit int

func init() {
	automationsLogCmd.Flags().IntVar(&automationsLogLimit, "limit", 20, "max log entries to show")
	automationsCmd.AddCommand(automationsTestCmd)
	automationsCmd.AddCommand(automationsLogCmd)
}

func runAutomationsList(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	engine := automations.NewEngine(heroDir, projectRoot)
	if err := engine.Load(); err != nil {
		return err
	}

	list := engine.List()
	if len(list) == 0 {
		fmt.Println("No automations configured.")
		fmt.Println("Create YAML files in .hero/automations/ to define triggers.")
		return nil
	}

	fmt.Printf("Automations (%d):\n\n", len(list))
	for _, as := range list {
		status := "enabled"
		if !as.Config.Enabled {
			status = "disabled"
		}
		lastFired := "never"
		if !as.LastFired.IsZero() {
			lastFired = as.LastFired.Format(time.RFC3339)
		}
		fmt.Printf("  %-25s  %-8s  trigger: %s/%s  last: %s\n",
			as.Config.Name, status,
			as.Config.Trigger.Type, as.Config.Trigger.Event,
			lastFired)
	}

	return nil
}

func runAutomationsTest(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	engine := automations.NewEngine(heroDir, projectRoot)
	if err := engine.Load(); err != nil {
		return err
	}

	// Sample payload for testing
	payload := map[string]string{
		"tracker_id": "SAMPLE-123",
		"type":       "Bug",
		"priority":   "Critical",
		"status":     "Open",
		"title":      "Sample bug for testing",
	}

	result, err := engine.Test(args[0], payload)
	if err != nil {
		return err
	}
	fmt.Print(result)
	return nil
}

func runAutomationsLog(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	engine := automations.NewEngine(heroDir, projectRoot)

	entries, err := engine.ReadLog(automationsLogLimit)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Println("No automation activity yet.")
		return nil
	}

	fmt.Printf("Recent automation activity (%d entries):\n\n", len(entries))
	for _, e := range entries {
		fmt.Printf("  %s  %-20s  %-10s  %s %s\n",
			e.Timestamp.Format("Jan 02 15:04"),
			e.Automation,
			e.Status,
			e.Action,
			truncateStr(e.Error, 40))
	}

	return nil
}

func truncateStr(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
