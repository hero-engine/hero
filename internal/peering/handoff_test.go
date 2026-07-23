package peering

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

func setupWorkspace(t *testing.T, root, peerID string) {
	t.Helper()
	for _, dir := range []string{
		filepath.Join(root, ".hero", "planning", "features"),
		filepath.Join(root, ".hero", "planning", "bugs"),
		filepath.Join(root, ".hero", "planning", "intake"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.DefaultConfig()
	cfg.PeerID = peerID
	if err := cfg.Save(root); err != nil {
		t.Fatal(err)
	}
}

// AC-3
func TestHandoffDismissCreatesNoWork(t *testing.T) {
	origin, peer, state := peerMailFixture(t)
	writeFeatureSpec(t, origin, "order-errors", "Order errors")
	sent, err := Handoff(origin, "order-errors", HandoffOptions{PeerAlias: "app", StateRoot: state})
	if err != nil {
		t.Fatal(err)
	}
	before := treeSnapshot(t, filepath.Join(peer, ".hero", "planning"))
	cfg, _ := config.Load(peer)
	svc, _ := projectMailService(peer, state, cfg)
	if _, err := svc.Action(mail.ActionRequest{
		MessageID: sent.MessageID, Action: mail.ActionDismiss, IdempotencyKey: "dismiss-1",
	}); err != nil {
		t.Fatal(err)
	}
	if after := treeSnapshot(t, filepath.Join(peer, ".hero", "planning")); after != before {
		t.Fatal("dismiss created receiver work")
	}
}

func writeFeatureSpec(t *testing.T, root, slug, title string) string {
	t.Helper()
	dir := filepath.Join(root, ".hero", "planning", "features", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "spec.md")
	content := "---\ntitle: \"" + title + "\"\nslug: " + slug + "\ntype: feature\nstatus: delivering\n---\n\n# " + title + "\n\n## Goal\n\nDo the thing.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeMinimalPeerManifest(t *testing.T, root, peerID, name string) {
	t.Helper()
	content := "schema: 1\ncontracts_version: 1\nrepo:\n  peer_id: " + peerID + "\n  name: " + name + "\n"
	if err := os.WriteFile(filepath.Join(root, ".hero", PeerManifestFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// AC-3, AC-9
func TestHandoffSendsMailWithoutChangingEitherSpecTree(t *testing.T) {
	origin, peer, state := peerMailFixture(t)
	originSpec := writeFeatureSpec(t, origin, "order-errors", "Order errors")
	beforeOrigin, _ := os.ReadFile(originSpec)
	beforePeer := treeSnapshot(t, peer)
	res, err := Handoff(origin, "order-errors", HandoffOptions{PeerAlias: "app", Reason: "backend owns it", StateRoot: state})
	if err != nil {
		t.Fatal(err)
	}
	if res.MessageID == "" || res.Status != "queued" || res.PeerPath != "" {
		t.Fatalf("unexpected handoff: %+v", res)
	}
	afterOrigin, _ := os.ReadFile(originSpec)
	if string(afterOrigin) != string(beforeOrigin) {
		t.Fatal("origin spec status/content changed")
	}
	if afterPeer := treeSnapshot(t, peer); afterPeer != beforePeer {
		t.Fatal("receiver checkout changed before explicit receive")
	}
}

// AC-4, AC-5
func TestReceivePromotesOnceAndReplies(t *testing.T) {
	origin, peer, state := peerMailFixture(t)
	writeFeatureSpec(t, origin, "order-errors", "Order errors")
	sent, err := Handoff(origin, "order-errors", HandoffOptions{PeerAlias: "app", StateRoot: state, IdempotencyKey: "transfer-1"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := Receive(peer, sent.MessageID, ReceiveOptions{Type: "feature", StateRoot: state, IdempotencyKey: "receive-1"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Receive(peer, sent.MessageID, ReceiveOptions{Type: "feature", StateRoot: state, IdempotencyKey: "receive-1"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Artifact == nil || first.Artifact.Slug == "" || second.Artifact.Slug != first.Artifact.Slug || !second.Replayed {
		t.Fatalf("receive not idempotent: first=%+v second=%+v", first, second)
	}
	found, _ := spec.Discover(filepath.Join(peer, ".hero"))
	count := 0
	for _, candidate := range found {
		if candidate.Slug == first.Artifact.Slug && candidate.Type == spec.TypeFeature {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("promoted artifacts=%d, want 1", count)
	}
}
