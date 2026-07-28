package codehostbroker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestOperationRegistryIsCompleteAndAuthoritative(t *testing.T) {
	operations := Operations()
	if len(operations) != 20 {
		t.Fatalf("operation count=%d", len(operations))
	}
	seen := map[Operation]bool{}
	for _, operation := range operations {
		if seen[operation] {
			t.Fatalf("duplicate operation %q", operation)
		}
		seen[operation] = true
		policy, ok := Policy(operation)
		if !ok || policy.Operation != operation || policy.Bounds != defaultBounds {
			t.Fatalf("policy[%q]=%+v ok=%v", operation, policy, ok)
		}
		if IsRead(operation) {
			if policy.Effect != EffectRead || policy.Consent != ConsentNone || policy.RequiresIdempotency || policy.RequiresFreshObservation || !policy.ReplaySafe {
				t.Fatalf("read policy[%q]=%+v", operation, policy)
			}
			continue
		}
		if !policy.RequiresUniqueTarget || !policy.RequiresIdempotency || !policy.RequiresFreshObservation || !policy.RequiresReconciliation || !policy.ReplaySafe {
			t.Fatalf("mutation policy[%q]=%+v", operation, policy)
		}
		if operation == OperationMerge {
			if policy.Effect != EffectCommitment || policy.Consent != ConsentExplicitAcceptance {
				t.Fatalf("merge policy=%+v", policy)
			}
		} else if policy.Effect != EffectExternalWrite || policy.Consent != ConsentExplicitUser {
			t.Fatalf("write policy[%q]=%+v", operation, policy)
		}
	}
}

func TestRepositoryQualifiedPullRequestIdentity(t *testing.T) {
	identity := fixtureIdentity()
	if err := ValidatePullRequestIdentity(identity); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*PullRequestIdentity){
		"connection":  func(value *PullRequestIdentity) { value.ConnectionID = "" },
		"repository":  func(value *PullRequestIdentity) { value.Repository.FullName = "other/repo" },
		"provider id": func(value *PullRequestIdentity) { value.ProviderID = "" },
		"number":      func(value *PullRequestIdentity) { value.Number = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			value := identity
			mutate(&value)
			if err := ValidatePullRequestIdentity(value); err == nil {
				t.Fatal("unqualified identity accepted")
			}
		})
	}
}

func TestForkRefsRoundTripWithoutCollapse(t *testing.T) {
	base := fixtureRepository()
	head := RepositoryIdentity{Host: "github.com", ProviderID: "fork", Owner: "contributor", Name: "hero", FullName: "contributor/hero"}
	payload := CreatePullRequestPayload{
		Base:  RefIdentity{Repository: base, Name: "main", SHA: strings.Repeat("a", 40)},
		Head:  RefIdentity{Repository: head, Name: "feature", SHA: strings.Repeat("b", 40)},
		Title: "fixture",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var decoded CreatePullRequestPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Base.Repository == decoded.Head.Repository || decoded.Base.Name == decoded.Head.Name || decoded.Base.SHA == decoded.Head.SHA {
		t.Fatalf("fork collapsed: %+v", decoded)
	}
}

func TestCanonicalFixtureCoversAndValidatesEveryOperation(t *testing.T) {
	data, err := CanonicalFixture()
	if err != nil {
		t.Fatal(err)
	}
	var fixture ConsumerFixtureBundle
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Version != Version || len(fixture.Cases) != len(Operations()) || len(fixture.Operations) != len(Operations())+1 {
		t.Fatalf("fixture inventory: version=%q cases=%d operations=%d", fixture.Version, len(fixture.Cases), len(fixture.Operations))
	}
	seen := map[Operation]bool{}
	statuses := map[ReconciliationStatus]bool{}
	actorResults := map[Operation]bool{}
	for _, fixtureCase := range fixture.Cases {
		if fixtureCase.Name != string(fixtureCase.Request.Operation) || fixtureCase.Request.Operation != fixtureCase.Response.Operation {
			t.Fatalf("case identity mismatch: %+v", fixtureCase)
		}
		if err := ValidateRequest(fixtureCase.Request); err != nil {
			t.Fatalf("%s request: %v", fixtureCase.Name, err)
		}
		if err := ValidateResponse(fixtureCase.Response); err != nil {
			t.Fatalf("%s response: %v", fixtureCase.Name, err)
		}
		seen[fixtureCase.Request.Operation] = true
		if fixtureCase.Response.Reconciliation != nil {
			statuses[fixtureCase.Response.Reconciliation.Status] = true
		}
		if fixtureCase.Request.Operation == OperationComment ||
			fixtureCase.Request.Operation == OperationSubmitReview ||
			fixtureCase.Request.Operation == OperationApprove ||
			fixtureCase.Request.Operation == OperationRequestChanges {
			var result MutationResult
			if err := json.Unmarshal(fixtureCase.Response.Result, &result); err != nil {
				t.Fatalf("%s mutation result: %v", fixtureCase.Name, err)
			}
			if result.Actor == nil || result.Actor.ProviderID == "" || result.Actor.Login == "" {
				t.Fatalf("%s fixture lacks typed actor: %+v", fixtureCase.Name, result)
			}
			actorResults[fixtureCase.Request.Operation] = true
		}
	}
	if len(seen) != 20 || len(statuses) != 7 || len(actorResults) != 4 {
		t.Fatalf("coverage operations=%d reconciliation=%v actors=%v", len(seen), statuses, actorResults)
	}
	if len(fixture.Errors) != len(ErrorCodes()) || len(fixture.UnknownFields) == 0 {
		t.Fatalf("errors=%d unknown=%d", len(fixture.Errors), len(fixture.UnknownFields))
	}
}

func TestMutationRequestsRequirePolicyMaterial(t *testing.T) {
	fixture := mustFixture(t)
	for _, fixtureCase := range fixture.Cases {
		if !IsMutation(fixtureCase.Request.Operation) {
			continue
		}
		t.Run(string(fixtureCase.Request.Operation), func(t *testing.T) {
			if err := ValidateRequest(fixtureCase.Request); err != nil {
				t.Fatal(err)
			}
			for field, mutate := range map[string]func(*Request){
				"intent":         func(value *Request) { value.IntentSource = "" },
				"consent":        func(value *Request) { value.Consent = "" },
				"idempotency":    func(value *Request) { value.IdempotencyKey = "" },
				"capability":     func(value *Request) { value.CapabilityRevision = "" },
				"observation":    func(value *Request) { value.ObservationRevision = "" },
				"reconciliation": func(value *Request) { value.ReconciliationKey = "" },
			} {
				t.Run(field, func(t *testing.T) {
					value := fixtureCase.Request
					mutate(&value)
					if err := ValidateRequest(value); err == nil {
						t.Fatalf("missing %s accepted", field)
					}
				})
			}
		})
	}
}

func TestPartialResultAndBoundsTruth(t *testing.T) {
	fixture := mustFixture(t)
	var checks Response
	for _, fixtureCase := range fixture.Cases {
		if fixtureCase.Response.Operation == OperationGetChecks {
			checks = fixtureCase.Response
		}
	}
	if checks.Completeness != CompletenessPartial || len(checks.PartialFailures) != 1 || len(checks.Result) == 0 {
		t.Fatalf("partial fixture=%+v", checks)
	}
	if err := ValidateResponse(checks); err != nil {
		t.Fatal(err)
	}
	checks.PartialFailures = make([]PartialFailure, MaxPartialFailures+1)
	if err := ValidateResponse(checks); err == nil || err.Code != ErrorInputTooLarge {
		t.Fatalf("partial bound error=%v", err)
	}
	checks = mustResponse(t, OperationGetDiff)
	checks.Result = json.RawMessage(`"` + strings.Repeat("x", MaxDiffBytes+1) + `"`)
	if err := ValidateResponse(checks); err == nil || err.Code != ErrorOutputTooLarge {
		t.Fatalf("diff bound error=%v", err)
	}
}

func TestCursorAndRevisionFingerprintsBindMutableMaterial(t *testing.T) {
	repository := fixtureRepository()
	material := CursorMaterial{
		Version: Version, Provider: "github", ConnectionID: "host", Repositories: []RepositoryIdentity{repository},
		Operation: OperationListPullRequests, Query: "state:open", Order: "updated_desc", Position: "page-1",
	}
	first, err := CursorFingerprint(material)
	if err != nil {
		t.Fatal(err)
	}
	material.Query = "state:closed"
	second, err := CursorFingerprint(material)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("query change did not change cursor fingerprint")
	}
	if mismatch := ValidateCursorFingerprint(first, second); mismatch == nil || mismatch.Code != ErrorCursorMismatch {
		t.Fatalf("mismatch=%v", mismatch)
	}
	encoded, contractErr := EncodeCursor(material)
	if contractErr != nil {
		t.Fatal(contractErr)
	}
	request := Request{
		Version: Version, Operation: material.Operation, Provider: material.Provider, ConnectionID: material.ConnectionID,
		Repository: fixtureRepository(), Query: material.Query, Order: material.Order, Cursor: encoded,
	}
	if contractErr := ValidateRequest(request); contractErr != nil {
		t.Fatal(contractErr)
	}
	request.Query = "different"
	if contractErr := ValidateRequest(request); contractErr == nil || contractErr.Code != ErrorCursorMismatch {
		t.Fatalf("query mismatch=%v", contractErr)
	}
	request.Query = material.Query
	request.Provider = "gitlab"
	if contractErr := ValidateRequest(request); contractErr == nil || contractErr.Code != ErrorCursorMismatch {
		t.Fatalf("provider mismatch=%v", contractErr)
	}
	request.Provider = material.Provider
	request.Repository.Host = "git.example.com"
	if contractErr := ValidateRequest(request); contractErr == nil || contractErr.Code != ErrorCursorMismatch {
		t.Fatalf("repository host mismatch=%v", contractErr)
	}
	tampered := encoded[:len(encoded)-1] + "A"
	if _, contractErr := DecodeCursor(tampered); contractErr == nil || contractErr.Code != ErrorCursorMismatch {
		t.Fatalf("tampered cursor error=%v", contractErr)
	}

	head := RefIdentity{Repository: repository, Name: "feature", SHA: strings.Repeat("a", 40)}
	revision, contractErr := RevisionFingerprint("observation", RevisionMaterial{ConnectionID: "host", Repository: repository, Head: &head})
	if contractErr != nil {
		t.Fatal(contractErr)
	}
	head.SHA = strings.Repeat("b", 40)
	changed, contractErr := RevisionFingerprint("observation", RevisionMaterial{ConnectionID: "host", Repository: repository, Head: &head})
	if contractErr != nil || revision == changed {
		t.Fatalf("revision=%q changed=%q err=%v", revision, changed, contractErr)
	}
}

func TestErrorAndRetryEnumsAreClosedAndFixtureComplete(t *testing.T) {
	fixture := mustFixture(t)
	codes := map[string]RetryGuidance{}
	for _, contractError := range fixture.Errors {
		codes[contractError.Code] = contractError.Retry
	}
	if len(codes) != 26 || !reflect.DeepEqual(sortedKeys(codes), sortedStrings(ErrorCodes())) {
		t.Fatalf("error inventory=%v", sortedKeys(codes))
	}
	if codes[ErrorRateLimited] != RetryAfter || codes[ErrorAmbiguousResult] != RetryReconcile || codes[ErrorStaleObservation] != RetryRefreshThenRetry {
		t.Fatalf("retry mapping=%v", codes)
	}
}

func TestFixtureIsByteStableAndMatchesPublishedDigest(t *testing.T) {
	first, err := CanonicalFixture()
	if err != nil {
		t.Fatal(err)
	}
	second, err := CanonicalFixture()
	if err != nil {
		t.Fatal(err)
	}
	embedded, err := ConsumerFixture()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) || !bytes.Equal(first, embedded) {
		t.Fatal("canonical fixture is not byte-stable or embedded bytes drifted")
	}
	want, err := CanonicalFixtureSHA256()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ConsumerFixtureDigest()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("digest=%q want=%q", got, want)
	}
}

func TestUnknownAdditiveFieldsDecodeAndMajorVersionFailsClosed(t *testing.T) {
	data, err := ConsumerFixture()
	if err != nil {
		t.Fatal(err)
	}
	var known struct {
		Version string `json:"version"`
		Cases   []struct {
			Name string `json:"name"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &known); err != nil || known.Version != Version || len(known.Cases) != 20 {
		t.Fatalf("known decode=%+v err=%v", known, err)
	}
	fixture := mustFixture(t)
	knownCapabilities := 0
	unknownCapabilities := 0
	for _, capability := range fixture.Operations {
		if IsOperation(capability.Policy.Operation) {
			knownCapabilities++
		} else {
			unknownCapabilities++
		}
	}
	if knownCapabilities != 20 || unknownCapabilities != 1 {
		t.Fatalf("known capabilities=%d unknown=%d", knownCapabilities, unknownCapabilities)
	}
	request := mustFixture(t).Cases[0].Request
	request.Version = "code-host-broker/v2"
	if contractErr := ValidateRequest(request); contractErr == nil || contractErr.Code != ErrorIncompatibleVersion {
		t.Fatalf("version error=%v", contractErr)
	}
}

func TestFixtureAndErrorsContainNoCredentialCanaries(t *testing.T) {
	data, err := ConsumerFixture()
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(string(data))
	for _, forbidden := range []string{"authorization", "bearer ", "ghp_", "github_pat_", "credential-canary", "raw_provider_body"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("fixture contains forbidden credential material %q", forbidden)
		}
	}
	for _, forbidden := range []string{"bounded fixture body", "fixture comment", "fixture review", "merge fixture", "add code-host broker"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("fixture contains mutation text %q", forbidden)
		}
	}
	if !strings.Contains(lower, redactedFixtureText) {
		t.Fatal("fixture does not demonstrate redacted text placeholders")
	}
}

func TestOperationSpecificResultsRejectInvalidShapesAndBounds(t *testing.T) {
	pullRequests := mustResponse(t, OperationListPullRequests)
	pullRequests.Result = json.RawMessage(`{"pull_requests":"not-an-array"}`)
	if err := ValidateResponse(pullRequests); err == nil || err.Code != ErrorInvalidInput {
		t.Fatalf("untyped pull request result accepted: %v", err)
	}

	checks := mustResponse(t, OperationGetChecks)
	checks.Result = json.RawMessage(`{"checks":[{"name":"bad","status":"done","availability":"invented"}]}`)
	if err := ValidateResponse(checks); err == nil || err.Code != ErrorInvalidInput {
		t.Fatalf("invalid availability accepted: %v", err)
	}

	comments := mustResponse(t, OperationGetComments)
	tooMany := CommentsResult{Comments: make([]Comment, MaxItems+1)}
	comments.Result, _ = json.Marshal(tooMany)
	if err := ValidateResponse(comments); err == nil || err.Code != ErrorInputTooLarge {
		t.Fatalf("item bound error=%v", err)
	}

	diff := mustResponse(t, OperationGetDiff)
	tooManyFiles := DiffResult{Files: make([]DiffFile, MaxDiffFiles+1)}
	diff.Result, _ = json.Marshal(tooManyFiles)
	if err := ValidateResponse(diff); err == nil || err.Code != ErrorInputTooLarge {
		t.Fatalf("diff file bound error=%v", err)
	}
	diff.Result, _ = json.Marshal(DiffResult{Files: []DiffFile{{
		Path: "file.go", Status: "modified", Hunks: make([]DiffHunk, MaxDiffHunks+1),
	}}})
	if err := ValidateResponse(diff); err == nil || err.Code != ErrorInputTooLarge {
		t.Fatalf("diff hunk bound error=%v", err)
	}

	mutation := mustResponse(t, OperationComment)
	var mutationResult MutationResult
	if err := json.Unmarshal(mutation.Result, &mutationResult); err != nil {
		t.Fatal(err)
	}
	mutationResult.Actor = &Actor{}
	mutation.Result, _ = json.Marshal(mutationResult)
	if err := ValidateResponse(mutation); err == nil || err.Code != ErrorInvalidInput {
		t.Fatalf("incomplete mutation actor accepted: %v", err)
	}
}

func TestMutationResponsesRequireReconciliationAndExactRetry(t *testing.T) {
	response := mustResponse(t, OperationComment)
	response.Receipt = nil
	if err := ValidateResponse(response); err == nil {
		t.Fatal("successful mutation without receipt accepted")
	}
	response = mustResponse(t, OperationComment)
	response.Reconciliation = nil
	if err := ValidateResponse(response); err == nil {
		t.Fatal("successful mutation without reconciliation accepted")
	}
	response = mustResponse(t, OperationComment)
	response.Result = nil
	response.Receipt = nil
	response.Error = &ContractError{Code: ErrorAmbiguousResult, Message: "outcome unknown", Retry: RetrySameKey}
	response.Reconciliation = &Reconciliation{Status: ReconciliationAmbiguous, Key: "same-key"}
	if err := ValidateResponse(response); err == nil {
		t.Fatal("ambiguous result with unsafe retry guidance accepted")
	}
	response.Error.Retry = RetryReconcile
	if err := ValidateResponse(response); err != nil {
		t.Fatalf("valid ambiguous response rejected: %v", err)
	}
	response.Reconciliation.Status = ReconciliationNotApplied
	if err := ValidateResponse(response); err == nil {
		t.Fatal("ambiguous error without ambiguous reconciliation accepted")
	}
}

func TestEveryPublishedBoundHasEnforcementEvidence(t *testing.T) {
	request := mustFixture(t).Cases[1].Request
	request.Repositories = make([]RepositoryIdentity, MaxRepositoryScopes)
	if err := ValidateRequest(request); err == nil || err.Code != ErrorInputTooLarge {
		t.Fatalf("primary plus %d additional repository scopes accepted: %v", MaxRepositoryScopes, err)
	}
	request = mustFixture(t).Cases[1].Request
	request.Limit = MaxPageSize + 1
	if err := ValidateRequest(request); err == nil {
		t.Fatal("page size bound accepted")
	}
	request = mustFixture(t).Cases[1].Request
	request.Query = strings.Repeat("x", MaxTextBytes+1)
	if err := ValidateRequest(request); err == nil || err.Code != ErrorInputTooLarge {
		t.Fatalf("text bound error=%v", err)
	}
	request = mustFixture(t).Cases[10].Request
	request.Payload = json.RawMessage(`"` + strings.Repeat("x", MaxBodyBytes+1) + `"`)
	if err := ValidateRequest(request); err == nil || err.Code != ErrorInputTooLarge {
		t.Fatalf("body bound error=%v", err)
	}
	request = mustFixture(t).Cases[10].Request
	request.IdempotencyKey = strings.Repeat("x", MaxIdempotencyBytes+1)
	if err := ValidateRequest(request); err == nil {
		t.Fatal("idempotency bound accepted")
	}
	response := mustResponse(t, OperationGetPullRequest)
	response.Redirects = MaxRedirects + 1
	if err := ValidateResponse(response); err == nil || err.Code != ErrorInputTooLarge {
		t.Fatalf("redirect bound error=%v", err)
	}
	response = mustResponse(t, OperationComment)
	response.JournalEntries = MaxJournalEntries + 1
	if err := ValidateResponse(response); err == nil || err.Code != ErrorInputTooLarge {
		t.Fatalf("journal bound error=%v", err)
	}
	response = mustResponse(t, OperationGetComments)
	response.Page.NextCursor = strings.Repeat("x", MaxTextBytes+1)
	if err := ValidateResponse(response); err == nil {
		t.Fatal("unbounded page cursor accepted")
	}
	response = mustResponse(t, OperationGetPullRequest)
	response.RateLimit.Remaining = response.RateLimit.Limit + 1
	if err := ValidateResponse(response); err == nil {
		t.Fatal("inconsistent rate limit accepted")
	}
	response = mustResponse(t, OperationGetPullRequest)
	response.DurationMS = MaxDurationMS + 1
	if err := ValidateResponse(response); err == nil {
		t.Fatal("duration bound accepted")
	}
	response = mustResponse(t, OperationGetChecks)
	response.PartialFailures = make([]PartialFailure, MaxPartialFailures+1)
	if err := ValidateResponse(response); err == nil {
		t.Fatal("partial-failure bound accepted")
	}
	response = mustResponse(t, OperationGetPullRequest)
	response.Result = nil
	response.Error = &ContractError{
		Code: ErrorProvider, Message: strings.Repeat("x", MaxErrorDetailBytes+1), Retry: RetryNone,
	}
	if err := ValidateResponse(response); err == nil {
		t.Fatal("error-detail bound accepted")
	}
	response = mustResponse(t, OperationGetPullRequest)
	response.CapabilityRevision = strings.Repeat("x", 513)
	if err := ValidateResponse(response); err == nil {
		t.Fatal("capability revision bound accepted")
	}
	response = mustResponse(t, OperationGetPullRequest)
	response.ObservationRevision = strings.Repeat("x", 513)
	if err := ValidateResponse(response); err == nil {
		t.Fatal("observation revision bound accepted")
	}
	response = mustResponse(t, OperationGetPullRequest)
	response.RateLimit.Resource = strings.Repeat("x", 129)
	if err := ValidateResponse(response); err == nil {
		t.Fatal("rate-limit resource bound accepted")
	}
	response = mustResponse(t, OperationGetPullRequest)
	response.RateLimit.ResetAt = strings.Repeat("x", 65)
	if err := ValidateResponse(response); err == nil {
		t.Fatal("rate-limit reset bound accepted")
	}
	response = mustResponse(t, OperationGetPullRequest)
	response.Result = nil
	response.Error = &ContractError{Code: ErrorProvider, Message: "safe", Field: strings.Repeat("x", 513), Retry: RetryNone}
	if err := ValidateResponse(response); err == nil {
		t.Fatal("error field bound accepted")
	}
	response.Error.Field = ""
	response.Error.RetryAt = strings.Repeat("x", 65)
	if err := ValidateResponse(response); err == nil {
		t.Fatal("error retry-at bound accepted")
	}
}

func TestFixtureCoversAvailabilityCompletenessAndPaginationStates(t *testing.T) {
	fixture := mustFixture(t)
	availability := map[Availability]bool{}
	completeness := map[Completeness]bool{}
	hasNext := false
	hasTerminal := false
	for _, fixtureCase := range fixture.Cases {
		completeness[fixtureCase.Response.Completeness] = true
		if fixtureCase.Response.Page != nil {
			if fixtureCase.Response.Page.NextCursor == "" {
				hasTerminal = true
			} else {
				hasNext = true
				if _, err := DecodeCursor(fixtureCase.Response.Page.NextCursor); err != nil {
					t.Fatalf("%s cursor: %v", fixtureCase.Name, err)
				}
			}
		}
		if fixtureCase.Response.Operation == OperationGetChecks {
			var result ChecksResult
			if err := json.Unmarshal(fixtureCase.Response.Result, &result); err != nil {
				t.Fatal(err)
			}
			for _, check := range result.Checks {
				availability[check.Availability] = true
			}
		}
	}
	for _, value := range []Availability{AvailabilityAvailable, AvailabilityPartial, AvailabilityUnavailable, AvailabilityUnknown} {
		if !availability[value] {
			t.Fatalf("missing availability %q", value)
		}
	}
	for _, value := range []Completeness{CompletenessComplete, CompletenessPartial, CompletenessTruncated, CompletenessUnavailable} {
		if !completeness[value] {
			t.Fatalf("missing completeness %q", value)
		}
	}
	if !hasNext || !hasTerminal {
		t.Fatalf("pagination next=%v terminal=%v", hasNext, hasTerminal)
	}
}

func TestFixtureDecodesWithIndependentConsumerShapes(t *testing.T) {
	data, err := ConsumerFixture()
	if err != nil {
		t.Fatal(err)
	}
	var consumer struct {
		Version    string `json:"version"`
		Operations []struct {
			Policy struct {
				Operation string `json:"operation"`
				Effect    string `json:"effect"`
			} `json:"policy"`
			Available bool `json:"available"`
		} `json:"operations"`
		Cases []struct {
			Name    string `json:"name"`
			Request struct {
				Version      string `json:"version"`
				Operation    string `json:"operation"`
				Provider     string `json:"provider"`
				ConnectionID string `json:"connection_id"`
			} `json:"request"`
			Response struct {
				Version      string          `json:"version"`
				Operation    string          `json:"operation"`
				Provider     string          `json:"provider"`
				ConnectionID string          `json:"connection_id"`
				Result       json.RawMessage `json:"result"`
			} `json:"response"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &consumer); err != nil {
		t.Fatal(err)
	}
	if consumer.Version != Version || len(consumer.Operations) != 21 || len(consumer.Cases) != 20 {
		t.Fatalf("independent decoder inventory=%+v", consumer)
	}
	for _, fixtureCase := range consumer.Cases {
		if fixtureCase.Name == "" || fixtureCase.Request.Version != Version ||
			fixtureCase.Request.Operation != fixtureCase.Response.Operation ||
			fixtureCase.Request.Provider != fixtureCase.Response.Provider ||
			fixtureCase.Response.Version != Version || fixtureCase.Response.Provider == "" ||
			fixtureCase.Request.ConnectionID != fixtureCase.Response.ConnectionID || !json.Valid(fixtureCase.Response.Result) {
			t.Fatalf("independent decoder rejected case %+v", fixtureCase)
		}
		if err := decodeIndependentConsumerResult(fixtureCase.Response.Operation, fixtureCase.Response.Result); err != nil {
			t.Fatalf("%s independent result decoder: %v", fixtureCase.Name, err)
		}
	}
	pullRequest := mustResponse(t, OperationGetPullRequest).Result
	var additive map[string]any
	if err := json.Unmarshal(pullRequest, &additive); err != nil {
		t.Fatal(err)
	}
	additive["future_result_field"] = map[string]any{"nested": true}
	withAdditive, err := json.Marshal(additive)
	if err != nil {
		t.Fatal(err)
	}
	if err := decodeIndependentConsumerResult(string(OperationGetPullRequest), withAdditive); err != nil {
		t.Fatalf("independent consumer rejected additive result field: %v", err)
	}
}

func FuzzValidatePullRequestIdentity(f *testing.F) {
	identity := fixtureIdentity()
	f.Add(identity.ConnectionID, identity.Repository.Host, identity.Repository.Owner, identity.Repository.Name, identity.ProviderID, identity.Number)
	f.Fuzz(func(t *testing.T, connection, host, owner, name, providerID string, number int64) {
		value := PullRequestIdentity{
			ConnectionID: connection,
			Repository:   RepositoryIdentity{Host: host, Owner: owner, Name: name, FullName: owner + "/" + name},
			ProviderID:   providerID,
			Number:       number,
		}
		_ = ValidatePullRequestIdentity(value)
	})
}

func FuzzCursorRoundTrip(f *testing.F) {
	f.Add("github", "connection", "github.com", "repo-id", "owner", "repo", "state:open", "updated_desc", "position")
	f.Fuzz(func(t *testing.T, provider, connection, host, repositoryID, owner, name, query, order, position string) {
		repository := RepositoryIdentity{
			Host: host, ProviderID: repositoryID, Owner: owner, Name: name, FullName: owner + "/" + name,
		}
		material := CursorMaterial{
			Version: Version, Provider: provider, ConnectionID: connection,
			Repositories: []RepositoryIdentity{repository}, Operation: OperationListPullRequests,
			Query: query, Order: order, Position: position,
		}
		encoded, err := EncodeCursor(material)
		if err != nil {
			return
		}
		decoded, err := DecodeCursor(encoded)
		if err != nil {
			t.Fatalf("encoded cursor did not decode: %v", err)
		}
		want := material
		want.Repositories = normalizedRepositoryScope(want.Repositories)
		if !reflect.DeepEqual(decoded.Material, want) {
			t.Fatalf("cursor material drifted: got=%+v want=%+v", decoded.Material, want)
		}
	})
}

func mustFixture(t *testing.T) ConsumerFixtureBundle {
	t.Helper()
	data, err := CanonicalFixture()
	if err != nil {
		t.Fatal(err)
	}
	var fixture ConsumerFixtureBundle
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func mustResponse(t *testing.T, operation Operation) Response {
	t.Helper()
	for _, fixtureCase := range mustFixture(t).Cases {
		if fixtureCase.Response.Operation == operation {
			return fixtureCase.Response
		}
	}
	t.Fatalf("missing response %q", operation)
	return Response{}
}

func fixtureRepository() RepositoryIdentity {
	return RepositoryIdentity{Host: "github.com", ProviderID: "repo", Owner: "hero-engine", Name: "hero", FullName: "hero-engine/hero"}
}

func fixtureIdentity() PullRequestIdentity {
	return PullRequestIdentity{ConnectionID: "github-host", Repository: fixtureRepository(), ProviderID: "pr-node", Number: 42}
}

func sortedKeys(values map[string]RetryGuidance) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	return sortedStrings(out)
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

type consumerRepository struct {
	Host       string `json:"host"`
	ProviderID string `json:"provider_id,omitempty"`
	Owner      string `json:"owner"`
	Name       string `json:"name"`
	FullName   string `json:"full_name"`
}

type consumerRef struct {
	Repository consumerRepository `json:"repository"`
	Name       string             `json:"name"`
	SHA        string             `json:"sha"`
}

type consumerIdentity struct {
	ConnectionID string             `json:"connection_id"`
	Repository   consumerRepository `json:"repository"`
	ProviderID   string             `json:"provider_id"`
	Number       int64              `json:"number"`
}

type consumerActor struct {
	ProviderID string `json:"provider_id,omitempty"`
	Login      string `json:"login"`
	Display    string `json:"display,omitempty"`
}

type consumerPullRequest struct {
	Identity  consumerIdentity `json:"identity"`
	Title     string           `json:"title"`
	Body      string           `json:"body,omitempty"`
	URL       string           `json:"url"`
	State     string           `json:"state"`
	Draft     bool             `json:"draft"`
	Author    consumerActor    `json:"author"`
	Base      consumerRef      `json:"base"`
	Head      consumerRef      `json:"head"`
	CreatedAt string           `json:"created_at,omitempty"`
	UpdatedAt string           `json:"updated_at,omitempty"`
	MergedAt  string           `json:"merged_at,omitempty"`
}

type consumerBounds struct {
	RepositoryScopes int `json:"repository_scopes"`
	PageSize         int `json:"page_size"`
	Items            int `json:"items"`
	TextBytes        int `json:"text_bytes"`
	BodyBytes        int `json:"body_bytes"`
	DiffBytes        int `json:"diff_bytes"`
	DiffFiles        int `json:"diff_files"`
	DiffHunks        int `json:"diff_hunks"`
	PartialFailures  int `json:"partial_failures"`
	ErrorDetailBytes int `json:"error_detail_bytes"`
	DurationMS       int `json:"duration_ms"`
	Redirects        int `json:"redirects"`
	JournalEntries   int `json:"journal_entries"`
	IdempotencyBytes int `json:"idempotency_bytes"`
}

type consumerPolicy struct {
	Operation                string         `json:"operation"`
	Effect                   string         `json:"effect"`
	Consent                  string         `json:"consent"`
	RequiresUniqueTarget     bool           `json:"requires_unique_target"`
	RequiresIdempotency      bool           `json:"requires_idempotency"`
	RequiresFreshObservation bool           `json:"requires_fresh_observation"`
	RequiresReconciliation   bool           `json:"requires_reconciliation"`
	ReplaySafe               bool           `json:"replay_safe"`
	Bounds                   consumerBounds `json:"bounds"`
}

func decodeIndependentConsumerResult(operation string, raw json.RawMessage) error {
	switch Operation(operation) {
	case OperationCapabilities:
		var value struct {
			Capabilities []struct {
				Policy    consumerPolicy `json:"policy"`
				Available bool           `json:"available"`
				Reason    string         `json:"reason,omitempty"`
			} `json:"capabilities"`
		}
		return strictConsumerDecode(raw, &value)
	case OperationListPullRequests, OperationSearchPullRequests:
		var value struct {
			PullRequests []consumerPullRequest `json:"pull_requests"`
		}
		return strictConsumerDecode(raw, &value)
	case OperationGetPullRequest:
		var value consumerPullRequest
		return strictConsumerDecode(raw, &value)
	case OperationGetCommits:
		var value struct {
			Commits []struct {
				SHA        string        `json:"sha"`
				Message    string        `json:"message"`
				Author     consumerActor `json:"author"`
				AuthoredAt string        `json:"authored_at,omitempty"`
				URL        string        `json:"url,omitempty"`
			} `json:"commits"`
		}
		return strictConsumerDecode(raw, &value)
	case OperationGetDiff:
		var value struct {
			Files []struct {
				Path      string `json:"path"`
				Status    string `json:"status"`
				Additions int    `json:"additions"`
				Deletions int    `json:"deletions"`
				Hunks     []struct {
					Header string `json:"header"`
					Patch  string `json:"patch"`
				} `json:"hunks"`
				Truncated bool `json:"truncated"`
			} `json:"files"`
		}
		return strictConsumerDecode(raw, &value)
	case OperationGetChecks:
		var value struct {
			Checks []struct {
				ProviderID   string `json:"provider_id,omitempty"`
				Name         string `json:"name"`
				Status       string `json:"status"`
				Conclusion   string `json:"conclusion,omitempty"`
				URL          string `json:"url,omitempty"`
				Availability string `json:"availability"`
			} `json:"checks"`
		}
		return strictConsumerDecode(raw, &value)
	case OperationGetReviews:
		var value struct {
			Reviews []struct {
				ProviderID  string        `json:"provider_id"`
				Author      consumerActor `json:"author"`
				State       string        `json:"state"`
				Body        string        `json:"body,omitempty"`
				HeadSHA     string        `json:"head_sha"`
				SubmittedAt string        `json:"submitted_at,omitempty"`
			} `json:"reviews"`
		}
		return strictConsumerDecode(raw, &value)
	case OperationGetComments:
		var value struct {
			Comments []struct {
				ProviderID string        `json:"provider_id"`
				Author     consumerActor `json:"author"`
				Body       string        `json:"body"`
				URL        string        `json:"url,omitempty"`
				CreatedAt  string        `json:"created_at,omitempty"`
				UpdatedAt  string        `json:"updated_at,omitempty"`
			} `json:"comments"`
		}
		return strictConsumerDecode(raw, &value)
	case OperationGetMergeReadiness:
		var value struct {
			State            string   `json:"state"`
			Checks           string   `json:"checks"`
			Reviews          string   `json:"reviews"`
			BranchProtection string   `json:"branch_protection"`
			Permissions      string   `json:"permissions"`
			Mergeability     string   `json:"mergeability"`
			Queue            string   `json:"queue"`
			Reasons          []string `json:"reasons"`
		}
		return strictConsumerDecode(raw, &value)
	default:
		var value struct {
			PullRequest consumerPullRequest `json:"pull_request"`
			Outcome     string              `json:"outcome"`
			Actor       *consumerActor      `json:"actor,omitempty"`
		}
		return strictConsumerDecode(raw, &value)
	}
}

func strictConsumerDecode(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}
