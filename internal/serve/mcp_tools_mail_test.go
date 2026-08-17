package serve

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/mail"
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
	for _, name := range []string{"hero_mail_list", "hero_mail_show", "hero_mail_send", "hero_mail_reply", "hero_mail_action"} {
		if _, ok := server.toolHandlers()[name]; !ok {
			t.Errorf("missing handler %s", name)
		}
		found := false
		for _, definition := range server.toolDefinitions() {
			if definition.Name == name {
				found = true
			}
		}
		if !found {
			t.Errorf("missing definition %s", name)
		}
	}
}
