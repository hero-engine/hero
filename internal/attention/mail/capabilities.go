package mail

import (
	"encoding/json"

	"github.com/hero-engine/hero/contracts/attention"
)

var (
	capabilityNoInput   = json.RawMessage(`{"type":"object","additionalProperties":false}`)
	capabilityNoteInput = json.RawMessage(
		`{"type":"object","properties":{"note":{"type":"string"}},"additionalProperties":false}`,
	)
	capabilityPromoteInput = json.RawMessage(
		`{"type":"object","properties":{"artifact_type":{"type":"string"}},"required":["artifact_type"],"additionalProperties":false}`,
	)
	capabilityReplyInput = json.RawMessage(
		`{"type":"object","properties":{"body":{"type":"string"},"subject":{"type":"string"},"kind":{"type":"string"}},"required":["body"],"additionalProperties":false}`,
	)
)

// Capabilities returns the canonical Mail capabilities for a receipt revision.
// Reply is advertised here even though it is dispatched through the dedicated
// typed reply request rather than the generic action request.
func Capabilities(revision int64) []attention.ActionDescriptor {
	return []attention.ActionDescriptor{
		capability(attention.OperationMailMarkRead, "mark_read", "Mark Read", revision, capabilityNoInput),
		capability(attention.OperationMailAcknowledge, ActionAcknowledge, "Acknowledge", revision, capabilityNoteInput),
		capability(attention.OperationMailDismiss, ActionDismiss, "Dismiss", revision, capabilityNoInput),
		capability(attention.OperationMailPromote, ActionPromote, "Promote", revision, capabilityPromoteInput),
		capability(attention.OperationMailAddToToday, ActionAddToToday, "Add to Today", revision, capabilityNoInput),
		capability(attention.OperationMailReply, "reply", "Reply", revision, capabilityReplyInput),
	}
}

// RowCapabilities preserves the existing Attention row semantics, where reply
// is not a generic row action.
func RowCapabilities(revision int64) []attention.ActionDescriptor {
	capabilities := Capabilities(revision)
	return capabilities[:len(capabilities)-1]
}

// SourceActionID translates a canonical consumer action into the owning Mail
// service's legacy source operation. Reply has its own Service.Reply path.
func SourceActionID(canonicalID string) (string, bool) {
	switch canonicalID {
	case "mark_read":
		return ActionRead, true
	case "mark_unread":
		return ActionUnread, true
	case ActionAcknowledge, ActionDismiss, ActionPromote, ActionAddToToday:
		return canonicalID, true
	default:
		return "", false
	}
}

// LifecycleReceiptCapabilities is the additive receipt surface published by
// the Mail thread contract. Mail-read v1 remains unchanged for pinned clients.
func LifecycleReceiptCapabilities(revision int64) []attention.ActionDescriptor {
	return []attention.ActionDescriptor{
		capability(attention.OperationMailMarkRead, "mark_read", "Mark Read", revision, capabilityNoInput),
		{ID: "mark_unread", Label: "Mark Unread", OperationID: "mail.mark_unread", Effect: string(attention.EffectStateWrite), Consent: string(attention.ConsentExplicitUser), RequiredRowRevision: revision, RequiresIdempotency: true, InputSchema: capabilityNoInput},
	}
}

func capability(operationID, id, label string, revision int64, schema json.RawMessage) attention.ActionDescriptor {
	descriptor, ok := attention.AnnotateActionDescriptor(attention.ActionDescriptor{
		ID:                  id,
		Label:               label,
		RequiredRowRevision: revision,
		RequiresIdempotency: true,
		InputSchema:         schema,
	}, operationID)
	if !ok {
		panic("unknown Attention operation policy: " + operationID)
	}
	return descriptor
}
