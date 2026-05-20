package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPeers_HappyPath(t *testing.T) {
	dir := t.TempDir()
	peerDir := filepath.Join(dir, "peer-repo")
	if err := os.MkdirAll(filepath.Join(peerDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(peerDir, ".hero", "peer-manifest.yaml"),
		[]byte("schema: 1\ncontracts_version: 1\nrepo:\n  peer_id: abc\n  name: peer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"),
		[]byte(`{"repos":{"peer":"`+peerDir+`"},"repo_meta":{"peer":{"peer_id":"abc","scanned_at":"2026-05-01T12:00:00Z"}}}`),
		0o644); err != nil {
		t.Fatal(err)
	}

	out := LoadPeers(PeersInputs{ProjectRoot: dir, HeroDir: heroDir})
	if len(out.Rows) != 1 {
		t.Fatalf("Rows len = %d, want 1", len(out.Rows))
	}
	row := out.Rows[0]
	if row.Alias != "peer" {
		t.Errorf("Alias = %q, want peer", row.Alias)
	}
	if !row.Reachable {
		t.Error("expected Reachable=true (peer manifest exists on disk)")
	}
	if !row.HasScan {
		t.Error("expected HasScan=true (scanned_at parsed)")
	}
	if row.PeerID != "abc" {
		t.Errorf("PeerID = %q, want abc", row.PeerID)
	}
}

func TestLoadPeers_NoPeersConfigured(t *testing.T) {
	dir := t.TempDir()
	// No hero.json — config.Load returns defaults with empty Repos.
	out := LoadPeers(PeersInputs{ProjectRoot: dir, HeroDir: filepath.Join(dir, ".hero")})
	if len(out.Rows) != 0 {
		t.Errorf("Rows len = %d, want 0", len(out.Rows))
	}
}
