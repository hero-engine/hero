package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
)

// why_resolution_test.go pins the two invariants that broke in
// graph-why-resolution-and-peer-spec-indexing:
//
//  1. every spec on disk is resolvable by `hero why` (the graph read path
//     self-heals from frontmatter, like the index's RefreshIfStale), and
//  2. every graph writer keys the partition by gitutil.RepoKey, never
//     filepath.Base(projectRoot).
//
// The bug: `hero why` reads graph.db, which had no read-side reconcile, so
// specs created since the last ingest (notably peer-received spec-out
// specs) were invisible to it while `hero graph`/`hero search` found them
// in the self-healing index.db. Compounding it, `graph reingest` (and four
// other writers) stamped the partition key as filepath.Base(projectRoot),
// which diverges from the reader's gitutil.RepoKey whenever an origin
// remote is set — so reingesting corrupted the graph instead of healing it.

// nodeRepo returns the repo partition of the live graph node for key.
func nodeRepo(t *testing.T, heroDir, key string) string {
	t.Helper()
	store, err := graph.Open(heroDir)
	if err != nil {
		t.Fatalf("open graph: %v", err)
	}
	defer store.Close()
	var repo string
	err = store.DB().QueryRow(
		`SELECT repo FROM nodes WHERE key = ? AND valid_to IS NULL LIMIT 1`, key,
	).Scan(&repo)
	if err != nil {
		t.Fatalf("no live node for key %q: %v", key, err)
	}
	return repo
}

// TestWhyResolvesSpecCreatedSinceLastIngest is the primary regression
// guard: a spec written to disk resolves through `hero why` even though no
// `hero scan`/`graph reingest` has run since it was written. Only the
// index was refreshed (indexAll) — the graph substrate that `hero why`
// reads was deliberately left untouched. Fails before Change 1 (runWhy had
// no reconcile → "no node with key"); passes after.
func TestWhyResolvesSpecCreatedSinceLastIngest(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/fresh-feature/spec.md", `---
title: Fresh Feature
type: feature
status: planning
slug: fresh-feature
---
# Fresh Feature
`)
	// Refresh the index only — the store hero graph/search read. The graph
	// substrate hero why reads gets NO ingest here, reproducing the
	// "created since last ingest" gap.
	env.indexAll()

	out, err := runCmd("why", "fresh-feature")
	if err != nil {
		t.Fatalf("hero why should resolve a spec present on disk without a prior graph ingest; got error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "fresh-feature") {
		t.Errorf("hero why output should mention the resolved slug, got:\n%s", out)
	}
}

// TestGraphReingestUsesGitRemoteRepoKey pins graph_memory.go's repoKey
// derivation: after `graph reingest work` in a repo whose origin remote
// slug ("acme/widgets") differs from its directory name, the node must be
// keyed under gitutil.RepoKey — the partition the reader filters on — and
// `hero why` must still resolve. Fails before Change 2 (reingest wrote the
// node under filepath.Base(projectRoot), unreachable by the reader).
func TestGraphReingestUsesGitRemoteRepoKey(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")
	gitRun(t, env.dir, "remote", "add", "origin", "git@github.com:acme/widgets.git")

	env.addSpec("planning/features/reingest-me/spec.md", `---
title: Reingest Me
type: feature
status: planning
slug: reingest-me
---
# Reingest Me
`)
	// A commit so `graph reingest work`'s git-log pass has history to read.
	gitRun(t, env.dir, "add", ".")
	gitRun(t, env.dir, "commit", "-q", "-m", "add reingest-me")
	env.indexAll()

	wantKey := gitutil.RepoKey(env.dir)
	if wantKey != "acme/widgets" {
		t.Fatalf("test setup: RepoKey = %q, want acme/widgets (from the origin remote)", wantKey)
	}
	if wantKey == filepath.Base(env.dir) {
		t.Fatalf("test setup: RepoKey %q must differ from dir base %q to exercise the bug", wantKey, filepath.Base(env.dir))
	}

	if out, err := runCmd("graph", "reingest", "work"); err != nil {
		t.Fatalf("graph reingest work: %v\n%s", err, out)
	}

	if got := nodeRepo(t, env.heroDir, "reingest-me"); got != wantKey {
		t.Errorf("reingest keyed node under repo %q, want %q (git-remote RepoKey, not filepath.Base)", got, wantKey)
	}

	if out, err := runCmd("why", "reingest-me"); err != nil {
		t.Fatalf("hero why should still resolve after reingest, got: %v\n%s", err, out)
	}
}

// TestWhyAndGraphAgreeOnResolution is the parity invariant: for every slug
// hero graph resolves via the index, hero why must also resolve via the
// graph substrate. Guards the two stores from silently drifting apart
// again. Fails before Change 1 (graph resolves the fresh specs; why
// errors).
func TestWhyAndGraphAgreeOnResolution(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/initiatives/parent-init/spec.md", `---
title: Parent Init
type: initiative
status: planning
slug: parent-init
---
# Parent Init
`)
	env.addSpec("planning/features/child-a/spec.md", `---
title: Child A
type: feature
status: planning
slug: child-a
parent: parent-init
---
# Child A
`)
	env.addSpec("planning/features/child-b/spec.md", `---
title: Child B
type: feature
status: planning
slug: child-b
depends-on: child-a
---
# Child B
`)
	env.indexAll()

	for _, slug := range []string{"parent-init", "child-a", "child-b"} {
		graphOut, graphErr := runCmd("graph", slug)
		if graphErr != nil {
			t.Fatalf("hero graph %s errored: %v\n%s", slug, graphErr, graphOut)
		}
		// hero graph resolved this slug (no error) — hero why must too.
		whyOut, whyErr := runCmd("why", slug)
		if whyErr != nil {
			t.Errorf("parity broken: hero graph resolves %q but hero why errors: %v\n%s", slug, whyErr, whyOut)
			continue
		}
		if !strings.Contains(whyOut, slug) {
			t.Errorf("hero why %s resolved but did not mention the slug:\n%s", slug, whyOut)
		}
	}
}

// TestNoFilepathBaseRepoKey keeps the fix from regressing: no graph writer
// anywhere under internal/ may derive its repo partition key via
// filepath.Base(projectRoot). All must route through graphRepoKey /
// gitutil.RepoKey. This is a source-level guard so another site can't
// silently reintroduce the divergence. (filepath.Base(projectRoot) used
// for a display projectName — e.g. checkpoint.go, report.go — is fine and
// deliberately not matched.)
//
// The sweep covers the whole internal/ tree, not just this package: the
// peer-receive write path is a graph writer outside internal/cli, so a
// package-local guard would have left it unguarded.
func TestNoFilepathBaseRepoKey(t *testing.T) {
	// Matches a repoKey/RepoKey assignment fed by filepath.Base(projectRoot).
	offender := regexp.MustCompile(`(?i)repokey\s*:?=?\s*filepath\.Base\(projectRoot\)|RepoKey:\s*filepath\.Base\(projectRoot\)`)

	root := ".."
	scanned := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		scanned++
		for i, line := range strings.Split(string(data), "\n") {
			if offender.MatchString(line) {
				t.Errorf("%s:%d derives a graph repoKey via filepath.Base(projectRoot); use gitutil.RepoKey(projectRoot) instead:\n\t%s",
					path, i+1, strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if scanned < 100 {
		t.Fatalf("guard scanned only %d files under internal/ — the sweep is not reaching the tree", scanned)
	}
}

// TestGraphSelfHealsColdIndex covers the third secondary defect in the spec:
// `hero graph` read index.db without ever refreshing it, so it only resolved
// a freshly-written spec when some earlier command (search / list / ask)
// happened to self-heal the index first. On a genuinely cold index it missed
// — the same staleness class as the `hero why` bug, one store over.
//
// The test deliberately never calls indexAll(): the specs exist only on
// disk, which is exactly the state a fresh clone or a just-received peer
// spec is in.
func TestGraphSelfHealsColdIndex(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/initiatives/payments/spec.md", `---
title: Payments
type: initiative
status: planning
slug: payments
---
# Payments
`)
	env.addSpec("planning/features/refunds/spec.md", `---
title: Refunds
type: feature
status: planning
slug: refunds
parent: payments
---
# Refunds
`)

	output, err := runCmd("graph", "refunds")
	if err != nil {
		t.Fatalf("graph errored: %v", err)
	}
	if !strings.Contains(output, "payments") {
		t.Errorf("hero graph should resolve a relation from a cold index without a manual `hero index`; got: %q", output)
	}
}

// TestWhyReconcileStaysWithinBudgetAtCorpusScale is the regression check the
// spec's "Regression scope" calls for and that the earlier landing left
// unwritten. Change 1 put spec.Discover + spec.WriteGraph on `hero why`'s hot
// path; the existing budget test (TestWhy_DepthFourUnder200ms) calls
// traversal.Why directly against a hand-seeded in-memory store and never
// enters runWhy, so the added reconcile cost had no coverage at all.
//
// This drives the real command over a corpus large enough that a per-spec
// regression would show, and asserts both halves of the contract: the
// reconcile makes an un-ingested spec resolvable, and it does so within
// budget. The budget is an order-of-magnitude guard, not a benchmark: it has
// to catch a real regression (e.g. reconciling once per hop rather than once
// per run) without becoming a flaky timing assertion on a slow runner.
func TestWhyReconcileStaysWithinBudgetAtCorpusScale(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/initiatives/platform/spec.md", `---
title: Platform
type: initiative
status: planning
slug: platform
---
# Platform
`)
	const corpus = 200
	for i := 0; i < corpus; i++ {
		slug := fmt.Sprintf("leaf-%03d", i)
		env.addSpec("planning/features/"+slug+"/spec.md", `---
title: Leaf `+slug+`
type: feature
status: planning
slug: `+slug+`
parent: platform
---
# Leaf `+slug+`
`)
	}
	// Deliberately no indexAll() and no scan: the corpus exists only on disk,
	// which is the state the read-side reconcile has to cope with.

	start := time.Now()
	output, err := runCmd("why", "leaf-199")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("why errored: %v", err)
	}
	if !strings.Contains(output, "leaf-199") {
		t.Fatalf("why should resolve a spec that was never ingested; got: %q", output)
	}

	// 2s: measured runs land at 50-215ms, so this is still ~10x headroom
	// against a slow runner while staying tight enough to catch a real
	// regression (e.g. reconciling once per hop rather than once per run).
	const budget = 2 * time.Second
	if elapsed > budget {
		t.Errorf("hero why over a %d-spec cold corpus took %v, budget %v — "+
			"the read-side reconcile has regressed beyond a one-shot cost",
			corpus, elapsed, budget)
	}
	t.Logf("hero why over %d cold specs: %v (budget %v)", corpus, elapsed, budget)
}

// TestWhySurvivesSiblingRepoIngest is AC-6 of graph-node-identity-repo-scoped,
// asserted end-to-end through the real command: the team-oauth failure.
//
// A federated/sibling ingest writes the same slug under its own repoKey. With
// node identity keyed on (type, key) alone that tombstoned the local node and
// re-keyed it to the sibling, so `hero why <local-slug>` — which filters on
// the local repoKey — failed outright. Repo-scoped identity keeps both live.
//
// The store assertion between the two `hero why` runs is load-bearing. A cold
// audit caught the first version of this test passing against the BROKEN code:
// `hero why` reconciles the spec subgraph from disk before resolving, so the
// second run re-asserted the local node and the command succeeded even while
// the sibling ingest had tombstoned it. Checking the graph directly is what
// actually exercises identity; the command assertions then cover the composed
// behavior on top.
func TestWhySurvivesSiblingRepoIngest(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/team-oauth/spec.md", `---
title: Team OAuth
type: feature
status: planning
slug: team-oauth
---
# Team OAuth
`)

	// Resolve once so the read-side reconcile writes the local node.
	if _, err := runCmd("why", "team-oauth"); err != nil {
		t.Fatalf("why (before sibling ingest) errored: %v", err)
	}

	// A sibling repo ingests its own copy of the slug.
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	if _, err := store.UpsertNode(&graph.Node{
		Type:        "Feature",
		Domain:      "engineering",
		Key:         "team-oauth",
		Props:       map[string]any{"title": "Peer Team OAuth"},
		Repo:        "hero-engine/hero-cloud",
		ContentHash: "peer-copy",
		Source:      map[string]any{"kind": "sibling-scan"},
	}); err != nil {
		t.Fatalf("sibling upsert: %v", err)
	}
	// Before any reconcile can heal it: the local node must still be live.
	// This is the assertion that fails on the pre-fix code.
	var localLive int
	if err := store.DB().QueryRow(
		`SELECT count(*) FROM nodes
		  WHERE type = 'Feature' AND key = 'team-oauth'
		    AND repo = ? AND valid_to IS NULL`,
		gitutil.RepoKey(env.dir),
	).Scan(&localLive); err != nil {
		t.Fatalf("count local live rows: %v", err)
	}
	store.Close()
	if localLive != 1 {
		t.Fatalf("local live rows = %d, want 1 — the sibling ingest tombstoned the local node", localLive)
	}

	output, err := runCmd("why", "team-oauth")
	if err != nil {
		t.Fatalf("a sibling repo's copy broke local resolution: %v", err)
	}
	if strings.Contains(output, "Peer Team OAuth") {
		t.Errorf("hero why resolved the SIBLING's node for a local query: %q", output)
	}
	if !strings.Contains(output, "team-oauth") {
		t.Errorf("hero why should still resolve the local spec; got: %q", output)
	}
}
