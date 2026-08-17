package mailread

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
)

func TestValidateListRequestCompositeScopeLimitsAndCursor(t *testing.T) {
	unread := false
	valid := ListRequest{SchemaVersion: 1, ProjectPeerID: "peer_alpha", ThreadID: "thread_1", UnreadOnly: &unread, Limit: 100, Cursor: "mailread.v1.cursor"}
	if err := ValidateListRequest(valid); err != nil {
		t.Fatalf("valid request: %#v", err)
	}

	tests := []struct {
		name  string
		value ListRequest
		code  string
		field string
	}{
		{"version", ListRequest{SchemaVersion: 2}, attention.ErrorIncompatibleVersion, "schema_version"},
		{"project whitespace", ListRequest{SchemaVersion: 1, ProjectPeerID: " "}, attention.ErrorValidation, "project_peer_id"},
		{"global thread", ListRequest{SchemaVersion: 1, ThreadID: "thread_1"}, attention.ErrorValidation, "project_peer_id"},
		{"negative limit", ListRequest{SchemaVersion: 1, Limit: -1}, attention.ErrorValidation, "limit"},
		{"large limit", ListRequest{SchemaVersion: 1, Limit: 101}, attention.ErrorValidation, "limit"},
		{"large cursor", ListRequest{SchemaVersion: 1, Cursor: strings.Repeat("c", MaxCursorBytes+1)}, attention.ErrorValidation, "cursor"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateListRequest(test.value)
			if err == nil || err.Code != test.code || err.Field != test.field {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func TestValidateSummaryDetailAndReceiptState(t *testing.T) {
	summary := validSummary(3)
	if err := ValidateMessageSummary(summary); err != nil {
		t.Fatalf("valid summary: %#v", err)
	}

	unread := summary.Unread
	receipt := summary.Receipt
	detail := DetailResponse{
		SchemaVersion: 1,
		Project:       &summary.Project,
		Envelope: &attention.MailEnvelope{
			SchemaVersion: 1,
			ID:            summary.MessageID,
			Recipient:     summary.Recipient,
			Sender:        summary.Sender,
			Subject:       summary.Subject,
			Body:          "full body",
			Kind:          summary.Kind,
			ThreadID:      summary.ThreadID,
			CreatedAt:     summary.CreatedAt,
		},
		ActivityAt: summary.ActivityAt,
		Unread:     &unread,
		Receipt:    &receipt,
		Actions:    summary.Actions,
	}
	if err := ValidateDetailResponse(detail); err != nil {
		t.Fatalf("valid detail: %#v", err)
	}

	t.Run("activity is immutable delivery instant", func(t *testing.T) {
		invalid := summary
		invalid.ActivityAt = "2026-08-16T12:00:01Z"
		if err := ValidateMessageSummary(invalid); err == nil || err.Field != "activity_at" {
			t.Fatalf("error = %#v", err)
		}
	})

	t.Run("malformed timestamp fails closed", func(t *testing.T) {
		invalid := summary
		invalid.CreatedAt = "2026-08-16 12:00:00"
		invalid.ActivityAt = invalid.CreatedAt
		if err := ValidateMessageSummary(invalid); err == nil || err.Field != "created_at" {
			t.Fatalf("error = %#v", err)
		}
	})

	t.Run("recipient and project peer must match", func(t *testing.T) {
		invalid := summary
		invalid.Recipient.PeerID = "peer_other"
		if err := ValidateMessageSummary(invalid); err == nil || err.Field != "project.peer_id" {
			t.Fatalf("error = %#v", err)
		}
	})

	t.Run("receipt unread is authoritative", func(t *testing.T) {
		invalid := summary
		invalid.Unread = false
		if err := ValidateMessageSummary(invalid); err == nil || err.Field != "unread" {
			t.Fatalf("error = %#v", err)
		}
	})

	t.Run("descriptor revision follows receipt", func(t *testing.T) {
		invalid := summary
		invalid.Actions = append([]attention.ActionDescriptor(nil), summary.Actions...)
		invalid.Actions[0].RequiredRowRevision++
		if err := ValidateMessageSummary(invalid); err == nil || err.Field != "actions[0].required_row_revision" {
			t.Fatalf("error = %#v", err)
		}
	})
}

func TestValidateListResponsePageInvariants(t *testing.T) {
	response := ListResponse{
		SchemaVersion: 1,
		Revision:      "revision_1",
		TotalCount:    2,
		UnreadCount:   2,
		Items:         []MessageSummary{validSummary(0)},
		Page:          &PageMetadata{Limit: 1, Returned: 1, HasMore: true},
		NextCursor:    "mailread.v1.next",
	}
	if err := ValidateListResponse(response); err != nil {
		t.Fatalf("valid response: %#v", err)
	}

	invalid := response
	invalid.NextCursor = ""
	if err := ValidateListResponse(invalid); err == nil || err.Field != "next_cursor" {
		t.Fatalf("cursor invariant error = %#v", err)
	}

	errorResponse := ListResponse{SchemaVersion: 1, Error: &attention.ContractError{Code: attention.ErrorUnavailable, Message: "registry unavailable"}}
	if err := ValidateListResponse(errorResponse); err != nil {
		t.Fatalf("structured failure: %#v", err)
	}
}

func TestValidateActionAndReplyDTOs(t *testing.T) {
	for _, actionID := range []string{ActionMarkRead, ActionAcknowledge, ActionDismiss, ActionPromote, ActionAddToToday} {
		request := ActionRequest{SchemaVersion: 1, ProjectPeerID: "peer_alpha", MessageID: "message_1", ActionID: actionID, ReceiptRevision: 0, IdempotencyKey: "key_" + actionID, Input: json.RawMessage(`{}`)}
		if err := ValidateActionRequest(request); err != nil {
			t.Fatalf("%s: %#v", actionID, err)
		}
	}

	unsupported := ActionRequest{SchemaVersion: 1, ProjectPeerID: "peer_alpha", MessageID: "message_1", ActionID: "read", IdempotencyKey: "key"}
	if err := ValidateActionRequest(unsupported); err == nil || err.Code != attention.ErrorUnsupported {
		t.Fatalf("legacy action error = %#v", err)
	}
	unknown := unsupported
	unknown.ActionID = "future.quantum"
	if err := ValidateActionRequest(unknown); err == nil || err.Code != attention.ErrorUnsupported {
		t.Fatalf("unknown action error = %#v", err)
	}

	reply := ReplyRequest{SchemaVersion: 1, ProjectPeerID: "peer_alpha", MessageID: "message_1", ThreadID: "thread_1", Body: strings.Repeat("x", attention.MaxBodyBytes), IdempotencyKey: "reply_key"}
	if err := ValidateReplyRequest(reply); err != nil {
		t.Fatalf("max body reply: %#v", err)
	}
	reply.Body += "x"
	if err := ValidateReplyRequest(reply); err == nil || err.Field != "body" {
		t.Fatalf("oversize body error = %#v", err)
	}

	delivery := &attention.MailDelivery{
		SchemaVersion:  1,
		MessageID:      "message_reply",
		ThreadID:       "thread_1",
		Sender:         project("peer_alpha", "alpha", "Alpha"),
		Recipient:      project("peer_beta", "beta", "Beta"),
		IdempotencyKey: "reply_key",
		DeliveredAt:    "2026-08-16T12:01:00Z",
	}
	if err := ValidateReplyResponse(ReplyResponse{SchemaVersion: 1, Delivery: delivery}); err != nil {
		t.Fatalf("reply response: %#v", err)
	}
}

func TestUnknownAdditiveFieldsAndRawIdentifiersRemainInert(t *testing.T) {
	raw := `{
		"schema_version":1,
		"revision":"revision_future",
		"total_count":1,
		"unread_count":1,
		"items":[{
			"project":{"peer_id":"peer_alpha","registry_slug":"alpha","display_name":"Alpha"},
			"message_id":"message_1","sender":{"peer_id":"peer_sender","display_name":"Sender"},
			"recipient":{"peer_id":"peer_alpha","display_name":"Alpha"},"subject":"Future",
			"kind":"future_kind","created_at":"2026-08-16T12:00:00Z","activity_at":"2026-08-16T12:00:00Z",
			"unread":true,"receipt":{"revision":0,"unread":true},
			"actions":[{"id":"future.action","label":"Future","style":"sparkle","operation_id":"future.operation","effect":"quantum_write","consent":"collective","required_row_revision":0,"requires_idempotency":true}],
			"future_item_field":{"prompt":"do not execute"}
		}],
		"page":{"limit":20,"returned":1,"has_more":false},
		"future_response_field":"preserve compatibility"
	}`
	var response ListResponse
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		t.Fatal(err)
	}
	if err := ValidateListResponse(response); err != nil {
		t.Fatalf("unknown additive response: %#v", err)
	}
	action := response.Items[0].Actions[0]
	if action.ID != "future.action" || action.OperationID != "future.operation" || action.Effect != "quantum_write" || action.Consent != "collective" {
		t.Fatalf("raw identifiers changed: %#v", action)
	}
	if isCanonicalMutationAction(action.ID) {
		t.Fatal("unknown descriptor became executable")
	}
}

func validSummary(revision int64) MessageSummary {
	return MessageSummary{
		Project:    project("peer_alpha", "alpha", "Alpha"),
		MessageID:  "message_1",
		ThreadID:   "thread_1",
		Sender:     project("peer_sender", "", "Sender"),
		Recipient:  project("peer_alpha", "", "Alpha"),
		Subject:    "Contract fixture",
		Kind:       attention.MailKindRequest,
		CreatedAt:  "2026-08-16T12:00:00Z",
		ActivityAt: "2026-08-16T12:00:00Z",
		Unread:     true,
		Receipt:    ReceiptView{Revision: revision, Unread: true},
		Actions: []attention.ActionDescriptor{{
			ID: ActionMarkRead, Label: "Mark read", OperationID: attention.OperationMailMarkRead,
			Effect: string(attention.EffectStateWrite), Consent: string(attention.ConsentExplicitUser),
			InputSchema: json.RawMessage(`{"type":"object"}`), RequiredRowRevision: revision, RequiresIdempotency: true,
		}},
	}
}

func project(peerID, slug, name string) attention.ProjectReference {
	return attention.ProjectReference{PeerID: peerID, RegistrySlug: slug, DisplayName: name}
}
