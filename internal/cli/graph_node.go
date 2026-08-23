package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/spf13/cobra"
)

// Subcommands of `hero graph node` — write-side surface for the
// knowledge graph nodes. Sibling of `hero graph edge add`: external
// callers (notably hero-code's brand "Hand off to /design" button)
// need to materialize PM-side nodes like `Story:<slug>` before they
// can connect them to engineering Features, because the hero ingest
// paths only stamp engineering-side nodes (Feature/Initiative/
// Decision/Convention/etc.) today. Writing through this CLI keeps
// `graph.UpsertNode`'s domain/idempotency invariants in one place.

var (
	graphNodeType         string
	graphNodeKey          string
	graphNodeDomain       string
	graphNodeHandlerOwner string
	graphNodeTitle        string
	graphNodeJSON         bool
)

var graphNodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Knowledge graph node operations",
}

var graphNodeAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Upsert a node into the graph",
	Long: `Inserts (or refreshes) a node identified by --type and --key.

Idempotent: re-running with the same arguments yields the same node_id.

Default domains by type (override with --domain):
  Feature, Initiative, Decision, Bug, Convention, Rule → engineering
  Story, PRD, RoadmapItem                              → pm

For routed artifacts, --handler-owner validates the selected handler's
durable owner against the enabled workspace composition and stamps it.
For types outside the defaults, --domain or --handler-owner is required
unless the type is in the global allow-list (Mission, Person, Org, Repo, Unit).

First write wins on domain: re-upserting an existing node with a
different domain returns ErrDomainMutation. To relocate a node across
domains, invalidate it first (v2 retag concern, not an upsert
concern).`,
	SilenceUsage: true,
	RunE:         runGraphNodeAdd,
}

func init() {
	graphNodeAddCmd.Flags().StringVar(&graphNodeType, "type", "", "node type, e.g. Story (required)")
	graphNodeAddCmd.Flags().StringVar(&graphNodeKey, "key", "", "node key (required)")
	graphNodeAddCmd.Flags().StringVar(&graphNodeDomain, "domain", "", "domain partition (engineering, pm, ...); defaults by type")
	graphNodeAddCmd.Flags().StringVar(&graphNodeHandlerOwner, "handler-owner", "", "durable owner selected by a composed pack handler")
	graphNodeAddCmd.Flags().StringVar(&graphNodeTitle, "title", "", "optional human title; falls back to key")
	graphNodeAddCmd.Flags().BoolVar(&graphNodeJSON, "json", false, "emit result as JSON")

	graphNodeCmd.AddCommand(graphNodeAddCmd)
	graphCmd.AddCommand(graphNodeCmd)
}

// nodeAddResult is the --json output shape. Documented at the CLI
// surface so external callers (hero-code's GraphWriter) can parse it
// stably.
type nodeAddResult struct {
	NodeID    int64  `json:"node_id"`
	Type      string `json:"type"`
	Key       string `json:"key"`
	Domain    string `json:"domain,omitempty"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
}

// defaultDomainFor returns the default domain partition for the given
// node type. Returns "" when no default applies — caller must supply
// --domain explicitly (or the type must be in graph.globalNodeTypes).
func defaultDomainFor(typ string) string {
	switch typ {
	case "Feature", "Initiative", "Decision", "Bug", "Convention", "Rule":
		return "engineering"
	case "Story", "PRD", "RoadmapItem":
		return "pm"
	}
	return ""
}

func runGraphNodeAdd(cmd *cobra.Command, args []string) error {
	if graphNodeType == "" {
		return fmt.Errorf("--type is required")
	}
	if graphNodeKey == "" {
		return fmt.Errorf("--key is required")
	}

	if graphNodeDomain != "" && graphNodeHandlerOwner != "" {
		return fmt.Errorf("--domain and --handler-owner are mutually exclusive")
	}

	title := graphNodeTitle
	if title == "" {
		title = graphNodeKey
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

	domain := graphNodeDomain
	if graphNodeHandlerOwner != "" {
		domain, err = graph.DomainForHandler(cfg, graphNodeHandlerOwner)
		if err != nil {
			return fmt.Errorf("resolving handler owner: %w", err)
		}
	} else if domain == "" {
		domain = defaultDomainFor(graphNodeType)
	}
	// If we still have no domain and the type isn't global, give the
	// caller a clear message before UpsertNode's lower-level error.
	if domain == "" && !graph.IsGlobalNodeType(graphNodeType) {
		return fmt.Errorf("must specify --domain or --handler-owner for type %q (no default registered)", graphNodeType)
	}

	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	node := &graph.Node{
		Type:   graphNodeType,
		Key:    graphNodeKey,
		Domain: domain,
		Props: map[string]any{
			"title": title,
		},
		Source: map[string]any{
			"writer": "hero graph node add",
		},
	}

	nodeID, err := store.UpsertNode(node)
	if err != nil {
		return fmt.Errorf("writing node: %w", err)
	}

	result := nodeAddResult{
		NodeID:    nodeID,
		Type:      graphNodeType,
		Key:       graphNodeKey,
		Domain:    domain,
		Title:     title,
		CreatedAt: node.ValidFrom,
	}

	if graphNodeJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Printf("node %d: %s:%s\n", result.NodeID, result.Type, result.Key)
	fmt.Printf("  domain:     %s\n", displayDomain(result.Domain))
	fmt.Printf("  title:      %s\n", result.Title)
	fmt.Printf("  created_at: %s\n", result.CreatedAt)
	return nil
}
