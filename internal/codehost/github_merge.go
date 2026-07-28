package codehost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strings"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/config"
)

type mergePayload struct {
	codehostbroker.MergePayload
}

type mergeRuntime struct {
	material   codehostbroker.MergeCapability
	available  bool
	reason     string
	revision   string
	permission string
}

type mergePreflight struct {
	pullRequest codehostbroker.PullRequest
	actor       codehostbroker.Actor
	revision    string
	external    *mutationEffect
}

type githubMergeRequest struct {
	CommitTitle   string `json:"commit_title,omitempty"`
	CommitMessage string `json:"commit_message,omitempty"`
	MergeMethod   string `json:"merge_method"`
	SHA           string `json:"sha"`
}

type githubMergeResponse struct {
	SHA     string `json:"sha"`
	Merged  bool   `json:"merged"`
	Message string `json:"message"`
}

const mergeCapabilityQuery = `query HeroMergeCapability($owner: String!, $name: String!, $branch: String!) {
  repository(owner: $owner, name: $name) {
    id
    mergeQueue(branch: $branch) { id }
  }
  rateLimit { limit remaining resetAt }
}`

type githubMergeCapabilityResponse struct {
	Data struct {
		Repository struct {
			ID         string `json:"id"`
			MergeQueue *struct {
				ID string `json:"id"`
			} `json:"mergeQueue"`
		} `json:"repository"`
		RateLimit githubGraphQLRateLimit `json:"rateLimit"`
	} `json:"data"`
	Errors []githubGraphQLError `json:"errors"`
}

func decodeMergePayload(raw json.RawMessage) (mergePayload, *codehostbroker.ContractError) {
	var payload codehostbroker.MergePayload
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return mergePayload{}, contractError(codehostbroker.ErrorInvalidInput, "merge payload does not match the v1 schema", "payload")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return mergePayload{}, contractError(codehostbroker.ErrorInvalidInput, "merge payload contains trailing data", "payload")
	}
	if payload.Method == "rebase" && (payload.CommitTitle != "" || payload.CommitMessage != "") {
		return mergePayload{}, contractError(codehostbroker.ErrorInvalidInput, "rebase merge does not accept commit title or message overrides", "payload")
	}
	return mergePayload{MergePayload: payload}, nil
}

func validateMergeScope(connection config.CodeHostConnection, request codehostbroker.Request, payload mergePayload) *codehostbroker.ContractError {
	if len(request.Repositories) != 0 {
		return contractError(codehostbroker.ErrorInvalidInput, "merge accepts one repository-qualified pull request", "repositories")
	}
	if _, err := validateRepositoryScope(connection, request); err != nil {
		return err
	}
	if !equalRepository(payload.ObservedBase.Repository, request.Repository) {
		return contractError(codehostbroker.ErrorInvalidInput, "observed merge base must match the pull-request repository", "payload.observed_base.repository")
	}
	return nil
}

// PrepareMerge reads the runtime repository policy and exact merge-readiness
// proof, then stamps the request with revisions consumed by Execute.
func (b *Broker) PrepareMerge(ctx context.Context, request codehostbroker.Request) (codehostbroker.Request, *codehostbroker.ContractError) {
	if request.Operation != codehostbroker.OperationMerge {
		return request, contractError(codehostbroker.ErrorUnsupportedOperation, "preparation only supports merge", "operation")
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
	payload, payloadErr := decodeMergePayload(request.Payload)
	if payloadErr != nil {
		return request, payloadErr
	}
	connection, adapter, resolveErr := b.resolveCollaborationAdapter(ctx, request)
	if resolveErr != nil {
		return request, resolveErr
	}
	if scopeErr := validateMergeScope(connection, request, payload); scopeErr != nil {
		return request, scopeErr
	}
	runtime, err := adapter.mergeRuntimeCapability(ctx, request.Repository, payload.ObservedBase.Name)
	if err != nil {
		return request, normalizeProviderError(err)
	}
	if err := requireMergeCapability(runtime, payload.Method); err != nil {
		return request, normalizeProviderError(err)
	}
	request.CapabilityRevision = runtime.revision
	preflight, err := adapter.observeMergePreflight(ctx, request, payload, runtime)
	if err != nil {
		return request, normalizeProviderError(err)
	}
	request.ObservationRevision = preflight.revision
	return request, nil
}

func (g *githubAdapter) mergeRuntimeCapability(ctx context.Context, repository codehostbroker.RepositoryIdentity, baseBranch string) (mergeRuntime, error) {
	var permission githubRepositoryPermission
	path := fmt.Sprintf("repos/%s/%s", url.PathEscape(repository.Owner), url.PathEscape(repository.Name))
	if _, err := g.transport.get(ctx, path, &permission); err != nil {
		return mergeRuntime{}, err
	}
	if baseBranch == "" {
		baseBranch = permission.DefaultBranch
	}
	if baseBranch == "" {
		return mergeRuntime{}, &providerError{code: codehostbroker.ErrorProviderUnavailable, message: "GitHub did not provide a branch for merge-queue policy"}
	}
	var queue githubMergeCapabilityResponse
	if _, err := g.transport.graphql(ctx, mergeCapabilityQuery, map[string]any{
		"owner": repository.Owner, "name": repository.Name, "branch": baseBranch,
	}, &queue); err != nil {
		return mergeRuntime{}, err
	}
	g.transport.observeGraphQLRateLimit(queue.Data.RateLimit)
	if len(queue.Errors) != 0 || queue.Data.Repository.ID == "" ||
		(queue.Data.Repository.MergeQueue != nil && queue.Data.Repository.MergeQueue.ID == "") {
		return mergeRuntime{}, &providerError{code: codehostbroker.ErrorPartialFailure, message: "GitHub merge-queue policy is unavailable"}
	}
	queueRequired := queue.Data.Repository.MergeQueue != nil
	methods := make([]string, 0, 3)
	if permission.AllowMergeCommit {
		methods = append(methods, "merge")
	}
	if permission.AllowSquashMerge {
		methods = append(methods, "squash")
	}
	if permission.AllowRebaseMerge {
		methods = append(methods, "rebase")
	}
	runtime := mergeRuntime{
		material: codehostbroker.MergeCapability{
			Methods:        methods,
			QueueSupported: false,
			QueueRequired:  queueRequired,
		},
		available:  true,
		permission: "pull_push",
	}
	switch {
	case !permission.Permissions.Pull || !permission.Permissions.Push:
		runtime.available = false
		runtime.permission = "denied"
		runtime.reason = "the selected GitHub credential cannot merge pull requests"
	case queueRequired:
		runtime.available = false
		runtime.reason = "the repository requires a merge queue that this adapter cannot reconcile"
	case len(methods) == 0:
		runtime.available = false
		runtime.reason = "the repository advertises no supported direct merge method"
	}
	if !runtime.available {
		runtime.material.Methods = []string{}
	}
	runtime.revision = fingerprint("capability", struct {
		Connection     string
		Repository     canonicalRepositoryTarget
		Methods        []string
		QueueSupported bool
		QueueRequired  bool
		Permission     string
	}{
		Connection:     g.connection.ID,
		Repository:     canonicalRepository(repository),
		Methods:        runtime.material.Methods,
		QueueSupported: runtime.material.QueueSupported,
		QueueRequired:  runtime.material.QueueRequired,
		Permission:     runtime.permission,
	})
	runtime.material.Revision = runtime.revision
	return runtime, nil
}

func requireMergeCapability(runtime mergeRuntime, method string) error {
	if !runtime.available {
		if runtime.permission == "denied" {
			return &providerError{code: codehostbroker.ErrorForbidden, message: runtime.reason}
		}
		return &providerError{code: codehostbroker.ErrorUnsupportedOperation, message: runtime.reason}
	}
	if !slices.Contains(runtime.material.Methods, method) {
		return &providerError{code: codehostbroker.ErrorUnsupportedOperation, message: "requested merge method is not supported by this repository"}
	}
	if runtime.material.QueueRequired || runtime.material.QueueSupported {
		return &providerError{code: codehostbroker.ErrorUnsupportedOperation, message: "merge queue execution is not available"}
	}
	return nil
}

func (g *githubAdapter) observeMergePreflight(ctx context.Context, request codehostbroker.Request, payload mergePayload, runtime mergeRuntime) (mergePreflight, error) {
	if err := requireMergeCapability(runtime, payload.Method); err != nil {
		return mergePreflight{}, err
	}
	pullRequest, wire, err := g.fetchPullRequestWithProviderState(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return mergePreflight{}, err
	}
	if pullRequest.Identity.ProviderID != request.PullRequest.ProviderID {
		return mergePreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "pull request provider identity changed"}
	}
	if normalizedLifecycleState(pullRequest) == "merged" {
		if pullRequest.Head.SHA != payload.ExpectedHeadSHA || !sameRefIdentity(pullRequest.Base, payload.ObservedBase) {
			return mergePreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "pull request was merged at a different head or base"}
		}
		if wire.MergeCommitSHA == "" {
			return mergePreflight{}, &providerError{code: codehostbroker.ErrorProviderUnavailable, message: "GitHub did not provide an exact merge receipt"}
		}
		effect := mutationEffect{pullRequest: pullRequest, receipt: safeMergeReceipt(pullRequest, wire.MergeCommitSHA, nil)}
		return mergePreflight{
			pullRequest: pullRequest,
			revision:    mergeObservationRevision(request, payload, runtime, pullRequest, codehostbroker.MergeReadiness{}, codehostbroker.Actor{}),
			external:    &effect,
		}, nil
	}
	if normalizedLifecycleState(pullRequest) != "open_ready" {
		return mergePreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "only open, ready pull requests can be merged"}
	}
	if pullRequest.Head.SHA != payload.ExpectedHeadSHA {
		return mergePreflight{}, &providerError{code: codehostbroker.ErrorStaleObservation, message: "pull request head changed before merge"}
	}
	if !sameRefIdentity(pullRequest.Base, payload.ObservedBase) {
		return mergePreflight{}, &providerError{code: codehostbroker.ErrorStaleObservation, message: "pull request base changed before merge"}
	}
	actor, err := g.authenticatedActor(ctx)
	if err != nil {
		return mergePreflight{}, err
	}
	readinessResult, err := g.getMergeReadiness(ctx, request)
	if err != nil {
		return mergePreflight{}, err
	}
	readiness, ok := readinessResult.result.(codehostbroker.MergeReadiness)
	if !ok ||
		readinessResult.completeness != codehostbroker.CompletenessComplete ||
		readinessResult.truncated ||
		len(readinessResult.partialFailures) != 0 {
		return mergePreflight{}, &providerError{code: codehostbroker.ErrorPartialFailure, message: "complete merge-readiness evidence is unavailable"}
	}
	if readinessResult.pullRequest == nil ||
		readinessResult.pullRequest.Identity != pullRequest.Identity ||
		readinessResult.pullRequest.Head.SHA != payload.ExpectedHeadSHA ||
		!sameRefIdentity(readinessResult.pullRequest.Base, payload.ObservedBase) ||
		readinessResult.observedHeadSHA != payload.ExpectedHeadSHA ||
		readinessResult.observedBaseSHA != payload.ObservedBase.SHA {
		return mergePreflight{}, &providerError{code: codehostbroker.ErrorStaleObservation, message: "merge-readiness evidence does not match the exact pull request revision"}
	}
	if readinessResult.queueID != "" {
		return mergePreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "pull request is already in a merge queue"}
	}
	if readinessResult.queueRequired {
		return mergePreflight{}, &providerError{code: codehostbroker.ErrorCapabilityChanged, message: "the base branch now requires a merge queue"}
	}
	if readiness.State != "ready" ||
		readiness.Checks != codehostbroker.AvailabilityAvailable ||
		readiness.Reviews != codehostbroker.AvailabilityAvailable ||
		readiness.BranchProtection != codehostbroker.AvailabilityAvailable ||
		readiness.Permissions != codehostbroker.AvailabilityAvailable ||
		readiness.Mergeability != codehostbroker.AvailabilityAvailable ||
		readiness.Queue != codehostbroker.AvailabilityAvailable ||
		len(readiness.Reasons) != 0 {
		return mergePreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "pull request is not provably ready to merge"}
	}
	return mergePreflight{
		pullRequest: pullRequest,
		actor:       actor,
		revision:    mergeObservationRevision(request, payload, runtime, pullRequest, readiness, actor),
	}, nil
}

func (g *githubAdapter) dispatchMerge(ctx context.Context, request codehostbroker.Request, payload mergePayload, preflight mergePreflight) (mutationEffect, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/merge",
		url.PathEscape(request.Repository.Owner), url.PathEscape(request.Repository.Name), request.PullRequest.Number)
	var response githubMergeResponse
	if _, err := g.transport.put(ctx, path, githubMergeRequest{
		CommitTitle: payload.CommitTitle, CommitMessage: payload.CommitMessage,
		MergeMethod: payload.Method, SHA: payload.ExpectedHeadSHA,
	}, &response); err != nil {
		return mutationEffect{}, err
	}
	if !response.Merged || response.SHA == "" {
		return mutationEffect{}, &providerError{code: codehostbroker.ErrorConflict, message: "GitHub did not accept the exact merge"}
	}
	pullRequest, wire, err := g.fetchPullRequestWithProviderState(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return mutationEffect{}, &providerError{code: codehostbroker.ErrorProviderUnavailable, message: "GitHub post-merge state is unavailable", ambiguous: true}
	}
	if normalizedLifecycleState(pullRequest) != "merged" ||
		pullRequest.Head.SHA != payload.ExpectedHeadSHA ||
		!sameRefIdentity(pullRequest.Base, payload.ObservedBase) ||
		wire.MergeCommitSHA == "" ||
		wire.MergeCommitSHA != response.SHA {
		return mutationEffect{}, &providerError{code: codehostbroker.ErrorConflict, message: "GitHub post-merge state does not match the requested effect", ambiguous: true}
	}
	return mutationEffect{
		pullRequest: pullRequest,
		receipt:     safeMergeReceipt(pullRequest, response.SHA, &preflight.actor),
	}, nil
}

func (g *githubAdapter) reconcileMerge(ctx context.Context, request codehostbroker.Request, payload mergePayload, actor *codehostbroker.Actor) (*mutationEffect, error) {
	pullRequest, wire, err := g.fetchPullRequestWithProviderState(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return nil, err
	}
	if pullRequest.Identity.ProviderID != request.PullRequest.ProviderID {
		return nil, &providerError{code: codehostbroker.ErrorConflict, message: "pull request provider identity changed during merge reconciliation"}
	}
	if normalizedLifecycleState(pullRequest) != "merged" {
		return nil, nil
	}
	if pullRequest.Head.SHA != payload.ExpectedHeadSHA || !sameRefIdentity(pullRequest.Base, payload.ObservedBase) {
		return nil, &providerError{code: codehostbroker.ErrorConflict, message: "pull request merged at a different head or base"}
	}
	if wire.MergeCommitSHA == "" {
		return nil, &providerError{code: codehostbroker.ErrorProviderUnavailable, message: "GitHub merge receipt is unavailable"}
	}
	return &mutationEffect{
		pullRequest: pullRequest,
		receipt:     safeMergeReceipt(pullRequest, wire.MergeCommitSHA, actor),
	}, nil
}

func (b *Broker) executeMerge(ctx context.Context, request codehostbroker.Request, payload mergePayload, connection config.CodeHostConnection, adapter *githubAdapter) (adapterResult, error) {
	var observed mergePreflight
	var runtime mergeRuntime
	plan := mutationPlan{
		request:       request,
		payloadDigest: canonicalMergeDigest(request, payload),
		target:        mutationTarget{ConnectionID: request.ConnectionID, Repository: request.Repository, PullRequest: request.PullRequest, ExpectedHeadSHA: payload.ExpectedHeadSHA, CurrentBase: &payload.ObservedBase},
		connection:    connection,
		adapter:       adapter,
		resolveCapability: func(ctx context.Context) (string, error) {
			var err error
			runtime, err = adapter.mergeRuntimeCapability(ctx, request.Repository, payload.ObservedBase.Name)
			if err != nil {
				return "", err
			}
			return runtime.revision, requireMergeCapability(runtime, payload.Method)
		},
		decisiveReconcileError: func(err error) bool {
			code := normalizeProviderError(err).Code
			return code == codehostbroker.ErrorConflict || code == codehostbroker.ErrorStaleObservation
		},
	}
	plan.preflight = func(ctx context.Context) (mutationPreflight, error) {
		var err error
		observed, err = adapter.observeMergePreflight(ctx, request, payload, runtime)
		if err != nil {
			return mutationPreflight{}, err
		}
		return mutationPreflight{observationRevision: observed.revision, external: observed.external}, nil
	}
	plan.dispatch = func(ctx context.Context) (mutationEffect, error) {
		return adapter.dispatchMerge(ctx, request, payload, observed)
	}
	plan.reconcile = func(ctx context.Context) (*mutationEffect, error) {
		var actor *codehostbroker.Actor
		if observed.actor.Login != "" {
			value := observed.actor
			actor = &value
		}
		return adapter.reconcileMerge(ctx, request, payload, actor)
	}
	return b.executeMutation(ctx, plan)
}

func mergeObservationRevision(request codehostbroker.Request, payload mergePayload, runtime mergeRuntime, pullRequest codehostbroker.PullRequest, readiness codehostbroker.MergeReadiness, actor codehostbroker.Actor) string {
	return fingerprint("observation", struct {
		PullRequest codehostbroker.PullRequestIdentity
		Head        codehostbroker.RefIdentity
		Base        codehostbroker.RefIdentity
		Lifecycle   string
		Method      string
		Readiness   codehostbroker.MergeReadiness
		Actor       codehostbroker.Actor
		Capability  string
	}{
		PullRequest: pullRequest.Identity, Head: pullRequest.Head, Base: pullRequest.Base,
		Lifecycle: normalizedLifecycleState(pullRequest), Method: payload.Method,
		Readiness: readiness, Actor: actor, Capability: runtime.revision,
	})
}

func canonicalMergeDigest(request codehostbroker.Request, payload mergePayload) string {
	return fingerprint("payload", struct {
		Version           string
		Operation         codehostbroker.Operation
		Provider          string
		ConnectionID      string
		PullRequest       codehostbroker.PullRequestIdentity
		Head              string
		Base              canonicalRefTarget
		Method            string
		CommitTitle       string
		CommitMessage     string
		IntentSource      string
		Consent           codehostbroker.Consent
		ReconciliationKey string
	}{
		Version: request.Version, Operation: request.Operation, Provider: strings.ToLower(request.Provider),
		ConnectionID: request.ConnectionID, PullRequest: *request.PullRequest,
		Head: strings.ToLower(payload.ExpectedHeadSHA), Base: canonicalRef(payload.ObservedBase), Method: payload.Method,
		CommitTitle: payload.CommitTitle, CommitMessage: payload.CommitMessage,
		IntentSource: request.IntentSource, Consent: request.Consent, ReconciliationKey: request.ReconciliationKey,
	})
}

func safeMergeReceipt(pullRequest codehostbroker.PullRequest, mergeCommitSHA string, actor *codehostbroker.Actor) *journalReceipt {
	var safeActor *codehostbroker.Actor
	if actor != nil {
		value := *actor
		value.Display = ""
		safeActor = &value
	}
	return &journalReceipt{
		Identity: pullRequest.Identity, Base: pullRequest.Base, Head: pullRequest.Head,
		ProviderReceiptID: mergeCommitSHA, Actor: safeActor,
		HeadSHA: pullRequest.Head.SHA, State: "merged",
	}
}
