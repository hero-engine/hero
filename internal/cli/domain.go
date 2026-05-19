package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/install"
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

var domainVerifyJSON bool

var domainVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Report graph row counts grouped by the `domain` partition",
	Long: `Reports node and edge counts grouped by the domain namespace
column. Cross-domain edges (where endpoint domains differ) are
listed with their edge kind so the operator can audit boundary
crossings.

Use this after a schema-v3 migration to confirm every row landed
under 'engineering' (the migration default) and that no domain
leakage occurred. Pair with 'hero admin schema rollback v3
--dry-run' to see the non-engineering row count before reverting.`,
	RunE: runDomainVerify,
}

func init() {
	domainCmd.AddCommand(domainListCmd)
	domainCmd.AddCommand(domainSwitchCmd)
	domainCmd.AddCommand(domainVerifyCmd)
	domainVerifyCmd.Flags().BoolVar(&domainVerifyJSON, "json", false, "emit raw JSON instead of a human-readable summary")
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

	domainFS, err := hero.DomainFS(domain)
	if err != nil {
		return err
	}
	// Overlay domain on universal core so reinstall renders core +
	// domain merged. Domain wins on collisions.
	mergedFS := hero.OverlayFS(domainFS, hero.CoreFS())

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

	// Reinstall content for every harness currently installed in the
	// project so the new domain's agents/commands/skills materialize
	// immediately. .hero/ data (specs, knowledge, jobs) is untouched —
	// only the harness-rendered content layer is replaced.
	targets := install.DetectInstalledTargets(projectRoot)
	if len(targets) == 0 {
		fmt.Println("No installed harness detected — run 'hero install project . --target <tool>' to materialize content.")
		return nil
	}

	binaryVersion := rootCmd.Version
	if binaryVersion == "" {
		binaryVersion = "dev"
	}

	for _, target := range targets {
		fmt.Printf("Reinstalling %s with %s domain...\n", target, domain)
		opts := install.Options{
			ContentFS: mergedFS,
			Target:    target,
			Mode:      install.ModeProject,
			TargetDir: projectRoot,
			Force:     true,
			Version:   binaryVersion,
			Domain:    domain,
		}
		if _, err := install.Run(opts); err != nil {
			return fmt.Errorf("reinstalling %s: %w", target, err)
		}
	}
	return nil
}

func runDomainVerify(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	stats, err := store.DomainStats()
	if err != nil {
		return fmt.Errorf("computing domain stats: %w", err)
	}

	if domainVerifyJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(stats)
	}

	fmt.Println("Nodes by domain:")
	printCountMap(stats.NodesByDomain)
	fmt.Println()
	fmt.Println("Edges by domain:")
	printCountMap(stats.EdgesByDomain)

	if len(stats.CrossDomainEdges) > 0 {
		fmt.Println()
		fmt.Println("Cross-domain edges (from → to, by kind):")
		for _, g := range stats.CrossDomainEdges {
			fmt.Printf("  %s → %s  %s: %d\n", g.FromDomain, g.ToDomain, g.Kind, g.Count)
		}
	}
	return nil
}

func printCountMap(m map[string]int) {
	if len(m) == 0 {
		fmt.Println("  (none)")
		return
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		label := k
		if label == "" {
			label = "(global)"
		}
		fmt.Printf("  %-20s %d\n", label, m[k])
	}
}
