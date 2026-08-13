package serve

import "encoding/json"

// ---------------------------------------------------------------------------
// JSON-RPC 2.0 types
// ---------------------------------------------------------------------------

// JSONRPCRequest represents an incoming JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ErrCodeParse          = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)

// ---------------------------------------------------------------------------
// MCP protocol types
// ---------------------------------------------------------------------------

// MCPServerInfo describes this server.
//
// Schema and GraphSchema are Hero extensions: the compiled binary schema
// and the workspace graph schema. They let a harness SEE version/schema
// skew (a stray binary reading a newer graph) instead of inventing a
// migration narrative. Additive and omitempty, so existing clients that
// only read Name/Version are unaffected.
type MCPServerInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Schema      string `json:"schema,omitempty"`
	GraphSchema string `json:"graphSchema,omitempty"`
}

// MCPCapabilities declares what the server (or, for client-sent
// initialize params, the client) supports. The HeroDispatch capability
// is a Hero extension: clients that can act as Hero adapters declare
// it on initialize so the chat registry can route dispatches to them.
type MCPCapabilities struct {
	Tools        *MCPToolsCapability     `json:"tools,omitempty"`
	HeroDispatch *HeroDispatchCapability `json:"hero_dispatch,omitempty"`
}

// HeroDispatchCapability is the wire shape an MCP client sends to
// register as a Hero adapter. See the hero-chat-and-model spec for
// the contract.
type HeroDispatchCapability struct {
	Kinds     []string `json:"kinds"`
	Adapter   string   `json:"adapter"`
	Version   string   `json:"version"`
	SessionID string   `json:"session_id,omitempty"`
}

// MCPToolsCapability declares tool support.
type MCPToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializeResult is the response to initialize.
type InitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    MCPCapabilities `json:"capabilities"`
	ServerInfo      MCPServerInfo   `json:"serverInfo"`
	Instructions    string          `json:"instructions,omitempty"`
}

// ToolDefinition describes a single MCP tool.
//
// Category and Tier are Hero's progressive-disclosure facets. They are the
// single co-located source of truth declared on each tool's literal; a fold
// step (finalizeToolMetadata) copies them into Meta under the namespaced
// hero.dev/category and hero.dev/tier keys at tools/list build time. Both carry
// json:"-" so they NEVER serialize as top-level wire fields — a client reads
// them from _meta or not at all, keeping tools/list additive and MCP-conformant.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema InputSchema            `json:"inputSchema"`
	Annotations *ToolAnnotations       `json:"annotations,omitempty"`
	Meta        map[string]interface{} `json:"_meta,omitempty"`
	Category    ToolCategory           `json:"-"`
	Tier        ToolTier               `json:"-"`
}

// ToolCategory is the functional family a Hero MCP tool belongs to. It is a
// closed, versioned enum: a harness groups and defers tools by these families,
// so the set is additive — an existing value is never renamed, only new ones
// added. Emitted on tools/list under _meta[MetaKeyCategory]; never a top-level
// wire field. The one-line comment on each const is the meaning the published
// taxonomy doc derives from.
type ToolCategory string

const (
	// CategorySearchAndKnowledge: retrieval and Q&A over specs, knowledge,
	// provenance, and saved skill workflows.
	CategorySearchAndKnowledge ToolCategory = "search-and-knowledge"
	// CategorySpecLifecycle: author, claim, score, verify, plan, diagnose, and
	// attach test/demo artifacts to specs.
	CategorySpecLifecycle ToolCategory = "spec-lifecycle"
	// CategoryPlanningAndStatus: workspace status, ready-work queues, and
	// initiative/session steering.
	CategoryPlanningAndStatus ToolCategory = "planning-and-status"
	// CategoryCoverageAndQuality: heuristic analysis of drift, coverage,
	// conflicts, blockers, blast radius, and CI/quality risk.
	CategoryCoverageAndQuality ToolCategory = "coverage-and-quality"
	// CategoryActivityAndMetrics: cross-session activity feed and
	// sprint/velocity reporting.
	CategoryActivityAndMetrics ToolCategory = "activity-and-metrics"
	// CategoryCodeIntelligence: code-symbol search, enrichment,
	// feature-knowledge synthesis, and error-pattern capture.
	CategoryCodeIntelligence ToolCategory = "code-intelligence"
	// CategoryAttentionAndMail: Attention window, Project Mail, and Focus
	// operations.
	CategoryAttentionAndMail ToolCategory = "attention-and-mail"
	// CategoryExternalIntegrations: credential-brokered tracker and code-host
	// operations against external systems.
	CategoryExternalIntegrations ToolCategory = "external-integrations"
)

// toolCategorySet is the closed membership set the drift guard and taxonomy doc
// share. Every emitted category must be a member.
var toolCategorySet = map[ToolCategory]bool{
	CategorySearchAndKnowledge:   true,
	CategorySpecLifecycle:        true,
	CategoryPlanningAndStatus:    true,
	CategoryCoverageAndQuality:   true,
	CategoryActivityAndMetrics:   true,
	CategoryCodeIntelligence:     true,
	CategoryAttentionAndMail:     true,
	CategoryExternalIntegrations: true,
}

// ToolCategoryValid reports whether category is a member of the closed taxonomy.
func ToolCategoryValid(category ToolCategory) bool { return toolCategorySet[category] }

// ToolTier is Hero's advisory recommendation on whether a tool is hot-path
// enough for a harness to keep eager, or safe to defer until first use. It is
// advisory only — a harness MAY override it. Emitted under _meta[MetaKeyTier].
type ToolTier string

const (
	// TierEager marks the small session-warmup set a harness should broadcast
	// up front. Advisory: a harness may promote or demote any tool.
	TierEager ToolTier = "eager"
	// TierDeferrable marks tools safe to list by name and load on first use.
	TierDeferrable ToolTier = "deferrable"
)

// ToolTierValid reports whether tier is one of the two defined values.
func ToolTierValid(tier ToolTier) bool { return tier == TierEager || tier == TierDeferrable }

// Namespaced _meta keys for the progressive-disclosure facets. Namespaced per
// MCP convention so they never collide with another server's _meta.
const (
	MetaKeyCategory = "hero.dev/category"
	MetaKeyTier     = "hero.dev/tier"
)

// ToolAnnotations are MCP's advisory tool-behavior hints. They help clients
// present and route calls but are never treated as authorization.
type ToolAnnotations struct {
	Title           string `json:"title,omitempty"`
	ReadOnlyHint    *bool  `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool  `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool  `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool  `json:"openWorldHint,omitempty"`
}

// InputSchema is a JSON Schema for tool input.
type InputSchema struct {
	Type                 string                `json:"type"`
	Properties           map[string]PropSchema `json:"properties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	AdditionalProperties *bool                 `json:"additionalProperties,omitempty"`
}

// PropSchema describes a single property in a JSON Schema.
type PropSchema struct {
	Type                 string                `json:"type"`
	Description          string                `json:"description,omitempty"`
	Items                *PropSchema           `json:"items,omitempty"`
	Properties           map[string]PropSchema `json:"properties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	AdditionalProperties *bool                 `json:"additionalProperties,omitempty"`
	Enum                 []string              `json:"enum,omitempty"`
}

// ToolsListResult is the response to tools/list.
type ToolsListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

// ToolCallParams are the parameters for tools/call.
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolCallResult is the response to tools/call.
type ToolCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent is a single content item in a tool result.
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
