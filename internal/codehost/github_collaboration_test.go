package codehost

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/mockcodehost"
)

const (
	collaborationBodyCanary = "COLLABORATION-BODY-CANARY"
	collaborationForcedSHA  = "cccccccccccccccccccccccccccccccccccccccc"
)

var collaborationOperations = []codehostbroker.Operation{
	codehostbroker.OperationComment,
	codehostbroker.OperationSubmitReview,
	codehostbroker.OperationApprove,
	codehostbroker.OperationRequestChanges,
}

func TestCollaborationOperationsAdvertisedAndApplied(t *testing.T) {
	for _, operation := range collaborationOperations {
		t.Run(string(operation), func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
			request := preparedCollaborationRequest(t, fixture, broker, operation, "success-"+string(operation), collaborationBodyCanary)
			response := broker.Execute(context.Background(), request)
			requireValidResponse(t, response)
			if response.Error != nil || response.Receipt == nil || response.Reconciliation == nil ||
				response.Reconciliation.Status != codehostbroker.ReconciliationApplied ||
				response.Policy.Effect != codehostbroker.EffectExternalWrite ||
				response.Policy.Consent != codehostbroker.ConsentExplicitUser ||
				fake.CollaborationAttempts() != 1 {
				t.Fatalf("response=%+v attempts=%d", response, fake.CollaborationAttempts())
			}
			var result codehostbroker.MutationResult
			decodeResult(t, response, &result)
			if result.Outcome != "applied" || result.PullRequest.Identity.Number != 42 ||
				result.PullRequest.Head.SHA != headSHAForTest ||
				result.Actor == nil || result.Actor.Login != "hero-user" || result.Actor.ProviderID != "U_99" ||
				response.Receipt.ProviderReceiptID == "" ||
				!strings.HasPrefix(response.Receipt.TargetRevision, "collaboration-target:") {
				t.Fatalf("result=%+v receipt=%+v", result, response.Receipt)
			}
			bodies := fake.CollaborationBodies()
			if len(bodies) != 1 || !strings.Contains(bodies[0], collaborationBodyCanary) ||
				strings.Count(bodies[0], heroMarkerPrefix) != 1 ||
				!containsExactHeroMarker(bodies[0], collaborationMarker(request)) {
				t.Fatalf("provider bodies=%q marker=%q", bodies, collaborationMarker(request))
			}
			replayed := broker.Execute(context.Background(), request)
			requireValidResponse(t, replayed)
			if replayed.Error != nil || replayed.Reconciliation == nil ||
				replayed.Reconciliation.Status != codehostbroker.ReconciliationReplayed ||
				replayed.Receipt.ProviderReceiptID != response.Receipt.ProviderReceiptID ||
				fake.CollaborationAttempts() != 1 {
				t.Fatalf("replay=%+v attempts=%d", replayed, fake.CollaborationAttempts())
			}
			assertCollaborationJournal(t, broker.projectRoot, request)
		})
	}

	fixture, _, broker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	capabilities := broker.Execute(context.Background(), fixture.request(codehostbroker.OperationCapabilities))
	requireValidResponse(t, capabilities)
	var result codehostbroker.CapabilitiesResult
	decodeResult(t, capabilities, &result)
	available := map[codehostbroker.Operation]codehostbroker.Capability{}
	for _, capability := range result.Capabilities {
		available[capability.Policy.Operation] = capability
	}
	for _, operation := range collaborationOperations {
		capability, ok := available[operation]
		if !ok || !capability.Available ||
			capability.Policy.Effect != codehostbroker.EffectExternalWrite ||
			capability.Policy.Consent != codehostbroker.ConsentExplicitUser {
			t.Fatalf("capability %s=%+v", operation, capability)
		}
	}
}

func TestCollaborationDuplicateConcurrentAndConflictSemantics(t *testing.T) {
	fixture, fake, broker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	request := preparedCollaborationRequest(t, fixture, broker, codehostbroker.OperationComment, "duplicate", collaborationBodyCanary)
	first := broker.Execute(context.Background(), request)
	second := broker.Execute(context.Background(), request)
	requireValidResponse(t, first)
	requireValidResponse(t, second)
	if first.Error != nil || second.Error != nil || second.Reconciliation == nil ||
		second.Reconciliation.Status != codehostbroker.ReconciliationReplayed ||
		first.Receipt.ProviderReceiptID != second.Receipt.ProviderReceiptID ||
		fake.CollaborationAttempts() != 1 {
		t.Fatalf("first=%+v second=%+v attempts=%d", first, second, fake.CollaborationAttempts())
	}

	for name, mutate := range map[string]func(*codehostbroker.Request){
		"operation": func(value *codehostbroker.Request) {
			value.Operation = codehostbroker.OperationSubmitReview
			value.Payload = reviewPayload(t, headSHAForTest, collaborationBodyCanary)
		},
		"target": func(value *codehostbroker.Request) {
			identity := *value.PullRequest
			identity.ProviderID = "PR_43"
			identity.Number = 43
			value.PullRequest = &identity
		},
		"head": func(value *codehostbroker.Request) {
			value.Payload = commentPayload(t, collaborationForcedSHA, collaborationBodyCanary)
		},
		"body": func(value *codehostbroker.Request) {
			value.Payload = commentPayload(t, headSHAForTest, "different body")
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := request
			mutate(&candidate)
			response := broker.Execute(context.Background(), candidate)
			requireValidResponse(t, response)
			if response.Error == nil || response.Error.Code != codehostbroker.ErrorIdempotencyConflict ||
				fake.CollaborationAttempts() != 1 {
				t.Fatalf("conflict=%+v attempts=%d", response.Error, fake.CollaborationAttempts())
			}
		})
	}

	for _, operation := range collaborationOperations {
		t.Run("concurrent_"+string(operation), func(t *testing.T) {
			concurrentFixture, concurrentFake, concurrentBroker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
			concurrent := preparedCollaborationRequest(t, concurrentFixture, concurrentBroker, operation, "concurrent-"+string(operation), collaborationBodyCanary)
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
					t.Fatalf("concurrent response=%+v", response)
				}
			}
			if concurrentFake.CollaborationAttempts() != 1 ||
				responses[0].Receipt.ProviderReceiptID != responses[1].Receipt.ProviderReceiptID {
				t.Fatalf("attempts=%d responses=%+v", concurrentFake.CollaborationAttempts(), responses)
			}
		})
	}

	deniedFixture, deniedFake, deniedBroker := createTestBroker(t, mockcodehost.CollaborationWriteDeniedScenario(), "acme/widgets")
	denied := preparedCollaborationRequest(t, deniedFixture, deniedBroker, codehostbroker.OperationComment, "write-denied", collaborationBodyCanary)
	deniedFirst := deniedBroker.Execute(context.Background(), denied)
	deniedSecond := deniedBroker.Execute(context.Background(), denied)
	requireValidResponse(t, deniedFirst)
	requireValidResponse(t, deniedSecond)
	if deniedFirst.Error == nil || deniedFirst.Error.Code != codehostbroker.ErrorForbidden ||
		deniedSecond.Error == nil || deniedSecond.Error.Code != codehostbroker.ErrorForbidden ||
		deniedFake.CollaborationAttempts() != 1 {
		t.Fatalf("denied=%+v/%+v attempts=%d", deniedFirst.Error, deniedSecond.Error, deniedFake.CollaborationAttempts())
	}
}

func TestCollaborationLostDelayedAmbiguousAndCancelledResponses(t *testing.T) {
	for _, operation := range collaborationOperations {
		t.Run("lost_"+string(operation), func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, mockcodehost.CollaborationLostResponseScenario(), "acme/widgets")
			request := preparedCollaborationRequest(t, fixture, broker, operation, "lost-"+string(operation), collaborationBodyCanary)
			response := broker.Execute(context.Background(), request)
			requireValidResponse(t, response)
			if response.Error != nil || response.Reconciliation == nil ||
				response.Reconciliation.Status != codehostbroker.ReconciliationReconciledApplied ||
				fake.CollaborationAttempts() != 1 {
				t.Fatalf("response=%+v attempts=%d", response, fake.CollaborationAttempts())
			}
		})
	}

	delayedFixture, delayedFake, delayedBroker := createTestBroker(t, mockcodehost.CollaborationDelayedVisibilityScenario(1), "acme/widgets")
	delayed := preparedCollaborationRequest(t, delayedFixture, delayedBroker, codehostbroker.OperationSubmitReview, "delayed", collaborationBodyCanary)
	delayedFirst := delayedBroker.Execute(context.Background(), delayed)
	delayedSecond := delayedBroker.Execute(context.Background(), delayed)
	requireValidResponse(t, delayedFirst)
	requireValidResponse(t, delayedSecond)
	if delayedFirst.Error == nil || delayedFirst.Error.Code != codehostbroker.ErrorAmbiguousResult ||
		delayedSecond.Error != nil || delayedSecond.Reconciliation.Status != codehostbroker.ReconciliationReconciledApplied ||
		delayedFake.CollaborationAttempts() != 1 {
		t.Fatalf("delayed=%+v/%+v attempts=%d", delayedFirst, delayedSecond, delayedFake.CollaborationAttempts())
	}

	for _, operation := range collaborationOperations {
		t.Run("ambiguous_"+string(operation), func(t *testing.T) {
			ambiguousFixture, ambiguousFake, ambiguousBroker := createTestBroker(t, mockcodehost.CollaborationAmbiguousScenario(), "acme/widgets")
			ambiguous := preparedCollaborationRequest(t, ambiguousFixture, ambiguousBroker, operation, "ambiguous-"+string(operation), collaborationBodyCanary)
			ambiguousFirst := ambiguousBroker.Execute(context.Background(), ambiguous)
			ambiguousSecond := ambiguousBroker.Execute(context.Background(), ambiguous)
			requireValidResponse(t, ambiguousFirst)
			requireValidResponse(t, ambiguousSecond)
			for _, response := range []codehostbroker.Response{ambiguousFirst, ambiguousSecond} {
				if response.Error == nil || response.Error.Code != codehostbroker.ErrorAmbiguousResult ||
					response.Reconciliation == nil || response.Reconciliation.Status != codehostbroker.ReconciliationAmbiguous {
					t.Fatalf("ambiguous response=%+v", response)
				}
				assertCollaborationErrorRedacted(t, response)
			}
			if ambiguousFake.CollaborationAttempts() != 1 {
				t.Fatalf("ambiguous attempts=%d", ambiguousFake.CollaborationAttempts())
			}
		})
	}

	mismatchedFixture, mismatchedFake, mismatchedBroker := createTestBroker(t, mockcodehost.CollaborationMismatchedReviewStateScenario(), "acme/widgets")
	mismatched := preparedCollaborationRequest(t, mismatchedFixture, mismatchedBroker, codehostbroker.OperationApprove, "mismatched-state", collaborationBodyCanary)
	mismatchedResponse := mismatchedBroker.Execute(context.Background(), mismatched)
	requireValidResponse(t, mismatchedResponse)
	if mismatchedResponse.Error == nil || mismatchedResponse.Error.Code != codehostbroker.ErrorAmbiguousResult ||
		mismatchedResponse.Reconciliation == nil || mismatchedResponse.Reconciliation.Status != codehostbroker.ReconciliationAmbiguous ||
		mismatchedFake.CollaborationAttempts() != 1 {
		t.Fatalf("mismatched review state=%+v attempts=%d", mismatchedResponse, mismatchedFake.CollaborationAttempts())
	}

	cancelFixture, cancelFake, cancelBroker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	cancelRequest := preparedCollaborationRequest(t, cancelFixture, cancelBroker, codehostbroker.OperationComment, "cancel-before", collaborationBodyCanary)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled := cancelBroker.Execute(ctx, cancelRequest)
	requireValidResponse(t, cancelled)
	if cancelled.Error == nil || cancelled.Error.Code != codehostbroker.ErrorCancelled ||
		cancelled.Reconciliation == nil || cancelled.Reconciliation.Status != codehostbroker.ReconciliationNotApplied ||
		cancelFake.CollaborationAttempts() != 0 {
		t.Fatalf("cancelled=%+v attempts=%d", cancelled, cancelFake.CollaborationAttempts())
	}

	for _, operation := range collaborationOperations {
		t.Run("cancel_after_"+string(operation), func(t *testing.T) {
			afterFixture, afterFake, afterBroker := createTestBroker(t, mockcodehost.CollaborationCancelledAfterApplyScenario(500*time.Millisecond), "acme/widgets")
			after := preparedCollaborationRequest(t, afterFixture, afterBroker, operation, "cancel-after-"+string(operation), collaborationBodyCanary)
			afterCtx, afterCancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer afterCancel()
			afterResponse := afterBroker.Execute(afterCtx, after)
			requireValidResponse(t, afterResponse)
			if afterResponse.Error != nil || afterResponse.Reconciliation == nil ||
				afterResponse.Reconciliation.Status != codehostbroker.ReconciliationReconciledApplied ||
				afterFake.CollaborationAttempts() != 1 {
				t.Fatalf("after response=%+v attempts=%d", afterResponse, afterFake.CollaborationAttempts())
			}
		})
	}
}

func TestCollaborationPreflightPermissionsStateAndFreshness(t *testing.T) {
	requiredFixture, requiredFake, requiredBroker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	required := preparedCollaborationRequest(t, requiredFixture, requiredBroker, codehostbroker.OperationComment, "required-fields", collaborationBodyCanary)
	for name, mutate := range map[string]func(*codehostbroker.Request){
		"pull_request":         func(value *codehostbroker.Request) { value.PullRequest = nil },
		"intent_source":        func(value *codehostbroker.Request) { value.IntentSource = "" },
		"idempotency_key":      func(value *codehostbroker.Request) { value.IdempotencyKey = "" },
		"capability_revision":  func(value *codehostbroker.Request) { value.CapabilityRevision = "" },
		"observation_revision": func(value *codehostbroker.Request) { value.ObservationRevision = "" },
		"reconciliation_key":   func(value *codehostbroker.Request) { value.ReconciliationKey = "" },
	} {
		t.Run("required_"+name, func(t *testing.T) {
			candidate := required
			mutate(&candidate)
			response := requiredBroker.Execute(context.Background(), candidate)
			requireValidResponse(t, response)
			if response.Error == nil || response.Error.Code != codehostbroker.ErrorInvalidInput ||
				requiredFake.CollaborationAttempts() != 0 {
				t.Fatalf("%s response=%+v attempts=%d", name, response, requiredFake.CollaborationAttempts())
			}
		})
	}

	unsupported := requiredFixture.request(codehostbroker.OperationMarkReady)
	unsupported.IntentSource = "user"
	unsupported.Consent = codehostbroker.ConsentExplicitUser
	unsupported.IdempotencyKey = "unsupported"
	unsupported.CapabilityRevision = "capability"
	unsupported.ObservationRevision = "observation"
	unsupported.ReconciliationKey = "reconcile:unsupported"
	unsupported.Payload, _ = json.Marshal(codehostbroker.LifecyclePayload{ExpectedHeadSHA: headSHAForTest})
	unsupportedResponse := requiredBroker.Execute(context.Background(), unsupported)
	requireValidResponse(t, unsupportedResponse)
	if unsupportedResponse.Error == nil || unsupportedResponse.Error.Code != codehostbroker.ErrorUnsupportedOperation ||
		requiredFake.CollaborationAttempts() != 0 {
		t.Fatalf("unsupported response=%+v attempts=%d", unsupportedResponse, requiredFake.CollaborationAttempts())
	}

	emptyPayload, _ := json.Marshal(codehostbroker.CommentPayload{ExpectedHeadSHA: headSHAForTest})
	oversizedBody := strings.Repeat("x", codehostbroker.MaxBodyBytes-len(emptyPayload))
	boundedButNoMarkerRoom := required
	boundedButNoMarkerRoom.IdempotencyKey = "no-marker-room"
	boundedButNoMarkerRoom.ReconciliationKey = "reconcile:no-marker-room"
	boundedButNoMarkerRoom.Payload = commentPayload(t, headSHAForTest, oversizedBody)
	if contractErr := codehostbroker.ValidateRequest(boundedButNoMarkerRoom); contractErr != nil {
		t.Fatalf("test payload must pass the public contract bound: %+v", contractErr)
	}
	noMarkerRoomResponse := requiredBroker.Execute(context.Background(), boundedButNoMarkerRoom)
	requireValidResponse(t, noMarkerRoomResponse)
	if noMarkerRoomResponse.Error == nil || noMarkerRoomResponse.Error.Code != codehostbroker.ErrorInputTooLarge ||
		requiredFake.CollaborationAttempts() != 0 {
		t.Fatalf("no marker room=%+v attempts=%d", noMarkerRoomResponse, requiredFake.CollaborationAttempts())
	}

	for _, operation := range collaborationOperations {
		t.Run("permission_"+string(operation), func(t *testing.T) {
			permissionFixture, permissionFake, permissionBroker := createTestBroker(t, mockcodehost.CollaborationPermissionDeniedScenario(), "acme/widgets")
			permission := collaborationRequest(permissionFixture, operation, "permission-"+string(operation), collaborationBodyCanary)
			if _, err := permissionBroker.PrepareCollaboration(context.Background(), permission); err == nil || err.Code != codehostbroker.ErrorForbidden {
				t.Fatalf("permission error=%+v", err)
			}
			if permissionFake.CollaborationAttempts() != 0 {
				t.Fatal("permission failure dispatched")
			}
		})
	}

	changeFixture, changeFake, changeBroker := createTestBroker(t, mockcodehost.CollaborationPermissionChangeScenario(), "acme/widgets")
	change := preparedCollaborationRequest(t, changeFixture, changeBroker, codehostbroker.OperationApprove, "permission-change", "")
	changeResponse := changeBroker.Execute(context.Background(), change)
	requireValidResponse(t, changeResponse)
	if changeResponse.Error == nil || changeResponse.Error.Code != codehostbroker.ErrorForbidden ||
		changeResponse.Reconciliation == nil || changeResponse.Reconciliation.Status != codehostbroker.ReconciliationNotApplied ||
		changeFake.CollaborationAttempts() != 0 {
		t.Fatalf("permission change=%+v attempts=%d", changeResponse, changeFake.CollaborationAttempts())
	}

	for _, operation := range collaborationOperations {
		t.Run("stale_"+string(operation), func(t *testing.T) {
			staleFixture, staleFake, staleBroker := createTestBroker(t, mockcodehost.ForcePushScenario(), "acme/widgets")
			stale := preparedCollaborationRequest(t, staleFixture, staleBroker, operation, "stale-"+string(operation), collaborationBodyCanary)
			staleResponse := staleBroker.Execute(context.Background(), stale)
			requireValidResponse(t, staleResponse)
			if staleResponse.Error == nil || staleResponse.Error.Code != codehostbroker.ErrorStaleObservation ||
				staleFake.CollaborationAttempts() != 0 {
				t.Fatalf("stale=%+v attempts=%d", staleResponse.Error, staleFake.CollaborationAttempts())
			}
		})
	}

	observationFixture, observationFake, observationBroker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	observation := preparedCollaborationRequest(t, observationFixture, observationBroker, codehostbroker.OperationComment, "observation", collaborationBodyCanary)
	observation.ObservationRevision = "observation:stale"
	observationResponse := observationBroker.Execute(context.Background(), observation)
	requireValidResponse(t, observationResponse)
	if observationResponse.Error == nil || observationResponse.Error.Code != codehostbroker.ErrorStaleObservation ||
		observationFake.CollaborationAttempts() != 0 {
		t.Fatalf("observation=%+v attempts=%d", observationResponse.Error, observationFake.CollaborationAttempts())
	}

	capability := observation
	capability.IdempotencyKey = "capability"
	capability.ReconciliationKey = "reconcile:capability"
	capability.CapabilityRevision = "capability:stale"
	capabilityResponse := observationBroker.Execute(context.Background(), capability)
	requireValidResponse(t, capabilityResponse)
	if capabilityResponse.Error == nil || capabilityResponse.Error.Code != codehostbroker.ErrorCapabilityChanged ||
		observationFake.CollaborationAttempts() != 0 {
		t.Fatalf("capability=%+v attempts=%d", capabilityResponse.Error, observationFake.CollaborationAttempts())
	}

	closedFixture, closedFake, closedBroker := createTestBroker(t, mockcodehost.CollaborationClosedScenario(), "acme/widgets")
	closed := collaborationRequest(closedFixture, codehostbroker.OperationComment, "closed", collaborationBodyCanary)
	if _, err := closedBroker.PrepareCollaboration(context.Background(), closed); err == nil || err.Code != codehostbroker.ErrorConflict {
		t.Fatalf("closed error=%+v", err)
	}
	if closedFake.CollaborationAttempts() != 0 {
		t.Fatal("closed PR dispatched")
	}
}

func TestCollaborationExternallyCompletedReviewRules(t *testing.T) {
	for _, test := range []struct {
		name      string
		operation codehostbroker.Operation
		scenario  mockcodehost.Scenario
		external  bool
	}{
		{"approved", codehostbroker.OperationApprove, mockcodehost.CollaborationExternallyCompletedScenario("APPROVED"), true},
		{"changes", codehostbroker.OperationRequestChanges, mockcodehost.CollaborationExternallyCompletedScenario("CHANGES_REQUESTED"), true},
		{"old_head", codehostbroker.OperationApprove, mockcodehost.CollaborationOldHeadReviewScenario("APPROVED"), false},
		{"dismissed", codehostbroker.OperationApprove, mockcodehost.CollaborationDismissedReviewScenario("APPROVED"), false},
		{"other_actor", codehostbroker.OperationApprove, mockcodehost.CollaborationOtherActorReviewScenario("APPROVED"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, fake, broker := createTestBroker(t, test.scenario, "acme/widgets")
			request := preparedCollaborationRequest(t, fixture, broker, test.operation, "external-"+test.name, "")
			response := broker.Execute(context.Background(), request)
			requireValidResponse(t, response)
			if response.Error != nil {
				t.Fatal(response.Error)
			}
			if test.external {
				if response.Reconciliation.Status != codehostbroker.ReconciliationExternallyCompleted ||
					fake.CollaborationAttempts() != 0 {
					t.Fatalf("external response=%+v attempts=%d", response, fake.CollaborationAttempts())
				}
				replayed := broker.Execute(context.Background(), request)
				requireValidResponse(t, replayed)
				if replayed.Error != nil || replayed.Reconciliation == nil ||
					replayed.Reconciliation.Status != codehostbroker.ReconciliationReplayed ||
					replayed.Receipt.ProviderReceiptID != response.Receipt.ProviderReceiptID ||
					fake.CollaborationAttempts() != 0 {
					t.Fatalf("external replay=%+v attempts=%d", replayed, fake.CollaborationAttempts())
				}
			} else if response.Reconciliation.Status != codehostbroker.ReconciliationApplied ||
				fake.CollaborationAttempts() != 1 {
				t.Fatalf("non-external response=%+v attempts=%d", response, fake.CollaborationAttempts())
			}
		})
	}
}

func TestCollaborationMarkersNormalizationCollisionAndJournalRedaction(t *testing.T) {
	fixture, fake, broker := createTestBroker(t, mockcodehost.DefaultScenario(), "acme/widgets")
	comment := preparedCollaborationRequest(t, fixture, broker, codehostbroker.OperationComment, "normalize-comment", "comment "+collaborationBodyCanary)
	review := preparedCollaborationRequest(t, fixture, broker, codehostbroker.OperationSubmitReview, "normalize-review", "review "+collaborationBodyCanary)
	commentResponse := broker.Execute(context.Background(), comment)
	reviewResponse := broker.Execute(context.Background(), review)
	requireValidResponse(t, commentResponse)
	requireValidResponse(t, reviewResponse)

	commentsRead := broker.Execute(context.Background(), fixture.request(codehostbroker.OperationGetComments))
	reviewsRead := broker.Execute(context.Background(), fixture.request(codehostbroker.OperationGetReviews))
	requireValidResponse(t, commentsRead)
	requireValidResponse(t, reviewsRead)
	var comments codehostbroker.CommentsResult
	var reviews codehostbroker.ReviewsResult
	decodeResult(t, commentsRead, &comments)
	decodeResult(t, reviewsRead, &reviews)
	foundComment, foundReview := false, false
	for _, value := range comments.Comments {
		if value.Author.Login == "hero-user" {
			foundComment = value.Body == "comment "+collaborationBodyCanary && !strings.Contains(value.Body, heroMarkerPrefix)
		}
	}
	for _, value := range reviews.Reviews {
		if value.Author.Login == "hero-user" {
			foundReview = value.Body == "review "+collaborationBodyCanary && !strings.Contains(value.Body, heroMarkerPrefix)
		}
	}
	if !foundComment || !foundReview {
		t.Fatalf("normalized comments=%+v reviews=%+v bodies=%q", comments, reviews, fake.CollaborationBodies())
	}

	valid := heroMarkerPrefix + strings.Repeat("a", 64) + heroMarkerSuffix
	malformed := []string{
		"<!-- hero-code-host-op:" + strings.Repeat("a", 63) + " -->",
		"<!-- hero-code-host-op:" + strings.Repeat("G", 64) + " -->",
		"<!--hero-code-host-op:" + strings.Repeat("a", 64) + "-->",
	}
	if got := stripHeroMarkers("before" + valid + "after"); got != "beforeafter" {
		t.Fatalf("valid marker strip=%q", got)
	}
	for _, value := range malformed {
		if got := stripHeroMarkers("before" + value + "after"); got != "before"+value+"after" {
			t.Fatalf("malformed marker changed: %q => %q", value, got)
		}
	}

	markerRequest := collaborationRequest(fixture, codehostbroker.OperationComment, "marker-input", valid)
	markerRequest.CapabilityRevision = currentCapabilityRevision(t, fixture, broker)
	markerResponse := broker.Execute(context.Background(), markerRequest)
	requireValidResponse(t, markerResponse)
	if markerResponse.Error == nil || markerResponse.Error.Code != codehostbroker.ErrorInvalidInput {
		t.Fatalf("marker input response=%+v", markerResponse.Error)
	}

	collisionFixture, collisionFake, collisionBroker := createTestBroker(t, mockcodehost.CollaborationMarkerCollisionScenario(), "acme/widgets")
	collision := preparedCollaborationRequest(t, collisionFixture, collisionBroker, codehostbroker.OperationComment, "collision", collaborationBodyCanary)
	collisionResponse := collisionBroker.Execute(context.Background(), collision)
	requireValidResponse(t, collisionResponse)
	if collisionResponse.Error == nil || collisionResponse.Error.Code != codehostbroker.ErrorAmbiguousResult ||
		collisionFake.CollaborationAttempts() != 1 {
		t.Fatalf("collision=%+v attempts=%d", collisionResponse, collisionFake.CollaborationAttempts())
	}

	unmarkedFixture, unmarkedFake, unmarkedBroker := createTestBroker(t, mockcodehost.CollaborationUnmarkedCommentScenario(collaborationBodyCanary), "acme/widgets")
	unmarked := preparedCollaborationRequest(t, unmarkedFixture, unmarkedBroker, codehostbroker.OperationComment, "unmarked-identical", collaborationBodyCanary)
	unmarkedResponse := unmarkedBroker.Execute(context.Background(), unmarked)
	requireValidResponse(t, unmarkedResponse)
	if unmarkedResponse.Error != nil ||
		unmarkedResponse.Reconciliation.Status != codehostbroker.ReconciliationApplied ||
		unmarkedFake.CollaborationAttempts() != 1 {
		t.Fatalf("unmarked identical=%+v attempts=%d", unmarkedResponse, unmarkedFake.CollaborationAttempts())
	}

	assertCollaborationJournal(t, broker.projectRoot, comment)
	assertCollaborationJournal(t, broker.projectRoot, review)
}

func TestCollaborationReadbackPartialFailureRemainsAmbiguous(t *testing.T) {
	scenario := mockcodehost.CollaborationLostResponseScenario()
	scenario.DeniedSections = map[string]int{"reviews": 403}
	fixture, fake, broker := createTestBroker(t, scenario, "acme/widgets")
	request := preparedCollaborationRequest(t, fixture, broker, codehostbroker.OperationSubmitReview, "partial", collaborationBodyCanary)
	response := broker.Execute(context.Background(), request)
	requireValidResponse(t, response)
	if response.Error == nil || response.Error.Code != codehostbroker.ErrorAmbiguousResult ||
		response.Reconciliation == nil || response.Reconciliation.Status != codehostbroker.ReconciliationAmbiguous ||
		fake.CollaborationAttempts() != 1 {
		t.Fatalf("partial response=%+v attempts=%d", response, fake.CollaborationAttempts())
	}
}

func FuzzHeroMarkerStripping(f *testing.F) {
	valid := heroMarkerPrefix + strings.Repeat("a", 64) + heroMarkerSuffix
	for _, seed := range []string{"plain", valid, "before" + valid + "after", "<!-- hero-code-host-op:not-hex -->", collaborationBodyCanary} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		stripped := stripHeroMarkers(value)
		if heroMarkerPattern.MatchString(stripped) {
			t.Fatalf("valid marker survived stripping: %q", stripped)
		}
		if !heroMarkerPattern.MatchString(value) && stripped != value {
			t.Fatalf("marker-free input changed: %q => %q", value, stripped)
		}
		if len(stripped) > len(value) {
			t.Fatalf("stripping grew input: %d => %d", len(value), len(stripped))
		}
	})
}

func collaborationRequest(fixture brokerFixture, operation codehostbroker.Operation, key, body string) codehostbroker.Request {
	request := fixture.request(operation)
	request.IntentSource = "user"
	request.Consent = codehostbroker.ConsentExplicitUser
	request.IdempotencyKey = key
	request.ReconciliationKey = "reconcile:" + key
	request.CapabilityRevision = "prepare"
	request.ObservationRevision = "prepare"
	if operation == codehostbroker.OperationComment {
		request.Payload, _ = json.Marshal(codehostbroker.CommentPayload{ExpectedHeadSHA: headSHAForTest, Body: body})
	} else {
		request.Payload, _ = json.Marshal(codehostbroker.ReviewPayload{ExpectedHeadSHA: headSHAForTest, Body: body})
	}
	return request
}

func preparedCollaborationRequest(t *testing.T, fixture brokerFixture, broker *Broker, operation codehostbroker.Operation, key, body string) codehostbroker.Request {
	t.Helper()
	request, err := broker.PrepareCollaboration(context.Background(), collaborationRequest(fixture, operation, key, body))
	if err != nil {
		t.Fatalf("prepare %s: %+v", operation, err)
	}
	return request
}

func commentPayload(t *testing.T, head, body string) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(codehostbroker.CommentPayload{ExpectedHeadSHA: head, Body: body})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func reviewPayload(t *testing.T, head, body string) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(codehostbroker.ReviewPayload{ExpectedHeadSHA: head, Body: body})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertCollaborationJournal(t *testing.T, projectRoot string, request codehostbroker.Request) {
	t.Helper()
	journal := newMutationJournal(projectRoot, time.Now)
	document, err := journal.load()
	if err != nil {
		t.Fatal(err)
	}
	entry := document.Entries[journalKeyDigest(request)]
	if entry == nil || entry.Receipt == nil || entry.Receipt.ProviderReceiptID == "" ||
		entry.Receipt.Actor == nil || entry.Receipt.Actor.Login != "hero-user" ||
		entry.Receipt.HeadSHA != headSHAForTest || entry.Target.Marker != collaborationMarker(request) ||
		entry.Target.PullRequest == nil || entry.Target.PullRequest.Number != 42 {
		t.Fatalf("journal entry=%+v", entry)
	}
	data, err := os.ReadFile(journal.path)
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(data)
	for _, forbidden := range []string{credentialCanary, collaborationBodyCanary, "Authorization", "Bearer"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("journal leaked %q: %s", forbidden, encoded)
		}
	}
}

func assertCollaborationErrorRedacted(t *testing.T, response codehostbroker.Response) {
	t.Helper()
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(data)
	for _, forbidden := range []string{credentialCanary, collaborationBodyCanary, "Authorization", "Bearer"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("error leaked %q: %s", forbidden, encoded)
		}
	}
}
