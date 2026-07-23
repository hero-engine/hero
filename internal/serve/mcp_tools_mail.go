package serve

import (
	"encoding/json"
	"errors"

	"github.com/hero-engine/hero/internal/attention/mail"
	attentionstate "github.com/hero-engine/hero/internal/attention/state"
	"github.com/hero-engine/hero/internal/config"
)

type mailMCPMessage struct {
	mail.ListedMessage
	Actions []mailActionDescriptor `json:"actions"`
}

type mailActionDescriptor struct {
	ID               string `json:"id"`
	RequiresRevision bool   `json:"requires_revision"`
	Idempotent       bool   `json:"idempotent"`
}

type mailMCPError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Current interface{} `json:"current,omitempty"`
}

func (s *MCPServer) mailService() (*mail.Service, error) {
	root := s.attentionStateRoot
	var err error
	if root == "" {
		root, err = attentionstate.Ensure(attentionstate.Options{ProjectRoot: s.projectRoot})
	} else {
		root, err = attentionstate.Ensure(attentionstate.Options{Root: root})
	}
	if err != nil {
		return nil, err
	}
	store, err := mail.NewStore(root)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return nil, err
	}
	return mail.NewService(store, s.projectRoot, cfg), nil
}

func (s *MCPServer) toolMailList(args map[string]interface{}) (string, error) {
	service, err := s.mailService()
	if err != nil {
		return "", err
	}
	unread, _ := args["unread"].(bool)
	items, err := service.Inbox("", unread)
	if err != nil {
		return mailToolErrorJSON(err)
	}
	out := make([]mailMCPMessage, 0, len(items))
	for _, item := range items {
		out = append(out, mailMCPMessage{ListedMessage: item, Actions: mailActionDescriptors()})
	}
	return mailJSON(out)
}

func (s *MCPServer) toolMailShow(args map[string]interface{}) (string, error) {
	id, _ := args["message_id"].(string)
	if id == "" {
		return mailJSON(mailMCPError{Code: "validation", Message: "message_id is required"})
	}
	service, err := s.mailService()
	if err != nil {
		return "", err
	}
	item, err := service.Show(id, false)
	if err != nil {
		return mailToolErrorJSON(err)
	}
	return mailJSON(mailMCPMessage{ListedMessage: item, Actions: mailActionDescriptors()})
}

func (s *MCPServer) toolMailAction(args map[string]interface{}) (string, error) {
	id, _ := args["message_id"].(string)
	action, _ := args["action"].(string)
	key, _ := args["idempotency_key"].(string)
	note, _ := args["note"].(string)
	artifactType, _ := args["artifact_type"].(string)
	revisionNumber, ok := args["revision"].(float64)
	if id == "" || action == "" || key == "" || !ok || revisionNumber < 0 || revisionNumber != float64(int64(revisionNumber)) {
		return mailJSON(mailMCPError{Code: "validation", Message: "message_id, action, non-negative integer revision, and idempotency_key are required"})
	}
	service, err := s.mailService()
	if err != nil {
		return "", err
	}
	result, err := service.Action(mail.ActionRequest{MessageID: id, Action: action, ExpectedRevision: int64(revisionNumber), IdempotencyKey: key, Note: note, ArtifactType: artifactType})
	if err != nil {
		return mailToolErrorJSON(err)
	}
	return mailJSON(result)
}

func mailActionDescriptors() []mailActionDescriptor {
	return []mailActionDescriptor{
		{ID: mail.ActionRead, RequiresRevision: true, Idempotent: true},
		{ID: mail.ActionAcknowledge, RequiresRevision: true, Idempotent: true},
		{ID: mail.ActionDismiss, RequiresRevision: true, Idempotent: true},
		{ID: mail.ActionPromote, RequiresRevision: true, Idempotent: true},
		{ID: mail.ActionAddToToday, RequiresRevision: true, Idempotent: true},
	}
}

func mailJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

func mailToolErrorJSON(err error) (string, error) {
	failure := mailMCPError{Code: "validation", Message: err.Error()}
	switch {
	case errors.Is(err, mail.ErrNotFound):
		failure.Code = "missing"
	case errors.Is(err, mail.ErrStale):
		failure.Code = "stale_revision"
		var stale *mail.StaleError
		if errors.As(err, &stale) {
			failure.Current = stale.Current
		}
	case errors.Is(err, mail.ErrIdempotencyConflict):
		failure.Code = "idempotency_conflict"
	case errors.Is(err, mail.ErrUnsupportedAction):
		failure.Code = "unsupported_action"
	}
	return mailJSON(failure)
}
