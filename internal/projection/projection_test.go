package projection

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

func openTestStore(t *testing.T) *graph.Store {
	t.Helper()
	s, err := graph.Open(filepath.Join(t.TempDir(), "hero"))
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// seedRepo populates a graph with a small but realistic dataset for
// projection tests.
func seedRepo(t *testing.T, store *graph.Store) {
	t.Helper()
	now := time.Now().UTC()

	// Repo
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Repo", Key: "test-repo", Repo: "test-repo", ContentHash: "h-repo",
	}); err != nil {
		t.Fatal(err)
	}
	// Recent commits
	for i, c := range []struct{ sha, subj string }{
		{"abc1234567", "feat: ship phase 4"},
		{"def8901234", "fix: corner case in parser"},
		{"012cdef567", "chore: bump deps"},
	} {
		date := now.Add(time.Duration(-i) * time.Hour).Format(time.RFC3339)
		if _, err := store.UpsertNode(&graph.Node{
			Type: "Commit", Key: c.sha, Repo: "test-repo",
			Domain:      "engineering",
			Props: map[string]any{
				"sha": c.sha, "subject": c.subj, "date": date,
			},
			ContentHash: "h-" + c.sha,
		}); err != nil {
			t.Fatal(err)
		}
	}
	// Open features
	for _, f := range []struct{ slug, title, status, prio string }{
		{"phase-5-queries", "Phase 5: cross-graph queries", "planning", "P0"},
		{"phase-6-jira", "Phase 6: Jira ingest", "planning", "P1"},
		{"phase-9-pages", "Phase 9: GitHub Pages", "planning", "P2"},
	} {
		if _, err := store.UpsertNode(&graph.Node{
			Type: "Feature", Key: f.slug, Repo: "test-repo",
			Domain:      "engineering",
			Props: map[string]any{
				"title": f.title, "status": f.status, "priority": f.prio,
			},
			ContentHash: "h-" + f.slug,
		}); err != nil {
			t.Fatal(err)
		}
	}
	// A completed feature (must NOT appear under "Next")
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Feature", Key: "graph-memory", Repo: "test-repo",
		Domain:      "engineering",
		Props: map[string]any{
			"title": "Graph memory", "status": "completed", "priority": "P0",
		},
		ContentHash: "h-graph-memory",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestNextMD_HappyPath(t *testing.T) {
	store := openTestStore(t)
	seedRepo(t, store)

	out, err := NextMD(store, NextMDOptions{
		RepoKey: "test-repo",
		Branch:  "main",
	})
	if err != nil {
		t.Fatalf("NextMD: %v", err)
	}

	// Frontmatter
	if !strings.HasPrefix(out, "---\n") {
		t.Errorf("missing frontmatter: %q", out[:50])
	}
	if !strings.Contains(out, "repo: test-repo") {
		t.Error("missing repo in frontmatter")
	}
	if !strings.Contains(out, "branch: main") {
		t.Error("missing branch in frontmatter")
	}

	// Sections
	for _, h := range []string{
		"## Just finished",
		"## Next",
		"## Blocked on",
		"## Tried and failed",
		"## Context to carry forward",
	} {
		if !strings.Contains(out, h) {
			t.Errorf("missing section %q", h)
		}
	}

	// Just finished — pointer to git log, not a frozen commit list
	if !strings.Contains(out, "git log") {
		t.Error("expected git log pointer in Just finished")
	}
	if strings.Contains(out, "abc1234") {
		t.Error("commit SHAs should not appear in Just finished (use git log)")
	}

	// Next — top-priority open feature
	if !strings.Contains(out, "phase-5-queries") {
		t.Error("expected P0 open feature in Next")
	}
	if !strings.Contains(out, "→ `/deliver phase-5-queries`") {
		t.Error("expected /deliver hint")
	}
	if strings.Contains(out, "graph-memory") {
		t.Error("completed feature should not appear in Next")
	}
}

func TestNextMD_PrioritiesOrdered(t *testing.T) {
	store := openTestStore(t)
	seedRepo(t, store)

	out, err := NextMD(store, NextMDOptions{
		RepoKey: "test-repo", NextN: 3,
	})
	if err != nil {
		t.Fatalf("NextMD: %v", err)
	}
	// Index order should be P0 (phase-5) before P1 (phase-6) before P2 (phase-9)
	i5 := strings.Index(out, "phase-5-queries")
	i6 := strings.Index(out, "phase-6-jira")
	i9 := strings.Index(out, "phase-9-pages")
	if !(i5 < i6 && i6 < i9) {
		t.Errorf("priorities out of order: 5=%d 6=%d 9=%d", i5, i6, i9)
	}
}

func TestNextMD_EmptyRepo(t *testing.T) {
	store := openTestStore(t)

	out, err := NextMD(store, NextMDOptions{RepoKey: "empty"})
	if err != nil {
		t.Fatalf("NextMD: %v", err)
	}
	if !strings.Contains(out, "git log") {
		t.Error("expected git log pointer in Just finished")
	}
	if !strings.Contains(out, "No open features in this repo.") {
		t.Error("expected message for empty Next")
	}
	if !strings.Contains(out, "Nothing.") {
		t.Error("expected 'Nothing.' for empty Blocked on")
	}
}

func TestNextMD_Blockers(t *testing.T) {
	store := openTestStore(t)
	seedRepo(t, store)

	// Make phase-6-jira depend on phase-5-queries
	fromID, _ := store.GetNodeID("Feature", "phase-6-jira")
	toID, _ := store.GetNodeID("Feature", "phase-5-queries")
	if _, err := store.UpsertEdge(&graph.Edge{
		FromID: fromID, ToID: toID, Type: "depends_on", Repo: "test-repo",
	}); err != nil {
		t.Fatalf("UpsertEdge: %v", err)
	}

	out, err := NextMD(store, NextMDOptions{RepoKey: "test-repo"})
	if err != nil {
		t.Fatalf("NextMD: %v", err)
	}
	if !strings.Contains(out, "phase-6-jira") || !strings.Contains(out, "phase-5-queries") {
		t.Errorf("expected blocker chain in Blocked on: %s", out)
	}
}

// TestNextMD_RoadmapShape_Emits covers the ambient size-drift surface:
// when AmbientDrift returns a non-quiet non-zero report, NextMD emits
// the `## Roadmap shape` section between `## Next` and `## Blocked on`.
func TestNextMD_RoadmapShape_Emits(t *testing.T) {
	store := openTestStore(t)
	seedRepo(t, store)

	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Drifted leaf — declared size doesn't match computed (lots of files
	// against declared trivial). ActiveSpec match fires rule 1.
	specDir := filepath.Join(heroDir, "planning", "features", "drifted-leaf")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := make([]string, 10)
	for i := range files {
		files[i] = fmt.Sprintf("    - `path/to/file_%d.go`", i)
	}
	body := "---\ntitle: Drifted\ntype: feature\nstatus: planning\nsize: trivial\n---\n\n## Changes\n\n" + strings.Join(files, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := NextMD(store, NextMDOptions{
		RepoKey:     "test-repo",
		HeroDir:     heroDir,
		ProjectRoot: tmp,
		ActiveSpec:  "drifted-leaf",
	})
	if err != nil {
		t.Fatalf("NextMD: %v", err)
	}

	if !strings.Contains(out, "## Roadmap shape") {
		t.Errorf("expected '## Roadmap shape' section, got:\n%s", out)
	}
	if !strings.Contains(out, "size drift") {
		t.Errorf("expected lens-agnostic 'size drift' phrasing, got:\n%s", out)
	}
	if !strings.Contains(out, "/roadmap-review") {
		t.Errorf("expected '/roadmap-review' CTA, got:\n%s", out)
	}

	// Placement check: `## Roadmap shape` must sit between `## Next`
	// and `## Blocked on`.
	idxNext := strings.Index(out, "## Next")
	idxRoadmap := strings.Index(out, "## Roadmap shape")
	idxBlocked := strings.Index(out, "## Blocked on")
	if !(idxNext < idxRoadmap && idxRoadmap < idxBlocked) {
		t.Errorf("placement wrong: next=%d roadmap=%d blocked=%d", idxNext, idxRoadmap, idxBlocked)
	}
}

// TestNextMD_RoadmapShape_OmittedWhenQuiet covers the quiet path:
// no drift → no `## Roadmap shape` header in the output at all.
func TestNextMD_RoadmapShape_OmittedWhenQuiet(t *testing.T) {
	store := openTestStore(t)
	seedRepo(t, store)

	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Empty hero workspace → no drift → quiet.

	out, err := NextMD(store, NextMDOptions{
		RepoKey:     "test-repo",
		HeroDir:     heroDir,
		ProjectRoot: tmp,
	})
	if err != nil {
		t.Fatalf("NextMD: %v", err)
	}
	if strings.Contains(out, "## Roadmap shape") {
		t.Errorf("expected no '## Roadmap shape' section when quiet, got:\n%s", out)
	}
}

// TestNextMD_RoadmapShape_NoHeroDir covers the legacy-caller path:
// when HeroDir/ProjectRoot aren't set, the section is omitted (no
// regression for existing NextMD callers that haven't been updated).
func TestNextMD_RoadmapShape_NoHeroDir(t *testing.T) {
	store := openTestStore(t)
	seedRepo(t, store)

	out, err := NextMD(store, NextMDOptions{RepoKey: "test-repo"})
	if err != nil {
		t.Fatalf("NextMD: %v", err)
	}
	if strings.Contains(out, "## Roadmap shape") {
		t.Errorf("expected no '## Roadmap shape' section when HeroDir unset, got:\n%s", out)
	}
}

func TestNextMD_AttemptsLinkedToSession(t *testing.T) {
	store := openTestStore(t)
	seedRepo(t, store)

	sessID, _ := store.UpsertNode(&graph.Node{
		Type: "Session", Key: "session-1", Repo: "test-repo", ContentHash: "h-sess",
		Domain:      "engineering",
	})
	a, _ := store.UpsertNode(&graph.Node{
		Type: "Attempt", Key: "session-1:abc",
		Domain:      "engineering",
		Repo:        "test-repo",
		Props:       map[string]any{"body": "tried bcrypt rounds=12 — too slow", "outcome": "failed"},
		ContentHash: "h-attempt",
	})
	if _, err := store.UpsertEdge(&graph.Edge{
		FromID: a, ToID: sessID, Type: "attempted_in", Repo: "test-repo",
	}); err != nil {
		t.Fatal(err)
	}

	out, err := NextMD(store, NextMDOptions{
		RepoKey: "test-repo", SessionID: "session-1",
	})
	if err != nil {
		t.Fatalf("NextMD: %v", err)
	}
	if !strings.Contains(out, "tried bcrypt") {
		t.Errorf("expected attempt body in Tried and failed: %s", out)
	}
	if strings.Contains(out, "Nothing this session.") {
		t.Error("Tried and failed should not be empty when session has attempts")
	}
}
