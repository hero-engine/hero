package mailthread

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hero-engine/hero/contracts/attention"
)

func ValidateIdentity(v Identity) *attention.ContractError {
	if err := required("identity.project_peer_id", v.ProjectPeerID); err != nil {
		return err
	}
	return required("identity.thread_id", v.ThreadID)
}

func ValidateState(v State) *attention.ContractError {
	if v.SchemaVersion != SchemaVersion {
		return incompatible("schema_version")
	}
	if err := ValidateIdentity(v.Identity); err != nil {
		return err
	}
	if v.Revision < 0 {
		return invalid("revision", "must not be negative")
	}
	switch v.Lifecycle {
	case LifecycleOpen:
		if v.Resolution != nil || v.ResolvedAt != "" || v.ArchiveEligibleAt != "" || v.ArchivedAt != "" {
			return invalid("lifecycle", "open threads cannot carry resolution or archive timestamps")
		}
	case LifecycleResolved:
		if v.Resolution == nil || v.ResolvedAt == "" || v.ArchivedAt != "" {
			return invalid("lifecycle", "resolved threads require resolution and resolved_at and cannot carry archived_at")
		}
	case LifecycleArchived:
		if v.ArchivedAt == "" {
			return invalid("archived_at", "is required for archived threads")
		}
	default:
		return invalid("lifecycle", "must be open, resolved, or archived")
	}
	if v.GraceClass != "" && v.GraceClass != GraceInformational && v.GraceClass != GraceLinkedWork {
		return invalid("grace_class", "must be informational or linked_work")
	}
	for field, value := range map[string]string{
		"resolved_at": v.ResolvedAt, "archive_eligible_at": v.ArchiveEligibleAt, "archived_at": v.ArchivedAt,
	} {
		if err := timestamp(field, value); err != nil {
			return err
		}
	}
	if v.Resolution != nil {
		if err := required("resolution.reason", v.Resolution.Reason); err != nil {
			return err
		}
		if err := required("resolution.source", v.Resolution.Source); err != nil {
			return err
		}
	}
	seen := map[string]bool{}
	for _, action := range v.Actions {
		if err := required("actions.idempotency_key", action.IdempotencyKey); err != nil {
			return err
		}
		if seen[action.IdempotencyKey] {
			return invalid("actions.idempotency_key", "must be unique")
		}
		seen[action.IdempotencyKey] = true
		if !canonicalAction(action.ActionID) {
			return invalid("actions.action_id", "is not canonical")
		}
		if decoded, err := hex.DecodeString(action.RequestHash); err != nil || len(decoded) != 32 {
			return invalid("actions.request_hash", "must be a SHA-256 hex digest")
		}
		if err := requiredTimestamp("actions.applied_at", action.AppliedAt); err != nil {
			return err
		}
	}
	events := map[string]bool{}
	for _, event := range v.Events {
		if err := required("events.event_id", event.EventID); err != nil {
			return err
		}
		if events[event.EventID] {
			return invalid("events.event_id", "must be unique")
		}
		events[event.EventID] = true
		if !canonicalEvent(event.Kind) {
			return invalid("events.kind", "is not canonical")
		}
		if decoded, err := hex.DecodeString(event.RequestHash); err != nil || len(decoded) != 32 {
			return invalid("events.request_hash", "must be a SHA-256 hex digest")
		}
		if err := requiredTimestamp("events.applied_at", event.AppliedAt); err != nil {
			return err
		}
		if err := required("events.source", event.Source); err != nil {
			return err
		}
		if err := required("events.source_id", event.SourceID); err != nil {
			return err
		}
		if !canonicalLifecycle(event.FromLifecycle) || !canonicalLifecycle(event.ToLifecycle) {
			return invalid("events.lifecycle", "must be open, resolved, or archived")
		}
		if err := validateMessageIDs("events.prior_message_ids", event.PriorMessageIDs); err != nil {
			return err
		}
	}
	return nil
}

func ValidateEvent(v Event) *attention.ContractError {
	if v.SchemaVersion != SchemaVersion {
		return incompatible("schema_version")
	}
	if err := ValidateIdentity(v.Identity); err != nil {
		return err
	}
	if !canonicalEvent(v.Kind) || v.Kind == EventGraceArchive {
		return invalid("kind", "must be a supported external lifecycle event")
	}
	if err := required("event_id", v.EventID); err != nil {
		return err
	}
	if v.ExpectedRevision < 0 {
		return invalid("expected_revision", "must not be negative")
	}
	if err := requiredTimestamp("occurred_at", v.OccurredAt); err != nil {
		return err
	}
	if err := required("source", v.Source); err != nil {
		return err
	}
	if err := required("source_id", v.SourceID); err != nil {
		return err
	}
	switch v.Kind {
	case EventAdvisoryTerminal, EventSpecOutTerminal:
		wantSource := "peer.advisory"
		if v.Kind == EventSpecOutTerminal {
			wantSource = "peer.spec_out"
		}
		if v.Source != wantSource {
			return invalid("source", "does not match the registered peering event kind")
		}
		if v.Outcome != OutcomeAnswered && v.Outcome != OutcomeCompleted && v.Outcome != OutcomeRejected && v.Outcome != OutcomeCancelled {
			return invalid("outcome", "must be a terminal advisory outcome")
		}
	case EventLinkedTerminal:
		if v.Source != "peer.handoff" && v.Source != "work.registry" {
			return invalid("source", "must be a registered handoff or work authority")
		}
		if v.Outcome != OutcomeCompleted && v.Outcome != OutcomeRejected && v.Outcome != OutcomeCancelled {
			return invalid("outcome", "must be completed, rejected, or cancelled")
		}
	case EventInboundActivity:
		if v.Source != "mail.delivery" {
			return invalid("source", "must be mail.delivery")
		}
		if err := required("message_id", v.MessageID); err != nil {
			return err
		}
	case EventReplySucceeded:
		if v.Source != "mail.reply" {
			return invalid("source", "must be mail.reply")
		}
	case EventActionSucceeded:
		if v.Source != "mail.action" {
			return invalid("source", "must be mail.action")
		}
	case EventForegroundRead:
		if v.Source != "mail.foreground" {
			return invalid("source", "must be mail.foreground")
		}
	default:
		if v.Outcome != "" {
			return invalid("outcome", "is only accepted for terminal events")
		}
	}
	if err := validateMessageIDs("prior_message_ids", v.PriorMessageIDs); err != nil {
		return err
	}
	if len(v.PriorMessageIDs) != 0 && v.Kind != EventReplySucceeded && v.Kind != EventActionSucceeded && v.Kind != EventForegroundRead {
		return invalid("prior_message_ids", "is only accepted for user interaction events")
	}
	return nil
}

func ValidateThreadView(v ThreadView) *attention.ContractError {
	if err := ValidateState(v.State); err != nil {
		return err
	}
	if v.Read.MessageCount < 0 || v.Read.UnreadCount < 0 || v.Read.UnreadCount > v.Read.MessageCount {
		return invalid("read", "counts must be non-negative and unread_count cannot exceed message_count")
	}
	seen := map[string]bool{}
	for _, action := range v.Actions {
		if !canonicalAction(action.ID) || seen[action.ID] {
			return invalid("actions", "must contain unique canonical lifecycle actions")
		}
		seen[action.ID] = true
		if action.RequiredRowRevision != v.State.Revision || !action.RequiresIdempotency || action.OperationID == "" || len(action.InputSchema) == 0 || !json.Valid(action.InputSchema) {
			return invalid("actions", "descriptor is incomplete or carries the wrong revision")
		}
	}
	return nil
}

func ValidateThreadListRequest(v ThreadListRequest) *attention.ContractError {
	if v.SchemaVersion != SchemaVersion {
		return incompatible("schema_version")
	}
	if v.ProjectPeerID != "" {
		if err := required("project_peer_id", v.ProjectPeerID); err != nil {
			return err
		}
	}
	if v.Bucket != "" && !canonicalBucket(v.Bucket) {
		return invalid("bucket", "must be needs_attention, updates, or history")
	}
	if v.Lifecycle != "" && !canonicalLifecycle(v.Lifecycle) {
		return invalid("lifecycle", "must be open, resolved, or archived")
	}
	if v.Limit < 0 || v.Limit > MaxListLimit {
		return invalid("limit", "must be between zero and 100")
	}
	if len(v.Cursor) > 4096 {
		return invalid("cursor", "must be at most 4096 bytes")
	}
	return nil
}

func ValidateThreadSummary(v ThreadSummary) *attention.ContractError {
	if err := ValidateIdentity(v.Identity); err != nil {
		return err
	}
	if v.Project.PeerID != v.Identity.ProjectPeerID {
		return invalid("project.peer_id", "must match identity.project_peer_id")
	}
	if err := required("subject", v.Subject); err != nil {
		return err
	}
	if err := requiredTimestamp("activity_at", v.ActivityAt); err != nil {
		return err
	}
	if !canonicalLifecycle(v.Lifecycle) || !canonicalBucket(v.Bucket) {
		return invalid("classification", "must contain a canonical lifecycle and bucket")
	}
	if v.MessageCount < 1 || v.UnreadCount < 0 || v.UnreadCount > v.MessageCount || v.Unread != (v.UnreadCount > 0) || v.Revision < 0 {
		return invalid("counts", "message and unread counts are inconsistent")
	}
	switch v.Bucket {
	case BucketNeedsAttention:
		if v.Lifecycle != LifecycleOpen || !v.Actionable {
			return invalid("bucket", "needs_attention requires an open actionable thread")
		}
	case BucketUpdates:
		if v.Lifecycle == LifecycleArchived || v.Actionable {
			return invalid("bucket", "updates requires a non-actionable open or resolved thread")
		}
	case BucketHistory:
		if v.Lifecycle != LifecycleArchived || v.Actionable {
			return invalid("bucket", "history requires an archived non-actionable thread")
		}
	}
	for _, action := range v.Actions {
		if action.RequiredRowRevision != v.Revision || !action.RequiresIdempotency || action.ID == "" || action.OperationID == "" || len(action.InputSchema) == 0 || !json.Valid(action.InputSchema) {
			return invalid("actions", "descriptor is incomplete or carries the wrong revision")
		}
	}
	return nil
}

func ValidateThreadListResponse(v ThreadListResponse) *attention.ContractError {
	if v.SchemaVersion != SchemaVersion {
		return incompatible("schema_version")
	}
	if v.Error != nil {
		if strings.TrimSpace(v.Error.Message) == "" {
			return invalid("error.message", "is required")
		}
		return nil
	}
	if v.Revision == "" || v.Counts.Total < 0 || v.Counts.Actionable < 0 || v.Counts.ActionableUnread < 0 || v.Counts.ActionableUnread > v.Counts.Actionable || v.Counts.Actionable > v.Counts.Total {
		return invalid("counts", "thread counts and revision are required and must be consistent")
	}
	for _, item := range v.Items {
		if err := ValidateThreadSummary(item); err != nil {
			return err
		}
	}
	return nil
}

func ValidateThreadDetailResponse(v ThreadDetailResponse) *attention.ContractError {
	if v.SchemaVersion != SchemaVersion {
		return incompatible("schema_version")
	}
	if v.Error != nil {
		if strings.TrimSpace(v.Error.Message) == "" {
			return invalid("error.message", "is required")
		}
		return nil
	}
	if v.Summary == nil || v.Thread == nil || len(v.Messages) == 0 {
		return invalid("detail", "summary, thread, and messages are required")
	}
	if err := ValidateThreadSummary(*v.Summary); err != nil {
		return err
	}
	if err := ValidateThreadView(*v.Thread); err != nil {
		return err
	}
	if v.Summary.Identity != v.Thread.State.Identity || len(v.Messages) != v.Summary.MessageCount {
		return invalid("detail", "summary, state, and messages must describe one thread")
	}
	for _, message := range v.Messages {
		if err := attention.ValidateMailEnvelope(message.Envelope); err != nil {
			return err
		}
		threadID := message.Envelope.ThreadID
		if threadID == "" {
			threadID = message.Envelope.ID
		}
		if message.Envelope.Recipient.PeerID != v.Summary.Identity.ProjectPeerID || threadID != v.Summary.Identity.ThreadID {
			return invalid("messages", "every envelope must belong to the exact thread identity")
		}
		if message.Receipt != nil {
			if err := attention.ValidateMailReceipt(*message.Receipt); err != nil {
				return err
			}
			if message.Receipt.EnvelopeID != message.Envelope.ID || message.Unread != (message.Receipt.ReadAt == "") {
				return invalid("messages.receipt", "must match its envelope and unread value")
			}
		} else if !message.Unread {
			return invalid("messages.unread", "messages without receipts are unread")
		}
	}
	return nil
}

func ValidateCapabilitySet(v CapabilitySet) *attention.ContractError {
	want := map[string]bool{ActionMarkRead: true, ActionMarkUnread: true, ActionResolve: true, ActionReopen: true, ActionArchive: true, ActionRestore: true}
	seen := map[string]bool{}
	for _, group := range [][]attention.ActionDescriptor{v.Receipt, v.Lifecycle} {
		for _, action := range group {
			if !want[action.ID] || seen[action.ID] || action.OperationID == "" || action.Effect == "" || action.Consent == "" || !action.RequiresIdempotency || len(action.InputSchema) == 0 || !json.Valid(action.InputSchema) {
				return invalid("actions", "must contain each canonical action exactly once with a complete descriptor")
			}
			seen[action.ID] = true
		}
	}
	if len(seen) != len(want) {
		return invalid("actions", "must contain mark_read, mark_unread, resolve, reopen, archive, and restore")
	}
	return nil
}

func ValidateActionRequest(v ActionRequest) *attention.ContractError {
	if v.SchemaVersion != SchemaVersion {
		return incompatible("schema_version")
	}
	if err := ValidateIdentity(v.Identity); err != nil {
		return err
	}
	if !canonicalAction(v.ActionID) {
		return &attention.ContractError{Code: attention.ErrorUnsupported, Message: "action is not supported by the Mail thread contract", Field: "action_id"}
	}
	if v.ThreadRevision < 0 {
		return invalid("thread_revision", "must not be negative")
	}
	if err := required("idempotency_key", v.IdempotencyKey); err != nil {
		return err
	}
	var input ActionInput
	if len(v.Input) != 0 {
		decoder := json.NewDecoder(bytes.NewReader(v.Input))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			return invalid("input", "must match the action input schema")
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			return invalid("input", "must contain exactly one JSON object")
		}
	}
	if v.ActionID == ActionResolve {
		if err := required("input.reason", input.Reason); err != nil {
			return err
		}
		if err := required("input.source", input.Source); err != nil {
			return err
		}
		if input.GraceClass != "" && input.GraceClass != GraceInformational && input.GraceClass != GraceLinkedWork {
			return invalid("input.grace_class", "must be informational or linked_work")
		}
	} else if input != (ActionInput{}) {
		return invalid("input", "fields are only accepted for resolve")
	}
	return nil
}

func ValidateActionResponse(v ActionResponse) *attention.ContractError {
	if v.SchemaVersion != SchemaVersion {
		return incompatible("schema_version")
	}
	if v.Error != nil {
		if strings.TrimSpace(v.Error.Message) == "" {
			return invalid("error.message", "is required")
		}
		return nil
	}
	if v.Thread == nil {
		return invalid("thread", "is required on success")
	}
	return ValidateThreadView(*v.Thread)
}

func ValidateMigrationResult(v MigrationResult) *attention.ContractError {
	if v.SchemaVersion != SchemaVersion || v.Source != "mailread_v1" {
		return incompatible("migration")
	}
	return ValidateThreadView(v.Thread)
}

func ValidateContractResponse(v ContractResponse) *attention.ContractError {
	if v.SchemaVersion != SchemaVersion || v.BundleVersion != BundleVersion || v.Compatibility != Compatibility {
		return incompatible("contract")
	}
	decoded, err := hex.DecodeString(v.BundleManifestSHA256)
	if err != nil || len(decoded) != 32 || strings.ToLower(v.BundleManifestSHA256) != v.BundleManifestSHA256 {
		return invalid("bundle_manifest_sha256", "must be a lowercase SHA-256 hex digest")
	}
	return nil
}

func canonicalAction(v string) bool {
	return v == ActionResolve || v == ActionReopen || v == ActionArchive || v == ActionRestore
}

func canonicalEvent(v EventKind) bool {
	switch v {
	case EventForegroundRead, EventReplySucceeded, EventActionSucceeded, EventAdvisoryTerminal, EventSpecOutTerminal, EventLinkedTerminal, EventInboundActivity, EventGraceArchive:
		return true
	default:
		return false
	}
}

func canonicalLifecycle(v Lifecycle) bool {
	return v == LifecycleOpen || v == LifecycleResolved || v == LifecycleArchived
}

func canonicalBucket(v Bucket) bool {
	return v == BucketNeedsAttention || v == BucketUpdates || v == BucketHistory
}

func validateMessageIDs(field string, values []string) *attention.ContractError {
	seen := map[string]bool{}
	for _, value := range values {
		if err := required(field, value); err != nil {
			return err
		}
		if seen[value] {
			return invalid(field, "must contain unique message identities")
		}
		seen[value] = true
	}
	return nil
}

func required(field, value string) *attention.ContractError {
	if strings.TrimSpace(value) == "" || !utf8.ValidString(value) || len(value) > 512 {
		return invalid(field, "is required and must be valid UTF-8 at most 512 bytes")
	}
	return nil
}

func timestamp(field, value string) *attention.ContractError {
	if value == "" {
		return nil
	}
	return requiredTimestamp(field, value)
}

func requiredTimestamp(field, value string) *attention.ContractError {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || !strings.HasSuffix(value, "Z") || parsed.Location() != time.UTC {
		return invalid(field, "must be a UTC RFC3339Nano timestamp")
	}
	return nil
}

func invalid(field, message string) *attention.ContractError {
	return &attention.ContractError{Code: attention.ErrorValidation, Message: message, Field: field}
}

func incompatible(field string) *attention.ContractError {
	return &attention.ContractError{Code: attention.ErrorIncompatibleVersion, Message: "unsupported Mail thread contract version", Field: field}
}
