package mailread

import (
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hero-engine/hero/contracts/attention"
)

func ValidateListRequest(v ListRequest) *attention.ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := validateOptionalID("project_peer_id", v.ProjectPeerID); err != nil {
		return err
	}
	if err := validateOptionalID("thread_id", v.ThreadID); err != nil {
		return err
	}
	if v.ThreadID != "" && v.ProjectPeerID == "" {
		return invalid("project_peer_id", "is required when thread_id is set")
	}
	if v.Limit < 0 || v.Limit > MaxListLimit {
		return invalid("limit", "must be omitted or between 1 and 100")
	}
	if !utf8.ValidString(v.Cursor) || len(v.Cursor) > MaxCursorBytes {
		return invalid("cursor", "must be valid UTF-8 and at most 4096 bytes")
	}
	return nil
}

func ValidateListResponse(v ListResponse) *attention.ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if v.Error != nil {
		return validateError(v.Error)
	}
	if err := validateRequiredID("revision", v.Revision); err != nil {
		return err
	}
	if v.TotalCount < 0 || v.UnreadCount < 0 || v.UnreadCount > v.TotalCount {
		return invalid("total_count", "counts must be non-negative and unread_count must not exceed total_count")
	}
	if v.Page == nil {
		return invalid("page", "is required on success")
	}
	if v.Page.Limit < 1 || v.Page.Limit > MaxListLimit {
		return invalid("page.limit", "must be between 1 and 100")
	}
	if v.Page.Returned != len(v.Items) || v.Page.Returned > v.Page.Limit || v.TotalCount < v.Page.Returned {
		return invalid("page.returned", "must match items and remain within the page and scoped total")
	}
	if v.Page.HasMore != (v.NextCursor != "") {
		return invalid("next_cursor", "must be present exactly when page.has_more is true")
	}
	if len(v.NextCursor) > MaxCursorBytes || !utf8.ValidString(v.NextCursor) {
		return invalid("next_cursor", "must be valid UTF-8 and at most 4096 bytes")
	}
	for i := range v.Items {
		if err := ValidateMessageSummary(v.Items[i]); err != nil {
			err.Field = indexed("items", i, err.Field)
			return err
		}
	}
	return nil
}

func ValidateMessageSummary(v MessageSummary) *attention.ContractError {
	if err := validateProject("project", v.Project); err != nil {
		return err
	}
	if err := validateRequiredID("message_id", v.MessageID); err != nil {
		return err
	}
	if err := validateOptionalID("thread_id", v.ThreadID); err != nil {
		return err
	}
	if err := validateOptionalID("in_reply_to", v.InReplyTo); err != nil {
		return err
	}
	if err := validateProject("sender", v.Sender); err != nil {
		return err
	}
	if err := validateProject("recipient", v.Recipient); err != nil {
		return err
	}
	if v.Project.PeerID != v.Recipient.PeerID {
		return invalid("project.peer_id", "must equal recipient.peer_id")
	}
	if !utf8.ValidString(v.Subject) || utf8.RuneCountInString(v.Subject) > attention.MaxSubjectCharacters {
		return invalid("subject", "must be valid UTF-8 and at most 200 characters")
	}
	if !utf8.ValidString(v.Kind) || len(v.Kind) > 64 {
		return invalid("kind", "must be valid UTF-8 and at most 64 bytes")
	}
	if err := validateTimestamp("created_at", v.CreatedAt, true); err != nil {
		return err
	}
	if err := validateTimestamp("activity_at", v.ActivityAt, true); err != nil {
		return err
	}
	if v.ActivityAt != v.CreatedAt {
		return invalid("activity_at", "must equal immutable envelope created_at")
	}
	if err := validateReceipt(v.Receipt); err != nil {
		return err
	}
	if v.Unread != v.Receipt.Unread {
		return invalid("unread", "must equal receipt.unread")
	}
	return validateActions(v.Actions, v.Receipt.Revision)
}

func ValidateDetailResponse(v DetailResponse) *attention.ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if v.Error != nil {
		return validateError(v.Error)
	}
	if v.Envelope == nil {
		return invalid("envelope", "is required on success")
	}
	if v.Project == nil {
		return invalid("project", "is required on success")
	}
	if err := validateProject("project", *v.Project); err != nil {
		return err
	}
	if err := attention.ValidateMailEnvelope(*v.Envelope); err != nil {
		err.Field = "envelope." + err.Field
		return err
	}
	if v.Project.PeerID != v.Envelope.Recipient.PeerID {
		return invalid("project.peer_id", "must equal envelope.recipient.peer_id")
	}
	if err := validateTimestamp("activity_at", v.ActivityAt, true); err != nil {
		return err
	}
	if v.ActivityAt != v.Envelope.CreatedAt {
		return invalid("activity_at", "must equal immutable envelope created_at")
	}
	if v.Receipt == nil {
		return invalid("receipt", "is required on success")
	}
	if err := validateReceipt(*v.Receipt); err != nil {
		return err
	}
	if v.Unread == nil {
		return invalid("unread", "is required on success")
	}
	if *v.Unread != v.Receipt.Unread {
		return invalid("unread", "must equal receipt.unread")
	}
	return validateActions(v.Actions, v.Receipt.Revision)
}

func ValidateActionRequest(v ActionRequest) *attention.ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := validateRequiredID("project_peer_id", v.ProjectPeerID); err != nil {
		return err
	}
	if err := validateRequiredID("message_id", v.MessageID); err != nil {
		return err
	}
	if !isCanonicalMutationAction(v.ActionID) {
		return &attention.ContractError{Code: attention.ErrorUnsupported, Message: "action is not supported by the Mail-read v1 contract", Field: "action_id"}
	}
	if v.ReceiptRevision < 0 {
		return invalid("receipt_revision", "must not be negative")
	}
	if err := validateRequiredID("idempotency_key", v.IdempotencyKey); err != nil {
		return err
	}
	if len(v.Input) != 0 {
		var input map[string]any
		if err := json.Unmarshal(v.Input, &input); err != nil || input == nil {
			return invalid("input", "must be a JSON object")
		}
	}
	return nil
}

func ValidateActionResponse(v ActionResponse) *attention.ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if v.Error != nil {
		return validateError(v.Error)
	}
	if v.Project == nil {
		return invalid("project", "is required on success")
	}
	if err := validateProject("project", *v.Project); err != nil {
		return err
	}
	if err := validateRequiredID("message_id", v.MessageID); err != nil {
		return err
	}
	if v.Receipt == nil {
		return invalid("receipt", "is required on success")
	}
	if err := validateReceipt(*v.Receipt); err != nil {
		return err
	}
	return validateActions(v.Actions, v.Receipt.Revision)
}

func ValidateReplyRequest(v ReplyRequest) *attention.ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := validateRequiredID("project_peer_id", v.ProjectPeerID); err != nil {
		return err
	}
	if err := validateRequiredID("message_id", v.MessageID); err != nil {
		return err
	}
	if err := validateRequiredID("thread_id", v.ThreadID); err != nil {
		return err
	}
	if !utf8.ValidString(v.Body) || len(v.Body) > attention.MaxBodyBytes {
		return invalid("body", "must be valid UTF-8 and at most 65536 bytes")
	}
	if !utf8.ValidString(v.Subject) || utf8.RuneCountInString(v.Subject) > attention.MaxSubjectCharacters {
		return invalid("subject", "must be valid UTF-8 and at most 200 characters")
	}
	if !utf8.ValidString(v.Kind) || len(v.Kind) > 64 {
		return invalid("kind", "must be valid UTF-8 and at most 64 bytes")
	}
	return validateRequiredID("idempotency_key", v.IdempotencyKey)
}

func ValidateReplyResponse(v ReplyResponse) *attention.ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if v.Error != nil {
		return validateError(v.Error)
	}
	if v.Delivery == nil {
		return invalid("delivery", "is required on success")
	}
	if v.Delivery.SchemaVersion != SchemaVersion {
		return incompatible("delivery.schema_version")
	}
	if err := validateRequiredID("delivery.message_id", v.Delivery.MessageID); err != nil {
		return err
	}
	if err := validateRequiredID("delivery.thread_id", v.Delivery.ThreadID); err != nil {
		return err
	}
	if err := validateProject("delivery.sender", v.Delivery.Sender); err != nil {
		return err
	}
	if err := validateProject("delivery.recipient", v.Delivery.Recipient); err != nil {
		return err
	}
	if err := validateRequiredID("delivery.idempotency_key", v.Delivery.IdempotencyKey); err != nil {
		return err
	}
	return validateTimestamp("delivery.delivered_at", v.Delivery.DeliveredAt, true)
}

func ValidateContractResponse(v ContractResponse) *attention.ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if v.BundleVersion != BundleVersion {
		return incompatible("bundle_version")
	}
	decoded, err := hex.DecodeString(v.BundleManifestSHA256)
	if err != nil || len(decoded) != 32 || strings.ToLower(v.BundleManifestSHA256) != v.BundleManifestSHA256 {
		return invalid("bundle_manifest_sha256", "must be a lowercase SHA-256 hex digest")
	}
	if v.Compatibility != Compatibility {
		return incompatible("compatibility")
	}
	return nil
}

func validateReceipt(v ReceiptView) *attention.ContractError {
	if v.Revision < 0 {
		return invalid("receipt.revision", "must not be negative")
	}
	for field, value := range map[string]string{
		"receipt.read_at":         v.ReadAt,
		"receipt.acknowledged_at": v.AcknowledgedAt,
		"receipt.dismissed_at":    v.DismissedAt,
	} {
		if err := validateTimestamp(field, value, false); err != nil {
			return err
		}
	}
	if v.PromotedArtifact != nil {
		if err := validateRequiredID("receipt.promoted_artifact.slug", v.PromotedArtifact.Slug); err != nil {
			return err
		}
		if err := validateRequiredID("receipt.promoted_artifact.type", v.PromotedArtifact.Type); err != nil {
			return err
		}
	}
	return validateOptionalID("receipt.focus_item_id", v.FocusItemID)
}

func validateActions(values []attention.ActionDescriptor, receiptRevision int64) *attention.ContractError {
	seen := make(map[string]bool, len(values))
	for i, descriptor := range values {
		field := indexed("actions", i, "")
		if strings.TrimSpace(descriptor.ID) == "" || strings.TrimSpace(descriptor.Label) == "" {
			return invalid(field, "id and label are required")
		}
		if seen[descriptor.ID] {
			return invalid(field+".id", "must be unique")
		}
		seen[descriptor.ID] = true
		if descriptor.RequiredRowRevision != receiptRevision {
			return invalid(field+".required_row_revision", "must equal receipt revision")
		}
		if len(descriptor.InputSchema) != 0 && !json.Valid(descriptor.InputSchema) {
			return invalid(field+".input_schema", "must be valid JSON")
		}
	}
	return nil
}

func isCanonicalMutationAction(actionID string) bool {
	return actionID == ActionMarkRead || actionID == ActionAcknowledge || actionID == ActionDismiss || actionID == ActionPromote || actionID == ActionAddToToday
}

func validateVersion(version int) *attention.ContractError {
	if version != SchemaVersion {
		return incompatible("schema_version")
	}
	return nil
}

func validateError(v *attention.ContractError) *attention.ContractError {
	if v == nil || strings.TrimSpace(v.Message) == "" {
		return invalid("error.message", "is required")
	}
	switch v.Code {
	case attention.ErrorValidation, attention.ErrorStale, attention.ErrorUnsupported, attention.ErrorMissing,
		attention.ErrorIncompatibleVersion, attention.ErrorUnavailable, attention.ErrorIdempotencyConflict, attention.ErrorPermission:
		return nil
	default:
		return invalid("error.code", "is not a stable v1 error code")
	}
}

func validateProject(field string, project attention.ProjectReference) *attention.ContractError {
	if err := validateRequiredID(field+".peer_id", project.PeerID); err != nil {
		return err
	}
	if strings.TrimSpace(project.DisplayName) == "" {
		return invalid(field+".display_name", "is required")
	}
	return validateOptionalID(field+".registry_slug", project.RegistrySlug)
}

func validateRequiredID(field, value string) *attention.ContractError {
	if strings.TrimSpace(value) == "" || !utf8.ValidString(value) {
		return invalid(field, "is required and must be valid UTF-8")
	}
	return nil
}

func validateOptionalID(field, value string) *attention.ContractError {
	if value != "" && (strings.TrimSpace(value) == "" || !utf8.ValidString(value)) {
		return invalid(field, "must be valid non-blank UTF-8 when present")
	}
	return nil
}

func validateTimestamp(field, value string, required bool) *attention.ContractError {
	if value == "" && !required {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if value == "" || err != nil || !strings.HasSuffix(value, "Z") || parsed.Location() != time.UTC {
		return invalid(field, "must be a UTC RFC3339Nano timestamp")
	}
	return nil
}

func invalid(field, message string) *attention.ContractError {
	return &attention.ContractError{Code: attention.ErrorValidation, Message: message, Field: field}
}

func incompatible(field string) *attention.ContractError {
	return &attention.ContractError{Code: attention.ErrorIncompatibleVersion, Message: "unsupported Mail-read contract version", Field: field}
}

func indexed(prefix string, index int, field string) string {
	base := prefix + "[" + strconv.Itoa(index) + "]"
	if field == "" {
		return base
	}
	return base + "." + field
}
