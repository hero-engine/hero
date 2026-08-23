package serve

import (
	"encoding/json"
	"errors"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailthread"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/attention/mailquery"
	attentionstate "github.com/hero-engine/hero/internal/attention/state"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/projectregistry"
)

func (s *MCPServer) toolMailSend(args map[string]interface{}) (string, error) {
	version, versionErr := directActionVersion(args)
	if versionErr != nil {
		return directActionFailure(versionErr)
	}
	request := attention.MailSendActionRequest{
		SchemaVersion: version, IntentSource: stringArg(args, "intent_source"),
		Recipient: stringArg(args, "recipient"), RecipientPeerID: stringArg(args, "recipient_peer_id"),
		Subject: stringArg(args, "subject"), Body: stringArg(args, "body"), Kind: stringArg(args, "kind"),
		SourceKind: stringArg(args, "source_kind"), SourceID: stringArg(args, "source_id"),
		IdempotencyKey: stringArg(args, "idempotency_key"),
	}
	if contractErr := attention.ValidateMailSendActionRequest(request); contractErr != nil {
		return directActionFailure(contractErr)
	}
	service, err := s.mailService()
	if err != nil {
		return directActionUnavailable(err)
	}
	result, err := service.Send(mail.SendRequest{
		RecipientAlias: request.Recipient, ExpectedRecipientPeer: request.RecipientPeerID,
		Subject: request.Subject, Body: request.Body, Kind: request.Kind, IdempotencyKey: request.IdempotencyKey,
		Provenance: directActionProvenance(request.SourceKind, request.SourceID),
	})
	if err != nil {
		return directActionFailure(directMailError(err, false))
	}
	return directActionSuccess(result)
}

func (s *MCPServer) toolMailReply(args map[string]interface{}) (string, error) {
	version, versionErr := directActionVersion(args)
	if versionErr != nil {
		return directActionFailure(versionErr)
	}
	request := attention.MailReplyActionRequest{
		SchemaVersion: version, IntentSource: stringArg(args, "intent_source"),
		MessageID: stringArg(args, "message_id"), ThreadID: stringArg(args, "thread_id"),
		Subject: stringArg(args, "subject"), Body: stringArg(args, "body"), Kind: stringArg(args, "kind"),
		SourceKind: stringArg(args, "source_kind"), SourceID: stringArg(args, "source_id"),
		IdempotencyKey: stringArg(args, "idempotency_key"),
	}
	if contractErr := attention.ValidateMailReplyActionRequest(request); contractErr != nil {
		return directActionFailure(contractErr)
	}
	service, err := s.mailService()
	if err != nil {
		return directActionUnavailable(err)
	}
	result, err := service.Reply(mail.ReplyRequest{
		MessageID: request.MessageID, ExpectedThread: request.ThreadID,
		Subject: request.Subject, Body: request.Body, Kind: request.Kind, IdempotencyKey: request.IdempotencyKey,
		Provenance: directActionProvenance(request.SourceKind, request.SourceID),
	})
	if err != nil {
		return directActionFailure(directMailError(err, true))
	}
	return directActionSuccess(result)
}

func directMailError(err error, reply bool) *attention.ContractError {
	var contractErr *attention.ContractError
	switch {
	case errors.As(err, &contractErr):
		return contractErr
	case errors.Is(err, mail.ErrIdempotencyConflict):
		return &attention.ContractError{Code: attention.ErrorIdempotencyConflict, Message: err.Error(), Field: "idempotency_key"}
	case errors.Is(err, mail.ErrRecipientMismatch):
		return &attention.ContractError{Code: attention.ErrorValidation, Message: err.Error(), Field: "recipient_peer_id"}
	case errors.Is(err, mail.ErrThreadMismatch):
		return &attention.ContractError{Code: attention.ErrorStale, Message: err.Error(), Field: "thread_id"}
	case errors.Is(err, mail.ErrNotFound):
		return &attention.ContractError{Code: attention.ErrorMissing, Message: err.Error(), Field: "message_id"}
	case errors.Is(err, mail.ErrRecipientMissing):
		field := "recipient"
		if reply {
			field = "message_id"
		}
		return &attention.ContractError{Code: attention.ErrorMissing, Message: err.Error(), Field: field}
	case errors.Is(err, mail.ErrUnavailable):
		return &attention.ContractError{Code: attention.ErrorUnavailable, Message: err.Error()}
	}
	return &attention.ContractError{Code: attention.ErrorUnavailable, Message: err.Error()}
}

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

func (s *MCPServer) toolMailThreadList(args map[string]interface{}) (string, error) {
	request := mailthread.ThreadListRequest{
		SchemaVersion: mailthread.SchemaVersion,
		ProjectPeerID: stringArg(args, "project_peer_id"),
		Bucket:        mailthread.Bucket(stringArg(args, "bucket")),
		Lifecycle:     mailthread.Lifecycle(stringArg(args, "lifecycle")),
		Cursor:        stringArg(args, "cursor"),
	}
	if _, ok := args["limit"]; ok {
		limit, err := int64Arg(args, "limit")
		if err != nil || int64(int(limit)) != limit {
			message := "limit must be an integer"
			if err == nil {
				message = "limit is outside the supported integer range"
			}
			return mailJSON(mailthread.ThreadListResponse{SchemaVersion: mailthread.SchemaVersion, Error: mailValidation(message, "limit")})
		}
		request.Limit = int(limit)
	}
	service, err := s.mailThreadQueryService()
	if err != nil {
		return mailJSON(mailthread.ThreadListResponse{SchemaVersion: mailthread.SchemaVersion, Error: mailUnavailable(err.Error())})
	}
	return mailJSON(service.Threads(request))
}

func (s *MCPServer) toolMailThreadShow(args map[string]interface{}) (string, error) {
	service, err := s.mailThreadQueryService()
	if err != nil {
		return mailJSON(mailthread.ThreadDetailResponse{SchemaVersion: mailthread.SchemaVersion, Error: mailUnavailable(err.Error())})
	}
	return mailJSON(service.ThreadDetail(stringArg(args, "project_peer_id"), stringArg(args, "thread_id")))
}

func (s *MCPServer) toolMailThreadAction(args map[string]interface{}) (string, error) {
	version, err := int64Arg(args, "schema_version")
	if err != nil || int64(int(version)) != version {
		return mailJSON(mailthread.ActionResponse{SchemaVersion: mailthread.SchemaVersion, Error: mailValidation("schema_version must be an integer", "schema_version")})
	}
	revision, err := int64Arg(args, "thread_revision")
	if err != nil {
		return mailJSON(mailthread.ActionResponse{SchemaVersion: mailthread.SchemaVersion, Error: mailValidation(err.Error(), "thread_revision")})
	}
	identityArgs, _ := args["identity"].(map[string]interface{})
	request := mailthread.ActionRequest{
		SchemaVersion: int(version),
		Identity: mailthread.Identity{
			ProjectPeerID: stringArg(identityArgs, "project_peer_id"),
			ThreadID:      stringArg(identityArgs, "thread_id"),
		},
		ActionID:       stringArg(args, "action_id"),
		ThreadRevision: revision,
		IdempotencyKey: stringArg(args, "idempotency_key"),
	}
	if input, ok := args["input"]; ok {
		request.Input, err = json.Marshal(input)
		if err != nil {
			return mailJSON(mailthread.ActionResponse{SchemaVersion: mailthread.SchemaVersion, Error: mailValidation("input must be valid JSON", "input")})
		}
	}
	service, err := s.mailThreadQueryService()
	if err != nil {
		return mailJSON(mailthread.ActionResponse{SchemaVersion: mailthread.SchemaVersion, Error: mailUnavailable(err.Error())})
	}
	return mailJSON(service.ThreadAction(request))
}

func (s *MCPServer) toolMailThreadContract(map[string]interface{}) (string, error) {
	return mailJSON(mailthread.ContractResponse{
		SchemaVersion: mailthread.SchemaVersion, BundleVersion: mailthread.BundleVersion,
		BundleManifestSHA256: mailthread.ConformanceManifestSHA256, Compatibility: mailthread.Compatibility,
	})
}

func (s *MCPServer) mailThreadQueryService() (*mailquery.Service, error) {
	if s.mailQueryService != nil {
		return s.mailQueryService()
	}
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
	registry, err := projectregistry.Load()
	if err != nil {
		return nil, err
	}
	return mailquery.NewService(root, registry)
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
