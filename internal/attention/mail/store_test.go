package mail

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
)

func testEnvelope(id string) attention.MailEnvelope {
	return attention.MailEnvelope{SchemaVersion: 1, ID: id, Recipient: attention.ProjectReference{PeerID: "peer_b", DisplayName: "B"}, Sender: attention.ProjectReference{PeerID: "peer_a", DisplayName: "A"}, Subject: "hello", Body: "world", Kind: attention.MailKindNotice, ThreadID: id, Revision: 7, IdempotencyKey: "key-1", CreatedAt: "2026-07-22T18:00:00Z"}
}
func testDelivery(env attention.MailEnvelope) attention.MailDelivery {
	return attention.MailDelivery{SchemaVersion: 1, MessageID: env.ID, ThreadID: env.ThreadID, Sender: env.Sender, Recipient: env.Recipient, IdempotencyKey: env.IdempotencyKey, DeliveredAt: env.CreatedAt}
}

func TestStorePrivateImmutableDeliveryAndReplay(t *testing.T) {
	root := t.TempDir()
	s, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	env := testEnvelope("mail_one")
	d := testDelivery(env)
	got, created, err := s.Deliver(env, d)
	if err != nil || !created || got.MessageID != env.ID {
		t.Fatalf("deliver: %#v %v %v", got, created, err)
	}
	got, created, err = s.Deliver(env, d)
	if err != nil || created || got.MessageID != env.ID {
		t.Fatalf("replay: %#v %v %v", got, created, err)
	}
	for _, p := range []string{filepath.Join(root, "mail", "boxes", "peer_b", "messages", "mail_one.json"), filepath.Join(root, "mail", "outbound", "peer_a", "mail_one.json")} {
		info, e := os.Stat(p)
		if e != nil {
			t.Fatal(e)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("%s mode %o", p, info.Mode().Perm())
		}
	}
	changed := env
	changed.Body = "different"
	if _, _, err := s.Deliver(changed, d); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
	stored, _ := s.Get("peer_b", "mail_one")
	if stored.Body != "world" {
		t.Fatal("immutable envelope was overwritten")
	}
}

func TestStoreRejectsTraversalAndMalformedBeforeWrite(t *testing.T) {
	root := t.TempDir()
	s, _ := NewStore(root)
	env := testEnvelope("mail_one")
	env.Recipient.PeerID = "../escape"
	if _, _, err := s.Deliver(env, testDelivery(env)); err == nil {
		t.Fatal("expected traversal rejection")
	}
	if _, err := os.Stat(filepath.Join(root, "escape")); !os.IsNotExist(err) {
		t.Fatalf("unexpected partial path: %v", err)
	}
	env = testEnvelope("mail_one")
	env.Body = string([]byte{0xff})
	if _, _, err := s.Deliver(env, testDelivery(env)); err == nil {
		t.Fatal("expected malformed body rejection")
	}
	items, _ := s.List("peer_b")
	if len(items) != 0 {
		t.Fatalf("partial delivery: %d", len(items))
	}
}

func TestStoreFailsClosedOnMalformedOutboundState(t *testing.T) {
	for _, record := range []string{
		"{not-json\n",
		`{"schema_version":1,"message_id":"../escape","thread_id":"mail_old","sender":{"peer_id":"peer_a","display_name":"A"},"recipient":{"peer_id":"peer_b","display_name":"B"},"idempotency_key":"old","delivered_at":"2026-07-22T18:00:00Z"}`,
	} {
		root := t.TempDir()
		s, err := NewStore(root)
		if err != nil {
			t.Fatal(err)
		}
		outDir := filepath.Join(root, "mail", "outbound", "peer_a")
		if err := os.MkdirAll(outDir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(outDir, "mail_old.json"), []byte(record), 0o600); err != nil {
			t.Fatal(err)
		}
		env := testEnvelope("mail_new")
		if _, _, err := s.Deliver(env, testDelivery(env)); err == nil {
			t.Fatal("expected corrupt outbound state to fail closed")
		}
		if _, err := os.Stat(filepath.Join(root, "mail", "boxes", "peer_b", "messages", "mail_new.json")); !os.IsNotExist(err) {
			t.Fatalf("partial message delivery: %v", err)
		}
		if _, err := os.Stat(filepath.Join(outDir, "mail_new.json")); !os.IsNotExist(err) {
			t.Fatalf("partial outbound delivery: %v", err)
		}
	}
}

func TestStoreConcurrentReceiptUpdatesPreserveState(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	env := testEnvelope("mail_one")
	_, _, _ = s.Deliver(env, testDelivery(env))
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = s.UpdateReceipt("peer_b", env.ID, env.Revision, func(r *attention.MailReceipt) { r.ReadAt = "2026-07-22T18:01:00Z" })
	}()
	go func() {
		defer wg.Done()
		_, _ = s.UpdateReceipt("peer_b", env.ID, env.Revision, func(r *attention.MailReceipt) { r.AcknowledgedAt = "2026-07-22T18:02:00Z" })
	}()
	wg.Wait()
	r, err := s.Receipt("peer_b", env.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.ReadAt == "" || r.AcknowledgedAt == "" {
		t.Fatalf("lost concurrent update: %#v", r)
	}
	stored, _ := s.Get("peer_b", env.ID)
	if !reflect.DeepEqual(stored, env) {
		t.Fatal("receipt update rewrote envelope")
	}
}
