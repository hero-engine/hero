package codehost

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strings"

	"github.com/hero-engine/hero/contracts/codehostbroker"
)

const (
	heroMarkerPrefix = "<!-- hero-code-host-op:"
	heroMarkerSuffix = " -->"
)

var heroMarkerPattern = regexp.MustCompile(`<!-- hero-code-host-op:[0-9a-f]{64} -->`)

type collaborationPayload struct {
	expectedHeadSHA string
	body            string
}

func isCollaborationOperation(operation codehostbroker.Operation) bool {
	switch operation {
	case codehostbroker.OperationComment,
		codehostbroker.OperationSubmitReview,
		codehostbroker.OperationApprove,
		codehostbroker.OperationRequestChanges:
		return true
	default:
		return false
	}
}

func decodeCollaborationPayload(request codehostbroker.Request) (collaborationPayload, *codehostbroker.ContractError) {
	switch request.Operation {
	case codehostbroker.OperationComment:
		var payload codehostbroker.CommentPayload
		if err := decodeCollaborationJSON(request.Payload, &payload); err != nil {
			return collaborationPayload{}, err
		}
		return validateCollaborationPayload(payload.ExpectedHeadSHA, payload.Body)
	case codehostbroker.OperationSubmitReview,
		codehostbroker.OperationApprove,
		codehostbroker.OperationRequestChanges:
		var payload codehostbroker.ReviewPayload
		if err := decodeCollaborationJSON(request.Payload, &payload); err != nil {
			return collaborationPayload{}, err
		}
		return validateCollaborationPayload(payload.ExpectedHeadSHA, payload.Body)
	default:
		return collaborationPayload{}, contractError(codehostbroker.ErrorUnsupportedOperation, "operation is not a collaboration mutation", "operation")
	}
}

func decodeCollaborationJSON(raw json.RawMessage, target any) *codehostbroker.ContractError {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return contractError(codehostbroker.ErrorInvalidInput, "collaboration payload does not match the v1 schema", "payload")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return contractError(codehostbroker.ErrorInvalidInput, "collaboration payload contains trailing data", "payload")
	}
	return nil
}

func validateCollaborationPayload(expectedHeadSHA, body string) (collaborationPayload, *codehostbroker.ContractError) {
	if heroMarkerPattern.MatchString(body) {
		return collaborationPayload{}, contractError(codehostbroker.ErrorInvalidInput, "collaboration body must not contain a Hero reconciliation marker", "payload.body")
	}
	if len(body)+len(heroMarkerPrefix)+64+len(heroMarkerSuffix) > codehostbroker.MaxBodyBytes {
		return collaborationPayload{}, contractError(codehostbroker.ErrorInputTooLarge, "collaboration body leaves no room for bounded reconciliation material", "payload.body")
	}
	return collaborationPayload{expectedHeadSHA: expectedHeadSHA, body: body}, nil
}

func collaborationMarker(request codehostbroker.Request) string {
	value := fingerprint("marker", operationID(request))
	_, opaque, _ := strings.Cut(value, ":")
	return heroMarkerPrefix + opaque + heroMarkerSuffix
}

func injectHeroMarker(body, marker string) string {
	return body + marker
}

func stripHeroMarkers(body string) string {
	return heroMarkerPattern.ReplaceAllString(body, "")
}

func containsExactHeroMarker(body, marker string) bool {
	if !isValidHeroMarker(marker) {
		return false
	}
	for _, candidate := range heroMarkerPattern.FindAllString(body, -1) {
		if candidate == marker {
			return true
		}
	}
	return false
}

func isValidHeroMarker(marker string) bool {
	return len(marker) == len(heroMarkerPrefix)+64+len(heroMarkerSuffix) &&
		heroMarkerPattern.MatchString(marker) &&
		heroMarkerPattern.FindString(marker) == marker
}

func canonicalCollaborationDigest(request codehostbroker.Request, payload collaborationPayload) string {
	bodyDigest := fingerprint("body", payload.body)
	material := struct {
		Version           string
		Operation         codehostbroker.Operation
		Provider          string
		ConnectionID      string
		PullRequest       codehostbroker.PullRequestIdentity
		ExpectedHeadSHA   string
		BodyDigest        string
		IntentSource      string
		Consent           codehostbroker.Consent
		ReconciliationKey string
	}{
		Version:           request.Version,
		Operation:         request.Operation,
		Provider:          strings.ToLower(request.Provider),
		ConnectionID:      request.ConnectionID,
		PullRequest:       *request.PullRequest,
		ExpectedHeadSHA:   strings.ToLower(payload.expectedHeadSHA),
		BodyDigest:        bodyDigest,
		IntentSource:      request.IntentSource,
		Consent:           request.Consent,
		ReconciliationKey: request.ReconciliationKey,
	}
	return fingerprint("payload", material)
}
