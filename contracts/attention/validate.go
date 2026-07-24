package attention

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxSubjectCharacters = 200
	MaxBodyBytes         = 64 << 10
	MaxEnvelopeBytes     = 128 << 10
	MaxProvenance        = 32
	MaxFocusPromptBytes  = 64 << 10
)

func ValidateMailEnvelope(v MailEnvelope) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := validateID("id", v.ID); err != nil {
		return err
	}
	if err := validateProject("recipient", v.Recipient); err != nil {
		return err
	}
	if err := validateProject("sender", v.Sender); err != nil {
		return err
	}
	if !utf8.ValidString(v.Subject) || utf8.RuneCountInString(v.Subject) > MaxSubjectCharacters {
		return invalid("subject", "must be valid UTF-8 and at most 200 characters")
	}
	if !utf8.ValidString(v.Body) || len(v.Body) > MaxBodyBytes {
		return invalid("body", "must be valid UTF-8 and at most 65536 bytes")
	}
	if v.Kind != "" && (!utf8.ValidString(v.Kind) || len(v.Kind) > 64) {
		return invalid("kind", "must be valid UTF-8 and at most 64 bytes")
	}
	if v.ThreadID != "" {
		if err := validateID("thread_id", v.ThreadID); err != nil {
			return err
		}
	}
	if v.InReplyTo != "" {
		if err := validateID("in_reply_to", v.InReplyTo); err != nil {
			return err
		}
	}
	if v.Revision < 0 {
		return invalid("revision", "must not be negative")
	}
	if !utf8.ValidString(v.IdempotencyKey) || len(v.IdempotencyKey) > 512 {
		return invalid("idempotency_key", "must be valid UTF-8 and at most 512 bytes")
	}
	if len(v.Provenance) > MaxProvenance {
		return invalid("provenance", "must contain at most 32 references")
	}
	if err := validateTimestamp("created_at", v.CreatedAt, true); err != nil {
		return err
	}
	if err := validateProvenance(v.Provenance); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return invalid("envelope", "cannot be encoded")
	}
	if len(b) > MaxEnvelopeBytes {
		return invalid("envelope", "encoded envelope exceeds 131072 bytes")
	}
	return nil
}

func ValidateMailReceipt(v MailReceipt) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := validateID("id", v.ID); err != nil {
		return err
	}
	if err := validateID("envelope_id", v.EnvelopeID); err != nil {
		return err
	}
	if err := validateProject("recipient", v.Recipient); err != nil {
		return err
	}
	if v.Kind != "" && v.Kind != ReceiptRead && v.Kind != ReceiptAcknowledged && v.Kind != ReceiptDismissed && v.Kind != ReceiptPromoted {
		return invalid("kind", "must be read, acknowledged, dismissed, or promoted")
	}
	if err := validateTimestamp("created_at", v.CreatedAt, true); err != nil {
		return err
	}
	if err := validateTimestamp("read_at", v.ReadAt, false); err != nil {
		return err
	}
	if err := validateTimestamp("acknowledged_at", v.AcknowledgedAt, false); err != nil {
		return err
	}
	if err := validateTimestamp("dismissed_at", v.DismissedAt, false); err != nil {
		return err
	}
	if v.Revision < 0 {
		return invalid("revision", "must not be negative")
	}
	if !utf8.ValidString(v.AcknowledgementNote) || utf8.RuneCountInString(v.AcknowledgementNote) > 500 {
		return invalid("acknowledgement_note", "must be valid UTF-8 and at most 500 characters")
	}
	return nil
}

func ValidateCreateFocus(v CreateFocusRequest) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := validateID("id", v.ID); err != nil {
		return err
	}
	if err := validateProject("project", v.Project); err != nil {
		return err
	}
	if !utf8.ValidString(v.Prompt) || len(v.Prompt) > MaxFocusPromptBytes {
		return invalid("prompt", "must be valid UTF-8 and at most 65536 bytes")
	}
	if !validLifecycle(v.Lifecycle) {
		return invalid("lifecycle", "must be inbox, today, later, or done")
	}
	return validateTimestamp("created_at", v.CreatedAt, true)
}

func ValidateFocusItem(v FocusItem) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := validateID("id", v.ID); err != nil {
		return err
	}
	if err := validateProject("project", v.Project); err != nil {
		return err
	}
	if !utf8.ValidString(v.Prompt) || len(v.Prompt) > MaxFocusPromptBytes {
		return invalid("prompt", "must be valid UTF-8 and at most 65536 bytes")
	}
	if !validLifecycle(v.Lifecycle) {
		return invalid("lifecycle", "must be inbox, today, later, or done")
	}
	if err := validateTimestamp("created_at", v.CreatedAt, true); err != nil {
		return err
	}
	return validateTimestamp("updated_at", v.UpdatedAt, true)
}

func ValidateUpdateFocus(v UpdateFocusRequest) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := validateID("id", v.ID); err != nil {
		return err
	}
	if v.Revision < 1 {
		return invalid("revision", "must be a positive row revision")
	}
	if v.Title == "" && v.Prompt == "" && v.Lifecycle == "" {
		return invalid("update", "must change title, prompt, or lifecycle")
	}
	if !utf8.ValidString(v.Prompt) || len(v.Prompt) > MaxFocusPromptBytes {
		return invalid("prompt", "must be valid UTF-8 and at most 65536 bytes")
	}
	if v.Lifecycle != "" && !validLifecycle(v.Lifecycle) {
		return invalid("lifecycle", "must be inbox, today, later, or done")
	}
	return validateTimestamp("updated_at", v.UpdatedAt, true)
}

func ValidateDeferredWorkSuggestion(v DeferredWorkSuggestion) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := validateID("id", v.ID); err != nil {
		return err
	}
	if err := validateProject("project", v.Project); err != nil {
		return err
	}
	if !utf8.ValidString(v.Prompt) || len(v.Prompt) > MaxFocusPromptBytes {
		return invalid("prompt", "must be valid UTF-8 and at most 65536 bytes")
	}
	if v.State != "" && v.State != "pending" && v.State != "accepted" && v.State != "dismissed" && v.State != "expired" {
		return invalid("state", "must be pending, accepted, dismissed, or expired")
	}
	if v.Revision < 0 {
		return invalid("revision", "must not be negative")
	}
	if err := validateTimestamp("expires_at", v.ExpiresAt, false); err != nil {
		return err
	}
	if len(v.Provenance) > MaxProvenance {
		return invalid("provenance", "must contain at most 32 references")
	}
	if err := validateProvenance(v.Provenance); err != nil {
		return err
	}
	return validateTimestamp("created_at", v.CreatedAt, true)
}

func ValidateSuggestionDecision(v SuggestionDecisionRequest) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := validateID("suggestion_id", v.SuggestionID); err != nil {
		return err
	}
	if v.Decision != "accept" && v.Decision != "today" && v.Decision != "later" && v.Decision != "do_next" && v.Decision != "dismiss" {
		return invalid("decision", "must be accept, today, later, do_next, or dismiss")
	}
	if v.Revision < 0 {
		return invalid("revision", "must not be negative")
	}
	if err := validateID("idempotency_key", v.IdempotencyKey); err != nil {
		return err
	}
	return nil
}

func ValidateActionRequest(v ActionRequest, descriptor ActionDescriptor) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := validateID("row_id", v.RowID); err != nil {
		return err
	}
	if strings.TrimSpace(v.ActionID) == "" || v.ActionID != descriptor.ID {
		return invalid("action_id", "is not advertised for this row")
	}
	if v.RowRevision != descriptor.RequiredRowRevision {
		return &ContractError{Code: ErrorStale, Message: "row revision is stale", Field: "row_revision"}
	}
	if descriptor.RequiresIdempotency && strings.TrimSpace(v.IdempotencyKey) == "" {
		return invalid("idempotency_key", "is required for this action")
	}
	return nil
}

func ValidateOperationPolicies(values []OperationPolicy) *ContractError {
	seen := make(map[string]bool, len(values))
	for i, policy := range values {
		field := fmt.Sprintf("operations[%d]", i)
		if strings.TrimSpace(policy.ID) == "" {
			return invalid(field+".id", "is required")
		}
		if seen[policy.ID] {
			return invalid(field+".id", "must be unique")
		}
		seen[policy.ID] = true
		if strings.TrimSpace(policy.ToolName) == "" && strings.TrimSpace(policy.ActionID) == "" {
			return invalid(field, "must name a tool or advertised action")
		}
		if !validInteractionEffect(policy.Effect) {
			return invalid(field+".effect", "is not a stable v1 interaction effect")
		}
		if !validConsentRequirement(policy.Consent) {
			return invalid(field+".consent", "is not a stable v1 consent requirement")
		}
		switch policy.Effect {
		case EffectRead:
			if policy.Consent != ConsentNone {
				return invalid(field+".consent", "read operations must use consent none")
			}
		case EffectAdvisoryWrite:
			if policy.Consent != ConsentNone {
				return invalid(field+".consent", "advisory writes must use consent none")
			}
		case EffectStateWrite:
			if policy.Consent != ConsentExplicitUser {
				return invalid(field+".consent", "state writes require explicit user consent")
			}
		case EffectExternalWrite:
			if policy.Consent != ConsentExplicitUser {
				return invalid(field+".consent", "external writes require explicit user consent")
			}
		case EffectCommitment:
			if policy.Consent != ConsentExplicitUser && policy.Consent != ConsentExplicitAcceptance {
				return invalid(field+".consent", "commitments require explicit user consent or acceptance")
			}
		}
		if policy.Effect != EffectRead && !policy.ReplaySafe {
			return invalid(field+".replay_safe", "writes must be replay-safe through idempotency")
		}
	}
	return nil
}

func ValidateInteractionPolicyFixture(v InteractionPolicyFixture) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := ValidateOperationPolicies(v.Operations); err != nil {
		return err
	}
	operations := make(map[string]OperationPolicy, len(v.Operations))
	for _, policy := range v.Operations {
		operations[policy.ID] = policy
	}
	seen := make(map[string]bool, len(v.Cases))
	for i, interactionCase := range v.Cases {
		field := fmt.Sprintf("cases[%d]", i)
		if strings.TrimSpace(interactionCase.ID) == "" || seen[interactionCase.ID] {
			return invalid(field+".id", "is required and must be unique")
		}
		seen[interactionCase.ID] = true
		if !validInteractionSource(interactionCase.Source) {
			return invalid(field+".source", "is not a stable v1 interaction source")
		}
		if strings.TrimSpace(interactionCase.Utterance) == "" {
			return invalid(field+".utterance", "is required")
		}
		if !validInteractionDisposition(interactionCase.Disposition) {
			return invalid(field+".disposition", "is not a stable v1 disposition")
		}
		if interactionCase.Source == SourceMailContent && interactionCase.Disposition != DispositionIgnoreUntrusted {
			return invalid(field+".disposition", "mail content must be ignored as untrusted authorization")
		}
		if interactionCase.Disposition == DispositionClarify || interactionCase.Disposition == DispositionIgnoreUntrusted {
			if interactionCase.ExpectedOperation != "" || interactionCase.ExpectedEffect != "" || interactionCase.ExpectedConsent != "" {
				return invalid(field+".expected_operation", "non-dispatch cases must not name an operation")
			}
			continue
		}
		policy, ok := operations[interactionCase.ExpectedOperation]
		if !ok {
			return invalid(field+".expected_operation", "must reference a published operation")
		}
		if interactionCase.ExpectedEffect != policy.Effect || interactionCase.ExpectedConsent != policy.Consent {
			return invalid(field, "expected effect and consent must match the operation policy")
		}
		if policy.RequiresUniqueTarget && interactionCase.Resolution.CandidateCount != 1 {
			return invalid(field+".resolution.candidate_count", "must be exactly one for the expected operation")
		}
		if interactionCase.Source == SourceModel && policy.ID == OperationFocusCreate {
			return invalid(field+".expected_operation", "model-originated work must use Focus suggestion")
		}
	}
	return nil
}

func ValidateActionResult(v ActionResult) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if v.Error != nil {
		if !validErrorCode(v.Error.Code) {
			return invalid("error.code", "is not a stable v1 error code")
		}
		return nil // current Row is allowed as refresh context on safe failures.
	}
	hasSuccess := v.Row != nil || v.RemovedRowID != "" || len(v.Source) != 0 || v.Navigation != nil || v.Launch != nil
	if !hasSuccess {
		return invalid("result", "must contain an authoritative result")
	}
	return nil
}

func ValidateAttentionRow(v AttentionRow) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := validateID("id", v.ID); err != nil {
		return err
	}
	if err := validateID("source_id", v.SourceID); err != nil {
		return err
	}
	if strings.TrimSpace(v.SourceKind) == "" {
		return invalid("source_kind", "is required")
	}
	if v.ID != v.SourceKind+":"+v.SourceID {
		return invalid("id", "must equal <source_kind>:<source_id>")
	}
	if err := validateProject("project", v.Project); err != nil {
		return err
	}
	if err := validateTimestamp("timestamp", v.Timestamp, true); err != nil {
		return err
	}
	return validateProvenance(v.Provenance)
}

func ValidateAttentionSnapshot(v AttentionSnapshot) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	if err := validateTimestamp("generated_at", v.GeneratedAt, true); err != nil {
		return err
	}
	if err := validateID("revision", v.Revision); err != nil {
		return err
	}
	for i, row := range v.Rows {
		if err := ValidateAttentionRow(row); err != nil {
			err.Field = fmt.Sprintf("rows[%d].%s", i, err.Field)
			return err
		}
	}
	if v.Window != nil {
		if v.Window.State != AttentionStateCurrent && v.Window.State != AttentionStateEmpty {
			return invalid("window.state", "must be current or empty")
		}
		if v.Window.Limit < 1 {
			return invalid("window.limit", "must be positive")
		}
		if v.Window.Returned != len(v.Rows) {
			return invalid("window.returned", "must equal the number of returned rows")
		}
		if v.Window.Returned > v.Window.Limit {
			return invalid("window.returned", "must not exceed the window limit")
		}
		if v.Counts.Total < v.Window.Returned {
			return invalid("counts.total", "must not be less than returned rows")
		}
		if v.Window.Truncated != (v.Counts.Total > v.Window.Returned) {
			return invalid("window.truncated", "must match the authoritative total")
		}
		if (v.Counts.Total == 0) != (v.Window.State == AttentionStateEmpty) {
			return invalid("window.state", "must match whether the authoritative snapshot is empty")
		}
	}
	return nil
}

func validateVersion(v int) *ContractError {
	if v != SchemaVersion {
		return &ContractError{Code: ErrorIncompatibleVersion, Message: "unsupported attention schema version", Field: "schema_version"}
	}
	return nil
}

func validateID(field, value string) *ContractError {
	if strings.TrimSpace(value) == "" {
		return invalid(field, "is required")
	}
	return nil
}

func validateProject(field string, p ProjectReference) *ContractError {
	if err := validateID(field+".peer_id", p.PeerID); err != nil {
		return err
	}
	if strings.TrimSpace(p.DisplayName) == "" {
		return invalid(field+".display_name", "is required")
	}
	return nil
}

func validateTimestamp(field, value string, required bool) *ContractError {
	if value == "" && !required {
		return nil
	}
	if value == "" {
		return invalid(field, "is required")
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || !strings.HasSuffix(value, "Z") || t.Location() != time.UTC {
		return invalid(field, "must be a UTC RFC3339Nano timestamp")
	}
	return nil
}

func validLifecycle(v string) bool {
	return v == FocusInbox || v == FocusToday || v == FocusLater || v == FocusDone
}

func validErrorCode(v string) bool {
	return v == ErrorValidation || v == ErrorStale || v == ErrorUnsupported || v == ErrorMissing || v == ErrorIncompatibleVersion || v == ErrorUnavailable || v == ErrorIdempotencyConflict || v == ErrorPermission
}

func validInteractionEffect(v InteractionEffect) bool {
	return v == EffectRead || v == EffectAdvisoryWrite || v == EffectStateWrite || v == EffectExternalWrite || v == EffectCommitment
}

func validConsentRequirement(v ConsentRequirement) bool {
	return v == ConsentNone || v == ConsentExplicitUser || v == ConsentExplicitAcceptance
}

func validInteractionDisposition(v InteractionDisposition) bool {
	return v == DispositionDispatch || v == DispositionSuggest || v == DispositionClarify || v == DispositionIgnoreUntrusted
}

func validInteractionSource(v InteractionSource) bool {
	return v == SourceUser || v == SourceModel || v == SourceMailContent
}

func validateProvenance(values []ProvenanceReference) *ContractError {
	for i, p := range values {
		if err := validateID(fmt.Sprintf("provenance[%d].source_id", i), p.SourceID); err != nil {
			return err
		}
		if strings.TrimSpace(p.Kind) == "" {
			return invalid(fmt.Sprintf("provenance[%d].kind", i), "is required")
		}
		if err := validateTimestamp(fmt.Sprintf("provenance[%d].created_at", i), p.CreatedAt, false); err != nil {
			return err
		}
	}
	return nil
}

func invalid(field, message string) *ContractError {
	return &ContractError{Code: ErrorValidation, Message: message, Field: field}
}
