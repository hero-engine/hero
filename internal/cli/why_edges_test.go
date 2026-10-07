package cli

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/graph"
)

// `hero why --edges` resolves its start node exactly like `hero why`: a
// promoted spec beats its same-slug intake even when the intake was
// ingested later (the old unordered LIMIT 1 picked either).
func TestWhyEdgesPromotedSpecWinsSlugTie(t *testing.T) {
	store, err := graph.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ids := map[string]int64{}
	// The intake is inserted first (lower id) so an unordered LIMIT 1 picks it.
	for _, typ := range []string{"Intake", "Feature"} {
		id, err := store.UpsertNode(&graph.Node{Type: typ, Domain: "engineering", Key: "promoted",
			Props: map[string]any{"title": typ + " title"}, Repo: "repo-x", ContentHash: typ})
		if err != nil {
			t.Fatal(err)
		}
		ids[typ] = id
	}
	for typ, at := range map[string]string{"Feature": "2026-10-07T03:00:00Z", "Intake": "2026-10-07T03:00:01Z"} {
		if _, err := store.DB().Exec(`UPDATE nodes SET ingested_at = ? WHERE id = ?`, at, ids[typ]); err != nil {
			t.Fatal(err)
		}
	}
	out := captureStdout(func() {
		if err := runWhyEdges(store, "repo-x", "promoted"); err != nil {
			t.Fatal(err)
		}
	})
	header := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(header, "(Feature)") {
		t.Fatalf("--edges started from the intake, not the promoted spec: %q", header)
	}
}
