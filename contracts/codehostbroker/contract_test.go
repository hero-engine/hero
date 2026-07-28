package codehostbroker

import (
	"bytes"
	"encoding/json"
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
	if fixture.Version != Version || len(fixture.Cases) != len(Operations()) || len(fixture.Operations) != len(Operations()) {
		t.Fatalf("fixture inventory: version=%q cases=%d operations=%d", fixture.Version, len(fixture.Cases), len(fixture.Operations))
	}
	seen := map[Operation]bool{}
	statuses := map[ReconciliationStatus]bool{}
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
	}
	if len(seen) != 20 || len(statuses) != 7 {
		t.Fatalf("coverage operations=%d reconciliation=%v", len(seen), statuses)
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
	material := CursorMaterial{
		Version: Version, Provider: "github", ConnectionID: "host", Repositories: []string{"hero-engine/hero"},
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

	repository := fixtureRepository()
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
