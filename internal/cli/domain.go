package cli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/domains"
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
  hero domain enable qa          # enable bounded QA
  hero domain disable qa         # disable QA, preserve artifacts
  hero domain switch pm          # make PM primary`,
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

var domainEnableCmd = &cobra.Command{
	Use: "enable <domain>", Short: "Enable a bounded domain extension",
	Args: cobra.ExactArgs(1), RunE: runDomainEnable,
}

var domainDisableCmd = &cobra.Command{
	Use: "disable <domain>", Short: "Disable an extension without deleting artifacts",
	Args: cobra.ExactArgs(1), RunE: runDomainDisable,
}

var domainContentCmd = &cobra.Command{
	Use:   "content [stable-id]",
	Short: "List or load deeper content from enabled packs",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDomainContent,
}

var domainVerifyJSON bool
var domainInstallRunner = install.Run

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
	domainCmd.AddCommand(domainEnableCmd)
	domainCmd.AddCommand(domainDisableCmd)
	domainCmd.AddCommand(domainContentCmd)
	domainCmd.AddCommand(domainVerifyCmd)
	domainVerifyCmd.Flags().BoolVar(&domainVerifyJSON, "json", false, "emit raw JSON instead of a human-readable summary")
}

func runDomainContent(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	resolved, err := cfg.ResolveDomains()
	if err != nil {
		return err
	}
	_, manifest, err := hero.ComposeContent(toPublicComposition(resolved))
	if err != nil {
		return err
	}
	if len(args) == 0 {
		if len(manifest.DeferredEntries) == 0 {
			fmt.Println("No deferred content is available for the enabled composition.")
			return nil
		}
		fmt.Println("Bundled content available on demand:")
		for _, entry := range manifest.DeferredEntries {
			fmt.Printf("  %-48s owner=%-4s kind=%-8s local=true\n", entry.ID, entry.Owner, entry.Kind)
		}
		return nil
	}
	_, data, err := hero.ResolveDeferredContent(manifest, args[0])
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}

func runDomainShow(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	resolved, err := cfg.ResolveDomains()
	if err != nil {
		return err
	}
	fmt.Printf("Primary domain: %s\n", resolved.Primary)
	fmt.Printf("Extensions: %s\n", displayExtensions(resolved.Extensions))
	fmt.Println("Composition: ready (bundled, local)")
	return nil
}

func runDomainList(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	resolved, err := cfg.ResolveDomains()
	if err != nil {
		return err
	}
	fmt.Println("Bundled domains:")
	for _, pack := range domains.AvailablePacks() {
		state := "disabled"
		if pack.ID == resolved.Primary {
			state = "primary"
		} else if resolved.Contains(pack.ID) {
			state = "extension"
		}
		roles := make([]string, 0, len(pack.Roles))
		for _, role := range pack.Roles {
			roles = append(roles, string(role))
		}
		fmt.Printf("  %-12s state=%-9s roles=%-17s ready=%t bundled=%t\n", pack.ID, state, strings.Join(roles, ","), pack.Bundled, pack.Bundled)
	}
	return nil
}

func runDomainSwitch(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	current, err := cfg.ResolveDomains()
	if err != nil {
		return err
	}
	next := domains.ResolvedComposition{Primary: domains.DomainID(args[0])}
	for _, extension := range current.Extensions {
		if extension != next.Primary {
			next.Extensions = append(next.Extensions, extension)
		}
	}
	if err := applyDomainComposition(projectRoot, &cfg, current, next); err != nil {
		return err
	}
	fmt.Printf("Switched primary domain to %s; extensions: %s\n", next.Primary, displayExtensions(next.Extensions))
	return nil
}

func runDomainEnable(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	current, err := cfg.ResolveDomains()
	if err != nil {
		return err
	}
	id := domains.DomainID(args[0])
	if current.Primary == id {
		return fmt.Errorf("domain %q is already primary", id)
	}
	next := domains.ResolvedComposition{Primary: current.Primary, Extensions: append([]domains.DomainID(nil), current.Extensions...)}
	if !next.Contains(id) {
		next.Extensions = append(next.Extensions, id)
	}
	if err := applyDomainComposition(projectRoot, &cfg, current, next); err != nil {
		return err
	}
	fmt.Printf("Enabled %s extension; extensions: %s\n", id, displayExtensions(next.Extensions))
	return nil
}

func runDomainDisable(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	current, err := cfg.ResolveDomains()
	if err != nil {
		return err
	}
	id := domains.DomainID(args[0])
	if current.Primary == id {
		return fmt.Errorf("cannot disable primary domain %q; switch primary first", id)
	}
	next := domains.ResolvedComposition{Primary: current.Primary}
	found := false
	for _, extension := range current.Extensions {
		if extension == id {
			found = true
			continue
		}
		next.Extensions = append(next.Extensions, extension)
	}
	if !found {
		return fmt.Errorf("domain %q is not an enabled extension", id)
	}
	if err := applyDomainComposition(projectRoot, &cfg, current, next); err != nil {
		return err
	}
	fmt.Printf("Disabled %s extension; historical artifacts were preserved\n", id)
	return nil
}

func applyDomainComposition(projectRoot string, cfg *config.Config, current, next domains.ResolvedComposition) error {
	nextFS, _, err := hero.ComposeContent(toPublicComposition(next))
	if err != nil {
		return err
	}
	currentFS, _, err := hero.ComposeContent(toPublicComposition(current))
	if err != nil {
		return err
	}
	targets := unionTargets(install.PreviouslyInstalledTargets(projectRoot), install.DetectInstalledTargets(projectRoot))
	version := rootCmd.Version
	if version == "" {
		version = "dev"
	}
	for _, target := range targets {
		if _, err := domainInstallRunner(install.Options{ContentFS: nextFS, Target: target, Mode: install.ModeProject, TargetDir: projectRoot, Force: true, Version: version, Domain: string(next.Primary)}); err != nil {
			if rollbackErr := rollbackDomainTargets(projectRoot, currentFS, current.Primary, targets, version); rollbackErr != nil {
				return fmt.Errorf("reinstalling %s failed: %w; rollback also failed: %v", target, err, rollbackErr)
			}
			return fmt.Errorf("reinstalling %s failed; prior composition restored: %w", target, err)
		}
	}
	previous := *cfg
	if err := cfg.SetDomainComposition(next.Primary, next.Extensions); err != nil {
		_ = rollbackDomainTargets(projectRoot, currentFS, current.Primary, targets, version)
		return err
	}
	if err := cfg.Save(projectRoot); err != nil {
		*cfg = previous
		if rollbackErr := rollbackDomainTargets(projectRoot, currentFS, current.Primary, targets, version); rollbackErr != nil {
			return fmt.Errorf("saving composition failed: %w; rollback also failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("saving composition failed; prior rendered state restored: %w", err)
	}
	return nil
}

func rollbackDomainTargets(projectRoot string, content fs.FS, primary domains.DomainID, targets []install.Target, version string) error {
	var failures []string
	for _, target := range targets {
		if _, err := domainInstallRunner(install.Options{ContentFS: content, Target: target, Mode: install.ModeProject, TargetDir: projectRoot, Force: true, Version: version, Domain: string(primary)}); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", target, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

func toPublicComposition(resolved domains.ResolvedComposition) hero.DomainComposition {
	out := hero.DomainComposition{Primary: string(resolved.Primary)}
	for _, extension := range resolved.Extensions {
		out.Extensions = append(out.Extensions, string(extension))
	}
	return out
}

func displayExtensions(ids []domains.DomainID) string {
	if len(ids) == 0 {
		return "(none)"
	}
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, string(id))
	}
	return strings.Join(values, ", ")
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
