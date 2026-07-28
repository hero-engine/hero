package codehost

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/mockcodehost"
)

func TestMergeCapabilitiesAreRepositorySpecificAndQueueFailsClosed(t *testing.T) {
	fixture, _, broker := createTestBroker(t, mockcodehost.MergeMethodScenario("squash"), "acme/widgets")
	response := broker.Execute(context.Background(), fixture.request(codehostbroker.OperationCapabilities))
	requireValidResponse(t, response)
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	capability := mergeCapability(t, response)
	if !capability.Available || capability.Merge == nil ||
		!slices.Equal(capability.Merge.Methods, []string{"squash"}) ||
		capability.Merge.QueueSupported || capability.Merge.QueueRequired ||
		capability.Merge.Revision == "" {
		t.Fatalf("merge capability=%+v", capability)
	}

	queueFixture, queueFake, queueBroker := createTestBroker(t, mockcodehost.MergeQueueRequiredScenario(), "acme/widgets")
	queueResponse := queueBroker.Execute(context.Background(), queueFixture.request(codehostbroker.OperationCapabilities))
	requireValidResponse(t, queueResponse)
	queueCapability := mergeCapability(t, queueResponse)
	if queueCapability.Available || queueCapability.Merge == nil ||
		!queueCapability.Merge.QueueRequired || queueCapability.Merge.QueueSupported ||
		len(queueCapability.Merge.Methods) != 0 || queueCapability.Reason == "" {
		t.Fatalf("queue capability=%+v", queueCapability)
	}
	request := mergeRequest(t, queueFixture, "queue")
	if _, err := queueBroker.PrepareMerge(context.Background(), request); err == nil || err.Code != codehostbroker.ErrorUnsupportedOperation {
		t.Fatalf("queue preparation error=%+v", err)
	}
	if queueFake.MergeAttempts() != 0 {
		t.Fatal("queue-only repository dispatched a direct merge")
	}
}

func TestMergeRequiresExplicitAcceptanceBeforeProviderAccess(t *testing.T) {
	fixture, fake, broker := createTestBroker(t, mockcodehost.MergeScenario(), "acme/widgets")
	request := mergeRequest(t, fixture, "consent")
	request.Consent = codehostbroker.ConsentExplicitUser
	response := broker.Execute(context.Background(), request)
	requireValidResponse(t, response)
	if response.Error == nil || response.Error.Code != codehostbroker.ErrorInvalidInput ||
		fake.RequestCount() != 0 || fake.MergeAttempts() != 0 {
		t.Fatalf("consent response=%+v provider requests=%d", response, fake.RequestCount())
	}
}

func TestMergeDispatchesExactHeadAndReturnsTypedReceipt(t *testing.T) {
	fixture, fake, broker := createTestBroker(t, mockcodehost.MergeScenario(), "acme/widgets")
	request := preparedMergeRequest(t, fixture, broker, "direct")
	response := broker.Execute(context.Background(), request)
	requireValidResponse(t, response)
	if response.Error != nil || response.Reconciliation == nil ||
		response.Reconciliation.Status != codehostbroker.ReconciliationApplied ||
		response.Receipt == nil || response.Receipt.ProviderReceiptID == "" {
		t.Fatalf("merge response=%+v", response)
	}
	var result codehostbroker.MutationResult
	decodeResult(t, response, &result)
	if result.Outcome != "applied" || result.PullRequest.State != "merged" ||
		result.PullRequest.MergedAt == "" || result.Actor == nil || result.Actor.Login != "hero-user" ||
		!slices.Equal(result.InvalidatedOperations, []codehostbroker.Operation{
			codehostbroker.OperationGetPullRequest,
			codehostbroker.OperationGetMergeReadiness,
		}) {
		t.Fatalf("merge result=%+v", result)
	}
	requests := fake.MergeRequests()
	if len(requests) != 1 || requests[0].SHA != headSHAForTest ||
		requests[0].Method != "squash" ||
		requests[0].CommitTitle != "Merge exact head" ||
		requests[0].CommitMessage != "Verified by Hero" {
		t.Fatalf("provider merge requests=%+v", requests)
	}
}

func TestMergeSupportsEveryAdvertisedDirectMethod(t *testing.T) {
	for _, method := range []string{"merge", "squash", "rebase"} {
		t.Run(method, func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, mockcodehost.MergeMethodScenario(method), "acme/widgets")
			request := mergeRequest(t, fixture, "method-"+method)
			title, message := "Merge exact head", "Verified by Hero"
			if method == "rebase" {
				title, message = "", ""
			}
			request.Payload = mergePayloadJSON(t, fixture, method, title, message)
			prepared, err := broker.PrepareMerge(context.Background(), request)
			if err != nil {
				t.Fatalf("prepare %s: %+v", method, err)
			}
			response := broker.Execute(context.Background(), prepared)
			requireValidResponse(t, response)
			if response.Error != nil || fake.MergeAttempts() != 1 {
				t.Fatalf("%s response=%+v attempts=%d", method, response.Error, fake.MergeAttempts())
			}
		})
	}
}

func TestMergeReadinessUnknownBlockedPartialAndRateLimitedNeverDispatch(t *testing.T) {
	tests := []struct {
		name     string
		scenario mockcodehost.Scenario
		code     string
	}{
		{name: "checks_pending", scenario: mockcodehost.MergeReadinessScenario("checks_pending"), code: codehostbroker.ErrorConflict},
		{name: "checks_failure", scenario: mockcodehost.MergeReadinessScenario("checks_failure"), code: codehostbroker.ErrorConflict},
		{name: "reviews_required", scenario: mockcodehost.MergeReadinessScenario("reviews_required"), code: codehostbroker.ErrorConflict},
		{name: "changes_requested", scenario: mockcodehost.MergeReadinessScenario("changes_requested"), code: codehostbroker.ErrorConflict},
		{name: "mergeability_unknown", scenario: mockcodehost.MergeReadinessScenario("mergeability_unknown"), code: codehostbroker.ErrorConflict},
		{name: "permission_denied", scenario: mockcodehost.MergeReadinessScenario("permission_denied"), code: codehostbroker.ErrorConflict},
		{name: "queue_entry", scenario: mockcodehost.MergeReadinessScenario("queue_entry"), code: codehostbroker.ErrorConflict},
		{name: "draft", scenario: mockcodehost.MergeReadinessScenario("draft"), code: codehostbroker.ErrorConflict},
		{name: "blocked", scenario: mockcodehost.MergeReadinessScenario("blocked"), code: codehostbroker.ErrorConflict},
		{name: "protection_unknown", scenario: mockcodehost.MergeReadinessScenario("protection_unknown"), code: codehostbroker.ErrorConflict},
		{name: "head_changed", scenario: mockcodehost.MergeReadinessScenario("head_changed"), code: codehostbroker.ErrorPartialFailure},
		{name: "base_changed", scenario: mockcodehost.MergeReadinessScenario("base_changed"), code: codehostbroker.ErrorStaleObservation},
		{name: "base_changed_during_read", scenario: mockcodehost.MergeBaseChangeScenario(), code: codehostbroker.ErrorStaleObservation},
		{name: "partial", scenario: mergePartialScenario(), code: codehostbroker.ErrorPartialFailure},
		{name: "rate_limit", scenario: mockcodehost.MergeRateLimitScenario(), code: codehostbroker.ErrorRateLimited},
		{name: "queue_policy_partial", scenario: mockcodehost.MergeQueuePolicyPartialScenario(), code: codehostbroker.ErrorPartialFailure},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, test.scenario, "acme/widgets")
			_, err := broker.PrepareMerge(context.Background(), mergeRequest(t, fixture, test.name))
			if err == nil || err.Code != test.code {
				t.Fatalf("preparation error=%+v", err)
			}
			if fake.MergeAttempts() != 0 {
				t.Fatal("unproven readiness dispatched a merge")
			}
		})
	}
}

func TestMergeRejectsStaleCapabilityPermissionHeadAndProviderRace(t *testing.T) {
	fixture, fake, broker := createTestBroker(t, mockcodehost.MergeForcePushScenario(), "acme/widgets")
	request := preparedMergeRequest(t, fixture, broker, "force-push")
	response := broker.Execute(context.Background(), request)
	requireValidResponse(t, response)
	if response.Error == nil || response.Error.Code != codehostbroker.ErrorStaleObservation || fake.MergeAttempts() != 0 {
		t.Fatalf("force-push response=%+v attempts=%d", response.Error, fake.MergeAttempts())
	}

	capFixture, capFake, capBroker := createTestBroker(t, mockcodehost.MergeScenario(), "acme/widgets")
	stale := preparedMergeRequest(t, capFixture, capBroker, "stale-capability")
	stale.CapabilityRevision = "capability:stale"
	staleResponse := capBroker.Execute(context.Background(), stale)
	requireValidResponse(t, staleResponse)
	if staleResponse.Error == nil || staleResponse.Error.Code != codehostbroker.ErrorCapabilityChanged || capFake.MergeAttempts() != 0 {
		t.Fatalf("stale capability response=%+v attempts=%d", staleResponse.Error, capFake.MergeAttempts())
	}

	permissionFixture, permissionFake, permissionBroker := createTestBroker(t, mockcodehost.MergePermissionRaceScenario(), "acme/widgets")
	permission := preparedMergeRequest(t, permissionFixture, permissionBroker, "permission-race")
	permissionResponse := permissionBroker.Execute(context.Background(), permission)
	requireValidResponse(t, permissionResponse)
	if permissionResponse.Error == nil || permissionResponse.Error.Code != codehostbroker.ErrorForbidden || permissionFake.MergeAttempts() != 0 {
		t.Fatalf("permission response=%+v attempts=%d", permissionResponse.Error, permissionFake.MergeAttempts())
	}

	protectionFixture, protectionFake, protectionBroker := createTestBroker(t, mockcodehost.MergeReadinessRaceScenario("ready", "protection_unknown"), "acme/widgets")
	protection := preparedMergeRequest(t, protectionFixture, protectionBroker, "protection-race")
	protectionResponse := protectionBroker.Execute(context.Background(), protection)
	requireValidResponse(t, protectionResponse)
	if protectionResponse.Error == nil || protectionResponse.Error.Code != codehostbroker.ErrorConflict ||
		protectionFake.MergeAttempts() != 0 {
		t.Fatalf("protection response=%+v attempts=%d", protectionResponse.Error, protectionFake.MergeAttempts())
	}

	raceFixture, raceFake, raceBroker := createTestBroker(t, mockcodehost.MergeProviderForcePushScenario(), "acme/widgets")
	race := preparedMergeRequest(t, raceFixture, raceBroker, "provider-race")
	raceResponse := raceBroker.Execute(context.Background(), race)
	requireValidResponse(t, raceResponse)
	if raceResponse.Error == nil || raceResponse.Error.Code != codehostbroker.ErrorConflict ||
		raceResponse.Reconciliation == nil || raceResponse.Reconciliation.Status != codehostbroker.ReconciliationNotApplied ||
		raceFake.MergeAttempts() != 1 {
		t.Fatalf("provider race response=%+v attempts=%d", raceResponse, raceFake.MergeAttempts())
	}
}

func TestMergeExternalCompletionAndConflictingHead(t *testing.T) {
	fixture, fake, broker := createTestBroker(t, mockcodehost.MergeExternallyCompletedScenario(), "acme/widgets")
	request := preparedMergeRequest(t, fixture, broker, "external")
	response := broker.Execute(context.Background(), request)
	requireValidResponse(t, response)
	if response.Error != nil || response.Reconciliation == nil ||
		response.Reconciliation.Status != codehostbroker.ReconciliationExternallyCompleted ||
		fake.MergeAttempts() != 0 {
		t.Fatalf("external response=%+v attempts=%d", response, fake.MergeAttempts())
	}
	var result codehostbroker.MutationResult
	decodeResult(t, response, &result)
	if result.Actor != nil || result.Outcome != "externally_completed" {
		t.Fatalf("external result=%+v", result)
	}

	conflictFixture, conflictFake, conflictBroker := createTestBroker(t, mockcodehost.MergeConflictingHeadScenario(), "acme/widgets")
	_, err := conflictBroker.PrepareMerge(context.Background(), mergeRequest(t, conflictFixture, "conflicting-external"))
	if err == nil || err.Code != codehostbroker.ErrorConflict || conflictFake.MergeAttempts() != 0 {
		t.Fatalf("conflicting external error=%+v attempts=%d", err, conflictFake.MergeAttempts())
	}
}

func TestMergeLostResponseReconcilesAndDuplicateRetryDoesNotRedispatch(t *testing.T) {
	fixture, fake, broker := createTestBroker(t, mockcodehost.MergeLostResponseScenario(), "acme/widgets")
	request := preparedMergeRequest(t, fixture, broker, "lost-response")
	first := broker.Execute(context.Background(), request)
	second := broker.Execute(context.Background(), request)
	for index, response := range []codehostbroker.Response{first, second} {
		requireValidResponse(t, response)
		if response.Error != nil || response.Reconciliation == nil {
			t.Fatalf("response[%d]=%+v", index, response)
		}
	}
	if first.Reconciliation.Status != codehostbroker.ReconciliationReconciledApplied ||
		second.Reconciliation.Status != codehostbroker.ReconciliationReplayed ||
		fake.MergeAttempts() != 1 {
		t.Fatalf("statuses=%s,%s attempts=%d", first.Reconciliation.Status, second.Reconciliation.Status, fake.MergeAttempts())
	}
}

func TestMergeReplayReconcilesBeforeChangedRuntimePermission(t *testing.T) {
	scenario := mockcodehost.MergeScenario()
	scenario.Name = "merge-replay-after-permission-change"
	scenario.MergePermissionSequence = []bool{true, true, false}
	fixture, fake, broker := createTestBroker(t, scenario, "acme/widgets")
	request := preparedMergeRequest(t, fixture, broker, "permission-changed-after-apply")
	first := broker.Execute(context.Background(), request)
	second := broker.Execute(context.Background(), request)
	for index, response := range []codehostbroker.Response{first, second} {
		requireValidResponse(t, response)
		if response.Error != nil || response.Reconciliation == nil {
			t.Fatalf("response[%d]=%+v", index, response)
		}
	}
	if first.Reconciliation.Status != codehostbroker.ReconciliationApplied ||
		second.Reconciliation.Status != codehostbroker.ReconciliationReplayed ||
		fake.MergeAttempts() != 1 {
		t.Fatalf("statuses=%s,%s attempts=%d", first.Reconciliation.Status, second.Reconciliation.Status, fake.MergeAttempts())
	}
}

func TestMergeAmbiguityCancellationAndIdempotencyConflictRemainBounded(t *testing.T) {
	beforeFixture, beforeFake, beforeBroker := createTestBroker(t, mockcodehost.MergeScenario(), "acme/widgets")
	beforeRequest := preparedMergeRequest(t, beforeFixture, beforeBroker, "cancelled-before")
	beforeContext, beforeCancel := context.WithCancel(context.Background())
	beforeCancel()
	beforeResponse := beforeBroker.Execute(beforeContext, beforeRequest)
	requireValidResponse(t, beforeResponse)
	if beforeResponse.Error == nil || beforeResponse.Error.Code != codehostbroker.ErrorCancelled ||
		beforeFake.MergeAttempts() != 0 {
		t.Fatalf("pre-dispatch cancel response=%+v attempts=%d", beforeResponse, beforeFake.MergeAttempts())
	}

	fixture, fake, broker := createTestBroker(t, mockcodehost.MergeAmbiguousScenario(), "acme/widgets")
	request := preparedMergeRequest(t, fixture, broker, "ambiguous")
	first := broker.Execute(context.Background(), request)
	second := broker.Execute(context.Background(), request)
	for _, response := range []codehostbroker.Response{first, second} {
		requireValidResponse(t, response)
		if response.Error == nil || response.Error.Code != codehostbroker.ErrorAmbiguousResult ||
			response.Reconciliation == nil || response.Reconciliation.Status != codehostbroker.ReconciliationAmbiguous {
			t.Fatalf("ambiguous response=%+v", response)
		}
	}
	if fake.MergeAttempts() != 1 {
		t.Fatalf("ambiguous retry dispatched %d merges", fake.MergeAttempts())
	}

	cancelFixture, cancelFake, cancelBroker := createTestBroker(t, mockcodehost.MergeCancelledAfterApplyScenario(500*time.Millisecond), "acme/widgets")
	cancelRequest := preparedMergeRequest(t, cancelFixture, cancelBroker, "cancelled")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	cancelResponse := cancelBroker.Execute(ctx, cancelRequest)
	requireValidResponse(t, cancelResponse)
	if cancelResponse.Error != nil || cancelResponse.Reconciliation == nil ||
		cancelResponse.Reconciliation.Status != codehostbroker.ReconciliationReconciledApplied ||
		cancelFake.MergeAttempts() != 1 {
		t.Fatalf("cancel response=%+v attempts=%d", cancelResponse, cancelFake.MergeAttempts())
	}

	conflictFixture, conflictFake, conflictBroker := createTestBroker(t, mockcodehost.MergeScenario(), "acme/widgets")
	original := preparedMergeRequest(t, conflictFixture, conflictBroker, "same-key")
	if response := conflictBroker.Execute(context.Background(), original); response.Error != nil {
		t.Fatalf("first merge=%+v", response.Error)
	}
	changed := original
	changed.Payload = mergePayloadJSON(t, conflictFixture, "merge", "changed title", "")
	response := conflictBroker.Execute(context.Background(), changed)
	requireValidResponse(t, response)
	if response.Error == nil || response.Error.Code != codehostbroker.ErrorIdempotencyConflict || conflictFake.MergeAttempts() != 1 {
		t.Fatalf("idempotency response=%+v attempts=%d", response, conflictFake.MergeAttempts())
	}
	encoded, _ := json.Marshal(response)
	if strings.Contains(string(encoded), "changed title") {
		t.Fatal("mutation response exposed commit text")
	}
}

func mergePartialScenario() mockcodehost.Scenario {
	scenario := mockcodehost.MergeScenario()
	scenario.Name = "merge-partial"
	scenario.GraphQLPartial = true
	return scenario
}

func mergeCapability(t *testing.T, response codehostbroker.Response) codehostbroker.Capability {
	t.Helper()
	var result codehostbroker.CapabilitiesResult
	decodeResult(t, response, &result)
	for _, capability := range result.Capabilities {
		if capability.Policy.Operation == codehostbroker.OperationMerge {
			return capability
		}
	}
	t.Fatal("merge capability missing")
	return codehostbroker.Capability{}
}

func mergeRequest(t *testing.T, fixture brokerFixture, key string) codehostbroker.Request {
	t.Helper()
	request := fixture.request(codehostbroker.OperationMerge)
	request.IntentSource = "user"
	request.Consent = codehostbroker.ConsentExplicitAcceptance
	request.IdempotencyKey = "merge:" + key
	request.ReconciliationKey = "reconcile:merge:" + key
	request.CapabilityRevision = "prepare"
	request.ObservationRevision = "prepare"
	request.Payload = mergePayloadJSON(t, fixture, "squash", "Merge exact head", "Verified by Hero")
	return request
}

func preparedMergeRequest(t *testing.T, fixture brokerFixture, broker *Broker, key string) codehostbroker.Request {
	t.Helper()
	request, err := broker.PrepareMerge(context.Background(), mergeRequest(t, fixture, key))
	if err != nil {
		t.Fatalf("prepare merge: %+v", err)
	}
	return request
}

func mergePayloadJSON(t *testing.T, fixture brokerFixture, method, title, message string) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(codehostbroker.MergePayload{
		ExpectedHeadSHA: headSHAForTest,
		ObservedBase: codehostbroker.RefIdentity{
			Repository: fixture.repository("acme/widgets"),
			Name:       "main",
			SHA:        baseSHAForTest,
		},
		Method: method, CommitTitle: title, CommitMessage: message,
	})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}
