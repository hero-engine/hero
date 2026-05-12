package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/hooks"
	"github.com/spf13/cobra"
)

var nlhookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all natural-language hooks in .hero/hooks/",
	RunE:  runNLHookList,
}

var nlhookShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Print a hook's content and computed match pattern",
	Args:  cobra.ExactArgs(1),
	RunE:  runNLHookShow,
}

var nlhookTestCmd = &cobra.Command{
	Use:   "test <name> --file <path>",
	Short: "Dry-run: classify match, render prompt, no fire",
	Args:  cobra.ExactArgs(1),
	RunE:  runNLHookTest,
}

var nlhookFireCmd = &cobra.Command{
	Use:   "fire <name> --file <path>",
	Short: "Fire a hook: check match and emit rendered prompt",
	Args:  cobra.ExactArgs(1),
	RunE:  runNLHookFire,
}

var nlhookFile string

func init() {
	nlhookTestCmd.Flags().StringVar(&nlhookFile, "file", "", "file path to test against")
	nlhookFireCmd.Flags().StringVar(&nlhookFile, "file", "", "file path that triggered the event")

	hookCmd.AddCommand(nlhookListCmd)
	hookCmd.AddCommand(nlhookShowCmd)
	hookCmd.AddCommand(nlhookTestCmd)
	hookCmd.AddCommand(nlhookFireCmd)
}

func runNLHookList(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	allHooks, err := hooks.Discover(heroDir)
	if err != nil {
		return err
	}

	if len(allHooks) == 0 {
		fmt.Println("No hooks found in .hero/hooks/.")
		fmt.Println("Create a .md file there — see skills/nl-event-hooks.md for the format.")
		return nil
	}

	for _, h := range allHooks {
		fmt.Printf("  %-30s  %-12s  match: %-30s  mode: %s\n", h.Name, h.Event, h.Match, h.Mode)
	}
	return nil
}

func runNLHookShow(cmd *cobra.Command, args []string) error {
	name := args[0]
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	allHooks, err := hooks.Discover(heroDir)
	if err != nil {
		return err
	}

	for _, h := range allHooks {
		if h.Name == name {
			data, _ := json.MarshalIndent(h, "", "  ")
			fmt.Println(string(data))
			fmt.Printf("\n--- Prompt ---\n%s\n", h.Body)
			return nil
		}
	}

	return fmt.Errorf("hook %q not found", name)
}

func runNLHookTest(cmd *cobra.Command, args []string) error {
	name := args[0]
	if nlhookFile == "" {
		return fmt.Errorf("--file is required")
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	h, err := findNLHook(heroDir, name)
	if err != nil {
		return err
	}

	relFile := nlhookFile
	if filepath.IsAbs(relFile) {
		relFile, _ = filepath.Rel(projectRoot, relFile)
	}

	if h.Matches(relFile) {
		fmt.Printf("✓ %q matches hook %q (pattern: %s)\n\n", relFile, h.Name, h.Match)
		fmt.Println("--- Rendered prompt ---")
		fmt.Println(h.Render(relFile))
	} else {
		fmt.Printf("✗ %q does not match hook %q (pattern: %s)\n", relFile, h.Name, h.Match)
	}
	return nil
}

func runNLHookFire(cmd *cobra.Command, args []string) error {
	name := args[0]
	if nlhookFile == "" {
		return fmt.Errorf("--file is required")
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	h, err := findNLHook(heroDir, name)
	if err != nil {
		return err
	}

	relFile := nlhookFile
	if filepath.IsAbs(relFile) {
		relFile, _ = filepath.Rel(projectRoot, relFile)
	}

	if !h.Matches(relFile) {
		// No match — exit silently (expected behavior for non-matching files)
		return nil
	}

	rendered := h.Render(relFile)

	// Emit as hook output JSON (additionalContext for Claude Code)
	output := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "PostToolUse",
			"additionalContext": rendered,
		},
	}
	data, _ := json.Marshal(output)
	fmt.Println(string(data))

	return nil
}

func findNLHook(heroDir, name string) (*hooks.Hook, error) {
	allHooks, err := hooks.Discover(heroDir)
	if err != nil {
		return nil, err
	}
	for _, h := range allHooks {
		if h.Name == name {
			return h, nil
		}
	}
	return nil, fmt.Errorf("hook %q not found in .hero/hooks/", name)
}

// Ensure nlhookFile doesn't shadow hookFile if it exists
var _ = os.Getenv // ensure os is used
