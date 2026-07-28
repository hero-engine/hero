package codehost

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/config"
)

type createPreflight struct {
	revision string
	existing []codehostbroker.PullRequest
}

type githubRepositoryPermission struct {
	Permissions struct {
		Pull bool `json:"pull"`
		Push bool `json:"push"`
	} `json:"permissions"`
}

type githubGitRef struct {
	Ref    string `json:"ref"`
	Object struct {
		SHA string `json:"sha"`
	} `json:"object"`
}

type githubCreateRequest struct {
	Title string `json:"title"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Body  string `json:"body,omitempty"`
	Draft bool   `json:"draft"`
}

// PrepareCreatePullRequest performs the non-mutating provider preflight and
// returns the same request with current capability and observation revisions.
// Callers still have to pass the prepared request to Execute; preparation
// never reserves an idempotency key and never dispatches a provider write.
func (b *Broker) PrepareCreatePullRequest(ctx context.Context, request codehostbroker.Request) (codehostbroker.Request, *codehostbroker.ContractError) {
	if request.Operation != codehostbroker.OperationCreatePullRequest {
		return request, contractError(codehostbroker.ErrorUnsupportedOperation, "preparation only supports create_pull_request", "operation")
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
	payload, payloadErr := decodeCreatePayload(request.Payload)
	if payloadErr != nil {
		return request, payloadErr
	}
	connection, adapter, scopeErr := b.resolveCreateAdapter(ctx, request, payload)
	if scopeErr != nil {
		return request, scopeErr
	}
	request.CapabilityRevision = capabilityRevision(connection, nil)
	preflight, err := adapter.observeCreatePreflight(ctx, payload)
	if err != nil {
		return request, normalizeProviderError(err)
	}
	request.ObservationRevision = preflight.revision
	return request, nil
}

func (b *Broker) resolveCreateAdapter(ctx context.Context, request codehostbroker.Request, payload createPayload) (config.CodeHostConnection, *githubAdapter, *codehostbroker.ContractError) {
	if err := ctx.Err(); err != nil {
		return config.CodeHostConnection{}, nil, cancelledError(err)
	}
	cfg, err := b.loadConfig(b.projectRoot)
	if err != nil {
		return config.CodeHostConnection{}, nil, contractError(codehostbroker.ErrorProviderUnavailable, "code-host configuration is unavailable", "")
	}
	connection, err := cfg.ResolveCodeHostConnection(request.ConnectionID)
	if err != nil {
		return config.CodeHostConnection{}, nil, resolutionError(err)
	}
	if request.Provider != connection.Provider {
		return config.CodeHostConnection{}, nil, contractError(codehostbroker.ErrorWrongConnectionCapability, "request provider does not match the selected connection", "provider")
	}
	if connection.Provider != "github" {
		return config.CodeHostConnection{}, nil, contractError(codehostbroker.ErrorUnsupportedProvider, "the selected code-host provider has no creation adapter", "provider")
	}
	if err := validateCreateScope(connection, request, payload); err != nil {
		return config.CodeHostConnection{}, nil, err
	}
	token, err := connection.ResolveToken()
	if err != nil {
		return config.CodeHostConnection{}, nil, resolutionError(err)
	}
	if len(token.Reveal()) > codehostbroker.MaxTextBytes {
		return config.CodeHostConnection{}, nil, contractError(codehostbroker.ErrorCredentialUnavailable, "the selected credential exceeds the safety bound", "")
	}
	transport, err := newGitHubTransport(connection, token, b.httpClient, b.now)
	if err != nil {
		return config.CodeHostConnection{}, nil, contractError(codehostbroker.ErrorInvalidInput, "the configured GitHub origin is invalid", "connection")
	}
	return connection, &githubAdapter{connection: connection, transport: transport, now: b.now}, nil
}

func validateCreateScope(connection config.CodeHostConnection, request codehostbroker.Request, payload createPayload) *codehostbroker.ContractError {
	if len(request.Repositories) != 0 {
		return contractError(codehostbroker.ErrorInvalidInput, "creation accepts one explicit base and head target, not a repository collection", "repositories")
	}
	if request.Repository != payload.Base.Repository {
		return contractError(codehostbroker.ErrorInvalidInput, "request repository must exactly match the creation base repository", "payload.base.repository")
	}
	scopeRequest := request
	scopeRequest.Repositories = []codehostbroker.RepositoryIdentity{payload.Head.Repository}
	if _, err := validateRepositoryScope(connection, scopeRequest); err != nil {
		return err
	}
	return nil
}

func (g *githubAdapter) observeCreatePreflight(ctx context.Context, payload createPayload) (createPreflight, error) {
	var permission githubRepositoryPermission
	repositoryPath := fmt.Sprintf("repos/%s/%s", url.PathEscape(payload.Base.Repository.Owner), url.PathEscape(payload.Base.Repository.Name))
	if _, err := g.transport.get(ctx, repositoryPath, &permission); err != nil {
		return createPreflight{}, err
	}
	if !permission.Permissions.Pull {
		return createPreflight{}, &providerError{code: codehostbroker.ErrorForbidden, message: "GitHub actor cannot create pull requests in the selected repository"}
	}
	base, err := g.resolveRef(ctx, payload.Base)
	if err != nil {
		return createPreflight{}, err
	}
	head, err := g.resolveRef(ctx, payload.Head)
	if err != nil {
		return createPreflight{}, err
	}
	if base.SHA != payload.Base.SHA {
		return createPreflight{}, &providerError{code: codehostbroker.ErrorStaleObservation, message: "base ref changed since creation intent was prepared"}
	}
	if head.SHA != payload.Head.SHA {
		return createPreflight{}, &providerError{code: codehostbroker.ErrorStaleObservation, message: "head ref changed since creation intent was prepared"}
	}
	existing, err := g.findExactPullRequests(ctx, payload)
	if err != nil {
		return createPreflight{}, err
	}
	material := struct {
		ConnectionID string
		Base         codehostbroker.RefIdentity
		Head         codehostbroker.RefIdentity
		CanCreate    bool
		Existing     []codehostbroker.PullRequestIdentity
	}{
		ConnectionID: g.connection.ID,
		Base:         base,
		Head:         head,
		CanCreate:    true,
		Existing:     make([]codehostbroker.PullRequestIdentity, 0, len(existing)),
	}
	for _, pullRequest := range existing {
		material.Existing = append(material.Existing, pullRequest.Identity)
	}
	return createPreflight{
		revision: fingerprint("observation", material),
		existing: existing,
	}, nil
}

func (g *githubAdapter) resolveRef(ctx context.Context, expected codehostbroker.RefIdentity) (codehostbroker.RefIdentity, error) {
	path := fmt.Sprintf("repos/%s/%s/git/ref/heads/%s",
		url.PathEscape(expected.Repository.Owner),
		url.PathEscape(expected.Repository.Name),
		url.PathEscape(expected.Name),
	)
	var wire githubGitRef
	if _, err := g.transport.get(ctx, path, &wire); err != nil {
		return codehostbroker.RefIdentity{}, err
	}
	if wire.Object.SHA == "" {
		return codehostbroker.RefIdentity{}, &providerError{code: codehostbroker.ErrorProvider, message: "GitHub returned an incomplete ref"}
	}
	return codehostbroker.RefIdentity{
		Repository: expected.Repository,
		Name:       expected.Name,
		SHA:        boundedText(wire.Object.SHA, 128),
	}, nil
}

func (g *githubAdapter) findExactPullRequests(ctx context.Context, payload createPayload) ([]codehostbroker.PullRequest, error) {
	query := url.Values{}
	query.Set("state", "open")
	query.Set("base", payload.Base.Name)
	query.Set("head", payload.Head.Repository.Owner+":"+payload.Head.Name)
	query.Set("per_page", "100")
	path := fmt.Sprintf("repos/%s/%s/pulls?%s",
		url.PathEscape(payload.Base.Repository.Owner),
		url.PathEscape(payload.Base.Repository.Name),
		query.Encode(),
	)
	var wire []githubPullRequest
	if _, err := g.transport.get(ctx, path, &wire); err != nil {
		return nil, err
	}
	matches := make([]codehostbroker.PullRequest, 0, len(wire))
	for _, item := range wire {
		pullRequest := normalizePullRequest(g.connection.ID, item)
		setPullRequestHost(&pullRequest, payload.Base.Repository.Host)
		if !sameCreateTarget(pullRequest, payload) {
			continue
		}
		if err := validateNormalizedPullRequest(pullRequest); err != nil {
			return nil, err
		}
		matches = append(matches, pullRequest)
	}
	return matches, nil
}

func (g *githubAdapter) createPullRequest(ctx context.Context, payload createPayload) (codehostbroker.PullRequest, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls",
		url.PathEscape(payload.Base.Repository.Owner),
		url.PathEscape(payload.Base.Repository.Name),
	)
	wireRequest := githubCreateRequest{
		Title: payload.Title,
		Head:  payload.Head.Repository.Owner + ":" + payload.Head.Name,
		Base:  payload.Base.Name,
		Body:  payload.Body,
		Draft: payload.Draft,
	}
	var wire githubPullRequest
	if _, err := g.transport.post(ctx, path, wireRequest, &wire); err != nil {
		return codehostbroker.PullRequest{}, err
	}
	pullRequest := normalizePullRequest(g.connection.ID, wire)
	setPullRequestHost(&pullRequest, payload.Base.Repository.Host)
	if err := validateNormalizedPullRequest(pullRequest); err != nil {
		return codehostbroker.PullRequest{}, &providerError{code: codehostbroker.ErrorProvider, message: "GitHub returned an incomplete creation receipt", ambiguous: true}
	}
	if !satisfiesCreate(pullRequest, payload) {
		return codehostbroker.PullRequest{}, &providerError{code: codehostbroker.ErrorProvider, message: "GitHub creation receipt did not match the requested target", ambiguous: true}
	}
	return pullRequest, nil
}

func (b *Broker) executeCreate(ctx context.Context, request codehostbroker.Request, payload createPayload, connection config.CodeHostConnection, adapter *githubAdapter) (adapterResult, error) {
	plan := mutationPlan{
		request:       request,
		payloadDigest: canonicalCreateDigest(request, payload),
		target: mutationTarget{
			ConnectionID: request.ConnectionID,
			Repository:   request.Repository,
			Base:         payload.Base,
			Head:         payload.Head,
		},
		connection: connection,
		adapter:    adapter,
	}
	plan.preflight = func(ctx context.Context) (mutationPreflight, error) {
		preflight, err := adapter.observeCreatePreflight(ctx, payload)
		if err != nil {
			return mutationPreflight{}, err
		}
		if len(preflight.existing) > 0 {
			if len(preflight.existing) != 1 || !satisfiesCreate(preflight.existing[0], payload) {
				return mutationPreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "an open pull request already uses the exact base and head target"}
			}
			pullRequest := preflight.existing[0]
			return mutationPreflight{
				observationRevision: preflight.revision,
				external: &mutationEffect{
					pullRequest: pullRequest,
					receipt:     safeJournalReceipt(pullRequest),
				},
			}, nil
		}
		return mutationPreflight{observationRevision: preflight.revision}, nil
	}
	plan.dispatch = func(ctx context.Context) (mutationEffect, error) {
		pullRequest, err := adapter.createPullRequest(ctx, payload)
		if err != nil {
			return mutationEffect{}, err
		}
		return mutationEffect{pullRequest: pullRequest, receipt: safeJournalReceipt(pullRequest)}, nil
	}
	plan.reconcile = func(ctx context.Context) (*mutationEffect, error) {
		matches, err := adapter.findExactPullRequests(ctx, payload)
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			return nil, nil
		}
		if len(matches) != 1 || !satisfiesCreate(matches[0], payload) {
			return nil, &providerError{code: codehostbroker.ErrorConflict, message: "creation target cannot be reconciled uniquely"}
		}
		pullRequest := matches[0]
		return &mutationEffect{pullRequest: pullRequest, receipt: safeJournalReceipt(pullRequest)}, nil
	}
	return b.executeMutation(ctx, plan)
}

func successfulMutationResult(request codehostbroker.Request, entry *journalEntry, document *journalDocument, pullRequest codehostbroker.PullRequest, status codehostbroker.ReconciliationStatus, outcome string) adapterResult {
	result := createMutationResult(pullRequest, outcome)
	if entry.Receipt != nil && entry.Receipt.Actor != nil {
		actor := *entry.Receipt.Actor
		result.Actor = &actor
	}
	return adapterResult{
		result:         result,
		pullRequest:    &pullRequest,
		completeness:   codehostbroker.CompletenessComplete,
		receipt:        createReceipt(entry, &pullRequest),
		reconciliation: &codehostbroker.Reconciliation{Status: status, Key: request.ReconciliationKey},
		journalEntries: len(document.Entries),
		observationRevision: fingerprint(
			"observation",
			pullRequest.Identity,
			pullRequest.Base,
			pullRequest.Head,
			pullRequest.State,
			pullRequest.Draft,
			pullRequest.MergedAt,
			pullRequest.UpdatedAt,
		),
	}
}

func mutationErrorResult(request codehostbroker.Request, entry *journalEntry, document *journalDocument, status codehostbroker.ReconciliationStatus) adapterResult {
	return adapterResult{
		completeness:        codehostbroker.CompletenessUnavailable,
		receipt:             createReceipt(entry, nil),
		reconciliation:      &codehostbroker.Reconciliation{Status: status, Key: request.ReconciliationKey},
		journalEntries:      len(document.Entries),
		observationRevision: fingerprint("observation", entry.Target, entry.State, entry.UpdatedAt),
	}
}

func withTransportMetadata(out adapterResult, adapter *githubAdapter) adapterResult {
	out.rateLimit = adapter.transport.rateLimit
	out.redirects = adapter.transport.redirects
	return out
}

func ambiguousProviderFailure(err error) bool {
	var provider *providerError
	return errors.As(err, &provider) && provider.ambiguous
}

func replayedCreateFailure(code string) error {
	switch code {
	case codehostbroker.ErrorUnauthorized:
		return &providerError{code: code, message: "GitHub rejected the recorded creation credential"}
	case codehostbroker.ErrorForbidden:
		return &providerError{code: code, message: "GitHub denied the recorded creation attempt"}
	case codehostbroker.ErrorConflict:
		return &providerError{code: code, message: "GitHub rejected the recorded creation target"}
	case codehostbroker.ErrorRateLimited:
		return &providerError{code: code, message: "GitHub rate limited the recorded creation attempt"}
	case codehostbroker.ErrorInvalidInput:
		return &providerError{code: code, message: "GitHub rejected the recorded creation input"}
	default:
		return &providerError{code: codehostbroker.ErrorProvider, message: "GitHub rejected the recorded creation attempt"}
	}
}
