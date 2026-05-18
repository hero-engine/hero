package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/extract"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/runner"
	"github.com/spf13/cobra"
)

// `hero extract` runs Tier-2 LLM extraction over hand-authored prose
// in the workspace (notes, specs, ingested docs) and writes the
// extracted Decision / Concept nodes into the graph.
//
// Provider-agnostic: defaults to Anthropic (ANTHROPIC_API_KEY) but
// any LLM provider supported by internal/runner can be selected via
// flags. Idempotent: unchanged content is skipped via content_hash
// caching.

var (
	extractProvider string // "anthropic" | "openai" | "azure"
	extractModel    string // override default extraction model
	extractDryRun   bool
	extractLimit    int
)

var extractCmd = &cobra.Command{
	Use:   "extract [target]",
	Short: "Run LLM extraction on prose to produce Decision / Concept nodes",
	Long: `Reads hand-authored prose from notes and specs in the workspace and
extracts structured Decision and Concept nodes into the knowledge
graph. The brief surfaces these nodes alongside hand-written
content, so the next session benefits from accumulated reasoning.

Provider-agnostic — works with Anthropic, OpenAI, Azure (or any
runner.LLMProvider). Defaults to Anthropic via ANTHROPIC_API_KEY.

Idempotent: each source's content_hash is the cache key. Unchanged
content is skipped without an LLM call.

Targets:
  notes   extract from .hero/knowledge/notes/*/spec.md  (default)
  specs   extract from .hero/planning/{features,initiatives}/*/spec.md
  all     run both`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExtract,
}

func init() {
	extractCmd.Flags().StringVar(&extractProvider, "provider", "", "LLM provider: anthropic (default), openai, azure")
	extractCmd.Flags().StringVar(&extractModel, "model", "", "model name override (default: provider-appropriate small model)")
	extractCmd.Flags().BoolVar(&extractDryRun, "dry-run", false, "list what would be extracted without calling the LLM")
	extractCmd.Flags().IntVar(&extractLimit, "limit", 0, "max sources to process (0 = no limit)")
	rootCmd.AddCommand(extractCmd)
}

func runExtract(cmd *cobra.Command, args []string) error {
	target := "notes"
	if len(args) > 0 {
		target = args[0]
	}
	if target != "notes" && target != "specs" && target != "all" {
		return fmt.Errorf("unknown target %q (want notes|specs|all)", target)
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

	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	repoKey := filepath.Base(projectRoot)

	// Build the LLM client. Provider override falls back to env-key
	// detection. Failing to find a key only matters when --dry-run is
	// false; dry-run mode lists candidate sources without an LLM.
	client, err := buildExtractClient(extractProvider, extractModel)
	if err != nil && !extractDryRun {
		return err
	}
	if !extractDryRun && (client == nil || !client.HasKey()) {
		return extract.ErrNoAPIKey
	}

	sources, err := collectExtractionSources(heroDir, target)
	if err != nil {
		return fmt.Errorf("collecting sources: %w", err)
	}
	if extractLimit > 0 && len(sources) > extractLimit {
		sources = sources[:extractLimit]
	}

	if extractDryRun {
		fmt.Printf("Would extract from %d source(s):\n", len(sources))
		for _, s := range sources {
			fmt.Printf("  %s `%s` (%s)\n", s.NodeType, s.NodeKey, s.Path)
		}
		return nil
	}

	x := &extract.DecisionExtractor{Client: client}
	var totalDec, totalCon, totalEdges, totalSkipped int
	var totalIn, totalOut, totalCacheRead int

	for _, s := range sources {
		summary, err := x.ExtractFromSource(context.Background(), store,
			s.NodeType, s.NodeKey, s.Body, repoKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s/%s: %v\n", s.NodeType, s.NodeKey, err)
			continue
		}
		if summary.Skipped {
			totalSkipped++
			continue
		}
		fmt.Printf("  %s/%s: %d decisions, %d concepts, %d edges (in=%d out=%d cache=%d)\n",
			s.NodeType, s.NodeKey,
			summary.Decisions, summary.Concepts, summary.Edges,
			summary.InputTokens, summary.OutputTokens, summary.CacheReads,
		)
		totalDec += summary.Decisions
		totalCon += summary.Concepts
		totalEdges += summary.Edges
		totalIn += summary.InputTokens
		totalOut += summary.OutputTokens
		totalCacheRead += summary.CacheReads
	}

	fmt.Printf("\nExtracted: %d decisions, %d concepts, %d edges across %d sources (%d skipped — unchanged)\n",
		totalDec, totalCon, totalEdges, len(sources)-totalSkipped, totalSkipped)
	if totalCacheRead > 0 {
		fmt.Printf("Tokens:    %d input (%d cache reads), %d output\n",
			totalIn, totalCacheRead, totalOut)
	}
	return nil
}

// extractSource is one prose blob to feed to the LLM, paired with
// the graph node identity it should be linked to.
type extractSource struct {
	NodeType string
	NodeKey  string
	Path     string
	Body     string
}

// collectExtractionSources walks the workspace for the target type
// and reads each source's prose.
func collectExtractionSources(heroDir, target string) ([]extractSource, error) {
	var out []extractSource

	if target == "notes" || target == "all" {
		notesDir := filepath.Join(heroDir, "knowledge", "notes")
		if entries, err := os.ReadDir(notesDir); err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				path := filepath.Join(notesDir, e.Name(), "spec.md")
				body, err := readSpecBody(path)
				if err != nil || body == "" {
					continue
				}
				out = append(out, extractSource{
					NodeType: "Note", NodeKey: e.Name(),
					Path: path, Body: body,
				})
			}
		}
	}

	if target == "specs" || target == "all" {
		for _, sub := range []struct {
			dir, nodeType string
		}{
			{"features", "Feature"},
			{"initiatives", "Initiative"},
		} {
			planningDir := filepath.Join(heroDir, "planning", sub.dir)
			entries, err := os.ReadDir(planningDir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				path := filepath.Join(planningDir, e.Name(), "spec.md")
				body, err := readSpecBody(path)
				if err != nil || body == "" {
					continue
				}
				out = append(out, extractSource{
					NodeType: sub.nodeType, NodeKey: e.Name(),
					Path: path, Body: body,
				})
			}
		}
	}

	return out, nil
}

// readSpecBody reads a spec file and returns the body (everything
// after the closing frontmatter ---). Returns "" for files without
// frontmatter or empty bodies.
func readSpecBody(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := string(bytes)
	if !strings.HasPrefix(s, "---") {
		return strings.TrimSpace(s), nil
	}
	// Find closing ---
	rest := s[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", nil
	}
	body := rest[end+len("\n---"):]
	return strings.TrimSpace(body), nil
}

// buildExtractClient creates an extract.Client backed by the
// requested provider. Empty providerOverride defaults to anthropic.
func buildExtractClient(providerOverride, modelOverride string) (*extract.Client, error) {
	provider := providerOverride
	if provider == "" {
		provider = "anthropic"
	}
	apiKey := runner.ResolveAPIKey(provider, "")
	if apiKey == "" {
		// No key — return a client that will ErrNoAPIKey on Run, so
		// dry-runs still work.
		return &extract.Client{}, nil
	}
	llm, err := runner.GetProvider(provider, apiKey, nil)
	if err != nil {
		return nil, fmt.Errorf("provider: %w", err)
	}
	return extract.NewClient(llm, modelOverride), nil
}
