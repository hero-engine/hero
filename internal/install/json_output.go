package install

// json_output.go — single-source-install P5 JSON output shapes.
//
// These types are the public ops API surface that a Hero-native client (and any
// future Hero consumer) subprocess-invokes and parses. Keep the field
// names stable; additive evolution only — new optional fields are
// fine, removing or renaming an existing field is a breaking change.

// InstallJSONOutput is what `hero install --json` emits on stdout
// once the install completes. On success Error is nil. On failure
// Result may still be partially populated (the operations that
// completed before the failure point).
type InstallJSONOutput struct {
	Target     string     `json:"target"`
	Mode       string     `json:"mode"`
	TargetDir  string     `json:"target_dir,omitempty"`
	Version    string     `json:"hero_version"`
	Result     *Result    `json:"result"`
	DurationMs int64      `json:"duration_ms"`
	Error      *JSONError `json:"error,omitempty"`
}

// JSONError is the structured error shape consumers parse. Code is a
// stable machine-readable string; Message is human-readable; Detail
// is optional supplemental info.
type JSONError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// NewJSONError wraps a Go error into the JSON shape. If err is nil,
// returns nil.
func NewJSONError(code string, err error) *JSONError {
	if err == nil {
		return nil
	}
	return &JSONError{
		Code:    code,
		Message: err.Error(),
	}
}
