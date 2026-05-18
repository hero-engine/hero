package chat

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/retrieval"
	"github.com/hero-engine/hero/internal/search"
)

// askHandler is the runner-free /ask slash. It wraps the existing
// retrieval pipeline (no inference): query the FTS / graph index,
// extract passages from the top citations, stream them as chat.token
// events, then emit chat.done.
//
// The handler reads .hero/ from req.Context.Workspace. When that is
// empty, it emits a chat.error rather than guessing.
func askHandler(ctx context.Context, req DispatchRequest, out chan<- Event) error {
	question := ""
	if req.Slash != nil {
		question = req.Slash.Args
	}
	if question == "" {
		question = strings.TrimSpace(req.Prompt)
		// Strip a leading "/ask" if it survived.
		question = strings.TrimPrefix(question, "/ask")
		question = strings.TrimSpace(question)
	}
	if question == "" {
		out <- ErrorEvent("slash_failed", "/ask needs a question", "")
		out <- DoneEvent(0, nil)
		return nil
	}

	heroDir, err := resolveHeroDir(req.Context.Workspace)
	if err != nil {
		out <- ErrorEvent("slash_failed", err.Error(), "")
		out <- DoneEvent(0, nil)
		return nil
	}

	ret, err := retrieval.New(heroDir)
	if err != nil {
		out <- ErrorEvent("slash_failed", fmt.Sprintf("open retrieval: %v", err), "")
		out <- DoneEvent(0, nil)
		return nil
	}
	defer ret.Close()

	results, err := ret.Retrieve(retrieval.Query{Text: question, Limit: 10})
	if err != nil {
		out <- ErrorEvent("slash_failed", fmt.Sprintf("search: %v", err), "")
		out <- DoneEvent(0, nil)
		return nil
	}
	if len(results) == 0 {
		out <- TokenEvent("No knowledge found for this question.")
		out <- DoneEvent(0, nil)
		return nil
	}

	tokens := search.Tokenize(question)
	var sentences []string
	var slugs []string
	for _, r := range results {
		if r.Path == "" {
			continue
		}
		data, err := os.ReadFile(r.Path)
		if err != nil {
			continue
		}
		passages := search.ExtractPassages(string(data), tokens, 2)
		if len(passages) == 0 {
			continue
		}
		for _, p := range passages {
			if len(sentences) >= 3 {
				break
			}
			sentences = append(sentences, p)
		}
		slugs = append(slugs, r.Key)
		if len(sentences) >= 3 {
			break
		}
	}

	if len(sentences) == 0 {
		out <- TokenEvent("No knowledge found for this question.")
		out <- DoneEvent(0, nil)
		return nil
	}

	answer := strings.Join(sentences, ". ")
	if !strings.HasSuffix(answer, ".") && !strings.HasSuffix(answer, "!") && !strings.HasSuffix(answer, "?") {
		answer += "."
	}

	// Stream the answer in one token event today; future versions can
	// chunk by sentence for incremental rendering.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case out <- TokenEvent(answer):
	}

	if len(slugs) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- TokenEvent("\n\nSources: " + strings.Join(slugs, ", ")):
		}
	}

	outcome := map[string]interface{}{"citations": slugs}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case out <- DoneEvent(0, outcome):
	}
	return nil
}
