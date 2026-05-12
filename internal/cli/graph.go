package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/spf13/cobra"
)

var graphFormat string

var graphCmd = &cobra.Command{
	Use:   "graph [spec-slug]",
	Short: "Show spec relationships",
	Long: `Displays parent, child, dependency, and other relationships for a spec.

With no argument, runs 'graph stats' — a high-level overview of node and
edge counts in the knowledge graph.

Use --format mermaid to output a Mermaid diagram instead of text.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGraph,
}

func init() {
	graphCmd.Flags().StringVar(&graphFormat, "format", "text", "output format: text, mermaid")
}

func runGraph(cmd *cobra.Command, args []string) error {
	// No slug → fall through to 'graph stats' as a useful default rather
	// than erroring out and forcing the user to discover the subcommand.
	if len(args) == 0 {
		return runGraphStats(cmd, args)
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	idx, err := index.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	slug := args[0]
	relations, err := idx.GetRelations(slug)
	if err != nil {
		return fmt.Errorf("getting relations: %w", err)
	}

	if len(relations) == 0 {
		fmt.Printf("No relationships found for %s.\n", slug)
		return nil
	}

	switch strings.ToLower(graphFormat) {
	case "mermaid":
		return renderMermaid(slug, relations)
	case "text", "":
		return renderText(slug, relations)
	default:
		return fmt.Errorf("unknown format %q — use text or mermaid", graphFormat)
	}
}

func renderText(slug string, relations []index.RelationResult) error {
	fmt.Printf("Relationships for %s:\n\n", slug)

	// Group by relation type
	groups := make(map[string][]index.RelationResult)
	var order []string
	for _, r := range relations {
		label := r.Relation
		if r.Direction == "incoming" {
			label = label + " (incoming)"
		}
		if _, exists := groups[label]; !exists {
			order = append(order, label)
		}
		groups[label] = append(groups[label], r)
	}

	for _, label := range order {
		fmt.Printf("  %s:\n", label)
		for _, r := range groups[label] {
			fmt.Printf("    %-30s  %-10s  %-10s  %s\n", r.Slug, r.Type, r.Status, r.Title)
		}
	}

	fmt.Printf("\n%d relationship(s)\n", len(relations))
	return nil
}

func renderMermaid(slug string, relations []index.RelationResult) error {
	fmt.Println("```mermaid")
	fmt.Println("graph TD")

	// Collect all unique nodes
	nodes := make(map[string]index.RelationResult)
	for _, r := range relations {
		nodes[r.Slug] = r
	}

	// Render the center node
	fmt.Printf("    %s[\"%s\"]\n", mermaidID(slug), slug)

	// Render related nodes
	for _, r := range nodes {
		label := fmt.Sprintf("%s\\n%s | %s", r.Slug, r.Type, r.Status)
		fmt.Printf("    %s[\"%s\"]\n", mermaidID(r.Slug), label)
	}

	fmt.Println()

	// Render edges
	for _, r := range relations {
		from := slug
		to := r.Slug
		edgeLabel := r.Relation

		if r.Direction == "incoming" {
			from = r.Slug
			to = slug
		}

		switch r.Relation {
		case "parent":
			fmt.Printf("    %s -->|parent| %s\n", mermaidID(from), mermaidID(to))
		case "child":
			fmt.Printf("    %s -->|child| %s\n", mermaidID(from), mermaidID(to))
		case "depends-on":
			fmt.Printf("    %s -.->|depends-on| %s\n", mermaidID(from), mermaidID(to))
		case "supersedes":
			fmt.Printf("    %s ==>|supersedes| %s\n", mermaidID(from), mermaidID(to))
		default:
			fmt.Printf("    %s -->|%s| %s\n", mermaidID(from), edgeLabel, mermaidID(to))
		}
	}

	// Style the center node
	fmt.Printf("\n    style %s fill:#4a9eff,color:#fff\n", mermaidID(slug))
	fmt.Println("```")

	fmt.Printf("\n%d relationship(s)\n", len(relations))
	return nil
}

// mermaidID converts a slug to a valid Mermaid node ID.
// Mermaid IDs cannot contain hyphens, so we replace them.
func mermaidID(slug string) string {
	return strings.ReplaceAll(slug, "-", "_")
}
