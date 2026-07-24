package attention

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

const IntentSourceUser = "user"

type MailSendActionRequest struct {
	SchemaVersion   int    `json:"schema_version"`
	IntentSource    string `json:"intent_source"`
	Recipient       string `json:"recipient"`
	RecipientPeerID string `json:"recipient_peer_id"`
	Subject         string `json:"subject"`
	Body            string `json:"body"`
	Kind            string `json:"kind,omitempty"`
	SourceKind      string `json:"source_kind,omitempty"`
	SourceID        string `json:"source_id,omitempty"`
	IdempotencyKey  string `json:"idempotency_key"`
}

type MailReplyActionRequest struct {
	SchemaVersion  int    `json:"schema_version"`
	IntentSource   string `json:"intent_source"`
	MessageID      string `json:"message_id"`
	ThreadID       string `json:"thread_id"`
	Subject        string `json:"subject,omitempty"`
	Body           string `json:"body"`
	Kind           string `json:"kind,omitempty"`
	SourceKind     string `json:"source_kind,omitempty"`
	SourceID       string `json:"source_id,omitempty"`
	IdempotencyKey string `json:"idempotency_key"`
}

type FocusCreateActionRequest struct {
	SchemaVersion  int    `json:"schema_version"`
	IntentSource   string `json:"intent_source"`
	Title          string `json:"title"`
	Prompt         string `json:"prompt"`
	Lifecycle      string `json:"lifecycle"`
	Project        string `json:"project,omitempty"`
	ProjectPeerID  string `json:"project_peer_id,omitempty"`
	SourceID       string `json:"source_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

type DirectActionCase struct {
	ID                 string          `json:"id"`
	OperationID        string          `json:"operation_id"`
	Request            json.RawMessage `json:"request"`
	ExpectedSourceKind string          `json:"expected_source_kind"`
}

type DirectActionFixture struct {
	SchemaVersion int                `json:"schema_version"`
	Cases         []DirectActionCase `json:"cases"`
}

func ValidateMailSendActionRequest(v MailSendActionRequest) *ContractError {
	if err := validateDirectActionCommon(v.SchemaVersion, v.IntentSource, v.IdempotencyKey); err != nil {
		return err
	}
	if err := validateID("recipient", v.Recipient); err != nil {
		return err
	}
	if err := validateID("recipient_peer_id", v.RecipientPeerID); err != nil {
		return err
	}
	if strings.TrimSpace(v.Subject) == "" || !utf8.ValidString(v.Subject) || utf8.RuneCountInString(v.Subject) > MaxSubjectCharacters {
		return invalid("subject", "must be non-empty valid UTF-8 and at most 200 characters")
	}
	if !utf8.ValidString(v.Body) || len(v.Body) > MaxBodyBytes {
		return invalid("body", "must be valid UTF-8 and at most 65536 bytes")
	}
	if v.Kind != "" && (!utf8.ValidString(v.Kind) || len(v.Kind) > 64) {
		return invalid("kind", "must be valid UTF-8 and at most 64 bytes")
	}
	return validateSourcePair(v.SourceKind, v.SourceID)
}

func ValidateMailReplyActionRequest(v MailReplyActionRequest) *ContractError {
	if err := validateDirectActionCommon(v.SchemaVersion, v.IntentSource, v.IdempotencyKey); err != nil {
		return err
	}
	if err := validateID("message_id", v.MessageID); err != nil {
		return err
	}
	if err := validateID("thread_id", v.ThreadID); err != nil {
		return err
	}
	if !utf8.ValidString(v.Subject) || utf8.RuneCountInString(v.Subject) > MaxSubjectCharacters {
		return invalid("subject", "must be valid UTF-8 and at most 200 characters")
	}
	if strings.TrimSpace(v.Body) == "" || !utf8.ValidString(v.Body) || len(v.Body) > MaxBodyBytes {
		return invalid("body", "must be non-empty valid UTF-8 and at most 65536 bytes")
	}
	if v.Kind != "" && (!utf8.ValidString(v.Kind) || len(v.Kind) > 64) {
		return invalid("kind", "must be valid UTF-8 and at most 64 bytes")
	}
	return validateSourcePair(v.SourceKind, v.SourceID)
}

func ValidateFocusCreateActionRequest(v FocusCreateActionRequest) *ContractError {
	if err := validateDirectActionCommon(v.SchemaVersion, v.IntentSource, v.IdempotencyKey); err != nil {
		return err
	}
	if strings.TrimSpace(v.Title) == "" || !utf8.ValidString(v.Title) || utf8.RuneCountInString(v.Title) > MaxSubjectCharacters {
		return invalid("title", "must be non-empty valid UTF-8 and at most 200 characters")
	}
	if strings.TrimSpace(v.Prompt) == "" || !utf8.ValidString(v.Prompt) || len(v.Prompt) > MaxFocusPromptBytes {
		return invalid("prompt", "must be non-empty valid UTF-8 and at most 65536 bytes")
	}
	if !validLifecycle(v.Lifecycle) {
		return invalid("lifecycle", "must be inbox, today, later, or done")
	}
	if (strings.TrimSpace(v.Project) == "") != (strings.TrimSpace(v.ProjectPeerID) == "") {
		return invalid("project", "project and project_peer_id must be supplied together")
	}
	if err := validateID("source_id", v.SourceID); err != nil {
		return err
	}
	return nil
}

func ValidateDirectActionFixture(v DirectActionFixture) *ContractError {
	if err := validateVersion(v.SchemaVersion); err != nil {
		return err
	}
	seen := make(map[string]bool, len(v.Cases))
	for i, actionCase := range v.Cases {
		field := fmt.Sprintf("cases[%d]", i)
		if strings.TrimSpace(actionCase.ID) == "" || seen[actionCase.ID] {
			return invalid(field+".id", "is required and must be unique")
		}
		seen[actionCase.ID] = true
		if strings.TrimSpace(actionCase.ExpectedSourceKind) == "" {
			return invalid(field+".expected_source_kind", "is required")
		}
		var err *ContractError
		switch actionCase.OperationID {
		case OperationMailSend:
			var request MailSendActionRequest
			if decodeErr := json.Unmarshal(actionCase.Request, &request); decodeErr != nil {
				return invalid(field+".request", "must decode as a Mail send request")
			}
			err = ValidateMailSendActionRequest(request)
		case OperationMailReply:
			var request MailReplyActionRequest
			if decodeErr := json.Unmarshal(actionCase.Request, &request); decodeErr != nil {
				return invalid(field+".request", "must decode as a Mail reply request")
			}
			err = ValidateMailReplyActionRequest(request)
		case OperationFocusCreate:
			var request FocusCreateActionRequest
			if decodeErr := json.Unmarshal(actionCase.Request, &request); decodeErr != nil {
				return invalid(field+".request", "must decode as a Focus create request")
			}
			err = ValidateFocusCreateActionRequest(request)
		default:
			return invalid(field+".operation_id", "must name a direct Attention action")
		}
		if err != nil {
			err.Field = field + ".request." + err.Field
			return err
		}
	}
	return nil
}

func validateDirectActionCommon(version int, source, key string) *ContractError {
	if err := validateVersion(version); err != nil {
		return err
	}
	if source != IntentSourceUser {
		return &ContractError{Code: ErrorPermission, Message: "direct Attention actions require an explicit user request", Field: "intent_source"}
	}
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || len(key) > 512 {
		return invalid("idempotency_key", "must be non-empty valid UTF-8 and at most 512 bytes")
	}
	return nil
}

func validateSourcePair(kind, id string) *ContractError {
	if (strings.TrimSpace(kind) == "") != (strings.TrimSpace(id) == "") {
		return invalid("source_kind", "source_kind and source_id must be supplied together")
	}
	if !utf8.ValidString(kind) || len(kind) > 64 {
		return invalid("source_kind", "must be valid UTF-8 and at most 64 bytes")
	}
	if !utf8.ValidString(id) || len(id) > 512 {
		return invalid("source_id", "must be valid UTF-8 and at most 512 bytes")
	}
	return nil
}
