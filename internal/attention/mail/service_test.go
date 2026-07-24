package mail

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/config"
)

func writePeer(t *testing.T, root, id, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := "schema: 1\ncontracts_version: 1\nrepo:\n  peer_id: " + id + "\n  name: " + name + "\ngenerated_at: 2026-07-22T18:00:00Z\n"
	if err := os.WriteFile(filepath.Join(root, ".hero", "peer-manifest.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
}
func testService(t *testing.T) (*Service, *Store, string, string) {
	t.Helper()
	state := t.TempDir()
	a := t.TempDir()
	b := t.TempDir()
	writePeer(t, b, "peer_b", "B")
	s, _ := NewStore(state)
	svc := NewService(s, a, config.Config{Folder: ".hero", PeerID: "peer_a", Repos: map[string]string{"b": b}, RepoMeta: map[string]config.RepoMetaEntry{"b": {PeerID: "peer_b"}}})
	svc.now = func() time.Time { return time.Date(2026, 7, 22, 18, 0, 0, 0, time.UTC) }
	n := 0
	svc.newID = func() (string, error) { n++; return fmt.Sprintf("mail_%d", n), nil }
	return svc, s, a, b
}

func TestServiceSendReplayConflictAndReceipts(t *testing.T) {
	svc, s, _, b := testService(t)
	req := SendRequest{RecipientAlias: "b", Subject: "Question", Body: "Body", Kind: "future-kind", IdempotencyKey: "retry"}
	first, err := svc.Send(req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Send(req)
	if err != nil {
		t.Fatal(err)
	}
	if first.MessageID != second.MessageID {
		t.Fatalf("retry duplicated: %s %s", first.MessageID, second.MessageID)
	}
	req.Body = "changed"
	if _, err := svc.Send(req); err != ErrIdempotencyConflict {
		t.Fatalf("want idempotency_conflict, got %v", err)
	}
	peerSvc := NewService(s, b, config.Config{Folder: ".hero", PeerID: "peer_b"})
	peerSvc.now = func() time.Time { return time.Date(2026, 7, 22, 18, 1, 0, 0, time.UTC) }
	shown, err := peerSvc.Show(first.MessageID, true)
	if err != nil {
		t.Fatal(err)
	}
	if shown.Kind != "future-kind" || shown.Receipt == nil || shown.Receipt.ReadAt == "" {
		t.Fatalf("show: %#v", shown)
	}
	before, _ := s.Get("peer_b", first.MessageID)
	ack1, err := peerSvc.Ack(first.MessageID, "received")
	if err != nil {
		t.Fatal(err)
	}
	peerSvc.now = func() time.Time { return time.Date(2026, 7, 22, 19, 0, 0, 0, time.UTC) }
	ack2, _ := peerSvc.Ack(first.MessageID, "received")
	if ack1.AcknowledgedAt != ack2.AcknowledgedAt {
		t.Fatal("ack replay changed timestamp")
	}
	ack3, _ := peerSvc.Ack(first.MessageID, "different")
	if ack3.AcknowledgementNote != "received" {
		t.Fatalf("ack replay overwrote first note: %#v", ack3)
	}
	shownAfterAck, err := peerSvc.Show(first.MessageID, true)
	if err != nil {
		t.Fatal(err)
	}
	if shownAfterAck.Receipt.Kind != "acknowledged" || shownAfterAck.Receipt.ReadAt == "" {
		t.Fatalf("show lost acknowledgement: %#v", shownAfterAck.Receipt)
	}
	events, err := os.ReadFile(filepath.Join(b, ".hero", "events.log"))
	if err != nil || !strings.Contains(string(events), `"type":"mail.read"`) || !strings.Contains(string(events), `"type":"mail.acknowledge"`) {
		t.Fatalf("shared triage events missing: %s, %v", events, err)
	}
	after, _ := s.Get("peer_b", first.MessageID)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("receipt operation changed envelope")
	}
}

func TestServiceDirectActionIdentityGuardsAndProvenance(t *testing.T) {
	svc, store, a, b := testService(t)
	writePeer(t, a, "peer_a", "A")
	provenance := []attention.ProvenanceReference{{Kind: "session", SourceID: "session_1"}}
	request := SendRequest{
		RecipientAlias: "b", ExpectedRecipientPeer: "peer_b", Subject: "Question", Body: "Body",
		IdempotencyKey: "guarded-send", Provenance: provenance,
	}
	sent, err := svc.Send(request)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := store.Get("peer_b", sent.MessageID)
	if err != nil || !reflect.DeepEqual(envelope.Provenance, provenance) {
		t.Fatalf("provenance = %#v, %v", envelope.Provenance, err)
	}
	request.IdempotencyKey = "wrong-peer"
	request.ExpectedRecipientPeer = "peer_other"
	if _, err := svc.Send(request); !errors.Is(err, ErrRecipientMismatch) {
		t.Fatalf("recipient guard = %v", err)
	}

	replier := NewService(store, b, config.Config{
		Folder: ".hero", PeerID: "peer_b", Repos: map[string]string{"a": a},
		RepoMeta: map[string]config.RepoMetaEntry{"a": {PeerID: "peer_a"}},
	})
	if _, err := replier.Reply(ReplyRequest{
		MessageID: sent.MessageID, ExpectedThread: "mail_wrong", Body: "answer", IdempotencyKey: "wrong-thread",
	}); !errors.Is(err, ErrThreadMismatch) {
		t.Fatalf("thread guard = %v", err)
	}
	reply, err := replier.Reply(ReplyRequest{
		MessageID: sent.MessageID, ExpectedThread: sent.ThreadID, Body: "answer",
		IdempotencyKey: "guarded-reply", Provenance: provenance,
	})
	if err != nil {
		t.Fatal(err)
	}
	replyEnvelope, err := store.Get("peer_a", reply.MessageID)
	if err != nil || !reflect.DeepEqual(replyEnvelope.Provenance, provenance) {
		t.Fatalf("reply provenance = %#v, %v", replyEnvelope.Provenance, err)
	}
}

func TestServiceResolutionFailsBeforeWriting(t *testing.T) {
	svc, s, _, _ := testService(t)
	if _, err := svc.Send(SendRequest{RecipientAlias: "missing", Subject: "x", Body: "y"}); err == nil {
		t.Fatal("expected missing alias")
	}
	items, _ := s.List("peer_b")
	if len(items) != 0 {
		t.Fatal("partial delivery")
	}
}

func TestServiceInboxPropagatesMalformedReceipt(t *testing.T) {
	svc, store, _, _ := testService(t)
	delivery, err := svc.Send(SendRequest{RecipientAlias: "b", Subject: "x", Body: "y", IdempotencyKey: "receipt-corrupt"})
	if err != nil {
		t.Fatal(err)
	}
	receiptDir := filepath.Join(store.boxes, "peer_b", "receipts")
	if err := os.MkdirAll(receiptDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(receiptDir, delivery.MessageID+".json"), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	receiver := NewService(store, "/peer", config.Config{PeerID: "peer_b"})
	if _, err := receiver.Inbox("", false); err == nil {
		t.Fatal("expected malformed receipt error")
	}
	if _, err := receiver.Show(delivery.MessageID, false); err == nil {
		t.Fatal("expected no-mark-read show to propagate malformed receipt error")
	}
}

func TestServiceReplyPreservesThreadAndRejectsBadTargets(t *testing.T) {
	sender, store, a, b := testService(t)
	writePeer(t, a, "peer_a", "A")
	root, err := sender.Send(SendRequest{RecipientAlias: "b", Subject: "root", Body: "question", Kind: "question", IdempotencyKey: "root-key"})
	if err != nil {
		t.Fatal(err)
	}
	replier := NewService(store, b, config.Config{Folder: ".hero", PeerID: "peer_b", Repos: map[string]string{"a": a}, RepoMeta: map[string]config.RepoMetaEntry{"a": {PeerID: "peer_a"}}})
	replier.now = func() time.Time { return time.Date(2026, 7, 22, 18, 2, 0, 0, time.UTC) }
	replier.newID = func() (string, error) { return "mail_reply", nil }
	reply, err := replier.Reply(ReplyRequest{MessageID: root.MessageID, Body: "answer", IdempotencyKey: "reply-key"})
	if err != nil {
		t.Fatal(err)
	}
	env, err := store.Get("peer_a", reply.MessageID)
	if err != nil {
		t.Fatal(err)
	}
	if env.ThreadID != root.MessageID || env.InReplyTo != root.MessageID || env.Kind != "response" {
		t.Fatalf("reply identity: %#v", env)
	}

	bad := testEnvelope("mail_bad")
	bad.ThreadID = "mail_other"
	bad.InReplyTo = "mail_missing"
	bad.Recipient.PeerID = "peer_b"
	bad.Sender.PeerID = "peer_a"
	bad.IdempotencyKey = "bad-key"
	_, _, _ = store.Deliver(bad, testDelivery(bad))
	if _, err := replier.Reply(ReplyRequest{MessageID: "mail_bad", Body: "x"}); err == nil {
		t.Fatal("expected missing reply target rejection")
	}
	target := testEnvelope("mail_target")
	target.Recipient.PeerID = "peer_b"
	target.Sender.PeerID = "peer_a"
	target.IdempotencyKey = "target-key"
	_, _, _ = store.Deliver(target, testDelivery(target))
	cross := testEnvelope("mail_cross")
	cross.Recipient.PeerID = "peer_b"
	cross.Sender.PeerID = "peer_a"
	cross.ThreadID = "mail_other"
	cross.InReplyTo = "mail_target"
	cross.IdempotencyKey = "cross-key"
	_, _, _ = store.Deliver(cross, testDelivery(cross))
	if _, err := replier.Reply(ReplyRequest{MessageID: "mail_cross", Body: "x"}); err == nil {
		t.Fatal("expected cross-thread reply rejection")
	}
}

func TestServiceEndToEndDoesNotMutateProjects(t *testing.T) {
	svc, s, a, b := testService(t)
	aMarker := filepath.Join(a, "tracked.txt")
	bMarker := filepath.Join(b, "tracked.txt")
	_ = os.WriteFile(aMarker, []byte("a\n"), 0o600)
	_ = os.WriteFile(bMarker, []byte("b\n"), 0o600)
	for _, root := range []string{a, b} {
		for _, args := range [][]string{{"init"}, {"add", "."}, {"-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "baseline"}} {
			cmd := exec.Command("git", args...)
			cmd.Dir = root
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v: %s", args, err, out)
			}
		}
	}
	beforeA, _ := os.ReadFile(aMarker)
	beforeB, _ := os.ReadFile(bMarker)
	d, err := svc.Send(SendRequest{RecipientAlias: "b", Subject: "hello", Body: "private", IdempotencyKey: "e2e"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("peer_b", d.MessageID); err != nil {
		t.Fatal(err)
	}
	afterA, _ := os.ReadFile(aMarker)
	afterB, _ := os.ReadFile(bMarker)
	if string(beforeA) != string(afterA) || string(beforeB) != string(afterB) {
		t.Fatal("project worktree content changed")
	}
	for _, root := range []string{a, b} {
		cmd := exec.Command("git", "status", "--porcelain")
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 0 {
			t.Fatalf("worktree %s mutated: %s", root, out)
		}
	}
}
