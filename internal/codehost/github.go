package codehost

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/config"
)

const (
	defaultGitHubAPI       = "https://api.github.com"
	maxGitHubResponseBytes = 8 << 20
	maxSearchPages         = 10
)

type providerError struct {
	code    string
	message string
	retryAt string
	// ambiguous is true only when a write may have reached GitHub but the
	// provider outcome could not be decoded or observed.
	ambiguous bool
}

func (e *providerError) Error() string { return e.message }

func normalizeProviderError(err error) *codehostbroker.ContractError {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return cancelledError(err)
	}
	var provider *providerError
	if errors.As(err, &provider) {
		normalized := contractError(provider.code, provider.message, "")
		normalized.RetryAt = provider.retryAt
		return normalized
	}
	return contractError(codehostbroker.ErrorProvider, "GitHub returned an unusable response", "")
}

type githubTransport struct {
	connection config.CodeHostConnection
	token      config.Secret
	client     *http.Client
	base       *url.URL
	graphqlURL *url.URL
	now        func() time.Time
	rateLimit  codehostbroker.RateLimit
	redirects  int
}

func newGitHubTransport(connection config.CodeHostConnection, token config.Secret, source *http.Client, now func() time.Time) (*githubTransport, error) {
	baseValue := strings.TrimSpace(connection.BaseURL)
	if baseValue == "" {
		baseValue = defaultGitHubAPI
	}
	base, err := url.Parse(strings.TrimRight(baseValue, "/"))
	if err != nil || base.Scheme == "" || base.Host == "" || base.User != nil || base.Fragment != "" ||
		(base.Scheme != "https" && base.Scheme != "http") {
		return nil, errors.New("invalid GitHub origin")
	}
	graph := *base
	switch {
	case base.Hostname() == "api.github.com" && (base.Path == "" || base.Path == "/"):
		graph.Path = "/graphql"
	case strings.HasSuffix(base.Path, "/api/v3"):
		graph.Path = strings.TrimSuffix(base.Path, "/api/v3") + "/api/graphql"
	default:
		graph.Path = strings.TrimRight(base.Path, "/") + "/graphql"
	}
	graph.RawQuery = ""
	graph.Fragment = ""

	client := *source
	client.Timeout = brokerTimeout
	transport := &githubTransport{
		connection: connection,
		token:      token,
		client:     &client,
		base:       base,
		graphqlURL: &graph,
		now:        now,
	}
	client.CheckRedirect = transport.checkRedirect
	transport.client = &client
	return transport, nil
}

func (t *githubTransport) checkRedirect(next *http.Request, via []*http.Request) error {
	if len(via) > codehostbroker.MaxRedirects {
		return &providerError{code: codehostbroker.ErrorProvider, message: "GitHub redirect limit exceeded"}
	}
	if !sameOrigin(next.URL, t.base) {
		return &providerError{code: codehostbroker.ErrorProvider, message: "GitHub redirect left the configured origin"}
	}
	t.redirects++
	t.authorize(next)
	return nil
}

func (t *githubTransport) restURL(relative string) (*url.URL, error) {
	if relative == "" || strings.HasPrefix(relative, "//") || strings.ContainsAny(relative, "\\\r\n\x00") {
		return nil, errors.New("invalid GitHub path")
	}
	parsed, err := url.Parse(relative)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil || parsed.Fragment != "" || parsed.Opaque != "" {
		return nil, errors.New("invalid GitHub path")
	}
	base := *t.base
	base.Path = strings.TrimRight(base.Path, "/") + "/"
	target := base.ResolveReference(parsed)
	if !sameOrigin(target, t.base) {
		return nil, errors.New("GitHub path left the configured origin")
	}
	return target, nil
}

func (t *githubTransport) authorize(request *http.Request) {
	request.Header.Set("Authorization", "Bearer "+t.token.Reveal())
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func (t *githubTransport) get(ctx context.Context, relative string, target any) (http.Header, error) {
	endpoint, err := t.restURL(relative)
	if err != nil {
		return nil, &providerError{code: codehostbroker.ErrorInvalidInput, message: "GitHub request path is invalid"}
	}
	return t.doJSON(ctx, http.MethodGet, endpoint, nil, target)
}

func (t *githubTransport) post(ctx context.Context, relative string, value, target any) (http.Header, error) {
	endpoint, err := t.restURL(relative)
	if err != nil {
		return nil, &providerError{code: codehostbroker.ErrorInvalidInput, message: "GitHub request path is invalid"}
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, &providerError{code: codehostbroker.ErrorEncoding, message: "could not encode the GitHub request"}
	}
	return t.doJSON(ctx, http.MethodPost, endpoint, bytes.NewReader(body), target)
}

func (t *githubTransport) patch(ctx context.Context, relative string, value, target any) (http.Header, error) {
	endpoint, err := t.restURL(relative)
	if err != nil {
		return nil, &providerError{code: codehostbroker.ErrorInvalidInput, message: "GitHub request path is invalid"}
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, &providerError{code: codehostbroker.ErrorEncoding, message: "could not encode the GitHub request"}
	}
	return t.doJSON(ctx, http.MethodPatch, endpoint, bytes.NewReader(body), target)
}

func (t *githubTransport) put(ctx context.Context, relative string, value, target any) (http.Header, error) {
	endpoint, err := t.restURL(relative)
	if err != nil {
		return nil, &providerError{code: codehostbroker.ErrorInvalidInput, message: "GitHub request path is invalid"}
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, &providerError{code: codehostbroker.ErrorEncoding, message: "could not encode the GitHub request"}
	}
	return t.doJSON(ctx, http.MethodPut, endpoint, bytes.NewReader(body), target)
}

func (t *githubTransport) graphql(ctx context.Context, query string, variables map[string]any, target any) (http.Header, error) {
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return nil, &providerError{code: codehostbroker.ErrorEncoding, message: "could not encode the GitHub GraphQL request"}
	}
	return t.doJSON(ctx, http.MethodPost, t.graphqlURL, bytes.NewReader(body), target)
}

func (t *githubTransport) doJSON(ctx context.Context, method string, endpoint *url.URL, body io.Reader, target any) (http.Header, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !sameOrigin(endpoint, t.base) {
		return nil, &providerError{code: codehostbroker.ErrorProvider, message: "GitHub pagination left the configured origin"}
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return nil, &providerError{code: codehostbroker.ErrorInvalidInput, message: "could not create the GitHub request"}
	}
	t.authorize(request)
	if method != http.MethodGet {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := t.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			if method == http.MethodGet {
				return nil, ctx.Err()
			}
			return nil, &providerError{code: codehostbroker.ErrorCancelled, message: "GitHub write response was cancelled", ambiguous: true}
		}
		var provider *providerError
		if errors.As(err, &provider) {
			return nil, provider
		}
		return nil, &providerError{code: codehostbroker.ErrorProviderUnavailable, message: "GitHub is unavailable", ambiguous: method != http.MethodGet}
	}
	defer response.Body.Close()
	t.observeRateLimit(response.Header)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		providerErr := t.httpError(response.StatusCode, response.Header)
		if method != http.MethodGet && response.StatusCode >= 500 {
			providerErr.ambiguous = true
		}
		return response.Header.Clone(), providerErr
	}
	limited := io.LimitReader(response.Body, maxGitHubResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return response.Header.Clone(), &providerError{code: codehostbroker.ErrorProvider, message: "could not read the GitHub response"}
	}
	if len(data) > maxGitHubResponseBytes {
		return response.Header.Clone(), &providerError{code: codehostbroker.ErrorOutputTooLarge, message: "GitHub response exceeded the bounded read limit"}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		return response.Header.Clone(), &providerError{code: codehostbroker.ErrorProvider, message: "GitHub returned malformed JSON", ambiguous: method != http.MethodGet}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return response.Header.Clone(), &providerError{code: codehostbroker.ErrorProvider, message: "GitHub returned trailing JSON data", ambiguous: method != http.MethodGet}
	}
	return response.Header.Clone(), nil
}

func (t *githubTransport) observeRateLimit(header http.Header) {
	observedAt := t.now().UTC().Format(time.RFC3339Nano)
	rate := codehostbroker.RateLimit{
		Resource:   header.Get("X-RateLimit-Resource"),
		Limit:      parseNonNegativeInt64(header.Get("X-RateLimit-Limit")),
		Remaining:  parseNonNegativeInt64(header.Get("X-RateLimit-Remaining")),
		RetryAfter: parseNonNegativeInt64(header.Get("Retry-After")),
		ObservedAt: observedAt,
	}
	if reset := parseNonNegativeInt64(header.Get("X-RateLimit-Reset")); reset > 0 {
		rate.ResetAt = time.Unix(reset, 0).UTC().Format(time.RFC3339Nano)
	}
	if rate.Resource != "" || rate.Limit > 0 || rate.Remaining > 0 || rate.ResetAt != "" || rate.RetryAfter > 0 {
		t.rateLimit = rate
	} else if t.rateLimit.ObservedAt == "" {
		t.rateLimit = rate
	}
}

func (t *githubTransport) observeGraphQLRateLimit(rate githubGraphQLRateLimit) {
	if rate.Limit < 0 || rate.Remaining < 0 || (rate.Limit > 0 && rate.Remaining > rate.Limit) {
		return
	}
	resetAt := rate.ResetAt
	if resetAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, resetAt); err != nil {
			resetAt = ""
		}
	}
	t.rateLimit = codehostbroker.RateLimit{
		Resource:   "graphql",
		Limit:      rate.Limit,
		Remaining:  rate.Remaining,
		ResetAt:    resetAt,
		RetryAfter: t.rateLimit.RetryAfter,
		ObservedAt: t.now().UTC().Format(time.RFC3339Nano),
	}
}

func (t *githubTransport) httpError(status int, header http.Header) *providerError {
	code := codehostbroker.ErrorProvider
	message := fmt.Sprintf("GitHub returned HTTP %d", status)
	switch status {
	case http.StatusUnauthorized:
		code = codehostbroker.ErrorUnauthorized
	case http.StatusForbidden:
		if header.Get("X-RateLimit-Remaining") == "0" {
			code = codehostbroker.ErrorRateLimited
			message = "GitHub rate limit was exhausted"
		} else if strings.Contains(
			strings.ToLower(header.Get("X-GitHub-SSO")),
			"required",
		) {
			code = codehostbroker.ErrorSSORequired
			message = "GitHub requires this token to be authorized for the organization's SAML SSO"
		} else {
			code = codehostbroker.ErrorForbidden
		}
	case http.StatusNotFound:
		code = codehostbroker.ErrorNotFound
	case http.StatusConflict:
		code = codehostbroker.ErrorConflict
	case http.StatusTooManyRequests:
		code = codehostbroker.ErrorRateLimited
		message = "GitHub rate limited the request"
	default:
		if status >= 500 {
			code = codehostbroker.ErrorProviderUnavailable
			message = "GitHub is unavailable"
		}
	}
	retryAt := ""
	if code == codehostbroker.ErrorRateLimited && t.rateLimit.ResetAt != "" {
		retryAt = t.rateLimit.ResetAt
	}
	return &providerError{code: code, message: message, retryAt: retryAt}
}

func (t *githubTransport) validateNextLink(header http.Header) (string, error) {
	next := nextLink(header.Get("Link"))
	if next == "" {
		return "", nil
	}
	parsed, err := url.Parse(next)
	if err != nil || !sameOrigin(parsed, t.base) {
		return "", &providerError{code: codehostbroker.ErrorProvider, message: "GitHub pagination left the configured origin"}
	}
	return next, nil
}

func sameOrigin(left, right *url.URL) bool {
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

func nextLink(header string) string {
	for _, value := range strings.Split(header, ",") {
		parts := strings.Split(value, ";")
		if len(parts) < 2 {
			continue
		}
		relation := false
		for _, attribute := range parts[1:] {
			if strings.TrimSpace(attribute) == `rel="next"` || strings.TrimSpace(attribute) == "rel=next" {
				relation = true
			}
		}
		if relation {
			return strings.Trim(strings.TrimSpace(parts[0]), "<>")
		}
	}
	return ""
}

func parseNonNegativeInt64(value string) int64 {
	number, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || number < 0 {
		return 0
	}
	return number
}

type githubAdapter struct {
	connection config.CodeHostConnection
	transport  *githubTransport
	now        func() time.Time
}

func (g *githubAdapter) execute(ctx context.Context, request codehostbroker.Request, scope []codehostbroker.RepositoryIdentity) (adapterResult, error) {
	var out adapterResult
	var err error
	switch request.Operation {
	case codehostbroker.OperationCapabilities:
		out, err = g.capabilities(ctx, request)
	case codehostbroker.OperationGetAuthenticatedActor:
		out, err = g.getAuthenticatedActor(ctx)
	case codehostbroker.OperationListPullRequests:
		out, err = g.list(ctx, request, scope, false)
	case codehostbroker.OperationSearchPullRequests:
		out, err = g.list(ctx, request, scope, true)
	case codehostbroker.OperationGetPullRequest:
		out, err = g.getPullRequest(ctx, request)
	case codehostbroker.OperationGetCommits:
		out, err = g.getCommits(ctx, request)
	case codehostbroker.OperationGetDiff:
		out, err = g.getDiff(ctx, request)
	case codehostbroker.OperationGetChecks:
		out, err = g.getChecks(ctx, request)
	case codehostbroker.OperationGetReviews:
		out, err = g.getReviews(ctx, request)
	case codehostbroker.OperationGetComments:
		out, err = g.getComments(ctx, request)
	case codehostbroker.OperationGetMergeReadiness:
		out, err = g.getMergeReadiness(ctx, request)
	default:
		err = &providerError{code: codehostbroker.ErrorUnsupportedOperation, message: "GitHub adapter does not implement the operation"}
	}
	if out.rateLimit.ObservedAt == "" {
		out.rateLimit = g.transport.rateLimit
	}
	out.redirects = g.transport.redirects
	return out, err
}

func (g *githubAdapter) getAuthenticatedActor(ctx context.Context) (adapterResult, error) {
	actor, err := g.authenticatedActor(ctx)
	if err != nil {
		return adapterResult{}, err
	}
	return adapterResult{
		result:       codehostbroker.AuthenticatedActorResult{Actor: actor},
		completeness: codehostbroker.CompletenessComplete,
	}, nil
}

func (g *githubAdapter) capabilities(ctx context.Context, request codehostbroker.Request) (adapterResult, error) {
	merge, err := g.mergeRuntimeCapability(ctx, request.Repository, "")
	if err != nil {
		return adapterResult{}, err
	}
	capabilities := make([]codehostbroker.Capability, 0, len(availableOperations))
	for _, operation := range availableOperations {
		policy, _ := codehostbroker.Policy(operation)
		capability := codehostbroker.Capability{Policy: policy, Available: true}
		if operation == codehostbroker.OperationMerge {
			capability.Available = merge.available
			capability.Reason = merge.reason
			capability.Merge = &merge.material
		}
		capabilities = append(capabilities, capability)
	}
	return adapterResult{
		result:             codehostbroker.CapabilitiesResult{Capabilities: capabilities},
		completeness:       codehostbroker.CompletenessComplete,
		capabilityRevision: capabilityRevision(g.connection, nil),
	}, nil
}

type cursorPosition struct {
	Repository int `json:"repository"`
	Page       int `json:"page"`
}

func decodePosition(request codehostbroker.Request) (cursorPosition, error) {
	position := cursorPosition{Page: 1}
	if request.Cursor == "" {
		return position, nil
	}
	envelope, contractErr := codehostbroker.DecodeCursor(request.Cursor)
	if contractErr != nil {
		return position, contractErr
	}
	data, err := base64.RawURLEncoding.DecodeString(envelope.Material.Position)
	if err != nil || json.Unmarshal(data, &position) != nil || position.Repository < 0 || position.Page < 1 {
		return cursorPosition{}, &providerError{code: codehostbroker.ErrorCursorMismatch, message: "cursor position is invalid"}
	}
	return position, nil
}

func encodePosition(request codehostbroker.Request, scope []codehostbroker.RepositoryIdentity, position cursorPosition) (string, error) {
	data, err := json.Marshal(position)
	if err != nil {
		return "", err
	}
	cursor, contractErr := codehostbroker.EncodeCursor(codehostbroker.CursorMaterial{
		Version:      codehostbroker.Version,
		Provider:     request.Provider,
		ConnectionID: request.ConnectionID,
		Repositories: scope,
		Operation:    request.Operation,
		Query:        request.Query,
		Order:        request.Order,
		Position:     base64.RawURLEncoding.EncodeToString(data),
	})
	if contractErr != nil {
		return "", contractErr
	}
	return cursor, nil
}

func (g *githubAdapter) list(ctx context.Context, request codehostbroker.Request, scope []codehostbroker.RepositoryIdentity, search bool) (adapterResult, error) {
	position, err := decodePosition(request)
	if err != nil {
		return adapterResult{}, err
	}
	if position.Repository >= len(scope) {
		return adapterResult{}, &providerError{code: codehostbroker.ErrorCursorMismatch, message: "cursor repository position is invalid"}
	}
	limit := normalizedLimit(request.Limit)
	state, terms := parseQuery(request.Query)
	sortValue, direction, err := normalizeOrder(request.Order)
	if err != nil {
		return adapterResult{}, err
	}
	result := codehostbroker.PullRequestsResult{PullRequests: []codehostbroker.PullRequest{}}
	nextPosition := cursorPosition{}
	hasNext := false
	pages := 0

	for repositoryIndex := position.Repository; repositoryIndex < len(scope) && len(result.PullRequests) < limit; repositoryIndex++ {
		page := 1
		if repositoryIndex == position.Repository {
			page = position.Page
		}
		for len(result.PullRequests) < limit {
			if err := ctx.Err(); err != nil {
				if len(result.PullRequests) == 0 {
					return adapterResult{}, err
				}
				return partialCollection(result, request, scope, limit, result.PullRequests, cursorPosition{Repository: repositoryIndex, Page: page}, err)
			}
			remaining := limit - len(result.PullRequests)
			path := pullListPath(scope[repositoryIndex], state, sortValue, direction, remaining, page)
			var wire []githubPullRequest
			header, fetchErr := g.transport.get(ctx, path, &wire)
			if fetchErr != nil {
				if len(result.PullRequests) == 0 {
					return adapterResult{}, fetchErr
				}
				return partialCollection(result, request, scope, limit, result.PullRequests, cursorPosition{Repository: repositoryIndex, Page: page}, fetchErr)
			}
			next, nextErr := g.transport.validateNextLink(header)
			if nextErr != nil {
				if len(result.PullRequests) == 0 {
					return adapterResult{}, nextErr
				}
				return partialCollection(result, request, scope, limit, result.PullRequests, cursorPosition{Repository: repositoryIndex, Page: page}, nextErr)
			}
			for _, item := range wire {
				pullRequest := normalizePullRequest(g.connection.ID, item)
				setPullRequestHost(&pullRequest, scope[repositoryIndex].Host)
				if err := validateNormalizedPullRequest(pullRequest); err != nil {
					if len(result.PullRequests) == 0 {
						return adapterResult{}, err
					}
					return partialCollection(result, request, scope, limit, result.PullRequests, cursorPosition{Repository: repositoryIndex, Page: page}, err)
				}
				if search && !matchesTerms(pullRequest, terms) {
					continue
				}
				result.PullRequests = append(result.PullRequests, pullRequest)
				if len(result.PullRequests) == limit {
					break
				}
			}
			pages++
			if next != "" {
				nextPosition = cursorPosition{Repository: repositoryIndex, Page: page + 1}
				hasNext = true
				if len(result.PullRequests) >= limit || (search && pages >= maxSearchPages) {
					break
				}
				page++
				continue
			}
			if repositoryIndex+1 < len(scope) {
				nextPosition = cursorPosition{Repository: repositoryIndex + 1, Page: 1}
				hasNext = true
			} else {
				hasNext = false
			}
			break
		}
		if len(result.PullRequests) >= limit || (search && pages >= maxSearchPages) {
			break
		}
	}
	page := &codehostbroker.Page{Limit: limit, Count: len(result.PullRequests)}
	if hasNext {
		page.NextCursor, err = encodePosition(request, scope, nextPosition)
		if err != nil {
			return adapterResult{}, err
		}
	}
	out := adapterResult{
		result:       result,
		page:         page,
		completeness: codehostbroker.CompletenessComplete,
	}
	if len(result.PullRequests) > 0 {
		out.pullRequest = &result.PullRequests[len(result.PullRequests)-1]
	}
	return out, nil
}

func partialCollection(result codehostbroker.PullRequestsResult, request codehostbroker.Request, scope []codehostbroker.RepositoryIdentity, limit int, items []codehostbroker.PullRequest, position cursorPosition, cause error) (adapterResult, error) {
	page := &codehostbroker.Page{Limit: limit, Count: len(items)}
	page.NextCursor, _ = encodePosition(request, scope, position)
	return adapterResult{
		result:       result,
		page:         page,
		completeness: codehostbroker.CompletenessPartial,
	}, cause
}

func normalizedLimit(limit int) int {
	if limit <= 0 {
		return 30
	}
	return min(limit, codehostbroker.MaxPageSize)
}

func parseQuery(query string) (state string, terms []string) {
	state = "open"
	for _, term := range strings.Fields(query) {
		switch strings.ToLower(term) {
		case "state:open", "is:open":
			state = "open"
		case "state:closed", "is:closed":
			state = "closed"
		case "state:all":
			state = "all"
		default:
			terms = append(terms, strings.ToLower(term))
		}
	}
	return state, terms
}

func normalizeOrder(order string) (sortValue, direction string, err error) {
	switch strings.ToLower(strings.TrimSpace(order)) {
	case "", "updated_desc":
		return "updated", "desc", nil
	case "updated_asc":
		return "updated", "asc", nil
	case "created_desc":
		return "created", "desc", nil
	case "created_asc":
		return "created", "asc", nil
	default:
		return "", "", &providerError{code: codehostbroker.ErrorInvalidInput, message: "order must be updated_desc, updated_asc, created_desc, or created_asc"}
	}
}

func pullListPath(repository codehostbroker.RepositoryIdentity, state, sortValue, direction string, limit, page int) string {
	query := url.Values{}
	query.Set("state", state)
	query.Set("sort", sortValue)
	query.Set("direction", direction)
	query.Set("per_page", strconv.Itoa(min(limit, codehostbroker.MaxPageSize)))
	query.Set("page", strconv.Itoa(page))
	return fmt.Sprintf("repos/%s/%s/pulls?%s", url.PathEscape(repository.Owner), url.PathEscape(repository.Name), query.Encode())
}

func matchesTerms(pullRequest codehostbroker.PullRequest, terms []string) bool {
	haystack := strings.ToLower(pullRequest.Title + "\n" + pullRequest.Body)
	for _, term := range terms {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func (g *githubAdapter) getPullRequest(ctx context.Context, request codehostbroker.Request) (adapterResult, error) {
	pullRequest, err := g.fetchPullRequest(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return adapterResult{}, err
	}
	return adapterResult{result: pullRequest, pullRequest: &pullRequest, completeness: codehostbroker.CompletenessComplete}, nil
}

func (g *githubAdapter) fetchPullRequest(ctx context.Context, repository codehostbroker.RepositoryIdentity, number int64) (codehostbroker.PullRequest, error) {
	pullRequest, _, err := g.fetchPullRequestWithProviderState(ctx, repository, number)
	return pullRequest, err
}

func (g *githubAdapter) fetchPullRequestWithProviderState(ctx context.Context, repository codehostbroker.RepositoryIdentity, number int64) (codehostbroker.PullRequest, githubPullRequest, error) {
	var wire githubPullRequest
	path := fmt.Sprintf("repos/%s/%s/pulls/%d", url.PathEscape(repository.Owner), url.PathEscape(repository.Name), number)
	if _, err := g.transport.get(ctx, path, &wire); err != nil {
		return codehostbroker.PullRequest{}, githubPullRequest{}, err
	}
	pullRequest := normalizePullRequest(g.connection.ID, wire)
	setPullRequestHost(&pullRequest, repository.Host)
	if err := validateNormalizedPullRequest(pullRequest); err != nil {
		return codehostbroker.PullRequest{}, githubPullRequest{}, err
	}
	return pullRequest, wire, nil
}

func setPullRequestHost(pullRequest *codehostbroker.PullRequest, host string) {
	pullRequest.Identity.Repository.Host = host
	pullRequest.Base.Repository.Host = host
	pullRequest.Head.Repository.Host = host
}

func validateNormalizedPullRequest(pullRequest codehostbroker.PullRequest) error {
	if pullRequest.Identity.ConnectionID == "" || pullRequest.Identity.ProviderID == "" ||
		pullRequest.Identity.Number <= 0 || pullRequest.Title == "" || pullRequest.URL == "" ||
		pullRequest.State == "" || pullRequest.Author.Login == "" ||
		pullRequest.Base.Name == "" || pullRequest.Base.SHA == "" ||
		pullRequest.Head.Name == "" || pullRequest.Head.SHA == "" ||
		pullRequest.Base.Repository.FullName == "" || pullRequest.Head.Repository.FullName == "" {
		return &providerError{code: codehostbroker.ErrorProvider, message: "GitHub returned an incomplete pull request"}
	}
	return nil
}

func (g *githubAdapter) getCommits(ctx context.Context, request codehostbroker.Request) (adapterResult, error) {
	pullRequest, err := g.fetchPullRequest(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return adapterResult{}, err
	}
	position, err := decodePosition(request)
	if err != nil {
		return adapterResult{}, err
	}
	limit := normalizedLimit(request.Limit)
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/commits?per_page=%d&page=%d",
		url.PathEscape(request.Repository.Owner), url.PathEscape(request.Repository.Name), request.PullRequest.Number, limit, position.Page)
	var wire []githubCommit
	header, err := g.transport.get(ctx, path, &wire)
	if err != nil {
		return adapterResult{}, err
	}
	next, err := g.transport.validateNextLink(header)
	if err != nil {
		return adapterResult{}, err
	}
	commits := make([]codehostbroker.Commit, 0, min(len(wire), limit))
	for _, item := range wire {
		if len(commits) >= limit {
			break
		}
		commits = append(commits, normalizeCommit(item))
	}
	out := adapterResult{
		result:       codehostbroker.CommitsResult{Commits: commits},
		pullRequest:  &pullRequest,
		completeness: codehostbroker.CompletenessComplete,
		page:         &codehostbroker.Page{Limit: limit, Count: len(commits)},
	}
	if next != "" {
		out.page.NextCursor, err = encodePosition(request, []codehostbroker.RepositoryIdentity{request.Repository}, cursorPosition{Page: position.Page + 1})
		if err != nil {
			return adapterResult{}, err
		}
		out.completeness = codehostbroker.CompletenessTruncated
		out.truncated = true
	}
	return g.rejectStaleSection(ctx, request, pullRequest, out)
}

func (g *githubAdapter) getDiff(ctx context.Context, request codehostbroker.Request) (adapterResult, error) {
	pullRequest, err := g.fetchPullRequest(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return adapterResult{}, err
	}
	fileLimit := codehostbroker.MaxDiffFiles
	if request.Limit > 0 {
		fileLimit = min(request.Limit, fileLimit)
	}
	files := make([]codehostbroker.DiffFile, 0, min(fileLimit, codehostbroker.MaxPageSize))
	page := 1
	totalHunks := 0
	totalBytes := 0
	truncated := false
	for len(files) < fileLimit {
		if err := ctx.Err(); err != nil {
			if len(files) == 0 {
				return adapterResult{}, err
			}
			out := adapterResult{
				result:          codehostbroker.DiffResult{Files: files},
				pullRequest:     &pullRequest,
				completeness:    codehostbroker.CompletenessPartial,
				truncated:       true,
				partialFailures: []codehostbroker.PartialFailure{{Section: "diff", Code: codehostbroker.ErrorCancelled, Message: "diff read was cancelled"}},
			}
			return out, nil
		}
		perPage := min(codehostbroker.MaxPageSize, fileLimit-len(files))
		path := fmt.Sprintf("repos/%s/%s/pulls/%d/files?per_page=%d&page=%d",
			url.PathEscape(request.Repository.Owner), url.PathEscape(request.Repository.Name), request.PullRequest.Number, perPage, page)
		var wire []githubDiffFile
		header, fetchErr := g.transport.get(ctx, path, &wire)
		if fetchErr != nil {
			if len(files) == 0 {
				return adapterResult{}, fetchErr
			}
			return adapterResult{
				result:       codehostbroker.DiffResult{Files: files},
				pullRequest:  &pullRequest,
				completeness: codehostbroker.CompletenessPartial,
				truncated:    true,
			}, fetchErr
		}
		next, nextErr := g.transport.validateNextLink(header)
		if nextErr != nil {
			if len(files) == 0 {
				return adapterResult{}, nextErr
			}
			return adapterResult{
				result:       codehostbroker.DiffResult{Files: files},
				pullRequest:  &pullRequest,
				completeness: codehostbroker.CompletenessPartial,
				truncated:    true,
			}, nextErr
		}
		for _, item := range wire {
			if len(files) >= fileLimit {
				truncated = true
				break
			}
			file, fileTruncated := normalizeDiffFile(item, &totalHunks, &totalBytes)
			files = append(files, file)
			truncated = truncated || fileTruncated
			if totalHunks >= codehostbroker.MaxDiffHunks || totalBytes >= codehostbroker.MaxDiffBytes-(64<<10) {
				truncated = true
				break
			}
		}
		if next == "" {
			break
		}
		if len(files) >= fileLimit || totalHunks >= codehostbroker.MaxDiffHunks || totalBytes >= codehostbroker.MaxDiffBytes-(64<<10) {
			truncated = true
			break
		}
		page++
	}
	out := adapterResult{
		result:       codehostbroker.DiffResult{Files: files},
		pullRequest:  &pullRequest,
		completeness: codehostbroker.CompletenessComplete,
		truncated:    truncated,
	}
	if truncated {
		out.completeness = codehostbroker.CompletenessTruncated
	}
	return g.rejectStaleSection(ctx, request, pullRequest, out)
}

func normalizeDiffFile(item githubDiffFile, totalHunks, totalBytes *int) (codehostbroker.DiffFile, bool) {
	file := codehostbroker.DiffFile{
		Path:                boundedText(item.Filename, 4096),
		Status:              boundedText(item.Status, 64),
		Additions:           max(item.Additions, 0),
		Deletions:           max(item.Deletions, 0),
		Hunks:               []codehostbroker.DiffHunk{},
		ContentAvailability: codehostbroker.DiffContentUnknown,
	}
	if item.Patch == nil {
		file.ContentAvailability = codehostbroker.DiffContentProviderOmitted
		return file, false
	}
	if *item.Patch != "" {
		file.ContentAvailability = codehostbroker.DiffContentText
	}
	truncated := false
	for _, hunk := range splitPatch(*item.Patch) {
		if *totalHunks >= codehostbroker.MaxDiffHunks {
			truncated = true
			break
		}
		remaining := codehostbroker.MaxDiffBytes - (64 << 10) - *totalBytes
		if remaining <= len(hunk.Header)+2 {
			truncated = true
			break
		}
		patch := hunk.Patch
		if len(patch) > remaining-len(hunk.Header) {
			patch = boundedText(patch, remaining-len(hunk.Header))
			truncated = true
		}
		file.Hunks = append(file.Hunks, codehostbroker.DiffHunk{Header: hunk.Header, Patch: patch})
		(*totalHunks)++
		*totalBytes += len(hunk.Header) + len(patch)
		if truncated {
			break
		}
	}
	file.Truncated = truncated
	if truncated {
		file.ContentAvailability = codehostbroker.DiffContentTruncated
	}
	return file, truncated
}

func splitPatch(patch string) []codehostbroker.DiffHunk {
	if patch == "" {
		return nil
	}
	lines := strings.SplitAfter(patch, "\n")
	hunks := []codehostbroker.DiffHunk{}
	var current *codehostbroker.DiffHunk
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			if current != nil {
				hunks = append(hunks, *current)
			}
			header := strings.TrimSuffix(line, "\n")
			current = &codehostbroker.DiffHunk{Header: header}
			continue
		}
		if current == nil {
			current = &codehostbroker.DiffHunk{Header: "@@ patch @@"}
		}
		current.Patch += line
	}
	if current != nil {
		hunks = append(hunks, *current)
	}
	return hunks
}

func (g *githubAdapter) getChecks(ctx context.Context, request codehostbroker.Request) (adapterResult, error) {
	pullRequest, err := g.fetchPullRequest(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return adapterResult{}, err
	}
	path := fmt.Sprintf("repos/%s/%s/commits/%s/check-runs?per_page=%d",
		url.PathEscape(request.Repository.Owner), url.PathEscape(request.Repository.Name), url.PathEscape(pullRequest.Head.SHA), codehostbroker.MaxItems)
	var wire githubCheckRuns
	if _, err := g.transport.get(ctx, path, &wire); err != nil {
		return adapterResult{}, err
	}
	checks := make([]codehostbroker.Check, 0, min(len(wire.CheckRuns), codehostbroker.MaxItems))
	for _, item := range wire.CheckRuns {
		if len(checks) == codehostbroker.MaxItems {
			break
		}
		checks = append(checks, normalizeCheck(item))
	}
	out := adapterResult{
		result:       codehostbroker.ChecksResult{Checks: checks},
		pullRequest:  &pullRequest,
		completeness: codehostbroker.CompletenessComplete,
		truncated:    wire.TotalCount > len(checks),
	}
	if out.truncated {
		out.completeness = codehostbroker.CompletenessTruncated
	}
	return g.rejectStaleSection(ctx, request, pullRequest, out)
}

func (g *githubAdapter) getReviews(ctx context.Context, request codehostbroker.Request) (adapterResult, error) {
	pullRequest, err := g.fetchPullRequest(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return adapterResult{}, err
	}
	position, err := decodePosition(request)
	if err != nil {
		return adapterResult{}, err
	}
	limit := normalizedLimit(request.Limit)
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews?per_page=%d&page=%d",
		url.PathEscape(request.Repository.Owner), url.PathEscape(request.Repository.Name), request.PullRequest.Number, limit, position.Page)
	var wire []githubReview
	header, err := g.transport.get(ctx, path, &wire)
	if err != nil {
		return adapterResult{}, err
	}
	next, err := g.transport.validateNextLink(header)
	if err != nil {
		return adapterResult{}, err
	}
	reviews := make([]codehostbroker.Review, 0, min(len(wire), limit))
	for _, item := range wire {
		if item.CommitID != pullRequest.Head.SHA {
			continue
		}
		reviews = append(reviews, normalizeReview(item))
		if len(reviews) == limit {
			break
		}
	}
	out := adapterResult{
		result:       codehostbroker.ReviewsResult{Reviews: reviews},
		pullRequest:  &pullRequest,
		completeness: codehostbroker.CompletenessComplete,
		page:         &codehostbroker.Page{Limit: limit, Count: len(reviews)},
	}
	if next != "" {
		out.page.NextCursor, err = encodePosition(request, []codehostbroker.RepositoryIdentity{request.Repository}, cursorPosition{Page: position.Page + 1})
		if err != nil {
			return adapterResult{}, err
		}
		out.completeness = codehostbroker.CompletenessPartial
	}
	return g.rejectStaleSection(ctx, request, pullRequest, out)
}

func (g *githubAdapter) getComments(ctx context.Context, request codehostbroker.Request) (adapterResult, error) {
	pullRequest, err := g.fetchPullRequest(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return adapterResult{}, err
	}
	position, err := decodePosition(request)
	if err != nil {
		return adapterResult{}, err
	}
	limit := normalizedLimit(request.Limit)
	path := fmt.Sprintf("repos/%s/%s/issues/%d/comments?per_page=%d&page=%d",
		url.PathEscape(request.Repository.Owner), url.PathEscape(request.Repository.Name), request.PullRequest.Number, limit, position.Page)
	var wire []githubComment
	header, err := g.transport.get(ctx, path, &wire)
	if err != nil {
		return adapterResult{}, err
	}
	next, err := g.transport.validateNextLink(header)
	if err != nil {
		return adapterResult{}, err
	}
	comments := make([]codehostbroker.Comment, 0, min(len(wire), limit))
	for _, item := range wire {
		comments = append(comments, normalizeComment(item))
		if len(comments) == limit {
			break
		}
	}
	out := adapterResult{
		result:       codehostbroker.CommentsResult{Comments: comments},
		pullRequest:  &pullRequest,
		completeness: codehostbroker.CompletenessComplete,
		page:         &codehostbroker.Page{Limit: limit, Count: len(comments)},
	}
	if next != "" {
		out.page.NextCursor, err = encodePosition(request, []codehostbroker.RepositoryIdentity{request.Repository}, cursorPosition{Page: position.Page + 1})
		if err != nil {
			return adapterResult{}, err
		}
		out.completeness = codehostbroker.CompletenessPartial
	}
	return g.rejectStaleSection(ctx, request, pullRequest, out)
}

func (g *githubAdapter) rejectStaleSection(ctx context.Context, request codehostbroker.Request, observed codehostbroker.PullRequest, out adapterResult) (adapterResult, error) {
	current, err := g.fetchPullRequest(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			out.completeness = codehostbroker.CompletenessPartial
			out.partialFailures = append(out.partialFailures, codehostbroker.PartialFailure{
				Section: string(request.Operation), Code: codehostbroker.ErrorCancelled, Message: "freshness recheck was cancelled",
			})
			return out, nil
		}
		return out, err
	}
	headMatches := current.Head.SHA == observed.Head.SHA
	baseMatches := request.Operation != codehostbroker.OperationGetMergeReadiness || sameRefIdentity(current.Base, observed.Base)
	if headMatches && baseMatches {
		return out, nil
	}
	out.pullRequest = &current
	out.completeness = codehostbroker.CompletenessPartial
	out.truncated = true
	out.partialFailures = append(out.partialFailures, codehostbroker.PartialFailure{
		Section: string(request.Operation), Code: codehostbroker.ErrorStaleObservation,
		Message: "pull request head or base changed during the read; section data was discarded",
	})
	switch request.Operation {
	case codehostbroker.OperationGetCommits:
		out.result = codehostbroker.CommitsResult{Commits: []codehostbroker.Commit{}}
	case codehostbroker.OperationGetDiff:
		out.result = codehostbroker.DiffResult{Files: []codehostbroker.DiffFile{}}
	case codehostbroker.OperationGetChecks:
		out.result = codehostbroker.ChecksResult{Checks: []codehostbroker.Check{}}
	case codehostbroker.OperationGetReviews:
		out.result = codehostbroker.ReviewsResult{Reviews: []codehostbroker.Review{}}
	case codehostbroker.OperationGetComments:
		out.result = codehostbroker.CommentsResult{Comments: []codehostbroker.Comment{}}
	case codehostbroker.OperationGetMergeReadiness:
		out.result = unavailableReadiness("pull request head changed during the read")
	}
	if out.page != nil {
		out.page.Count = 0
		out.page.NextCursor = ""
	}
	return out, nil
}

const mergeReadinessQuery = `query HeroMergeReadiness($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      id
      headRefOid
      baseRefOid
      isDraft
      mergeable
      mergeStateStatus
      reviewDecision
      viewerCanMerge
      mergeQueue { id }
      mergeQueueEntry { id }
      baseRef { branchProtectionRule { requiresApprovingReviews requiredApprovingReviewCount requiresStatusChecks } }
      commits(last: 1) { nodes { commit { statusCheckRollup { state } } } }
    }
  }
  rateLimit { limit remaining resetAt }
}`

func (g *githubAdapter) getMergeReadiness(ctx context.Context, request codehostbroker.Request) (adapterResult, error) {
	pullRequest, err := g.fetchPullRequest(ctx, request.Repository, request.PullRequest.Number)
	if err != nil {
		return adapterResult{}, err
	}
	var wire githubGraphQLResponse
	_, err = g.transport.graphql(ctx, mergeReadinessQuery, map[string]any{
		"owner": request.Repository.Owner, "name": request.Repository.Name, "number": request.PullRequest.Number,
	}, &wire)
	if err != nil {
		return adapterResult{}, err
	}
	g.transport.observeGraphQLRateLimit(wire.Data.RateLimit)
	graphQLRateLimit := g.transport.rateLimit
	if graphHead := wire.Data.Repository.PullRequest.HeadRefOID; graphHead != "" && graphHead != pullRequest.Head.SHA {
		return adapterResult{
			result:       unavailableReadiness("GitHub readiness data was observed for a different pull request head"),
			pullRequest:  &pullRequest,
			completeness: codehostbroker.CompletenessPartial,
			truncated:    true,
			rateLimit:    graphQLRateLimit,
			partialFailures: []codehostbroker.PartialFailure{{
				Section: "merge_readiness", Code: codehostbroker.ErrorStaleObservation,
				Message: "GitHub readiness data was observed for a different pull request head",
			}},
		}, nil
	}
	readiness, failures := normalizeMergeReadiness(wire)
	out := adapterResult{
		result:          readiness,
		pullRequest:     &pullRequest,
		completeness:    codehostbroker.CompletenessComplete,
		partialFailures: failures,
		rateLimit:       graphQLRateLimit,
		observedHeadSHA: wire.Data.Repository.PullRequest.HeadRefOID,
		observedBaseSHA: wire.Data.Repository.PullRequest.BaseRefOID,
	}
	if wire.Data.Repository.PullRequest.MergeQueueEntry != nil {
		out.queueID = wire.Data.Repository.PullRequest.MergeQueueEntry.ID
	}
	if wire.Data.Repository.PullRequest.MergeQueue != nil {
		out.queueRequired = true
	}
	if len(failures) > 0 {
		out.completeness = codehostbroker.CompletenessPartial
	}
	return g.rejectStaleSection(ctx, request, pullRequest, out)
}

func normalizeMergeReadiness(response githubGraphQLResponse) (codehostbroker.MergeReadiness, []codehostbroker.PartialFailure) {
	pullRequest := response.Data.Repository.PullRequest
	readiness := codehostbroker.MergeReadiness{
		State:            "unknown",
		Checks:           codehostbroker.AvailabilityUnknown,
		Reviews:          codehostbroker.AvailabilityUnknown,
		BranchProtection: codehostbroker.AvailabilityAvailable,
		Permissions:      codehostbroker.AvailabilityAvailable,
		Mergeability:     codehostbroker.AvailabilityUnknown,
		Queue:            codehostbroker.AvailabilityAvailable,
		Reasons:          []string{},
	}
	failures := make([]codehostbroker.PartialFailure, 0, len(response.Errors))
	for _, graphError := range response.Errors {
		section := graphErrorSection(graphError.Path)
		code := codehostbroker.ErrorPartialFailure
		if strings.EqualFold(graphError.Type, "FORBIDDEN") {
			code = codehostbroker.ErrorForbidden
		} else if strings.Contains(strings.ToLower(graphError.Message), "rate limit") {
			code = codehostbroker.ErrorRateLimited
		}
		failures = append(failures, codehostbroker.PartialFailure{
			Section: section,
			Code:    code,
			Message: "GitHub could not provide the " + section + " section",
		})
		setReadinessUnavailable(&readiness, section)
	}
	if pullRequest.ID == "" {
		readiness.State = "unavailable"
		readiness.Reasons = append(readiness.Reasons, "pull request readiness data is unavailable")
		return readiness, failures
	}

	switch strings.ToUpper(pullRequest.Mergeable) {
	case "MERGEABLE":
		readiness.Mergeability = codehostbroker.AvailabilityAvailable
	case "CONFLICTING":
		readiness.Mergeability = codehostbroker.AvailabilityAvailable
		readiness.Reasons = append(readiness.Reasons, "GitHub reports merge conflicts")
	case "UNKNOWN", "":
		if readiness.Mergeability != codehostbroker.AvailabilityUnavailable {
			readiness.Mergeability = codehostbroker.AvailabilityUnknown
		}
		readiness.Reasons = append(readiness.Reasons, "GitHub mergeability is pending")
	}
	if pullRequest.ViewerCanMerge != nil && readiness.Permissions != codehostbroker.AvailabilityUnavailable {
		readiness.Permissions = codehostbroker.AvailabilityAvailable
		if !*pullRequest.ViewerCanMerge {
			readiness.Reasons = append(readiness.Reasons, "the selected credential cannot merge this pull request")
		}
	} else if readiness.Permissions != codehostbroker.AvailabilityUnavailable {
		readiness.Permissions = codehostbroker.AvailabilityUnknown
	}
	if pullRequest.MergeQueueEntry != nil && readiness.Queue != codehostbroker.AvailabilityUnavailable {
		readiness.Queue = codehostbroker.AvailabilityAvailable
		readiness.Reasons = append(readiness.Reasons, "pull request is in the merge queue")
	}
	if pullRequest.MergeQueue != nil && readiness.Queue != codehostbroker.AvailabilityUnavailable {
		readiness.Queue = codehostbroker.AvailabilityAvailable
		readiness.Reasons = append(readiness.Reasons, "base branch requires the merge queue")
	}
	if readiness.BranchProtection != codehostbroker.AvailabilityUnavailable {
		readiness.BranchProtection = codehostbroker.AvailabilityAvailable
	}
	checkState := strings.ToUpper(pullRequest.CheckState())
	if readiness.Checks != codehostbroker.AvailabilityUnavailable {
		switch checkState {
		case "SUCCESS":
			readiness.Checks = codehostbroker.AvailabilityAvailable
		case "FAILURE", "ERROR":
			readiness.Checks = codehostbroker.AvailabilityAvailable
			readiness.Reasons = append(readiness.Reasons, "required checks are not successful")
		case "PENDING", "EXPECTED":
			readiness.Checks = codehostbroker.AvailabilityUnknown
			readiness.Reasons = append(readiness.Reasons, "required checks are pending")
		default:
			if pullRequest.BaseRef.BranchProtectionRule != nil && pullRequest.BaseRef.BranchProtectionRule.RequiresStatusChecks {
				readiness.Checks = codehostbroker.AvailabilityUnknown
				readiness.Reasons = append(readiness.Reasons, "required check evidence is unavailable")
			} else {
				readiness.Checks = codehostbroker.AvailabilityAvailable
			}
		}
	}
	if readiness.Reviews != codehostbroker.AvailabilityUnavailable {
		switch strings.ToUpper(pullRequest.ReviewDecision) {
		case "APPROVED":
			readiness.Reviews = codehostbroker.AvailabilityAvailable
		case "CHANGES_REQUESTED":
			readiness.Reviews = codehostbroker.AvailabilityAvailable
			readiness.Reasons = append(readiness.Reasons, "changes are requested")
		case "REVIEW_REQUIRED":
			readiness.Reviews = codehostbroker.AvailabilityAvailable
			readiness.Reasons = append(readiness.Reasons, "required reviews are missing")
		default:
			if pullRequest.BaseRef.BranchProtectionRule != nil && pullRequest.BaseRef.BranchProtectionRule.RequiresApprovingReviews {
				readiness.Reviews = codehostbroker.AvailabilityUnknown
				readiness.Reasons = append(readiness.Reasons, "required review evidence is unavailable")
			} else {
				readiness.Reviews = codehostbroker.AvailabilityAvailable
			}
		}
	}
	if pullRequest.IsDraft {
		readiness.Reasons = append(readiness.Reasons, "pull request is a draft")
	}
	switch strings.ToUpper(pullRequest.MergeStateStatus) {
	case "BLOCKED", "DIRTY", "BEHIND", "UNSTABLE":
		readiness.Reasons = append(readiness.Reasons, "GitHub merge state is "+strings.ToLower(pullRequest.MergeStateStatus))
	case "UNKNOWN", "":
		if readiness.Mergeability == codehostbroker.AvailabilityAvailable {
			readiness.Mergeability = codehostbroker.AvailabilityUnknown
		}
		readiness.Reasons = append(readiness.Reasons, "GitHub merge state is pending")
	}

	switch {
	case readiness.Mergeability == codehostbroker.AvailabilityUnavailable ||
		readiness.Permissions == codehostbroker.AvailabilityUnavailable ||
		readiness.Checks == codehostbroker.AvailabilityUnavailable ||
		readiness.Reviews == codehostbroker.AvailabilityUnavailable ||
		readiness.BranchProtection == codehostbroker.AvailabilityUnavailable ||
		readiness.Queue == codehostbroker.AvailabilityUnavailable:
		readiness.State = "unavailable"
	case readiness.Mergeability == codehostbroker.AvailabilityUnknown ||
		readiness.Checks == codehostbroker.AvailabilityUnknown ||
		readiness.Reviews == codehostbroker.AvailabilityUnknown ||
		readiness.BranchProtection == codehostbroker.AvailabilityUnknown ||
		readiness.Permissions == codehostbroker.AvailabilityUnknown:
		readiness.State = "unknown"
	case pullRequest.IsDraft ||
		strings.EqualFold(pullRequest.Mergeable, "CONFLICTING") ||
		containsAnyFold(readiness.Reasons, "required checks are not successful", "changes are requested", "required reviews are missing", "cannot merge", "requires the merge queue", "merge state is blocked", "merge state is dirty", "merge state is behind", "merge state is unstable"):
		readiness.State = "blocked"
	case strings.EqualFold(pullRequest.Mergeable, "MERGEABLE") &&
		pullRequest.ViewerCanMerge != nil && *pullRequest.ViewerCanMerge:
		readiness.State = "ready"
	default:
		readiness.State = "unknown"
	}
	return readiness, failures
}

func graphErrorSection(path []any) string {
	joined := strings.ToLower(fmt.Sprint(path))
	switch {
	case strings.Contains(joined, "statuscheck"), strings.Contains(joined, "commit"):
		return "checks"
	case strings.Contains(joined, "review"):
		return "reviews"
	case strings.Contains(joined, "branchprotection"), strings.Contains(joined, "baseref"):
		return "branch_protection"
	case strings.Contains(joined, "viewercanmerge"):
		return "permissions"
	case strings.Contains(joined, "queue"):
		return "queue"
	default:
		return "mergeability"
	}
}

func setReadinessUnavailable(readiness *codehostbroker.MergeReadiness, section string) {
	switch section {
	case "checks":
		readiness.Checks = codehostbroker.AvailabilityUnavailable
	case "reviews":
		readiness.Reviews = codehostbroker.AvailabilityUnavailable
	case "branch_protection":
		readiness.BranchProtection = codehostbroker.AvailabilityUnavailable
	case "permissions":
		readiness.Permissions = codehostbroker.AvailabilityUnavailable
	case "queue":
		readiness.Queue = codehostbroker.AvailabilityUnavailable
	default:
		readiness.Mergeability = codehostbroker.AvailabilityUnavailable
	}
}

func unavailableReadiness(reason string) codehostbroker.MergeReadiness {
	return codehostbroker.MergeReadiness{
		State:            "unavailable",
		Checks:           codehostbroker.AvailabilityUnavailable,
		Reviews:          codehostbroker.AvailabilityUnavailable,
		BranchProtection: codehostbroker.AvailabilityUnavailable,
		Permissions:      codehostbroker.AvailabilityUnavailable,
		Mergeability:     codehostbroker.AvailabilityUnavailable,
		Queue:            codehostbroker.AvailabilityUnavailable,
		Reasons:          []string{reason},
	}
}

func containsAnyFold(values []string, candidates ...string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, candidate := range candidates {
			if strings.Contains(lower, strings.ToLower(candidate)) {
				return true
			}
		}
	}
	return false
}
