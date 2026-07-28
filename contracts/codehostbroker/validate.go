package codehostbroker

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

func ValidateRepository(repository RepositoryIdentity) *ContractError {
	if repository.Host == "" || tooLong(repository.Host, 255) || strings.Contains(repository.Host, "://") || strings.ContainsAny(repository.Host, "/?#") || net.ParseIP(repository.Host) != nil {
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
	if request.Provider == "" || tooLong(request.Provider, 64) || request.ConnectionID == "" || tooLong(request.ConnectionID, 128) {
		return invalid("connection_id", "bounded provider and connection_id are required")
	}
	if err := ValidateRepository(request.Repository); err != nil {
		return err
	}
	if len(request.Repositories)+1 > policy.Bounds.RepositoryScopes {
		return tooLarge("repositories", "repository scope count exceeds the operation bound")
	}
	for i, repository := range request.Repositories {
		if err := ValidateRepository(repository); err != nil {
			err.Field = fmt.Sprintf("repositories.%d.%s", i, err.Field)
			return err
		}
	}
	if request.Limit < 0 || request.Limit > policy.Bounds.PageSize {
		return invalid("limit", "page limit exceeds the operation bound")
	}
	if tooLong(request.Query, MaxTextBytes) || tooLong(request.Order, 128) || tooLong(request.Cursor, MaxTextBytes) {
		return tooLarge("query", "query, order, or cursor exceeds its bound")
	}
	if request.Cursor != "" {
		if err := validateRequestCursor(request); err != nil {
			return err
		}
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
	if response.CapabilityRevision == "" || tooLong(response.CapabilityRevision, 512) ||
		response.ObservationRevision == "" || tooLong(response.ObservationRevision, 512) {
		return invalid("capability_revision", "bounded capability and observation revisions are required")
	}
	if _, err := time.Parse(time.RFC3339Nano, response.ObservedAt); err != nil {
		return invalid("observed_at", "RFC3339 observed_at is required")
	}
	if _, err := time.Parse(time.RFC3339Nano, response.RateLimit.ObservedAt); err != nil {
		return invalid("rate_limit.observed_at", "RFC3339 rate-limit observation is required")
	}
	if tooLong(response.RateLimit.Resource, 128) || tooLong(response.RateLimit.ResetAt, 64) {
		return tooLarge("rate_limit", "rate-limit resource or reset time exceeds its bound")
	}
	if response.RateLimit.ResetAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, response.RateLimit.ResetAt); err != nil {
			return invalid("rate_limit.reset_at", "rate-limit reset_at must be RFC3339")
		}
	}
	if !validFreshness(response.Freshness) || !validCompleteness(response.Completeness) {
		return invalid("freshness", "freshness and completeness must use declared values")
	}
	if response.DurationMS < 0 || response.DurationMS > int64(policy.Bounds.DurationMS) {
		return tooLarge("duration_ms", "duration exceeds the operation bound")
	}
	if response.Redirects < 0 || response.Redirects > policy.Bounds.Redirects {
		return tooLarge("redirects", "redirect count exceeds the operation bound")
	}
	if response.JournalEntries < 0 || response.JournalEntries > policy.Bounds.JournalEntries {
		return tooLarge("journal_entries", "journal entry count exceeds the operation bound")
	}
	if response.RateLimit.Limit < 0 || response.RateLimit.Remaining < 0 || response.RateLimit.RetryAfter < 0 ||
		(response.RateLimit.Limit > 0 && response.RateLimit.Remaining > response.RateLimit.Limit) {
		return invalid("rate_limit", "rate-limit values must be non-negative and internally consistent")
	}
	if len(response.PartialFailures) > policy.Bounds.PartialFailures {
		return tooLarge("partial_failures", "partial failure count exceeds the operation bound")
	}
	for i, failure := range response.PartialFailures {
		if failure.Section == "" || tooLong(failure.Section, 256) || !containsString(errorCodes, failure.Code) || tooLong(failure.Message, policy.Bounds.ErrorDetailBytes) {
			return invalid(fmt.Sprintf("partial_failures.%d", i), "partial failure is incomplete or unbounded")
		}
	}
	if response.Page != nil {
		if response.Page.Limit < 0 || response.Page.Limit > policy.Bounds.PageSize || response.Page.Count < 0 || response.Page.Count > policy.Bounds.Items || tooLong(response.Page.NextCursor, MaxTextBytes) {
			return invalid("page", "page metadata exceeds the operation bound")
		}
		if response.Page.NextCursor != "" {
			if err := validateResponseCursor(response, response.Page.NextCursor); err != nil {
				return err
			}
		}
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
		if tooLong(response.Error.Field, 512) || tooLong(response.Error.RetryAt, 64) {
			return tooLarge("error", "error field or retry time exceeds its bound")
		}
		if response.Error.RetryAt != "" {
			if _, err := time.Parse(time.RFC3339Nano, response.Error.RetryAt); err != nil {
				return invalid("error.retry_at", "error retry_at must be RFC3339")
			}
		}
		if response.Error.Retry != RetryForError(response.Error.Code) {
			return invalid("error.retry", "retry guidance does not match the normalized error code")
		}
		if response.Error.Code == ErrorAmbiguousResult &&
			(response.Reconciliation == nil || response.Reconciliation.Status != ReconciliationAmbiguous) {
			return invalid("reconciliation", "ambiguous_result requires ambiguous reconciliation state")
		}
	} else if err := validateOperationResult(response.Operation, response.Result, policy.Bounds); err != nil {
		return err
	}
	if response.Reconciliation != nil {
		if !validReconciliation(response.Reconciliation.Status) || response.Reconciliation.Key == "" || tooLong(response.Reconciliation.Key, 512) {
			return invalid("reconciliation", "reconciliation state is invalid or unbounded")
		}
	}
	if response.Receipt != nil && (response.Receipt.OperationID == "" || tooLong(response.Receipt.OperationID, 512) || tooLong(response.Receipt.ProviderReceiptID, 512) || tooLong(response.Receipt.TargetRevision, 512)) {
		return invalid("receipt", "receipt is incomplete or unbounded")
	}
	if IsMutation(response.Operation) && response.Error == nil && (response.Receipt == nil || response.Reconciliation == nil) {
		return invalid("reconciliation", "successful mutation responses require receipt and reconciliation state")
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
	material.Repositories = normalizedRepositoryScope(material.Repositories)
	if material.Version != Version || material.Provider == "" || tooLong(material.Provider, 64) ||
		material.ConnectionID == "" || tooLong(material.ConnectionID, 128) || !IsOperation(material.Operation) {
		return "", invalid("cursor", "cursor material is incomplete")
	}
	if len(material.Repositories) == 0 || len(material.Repositories) > MaxRepositoryScopes || tooLong(material.Query, MaxTextBytes) || tooLong(material.Order, 128) || tooLong(material.Position, 512) {
		return "", tooLarge("cursor", "cursor material exceeds its bound")
	}
	for _, repository := range material.Repositories {
		if err := ValidateRepository(repository); err != nil {
			err.Field = "cursor.repositories." + err.Field
			return "", err
		}
	}
	return fingerprint("cursor", material)
}

func EncodeCursor(material CursorMaterial) (string, *ContractError) {
	material.Repositories = normalizedRepositoryScope(material.Repositories)
	fingerprintValue, err := CursorFingerprint(material)
	if err != nil {
		return "", err
	}
	data, encodeErr := json.Marshal(CursorEnvelope{Material: material, Fingerprint: fingerprintValue})
	if encodeErr != nil {
		return "", &ContractError{Code: ErrorEncoding, Message: "could not encode cursor", Retry: RetryNone}
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	if tooLong(encoded, MaxTextBytes) {
		return "", tooLarge("cursor", "encoded cursor exceeds its bound")
	}
	return encoded, nil
}

func DecodeCursor(encoded string) (CursorEnvelope, *ContractError) {
	if encoded == "" || tooLong(encoded, MaxTextBytes) {
		return CursorEnvelope{}, &ContractError{Code: ErrorCursorMismatch, Message: "cursor is empty or unbounded", Field: "cursor", Retry: RetryNone}
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return CursorEnvelope{}, &ContractError{Code: ErrorCursorMismatch, Message: "cursor encoding is invalid", Field: "cursor", Retry: RetryNone}
	}
	var envelope CursorEnvelope
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return CursorEnvelope{}, &ContractError{Code: ErrorCursorMismatch, Message: "cursor envelope is invalid", Field: "cursor", Retry: RetryNone}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return CursorEnvelope{}, &ContractError{Code: ErrorCursorMismatch, Message: "cursor envelope contains trailing data", Field: "cursor", Retry: RetryNone}
	}
	envelope.Material.Repositories = normalizedRepositoryScope(envelope.Material.Repositories)
	expected, contractErr := CursorFingerprint(envelope.Material)
	if contractErr != nil || ValidateCursorFingerprint(expected, envelope.Fingerprint) != nil {
		return CursorEnvelope{}, &ContractError{Code: ErrorCursorMismatch, Message: "cursor fingerprint is invalid", Field: "cursor", Retry: RetryNone}
	}
	return envelope, nil
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

func RetryForError(code string) RetryGuidance {
	switch code {
	case ErrorRateLimited:
		return RetryAfter
	case ErrorStaleObservation, ErrorCapabilityChanged, ErrorConflict:
		return RetryRefreshThenRetry
	case ErrorIdempotencyConflict, ErrorOperationInProgress:
		return RetrySameKey
	case ErrorAmbiguousResult:
		return RetryReconcile
	default:
		return RetryNone
	}
}

func validateOperationResult(operation Operation, raw json.RawMessage, bounds Bounds) *ContractError {
	if !json.Valid(raw) {
		return invalid("result", "result must be valid JSON")
	}
	switch operation {
	case OperationCapabilities:
		var result CapabilitiesResult
		if err := decodeResult(raw, &result); err != nil {
			return err
		}
		if len(result.Capabilities) > bounds.Items {
			return tooLarge("result.capabilities", "capability count exceeds the operation bound")
		}
		for i, capability := range result.Capabilities {
			if capability.Policy.Operation == "" || tooLong(string(capability.Policy.Operation), 128) || tooLong(capability.Reason, bounds.ErrorDetailBytes) {
				return invalid(fmt.Sprintf("result.capabilities.%d", i), "capability is incomplete or unbounded")
			}
			if policy, known := Policy(capability.Policy.Operation); known && !reflect.DeepEqual(capability.Policy, policy) {
				return invalid(fmt.Sprintf("result.capabilities.%d.policy", i), "known capability policy does not match the authoritative registry")
			}
		}
	case OperationListPullRequests, OperationSearchPullRequests:
		var result PullRequestsResult
		if err := decodeResult(raw, &result); err != nil {
			return err
		}
		if len(result.PullRequests) > bounds.Items {
			return tooLarge("result.pull_requests", "pull request count exceeds the operation bound")
		}
		for i := range result.PullRequests {
			if err := validatePullRequest(result.PullRequests[i], fmt.Sprintf("result.pull_requests.%d", i)); err != nil {
				return err
			}
		}
	case OperationGetPullRequest:
		var result PullRequest
		if err := decodeResult(raw, &result); err != nil {
			return err
		}
		return validatePullRequest(result, "result")
	case OperationGetCommits:
		var result CommitsResult
		if err := decodeResult(raw, &result); err != nil {
			return err
		}
		if len(result.Commits) > bounds.Items {
			return tooLarge("result.commits", "commit count exceeds the operation bound")
		}
		for i, commit := range result.Commits {
			if commit.SHA == "" || tooLong(commit.SHA, 128) || commit.Message == "" || tooLong(commit.Message, bounds.TextBytes) ||
				tooLong(commit.AuthoredAt, 64) || tooLong(commit.URL, 2048) {
				return invalid(fmt.Sprintf("result.commits.%d", i), "commit is incomplete or unbounded")
			}
			if err := validateActor(commit.Author, fmt.Sprintf("result.commits.%d.author", i)); err != nil {
				return err
			}
		}
	case OperationGetDiff:
		var result DiffResult
		if err := decodeResult(raw, &result); err != nil {
			return err
		}
		if len(result.Files) > bounds.DiffFiles {
			return tooLarge("result.files", "diff file count exceeds the operation bound")
		}
		hunks := 0
		for i, file := range result.Files {
			hunks += len(file.Hunks)
			if file.Path == "" || tooLong(file.Path, 4096) || file.Status == "" || tooLong(file.Status, 64) ||
				file.Additions < 0 || file.Deletions < 0 {
				return invalid(fmt.Sprintf("result.files.%d", i), "diff file is incomplete, negative, or unbounded")
			}
			for j, hunk := range file.Hunks {
				if hunk.Header == "" || tooLong(hunk.Header, bounds.TextBytes) || tooLong(hunk.Patch, bounds.DiffBytes) {
					return tooLarge(fmt.Sprintf("result.files.%d.hunks.%d", i, j), "diff hunk is empty or unbounded")
				}
			}
		}
		if hunks > bounds.DiffHunks {
			return tooLarge("result.files.hunks", "diff hunk count exceeds the operation bound")
		}
	case OperationGetChecks:
		var result ChecksResult
		if err := decodeResult(raw, &result); err != nil {
			return err
		}
		if len(result.Checks) > bounds.Items {
			return tooLarge("result.checks", "check count exceeds the operation bound")
		}
		for i, check := range result.Checks {
			if check.Name == "" || tooLong(check.ProviderID, 512) || tooLong(check.Name, bounds.TextBytes) ||
				check.Status == "" || tooLong(check.Status, 128) || tooLong(check.Conclusion, 128) ||
				tooLong(check.URL, 2048) || !validAvailability(check.Availability) {
				return invalid(fmt.Sprintf("result.checks.%d", i), "check is incomplete, unbounded, or has invalid availability")
			}
		}
	case OperationGetReviews:
		var result ReviewsResult
		if err := decodeResult(raw, &result); err != nil {
			return err
		}
		if len(result.Reviews) > bounds.Items {
			return tooLarge("result.reviews", "review count exceeds the operation bound")
		}
		for i, review := range result.Reviews {
			if review.ProviderID == "" || tooLong(review.ProviderID, 512) || review.State == "" ||
				tooLong(review.State, 128) || tooLong(review.Body, bounds.BodyBytes) ||
				review.HeadSHA == "" || tooLong(review.HeadSHA, 128) || tooLong(review.SubmittedAt, 64) {
				return invalid(fmt.Sprintf("result.reviews.%d", i), "review is incomplete or unbounded")
			}
			if err := validateActor(review.Author, fmt.Sprintf("result.reviews.%d.author", i)); err != nil {
				return err
			}
		}
	case OperationGetComments:
		var result CommentsResult
		if err := decodeResult(raw, &result); err != nil {
			return err
		}
		if len(result.Comments) > bounds.Items {
			return tooLarge("result.comments", "comment count exceeds the operation bound")
		}
		for i, comment := range result.Comments {
			if comment.ProviderID == "" || tooLong(comment.ProviderID, 512) || comment.Body == "" ||
				tooLong(comment.Body, bounds.BodyBytes) || tooLong(comment.URL, 2048) ||
				tooLong(comment.CreatedAt, 64) || tooLong(comment.UpdatedAt, 64) {
				return invalid(fmt.Sprintf("result.comments.%d", i), "comment is incomplete or unbounded")
			}
			if err := validateActor(comment.Author, fmt.Sprintf("result.comments.%d.author", i)); err != nil {
				return err
			}
		}
	case OperationGetMergeReadiness:
		var result MergeReadiness
		if err := decodeResult(raw, &result); err != nil {
			return err
		}
		if result.State == "" || tooLong(result.State, 128) ||
			!validAvailability(result.Checks) || !validAvailability(result.Reviews) ||
			!validAvailability(result.BranchProtection) || !validAvailability(result.Permissions) ||
			!validAvailability(result.Mergeability) || !validAvailability(result.Queue) ||
			len(result.Reasons) > bounds.Items {
			return invalid("result", "merge readiness is incomplete, unbounded, or has invalid availability")
		}
		for i, reason := range result.Reasons {
			if reason == "" || tooLong(reason, bounds.TextBytes) {
				return tooLarge(fmt.Sprintf("result.reasons.%d", i), "merge-readiness reason is empty or unbounded")
			}
		}
	default:
		if !IsMutation(operation) {
			return &ContractError{Code: ErrorUnsupportedOperation, Message: "result operation is unsupported", Field: "operation", Retry: RetryNone}
		}
		var result MutationResult
		if err := decodeResult(raw, &result); err != nil {
			return err
		}
		if result.Outcome == "" || tooLong(result.Outcome, 128) {
			return invalid("result.outcome", "bounded mutation outcome is required")
		}
		return validatePullRequest(result.PullRequest, "result.pull_request")
	}
	return nil
}

func decodeResult(raw json.RawMessage, target any) *ContractError {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return invalid("result", "result does not match the operation schema")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return invalid("result", "result contains trailing JSON values")
	}
	return nil
}

func validatePullRequest(pullRequest PullRequest, field string) *ContractError {
	if err := ValidatePullRequestIdentity(pullRequest.Identity); err != nil {
		err.Field = field + "." + err.Field
		return err
	}
	if pullRequest.Title == "" || tooLong(pullRequest.Title, MaxTextBytes) || tooLong(pullRequest.Body, MaxBodyBytes) ||
		pullRequest.URL == "" || tooLong(pullRequest.URL, 2048) || pullRequest.State == "" || tooLong(pullRequest.State, 128) ||
		tooLong(pullRequest.CreatedAt, 64) || tooLong(pullRequest.UpdatedAt, 64) || tooLong(pullRequest.MergedAt, 64) {
		return invalid(field, "pull request is incomplete or unbounded")
	}
	if err := validateActor(pullRequest.Author, field+".author"); err != nil {
		return err
	}
	if err := ValidateRef(pullRequest.Base, field+".base"); err != nil {
		return err
	}
	return ValidateRef(pullRequest.Head, field+".head")
}

func validateActor(actor Actor, field string) *ContractError {
	if actor.Login == "" || tooLong(actor.Login, 255) || tooLong(actor.ProviderID, 512) || tooLong(actor.Display, MaxTextBytes) {
		return invalid(field, "actor is incomplete or unbounded")
	}
	return nil
}

func validateRequestCursor(request Request) *ContractError {
	envelope, err := DecodeCursor(request.Cursor)
	if err != nil {
		return err
	}
	material := envelope.Material
	scope := requestRepositoryScope(request)
	if material.Version != Version || material.Provider != request.Provider ||
		material.ConnectionID != request.ConnectionID ||
		material.Operation != request.Operation || material.Query != request.Query ||
		material.Order != request.Order || !reflect.DeepEqual(material.Repositories, scope) {
		return &ContractError{Code: ErrorCursorMismatch, Message: "cursor does not match request scope, operation, query, or ordering", Field: "cursor", Retry: RetryNone}
	}
	return nil
}

func validateResponseCursor(response Response, encoded string) *ContractError {
	envelope, err := DecodeCursor(encoded)
	if err != nil {
		return err
	}
	material := envelope.Material
	if material.Version != Version || material.Provider != response.Provider ||
		material.ConnectionID != response.ConnectionID || material.Operation != response.Operation ||
		!containsRepository(material.Repositories, response.Repository) {
		return &ContractError{Code: ErrorCursorMismatch, Message: "response cursor does not match operation identity", Field: "page.next_cursor", Retry: RetryNone}
	}
	return nil
}

func containsRepository(repositories []RepositoryIdentity, candidate RepositoryIdentity) bool {
	for _, repository := range repositories {
		if repository == candidate {
			return true
		}
	}
	return false
}

func requestRepositoryScope(request Request) []RepositoryIdentity {
	scope := make([]RepositoryIdentity, 0, len(request.Repositories)+1)
	scope = append(scope, request.Repository)
	for _, repository := range request.Repositories {
		scope = append(scope, repository)
	}
	return normalizedRepositoryScope(scope)
}

func normalizedRepositoryScope(scope []RepositoryIdentity) []RepositoryIdentity {
	seen := make(map[string]struct{}, len(scope))
	out := make([]RepositoryIdentity, 0, len(scope))
	for _, repository := range scope {
		key := repository.Host + "\x00" + repository.ProviderID + "\x00" + repository.FullName
		if repository.Host == "" || repository.FullName == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, repository)
	}
	sort.Slice(out, func(i, j int) bool {
		left := out[i].Host + "\x00" + out[i].ProviderID + "\x00" + out[i].FullName
		right := out[j].Host + "\x00" + out[j].ProviderID + "\x00" + out[j].FullName
		return left < right
	})
	return out
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
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return invalid("payload", "payload contains trailing JSON values")
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

func validAvailability(value Availability) bool {
	return containsString([]Availability{AvailabilityAvailable, AvailabilityPartial, AvailabilityUnavailable, AvailabilityUnknown}, value)
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
