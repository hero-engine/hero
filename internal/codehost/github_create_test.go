package codehost

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/mockcodehost"
)

const (
	createTitleCanary = "Create broker PR"
	createBodyCanary  = "CREATE-BODY-CANARY create body"
)

func TestCreatePullRequestSuccessAndContractConformance(t *testing.T) {
	fixture, fake, broker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	request := preparedCreateRequest(t, fixture, broker, fixture.repository("acme/widgets"), "create-success")

	response := broker.Execute(context.Background(), request)
	requireValidResponse(t, response)
	if response.Error != nil || response.Receipt == nil || response.Reconciliation == nil {
		t.Fatalf("create response error=%v receipt=%+v reconciliation=%+v", response.Error, response.Receipt, response.Reconciliation)
	}
	if response.Policy.Effect != codehostbroker.EffectExternalWrite ||
		response.Policy.Consent != codehostbroker.ConsentExplicitUser ||
		response.Reconciliation.Status != codehostbroker.ReconciliationApplied ||
		response.JournalEntries != 1 || response.RateLimit.Resource != "core" ||
		response.RateLimit.Limit != 5000 || response.RateLimit.Remaining != 4999 ||
		fake.CreateAttempts() != 1 {
		t.Fatalf("create metadata=%+v attempts=%d", response, fake.CreateAttempts())
	}
	var result codehostbroker.MutationResult
	decodeResult(t, response, &result)
	if result.Outcome != "applied" || result.PullRequest.Title != createTitleCanary ||
		result.PullRequest.Body != createBodyCanary ||
		result.PullRequest.Head.Repository.FullName != "acme/widgets" ||
		result.PullRequest.Head.Name != "feature/create" {
		t.Fatalf("create result=%+v", result)
	}
	assertJournalPrivateAndRedacted(t, broker.projectRoot)
}

func TestCreateRequiresExplicitDraftAndPolicyMaterial(t *testing.T) {
	fixture, fake, broker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	request := createRequest(fixture, fixture.repository("acme/widgets"), "missing-draft")
	request.Payload = json.RawMessage(`{
		"base":{"repository":{"host":"` + fixture.host + `","provider_id":"R_acme_widgets","owner":"acme","name":"widgets","full_name":"acme/widgets"},"name":"main","sha":"` + baseSHAForTest + `"},
		"head":{"repository":{"host":"` + fixture.host + `","provider_id":"R_acme_widgets","owner":"acme","name":"widgets","full_name":"acme/widgets"},"name":"feature/create","sha":"` + headSHAForTest + `"},
		"title":"Create broker PR"
	}`)
	response := broker.Execute(context.Background(), request)
	requireValidResponse(t, response)
	if response.Error == nil || response.Error.Field != "payload.draft" || fake.CreateAttempts() != 0 {
		t.Fatalf("missing draft response=%+v attempts=%d", response.Error, fake.CreateAttempts())
	}

	valid := createRequest(fixture, fixture.repository("acme/widgets"), "missing-policy")
	for name, mutate := range map[string]func(*codehostbroker.Request){
		"intent":         func(value *codehostbroker.Request) { value.IntentSource = "" },
		"idempotency":    func(value *codehostbroker.Request) { value.IdempotencyKey = "" },
		"capability":     func(value *codehostbroker.Request) { value.CapabilityRevision = "" },
		"observation":    func(value *codehostbroker.Request) { value.ObservationRevision = "" },
		"reconciliation": func(value *codehostbroker.Request) { value.ReconciliationKey = "" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			response := broker.Execute(context.Background(), candidate)
			requireValidResponse(t, response)
			if response.Error == nil || fake.CreateAttempts() != 0 {
				t.Fatalf("%s response=%+v attempts=%d", name, response.Error, fake.CreateAttempts())
			}
		})
	}
}

func TestCreateDuplicateConcurrentAndConflictingRetries(t *testing.T) {
	fixture, fake, broker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	request := preparedCreateRequest(t, fixture, broker, fixture.repository("acme/widgets"), "duplicate")
	first := broker.Execute(context.Background(), request)
	second := broker.Execute(context.Background(), request)
	requireValidResponse(t, first)
	requireValidResponse(t, second)
	if first.Error != nil || second.Error != nil || second.Reconciliation == nil ||
		second.Reconciliation.Status != codehostbroker.ReconciliationReplayed ||
		first.Receipt == nil || second.Receipt == nil ||
		first.Receipt.ProviderReceiptID != second.Receipt.ProviderReceiptID ||
		fake.CreateAttempts() != 1 {
		t.Fatalf("duplicate first=%+v second=%+v attempts=%d", first, second, fake.CreateAttempts())
	}

	conflicting := request
	var payload codehostbroker.CreatePullRequestPayload
	if err := json.Unmarshal(conflicting.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	payload.Title = "different title"
	conflicting.Payload = mustJSON(t, payload)
	conflict := broker.Execute(context.Background(), conflicting)
	requireValidResponse(t, conflict)
	if conflict.Error == nil || conflict.Error.Code != codehostbroker.ErrorIdempotencyConflict ||
		fake.CreateAttempts() != 1 {
		t.Fatalf("conflict=%+v attempts=%d", conflict.Error, fake.CreateAttempts())
	}

	concurrentFixture, concurrentFake, concurrentBroker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	concurrent := preparedCreateRequest(t, concurrentFixture, concurrentBroker, concurrentFixture.repository("acme/widgets"), "concurrent")
	responses := make([]codehostbroker.Response, 2)
	var wait sync.WaitGroup
	for index := range responses {
		wait.Add(1)
		go func() {
			defer wait.Done()
			responses[index] = concurrentBroker.Execute(context.Background(), concurrent)
		}()
	}
	wait.Wait()
	for _, response := range responses {
		requireValidResponse(t, response)
		if response.Error != nil {
			t.Fatalf("concurrent response error=%v", response.Error)
		}
	}
	if concurrentFake.CreateAttempts() != 1 ||
		responses[0].Receipt.ProviderReceiptID != responses[1].Receipt.ProviderReceiptID {
		t.Fatalf("concurrent attempts=%d responses=%+v", concurrentFake.CreateAttempts(), responses)
	}

	canonicalFixture, canonicalFake, canonicalBroker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets", "acme/gadgets")
	canonical := preparedCreateRequest(t, canonicalFixture, canonicalBroker, canonicalFixture.repository("acme/widgets"), "canonical")
	canonicalFirst := canonicalBroker.Execute(context.Background(), canonical)
	var canonicalPayload codehostbroker.CreatePullRequestPayload
	if err := json.Unmarshal(canonical.Payload, &canonicalPayload); err != nil {
		t.Fatal(err)
	}
	canonical.Repository.ProviderID = ""
	canonicalPayload.Base.Repository.ProviderID = ""
	canonicalPayload.Head.Repository.ProviderID = ""
	canonical.Payload = mustJSON(t, canonicalPayload)
	canonicalReplay := canonicalBroker.Execute(context.Background(), canonical)
	requireValidResponse(t, canonicalFirst)
	requireValidResponse(t, canonicalReplay)
	if canonicalReplay.Error != nil || canonicalReplay.Reconciliation == nil ||
		canonicalReplay.Reconciliation.Status != codehostbroker.ReconciliationReplayed ||
		canonicalReplay.JournalEntries != 1 || canonicalFake.CreateAttempts() != 1 {
		t.Fatalf("canonical replay=%+v attempts=%d", canonicalReplay, canonicalFake.CreateAttempts())
	}

	targetFixture, targetFake, targetBroker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets", "acme/gadgets")
	widgets := preparedCreateRequest(t, targetFixture, targetBroker, targetFixture.repository("acme/widgets"), "cross-target")
	gadgetsBase := targetFixture.repository("acme/gadgets")
	gadgets := createRequest(targetFixture, gadgetsBase, "cross-target")
	var gadgetsPayload codehostbroker.CreatePullRequestPayload
	if err := json.Unmarshal(gadgets.Payload, &gadgetsPayload); err != nil {
		t.Fatal(err)
	}
	gadgets.Repository = gadgetsBase
	gadgetsPayload.Base.Repository = gadgetsBase
	gadgets.Payload = mustJSON(t, gadgetsPayload)
	var prepareErr *codehostbroker.ContractError
	gadgets, prepareErr = targetBroker.PrepareCreatePullRequest(context.Background(), gadgets)
	if prepareErr != nil {
		t.Fatalf("prepare cross-target create: %+v", prepareErr)
	}
	widgetsResponse := targetBroker.Execute(context.Background(), widgets)
	targetConflict := targetBroker.Execute(context.Background(), gadgets)
	requireValidResponse(t, widgetsResponse)
	requireValidResponse(t, targetConflict)
	if targetConflict.Error == nil || targetConflict.Error.Code != codehostbroker.ErrorIdempotencyConflict ||
		targetFake.CreateAttempts() != 1 {
		t.Fatalf("cross-target conflict=%+v attempts=%d", targetConflict.Error, targetFake.CreateAttempts())
	}

	deniedFixture, deniedFake, deniedBroker := createTestBroker(t, mockcodehost.CreateWriteDeniedScenario(), "acme/widgets")
	deniedRequest := preparedCreateRequest(t, deniedFixture, deniedBroker, deniedFixture.repository("acme/widgets"), "write-denied")
	deniedFirst := deniedBroker.Execute(context.Background(), deniedRequest)
	deniedSecond := deniedBroker.Execute(context.Background(), deniedRequest)
	requireValidResponse(t, deniedFirst)
	requireValidResponse(t, deniedSecond)
	if deniedFirst.Error == nil || deniedFirst.Error.Code != codehostbroker.ErrorForbidden ||
		deniedSecond.Error == nil || deniedSecond.Error.Code != codehostbroker.ErrorForbidden ||
		deniedFake.CreateAttempts() != 1 {
		t.Fatalf("denied retries first=%+v second=%+v attempts=%d", deniedFirst.Error, deniedSecond.Error, deniedFake.CreateAttempts())
	}
}

func TestCreateReconcilesLostAndCancelledResponses(t *testing.T) {
	for name, scenario := range map[string]mockcodehost.Scenario{
		"lost_response":         mockcodehost.CreateLostResponseScenario(),
		"cancelled_after_apply": mockcodehost.CreateCancelledAfterApplyScenario(cancelAfterApplyResponseDelay),
	} {
		t.Run(name, func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, scenario, "acme/widgets")
			request := preparedCreateRequest(t, fixture, broker, fixture.repository("acme/widgets"), name)
			var response codehostbroker.Response
			if name == "cancelled_after_apply" {
				response = executeThenCancelAfterAttempt(t, broker, request, fake.CreateAttempts)
			} else {
				response = broker.Execute(context.Background(), request)
			}
			requireValidResponse(t, response)
			if response.Error != nil || response.Reconciliation == nil ||
				response.Reconciliation.Status != codehostbroker.ReconciliationReconciledApplied ||
				fake.CreateAttempts() != 1 {
				t.Fatalf("response=%+v attempts=%d", response, fake.CreateAttempts())
			}
		})
	}
}

func TestCreateExternallyCompletedForkAndAmbiguousRecovery(t *testing.T) {
	externalFixture, externalFake, externalBroker := createTestBroker(t, mockcodehost.CreateExternallyCompletedScenario(), "acme/widgets")
	external := preparedCreateRequest(t, externalFixture, externalBroker, externalFixture.repository("acme/widgets"), "external")
	externalResponse := externalBroker.Execute(context.Background(), external)
	requireValidResponse(t, externalResponse)
	if externalResponse.Error != nil || externalResponse.Reconciliation == nil ||
		externalResponse.Reconciliation.Status != codehostbroker.ReconciliationExternallyCompleted ||
		externalFake.CreateAttempts() != 0 {
		t.Fatalf("external response=%+v attempts=%d", externalResponse, externalFake.CreateAttempts())
	}

	forkFixture, forkFake, forkBroker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets", "contributor/widgets")
	fork := preparedCreateRequest(t, forkFixture, forkBroker, forkFixture.repository("contributor/widgets"), "fork")
	forkResponse := forkBroker.Execute(context.Background(), fork)
	requireValidResponse(t, forkResponse)
	var forkResult codehostbroker.MutationResult
	decodeResult(t, forkResponse, &forkResult)
	if forkResponse.Error != nil || forkResult.PullRequest.Head.Repository.FullName != "contributor/widgets" ||
		forkFake.CreateAttempts() != 1 {
		t.Fatalf("fork response=%+v result=%+v", forkResponse.Error, forkResult)
	}

	ambiguousFixture, ambiguousFake, ambiguousBroker := createTestBroker(t, mockcodehost.CreateAmbiguousScenario(), "acme/widgets")
	ambiguous := preparedCreateRequest(t, ambiguousFixture, ambiguousBroker, ambiguousFixture.repository("acme/widgets"), "ambiguous")
	first := ambiguousBroker.Execute(context.Background(), ambiguous)
	second := ambiguousBroker.Execute(context.Background(), ambiguous)
	requireValidResponse(t, first)
	requireValidResponse(t, second)
	for _, response := range []codehostbroker.Response{first, second} {
		if response.Error == nil || response.Error.Code != codehostbroker.ErrorAmbiguousResult ||
			response.Reconciliation == nil || response.Reconciliation.Status != codehostbroker.ReconciliationAmbiguous {
			t.Fatalf("ambiguous response=%+v", response)
		}
		assertCreateErrorRedacted(t, response)
	}
	if ambiguousFake.CreateAttempts() != 1 {
		t.Fatalf("ambiguous retry dispatched %d creates", ambiguousFake.CreateAttempts())
	}
}

func TestCreateRejectsStalePermissionScopeAndCancellationBeforeDispatch(t *testing.T) {
	permissionFixture, permissionFake, permissionBroker := createTestBroker(t, mockcodehost.CreatePermissionDeniedScenario(), "acme/widgets")
	permission := createRequest(permissionFixture, permissionFixture.repository("acme/widgets"), "permission")
	permission.CapabilityRevision = currentCapabilityRevision(t, permissionFixture, permissionBroker)
	if _, err := permissionBroker.PrepareCreatePullRequest(context.Background(), permission); err == nil || err.Code != codehostbroker.ErrorForbidden {
		t.Fatalf("permission preparation error=%+v", err)
	}
	permissionResponse := permissionBroker.Execute(context.Background(), permission)
	requireValidResponse(t, permissionResponse)
	if permissionResponse.Error == nil || permissionResponse.Error.Code != codehostbroker.ErrorForbidden ||
		permissionResponse.Reconciliation == nil || permissionResponse.Reconciliation.Status != codehostbroker.ReconciliationNotApplied {
		t.Fatalf("permission response=%+v", permissionResponse.Error)
	}
	assertCreateErrorRedacted(t, permissionResponse)
	if permissionFake.CreateAttempts() != 0 {
		t.Fatal("permission failure dispatched a write")
	}

	headFixture, headFake, headBroker := createTestBroker(t, mockcodehost.CreateStaleHeadScenario(), "acme/widgets")
	headRequest := createRequest(headFixture, headFixture.repository("acme/widgets"), "stale-head")
	headRequest.CapabilityRevision = currentCapabilityRevision(t, headFixture, headBroker)
	if _, err := headBroker.PrepareCreatePullRequest(context.Background(), headRequest); err == nil || err.Code != codehostbroker.ErrorStaleObservation {
		t.Fatalf("stale head preparation error=%+v", err)
	}
	headResponse := headBroker.Execute(context.Background(), headRequest)
	requireValidResponse(t, headResponse)
	if headResponse.Error == nil || headResponse.Error.Code != codehostbroker.ErrorStaleObservation ||
		headResponse.Reconciliation == nil || headResponse.Reconciliation.Status != codehostbroker.ReconciliationNotApplied {
		t.Fatalf("stale head response=%+v", headResponse.Error)
	}
	if headFake.CreateAttempts() != 0 {
		t.Fatal("stale head failure dispatched a write")
	}

	staleFixture, staleFake, staleBroker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	stale := preparedCreateRequest(t, staleFixture, staleBroker, staleFixture.repository("acme/widgets"), "stale")
	stale.ObservationRevision = "observation:stale"
	staleResponse := staleBroker.Execute(context.Background(), stale)
	requireValidResponse(t, staleResponse)
	if staleResponse.Error == nil || staleResponse.Error.Code != codehostbroker.ErrorStaleObservation ||
		staleFake.CreateAttempts() != 0 {
		t.Fatalf("stale response=%+v attempts=%d", staleResponse.Error, staleFake.CreateAttempts())
	}

	capability := stale
	capability.IdempotencyKey = "capability"
	capability.ReconciliationKey = "reconcile:capability"
	capability.ObservationRevision, capability.CapabilityRevision = staleResponse.ObservationRevision, "capability:stale"
	capabilityResponse := staleBroker.Execute(context.Background(), capability)
	requireValidResponse(t, capabilityResponse)
	if capabilityResponse.Error == nil || capabilityResponse.Error.Code != codehostbroker.ErrorCapabilityChanged ||
		staleFake.CreateAttempts() != 0 {
		t.Fatalf("capability response=%+v attempts=%d", capabilityResponse.Error, staleFake.CreateAttempts())
	}

	scopeFixture, scopeFake, scopeBroker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	scope := createRequest(scopeFixture, scopeFixture.repository("other/widgets"), "scope")
	scopeResponse := scopeBroker.Execute(context.Background(), scope)
	requireValidResponse(t, scopeResponse)
	if scopeResponse.Error == nil || scopeResponse.Error.Code != codehostbroker.ErrorForbidden ||
		scopeFake.CreateAttempts() != 0 {
		t.Fatalf("scope response=%+v attempts=%d", scopeResponse.Error, scopeFake.CreateAttempts())
	}

	cancelFixture, cancelFake, cancelBroker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	cancelRequest := preparedCreateRequest(t, cancelFixture, cancelBroker, cancelFixture.repository("acme/widgets"), "cancel")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled := cancelBroker.Execute(ctx, cancelRequest)
	requireValidResponse(t, cancelled)
	if cancelled.Error == nil || cancelled.Error.Code != codehostbroker.ErrorCancelled ||
		cancelled.Reconciliation == nil || cancelled.Reconciliation.Status != codehostbroker.ReconciliationNotApplied ||
		cancelFake.CreateAttempts() != 0 {
		t.Fatalf("cancelled response=%+v attempts=%d", cancelled, cancelFake.CreateAttempts())
	}
	retried := cancelBroker.Execute(context.Background(), cancelRequest)
	requireValidResponse(t, retried)
	if retried.Error != nil || cancelFake.CreateAttempts() != 1 {
		t.Fatalf("cancel retry=%+v attempts=%d", retried.Error, cancelFake.CreateAttempts())
	}
}

func TestCreateJournalCrashRecoveryRetentionAndRedaction(t *testing.T) {
	fixture, fake, broker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	request := preparedCreateRequest(t, fixture, broker, fixture.repository("acme/widgets"), "crash-in-progress")
	payload, payloadErr := decodeCreatePayload(request.Payload)
	if payloadErr != nil {
		t.Fatal(payloadErr)
	}
	journal := newMutationJournal(broker.projectRoot, broker.now)
	key := journalKeyDigest(request)
	if err := journal.withLock(func(document *journalDocument) error {
		timestamp := journal.timestamp()
		document.Entries[key] = &journalEntry{
			KeyDigest: key, PayloadDigest: canonicalCreateDigest(request, payload),
			OperationID: operationID(request), Target: mutationTarget{
				ConnectionID: request.ConnectionID, Repository: request.Repository,
				Base: payload.Base, Head: payload.Head,
			},
			State: journalInProgress, CreatedAt: timestamp, UpdatedAt: timestamp,
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	recovered := broker.Execute(context.Background(), request)
	requireValidResponse(t, recovered)
	if recovered.Error != nil || fake.CreateAttempts() != 1 {
		t.Fatalf("in-progress recovery=%+v attempts=%d", recovered.Error, fake.CreateAttempts())
	}

	dispatchedFixture, dispatchedFake, dispatchedBroker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	dispatched := preparedCreateRequest(t, dispatchedFixture, dispatchedBroker, dispatchedFixture.repository("acme/widgets"), "crash-dispatched")
	dispatchedPayload, _ := decodeCreatePayload(dispatched.Payload)
	dispatchedJournal := newMutationJournal(dispatchedBroker.projectRoot, dispatchedBroker.now)
	dispatchedKey := journalKeyDigest(dispatched)
	if err := dispatchedJournal.withLock(func(document *journalDocument) error {
		timestamp := dispatchedJournal.timestamp()
		document.Entries[dispatchedKey] = &journalEntry{
			KeyDigest: dispatchedKey, PayloadDigest: canonicalCreateDigest(dispatched, dispatchedPayload),
			OperationID: operationID(dispatched), Target: mutationTarget{
				ConnectionID: dispatched.ConnectionID, Repository: dispatched.Repository,
				Base: dispatchedPayload.Base, Head: dispatchedPayload.Head,
			},
			State: journalDispatched, CreatedAt: timestamp, UpdatedAt: timestamp, ProviderAttempts: 1,
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	uncertain := dispatchedBroker.Execute(context.Background(), dispatched)
	requireValidResponse(t, uncertain)
	if uncertain.Error == nil || uncertain.Error.Code != codehostbroker.ErrorAmbiguousResult ||
		dispatchedFake.CreateAttempts() != 0 {
		t.Fatalf("dispatched recovery=%+v attempts=%d", uncertain.Error, dispatchedFake.CreateAttempts())
	}

	now := time.Date(2026, 7, 27, 20, 31, 0, 0, time.UTC)
	retention := newMutationJournal(t.TempDir(), func() time.Time { return now })
	document := &journalDocument{Version: mutationJournalVersion, Entries: map[string]*journalEntry{
		"expired":   {State: journalApplied, UpdatedAt: now.Add(-31 * 24 * time.Hour).Format(time.RFC3339Nano)},
		"current":   {State: journalNotApplied, UpdatedAt: now.Format(time.RFC3339Nano)},
		"ambiguous": {State: journalAmbiguous, UpdatedAt: now.Add(-365 * 24 * time.Hour).Format(time.RFC3339Nano)},
		"progress":  {State: journalInProgress, UpdatedAt: now.Add(-365 * 24 * time.Hour).Format(time.RFC3339Nano)},
	}}
	retention.prune(document)
	if _, ok := document.Entries["expired"]; ok || len(document.Entries) != 3 {
		t.Fatalf("retention entries=%v", document.Entries)
	}
	if document.Entries["ambiguous"] == nil || document.Entries["progress"] == nil {
		t.Fatal("retention removed unresolved safety evidence")
	}
	assertJournalPrivateAndRedacted(t, broker.projectRoot)
}

func createTestBroker(t *testing.T, scenario mockcodehost.Scenario, repositories ...string) (brokerFixture, *mockcodehost.Server, *Broker) {
	t.Helper()
	fixture, fake, broker := testBroker(t, scenario, repositories...)
	broker.projectRoot = t.TempDir()
	return fixture, fake, broker
}

func createRequest(fixture brokerFixture, head codehostbroker.RepositoryIdentity, key string) codehostbroker.Request {
	base := fixture.repository(fixture.repositories[0])
	payload, _ := json.Marshal(codehostbroker.CreatePullRequestPayload{
		Base:  codehostbroker.RefIdentity{Repository: base, Name: "main", SHA: baseSHAForTest},
		Head:  codehostbroker.RefIdentity{Repository: head, Name: "feature/create", SHA: headSHAForTest},
		Title: createTitleCanary,
		Body:  createBodyCanary,
		Draft: false,
	})
	return codehostbroker.Request{
		Version: codehostbroker.Version, Operation: codehostbroker.OperationCreatePullRequest,
		Provider: "github", ConnectionID: fixture.connectionID, Repository: base,
		IntentSource: "user", Consent: codehostbroker.ConsentExplicitUser,
		IdempotencyKey: key, ReconciliationKey: "reconcile:" + key,
		CapabilityRevision: "prepare", ObservationRevision: "prepare",
		Payload: payload,
	}
}

func preparedCreateRequest(t *testing.T, fixture brokerFixture, broker *Broker, head codehostbroker.RepositoryIdentity, key string) codehostbroker.Request {
	t.Helper()
	request, err := broker.PrepareCreatePullRequest(context.Background(), createRequest(fixture, head, key))
	if err != nil {
		t.Fatalf("prepare create: %+v", err)
	}
	return request
}

func currentCapabilityRevision(t *testing.T, fixture brokerFixture, broker *Broker) string {
	t.Helper()
	response := broker.Execute(context.Background(), fixture.request(codehostbroker.OperationCapabilities))
	requireValidResponse(t, response)
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	return response.CapabilityRevision
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertJournalPrivateAndRedacted(t *testing.T, projectRoot string) {
	t.Helper()
	journal := newMutationJournal(projectRoot, time.Now)
	data, err := os.ReadFile(journal.path)
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(data)
	for _, forbidden := range []string{credentialCanary, createTitleCanary, createBodyCanary, "Authorization", "Bearer"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("journal contains forbidden material %q: %s", forbidden, encoded)
		}
	}
	for _, path := range []string{journal.directory, journal.path, journal.lockPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if path == journal.directory {
			if info.Mode().Perm()&0o077 != 0 {
				t.Fatalf("journal directory mode=%o", info.Mode().Perm())
			}
		} else if info.Mode().Perm()&0o077 != 0 {
			t.Fatalf("journal file %s mode=%o", filepath.Base(path), info.Mode().Perm())
		}
	}
}

func assertCreateErrorRedacted(t *testing.T, response codehostbroker.Response) {
	t.Helper()
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(data)
	for _, forbidden := range []string{credentialCanary, createTitleCanary, createBodyCanary, "Authorization", "Bearer"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("error response contains forbidden material %q: %s", forbidden, encoded)
		}
	}
}
