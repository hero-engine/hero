package mailthread

import (
	"encoding/json"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
)

func openView() ThreadView {
	state := State{SchemaVersion: 1, Identity: Identity{ProjectPeerID: "peer_a", ThreadID: "mail_one"}, Lifecycle: LifecycleOpen, Revision: 7}
	actions := []attention.ActionDescriptor{{ID: ActionResolve, Label: "Resolve", OperationID: "mail.resolve", Effect: "state_write", Consent: "explicit_user", InputSchema: json.RawMessage(`{"type":"object"}`), RequiredRowRevision: 7, RequiresIdempotency: true}}
	return ThreadView{State: state, Read: ReadSummary{MessageCount: 2, UnreadCount: 1}, Actions: actions}
}

func TestStateKeepsReadAndLifecycleOrthogonal(t *testing.T) {
	view := openView()
	if err := ValidateThreadView(view); err != nil {
		t.Fatal(err)
	}
	view.Read.UnreadCount = 0
	if err := ValidateThreadView(view); err != nil || view.State.Lifecycle != LifecycleOpen || view.State.ArchiveEligibleAt != "" {
		t.Fatalf("read change altered open lifecycle: %#v, %v", view, err)
	}
}

func TestActionValidationIsClosedWorldAndStrict(t *testing.T) {
	base := ActionRequest{SchemaVersion: 1, Identity: Identity{ProjectPeerID: "peer_a", ThreadID: "mail_one"}, ActionID: ActionResolve, ThreadRevision: 7, IdempotencyKey: "resolve-1", Input: json.RawMessage(`{"reason":"done","source":"user"}`)}
	if err := ValidateActionRequest(base); err != nil {
		t.Fatal(err)
	}
	unknown := base
	unknown.ActionID = "future_action"
	if err := ValidateActionRequest(unknown); err == nil || err.Code != attention.ErrorUnsupported {
		t.Fatalf("unknown action = %#v", err)
	}
	malformed := base
	malformed.Input = json.RawMessage(`{"reason":"done","source":"user","execute":true}`)
	if err := ValidateActionRequest(malformed); err == nil || err.Field != "input" {
		t.Fatalf("unknown executable input = %#v", err)
	}
	for _, actionID := range []string{ActionMarkRead, ActionMarkUnread} {
		receipt := ActionRequest{SchemaVersion: 1, Identity: base.Identity, ActionID: actionID, ThreadRevision: 7, IdempotencyKey: actionID + "-1"}
		if err := ValidateActionRequest(receipt); err != nil {
			t.Fatalf("%s action = %#v", actionID, err)
		}
		receipt.Input = json.RawMessage(`{"reason":"not-allowed"}`)
		if err := ValidateActionRequest(receipt); err == nil || err.Field != "input" {
			t.Fatalf("%s accepted lifecycle input: %#v", actionID, err)
		}
	}
}

func TestOpenStateRejectsArchiveEligibility(t *testing.T) {
	state := openView().State
	state.ArchiveEligibleAt = "2026-08-30T00:00:00Z"
	if err := ValidateState(state); err == nil {
		t.Fatal("open thread accepted archive eligibility")
	}
}

func TestThreadSummaryClassificationKeepsReadAndActionableIndependent(t *testing.T) {
	base := ThreadSummary{
		Identity: Identity{ProjectPeerID: "peer_a", ThreadID: "mail_one"},
		Project:  attention.ProjectReference{PeerID: "peer_a", DisplayName: "Alpha"},
		Sender:   attention.ProjectReference{PeerID: "peer_b", DisplayName: "Beta"},
		Subject:  "Review", ActivityAt: "2026-08-22T10:00:00Z",
		Actionable: true, Lifecycle: LifecycleOpen, Bucket: BucketNeedsAttention,
		MessageCount: 2, Revision: 7,
	}
	if err := ValidateThreadSummary(base); err != nil {
		t.Fatal(err)
	}
	base.Unread, base.UnreadCount = true, 2
	if err := ValidateThreadSummary(base); err != nil {
		t.Fatal(err)
	}
	invalid := base
	invalid.Actionable = false
	if err := ValidateThreadSummary(invalid); err == nil || err.Field != "bucket" {
		t.Fatalf("needs-attention accepted non-actionable summary: %#v", err)
	}
	invalid = base
	invalid.Bucket = BucketUpdates
	if err := ValidateThreadSummary(invalid); err == nil || err.Field != "bucket" {
		t.Fatalf("updates accepted actionable summary: %#v", err)
	}
}
