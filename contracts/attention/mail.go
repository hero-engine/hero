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
	SchemaVersion int                   `json:"schema_version"`
	ID            string                `json:"id"`
	Recipient     ProjectReference      `json:"recipient"`
	Sender        ProjectReference      `json:"sender"`
	Subject       string                `json:"subject"`
	Body          string                `json:"body"`
	CreatedAt     string                `json:"created_at"`
	Provenance    []ProvenanceReference `json:"provenance,omitempty"`
}

type MailReceipt struct {
	SchemaVersion int              `json:"schema_version"`
	ID            string           `json:"id"`
	EnvelopeID    string           `json:"envelope_id"`
	Recipient     ProjectReference `json:"recipient"`
	Kind          string           `json:"kind"`
	CreatedAt     string           `json:"created_at"`
}

const (
	ReceiptRead         = "read"
	ReceiptAcknowledged = "acknowledged"
	ReceiptDismissed    = "dismissed"
	ReceiptPromoted     = "promoted"
)
