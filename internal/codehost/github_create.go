package codehost

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

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
	journal := newMutationJournal(b.projectRoot, b.now)
	key := journalKeyDigest(request)
	digest := canonicalCreateDigest(request, payload)
	var out adapterResult
	var executionErr error
	err := journal.withLock(func(document *journalDocument) error {
		entry := document.Entries[key]
		if entry != nil && entry.PayloadDigest != digest {
			out = mutationErrorResult(request, entry, document, codehostbroker.ReconciliationNotApplied)
			executionErr = &providerError{code: codehostbroker.ErrorIdempotencyConflict, message: "idempotency key was already used for a different creation payload"}
			return nil
		}
		if entry == nil {
			if len(document.Entries) >= journal.maxEntries {
				executionErr = &providerError{code: codehostbroker.ErrorProviderUnavailable, message: "mutation journal is full of unresolved safety records"}
				return nil
			}
			timestamp := journal.timestamp()
			entry = &journalEntry{
				KeyDigest:     key,
				PayloadDigest: digest,
				OperationID:   operationID(request),
				Target: mutationTarget{
					ConnectionID: request.ConnectionID,
					Repository:   request.Repository,
					Base:         payload.Base,
					Head:         payload.Head,
				},
				State:     journalInProgress,
				CreatedAt: timestamp,
				UpdatedAt: timestamp,
			}
			document.Entries[key] = entry
			if saveErr := journal.save(document); saveErr != nil {
				return saveErr
			}
		} else {
			replayed, replayErr, decisive := b.reconcileExistingCreate(ctx, request, payload, adapter, entry, document)
			if decisive {
				out = replayed
				executionErr = replayErr
				return nil
			}
			if saveErr := journal.save(document); saveErr != nil {
				return saveErr
			}
		}

		out.receipt = createReceipt(entry, nil)
		out.reconciliation = &codehostbroker.Reconciliation{Status: codehostbroker.ReconciliationInProgress, Key: request.ReconciliationKey}
		out.journalEntries = len(document.Entries)
		expectedCapability := capabilityRevision(connection, nil)
		out.capabilityRevision = expectedCapability
		if request.CapabilityRevision != expectedCapability {
			entry.State = journalNotApplied
			entry.UpdatedAt = journal.timestamp()
			out.reconciliation.Status = codehostbroker.ReconciliationNotApplied
			executionErr = &providerError{code: codehostbroker.ErrorCapabilityChanged, message: "code-host capability revision changed before creation"}
			return nil
		}
		if err := ctx.Err(); err != nil {
			entry.State = journalNotApplied
			entry.UpdatedAt = journal.timestamp()
			out.reconciliation.Status = codehostbroker.ReconciliationNotApplied
			executionErr = err
			return nil
		}
		preflight, preflightErr := adapter.observeCreatePreflight(ctx, payload)
		out.rateLimit = adapter.transport.rateLimit
		out.redirects = adapter.transport.redirects
		if preflightErr != nil {
			entry.State = journalNotApplied
			entry.UpdatedAt = journal.timestamp()
			out.reconciliation.Status = codehostbroker.ReconciliationNotApplied
			executionErr = preflightErr
			return nil
		}
		out.observationRevision = preflight.revision
		if request.ObservationRevision != preflight.revision {
			entry.State = journalNotApplied
			entry.UpdatedAt = journal.timestamp()
			out.reconciliation.Status = codehostbroker.ReconciliationNotApplied
			executionErr = &providerError{code: codehostbroker.ErrorStaleObservation, message: "creation preflight observation is stale"}
			return nil
		}
		if len(preflight.existing) > 0 {
			if len(preflight.existing) == 1 && satisfiesCreate(preflight.existing[0], payload) {
				pullRequest := preflight.existing[0]
				entry.State = journalExternal
				entry.Receipt = safeJournalReceipt(pullRequest)
				entry.UpdatedAt = journal.timestamp()
				entry.ReconciledAt = entry.UpdatedAt
				out = successfulMutationResult(request, entry, document, pullRequest, codehostbroker.ReconciliationExternallyCompleted, "externally_completed")
				out = withTransportMetadata(out, adapter)
				return nil
			}
			entry.State = journalNotApplied
			entry.UpdatedAt = journal.timestamp()
			out.reconciliation.Status = codehostbroker.ReconciliationNotApplied
			executionErr = &providerError{code: codehostbroker.ErrorConflict, message: "an open pull request already uses the exact base and head target"}
			return nil
		}

		entry.State = journalDispatched
		entry.ProviderAttempts++
		entry.UpdatedAt = journal.timestamp()
		if saveErr := journal.save(document); saveErr != nil {
			return saveErr
		}
		pullRequest, createErr := adapter.createPullRequest(ctx, payload)
		out.rateLimit = adapter.transport.rateLimit
		out.redirects = adapter.transport.redirects
		if createErr == nil {
			entry.State = journalApplied
			entry.Receipt = safeJournalReceipt(pullRequest)
			entry.UpdatedAt = journal.timestamp()
			out = successfulMutationResult(request, entry, document, pullRequest, codehostbroker.ReconciliationApplied, "applied")
			out = withTransportMetadata(out, adapter)
			return nil
		}
		if !ambiguousProviderFailure(createErr) {
			entry.State = journalNotApplied
			entry.UpdatedAt = journal.timestamp()
			entry.FailureCode = normalizeProviderError(createErr).Code
			out = mutationErrorResult(request, entry, document, codehostbroker.ReconciliationNotApplied)
			out = withTransportMetadata(out, adapter)
			executionErr = createErr
			return nil
		}

		reconcileContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		matches, reconcileErr := adapter.findExactPullRequests(reconcileContext, payload)
		entry.ReconciledAt = journal.timestamp()
		if reconcileErr == nil && len(matches) == 1 && satisfiesCreate(matches[0], payload) {
			pullRequest = matches[0]
			entry.State = journalApplied
			entry.Receipt = safeJournalReceipt(pullRequest)
			entry.UpdatedAt = journal.timestamp()
			out = successfulMutationResult(request, entry, document, pullRequest, codehostbroker.ReconciliationReconciledApplied, "reconciled_applied")
			out = withTransportMetadata(out, adapter)
			return nil
		}
		entry.State = journalAmbiguous
		entry.UpdatedAt = journal.timestamp()
		out = mutationErrorResult(request, entry, document, codehostbroker.ReconciliationAmbiguous)
		out = withTransportMetadata(out, adapter)
		out.observationRevision = preflight.revision
		executionErr = &providerError{code: codehostbroker.ErrorAmbiguousResult, message: "GitHub creation outcome is ambiguous"}
		return nil
	})
	if err != nil {
		return adapterResult{}, &providerError{code: codehostbroker.ErrorProviderUnavailable, message: "mutation journal is unavailable"}
	}
	return out, executionErr
}

func (b *Broker) reconcileExistingCreate(ctx context.Context, request codehostbroker.Request, payload createPayload, adapter *githubAdapter, entry *journalEntry, document *journalDocument) (adapterResult, error, bool) {
	reconcileContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	matches, err := adapter.findExactPullRequests(reconcileContext, payload)
	entry.ReconciledAt = b.now().UTC().Format(time.RFC3339Nano)
	if err == nil && len(matches) == 1 && satisfiesCreate(matches[0], payload) {
		pullRequest := matches[0]
		status := codehostbroker.ReconciliationReplayed
		outcome := "replayed"
		if entry.State == journalDispatched || entry.State == journalAmbiguous {
			status = codehostbroker.ReconciliationReconciledApplied
			outcome = "reconciled_applied"
		}
		entry.State = journalApplied
		entry.Receipt = safeJournalReceipt(pullRequest)
		entry.UpdatedAt = entry.ReconciledAt
		return withTransportMetadata(successfulMutationResult(request, entry, document, pullRequest, status, outcome), adapter), nil, true
	}
	switch entry.State {
	case journalApplied, journalExternal, journalDispatched, journalAmbiguous:
		entry.State = journalAmbiguous
		entry.UpdatedAt = entry.ReconciledAt
		out := mutationErrorResult(request, entry, document, codehostbroker.ReconciliationAmbiguous)
		return withTransportMetadata(out, adapter), &providerError{code: codehostbroker.ErrorAmbiguousResult, message: "recorded GitHub creation cannot be reconciled safely"}, true
	case journalInProgress:
		if err != nil || len(matches) > 0 {
			entry.State = journalAmbiguous
			entry.UpdatedAt = entry.ReconciledAt
			out := mutationErrorResult(request, entry, document, codehostbroker.ReconciliationAmbiguous)
			return withTransportMetadata(out, adapter), &providerError{code: codehostbroker.ErrorAmbiguousResult, message: "interrupted GitHub creation cannot be reconciled safely"}, true
		}
		entry.State = journalNotApplied
		entry.UpdatedAt = entry.ReconciledAt
		return adapterResult{}, nil, false
	case journalNotApplied:
		if entry.ProviderAttempts > 0 {
			out := mutationErrorResult(request, entry, document, codehostbroker.ReconciliationNotApplied)
			return withTransportMetadata(out, adapter), replayedCreateFailure(entry.FailureCode), true
		}
		return adapterResult{}, nil, false
	default:
		entry.State = journalAmbiguous
		entry.UpdatedAt = entry.ReconciledAt
		out := mutationErrorResult(request, entry, document, codehostbroker.ReconciliationAmbiguous)
		return withTransportMetadata(out, adapter), &providerError{code: codehostbroker.ErrorAmbiguousResult, message: "mutation journal state cannot be reconciled safely"}, true
	}
}

func successfulMutationResult(request codehostbroker.Request, entry *journalEntry, document *journalDocument, pullRequest codehostbroker.PullRequest, status codehostbroker.ReconciliationStatus, outcome string) adapterResult {
	return adapterResult{
		result:              createMutationResult(pullRequest, outcome),
		pullRequest:         &pullRequest,
		completeness:        codehostbroker.CompletenessComplete,
		receipt:             createReceipt(entry, &pullRequest),
		reconciliation:      &codehostbroker.Reconciliation{Status: status, Key: request.ReconciliationKey},
		journalEntries:      len(document.Entries),
		observationRevision: fingerprint("observation", pullRequest.Identity, pullRequest.Base, pullRequest.Head, pullRequest.State, pullRequest.UpdatedAt),
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
