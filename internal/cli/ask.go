package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/retrieval"
	"github.com/hero-engine/hero/internal/search"
	"github.com/spf13/cobra"
)

// AskResult is the JSON output structure for hero ask.
type AskResult struct {
	Question   string     `json:"question"`
	Answer     string     `json:"answer"`
	Citations  []Citation `json:"citations"`
	Confidence string     `json:"confidence"`
}

// Citation holds source information for a passage used in the answer.
type Citation struct {
	Slug    string `json:"slug"`
	Path    string `json:"path"`
	Excerpt string `json:"excerpt"`
}

var askCmd = &cobra.Command{
	Use:   "ask <question>",
	Short: "Answer a question from the knowledge base",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runAsk,
}

var (
	askJSON      bool
	askCitations bool
	askType      string
	askLimit     int
)

func init() {
	askCmd.Flags().BoolVar(&askJSON, "json", false, "output JSON")
	askCmd.Flags().BoolVar(&askCitations, "citations", false, "show source slugs/paths")
	askCmd.Flags().StringVar(&askType, "type", "", "restrict to: convention, decision, context, rule")
	askCmd.Flags().IntVar(&askLimit, "limit", 20, "max entries to search")
}

func runAsk(cmd *cobra.Command, args []string) error {
	question := strings.Join(args, " ")

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Self-heal the index (incl. the knowledge corpus) so freshly-authored
	// .hero/knowledge/** files are answerable without a manual reindex.
	if _, err := index.RefreshIfStale(heroDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: index refresh failed: %v\n", err)
	}

	// Route through the unified retrieval layer. hero ask is always a
	// text-content question against the knowledge base, so it merges the
	// hand-authored .hero/knowledge/** corpus with the spec corpus.
	// --type forces FTS5 (spec-corpus type filter) and also filters the
	// knowledge corpus by kind. Spec: knowledge-surfacing.
	q := retrieval.Query{
		Text:             question,
		Limit:            askLimit,
		IncludeKnowledge: true,
	}
	if askType != "" {
		q.Types = []string{askType}
		q.Filters = map[string]string{"type": askType}
	}

	ret, err := retrieval.New(heroDir)
	if err != nil {
		return fmt.Errorf("opening retrieval layer: %w", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(q)
	if err != nil {
		return fmt.Errorf("searching: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No knowledge found for this question.")
		return nil
	}

	queryTokens := search.Tokenize(question)

	var citations []Citation
	var topScore float64
	var answerSentences []string

	for _, r := range results {
		// Passage extraction requires a file on disk. Only FTS5 results carry
		// a Path; graph-node results do not correspond to readable spec files.
		if r.Path == "" {
			continue
		}

		data, err := os.ReadFile(r.Path)
		if err != nil {
			continue
		}

		passages := search.ExtractPassages(string(data), queryTokens, 3)
		if len(passages) == 0 {
			continue
		}

		bestScore := search.ScorePassage(passages[0], queryTokens)
		if bestScore > topScore {
			topScore = bestScore
		}

		for _, p := range passages {
			if len(answerSentences) < 3 {
				answerSentences = append(answerSentences, p)
			}
		}

		citations = append(citations, Citation{
			Slug:    r.Key,
			Path:    r.Path,
			Excerpt: passages[0],
		})
	}

	if len(answerSentences) == 0 {
		fmt.Println("No knowledge found for this question.")
		return nil
	}

	answer := strings.Join(answerSentences, ". ")
	if !strings.HasSuffix(answer, ".") && !strings.HasSuffix(answer, "!") && !strings.HasSuffix(answer, "?") {
		answer += "."
	}

	confidence := search.Confidence(topScore)

	if askJSON {
		out := AskResult{
			Question:   question,
			Answer:     answer,
			Citations:  citations,
			Confidence: confidence,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Println(answer)

	slugs := make([]string, 0, len(citations))
	for _, c := range citations {
		slugs = append(slugs, c.Slug)
	}
	if len(slugs) > 0 {
		fmt.Printf("\nSources: %s\n", strings.Join(slugs, ", "))
	}

	if askCitations {
		for _, c := range citations {
			fmt.Printf("  %s  %s\n", c.Slug, c.Path)
		}
	}

	if confidence == "low" {
		fmt.Println("\nNo strong match found. Try 'hero search' for more results.")
	}

	return nil
}
