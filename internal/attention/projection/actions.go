package projection

import (
	"encoding/json"
	"strings"

	"github.com/hero-engine/hero/contracts/attention"
)

var (
	noInput      = json.RawMessage(`{"type":"object","additionalProperties":false}`)
	noteInput    = json.RawMessage(`{"type":"object","properties":{"note":{"type":"string"}},"additionalProperties":false}`)
	promoteInput = json.RawMessage(`{"type":"object","properties":{"artifact_type":{"type":"string"}},"required":["artifact_type"],"additionalProperties":false}`)
)

func action(id, label string, revision int64, schema json.RawMessage) attention.ActionDescriptor {
	return attention.ActionDescriptor{
		ID: id, Label: label, RequiredRowRevision: revision,
		RequiresIdempotency: true, InputSchema: schema,
	}
}

func mailActions(revision int64) []attention.ActionDescriptor {
	return []attention.ActionDescriptor{
		action("mark_read", "Mark Read", revision, noInput),
		action("acknowledge", "Acknowledge", revision, noteInput),
		action("dismiss", "Dismiss", revision, noInput),
		action("promote", "Promote", revision, promoteInput),
		action("add_to_today", "Add to Today", revision, noInput),
	}
}

func focusActions(revision int64, available bool) []attention.ActionDescriptor {
	out := []attention.ActionDescriptor{
		action("move_inbox", "Move to Inbox", revision, noInput),
		action("move_later", "Move to Later", revision, noInput),
		action("complete", "Complete", revision, noInput),
	}
	if available {
		out = append([]attention.ActionDescriptor{action("launch", "Launch", revision, noInput)}, out...)
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
