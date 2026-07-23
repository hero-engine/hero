package serve

import (
	"encoding/json"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/attention/projection"
	attentionstate "github.com/hero-engine/hero/internal/attention/state"
	"github.com/hero-engine/hero/internal/attention/suggestion"
	"github.com/hero-engine/hero/internal/projectregistry"
)

func (s *MCPServer) toolAttentionSnapshot(map[string]interface{}) (string, error) {
	service, err := s.attentionProjectionService()
	if err != nil {
		return marshalSuggestion(attention.ActionResult{
			SchemaVersion: attention.SchemaVersion,
			Error:         &attention.ContractError{Code: attention.ErrorUnavailable, Message: err.Error()},
		})
	}
	result, err := service.Snapshot()
	if err != nil {
		if contractErr, ok := err.(*attention.ContractError); ok {
			return marshalSuggestion(attention.ActionResult{SchemaVersion: attention.SchemaVersion, Error: contractErr})
		}
		return "", err
	}
	return marshalSuggestion(result)
}

func (s *MCPServer) toolAttentionAction(args map[string]interface{}) (string, error) {
	service, err := s.attentionProjectionService()
	if err != nil {
		return marshalSuggestion(attention.ActionResult{
			SchemaVersion: attention.SchemaVersion,
			Error:         &attention.ContractError{Code: attention.ErrorUnavailable, Message: err.Error()},
		})
	}
	revision, err := int64Arg(args, "row_revision")
	if err != nil {
		return marshalSuggestion(attention.ActionResult{
			SchemaVersion: attention.SchemaVersion,
			Error:         &attention.ContractError{Code: attention.ErrorValidation, Message: err.Error(), Field: "row_revision"},
		})
	}
	var input json.RawMessage
	if value, ok := args["input"]; ok {
		input, err = json.Marshal(value)
		if err != nil {
			return "", err
		}
	}
	return marshalSuggestion(service.Dispatch(attention.ActionRequest{
		SchemaVersion: attention.SchemaVersion,
		RowID:         stringArg(args, "row_id"), ActionID: stringArg(args, "action_id"),
		RowRevision: revision, IdempotencyKey: stringArg(args, "idempotency_key"), Input: input,
	}))
}

func (s *MCPServer) attentionProjectionService() (*projection.Service, error) {
	if s.attentionService != nil {
		return s.attentionService()
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
	mailSource, err := projection.NewRegistryMailSource(root, registry)
	if err != nil {
		return nil, err
	}
	resolver := s.attentionResolver
	if resolver == nil {
		resolver = focus.NewRegistryResolver(registry)
	}
	focusStore, err := focus.NewStore(root)
	if err != nil {
		return nil, err
	}
	focusService := focus.NewService(focusStore, resolver)
	suggestionStore, err := suggestion.NewStore(root)
	if err != nil {
		return nil, err
	}
	return projection.NewService(mailSource, focusService, suggestion.NewService(suggestionStore, focusService, resolver)), nil
}
