// Package mailread defines the portable v1 contract for cross-project Mail
// metadata reads, exact detail, and source-owned mutations.
package mailread

import (
	"encoding/json"

	"github.com/hero-engine/hero/contracts/attention"
)

const (
	SchemaVersion             = 1
	BundleVersion             = 1
	DefaultListLimit          = 20
	MaxListLimit              = 100
	MaxListDiagnostics        = 8
	MaxCursorBytes            = 4096
	ConformanceManifestSHA256 = "69cc93c0a4566d1fd4f9678ead9d2fef3c7c9655559d2c55eb94551f0facb69e"

	Compatibility = "Unknown additive fields and identifiers must remain inert but decodable; never grant executable behavior from an unknown value."
)

const (
	ActionMarkRead    = "mark_read"
	ActionAcknowledge = "acknowledge"
	ActionDismiss     = "dismiss"
	ActionPromote     = "promote"
	ActionAddToToday  = "add_to_today"
	ActionReply       = "reply"
)

type ListRequest struct {
	SchemaVersion int    `json:"schema_version"`
	ProjectPeerID string `json:"project_peer_id,omitempty"`
	ThreadID      string `json:"thread_id,omitempty"`
	UnreadOnly    *bool  `json:"unread_only,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Cursor        string `json:"cursor,omitempty"`
}

type PageMetadata struct {
	Limit    int  `json:"limit"`
	Returned int  `json:"returned"`
	HasMore  bool `json:"has_more"`
}

type ReceiptView struct {
	Revision         int64                   `json:"revision"`
	Unread           bool                    `json:"unread"`
	ReadAt           string                  `json:"read_at,omitempty"`
	AcknowledgedAt   string                  `json:"acknowledged_at,omitempty"`
	DismissedAt      string                  `json:"dismissed_at,omitempty"`
	PromotedArtifact *attention.MailArtifact `json:"promoted_artifact,omitempty"`
	FocusItemID      string                  `json:"focus_item_id,omitempty"`
}

type MessageSummary struct {
	Project    attention.ProjectReference   `json:"project"`
	MessageID  string                       `json:"message_id"`
	ThreadID   string                       `json:"thread_id,omitempty"`
	InReplyTo  string                       `json:"in_reply_to,omitempty"`
	Sender     attention.ProjectReference   `json:"sender"`
	Recipient  attention.ProjectReference   `json:"recipient"`
	Subject    string                       `json:"subject"`
	Kind       string                       `json:"kind,omitempty"`
	CreatedAt  string                       `json:"created_at"`
	ActivityAt string                       `json:"activity_at"`
	Unread     bool                         `json:"unread"`
	Receipt    ReceiptView                  `json:"receipt"`
	Actions    []attention.ActionDescriptor `json:"actions"`
}

type ListResponse struct {
	SchemaVersion int                       `json:"schema_version"`
	Revision      string                    `json:"revision,omitempty"`
	TotalCount    int                       `json:"total_count"`
	UnreadCount   int                       `json:"unread_count"`
	Items         []MessageSummary          `json:"items"`
	Page          *PageMetadata             `json:"page,omitempty"`
	NextCursor    string                    `json:"next_cursor,omitempty"`
	Diagnostics   []attention.ContractError `json:"diagnostics,omitempty"`
	Error         *attention.ContractError  `json:"error,omitempty"`
}

type DetailResponse struct {
	SchemaVersion int                          `json:"schema_version"`
	Project       *attention.ProjectReference  `json:"project,omitempty"`
	Envelope      *attention.MailEnvelope      `json:"envelope,omitempty"`
	ActivityAt    string                       `json:"activity_at,omitempty"`
	Unread        *bool                        `json:"unread,omitempty"`
	Receipt       *ReceiptView                 `json:"receipt,omitempty"`
	Actions       []attention.ActionDescriptor `json:"actions,omitempty"`
	Error         *attention.ContractError     `json:"error,omitempty"`
}

type ActionRequest struct {
	SchemaVersion   int             `json:"schema_version"`
	ProjectPeerID   string          `json:"project_peer_id"`
	MessageID       string          `json:"message_id"`
	ActionID        string          `json:"action_id"`
	ReceiptRevision int64           `json:"receipt_revision"`
	IdempotencyKey  string          `json:"idempotency_key"`
	Input           json.RawMessage `json:"input,omitempty"`
}

type ActionResponse struct {
	SchemaVersion int                            `json:"schema_version"`
	Project       *attention.ProjectReference    `json:"project,omitempty"`
	MessageID     string                         `json:"message_id,omitempty"`
	Receipt       *ReceiptView                   `json:"receipt,omitempty"`
	Actions       []attention.ActionDescriptor   `json:"actions,omitempty"`
	Navigation    *attention.NavigationReference `json:"navigation,omitempty"`
	Error         *attention.ContractError       `json:"error,omitempty"`
}

type ReplyRequest struct {
	SchemaVersion  int    `json:"schema_version"`
	ProjectPeerID  string `json:"project_peer_id"`
	MessageID      string `json:"message_id"`
	ThreadID       string `json:"thread_id"`
	Body           string `json:"body"`
	Subject        string `json:"subject,omitempty"`
	Kind           string `json:"kind,omitempty"`
	IdempotencyKey string `json:"idempotency_key"`
}

type ReplyResponse struct {
	SchemaVersion int                      `json:"schema_version"`
	Delivery      *attention.MailDelivery  `json:"delivery,omitempty"`
	Error         *attention.ContractError `json:"error,omitempty"`
}

type ContractResponse struct {
	SchemaVersion        int    `json:"schema_version"`
	BundleVersion        int    `json:"bundle_version"`
	BundleManifestSHA256 string `json:"bundle_manifest_sha256"`
	Compatibility        string `json:"compatibility"`
}
