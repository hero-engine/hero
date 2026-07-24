package attention

type AttentionRow struct {
	SchemaVersion int                   `json:"schema_version"`
	ID            string                `json:"id"`
	SourceKind    string                `json:"source_kind"`
	SourceID      string                `json:"source_id"`
	Project       ProjectReference      `json:"project"`
	Title         string                `json:"title"`
	Summary       string                `json:"summary,omitempty"`
	Body          string                `json:"body,omitempty"`
	Timestamp     string                `json:"timestamp"`
	CreatedAt     string                `json:"created_at,omitempty"`
	UpdatedAt     string                `json:"updated_at,omitempty"`
	ActivityAt    string                `json:"activity_at,omitempty"`
	Group         string                `json:"group,omitempty"`
	Unread        bool                  `json:"unread,omitempty"`
	Today         bool                  `json:"today,omitempty"`
	Availability  string                `json:"availability,omitempty"`
	Revision      int64                 `json:"revision"`
	Actions       []ActionDescriptor    `json:"actions,omitempty"`
	Provenance    []ProvenanceReference `json:"provenance,omitempty"`
}

type AttentionCounts struct {
	Mail       int `json:"mail"`
	Focus      int `json:"focus"`
	Suggestion int `json:"suggestion"`
	Total      int `json:"total"`
}

const (
	AttentionStateCurrent = "current"
	AttentionStateEmpty   = "empty"
)

// AttentionWindow describes the bounded presentation applied to a full
// authoritative snapshot. Counts and revision still describe the complete
// projection; Rows contains only this window.
type AttentionWindow struct {
	State     string `json:"state"`
	Limit     int    `json:"limit"`
	Returned  int    `json:"returned"`
	Truncated bool   `json:"truncated"`
}

type AttentionSnapshot struct {
	SchemaVersion int              `json:"schema_version"`
	GeneratedAt   string           `json:"generated_at"`
	Revision      string           `json:"revision"`
	RefreshToken  string           `json:"refresh_token,omitempty"`
	Counts        AttentionCounts  `json:"counts,omitempty"`
	Rows          []AttentionRow   `json:"rows"`
	Window        *AttentionWindow `json:"window,omitempty"`
}
