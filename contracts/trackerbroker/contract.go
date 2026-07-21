// Package trackerbroker defines the stable JSON contract shared by Hero's
// in-process tracker broker, CLI, and MCP surfaces.
package trackerbroker

import (
	"embed"
	"encoding/json"
)

const Version = "tracker-broker/v1"

type Operation string

const (
	OperationGetIssue Operation = "get_issue"
	OperationSearch   Operation = "search"
	OperationRequest  Operation = "request"
	OperationCLI      Operation = "cli"
)

type Effect string

const (
	EffectRead               Effect = "read"
	EffectWriteIdempotent    Effect = "write_idempotent"
	EffectWriteNonIdempotent Effect = "write_non_idempotent"
)

type Detail string

const (
	DetailNormalized Detail = "normalized"
	DetailEvidence   Detail = "evidence"
)

type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

// Stable v1 error codes. Consumers branch on these values and render Message;
// they must not parse Message text.
const (
	ErrorInvalidInput          = "invalid_input"
	ErrorAmbiguousConnection   = "ambiguous_connection"
	ErrorConnectionNotFound    = "connection_not_found"
	ErrorConnectionUnavailable = "connection_unavailable"
	ErrorCredentialUnavailable = "credential_unavailable"
	ErrorConfiguration         = "configuration_error"
	ErrorInvalidIssueID        = "invalid_issue_id"
	ErrorInvalidDetail         = "invalid_detail"
	ErrorInvalidQuery          = "invalid_query"
	ErrorInvalidLimit          = "invalid_limit"
	ErrorCursorMismatch        = "cursor_mismatch"
	ErrorUnsupportedOperation  = "unsupported_operation"
	ErrorProvider              = "provider_error"
	ErrorProviderHTTP          = "provider_http_error"
	ErrorEncoding              = "encoding_error"
	ErrorInvalidMethod         = "invalid_method"
	ErrorInvalidRequest        = "invalid_request"
	ErrorUnsafeRequest         = "unsafe_request"
	ErrorUnsafeHeaders         = "unsafe_headers"
	ErrorUnsafeRedirect        = "unsafe_redirect"
	ErrorUnsafeCLI             = "unsafe_cli"
	ErrorExecutableUnavailable = "executable_unavailable"
	ErrorExecution             = "execution_error"
	ErrorCommandFailed         = "command_failed"
	ErrorInputTooLarge         = "input_too_large"
	ErrorCancelled             = "cancelled"
)

// Response is the one envelope returned from every broker surface. Result is
// operation-owned structured JSON; Body is reserved for arbitrary provider
// HTTP response bytes represented as text.
type Response struct {
	Version      string          `json:"version"`
	Operation    Operation       `json:"operation"`
	Provider     string          `json:"provider,omitempty"`
	ConnectionID string          `json:"connection_id,omitempty"`
	Effect       Effect          `json:"effect"`
	StatusCode   *int            `json:"status_code,omitempty"`
	ExitCode     *int            `json:"exit_code,omitempty"`
	Result       json.RawMessage `json:"result,omitempty"`
	Body         string          `json:"body,omitempty"`
	Stdout       string          `json:"stdout,omitempty"`
	Stderr       string          `json:"stderr,omitempty"`
	Truncated    bool            `json:"truncated"`
	DurationMS   int64           `json:"duration_ms"`
	NextCursor   string          `json:"next_cursor,omitempty"`
	Error        *Error          `json:"error,omitempty"`
}

type GetIssueRequest struct {
	ConnectionID string `json:"connection_id,omitempty"`
	IssueID      string `json:"issue_id"`
	Detail       Detail `json:"detail,omitempty"`
}

type SearchRequest struct {
	ConnectionID string `json:"connection_id,omitempty"`
	Query        string `json:"query"`
	Limit        int    `json:"limit,omitempty"`
	Cursor       string `json:"cursor,omitempty"`
}

type RequestRequest struct {
	ConnectionID string              `json:"connection_id,omitempty"`
	Method       string              `json:"method"`
	RelativePath string              `json:"relative_path"`
	Query        map[string][]string `json:"query,omitempty"`
	Headers      map[string]string   `json:"headers,omitempty"`
	Body         string              `json:"body,omitempty"`
	OutputLimit  int                 `json:"output_limit,omitempty"`
}

type CLIRequest struct {
	ConnectionID string   `json:"connection_id,omitempty"`
	Executable   string   `json:"executable"`
	Arguments    []string `json:"arguments,omitempty"`
	Stdin        string   `json:"stdin,omitempty"`
	OutputLimit  int      `json:"output_limit,omitempty"`
}

type SearchResult struct {
	Items []Issue `json:"items"`
}

type Issue struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Status       string            `json:"status"`
	URL          string            `json:"url,omitempty"`
	Priority     string            `json:"priority,omitempty"`
	Severity     string            `json:"severity,omitempty"`
	Assignee     string            `json:"assignee,omitempty"`
	Labels       []string          `json:"labels,omitempty"`
	IssueType    string            `json:"issue_type,omitempty"`
	Reporter     string            `json:"reporter,omitempty"`
	CreatedAt    string            `json:"created_at,omitempty"`
	UpdatedAt    string            `json:"updated_at,omitempty"`
	Description  string            `json:"description,omitempty"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`
}

//go:embed testdata/v1/consumer-fixture.json
var fixtures embed.FS

// ConsumerFixture returns an isolated copy of the canonical v1 fixture.
func ConsumerFixture() ([]byte, error) {
	b, err := fixtures.ReadFile("testdata/v1/consumer-fixture.json")
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), b...), nil
}
