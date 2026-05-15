package governance

// Purpose names why a retrieval is happening. It feeds into egress
// policy: rules may permit a classification for user_view but deny it
// for llm_egress to an external model.
type Purpose string

const (
	// PurposeUserView is a direct user-facing read (CLI display, dashboard).
	PurposeUserView Purpose = "user_view"
	// PurposeAgentContext is a read assembled for an agent's working set,
	// not yet sent to an external model.
	PurposeAgentContext Purpose = "agent_context"
	// PurposeLLMExternal is a read that will be included in a prompt to
	// an external LLM provider. Enterprise policy may forbid this purpose
	// for any classification above Internal.
	PurposeLLMExternal Purpose = "llm_external"
	// PurposeLLMSelfHosted is a read that will be included in a prompt to
	// a self-hosted model under the org's control.
	PurposeLLMSelfHosted Purpose = "llm_self_hosted"
	// PurposeEnrichmentInput is a read whose result feeds a new
	// enrichment node; the enrichment inherits max classification.
	PurposeEnrichmentInput Purpose = "enrichment_input"
)
