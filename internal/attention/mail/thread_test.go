package mail

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailthread"
)

func TestThreadMigrationPreservesReceiptsAndUsesMessageIdentity(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	env := testEnvelope("mail_legacy")
	env.ThreadID = ""
	delivery := testDelivery(env)
	delivery.ThreadID = ""
	if _, _, err := store.Deliver(env, delivery); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateReceipt("peer_b", env.ID, env.Revision, func(receipt *attention.MailReceipt) {
		receipt.ReadAt = "2026-07-22T19:00:00Z"
	}); err != nil {
		t.Fatal(err)
	}
	before, _ := store.Receipt("peer_b", env.ID)
	view, created, err := store.Thread("peer_b", env.ID)
	if err != nil || !created || view.State.Identity.ThreadID != env.ID || view.State.Lifecycle != mailthread.LifecycleOpen || view.Read.UnreadCount != 0 {
		t.Fatalf("migration = %#v, created=%v, err=%v", view, created, err)
	}
	after, _ := store.Receipt("peer_b", env.ID)
	if before.Revision != after.Revision || before.ReadAt != after.ReadAt {
		t.Fatalf("receipt changed during migration: before=%#v after=%#v", before, after)
	}
	data, _ := os.ReadFile(store.threadPath("peer_b", env.ID))
	if strings.Contains(string(data), env.Subject) || strings.Contains(string(data), env.Body) {
		t.Fatalf("thread state copied Mail content: %s", data)
	}
}

func TestThreadMigrationArchivesOnlyFullyDismissedThread(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	for _, id := range []string{"mail_root", "mail_reply"} {
		env := testEnvelope(id)
		env.ThreadID = "mail_root"
		env.IdempotencyKey = id
		if _, _, err := store.Deliver(env, testDelivery(env)); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateReceipt("peer_b", id, env.Revision, func(receipt *attention.MailReceipt) {
			receipt.DismissedAt = "2026-07-22T20:00:00Z"
			receipt.Kind = attention.ReceiptDismissed
		}); err != nil {
			t.Fatal(err)
		}
	}
	view, _, err := store.Thread("peer_b", "mail_root")
	if err != nil || view.State.Lifecycle != mailthread.LifecycleArchived || view.State.ArchivedAt == "" {
		t.Fatalf("fully dismissed migration = %#v, %v", view, err)
	}
}

func TestThreadMigrationKeepsPartiallyDismissedAndAcknowledgedThreadsOpen(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	for _, id := range []string{"mail_partial_root", "mail_partial_reply"} {
		env := testEnvelope(id)
		env.ThreadID = "mail_partial_root"
		env.IdempotencyKey = id
		if _, _, err := store.Deliver(env, testDelivery(env)); err != nil {
			t.Fatal(err)
		}
	}
	root, _ := store.Get("peer_b", "mail_partial_root")
	if _, err := store.UpdateReceipt("peer_b", root.ID, root.Revision, func(receipt *attention.MailReceipt) {
		receipt.DismissedAt = "2026-07-22T20:00:00Z"
		receipt.Kind = attention.ReceiptDismissed
	}); err != nil {
		t.Fatal(err)
	}
	reply, _ := store.Get("peer_b", "mail_partial_reply")
	if _, err := store.UpdateReceipt("peer_b", reply.ID, reply.Revision, func(receipt *attention.MailReceipt) {
		receipt.AcknowledgedAt = "2026-07-22T20:01:00Z"
		receipt.Kind = attention.ReceiptAcknowledged
	}); err != nil {
		t.Fatal(err)
	}
	view, _, err := store.Thread("peer_b", "mail_partial_root")
	if err != nil || view.State.Lifecycle != mailthread.LifecycleOpen || view.State.ArchiveEligibleAt != "" {
		t.Fatalf("partial migration = %#v, %v", view, err)
	}
}

func TestThreadLifecycleActionsAreRevisionedIdempotentAndAtomic(t *testing.T) {
	service, store, env, _ := triageService(t, "mail_thread_action")
	open, _, err := store.Thread("peer_b", env.ThreadID)
	if err != nil {
		t.Fatal(err)
	}
	input := json.RawMessage(`{"reason":"completed","source":"user","outcome":"done"}`)
	request := mailthread.ActionRequest{SchemaVersion: 1, Identity: open.State.Identity, ActionID: mailthread.ActionResolve, ThreadRevision: open.State.Revision, IdempotencyKey: "resolve-1", Input: input}
	resolved, err := service.ThreadAction(request)
	if err != nil || resolved.State.Lifecycle != mailthread.LifecycleResolved || resolved.State.Resolution.Outcome != "done" {
		t.Fatalf("resolve = %#v, %v", resolved, err)
	}
	replay, err := service.ThreadAction(request)
	if err != nil || replay.State.Revision != resolved.State.Revision || len(replay.State.Actions) != 1 {
		t.Fatalf("replay = %#v, %v", replay, err)
	}
	conflict := request
	conflict.Input = json.RawMessage(`{"reason":"different","source":"user"}`)
	if _, err := service.ThreadAction(conflict); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("conflict = %v", err)
	}
	stale := mailthread.ActionRequest{SchemaVersion: 1, Identity: open.State.Identity, ActionID: mailthread.ActionReopen, ThreadRevision: open.State.Revision, IdempotencyKey: "reopen-stale", Input: json.RawMessage(`{}`)}
	if _, err := service.ThreadAction(stale); !errors.Is(err, ErrStale) {
		t.Fatalf("stale = %v", err)
	}
	after, _, _ := store.Thread("peer_b", env.ThreadID)
	if after.State.Revision != resolved.State.Revision || after.State.Lifecycle != mailthread.LifecycleResolved {
		t.Fatalf("failed actions partially mutated state: %#v", after.State)
	}
	info, err := os.Stat(filepath.Join(store.boxes, "peer_b", "threads", env.ThreadID+".json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("thread state permissions = %v, %v", info, err)
	}
}

func TestConcurrentThreadActionsAcceptOneRevisionAndPreserveValidState(t *testing.T) {
	service, store, env, _ := triageService(t, "mail_thread_concurrent")
	open, _, _ := store.Thread("peer_b", env.ThreadID)
	requests := []mailthread.ActionRequest{
		{SchemaVersion: 1, Identity: open.State.Identity, ActionID: mailthread.ActionResolve, ThreadRevision: open.State.Revision, IdempotencyKey: "concurrent-resolve", Input: json.RawMessage(`{"reason":"done","source":"user"}`)},
		{SchemaVersion: 1, Identity: open.State.Identity, ActionID: mailthread.ActionArchive, ThreadRevision: open.State.Revision, IdempotencyKey: "concurrent-archive", Input: json.RawMessage(`{}`)},
	}
	var wg sync.WaitGroup
	results := make(chan error, len(requests))
	for _, request := range requests {
		wg.Add(1)
		go func(request mailthread.ActionRequest) {
			defer wg.Done()
			_, err := service.ThreadAction(request)
			results <- err
		}(request)
	}
	wg.Wait()
	close(results)
	var succeeded, stale int
	for err := range results {
		if err == nil {
			succeeded++
		} else if errors.Is(err, ErrStale) {
			stale++
		} else {
			t.Fatalf("unexpected concurrent error: %v", err)
		}
	}
	view, _, err := store.Thread("peer_b", env.ThreadID)
	if err != nil || succeeded != 1 || stale != 1 || len(view.State.Actions) != 1 || mailthread.ValidateState(view.State) != nil {
		t.Fatalf("concurrent result: succeeded=%d stale=%d view=%#v err=%v", succeeded, stale, view, err)
	}
}

func TestMarkUnreadChangesReceiptWithoutChangingLifecycle(t *testing.T) {
	service, store, env, _ := triageService(t, "mail_unread")
	service.now = func() time.Time { return time.Date(2026, 7, 22, 20, 0, 0, 0, time.UTC) }
	read, err := service.Action(ActionRequest{MessageID: env.ID, Action: ActionRead, ExpectedRevision: 0, IdempotencyKey: "read"})
	if err != nil {
		t.Fatal(err)
	}
	before, _, _ := store.Thread("peer_b", env.ThreadID)
	unread, err := service.Action(ActionRequest{MessageID: env.ID, Action: ActionUnread, ExpectedRevision: read.Receipt.Revision, IdempotencyKey: "unread"})
	if err != nil || unread.Receipt.ReadAt != "" {
		t.Fatalf("mark unread = %#v, %v", unread, err)
	}
	after, _, _ := store.Thread("peer_b", env.ThreadID)
	if before.State.Revision != after.State.Revision || after.State.Lifecycle != mailthread.LifecycleOpen || after.Read.UnreadCount != 1 {
		t.Fatalf("read/lifecycle coupling: before=%#v after=%#v", before, after)
	}
}
