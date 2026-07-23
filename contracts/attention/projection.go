package attention

type AttentionRow struct {
	SchemaVersion int                   `json:"schema_version"`
	ID            string                `json:"id"`
	SourceKind    string                `json:"source_kind"`
	SourceID      string                `json:"source_id"`
	Project       ProjectReference      `json:"project"`
	Title         string                `json:"title"`
	Body          string                `json:"body,omitempty"`
	Timestamp     string                `json:"timestamp"`
	Revision      int64                 `json:"revision"`
	Actions       []ActionDescriptor    `json:"actions,omitempty"`
	Provenance    []ProvenanceReference `json:"provenance,omitempty"`
}

type AttentionSnapshot struct {
	SchemaVersion int            `json:"schema_version"`
	GeneratedAt   string         `json:"generated_at"`
	Revision      string         `json:"revision"`
	Rows          []AttentionRow `json:"rows"`
}
