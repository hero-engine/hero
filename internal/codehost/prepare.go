package codehost

import (
	"context"

	"github.com/hero-engine/hero/contracts/codehostbroker"
)

// Prepare performs the non-mutating preflight for one mutation operation and
// returns only the revision material needed by a separate Execute call.
func (b *Broker) Prepare(ctx context.Context, request codehostbroker.Request) codehostbroker.PreparationResponse {
	response := codehostbroker.PreparationResponse{
		Version:   codehostbroker.Version,
		Operation: request.Operation,
	}

	var (
		prepared codehostbroker.Request
		err      *codehostbroker.ContractError
	)
	switch {
	case request.Operation == codehostbroker.OperationCreatePullRequest:
		prepared, err = b.PrepareCreatePullRequest(ctx, request)
	case isCollaborationOperation(request.Operation):
		prepared, err = b.PrepareCollaboration(ctx, request)
	case isStateTransitionOperation(request.Operation):
		prepared, err = b.PrepareStateTransition(ctx, request)
	case request.Operation == codehostbroker.OperationMerge:
		prepared, err = b.PrepareMerge(ctx, request)
	default:
		err = contractError(codehostbroker.ErrorUnsupportedOperation, "operation does not support preparation", "operation")
	}
	if err != nil {
		response.Error = err
		return response
	}
	response.CapabilityRevision = prepared.CapabilityRevision
	response.ObservationRevision = prepared.ObservationRevision
	return response
}
