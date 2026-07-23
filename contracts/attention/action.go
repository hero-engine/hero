package attention

import "encoding/json"

const (
	ErrorValidation          = "validation"
	ErrorStale               = "stale"
	ErrorUnsupported         = "unsupported"
	ErrorMissing             = "missing"
	ErrorIncompatibleVersion = "incompatible_version"
	ErrorUnavailable         = "unavailable"
)

type ActionDescriptor struct {
	ID                  string          `json:"id"`
	Label               string          `json:"label"`
	Style               string          `json:"style,omitempty"`
	Confirmation        string          `json:"confirmation,omitempty"`
	OperationID         string          `json:"operation_id,omitempty"`
	Effect              string          `json:"effect,omitempty"`
	Consent             string          `json:"consent,omitempty"`
	InputSchema         json.RawMessage `json:"input_schema,omitempty"`
	RequiredRowRevision int64           `json:"required_row_revision"`
	RequiresIdempotency bool            `json:"requires_idempotency"`
}

type ActionRequest struct {
	SchemaVersion  int             `json:"schema_version"`
	RowID          string          `json:"row_id"`
	ActionID       string          `json:"action_id"`
	RowRevision    int64           `json:"row_revision"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	Input          json.RawMessage `json:"input,omitempty"`
}

type NavigationReference struct {
	Project ProjectReference `json:"project"`
	Target  string           `json:"target"`
}

type LaunchIntent struct {
	Project ProjectReference `json:"project"`
	Prompt  string           `json:"prompt"`
}

type ActionResult struct {
	SchemaVersion    int                  `json:"schema_version"`
	Row              *AttentionRow        `json:"row,omitempty"`
	RemovedRowID     string               `json:"removed_row_id,omitempty"`
	SnapshotRevision string               `json:"snapshot_revision,omitempty"`
	Source           json.RawMessage      `json:"source,omitempty"`
	Navigation       *NavigationReference `json:"navigation,omitempty"`
	Launch           *LaunchIntent        `json:"launch,omitempty"`
	Error            *ContractError       `json:"error,omitempty"`
}

type ContractError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Field   string            `json:"field,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

func (e *ContractError) Error() string { return e.Message }
