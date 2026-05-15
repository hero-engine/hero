package spec

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCrossRepoResolverPeerIDDualKey confirms the resolver maps
// alias↔peer_id in both directions and resolves correctly by peer_id.
func TestCrossRepoResolverPeerIDDualKey(t *testing.T) {
	root := t.TempDir()
	peerRoot := filepath.Join(root, "peer-repo")
	heroDir := filepath.Join(peerRoot, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("mkdir peer hero: %v", err)
	}
	// Write a minimal hero.json with peer_id.
	const wantPeerID = "9c1c2f3e-4a8b-4f9d-9a0e-7e1f0d8c3a55"
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"),
		[]byte(`{"folder":".hero","peer_id":"`+wantPeerID+`"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write peer hero.json: %v", err)
	}

	r := NewCrossRepoResolver(map[string]string{
		"app": peerRoot,
	}, ".hero")

	if got := r.PeerIDForAlias("app"); got != wantPeerID {
		t.Fatalf("PeerIDForAlias: want %q, got %q", wantPeerID, got)
	}
	if got := r.AliasForPeerID(wantPeerID); got != "app" {
		t.Fatalf("AliasForPeerID: want %q, got %q", "app", got)
	}
	// Unknown peer_id returns empty.
	if got := r.AliasForPeerID("00000000-0000-0000-0000-000000000000"); got != "" {
		t.Fatalf("AliasForPeerID unknown: want empty, got %q", got)
	}
}

// TestCrossRepoResolverPeerIDPrePopulated confirms WithPeerIDs
// short-circuits the on-disk lookup.
func TestCrossRepoResolverPeerIDPrePopulated(t *testing.T) {
	r := NewCrossRepoResolver(map[string]string{"x": "/nonexistent"}, ".hero")
	r.WithPeerIDs(map[string]string{"x": "abc-123"})
	if got := r.PeerIDForAlias("x"); got != "abc-123" {
		t.Fatalf("pre-populated lookup: want %q, got %q", "abc-123", got)
	}
	if got := r.AliasForPeerID("abc-123"); got != "x" {
		t.Fatalf("reverse pre-populated lookup: want %q, got %q", "x", got)
	}
}
