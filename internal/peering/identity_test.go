package peering

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

// TestMintPeerIDFormat confirms MintPeerID returns a non-empty UUID
// string in the canonical hyphenated form.
func TestMintPeerIDFormat(t *testing.T) {
	id := MintPeerID()
	if id == "" {
		t.Fatal("MintPeerID returned empty string")
	}
	if len(id) != 36 {
		t.Fatalf("expected 36-char UUID, got %d (%q)", len(id), id)
	}
	if strings.Count(id, "-") != 4 {
		t.Fatalf("expected 4 hyphens in canonical UUID, got %q", id)
	}
	// Two consecutive mints should differ.
	if MintPeerID() == id {
		t.Fatal("two consecutive mints returned the same UUID")
	}
}

// TestEnsurePeerIDIdempotent checks that EnsurePeerID mints once and
// returns the same id (minted=false) on every subsequent invocation
// against the same workspace.
func TestEnsurePeerIDIdempotent(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("mkdir hero dir: %v", err)
	}
	cfg := config.DefaultConfig()
	if err := cfg.Save(root); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	first, minted, err := EnsurePeerID(root, "test")
	if err != nil {
		t.Fatalf("first EnsurePeerID: %v", err)
	}
	if !minted {
		t.Fatal("first EnsurePeerID should report minted=true")
	}
	if first == "" {
		t.Fatal("first EnsurePeerID returned empty peer_id")
	}

	second, minted2, err := EnsurePeerID(root, "test")
	if err != nil {
		t.Fatalf("second EnsurePeerID: %v", err)
	}
	if minted2 {
		t.Fatal("second EnsurePeerID should report minted=false")
	}
	if second != first {
		t.Fatalf("peer_id changed across calls: %q vs %q", first, second)
	}

	// Confirm the events log captured a single mint event.
	logData, err := os.ReadFile(filepath.Join(heroDir, "events.log"))
	if err != nil {
		t.Fatalf("read events log: %v", err)
	}
	mints := strings.Count(string(logData), "workspace.peer_id_minted")
	if mints != 1 {
		t.Fatalf("expected exactly 1 mint event, got %d in log:\n%s", mints, logData)
	}
}
