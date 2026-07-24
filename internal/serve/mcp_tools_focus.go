package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
	attentionstate "github.com/hero-engine/hero/internal/attention/state"
	"github.com/hero-engine/hero/internal/attention/suggestion"
)

func (s *MCPServer) toolFocusSuggest(args map[string]interface{}) (string, error) {
	service, resolver, err := s.suggestionService()
	if err != nil {
		return "", err
	}
	project, _ := args["project"].(string)
	var ref *attention.ProjectReference
	if project == "" {
		ref, err = resolver.ResolveCurrent()
	} else {
		ref, err = resolver.ResolveInput(project)
	}
	if err != nil {
		return marshalSuggestionError(err)
	}
	result, err := service.Create(suggestion.CreateRequest{
		Title: stringArg(args, "title"), Reason: stringArg(args, "reason"), Prompt: stringArg(args, "prompt"), Project: ref,
		Provenance:     &attention.ProvenanceReference{Kind: stringArg(args, "source_kind"), SourceID: stringArg(args, "source_id")},
		IdempotencyKey: stringArg(args, "idempotency_key"),
	})
	if err != nil {
		return marshalSuggestionError(err)
	}
	return marshalSuggestion(result)
}

func (s *MCPServer) toolFocusCreate(args map[string]interface{}) (string, error) {
	version, versionErr := directActionVersion(args)
	if versionErr != nil {
		return directActionFailure(versionErr)
	}
	request := attention.FocusCreateActionRequest{
		SchemaVersion: version, IntentSource: stringArg(args, "intent_source"),
		Title: stringArg(args, "title"), Prompt: stringArg(args, "prompt"), Lifecycle: stringArg(args, "lifecycle"),
		Project: stringArg(args, "project"), ProjectPeerID: stringArg(args, "project_peer_id"),
		SourceID: stringArg(args, "source_id"), IdempotencyKey: stringArg(args, "idempotency_key"),
	}
	if contractErr := attention.ValidateFocusCreateActionRequest(request); contractErr != nil {
		return directActionFailure(contractErr)
	}
	service, resolver, err := s.directFocusService()
	if err != nil {
		return directActionUnavailable(err)
	}
	var project *attention.ProjectReference
	if request.Project != "" {
		project, err = resolver.ResolveInput(request.Project)
		if err != nil || project == nil {
			if err == nil {
				err = focus.ErrProjectMissing
			}
			code := attention.ErrorUnavailable
			if errors.Is(err, focus.ErrProjectMissing) {
				code = attention.ErrorMissing
			}
			return directActionFailure(&attention.ContractError{Code: code, Message: err.Error(), Field: "project"})
		}
		if project.PeerID != request.ProjectPeerID {
			return directActionFailure(&attention.ContractError{
				Code: attention.ErrorValidation, Message: "project peer_id no longer matches the resolved target", Field: "project_peer_id",
			})
		}
	}
	item, _, err := service.CreateOrGet(focus.CreateRequest{
		Title: request.Title, Prompt: request.Prompt, Lifecycle: request.Lifecycle, Project: project,
		Origin:    &attention.ProvenanceReference{Kind: attention.IntentSourceUser, SourceID: request.SourceID},
		OriginKey: directActionOriginKey(attention.OperationFocusCreate, request.IdempotencyKey),
	})
	if err != nil {
		if errors.Is(err, focus.ErrIdempotencyConflict) {
			return directActionFailure(&attention.ContractError{
				Code: attention.ErrorIdempotencyConflict, Message: err.Error(), Field: "idempotency_key",
			})
		}
		return directActionUnavailable(err)
	}
	return directActionSuccess(item)
}

func (s *MCPServer) toolFocusSuggestions(args map[string]interface{}) (string, error) {
	service, _, err := s.suggestionService()
	if err != nil {
		return "", err
	}
	pending, _ := args["pending"].(bool)
	result, err := service.List(pending)
	if err != nil {
		return marshalSuggestionError(err)
	}
	return marshalSuggestion(result)
}

func (s *MCPServer) toolFocusSuggestionAction(args map[string]interface{}) (string, error) {
	service, _, err := s.suggestionService()
	if err != nil {
		return "", err
	}
	revision, err := int64Arg(args, "revision")
	if err != nil {
		return marshalSuggestionError(&suggestion.Error{Code: attention.ErrorValidation, Message: err.Error(), Field: "revision"})
	}
	result, err := service.Act(stringArg(args, "suggestion_id"), stringArg(args, "action"), revision, stringArg(args, "idempotency_key"))
	if err != nil {
		return marshalSuggestionError(err)
	}
	return marshalSuggestion(result)
}

func (s *MCPServer) suggestionService() (*suggestion.Service, focus.ProjectResolver, error) {
	root := s.attentionStateRoot
	var err error
	if root == "" {
		root, err = attentionstate.Ensure(attentionstate.Options{ProjectRoot: s.projectRoot})
	} else {
		root, err = attentionstate.Ensure(attentionstate.Options{Root: root})
	}
	if err != nil {
		return nil, nil, err
	}
	resolver := s.attentionResolver
	if resolver == nil {
		resolver, err = focus.LoadRegistryResolver(s.projectRoot)
		if err != nil {
			return nil, nil, err
		}
	}
	proposalStore, err := suggestion.NewStore(root)
	if err != nil {
		return nil, nil, err
	}
	focusStore, err := focus.NewStore(root)
	if err != nil {
		return nil, nil, err
	}
	return suggestion.NewService(proposalStore, focus.NewService(focusStore, resolver), resolver), resolver, nil
}

func (s *MCPServer) directFocusService() (*focus.Service, focus.ProjectResolver, error) {
	root := s.attentionStateRoot
	var err error
	if root == "" {
		root, err = attentionstate.Ensure(attentionstate.Options{ProjectRoot: s.projectRoot})
	} else {
		root, err = attentionstate.Ensure(attentionstate.Options{Root: root})
	}
	if err != nil {
		return nil, nil, err
	}
	resolver := s.attentionResolver
	if resolver == nil {
		resolver, err = focus.LoadRegistryResolver(s.projectRoot)
		if err != nil {
			return nil, nil, err
		}
	}
	store, err := focus.NewStore(root)
	if err != nil {
		return nil, nil, err
	}
	return focus.NewService(store, resolver), resolver, nil
}

func stringArg(args map[string]interface{}, key string) string {
	value, _ := args[key].(string)
	return value
}

func int64Arg(args map[string]interface{}, key string) (int64, error) {
	switch value := args[key].(type) {
	case float64:
		if value != float64(int64(value)) {
			return 0, fmt.Errorf("%s must be an integer", key)
		}
		return int64(value), nil
	case int64:
		return value, nil
	case int:
		return int64(value), nil
	case string:
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s must be a decimal int64", key)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("%s is required", key)
	}
}

func marshalSuggestion(value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func marshalSuggestionError(err error) (string, error) {
	var operationError *suggestion.Error
	if errors.As(err, &operationError) {
		return marshalSuggestion(map[string]any{"error": operationError})
	}
	return "", err
}
