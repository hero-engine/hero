package chat

// DispatchRequest is the wire envelope hero serve sends to an adapter
// (or to a runner-free slash handler). All fields except Kind, Prompt,
// and ConversationID are best-effort context the adapter may use.
type DispatchRequest struct {
	Kind           Kind            `json:"kind"`
	ConversationID string          `json:"conversation_id"`
	Prompt         string          `json:"prompt"`
	Context        DispatchContext `json:"context"`
	History        []HistoryTurn   `json:"history"`
	Slash          *SlashInvoc     `json:"slash,omitempty"`
}

// DispatchContext carries the active page state when the turn was
// submitted. Every field is optional; an empty Workspace is valid for
// scopeless dispatches (e.g. global ⌘K with no project).
type DispatchContext struct {
	Page      PageRef     `json:"page,omitempty"`
	Artifact  ArtifactRef `json:"artifact,omitempty"`
	Selection Selection   `json:"selection,omitempty"`
	Workspace string      `json:"workspace,omitempty"`
}

// PageRef identifies the home view that owns the chat surface.
type PageRef struct {
	Pack string `json:"pack,omitempty"`
	Home string `json:"home,omitempty"`
	View string `json:"view,omitempty"`
}

// ArtifactRef identifies the artifact the user is looking at.
type ArtifactRef struct {
	Kind string `json:"kind,omitempty"`
	Slug string `json:"slug,omitempty"`
}

// Selection is a fragment of the artifact the user has highlighted.
type Selection struct {
	Text  string `json:"text,omitempty"`
	Range string `json:"range,omitempty"`
}

// HistoryTurn is one prior turn in the conversation, in chronological
// order. Adapters typically receive the last ~50 turns.
type HistoryTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SlashInvoc captures a parsed leading-slash command and its
// argument string. Adapters MAY ignore this and treat Prompt as raw
// text; the runner-free slashes inside hero serve rely on it.
type SlashInvoc struct {
	Name string `json:"name"`
	Args string `json:"args,omitempty"`
}

// Event is one emission from an adapter or a runner-free slash
// handler. Type is the short suffix; the server adds the "chat."
// prefix when republishing on the bus.
type Event struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// Standard event-type suffixes. The bus event type is "chat." + these.
const (
	EvToken      = "token"
	EvToolCall   = "tool_call"
	EvToolResult = "tool_result"
	EvError      = "error"
	EvCost       = "cost"
	EvDone       = "done"
)

// TokenEvent constructs a chat.token event.
func TokenEvent(text string) Event {
	return Event{Type: EvToken, Payload: map[string]interface{}{"text": text}}
}

// ToolCallEvent constructs a chat.tool_call event.
func ToolCallEvent(name, args string) Event {
	return Event{Type: EvToolCall, Payload: map[string]interface{}{"name": name, "args": args}}
}

// ToolResultEvent constructs a chat.tool_result event.
func ToolResultEvent(name, preview string) Event {
	return Event{Type: EvToolResult, Payload: map[string]interface{}{"name": name, "preview": preview}}
}

// ErrorEvent constructs a chat.error event. link is optional (empty
// strings are omitted from the payload).
func ErrorEvent(code, message, link string) Event {
	p := map[string]interface{}{"code": code, "message": message}
	if link != "" {
		p["link"] = link
	}
	return Event{Type: EvError, Payload: p}
}

// CostEvent constructs a chat.cost event.
func CostEvent(usd float64, runner string) Event {
	return Event{Type: EvCost, Payload: map[string]interface{}{"usd": usd, "runner": runner}}
}

// DoneEvent constructs a chat.done event. outcome may be nil.
func DoneEvent(usd float64, outcome map[string]interface{}) Event {
	p := map[string]interface{}{"usd": usd}
	if outcome != nil {
		p["outcome"] = outcome
	}
	return Event{Type: EvDone, Payload: p}
}
