package contracts

import (
	"encoding/json"
	"time"
)

// EventKind names the category of an Event payload. Concrete kinds are
// defined by the producer; this package ships only a few representative
// constants used by the transport layer itself.
type EventKind string

const (
	// EventKindCapture is emitted when a CLI captures a new node.
	EventKindCapture EventKind = "capture"
	// EventKindSync is emitted by CLI <-> server synchronization.
	EventKindSync EventKind = "sync"
	// EventKindAudit is emitted by the governance audit pipeline.
	EventKindAudit EventKind = "audit"
)

// Event is the envelope used for every message on the CLI <-> server
// wire. ContractsVersion identifies the shape of Payload; servers reject
// envelopes below ServerMinContractsVersion.
type Event struct {
	ContractsVersion int             `json:"contracts_version"`
	Kind             EventKind       `json:"kind"`
	OccurredAt       time.Time       `json:"occurred_at"`
	Payload          json.RawMessage `json:"payload"`
}
