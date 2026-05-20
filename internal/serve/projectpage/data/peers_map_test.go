package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPeersMap_Empty(t *testing.T) {
	out := LoadPeersMap(PeersMapInputs{})
	if len(out.Rows) != 0 {
		t.Errorf("Rows len = %d, want 0", len(out.Rows))
	}
}

func TestLoadPeersMap_ResolvesPeerProjectByPath(t *testing.T) {
	srcDir := mkProject(t, "src")
	peerDir := mkProject(t, "peer")
	// Drop a peer-manifest.yaml inside the peer's .hero so the row
	// reports ManifestExists + Reachable=true.
	if err := os.WriteFile(filepath.Join(peerDir, ".hero", "peer-manifest.yaml"),
		[]byte("schema: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, ".hero", "hero.json"),
		[]byte(`{"repos":{"peer":"`+peerDir+`"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out := LoadPeersMap(PeersMapInputs{Projects: []DirectoryProject{
		{Slug: "src", ProjectRoot: srcDir, HeroDir: filepath.Join(srcDir, ".hero")},
		{Slug: "peer", ProjectRoot: peerDir, HeroDir: filepath.Join(peerDir, ".hero")},
	}})
	if len(out.Rows) != 1 {
		t.Fatalf("Rows len = %d, want 1; rows=%+v", len(out.Rows), out.Rows)
	}
	row := out.Rows[0]
	if row.SourceProject != "src" || row.PeerAlias != "peer" {
		t.Errorf("row identity = %s/%s, want src/peer", row.SourceProject, row.PeerAlias)
	}
	if row.PeerProject != "peer" {
		t.Errorf("PeerProject = %q, want peer (resolved by path)", row.PeerProject)
	}
	if !row.Reachable || !row.ManifestExists {
		t.Errorf("expected reachable+manifest-exists; got reachable=%v manifest=%v", row.Reachable, row.ManifestExists)
	}
}

func TestLoadPeersMap_UnresolvedPeerLeavesProjectEmpty(t *testing.T) {
	srcDir := mkProject(t, "src")
	stray := filepath.Join(t.TempDir(), "stray-peer")
	if err := os.WriteFile(filepath.Join(srcDir, ".hero", "hero.json"),
		[]byte(`{"repos":{"ghost":"`+stray+`"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out := LoadPeersMap(PeersMapInputs{Projects: []DirectoryProject{
		{Slug: "src", ProjectRoot: srcDir, HeroDir: filepath.Join(srcDir, ".hero")},
	}})
	if len(out.Rows) != 1 {
		t.Fatalf("Rows len = %d, want 1", len(out.Rows))
	}
	if out.Rows[0].PeerProject != "" {
		t.Errorf("PeerProject = %q, want empty (not registered)", out.Rows[0].PeerProject)
	}
}
