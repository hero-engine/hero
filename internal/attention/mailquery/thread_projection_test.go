package mailquery

import (
	"encoding/json"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailread"
	"github.com/hero-engine/hero/contracts/attention/mailthread"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/config"
)

func TestThreadProjectionBucketsCountsHistoryAndStablePagination(t *testing.T) {
	state := t.TempDir()
	projectA, projectB := t.TempDir(), t.TempDir()
	writeQueryProject(t, projectA, "peer_a", "Alpha", nil)
	writeQueryProject(t, projectB, "peer_b", "Beta", nil)
	store, _ := mail.NewStore(state)
	deliverThreadMessage(t, store, "peer_a", "mail_request_1", "shared_thread", attention.MailKindRequest, "2026-08-17T10:00:00Z")
	deliverThreadMessage(t, store, "peer_a", "mail_request_2", "shared_thread", attention.MailKindRequest, "2026-08-17T10:01:00Z")
	deliverThreadMessage(t, store, "peer_a", "mail_request_status", "shared_thread", attention.MailKindNotice, "2026-08-17T10:01:30Z")
	deliverThreadMessage(t, store, "peer_a", "mail_notice", "updates_thread", attention.MailKindNotice, "2026-08-17T10:02:00Z")
	deliverThreadMessage(t, store, "peer_a", "mail_notice_2", "updates_thread", attention.MailKindNotice, "2026-08-17T10:02:30Z")
	deliverThreadMessage(t, store, "peer_a", "mail_archive", "history_thread", attention.MailKindRequest, "2026-08-17T10:03:00Z")
	deliverThreadMessage(t, store, "peer_a", "mail_resolved", "resolved_thread", attention.MailKindRequest, "2026-08-17T10:03:30Z")
	deliverThreadMessage(t, store, "peer_b", "mail_other", "shared_thread", attention.MailKindRequest, "2026-08-17T10:04:00Z")

	cfgA, _ := config.Load(projectA)
	mailA := mail.NewService(store, projectA, cfgA)
	history, _, err := mailA.Thread("peer_a", "history_thread")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mailA.ThreadAction(mailthread.ActionRequest{SchemaVersion: mailthread.SchemaVersion, Identity: history.State.Identity, ActionID: mailthread.ActionArchive, ThreadRevision: history.State.Revision, IdempotencyKey: "archive-history"}); err != nil {
		t.Fatal(err)
	}
	resolved, _, _ := mailA.Thread("peer_a", "resolved_thread")
	if _, err := mailA.ThreadAction(mailthread.ActionRequest{SchemaVersion: mailthread.SchemaVersion, Identity: resolved.State.Identity, ActionID: mailthread.ActionResolve, ThreadRevision: resolved.State.Revision, IdempotencyKey: "resolve-update", Input: json.RawMessage(`{"reason":"done","source":"user","grace_class":"informational"}`)}); err != nil {
		t.Fatal(err)
	}

	service, _ := NewService(state, queryRegistry(map[string]string{"a": projectA, "b": projectB}))
	first := service.Threads(mailthread.ThreadListRequest{SchemaVersion: mailthread.SchemaVersion, Limit: 2})
	if first.Error != nil || first.Counts.Total != 5 || first.Counts.Actionable != 2 || first.Counts.ActionableUnread != 2 || len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("first thread page = %#v", first)
	}
	for _, item := range first.Items {
		if item.Bucket != mailthread.BucketHistory && item.Bucket != mailthread.BucketUpdates {
			t.Fatalf("lifecycle activity was not sorted first: %#v", first.Items)
		}
	}
	needs := service.Threads(mailthread.ThreadListRequest{SchemaVersion: mailthread.SchemaVersion, Bucket: mailthread.BucketNeedsAttention})
	if needs.Error != nil || len(needs.Items) != 2 || needs.Items[0].Identity.ProjectPeerID == needs.Items[1].Identity.ProjectPeerID {
		t.Fatalf("composite needs-attention identities = %#v", needs)
	}
	for _, item := range needs.Items {
		if item.Identity.ProjectPeerID == "peer_a" && (item.Kind != attention.MailKindRequest || item.MessageCount != 3 || item.Subject == "Subject mail_request_status") {
			t.Fatalf("status traffic hid actionable request: %#v", item)
		}
	}
	requestDetail := service.ThreadDetail("peer_a", "shared_thread")
	if requestDetail.Error != nil || len(requestDetail.Messages) != 3 || requestDetail.Messages[2].Envelope.ID != "mail_request_status" {
		t.Fatalf("collapsed status detail = %#v", requestDetail)
	}
	updates := service.Threads(mailthread.ThreadListRequest{SchemaVersion: mailthread.SchemaVersion, ProjectPeerID: "peer_a", Bucket: mailthread.BucketUpdates})
	if updates.Error != nil || len(updates.Items) != 2 || updates.Items[0].Actionable || updates.Items[1].Actionable {
		t.Fatalf("updates = %#v", updates)
	}
	historyPage := service.Threads(mailthread.ThreadListRequest{SchemaVersion: mailthread.SchemaVersion, Bucket: mailthread.BucketHistory})
	if historyPage.Error != nil || len(historyPage.Items) != 1 || historyPage.Items[0].Identity.ThreadID != "history_thread" {
		t.Fatalf("history = %#v", historyPage)
	}
	detail := service.ThreadDetail("peer_a", "history_thread")
	if detail.Error != nil || detail.Summary == nil || detail.Thread == nil || len(detail.Messages) != 1 || detail.Messages[0].Envelope.ID != "mail_archive" {
		t.Fatalf("history detail = %#v", detail)
	}

	for _, id := range []string{"mail_request_1", "mail_request_2", "mail_request_status"} {
		item, _ := mailA.Show(id, false)
		revision := int64(0)
		if item.Receipt != nil {
			revision = item.Receipt.Revision
		}
		if _, err := mailA.Action(mail.ActionRequest{MessageID: id, Action: mail.ActionRead, ExpectedRevision: revision, IdempotencyKey: "read-" + id}); err != nil {
			t.Fatal(err)
		}
	}
	readOpen := service.Threads(mailthread.ThreadListRequest{SchemaVersion: mailthread.SchemaVersion, ProjectPeerID: "peer_a", Bucket: mailthread.BucketNeedsAttention})
	if readOpen.Error != nil || len(readOpen.Items) != 1 || readOpen.Items[0].Unread || !readOpen.Items[0].Actionable || readOpen.Counts.ActionableUnread != 0 {
		t.Fatalf("read open thread = %#v", readOpen)
	}
	stalePage := service.Threads(mailthread.ThreadListRequest{SchemaVersion: mailthread.SchemaVersion, Limit: 2, Cursor: first.NextCursor})
	if stalePage.Error == nil || stalePage.Error.Code != attention.ErrorStale {
		t.Fatalf("stale thread cursor = %#v", stalePage)
	}

	v1 := service.List(mailreadRequest())
	if v1.Error != nil || v1.TotalCount != 8 || len(v1.Items) != 8 {
		t.Fatalf("v1 message compatibility = %#v", v1)
	}
}

func deliverThreadMessage(t *testing.T, store *mail.Store, recipient, id, threadID, kind, createdAt string) {
	t.Helper()
	envelope := attention.MailEnvelope{
		SchemaVersion: attention.SchemaVersion, ID: id,
		Recipient: attention.ProjectReference{PeerID: recipient, DisplayName: recipient},
		Sender:    attention.ProjectReference{PeerID: "sender", DisplayName: "sender"},
		Subject:   "Subject " + id, Body: "body", Kind: kind, ThreadID: threadID,
		IdempotencyKey: recipient + "-" + id, CreatedAt: createdAt,
	}
	delivery := attention.MailDelivery{SchemaVersion: attention.SchemaVersion, MessageID: id, ThreadID: threadID, Sender: envelope.Sender, Recipient: envelope.Recipient, IdempotencyKey: envelope.IdempotencyKey, DeliveredAt: createdAt}
	if _, _, err := store.Deliver(envelope, delivery); err != nil {
		t.Fatal(err)
	}
}

func mailreadRequest() mailread.ListRequest {
	return mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, Limit: 100}
}
