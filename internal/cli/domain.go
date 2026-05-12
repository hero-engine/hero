package cli

import (
	"fmt"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

var domainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Show or switch the active domain",
	Long: `Show the current domain or list available domains.

Examples:
  hero domain                    # show current domain
  hero domain list               # list available domains
  hero domain switch sales       # switch to sales domain`,
	RunE: runDomainShow,
}

var domainListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available domains",
	RunE:  runDomainList,
}

var domainSwitchCmd = &cobra.Command{
	Use:   "switch <domain>",
	Short: "Switch the active domain",
	Args:  cobra.ExactArgs(1),
	RunE:  runDomainSwitch,
}

func init() {
	domainCmd.AddCommand(domainListCmd)
	domainCmd.AddCommand(domainSwitchCmd)
}

func runDomainShow(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	domain := cfg.Domain
	if domain == "" {
		domain = "engineering"
	}
	fmt.Printf("Active domain: %s\n", domain)
	return nil
}

func runDomainList(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, _ := config.Load(projectRoot)

	active := cfg.Domain
	if active == "" {
		active = "engineering"
	}

	fmt.Println("Available domains:")
	for _, d := range hero.AvailableDomains() {
		marker := "  "
		if d == active {
			marker = "* "
		}
		fmt.Printf("  %s%s\n", marker, d)
	}
	return nil
}

func runDomainSwitch(cmd *cobra.Command, args []string) error {
	domain := args[0]

	// Validate domain exists
	if _, err := hero.DomainFS(domain); err != nil {
		return err
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	cfg.Domain = domain
	if err := cfg.Save(projectRoot); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Switched to domain: %s\n", domain)
	fmt.Println("Run 'hero install project . --target <tool>' to reinstall content.")
	return nil
}
