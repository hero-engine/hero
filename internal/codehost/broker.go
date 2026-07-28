// Package codehost implements Hero's credential-safe, provider-neutral
// code-host broker.
package codehost

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/config"
)

const brokerTimeout = 120 * time.Second

var readOperations = []codehostbroker.Operation{
	codehostbroker.OperationCapabilities,
	codehostbroker.OperationListPullRequests,
	codehostbroker.OperationSearchPullRequests,
	codehostbroker.OperationGetPullRequest,
	codehostbroker.OperationGetCommits,
	codehostbroker.OperationGetDiff,
	codehostbroker.OperationGetChecks,
	codehostbroker.OperationGetReviews,
	codehostbroker.OperationGetComments,
	codehostbroker.OperationGetMergeReadiness,
}

var availableOperations = append(append([]codehostbroker.Operation(nil), readOperations...),
	codehostbroker.OperationCreatePullRequest,
	codehostbroker.OperationComment,
	codehostbroker.OperationSubmitReview,
	codehostbroker.OperationApprove,
	codehostbroker.OperationRequestChanges,
)

// Broker is the in-process code-host credential boundary.
type Broker struct {
	projectRoot string
	loadConfig  func(string) (config.Config, error)
	httpClient  *http.Client
	now         func() time.Time
}

// NewBroker creates a broker rooted at projectRoot.
func NewBroker(projectRoot string) *Broker {
	return &Broker{
		projectRoot: projectRoot,
		loadConfig:  config.Load,
		httpClient:  &http.Client{Timeout: brokerTimeout},
		now:         time.Now,
	}
}

type adapterResult struct {
	result              any
	pullRequest         *codehostbroker.PullRequest
	page                *codehostbroker.Page
	completeness        codehostbroker.Completeness
	partialFailures     []codehostbroker.PartialFailure
	truncated           bool
	rateLimit           codehostbroker.RateLimit
	redirects           int
	receipt             *codehostbroker.Receipt
	reconciliation      *codehostbroker.Reconciliation
	journalEntries      int
	capabilityRevision  string
	observationRevision string
}

// Execute validates and dispatches one provider-neutral code-host operation.
func (b *Broker) Execute(ctx context.Context, request codehostbroker.Request) codehostbroker.Response {
	start := b.now()
	observedAt := start.UTC().Format(time.RFC3339Nano)
	response := baseResponse(request, observedAt)

	if request.Cursor != "" {
		envelope, cursorErr := codehostbroker.DecodeCursor(request.Cursor)
		if cursorErr != nil {
			response.Error = cursorErr
			return b.finish(response, start)
		}
		if envelope.Material.Version != request.Version {
			response.Error = contractError(codehostbroker.ErrorCursorMismatch, "cursor does not match the requested contract version", "cursor")
			return b.finish(response, start)
		}
	}
	if err := codehostbroker.ValidateRequest(request); err != nil {
		response.Error = err
		return b.finish(response, start)
	}
	if !codehostbroker.IsRead(request.Operation) &&
		request.Operation != codehostbroker.OperationCreatePullRequest &&
		!isCollaborationOperation(request.Operation) {
		response.Error = contractError(codehostbroker.ErrorUnsupportedOperation, "the selected adapter does not implement the operation", "operation")
		return b.finish(response, start)
	}
	var creation createPayload
	var collaboration collaborationPayload
	if request.Operation == codehostbroker.OperationCreatePullRequest {
		var payloadErr *codehostbroker.ContractError
		creation, payloadErr = decodeCreatePayload(request.Payload)
		if payloadErr != nil {
			response.Error = payloadErr
			return b.finish(response, start)
		}
	} else if isCollaborationOperation(request.Operation) {
		var payloadErr *codehostbroker.ContractError
		collaboration, payloadErr = decodeCollaborationPayload(request)
		if payloadErr != nil {
			response.Error = payloadErr
			return b.finish(response, start)
		}
	}
	if err := ctx.Err(); err != nil && codehostbroker.IsRead(request.Operation) {
		response.Error = cancelledError(err)
		return b.finish(response, start)
	}

	cfg, err := b.loadConfig(b.projectRoot)
	if err != nil {
		response.Error = contractError(codehostbroker.ErrorProviderUnavailable, "code-host configuration is unavailable", "")
		return b.finish(response, start)
	}
	connection, err := cfg.ResolveCodeHostConnection(request.ConnectionID)
	if err != nil {
		response.Error = resolutionError(err)
		return b.finish(response, start)
	}
	response.Provider = connection.Provider
	response.ConnectionID = connection.ID
	if request.Provider != connection.Provider {
		response.Error = contractError(codehostbroker.ErrorWrongConnectionCapability, "request provider does not match the selected connection", "provider")
		return b.finish(response, start)
	}
	if connection.Provider != "github" {
		response.Error = contractError(codehostbroker.ErrorUnsupportedProvider, "the selected code-host provider has no broker adapter", "provider")
		return b.finish(response, start)
	}

	var scope []codehostbroker.RepositoryIdentity
	if request.Operation == codehostbroker.OperationCreatePullRequest {
		if scopeErr := validateCreateScope(connection, request, creation); scopeErr != nil {
			response.Error = scopeErr
			return b.finish(response, start)
		}
		scope = []codehostbroker.RepositoryIdentity{creation.Base.Repository}
		if creation.Head.Repository != creation.Base.Repository {
			scope = append(scope, creation.Head.Repository)
		}
	} else if isCollaborationOperation(request.Operation) {
		if scopeErr := validateCollaborationScope(connection, request); scopeErr != nil {
			response.Error = scopeErr
			return b.finish(response, start)
		}
		scope = []codehostbroker.RepositoryIdentity{request.Repository}
	} else {
		var scopeErr *codehostbroker.ContractError
		scope, scopeErr = validateRepositoryScope(connection, request)
		if scopeErr != nil {
			response.Error = scopeErr
			return b.finish(response, start)
		}
	}
	token, err := connection.ResolveToken()
	if err != nil {
		response.Error = resolutionError(err)
		return b.finish(response, start)
	}
	if len(token.Reveal()) > codehostbroker.MaxTextBytes {
		response.Error = contractError(codehostbroker.ErrorCredentialUnavailable, "the selected credential exceeds the safety bound", "")
		return b.finish(response, start)
	}
	response.CapabilityRevision = capabilityRevision(connection, scope)

	if err := ctx.Err(); err != nil && codehostbroker.IsRead(request.Operation) {
		response.Error = cancelledError(err)
		return b.finish(response, start)
	}
	ctx, cancel := context.WithTimeout(ctx, brokerTimeout)
	defer cancel()

	transport, err := newGitHubTransport(connection, token, b.httpClient, b.now)
	if err != nil {
		response.Error = contractError(codehostbroker.ErrorInvalidInput, "the configured GitHub origin is invalid", "connection")
		return b.finish(response, start)
	}
	adapter := githubAdapter{connection: connection, transport: transport, now: b.now}
	var out adapterResult
	var dispatchErr error
	if request.Operation == codehostbroker.OperationCreatePullRequest {
		out, dispatchErr = b.executeCreate(ctx, request, creation, connection, &adapter)
	} else if isCollaborationOperation(request.Operation) {
		out, dispatchErr = b.executeCollaboration(ctx, request, collaboration, connection, &adapter)
	} else {
		out, dispatchErr = adapter.execute(ctx, request, scope)
	}
	out = boundAdapterResult(request.Operation, out, response.Bounds)
	response.RateLimit = out.rateLimit
	if response.RateLimit.ObservedAt == "" {
		response.RateLimit.ObservedAt = observedAt
	}
	response.Redirects = out.redirects
	response.Receipt = out.receipt
	response.Reconciliation = out.reconciliation
	response.JournalEntries = out.journalEntries
	if out.capabilityRevision != "" {
		response.CapabilityRevision = out.capabilityRevision
	}
	response.Page = out.page
	response.Completeness = out.completeness
	if response.Completeness == "" {
		response.Completeness = codehostbroker.CompletenessComplete
	}
	response.PartialFailures = boundedFailures(out.partialFailures)
	response.Truncated = out.truncated
	if dispatchErr != nil {
		normalized := normalizeProviderError(dispatchErr)
		if out.result == nil {
			response.Error = normalized
		} else {
			response.PartialFailures = boundedFailures(append(response.PartialFailures, codehostbroker.PartialFailure{
				Section: string(request.Operation),
				Code:    normalized.Code,
				Message: normalized.Message,
			}))
			response.Completeness = codehostbroker.CompletenessPartial
		}
	}
	if response.Error == nil {
		raw, marshalErr := json.Marshal(out.result)
		if marshalErr != nil {
			response.Error = contractError(codehostbroker.ErrorEncoding, "could not encode the normalized code-host result", "")
		} else {
			response.Result = raw
		}
	}
	if out.observationRevision != "" {
		response.ObservationRevision = out.observationRevision
	} else {
		response.ObservationRevision = observationRevision(request, out, response)
	}
	return b.finish(response, start)
}

func baseResponse(request codehostbroker.Request, observedAt string) codehostbroker.Response {
	policy, _ := codehostbroker.Policy(request.Operation)
	return codehostbroker.Response{
		Version:             codehostbroker.Version,
		Operation:           request.Operation,
		Provider:            request.Provider,
		ConnectionID:        request.ConnectionID,
		Repository:          request.Repository,
		Policy:              policy,
		CapabilityRevision:  fingerprint("capability", request.Provider, request.ConnectionID),
		ObservationRevision: fingerprint("observation", request.Operation, request.Repository),
		ObservedAt:          observedAt,
		Freshness:           codehostbroker.FreshnessCurrent,
		RateLimit:           codehostbroker.RateLimit{ObservedAt: observedAt},
		Bounds:              policy.Bounds,
		Completeness:        codehostbroker.CompletenessUnavailable,
		PartialFailures:     []codehostbroker.PartialFailure{},
	}
}

func (b *Broker) finish(response codehostbroker.Response, start time.Time) codehostbroker.Response {
	response.DurationMS = b.now().Sub(start).Milliseconds()
	if response.DurationMS < 0 {
		response.DurationMS = 0
	}
	if max := int64(response.Bounds.DurationMS); max > 0 && response.DurationMS > max {
		response.DurationMS = max
	}
	if response.Error != nil {
		response.Result = nil
		response.Completeness = codehostbroker.CompletenessUnavailable
		response.Freshness = codehostbroker.FreshnessUnavailable
	}
	return response
}

func validateRepositoryScope(connection config.CodeHostConnection, request codehostbroker.Request) ([]codehostbroker.RepositoryIdentity, *codehostbroker.ContractError) {
	host, err := repositoryHost(connection)
	if err != nil {
		return nil, contractError(codehostbroker.ErrorInvalidInput, "the configured GitHub origin is invalid", "connection")
	}
	allowed := make(map[string]struct{}, len(connection.Repositories))
	for _, repository := range connection.Repositories {
		allowed[strings.ToLower(repository)] = struct{}{}
	}
	scope := append([]codehostbroker.RepositoryIdentity{request.Repository}, request.Repositories...)
	seen := make(map[string]struct{}, len(scope))
	normalized := make([]codehostbroker.RepositoryIdentity, 0, len(scope))
	for _, repository := range scope {
		if !strings.EqualFold(repository.Host, host) {
			return nil, contractError(codehostbroker.ErrorInvalidInput, "repository host is outside the selected connection origin", "repository.host")
		}
		key := strings.ToLower(repository.FullName)
		if _, ok := allowed[key]; !ok {
			return nil, contractError(codehostbroker.ErrorForbidden, "repository is outside the selected connection scope", "repository.full_name")
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, repository)
	}
	sort.Slice(normalized, func(i, j int) bool {
		return strings.ToLower(normalized[i].FullName) < strings.ToLower(normalized[j].FullName)
	})
	return normalized, nil
}

func repositoryHost(connection config.CodeHostConnection) (string, error) {
	if strings.TrimSpace(connection.BaseURL) == "" {
		return "github.com", nil
	}
	parsed, err := url.Parse(connection.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return "", errors.New("invalid origin")
	}
	return parsed.Hostname(), nil
}

func capabilityRevision(connection config.CodeHostConnection, _ []codehostbroker.RepositoryIdentity) string {
	available := make([]codehostbroker.OperationPolicy, 0, len(availableOperations))
	for _, operation := range availableOperations {
		policy, _ := codehostbroker.Policy(operation)
		available = append(available, policy)
	}
	repositories := append([]string(nil), connection.Repositories...)
	sort.Slice(repositories, func(i, j int) bool {
		return strings.ToLower(repositories[i]) < strings.ToLower(repositories[j])
	})
	host, _ := repositoryHost(connection)
	return fingerprint("capability", connection.ID, connection.Provider, host, repositories, available)
}

func observationRevision(request codehostbroker.Request, out adapterResult, response codehostbroker.Response) string {
	material := struct {
		Version      string
		Operation    codehostbroker.Operation
		ConnectionID string
		Repository   codehostbroker.RepositoryIdentity
		PullRequest  *codehostbroker.PullRequest
		Result       any
		Page         *codehostbroker.Page
		Completeness codehostbroker.Completeness
		Partial      []codehostbroker.PartialFailure
		Truncated    bool
	}{
		Version:      codehostbroker.Version,
		Operation:    request.Operation,
		ConnectionID: response.ConnectionID,
		Repository:   request.Repository,
		PullRequest:  out.pullRequest,
		Result:       out.result,
		Page:         response.Page,
		Completeness: response.Completeness,
		Partial:      response.PartialFailures,
		Truncated:    response.Truncated,
	}
	return fingerprint("observation", material)
}

func fingerprint(prefix string, values ...any) string {
	data, _ := json.Marshal(values)
	sum := sha256.Sum256(data)
	return prefix + ":" + hex.EncodeToString(sum[:])
}

func contractError(code, message, field string) *codehostbroker.ContractError {
	return &codehostbroker.ContractError{
		Code:    code,
		Message: boundedText(message, codehostbroker.MaxErrorDetailBytes),
		Field:   field,
		Retry:   codehostbroker.RetryForError(code),
	}
}

func resolutionError(err error) *codehostbroker.ContractError {
	var resolution *config.IntegrationResolutionError
	if errors.As(err, &resolution) {
		switch resolution.Code {
		case codehostbroker.ErrorConnectionNotFound:
			return contractError(codehostbroker.ErrorConnectionNotFound, "the selected integration was not found", "connection_id")
		case codehostbroker.ErrorCodeHostRoleMissing:
			return contractError(codehostbroker.ErrorCodeHostRoleMissing, "no code-host integration is selected", "connection_id")
		case codehostbroker.ErrorWrongConnectionCapability:
			return contractError(codehostbroker.ErrorWrongConnectionCapability, "the selected integration is not a code-host connection", "connection_id")
		case codehostbroker.ErrorCredentialUnavailable:
			return contractError(codehostbroker.ErrorCredentialUnavailable, "the selected integration has no usable credential", "")
		}
	}
	return contractError(codehostbroker.ErrorProviderUnavailable, "code-host integration resolution failed", "")
}

func cancelledError(err error) *codehostbroker.ContractError {
	message := "code-host request was cancelled"
	if errors.Is(err, context.DeadlineExceeded) {
		message = "code-host request deadline was exceeded"
	}
	return contractError(codehostbroker.ErrorCancelled, message, "")
}

func boundedFailures(failures []codehostbroker.PartialFailure) []codehostbroker.PartialFailure {
	if len(failures) > codehostbroker.MaxPartialFailures {
		failures = failures[:codehostbroker.MaxPartialFailures]
	}
	out := make([]codehostbroker.PartialFailure, len(failures))
	for i, failure := range failures {
		out[i] = failure
		out[i].Message = boundedText(failure.Message, codehostbroker.MaxErrorDetailBytes)
	}
	return out
}

func boundedText(value string, limit int) string {
	value = strings.ToValidUTF8(value, "�")
	if len(value) <= limit {
		return value
	}
	value = value[:limit]
	for !utf8.ValidString(value) && len(value) > 0 {
		value = value[:len(value)-1]
	}
	return value
}

func boundAdapterResult(operation codehostbroker.Operation, out adapterResult, bounds codehostbroker.Bounds) adapterResult {
	limit := bounds.BodyBytes
	if operation == codehostbroker.OperationGetDiff {
		limit = bounds.DiffBytes
	}
	if out.result == nil || resultSize(out.result) <= limit {
		return out
	}
	out.truncated = true
	out.completeness = codehostbroker.CompletenessTruncated
	switch result := out.result.(type) {
	case codehostbroker.PullRequestsResult:
		for len(result.PullRequests) > 1 && resultSize(result) > limit {
			result.PullRequests = result.PullRequests[:len(result.PullRequests)-1]
		}
		if len(result.PullRequests) == 1 {
			shrinkPullRequest(&result.PullRequests[0], func() bool { return resultSize(result) <= limit })
		}
		if out.page != nil {
			out.page.Count = len(result.PullRequests)
		}
		out.result = result
	case codehostbroker.PullRequest:
		shrinkPullRequest(&result, func() bool { return resultSize(result) <= limit })
		out.result = result
	case codehostbroker.CommitsResult:
		for len(result.Commits) > 1 && resultSize(result) > limit {
			result.Commits = result.Commits[:len(result.Commits)-1]
		}
		if len(result.Commits) == 1 {
			shrinkString(&result.Commits[0].Message, func() bool { return resultSize(result) <= limit })
		}
		if out.page != nil {
			out.page.Count = len(result.Commits)
		}
		out.result = result
	case codehostbroker.DiffResult:
		for resultSize(result) > limit && len(result.Files) > 0 {
			last := len(result.Files) - 1
			result.Files[last].Truncated = true
			hunks := result.Files[last].Hunks
			if len(hunks) > 0 {
				hunk := &result.Files[last].Hunks[len(hunks)-1]
				if len(hunk.Patch) > 1 {
					hunk.Patch = boundedText(hunk.Patch, len(hunk.Patch)/2)
				} else {
					result.Files[last].Hunks = hunks[:len(hunks)-1]
				}
				continue
			}
			if len(result.Files) > 1 {
				result.Files = result.Files[:last]
				continue
			}
			break
		}
		out.result = result
	case codehostbroker.ChecksResult:
		for len(result.Checks) > 0 && resultSize(result) > limit {
			result.Checks = result.Checks[:len(result.Checks)-1]
		}
		out.result = result
	case codehostbroker.ReviewsResult:
		for len(result.Reviews) > 1 && resultSize(result) > limit {
			result.Reviews = result.Reviews[:len(result.Reviews)-1]
		}
		if len(result.Reviews) == 1 {
			shrinkString(&result.Reviews[0].Body, func() bool { return resultSize(result) <= limit })
		}
		if out.page != nil {
			out.page.Count = len(result.Reviews)
		}
		out.result = result
	case codehostbroker.CommentsResult:
		for len(result.Comments) > 1 && resultSize(result) > limit {
			result.Comments = result.Comments[:len(result.Comments)-1]
		}
		if len(result.Comments) == 1 {
			shrinkString(&result.Comments[0].Body, func() bool { return resultSize(result) <= limit })
		}
		if out.page != nil {
			out.page.Count = len(result.Comments)
		}
		out.result = result
	case codehostbroker.MergeReadiness:
		for len(result.Reasons) > 0 && resultSize(result) > limit {
			result.Reasons = result.Reasons[:len(result.Reasons)-1]
		}
		out.result = result
	case codehostbroker.MutationResult:
		shrinkPullRequest(&result.PullRequest, func() bool { return resultSize(result) <= limit })
		out.result = result
	}
	return out
}

func resultSize(value any) int {
	data, _ := json.Marshal(value)
	return len(data)
}

func shrinkPullRequest(pullRequest *codehostbroker.PullRequest, fits func() bool) {
	shrinkString(&pullRequest.Body, fits)
	if !fits() {
		shrinkString(&pullRequest.Title, fits)
	}
}

func shrinkString(value *string, fits func() bool) {
	for !fits() && len(*value) > 1 {
		*value = boundedText(*value, len(*value)/2)
	}
}
