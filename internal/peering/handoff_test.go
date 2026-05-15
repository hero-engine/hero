package peering

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// setupWorkspace materializes a minimal hero workspace with a given
// peer_id and returns the project root.
func setupWorkspace(t *testing.T, root, peerID string) {
	t.Helper()
	heroDir := filepath.Join(root, ".hero")
	for _, d := range []string{
		heroDir,
		filepath.Join(heroDir, "planning", "features"),
		filepath.Join(heroDir, "planning", "bugs"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	cfg := config.DefaultConfig()
	cfg.PeerID = peerID
	if err := cfg.Save(root); err != nil {
		t.Fatalf("save cfg %s: %v", root, err)
	}
}

func writeFeatureSpec(t *testing.T, root, slug, title string) string {
	t.Helper()
	dir := filepath.Join(root, ".hero", "planning", "features", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir spec dir: %v", err)
	}
	path := filepath.Join(dir, "spec.md")
	content := "---\ntitle: \"" + title + "\"\ntype: feature\nstatus: delivering\n---\n\n# " + title + "\n\n## Goal\n\nDo the thing.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	return path
}

// TestHandoffHappyPath verifies the basic two-side write: receiver
// scaffold appears, originator status flips, both sides carry trail
// entries with peer_id as the join key.
func TestHandoffHappyPath(t *testing.T) {
	root := t.TempDir()
	originRoot := filepath.Join(root, "client")
	peerRoot := filepath.Join(root, "app")

	const originPeerID = "11111111-1111-4111-8111-111111111111"
	const peerPeerID = "22222222-2222-4222-8222-222222222222"

	setupWorkspace(t, originRoot, originPeerID)
	setupWorkspace(t, peerRoot, peerPeerID)

	// Register the peer in the originator's hero.json.
	cfg, err := config.Load(originRoot)
	if err != nil {
		t.Fatalf("load origin cfg: %v", err)
	}
	cfg.Repos = map[string]string{"app": peerRoot}
	if err := cfg.Save(originRoot); err != nil {
		t.Fatalf("save origin cfg: %v", err)
	}

	writeFeatureSpec(t, originRoot, "order-failure-error-display", "Order failure error display")

	at := time.Date(2026, 5, 15, 14, 0, 0, 0, time.UTC)
	res, err := Handoff(originRoot, "order-failure-error-display", HandoffOptions{
		PeerAlias: "app",
		Reason:    "Symptom is in the client, root cause is the API response shape.",
		At:        at,
	})
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	if res.PeerSlug != "order-failure-error-display" {
		t.Errorf("unexpected peer slug %q", res.PeerSlug)
	}
	if res.PeerID != peerPeerID {
		t.Errorf("peer_id mismatch: %q vs %q", res.PeerID, peerPeerID)
	}

	// Receiver-side checks.
	peerData, err := os.ReadFile(res.PeerPath)
	if err != nil {
		t.Fatalf("read peer spec: %v", err)
	}
	peerSpec, err := spec.Parse(string(peerData), res.PeerPath, time.Now())
	if err != nil {
		t.Fatalf("parse peer spec: %v", err)
	}
	if peerSpec.Status != spec.StatusPlanning {
		t.Errorf("peer status: want planning, got %q", peerSpec.Status)
	}
	if peerSpec.ReceivedFrom == nil {
		t.Fatal("peer spec missing received_from block")
	}
	if peerSpec.ReceivedFrom.PeerID != originPeerID {
		t.Errorf("peer received_from.peer_id: want %q, got %q", originPeerID, peerSpec.ReceivedFrom.PeerID)
	}
	if peerSpec.ReceivedFrom.OriginatorSlug != "order-failure-error-display" {
		t.Errorf("peer received_from.originator_slug: %q", peerSpec.ReceivedFrom.OriginatorSlug)
	}
	peerTrail, err := ReadTrail(res.PeerPath)
	if err != nil {
		t.Fatalf("read peer trail: %v", err)
	}
	if len(peerTrail) != 1 {
		t.Fatalf("peer trail: want 1 entry, got %d", len(peerTrail))
	}
	if peerTrail[0].Direction != "in" {
		t.Errorf("peer trail direction: %q", peerTrail[0].Direction)
	}
	if peerTrail[0].PeerID != originPeerID {
		t.Errorf("peer trail peer_id: %q", peerTrail[0].PeerID)
	}

	// Originator-side checks.
	originPath := filepath.Join(originRoot, ".hero", "planning", "features", "order-failure-error-display", "spec.md")
	originData, err := os.ReadFile(originPath)
	if err != nil {
		t.Fatalf("read origin spec: %v", err)
	}
	originSpec, err := spec.Parse(string(originData), originPath, time.Now())
	if err != nil {
		t.Fatalf("parse origin spec: %v", err)
	}
	if originSpec.Status != spec.StatusHandedOff {
		t.Errorf("origin status: want %q, got %q", spec.StatusHandedOff, originSpec.Status)
	}
	originTrail, err := ReadTrail(originPath)
	if err != nil {
		t.Fatalf("read origin trail: %v", err)
	}
	if len(originTrail) != 1 {
		t.Fatalf("origin trail: want 1 entry, got %d", len(originTrail))
	}
	if originTrail[0].Direction != "out" {
		t.Errorf("origin trail direction: %q", originTrail[0].Direction)
	}
	if originTrail[0].PeerID != peerPeerID {
		t.Errorf("origin trail peer_id: %q vs %q", originTrail[0].PeerID, peerPeerID)
	}

	// Event log on both sides.
	originLog, _ := os.ReadFile(filepath.Join(originRoot, ".hero", "events.log"))
	if !strings.Contains(string(originLog), "peer.handoff.sent") {
		t.Errorf("origin events.log missing peer.handoff.sent: %s", originLog)
	}
	peerLog, _ := os.ReadFile(filepath.Join(peerRoot, ".hero", "events.log"))
	if !strings.Contains(string(peerLog), "peer.handoff.received") {
		t.Errorf("peer events.log missing peer.handoff.received: %s", peerLog)
	}
}

// TestHandoffSlugCollision verifies the -2, -3 suffix logic when the
// receiver already has a spec of the originator's slug.
func TestHandoffSlugCollision(t *testing.T) {
	root := t.TempDir()
	originRoot := filepath.Join(root, "client")
	peerRoot := filepath.Join(root, "app")

	setupWorkspace(t, originRoot, "11111111-1111-4111-8111-111111111111")
	setupWorkspace(t, peerRoot, "22222222-2222-4222-8222-222222222222")

	cfg, _ := config.Load(originRoot)
	cfg.Repos = map[string]string{"app": peerRoot}
	_ = cfg.Save(originRoot)

	// Pre-create a colliding peer spec.
	collide := filepath.Join(peerRoot, ".hero", "planning", "features", "shared-slug")
	if err := os.MkdirAll(collide, 0o755); err != nil {
		t.Fatalf("mkdir collide: %v", err)
	}
	if err := os.WriteFile(filepath.Join(collide, "spec.md"), []byte("---\ntitle: existing\n---\n"), 0o644); err != nil {
		t.Fatalf("write collide spec: %v", err)
	}

	writeFeatureSpec(t, originRoot, "shared-slug", "Shared")
	res, err := Handoff(originRoot, "shared-slug", HandoffOptions{PeerAlias: "app"})
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	if res.PeerSlug != "shared-slug-2" {
		t.Errorf("expected slug %q, got %q", "shared-slug-2", res.PeerSlug)
	}
}

// TestHandoffMissingPeer verifies the error path when the peer alias
// isn't configured.
func TestHandoffMissingPeer(t *testing.T) {
	root := t.TempDir()
	originRoot := filepath.Join(root, "client")
	setupWorkspace(t, originRoot, "11111111-1111-4111-8111-111111111111")
	writeFeatureSpec(t, originRoot, "x", "X")

	_, err := Handoff(originRoot, "x", HandoffOptions{PeerAlias: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing peer alias")
	}
}
