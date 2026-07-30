package codehostbroker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

//go:generate go run ./cmd/generate

const redactedFixtureText = "[redacted]"

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
		Identity: identity, Title: redactedFixtureText, Body: redactedFixtureText, URL: "https://github.com/hero-engine/hero/pull/42",
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
			Version: Version, Operation: operation, Provider: "github", ConnectionID: identity.ConnectionID, Repository: repository,
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
		if operation == OperationGetPullRequest {
			response.Freshness = FreshnessStale
		}
		if operation == OperationListPullRequests || operation == OperationSearchPullRequests ||
			operation == OperationGetCommits || operation == OperationGetReviews || operation == OperationGetComments {
			cursor, err := EncodeCursor(CursorMaterial{
				Version: Version, Provider: "github", ConnectionID: identity.ConnectionID,
				Repositories: []RepositoryIdentity{repository}, Operation: operation,
				Query: request.Query, Order: request.Order, Position: "page-2",
			})
			if err != nil {
				return nil, err
			}
			response.Page = &Page{Limit: 25, Count: 1, NextCursor: cursor}
			if operation == OperationGetReviews {
				response.Completeness = CompletenessUnavailable
				response.Page.Count = 0
				response.Page.NextCursor = ""
			}
			if operation == OperationGetComments {
				response.Page.NextCursor = ""
			}
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
			response.JournalEntries = 1
		}
		cases = append(cases, FixtureCase{Name: string(operation), Request: request, Response: response})
	}

	errors := make([]ContractError, 0, len(errorCodes))
	for _, code := range errorCodes {
		errors = append(errors, ContractError{
			Code: code, Message: "bounded normalized fixture error", Retry: fixtureRetry(code),
		})
	}
	advertised := fixtureCapabilities()
	advertised = append(advertised, futureCapability())
	preparations := make([]PreparationResponse, 0, len(orderedOperations)-len(readOperations))
	for _, operation := range orderedOperations {
		if !IsMutation(operation) {
			continue
		}
		preparations = append(preparations, PreparationResponse{
			Version:             Version,
			Operation:           operation,
			CapabilityRevision:  "cap:fixture",
			ObservationRevision: "obs:fixture",
		})
	}
	bundle := ConsumerFixtureBundle{
		Version:      Version,
		Operations:   advertised,
		Cases:        cases,
		Preparations: preparations,
		Errors:       errors,
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
		payload = CreatePullRequestPayload{Base: base, Head: head, Title: redactedFixtureText, Body: redactedFixtureText}
	case OperationComment:
		payload = CommentPayload{ExpectedHeadSHA: head.SHA, Body: redactedFixtureText}
	case OperationSubmitReview, OperationApprove, OperationRequestChanges:
		payload = ReviewPayload{ExpectedHeadSHA: head.SHA, Body: redactedFixtureText}
	case OperationRetarget:
		payload = RetargetPayload{ExpectedHeadSHA: head.SHA, CurrentBase: base, NewBase: RefIdentity{Repository: base.Repository, Name: "release", SHA: base.SHA}}
	case OperationMarkReady, OperationClose, OperationReopen:
		payload = LifecyclePayload{ExpectedHeadSHA: head.SHA}
	case OperationMerge:
		payload = MergePayload{ExpectedHeadSHA: head.SHA, ObservedBase: base, Method: "squash", CommitTitle: redactedFixtureText, CommitMessage: redactedFixtureText}
	}
	data, _ := json.Marshal(payload)
	return data
}

func fixtureResult(operation Operation, pullRequest PullRequest) any {
	switch operation {
	case OperationCapabilities:
		return CapabilitiesResult{Capabilities: append(fixtureCapabilities(), futureCapability())}
	case OperationGetAuthenticatedActor:
		return AuthenticatedActorResult{Actor: pullRequest.Author}
	case OperationListPullRequests, OperationSearchPullRequests:
		return map[string]any{"pull_requests": []PullRequest{pullRequest}}
	case OperationGetPullRequest:
		return pullRequest
	case OperationGetCommits:
		return CommitsResult{Commits: []Commit{{SHA: pullRequest.Head.SHA, Message: redactedFixtureText, Author: pullRequest.Author}}}
	case OperationGetDiff:
		return map[string]any{"files": []DiffFile{
			{
				Path:                "text.go",
				Status:              "modified",
				Additions:           1,
				Deletions:           1,
				Hunks:               []DiffHunk{{Header: "@@ -1 +1 @@", Patch: "-old\n+new"}},
				ContentAvailability: DiffContentText,
			},
			{
				Path:                "provider-omitted.dat",
				Status:              "modified",
				Hunks:               []DiffHunk{},
				ContentAvailability: DiffContentProviderOmitted,
			},
			{
				Path:                "binary.dat",
				Status:              "modified",
				Hunks:               []DiffHunk{},
				ContentAvailability: DiffContentBinary,
			},
			{
				Path:                "generated.go",
				Status:              "modified",
				Hunks:               []DiffHunk{},
				ContentAvailability: DiffContentGenerated,
			},
			{
				Path:                "unknown.dat",
				Status:              "modified",
				Hunks:               []DiffHunk{},
				ContentAvailability: DiffContentUnknown,
			},
			{
				Path:                "truncated.go",
				Status:              "modified",
				Additions:           1,
				Hunks:               []DiffHunk{{Header: "@@ -1 +1 @@", Patch: "+partial"}},
				Truncated:           true,
				ContentAvailability: DiffContentTruncated,
			},
		}}
	case OperationGetChecks:
		return ChecksResult{Checks: []Check{
			{ProviderID: "check-1", Name: "available", Status: "completed", Conclusion: "success", Availability: AvailabilityAvailable},
			{ProviderID: "check-2", Name: "partial", Status: "in_progress", Availability: AvailabilityPartial},
			{ProviderID: "check-3", Name: "unavailable", Status: "unknown", Availability: AvailabilityUnavailable},
			{ProviderID: "check-4", Name: "unknown", Status: "unknown", Availability: AvailabilityUnknown},
		}}
	case OperationGetReviews:
		return ReviewsResult{Reviews: []Review{}}
	case OperationGetComments:
		return CommentsResult{Comments: []Comment{{ProviderID: "comment-1", Author: pullRequest.Author, Body: redactedFixtureText}}}
	case OperationGetMergeReadiness:
		return MergeReadiness{State: "unknown", Checks: AvailabilityAvailable, Reviews: AvailabilityPartial, BranchProtection: AvailabilityUnavailable, Permissions: AvailabilityAvailable, Mergeability: AvailabilityUnknown, Queue: AvailabilityUnavailable, Reasons: []string{redactedFixtureText}}
	case OperationComment, OperationSubmitReview, OperationApprove, OperationRequestChanges:
		actor := pullRequest.Author
		return MutationResult{PullRequest: pullRequest, Outcome: "fixture", Actor: &actor}
	case OperationMarkReady, OperationRetarget, OperationClose, OperationReopen:
		return MutationResult{
			PullRequest: pullRequest,
			Outcome:     "fixture",
			InvalidatedOperations: []Operation{
				OperationGetMergeReadiness,
			},
		}
	case OperationMerge:
		actor := pullRequest.Author
		pullRequest.State = "merged"
		pullRequest.MergedAt = "2026-07-27T20:31:00Z"
		return MutationResult{
			PullRequest: pullRequest,
			Outcome:     "fixture",
			Actor:       &actor,
			Merge: &MergeMutationResult{
				State:         "merged",
				MergeCommitID: "receipt-merge",
			},
			InvalidatedOperations: []Operation{
				OperationGetPullRequest,
				OperationGetMergeReadiness,
			},
		}
	default:
		return MutationResult{PullRequest: pullRequest, Outcome: "fixture"}
	}
}

func fixtureRetry(code string) RetryGuidance {
	return RetryForError(code)
}

func allOperationsAvailable() map[Operation]bool {
	out := make(map[Operation]bool, len(orderedOperations))
	for _, operation := range orderedOperations {
		out[operation] = true
	}
	return out
}

func fixtureCapabilities() []Capability {
	capabilities := Capabilities(allOperationsAvailable())
	for index := range capabilities {
		if capabilities[index].Policy.Operation == OperationMerge {
			capabilities[index].Merge = &MergeCapability{
				Methods:  []string{"merge", "squash", "rebase"},
				Revision: "capability:fixture-merge-runtime",
			}
		}
	}
	return capabilities
}

func futureCapability() Capability {
	return Capability{
		Policy: OperationPolicy{
			Operation: Operation("future_operation"),
			Effect:    Effect("future_effect"),
			Consent:   Consent("future_consent"),
			Bounds:    defaultBounds,
		},
		Available: true,
	}
}
