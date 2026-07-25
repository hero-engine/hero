package graph

import (
	"strings"
	"testing"
)

func upsert(t *testing.T, s *Store, typ, key, repo, hash string) int64 {
	t.Helper()
	id, err := s.UpsertNode(&Node{
		Type:        typ,
		Domain:      "engineering",
		Key:         key,
		Props:       map[string]any{"title": repo + "/" + key},
		Repo:        repo,
		ContentHash: hash,
		Source:      map[string]any{"kind": "test"},
	})
	if err != nil {
		t.Fatalf("upsert %s/%s in %q: %v", typ, key, repo, err)
	}
	return id
}

// mustUpsert is for cases that need control over fields the `upsert` shorthand
// fixes — notably ValidFrom, where second-precision wall-clock stamps make two
// consecutive writes indistinguishable.
func mustUpsert(t *testing.T, s *Store, n *Node) int64 {
	t.Helper()
	id, err := s.UpsertNode(n)
	if err != nil {
		t.Fatalf("upsert %s/%s in %q: %v", n.Type, n.Key, n.Repo, err)
	}
	return id
}

func liveRows(t *testing.T, s *Store, typ, key string) map[string]int64 {
	t.Helper()
	rows, err := s.DB().Query(
		`SELECT repo, id FROM nodes WHERE type = ? AND key = ? AND valid_to IS NULL`,
		typ, key)
	if err != nil {
		t.Fatalf("query live rows: %v", err)
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var repo string
		var id int64
		if err := rows.Scan(&repo, &id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out[repo] = id
	}
	return out
}

// TestSiblingRepoIngestDoesNotTombstoneLocal is the team-oauth regression
// (AC-1). Identity was (type, key) alone, so a sibling repo ingesting a slug
// the local repo also owned matched the local live row, saw a differing
// partition, and invalidated-and-reinserted it under the sibling's repoKey.
// Every reader filtering on the local repoKey then found nothing.
func TestSiblingRepoIngestDoesNotTombstoneLocal(t *testing.T) {
	const (
		local = "hero-engine/hero"
		peer  = "hero-engine/hero-cloud"
		slug  = "team-oauth"
	)
	s := openTestStore(t)

	localID := upsert(t, s, "Feature", slug, local, "local-v1")
	peerID := upsert(t, s, "Feature", slug, peer, "peer-v1")

	if peerID == localID {
		t.Fatalf("peer ingest reused the local row id %d — identity is not repo-scoped", localID)
	}
	live := liveRows(t, s, "Feature", slug)
	if len(live) != 2 {
		t.Fatalf("live rows by repo = %v, want one per repo", live)
	}
	if live[local] != localID {
		t.Errorf("local live row = %d, want %d — the sibling ingest tombstoned it", live[local], localID)
	}
	if live[peer] != peerID {
		t.Errorf("peer live row = %d, want %d", live[peer], peerID)
	}

	// And each partition resolves to its own node, not the other's.
	if got, err := s.GetNodeID("Feature", slug, local); err != nil || got != localID {
		t.Errorf("GetNodeID(local) = %d, %v; want %d", got, err, localID)
	}
	if got, err := s.GetNodeID("Feature", slug, peer); err != nil || got != peerID {
		t.Errorf("GetNodeID(peer) = %d, %v; want %d", got, err, peerID)
	}
}

// TestSameRepoUpsertStaysIdempotent guards AC-2: repo-scoping must not cost
// the existing idempotency contract. Re-upserting unchanged content in the
// same partition is still a no-op, and changed content still supersedes
// rather than accumulating a second live row.
func TestSameRepoUpsertStaysIdempotent(t *testing.T) {
	const repo = "hero-engine/hero"
	s := openTestStore(t)

	first := upsert(t, s, "Feature", "alpha", repo, "v1")
	again := upsert(t, s, "Feature", "alpha", repo, "v1")
	if again != first {
		t.Errorf("re-upsert of unchanged content returned id %d, want the same row %d", again, first)
	}
	if live := liveRows(t, s, "Feature", "alpha"); len(live) != 1 {
		t.Fatalf("live rows = %v, want exactly 1", live)
	}

	changed := upsert(t, s, "Feature", "alpha", repo, "v2")
	if changed == first {
		t.Error("changed content should supersede into a new row")
	}
	live := liveRows(t, s, "Feature", "alpha")
	if len(live) != 1 || live[repo] != changed {
		t.Fatalf("live rows = %v, want exactly the superseding row %d", live, changed)
	}
}

// TestLegacyUnpartitionedRowIsUpgradedInPlace guards AC-3, the v1→v2
// backfill. A row written before the repo column existed carries repo = ”;
// a writer that now stamps a repoKey must upgrade it rather than leave it
// behind as a second live row. This is why the upsert lookup falls back to
// repo = ” instead of matching the exact partition only.
func TestLegacyUnpartitionedRowIsUpgradedInPlace(t *testing.T) {
	const repo = "hero-engine/hero"
	s := openTestStore(t)

	legacy := upsert(t, s, "Feature", "alpha", "", "legacy")
	upgraded := upsert(t, s, "Feature", "alpha", repo, "stamped")

	if upgraded == legacy {
		t.Error("the upgrade should supersede the legacy row, not reuse it")
	}
	live := liveRows(t, s, "Feature", "alpha")
	if len(live) != 1 {
		t.Fatalf("live rows = %v, want exactly 1 — the legacy row should be superseded, not duplicated", live)
	}
	if live[repo] != upgraded {
		t.Errorf("live row = %v, want the repo-stamped row %d", live, upgraded)
	}
}

// TestUnscopedLookupPreservesPreV5Behavior pins the compatibility contract
// for callers that genuinely cannot know their partition: repo == "" matches
// any partition, as it always did, so repo-scoping could be adopted
// per-caller rather than in one flag day.
func TestUnscopedLookupPreservesPreV5Behavior(t *testing.T) {
	s := openTestStore(t)
	peerID := upsert(t, s, "Feature", "only-peer", "hero-engine/hero-cloud", "peer")

	got, err := s.GetNodeID("Feature", "only-peer", "")
	if err != nil {
		t.Fatalf("unscoped lookup should still find a node in any partition: %v", err)
	}
	if got != peerID {
		t.Errorf("unscoped GetNodeID = %d, want %d", got, peerID)
	}

	// A scoped lookup from a different partition must NOT find it.
	if _, err := s.GetNodeID("Feature", "only-peer", "hero-engine/hero"); err != ErrNotFound {
		t.Errorf("scoped lookup found another partition's node: err = %v, want ErrNotFound", err)
	}
}

// TestInvalidateNodeIsPartitionScoped covers the other half of AC-1: an
// invalidate must retire only the named partition's row. Unscoped, the
// UPDATE matched every repo's live row for the key and tombstoned them all.
func TestInvalidateNodeIsPartitionScoped(t *testing.T) {
	const (
		local = "hero-engine/hero"
		peer  = "hero-engine/hero-cloud"
	)
	s := openTestStore(t)
	localID := upsert(t, s, "Feature", "shared", local, "local")
	upsert(t, s, "Feature", "shared", peer, "peer")

	if err := s.InvalidateNode("Feature", "shared", peer); err != nil {
		t.Fatalf("InvalidateNode(peer): %v", err)
	}
	live := liveRows(t, s, "Feature", "shared")
	if len(live) != 1 || live[local] != localID {
		t.Fatalf("live rows = %v, want only the local row %d — invalidate crossed partitions", live, localID)
	}
	if _, ok := live[peer]; ok {
		t.Error("the peer row should be tombstoned")
	}
}

// TestGetNodeAtIsPartitionScoped covers the bitemporal accessor: asking what
// was true at time t must answer from the requested partition, and must keep
// bitemporal ordering as its primary sort rather than partition preference.
func TestGetNodeAtIsPartitionScoped(t *testing.T) {
	const (
		local = "hero-engine/hero"
		peer  = "hero-engine/hero-cloud"
	)
	s := openTestStore(t)
	upsert(t, s, "Feature", "shared", local, "local")
	upsert(t, s, "Feature", "shared", peer, "peer")

	now := nowRFC3339()
	got, err := s.GetNodeAt("Feature", "shared", now, local)
	if err != nil {
		t.Fatalf("GetNodeAt(local): %v", err)
	}
	if got.Repo != local {
		t.Errorf("GetNodeAt(local).Repo = %q, want %q", got.Repo, local)
	}
}

// TestSchemaIndexIsRepoScoped asserts the migration actually landed: the
// live-row unique index must include repo. Without it the schema itself
// forbids two partitions holding the same slug, and no amount of
// application-level scoping can fix that.
func TestSchemaIndexIsRepoScoped(t *testing.T) {
	s := openTestStore(t)
	var sql string
	if err := s.DB().QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='index' AND name='idx_nodes_current'`,
	).Scan(&sql); err != nil {
		t.Fatalf("read idx_nodes_current: %v", err)
	}
	for _, col := range []string{"type", "key", "repo"} {
		if !strings.Contains(sql, col) {
			t.Errorf("idx_nodes_current does not include %q: %s", col, sql)
		}
	}
}

// TestEdgeEndpointsResolveWithinPartition covers AC-4. Once two repos can
// hold live nodes for one slug, an unscoped endpoint lookup would bind an
// edge to whichever partition the query happened to return — silently
// wiring a federated edge into the wrong repo.
func TestEdgeEndpointsResolveWithinPartition(t *testing.T) {
	const (
		local = "hero-engine/hero"
		peer  = "hero-engine/hero-cloud"
	)
	s := openTestStore(t)
	localFrom := upsert(t, s, "Feature", "shared", local, "local-from")
	upsert(t, s, "Feature", "shared", peer, "peer-from")
	localTo := upsert(t, s, "Initiative", "umbrella", local, "local-to")
	upsert(t, s, "Initiative", "umbrella", peer, "peer-to")

	if err := s.MakeAlias("Feature", "shared", "Initiative", "umbrella", local); err != nil {
		t.Fatalf("MakeAlias(local): %v", err)
	}
	var from, to int64
	if err := s.DB().QueryRow(
		`SELECT from_id, to_id FROM edges WHERE type = 'alias_of' AND valid_to IS NULL`,
	).Scan(&from, &to); err != nil {
		t.Fatalf("read alias edge: %v", err)
	}
	if from != localFrom || to != localTo {
		t.Errorf("alias edge bound (%d→%d), want the local partition's nodes (%d→%d)",
			from, to, localFrom, localTo)
	}
}

// TestMigrationV5RepoScopesExistingDatabase covers AC-5: an existing
// graph.db created under the (type, key) index must migrate cleanly, leaving
// no duplicate live rows and no orphaned edges, and must accept a
// second partition's node afterwards — which the old index forbade outright.
func TestMigrationV5RepoScopesExistingDatabase(t *testing.T) {
	dir := t.TempDir()

	// Build a store, then rewind it to the pre-v5 shape: the old
	// (type, key) unique index and schema_version 4.
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	localID := upsert(t, s, "Feature", "team-oauth", "hero-engine/hero", "local")
	for _, stmt := range []string{
		`DROP INDEX IF EXISTS idx_nodes_current`,
		`CREATE UNIQUE INDEX idx_nodes_current ON nodes(type, key) WHERE valid_to IS NULL`,
		`UPDATE meta SET value = '4' WHERE key = 'schema_version'`,
	} {
		if _, err := s.DB().Exec(stmt); err != nil {
			t.Fatalf("rewind to v4 (%s): %v", stmt, err)
		}
	}
	s.Close()

	// Reopening runs the v5 migration against real existing data.
	s2, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen (migrate to v5): %v", err)
	}
	defer s2.Close()

	st, err := s2.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if st.SchemaVersion != schemaVersion {
		t.Errorf("schema_version = %q, want %q", st.SchemaVersion, schemaVersion)
	}

	// The pre-existing row survived the migration untouched...
	live := liveRows(t, s2, "Feature", "team-oauth")
	if len(live) != 1 || live["hero-engine/hero"] != localID {
		t.Fatalf("after migration live rows = %v, want just the original %d", live, localID)
	}
	// ...and a sibling partition is now accepted, which the old unique
	// index made impossible.
	peerID := upsert(t, s2, "Feature", "team-oauth", "hero-engine/hero-cloud", "peer")
	live = liveRows(t, s2, "Feature", "team-oauth")
	if len(live) != 2 || live["hero-engine/hero"] != localID || live["hero-engine/hero-cloud"] != peerID {
		t.Fatalf("after sibling ingest live rows = %v, want one per repo", live)
	}

	// No orphaned edges: every live edge still points at live nodes.
	var orphans int
	if err := s2.DB().QueryRow(`
		SELECT count(*) FROM edges e
		 WHERE e.valid_to IS NULL
		   AND (NOT EXISTS (SELECT 1 FROM nodes n WHERE n.id = e.from_id AND n.valid_to IS NULL)
		     OR NOT EXISTS (SELECT 1 FROM nodes n WHERE n.id = e.to_id   AND n.valid_to IS NULL))`,
	).Scan(&orphans); err != nil {
		t.Fatalf("orphan edge check: %v", err)
	}
	if orphans != 0 {
		t.Errorf("migration left %d orphaned live edges", orphans)
	}
}

// TestUnpartitionedWriteDoesNotClobberStampedNode closes the hole a cold
// audit found in the first cut of this fix: repo-scoping the READ rule was
// not enough, because an upsert with no Repo still preferred a repo-stamped
// row, saw a partition mismatch, and tombstoned it — the original bug
// approached from the unpartitioned side.
//
// This is reachable, not theoretical: production writers in tracker/,
// gitutil/, acceptance/ and extract/ still upsert without stamping a Repo.
func TestUnpartitionedWriteDoesNotClobberStampedNode(t *testing.T) {
	const repo = "hero-engine/hero"
	s := openTestStore(t)

	stamped := upsert(t, s, "Feature", "shared", repo, "stamped")
	unstamped := upsert(t, s, "Feature", "shared", "", "unstamped")

	live := liveRows(t, s, "Feature", "shared")
	if live[repo] != stamped {
		t.Fatalf("an unpartitioned write tombstoned the %q node: live rows = %v, want %d still live",
			repo, live, stamped)
	}
	if live[""] != unstamped {
		t.Errorf("the unpartitioned row should be live in its own partition: %v", live)
	}
}

// TestUnpartitionedWriteWithBothRowsLiveDoesNotError covers repeated
// unpartitioned writes while a stamped row is live for the same key.
//
// Under the read rule this either tombstoned the stamped row or, once a live
// unpartitioned row also existed, collided with it on the v5 unique index.
// Reaching that collision needs state UpsertNode alone cannot produce, so what
// this test actually pins is the reachable half: the write stays confined to
// its own partition and both rows survive.
func TestUnpartitionedWriteWithBothRowsLiveDoesNotError(t *testing.T) {
	const repo = "hero-engine/hero"
	s := openTestStore(t)

	upsert(t, s, "Feature", "shared", repo, "stamped")
	upsert(t, s, "Feature", "shared", "", "unstamped-v1")

	// Second unpartitioned write with different content: under the read rule
	// this reached for the stamped row instead of its own.
	if _, err := s.UpsertNode(&Node{
		Type: "Feature", Domain: "engineering", Key: "shared",
		Props: map[string]any{"title": "unstamped v2"},
		Repo:  "", ContentHash: "unstamped-v2",
		Source: map[string]any{"kind": "test"},
	}); err != nil {
		t.Fatalf("unpartitioned upsert alongside a live stamped row: %v", err)
	}
	if live := liveRows(t, s, "Feature", "shared"); len(live) != 2 {
		t.Fatalf("live rows = %v, want one per partition", live)
	}
}

// TestGetNodeAtPrefersExactPartitionOverRecency pins the bitemporal
// accessor's tie-break. A scoped query asks what was true in THIS repo at t;
// answering it with a newer legacy repo = ” row would be wrong, and the
// first cut of this fix did exactly that because valid_from DESC outranked
// the partition.
func TestGetNodeAtPrefersExactPartitionOverRecency(t *testing.T) {
	const repo = "hero-engine/hero"
	s := openTestStore(t)

	// ValidFrom is set explicitly and far apart. Two back-to-back upserts
	// share a valid_from — nowRFC3339 is second-precision — which leaves no
	// recency gap for the partition preference to win against, so the test
	// passes either way. A cold audit caught exactly that here.
	mustUpsert(t, s, &Node{
		Type: "Feature", Domain: "engineering", Key: "shared",
		Props: map[string]any{"title": "stamped"}, Repo: repo,
		ContentHash: "stamped", ValidFrom: "2015-01-01T00:00:00Z",
		Source: map[string]any{"kind": "test"},
	})
	// Newer by a decade, and unpartitioned: on recency alone it would win.
	mustUpsert(t, s, &Node{
		Type: "Feature", Domain: "engineering", Key: "shared",
		Props: map[string]any{"title": "unstamped"}, Repo: "",
		ContentHash: "unstamped", ValidFrom: "2025-01-01T00:00:00Z",
		Source: map[string]any{"kind": "test"},
	})

	got, err := s.GetNodeAt("Feature", "shared", "2026-01-01T00:00:00Z", repo)
	if err != nil {
		t.Fatalf("GetNodeAt(%q): %v", repo, err)
	}
	if got.Repo != repo {
		t.Errorf("GetNodeAt(%q).Repo = %q — a newer unpartitioned row answered a scoped query",
			repo, got.Repo)
	}
}
