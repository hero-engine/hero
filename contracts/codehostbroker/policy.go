package codehostbroker

const (
	MaxRepositoryScopes = 100
	MaxPageSize         = 100
	MaxItems            = 100
	MaxTextBytes        = 64 << 10
	MaxBodyBytes        = 256 << 10
	MaxDiffBytes        = 2 << 20
	MaxDiffFiles        = 300
	MaxDiffHunks        = 2000
	MaxPartialFailures  = 20
	MaxErrorDetailBytes = 4 << 10
	MaxDurationMS       = 120_000
	MaxRedirects        = 3
	MaxJournalEntries   = 10_000
	MaxIdempotencyBytes = 512
)

var defaultBounds = Bounds{
	RepositoryScopes: MaxRepositoryScopes,
	PageSize:         MaxPageSize,
	Items:            MaxItems,
	TextBytes:        MaxTextBytes,
	BodyBytes:        MaxBodyBytes,
	DiffBytes:        MaxDiffBytes,
	DiffFiles:        MaxDiffFiles,
	DiffHunks:        MaxDiffHunks,
	PartialFailures:  MaxPartialFailures,
	ErrorDetailBytes: MaxErrorDetailBytes,
	DurationMS:       MaxDurationMS,
	Redirects:        MaxRedirects,
	JournalEntries:   MaxJournalEntries,
	IdempotencyBytes: MaxIdempotencyBytes,
}

var orderedOperations = []Operation{
	OperationCapabilities,
	OperationGetAuthenticatedActor,
	OperationListPullRequests,
	OperationSearchPullRequests,
	OperationGetPullRequest,
	OperationGetCommits,
	OperationGetDiff,
	OperationGetChecks,
	OperationGetReviews,
	OperationGetComments,
	OperationGetMergeReadiness,
	OperationCreatePullRequest,
	OperationComment,
	OperationSubmitReview,
	OperationApprove,
	OperationRequestChanges,
	OperationMarkReady,
	OperationRetarget,
	OperationClose,
	OperationReopen,
	OperationMerge,
}

var readOperations = map[Operation]bool{
	OperationCapabilities:          true,
	OperationGetAuthenticatedActor: true,
	OperationListPullRequests:      true,
	OperationSearchPullRequests:    true,
	OperationGetPullRequest:        true,
	OperationGetCommits:            true,
	OperationGetDiff:               true,
	OperationGetChecks:             true,
	OperationGetReviews:            true,
	OperationGetComments:           true,
	OperationGetMergeReadiness:     true,
}

var collectionOperations = map[Operation]bool{
	OperationCapabilities:          true,
	OperationGetAuthenticatedActor: true,
	OperationListPullRequests:      true,
	OperationSearchPullRequests:    true,
}

func Operations() []Operation {
	return append([]Operation(nil), orderedOperations...)
}

func IsOperation(operation Operation) bool {
	_, ok := operationPolicies[operation]
	return ok
}

func IsRead(operation Operation) bool {
	return readOperations[operation]
}

func IsMutation(operation Operation) bool {
	return IsOperation(operation) && !IsRead(operation)
}

func Policy(operation Operation) (OperationPolicy, bool) {
	policy, ok := operationPolicies[operation]
	return policy, ok
}

func Policies() []OperationPolicy {
	out := make([]OperationPolicy, 0, len(orderedOperations))
	for _, operation := range orderedOperations {
		out = append(out, operationPolicies[operation])
	}
	return out
}

func Capabilities(available map[Operation]bool) []Capability {
	out := make([]Capability, 0, len(orderedOperations))
	for _, operation := range orderedOperations {
		capability := Capability{Policy: operationPolicies[operation], Available: available[operation]}
		if !capability.Available {
			capability.Reason = "adapter does not implement this operation"
		}
		out = append(out, capability)
	}
	return out
}

var operationPolicies = func() map[Operation]OperationPolicy {
	policies := make(map[Operation]OperationPolicy, len(orderedOperations))
	for _, operation := range orderedOperations {
		policies[operation] = OperationPolicy{
			Operation:                operation,
			Effect:                   EffectRead,
			Consent:                  ConsentNone,
			RequiresUniqueTarget:     !collectionOperations[operation],
			RequiresIdempotency:      false,
			RequiresFreshObservation: false,
			RequiresReconciliation:   false,
			ReplaySafe:               true,
			Bounds:                   defaultBounds,
		}
	}
	for _, operation := range orderedOperations {
		if readOperations[operation] {
			continue
		}
		policy := policies[operation]
		policy.Effect = EffectExternalWrite
		policy.Consent = ConsentExplicitUser
		policy.RequiresUniqueTarget = true
		policy.RequiresIdempotency = true
		policy.RequiresFreshObservation = true
		policy.RequiresReconciliation = true
		policy.ReplaySafe = true
		policies[operation] = policy
	}
	merge := policies[OperationMerge]
	merge.Effect = EffectCommitment
	merge.Consent = ConsentExplicitAcceptance
	policies[OperationMerge] = merge
	return policies
}()

const (
	ErrorInvalidInput              = "invalid_input"
	ErrorIncompatibleVersion       = "incompatible_version"
	ErrorConnectionNotFound        = "connection_not_found"
	ErrorCodeHostRoleMissing       = "code_host_role_missing"
	ErrorWrongConnectionCapability = "wrong_connection_capability"
	ErrorCredentialUnavailable     = "credential_unavailable"
	ErrorUnauthorized              = "unauthorized"
	ErrorForbidden                 = "forbidden"
	ErrorUnsupportedProvider       = "unsupported_provider"
	ErrorUnsupportedOperation      = "unsupported_operation"
	ErrorNotFound                  = "not_found"
	ErrorStaleObservation          = "stale_observation"
	ErrorCapabilityChanged         = "capability_changed"
	ErrorConflict                  = "conflict"
	ErrorRateLimited               = "rate_limited"
	ErrorCursorMismatch            = "cursor_mismatch"
	ErrorIdempotencyConflict       = "idempotency_conflict"
	ErrorOperationInProgress       = "operation_in_progress"
	ErrorAmbiguousResult           = "ambiguous_result"
	ErrorProviderUnavailable       = "provider_unavailable"
	ErrorProvider                  = "provider_error"
	ErrorPartialFailure            = "partial_failure"
	ErrorInputTooLarge             = "input_too_large"
	ErrorOutputTooLarge            = "output_too_large"
	ErrorCancelled                 = "cancelled"
	ErrorEncoding                  = "encoding_error"
)

var errorCodes = []string{
	ErrorInvalidInput,
	ErrorIncompatibleVersion,
	ErrorConnectionNotFound,
	ErrorCodeHostRoleMissing,
	ErrorWrongConnectionCapability,
	ErrorCredentialUnavailable,
	ErrorUnauthorized,
	ErrorForbidden,
	ErrorUnsupportedProvider,
	ErrorUnsupportedOperation,
	ErrorNotFound,
	ErrorStaleObservation,
	ErrorCapabilityChanged,
	ErrorConflict,
	ErrorRateLimited,
	ErrorCursorMismatch,
	ErrorIdempotencyConflict,
	ErrorOperationInProgress,
	ErrorAmbiguousResult,
	ErrorProviderUnavailable,
	ErrorProvider,
	ErrorPartialFailure,
	ErrorInputTooLarge,
	ErrorOutputTooLarge,
	ErrorCancelled,
	ErrorEncoding,
}

func ErrorCodes() []string {
	return append([]string(nil), errorCodes...)
}

func RetryGuidanceValues() []RetryGuidance {
	return []RetryGuidance{RetryNone, RetrySameKey, RetryRefreshThenRetry, RetryAfter, RetryReconcile}
}
