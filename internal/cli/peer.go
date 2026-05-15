package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	contractpeering "github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/peering"
	"github.com/hero-engine/hero/internal/spec"
)

// `hero peer` — umbrella for cross-repo peering CLI surfaces beyond the
// handoff lifecycle (which lives under `hero handoff`). Subcommands:
//
//	hero peer manifest [--out <path>]   regenerate the peer manifest
//	hero peer list                       configured peers + status snapshot
//	hero peer show <alias>               detailed view of one peer
//	hero peer call <alias> --mode=...   sync peer call (advisory | spec-out)

var peerCmd = &cobra.Command{
	Use:   "peer",
	Short: "Cross-repo peering — manifest, list, show, sync calls",
	Long: `Cross-repo peering operations beyond async handoff.

Subcommands:
  manifest      Regenerate this workspace's peer-manifest.yaml
  list          List configured peers and their reachability + reciprocity
  show          Inspect one peer (manifest contents, in-flight handoffs)
  call          Sync peer call — spawn a subagent in the peer's workspace`,
}

// --- hero peer manifest -----------------------------------------------

var peerManifestOut string

var peerManifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "Regenerate this workspace's peer manifest",
	Long: `Generates .hero/peer-manifest.yaml from the local conventions and
hero.json peering config. ` + "`hero index`" + ` runs this automatically; use this
command directly when you've edited a convention's peer-surface flag
and want to refresh without a full reindex.

With --out, writes to a custom path instead of the default
.hero/peer-manifest.yaml.`,
	RunE: runPeerManifest,
}

// --- hero peer list ---------------------------------------------------

var peerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured peers with reachability + reciprocity check",
	RunE:  runPeerList,
}

// --- hero peer show ---------------------------------------------------

var peerShowCmd = &cobra.Command{
	Use:   "show <alias>",
	Short: "Inspect one configured peer in detail",
	Args:  cobra.ExactArgs(1),
	RunE:  runPeerShow,
}

// --- hero peer call ---------------------------------------------------

var (
	peerCallMode         string
	peerCallBudgetTurns  int
	peerCallBudgetTokens int
	peerCallRelatedSpec  string
	peerCallReason       string
	peerCallDryRun       bool
)

var peerCallCmd = &cobra.Command{
	Use:   "call <alias> \"<prompt>\"",
	Short: "Sync peer call: spawn a subagent in the peer workspace",
	Long: `Spawns a subagent in the named peer's workspace and runs the given
prompt with that workspace's Hero context loaded.

Modes:
  --mode=advisory    Investigate and return findings. No writes on the
                     peer side. Default budget: ` + fmt.Sprintf("%d turns / %d tokens", peering.DefaultAdvisoryTurns, peering.DefaultAdvisoryTokens) + `.
  --mode=spec-out    Run the peer's /design flow under the subagent.
                     The subagent writes a planning-status spec on the
                     peer side with a received_from block. Default budget:
                     ` + fmt.Sprintf("%d turns / %d tokens", peering.DefaultSpecOutTurns, peering.DefaultSpecOutTokens) + `.

Examples:
  hero peer call app --mode=advisory "What's your error envelope shape?"
  hero peer call app --mode=spec-out --related-spec=order-failure-display "Design the fix"
  hero peer call app --mode=advisory --dry-run "..."   # show envelope only

The subagent CLI is configured in hero.json under peering.subagent
(default: claude).`,
	Args: cobra.MinimumNArgs(2),
	RunE: runPeerCall,
}

func init() {
	peerManifestCmd.Flags().StringVar(&peerManifestOut, "out", "", "alternate output path (default: .hero/peer-manifest.yaml)")

	peerCallCmd.Flags().StringVar(&peerCallMode, "mode", "advisory", "call mode: advisory | spec-out")
	peerCallCmd.Flags().IntVar(&peerCallBudgetTurns, "budget-turns", 0, "max turns the subagent may consume (0 = default per mode)")
	peerCallCmd.Flags().IntVar(&peerCallBudgetTokens, "budget-tokens", 0, "max tokens the subagent may consume (0 = default per mode)")
	peerCallCmd.Flags().StringVar(&peerCallRelatedSpec, "related-spec", "", "originator-side spec slug to record the call against")
	peerCallCmd.Flags().StringVar(&peerCallReason, "reason", "", "rationale for the call (recorded in trail entry)")
	peerCallCmd.Flags().BoolVar(&peerCallDryRun, "dry-run", false, "print the envelope that would be sent without dispatching the subagent")

	peerCmd.AddCommand(peerManifestCmd)
	peerCmd.AddCommand(peerListCmd)
	peerCmd.AddCommand(peerShowCmd)
	peerCmd.AddCommand(peerCallCmd)
}

func runPeerManifest(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	manifest, err := peering.GenerateManifest(projectRoot)
	if err != nil {
		return err
	}
	heroDir := cfg.HeroDir(projectRoot)
	target := peerManifestOut
	if target == "" {
		target = filepath.Join(heroDir, peering.PeerManifestFileName)
	}
	// Use WriteManifest's atomicity when writing to the default
	// location. For a custom --out path, serialize directly.
	if target == filepath.Join(heroDir, peering.PeerManifestFileName) {
		if err := peering.WriteManifest(heroDir, manifest); err != nil {
			return err
		}
	} else {
		data, err := yaml.Marshal(manifest)
		if err != nil {
			return fmt.Errorf("marshal manifest: %w", err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Peer manifest written to %s (%d conventions)\n",
		target, len(manifest.Conventions))
	return nil
}

func runPeerList(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if len(cfg.Repos) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No peers configured. Use `hero repos add <alias> <path>`.")
		return nil
	}
	aliases := make([]string, 0, len(cfg.Repos))
	for a := range cfg.Repos {
		aliases = append(aliases, a)
	}
	sort.Strings(aliases)

	// Pre-load local specs once for the in-flight counter.
	heroDir := cfg.HeroDir(projectRoot)
	specs, _ := spec.Discover(heroDir)

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "%-16s %-38s %-9s %-9s %-7s %s\n",
		"ALIAS", "PEER_ID", "REACHABLE", "MANIFEST", "CONV", "IN-FLIGHT")
	for _, alias := range aliases {
		peerPath, _ := cfg.ResolveRepoPath(projectRoot, alias)
		reachable := "no"
		manifestPresent := "no"
		conv := "-"
		peerID := "-"
		if info, err := os.Stat(peerPath); err == nil && info.IsDir() {
			reachable = "yes"
			m, mErr := peering.ReadPeerManifest(peerPath, cfg.Folder)
			if mErr == nil && m != nil {
				manifestPresent = "yes"
				conv = fmt.Sprintf("%d", len(m.Conventions))
				if m.Repo.PeerID != "" {
					peerID = m.Repo.PeerID
				}
			}
		}
		// Fall back to recorded peer_id from cfg.RepoMeta if we
		// couldn't read the manifest.
		if peerID == "-" {
			if meta, ok := cfg.RepoMeta[alias]; ok && meta.PeerID != "" {
				peerID = meta.PeerID
			}
		}

		inFlight := countInFlightTowards(specs, alias)
		fmt.Fprintf(w, "%-16s %-38s %-9s %-9s %-7s %d\n",
			alias, peerID, reachable, manifestPresent, conv, inFlight)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Use `hero peer show <alias>` for detail.")
	return nil
}

// countInFlightTowards returns the number of local specs whose latest
// outgoing trail entry targets the given peer alias (display form).
func countInFlightTowards(specs []*spec.Spec, alias string) int {
	n := 0
	for _, s := range specs {
		if s.Status != spec.StatusHandedOff && s.Status != spec.StatusAwaitingPeer {
			continue
		}
		entries, err := peering.ReadTrail(s.Path)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.Direction == contractpeering.DirectionOut && e.PeerAliasDisplay == alias {
				n++
				break
			}
		}
	}
	return n
}

func runPeerShow(cmd *cobra.Command, args []string) error {
	alias := args[0]
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if _, ok := cfg.Repos[alias]; !ok {
		return fmt.Errorf("peer %q not configured — run `hero repos add %s <path>`", alias, alias)
	}
	peerPath, err := cfg.ResolveRepoPath(projectRoot, alias)
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Peer: %s\n", alias)
	fmt.Fprintf(w, "  path: %s\n", peerPath)

	manifest, mErr := peering.ReadPeerManifest(peerPath, cfg.Folder)
	if mErr != nil {
		fmt.Fprintf(w, "  manifest: ERROR — %v\n", mErr)
	} else if manifest != nil {
		fmt.Fprintf(w, "  peer_id: %s\n", manifest.Repo.PeerID)
		if manifest.Repo.Display != "" {
			fmt.Fprintf(w, "  display: %s\n", manifest.Repo.Display)
		}
		if manifest.Repo.ScopeHint != "" {
			fmt.Fprintf(w, "  scope_hint: %s\n", manifest.Repo.ScopeHint)
		}
		fmt.Fprintf(w, "  generated_at: %s\n", manifest.GeneratedAt.Format("2006-01-02 15:04:05Z07"))
		fmt.Fprintf(w, "  conventions: %d\n", len(manifest.Conventions))
		for _, c := range manifest.Conventions {
			surface := strings.Join(c.Surface, ",")
			fmt.Fprintf(w, "    - %s  %s  (surface: %s)\n", c.Slug, c.Title, surface)
		}
		if manifest.Contracts != nil && len(manifest.Contracts.Shapes) > 0 {
			fmt.Fprintf(w, "  contracts (%s v%d):\n", manifest.Contracts.Package, manifest.Contracts.Version)
			for _, s := range manifest.Contracts.Shapes {
				fmt.Fprintf(w, "    - %s  %s  → %s\n", s.Kind, s.GoSymbol, s.Convention)
			}
		}
	}

	// Reciprocity check: does the peer list us in its repos?
	reciprocal := "unknown"
	peerCfgPath := filepath.Join(peerPath, cfg.Folder, "hero.json")
	if peerCfg, err := readPeerHeroJSON(peerCfgPath); err == nil {
		reciprocal = "no"
		for _, p := range peerCfg.Repos {
			abs := p
			if !filepath.IsAbs(p) {
				abs = filepath.Join(peerPath, p)
			}
			absResolved, _ := filepath.Abs(abs)
			absMe, _ := filepath.Abs(projectRoot)
			if absResolved == absMe {
				reciprocal = "yes"
				break
			}
		}
	}
	fmt.Fprintf(w, "  reciprocal: %s\n", reciprocal)

	// In-flight specs targeting this peer.
	heroDir := cfg.HeroDir(projectRoot)
	specs, _ := spec.Discover(heroDir)
	count := countInFlightTowards(specs, alias)
	fmt.Fprintf(w, "  in-flight handoffs/calls to this peer: %d\n", count)

	return nil
}

// minimalPeerHeroJSON is the subset of hero.json we need to perform
// reciprocity checks without dragging the full Config loader's
// auto-migration semantics through a read-only inspection path.
type minimalPeerHeroJSON struct {
	PeerID string            `json:"peer_id"`
	Repos  map[string]string `json:"repos"`
}

func readPeerHeroJSON(path string) (*minimalPeerHeroJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m minimalPeerHeroJSON
	// Use yaml because it parses JSON too and is already imported by
	// this package via the manifest writer.
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func runPeerCall(cmd *cobra.Command, args []string) error {
	alias := args[0]
	// Everything after the alias is the prompt — join remaining args
	// with a space so quoted prompts and unquoted prompts both work.
	prompt := strings.Join(args[1:], " ")

	mode := contractpeering.PeerCallMode(peerCallMode)
	switch mode {
	case contractpeering.PeerCallAdvisory, contractpeering.PeerCallSpecOut:
		// ok
	case contractpeering.PeerCallFull:
		return fmt.Errorf("mode=full is deferred to v2")
	default:
		return fmt.Errorf("unknown mode %q — use advisory or spec-out", peerCallMode)
	}

	projectRoot := findProjectRoot()
	res, err := peering.Call(projectRoot, peering.CallOptions{
		PeerAlias: alias,
		Mode:      mode,
		Prompt:    prompt,
		Budget: contractpeering.BudgetSpec{
			Turns:  peerCallBudgetTurns,
			Tokens: peerCallBudgetTokens,
		},
		RelatedSpec: peerCallRelatedSpec,
		Reason:      peerCallReason,
		DryRun:      peerCallDryRun,
	})
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Peer call ok  mode=%s  alias=%s  call_id=%s\n", mode, alias, res.CallID)
	fmt.Fprintf(w, "  peer_id: %s\n", res.PeerID)
	fmt.Fprintf(w, "  result_kind: %s\n", res.Result.Kind)
	if res.Result.SpecSlug != "" {
		fmt.Fprintf(w, "  peer_spec: %s/%s (status=%s)\n", alias, res.Result.SpecSlug, res.Result.PeerStatus)
	}
	if res.Result.Findings != "" && !peerCallDryRun {
		preview := res.Result.Findings
		if len(preview) > 400 {
			preview = preview[:400] + "..."
		}
		fmt.Fprintf(w, "  findings:\n    %s\n", strings.ReplaceAll(preview, "\n", "\n    "))
	}
	if res.Result.BudgetConsumed.Turns != 0 || res.Result.BudgetConsumed.Tokens != 0 {
		fmt.Fprintf(w, "  budget_consumed: %d turns / %d tokens\n",
			res.Result.BudgetConsumed.Turns, res.Result.BudgetConsumed.Tokens)
	}
	if peerCallRelatedSpec != "" {
		fmt.Fprintf(w, "  trail entry appended to spec %s\n", peerCallRelatedSpec)
		if mode == contractpeering.PeerCallSpecOut {
			fmt.Fprintf(w, "  status: awaiting_peer\n")
		}
	}
	return nil
}
