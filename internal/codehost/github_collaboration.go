package codehost

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/config"
)

type collaborationPreflight struct {
	pullRequest codehostbroker.PullRequest
	actor       codehostbroker.Actor
	revision    string
	external    *mutationEffect
}

type githubAuthenticatedUser githubUser

type githubIssueCommentRequest struct {
	Body string `json:"body"`
}

type githubReviewRequest struct {
	Body     string `json:"body"`
	Event    string `json:"event"`
	CommitID string `json:"commit_id"`
}

// PrepareCollaboration performs the non-mutating PR/head/actor/permission
// preflight and fills current capability and observation revisions.
func (b *Broker) PrepareCollaboration(ctx context.Context, request codehostbroker.Request) (codehostbroker.Request, *codehostbroker.ContractError) {
	if !isCollaborationOperation(request.Operation) {
		return request, contractError(codehostbroker.ErrorUnsupportedOperation, "preparation only supports collaboration mutations", "operation")
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
	payload, payloadErr := decodeCollaborationPayload(request)
	if payloadErr != nil {
		return request, payloadErr
	}
	connection, adapter, resolveErr := b.resolveCollaborationAdapter(ctx, request)
	if resolveErr != nil {
		return request, resolveErr
	}
	request.CapabilityRevision = capabilityRevision(connection, nil)
	preflight, err := adapter.observeCollaborationPreflight(ctx, request, payload, collaborationMarker(request))
	if err != nil {
		return request, normalizeProviderError(err)
	}
	request.ObservationRevision = preflight.revision
	return request, nil
}

func (b *Broker) resolveCollaborationAdapter(ctx context.Context, request codehostbroker.Request) (config.CodeHostConnection, *githubAdapter, *codehostbroker.ContractError) {
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
		return config.CodeHostConnection{}, nil, contractError(codehostbroker.ErrorUnsupportedProvider, "the selected code-host provider has no collaboration adapter", "provider")
	}
	if scopeErr := validateCollaborationScope(connection, request); scopeErr != nil {
		return config.CodeHostConnection{}, nil, scopeErr
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

func validateCollaborationScope(connection config.CodeHostConnection, request codehostbroker.Request) *codehostbroker.ContractError {
	if len(request.Repositories) != 0 {
		return contractError(codehostbroker.ErrorInvalidInput, "collaboration accepts one repository-qualified pull request", "repositories")
	}
	if _, scopeErr := validateRepositoryScope(connection, request); scopeErr != nil {
		return scopeErr
	}
	return nil
}

func (g *githubAdapter) observeCollaborationPreflight(ctx context.Context, request codehostbroker.Request, payload collaborationPayload, marker string) (collaborationPreflight, error) {
	pullRequest, err := g.fetchPullRequest(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return collaborationPreflight{}, err
	}
	if pullRequest.Identity.ProviderID != request.PullRequest.ProviderID {
		return collaborationPreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "pull request provider identity changed"}
	}
	if pullRequest.State != "open" {
		return collaborationPreflight{}, &providerError{code: codehostbroker.ErrorConflict, message: "collaboration requires an open pull request"}
	}
	if pullRequest.Head.SHA != payload.expectedHeadSHA {
		return collaborationPreflight{}, &providerError{code: codehostbroker.ErrorStaleObservation, message: "pull request head changed before collaboration"}
	}
	permission, err := g.collaborationPermission(ctx, request)
	if err != nil {
		return collaborationPreflight{}, err
	}
	actor, err := g.authenticatedActor(ctx)
	if err != nil {
		return collaborationPreflight{}, err
	}

	var reviews []githubReview
	var external *mutationEffect
	if request.Operation == codehostbroker.OperationApprove || request.Operation == codehostbroker.OperationRequestChanges {
		reviews, err = g.rawReviews(ctx, request)
		if err != nil {
			return collaborationPreflight{}, err
		}
		if review := authoritativeCurrentHeadReview(reviews, actor, payload.expectedHeadSHA); review != nil &&
			strings.EqualFold(review.State, collaborationReviewState(request.Operation)) {
			external = &mutationEffect{
				pullRequest: pullRequest,
				receipt: safeCollaborationReceipt(
					pullRequest.Identity,
					providerID(review.Node, review.ID),
					actor,
					payload.expectedHeadSHA,
				),
			}
		}
	}
	reviewMaterial := make([]struct {
		ProviderID string
		Actor      string
		HeadSHA    string
		State      string
	}, 0, len(reviews))
	for _, review := range reviews {
		reviewMaterial = append(reviewMaterial, struct {
			ProviderID string
			Actor      string
			HeadSHA    string
			State      string
		}{
			ProviderID: providerID(review.Node, review.ID),
			Actor:      providerID(review.User.Node, review.User.ID) + ":" + review.User.Login,
			HeadSHA:    review.CommitID,
			State:      strings.ToUpper(review.State),
		})
	}
	revision := fingerprint("observation", struct {
		Operation   codehostbroker.Operation
		PullRequest codehostbroker.PullRequestIdentity
		State       string
		HeadSHA     string
		Actor       codehostbroker.Actor
		Permission  string
		Reviews     any
		Marker      string
	}{
		Operation: request.Operation, PullRequest: pullRequest.Identity, State: pullRequest.State,
		HeadSHA: pullRequest.Head.SHA, Actor: actor, Permission: permission,
		Reviews: reviewMaterial, Marker: marker,
	})
	return collaborationPreflight{
		pullRequest: pullRequest,
		actor:       actor,
		revision:    revision,
		external:    external,
	}, nil
}

func (g *githubAdapter) collaborationPermission(ctx context.Context, request codehostbroker.Request) (string, error) {
	var permission githubRepositoryPermission
	path := fmt.Sprintf("repos/%s/%s", url.PathEscape(request.Repository.Owner), url.PathEscape(request.Repository.Name))
	if _, err := g.transport.get(ctx, path, &permission); err != nil {
		return "", err
	}
	if !permission.Permissions.Pull {
		return "", &providerError{code: codehostbroker.ErrorForbidden, message: "GitHub actor cannot collaborate on the selected pull request"}
	}
	if (request.Operation == codehostbroker.OperationApprove || request.Operation == codehostbroker.OperationRequestChanges) &&
		!permission.Permissions.Push {
		return "", &providerError{code: codehostbroker.ErrorForbidden, message: "GitHub actor cannot submit a decision review"}
	}
	if permission.Permissions.Push {
		return "push", nil
	}
	return "pull", nil
}

func (g *githubAdapter) authenticatedActor(ctx context.Context) (codehostbroker.Actor, error) {
	var user githubAuthenticatedUser
	if _, err := g.transport.get(ctx, "user", &user); err != nil {
		return codehostbroker.Actor{}, err
	}
	actor := normalizeActor(githubUser(user))
	if actor.Login == "" || actor.ProviderID == "" {
		return codehostbroker.Actor{}, &providerError{code: codehostbroker.ErrorProvider, message: "GitHub returned an incomplete authenticated actor"}
	}
	actor.Display = ""
	return actor, nil
}

func (g *githubAdapter) rawComments(ctx context.Context, request codehostbroker.Request) ([]githubComment, error) {
	path := fmt.Sprintf("repos/%s/%s/issues/%d/comments?per_page=%d&page=1",
		url.PathEscape(request.Repository.Owner), url.PathEscape(request.Repository.Name),
		request.PullRequest.Number, codehostbroker.MaxItems)
	var comments []githubComment
	if _, err := g.transport.get(ctx, path, &comments); err != nil {
		return nil, err
	}
	if len(comments) > codehostbroker.MaxItems {
		comments = comments[:codehostbroker.MaxItems]
	}
	return comments, nil
}

func (g *githubAdapter) rawReviews(ctx context.Context, request codehostbroker.Request) ([]githubReview, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews?per_page=%d&page=1",
		url.PathEscape(request.Repository.Owner), url.PathEscape(request.Repository.Name),
		request.PullRequest.Number, codehostbroker.MaxItems)
	var reviews []githubReview
	if _, err := g.transport.get(ctx, path, &reviews); err != nil {
		return nil, err
	}
	if len(reviews) > codehostbroker.MaxItems {
		reviews = reviews[:codehostbroker.MaxItems]
	}
	return reviews, nil
}

func (g *githubAdapter) dispatchCollaboration(ctx context.Context, request codehostbroker.Request, payload collaborationPayload, preflight collaborationPreflight, marker string) (mutationEffect, error) {
	body := injectHeroMarker(payload.body, marker)
	switch request.Operation {
	case codehostbroker.OperationComment:
		path := fmt.Sprintf("repos/%s/%s/issues/%d/comments",
			url.PathEscape(request.Repository.Owner), url.PathEscape(request.Repository.Name), request.PullRequest.Number)
		var wire githubComment
		if _, err := g.transport.post(ctx, path, githubIssueCommentRequest{Body: body}, &wire); err != nil {
			return mutationEffect{}, err
		}
		if !containsExactHeroMarker(wire.Body, marker) || !sameActor(normalizeActor(wire.User), preflight.actor) {
			return mutationEffect{}, &providerError{code: codehostbroker.ErrorProvider, message: "GitHub returned an invalid comment receipt", ambiguous: true}
		}
		return mutationEffect{
			pullRequest: preflight.pullRequest,
			receipt: safeCollaborationReceipt(
				preflight.pullRequest.Identity,
				providerID(wire.Node, wire.ID),
				preflight.actor,
				payload.expectedHeadSHA,
			),
		}, nil
	default:
		path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews",
			url.PathEscape(request.Repository.Owner), url.PathEscape(request.Repository.Name), request.PullRequest.Number)
		var wire githubReview
		if _, err := g.transport.post(ctx, path, githubReviewRequest{
			Body: body, Event: collaborationReviewEvent(request.Operation), CommitID: payload.expectedHeadSHA,
		}, &wire); err != nil {
			return mutationEffect{}, err
		}
		if !containsExactHeroMarker(wire.Body, marker) || wire.CommitID != payload.expectedHeadSHA ||
			!sameActor(normalizeActor(wire.User), preflight.actor) ||
			!strings.EqualFold(wire.State, collaborationReviewState(request.Operation)) {
			return mutationEffect{}, &providerError{code: codehostbroker.ErrorProvider, message: "GitHub returned an invalid review receipt", ambiguous: true}
		}
		return mutationEffect{
			pullRequest: preflight.pullRequest,
			receipt: safeCollaborationReceipt(
				preflight.pullRequest.Identity,
				providerID(wire.Node, wire.ID),
				preflight.actor,
				payload.expectedHeadSHA,
			),
		}, nil
	}
}

func (g *githubAdapter) reconcileCollaboration(ctx context.Context, request codehostbroker.Request, payload collaborationPayload, actor codehostbroker.Actor, marker string) (*mutationEffect, error) {
	pullRequest, err := g.fetchPullRequest(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return nil, err
	}
	if pullRequest.Head.SHA != payload.expectedHeadSHA {
		return nil, &providerError{code: codehostbroker.ErrorStaleObservation, message: "pull request head changed during collaboration reconciliation"}
	}
	if request.Operation == codehostbroker.OperationComment {
		comments, err := g.rawComments(ctx, request)
		if err != nil {
			return nil, err
		}
		for _, comment := range comments {
			if containsExactHeroMarker(comment.Body, marker) && sameActor(normalizeActor(comment.User), actor) {
				return &mutationEffect{
					pullRequest: pullRequest,
					receipt: safeCollaborationReceipt(
						pullRequest.Identity, providerID(comment.Node, comment.ID), actor, payload.expectedHeadSHA,
					),
				}, nil
			}
		}
		return nil, nil
	}
	reviews, err := g.rawReviews(ctx, request)
	if err != nil {
		return nil, err
	}
	for _, review := range reviews {
		if containsExactHeroMarker(review.Body, marker) && review.CommitID == payload.expectedHeadSHA &&
			sameActor(normalizeActor(review.User), actor) &&
			strings.EqualFold(review.State, collaborationReviewState(request.Operation)) {
			return &mutationEffect{
				pullRequest: pullRequest,
				receipt: safeCollaborationReceipt(
					pullRequest.Identity, providerID(review.Node, review.ID), actor, payload.expectedHeadSHA,
				),
			}, nil
		}
	}
	return nil, nil
}

func (b *Broker) executeCollaboration(ctx context.Context, request codehostbroker.Request, payload collaborationPayload, connection config.CodeHostConnection, adapter *githubAdapter) (adapterResult, error) {
	marker := collaborationMarker(request)
	var observed collaborationPreflight
	plan := mutationPlan{
		request:       request,
		payloadDigest: canonicalCollaborationDigest(request, payload),
		target: mutationTarget{
			ConnectionID:    request.ConnectionID,
			Repository:      request.Repository,
			PullRequest:     request.PullRequest,
			ExpectedHeadSHA: payload.expectedHeadSHA,
			Marker:          marker,
		},
		connection: connection,
		adapter:    adapter,
	}
	plan.preflight = func(ctx context.Context) (mutationPreflight, error) {
		var err error
		observed, err = adapter.observeCollaborationPreflight(ctx, request, payload, marker)
		if err != nil {
			return mutationPreflight{}, err
		}
		return mutationPreflight{observationRevision: observed.revision, external: observed.external}, nil
	}
	plan.dispatch = func(ctx context.Context) (mutationEffect, error) {
		return adapter.dispatchCollaboration(ctx, request, payload, observed, marker)
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
		return adapter.reconcileCollaboration(ctx, request, payload, actor, marker)
	}
	return b.executeMutation(ctx, plan)
}

func authoritativeCurrentHeadReview(reviews []githubReview, actor codehostbroker.Actor, headSHA string) *githubReview {
	for index := len(reviews) - 1; index >= 0; index-- {
		review := &reviews[index]
		if review.CommitID != headSHA || !sameActor(normalizeActor(review.User), actor) {
			continue
		}
		if strings.EqualFold(review.State, "dismissed") {
			return nil
		}
		return review
	}
	return nil
}

func collaborationReviewEvent(operation codehostbroker.Operation) string {
	switch operation {
	case codehostbroker.OperationApprove:
		return "APPROVE"
	case codehostbroker.OperationRequestChanges:
		return "REQUEST_CHANGES"
	default:
		return "COMMENT"
	}
}

func collaborationReviewState(operation codehostbroker.Operation) string {
	switch operation {
	case codehostbroker.OperationApprove:
		return "APPROVED"
	case codehostbroker.OperationRequestChanges:
		return "CHANGES_REQUESTED"
	default:
		return "COMMENTED"
	}
}

func sameActor(left, right codehostbroker.Actor) bool {
	if left.ProviderID != "" && right.ProviderID != "" {
		return left.ProviderID == right.ProviderID
	}
	return strings.EqualFold(left.Login, right.Login)
}

func safeCollaborationReceipt(identity codehostbroker.PullRequestIdentity, providerReceiptID string, actor codehostbroker.Actor, headSHA string) *journalReceipt {
	actor.Display = ""
	return &journalReceipt{
		Identity:          identity,
		ProviderReceiptID: boundedText(providerReceiptID, 512),
		Actor:             &actor,
		HeadSHA:           boundedText(headSHA, 128),
	}
}
