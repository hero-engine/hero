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
			Domain: "engineering",
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
			Domain: "engineering",
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
		Domain: "engineering",
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
	// branch: is deliberately NOT emitted even when Branch is set — it has
	// no consumer, is stamped from the committer's checkout, and made the CI
	// drift gate unwinnable (next-drift-gate-branch-line-drift). Guard
	// against reintroduction.
	if strings.Contains(out, "branch:") {
		t.Error("branch: must not appear in NEXT.md frontmatter — it drifts between branch and CI's main checkout")
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

// TestNextMD_RoadmapShape_NeverEmitted is the regression guard for
// next-drift-gate-unwinnable: the ambient size-drift line was removed from the
// NEXT.md projection because its corpus-derived count is stale-by-construction
// in the committed file and made the CI byte-exact drift gate unwinnable. Even
// with real drift present (a leaf whose declared size mismatches computed), the
// projection must NOT emit `## Roadmap shape` or the churny count — that signal
// now lives only in `hero size --check` / pulse / MCP.
func TestNextMD_RoadmapShape_NeverEmitted(t *testing.T) {
	store := openTestStore(t)
	seedRepo(t, store)

	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
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

	// ActiveSpec fires rule 1 of the AmbientDrift noise filter so the report
	// is non-quiet with Count>0 — i.e. the removed emission WOULD fire here if
	// it were restored. Without this the guard is vacuous (Quiet=true → the
	// emission's `if !rep.Quiet && rep.Count > 0` never fires regardless).
	out, err := NextMD(store, NextMDOptions{
		RepoKey:     "test-repo",
		HeroDir:     heroDir,
		ProjectRoot: tmp,
		ActiveSpec:  "drifted-leaf",
	})
	if err != nil {
		t.Fatalf("NextMD: %v", err)
	}
	if strings.Contains(out, "## Roadmap shape") {
		t.Errorf("`## Roadmap shape` must not appear in NEXT.md (removed for the drift gate), got:\n%s", out)
	}
	if strings.Contains(out, "size drift") {
		t.Errorf("churny size-drift count must not appear in NEXT.md, got:\n%s", out)
	}
}

// section returns the text of a `## Heading` block — from the heading
// line up to (but not including) the next `## ` heading or EOF. Used to
// compare a single projected section byte-for-byte across graph states.
func section(out, heading string) string {
	start := strings.Index(out, heading)
	if start < 0 {
		return ""
	}
	rest := out[start+len(heading):]
	if next := strings.Index(rest, "\n## "); next >= 0 {
		return heading + rest[:next]
	}
	return heading + rest
}

// TestNextMD_CarryForward_DeterministicAcrossIngestOrder is the core
// regression guard for next-context-carry-forward-drift: the
// `## Context to carry forward` list must order on committed-derivable
// fields only (priority, created, key), never on the graph-runtime
// `ingested_at`. We project the same committed Decisions/Initiatives from
// two graphs — one clean-scan-like (single clustered ingested_at, natural
// insert order) and one working-graph-like (rows inserted in reverse with
// a subset's ingested_at bumped to "now", simulating a dev who recently
// touched them) — and assert the section is byte-identical.
func TestNextMD_CarryForward_DeterministicAcrossIngestOrder(t *testing.T) {
	type ctxNode struct {
		typ, key, title, prio, created string
	}
	// Same committed props in both graphs; only insert order and
	// ingested_at differ between store A and store B.
	nodes := []ctxNode{
		{"Decision", "d-alpha", "Alpha decision", "P1", "2026-07-01"},
		{"Decision", "d-bravo", "Bravo decision", "P0", "2026-07-02"},
		{"Initiative", "i-charlie", "Charlie initiative", "P0", "2026-07-03"},
		{"Decision", "d-delta", "Delta decision", "P2", "2026-07-01"},
		{"Initiative", "i-echo", "Echo initiative", "P1", "2026-07-05"},
	}

	seed := func(t *testing.T, reverse bool, bump bool) *graph.Store {
		store := openTestStore(t)
		if _, err := store.UpsertNode(&graph.Node{
			Type: "Repo", Key: "test-repo", Repo: "test-repo", ContentHash: "h-repo",
		}); err != nil {
			t.Fatal(err)
		}
		order := make([]ctxNode, len(nodes))
		copy(order, nodes)
		if reverse {
			for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
				order[i], order[j] = order[j], order[i]
			}
		}
		for i, n := range order {
			ingested := "2026-07-10T00:00:00Z" // clean-scan-like: one cluster
			if bump {
				// working-graph-like: distinct, recent, and in an order
				// that would flip an ingested_at DESC sort if it still
				// governed the ranking.
				ingested = time.Date(2026, 8, 1, 0, 0, i, 0, time.UTC).Format(time.RFC3339)
			}
			if _, err := store.UpsertNode(&graph.Node{
				Type: n.typ, Key: n.key, Repo: "test-repo", Domain: "engineering",
				Props: map[string]any{
					"title": n.title, "status": "planning",
					"priority": n.prio, "created": n.created,
				},
				IngestedAt:  ingested,
				ContentHash: "h-" + n.key,
			}); err != nil {
				t.Fatal(err)
			}
		}
		return store
	}

	storeA := seed(t, false, false) // natural order, clustered ingested_at
	storeB := seed(t, true, true)   // reversed order, bumped ingested_at

	outA, err := NextMD(storeA, NextMDOptions{RepoKey: "test-repo"})
	if err != nil {
		t.Fatalf("NextMD A: %v", err)
	}
	outB, err := NextMD(storeB, NextMDOptions{RepoKey: "test-repo"})
	if err != nil {
		t.Fatalf("NextMD B: %v", err)
	}

	secA := section(outA, "## Context to carry forward")
	secB := section(outB, "## Context to carry forward")
	if secA == "" {
		t.Fatal("Context to carry forward section missing")
	}
	if secA != secB {
		t.Errorf("carry-forward section not deterministic across ingest order/state:\n--- clean-scan ---\n%s\n--- working-graph ---\n%s", secA, secB)
	}
	// Handoff-magic guard: the section must still carry real pinned
	// context, not be empty.
	if !strings.Contains(secA, "`i-charlie`") || !strings.Contains(secA, "`d-bravo`") {
		t.Errorf("carry-forward section lost its pinned context:\n%s", secA)
	}
	// And it must be ordered priority ASC, created DESC, key ASC:
	// i-charlie(P0,07-03) < d-bravo(P0,07-02) < i-echo(P1,07-05) < d-alpha(P1,07-01) < d-delta(P2).
	got := []int{
		strings.Index(secA, "`i-charlie`"),
		strings.Index(secA, "`d-bravo`"),
		strings.Index(secA, "`i-echo`"),
		strings.Index(secA, "`d-alpha`"),
		strings.Index(secA, "`d-delta`"),
	}
	for i := 1; i < len(got); i++ {
		if !(got[i-1] < got[i]) {
			t.Errorf("carry-forward not ordered priority/created/key: positions=%v\n%s", got, secA)
			break
		}
	}
}

// TestNextMD_Next_TieBreakDeterministic guards the `## Next` pick: with
// multiple features tied on the top priority, the ordering must break the
// tie on committed-derivable fields (created DESC, key ASC), never on
// `ingested_at`. Projecting the same committed features from a clean-scan-
// like graph and a working-graph-like graph must yield an identical
// `## Next` section.
func TestNextMD_Next_TieBreakDeterministic(t *testing.T) {
	type featNode struct {
		key, title, created string
	}
	feats := []featNode{
		{"f-one", "Feature one", "2026-07-01"},
		{"f-two", "Feature two", "2026-07-03"},
		{"f-three", "Feature three", "2026-07-02"},
	}

	seed := func(t *testing.T, reverse bool, bump bool) *graph.Store {
		store := openTestStore(t)
		if _, err := store.UpsertNode(&graph.Node{
			Type: "Repo", Key: "test-repo", Repo: "test-repo", ContentHash: "h-repo",
		}); err != nil {
			t.Fatal(err)
		}
		order := make([]featNode, len(feats))
		copy(order, feats)
		if reverse {
			for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
				order[i], order[j] = order[j], order[i]
			}
		}
		for i, f := range order {
			ingested := "2026-07-11T23:01:00Z" // clean-scan-like cluster
			if bump {
				ingested = time.Date(2026, 8, 1, 0, 0, i, 0, time.UTC).Format(time.RFC3339)
			}
			if _, err := store.UpsertNode(&graph.Node{
				Type: "Feature", Key: f.key, Repo: "test-repo", Domain: "engineering",
				Props: map[string]any{
					"title": f.title, "status": "planning",
					"priority": "P0", "created": f.created,
				},
				IngestedAt:  ingested,
				ContentHash: "h-" + f.key,
			}); err != nil {
				t.Fatal(err)
			}
		}
		return store
	}

	storeA := seed(t, false, false)
	storeB := seed(t, true, true)

	outA, err := NextMD(storeA, NextMDOptions{RepoKey: "test-repo", NextN: 3})
	if err != nil {
		t.Fatalf("NextMD A: %v", err)
	}
	outB, err := NextMD(storeB, NextMDOptions{RepoKey: "test-repo", NextN: 3})
	if err != nil {
		t.Fatalf("NextMD B: %v", err)
	}

	secA := section(outA, "## Next")
	secB := section(outB, "## Next")
	if secA == "" {
		t.Fatal("Next section missing")
	}
	if secA != secB {
		t.Errorf("Next section not deterministic across ingest order/state:\n--- clean-scan ---\n%s\n--- working-graph ---\n%s", secA, secB)
	}
	// Tie-break must be created DESC then key ASC: f-two(07-03) < f-three(07-02) < f-one(07-01).
	i2 := strings.Index(secA, "f-two")
	i3 := strings.Index(secA, "f-three")
	i1 := strings.Index(secA, "f-one")
	if !(i2 < i3 && i3 < i1) {
		t.Errorf("Next tie-break not created-DESC/key-ASC: two=%d three=%d one=%d\n%s", i2, i3, i1, secA)
	}
}

func TestNextMD_AttemptsLinkedToSession(t *testing.T) {
	store := openTestStore(t)
	seedRepo(t, store)

	sessID, _ := store.UpsertNode(&graph.Node{
		Type: "Session", Key: "session-1", Repo: "test-repo", ContentHash: "h-sess",
		Domain: "engineering",
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

// TestNextMD_EmitsSessionButNotBranch pins the split decided in
// next-drift-gate-branch-line-drift: branch: is dropped (no consumer,
// causes CI drift) but session: is KEPT (readSessionFromExistingNext reads
// it back to anchor "## Tried and failed", and graph_ingest emits a Session
// node from it — removing it silently empties the handoff section).
func TestNextMD_EmitsSessionButNotBranch(t *testing.T) {
	store := openTestStore(t)
	seedRepo(t, store)

	out, err := NextMD(store, NextMDOptions{
		RepoKey:   "test-repo",
		Branch:    "some-feature-branch",
		SessionID: "sess-xyz",
	})
	if err != nil {
		t.Fatalf("NextMD: %v", err)
	}
	if !strings.Contains(out, "session: sess-xyz") {
		t.Errorf("session: must be emitted when SessionID is set (it anchors 'Tried and failed'); got:\n%s", out[:200])
	}
	if strings.Contains(out, "branch:") {
		t.Errorf("branch: must never be emitted even when Branch is set; got:\n%s", out[:200])
	}
}
