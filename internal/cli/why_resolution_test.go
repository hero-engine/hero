package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

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

// TestNoFilepathBaseRepoKey keeps the fix from regressing: no graph-writer
// in internal/cli may derive its repo partition key via
// filepath.Base(projectRoot). All must route through graphRepoKey /
// gitutil.RepoKey. This is a source-level guard so a 6th site can't
// silently reintroduce the divergence. (filepath.Base(projectRoot) used
// for a display projectName — e.g. checkpoint.go, report.go — is fine and
// deliberately not matched.)
func TestNoFilepathBaseRepoKey(t *testing.T) {
	// Matches a repoKey/RepoKey assignment fed by filepath.Base(projectRoot).
	offender := regexp.MustCompile(`(?i)repokey\s*:?=?\s*filepath\.Base\(projectRoot\)|RepoKey:\s*filepath\.Base\(projectRoot\)`)

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read internal/cli dir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			if offender.MatchString(line) {
				t.Errorf("%s:%d derives a graph repoKey via filepath.Base(projectRoot); use graphRepoKey(projectRoot) instead:\n\t%s",
					name, i+1, strings.TrimSpace(line))
			}
		}
	}
}
