package governance

import (
	"context"
	"time"
)

// Retriever is the single seam every read against the graph flows
// through. The contract is interface-only; implementations live in the
// enforcement engine. No code path may bypass this interface — there
// is no admin escape hatch.
type Retriever interface {
	// Filter takes a candidate node set and returns the subset the caller
	// is allowed to see, per-node decisions for audit, and an AuditToken
	// the caller MUST emit when it commits to using the result.
	Filter(ctx context.Context, q Query, candidates []NodeID) (
		allowed []NodeID,
		decisions []NodeDecision,
		audit AuditToken,
		err error,
	)
}

// NodeID is the stable identifier for a graph node within the governance
// contracts. It mirrors contracts.NodeID; duplicated here to keep this
// subpackage independent of its parent.
type NodeID string

// Query carries the caller context for a Filter invocation.
type Query struct {
	Principal   Principal
	Scope       Scope
	Purpose     Purpose
	RequestedAt time.Time
}

// NodeDecision is the per-node verdict Filter returns alongside the
// allowed set. Reason names the matched policy rule and its version.
type NodeDecision struct {
	NodeID         NodeID         `json:"node_id"`
	Allowed        bool           `json:"allowed"`
	Reason         string         `json:"reason"`
	Classification Classification `json:"classification"`
	Subjects       []Subject      `json:"subjects,omitempty"`
}
