package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/mockcodehost"
)

const codeHostCLICredentialCanary = "CODE-HOST-CLI-CREDENTIAL-CANARY"

func TestCodeHostContractEmitsFixturePoliciesBoundsAndDigestWithoutWorkspace(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	output := executeCodeHostCommand(t, context.Background(), "", "contract")

	var emission codeHostContractEmission
	if err := json.Unmarshal(output, &emission); err != nil {
		t.Fatal(err)
	}
	if emission.Version != codehostbroker.Version ||
		emission.SHA256 == "" ||
		len(emission.Policies) != len(codehostbroker.Operations()) ||
		emission.Bounds.BodyBytes != codehostbroker.MaxBodyBytes ||
		len(emission.Fixture) == 0 {
		t.Fatalf("contract emission=%+v fixture_bytes=%d", emission, len(emission.Fixture))
	}
	sum := sha256.Sum256([]byte(emission.Fixture))
	if got := hex.EncodeToString(sum[:]); got != emission.SHA256 {
		t.Fatalf("fixture digest=%s emitted=%s", got, emission.SHA256)
	}
	embeddedDigest, err := codehostbroker.ConsumerFixtureDigest()
	if err != nil {
		t.Fatal(err)
	}
	if emission.SHA256 != embeddedDigest {
		t.Fatalf("emitted digest=%s embedded=%s", emission.SHA256, embeddedDigest)
	}
	var fixture codehostbroker.ConsumerFixtureBundle
	if err := json.Unmarshal([]byte(emission.Fixture), &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Version != codehostbroker.Version {
		t.Fatalf("fixture version=%q", fixture.Version)
	}
}

func TestCodeHostBrokerExecutesOneRequestAndDoesNotExposeCredential(t *testing.T) {
	root, fake, repository := codeHostCLIWorkspace(t, mockcodehost.DefaultScenario())
	t.Chdir(root)
	request := codeHostCLIRequest(repository, codehostbroker.OperationCapabilities)
	output := executeCodeHostCommand(t, context.Background(), mustCodeHostJSON(t, request), "broker", "capabilities")

	var response codehostbroker.Response
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatal(err)
	}
	if err := codehostbroker.ValidateResponse(response); err != nil {
		t.Fatalf("response invalid: %+v", err)
	}
	if response.Error != nil || fake.RequestCount() == 0 {
		t.Fatalf("response=%+v provider requests=%d", response, fake.RequestCount())
	}
	if bytes.Contains(output, []byte(codeHostCLICredentialCanary)) {
		t.Fatal("broker stdout exposed the configured credential")
	}
}

func TestCodeHostBrokerMatchesEveryCanonicalFixtureCaseWithoutTranslation(t *testing.T) {
	fixtureBytes, err := codehostbroker.ConsumerFixture()
	if err != nil {
		t.Fatal(err)
	}
	var fixture codehostbroker.ConsumerFixtureBundle
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) != len(codehostbroker.Operations()) {
		t.Fatalf("fixture cases=%d operations=%d", len(fixture.Cases), len(codehostbroker.Operations()))
	}
	originalFactory := newCodeHostBrokerService
	t.Cleanup(func() { newCodeHostBrokerService = originalFactory })
	for _, fixtureCase := range fixture.Cases {
		t.Run(fixtureCase.Name, func(t *testing.T) {
			stub := &fixtureCodeHostBroker{
				t: t, expected: fixtureCase.Request, response: fixtureCase.Response,
			}
			newCodeHostBrokerService = func(string) codeHostBrokerService { return stub }
			output := executeCodeHostCommand(
				t, context.Background(), mustCodeHostJSON(t, fixtureCase.Request),
				"broker", string(fixtureCase.Request.Operation),
			)
			var actual codehostbroker.Response
			if err := json.Unmarshal(output, &actual); err != nil {
				t.Fatal(err)
			}
			if stub.calls != 1 {
				t.Fatalf("broker calls=%d", stub.calls)
			}
			if !reflect.DeepEqual(canonicalCodeHostJSON(t, actual), canonicalCodeHostJSON(t, fixtureCase.Response)) {
				t.Fatalf("response drift\nactual=%s\nexpected=%s", output, mustCodeHostJSON(t, fixtureCase.Response))
			}
		})
	}
}

func TestCodeHostBrokerPreservesEveryCanonicalError(t *testing.T) {
	fixtureBytes, err := codehostbroker.ConsumerFixture()
	if err != nil {
		t.Fatal(err)
	}
	var fixture codehostbroker.ConsumerFixtureBundle
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatal(err)
	}
	readCase, mergeCase := fixture.Cases[0], fixture.Cases[len(fixture.Cases)-1]
	originalFactory := newCodeHostBrokerService
	t.Cleanup(func() { newCodeHostBrokerService = originalFactory })

	for _, fixtureError := range fixture.Errors {
		t.Run(fixtureError.Code, func(t *testing.T) {
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
				t.Fatalf("fixture error response is invalid: %v", err)
			}
			stub := &fixtureCodeHostBroker{
				t: t, expected: fixtureCase.Request, response: response,
			}
			newCodeHostBrokerService = func(string) codeHostBrokerService { return stub }
			output := executeCodeHostCommand(
				t,
				context.Background(),
				mustCodeHostJSON(t, fixtureCase.Request),
				"broker",
				string(fixtureCase.Request.Operation),
			)
			var actual codehostbroker.Response
			if err := json.Unmarshal(output, &actual); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(canonicalCodeHostJSON(t, actual), canonicalCodeHostJSON(t, response)) {
				t.Fatalf("error response drift\nactual=%s\nexpected=%s", output, mustCodeHostJSON(t, response))
			}
		})
	}
}

func TestCodeHostBrokerRejectsMismatchTrailingUnknownAndOversizedInputBeforeProvider(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		input     func(codehostbroker.RepositoryIdentity) string
		errorCode string
		canary    string
	}{
		{
			name: "operation_mismatch", path: "get_pull_request",
			input: func(repository codehostbroker.RepositoryIdentity) string {
				return mustCodeHostJSON(t, codeHostCLIRequest(repository, codehostbroker.OperationCapabilities))
			},
			errorCode: codehostbroker.ErrorInvalidInput,
		},
		{
			name: "trailing_object", path: "capabilities",
			input: func(repository codehostbroker.RepositoryIdentity) string {
				return mustCodeHostJSON(t, codeHostCLIRequest(repository, codehostbroker.OperationCapabilities)) + ` {}`
			},
			errorCode: codehostbroker.ErrorInvalidInput,
		},
		{
			name: "unknown_field", path: "capabilities",
			input: func(codehostbroker.RepositoryIdentity) string {
				return `{"unknown":"CLI-UNKNOWN-FIELD-CANARY"}`
			},
			errorCode: codehostbroker.ErrorInvalidInput, canary: "CLI-UNKNOWN-FIELD-CANARY",
		},
		{
			name: "oversized", path: "capabilities",
			input: func(codehostbroker.RepositoryIdentity) string {
				return `{}` + strings.Repeat(" ", maxCodeHostBrokerInputBytes+1)
			},
			errorCode: codehostbroker.ErrorInputTooLarge,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, fake, repository := codeHostCLIWorkspace(t, mockcodehost.DefaultScenario())
			t.Chdir(root)
			output := executeCodeHostCommand(t, context.Background(), test.input(repository), "broker", test.path)
			var response codehostbroker.Response
			if err := json.Unmarshal(output, &response); err != nil {
				t.Fatal(err)
			}
			if err := codehostbroker.ValidateResponse(response); err != nil {
				t.Fatalf("response invalid: %+v", err)
			}
			if response.Error == nil || response.Error.Code != test.errorCode || fake.RequestCount() != 0 {
				t.Fatalf("response=%+v provider requests=%d", response, fake.RequestCount())
			}
			if test.canary != "" && bytes.Contains(output, []byte(test.canary)) {
				t.Fatal("structured CLI error reflected untrusted input")
			}
			var extra any
			decoder := json.NewDecoder(bytes.NewReader(output))
			if err := decoder.Decode(&extra); err != nil {
				t.Fatal(err)
			}
			if err := decoder.Decode(&extra); err != io.EOF {
				t.Fatalf("stdout contained more than one JSON value: %v", err)
			}
		})
	}
}

func TestCodeHostBrokerPrepareInputFailuresUsePreparationEnvelope(t *testing.T) {
	root, fake, repository := codeHostCLIWorkspace(t, mockcodehost.DefaultScenario())
	t.Chdir(root)
	validComment := codeHostCLIRequest(repository, codehostbroker.OperationComment)
	tests := []struct {
		name      string
		input     string
		errorCode string
	}{
		{
			name:      "operation_mismatch",
			input:     mustCodeHostJSON(t, codeHostCLIRequest(repository, codehostbroker.OperationCapabilities)),
			errorCode: codehostbroker.ErrorInvalidInput,
		},
		{
			name:      "trailing",
			input:     mustCodeHostJSON(t, validComment) + ` {}`,
			errorCode: codehostbroker.ErrorInvalidInput,
		},
		{
			name:      "oversized",
			input:     `{}` + strings.Repeat(" ", maxCodeHostBrokerInputBytes+1),
			errorCode: codehostbroker.ErrorInputTooLarge,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := executeCodeHostCommand(t, context.Background(), test.input, "broker", "comment", "--prepare")
			var response codehostbroker.PreparationResponse
			if err := json.Unmarshal(output, &response); err != nil {
				t.Fatal(err)
			}
			if err := codehostbroker.ValidatePreparationResponse(response); err != nil {
				t.Fatalf("preparation error response invalid: %v", err)
			}
			if response.Version != codehostbroker.Version ||
				response.Operation != codehostbroker.OperationComment ||
				response.Error == nil ||
				response.Error.Code != test.errorCode ||
				response.CapabilityRevision != "" ||
				response.ObservationRevision != "" {
				t.Fatalf("preparation error response=%+v", response)
			}
		})
	}
	if fake.RequestCount() != 0 {
		t.Fatalf("invalid preparation input reached provider %d times", fake.RequestCount())
	}
}

func TestCodeHostBrokerPrepareThenExecuteIsExplicitAndDoesNotEchoPayload(t *testing.T) {
	root, fake, repository := codeHostCLIWorkspace(t, mockcodehost.DefaultScenario())
	t.Chdir(root)
	request := codeHostCLIRequest(repository, codehostbroker.OperationComment)
	request.IntentSource = "user"
	request.Consent = codehostbroker.ConsentExplicitUser
	request.IdempotencyKey = "cli-comment"
	request.CapabilityRevision = "prepare"
	request.ObservationRevision = "prepare"
	request.ReconciliationKey = "reconcile:cli-comment"
	payload, err := json.Marshal(codehostbroker.CommentPayload{
		ExpectedHeadSHA: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Body:            "CLI-PREPARATION-BODY-CANARY",
	})
	if err != nil {
		t.Fatal(err)
	}
	request.Payload = payload

	preparedOutput := executeCodeHostCommand(t, context.Background(), mustCodeHostJSON(t, request), "broker", "comment", "--prepare")
	var prepared codehostbroker.PreparationResponse
	if err := json.Unmarshal(preparedOutput, &prepared); err != nil {
		t.Fatal(err)
	}
	if err := codehostbroker.ValidatePreparationResponse(prepared); err != nil {
		t.Fatalf("preparation invalid: %+v", err)
	}
	if prepared.Error != nil || prepared.CapabilityRevision == "" || prepared.ObservationRevision == "" ||
		fake.CollaborationAttempts() != 0 {
		t.Fatalf("preparation=%+v attempts=%d", prepared, fake.CollaborationAttempts())
	}
	if bytes.Contains(preparedOutput, []byte("CLI-PREPARATION-BODY-CANARY")) {
		t.Fatal("preparation response echoed mutation payload")
	}

	request.CapabilityRevision = prepared.CapabilityRevision
	request.ObservationRevision = prepared.ObservationRevision
	executedOutput := executeCodeHostCommand(t, context.Background(), mustCodeHostJSON(t, request), "broker", "comment")
	var executed codehostbroker.Response
	if err := json.Unmarshal(executedOutput, &executed); err != nil {
		t.Fatal(err)
	}
	if err := codehostbroker.ValidateResponse(executed); err != nil {
		t.Fatalf("execution invalid: %+v", err)
	}
	if executed.Error != nil || fake.CollaborationAttempts() != 1 {
		t.Fatalf("execution=%+v attempts=%d", executed, fake.CollaborationAttempts())
	}
}

func TestCodeHostBrokerPrepareRejectsReadsAndContextCancellationPropagates(t *testing.T) {
	root, fake, repository := codeHostCLIWorkspace(t, mockcodehost.DefaultScenario())
	t.Chdir(root)
	request := codeHostCLIRequest(repository, codehostbroker.OperationCapabilities)

	preparedOutput := executeCodeHostCommand(t, context.Background(), mustCodeHostJSON(t, request), "broker", "capabilities", "--prepare")
	var prepared codehostbroker.PreparationResponse
	if err := json.Unmarshal(preparedOutput, &prepared); err != nil {
		t.Fatal(err)
	}
	if prepared.Error == nil || prepared.Error.Code != codehostbroker.ErrorUnsupportedOperation || fake.RequestCount() != 0 {
		t.Fatalf("preparation=%+v provider requests=%d", prepared, fake.RequestCount())
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	output := executeCodeHostCommand(t, ctx, mustCodeHostJSON(t, request), "broker", "capabilities")
	var response codehostbroker.Response
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatal(err)
	}
	if err := codehostbroker.ValidateResponse(response); err != nil {
		t.Fatalf("cancel response invalid: %+v", err)
	}
	if response.Error == nil || response.Error.Code != codehostbroker.ErrorCancelled || fake.RequestCount() != 0 {
		t.Fatalf("cancel response=%+v provider requests=%d", response, fake.RequestCount())
	}
}

func TestCodeHostBrokerReadPreparationRejectsBeforeReadingInput(t *testing.T) {
	root, fake, _ := codeHostCLIWorkspace(t, mockcodehost.DefaultScenario())
	t.Chdir(root)
	output := executeCodeHostCommand(
		t,
		context.Background(),
		`{"malformed":"READ-PREPARE-CANARY"}`,
		"broker",
		"capabilities",
		"--prepare",
	)
	var response codehostbroker.PreparationResponse
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatal(err)
	}
	if err := codehostbroker.ValidatePreparationResponse(response); err != nil {
		t.Fatalf("read preparation rejection is not a valid envelope: %v", err)
	}
	if response.Error == nil || response.Error.Code != codehostbroker.ErrorUnsupportedOperation ||
		fake.RequestCount() != 0 || bytes.Contains(output, []byte("READ-PREPARE-CANARY")) {
		t.Fatalf("response=%+v provider_requests=%d", response, fake.RequestCount())
	}
}

func TestCodeHostBrokerRejectsArgvContentBeforeBrokerConstruction(t *testing.T) {
	originalFactory := newCodeHostBrokerService
	constructions := 0
	newCodeHostBrokerService = func(projectRoot string) codeHostBrokerService {
		constructions++
		return originalFactory(projectRoot)
	}
	t.Cleanup(func() { newCodeHostBrokerService = originalFactory })

	tests := []struct {
		name   string
		args   []string
		canary string
	}{
		{name: "extra_token", args: []string{"broker", "capabilities", "CLI-POSITIONAL-CANARY"}, canary: "CLI-POSITIONAL-CANARY"},
		{name: "token_flag", args: []string{"broker", "capabilities", "--token=CLI-TOKEN-CANARY"}, canary: "CLI-TOKEN-CANARY"},
		{name: "header_flag", args: []string{"broker", "capabilities", "--header=CLI-HEADER-CANARY"}, canary: "CLI-HEADER-CANARY"},
		{name: "body_flag", args: []string{"broker", "capabilities", "--body=CLI-BODY-CANARY"}, canary: "CLI-BODY-CANARY"},
		{name: "review_flag", args: []string{"broker", "capabilities", "--review-text=CLI-REVIEW-CANARY"}, canary: "CLI-REVIEW-CANARY"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := newCodeHostCommand()
			command.SetArgs(test.args)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			err := command.ExecuteContext(context.Background())
			if err == nil {
				t.Fatal("argv content unexpectedly accepted")
			}
			visible := stdout.String() + stderr.String() + err.Error()
			if strings.Contains(visible, test.canary) {
				t.Fatalf("argv canary reflected in CLI output: %q", visible)
			}
		})
	}
	if constructions != 0 {
		t.Fatalf("invalid argv constructed the broker %d times", constructions)
	}
}

func TestReleasedShapeHeroBinaryExercisesContractReadAndPreparedWrite(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "hero")
	build := exec.Command("go", "build", "-o", binary, "../../cmd/hero")
	build.Dir = "."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build hero: %v\n%s", err, output)
	}
	output := executeCodeHostBinary(t, binary, t.TempDir(), "", "code-host", "contract")
	var emission codeHostContractEmission
	if err := json.Unmarshal(output, &emission); err != nil {
		t.Fatal(err)
	}
	if emission.Version != codehostbroker.Version || emission.SHA256 == "" || len(emission.Fixture) == 0 {
		t.Fatalf("binary contract emission=%+v", emission)
	}

	root, fake, repository := codeHostCLIWorkspace(t, mockcodehost.DefaultScenario())
	readRequest := codeHostCLIRequest(repository, codehostbroker.OperationCapabilities)
	readOutput := executeCodeHostBinary(t, binary, root, mustCodeHostJSON(t, readRequest), "code-host", "broker", "capabilities")
	var readResponse codehostbroker.Response
	if err := json.Unmarshal(readOutput, &readResponse); err != nil {
		t.Fatal(err)
	}
	if err := codehostbroker.ValidateResponse(readResponse); err != nil || readResponse.Error != nil {
		t.Fatalf("binary read response=%+v validation=%+v", readResponse, err)
	}

	writeRequest := codeHostCLIRequest(repository, codehostbroker.OperationComment)
	writeRequest.IntentSource = "user"
	writeRequest.Consent = codehostbroker.ConsentExplicitUser
	writeRequest.IdempotencyKey = "binary-comment"
	writeRequest.CapabilityRevision = "prepare"
	writeRequest.ObservationRevision = "prepare"
	writeRequest.ReconciliationKey = "reconcile:binary-comment"
	payload, err := json.Marshal(codehostbroker.CommentPayload{
		ExpectedHeadSHA: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Body:            "BINARY-COMMENT-CANARY",
	})
	if err != nil {
		t.Fatal(err)
	}
	writeRequest.Payload = payload
	preparedOutput := executeCodeHostBinary(t, binary, root, mustCodeHostJSON(t, writeRequest), "code-host", "broker", "comment", "--prepare")
	var prepared codehostbroker.PreparationResponse
	if err := json.Unmarshal(preparedOutput, &prepared); err != nil {
		t.Fatal(err)
	}
	if err := codehostbroker.ValidatePreparationResponse(prepared); err != nil || prepared.Error != nil {
		t.Fatalf("binary preparation=%+v validation=%+v", prepared, err)
	}
	if bytes.Contains(preparedOutput, []byte("BINARY-COMMENT-CANARY")) {
		t.Fatal("binary preparation echoed mutation body")
	}
	writeRequest.CapabilityRevision = prepared.CapabilityRevision
	writeRequest.ObservationRevision = prepared.ObservationRevision
	writeOutput := executeCodeHostBinary(t, binary, root, mustCodeHostJSON(t, writeRequest), "code-host", "broker", "comment")
	var writeResponse codehostbroker.Response
	if err := json.Unmarshal(writeOutput, &writeResponse); err != nil {
		t.Fatal(err)
	}
	if err := codehostbroker.ValidateResponse(writeResponse); err != nil ||
		writeResponse.Error != nil ||
		writeResponse.Reconciliation == nil ||
		writeResponse.Reconciliation.Status != codehostbroker.ReconciliationApplied ||
		fake.CollaborationAttempts() != 1 {
		t.Fatalf("binary write response=%+v validation=%+v attempts=%d", writeResponse, err, fake.CollaborationAttempts())
	}
}

func executeCodeHostCommand(t *testing.T, ctx context.Context, stdin string, args ...string) []byte {
	t.Helper()
	command := newCodeHostCommand()
	command.SetArgs(args)
	command.SetIn(strings.NewReader(stdin))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	if err := command.ExecuteContext(ctx); err != nil {
		t.Fatalf("code-host %v: %v stderr=%s", args, err, stderr.String())
	}
	return stdout.Bytes()
}

func executeCodeHostBinary(t *testing.T, binary, directory, stdin string, args ...string) []byte {
	t.Helper()
	command := exec.Command(binary, args...)
	command.Dir = directory
	command.Stdin = strings.NewReader(stdin)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("hero %v: %v stderr=%s", args, err, stderr.String())
	}
	return stdout.Bytes()
}

func codeHostCLIWorkspace(t *testing.T, scenario mockcodehost.Scenario) (string, *mockcodehost.Server, codehostbroker.RepositoryIdentity) {
	t.Helper()
	fake := mockcodehost.NewServer(scenario)
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Host = "localhost:" + parsed.Port()
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	shared := fmt.Sprintf(`{
		"folder": ".hero",
		"integrations": {
			"roles": {"code-host": "github-code-host"},
			"connections": {
				"github-code-host": {
					"provider": "github",
					"capabilities": ["code-host"],
					"settings": {"project": "acme/widgets", "base_url": %q}
				}
			}
		}
	}`, parsed.String())
	local := fmt.Sprintf(`{"integrations":{"connections":{"github-code-host":{"auth":{"token":%q}}}}}`, codeHostCLICredentialCanary)
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"), []byte(shared), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "hero.local.json"), []byte(local), 0o600); err != nil {
		t.Fatal(err)
	}
	repository := codehostbroker.RepositoryIdentity{
		Host: "localhost", ProviderID: "R_acme_widgets",
		Owner: "acme", Name: "widgets", FullName: "acme/widgets",
	}
	return root, fake, repository
}

func codeHostCLIRequest(repository codehostbroker.RepositoryIdentity, operation codehostbroker.Operation) codehostbroker.Request {
	request := codehostbroker.Request{
		Version: codehostbroker.Version, Operation: operation, Provider: "github",
		ConnectionID: "github-code-host", Repository: repository,
	}
	if operation != codehostbroker.OperationCapabilities &&
		operation != codehostbroker.OperationListPullRequests &&
		operation != codehostbroker.OperationSearchPullRequests {
		request.PullRequest = &codehostbroker.PullRequestIdentity{
			ConnectionID: request.ConnectionID, Repository: repository, ProviderID: "PR_42", Number: 42,
		}
	}
	return request
}

func mustCodeHostJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

type fixtureCodeHostBroker struct {
	t        *testing.T
	expected codehostbroker.Request
	response codehostbroker.Response
	calls    int
}

func (f *fixtureCodeHostBroker) Execute(_ context.Context, request codehostbroker.Request) codehostbroker.Response {
	f.t.Helper()
	f.calls++
	if !reflect.DeepEqual(canonicalCodeHostJSON(f.t, request), canonicalCodeHostJSON(f.t, f.expected)) {
		f.t.Fatalf("request drift\nactual=%s\nexpected=%s", mustCodeHostJSON(f.t, request), mustCodeHostJSON(f.t, f.expected))
	}
	return f.response
}

func (f *fixtureCodeHostBroker) Prepare(_ context.Context, request codehostbroker.Request) codehostbroker.PreparationResponse {
	f.t.Helper()
	f.t.Fatalf("unexpected prepare for %s", request.Operation)
	return codehostbroker.PreparationResponse{}
}

func canonicalCodeHostJSON(t *testing.T, value any) any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var canonical any
	if err := json.Unmarshal(data, &canonical); err != nil {
		t.Fatal(err)
	}
	return canonical
}
