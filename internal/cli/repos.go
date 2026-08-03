package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/cli/prompt"
	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

// readPeerIDFromRepo reads the peer_id from a sibling workspace's
// hero.json. Returns "" if the file is missing or has no peer_id.
func readPeerIDFromRepo(absPath, heroFolder string) string {
	path := filepath.Join(absPath, heroFolder, "hero.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var shape struct {
		PeerID string `json:"peer_id"`
	}
	if json.Unmarshal(data, &shape) != nil {
		return ""
	}
	return shape.PeerID
}

// recordPeerMeta upserts the peer_id (+scanned_at) for an alias into
// cfg.RepoMeta. Caller is responsible for persisting cfg.
func recordPeerMeta(cfg *config.Config, alias, peerID string) {
	if peerID == "" {
		return
	}
	if cfg.RepoMeta == nil {
		cfg.RepoMeta = make(map[string]config.RepoMetaEntry)
	}
	cfg.RepoMeta[alias] = config.RepoMetaEntry{
		PeerID:    peerID,
		ScannedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "Manage cross-repo project registry",
	Long: `Configure and discover sibling repositories so Hero can resolve
cross-repo spec dependencies, pull in conventions from other repos,
and detect drift across repo boundaries.

Subcommands:

  hero repos              — list configured repos and their status
  hero repos add          — register a repo alias
  hero repos remove       — unregister a repo alias
  hero repos scan         — auto-discover nearby hero workspaces
  hero repos check        — validate all configured repo paths`,
	RunE: runReposList,
}

var (
	reposScanPaths []string
	reposScanDepth int
	reposScanAuto  bool
	reposLocal     bool
)

var reposAddCmd = &cobra.Command{
	Use:   "add <alias> <path>",
	Short: "Register a repo alias → path mapping",
	Long: `Register a sibling repository so specs can reference it in relations.

Examples:
  hero repos add auth-service ../auth-service
  hero repos add shared-libs /Users/dev/projects/shared-libs

Use --local to write to hero.local.json (gitignored) instead of hero.json.
This is useful when your local directory layout differs from teammates.`,
	Args: promptableArgs(2, cobra.ExactArgs(2)),
	RunE: runReposAdd,
}

var reposRemoveCmd = &cobra.Command{
	Use:   "remove <alias>",
	Short: "Unregister a repo alias",
	Args:  cobra.ExactArgs(1),
	RunE:  runReposRemove,
}

var reposScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Auto-discover nearby hero workspaces",
	Long: `Walks sibling directories (and any --search-path directories) looking
for directories that contain a .hero/ folder. Discovered repos are shown
with suggested aliases derived from the directory name.

Run with no flags to scan the parent directory (sibling repos).

Examples:
  hero repos scan                                    # scan siblings
  hero repos scan --search-path ~/projects           # scan a custom path
  hero repos scan --search-path ~/work --depth 2     # scan deeper`,
	RunE: runReposScan,
}

var reposCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate all configured repo paths are accessible",
	RunE:  runReposCheck,
}

func init() {
	reposAddCmd.Flags().BoolVar(&reposLocal, "local", false, "write to hero.local.json instead of hero.json")
	reposScanCmd.Flags().StringSliceVar(&reposScanPaths, "search-path", nil, "additional directories to scan for hero workspaces")
	reposScanCmd.Flags().IntVar(&reposScanDepth, "depth", 1, "how many directory levels deep to scan")
	reposScanCmd.Flags().BoolVar(&reposScanAuto, "auto", false, "automatically register all discovered repos")

	reposCmd.AddCommand(reposAddCmd)
	reposCmd.AddCommand(reposRemoveCmd)
	reposCmd.AddCommand(reposScanCmd)
	reposCmd.AddCommand(reposCheckCmd)
}

func runReposList(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Repos) == 0 {
		fmt.Println("No repos configured. Run 'hero repos scan' to discover nearby workspaces,")
		fmt.Println("or 'hero repos add <alias> <path>' to register one manually.")
		return nil
	}

	statuses := cfg.ResolveAllRepos(projectRoot)

	// Sort by alias
	aliases := make([]string, 0, len(statuses))
	for a := range statuses {
		aliases = append(aliases, a)
	}
	sort.Strings(aliases)

	for _, alias := range aliases {
		s := statuses[alias]
		peerInfo := ""
		if meta, ok := cfg.RepoMeta[alias]; ok && meta.PeerID != "" {
			peerInfo = "  peer_id=" + meta.PeerID
		} else if s.Accessible {
			// Try lazily reading the sibling's peer_id so the list
			// reflects current ground truth even for repos that were
			// registered before peer_id existed.
			if id := readPeerIDFromRepo(s.Path, cfg.Folder); id != "" {
				peerInfo = "  peer_id=" + id
			}
		}
		if s.Accessible {
			fmt.Printf("  ✓ %-25s  %s%s\n", alias, s.Path, peerInfo)
		} else {
			fmt.Printf("  ✗ %-25s  %s  (%s)%s\n", alias, s.Path, s.Error, peerInfo)
		}
	}
	return nil
}

func reposAddArgs(cmd *cobra.Command, args []string) (string, string, error) {
	alias, repoPath := "", ""
	if len(args) > 0 {
		alias = args[0]
	}
	if len(args) > 1 {
		repoPath = args[1]
	}
	if alias == "" {
		value, err := prompt.Prompt(cmd.InOrStdin(), cmd.OutOrStdout(), "Repo alias: ")
		if err != nil {
			return "", "", err
		}
		if value == "" {
			return "", "", errors.New("alias is required")
		}
		alias = value
	}
	if repoPath == "" {
		value, err := prompt.Prompt(cmd.InOrStdin(), cmd.OutOrStdout(), "Path to the repo: ")
		if err != nil {
			return "", "", err
		}
		if value == "" {
			return "", "", errors.New("path is required")
		}
		repoPath = value
	}
	return alias, repoPath, nil
}

func runReposAdd(cmd *cobra.Command, args []string) error {
	alias, repoPath, err := reposAddArgs(cmd, args)
	if err != nil {
		return err
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Validate the path
	absPath := repoPath
	if !filepath.IsAbs(repoPath) {
		absPath = filepath.Join(projectRoot, repoPath)
	}
	absPath, _ = filepath.Abs(absPath)
	heroDir := filepath.Join(absPath, cfg.Folder)
	if info, err := os.Stat(heroDir); err != nil || !info.IsDir() {
		fmt.Printf("Warning: %s does not contain a .hero/ directory.\n", absPath)
		fmt.Printf("The repo will be registered but cross-repo features won't work until it's initialized.\n\n")
	}

	peerID := readPeerIDFromRepo(absPath, cfg.Folder)

	if reposLocal {
		// Write to hero.local.json
		localCfg, err := config.LoadLocal(projectRoot, cfg.Folder)
		if err != nil {
			return fmt.Errorf("loading local config: %w", err)
		}
		if localCfg.Repos == nil {
			localCfg.Repos = make(map[string]string)
		}
		localCfg.Repos[alias] = repoPath
		recordPeerMeta(&localCfg, alias, peerID)
		if err := config.SaveLocal(projectRoot, cfg.Folder, localCfg); err != nil {
			return fmt.Errorf("saving local config: %w", err)
		}
		fmt.Printf("Added %s → %s (in hero.local.json, gitignored)\n", alias, repoPath)
	} else {
		if cfg.Repos == nil {
			cfg.Repos = make(map[string]string)
		}
		cfg.Repos[alias] = repoPath
		recordPeerMeta(&cfg, alias, peerID)
		if err := cfg.Save(projectRoot); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Printf("Added %s → %s (in hero.json)\n", alias, repoPath)
	}
	if peerID != "" {
		fmt.Printf("  peer_id %s recorded\n", peerID)
	}

	return nil
}

func runReposRemove(cmd *cobra.Command, args []string) error {
	alias := args[0]

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if _, ok := cfg.Repos[alias]; !ok {
		return fmt.Errorf("repo alias %q not configured", alias)
	}

	delete(cfg.Repos, alias)
	if err := cfg.Save(projectRoot); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("Removed repo alias %q\n", alias)
	return nil
}

func runReposScan(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Build search roots: always include parent directory, plus any --search-path
	searchRoots := []string{filepath.Dir(projectRoot)}
	for _, sp := range reposScanPaths {
		abs, err := filepath.Abs(sp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  warning: skipping %s: %v\n", sp, err)
			continue
		}
		searchRoots = append(searchRoots, abs)
	}

	// Deduplicate
	seen := make(map[string]bool)
	var uniqueRoots []string
	for _, r := range searchRoots {
		if !seen[r] {
			seen[r] = true
			uniqueRoots = append(uniqueRoots, r)
		}
	}

	absProjectRoot, _ := filepath.Abs(projectRoot)
	var raw []repoDiscovery

	for _, root := range uniqueRoots {
		scanDir(root, reposScanDepth, absProjectRoot, cfg.Folder, &raw)
	}

	// Deduplicate by absolute path
	seenPath := make(map[string]bool)
	var found []repoDiscovery
	for _, d := range raw {
		if !seenPath[d.path] {
			seenPath[d.path] = true
			found = append(found, d)
		}
	}

	if len(found) == 0 {
		fmt.Println("No hero workspaces discovered nearby.")
		fmt.Printf("Searched: %s\n", strings.Join(uniqueRoots, ", "))
		return nil
	}

	// Filter out already-configured repos
	existing := make(map[string]bool, len(cfg.Repos))
	for _, p := range cfg.Repos {
		abs := p
		if !filepath.IsAbs(p) {
			abs = filepath.Join(projectRoot, p)
		}
		abs, _ = filepath.Abs(abs)
		existing[abs] = true
	}

	var newRepos []repoDiscovery
	for _, d := range found {
		if !existing[d.path] {
			newRepos = append(newRepos, d)
		}
	}

	if len(newRepos) == 0 {
		fmt.Printf("Found %d hero workspace(s), all already configured.\n", len(found))
		return nil
	}

	if reposScanAuto {
		// Auto-register all discovered repos
		if cfg.Repos == nil {
			cfg.Repos = make(map[string]string)
		}
		for _, d := range newRepos {
			cfg.Repos[d.alias] = d.rel
			recordPeerMeta(&cfg, d.alias, d.peerID)
			if d.peerID != "" {
				fmt.Printf("  ✓ %s → %s  peer_id=%s\n", d.alias, d.rel, d.peerID)
			} else {
				fmt.Printf("  ✓ %s → %s  (no peer_id — peer predates peering)\n", d.alias, d.rel)
			}
		}
		if err := cfg.Save(projectRoot); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Printf("\nRegistered %d repo(s) in hero.json.\n", len(newRepos))
		return nil
	}

	fmt.Printf("Discovered %d new hero workspace(s):\n\n", len(newRepos))
	for _, d := range newRepos {
		if d.peerID != "" {
			fmt.Printf("  %-25s  %-30s  peer_id=%s\n", d.alias, d.rel, d.peerID)
		} else {
			fmt.Printf("  %-25s  %-30s  (no peer_id)\n", d.alias, d.rel)
		}
	}
	fmt.Printf("\nTo register them all at once:\n\n")
	fmt.Printf("  hero repos scan --auto\n\n")
	fmt.Printf("Or individually:\n\n")
	for _, d := range newRepos {
		fmt.Printf("  hero repos add %s %s\n", d.alias, d.rel)
	}
	fmt.Println()

	return nil
}

type repoDiscovery struct {
	alias  string
	path   string
	rel    string
	peerID string
}

func scanDir(root string, maxDepth int, skipPath, heroFolder string, found *[]repoDiscovery) {
	if maxDepth < 0 {
		return
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}

	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}

		dir := filepath.Join(root, e.Name())
		absDir, _ := filepath.Abs(dir)

		// Skip the current project
		if absDir == skipPath {
			continue
		}

		// Check for .hero/ folder
		heroDir := filepath.Join(dir, heroFolder)
		if info, err := os.Stat(heroDir); err == nil && info.IsDir() {
			// Calculate relative path from the project being skipped
			rel, _ := filepath.Rel(filepath.Dir(skipPath), absDir)
			if rel == "" {
				rel = absDir
			}
			alias := filepath.Base(absDir)
			peerID := readPeerIDFromRepo(absDir, heroFolder)
			*found = append(*found, repoDiscovery{alias, absDir, rel, peerID})
		}

		// Recurse if depth allows
		if maxDepth > 1 {
			scanDir(dir, maxDepth-1, skipPath, heroFolder, found)
		}
	}
}

func runReposCheck(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Repos) == 0 {
		fmt.Println("No repos configured.")
		return nil
	}

	statuses := cfg.ResolveAllRepos(projectRoot)
	allOK := true

	aliases := make([]string, 0, len(statuses))
	for a := range statuses {
		aliases = append(aliases, a)
	}
	sort.Strings(aliases)

	for _, alias := range aliases {
		s := statuses[alias]
		if s.Accessible {
			fmt.Printf("  ✓ %s\n", alias)
		} else {
			fmt.Printf("  ✗ %s — %s\n", alias, s.Error)
			allOK = false
		}
	}

	if !allOK {
		fmt.Println("\nSome repos are inaccessible. Fix paths with:")
		fmt.Println("  hero repos add <alias> <correct-path> --local")
		os.Exit(1)
	}

	return nil
}
