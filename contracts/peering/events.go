package peering

import "time"

// EventKind names a peer-related event type. These ride on the
// existing events.log alongside other workspace events.
type EventKind string

const (
	// EventHandoffSent fires on the originator after an async drop or
	// Historical spec-out call wrote the receiver-side scaffold.
	EventHandoffSent EventKind = "peer.handoff.sent"

	// EventHandoffReceived fires on the receiver when a peer drops a
	// historical scaffolded spec.
	EventHandoffReceived EventKind = "peer.handoff.received"

	// EventHandoffBounced fires when the receiver explicitly returns
	// a spec without completion (rejection or "not for me").
	EventHandoffBounced EventKind = "peer.handoff.bounced"

	// EventHandoffAccepted fires on the originator when the user
	// runs `hero handoff accept` on a handed_back spec.
	EventHandoffAccepted EventKind = "peer.handoff.accepted"

	// EventCallInvoked fires on the originator when a peer call
	// starts.
	EventCallInvoked EventKind = "peer.call.invoked"

	// EventCallCompleted fires on the originator when a peer call
	// returns a result.
	EventCallCompleted EventKind = "peer.call.completed"

	// EventPeerIDMinted fires once on the very first hero invocation
	// in a workspace that lacks a peer_id — records the moment of
	// identity assignment so it's recoverable from the audit log.
	EventPeerIDMinted EventKind = "workspace.peer_id_minted"
)

// HandoffEvent is the payload for peer.handoff.{sent,received,bounced,
// accepted}.
type HandoffEvent struct {
	ContractsVersion int       `json:"contracts_version"`
	Kind             EventKind `json:"kind"`
	OccurredAt       time.Time `json:"occurred_at"`
	OriginPeerID     string    `json:"origin_peer_id"`
	OriginSlug       string    `json:"origin_slug"`
	TargetPeerID     string    `json:"target_peer_id"`
	TargetSlug       string    `json:"target_slug"`
	Mode             TrailMode `json:"mode"`
	AtCommit         string    `json:"at_commit,omitempty"`
	Reason           string    `json:"reason,omitempty"`
}

// CallEvent is the payload for peer.call.{invoked,completed}.
type CallEvent struct {
	ContractsVersion int                `json:"contracts_version"`
	Kind             EventKind          `json:"kind"`
	OccurredAt       time.Time          `json:"occurred_at"`
	CallID           string             `json:"call_id"`
	OriginPeerID     string             `json:"origin_peer_id"`
	TargetPeerID     string             `json:"target_peer_id"`
	Mode             PeerCallMode       `json:"mode"`
	RelatedSpec      string             `json:"related_spec,omitempty"`
	ResultKind       PeerCallResultKind `json:"result_kind,omitempty"`
	BudgetConsumed   BudgetConsumed     `json:"budget_consumed,omitempty"`
	Error            string             `json:"error,omitempty"`
}

// PeerIDMintedEvent is the payload for workspace.peer_id_minted.
type PeerIDMintedEvent struct {
	ContractsVersion int       `json:"contracts_version"`
	Kind             EventKind `json:"kind"`
	OccurredAt       time.Time `json:"occurred_at"`
	PeerID           string    `json:"peer_id"`
	// Trigger describes what caused the mint: "init" (fresh hero init)
	// or "migration" (first invocation on a pre-peer_id workspace).
	Trigger string `json:"trigger"`
}

// AllEventKinds returns every peer-related event kind. Useful for
// validators that need an enumerable list.
func AllEventKinds() []EventKind {
	return []EventKind{
		EventHandoffSent,
		EventHandoffReceived,
		EventHandoffBounced,
		EventHandoffAccepted,
		EventCallInvoked,
		EventCallCompleted,
		EventPeerIDMinted,
	}
}

// IsPeerEventKind reports whether s names a peer-related event kind.
func IsPeerEventKind(s string) bool {
	for _, k := range AllEventKinds() {
		if string(k) == s {
			return true
		}
	}
	return false
}
