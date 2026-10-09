package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/spec"
)

// next-projection-stale-graph: a spec verified (status flipped and
// archived) after the graph last ingested it must not linger in NEXT.md as
// Next or as a blocker. The checkpoint reconciles the spec subgraph from
// frontmatter before projecting.
func TestCheckpointProjectsCurrentSpecStatus(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Next = &config.NextConfig{Projected: true}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatal(err)
	}
	write := func(rel, content string) {
		path := filepath.Join(env.heroDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("planning/features/engine/spec.md", "---\ntitle: Engine\nslug: engine\ntype: feature\nstatus: delivering\npriority: critical\n---\n# Engine\n")
	write("planning/features/ui/spec.md", "---\ntitle: UI\nslug: ui\ntype: feature\nstatus: planning\npriority: high\ndepends-on: [engine]\n---\n# UI\n")

	// The graph ingests the specs while engine is still delivering.
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatal(err)
	}
	specs, err := spec.Discover(env.heroDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := spec.WriteGraph(specs, gitutil.RepoKey(env.dir), "engineering", store); err != nil {
		t.Fatal(err)
	}
	store.Close()

	// engine is verified and archived; nothing writes the graph.
	if err := os.RemoveAll(filepath.Join(env.heroDir, "planning", "features", "engine")); err != nil {
		t.Fatal(err)
	}
	write("specs/engine/spec.md", "---\ntitle: Engine\nslug: engine\ntype: feature\nstatus: completed\npriority: critical\ncompleted_at: 2026-10-08T12:00:00Z\n---\n# Engine\n")

	if _, err := writeCheckpoint(); err != nil {
		t.Fatalf("writeCheckpoint: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(env.heroDir, "NEXT.md"))
	if err != nil {
		t.Fatal(err)
	}
	next := string(data)
	if !strings.Contains(next, "→ `/deliver ui`") {
		t.Errorf("Next should be ui now that engine is completed:\n%s", next)
	}
	if strings.Contains(next, "waiting on `engine`") {
		t.Errorf("a completed spec is still listed as a blocker:\n%s", next)
	}
	if !strings.Contains(next, "- **Engine** (`engine`, completed 2026-10-08)") {
		t.Errorf("Just finished does not list the verified spec:\n%s", next)
	}
}
