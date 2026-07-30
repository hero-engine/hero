package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/codehost"
)

const (
	codeHostMCPToolPrefix = "hero_code_host_"
	maxCodeHostMCPInput   = 1 << 20
)

var errCodeHostMCPInputTooLarge = errors.New("code-host MCP input exceeds bound")

type codeHostMCPBroker interface {
	Execute(context.Context, codehostbroker.Request) codehostbroker.Response
	Prepare(context.Context, codehostbroker.Request) codehostbroker.PreparationResponse
}

var newCodeHostMCPBroker = func(projectRoot string) codeHostMCPBroker {
	return codehost.NewBroker(projectRoot)
}

// CodeHostToolDefinitions returns the complete operation-specific MCP
// inventory in the authoritative contract registry order.
func CodeHostToolDefinitions() []ToolDefinition {
	operations := codehostbroker.Operations()
	definitions := make([]ToolDefinition, 0, len(operations))
	for _, operation := range operations {
		policy, _ := codehostbroker.Policy(operation)
		readOnly := policy.Effect == codehostbroker.EffectRead
		destructive := policy.Effect == codehostbroker.EffectCommitment
		idempotent := policy.ReplaySafe
		openWorld := true
		definitions = append(definitions, ToolDefinition{
			Name:        codeHostMCPToolName(operation),
			Description: codeHostMCPDescription(operation, policy),
			InputSchema: codeHostMCPInputSchema(operation, policy),
			Annotations: &ToolAnnotations{
				Title:           codehostbroker.Version + "." + string(operation),
				ReadOnlyHint:    &readOnly,
				DestructiveHint: &destructive,
				IdempotentHint:  &idempotent,
				OpenWorldHint:   &openWorld,
			},
			Meta: map[string]interface{}{
				"hero.dev/contract":     codehostbroker.Version,
				"hero.dev/operation_id": string(operation),
				"hero.dev/effect":       string(policy.Effect),
				"hero.dev/consent":      string(policy.Consent),
			},
		})
	}
	return definitions
}

func codeHostMCPToolName(operation codehostbroker.Operation) string {
	return codeHostMCPToolPrefix + string(operation)
}

func codeHostOperationForMCPTool(name string) (codehostbroker.Operation, bool) {
	if !strings.HasPrefix(name, codeHostMCPToolPrefix) {
		return "", false
	}
	operation := codehostbroker.Operation(strings.TrimPrefix(name, codeHostMCPToolPrefix))
	_, ok := codehostbroker.Policy(operation)
	return operation, ok
}

func codeHostMCPOperations() []codehostbroker.Operation {
	return codehostbroker.Operations()
}

func codeHostMCPDescription(operation codehostbroker.Operation, policy codehostbroker.OperationPolicy) string {
	if codehostbroker.IsRead(operation) {
		return fmt.Sprintf("Run the bounded %s %s operation through Hero's credential-safe code-host broker.", codehostbroker.Version, operation)
	}
	return fmt.Sprintf(
		"Prepare or execute the bounded %s %s operation through Hero's credential-safe code-host broker. Set prepare=true to read current revisions without writing; omit it to execute the exact prepared request. Effect: %s. Consent: %s.",
		codehostbroker.Version, operation, policy.Effect, policy.Consent,
	)
}

func codeHostMCPInputSchema(operation codehostbroker.Operation, policy codehostbroker.OperationPolicy) InputSchema {
	properties := map[string]PropSchema{
		"version": {
			Type:        "string",
			Description: "Contract version; must be code-host-broker/v1",
			Enum:        []string{codehostbroker.Version},
		},
		"provider": {
			Type:        "string",
			Description: "Expected provider identity; verified against the selected connection",
		},
		"connection_id": {
			Type:        "string",
			Description: "Stable code-host connection ID",
		},
		"repository": {
			Type:                 "object",
			Description:          "Repository-qualified identity",
			Properties:           codeHostMCPRepositoryProperties(),
			Required:             []string{"host", "owner", "name", "full_name"},
			AdditionalProperties: boolPointer(false),
		},
	}
	required := []string{"version", "provider", "connection_id", "repository"}

	switch operation {
	case codehostbroker.OperationCapabilities:
		properties["repositories"] = PropSchema{
			Type: "array", Description: "Additional repository-qualified scopes",
			Items: codeHostMCPRepositoryProperty(),
		}
	case codehostbroker.OperationGetAuthenticatedActor:
	case codehostbroker.OperationListPullRequests, codehostbroker.OperationSearchPullRequests:
		properties["repositories"] = PropSchema{
			Type: "array", Description: "Additional repository-qualified scopes",
			Items: codeHostMCPRepositoryProperty(),
		}
		addCodeHostMCPCollectionProperties(properties)
	case codehostbroker.OperationGetCommits, codehostbroker.OperationGetReviews, codehostbroker.OperationGetComments:
		properties["pull_request"] = codeHostMCPPullRequestProperty()
		properties["limit"] = PropSchema{Type: "integer", Description: fmt.Sprintf("Page size from 1 through %d", policy.Bounds.PageSize)}
		properties["cursor"] = PropSchema{Type: "string", Description: "Opaque cursor from an identical prior request"}
		required = append(required, "pull_request")
	default:
		if operation != codehostbroker.OperationCreatePullRequest {
			properties["pull_request"] = codeHostMCPPullRequestProperty()
			required = append(required, "pull_request")
		}
	}

	if codehostbroker.IsMutation(operation) {
		properties["prepare"] = PropSchema{
			Type:        "boolean",
			Description: "When true, perform non-mutating preflight and return only current capability and observation revisions",
		}
		properties["intent_source"] = PropSchema{Type: "string", Description: "Semantic authorization source; must be user", Enum: []string{"user"}}
		properties["consent"] = PropSchema{Type: "string", Description: "Required consent: " + string(policy.Consent), Enum: []string{string(policy.Consent)}}
		properties["idempotency_key"] = PropSchema{Type: "string", Description: "Stable retry key"}
		properties["capability_revision"] = PropSchema{Type: "string", Description: "Prepared capability revision; required when prepare is false"}
		properties["observation_revision"] = PropSchema{Type: "string", Description: "Prepared observation revision; required when prepare is false"}
		properties["reconciliation_key"] = PropSchema{Type: "string", Description: "Stable reconciliation key"}
		properties["payload"] = codeHostMCPPayloadProperty(operation)
		required = append(required,
			"intent_source", "consent", "idempotency_key", "reconciliation_key", "payload",
		)
	}
	sort.Strings(required)
	return InputSchema{
		Type: "object", Properties: properties, Required: required,
		AdditionalProperties: boolPointer(false),
	}
}

func addCodeHostMCPCollectionProperties(properties map[string]PropSchema) {
	properties["query"] = PropSchema{Type: "string", Description: "Bounded provider-neutral query"}
	properties["order"] = PropSchema{Type: "string", Description: "Stable result order"}
	properties["limit"] = PropSchema{Type: "integer", Description: fmt.Sprintf("Page size from 1 through %d", codehostbroker.MaxPageSize)}
	properties["cursor"] = PropSchema{Type: "string", Description: "Opaque cursor from an identical prior request"}
}

func codeHostMCPPullRequestProperty() PropSchema {
	return PropSchema{
		Type:        "object",
		Description: "Repository-qualified pull-request identity",
		Properties: map[string]PropSchema{
			"connection_id": {Type: "string"},
			"repository":    *codeHostMCPRepositoryProperty(),
			"provider_id":   {Type: "string"},
			"number":        {Type: "integer"},
		},
		Required:             []string{"connection_id", "repository", "provider_id", "number"},
		AdditionalProperties: boolPointer(false),
	}
}

func codeHostMCPRepositoryProperty() *PropSchema {
	return &PropSchema{
		Type:                 "object",
		Description:          "Repository-qualified identity",
		Properties:           codeHostMCPRepositoryProperties(),
		Required:             []string{"host", "owner", "name", "full_name"},
		AdditionalProperties: boolPointer(false),
	}
}

func codeHostMCPRepositoryProperties() map[string]PropSchema {
	return map[string]PropSchema{
		"host":        {Type: "string"},
		"provider_id": {Type: "string"},
		"owner":       {Type: "string"},
		"name":        {Type: "string"},
		"full_name":   {Type: "string"},
	}
}

func codeHostMCPRefProperty() PropSchema {
	return PropSchema{
		Type: "object",
		Properties: map[string]PropSchema{
			"repository": *codeHostMCPRepositoryProperty(),
			"name":       {Type: "string"},
			"sha":        {Type: "string"},
		},
		Required:             []string{"repository", "name", "sha"},
		AdditionalProperties: boolPointer(false),
	}
}

func codeHostMCPPayloadProperty(operation codehostbroker.Operation) PropSchema {
	property := PropSchema{
		Type:                 "object",
		Description:          codeHostMCPPayloadDescription(operation),
		Properties:           map[string]PropSchema{},
		AdditionalProperties: boolPointer(false),
	}
	switch operation {
	case codehostbroker.OperationCreatePullRequest:
		property.Properties = map[string]PropSchema{
			"base":  codeHostMCPRefProperty(),
			"head":  codeHostMCPRefProperty(),
			"title": {Type: "string"},
			"body":  {Type: "string"},
			"draft": {Type: "boolean"},
		}
		property.Required = []string{"base", "head", "title", "draft"}
	case codehostbroker.OperationComment:
		property.Properties = map[string]PropSchema{
			"expected_head_sha": {Type: "string"},
			"body":              {Type: "string"},
		}
		property.Required = []string{"expected_head_sha", "body"}
	case codehostbroker.OperationSubmitReview, codehostbroker.OperationApprove, codehostbroker.OperationRequestChanges:
		property.Properties = map[string]PropSchema{
			"expected_head_sha": {Type: "string"},
			"body":              {Type: "string"},
		}
		property.Required = []string{"expected_head_sha"}
	case codehostbroker.OperationRetarget:
		property.Properties = map[string]PropSchema{
			"expected_head_sha": {Type: "string"},
			"current_base":      codeHostMCPRefProperty(),
			"new_base":          codeHostMCPRefProperty(),
		}
		property.Required = []string{"expected_head_sha", "current_base", "new_base"}
	case codehostbroker.OperationMarkReady, codehostbroker.OperationClose, codehostbroker.OperationReopen:
		property.Properties = map[string]PropSchema{
			"expected_head_sha": {Type: "string"},
		}
		property.Required = []string{"expected_head_sha"}
	case codehostbroker.OperationMerge:
		property.Properties = map[string]PropSchema{
			"expected_head_sha": {Type: "string"},
			"observed_base":     codeHostMCPRefProperty(),
			"method":            {Type: "string", Enum: []string{"merge", "squash", "rebase"}},
			"commit_title":      {Type: "string"},
			"commit_message":    {Type: "string"},
		}
		property.Required = []string{"expected_head_sha", "observed_base", "method"}
	}
	sort.Strings(property.Required)
	return property
}

func codeHostMCPPayloadDescription(operation codehostbroker.Operation) string {
	switch operation {
	case codehostbroker.OperationCreatePullRequest:
		return "CreatePullRequestPayload"
	case codehostbroker.OperationComment:
		return "CommentPayload"
	case codehostbroker.OperationSubmitReview, codehostbroker.OperationApprove, codehostbroker.OperationRequestChanges:
		return "ReviewPayload"
	case codehostbroker.OperationRetarget:
		return "RetargetPayload"
	case codehostbroker.OperationMarkReady, codehostbroker.OperationClose, codehostbroker.OperationReopen:
		return "LifecyclePayload"
	case codehostbroker.OperationMerge:
		return "MergePayload"
	default:
		return "Operation-specific bounded mutation payload"
	}
}

func (s *MCPServer) toolCodeHost(operation codehostbroker.Operation, args map[string]interface{}) (string, error) {
	return s.toolCodeHostContext(s.ctx, operation, args)
}

func (s *MCPServer) toolCodeHostContext(ctx context.Context, operation codehostbroker.Operation, args map[string]interface{}) (string, error) {
	request, prepare, err := decodeCodeHostMCPArgs(operation, args)
	if err != nil {
		code := codehostbroker.ErrorInvalidInput
		if errors.Is(err, errCodeHostMCPInputTooLarge) {
			code = codehostbroker.ErrorInputTooLarge
		}
		if prepare {
			return marshalCodeHostMCP(codehostbroker.PreparationResponse{
				Version: codehostbroker.Version, Operation: operation,
				Error: &codehostbroker.ContractError{
					Code: code, Message: "MCP input must match the bounded operation-specific schema",
					Field: "arguments", Retry: codehostbroker.RetryForError(code),
				},
			}), nil
		}
		return marshalCodeHostMCP(codeHostMCPErrorResponse(operation, code, "MCP input must match the bounded operation-specific schema", "arguments")), nil
	}
	broker := newCodeHostMCPBroker(s.projectRoot)
	if prepare {
		return marshalCodeHostMCP(broker.Prepare(ctx, request)), nil
	}
	return marshalCodeHostMCP(broker.Execute(ctx, request)), nil
}

func decodeCodeHostMCPArgs(operation codehostbroker.Operation, args map[string]interface{}) (codehostbroker.Request, bool, error) {
	policy, ok := codehostbroker.Policy(operation)
	if !ok {
		return codehostbroker.Request{}, false, errors.New("unsupported operation")
	}
	prepare := false
	if value, present := args["prepare"]; present {
		var valid bool
		prepare, valid = value.(bool)
		if !valid {
			return codehostbroker.Request{}, false, errors.New("prepare must be a boolean")
		}
		if !codehostbroker.IsMutation(operation) {
			return codehostbroker.Request{}, false, errors.New("read operations do not support preparation")
		}
	}
	schema := codeHostMCPInputSchema(operation, policy)
	for key := range args {
		if _, allowed := schema.Properties[key]; !allowed {
			return codehostbroker.Request{}, prepare, fmt.Errorf("unknown field %q", key)
		}
	}
	for _, key := range schema.Required {
		if value, present := args[key]; !present || value == nil {
			return codehostbroker.Request{}, prepare, fmt.Errorf("required field %q is missing", key)
		}
	}

	transport := make(map[string]interface{}, len(args)+1)
	for key, value := range args {
		if key != "prepare" {
			transport[key] = value
		}
	}
	transport["operation"] = operation
	data, err := json.Marshal(transport)
	if err != nil {
		return codehostbroker.Request{}, prepare, err
	}
	if len(data) > maxCodeHostMCPInput {
		return codehostbroker.Request{}, prepare, errCodeHostMCPInputTooLarge
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var request codehostbroker.Request
	if err := decoder.Decode(&request); err != nil {
		return codehostbroker.Request{}, prepare, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return codehostbroker.Request{}, prepare, errors.New("expected exactly one request object")
	}
	return request, prepare, nil
}

func codeHostMCPErrorResponse(operation codehostbroker.Operation, code, message, field string) codehostbroker.Response {
	policy, _ := codehostbroker.Policy(operation)
	observedAt := time.Now().UTC().Format(time.RFC3339Nano)
	return codehostbroker.Response{
		Version: codehostbroker.Version, Operation: operation,
		Provider: "unresolved", ConnectionID: "unresolved",
		Repository: codehostbroker.RepositoryIdentity{
			Host: "unresolved.invalid", Owner: "unresolved", Name: "unresolved", FullName: "unresolved/unresolved",
		},
		Policy: policy, Bounds: policy.Bounds,
		CapabilityRevision:  "capability:mcp-input-rejected",
		ObservationRevision: "observation:mcp-input-rejected",
		ObservedAt:          observedAt,
		Freshness:           codehostbroker.FreshnessUnavailable,
		RateLimit:           codehostbroker.RateLimit{ObservedAt: observedAt},
		Completeness:        codehostbroker.CompletenessUnavailable,
		PartialFailures:     []codehostbroker.PartialFailure{},
		Error: &codehostbroker.ContractError{
			Code: code, Message: message, Field: field, Retry: codehostbroker.RetryForError(code),
		},
	}
}

func marshalCodeHostMCP(value any) string {
	data, err := json.Marshal(value)
	if err == nil {
		return string(data)
	}
	return `{"version":"code-host-broker/v1","error":{"code":"encoding_error","message":"could not encode bounded code-host response","retry":"none"}}`
}
