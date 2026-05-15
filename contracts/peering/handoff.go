package peering

import "time"

// HandoffStatus enumerates the originator-side states a spec moves
// through during a handoff. Receiver-side specs use the standard
// spec lifecycle (planning → in-review → delivering → completed).
type HandoffStatus string

const (
	// StatusHandedOff is the initial transition — "I gave this to peer B."
	StatusHandedOff HandoffStatus = "handed_off"

	// StatusAwaitingPeer is the steady state — "B has it, I'm waiting."
	StatusAwaitingPeer HandoffStatus = "awaiting_peer"

	// StatusHandedBack means the peer is done (or bounced); the ball
	// is back on the originator's side for verification or follow-up.
	StatusHandedBack HandoffStatus = "handed_back"
)

// ReceivedFrom is the frontmatter block on a receiver-side spec that
// records its origin. The combination of PeerID + OriginatorSlug
// uniquely identifies the originating spec across workspaces.
type ReceivedFrom struct {
	// PeerID is the originating workspace's UUID. Canonical.
	PeerID string `yaml:"peer_id" json:"peer_id"`

	// PeerAliasDisplay is the originator's local alias for THIS
	// workspace at handoff time — recorded for human reading only.
	// Resolution always uses PeerID.
	PeerAliasDisplay string `yaml:"peer_alias_display,omitempty" json:"peer_alias_display,omitempty"`

	// OriginatorSlug is the slug of the originating spec on the
	// originator's side.
	OriginatorSlug string `yaml:"originator_slug" json:"originator_slug"`

	// HandedOffAt is the timestamp of the original transition.
	HandedOffAt time.Time `yaml:"handed_off_at" json:"handed_off_at"`

	// AtCommit is the originator's git commit SHA at handoff time,
	// recorded for audit.
	AtCommit string `yaml:"at_commit,omitempty" json:"at_commit,omitempty"`

	// Reason is the free-form rationale captured at handoff.
	Reason string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// TrailDirection records who initiated a trail entry from this side's
// perspective.
type TrailDirection string

const (
	// DirectionOut means this side initiated the action.
	DirectionOut TrailDirection = "out"
	// DirectionIn means this side received the action.
	DirectionIn TrailDirection = "in"
)

// TrailMode classifies a trail entry by interaction kind.
type TrailMode string

const (
	// ModeAdvisory — sync peer call returning findings, no writes.
	ModeAdvisory TrailMode = "advisory"
	// ModeSpecOut — sync peer call that produced a peer-side spec.
	ModeSpecOut TrailMode = "spec-out"
	// ModeAsyncDrop — async `hero handoff` drop.
	ModeAsyncDrop TrailMode = "async-drop"
	// ModeFullDelivery — sync peer call with commit/PR result (v2).
	ModeFullDelivery TrailMode = "full-delivery"
	// ModeHandedBack — peer-side completion or explicit bounce
	// signaling the originator's spec should move to handed_back.
	ModeHandedBack TrailMode = "handed-back"
	// ModeAccepted — originator picks up a handed_back spec via
	// `hero handoff accept`.
	ModeAccepted TrailMode = "accepted"
)

// TrailEntry is one line in a spec's `## Handoff Trail` section.
// Every cross-workspace event involving a spec writes a trail entry
// on both sides (originator + peer), with PeerID as the join key.
type TrailEntry struct {
	// At is the wall-clock time of the event.
	At time.Time `yaml:"at" json:"at"`

	// Direction — out (this side initiated) or in (this side received).
	Direction TrailDirection `yaml:"direction" json:"direction"`

	// PeerAliasDisplay is the local alias for the other side at event
	// time — display only; not a join key.
	PeerAliasDisplay string `yaml:"peer_alias_display,omitempty" json:"peer_alias_display,omitempty"`

	// PeerID is the other side's workspace UUID. Canonical join key.
	PeerID string `yaml:"peer_id" json:"peer_id"`

	// Mode classifies the interaction.
	Mode TrailMode `yaml:"mode" json:"mode"`

	// OriginatingSpec is the originator-side slug (set on the
	// originator's trail entries; mirrored on the receiver's).
	OriginatingSpec string `yaml:"originating_spec,omitempty" json:"originating_spec,omitempty"`

	// PeerSpec is "<peer-alias-or-id>/<slug>" referencing the
	// counterpart spec on the other side.
	PeerSpec string `yaml:"peer_spec,omitempty" json:"peer_spec,omitempty"`

	// PeerStatus snapshots the peer counterpart's status at trail-
	// write time, so a reader can see the peer's state without a
	// cross-repo round trip.
	PeerStatus string `yaml:"peer_status,omitempty" json:"peer_status,omitempty"`

	// AtCommit is this side's git commit SHA at event time.
	AtCommit string `yaml:"at_commit,omitempty" json:"at_commit,omitempty"`

	// ResultRef is a free-form reference to the work product (commit
	// SHA, PR URL, peer call_id) when applicable. Used by
	// ModeHandedBack and ModeFullDelivery.
	ResultRef string `yaml:"result_ref,omitempty" json:"result_ref,omitempty"`

	// Reason is the free-form rationale captured at event time.
	Reason string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// HandoffRecord is the wire shape for a complete handoff operation,
// used by event payloads and (Phase 4+) cloud transport. The on-disk
// representation is the originator's spec status + trail entries plus
// the receiver's `received_from` block + trail entries.
type HandoffRecord struct {
	// ContractsVersion is the PeeringContractsVersion at write time.
	ContractsVersion int `json:"contracts_version"`

	// OriginPeerID is the originator workspace's UUID.
	OriginPeerID string `json:"origin_peer_id"`

	// OriginSlug is the originator spec slug.
	OriginSlug string `json:"origin_slug"`

	// TargetPeerID is the receiver workspace's UUID.
	TargetPeerID string `json:"target_peer_id"`

	// TargetSlug is the chosen receiver spec slug (after any
	// collision suffix).
	TargetSlug string `json:"target_slug"`

	// TargetType is the receiver spec type (feature, bug, …).
	TargetType string `json:"target_type,omitempty"`

	// Mode is the kind of handoff (async-drop, spec-out, …).
	Mode TrailMode `json:"mode"`

	// HandedOffAt is the originator-side transition timestamp.
	HandedOffAt time.Time `json:"handed_off_at"`

	// AtCommit is the originator's git commit SHA at handoff time.
	AtCommit string `json:"at_commit,omitempty"`

	// Reason is the rationale captured at handoff.
	Reason string `json:"reason,omitempty"`
}
