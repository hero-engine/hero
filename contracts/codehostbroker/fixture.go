package codehostbroker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

//go:generate go run ./cmd/generate

func CanonicalFixture() ([]byte, error) {
	repository := RepositoryIdentity{
		Host: "github.com", ProviderID: "R_hero", Owner: "hero-engine", Name: "hero", FullName: "hero-engine/hero",
	}
	fork := RepositoryIdentity{
		Host: "github.com", ProviderID: "R_fork", Owner: "contributor", Name: "hero", FullName: "contributor/hero",
	}
	base := RefIdentity{Repository: repository, Name: "main", SHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	head := RefIdentity{Repository: fork, Name: "feature/code-host", SHA: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	identity := PullRequestIdentity{
		ConnectionID: "github-code-host", Repository: repository, ProviderID: "PR_kwDO_fixture", Number: 42,
	}
	pullRequest := PullRequest{
		Identity: identity, Title: "Add code-host broker", Body: "Bounded fixture body", URL: "https://github.com/hero-engine/hero/pull/42",
		State: "open", Draft: false, Author: Actor{ProviderID: "U_fixture", Login: "contributor"}, Base: base, Head: head,
		CreatedAt: "2026-07-27T20:00:00Z", UpdatedAt: "2026-07-27T20:30:00Z",
	}

	cases := make([]FixtureCase, 0, len(orderedOperations))
	reconciliations := []ReconciliationStatus{
		ReconciliationApplied,
		ReconciliationReplayed,
		ReconciliationReconciledApplied,
		ReconciliationExternallyCompleted,
		ReconciliationNotApplied,
		ReconciliationInProgress,
		ReconciliationAmbiguous,
	}
	mutationIndex := 0
	for _, operation := range orderedOperations {
		policy := operationPolicies[operation]
		request := Request{
			Version: Version, Operation: operation, ConnectionID: identity.ConnectionID, Repository: repository,
		}
		if requiresPullRequest(operation) {
			request.PullRequest = &identity
		}
		if operation == OperationListPullRequests || operation == OperationSearchPullRequests {
			request.Query = "state:open"
			request.Order = "updated_desc"
			request.Limit = 25
		}
		if IsMutation(operation) {
			request.IntentSource = "user"
			request.Consent = policy.Consent
			request.IdempotencyKey = "fixture-" + string(operation)
			request.CapabilityRevision = "cap:fixture"
			request.ObservationRevision = "obs:fixture"
			request.ReconciliationKey = "reconcile:" + string(operation)
			request.Payload = fixturePayload(operation, base, head)
		}

		result, _ := json.Marshal(fixtureResult(operation, pullRequest))
		response := Response{
			Version: Version, Operation: operation, Provider: "github", ConnectionID: identity.ConnectionID,
			Repository: repository, Policy: policy, CapabilityRevision: "cap:fixture", ObservationRevision: "obs:fixture",
			ObservedAt: "2026-07-27T20:31:00Z", Freshness: FreshnessCurrent,
			RateLimit: RateLimit{Resource: "core", Limit: 5000, Remaining: 4999, ResetAt: "2026-07-27T21:00:00Z", ObservedAt: "2026-07-27T20:31:00Z"},
			Bounds:    policy.Bounds, Completeness: CompletenessComplete, PartialFailures: []PartialFailure{},
			Result: result, Truncated: false, DurationMS: 42,
		}
		if collectionOperations[operation] || operation == OperationGetCommits || operation == OperationGetReviews || operation == OperationGetComments {
			response.Page = &Page{Limit: 25, Count: 1, NextCursor: "cursor:opaque-fixture"}
		}
		if operation == OperationGetChecks {
			response.Completeness = CompletenessPartial
			response.PartialFailures = []PartialFailure{{Section: "branch_protection", Code: ErrorForbidden, Message: "section unavailable"}}
		}
		if operation == OperationGetDiff {
			response.Completeness = CompletenessTruncated
			response.Truncated = true
		}
		if IsMutation(operation) {
			status := reconciliations[mutationIndex%len(reconciliations)]
			mutationIndex++
			response.Receipt = &Receipt{
				ProviderReceiptID: "receipt-" + string(operation), OperationID: "operation-" + string(operation), TargetRevision: head.SHA,
			}
			response.Reconciliation = &Reconciliation{Status: status, Key: request.ReconciliationKey}
		}
		cases = append(cases, FixtureCase{Name: string(operation), Request: request, Response: response})
	}

	errors := make([]ContractError, 0, len(errorCodes))
	for _, code := range errorCodes {
		errors = append(errors, ContractError{
			Code: code, Message: "bounded normalized fixture error", Retry: fixtureRetry(code),
		})
	}
	bundle := ConsumerFixtureBundle{
		Version:    Version,
		Operations: Capabilities(allOperationsAvailable()),
		Cases:      cases,
		Errors:     errors,
		UnknownFields: map[string]json.RawMessage{
			"future_capability": json.RawMessage(`{"operation":"future_operation","effect":"future_effect"}`),
			"future_field":      json.RawMessage(`{"nested":true}`),
		},
	}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func CanonicalFixtureSHA256() (string, error) {
	data, err := CanonicalFixture()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func fixturePayload(operation Operation, base, head RefIdentity) json.RawMessage {
	var payload any
	switch operation {
	case OperationCreatePullRequest:
		payload = CreatePullRequestPayload{Base: base, Head: head, Title: "Add code-host broker", Body: "Bounded fixture body"}
	case OperationComment:
		payload = CommentPayload{ExpectedHeadSHA: head.SHA, Body: "Fixture comment"}
	case OperationSubmitReview, OperationApprove, OperationRequestChanges:
		payload = ReviewPayload{ExpectedHeadSHA: head.SHA, Body: "Fixture review"}
	case OperationRetarget:
		payload = RetargetPayload{ExpectedHeadSHA: head.SHA, CurrentBase: base, NewBase: RefIdentity{Repository: base.Repository, Name: "release", SHA: base.SHA}}
	case OperationMarkReady, OperationClose, OperationReopen:
		payload = LifecyclePayload{ExpectedHeadSHA: head.SHA}
	case OperationMerge:
		payload = MergePayload{ExpectedHeadSHA: head.SHA, ObservedBase: base, Method: "squash", CommitTitle: "Merge fixture"}
	}
	data, _ := json.Marshal(payload)
	return data
}

func fixtureResult(operation Operation, pullRequest PullRequest) any {
	switch operation {
	case OperationCapabilities:
		return map[string]any{"capabilities": Capabilities(allOperationsAvailable())}
	case OperationListPullRequests, OperationSearchPullRequests:
		return map[string]any{"pull_requests": []PullRequest{pullRequest}}
	case OperationGetPullRequest:
		return pullRequest
	case OperationGetCommits:
		return map[string]any{"commits": []Commit{{SHA: pullRequest.Head.SHA, Message: "fixture", Author: pullRequest.Author}}}
	case OperationGetDiff:
		return map[string]any{"files": []DiffFile{{Path: "contracts/codehostbroker/contract.go", Status: "modified", Additions: 1, Hunks: []DiffHunk{{Header: "@@ -1 +1 @@", Patch: "+fixture"}}, Truncated: true}}}
	case OperationGetChecks:
		return map[string]any{"checks": []Check{{ProviderID: "check-1", Name: "test", Status: "completed", Conclusion: "success", Availability: AvailabilityAvailable}}}
	case OperationGetReviews:
		return map[string]any{"reviews": []Review{{ProviderID: "review-1", Author: pullRequest.Author, State: "approved", HeadSHA: pullRequest.Head.SHA}}}
	case OperationGetComments:
		return map[string]any{"comments": []Comment{{ProviderID: "comment-1", Author: pullRequest.Author, Body: "fixture comment"}}}
	case OperationGetMergeReadiness:
		return MergeReadiness{State: "ready", Checks: AvailabilityAvailable, Reviews: AvailabilityAvailable, BranchProtection: AvailabilityAvailable, Permissions: AvailabilityAvailable, Mergeability: AvailabilityAvailable, Queue: AvailabilityUnavailable, Reasons: []string{}}
	default:
		return map[string]any{"pull_request": pullRequest, "outcome": "fixture"}
	}
}

func fixtureRetry(code string) RetryGuidance {
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

func allOperationsAvailable() map[Operation]bool {
	out := make(map[Operation]bool, len(orderedOperations))
	for _, operation := range orderedOperations {
		out[operation] = true
	}
	return out
}
