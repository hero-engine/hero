package contracts

import (
	"time"

	"github.com/hero-engine/hero/contracts/governance"
)

// NodeID is the stable identifier for a graph node on the wire.
type NodeID string

// Kind names the category of a graph node (e.g. "note", "spec",
// "decision", "policy"). Kinds are open-ended strings; this package
// does not enumerate them.
type Kind string

// Node is the base shape every graph node carries on the wire.
// Type-specific payload travels separately; this is the envelope used
// by retrieval, audit, and policy code that does not need the payload.
type Node struct {
	ID             NodeID
	Kind           Kind
	Classification governance.Classification
	Subjects       []governance.Subject
	Origin         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
