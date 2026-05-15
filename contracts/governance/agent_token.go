package governance

import "time"

// AgentToken is the signed credential an AI agent presents on every
// retrieval. The struct fields map one-to-one to JWT claims; this
// package defines the claim shape but not the signing or validation
// implementation (those live in the enforcement engine).
type AgentToken struct {
	// AgentID is the stable identifier of the agent. Maps to JWT "sub".
	AgentID string `json:"sub"`
	// ActsOnBehalfOf names the human user who issued the token. Maps to
	// a custom "obo" claim.
	ActsOnBehalfOf string `json:"obo"`
	// Issuer names the signing authority (server URL or "local"). Maps
	// to JWT "iss".
	Issuer string `json:"iss"`
	// SessionID links this token to one agent session for audit grouping.
	SessionID string `json:"sid"`
	// ReadScope bounds what the agent may retrieve.
	ReadScope Scope `json:"read_scope"`
	// WriteScope bounds what the agent may write back into the graph.
	WriteScope Scope `json:"write_scope"`
	// EgressClearance is the maximum classification the agent may include
	// in an LLM-context egress, independent of ReadScope.Classification.
	EgressClearance Classification `json:"egress_clearance"`
	// Capabilities lists named outpost capabilities granted to this agent
	// (e.g. "outpost:prod-api").
	Capabilities []string `json:"caps,omitempty"`
	// NotBefore is the earliest time the token is valid. Maps to JWT "nbf".
	NotBefore time.Time `json:"nbf"`
	// ExpiresAt is the hard expiry. Maps to JWT "exp". Required.
	ExpiresAt time.Time `json:"exp"`
}
