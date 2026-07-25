package peering

import (
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
)

// liveNodeCount reports how many live graph nodes carry the slug, scoped to
// the partition a reader would filter on. This is the same predicate
// traversal.resolveTarget applies, so a count of zero here is exactly the
// "no node with key" failure `hero why` reported.
func liveNodeCount(t *testing.T, projectRoot, slug string) int {
	t.Helper()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	store, err := graph.Open(cfg.HeroDir(projectRoot))
	if err != nil {
		t.Fatalf("open graph: %v", err)
	}
	defer store.Close()
	var n int
	if err := store.DB().QueryRow(
		`SELECT count(*) FROM nodes WHERE key = ? AND repo = ? AND valid_to IS NULL`,
		slug, gitutil.RepoKey(projectRoot),
	).Scan(&n); err != nil {
		t.Fatalf("count nodes: %v", err)
	}
	return n
}

// TestReceivedSpecIsGraphVisible covers the write-time half of the
// staleness fix: a handed-off spec must be resolvable in the graph
// substrate as soon as the receive completes, with no separate ingest.
//
// This is the shape that broke — peer-received specs landed on disk (and so
// in the self-healing index) but never in graph.db, so `hero graph` found
// them while `hero why` reported "no node with key". The node must also
// land in the LOCAL repo partition, not a peer one, or the reader's filter
// misses it just the same.
func TestReceivedSpecIsGraphVisible(t *testing.T) {
	origin, peer, state := peerMailFixture(t)
	writeFeatureSpec(t, origin, "order-errors", "Order errors")
	sent, err := Handoff(origin, "order-errors", HandoffOptions{
		PeerAlias: "app", StateRoot: state, IdempotencyKey: "transfer-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := Receive(peer, sent.MessageID, ReceiveOptions{
		Type: "feature", StateRoot: state, IdempotencyKey: "receive-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Artifact == nil || res.Artifact.Slug == "" {
		t.Fatalf("receive produced no artifact: %+v", res)
	}

	if got := liveNodeCount(t, peer, res.Artifact.Slug); got == 0 {
		t.Fatalf("received spec %q has no live graph node in repo %q — "+
			"it would be invisible to `hero why` until an unrelated ingest ran",
			res.Artifact.Slug, gitutil.RepoKey(peer))
	}
}

// TestReceivedSpecGraphIngestIsIdempotent guards the replay path: receiving
// the same transfer twice must not multiply live nodes. Receive is already
// idempotent on the mail side and spec.WriteGraph upserts, so the live count
// must be identical across the two receives.
//
// The count is asserted as "unchanged" rather than a literal, because a
// promoted transfer legitimately produces two live nodes for one slug — an
// Intake and the Feature it promoted to, the provenance chain `hero why`
// walks back through. Pinning a literal would encode that pair as if it were
// the thing under test.
func TestReceivedSpecGraphIngestIsIdempotent(t *testing.T) {
	origin, peer, state := peerMailFixture(t)
	writeFeatureSpec(t, origin, "order-errors", "Order errors")
	sent, err := Handoff(origin, "order-errors", HandoffOptions{
		PeerAlias: "app", StateRoot: state, IdempotencyKey: "transfer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := Receive(peer, sent.MessageID, ReceiveOptions{
		Type: "feature", StateRoot: state, IdempotencyKey: "receive-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	before := liveNodeCount(t, peer, first.Artifact.Slug)
	if before == 0 {
		t.Fatalf("no live node for %q after the first receive", first.Artifact.Slug)
	}
	if _, err := Receive(peer, sent.MessageID, ReceiveOptions{
		Type: "feature", StateRoot: state, IdempotencyKey: "receive-1",
	}); err != nil {
		t.Fatal(err)
	}
	if after := liveNodeCount(t, peer, first.Artifact.Slug); after != before {
		t.Fatalf("live nodes for %q went %d\u2192%d on a replayed receive; the ingest is not idempotent",
			first.Artifact.Slug, before, after)
	}
}

// TestIngestPromotedSpecIsBestEffort pins the contract that a receive is
// never failed by the graph write. The guard clauses must absorb a slug that
// resolves to nothing on disk and an empty slug, writing nothing and
// returning quietly rather than panicking.
func TestIngestPromotedSpecIsBestEffort(t *testing.T) {
	_, peer, _ := peerMailFixture(t)
	cfg, err := config.Load(peer)
	if err != nil {
		t.Fatal(err)
	}
	ingestPromotedSpec(peer, cfg, "")
	ingestPromotedSpec(peer, cfg, "no-such-spec-on-disk")

	if got := liveNodeCount(t, peer, "no-such-spec-on-disk"); got != 0 {
		t.Fatalf("live nodes for an unknown slug = %d, want 0", got)
	}
}
