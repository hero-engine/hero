package serve

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestMCPMailDefinitionsAndDispatch(t *testing.T) {
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	for _, name := range []string{"hero_mail_list", "hero_mail_show", "hero_mail_action"} {
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
