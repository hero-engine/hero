package peering

import "time"

// PeerCallMode classifies a sync peer call by its allowed effects.
type PeerCallMode string

const (
	// PeerCallAdvisory — investigate and return findings. No writes
	// on the peer side beyond the call record. v1.
	PeerCallAdvisory PeerCallMode = "advisory"

	// PeerCallSpecOut — run the peer's /design flow under a subagent
	// and persist a spec on the peer side with `received_from`. v1.
	PeerCallSpecOut PeerCallMode = "spec-out"

	// PeerCallFull — design AND deliver on the peer side. Gated by
	// approval + budget. v2.
	PeerCallFull PeerCallMode = "full"
)

// BudgetSpec caps how much work a peer call may consume.
type BudgetSpec struct {
	Turns  int `yaml:"turns,omitempty" json:"turns,omitempty"`
	Tokens int `yaml:"tokens,omitempty" json:"tokens,omitempty"`
}

// BudgetConsumed records actual consumption after the call returns.
type BudgetConsumed struct {
	Turns  int `yaml:"turns" json:"turns"`
	Tokens int `yaml:"tokens" json:"tokens"`
}

// PeerCallRequest is the envelope a caller in A sends to B's
// peer-call subagent.
type PeerCallRequest struct {
	// ContractsVersion is the PeeringContractsVersion at call time.
	ContractsVersion int `json:"contracts_version"`

	// CallID is a unique ULID for this call. Recorded on both sides.
	CallID string `json:"call_id"`

	// OriginPeerID is the caller workspace's UUID.
	OriginPeerID string `json:"origin_peer_id"`

	// TargetPeerID is the target workspace's UUID.
	TargetPeerID string `json:"target_peer_id"`

	// Mode is the interaction mode.
	Mode PeerCallMode `json:"mode"`

	// Prompt is the user-supplied prompt for the peer's subagent.
	Prompt string `json:"prompt"`

	// Budget caps consumption. Zero values mean "use default".
	Budget BudgetSpec `json:"budget,omitempty"`

	// RelatedSpec is the originator-side spec slug, if any — used
	// to anchor trail entries.
	RelatedSpec string `json:"related_spec,omitempty"`

	// Reason is a free-form rationale captured at call time.
	Reason string `json:"reason,omitempty"`

	// At is the wall-clock time of the call.
	At time.Time `json:"at"`

	// AtCommit is the originator's git commit SHA at call time.
	AtCommit string `json:"at_commit,omitempty"`
}

// PeerCallResultKind enumerates the result shapes a peer call returns.
type PeerCallResultKind string

const (
	// ResultFindings — advisory mode result: free-form findings text.
	ResultFindings PeerCallResultKind = "findings"
	// ResultSpecRef — spec-out mode result: a peer-side spec slug.
	ResultSpecRef PeerCallResultKind = "spec-ref"
	// ResultCommitRef — full-delivery mode result: a commit/PR ref.
	ResultCommitRef PeerCallResultKind = "commit-ref"
)

// PeerCallResult is the envelope the peer subagent returns.
type PeerCallResult struct {
	// ContractsVersion is the PeeringContractsVersion at return time.
	ContractsVersion int `json:"contracts_version"`

	// CallID echoes the request's CallID.
	CallID string `json:"call_id"`

	// Mode echoes the request's mode.
	Mode PeerCallMode `json:"mode"`

	// Kind describes which result fields are populated.
	Kind PeerCallResultKind `json:"kind"`

	// Findings is set when Kind == ResultFindings (advisory).
	Findings string `json:"findings,omitempty"`

	// SpecSlug is set when Kind == ResultSpecRef (spec-out).
	SpecSlug string `json:"spec_slug,omitempty"`

	// PeerStatus snapshots the produced spec's status.
	PeerStatus string `json:"peer_status,omitempty"`

	// CommitRef is set when Kind == ResultCommitRef (full delivery).
	CommitRef string `json:"commit_ref,omitempty"`

	// PRURL is set when Kind == ResultCommitRef and a PR was opened.
	PRURL string `json:"pr_url,omitempty"`

	// BudgetConsumed records what the peer actually used.
	BudgetConsumed BudgetConsumed `json:"budget_consumed"`

	// At is the wall-clock time of the result.
	At time.Time `json:"at"`

	// Error is non-empty when the call failed to complete.
	Error string `json:"error,omitempty"`
}
