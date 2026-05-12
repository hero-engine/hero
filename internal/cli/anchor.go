package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/mission"
	"github.com/spf13/cobra"
)

var anchorCmd = &cobra.Command{
	Use:   "anchor [context]",
	Short: "Re-anchor on project first principles — mission + tripwires",
	Long: `Print the project mission statement and all active tripwires.
Optionally provide context text to highlight relevant tripwires
whose triggers match your current decision context.`,
	RunE: runAnchor,
}

func runAnchor(cmd *cobra.Command, args []string) error {
	heroDir := filepath.Join(findProjectRoot(), ".hero")

	ctx := strings.Join(args, " ")

	// Load mission
	m, _ := mission.LoadFile(heroDir)
	if m != nil && m.MissionStatement != "" {
		fmt.Println("## Mission")
		fmt.Println()
		fmt.Println(m.MissionStatement)
		fmt.Println()
	}

	// Load tripwires
	idx, err := index.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	tripwires, err := idx.FindAllTripwires(heroDir)
	if err != nil {
		return fmt.Errorf("loading tripwires: %w", err)
	}

	// Check for trigger matches if context provided
	var highlighted map[string]bool
	if ctx != "" {
		matched, _ := idx.FindTripwiresByTrigger(ctx)
		if len(matched) > 0 {
			highlighted = make(map[string]bool)
			for _, tw := range matched {
				highlighted[tw.Slug] = true
			}
		}
	}

	if len(tripwires) > 0 {
		fmt.Println("## Tripwires (Do Not Violate)")
		fmt.Println()

		if len(highlighted) > 0 {
			fmt.Println("### Relevant to your context")
			fmt.Println()
			for _, tw := range tripwires {
				if !highlighted[tw.Slug] {
					continue
				}
				printTripwire(tw)
			}
			fmt.Println("### All active tripwires")
			fmt.Println()
		}

		for _, tw := range tripwires {
			if highlighted[tw.Slug] {
				continue
			}
			printTripwire(tw)
		}
	} else {
		fmt.Println("No active tripwires defined.")
	}

	if m != nil && m.MissionFitTest != "" {
		fmt.Println()
		fmt.Println("## Mission-Fit Test")
		fmt.Println()
		fmt.Println(m.MissionFitTest)
	}

	return nil
}

func printTripwire(tw index.TripwireResult) {
	fmt.Printf("  %s [%s]: %s\n", tw.Slug, tw.Severity, tw.Title)
	if tw.Constraint != "" {
		fmt.Printf("    Constraint: %s\n", tw.Constraint)
	}
	if tw.Why != "" {
		fmt.Printf("    Why: %s\n", tw.Why)
	}
	if tw.Instead != "" {
		fmt.Printf("    Instead: %s\n", tw.Instead)
	}
	fmt.Println()
}
