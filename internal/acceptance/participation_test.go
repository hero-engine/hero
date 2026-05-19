package acceptance

import (
	"testing"

	"github.com/hero-engine/hero/internal/graph"
)

func TestComputeParticipation_DerivesEdgesFromJoin(t *testing.T) {
	store := openTestStore(t)

	// Seed: criterion satisfied_by commit, commit touches two files.
	critID := mustUpsertNode(t, store, "Criterion", "feat:AC-1", map[string]any{
		"statement": "first AC", "status": "passing",
	}, "repo-x")
	commitID := mustUpsertNode(t, store, "Commit", "abc123", map[string]any{
		"sha": "abc123", "subject": "feat: ship AC-1",
	}, "repo-x")
	fileA := mustUpsertNode(t, store, "File", "internal/foo/foo.go", nil, "repo-x")
	fileB := mustUpsertNode(t, store, "File", "internal/foo/bar.go", nil, "repo-x")

	mustUpsertEdge(t, store, critID, commitID, "satisfied_by", "repo-x")
	mustUpsertEdge(t, store, commitID, fileA, "touches", "repo-x")
	mustUpsertEdge(t, store, commitID, fileB, "touches", "repo-x")

	summary, err := ComputeParticipation(store, "repo-x")
	if err != nil {
		t.Fatalf("ComputeParticipation: %v", err)
	}
	if summary.Edges != 2 {
		t.Errorf("Edges = %d, want 2", summary.Edges)
	}
	if summary.Touched != 2 {
		t.Errorf("Touched = %d, want 2", summary.Touched)
	}
	if summary.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", summary.Skipped)
	}

	// Verify the two participates_in edges land.
	edges, err := store.EdgesFrom(fileA, "participates_in")
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 {
		t.Errorf("fileA edges = %d, want 1", len(edges))
	}
	if len(edges) == 1 && edges[0].ToID != critID {
		t.Errorf("fileA edge points at %d, want criterion %d", edges[0].ToID, critID)
	}
}

func TestComputeParticipation_IdempotentOnReRun(t *testing.T) {
	store := openTestStore(t)

	critID := mustUpsertNode(t, store, "Criterion", "feat:AC-1", map[string]any{"statement": "x"}, "repo-x")
	commitID := mustUpsertNode(t, store, "Commit", "abc123", nil, "repo-x")
	fileID := mustUpsertNode(t, store, "File", "foo.go", nil, "repo-x")
	mustUpsertEdge(t, store, critID, commitID, "satisfied_by", "repo-x")
	mustUpsertEdge(t, store, commitID, fileID, "touches", "repo-x")

	first, err := ComputeParticipation(store, "repo-x")
	if err != nil {
		t.Fatal(err)
	}
	if first.Edges != 1 {
		t.Fatalf("first run Edges = %d, want 1", first.Edges)
	}

	second, err := ComputeParticipation(store, "repo-x")
	if err != nil {
		t.Fatal(err)
	}
	if second.Edges != 0 {
		t.Errorf("second run Edges = %d, want 0 (idempotent)", second.Edges)
	}
	if second.Skipped != 1 {
		t.Errorf("second run Skipped = %d, want 1", second.Skipped)
	}
}

func TestComputeParticipation_DistinctOnMultipleSatisfyingCommits(t *testing.T) {
	// One file touched by two different commits that both satisfy
	// the same criterion. Should produce ONE participates_in edge,
	// not two.
	store := openTestStore(t)

	critID := mustUpsertNode(t, store, "Criterion", "feat:AC-1", map[string]any{"statement": "x"}, "repo-x")
	commit1 := mustUpsertNode(t, store, "Commit", "abc111", nil, "repo-x")
	commit2 := mustUpsertNode(t, store, "Commit", "abc222", nil, "repo-x")
	fileID := mustUpsertNode(t, store, "File", "foo.go", nil, "repo-x")

	mustUpsertEdge(t, store, critID, commit1, "satisfied_by", "repo-x")
	mustUpsertEdge(t, store, critID, commit2, "satisfied_by", "repo-x")
	mustUpsertEdge(t, store, commit1, fileID, "touches", "repo-x")
	mustUpsertEdge(t, store, commit2, fileID, "touches", "repo-x")

	summary, err := ComputeParticipation(store, "repo-x")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Edges != 1 {
		t.Errorf("Edges = %d, want 1 (deduped)", summary.Edges)
	}
}

func TestComputeParticipation_NoEdgesWhenNoSatisfiedBy(t *testing.T) {
	store := openTestStore(t)
	mustUpsertNode(t, store, "Criterion", "feat:AC-1", map[string]any{"statement": "x"}, "repo-x")
	commitID := mustUpsertNode(t, store, "Commit", "abc123", nil, "repo-x")
	fileID := mustUpsertNode(t, store, "File", "foo.go", nil, "repo-x")
	mustUpsertEdge(t, store, commitID, fileID, "touches", "repo-x")

	summary, err := ComputeParticipation(store, "repo-x")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Edges != 0 {
		t.Errorf("Edges = %d, want 0 (no satisfied_by → no participation)", summary.Edges)
	}
}

func mustUpsertNode(t *testing.T, store *graph.Store, typ, key string, props map[string]any, repo string) int64 {
	t.Helper()
	id, err := store.UpsertNode(&graph.Node{
		Type:   typ,
		Domain: domainForTestType(typ),
		Key:    key,
		Props:  props,
		Repo:   repo,
		Source: map[string]any{"kind": "test"},
	})
	if err != nil {
		t.Fatalf("upsert %s/%s: %v", typ, key, err)
	}
	return id
}

func mustUpsertEdge(t *testing.T, store *graph.Store, from, to int64, typ, repo string) {
	t.Helper()
	if _, err := store.UpsertEdge(&graph.Edge{
		FromID: from, ToID: to, Type: typ, Repo: repo,
		Source: map[string]any{"kind": "test"},
	}); err != nil {
		t.Fatalf("upsert edge %d-%s->%d: %v", from, typ, to, err)
	}
}

// domainForTestType returns the test-default Domain for a node type:
// "" for the global allow-list (Mission/Person/Org/Repo/Unit) and
// "engineering" otherwise. Lets test seed helpers stamp non-global
// nodes without forcing every test to write Domain at every call site.
func domainForTestType(typ string) string {
	if graph.IsGlobalNodeType(typ) {
		return ""
	}
	return "engineering"
}
