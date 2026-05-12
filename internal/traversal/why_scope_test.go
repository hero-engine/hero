package traversal

import (
	"strings"
	"testing"
)

func TestMarkdownScopedHighlightsInScope(t *testing.T) {
	trace := &Trace{
		Target: Hop{
			NodeType:   "Feature",
			NodeKey:    "tokenizer-fix",
			NodeTitle:  "Tokenizer fix",
			Subproject: "engines/mlx",
		},
		Chains: []Hop{
			{Depth: 1, EdgeType: "belongs_to", NodeType: "Feature", NodeKey: "mlx-engine", NodeTitle: "MLX engine", Subproject: "engines/mlx"},
			{Depth: 2, EdgeType: "depends_on", NodeType: "Feature", NodeKey: "shared-numerics", NodeTitle: "Shared numerics", Subproject: "engines/shared"},
			{Depth: 3, EdgeType: "originated_in", NodeType: "Decision", NodeKey: "platform-vision", NodeTitle: "Platform vision"},
		},
	}

	got := trace.MarkdownScoped("engines/mlx")
	if !strings.Contains(got, "* tokenizer-fix") {
		// Title comes first; check that the in-scope marker rendered on the target.
		if !strings.Contains(got, "* Tokenizer fix") {
			t.Errorf("expected target to be marked as in-scope:\n%s", got)
		}
	}
	if !strings.Contains(got, "[scope: engines/mlx]") {
		t.Errorf("expected mlx scope tag:\n%s", got)
	}
	if !strings.Contains(got, "[scope: engines/shared]") {
		t.Errorf("expected shared scope tag:\n%s", got)
	}
	if !strings.Contains(got, "[scope: (root)]") {
		t.Errorf("expected (root) tag for hop with no scope:\n%s", got)
	}
	// In-scope hop should have the marker.
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "mlx-engine") {
			if !strings.Contains(line, "* ←") {
				t.Errorf("expected mlx-engine to be marked in-scope:\n%s", line)
			}
		}
		if strings.Contains(line, "shared-numerics") {
			if strings.Contains(line, "* ←") {
				t.Errorf("expected shared-numerics NOT to be marked in-scope:\n%s", line)
			}
		}
	}
}

func TestMarkdownScopedNoActiveScope(t *testing.T) {
	trace := &Trace{
		Target: Hop{NodeType: "Feature", NodeKey: "x", NodeTitle: "X", Subproject: "engines/mlx"},
		Chains: []Hop{
			{Depth: 1, EdgeType: "belongs_to", NodeType: "Feature", NodeKey: "y", NodeTitle: "Y", Subproject: "engines/mlx"},
		},
	}
	got := trace.MarkdownScoped("")
	// Should still show the scope tag, but with no in-scope highlight.
	if !strings.Contains(got, "[scope: engines/mlx]") {
		t.Errorf("expected scope tag in output:\n%s", got)
	}
	if strings.Contains(got, "* ←") || strings.Contains(got, "* X") {
		t.Errorf("unexpected in-scope marker without active scope:\n%s", got)
	}
	// The hop with no scope and no active scope shouldn't get a "(root)" tag.
}

func TestMarkdownDelegatesToScoped(t *testing.T) {
	trace := &Trace{
		Target: Hop{NodeType: "Feature", NodeKey: "x", NodeTitle: "X"},
	}
	plain := trace.Markdown()
	scoped := trace.MarkdownScoped("")
	if plain != scoped {
		t.Errorf("Markdown() should equal MarkdownScoped(\"\"):\nplain:  %q\nscoped: %q", plain, scoped)
	}
}
