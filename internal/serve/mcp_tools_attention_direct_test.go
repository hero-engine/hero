package serve

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/config"
)

func TestDirectAttentionDefinitionsUseCanonicalPolicyMetadata(t *testing.T) {
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	byName := make(map[string]ToolDefinition)
	for _, definition := range server.toolDefinitions() {
		byName[definition.Name] = definition
	}
	for _, operationID := range []string{attention.OperationMailSend, attention.OperationMailReply, attention.OperationFocusCreate} {
		policy, _ := attention.OperationPolicyByID(operationID)
		definition, ok := byName[policy.ToolName]
		if !ok {
			t.Fatalf("missing definition for %s", operationID)
		}
		if definition.Annotations == nil || definition.Annotations.ReadOnlyHint == nil || *definition.Annotations.ReadOnlyHint ||
			definition.Annotations.DestructiveHint == nil || *definition.Annotations.DestructiveHint ||
			definition.Annotations.IdempotentHint == nil || *definition.Annotations.IdempotentHint != policy.ReplaySafe ||
			definition.Annotations.OpenWorldHint == nil || *definition.Annotations.OpenWorldHint != policy.OpenWorld {
			t.Fatalf("%s annotations = %#v", operationID, definition.Annotations)
		}
		if definition.Meta["hero.dev/operation_id"] != policy.ID ||
			definition.Meta["hero.dev/effect"] != string(policy.Effect) ||
			definition.Meta["hero.dev/consent"] != string(policy.Consent) {
			t.Fatalf("%s metadata = %#v", operationID, definition.Meta)
		}
	}
}

func TestDirectAttentionHandlersDoNotAccessStoreFilesOrProcesses(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	dir := filepath.Dir(current)
	for _, name := range []string{"mcp_tools_attention_direct.go", "mcp_tools_mail.go", "mcp_tools_focus.go"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"os.ReadFile", "os.WriteFile", "os.OpenFile", "exec.Command"} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("%s contains direct file/process boundary %q", name, forbidden)
			}
		}
	}
}

func TestDirectAttentionToolsRequireExplicitProfileInclusion(t *testing.T) {
	filter := NewToolFilter(&config.MCPToolFilter{Profiles: map[string][]string{
		"attention-read":  {"hero_attention_snapshot", "hero_mail_list", "hero_mail_show"},
		"attention-write": {"hero_mail_send", "hero_mail_reply", "hero_focus_create"},
	}})
	server := NewMCPServerWithFilter(t.TempDir(), t.TempDir(), "test", filter)
	for _, testCase := range []struct {
		profile string
		want    map[string]bool
	}{
		{"attention-read", map[string]bool{}},
		{"attention-write", map[string]bool{"hero_mail_send": true, "hero_mail_reply": true, "hero_focus_create": true}},
	} {
		tools := filter.FilterTools(server.toolDefinitions(), testCase.profile)
		found := map[string]bool{}
		for _, definition := range tools {
			if definition.Name == "hero_mail_send" || definition.Name == "hero_mail_reply" || definition.Name == "hero_focus_create" {
				found[definition.Name] = true
			}
		}
		if len(found) != len(testCase.want) {
			t.Fatalf("%s direct tools = %#v", testCase.profile, found)
		}
	}

	server.profile = "attention-read"
	var output bytes.Buffer
	server.output = &output
	params, _ := json.Marshal(ToolCallParams{Name: "hero_focus_create", Arguments: map[string]interface{}{}})
	server.handleToolsCall(&JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1"), Params: params})
	var response JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || response.Error == nil || response.Error.Code != ErrCodeMethodNotFound {
		t.Fatalf("filtered dispatch = %s, %v", output.String(), err)
	}
}
