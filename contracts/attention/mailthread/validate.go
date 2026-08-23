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
