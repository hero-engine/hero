package cli

import (
	"fmt"

	"github.com/hero-engine/hero/internal/install"
	"github.com/spf13/cobra"
)

var trustCmd = &cobra.Command{
	Use:   "trust <codex|claude>",
	Short: "Apply or show one-time harness permission setup for Hero",
	Args:  cobra.ExactArgs(1),
	RunE:  runTrust,
}

func runTrust(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "codex":
		printCodexTrustHint()
		return nil
	case "claude":
		return applyClaudeTrust()
	default:
		return fmt.Errorf("unsupported trust target %q; supported targets: codex, claude", args[0])
	}
}

func printCodexTrustHint() {
	fmt.Println()
	fmt.Println("Codex permissions: optional one-time setup")
	fmt.Println("Hero cannot grant Codex permissions itself; Codex owns the approval.")
	fmt.Println("To avoid repeated Hero CLI prompts, ask Codex:")
	fmt.Println()
	fmt.Println("  Please run `hero status` and request persistent approval for the `hero` command prefix.")
	fmt.Println()
	fmt.Println("You can show this again with `hero trust codex`.")
}

func applyClaudeTrust() error {
	projectRoot := findProjectRoot()
	if projectRoot == "" {
		return fmt.Errorf("could not resolve project root; run from inside a hero workspace")
	}

	added, path, err := install.EnsureClaudeHeroAllowlist(projectRoot)
	if err != nil {
		return fmt.Errorf("applying claude allowlist: %w", err)
	}

	fmt.Println()
	if added {
		fmt.Printf("Claude Code: added %s to %s\n", "Bash(hero:*)", path)
		fmt.Println("Claude Code will stop prompting for `hero` commands after it reloads settings.")
	} else {
		fmt.Printf("Claude Code: Bash(hero:*) already present in %s.\n", path)
		fmt.Println("No change needed.")
	}
	return nil
}
