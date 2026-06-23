package extract

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/runner"
)

// RunAuto runs Tier-2 extraction over all notes + planning specs in
// the workspace, best-effort. It is the entry point used by
// `hero scan` to keep extraction automatic per the v2 charter
// ("Tier 2 enriches over time without blocking anything").
//
// If no API key is available for the default provider, returns a
// summary marked Skipped with Reason set — never errors. Callers
// can surface the reason or ignore it.
//
// For per-source extraction with override flags, see the CLI's
// `hero extract` command, which calls ExtractFromSource directly.
func RunAuto(ctx context.Context, store *graph.Store, heroDir, repoKey string) (*RunSummary, error) {
	if store == nil {
		return nil, fmt.Errorf("extract: RunAuto requires non-nil Store")
	}

	apiKey := runner.ResolveAPIKey("anthropic", "")
	if apiKey == "" {
		return &RunSummary{
			Skipped: true,
			Reason:  "optional LLM enrichment skipped — no provider key set (ANTHROPIC_API_KEY / OPENAI_API_KEY / AZURE_OPENAI_KEY). Structural graph is unaffected.",
		}, nil
	}

	llm, err := runner.GetProvider("anthropic", apiKey, nil)
	if err != nil {
		return &RunSummary{
			Skipped: true,
			Reason:  fmt.Sprintf("provider init failed: %v", err),
		}, nil
	}
	client := NewClient(llm, "")

	sources := collectAutoSources(heroDir)
	if len(sources) == 0 {
		return &RunSummary{}, nil
	}

	x := &DecisionExtractor{Client: client}
	out := &RunSummary{Sources: len(sources)}

	for _, s := range sources {
		summary, err := x.ExtractFromSource(ctx, store, s.NodeType, s.NodeKey, s.Body, repoKey)
		if err != nil {
			out.Errors++
			continue
		}
		if summary.Skipped {
			out.Cached++
			continue
		}
		out.Decisions += summary.Decisions
		out.Concepts += summary.Concepts
		out.Edges += summary.Edges
		out.InputTokens += summary.InputTokens
		out.OutputTokens += summary.OutputTokens
		out.CacheReads += summary.CacheReads
	}
	return out, nil
}

// RunSummary aggregates results from RunAuto for the scan summary
// block. Skipped + Reason are populated when extraction couldn't
// run at all (no API key, provider failure); the count fields stay
// zero in that case.
type RunSummary struct {
	Skipped      bool
	Reason       string
	Sources      int
	Cached       int
	Errors       int
	Decisions    int
	Concepts     int
	Edges        int
	InputTokens  int
	OutputTokens int
	CacheReads   int
}

// autoSource is one prose blob to feed to the extractor along with
// its node identity.
type autoSource struct {
	NodeType string
	NodeKey  string
	Body     string
}

// collectAutoSources walks notes/ + planning/ and reads each spec's
// body. Same shape as the CLI extract command, but deduplicated
// inside this package so RunAuto stays self-contained.
func collectAutoSources(heroDir string) []autoSource {
	var out []autoSource

	notesDir := filepath.Join(heroDir, "knowledge", "notes")
	if entries, err := os.ReadDir(notesDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			body, err := readSpecBody(filepath.Join(notesDir, e.Name(), "spec.md"))
			if err != nil || body == "" {
				continue
			}
			out = append(out, autoSource{NodeType: "Note", NodeKey: e.Name(), Body: body})
		}
	}

	for _, sub := range []struct{ dir, nodeType string }{
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
			body, err := readSpecBody(filepath.Join(planningDir, e.Name(), "spec.md"))
			if err != nil || body == "" {
				continue
			}
			out = append(out, autoSource{NodeType: sub.nodeType, NodeKey: e.Name(), Body: body})
		}
	}

	return out
}

// readSpecBody returns everything after the closing frontmatter ---
// in a spec file. Empty body is "" with nil error so callers can
// skip silently.
func readSpecBody(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := string(bytes)
	if !strings.HasPrefix(s, "---") {
		return strings.TrimSpace(s), nil
	}
	rest := s[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", nil
	}
	return strings.TrimSpace(rest[end+len("\n---"):]), nil
}
