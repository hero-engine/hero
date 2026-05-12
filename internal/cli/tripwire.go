package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var tripwireCmd = &cobra.Command{
	Use:   "tripwire",
	Short: "Manage tripwires — forbidden-option guardrails",
}

var tripwireListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active tripwires",
	RunE:  runTripwireList,
}

var tripwireCheckCmd = &cobra.Command{
	Use:   "check <text>",
	Short: "Check if text matches any tripwire triggers",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runTripwireCheck,
}

func init() {
	tripwireCmd.AddCommand(tripwireListCmd)
	tripwireCmd.AddCommand(tripwireCheckCmd)
}

func runTripwireList(cmd *cobra.Command, args []string) error {
	heroDir := filepath.Join(findProjectRoot(), ".hero")

	idx, err := index.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	tripwires, err := idx.FindAllTripwires(heroDir)
	if err != nil {
		return fmt.Errorf("loading tripwires: %w", err)
	}

	if len(tripwires) == 0 {
		fmt.Println("No active tripwires.")
		fmt.Println("\nCreate one at .hero/knowledge/tripwires/<slug>/spec.md with type: tripwire")
		return nil
	}

	for _, tw := range tripwires {
		fmt.Printf("%-30s [%s]  %s\n", tw.Slug, tw.Severity, tw.Title)
		if len(tw.Triggers) > 0 {
			fmt.Printf("  triggers: %s\n", strings.Join(tw.Triggers, ", "))
		}
		if tw.Constraint != "" {
			lines := strings.SplitN(tw.Constraint, "\n", 2)
			fmt.Printf("  constraint: %s\n", lines[0])
		}
	}

	fmt.Printf("\n%d tripwire(s)\n", len(tripwires))
	return nil
}

func runTripwireCheck(cmd *cobra.Command, args []string) error {
	heroDir := filepath.Join(findProjectRoot(), ".hero")

	text := strings.Join(args, " ")

	idx, err := index.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	matched, err := idx.FindTripwiresByTrigger(text)
	if err != nil {
		return fmt.Errorf("checking triggers: %w", err)
	}

	if len(matched) == 0 {
		fmt.Println("No tripwire triggers matched.")
		return nil
	}

	fmt.Printf("TRIPWIRE MATCH — %d tripwire(s) triggered:\n\n", len(matched))
	for _, tw := range matched {
		fmt.Printf("  %s [%s]: %s\n", tw.Slug, tw.Severity, tw.Title)
		if tw.Constraint != "" {
			fmt.Printf("    Constraint: %s\n", tw.Constraint)
		}
		if tw.Instead != "" {
			fmt.Printf("    Instead: %s\n", tw.Instead)
		}
		fmt.Println()
	}

	os.Exit(1)
	return nil
}

// Ensure tripwire specs can be discovered from the knowledge directory.
var _ = spec.TypeTripwire
