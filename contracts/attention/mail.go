package attention

type ProjectReference struct {
	PeerID       string `json:"peer_id"`
	RegistrySlug string `json:"registry_slug,omitempty"`
	DisplayName  string `json:"display_name"`
}

type ProvenanceReference struct {
	Kind      string `json:"kind"`
	SourceID  string `json:"source_id"`
	Label     string `json:"label,omitempty"`
	URL       string `json:"url,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type MailEnvelope struct {
	SchemaVersion  int                   `json:"schema_version"`
	ID             string                `json:"id"`
	Recipient      ProjectReference      `json:"recipient"`
	Sender         ProjectReference      `json:"sender"`
	Subject        string                `json:"subject"`
	Body           string                `json:"body"`
	Kind           string                `json:"kind,omitempty"`
	ThreadID       string                `json:"thread_id,omitempty"`
	InReplyTo      string                `json:"in_reply_to,omitempty"`
	Revision       int64                 `json:"revision,omitempty"`
	IdempotencyKey string                `json:"idempotency_key,omitempty"`
	CreatedAt      string                `json:"created_at"`
	Provenance     []ProvenanceReference `json:"provenance,omitempty"`
}

type MailReceipt struct {
	SchemaVersion       int              `json:"schema_version"`
	ID                  string           `json:"id"`
	EnvelopeID          string           `json:"envelope_id"`
	Recipient           ProjectReference `json:"recipient"`
	Kind                string           `json:"kind"`
	CreatedAt           string           `json:"created_at"`
	EnvelopeRevision    int64            `json:"envelope_revision,omitempty"`
	Revision            int64            `json:"revision,omitempty"`
	ReadAt              string           `json:"read_at,omitempty"`
	AcknowledgedAt      string           `json:"acknowledged_at,omitempty"`
	AcknowledgementNote string           `json:"acknowledgement_note,omitempty"`
	DismissedAt         string           `json:"dismissed_at,omitempty"`
	PromotedArtifact    *MailArtifact    `json:"promoted_artifact,omitempty"`
	FocusItemID         string           `json:"focus_item_id,omitempty"`
	Promotion           *MailPromotion   `json:"promotion,omitempty"`
	Actions             []MailAction     `json:"actions,omitempty"`
}

type MailArtifact struct {
	Slug   string `json:"slug"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type MailPromotion struct {
	IntakeSlug        string        `json:"intake_slug,omitempty"`
	IdempotencyKey    string        `json:"idempotency_key,omitempty"`
	RequestHash       string        `json:"request_hash,omitempty"`
	Captured          bool          `json:"captured,omitempty"`
	Artifact          *MailArtifact `json:"artifact,omitempty"`
	ProvenanceWritten bool          `json:"provenance_written,omitempty"`
	EventEmitted      bool          `json:"event_emitted,omitempty"`
}

type MailAction struct {
	ID             string `json:"id"`
	IdempotencyKey string `json:"idempotency_key"`
	RequestHash    string `json:"request_hash"`
	EventEmitted   bool   `json:"event_emitted,omitempty"`
}

type MailDelivery struct {
	SchemaVersion  int              `json:"schema_version"`
	MessageID      string           `json:"message_id"`
	ThreadID       string           `json:"thread_id"`
	Sender         ProjectReference `json:"sender"`
	Recipient      ProjectReference `json:"recipient"`
	IdempotencyKey string           `json:"idempotency_key"`
	DeliveredAt    string           `json:"delivered_at"`
}

const (
	MailKindQuestion = "question"
	MailKindRequest  = "request"
	MailKindResponse = "response"
	MailKindNotice   = "notice"
)

const (
	ReceiptRead         = "read"
	ReceiptAcknowledged = "acknowledged"
	ReceiptDismissed    = "dismissed"
	ReceiptPromoted     = "promoted"
)
