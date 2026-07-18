package traversal

import "testing"

// TestResolveTarget_FederatedPeerCopyDoesNotShadowLocal pins the
// federation-scoping edge from graph-why-resolution-and-peer-spec-indexing
// (the team-oauth case).
//
// The graph enforces a single live node per (type, key) across all repos,
// so a federated peer scan that ingests a slug under its own repoKey
// tombstones the local partition's copy. resolveTarget must:
//
//   - return the LOCAL node when it is live (never a peer copy), and
//   - refuse to answer a local-repo query with a peer-only live node — a
//     sibling repo's copy does not "shadow" (masquerade as) the local one.
//
// The read-side reconcile on hero why is what keeps the local node live at
// query time; this test documents the scoping resolveTarget relies on.
func TestResolveTarget_FederatedPeerCopyDoesNotShadowLocal(t *testing.T) {
	const (
		localRepo = "hero-engine/hero"
		peerRepo  = "hero-engine/hero-cloud"
		slug      = "team-oauth"
	)

	store := openStore(t)

	// Local copy live → resolves to the local node.
	localID := seedNode(t, store, "Feature", slug, "Local Team OAuth", localRepo)
	hop, gotID, err := resolveTarget(store, localRepo, slug)
	if err != nil {
		t.Fatalf("resolveTarget(local) with a live local node: %v", err)
	}
	if gotID != localID {
		t.Fatalf("resolved id = %d, want local id %d", gotID, localID)
	}
	if hop.NodeTitle != "Local Team OAuth" {
		t.Errorf("resolved title = %q, want the local node's title", hop.NodeTitle)
	}

	// A federated peer scan ingests the same slug under the peer repoKey.
	// The single-live-node-per-key invariant tombstones the local copy.
	seedNode(t, store, "Feature", slug, "Peer Team OAuth", peerRepo)

	// resolveTarget scoped to the local repo must NOT return the peer copy:
	// a sibling repo's node does not shadow the local partition.
	if _, _, err := resolveTarget(store, localRepo, slug); err == nil {
		t.Errorf("resolveTarget(local) returned a peer-only node as if it were local; a federated copy must not shadow the local partition")
	}

	// The peer repo, of course, still resolves its own copy.
	if _, _, err := resolveTarget(store, peerRepo, slug); err != nil {
		t.Errorf("resolveTarget(peer) should resolve the peer's own live copy: %v", err)
	}
}
