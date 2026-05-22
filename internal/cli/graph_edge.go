package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/spf13/cobra"
)

// Subcommands of `hero graph edge` — write-side surface for the
// knowledge graph. Exists so external callers (notably hero-code's
// brand "Hand off to /design" button) can write cross-domain edges
// without re-implementing graph.UpsertEdge invariants out of process.

var (
	graphEdgeFrom       string
	graphEdgeTo         string
	graphEdgeKind       string
	graphEdgeFromDomain string
	graphEdgeToDomain   string
	graphEdgeJSON       bool
)

var graphEdgeCmd = &cobra.Command{
	Use:   "edge",
	Short: "Knowledge graph edge operations",
}

var graphEdgeAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Write an edge between two existing nodes",
	Long: `Writes (or refreshes) an edge between two nodes already in the graph.

Both --from and --to take a "<type>:<key>" pair (e.g. story:reduce-churn).
Idempotent: re-running with the same arguments yields the same edge_id.

Cross-domain kinds explicitly allowed in v1: handoff, derived_from, realizes.
Other cross-domain kinds still write but are surfaced as warnings via
'hero warnings' (see internal/graph/edge.go crossDomainAllowedKinds).`,
	SilenceUsage: true,
	RunE:         runGraphEdgeAdd,
}

func init() {
	graphEdgeAddCmd.Flags().StringVar(&graphEdgeFrom, "from", "", "from-node as <type>:<key> (required)")
	graphEdgeAddCmd.Flags().StringVar(&graphEdgeTo, "to", "", "to-node as <type>:<key> (required)")
	graphEdgeAddCmd.Flags().StringVar(&graphEdgeKind, "kind", "", "edge kind, e.g. handoff (required)")
	graphEdgeAddCmd.Flags().StringVar(&graphEdgeFromDomain, "from-domain", "", "edge domain partition (defaults to from-node's domain)")
	graphEdgeAddCmd.Flags().StringVar(&graphEdgeToDomain, "to-domain", "", "to-side domain hint (informational; for cross-domain output)")
	graphEdgeAddCmd.Flags().BoolVar(&graphEdgeJSON, "json", false, "emit result as JSON")

	graphEdgeCmd.AddCommand(graphEdgeAddCmd)
	graphCmd.AddCommand(graphEdgeCmd)
}

// edgeAddResult is the --json output shape. Documented at the CLI surface
// so external callers (hero-code's GraphWriter) can parse it stably.
type edgeAddResult struct {
	EdgeID     int64  `json:"edge_id"`
	From       string `json:"from"`
	To         string `json:"to"`
	Kind       string `json:"kind"`
	FromDomain string `json:"from_domain,omitempty"`
	ToDomain   string `json:"to_domain,omitempty"`
	CreatedAt  string `json:"created_at"`
}

func runGraphEdgeAdd(cmd *cobra.Command, args []string) error {
	if graphEdgeFrom == "" || graphEdgeTo == "" || graphEdgeKind == "" {
		return fmt.Errorf("--from, --to, and --kind are required")
	}

	fromType, fromKey, err := parseTypeKey(graphEdgeFrom, "--from")
	if err != nil {
		return err
	}
	toType, toKey, err := parseTypeKey(graphEdgeTo, "--to")
	if err != nil {
		return err
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, statErr := os.Stat(heroDir); os.IsNotExist(statErr) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	fromNode, err := store.GetNode(fromType, fromKey)
	if err != nil {
		if errors.Is(err, graph.ErrNotFound) {
			return fmt.Errorf("from-node not found: %s:%s", fromType, fromKey)
		}
		return fmt.Errorf("resolving from-node %s:%s: %w", fromType, fromKey, err)
	}
	toNode, err := store.GetNode(toType, toKey)
	if err != nil {
		if errors.Is(err, graph.ErrNotFound) {
			return fmt.Errorf("to-node not found: %s:%s", toType, toKey)
		}
		return fmt.Errorf("resolving to-node %s:%s: %w", toType, toKey, err)
	}

	edge := &graph.Edge{
		FromID: fromNode.ID,
		ToID:   toNode.ID,
		Type:   graphEdgeKind,
		Domain: graphEdgeFromDomain, // empty → UpsertEdge inherits from from-node
		Source: map[string]any{
			"writer": "hero graph edge add",
		},
	}

	edgeID, err := store.UpsertEdge(edge)
	if err != nil {
		return fmt.Errorf("writing edge: %w", err)
	}

	// Resolve the to-domain for output. Spec asks us to label cross-domain
	// vs same-domain; do it from the actual node rows so we don't lie when
	// the caller's --to-domain hint disagrees with the stored to-node.
	toDomain := graphEdgeToDomain
	if toDomain == "" {
		toDomain = toNode.Domain
	}

	result := edgeAddResult{
		EdgeID:     edgeID,
		From:       fmt.Sprintf("%s:%s", fromType, fromKey),
		To:         fmt.Sprintf("%s:%s", toType, toKey),
		Kind:       graphEdgeKind,
		FromDomain: edge.Domain,
		ToDomain:   toDomain,
		CreatedAt:  edge.ValidFrom,
	}

	if graphEdgeJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	crossDomain := result.FromDomain != "" && result.ToDomain != "" && result.FromDomain != result.ToDomain
	fmt.Printf("edge %d: %s --[%s]--> %s\n", result.EdgeID, result.From, result.Kind, result.To)
	fmt.Printf("  from_domain: %s\n", displayDomain(result.FromDomain))
	fmt.Printf("  to_domain:   %s\n", displayDomain(result.ToDomain))
	if crossDomain {
		fmt.Printf("  cross-domain edge (kind=%s, allowed=%t)\n",
			result.Kind, graph.IsCrossDomainAllowedKind(result.Kind))
	}
	fmt.Printf("  created_at:  %s\n", result.CreatedAt)
	return nil
}

// parseTypeKey splits "<type>:<key>" with a clear error for the
// caller's flag name. The Type and Key sides come back trimmed; the
// key may itself contain colons (e.g. URLs as keys), so we only split
// on the first one.
func parseTypeKey(raw, flag string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("%s is required", flag)
	}
	idx := strings.IndexByte(raw, ':')
	if idx <= 0 || idx == len(raw)-1 {
		return "", "", fmt.Errorf("%s must be <type>:<key>, got %q", flag, raw)
	}
	typ := strings.TrimSpace(raw[:idx])
	key := strings.TrimSpace(raw[idx+1:])
	if typ == "" || key == "" {
		return "", "", fmt.Errorf("%s must be <type>:<key>, got %q", flag, raw)
	}
	return typ, key, nil
}

func displayDomain(d string) string {
	if d == "" {
		return "(global)"
	}
	return d
}
