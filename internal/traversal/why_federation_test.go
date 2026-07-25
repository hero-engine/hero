package traversal

import "testing"

// TestResolveTarget_FederatedPeerCopyDoesNotShadowLocal pins the
// federation-scoping edge from graph-why-resolution-and-peer-spec-indexing
// (the team-oauth case).
//
// Node identity is (type, key, repo), so a federated peer scan that ingests
// a slug under its own repoKey leaves the local partition's copy live
// alongside it. resolveTarget must:
//
//   - return the LOCAL node when it is live, never a peer copy, and
//   - keep doing so after a sibling ingest of the same slug — a peer copy
//     neither shadows (masquerades as) nor tombstones the local one.
//
// This test previously characterized the OPPOSITE, because identity was
// (type, key) alone: the sibling ingest tombstoned the local node, and the
// test asserted that a local query then failed. That was the team-oauth
// bug, pinned as if it were the contract. Repo-scoped identity is what
// makes the wished-for behavior in the bullets above actually true.
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
	peerID := seedNode(t, store, "Feature", slug, "Peer Team OAuth", peerRepo)
	if peerID == localID {
		t.Fatalf("peer ingest reused the local node id %d — identity is not repo-scoped", localID)
	}

	// The local node must still be live and still be what a local query
	// resolves to. Before repo-scoped identity the sibling ingest tombstoned
	// it and this query failed outright.
	hop, gotID, err = resolveTarget(store, localRepo, slug)
	if err != nil {
		t.Fatalf("a sibling ingest tombstoned the local node: %v", err)
	}
	if gotID != localID {
		t.Errorf("resolved id = %d, want the local id %d — a peer copy shadowed the local partition", gotID, localID)
	}
	if hop.NodeTitle != "Local Team OAuth" {
		t.Errorf("resolved title = %q, want the local node's title", hop.NodeTitle)
	}

	// The peer repo, of course, still resolves its own copy.
	if _, _, err := resolveTarget(store, peerRepo, slug); err != nil {
		t.Errorf("resolveTarget(peer) should resolve the peer's own live copy: %v", err)
	}
}

// TestResolveTarget_PeerOnlyNodeDoesNotAnswerLocalQuery keeps the second guard
// the original version of the test above carried: when a slug exists ONLY in a
// peer partition, a local-repo query must refuse it rather than hand back the
// peer's node.
//
// The original asserted this via the tombstoned state — after a sibling ingest
// the local node was gone, so a local query failed. Repo-scoped identity
// removed that state, and rewriting the test above for the new behavior
// dropped this guard with it. The guard itself is still meaningful and still
// holds: a slug only the peer owns is a real case, independent of tombstoning.
func TestResolveTarget_PeerOnlyNodeDoesNotAnswerLocalQuery(t *testing.T) {
	const (
		localRepo = "hero-engine/hero"
		peerRepo  = "hero-engine/hero-cloud"
		slug      = "peer-only-spec"
	)

	store := openStore(t)
	seedNode(t, store, "Feature", slug, "Peer Only Spec", peerRepo)

	if _, _, err := resolveTarget(store, localRepo, slug); err == nil {
		t.Error("resolveTarget(local) returned a peer-only node as if it were local; " +
			"a federated copy must not shadow the local partition")
	}
	// And the peer's own query still finds it.
	if _, _, err := resolveTarget(store, peerRepo, slug); err != nil {
		t.Errorf("resolveTarget(peer) should resolve its own node: %v", err)
	}
}
