package codehost

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/hero-engine/hero/contracts/codehostbroker"
)

type createPayload struct {
	codehostbroker.CreatePullRequestPayload
}

type canonicalRepositoryTarget struct {
	Host     string
	FullName string
}

type canonicalRefTarget struct {
	Repository canonicalRepositoryTarget
	Name       string
	SHA        string
}

func decodeCreatePayload(raw json.RawMessage) (createPayload, *codehostbroker.ContractError) {
	var fields map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return createPayload{}, contractError(codehostbroker.ErrorInvalidInput, "creation payload is invalid", "payload")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return createPayload{}, contractError(codehostbroker.ErrorInvalidInput, "creation payload contains trailing data", "payload")
	}
	draft, present := fields["draft"]
	if !present || (string(draft) != "true" && string(draft) != "false") {
		return createPayload{}, contractError(codehostbroker.ErrorInvalidInput, "creation draft state must be explicitly true or false", "payload.draft")
	}
	var payload codehostbroker.CreatePullRequestPayload
	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return createPayload{}, contractError(codehostbroker.ErrorInvalidInput, "creation payload does not match the v1 schema", "payload")
	}
	return createPayload{CreatePullRequestPayload: payload}, nil
}

func canonicalCreateDigest(request codehostbroker.Request, payload createPayload) string {
	material := struct {
		Version           string
		Operation         codehostbroker.Operation
		Provider          string
		ConnectionID      string
		Repository        canonicalRepositoryTarget
		Base              canonicalRefTarget
		Head              canonicalRefTarget
		Title             string
		Body              string
		Draft             bool
		IntentSource      string
		Consent           codehostbroker.Consent
		ReconciliationKey string
	}{
		Version:           request.Version,
		Operation:         request.Operation,
		Provider:          strings.ToLower(request.Provider),
		ConnectionID:      request.ConnectionID,
		Repository:        canonicalRepository(request.Repository),
		Base:              canonicalRef(payload.Base),
		Head:              canonicalRef(payload.Head),
		Title:             payload.Title,
		Body:              payload.Body,
		Draft:             payload.Draft,
		IntentSource:      request.IntentSource,
		Consent:           request.Consent,
		ReconciliationKey: request.ReconciliationKey,
	}
	return fingerprint("payload", material)
}

func journalKeyDigest(request codehostbroker.Request) string {
	return fingerprint("idempotency", request.Version, strings.ToLower(request.Provider), request.ConnectionID, request.IdempotencyKey)
}

func operationID(request codehostbroker.Request) string {
	return fingerprint("operation", journalKeyDigest(request), request.Operation)
}

func canonicalRepository(repository codehostbroker.RepositoryIdentity) canonicalRepositoryTarget {
	return canonicalRepositoryTarget{
		Host:     strings.ToLower(repository.Host),
		FullName: strings.ToLower(repository.FullName),
	}
}

func canonicalRef(ref codehostbroker.RefIdentity) canonicalRefTarget {
	return canonicalRefTarget{
		Repository: canonicalRepository(ref.Repository),
		Name:       ref.Name,
		SHA:        strings.ToLower(ref.SHA),
	}
}

func createMutationResult(pullRequest codehostbroker.PullRequest, outcome string) codehostbroker.MutationResult {
	return codehostbroker.MutationResult{
		PullRequest: pullRequest,
		Outcome:     boundedText(outcome, 128),
	}
}

func createReceipt(entry *journalEntry, pullRequest *codehostbroker.PullRequest) *codehostbroker.Receipt {
	receipt := &codehostbroker.Receipt{OperationID: entry.OperationID}
	if entry.Receipt != nil && entry.Receipt.ProviderReceiptID != "" {
		receipt.ProviderReceiptID = entry.Receipt.ProviderReceiptID
		receipt.TargetRevision = fingerprint("collaboration-target", entry.Receipt.Actor, entry.Receipt.HeadSHA, entry.Receipt.Identity)
	} else if pullRequest != nil {
		receipt.ProviderReceiptID = pullRequest.Identity.ProviderID
		receipt.TargetRevision = fingerprint("target", pullRequest.Identity, pullRequest.Base, pullRequest.Head, pullRequest.State)
	} else if entry.Receipt != nil {
		receipt.ProviderReceiptID = entry.Receipt.Identity.ProviderID
		receipt.TargetRevision = fingerprint("target", entry.Receipt.Identity, entry.Receipt.Base, entry.Receipt.Head)
	}
	return receipt
}

func safeJournalReceipt(pullRequest codehostbroker.PullRequest) *journalReceipt {
	return &journalReceipt{
		Identity: pullRequest.Identity,
		Base:     pullRequest.Base,
		Head:     pullRequest.Head,
	}
}

func sameCreateTarget(pullRequest codehostbroker.PullRequest, payload createPayload) bool {
	return equalRepository(pullRequest.Base.Repository, payload.Base.Repository) &&
		equalRepository(pullRequest.Head.Repository, payload.Head.Repository) &&
		pullRequest.Base.Name == payload.Base.Name &&
		pullRequest.Head.Name == payload.Head.Name
}

func satisfiesCreate(pullRequest codehostbroker.PullRequest, payload createPayload) bool {
	return sameCreateTarget(pullRequest, payload) &&
		pullRequest.Base.SHA == payload.Base.SHA &&
		pullRequest.Head.SHA == payload.Head.SHA &&
		pullRequest.Title == payload.Title &&
		pullRequest.Body == payload.Body &&
		pullRequest.Draft == payload.Draft &&
		pullRequest.State == "open"
}

func equalRepository(left, right codehostbroker.RepositoryIdentity) bool {
	return strings.EqualFold(left.Host, right.Host) &&
		strings.EqualFold(left.FullName, right.FullName)
}
