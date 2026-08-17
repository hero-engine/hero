package projection

import (
	"encoding/json"
	"strings"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/mail"
)

var (
	noInput = json.RawMessage(`{"type":"object","additionalProperties":false}`)
)

func action(operationID, id, label string, revision int64, schema json.RawMessage) attention.ActionDescriptor {
	descriptor, ok := attention.AnnotateActionDescriptor(attention.ActionDescriptor{
		ID: id, Label: label, RequiredRowRevision: revision,
		RequiresIdempotency: true, InputSchema: schema,
	}, operationID)
	if !ok {
		panic("unknown Attention operation policy: " + operationID)
	}
	return descriptor
}

func mailActions(revision int64) []attention.ActionDescriptor {
	return mail.RowCapabilities(revision)
}

func focusActions(revision int64, available bool) []attention.ActionDescriptor {
	out := []attention.ActionDescriptor{
		action(attention.OperationFocusMoveInbox, "move_inbox", "Move to Inbox", revision, noInput),
		action(attention.OperationFocusMoveLater, "move_later", "Move to Later", revision, noInput),
		action(attention.OperationFocusComplete, "complete", "Complete", revision, noInput),
	}
	if available {
		out = append([]attention.ActionDescriptor{action(attention.OperationFocusLaunch, "launch", "Launch", revision, noInput)}, out...)
	}
	return out
}

func validateInput(schema, input json.RawMessage) *attention.ContractError {
	if len(input) == 0 {
		input = json.RawMessage(`{}`)
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(input, &value); err != nil {
		return contractError(attention.ErrorValidation, "input must be a JSON object", "input")
	}
	text := string(schema)
	allowed := map[string]bool{}
	if strings.Contains(text, `"note"`) {
		allowed["note"] = true
	}
	if strings.Contains(text, `"artifact_type"`) {
		allowed["artifact_type"] = true
	}
	for key := range value {
		if !allowed[key] {
			return contractError(attention.ErrorValidation, "input field is not allowed", "input."+key)
		}
	}
	if raw, ok := value["note"]; ok {
		var note string
		if json.Unmarshal(raw, &note) != nil {
			return contractError(attention.ErrorValidation, "note must be a string", "input.note")
		}
	}
	if strings.Contains(text, `"artifact_type"`) && strings.Contains(text, `"required"`) {
		var artifact string
		if raw, ok := value["artifact_type"]; !ok || json.Unmarshal(raw, &artifact) != nil || strings.TrimSpace(artifact) == "" {
			return contractError(attention.ErrorValidation, "artifact_type is required", "input.artifact_type")
		}
	}
	return nil
}

func contractError(code, message, field string) *attention.ContractError {
	return &attention.ContractError{Code: code, Message: message, Field: field}
}
