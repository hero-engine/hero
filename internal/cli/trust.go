package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var trustCmd = &cobra.Command{
	Use:   "trust <codex>",
	Short: "Show one-time harness permission guidance for Hero",
	Args:  cobra.ExactArgs(1),
	RunE:  runTrust,
}

func runTrust(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "codex":
		printCodexTrustHint()
		return nil
	default:
		return fmt.Errorf("unsupported trust target %q; supported targets: codex", args[0])
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
