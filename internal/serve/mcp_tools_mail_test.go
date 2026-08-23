package serve

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailthread"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/attention/mailquery"
)

func testMCPEnvelope(id string) attention.MailEnvelope {
	return attention.MailEnvelope{
		SchemaVersion: 1, ID: id, Recipient: attention.ProjectReference{PeerID: "peer_b", DisplayName: "B"},
		Sender: attention.ProjectReference{PeerID: "peer_a", DisplayName: "A"}, Subject: "Request", Body: "Please inspect.",
		Kind: attention.MailKindRequest, ThreadID: id, Revision: 7, IdempotencyKey: "delivery-1", CreatedAt: "2026-07-22T18:00:00Z",
	}
}

func testMCPDelivery(env attention.MailEnvelope) attention.MailDelivery {
	return attention.MailDelivery{SchemaVersion: 1, MessageID: env.ID, ThreadID: env.ThreadID, Sender: env.Sender, Recipient: env.Recipient, IdempotencyKey: env.IdempotencyKey, DeliveredAt: env.CreatedAt}
}

func setupMCPMail(t *testing.T) (*MCPServer, string) {
	t.Helper()
	project := t.TempDir()
	heroDir := filepath.Join(project, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"), []byte(`{"folder":".hero","peer_id":"peer_b"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	state := t.TempDir()
	store, _ := mail.NewStore(state)
	env := testMCPEnvelope("mail_mcp")
	if _, _, err := store.Deliver(env, testMCPDelivery(env)); err != nil {
		t.Fatal(err)
	}
	server := NewMCPServer(heroDir, project, "test")
	server.attentionStateRoot = state
	return server, env.ID
}

func TestMCPMailToolsAdvertiseAndReturnStructuredFailures(t *testing.T) {
	server, id := setupMCPMail(t)
	list, err := server.toolMailList(map[string]interface{}{"unread": true})
	if err != nil {
		t.Fatal(err)
	}
	var messages []mailMCPMessage
	if err := json.Unmarshal([]byte(list), &messages); err != nil || len(messages) != 1 || len(messages[0].Actions) != 5 {
		t.Fatalf("list = %s, %v", list, err)
	}
	if messages[0].Body != "Please inspect." {
		t.Fatalf("legacy list body shape changed: %#v", messages[0])
	}
	legacyIDs := make(map[string]bool, len(messages[0].Actions))
	for _, descriptor := range messages[0].Actions {
		legacyIDs[descriptor.ID] = true
	}
	if !legacyIDs[mail.ActionRead] || legacyIDs["mark_read"] || strings.Contains(list, `"operation_id"`) {
		t.Fatalf("legacy descriptor shape changed: %s", list)
	}
	shown, err := server.toolMailShow(map[string]interface{}{"message_id": id})
	if err != nil || !strings.Contains(shown, `"body":"Please inspect."`) || !strings.Contains(shown, `"id":"read"`) || strings.Contains(shown, "mark_read") {
		t.Fatalf("legacy show shape changed: %s, %v", shown, err)
	}
	missingShow, err := server.toolMailShow(map[string]interface{}{"message_id": "mail_missing"})
	if err != nil {
		t.Fatal(err)
	}
	var missingFailure mailMCPError
	if err := json.Unmarshal([]byte(missingShow), &missingFailure); err != nil || missingFailure.Code != "missing" {
		t.Fatalf("missing show = %s, %v", missingShow, err)
	}
	resultText, err := server.toolMailAction(map[string]interface{}{"message_id": id, "action": "read", "revision": float64(0), "idempotency_key": "read-1"})
	if err != nil {
		t.Fatal(err)
	}
	var result mail.ActionResult
	if err := json.Unmarshal([]byte(resultText), &result); err != nil || result.Receipt.Revision == 0 {
		t.Fatalf("action = %s, %v", resultText, err)
	}
	for name, testCase := range map[string]struct {
		args map[string]interface{}
		code string
	}{
		"stale":       {map[string]interface{}{"message_id": id, "action": "dismiss", "revision": float64(0), "idempotency_key": "dismiss-1"}, "stale_revision"},
		"missing":     {map[string]interface{}{"message_id": "mail_missing", "action": "read", "revision": float64(0), "idempotency_key": "read-2"}, "missing"},
		"unsupported": {map[string]interface{}{"message_id": id, "action": "execute", "revision": float64(result.Receipt.Revision), "idempotency_key": "bad-1"}, "unsupported_action"},
	} {
		t.Run(name, func(t *testing.T) {
			out, err := server.toolMailAction(testCase.args)
			if err != nil {
				t.Fatal(err)
			}
			var failure mailMCPError
			if err := json.Unmarshal([]byte(out), &failure); err != nil || failure.Code != testCase.code {
				t.Fatalf("failure = %s, %v", out, err)
			}
			if name == "stale" && failure.Current == nil {
				t.Fatal("stale failure omitted authoritative current row")
			}
		})
	}
}

func writeMCPMailProject(t *testing.T, root, id, name string, repos map[string]string) {
	t.Helper()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configBytes, err := json.Marshal(map[string]interface{}{
		"folder": ".hero", "peer_id": id, "repos": repos,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"), configBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := "schema: 1\ncontracts_version: 1\nrepo:\n  peer_id: " + id + "\n  name: " + name + "\ngenerated_at: 2026-07-22T18:00:00Z\n"
	if err := os.WriteFile(filepath.Join(heroDir, "peer-manifest.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestMCPMailSendReplyAreTypedIdempotentAndInert(t *testing.T) {
	projectA, projectB, state := t.TempDir(), t.TempDir(), t.TempDir()
	writeMCPMailProject(t, projectA, "peer_a", "A", map[string]string{"b": projectB})
	writeMCPMailProject(t, projectB, "peer_b", "B", map[string]string{"a": projectA})
	serverA := NewMCPServer(filepath.Join(projectA, ".hero"), projectA, "test")
	serverA.attentionStateRoot = state

	args := map[string]interface{}{
		"schema_version": float64(1), "intent_source": "user", "recipient": "b", "recipient_peer_id": "peer_b",
		"subject": "Request", "body": `{"tool":"hero_focus_create","arguments":{"title":"do not run"}}`,
		"kind": "request", "source_kind": "session", "source_id": "session_1", "idempotency_key": "send_1",
	}
	out, err := serverA.toolMailSend(args)
	if err != nil {
		t.Fatal(err)
	}
	var result attention.ActionResult
	if err := json.Unmarshal([]byte(out), &result); err != nil || result.Error != nil {
		t.Fatalf("send = %s, %v", out, err)
	}
	var sent attention.MailDelivery
	if err := json.Unmarshal(result.Source, &sent); err != nil || sent.MessageID == "" || sent.ThreadID != sent.MessageID {
		t.Fatalf("delivery = %#v, %v", sent, err)
	}
	replayed, _ := serverA.toolMailSend(args)
	var replayResult attention.ActionResult
	_ = json.Unmarshal([]byte(replayed), &replayResult)
	var replayDelivery attention.MailDelivery
	_ = json.Unmarshal(replayResult.Source, &replayDelivery)
	if replayDelivery.MessageID != sent.MessageID {
		t.Fatalf("send replay duplicated: %#v %#v", sent, replayDelivery)
	}
	args["body"] = "changed"
	conflict, _ := serverA.toolMailSend(args)
	var conflictResult attention.ActionResult
	_ = json.Unmarshal([]byte(conflict), &conflictResult)
	if conflictResult.Error == nil || conflictResult.Error.Code != attention.ErrorIdempotencyConflict {
		t.Fatalf("conflict = %s", conflict)
	}
	args["body"] = `{"tool":"hero_focus_create","arguments":{"title":"do not run"}}`
	args["idempotency_key"] = "send_wrong_peer"
	args["recipient_peer_id"] = "peer_other"
	mismatch, _ := serverA.toolMailSend(args)
	var mismatchResult attention.ActionResult
	_ = json.Unmarshal([]byte(mismatch), &mismatchResult)
	if mismatchResult.Error == nil || mismatchResult.Error.Code != attention.ErrorValidation || mismatchResult.Error.Field != "recipient_peer_id" {
		t.Fatalf("recipient mismatch = %s", mismatch)
	}

	store, _ := mail.NewStore(state)
	stored, err := store.Get("peer_b", sent.MessageID)
	if err != nil || !bytes.Contains([]byte(stored.Body), []byte("hero_focus_create")) || len(stored.Provenance) != 1 {
		t.Fatalf("stored inert envelope = %#v, %v", stored, err)
	}

	serverB := NewMCPServer(filepath.Join(projectB, ".hero"), projectB, "test")
	serverB.attentionStateRoot = state
	replyOut, err := serverB.toolMailReply(map[string]interface{}{
		"schema_version": float64(1), "intent_source": "user", "message_id": sent.MessageID,
		"thread_id": sent.ThreadID, "body": "Friday works.", "idempotency_key": "reply_1",
	})
	if err != nil {
		t.Fatal(err)
	}
	var replyResult attention.ActionResult
	_ = json.Unmarshal([]byte(replyOut), &replyResult)
	var replied attention.MailDelivery
	_ = json.Unmarshal(replyResult.Source, &replied)
	envelope, err := store.Get("peer_a", replied.MessageID)
	if err != nil || envelope.ThreadID != sent.ThreadID || envelope.InReplyTo != sent.MessageID || envelope.Recipient.PeerID != "peer_a" {
		t.Fatalf("threaded reply = %#v, %v", envelope, err)
	}
	stale, _ := serverB.toolMailReply(map[string]interface{}{
		"schema_version": float64(1), "intent_source": "user", "message_id": sent.MessageID,
		"thread_id": "mail_wrong", "body": "No", "idempotency_key": "reply_2",
	})
	var staleResult attention.ActionResult
	_ = json.Unmarshal([]byte(stale), &staleResult)
	if staleResult.Error == nil || staleResult.Error.Code != attention.ErrorStale || staleResult.Error.Field != "thread_id" {
		t.Fatalf("stale thread = %s", stale)
	}
	versioned, _ := serverB.toolMailReply(map[string]interface{}{
		"schema_version": float64(2), "intent_source": "user", "message_id": sent.MessageID,
		"thread_id": sent.ThreadID, "body": "No", "idempotency_key": "reply_v2",
	})
	var versionResult attention.ActionResult
	_ = json.Unmarshal([]byte(versioned), &versionResult)
	if versionResult.Error == nil || versionResult.Error.Code != attention.ErrorIncompatibleVersion {
		t.Fatalf("version mismatch = %s", versioned)
	}

	if err := os.WriteFile(filepath.Join(projectB, ".hero", "peer-manifest.yaml"), []byte("not: [valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	unavailable, _ := serverA.toolMailSend(map[string]interface{}{
		"schema_version": float64(1), "intent_source": "user", "recipient": "b", "recipient_peer_id": "peer_b",
		"subject": "Unavailable", "body": "No write.", "idempotency_key": "send_unavailable",
	})
	var unavailableResult attention.ActionResult
	_ = json.Unmarshal([]byte(unavailable), &unavailableResult)
	if unavailableResult.Error == nil || unavailableResult.Error.Code != attention.ErrorUnavailable {
		t.Fatalf("unavailable mail authority = %s", unavailable)
	}
}

func TestMCPMailDefinitionsAndDispatch(t *testing.T) {
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	definitions := make(map[string]ToolDefinition)
	for _, definition := range server.toolDefinitions() {
		definitions[definition.Name] = definition
	}
	for _, name := range []string{
		"hero_mail_list", "hero_mail_show", "hero_mail_send", "hero_mail_reply", "hero_mail_action",
		"hero_mail_thread_list", "hero_mail_thread_show", "hero_mail_thread_action", "hero_mail_thread_contract",
	} {
		if _, ok := server.toolHandlers()[name]; !ok {
			t.Errorf("missing handler %s", name)
		}
		found := false
		if _, ok := definitions[name]; ok {
			found = true
		}
		if !found {
			t.Errorf("missing definition %s", name)
		}
	}
	for _, name := range []string{"hero_mail_thread_list", "hero_mail_thread_show", "hero_mail_thread_contract"} {
		definition := definitions[name]
		if definition.Annotations == nil || definition.Annotations.ReadOnlyHint == nil || !*definition.Annotations.ReadOnlyHint ||
			definition.Meta["hero.dev/effect"] != string(attention.EffectRead) || definition.Meta["hero.dev/consent"] != string(attention.ConsentNone) {
			t.Fatalf("%s metadata = %#v / %#v", name, definition.Annotations, definition.Meta)
		}
	}
	action := definitions["hero_mail_thread_action"]
	if action.InputSchema.Properties["identity"].Type != "object" ||
		len(action.InputSchema.Properties["identity"].Required) != 2 ||
		len(action.InputSchema.Required) != 5 {
		t.Fatalf("thread action schema = %#v", action.InputSchema)
	}
}

func TestMCPMailThreadDirectDispatchMatchesHTTP(t *testing.T) {
	api, _, _, threadID := setupAttentionMailHTTP(t, "thread parity body")
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	server.mailQueryService = api.mailQueryService

	assertParity := func(name, path string, args map[string]interface{}, target any) string {
		t.Helper()
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("HTTP %s = %d %s", name, response.Code, response.Body.String())
		}
		call := callTool(t, server, name, args)
		if call.IsError || len(call.Content) != 1 {
			t.Fatalf("MCP %s = %#v", name, call)
		}
		var compact bytes.Buffer
		if err := json.Compact(&compact, response.Body.Bytes()); err != nil {
			t.Fatalf("compact HTTP %s: %v", name, err)
		}
		if compact.String() != call.Content[0].Text {
			t.Fatalf("%s parity mismatch\nHTTP: %s\nMCP:  %s", name, response.Body.String(), call.Content[0].Text)
		}
		if err := json.Unmarshal([]byte(call.Content[0].Text), target); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		return call.Content[0].Text
	}

	var list mailthread.ThreadListResponse
	assertParity(
		"hero_mail_thread_list",
		"/api/attention/v1/mail/threads?project_peer_id=peer_b&bucket=needs_attention",
		map[string]interface{}{"project_peer_id": "peer_b", "bucket": "needs_attention"},
		&list,
	)
	if list.Error != nil || len(list.Items) != 1 || list.Items[0].Identity.ThreadID != threadID {
		t.Fatalf("thread list = %#v", list)
	}

	var detail mailthread.ThreadDetailResponse
	assertParity(
		"hero_mail_thread_show",
		"/api/attention/v1/mail/threads/"+threadID+"?project_peer_id=peer_b",
		map[string]interface{}{"project_peer_id": "peer_b", "thread_id": threadID},
		&detail,
	)
	if detail.Error != nil || len(detail.Messages) != 1 || detail.Messages[0].Envelope.Body != "thread parity body" {
		t.Fatalf("thread detail = %#v", detail)
	}

	var contract mailthread.ContractResponse
	assertParity(
		"hero_mail_thread_contract",
		"/api/attention/v1/mail/thread-contract",
		map[string]interface{}{},
		&contract,
	)
	if err := mailthread.ValidateContractResponse(contract); err != nil {
		t.Fatalf("thread contract = %#v: %v", contract, err)
	}

	request := mailthread.ActionRequest{
		SchemaVersion:  mailthread.SchemaVersion,
		Identity:       list.Items[0].Identity,
		ActionID:       mailthread.ActionMarkRead,
		ThreadRevision: list.Items[0].Revision,
		IdempotencyKey: "mcp-http-action-parity",
	}
	body, _ := json.Marshal(request)
	httpAction := httptest.NewRecorder()
	api.Handler().ServeHTTP(httpAction, httptest.NewRequest(http.MethodPost, "/api/attention/v1/mail/thread-actions", bytes.NewReader(body)))
	if httpAction.Code != http.StatusOK {
		t.Fatalf("HTTP action = %d %s", httpAction.Code, httpAction.Body.String())
	}
	var args map[string]interface{}
	if err := json.Unmarshal(body, &args); err != nil {
		t.Fatal(err)
	}
	args["thread_revision"] = strconv.FormatInt(request.ThreadRevision, 10)
	mcpAction := callTool(t, server, "hero_mail_thread_action", args)
	var compactAction bytes.Buffer
	if err := json.Compact(&compactAction, httpAction.Body.Bytes()); err != nil {
		t.Fatal(err)
	}
	if mcpAction.IsError || len(mcpAction.Content) != 1 || compactAction.String() != mcpAction.Content[0].Text {
		t.Fatalf("action parity mismatch\nHTTP: %s\nMCP:  %#v", httpAction.Body.String(), mcpAction)
	}
}

func TestMCPMailThreadPagingAndStructuredErrors(t *testing.T) {
	api, store, _, _ := setupAttentionMailHTTP(t, "first")
	second := testMCPEnvelope("mail_second_thread")
	second.IdempotencyKey = "delivery_second_thread"
	second.Subject = "Second thread"
	second.CreatedAt = "2026-08-18T10:00:00Z"
	if _, _, err := store.Deliver(second, testMCPDelivery(second)); err != nil {
		t.Fatal(err)
	}
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	server.mailQueryService = api.mailQueryService

	firstText, err := server.toolHandlers()["hero_mail_thread_list"](map[string]interface{}{
		"project_peer_id": "peer_b", "limit": float64(1),
	})
	if err != nil {
		t.Fatal(err)
	}
	var first mailthread.ThreadListResponse
	if err := json.Unmarshal([]byte(firstText), &first); err != nil || first.Error != nil || first.NextCursor == "" || first.Page == nil || !first.Page.HasMore {
		t.Fatalf("first page = %s, %v", firstText, err)
	}
	secondText, err := server.toolHandlers()["hero_mail_thread_list"](map[string]interface{}{
		"project_peer_id": "peer_b", "limit": float64(1), "cursor": first.NextCursor,
	})
	if err != nil {
		t.Fatal(err)
	}
	var secondPage mailthread.ThreadListResponse
	if err := json.Unmarshal([]byte(secondText), &secondPage); err != nil || secondPage.Error != nil || len(secondPage.Items) != 1 || secondPage.Items[0].Identity == first.Items[0].Identity {
		t.Fatalf("second page = %s, %v", secondText, err)
	}

	staleText, _ := server.toolHandlers()["hero_mail_thread_list"](map[string]interface{}{
		"project_peer_id": "peer_b", "bucket": "history", "limit": float64(1), "cursor": first.NextCursor,
	})
	var stale mailthread.ThreadListResponse
	if err := json.Unmarshal([]byte(staleText), &stale); err != nil || stale.Error == nil || stale.Error.Code != attention.ErrorStale || stale.Error.Field != "cursor" {
		t.Fatalf("stale cursor = %s, %v", staleText, err)
	}

	invalidText, _ := server.toolHandlers()["hero_mail_thread_list"](map[string]interface{}{"limit": "not-an-int"})
	var invalid mailthread.ThreadListResponse
	if err := json.Unmarshal([]byte(invalidText), &invalid); err != nil || invalid.Error == nil || invalid.Error.Code != attention.ErrorValidation || invalid.Error.Field != "limit" {
		t.Fatalf("invalid limit = %s, %v", invalidText, err)
	}

	unavailableServer := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	unavailableServer.mailQueryService = func() (*mailquery.Service, error) { return nil, errors.New("registry offline") }
	unavailableText, _ := unavailableServer.toolHandlers()["hero_mail_thread_show"](map[string]interface{}{
		"project_peer_id": "peer_b", "thread_id": "mail_thread",
	})
	var unavailable mailthread.ThreadDetailResponse
	if err := json.Unmarshal([]byte(unavailableText), &unavailable); err != nil || unavailable.Error == nil || unavailable.Error.Code != attention.ErrorUnavailable {
		t.Fatalf("unavailable detail = %s, %v", unavailableText, err)
	}
}

func TestMCPMailThreadActionPreservesRevisionInputAndIdempotency(t *testing.T) {
	api, _, _, _ := setupAttentionMailHTTP(t, "lifecycle")
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	server.mailQueryService = api.mailQueryService

	listText, _ := server.toolMailThreadList(map[string]interface{}{"project_peer_id": "peer_b"})
	var list mailthread.ThreadListResponse
	if err := json.Unmarshal([]byte(listText), &list); err != nil || len(list.Items) != 1 {
		t.Fatalf("list = %s, %v", listText, err)
	}
	args := map[string]interface{}{
		"schema_version":  float64(mailthread.SchemaVersion),
		"identity":        map[string]interface{}{"project_peer_id": "peer_b", "thread_id": list.Items[0].Identity.ThreadID},
		"action_id":       mailthread.ActionResolve,
		"thread_revision": strconv.FormatInt(list.Items[0].Revision, 10),
		"idempotency_key": "mcp-resolve-replay",
		"input":           map[string]interface{}{"reason": "answered", "source": "user", "outcome": "answered"},
	}
	firstText, err := server.toolHandlers()["hero_mail_thread_action"](args)
	if err != nil {
		t.Fatal(err)
	}
	replayText, err := server.toolHandlers()["hero_mail_thread_action"](args)
	if err != nil || firstText != replayText {
		t.Fatalf("idempotent replay differs: %s / %s / %v", firstText, replayText, err)
	}
	var first mailthread.ActionResponse
	if err := json.Unmarshal([]byte(firstText), &first); err != nil || first.Error != nil || first.Thread == nil ||
		first.Thread.State.Lifecycle != mailthread.LifecycleResolved || first.Thread.State.Resolution == nil || first.Thread.State.Resolution.Reason != "answered" {
		t.Fatalf("resolve response = %s, %v", firstText, err)
	}

	conflictArgs := maps.Clone(args)
	conflictArgs["action_id"] = mailthread.ActionArchive
	delete(conflictArgs, "input")
	conflictText, _ := server.toolHandlers()["hero_mail_thread_action"](conflictArgs)
	var conflict mailthread.ActionResponse
	if err := json.Unmarshal([]byte(conflictText), &conflict); err != nil || conflict.Error == nil || conflict.Error.Code != attention.ErrorIdempotencyConflict {
		t.Fatalf("idempotency conflict = %s, %v", conflictText, err)
	}

	staleArgs := maps.Clone(args)
	staleArgs["action_id"] = mailthread.ActionReopen
	staleArgs["idempotency_key"] = "mcp-reopen-stale"
	delete(staleArgs, "input")
	staleText, _ := server.toolHandlers()["hero_mail_thread_action"](staleArgs)
	var stale mailthread.ActionResponse
	if err := json.Unmarshal([]byte(staleText), &stale); err != nil || stale.Error == nil || stale.Error.Code != attention.ErrorStale || stale.Error.Field != "thread_revision" {
		t.Fatalf("stale action = %s, %v", staleText, err)
	}
}
