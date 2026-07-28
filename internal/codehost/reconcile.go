package codehost

import (
	"context"
	"time"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/config"
)

type mutationEffect struct {
	pullRequest codehostbroker.PullRequest
	receipt     *journalReceipt
}

type mutationPreflight struct {
	observationRevision string
	external            *mutationEffect
}

// mutationPlan supplies operation-specific provider behavior to the one
// journal/retry/reconciliation state machine shared by all broker mutations.
type mutationPlan struct {
	request                codehostbroker.Request
	payloadDigest          string
	target                 mutationTarget
	connection             config.CodeHostConnection
	adapter                *githubAdapter
	capabilityRevision     string
	resolveCapability      func(context.Context) (string, error)
	preflight              func(context.Context) (mutationPreflight, error)
	dispatch               func(context.Context) (mutationEffect, error)
	reconcile              func(context.Context) (*mutationEffect, error)
	decisiveReconcileError func(error) bool
}

func (b *Broker) executeMutation(ctx context.Context, plan mutationPlan) (adapterResult, error) {
	journal := newMutationJournal(b.projectRoot, b.now)
	key := journalKeyDigest(plan.request)
	var out adapterResult
	var executionErr error
	err := journal.withLock(func(document *journalDocument) error {
		entry := document.Entries[key]
		if entry != nil && entry.PayloadDigest != plan.payloadDigest {
			out = mutationErrorResult(plan.request, entry, document, codehostbroker.ReconciliationNotApplied)
			executionErr = &providerError{code: codehostbroker.ErrorIdempotencyConflict, message: "idempotency key was already used for a different mutation payload"}
			return nil
		}
		if entry == nil {
			if len(document.Entries) >= journal.maxEntries {
				executionErr = &providerError{code: codehostbroker.ErrorProviderUnavailable, message: "mutation journal is full of unresolved safety records"}
				return nil
			}
			timestamp := journal.timestamp()
			entry = &journalEntry{
				KeyDigest:     key,
				PayloadDigest: plan.payloadDigest,
				OperationID:   operationID(plan.request),
				Target:        plan.target,
				State:         journalInProgress,
				CreatedAt:     timestamp,
				UpdatedAt:     timestamp,
			}
			document.Entries[key] = entry
			if saveErr := journal.save(document); saveErr != nil {
				return saveErr
			}
		} else {
			replayed, replayErr, decisive := b.reconcileExistingMutation(ctx, plan, entry, document)
			if decisive {
				out = replayed
				executionErr = replayErr
				return nil
			}
			if saveErr := journal.save(document); saveErr != nil {
				return saveErr
			}
		}

		out.receipt = createReceipt(entry, nil)
		out.reconciliation = &codehostbroker.Reconciliation{Status: codehostbroker.ReconciliationInProgress, Key: plan.request.ReconciliationKey}
		out.journalEntries = len(document.Entries)
		expectedCapability := capabilityRevision(plan.connection, nil)
		if plan.capabilityRevision != "" {
			expectedCapability = plan.capabilityRevision
		}
		if plan.resolveCapability != nil {
			var capabilityErr error
			expectedCapability, capabilityErr = plan.resolveCapability(ctx)
			out.capabilityRevision = expectedCapability
			if capabilityErr != nil {
				entry.State = journalNotApplied
				entry.UpdatedAt = journal.timestamp()
				out.reconciliation.Status = codehostbroker.ReconciliationNotApplied
				executionErr = capabilityErr
				return nil
			}
		}
		out.capabilityRevision = expectedCapability
		if plan.request.CapabilityRevision != expectedCapability {
			entry.State = journalNotApplied
			entry.UpdatedAt = journal.timestamp()
			out.reconciliation.Status = codehostbroker.ReconciliationNotApplied
			executionErr = &providerError{code: codehostbroker.ErrorCapabilityChanged, message: "code-host capability revision changed before mutation"}
			return nil
		}
		if err := ctx.Err(); err != nil {
			entry.State = journalNotApplied
			entry.UpdatedAt = journal.timestamp()
			out.reconciliation.Status = codehostbroker.ReconciliationNotApplied
			executionErr = err
			return nil
		}
		preflight, preflightErr := plan.preflight(ctx)
		out = withTransportMetadata(out, plan.adapter)
		if preflightErr != nil {
			entry.State = journalNotApplied
			entry.UpdatedAt = journal.timestamp()
			out.reconciliation.Status = codehostbroker.ReconciliationNotApplied
			executionErr = preflightErr
			return nil
		}
		out.observationRevision = preflight.observationRevision
		if plan.request.ObservationRevision != preflight.observationRevision {
			entry.State = journalNotApplied
			entry.UpdatedAt = journal.timestamp()
			out.reconciliation.Status = codehostbroker.ReconciliationNotApplied
			executionErr = &providerError{code: codehostbroker.ErrorStaleObservation, message: "mutation preflight observation is stale"}
			return nil
		}
		if preflight.external != nil {
			entry.State = journalExternal
			entry.Receipt = preflight.external.receipt
			entry.UpdatedAt = journal.timestamp()
			entry.ReconciledAt = entry.UpdatedAt
			out = successfulMutationResult(plan.request, entry, document, preflight.external.pullRequest, codehostbroker.ReconciliationExternallyCompleted, "externally_completed")
			out = withTransportMetadata(out, plan.adapter)
			return nil
		}

		entry.State = journalDispatched
		entry.ProviderAttempts++
		entry.UpdatedAt = journal.timestamp()
		if saveErr := journal.save(document); saveErr != nil {
			return saveErr
		}
		effect, dispatchErr := plan.dispatch(ctx)
		if dispatchErr == nil {
			entry.State = journalApplied
			entry.Receipt = effect.receipt
			entry.UpdatedAt = journal.timestamp()
			out = successfulMutationResult(plan.request, entry, document, effect.pullRequest, codehostbroker.ReconciliationApplied, "applied")
			out = withTransportMetadata(out, plan.adapter)
			return nil
		}
		if !ambiguousProviderFailure(dispatchErr) {
			entry.State = journalNotApplied
			entry.UpdatedAt = journal.timestamp()
			entry.FailureCode = normalizeProviderError(dispatchErr).Code
			out = mutationErrorResult(plan.request, entry, document, codehostbroker.ReconciliationNotApplied)
			out = withTransportMetadata(out, plan.adapter)
			executionErr = dispatchErr
			return nil
		}

		reconcileContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		reconciled, reconcileErr := plan.reconcile(reconcileContext)
		entry.ReconciledAt = journal.timestamp()
		if reconcileErr == nil && reconciled != nil {
			entry.State = journalApplied
			entry.Receipt = reconciled.receipt
			entry.UpdatedAt = journal.timestamp()
			out = successfulMutationResult(plan.request, entry, document, reconciled.pullRequest, codehostbroker.ReconciliationReconciledApplied, "reconciled_applied")
			out = withTransportMetadata(out, plan.adapter)
			return nil
		}
		if reconcileErr != nil && plan.decisiveReconcileError != nil && plan.decisiveReconcileError(reconcileErr) {
			entry.State = journalNotApplied
			entry.UpdatedAt = journal.timestamp()
			entry.FailureCode = normalizeProviderError(reconcileErr).Code
			out = mutationErrorResult(plan.request, entry, document, codehostbroker.ReconciliationNotApplied)
			out = withTransportMetadata(out, plan.adapter)
			out.observationRevision = preflight.observationRevision
			executionErr = reconcileErr
			return nil
		}
		entry.State = journalAmbiguous
		entry.UpdatedAt = journal.timestamp()
		out = mutationErrorResult(plan.request, entry, document, codehostbroker.ReconciliationAmbiguous)
		out = withTransportMetadata(out, plan.adapter)
		out.observationRevision = preflight.observationRevision
		executionErr = &providerError{code: codehostbroker.ErrorAmbiguousResult, message: "provider mutation outcome is ambiguous"}
		return nil
	})
	if err != nil {
		return adapterResult{}, &providerError{code: codehostbroker.ErrorProviderUnavailable, message: "mutation journal is unavailable"}
	}
	return out, executionErr
}

func (b *Broker) reconcileExistingMutation(ctx context.Context, plan mutationPlan, entry *journalEntry, document *journalDocument) (adapterResult, error, bool) {
	reconcileContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if entry.State == journalExternal {
		preflight, preflightErr := plan.preflight(reconcileContext)
		entry.ReconciledAt = b.now().UTC().Format(time.RFC3339Nano)
		if preflightErr == nil && preflight.external != nil {
			entry.Receipt = preflight.external.receipt
			entry.UpdatedAt = entry.ReconciledAt
			out := successfulMutationResult(plan.request, entry, document, preflight.external.pullRequest, codehostbroker.ReconciliationReplayed, "replayed")
			out.observationRevision = preflight.observationRevision
			return withTransportMetadata(out, plan.adapter), nil, true
		}
	}
	effect, err := plan.reconcile(reconcileContext)
	entry.ReconciledAt = b.now().UTC().Format(time.RFC3339Nano)
	if err == nil && effect != nil {
		status := codehostbroker.ReconciliationReplayed
		outcome := "replayed"
		if entry.ProviderAttempts == 0 && entry.State != journalExternal {
			status = codehostbroker.ReconciliationExternallyCompleted
			outcome = "externally_completed"
			entry.State = journalExternal
		} else if entry.State == journalDispatched || entry.State == journalAmbiguous {
			status = codehostbroker.ReconciliationReconciledApplied
			outcome = "reconciled_applied"
			entry.State = journalApplied
		} else {
			entry.State = journalApplied
		}
		entry.Receipt = effect.receipt
		entry.UpdatedAt = entry.ReconciledAt
		out := successfulMutationResult(plan.request, entry, document, effect.pullRequest, status, outcome)
		return withTransportMetadata(out, plan.adapter), nil, true
	}
	if err != nil && plan.decisiveReconcileError != nil && plan.decisiveReconcileError(err) {
		entry.State = journalNotApplied
		entry.UpdatedAt = entry.ReconciledAt
		entry.FailureCode = normalizeProviderError(err).Code
		out := mutationErrorResult(plan.request, entry, document, codehostbroker.ReconciliationNotApplied)
		return withTransportMetadata(out, plan.adapter), err, true
	}
	switch entry.State {
	case journalApplied, journalExternal, journalDispatched, journalAmbiguous:
		entry.State = journalAmbiguous
		entry.UpdatedAt = entry.ReconciledAt
		out := mutationErrorResult(plan.request, entry, document, codehostbroker.ReconciliationAmbiguous)
		return withTransportMetadata(out, plan.adapter), &providerError{code: codehostbroker.ErrorAmbiguousResult, message: "recorded mutation cannot be reconciled safely"}, true
	case journalInProgress:
		if err != nil {
			entry.State = journalAmbiguous
			entry.UpdatedAt = entry.ReconciledAt
			out := mutationErrorResult(plan.request, entry, document, codehostbroker.ReconciliationAmbiguous)
			return withTransportMetadata(out, plan.adapter), &providerError{code: codehostbroker.ErrorAmbiguousResult, message: "interrupted mutation cannot be reconciled safely"}, true
		}
		entry.State = journalNotApplied
		entry.UpdatedAt = entry.ReconciledAt
		return adapterResult{}, nil, false
	case journalNotApplied:
		if entry.ProviderAttempts > 0 {
			out := mutationErrorResult(plan.request, entry, document, codehostbroker.ReconciliationNotApplied)
			return withTransportMetadata(out, plan.adapter), replayedCreateFailure(entry.FailureCode), true
		}
		return adapterResult{}, nil, false
	default:
		entry.State = journalAmbiguous
		entry.UpdatedAt = entry.ReconciledAt
		out := mutationErrorResult(plan.request, entry, document, codehostbroker.ReconciliationAmbiguous)
		return withTransportMetadata(out, plan.adapter), &providerError{code: codehostbroker.ErrorAmbiguousResult, message: "mutation journal state cannot be reconciled safely"}, true
	}
}
