package attention

type DeferredWorkSuggestion struct {
	SchemaVersion int                   `json:"schema_version"`
	ID            string                `json:"id"`
	Project       ProjectReference      `json:"project"`
	Title         string                `json:"title"`
	Prompt        string                `json:"prompt"`
	CreatedAt     string                `json:"created_at"`
	Provenance    []ProvenanceReference `json:"provenance,omitempty"`
}

type SuggestionDecisionRequest struct {
	SchemaVersion  int    `json:"schema_version"`
	SuggestionID   string `json:"suggestion_id"`
	Decision       string `json:"decision"`
	IdempotencyKey string `json:"idempotency_key"`
}
