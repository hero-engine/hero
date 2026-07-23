package attention

type DeferredWorkSuggestion struct {
	SchemaVersion int                   `json:"schema_version"`
	ID            string                `json:"id"`
	Project       ProjectReference      `json:"project"`
	Kind          string                `json:"kind,omitempty"`
	Title         string                `json:"title"`
	Reason        string                `json:"reason,omitempty"`
	Prompt        string                `json:"prompt"`
	CreatedAt     string                `json:"created_at"`
	ExpiresAt     string                `json:"expires_at,omitempty"`
	Revision      int64                 `json:"revision,omitempty"`
	State         string                `json:"state,omitempty"`
	Provenance    []ProvenanceReference `json:"provenance,omitempty"`
	Actions       []ActionDescriptor    `json:"actions,omitempty"`
}

type SuggestionDecisionRequest struct {
	SchemaVersion  int    `json:"schema_version"`
	SuggestionID   string `json:"suggestion_id"`
	Decision       string `json:"decision"`
	Revision       int64  `json:"revision,omitempty"`
	IdempotencyKey string `json:"idempotency_key"`
}
