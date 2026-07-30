package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/codehost"
)

func TestCodeHostMCPInventoryAndPoliciesMatchRegistry(t *testing.T) {
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	definitions := CodeHostToolDefinitions()
	operations := codehostbroker.Operations()
	if len(definitions) != len(operations) {
		t.Fatalf("definitions = %d, operations = %d", len(definitions), len(operations))
	}
	handlers := server.toolHandlers()
	seen := map[string]bool{}
	for index, operation := range operations {
		policy, _ := codehostbroker.Policy(operation)
		definition := definitions[index]
		wantName := codeHostMCPToolName(operation)
		if definition.Name != wantName {
			t.Fatalf("definition %d name = %q, want %q", index, definition.Name, wantName)
		}
		if seen[definition.Name] {
			t.Fatalf("duplicate code-host MCP tool %q", definition.Name)
		}
		seen[definition.Name] = true
		if _, ok := handlers[definition.Name]; !ok {
			t.Fatalf("missing handler for %q", definition.Name)
		}
		annotations := definition.Annotations
		if annotations == nil ||
			annotations.ReadOnlyHint == nil || *annotations.ReadOnlyHint != (policy.Effect == codehostbroker.EffectRead) ||
			annotations.DestructiveHint == nil || *annotations.DestructiveHint != (policy.Effect == codehostbroker.EffectCommitment) ||
			annotations.IdempotentHint == nil || *annotations.IdempotentHint != policy.ReplaySafe ||
			annotations.OpenWorldHint == nil || !*annotations.OpenWorldHint {
			t.Fatalf("%s annotations = %#v for policy %#v", operation, annotations, policy)
		}
		if definition.Meta["hero.dev/contract"] != codehostbroker.Version ||
			definition.Meta["hero.dev/operation_id"] != string(operation) ||
			definition.Meta["hero.dev/effect"] != string(policy.Effect) ||
			definition.Meta["hero.dev/consent"] != string(policy.Consent) {
			t.Fatalf("%s metadata = %#v", operation, definition.Meta)
		}
		if definition.InputSchema.AdditionalProperties == nil || *definition.InputSchema.AdditionalProperties {
			t.Fatalf("%s top-level schema is not closed", operation)
		}
		if _, acceptsRuntimeOperation := definition.InputSchema.Properties["operation"]; acceptsRuntimeOperation {
			t.Fatalf("%s schema accepts a runtime operation selector", operation)
		}
		_, hasPrepare := definition.InputSchema.Properties["prepare"]
		if hasPrepare != codehostbroker.IsMutation(operation) {
			t.Fatalf("%s prepare property = %v", operation, hasPrepare)
		}
	}
	if _, exists := handlers["hero_code_host_broker"]; exists {
		t.Fatal("generic mixed-effect code-host broker tool is registered")
	}
}

func TestCodeHostMCPNestedSchemasAreClosedAndOperationSpecific(t *testing.T) {
	definitions := map[codehostbroker.Operation]ToolDefinition{}
	for _, definition := range CodeHostToolDefinitions() {
		operation := codehostbroker.Operation(strings.TrimPrefix(definition.Name, codeHostMCPToolPrefix))
		definitions[operation] = definition
		repository := definition.InputSchema.Properties["repository"]
		requireClosedSchema(t, "repository", repository, []string{"host", "owner", "name", "full_name"})
		if repository.Properties["provider_id"].Type != "string" {
			t.Fatalf("%s repository provider_id is not typed", operation)
		}
		if pullRequest, ok := definition.InputSchema.Properties["pull_request"]; ok {
			requireClosedSchema(t, string(operation)+".pull_request", pullRequest,
				[]string{"connection_id", "repository", "provider_id", "number"})
			nestedRepository := pullRequest.Properties["repository"]
			requireClosedSchema(t, string(operation)+".pull_request.repository", nestedRepository,
				[]string{"host", "owner", "name", "full_name"})
		}
	}

	expectedPayloadRequired := map[codehostbroker.Operation][]string{
		codehostbroker.OperationCreatePullRequest: {"base", "head", "title", "draft"},
		codehostbroker.OperationComment:           {"expected_head_sha", "body"},
		codehostbroker.OperationSubmitReview:      {"expected_head_sha"},
		codehostbroker.OperationApprove:           {"expected_head_sha"},
		codehostbroker.OperationRequestChanges:    {"expected_head_sha"},
		codehostbroker.OperationMarkReady:         {"expected_head_sha"},
		codehostbroker.OperationRetarget:          {"expected_head_sha", "current_base", "new_base"},
		codehostbroker.OperationClose:             {"expected_head_sha"},
		codehostbroker.OperationReopen:            {"expected_head_sha"},
		codehostbroker.OperationMerge:             {"expected_head_sha", "observed_base", "method"},
	}
	for operation, required := range expectedPayloadRequired {
		payload := definitions[operation].InputSchema.Properties["payload"]
		requireClosedSchema(t, string(operation)+".payload", payload, required)
	}
	retarget := definitions[codehostbroker.OperationRetarget].InputSchema.Properties["payload"]
	for _, field := range []string{"current_base", "new_base"} {
		requireClosedSchema(t, "retarget."+field, retarget.Properties[field],
			[]string{"repository", "name", "sha"})
	}
	merge := definitions[codehostbroker.OperationMerge].InputSchema.Properties["payload"]
	if !slices.Equal(merge.Properties["method"].Enum, []string{"merge", "squash", "rebase"}) {
		t.Fatalf("merge method enum = %#v", merge.Properties["method"].Enum)
	}
	requireClosedSchema(t, "merge.observed_base", merge.Properties["observed_base"],
		[]string{"repository", "name", "sha"})
}

func TestCodeHostMCPAuthenticatedActorIsRepositoryScoped(t *testing.T) {
	definitions := map[codehostbroker.Operation]ToolDefinition{}
	for _, definition := range CodeHostToolDefinitions() {
		operation := codehostbroker.Operation(strings.TrimPrefix(definition.Name, codeHostMCPToolPrefix))
		definitions[operation] = definition
	}

	actorSchema := definitions[codehostbroker.OperationGetAuthenticatedActor].InputSchema
	if _, ok := actorSchema.Properties["pull_request"]; ok {
		t.Fatal("get_authenticated_actor schema accepts pull_request")
	}
	if !slices.Equal(actorSchema.Required, []string{"connection_id", "provider", "repository", "version"}) {
		t.Fatalf("get_authenticated_actor required = %#v", actorSchema.Required)
	}

	pullRequestSchema := definitions[codehostbroker.OperationGetPullRequest].InputSchema
	if _, ok := pullRequestSchema.Properties["pull_request"]; !ok {
		t.Fatal("get_pull_request schema does not accept pull_request")
	}
	if !slices.Contains(pullRequestSchema.Required, "pull_request") {
		t.Fatalf("get_pull_request required = %#v", pullRequestSchema.Required)
	}
}

func TestCodeHostMCPDispatchForcesOperationAndReturnsOneStructuredContent(t *testing.T) {
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	const credentialCanary = "AUTHORIZATION-CANARY-DO-NOT-RETURN"
	for _, operation := range codehostbroker.Operations() {
		params, err := json.Marshal(ToolCallParams{
			Name: codeHostMCPToolName(operation),
			Arguments: map[string]interface{}{
				"operation":     codehostbroker.OperationMerge,
				"authorization": credentialCanary,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		var output strings.Builder
		server.output = &output
		server.handleToolsCall(&JSONRPCRequest{
			JSONRPC: "2.0", ID: json.RawMessage(`1`), Params: params,
		})
		var response JSONRPCResponse
		if err := json.Unmarshal([]byte(output.String()), &response); err != nil {
			t.Fatalf("%s JSON-RPC response: %v", operation, err)
		}
		raw, _ := json.Marshal(response.Result)
		var call ToolCallResult
		if err := json.Unmarshal(raw, &call); err != nil {
			t.Fatalf("%s tool result: %v", operation, err)
		}
		if len(call.Content) != 1 || call.Content[0].Type != "text" || call.IsError {
			t.Fatalf("%s result = %#v", operation, call)
		}
		var envelope codehostbroker.Response
		if err := json.Unmarshal([]byte(call.Content[0].Text), &envelope); err != nil {
			t.Fatalf("%s content is not structured v1 JSON: %v", operation, err)
		}
		if envelope.Version != codehostbroker.Version || envelope.Operation != operation ||
			envelope.Error == nil || envelope.Error.Code != codehostbroker.ErrorInvalidInput {
			t.Fatalf("%s envelope = %#v", operation, envelope)
		}
		if strings.Contains(call.Content[0].Text, credentialCanary) {
			t.Fatalf("%s leaked credential canary", operation)
		}
	}
}

func TestCodeHostMCPUsesServerCancellationContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	server.SetContext(ctx)
	out, err := server.toolCodeHost(codehostbroker.OperationCapabilities, codeHostMCPReadArgs())
	if err != nil {
		t.Fatal(err)
	}
	var response codehostbroker.Response
	if err := json.Unmarshal([]byte(out), &response); err != nil {
		t.Fatal(err)
	}
	if response.Operation != codehostbroker.OperationCapabilities ||
		response.Error == nil || response.Error.Code != codehostbroker.ErrorCancelled {
		t.Fatalf("cancelled response = %#v", response)
	}
}

func TestCodeHostMCPRequestCancellationNotificationCancelsExactCall(t *testing.T) {
	originalFactory := newCodeHostMCPBroker
	t.Cleanup(func() { newCodeHostMCPBroker = originalFactory })
	broker := &cancellableCodeHostMCPBroker{started: make(chan struct{})}
	newCodeHostMCPBroker = func(string) codeHostMCPBroker { return broker }

	inputReader, inputWriter := io.Pipe()
	outputReader, outputWriter := io.Pipe()
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	server.SetIO(inputReader, outputWriter)
	baseCtx := context.Background()
	server.SetContext(baseCtx)
	runErr := make(chan error, 1)
	go func() { runErr <- server.Run() }()

	encoder := json.NewEncoder(inputWriter)
	params, err := json.Marshal(ToolCallParams{
		Name:      codeHostMCPToolName(codehostbroker.OperationCapabilities),
		Arguments: codeHostMCPReadArgs(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := encoder.Encode(JSONRPCRequest{
		JSONRPC: "2.0", ID: json.RawMessage(`73`), Method: "tools/call", Params: params,
	}); err != nil {
		t.Fatal(err)
	}
	<-broker.started
	if err := encoder.Encode(JSONRPCRequest{
		JSONRPC: "2.0", Method: "notifications/cancelled",
		Params: json.RawMessage(`{"requestId":73,"reason":"user stopped review"}`),
	}); err != nil {
		t.Fatal(err)
	}

	var response JSONRPCResponse
	if err := json.NewDecoder(outputReader).Decode(&response); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(response.Result)
	if err != nil {
		t.Fatal(err)
	}
	var call ToolCallResult
	if err := json.Unmarshal(raw, &call); err != nil {
		t.Fatal(err)
	}
	if len(call.Content) != 1 {
		t.Fatalf("tool result = %#v", call)
	}
	var envelope codehostbroker.Response
	if err := json.Unmarshal([]byte(call.Content[0].Text), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error == nil || envelope.Error.Code != codehostbroker.ErrorCancelled {
		t.Fatalf("cancelled response = %#v", envelope)
	}
	if baseCtx.Err() != nil {
		t.Fatalf("request cancellation cancelled server context: %v", baseCtx.Err())
	}
	if err := inputWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-runErr; err != nil {
		t.Fatal(err)
	}
}

func TestCodeHostMCPRunReturnsTypedErrorBeyondArgumentBound(t *testing.T) {
	args := codeHostMCPReadArgs()
	args["query"] = strings.Repeat("x", maxCodeHostMCPInput)
	params, err := json.Marshal(ToolCallParams{
		Name:      codeHostMCPToolName(codehostbroker.OperationListPullRequests),
		Arguments: args,
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := json.Marshal(JSONRPCRequest{
		JSONRPC: "2.0", ID: json.RawMessage(`91`), Method: "tools/call", Params: params,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(request) <= maxCodeHostMCPInput || len(request) >= maxMCPMessageBytes {
		t.Fatalf("test request size = %d, want argument bound < size < message bound", len(request))
	}

	var output bytes.Buffer
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	server.SetIO(bytes.NewReader(append(request, '\n')), &output)
	if err := server.Run(); err != nil {
		t.Fatalf("MCP transport rejected bounded-envelope request: %v", err)
	}
	var response JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(response.Result)
	if err != nil {
		t.Fatal(err)
	}
	var call ToolCallResult
	if err := json.Unmarshal(raw, &call); err != nil {
		t.Fatal(err)
	}
	var envelope codehostbroker.Response
	if len(call.Content) != 1 || json.Unmarshal([]byte(call.Content[0].Text), &envelope) != nil {
		t.Fatalf("tool result = %#v", call)
	}
	if envelope.Error == nil || envelope.Error.Code != codehostbroker.ErrorInputTooLarge {
		t.Fatalf("oversized input response = %#v", envelope)
	}
}

func TestCodeHostMCPDebugLogDoesNotRecordArgumentsOrResults(t *testing.T) {
	t.Setenv("HERO_MCP_DEBUG", "1")
	const (
		bodyCanary       = "MUTATION-BODY-DEBUG-CANARY"
		credentialCanary = "AUTHORIZATION-DEBUG-CANARY"
	)
	args := codeHostMCPCommentArgs(bodyCanary)
	args["prepare"] = true
	args["authorization"] = credentialCanary
	params, err := json.Marshal(ToolCallParams{
		Name: codeHostMCPToolName(codehostbroker.OperationComment), Arguments: args,
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := json.Marshal(JSONRPCRequest{
		JSONRPC: "2.0", ID: json.RawMessage(`101`), Method: "tools/call", Params: params,
	})
	if err != nil {
		t.Fatal(err)
	}

	heroDir := t.TempDir()
	var output bytes.Buffer
	server := NewMCPServer(heroDir, t.TempDir(), "test")
	server.SetIO(bytes.NewReader(append(request, '\n')), &output)
	if err := server.Run(); err != nil {
		t.Fatal(err)
	}
	logData, err := os.ReadFile(filepath.Join(heroDir, "mcp-debug.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, canary := range []string{bodyCanary, credentialCanary, `"result"`, `"arguments"`, `"authorization"`} {
		if strings.Contains(string(logData), canary) {
			t.Fatalf("debug log leaked %q: %s", canary, logData)
		}
	}
	if !strings.Contains(string(logData), "request params_bytes=") ||
		!strings.Contains(string(logData), "response status=") {
		t.Fatalf("debug log omitted safe diagnostics: %s", logData)
	}
}

func TestCodeHostMCPPreparationModeIsExplicitAndClosed(t *testing.T) {
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	args := codeHostMCPCommentArgs("MUTATION-BODY-CANARY")
	args["prepare"] = true

	out, err := server.toolCodeHost(codehostbroker.OperationComment, args)
	if err != nil {
		t.Fatal(err)
	}
	var preparation codehostbroker.PreparationResponse
	if err := json.Unmarshal([]byte(out), &preparation); err != nil {
		t.Fatal(err)
	}
	if preparation.Operation != codehostbroker.OperationComment || preparation.Error == nil ||
		preparation.Error.Code == codehostbroker.ErrorInvalidInput {
		t.Fatalf("preparation = %#v", preparation)
	}
	if err := codehostbroker.ValidatePreparationResponse(preparation); err != nil {
		t.Fatalf("invalid preparation response: %v", err)
	}
	if strings.Contains(out, "MUTATION-BODY-CANARY") || strings.Contains(out, `"result"`) {
		t.Fatalf("preparation echoed mutation material: %s", out)
	}

	delete(args, "prepare")
	out, err = server.toolCodeHost(codehostbroker.OperationComment, args)
	if err != nil {
		t.Fatal(err)
	}
	var execution codehostbroker.Response
	if err := json.Unmarshal([]byte(out), &execution); err != nil {
		t.Fatal(err)
	}
	if execution.Error == nil || execution.Error.Code != codehostbroker.ErrorInvalidInput ||
		execution.Error.Field != "capability_revision" {
		t.Fatalf("unprepared execution = %#v", execution)
	}

	args["prepare"] = true
	args["unknown"] = "SCHEMA-CANARY"
	out, err = server.toolCodeHost(codehostbroker.OperationComment, args)
	if err != nil {
		t.Fatal(err)
	}
	preparation = codehostbroker.PreparationResponse{}
	if err := json.Unmarshal([]byte(out), &preparation); err != nil {
		t.Fatal(err)
	}
	if preparation.Error == nil || preparation.Error.Code != codehostbroker.ErrorInvalidInput ||
		strings.Contains(out, "SCHEMA-CANARY") {
		t.Fatalf("invalid preparation-mode input = %s", out)
	}

	args["prepare"] = "true"
	out, err = server.toolCodeHost(codehostbroker.OperationComment, args)
	if err != nil {
		t.Fatal(err)
	}
	var nonBoolean codehostbroker.Response
	if err := json.Unmarshal([]byte(out), &nonBoolean); err != nil {
		t.Fatal(err)
	}
	if nonBoolean.Error == nil || nonBoolean.Error.Code != codehostbroker.ErrorInvalidInput {
		t.Fatalf("non-boolean prepare did not return an execution envelope: %s", out)
	}
}

func TestCodeHostMCPInputBoundReturnsTypedSafeEnvelope(t *testing.T) {
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	args := codeHostMCPReadArgs()
	const boundCanary = "BOUND-CANARY-DO-NOT-RETURN"
	args["query"] = boundCanary + strings.Repeat("x", maxCodeHostMCPInput)
	out, err := server.toolCodeHost(codehostbroker.OperationListPullRequests, args)
	if err != nil {
		t.Fatal(err)
	}
	var response codehostbroker.Response
	if err := json.Unmarshal([]byte(out), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error == nil || response.Error.Code != codehostbroker.ErrorInputTooLarge {
		t.Fatalf("oversized input response = %#v", response)
	}
	if strings.Contains(out, boundCanary) || len(out) > codehostbroker.MaxErrorDetailBytes*2 {
		t.Fatalf("oversized input leaked or produced unbounded output: %d bytes", len(out))
	}
}

func TestCodeHostMCPReadResponseMatchesInProcessBrokerSemantics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	root := t.TempDir()
	request := codehostbroker.Request{
		Version: codehostbroker.Version, Operation: codehostbroker.OperationCapabilities,
		Provider: "github", ConnectionID: "github-code-host",
		Repository: codehostbroker.RepositoryIdentity{
			Host: "github.com", Owner: "acme", Name: "widgets", FullName: "acme/widgets",
		},
	}
	direct := codehost.NewBroker(root).Execute(ctx, request)

	server := NewMCPServer(t.TempDir(), root, "test")
	server.SetContext(ctx)
	args := requestToCodeHostMCPArgs(t, request)
	out, err := server.toolCodeHost(codehostbroker.OperationCapabilities, args)
	if err != nil {
		t.Fatal(err)
	}
	var transported codehostbroker.Response
	if err := json.Unmarshal([]byte(out), &transported); err != nil {
		t.Fatal(err)
	}
	direct.ObservedAt, transported.ObservedAt = "", ""
	direct.RateLimit.ObservedAt, transported.RateLimit.ObservedAt = "", ""
	direct.DurationMS, transported.DurationMS = 0, 0
	if direct.Result == nil {
		direct.Result = json.RawMessage("null")
	}
	if !reflect.DeepEqual(direct, transported) {
		t.Fatalf("MCP response diverged from in-process broker\n direct: %#v\nMCP: %#v", direct, transported)
	}
}

func TestCodeHostMCPCanonicalFixtureParityAllOperations(t *testing.T) {
	data, err := codehostbroker.ConsumerFixture()
	if err != nil {
		t.Fatal(err)
	}
	var bundle codehostbroker.ConsumerFixtureBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatal(err)
	}
	if len(bundle.Cases) != len(codehostbroker.Operations()) {
		t.Fatalf("fixture cases = %d, operations = %d", len(bundle.Cases), len(codehostbroker.Operations()))
	}

	originalFactory := newCodeHostMCPBroker
	t.Cleanup(func() { newCodeHostMCPBroker = originalFactory })
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	handlers := server.toolHandlers()
	for _, fixtureCase := range bundle.Cases {
		fake := &fixtureCodeHostMCPBroker{response: fixtureCase.Response}
		newCodeHostMCPBroker = func(string) codeHostMCPBroker { return fake }
		handler, ok := handlers[codeHostMCPToolName(fixtureCase.Request.Operation)]
		if !ok {
			t.Fatalf("%s handler is missing", fixtureCase.Request.Operation)
		}
		out, err := handler(requestToCodeHostMCPArgs(t, fixtureCase.Request))
		if err != nil {
			t.Fatalf("%s handler error: %v", fixtureCase.Request.Operation, err)
		}
		var transported codehostbroker.Response
		if err := json.Unmarshal([]byte(out), &transported); err != nil {
			t.Fatalf("%s response decode: %v", fixtureCase.Request.Operation, err)
		}
		expectedResponse := fixtureCase.Response
		expectedResponse.Result = compactCodeHostRawJSON(t, expectedResponse.Result)
		transported.Result = compactCodeHostRawJSON(t, transported.Result)
		if !reflect.DeepEqual(transported, expectedResponse) {
			t.Fatalf("%s fixture parity mismatch", fixtureCase.Request.Operation)
		}
		expectedRequest := fixtureCase.Request
		if len(fake.requests) != 1 {
			t.Fatalf("%s dispatched request = %#v", fixtureCase.Request.Operation, fake.requests)
		}
		actualRequest := fake.requests[0]
		if !semanticCodeHostRawJSONEqual(t, actualRequest.Payload, expectedRequest.Payload) {
			t.Fatalf("%s dispatched payload changed semantics", fixtureCase.Request.Operation)
		}
		actualRequest.Payload, expectedRequest.Payload = nil, nil
		if !reflect.DeepEqual(actualRequest, expectedRequest) {
			t.Fatalf("%s dispatched request changed contract fields", fixtureCase.Request.Operation)
		}
	}
}

func TestCodeHostMCPCanonicalErrorParity(t *testing.T) {
	data, err := codehostbroker.ConsumerFixture()
	if err != nil {
		t.Fatal(err)
	}
	var bundle codehostbroker.ConsumerFixtureBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatal(err)
	}
	readCase, mergeCase := bundle.Cases[0], bundle.Cases[len(bundle.Cases)-1]
	originalFactory := newCodeHostMCPBroker
	t.Cleanup(func() { newCodeHostMCPBroker = originalFactory })
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")

	for _, fixtureError := range bundle.Errors {
		fixtureCase := readCase
		if fixtureError.Code == codehostbroker.ErrorAmbiguousResult {
			fixtureCase = mergeCase
		}
		response := fixtureCase.Response
		response.Result = json.RawMessage("null")
		response.Error = &codehostbroker.ContractError{
			Code: fixtureError.Code, Message: fixtureError.Message, Retry: fixtureError.Retry,
		}
		response.Page = nil
		response.Receipt = nil
		response.Reconciliation = nil
		if fixtureError.Code == codehostbroker.ErrorAmbiguousResult {
			response.Reconciliation = &codehostbroker.Reconciliation{
				Status: codehostbroker.ReconciliationAmbiguous,
				Key:    fixtureCase.Request.ReconciliationKey,
			}
		}
		if err := codehostbroker.ValidateResponse(response); err != nil {
			t.Fatalf("%s fixture error response is invalid: %v", fixtureError.Code, err)
		}
		fake := &fixtureCodeHostMCPBroker{response: response}
		newCodeHostMCPBroker = func(string) codeHostMCPBroker { return fake }
		out, err := server.toolCodeHost(
			fixtureCase.Request.Operation,
			requestToCodeHostMCPArgs(t, fixtureCase.Request),
		)
		if err != nil {
			t.Fatalf("%s handler error: %v", fixtureError.Code, err)
		}
		var transported codehostbroker.Response
		if err := json.Unmarshal([]byte(out), &transported); err != nil {
			t.Fatalf("%s response decode: %v", fixtureError.Code, err)
		}
		if !reflect.DeepEqual(transported, response) {
			t.Fatalf("%s error response drift", fixtureError.Code)
		}
	}
}

type fixtureCodeHostMCPBroker struct {
	response codehostbroker.Response
	requests []codehostbroker.Request
}

type cancellableCodeHostMCPBroker struct {
	started chan struct{}
	once    sync.Once
}

func (b *cancellableCodeHostMCPBroker) Execute(ctx context.Context, request codehostbroker.Request) codehostbroker.Response {
	b.once.Do(func() { close(b.started) })
	<-ctx.Done()
	return codeHostMCPErrorResponse(
		request.Operation,
		codehostbroker.ErrorCancelled,
		"request cancelled",
		"",
	)
}

func (b *cancellableCodeHostMCPBroker) Prepare(ctx context.Context, request codehostbroker.Request) codehostbroker.PreparationResponse {
	b.once.Do(func() { close(b.started) })
	<-ctx.Done()
	return codehostbroker.PreparationResponse{
		Version:   codehostbroker.Version,
		Operation: request.Operation,
		Error: &codehostbroker.ContractError{
			Code:    codehostbroker.ErrorCancelled,
			Message: "request cancelled",
			Retry:   codehostbroker.RetryForError(codehostbroker.ErrorCancelled),
		},
	}
}

func (b *fixtureCodeHostMCPBroker) Execute(_ context.Context, request codehostbroker.Request) codehostbroker.Response {
	b.requests = append(b.requests, request)
	return b.response
}

func (b *fixtureCodeHostMCPBroker) Prepare(_ context.Context, request codehostbroker.Request) codehostbroker.PreparationResponse {
	b.requests = append(b.requests, request)
	return codehostbroker.PreparationResponse{
		Version: codehostbroker.Version, Operation: request.Operation,
		CapabilityRevision: "cap:fixture", ObservationRevision: "obs:fixture",
	}
}

func requireClosedSchema(t *testing.T, label string, schema PropSchema, required []string) {
	t.Helper()
	if schema.Type != "object" || schema.AdditionalProperties == nil || *schema.AdditionalProperties {
		t.Fatalf("%s schema is not a closed object: %#v", label, schema)
	}
	want := append([]string(nil), required...)
	sortStrings(want)
	got := append([]string(nil), schema.Required...)
	sortStrings(got)
	if !slices.Equal(got, want) {
		t.Fatalf("%s required = %#v, want %#v", label, got, want)
	}
	for _, field := range want {
		if _, ok := schema.Properties[field]; !ok {
			t.Fatalf("%s required field %q has no schema", label, field)
		}
	}
}

func sortStrings(values []string) {
	slices.Sort(values)
}

func codeHostMCPReadArgs() map[string]interface{} {
	return map[string]interface{}{
		"version":       codehostbroker.Version,
		"provider":      "github",
		"connection_id": "github-code-host",
		"repository": map[string]interface{}{
			"host": "github.com", "owner": "acme", "name": "widgets", "full_name": "acme/widgets",
		},
	}
}

func codeHostMCPCommentArgs(body string) map[string]interface{} {
	args := codeHostMCPReadArgs()
	args["pull_request"] = map[string]interface{}{
		"connection_id": "github-code-host",
		"repository":    args["repository"],
		"provider_id":   "PR_42",
		"number":        42,
	}
	args["intent_source"] = "user"
	args["consent"] = string(codehostbroker.ConsentExplicitUser)
	args["idempotency_key"] = "mcp-comment"
	args["reconciliation_key"] = "reconcile:mcp-comment"
	args["payload"] = map[string]interface{}{
		"expected_head_sha": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"body":              body,
	}
	return args
}

func requestToCodeHostMCPArgs(t *testing.T, request codehostbroker.Request) map[string]interface{} {
	t.Helper()
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var args map[string]interface{}
	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatal(err)
	}
	delete(args, "operation")
	return args
}

func compactCodeHostRawJSON(t *testing.T, value json.RawMessage) json.RawMessage {
	t.Helper()
	if len(value) == 0 {
		return value
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, value); err != nil {
		t.Fatal(err)
	}
	return json.RawMessage(compact.String())
}

func semanticCodeHostRawJSONEqual(t *testing.T, left, right json.RawMessage) bool {
	t.Helper()
	if len(left) == 0 || len(right) == 0 {
		return len(left) == len(right)
	}
	var leftValue, rightValue interface{}
	if err := json.Unmarshal(left, &leftValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(right, &rightValue); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(leftValue, rightValue)
}
