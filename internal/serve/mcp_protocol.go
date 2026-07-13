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
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema is a JSON Schema for tool input.
type InputSchema struct {
	Type       string                `json:"type"`
	Properties map[string]PropSchema `json:"properties,omitempty"`
	Required   []string              `json:"required,omitempty"`
}

// PropSchema describes a single property in a JSON Schema.
type PropSchema struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Items       *PropSchema `json:"items,omitempty"`
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
