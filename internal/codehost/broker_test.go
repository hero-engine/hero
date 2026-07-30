package codehost

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/mockcodehost"
)

const credentialCanary = "CODEHOST-CREDENTIAL-CANARY"

func TestCapabilitiesAdvertiseExactlyImplementedOperations(t *testing.T) {
	fixture, server, broker := testBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	response := broker.Execute(context.Background(), fixture.request(codehostbroker.OperationCapabilities))
	requireValidResponse(t, response)
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	var result codehostbroker.CapabilitiesResult
	decodeResult(t, response, &result)
	if len(result.Capabilities) != len(availableOperations) {
		t.Fatalf("capabilities=%d", len(result.Capabilities))
	}
	for index, capability := range result.Capabilities {
		if !capability.Available || capability.Policy.Operation != availableOperations[index] {
			t.Fatalf("capability[%d]=%+v", index, capability)
		}
		if capability.Policy.Operation == codehostbroker.OperationMerge &&
			(capability.Merge == nil || len(capability.Merge.Methods) != 3 || capability.Merge.Revision == "") {
			t.Fatalf("merge capability=%+v", capability.Merge)
		}
		want, _ := codehostbroker.Policy(capability.Policy.Operation)
		if capability.Policy != want {
			t.Fatalf("capability policy drift: got=%+v want=%+v", capability.Policy, want)
		}
	}
	if server.RequestCount() != 2 {
		t.Fatalf("capability discovery made %d provider requests", server.RequestCount())
	}
}

func TestListAndSearchUseConfiguredRepositoryScopeAndOpaqueCursor(t *testing.T) {
	fixture, server, broker := testBroker(t, mockcodehost.PaginationScenario(), "acme/widgets", "acme/gadgets")
	request := fixture.request(codehostbroker.OperationListPullRequests)
	request.Repositories = []codehostbroker.RepositoryIdentity{fixture.repository("acme/gadgets")}
	request.Limit = 2
	response := broker.Execute(context.Background(), request)
	requireValidResponse(t, response)
	if response.Error != nil || response.Page == nil || response.Page.NextCursor == "" {
		t.Fatalf("response error=%v page=%+v", response.Error, response.Page)
	}
	var result codehostbroker.PullRequestsResult
	decodeResult(t, response, &result)
	if len(result.PullRequests) != 2 {
		t.Fatalf("pull requests=%d", len(result.PullRequests))
	}
	for _, pullRequest := range result.PullRequests {
		if pullRequest.Identity.Repository.FullName != "acme/gadgets" &&
			pullRequest.Identity.Repository.FullName != "acme/widgets" {
			t.Fatalf("unconfigured repository returned: %+v", pullRequest.Identity.Repository)
		}
	}
	envelope, cursorErr := codehostbroker.DecodeCursor(response.Page.NextCursor)
	if cursorErr != nil || envelope.Material.Position == "" {
		t.Fatalf("cursor=%v error=%v", envelope, cursorErr)
	}

	count := server.RequestCount()
	mismatched := request
	mismatched.Cursor = response.Page.NextCursor
	mismatched.Query = "different"
	rejected := broker.Execute(context.Background(), mismatched)
	if rejected.Error == nil || rejected.Error.Code != codehostbroker.ErrorCursorMismatch {
		t.Fatalf("cursor mismatch response=%+v", rejected.Error)
	}
	if server.RequestCount() != count {
		t.Fatal("cursor mismatch reached the provider")
	}
	versionMismatch := request
	versionMismatch.Cursor = response.Page.NextCursor
	versionMismatch.Version = "code-host-broker/v2"
	rejected = broker.Execute(context.Background(), versionMismatch)
	if rejected.Error == nil || rejected.Error.Code != codehostbroker.ErrorCursorMismatch {
		t.Fatalf("cursor version mismatch response=%+v", rejected.Error)
	}
	if server.RequestCount() != count {
		t.Fatal("cursor version mismatch reached the provider")
	}

	search := fixture.request(codehostbroker.OperationSearchPullRequests)
	search.Query = "request 43"
	search.Limit = 10
	searched := broker.Execute(context.Background(), search)
	requireValidResponse(t, searched)
	if searched.Error != nil {
		t.Fatal(searched.Error)
	}
	decodeResult(t, searched, &result)
	if len(result.PullRequests) != 1 || result.PullRequests[0].Identity.Number != 43 {
		t.Fatalf("search results=%+v", result.PullRequests)
	}
}

func TestForkAndForcePushPreserveIdentityAndFreshness(t *testing.T) {
	forkFixture, _, forkBroker := testBroker(t, mockcodehost.ForkScenario(), "acme/widgets")
	forkResponse := forkBroker.Execute(context.Background(), forkFixture.request(codehostbroker.OperationGetPullRequest))
	requireValidResponse(t, forkResponse)
	var fork codehostbroker.PullRequest
	decodeResult(t, forkResponse, &fork)
	if fork.Base.Repository.FullName != "acme/widgets" ||
		fork.Head.Repository.FullName != "contributor/widgets" ||
		fork.Base.Repository == fork.Head.Repository ||
		fork.Head.SHA == fork.Base.SHA ||
		fork.Head.Name != "feature/code-host" {
		t.Fatalf("fork identity collapsed: %+v", fork)
	}

	forceFixture, _, forceBroker := testBroker(t, mockcodehost.ForcePushScenario(), "acme/widgets")
	first := forceBroker.Execute(context.Background(), forceFixture.request(codehostbroker.OperationGetPullRequest))
	second := forceBroker.Execute(context.Background(), forceFixture.request(codehostbroker.OperationGetPullRequest))
	requireValidResponse(t, first)
	requireValidResponse(t, second)
	var before, after codehostbroker.PullRequest
	decodeResult(t, first, &before)
	decodeResult(t, second, &after)
	if before.Head.SHA == after.Head.SHA || first.ObservationRevision == second.ObservationRevision {
		t.Fatalf("force push did not change freshness: before=%q after=%q revisions=%q/%q",
			before.Head.SHA, after.Head.SHA, first.ObservationRevision, second.ObservationRevision)
	}

	staleFixture, _, staleBroker := testBroker(t, mockcodehost.ForcePushScenario(), "acme/widgets")
	stale := staleBroker.Execute(context.Background(), staleFixture.request(codehostbroker.OperationGetDiff))
	requireValidResponse(t, stale)
	if stale.Error != nil || stale.Completeness != codehostbroker.CompletenessPartial ||
		len(stale.PartialFailures) != 1 || stale.PartialFailures[0].Code != codehostbroker.ErrorStaleObservation {
		t.Fatalf("stale diff response=%+v", stale)
	}
	var diff codehostbroker.DiffResult
	decodeResult(t, stale, &diff)
	if len(diff.Files) != 0 {
		t.Fatal("diff from the prior head was reported as current")
	}
	for _, operation := range []codehostbroker.Operation{
		codehostbroker.OperationGetChecks,
		codehostbroker.OperationGetReviews,
		codehostbroker.OperationGetMergeReadiness,
	} {
		t.Run("stale_"+string(operation), func(t *testing.T) {
			fixture, _, broker := testBroker(t, mockcodehost.ForcePushScenario(), "acme/widgets")
			response := broker.Execute(context.Background(), fixture.request(operation))
			requireValidResponse(t, response)
			if response.Error != nil || response.Completeness != codehostbroker.CompletenessPartial ||
				len(response.PartialFailures) != 1 ||
				response.PartialFailures[0].Code != codehostbroker.ErrorStaleObservation {
				t.Fatalf("stale %s response=%+v", operation, response)
			}
			switch operation {
			case codehostbroker.OperationGetChecks:
				var result codehostbroker.ChecksResult
				decodeResult(t, response, &result)
				if len(result.Checks) != 0 {
					t.Fatal("checks from the prior head were reported as current")
				}
			case codehostbroker.OperationGetReviews:
				var result codehostbroker.ReviewsResult
				decodeResult(t, response, &result)
				if len(result.Reviews) != 0 {
					t.Fatal("reviews from the prior head were reported as current")
				}
			case codehostbroker.OperationGetMergeReadiness:
				var result codehostbroker.MergeReadiness
				decodeResult(t, response, &result)
				if result.State != "unavailable" {
					t.Fatalf("readiness from the prior head was reported as %q", result.State)
				}
			}
		})
	}
}

func TestCommitAndDiffBoundsAreExplicit(t *testing.T) {
	fixture, _, broker := testBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	commitsRequest := fixture.request(codehostbroker.OperationGetCommits)
	commitsRequest.Limit = codehostbroker.MaxItems
	commits := broker.Execute(context.Background(), commitsRequest)
	requireValidResponse(t, commits)
	if commits.Error != nil || !commits.Truncated || commits.Completeness != codehostbroker.CompletenessTruncated ||
		commits.Page == nil || commits.Page.NextCursor == "" {
		t.Fatalf("commit bounds response=%+v", commits)
	}
	var commitResult codehostbroker.CommitsResult
	decodeResult(t, commits, &commitResult)
	if len(commitResult.Commits) != codehostbroker.MaxItems {
		t.Fatalf("commits=%d", len(commitResult.Commits))
	}

	diffFixture, _, diffBroker := testBroker(t, mockcodehost.OversizedDiffScenario(), "acme/widgets")
	diff := diffBroker.Execute(context.Background(), diffFixture.request(codehostbroker.OperationGetDiff))
	requireValidResponse(t, diff)
	if diff.Error != nil || !diff.Truncated || diff.Completeness != codehostbroker.CompletenessTruncated {
		t.Fatalf("diff bounds response error=%v truncated=%v completeness=%s", diff.Error, diff.Truncated, diff.Completeness)
	}
	var diffResult codehostbroker.DiffResult
	decodeResult(t, diff, &diffResult)
	if len(diffResult.Files) == 0 || len(diffResult.Files) > codehostbroker.MaxDiffFiles || len(diff.Result) > codehostbroker.MaxDiffBytes {
		t.Fatalf("diff files=%d bytes=%d", len(diffResult.Files), len(diff.Result))
	}
}

func TestPartialGraphQLAndMergeabilityNeverInventReadiness(t *testing.T) {
	fixture, _, broker := testBroker(t, mockcodehost.PartialFailureScenario(), "acme/widgets")
	partial := broker.Execute(context.Background(), fixture.request(codehostbroker.OperationGetMergeReadiness))
	requireValidResponse(t, partial)
	if partial.Error != nil || partial.Completeness != codehostbroker.CompletenessPartial ||
		len(partial.PartialFailures) != 1 || partial.PartialFailures[0].Section != "checks" {
		t.Fatalf("partial readiness=%+v", partial)
	}
	if partial.RateLimit.Resource != "graphql" || partial.RateLimit.Limit != 5000 ||
		partial.RateLimit.Remaining != 4998 || partial.RateLimit.ResetAt == "" ||
		partial.RateLimit.ObservedAt == "" {
		t.Fatalf("GraphQL rate limit=%+v", partial.RateLimit)
	}
	var readiness codehostbroker.MergeReadiness
	decodeResult(t, partial, &readiness)
	if readiness.Checks != codehostbroker.AvailabilityUnavailable ||
		(readiness.State != "unknown" && readiness.State != "unavailable") {
		t.Fatalf("partial readiness invented certainty: %+v", readiness)
	}

	changingFixture, _, changingBroker := testBroker(t, mockcodehost.ChangingMergeabilityScenario(), "acme/widgets")
	pending := changingBroker.Execute(context.Background(), changingFixture.request(codehostbroker.OperationGetMergeReadiness))
	ready := changingBroker.Execute(context.Background(), changingFixture.request(codehostbroker.OperationGetMergeReadiness))
	requireValidResponse(t, pending)
	requireValidResponse(t, ready)
	var pendingResult, readyResult codehostbroker.MergeReadiness
	decodeResult(t, pending, &pendingResult)
	decodeResult(t, ready, &readyResult)
	if pendingResult.State != "unknown" || readyResult.State != "ready" ||
		pending.ObservationRevision == ready.ObservationRevision {
		t.Fatalf("changing mergeability pending=%+v ready=%+v", pendingResult, readyResult)
	}
}

func TestRateLimitsPermissionsAndNormalizedErrors(t *testing.T) {
	rateFixture, _, rateBroker := testBroker(t, mockcodehost.RateLimitScenario(), "acme/widgets")
	rate := rateBroker.Execute(context.Background(), rateFixture.request(codehostbroker.OperationGetChecks))
	requireValidResponse(t, rate)
	if rate.Error == nil || rate.Error.Code != codehostbroker.ErrorRateLimited ||
		rate.Error.Retry != codehostbroker.RetryAfter || rate.RateLimit.Resource != "checks" ||
		rate.RateLimit.Limit != 5000 || rate.RateLimit.Remaining != 0 ||
		rate.RateLimit.RetryAfter != 17 || rate.RateLimit.ResetAt == "" || rate.RateLimit.ObservedAt == "" {
		t.Fatalf("rate response=%+v", rate)
	}

	permissionFixture, _, permissionBroker := testBroker(t, mockcodehost.PermissionsScenario(), "acme/widgets")
	permission := permissionBroker.Execute(context.Background(), permissionFixture.request(codehostbroker.OperationGetReviews))
	requireValidResponse(t, permission)
	if permission.Error == nil || permission.Error.Code != codehostbroker.ErrorForbidden {
		t.Fatalf("permission response=%+v", permission.Error)
	}
}

func TestCancellationBeforeAndDuringPagination(t *testing.T) {
	fixture, server, broker := testBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	response := broker.Execute(ctx, fixture.request(codehostbroker.OperationListPullRequests))
	requireValidResponse(t, response)
	if response.Error == nil || response.Error.Code != codehostbroker.ErrorCancelled || server.RequestCount() != 0 {
		t.Fatalf("pre-dispatch cancellation response=%+v requests=%d", response.Error, server.RequestCount())
	}

	fake := mockcodehost.NewServer(mockcodehost.PaginationScenario())
	readCtx, readCancel := context.WithCancel(context.Background())
	httpServer := httptest.NewServer(fake)
	t.Cleanup(httpServer.Close)
	duringFixture, duringBroker := testBrokerAtURL(t, httpServer.URL, "acme/widgets")
	var calls atomic.Int64
	duringBroker.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response, err := http.DefaultTransport.RoundTrip(request)
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		response.Body = io.NopCloser(bytes.NewReader(data))
		if calls.Add(1) == 1 {
			readCancel()
		}
		return response, nil
	})
	duringRequest := duringFixture.request(codehostbroker.OperationListPullRequests)
	duringRequest.Limit = 3
	during := duringBroker.Execute(readCtx, duringRequest)
	requireValidResponse(t, during)
	if during.Error != nil || during.Completeness != codehostbroker.CompletenessPartial ||
		len(during.PartialFailures) != 1 || during.PartialFailures[0].Code != codehostbroker.ErrorCancelled {
		t.Fatalf("during-read cancellation=%+v", during)
	}
	var result codehostbroker.PullRequestsResult
	decodeResult(t, during, &result)
	if len(result.PullRequests) != 1 || fake.RequestCount() != 1 {
		t.Fatalf("partial items=%d calls=%d", len(result.PullRequests), fake.RequestCount())
	}
}

func TestRepositoryScopeAndCrossOriginPaginationFailClosed(t *testing.T) {
	fixture, server, broker := testBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	outside := fixture.request(codehostbroker.OperationListPullRequests)
	outside.Repository = fixture.repository("other/private")
	rejected := broker.Execute(context.Background(), outside)
	requireValidResponse(t, rejected)
	if rejected.Error == nil || rejected.Error.Code != codehostbroker.ErrorForbidden || server.RequestCount() != 0 {
		t.Fatalf("scope rejection=%+v requests=%d", rejected.Error, server.RequestCount())
	}

	scenario := mockcodehost.PaginationScenario()
	scenario.CrossOriginNext = "https://attacker.invalid/steal"
	hostileFixture, hostileServer, hostileBroker := testBroker(t, scenario, "acme/widgets")
	hostile := hostileBroker.Execute(context.Background(), hostileFixture.request(codehostbroker.OperationListPullRequests))
	requireValidResponse(t, hostile)
	if hostile.Error == nil || hostile.Error.Code != codehostbroker.ErrorProvider || hostileServer.RequestCount() != 1 {
		t.Fatalf("hostile pagination=%+v requests=%d", hostile.Error, hostileServer.RequestCount())
	}
	for _, request := range hostileServer.Requests() {
		if request.Path == "/steal" {
			t.Fatal("cross-origin pagination target was requested")
		}
	}
}

func TestCrossOriginRedirectDoesNotForwardAuthorization(t *testing.T) {
	var targetCalls atomic.Int64
	target := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		targetCalls.Add(1)
		if request.Header.Get("Authorization") != "" {
			t.Error("authorization reached the redirect target")
		}
		writeTestJSON(writer, http.StatusOK, map[string]any{})
	}))
	t.Cleanup(target.Close)
	origin := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Location", target.URL+"/steal")
		writer.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(origin.Close)
	fixture, broker := testBrokerAtURL(t, origin.URL, "acme/widgets")
	response := broker.Execute(context.Background(), fixture.request(codehostbroker.OperationGetPullRequest))
	requireValidResponse(t, response)
	if response.Error == nil || response.Error.Code != codehostbroker.ErrorProvider || targetCalls.Load() != 0 {
		t.Fatalf("redirect response=%+v target calls=%d", response.Error, targetCalls.Load())
	}
	encoded, _ := json.Marshal(response)
	if strings.Contains(string(encoded), credentialCanary) {
		t.Fatalf("credential leaked through redirect error: %s", encoded)
	}
}

func TestEnterpriseOriginAndCredentialRedaction(t *testing.T) {
	fake := mockcodehost.NewServer(mockcodehost.DefaultScenario())
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	fixture, broker := testBrokerAtURL(t, server.URL+"/api/v3", "acme/widgets")
	response := broker.Execute(context.Background(), fixture.request(codehostbroker.OperationGetPullRequest))
	requireValidResponse(t, response)
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	var pullRequest codehostbroker.PullRequest
	decodeResult(t, response, &pullRequest)
	if pullRequest.Identity.Repository.Host != fixture.host || pullRequest.Head.Repository.Host != fixture.host {
		t.Fatalf("enterprise repository host=%+v head=%+v", pullRequest.Identity.Repository, pullRequest.Head.Repository)
	}

	leaking := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "provider echoed "+credentialCanary, http.StatusInternalServerError)
	}))
	t.Cleanup(leaking.Close)
	leakFixture, leakBroker := testBrokerAtURL(t, leaking.URL, "acme/widgets")
	leak := leakBroker.Execute(context.Background(), leakFixture.request(codehostbroker.OperationGetPullRequest))
	requireValidResponse(t, leak)
	rendered, _ := json.Marshal(leak)
	if strings.Contains(string(rendered), credentialCanary) {
		t.Fatalf("credential leaked in response: %s", rendered)
	}
}

func TestAuthenticatedActorReadIsConnectionQualifiedAndCredentialSafe(t *testing.T) {
	fixture, server, broker := testBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	response := broker.Execute(
		context.Background(),
		fixture.request(codehostbroker.OperationGetAuthenticatedActor),
	)
	requireValidResponse(t, response)
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	var result codehostbroker.AuthenticatedActorResult
	decodeResult(t, response, &result)
	if response.ConnectionID != fixture.connectionID ||
		response.Provider != "github" ||
		result.Actor.ProviderID != "U_99" ||
		result.Actor.Login != "hero-user" ||
		result.Actor.Display != "" {
		t.Fatalf("response identity=%s/%s actor=%+v", response.ConnectionID, response.Provider, result.Actor)
	}
	if server.RequestCount() != 1 || server.Requests()[0].Path != "/user" {
		t.Fatalf("provider requests=%+v", server.Requests())
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	rendered := strings.ToLower(string(encoded))
	for _, forbidden := range []string{"token", "email", "credential", strings.ToLower(credentialCanary)} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("authenticated actor response exposed %q: %s", forbidden, encoded)
		}
	}
}

func TestEveryReadOperationProducesContractConformantResponse(t *testing.T) {
	fixture, _, broker := testBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	for _, operation := range readOperations {
		t.Run(string(operation), func(t *testing.T) {
			response := broker.Execute(context.Background(), fixture.request(operation))
			requireValidResponse(t, response)
			if response.Error != nil {
				t.Fatalf("operation failed: %+v", response.Error)
			}
		})
	}
}

func TestAggregateResultBoundReturnsExplicitPrefix(t *testing.T) {
	repository := codehostbroker.RepositoryIdentity{
		Host: "github.com", ProviderID: "R_fixture", Owner: "acme", Name: "widgets", FullName: "acme/widgets",
	}
	items := make([]codehostbroker.PullRequest, 100)
	for index := range items {
		items[index] = codehostbroker.PullRequest{
			Identity: codehostbroker.PullRequestIdentity{
				ConnectionID: "github", Repository: repository, ProviderID: "PR", Number: int64(index + 1),
			},
			Title: strings.Repeat("title", 1000), Body: strings.Repeat("\\quoted\"", 10000),
			URL: "https://github.com/acme/widgets/pull/1", State: "open",
			Author: codehostbroker.Actor{Login: "author"},
			Base:   codehostbroker.RefIdentity{Repository: repository, Name: "main", SHA: baseSHAForTest},
			Head:   codehostbroker.RefIdentity{Repository: repository, Name: "feature", SHA: headSHAForTest},
		}
	}
	policy, _ := codehostbroker.Policy(codehostbroker.OperationListPullRequests)
	out := adapterResult{
		result:       codehostbroker.PullRequestsResult{PullRequests: items},
		page:         &codehostbroker.Page{Limit: 100, Count: 100},
		completeness: codehostbroker.CompletenessComplete,
	}
	bounded := boundAdapterResult(codehostbroker.OperationListPullRequests, out, policy.Bounds)
	if resultSize(bounded.result) > policy.Bounds.BodyBytes || !bounded.truncated ||
		bounded.completeness != codehostbroker.CompletenessTruncated ||
		bounded.page.Count <= 0 || bounded.page.Count >= len(items) {
		t.Fatalf("bounded size=%d truncated=%v completeness=%s count=%d",
			resultSize(bounded.result), bounded.truncated, bounded.completeness, bounded.page.Count)
	}
}

type brokerFixture struct {
	host         string
	connectionID string
	repositories []string
}

const (
	baseSHAForTest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	headSHAForTest = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func testBroker(t *testing.T, scenario mockcodehost.Scenario, repositories ...string) (brokerFixture, *mockcodehost.Server, *Broker) {
	t.Helper()
	fake := mockcodehost.NewServer(scenario)
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	fixture, broker := testBrokerAtURL(t, server.URL, repositories...)
	return fixture, fake, broker
}

func testBrokerAtURL(t *testing.T, baseURL string, repositories ...string) (brokerFixture, *Broker) {
	t.Helper()
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Hostname() == "127.0.0.1" {
		parsed.Host = "localhost:" + parsed.Port()
	}
	if len(repositories) == 0 {
		repositories = []string{"acme/widgets"}
	}
	rawProject, _ := json.Marshal(repositories[0])
	rawRepositories, _ := json.Marshal(repositories[1:])
	settings := map[string]json.RawMessage{"project": rawProject}
	if len(repositories) > 1 {
		settings["repositories"] = rawRepositories
	}
	cfg := config.Config{Integrations: &config.IntegrationsConfig{
		Roles: map[string]string{"code-host": "github-code-host"},
		Connections: map[string]config.IntegrationConfig{
			"github-code-host": {
				Provider:     "github",
				Capabilities: []config.IntegrationCapability{config.CapabilityCodeHost},
				Settings:     settings,
				Auth:         &config.IntegrationAuth{Token: config.Secret(credentialCanary)},
			},
		},
	}}
	connection := cfg.Integrations.Connections["github-code-host"]
	rawBaseURL, _ := json.Marshal(parsed.String())
	connection.Settings["base_url"] = rawBaseURL
	cfg.Integrations.Connections["github-code-host"] = connection

	broker := NewBroker("/unused")
	broker.loadConfig = func(string) (config.Config, error) { return cfg, nil }
	fixed := time.Date(2026, 7, 27, 20, 31, 0, 0, time.UTC)
	broker.now = func() time.Time { return fixed }
	return brokerFixture{
		host: parsed.Hostname(), connectionID: "github-code-host", repositories: repositories,
	}, broker
}

func (fixture brokerFixture) repository(fullName string) codehostbroker.RepositoryIdentity {
	owner, name, _ := strings.Cut(fullName, "/")
	return codehostbroker.RepositoryIdentity{
		Host: fixture.host, ProviderID: "R_" + strings.ReplaceAll(fullName, "/", "_"),
		Owner: owner, Name: name, FullName: fullName,
	}
}

func (fixture brokerFixture) request(operation codehostbroker.Operation) codehostbroker.Request {
	repository := fixture.repository(fixture.repositories[0])
	request := codehostbroker.Request{
		Version: codehostbroker.Version, Operation: operation, Provider: "github",
		ConnectionID: fixture.connectionID, Repository: repository,
	}
	if operation != codehostbroker.OperationCapabilities &&
		operation != codehostbroker.OperationListPullRequests &&
		operation != codehostbroker.OperationSearchPullRequests {
		request.PullRequest = &codehostbroker.PullRequestIdentity{
			ConnectionID: fixture.connectionID, Repository: repository, ProviderID: "PR_42", Number: 42,
		}
	}
	return request
}

func requireValidResponse(t *testing.T, response codehostbroker.Response) {
	t.Helper()
	if err := codehostbroker.ValidateResponse(response); err != nil {
		encoded, _ := json.MarshalIndent(response, "", "  ")
		t.Fatalf("contract-invalid response: %v\n%s", err, encoded)
	}
}

func decodeResult(t *testing.T, response codehostbroker.Response, target any) {
	t.Helper()
	if err := json.Unmarshal(response.Result, target); err != nil {
		t.Fatal(err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func writeTestJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func TestConfigurationFailureDoesNotLeakUnderlyingError(t *testing.T) {
	broker := NewBroker("/unused")
	broker.loadConfig = func(string) (config.Config, error) {
		return config.Config{}, errors.New("configuration includes " + credentialCanary)
	}
	fixture := brokerFixture{host: "github.com", connectionID: "github-code-host", repositories: []string{"acme/widgets"}}
	response := broker.Execute(context.Background(), fixture.request(codehostbroker.OperationCapabilities))
	requireValidResponse(t, response)
	encoded, _ := json.Marshal(response)
	if response.Error == nil || strings.Contains(string(encoded), credentialCanary) {
		t.Fatalf("configuration error leaked: %s", encoded)
	}
}
