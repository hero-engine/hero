package governance

import "time"

// AuditEvent is the append-only record emitted for every retrieval that
// returns a non-empty allowed set. Audit events are themselves graph
// nodes of kind "event" with classification Restricted.
type AuditEvent struct {
	EventID        string         `json:"event_id"`
	OrgID          string         `json:"org_id"`
	OccurredAt     time.Time      `json:"occurred_at"`
	PrincipalID    string         `json:"principal_id"`
	PrincipalKind  PrincipalKind  `json:"principal_kind"`
	AgentSessionID string         `json:"agent_session_id,omitempty"`
	Purpose        Purpose        `json:"purpose"`
	PolicyVersion  int            `json:"policy_version"`
	PolicyNodeID   string         `json:"policy_node_id"`
	Scope          Scope          `json:"scope"`
	AllowedNodeIDs []string       `json:"allowed_node_ids"`
	DeniedCount    int            `json:"denied_count"`
	EgressTarget   string         `json:"egress_target,omitempty"`
	EgressClassMax Classification `json:"egress_class_max"`
	DecisionReason string         `json:"decision_reason,omitempty"`
}

// AuditToken is returned by Retriever.Filter and must be passed to the
// audit emitter when the caller commits to using the filtered result.
// Tokens expire if not consumed within a short window.
type AuditToken struct {
	// TokenID uniquely identifies the pending audit record.
	TokenID string `json:"token_id"`
	// EventID is the AuditEvent.EventID this token will produce.
	EventID string `json:"event_id"`
	// IssuedAt is when Filter produced the token.
	IssuedAt time.Time `json:"issued_at"`
	// ExpiresAt is the deadline by which the caller must emit-or-discard.
	ExpiresAt time.Time `json:"expires_at"`
}
