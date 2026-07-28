package codehostbroker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

func ValidateRepository(repository RepositoryIdentity) *ContractError {
	if repository.Host == "" || len(repository.Host) > 255 || strings.Contains(repository.Host, "://") || strings.ContainsAny(repository.Host, "/?#") || net.ParseIP(repository.Host) != nil {
		return invalid("repository.host", "host must be a bounded DNS host name")
	}
	if repository.Owner == "" || repository.Name == "" {
		return invalid("repository", "owner and name are required")
	}
	if repository.FullName != repository.Owner+"/"+repository.Name {
		return invalid("repository.full_name", "full_name must equal owner/name")
	}
	if tooLong(repository.Owner, 255) || tooLong(repository.Name, 255) || tooLong(repository.FullName, 512) || tooLong(repository.ProviderID, 512) {
		return tooLarge("repository", "repository identity exceeds its bound")
	}
	return nil
}

func ValidatePullRequestIdentity(identity PullRequestIdentity) *ContractError {
	if identity.ConnectionID == "" || tooLong(identity.ConnectionID, 128) {
		return invalid("pull_request.connection_id", "bounded connection_id is required")
	}
	if err := ValidateRepository(identity.Repository); err != nil {
		err.Field = "pull_request." + err.Field
		return err
	}
	if identity.ProviderID == "" || tooLong(identity.ProviderID, 512) {
		return invalid("pull_request.provider_id", "bounded provider PR identity is required")
	}
	if identity.Number <= 0 {
		return invalid("pull_request.number", "positive repository-local number is required")
	}
	return nil
}

func ValidateRef(ref RefIdentity, field string) *ContractError {
	if err := ValidateRepository(ref.Repository); err != nil {
		err.Field = field + "." + err.Field
		return err
	}
	if ref.Name == "" || tooLong(ref.Name, 1024) {
		return invalid(field+".name", "bounded ref name is required")
	}
	if ref.SHA == "" || tooLong(ref.SHA, 128) {
		return invalid(field+".sha", "bounded commit SHA is required")
	}
	return nil
}

func ValidateRequest(request Request) *ContractError {
	if request.Version != Version {
		return &ContractError{Code: ErrorIncompatibleVersion, Message: "unsupported code-host broker version", Field: "version", Retry: RetryNone}
	}
	policy, ok := Policy(request.Operation)
	if !ok {
		return &ContractError{Code: ErrorUnsupportedOperation, Message: "unsupported code-host operation", Field: "operation", Retry: RetryNone}
	}
	if request.ConnectionID == "" || tooLong(request.ConnectionID, 128) {
		return invalid("connection_id", "bounded connection_id is required")
	}
	if err := ValidateRepository(request.Repository); err != nil {
		return err
	}
	if request.Limit < 0 || request.Limit > policy.Bounds.PageSize {
		return invalid("limit", "page limit exceeds the operation bound")
	}
	if tooLong(request.Query, MaxTextBytes) || tooLong(request.Order, 128) || tooLong(request.Cursor, MaxTextBytes) {
		return tooLarge("query", "query, order, or cursor exceeds its bound")
	}
	if requiresPullRequest(request.Operation) {
		if request.PullRequest == nil {
			return invalid("pull_request", "repository-qualified pull request identity is required")
		}
		if err := ValidatePullRequestIdentity(*request.PullRequest); err != nil {
			return err
		}
		if request.PullRequest.ConnectionID != request.ConnectionID || request.PullRequest.Repository != request.Repository {
			return invalid("pull_request", "pull request identity must match request connection and repository")
		}
	}
	if IsMutation(request.Operation) {
		if request.IntentSource != "user" {
			return invalid("intent_source", "mutations require intent_source user")
		}
		if request.Consent != policy.Consent {
			return invalid("consent", "mutation consent does not match operation policy")
		}
		if request.IdempotencyKey == "" || tooLong(request.IdempotencyKey, policy.Bounds.IdempotencyBytes) {
			return invalid("idempotency_key", "bounded idempotency key is required")
		}
		if request.CapabilityRevision == "" || tooLong(request.CapabilityRevision, 512) {
			return invalid("capability_revision", "bounded capability revision is required")
		}
		if request.ObservationRevision == "" || tooLong(request.ObservationRevision, 512) {
			return invalid("observation_revision", "bounded observation revision is required")
		}
		if request.ReconciliationKey == "" || tooLong(request.ReconciliationKey, 512) {
			return invalid("reconciliation_key", "bounded reconciliation key is required")
		}
		if len(request.Payload) == 0 || len(request.Payload) > policy.Bounds.BodyBytes {
			return tooLarge("payload", "bounded mutation payload is required")
		}
		if err := validateMutationPayload(request); err != nil {
			return err
		}
	}
	return nil
}

func ValidateResponse(response Response) *ContractError {
	if response.Version != Version {
		return &ContractError{Code: ErrorIncompatibleVersion, Message: "unsupported code-host broker version", Field: "version", Retry: RetryNone}
	}
	policy, ok := Policy(response.Operation)
	if !ok {
		return &ContractError{Code: ErrorUnsupportedOperation, Message: "unsupported code-host operation", Field: "operation", Retry: RetryNone}
	}
	if !reflect.DeepEqual(response.Policy, policy) || !reflect.DeepEqual(response.Bounds, policy.Bounds) {
		return invalid("policy", "response policy and bounds must match the authoritative registry")
	}
	if response.Provider == "" || tooLong(response.Provider, 64) || response.ConnectionID == "" || tooLong(response.ConnectionID, 128) {
		return invalid("connection_id", "bounded provider and connection identity are required")
	}
	if err := ValidateRepository(response.Repository); err != nil {
		return err
	}
	if response.CapabilityRevision == "" || response.ObservationRevision == "" {
		return invalid("capability_revision", "capability and observation revisions are required")
	}
	if _, err := time.Parse(time.RFC3339Nano, response.ObservedAt); err != nil {
		return invalid("observed_at", "RFC3339 observed_at is required")
	}
	if _, err := time.Parse(time.RFC3339Nano, response.RateLimit.ObservedAt); err != nil {
		return invalid("rate_limit.observed_at", "RFC3339 rate-limit observation is required")
	}
	if !validFreshness(response.Freshness) || !validCompleteness(response.Completeness) {
		return invalid("freshness", "freshness and completeness must use declared values")
	}
	if response.DurationMS < 0 || response.DurationMS > int64(policy.Bounds.DurationMS) {
		return tooLarge("duration_ms", "duration exceeds the operation bound")
	}
	if len(response.PartialFailures) > policy.Bounds.PartialFailures {
		return tooLarge("partial_failures", "partial failure count exceeds the operation bound")
	}
	for i, failure := range response.PartialFailures {
		if failure.Section == "" || failure.Code == "" || tooLong(failure.Message, policy.Bounds.ErrorDetailBytes) {
			return invalid(fmt.Sprintf("partial_failures.%d", i), "partial failure is incomplete or unbounded")
		}
	}
	if response.Page != nil && (response.Page.Limit < 0 || response.Page.Limit > policy.Bounds.PageSize || response.Page.Count < 0 || response.Page.Count > policy.Bounds.Items) {
		return invalid("page", "page metadata exceeds the operation bound")
	}
	if response.Error == nil && len(response.Result) == 0 {
		return invalid("result", "result is required when error is null")
	}
	resultLimit := policy.Bounds.BodyBytes
	if response.Operation == OperationGetDiff {
		resultLimit = policy.Bounds.DiffBytes
	}
	if len(response.Result) > resultLimit {
		return &ContractError{Code: ErrorOutputTooLarge, Message: "result exceeds the operation output bound", Field: "result", Retry: RetryNone}
	}
	if response.Error != nil {
		if len(response.Result) != 0 && string(response.Result) != "null" {
			return invalid("result", "top-level error and result are mutually exclusive")
		}
		if !containsString(errorCodes, response.Error.Code) || !containsRetry(RetryGuidanceValues(), response.Error.Retry) || tooLong(response.Error.Message, policy.Bounds.ErrorDetailBytes) {
			return invalid("error", "error code, retry guidance, and detail must be bounded declared values")
		}
	}
	if response.Reconciliation != nil {
		if !validReconciliation(response.Reconciliation.Status) || response.Reconciliation.Key == "" || tooLong(response.Reconciliation.Key, 512) || tooLong(response.Reconciliation.Detail, policy.Bounds.ErrorDetailBytes) {
			return invalid("reconciliation", "reconciliation state is invalid or unbounded")
		}
	}
	if response.Receipt != nil && (response.Receipt.OperationID == "" || tooLong(response.Receipt.OperationID, 512) || tooLong(response.Receipt.ProviderReceiptID, 512) || tooLong(response.Receipt.TargetRevision, 512)) {
		return invalid("receipt", "receipt is incomplete or unbounded")
	}
	return nil
}

func ValidateCursorFingerprint(expected, actual string) *ContractError {
	if expected == "" || actual == "" || expected != actual {
		return &ContractError{Code: ErrorCursorMismatch, Message: "cursor does not match the originating operation and scope", Field: "cursor", Retry: RetryNone}
	}
	return nil
}

func CursorFingerprint(material CursorMaterial) (string, *ContractError) {
	if material.Version != Version || material.Provider == "" || material.ConnectionID == "" || !IsOperation(material.Operation) {
		return "", invalid("cursor", "cursor material is incomplete")
	}
	if len(material.Repositories) == 0 || len(material.Repositories) > MaxRepositoryScopes || tooLong(material.Query, MaxTextBytes) || tooLong(material.Order, 128) || tooLong(material.Position, 512) {
		return "", tooLarge("cursor", "cursor material exceeds its bound")
	}
	return fingerprint("cursor", material)
}

func RevisionFingerprint(kind string, material RevisionMaterial) (string, *ContractError) {
	if kind == "" || material.ConnectionID == "" {
		return "", invalid("revision", "revision kind and connection are required")
	}
	if err := ValidateRepository(material.Repository); err != nil {
		return "", err
	}
	if material.PullRequest != nil {
		if err := ValidatePullRequestIdentity(*material.PullRequest); err != nil {
			return "", err
		}
	}
	if material.Base != nil {
		if err := ValidateRef(*material.Base, "base"); err != nil {
			return "", err
		}
	}
	if material.Head != nil {
		if err := ValidateRef(*material.Head, "head"); err != nil {
			return "", err
		}
	}
	return fingerprint(kind, material)
}

func requiresPullRequest(operation Operation) bool {
	return operation != OperationCapabilities &&
		operation != OperationListPullRequests &&
		operation != OperationSearchPullRequests &&
		operation != OperationCreatePullRequest
}

func validateMutationPayload(request Request) *ContractError {
	switch request.Operation {
	case OperationCreatePullRequest:
		var payload CreatePullRequestPayload
		if err := decodePayload(request.Payload, &payload); err != nil {
			return err
		}
		if err := ValidateRef(payload.Base, "payload.base"); err != nil {
			return err
		}
		if err := ValidateRef(payload.Head, "payload.head"); err != nil {
			return err
		}
		if payload.Title == "" || tooLong(payload.Title, MaxTextBytes) || tooLong(payload.Body, MaxBodyBytes) {
			return tooLarge("payload", "creation title or body is empty or unbounded")
		}
	case OperationComment:
		var payload CommentPayload
		if err := decodePayload(request.Payload, &payload); err != nil {
			return err
		}
		if payload.ExpectedHeadSHA == "" || payload.Body == "" || tooLong(payload.ExpectedHeadSHA, 128) || tooLong(payload.Body, MaxBodyBytes) {
			return tooLarge("payload", "comment head or body is empty or unbounded")
		}
	case OperationSubmitReview, OperationApprove, OperationRequestChanges:
		var payload ReviewPayload
		if err := decodePayload(request.Payload, &payload); err != nil {
			return err
		}
		if payload.ExpectedHeadSHA == "" || tooLong(payload.ExpectedHeadSHA, 128) || tooLong(payload.Body, MaxBodyBytes) {
			return tooLarge("payload", "review head or body is unbounded")
		}
	case OperationRetarget:
		var payload RetargetPayload
		if err := decodePayload(request.Payload, &payload); err != nil {
			return err
		}
		if payload.ExpectedHeadSHA == "" || tooLong(payload.ExpectedHeadSHA, 128) {
			return invalid("payload.expected_head_sha", "expected head SHA is required")
		}
		if err := ValidateRef(payload.CurrentBase, "payload.current_base"); err != nil {
			return err
		}
		if err := ValidateRef(payload.NewBase, "payload.new_base"); err != nil {
			return err
		}
	case OperationMarkReady, OperationClose, OperationReopen:
		var payload LifecyclePayload
		if err := decodePayload(request.Payload, &payload); err != nil {
			return err
		}
		if payload.ExpectedHeadSHA == "" || tooLong(payload.ExpectedHeadSHA, 128) {
			return invalid("payload.expected_head_sha", "expected head SHA is required")
		}
	case OperationMerge:
		var payload MergePayload
		if err := decodePayload(request.Payload, &payload); err != nil {
			return err
		}
		if payload.ExpectedHeadSHA == "" || tooLong(payload.ExpectedHeadSHA, 128) || !containsString([]string{"merge", "squash", "rebase"}, payload.Method) || tooLong(payload.CommitTitle, MaxTextBytes) || tooLong(payload.CommitMessage, MaxBodyBytes) {
			return invalid("payload", "merge head, method, or commit text is invalid")
		}
		if err := ValidateRef(payload.ObservedBase, "payload.observed_base"); err != nil {
			return err
		}
	default:
		return &ContractError{Code: ErrorUnsupportedOperation, Message: "mutation payload operation is unsupported", Field: "operation", Retry: RetryNone}
	}
	return nil
}

func decodePayload(raw json.RawMessage, target any) *ContractError {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return invalid("payload", "payload does not match operation schema")
	}
	return nil
}

func fingerprint(prefix string, material any) (string, *ContractError) {
	data, err := json.Marshal(material)
	if err != nil {
		return "", &ContractError{Code: ErrorEncoding, Message: "could not encode revision material", Retry: RetryNone}
	}
	sum := sha256.Sum256(data)
	return prefix + ":" + hex.EncodeToString(sum[:]), nil
}

func invalid(field, message string) *ContractError {
	return &ContractError{Code: ErrorInvalidInput, Message: message, Field: field, Retry: RetryNone}
}

func tooLarge(field, message string) *ContractError {
	return &ContractError{Code: ErrorInputTooLarge, Message: message, Field: field, Retry: RetryNone}
}

func tooLong(value string, limit int) bool {
	return !utf8.ValidString(value) || len(value) > limit
}

func containsString[T ~string](values []T, candidate T) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func containsRetry(values []RetryGuidance, candidate RetryGuidance) bool {
	return containsString(values, candidate)
}

func validFreshness(value Freshness) bool {
	return containsString([]Freshness{FreshnessCurrent, FreshnessStale, FreshnessUnknown, FreshnessUnavailable}, value)
}

func validCompleteness(value Completeness) bool {
	return containsString([]Completeness{CompletenessComplete, CompletenessPartial, CompletenessTruncated, CompletenessUnavailable}, value)
}

func validReconciliation(value ReconciliationStatus) bool {
	return containsString([]ReconciliationStatus{
		ReconciliationApplied,
		ReconciliationReplayed,
		ReconciliationReconciledApplied,
		ReconciliationExternallyCompleted,
		ReconciliationNotApplied,
		ReconciliationInProgress,
		ReconciliationAmbiguous,
	}, value)
}
