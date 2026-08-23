// Package mailthread defines the portable Project Mail thread lifecycle
// contract. It is additive to the message-oriented mailread/v1 contract.
package mailthread

import (
	"encoding/json"

	"github.com/hero-engine/hero/contracts/attention"
)

const (
	SchemaVersion             = 1
	BundleVersion             = 1
	ConformanceManifestSHA256 = "ef5c1b7fb929ff2e7f431c90292fbe65bc30bb57391775afbdfa6e52fe05339d"

	Compatibility = "Mail-read v1 remains authoritative for message receipts; unknown additive fields and identifiers remain inert and decodable, and unknown actions are never executable."
)

type Lifecycle string

const (
	LifecycleOpen     Lifecycle = "open"
	LifecycleResolved Lifecycle = "resolved"
	LifecycleArchived Lifecycle = "archived"
)

type GraceClass string

const (
	GraceInformational GraceClass = "informational"
	GraceLinkedWork    GraceClass = "linked_work"
)

const (
	ActionMarkRead   = "mark_read"
	ActionMarkUnread = "mark_unread"
	ActionResolve    = "resolve"
	ActionReopen     = "reopen"
	ActionArchive    = "archive"
	ActionRestore    = "restore"
)

type Identity struct {
	ProjectPeerID string `json:"project_peer_id"`
	ThreadID      string `json:"thread_id"`
}

type Resolution struct {
	Reason   string `json:"reason"`
	Source   string `json:"source"`
	Outcome  string `json:"outcome,omitempty"`
	SourceID string `json:"source_id,omitempty"`
}

type ActionRecord struct {
	ActionID       string `json:"action_id"`
	IdempotencyKey string `json:"idempotency_key"`
	RequestHash    string `json:"request_hash"`
	AppliedAt      string `json:"applied_at"`
}

type EventKind string

const (
	EventForegroundRead   EventKind = "foreground_read"
	EventReplySucceeded   EventKind = "reply_succeeded"
	EventActionSucceeded  EventKind = "action_succeeded"
	EventAdvisoryTerminal EventKind = "advisory_terminal"
	EventSpecOutTerminal  EventKind = "spec_out_terminal"
	EventLinkedTerminal   EventKind = "linked_work_terminal"
	EventInboundActivity  EventKind = "inbound_activity"
	EventGraceArchive     EventKind = "grace_archive"
)

const (
	OutcomeAnswered  = "answered"
	OutcomeCompleted = "completed"
	OutcomeRejected  = "rejected"
	OutcomeCancelled = "cancelled"
)

type Event struct {
	SchemaVersion    int       `json:"schema_version"`
	Identity         Identity  `json:"identity"`
	Kind             EventKind `json:"kind"`
	EventID          string    `json:"event_id"`
	ExpectedRevision int64     `json:"expected_revision"`
	OccurredAt       string    `json:"occurred_at"`
	Source           string    `json:"source"`
	SourceID         string    `json:"source_id"`
	Outcome          string    `json:"outcome,omitempty"`
	MessageID        string    `json:"message_id,omitempty"`
	PriorMessageIDs  []string  `json:"prior_message_ids,omitempty"`
}

type EventRecord struct {
	EventID         string    `json:"event_id"`
	Kind            EventKind `json:"kind"`
	RequestHash     string    `json:"request_hash"`
	AppliedAt       string    `json:"applied_at"`
	Source          string    `json:"source"`
	SourceID        string    `json:"source_id"`
	Outcome         string    `json:"outcome,omitempty"`
	FromLifecycle   Lifecycle `json:"from_lifecycle"`
	ToLifecycle     Lifecycle `json:"to_lifecycle"`
	PriorMessageIDs []string  `json:"prior_message_ids,omitempty"`
}

type State struct {
	SchemaVersion     int            `json:"schema_version"`
	Identity          Identity       `json:"identity"`
	Lifecycle         Lifecycle      `json:"lifecycle"`
	Revision          int64          `json:"revision"`
	Resolution        *Resolution    `json:"resolution,omitempty"`
	GraceClass        GraceClass     `json:"grace_class,omitempty"`
	ResolvedAt        string         `json:"resolved_at,omitempty"`
	ArchiveEligibleAt string         `json:"archive_eligible_at,omitempty"`
	ArchivedAt        string         `json:"archived_at,omitempty"`
	Actions           []ActionRecord `json:"actions,omitempty"`
	Events            []EventRecord  `json:"events,omitempty"`
}

type ReadSummary struct {
	MessageCount int `json:"message_count"`
	UnreadCount  int `json:"unread_count"`
}

type ThreadView struct {
	State   State                        `json:"state"`
	Read    ReadSummary                  `json:"read"`
	Actions []attention.ActionDescriptor `json:"actions"`
}

type CapabilitySet struct {
	Receipt   []attention.ActionDescriptor `json:"receipt"`
	Lifecycle []attention.ActionDescriptor `json:"lifecycle"`
}

type ActionInput struct {
	Reason     string     `json:"reason,omitempty"`
	Source     string     `json:"source,omitempty"`
	Outcome    string     `json:"outcome,omitempty"`
	SourceID   string     `json:"source_id,omitempty"`
	GraceClass GraceClass `json:"grace_class,omitempty"`
}

type ActionRequest struct {
	SchemaVersion  int             `json:"schema_version"`
	Identity       Identity        `json:"identity"`
	ActionID       string          `json:"action_id"`
	ThreadRevision int64           `json:"thread_revision"`
	IdempotencyKey string          `json:"idempotency_key"`
	Input          json.RawMessage `json:"input,omitempty"`
}

type ActionResponse struct {
	SchemaVersion int                      `json:"schema_version"`
	Thread        *ThreadView              `json:"thread,omitempty"`
	Error         *attention.ContractError `json:"error,omitempty"`
}

type MigrationResult struct {
	SchemaVersion int        `json:"schema_version"`
	Thread        ThreadView `json:"thread"`
	Created       bool       `json:"created"`
	Source        string     `json:"source"`
}

type ContractResponse struct {
	SchemaVersion        int    `json:"schema_version"`
	BundleVersion        int    `json:"bundle_version"`
	BundleManifestSHA256 string `json:"bundle_manifest_sha256"`
	Compatibility        string `json:"compatibility"`
}
