package codehost

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/mockcodehost"
)

const (
	releaseSHAForTest = "dddddddddddddddddddddddddddddddddddddddd"
	forcedSHAForTest  = "cccccccccccccccccccccccccccccccccccccccc"
)

var stateTransitionOperations = []codehostbroker.Operation{
	codehostbroker.OperationMarkReady,
	codehostbroker.OperationRetarget,
	codehostbroker.OperationClose,
	codehostbroker.OperationReopen,
}

func TestStateTransitionOperationsAdvertisedAppliedAndReplayed(t *testing.T) {
	for _, operation := range stateTransitionOperations {
		t.Run(string(operation), func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, mockcodehost.StateScenario(string(operation)), "acme/widgets")
			readiness := broker.Execute(context.Background(), fixture.request(codehostbroker.OperationGetMergeReadiness))
			requireValidResponse(t, readiness)
			if readiness.Error != nil {
				t.Fatalf("readiness baseline=%+v", readiness)
			}
			request := preparedStateTransitionRequest(t, fixture, broker, operation, "state-success-"+string(operation))
			preTransitionRevision := request.ObservationRevision
			response := broker.Execute(context.Background(), request)
			requireValidResponse(t, response)
			if response.Error != nil || response.Receipt == nil || response.Reconciliation == nil ||
				response.Reconciliation.Status != codehostbroker.ReconciliationApplied ||
				response.Policy.Effect != codehostbroker.EffectExternalWrite ||
				response.Policy.Consent != codehostbroker.ConsentExplicitUser ||
				response.ObservationRevision == preTransitionRevision ||
				response.ObservationRevision == readiness.ObservationRevision ||
				fake.StateAttempts() != 1 {
				t.Fatalf("response=%+v attempts=%d", response, fake.StateAttempts())
			}
			var result codehostbroker.MutationResult
			decodeResult(t, response, &result)
			if result.Outcome != "applied" || result.Actor == nil ||
				result.Actor.Login != "hero-user" || result.Actor.ProviderID != "U_99" ||
				!stateTransitionDesired(operation, result.PullRequest, stateDesiredBase(fixture, operation)) ||
				len(result.InvalidatedOperations) != 1 ||
				result.InvalidatedOperations[0] != codehostbroker.OperationGetMergeReadiness ||
				response.Receipt.ProviderReceiptID != "PR_42" {
				t.Fatalf("result=%+v receipt=%+v", result, response.Receipt)
			}
			replayed := broker.Execute(context.Background(), request)
			requireValidResponse(t, replayed)
			if replayed.Error != nil || replayed.Reconciliation == nil ||
				replayed.Reconciliation.Status != codehostbroker.ReconciliationReplayed ||
				replayed.Receipt.ProviderReceiptID != response.Receipt.ProviderReceiptID ||
				replayed.ObservationRevision != response.ObservationRevision ||
				fake.StateAttempts() != 1 {
				t.Fatalf("replayed=%+v attempts=%d", replayed, fake.StateAttempts())
			}
			assertStateTransitionJournal(t, broker.projectRoot, request)
		})
	}

	fixture, _, broker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	response := broker.Execute(context.Background(), fixture.request(codehostbroker.OperationCapabilities))
	requireValidResponse(t, response)
	var capabilities codehostbroker.CapabilitiesResult
	decodeResult(t, response, &capabilities)
	available := map[codehostbroker.Operation]codehostbroker.Capability{}
	for _, capability := range capabilities.Capabilities {
		available[capability.Policy.Operation] = capability
	}
	for _, operation := range stateTransitionOperations {
		capability := available[operation]
		if !capability.Available ||
			capability.Policy.Effect != codehostbroker.EffectExternalWrite ||
			capability.Policy.Consent != codehostbroker.ConsentExplicitUser {
			t.Fatalf("capability %s=%+v", operation, capability)
		}
	}
}

func TestStateTransitionExternallyCompletedAndMergedTerminal(t *testing.T) {
	for _, operation := range stateTransitionOperations {
		t.Run("external_"+string(operation), func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, mockcodehost.StateExternallyCompletedScenario(string(operation)), "acme/widgets")
			request := stateTransitionRequest(fixture, operation, "state-external-"+string(operation))
			if operation == codehostbroker.OperationRetarget {
				request.Payload = retargetPayload(t, fixture, headSHAForTest, "release", releaseSHAForTest, "release", releaseSHAForTest)
			}
			prepared, err := broker.PrepareStateTransition(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			response := broker.Execute(context.Background(), prepared)
			requireValidResponse(t, response)
			if response.Error != nil || response.Reconciliation == nil ||
				response.Reconciliation.Status != codehostbroker.ReconciliationExternallyCompleted ||
				fake.StateAttempts() != 0 {
				t.Fatalf("external=%+v attempts=%d", response, fake.StateAttempts())
			}
			replayed := broker.Execute(context.Background(), prepared)
			requireValidResponse(t, replayed)
			if replayed.Error != nil || replayed.Reconciliation.Status != codehostbroker.ReconciliationReplayed ||
				fake.StateAttempts() != 0 {
				t.Fatalf("external replay=%+v attempts=%d", replayed, fake.StateAttempts())
			}
		})

		t.Run("merged_"+string(operation), func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, mockcodehost.StateMergedScenario(string(operation)), "acme/widgets")
			request := stateTransitionRequest(fixture, operation, "state-merged-"+string(operation))
			if _, err := broker.PrepareStateTransition(context.Background(), request); err == nil ||
				err.Code != codehostbroker.ErrorConflict {
				t.Fatalf("merged error=%+v", err)
			}
			if fake.StateAttempts() != 0 {
				t.Fatal("merged PR dispatched")
			}
		})
	}

	for _, test := range []struct {
		state string
		draft bool
		want  string
	}{
		{"open", true, "open_draft"},
		{"open", false, "open_ready"},
		{"closed", false, "closed_unmerged"},
		{"merged", false, "merged"},
	} {
		pullRequest := codehostbroker.PullRequest{State: test.state, Draft: test.draft}
		if got := normalizedLifecycleState(pullRequest); got != test.want {
			t.Fatalf("state=%s draft=%t got=%s want=%s", test.state, test.draft, got, test.want)
		}
	}

	for _, operation := range []codehostbroker.Operation{
		codehostbroker.OperationMarkReady,
		codehostbroker.OperationRetarget,
	} {
		t.Run("closed_invalid_"+string(operation), func(t *testing.T) {
			scenario := mockcodehost.StateScenario("reopen")
			scenario.Name = "state-closed-invalid-" + string(operation)
			fixture, fake, broker := createTestBroker(t, scenario, "acme/widgets")
			request := stateTransitionRequest(fixture, operation, "state-closed-invalid-"+string(operation))
			if _, err := broker.PrepareStateTransition(context.Background(), request); err == nil ||
				err.Code != codehostbroker.ErrorConflict {
				t.Fatalf("closed transition error=%+v", err)
			}
			if fake.StateAttempts() != 0 {
				t.Fatal("invalid closed transition dispatched")
			}
		})
	}
}

func TestStateTransitionDesiredStateMatrix(t *testing.T) {
	repository := codehostbroker.RepositoryIdentity{Host: "github.com", Owner: "acme", Name: "widgets", FullName: "acme/widgets"}
	main := codehostbroker.RefIdentity{Repository: repository, Name: "main", SHA: baseSHAForTest}
	release := codehostbroker.RefIdentity{Repository: repository, Name: "release", SHA: releaseSHAForTest}
	for _, test := range []struct {
		name      string
		operation codehostbroker.Operation
		state     string
		draft     bool
		base      codehostbroker.RefIdentity
		desired   *codehostbroker.RefIdentity
		want      bool
	}{
		{"mark_draft", codehostbroker.OperationMarkReady, "open", true, main, nil, false},
		{"mark_ready", codehostbroker.OperationMarkReady, "open", false, main, nil, true},
		{"mark_closed", codehostbroker.OperationMarkReady, "closed", false, main, nil, false},
		{"mark_merged", codehostbroker.OperationMarkReady, "merged", false, main, nil, false},
		{"retarget_draft_pending", codehostbroker.OperationRetarget, "open", true, main, &release, false},
		{"retarget_ready_pending", codehostbroker.OperationRetarget, "open", false, main, &release, false},
		{"retarget_draft_done", codehostbroker.OperationRetarget, "open", true, release, &release, true},
		{"retarget_ready_done", codehostbroker.OperationRetarget, "open", false, release, &release, true},
		{"retarget_moved_target", codehostbroker.OperationRetarget, "open", false, codehostbroker.RefIdentity{Repository: repository, Name: "release", SHA: forcedSHAForTest}, &release, false},
		{"retarget_closed", codehostbroker.OperationRetarget, "closed", false, release, &release, false},
		{"retarget_merged", codehostbroker.OperationRetarget, "merged", false, release, &release, false},
		{"close_open_draft", codehostbroker.OperationClose, "open", true, main, nil, false},
		{"close_open_ready", codehostbroker.OperationClose, "open", false, main, nil, false},
		{"close_closed", codehostbroker.OperationClose, "closed", false, main, nil, true},
		{"close_merged", codehostbroker.OperationClose, "merged", false, main, nil, false},
		{"reopen_open_draft", codehostbroker.OperationReopen, "open", true, main, nil, true},
		{"reopen_open_ready", codehostbroker.OperationReopen, "open", false, main, nil, true},
		{"reopen_closed", codehostbroker.OperationReopen, "closed", false, main, nil, false},
		{"reopen_merged", codehostbroker.OperationReopen, "merged", false, main, nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			pullRequest := codehostbroker.PullRequest{State: test.state, Draft: test.draft, Base: test.base}
			if got := stateTransitionDesired(test.operation, pullRequest, test.desired); got != test.want {
				t.Fatalf("desired=%t want=%t pull_request=%+v", got, test.want, pullRequest)
			}
		})
	}
}

func TestStateTransitionStalePermissionAndTargetGates(t *testing.T) {
	for _, operation := range stateTransitionOperations {
		t.Run("force_push_"+string(operation), func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, mockcodehost.StateForcePushScenario(string(operation)), "acme/widgets")
			request := preparedStateTransitionRequest(t, fixture, broker, operation, "state-force-push-"+string(operation))
			response := broker.Execute(context.Background(), request)
			requireValidResponse(t, response)
			if response.Error == nil || response.Error.Code != codehostbroker.ErrorStaleObservation ||
				fake.StateAttempts() != 0 {
				t.Fatalf("force push=%+v attempts=%d", response, fake.StateAttempts())
			}
		})

		t.Run("permission_race_"+string(operation), func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, mockcodehost.StatePermissionRaceScenario(string(operation)), "acme/widgets")
			request := preparedStateTransitionRequest(t, fixture, broker, operation, "state-permission-"+string(operation))
			response := broker.Execute(context.Background(), request)
			requireValidResponse(t, response)
			if response.Error == nil || response.Error.Code != codehostbroker.ErrorForbidden ||
				fake.StateAttempts() != 0 {
				t.Fatalf("permission=%+v attempts=%d", response, fake.StateAttempts())
			}
		})
	}

	baseFixture, baseFake, baseBroker := createTestBroker(t, mockcodehost.StateBaseChangesScenario(), "acme/widgets")
	baseRequest := preparedStateTransitionRequest(t, baseFixture, baseBroker, codehostbroker.OperationRetarget, "state-base-change")
	baseResponse := baseBroker.Execute(context.Background(), baseRequest)
	requireValidResponse(t, baseResponse)
	if baseResponse.Error == nil || baseResponse.Error.Code != codehostbroker.ErrorStaleObservation ||
		baseFake.StateAttempts() != 0 {
		t.Fatalf("base change=%+v attempts=%d", baseResponse, baseFake.StateAttempts())
	}

	targetFixture, targetFake, targetBroker := createTestBroker(t, mockcodehost.StateTargetMovesScenario(), "acme/widgets")
	targetRequest := preparedStateTransitionRequest(t, targetFixture, targetBroker, codehostbroker.OperationRetarget, "state-target-move")
	targetResponse := targetBroker.Execute(context.Background(), targetRequest)
	requireValidResponse(t, targetResponse)
	if targetResponse.Error == nil || targetResponse.Error.Code != codehostbroker.ErrorStaleObservation ||
		targetFake.StateAttempts() != 0 {
		t.Fatalf("target move=%+v attempts=%d", targetResponse, targetFake.StateAttempts())
	}

	postWriteFixture, postWriteFake, postWriteBroker := createTestBroker(t, mockcodehost.StateTargetMovesAfterWriteScenario(), "acme/widgets")
	postWriteRequest := preparedStateTransitionRequest(t, postWriteFixture, postWriteBroker, codehostbroker.OperationRetarget, "state-target-move-after-write")
	postWriteResponse := postWriteBroker.Execute(context.Background(), postWriteRequest)
	requireValidResponse(t, postWriteResponse)
	if postWriteResponse.Error == nil || postWriteResponse.Error.Code != codehostbroker.ErrorAmbiguousResult ||
		postWriteResponse.Reconciliation == nil || postWriteResponse.Reconciliation.Status != codehostbroker.ReconciliationAmbiguous ||
		postWriteFake.StateAttempts() != 1 {
		t.Fatalf("target move after write=%+v attempts=%d", postWriteResponse, postWriteFake.StateAttempts())
	}

	missingFixture, missingFake, missingBroker := createTestBroker(t, mockcodehost.StateMissingTargetBranchScenario(), "acme/widgets")
	missing := stateTransitionRequest(missingFixture, codehostbroker.OperationRetarget, "state-target-missing")
	if _, err := missingBroker.PrepareStateTransition(context.Background(), missing); err == nil ||
		err.Code != codehostbroker.ErrorNotFound {
		t.Fatalf("missing target error=%+v", err)
	}
	if missingFake.StateAttempts() != 0 {
		t.Fatal("missing target dispatched")
	}

	revisionFixture, revisionFake, revisionBroker := createTestBroker(t, mockcodehost.StateScenario("close"), "acme/widgets")
	revision := preparedStateTransitionRequest(t, revisionFixture, revisionBroker, codehostbroker.OperationClose, "state-revisions")
	staleObservation := revision
	staleObservation.ObservationRevision = "observation:stale"
	observationResponse := revisionBroker.Execute(context.Background(), staleObservation)
	requireValidResponse(t, observationResponse)
	if observationResponse.Error == nil || observationResponse.Error.Code != codehostbroker.ErrorStaleObservation ||
		revisionFake.StateAttempts() != 0 {
		t.Fatalf("observation=%+v attempts=%d", observationResponse, revisionFake.StateAttempts())
	}
	staleCapability := revision
	staleCapability.IdempotencyKey = "state-capability"
	staleCapability.ReconciliationKey = "reconcile:state-capability"
	staleCapability.CapabilityRevision = "capability:stale"
	capabilityResponse := revisionBroker.Execute(context.Background(), staleCapability)
	requireValidResponse(t, capabilityResponse)
	if capabilityResponse.Error == nil || capabilityResponse.Error.Code != codehostbroker.ErrorCapabilityChanged ||
		revisionFake.StateAttempts() != 0 {
		t.Fatalf("capability=%+v attempts=%d", capabilityResponse, revisionFake.StateAttempts())
	}
}

func TestStateTransitionLostAmbiguousDelayedAndCancellation(t *testing.T) {
	for _, operation := range stateTransitionOperations {
		t.Run("lost_"+string(operation), func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, mockcodehost.StateLostResponseScenario(string(operation)), "acme/widgets")
			request := preparedStateTransitionRequest(t, fixture, broker, operation, "state-lost-"+string(operation))
			response := broker.Execute(context.Background(), request)
			requireValidResponse(t, response)
			if response.Error != nil || response.Reconciliation == nil ||
				response.Reconciliation.Status != codehostbroker.ReconciliationReconciledApplied ||
				response.ObservationRevision == request.ObservationRevision ||
				fake.StateAttempts() != 1 {
				t.Fatalf("lost=%+v attempts=%d", response, fake.StateAttempts())
			}
		})

		t.Run("ambiguous_"+string(operation), func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, mockcodehost.StateAmbiguousScenario(string(operation)), "acme/widgets")
			request := preparedStateTransitionRequest(t, fixture, broker, operation, "state-ambiguous-"+string(operation))
			first := broker.Execute(context.Background(), request)
			second := broker.Execute(context.Background(), request)
			requireValidResponse(t, first)
			requireValidResponse(t, second)
			for _, response := range []codehostbroker.Response{first, second} {
				if response.Error == nil || response.Error.Code != codehostbroker.ErrorAmbiguousResult ||
					response.Reconciliation == nil || response.Reconciliation.Status != codehostbroker.ReconciliationAmbiguous {
					t.Fatalf("ambiguous=%+v", response)
				}
			}
			if fake.StateAttempts() != 1 {
				t.Fatalf("ambiguous attempts=%d", fake.StateAttempts())
			}
		})

		t.Run("cancel_after_"+string(operation), func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, mockcodehost.StateCancelledAfterApplyScenario(string(operation), cancelAfterApplyResponseDelay), "acme/widgets")
			request := preparedStateTransitionRequest(t, fixture, broker, operation, "state-cancel-after-"+string(operation))
			response := executeThenCancelAfterAttempt(t, broker, request, fake.StateAttempts)
			requireValidResponse(t, response)
			if response.Error != nil || response.Reconciliation == nil ||
				response.Reconciliation.Status != codehostbroker.ReconciliationReconciledApplied ||
				fake.StateAttempts() != 1 {
				t.Fatalf("cancel after=%+v attempts=%d", response, fake.StateAttempts())
			}
		})
	}

	fixture, fake, broker := createTestBroker(t, mockcodehost.StateDelayedVisibilityScenario("retarget", 1), "acme/widgets")
	request := preparedStateTransitionRequest(t, fixture, broker, codehostbroker.OperationRetarget, "state-delayed")
	first := broker.Execute(context.Background(), request)
	second := broker.Execute(context.Background(), request)
	requireValidResponse(t, first)
	requireValidResponse(t, second)
	if first.Error == nil || first.Error.Code != codehostbroker.ErrorAmbiguousResult ||
		second.Error != nil || second.Reconciliation.Status != codehostbroker.ReconciliationReconciledApplied ||
		fake.StateAttempts() != 1 {
		t.Fatalf("delayed=%+v/%+v attempts=%d", first, second, fake.StateAttempts())
	}

	cancelFixture, cancelFake, cancelBroker := createTestBroker(t, mockcodehost.StateScenario("close"), "acme/widgets")
	cancelRequest := preparedStateTransitionRequest(t, cancelFixture, cancelBroker, codehostbroker.OperationClose, "state-cancel-before")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled := cancelBroker.Execute(ctx, cancelRequest)
	requireValidResponse(t, cancelled)
	if cancelled.Error == nil || cancelled.Error.Code != codehostbroker.ErrorCancelled ||
		cancelled.Reconciliation.Status != codehostbroker.ReconciliationNotApplied ||
		cancelFake.StateAttempts() != 0 {
		t.Fatalf("cancel before=%+v attempts=%d", cancelled, cancelFake.StateAttempts())
	}
}

func TestStateTransitionDuplicateConcurrentConflictAndProviderRejection(t *testing.T) {
	for _, operation := range stateTransitionOperations {
		t.Run("concurrent_"+string(operation), func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, mockcodehost.StateScenario(string(operation)), "acme/widgets")
			request := preparedStateTransitionRequest(t, fixture, broker, operation, "state-concurrent-"+string(operation))
			responses := make([]codehostbroker.Response, 2)
			var wait sync.WaitGroup
			for index := range responses {
				wait.Add(1)
				go func() {
					defer wait.Done()
					responses[index] = broker.Execute(context.Background(), request)
				}()
			}
			wait.Wait()
			for _, response := range responses {
				requireValidResponse(t, response)
				if response.Error != nil {
					t.Fatalf("concurrent response=%+v", response)
				}
			}
			if fake.StateAttempts() != 1 ||
				responses[0].Receipt.ProviderReceiptID != responses[1].Receipt.ProviderReceiptID {
				t.Fatalf("attempts=%d responses=%+v", fake.StateAttempts(), responses)
			}
		})
	}

	fixture, fake, broker := createTestBroker(t, mockcodehost.StateScenario("retarget"), "acme/widgets")
	request := preparedStateTransitionRequest(t, fixture, broker, codehostbroker.OperationRetarget, "state-conflict")
	first := broker.Execute(context.Background(), request)
	requireValidResponse(t, first)
	for name, mutate := range map[string]func(*codehostbroker.Request){
		"operation": func(value *codehostbroker.Request) {
			value.Operation = codehostbroker.OperationClose
			value.Payload = lifecyclePayload(t, headSHAForTest)
		},
		"pr": func(value *codehostbroker.Request) {
			identity := *value.PullRequest
			identity.ProviderID, identity.Number = "PR_43", 43
			value.PullRequest = &identity
		},
		"head": func(value *codehostbroker.Request) {
			value.Payload = retargetPayload(t, fixture, forcedSHAForTest, "main", baseSHAForTest, "release", releaseSHAForTest)
		},
		"current_base": func(value *codehostbroker.Request) {
			value.Payload = retargetPayload(t, fixture, headSHAForTest, "develop", forcedSHAForTest, "release", releaseSHAForTest)
		},
		"desired_base": func(value *codehostbroker.Request) {
			value.Payload = retargetPayload(t, fixture, headSHAForTest, "main", baseSHAForTest, "other", forcedSHAForTest)
		},
	} {
		t.Run("conflict_"+name, func(t *testing.T) {
			candidate := request
			mutate(&candidate)
			response := broker.Execute(context.Background(), candidate)
			requireValidResponse(t, response)
			if response.Error == nil || response.Error.Code != codehostbroker.ErrorIdempotencyConflict ||
				fake.StateAttempts() != 1 {
				t.Fatalf("conflict=%+v attempts=%d", response, fake.StateAttempts())
			}
		})
	}

	deniedFixture, deniedFake, deniedBroker := createTestBroker(t, mockcodehost.StateWriteDeniedScenario("close"), "acme/widgets")
	denied := preparedStateTransitionRequest(t, deniedFixture, deniedBroker, codehostbroker.OperationClose, "state-denied")
	deniedFirst := deniedBroker.Execute(context.Background(), denied)
	deniedSecond := deniedBroker.Execute(context.Background(), denied)
	requireValidResponse(t, deniedFirst)
	requireValidResponse(t, deniedSecond)
	if deniedFirst.Error == nil || deniedFirst.Error.Code != codehostbroker.ErrorForbidden ||
		deniedSecond.Error == nil || deniedSecond.Error.Code != codehostbroker.ErrorForbidden ||
		deniedFake.StateAttempts() != 1 {
		t.Fatalf("denied=%+v/%+v attempts=%d", deniedFirst, deniedSecond, deniedFake.StateAttempts())
	}
}

func TestStateTransitionRequiredFieldsAndScope(t *testing.T) {
	fixture, fake, broker := createTestBroker(t, mockcodehost.StateScenario("close"), "acme/widgets")
	request := preparedStateTransitionRequest(t, fixture, broker, codehostbroker.OperationClose, "state-required")
	for name, mutate := range map[string]func(*codehostbroker.Request){
		"pull_request":         func(value *codehostbroker.Request) { value.PullRequest = nil },
		"intent":               func(value *codehostbroker.Request) { value.IntentSource = "" },
		"idempotency":          func(value *codehostbroker.Request) { value.IdempotencyKey = "" },
		"capability_revision":  func(value *codehostbroker.Request) { value.CapabilityRevision = "" },
		"observation_revision": func(value *codehostbroker.Request) { value.ObservationRevision = "" },
		"reconciliation":       func(value *codehostbroker.Request) { value.ReconciliationKey = "" },
		"head":                 func(value *codehostbroker.Request) { value.Payload = lifecyclePayload(t, "") },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := request
			mutate(&candidate)
			response := broker.Execute(context.Background(), candidate)
			requireValidResponse(t, response)
			if response.Error == nil || fake.StateAttempts() != 0 {
				t.Fatalf("%s=%+v attempts=%d", name, response, fake.StateAttempts())
			}
		})
	}

	retargetFixture, retargetFake, retargetBroker := createTestBroker(t, mockcodehost.StateScenario("retarget"), "acme/widgets")
	retarget := stateTransitionRequest(retargetFixture, codehostbroker.OperationRetarget, "state-scope")
	other := retargetFixture.repository("other/widgets")
	payload := codehostbroker.RetargetPayload{
		ExpectedHeadSHA: headSHAForTest,
		CurrentBase:     codehostbroker.RefIdentity{Repository: retargetFixture.repository("acme/widgets"), Name: "main", SHA: baseSHAForTest},
		NewBase:         codehostbroker.RefIdentity{Repository: other, Name: "release", SHA: releaseSHAForTest},
	}
	retarget.Payload = mustJSON(t, payload)
	response := retargetBroker.Execute(context.Background(), retarget)
	requireValidResponse(t, response)
	if response.Error == nil || response.Error.Code != codehostbroker.ErrorInvalidInput ||
		retargetFake.StateAttempts() != 0 {
		t.Fatalf("scope=%+v attempts=%d", response, retargetFake.StateAttempts())
	}
}

func stateTransitionRequest(fixture brokerFixture, operation codehostbroker.Operation, key string) codehostbroker.Request {
	request := fixture.request(operation)
	request.IntentSource = "user"
	request.Consent = codehostbroker.ConsentExplicitUser
	request.IdempotencyKey = key
	request.ReconciliationKey = "reconcile:" + key
	request.CapabilityRevision = "prepare"
	request.ObservationRevision = "prepare"
	if operation == codehostbroker.OperationRetarget {
		request.Payload = retargetPayload(nil, fixture, headSHAForTest, "main", baseSHAForTest, "release", releaseSHAForTest)
	} else {
		request.Payload = lifecyclePayload(nil, headSHAForTest)
	}
	return request
}

func preparedStateTransitionRequest(t *testing.T, fixture brokerFixture, broker *Broker, operation codehostbroker.Operation, key string) codehostbroker.Request {
	t.Helper()
	request, err := broker.PrepareStateTransition(context.Background(), stateTransitionRequest(fixture, operation, key))
	if err != nil {
		t.Fatalf("prepare %s: %+v", operation, err)
	}
	return request
}

func lifecyclePayload(t *testing.T, head string) json.RawMessage {
	if t != nil {
		t.Helper()
	}
	data, err := json.Marshal(codehostbroker.LifecyclePayload{ExpectedHeadSHA: head})
	if err != nil && t != nil {
		t.Fatal(err)
	}
	return data
}

func retargetPayload(t *testing.T, fixture brokerFixture, head, currentName, currentSHA, desiredName, desiredSHA string) json.RawMessage {
	if t != nil {
		t.Helper()
	}
	repository := fixture.repository("acme/widgets")
	data, err := json.Marshal(codehostbroker.RetargetPayload{
		ExpectedHeadSHA: head,
		CurrentBase:     codehostbroker.RefIdentity{Repository: repository, Name: currentName, SHA: currentSHA},
		NewBase:         codehostbroker.RefIdentity{Repository: repository, Name: desiredName, SHA: desiredSHA},
	})
	if err != nil && t != nil {
		t.Fatal(err)
	}
	return data
}

func stateDesiredBase(fixture brokerFixture, operation codehostbroker.Operation) *codehostbroker.RefIdentity {
	if operation != codehostbroker.OperationRetarget {
		return nil
	}
	value := codehostbroker.RefIdentity{
		Repository: fixture.repository("acme/widgets"),
		Name:       "release",
		SHA:        releaseSHAForTest,
	}
	return &value
}

func assertStateTransitionJournal(t *testing.T, projectRoot string, request codehostbroker.Request) {
	t.Helper()
	journal := newMutationJournal(projectRoot, time.Now)
	document, err := journal.load()
	if err != nil {
		t.Fatal(err)
	}
	entry := document.Entries[journalKeyDigest(request)]
	if entry == nil || entry.Receipt == nil || entry.Receipt.ProviderReceiptID != "PR_42" ||
		entry.Receipt.Actor == nil || entry.Receipt.Actor.Login != "hero-user" ||
		entry.Receipt.HeadSHA != headSHAForTest || entry.Receipt.State == "" ||
		entry.Target.PullRequest == nil || entry.Target.ExpectedHeadSHA != headSHAForTest {
		t.Fatalf("journal entry=%+v", entry)
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{credentialCanary, "Authorization", "Bearer"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("journal leaked %q: %s", forbidden, encoded)
		}
	}
}
