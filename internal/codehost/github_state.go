package codehost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/config"
)

const markReadyMutation = `mutation HeroMarkPullRequestReady($pullRequestId: ID!) {
  markPullRequestReadyForReview(input: {pullRequestId: $pullRequestId}) {
    pullRequest { id }
  }
}`

type stateTransitionPayload struct {
	expectedHeadSHA string
	currentBase     *codehostbroker.RefIdentity
	newBase         *codehostbroker.RefIdentity
}

type stateTransitionPreflight struct {
	pullRequest codehostbroker.PullRequest
	actor       codehostbroker.Actor
	revision    string
	external    *mutationEffect
}

type githubPullRequestUpdate struct {
	State string `json:"state,omitempty"`
	Base  string `json:"base,omitempty"`
}

type githubMarkReadyResponse struct {
	Data struct {
		MarkPullRequestReadyForReview struct {
			PullRequest struct {
				ID string `json:"id"`
			} `json:"pullRequest"`
		} `json:"markPullRequestReadyForReview"`
	} `json:"data"`
	Errors []githubGraphQLError `json:"errors"`
}

func isStateTransitionOperation(operation codehostbroker.Operation) bool {
	switch operation {
	case codehostbroker.OperationMarkReady,
		codehostbroker.OperationRetarget,
		codehostbroker.OperationClose,
		codehostbroker.OperationReopen:
		return true
	default:
		return false
	}
}

func decodeStateTransitionPayload(request codehostbroker.Request) (stateTransitionPayload, *codehostbroker.ContractError) {
	switch request.Operation {
	case codehostbroker.OperationRetarget:
		var payload codehostbroker.RetargetPayload
		if err := decodeStateTransitionJSON(request.Payload, &payload); err != nil {
			return stateTransitionPayload{}, err
		}
		return stateTransitionPayload{
			expectedHeadSHA: payload.ExpectedHeadSHA,
			currentBase:     &payload.CurrentBase,
			newBase:         &payload.NewBase,
		}, nil
	case codehostbroker.OperationMarkReady, codehostbroker.OperationClose, codehostbroker.OperationReopen:
		var payload codehostbroker.LifecyclePayload
		if err := decodeStateTransitionJSON(request.Payload, &payload); err != nil {
			return stateTransitionPayload{}, err
		}
		return stateTransitionPayload{expectedHeadSHA: payload.ExpectedHeadSHA}, nil
	default:
		return stateTransitionPayload{}, contractError(codehostbroker.ErrorUnsupportedOperation, "operation is not a pull-request state transition", "operation")
	}
}

func decodeStateTransitionJSON(raw json.RawMessage, target any) *codehostbroker.ContractError {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return contractError(codehostbroker.ErrorInvalidInput, "state-transition payload does not match the v1 schema", "payload")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return contractError(codehostbroker.ErrorInvalidInput, "state-transition payload contains trailing data", "payload")
	}
	return nil
}

func validateStateTransitionScope(connection config.CodeHostConnection, request codehostbroker.Request, payload stateTransitionPayload) *codehostbroker.ContractError {
	if len(request.Repositories) != 0 {
		return contractError(codehostbroker.ErrorInvalidInput, "state transitions accept one repository-qualified pull request", "repositories")
	}
	if _, scopeErr := validateRepositoryScope(connection, request); scopeErr != nil {
		return scopeErr
	}
	if request.Operation != codehostbroker.OperationRetarget {
		return nil
	}
	if payload.currentBase == nil || payload.newBase == nil ||
		*payload.currentBase == (codehostbroker.RefIdentity{}) ||
		*payload.newBase == (codehostbroker.RefIdentity{}) {
		return contractError(codehostbroker.ErrorInvalidInput, "retarget requires current and desired base refs", "payload")
	}
	if payload.currentBase.Repository != request.Repository || payload.newBase.Repository != request.Repository {
		return contractError(codehostbroker.ErrorInvalidInput, "retarget bases must exactly match the pull-request repository", "payload.new_base.repository")
	}
	return nil
}

// PrepareStateTransition performs bounded provider reads and fills the current
// capability and lifecycle observation revisions without reserving a key.
func (b *Broker) PrepareStateTransition(ctx context.Context, request codehostbroker.Request) (codehostbroker.Request, *codehostbroker.ContractError) {
	if !isStateTransitionOperation(request.Operation) {
		return request, contractError(codehostbroker.ErrorUnsupportedOperation, "preparation only supports pull-request state transitions", "operation")
	}
	if request.CapabilityRevision == "" {
		request.CapabilityRevision = "prepare"
	}
	if request.ObservationRevision == "" {
		request.ObservationRevision = "prepare"
	}
	if err := codehostbroker.ValidateRequest(request); err != nil {
		return request, err
	}
	payload, payloadErr := decodeStateTransitionPayload(request)
	if payloadErr != nil {
		return request, payloadErr
	}
	connection, adapter, resolveErr := b.resolveCollaborationAdapter(ctx, request)
	if resolveErr != nil {
		return request, resolveErr
	}
	if scopeErr := validateStateTransitionScope(connection, request, payload); scopeErr != nil {
		return request, scopeErr
	}
	request.CapabilityRevision = capabilityRevision(connection, nil)
	preflight, err := adapter.observeStateTransitionPreflight(ctx, request, payload)
	if err != nil {
		return request, normalizeProviderError(err)
	}
	request.ObservationRevision = preflight.revision
	return request, nil
}

func (g *githubAdapter) observeStateTransitionPreflight(ctx context.Context, request codehostbroker.Request, payload stateTransitionPayload) (stateTransitionPreflight, error) {
	pullRequest, err := g.fetchPullRequest(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return stateTransitionPreflight{}, err
	}
	if pullRequest.Identity.ProviderID != request.PullRequest.ProviderID {
		return stateTransitionPreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "pull request provider identity changed"}
	}
	lifecycle := normalizedLifecycleState(pullRequest)
	if lifecycle == "merged" {
		return stateTransitionPreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "merged pull requests are terminal"}
	}
	if pullRequest.Head.SHA != payload.expectedHeadSHA {
		return stateTransitionPreflight{}, &providerError{code: codehostbroker.ErrorStaleObservation, message: "pull request head changed before state transition"}
	}
	permission, err := g.stateTransitionPermission(ctx, request)
	if err != nil {
		return stateTransitionPreflight{}, err
	}
	actor, err := g.authenticatedActor(ctx)
	if err != nil {
		return stateTransitionPreflight{}, err
	}

	var target *codehostbroker.RefIdentity
	if request.Operation == codehostbroker.OperationRetarget {
		if lifecycle != "open_draft" && lifecycle != "open_ready" {
			return stateTransitionPreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "only open pull requests can be retargeted"}
		}
		if !sameRefIdentity(pullRequest.Base, *payload.currentBase) {
			return stateTransitionPreflight{}, &providerError{code: codehostbroker.ErrorStaleObservation, message: "pull request base changed before retarget"}
		}
		resolved, resolveErr := g.resolveRef(ctx, *payload.newBase)
		if resolveErr != nil {
			return stateTransitionPreflight{}, resolveErr
		}
		if resolved.SHA != payload.newBase.SHA {
			return stateTransitionPreflight{}, &providerError{code: codehostbroker.ErrorStaleObservation, message: "retarget branch changed before dispatch"}
		}
		target = &resolved
	}
	switch request.Operation {
	case codehostbroker.OperationMarkReady:
		if lifecycle != "open_draft" && lifecycle != "open_ready" {
			return stateTransitionPreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "only open draft pull requests can be marked ready"}
		}
	case codehostbroker.OperationClose:
		if lifecycle != "open_draft" && lifecycle != "open_ready" && lifecycle != "closed_unmerged" {
			return stateTransitionPreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "pull request cannot be closed from its current lifecycle state"}
		}
	case codehostbroker.OperationReopen:
		if lifecycle != "closed_unmerged" && lifecycle != "open_draft" && lifecycle != "open_ready" {
			return stateTransitionPreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "pull request cannot be reopened from its current lifecycle state"}
		}
	}

	var external *mutationEffect
	if stateTransitionDesired(request.Operation, pullRequest, target) {
		external = &mutationEffect{
			pullRequest: pullRequest,
			receipt:     safeStateTransitionReceipt(pullRequest, actor),
		}
	}
	revision := stateTransitionObservationRevision(request.Operation, pullRequest, target, actor, permission)
	return stateTransitionPreflight{
		pullRequest: pullRequest,
		actor:       actor,
		revision:    revision,
		external:    external,
	}, nil
}

func (g *githubAdapter) stateTransitionPermission(ctx context.Context, request codehostbroker.Request) (string, error) {
	var permission githubRepositoryPermission
	path := fmt.Sprintf("repos/%s/%s", url.PathEscape(request.Repository.Owner), url.PathEscape(request.Repository.Name))
	if _, err := g.transport.get(ctx, path, &permission); err != nil {
		return "", err
	}
	if !permission.Permissions.Pull || !permission.Permissions.Push {
		return "", &providerError{code: codehostbroker.ErrorForbidden, message: "GitHub actor cannot change pull-request lifecycle state"}
	}
	return "push", nil
}

func (g *githubAdapter) dispatchStateTransition(ctx context.Context, request codehostbroker.Request, payload stateTransitionPayload, preflight stateTransitionPreflight) (mutationEffect, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls/%d",
		url.PathEscape(request.Repository.Owner),
		url.PathEscape(request.Repository.Name),
		request.PullRequest.Number,
	)
	switch request.Operation {
	case codehostbroker.OperationMarkReady:
		var wire githubMarkReadyResponse
		if _, err := g.transport.graphql(ctx, markReadyMutation, map[string]any{
			"pullRequestId": request.PullRequest.ProviderID,
		}, &wire); err != nil {
			return mutationEffect{}, err
		}
		if len(wire.Errors) > 0 ||
			wire.Data.MarkPullRequestReadyForReview.PullRequest.ID != request.PullRequest.ProviderID {
			return mutationEffect{}, &providerError{code: codehostbroker.ErrorProvider, message: "GitHub returned an invalid mark-ready receipt", ambiguous: true}
		}
	case codehostbroker.OperationRetarget:
		var wire githubPullRequest
		if _, err := g.transport.patch(ctx, path, githubPullRequestUpdate{Base: payload.newBase.Name}, &wire); err != nil {
			return mutationEffect{}, err
		}
	case codehostbroker.OperationClose:
		var wire githubPullRequest
		if _, err := g.transport.patch(ctx, path, githubPullRequestUpdate{State: "closed"}, &wire); err != nil {
			return mutationEffect{}, err
		}
	case codehostbroker.OperationReopen:
		var wire githubPullRequest
		if _, err := g.transport.patch(ctx, path, githubPullRequestUpdate{State: "open"}, &wire); err != nil {
			return mutationEffect{}, err
		}
	}
	pullRequest, err := g.fetchPullRequest(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return mutationEffect{}, &providerError{code: codehostbroker.ErrorProviderUnavailable, message: "GitHub post-transition state is unavailable", ambiguous: true}
	}
	if pullRequest.Head.SHA != payload.expectedHeadSHA ||
		normalizedLifecycleState(pullRequest) == "merged" ||
		!stateTransitionDesired(request.Operation, pullRequest, payload.newBase) {
		return mutationEffect{}, &providerError{code: codehostbroker.ErrorConflict, message: "GitHub post-transition state does not match the requested effect", ambiguous: true}
	}
	return mutationEffect{
		pullRequest: pullRequest,
		receipt:     safeStateTransitionReceipt(pullRequest, preflight.actor),
	}, nil
}

func (g *githubAdapter) reconcileStateTransition(ctx context.Context, request codehostbroker.Request, payload stateTransitionPayload, actor codehostbroker.Actor) (*mutationEffect, error) {
	pullRequest, err := g.fetchPullRequest(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return nil, err
	}
	if pullRequest.Identity.ProviderID != request.PullRequest.ProviderID {
		return nil, &providerError{code: codehostbroker.ErrorConflict, message: "pull request provider identity changed during reconciliation"}
	}
	if pullRequest.Head.SHA != payload.expectedHeadSHA {
		return nil, &providerError{code: codehostbroker.ErrorStaleObservation, message: "pull request head changed during state-transition reconciliation"}
	}
	if normalizedLifecycleState(pullRequest) == "merged" {
		return nil, &providerError{code: codehostbroker.ErrorConflict, message: "pull request merged during state-transition reconciliation"}
	}
	if !stateTransitionDesired(request.Operation, pullRequest, payload.newBase) {
		return nil, nil
	}
	return &mutationEffect{
		pullRequest: pullRequest,
		receipt:     safeStateTransitionReceipt(pullRequest, actor),
	}, nil
}

func (b *Broker) executeStateTransition(ctx context.Context, request codehostbroker.Request, payload stateTransitionPayload, connection config.CodeHostConnection, adapter *githubAdapter) (adapterResult, error) {
	var observed stateTransitionPreflight
	plan := mutationPlan{
		request:       request,
		payloadDigest: canonicalStateTransitionDigest(request, payload),
		target: mutationTarget{
			ConnectionID:    request.ConnectionID,
			Repository:      request.Repository,
			PullRequest:     request.PullRequest,
			ExpectedHeadSHA: payload.expectedHeadSHA,
			CurrentBase:     payload.currentBase,
			DesiredBase:     payload.newBase,
		},
		connection: connection,
		adapter:    adapter,
	}
	plan.preflight = func(ctx context.Context) (mutationPreflight, error) {
		var err error
		observed, err = adapter.observeStateTransitionPreflight(ctx, request, payload)
		if err != nil {
			return mutationPreflight{}, err
		}
		return mutationPreflight{observationRevision: observed.revision, external: observed.external}, nil
	}
	plan.dispatch = func(ctx context.Context) (mutationEffect, error) {
		return adapter.dispatchStateTransition(ctx, request, payload, observed)
	}
	plan.reconcile = func(ctx context.Context) (*mutationEffect, error) {
		actor := observed.actor
		if actor.Login == "" {
			var err error
			actor, err = adapter.authenticatedActor(ctx)
			if err != nil {
				return nil, err
			}
		}
		return adapter.reconcileStateTransition(ctx, request, payload, actor)
	}
	return b.executeMutation(ctx, plan)
}

func normalizedLifecycleState(pullRequest codehostbroker.PullRequest) string {
	if pullRequest.State == "merged" || pullRequest.MergedAt != "" {
		return "merged"
	}
	switch pullRequest.State {
	case "closed":
		return "closed_unmerged"
	case "open":
		if pullRequest.Draft {
			return "open_draft"
		}
		return "open_ready"
	default:
		return "unknown"
	}
}

func stateTransitionDesired(operation codehostbroker.Operation, pullRequest codehostbroker.PullRequest, desiredBase *codehostbroker.RefIdentity) bool {
	lifecycle := normalizedLifecycleState(pullRequest)
	switch operation {
	case codehostbroker.OperationMarkReady:
		return lifecycle == "open_ready"
	case codehostbroker.OperationRetarget:
		return (lifecycle == "open_draft" || lifecycle == "open_ready") &&
			desiredBase != nil &&
			equalRepository(pullRequest.Base.Repository, desiredBase.Repository) &&
			pullRequest.Base.Name == desiredBase.Name &&
			pullRequest.Base.SHA == desiredBase.SHA
	case codehostbroker.OperationClose:
		return lifecycle == "closed_unmerged"
	case codehostbroker.OperationReopen:
		return lifecycle == "open_draft" || lifecycle == "open_ready"
	default:
		return false
	}
}

func sameRefIdentity(left, right codehostbroker.RefIdentity) bool {
	return equalRepository(left.Repository, right.Repository) &&
		left.Name == right.Name &&
		left.SHA == right.SHA
}

func stateTransitionObservationRevision(operation codehostbroker.Operation, pullRequest codehostbroker.PullRequest, target *codehostbroker.RefIdentity, actor codehostbroker.Actor, permission string) string {
	return fingerprint("observation", struct {
		Operation   codehostbroker.Operation
		PullRequest codehostbroker.PullRequestIdentity
		Lifecycle   string
		Draft       bool
		Base        codehostbroker.RefIdentity
		Head        codehostbroker.RefIdentity
		Target      *codehostbroker.RefIdentity
		Actor       codehostbroker.Actor
		Permission  string
	}{
		Operation: operation, PullRequest: pullRequest.Identity,
		Lifecycle: normalizedLifecycleState(pullRequest), Draft: pullRequest.Draft,
		Base: pullRequest.Base, Head: pullRequest.Head, Target: target,
		Actor: actor, Permission: permission,
	})
}

func canonicalStateTransitionDigest(request codehostbroker.Request, payload stateTransitionPayload) string {
	return fingerprint("payload", struct {
		Version           string
		Operation         codehostbroker.Operation
		Provider          string
		ConnectionID      string
		PullRequest       codehostbroker.PullRequestIdentity
		ExpectedHeadSHA   string
		CurrentBase       *canonicalRefTarget
		DesiredBase       *canonicalRefTarget
		IntentSource      string
		Consent           codehostbroker.Consent
		ReconciliationKey string
	}{
		Version:           request.Version,
		Operation:         request.Operation,
		Provider:          strings.ToLower(request.Provider),
		ConnectionID:      request.ConnectionID,
		PullRequest:       *request.PullRequest,
		ExpectedHeadSHA:   strings.ToLower(payload.expectedHeadSHA),
		CurrentBase:       canonicalOptionalRef(payload.currentBase),
		DesiredBase:       canonicalOptionalRef(payload.newBase),
		IntentSource:      request.IntentSource,
		Consent:           request.Consent,
		ReconciliationKey: request.ReconciliationKey,
	})
}

func canonicalOptionalRef(ref *codehostbroker.RefIdentity) *canonicalRefTarget {
	if ref == nil {
		return nil
	}
	value := canonicalRef(*ref)
	return &value
}

func safeStateTransitionReceipt(pullRequest codehostbroker.PullRequest, actor codehostbroker.Actor) *journalReceipt {
	actor.Display = ""
	return &journalReceipt{
		Identity:          pullRequest.Identity,
		Base:              pullRequest.Base,
		Head:              pullRequest.Head,
		ProviderReceiptID: pullRequest.Identity.ProviderID,
		Actor:             &actor,
		HeadSHA:           pullRequest.Head.SHA,
		State:             normalizedLifecycleState(pullRequest),
		Draft:             pullRequest.Draft,
	}
}
