package cli

import (
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Show and manage model role configuration",
	Long: `Display configured model roles or check that role-model mappings are valid.

Model roles let you assign specific AI models to each phase of the Hero workflow:
  design     — architect agents (e.g. a reasoning model for complex design work)
  execution  — engineer agents (e.g. a fast, cost-effective coding model)
  review     — reviewer agents (e.g. a high-capability model for critical review)

Configure in hero.json:
  "models": {
    "roles": {
      "design":    "claude-opus-4",
      "execution": "claude-sonnet-4-5",
      "review":    "o3"
    },
    "default_model": "claude-sonnet-4-5"
  }

Role hints are passed to agents as context; the actual model selection depends
on your AI tool's configuration.`,
	RunE: runModels,
}

var modelsCheckFlag bool

func init() {
	modelsCmd.Flags().BoolVar(&modelsCheckFlag, "check", false, "verify that model roles are configured")
}

func runModels(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Models == nil || (len(cfg.Models.Roles) == 0 && cfg.Models.DefaultModel == "") {
		if modelsCheckFlag {
			fmt.Fprintf(os.Stderr, "No model roles configured. Add a models section to .hero/hero.json.\n")
			os.Exit(1)
		}
		fmt.Println("No model roles configured.")
		fmt.Println()
		fmt.Println("Add to .hero/hero.json:")
		fmt.Println(`  "models": {`)
		fmt.Println(`    "roles": {`)
		fmt.Println(`      "design":    "<model-id>",`)
		fmt.Println(`      "execution": "<model-id>",`)
		fmt.Println(`      "review":    "<model-id>"`)
		fmt.Println(`    }`)
		fmt.Println(`  }`)
		return nil
	}

	fmt.Println("Model role configuration:")
	fmt.Println()

	knownRoles := []string{"design", "execution", "review"}
	for _, role := range knownRoles {
		model := cfg.Models.Roles[role]
		if model == "" {
			model = "(not set)"
		}
		fmt.Printf("  %-12s %s\n", role+":", model)
	}

	// Show any extra roles defined
	for role, model := range cfg.Models.Roles {
		isKnown := false
		for _, k := range knownRoles {
			if role == k {
				isKnown = true
				break
			}
		}
		if !isKnown {
			fmt.Printf("  %-12s %s\n", role+":", model)
		}
	}

	if cfg.Models.DefaultModel != "" {
		fmt.Printf("\n  %-12s %s\n", "default:", cfg.Models.DefaultModel)
	}

	if modelsCheckFlag {
		missing := []string{}
		for _, role := range knownRoles {
			if cfg.Models.Roles[role] == "" {
				missing = append(missing, role)
			}
		}
		if len(missing) > 0 {
			fmt.Printf("\nWarning: no model configured for roles: %v\n", missing)
		} else {
			fmt.Println("\nAll standard roles are configured.")
		}
	}

	return nil
}
