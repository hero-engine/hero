package serve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/spec"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// sendRecv sends one JSON-RPC request and returns the decoded response.
func sendRecv(t *testing.T, srv *MCPServer, req JSONRPCRequest) JSONRPCResponse {
	t.Helper()

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	in := bytes.NewBufferString(string(data) + "\n")
	var out bytes.Buffer
	srv.SetIO(in, &out)
	if err := srv.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, out.String())
	}
	return resp
}

// sendMulti sends multiple requests (newline-separated) and returns all responses.
func sendMulti(t *testing.T, srv *MCPServer, reqs ...JSONRPCRequest) []JSONRPCResponse {
	t.Helper()

	var input strings.Builder
	for _, req := range reqs {
		data, _ := json.Marshal(req)
		input.Write(data)
		input.WriteByte('\n')
	}

	in := bytes.NewBufferString(input.String())
	var out bytes.Buffer
	srv.SetIO(in, &out)
	if err := srv.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var resps []JSONRPCResponse
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var resp JSONRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("unmarshal response line: %v\nraw: %s", err, line)
		}
		resps = append(resps, resp)
	}
	return resps
}

// setupTestWorkspace creates a temp dir with .hero, a spec file, and an index.
// Returns heroDir, projectRoot, cleanup func.
func setupTestWorkspace(t *testing.T) (string, string) {
	t.Helper()
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, ".hero")
	specsDir := filepath.Join(heroDir, "specs", "auth-login")
	knowledgeDir := filepath.Join(heroDir, "knowledge", "conventions", "naming")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a spec
	specPath := filepath.Join(specsDir, "spec.md")
	specContent := `---
title: Auth Login
type: feature
status: delivering
claimed-by: alice
tags: auth, security
---

# Auth Login

Implement login flow with OAuth2 support.

## Changes

- src/auth/login.go
- src/auth/oauth.go
`
	if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a convention
	convPath := filepath.Join(knowledgeDir, "spec.md")
	convContent := `---
title: Naming Convention
type: convention
status: active
scope:
  - "src/**"
---

# Naming Convention

Use camelCase for variables.
`
	if err := os.WriteFile(convPath, []byte(convContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Build the index
	idx, err := index.Open(heroDir)
	if err != nil {
		t.Fatal(err)
	}

	sp := &spec.Spec{
		Slug:       "auth-login",
		Title:      "Auth Login",
		Type:       spec.TypeFeature,
		Status:     spec.StatusDelivering,
		Path:       specPath,
		ClaimedBy:  "alice",
		Tags:       []string{"auth", "security"},
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
		Sections: map[string]string{
			"changes": "- src/auth/login.go\n- src/auth/oauth.go",
		},
		FilesTouched: []string{"src/auth/login.go", "src/auth/oauth.go"},
	}
	if err := idx.IndexSpec(sp, specContent); err != nil {
		idx.Close()
		t.Fatal(err)
	}

	conv := &spec.Spec{
		Slug:       "naming",
		Title:      "Naming Convention",
		Type:       spec.TypeConvention,
		Status:     spec.StatusActive,
		Path:       convPath,
		Scope:      []string{"src/**"},
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
		Sections:   map[string]string{},
	}
	if err := idx.IndexSpec(conv, convContent); err != nil {
		idx.Close()
		t.Fatal(err)
	}

	idx.Close()

	// Write hero.json so config.Load works
	configPath := filepath.Join(tmpDir, "hero.json")
	configContent := `{"directory": ".hero"}`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	return heroDir, tmpDir
}

func rawID(id int) json.RawMessage {
	data, _ := json.Marshal(id)
	return data
}

// ---------------------------------------------------------------------------
// Protocol tests
// ---------------------------------------------------------------------------

func TestMCP_Initialize(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0-test")

	resp := sendRecv(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      rawID(1),
		Method:  "initialize",
	})

	if resp.Error != nil {
		t.Fatalf("expected no error, got: %s", resp.Error.Message)
	}

	// Decode result
	resultBytes, _ := json.Marshal(resp.Result)
	var result InitializeResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}

	if result.ProtocolVersion != "2025-03-26" {
		t.Errorf("protocol version = %q, want 2025-03-26", result.ProtocolVersion)
	}
	if result.ServerInfo.Name != "hero" {
		t.Errorf("server name = %q, want hero", result.ServerInfo.Name)
	}
	if result.ServerInfo.Version != "1.0.0-test" {
		t.Errorf("server version = %q, want 1.0.0-test", result.ServerInfo.Version)
	}
	if result.Capabilities.Tools == nil {
		t.Error("expected tools capability to be present")
	}
	if result.Instructions == "" {
		t.Error("expected non-empty instructions")
	}
}

func TestMCP_ToolsList(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	resp := sendRecv(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      rawID(2),
		Method:  "tools/list",
	})

	if resp.Error != nil {
		t.Fatalf("expected no error, got: %s", resp.Error.Message)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var result ToolsListResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}

	if len(result.Tools) != 42 {
		t.Errorf("expected 42 tools, got %d", len(result.Tools))
	}

	expectedNames := map[string]bool{
		"hero_context": true, "hero_search": true, "hero_status": true,
		"hero_check": true, "hero_nudge": true, "hero_list": true,
		"hero_queue": true, "hero_kickoff": true,
		"hero_knowledge": true, "hero_read_spec": true,
		"hero_ask": true, "hero_anchor": true,
		"hero_pulse": true, "hero_skill_run": true,
		"hero_claim": true, "hero_velocity": true,
		"hero_test_generate": true, "hero_demo_record": true,
		"hero_code": true,
		"hero_error_pattern": true,
		"hero_enrich": true,
		"hero_diagnose": true,
		"hero_score": true,
		"hero_verify": true,
		"hero_conflicts": true,
		"hero_sequence": true,
		"hero_warnings": true,
		"hero_insights": true,
		"hero_drift": true,
		"hero_plan": true,
		"hero_contract": true,
		"hero_impact": true,
		"hero_recap": true,
		"hero_active": true,
		"hero_coverage": true,
		"hero_ci": true,
		"hero_feed": true,
		"hero_event": true,
		"hero_why": true,
		"hero_blocked": true,
		"hero_expand": true,
		"hero_snapshot": true,
	}
	for _, tool := range result.Tools {
		if !expectedNames[tool.Name] {
			t.Errorf("unexpected tool: %s", tool.Name)
		}
		if tool.Description == "" {
			t.Errorf("tool %s has empty description", tool.Name)
		}
		if tool.InputSchema.Type != "object" {
			t.Errorf("tool %s input schema type = %q, want object", tool.Name, tool.InputSchema.Type)
		}
	}
}

func TestMCP_Ping(t *testing.T) {
	srv := NewMCPServer(t.TempDir(), t.TempDir(), "1.0.0")

	resp := sendRecv(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      rawID(3),
		Method:  "ping",
	})

	if resp.Error != nil {
		t.Fatalf("expected no error, got: %s", resp.Error.Message)
	}
}

func TestMCP_UnknownMethod(t *testing.T) {
	srv := NewMCPServer(t.TempDir(), t.TempDir(), "1.0.0")

	resp := sendRecv(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      rawID(4),
		Method:  "foobar/unknown",
	})

	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("error code = %d, want %d", resp.Error.Code, ErrCodeMethodNotFound)
	}
}

func TestMCP_ParseError(t *testing.T) {
	srv := NewMCPServer(t.TempDir(), t.TempDir(), "1.0.0")
	in := bytes.NewBufferString("this is not json\n")
	var out bytes.Buffer
	srv.SetIO(in, &out)
	if err := srv.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v\nraw: %s", err, out.String())
	}

	if resp.Error == nil {
		t.Fatal("expected parse error")
	}
	if resp.Error.Code != ErrCodeParse {
		t.Errorf("error code = %d, want %d", resp.Error.Code, ErrCodeParse)
	}
}

func TestMCP_BlankLinesIgnored(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	req, _ := json.Marshal(JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      rawID(1),
		Method:  "ping",
	})
	// Include blank lines around the request
	input := "\n\n" + string(req) + "\n\n"

	in := bytes.NewBufferString(input)
	var out bytes.Buffer
	srv.SetIO(in, &out)
	srv.Run()

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 response line, got %d: %v", len(lines), lines)
	}
}

func TestMCP_NotificationNoResponse(t *testing.T) {
	srv := NewMCPServer(t.TempDir(), t.TempDir(), "1.0.0")

	req, _ := json.Marshal(JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	})
	in := bytes.NewBufferString(string(req) + "\n")
	var out bytes.Buffer
	srv.SetIO(in, &out)
	srv.Run()

	if out.String() != "" {
		t.Errorf("expected no response for notification, got: %s", out.String())
	}
}

func TestMCP_MultipleRequests(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	resps := sendMulti(t, srv,
		JSONRPCRequest{JSONRPC: "2.0", ID: rawID(1), Method: "initialize"},
		JSONRPCRequest{JSONRPC: "2.0", Method: "notifications/initialized"},
		JSONRPCRequest{JSONRPC: "2.0", ID: rawID(2), Method: "tools/list"},
		JSONRPCRequest{JSONRPC: "2.0", ID: rawID(3), Method: "ping"},
	)

	// notification produces no response, so we get 3 responses
	if len(resps) != 3 {
		t.Fatalf("expected 3 responses, got %d", len(resps))
	}
	for _, r := range resps {
		if r.Error != nil {
			t.Errorf("unexpected error: %s", r.Error.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// Tool call tests
// ---------------------------------------------------------------------------

func callTool(t *testing.T, srv *MCPServer, name string, args map[string]interface{}) ToolCallResult {
	t.Helper()
	argsJSON, _ := json.Marshal(ToolCallParams{Name: name, Arguments: args})

	resp := sendRecv(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      rawID(100),
		Method:  "tools/call",
		Params:  argsJSON,
	})

	if resp.Error != nil {
		t.Fatalf("tools/call error: %s", resp.Error.Message)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var result ToolCallResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	return result
}

func TestMCP_ToolCall_Search(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_search", map[string]interface{}{
		"query": "auth login",
	})

	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content")
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "auth-login") {
		t.Errorf("search result should contain 'auth-login', got: %s", text)
	}
}

func TestMCP_ToolCall_Search_NoQuery(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_search", map[string]interface{}{})

	if !result.IsError {
		t.Fatal("expected error for missing query")
	}
	if !strings.Contains(result.Content[0].Text, "query parameter is required") {
		t.Errorf("unexpected error: %s", result.Content[0].Text)
	}
}

func TestMCP_ToolCall_Status(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_status", nil)

	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "auth-login") {
		t.Errorf("status should contain auth-login, got: %s", text)
	}
}

func TestMCP_ToolCall_Check(t *testing.T) {
	heroDir, projectRoot := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, projectRoot, "1.0.0")

	result := callTool(t, srv, "hero_check", nil)

	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "health check") {
		t.Errorf("check output should contain 'health check', got: %s", text)
	}
}

func TestMCP_ToolCall_List(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_list", map[string]interface{}{
		"type": "feature",
	})

	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "auth-login") {
		t.Errorf("list should contain auth-login, got: %s", text)
	}
}

func TestMCP_ToolCall_List_NoMatch(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_list", map[string]interface{}{
		"type": "bug",
	})

	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "No specs found") {
		t.Errorf("expected 'No specs found' for bug type, got: %s", text)
	}
}

func TestMCP_ToolCall_Knowledge(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_knowledge", nil)

	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "naming") || !strings.Contains(text, "Convention") {
		t.Errorf("knowledge should contain naming convention, got: %s", text)
	}
}

func TestMCP_ToolCall_Knowledge_TypeFilter(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_knowledge", map[string]interface{}{
		"type": "decision",
	})

	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "No decision entries") {
		t.Errorf("expected 'No decision entries', got: %s", text)
	}
}

func TestMCP_ToolCall_ReadSpec(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_read_spec", map[string]interface{}{
		"slug": "auth-login",
	})

	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "Auth Login") {
		t.Errorf("read_spec should return spec content, got: %s", text)
	}
	if !strings.Contains(text, "OAuth2") {
		t.Errorf("read_spec should contain OAuth2, got: %s", text)
	}
}

func TestMCP_ToolCall_ReadSpec_NotFound(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_read_spec", map[string]interface{}{
		"slug": "nonexistent-spec",
	})

	if result.IsError {
		t.Fatalf("expected non-error result with not-found message, got error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "not found") {
		t.Errorf("expected 'not found' message, got: %s", text)
	}
}

func TestMCP_ToolCall_ReadSpec_NoSlug(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_read_spec", map[string]interface{}{})

	if !result.IsError {
		t.Fatal("expected error for missing slug")
	}
}

func TestMCP_ToolCall_Context(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_context", map[string]interface{}{
		"files": "src/auth/login.go",
	})

	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}
	// The context block should have at least the convention (naming matches src/**)
	// and the past work (auth-login touches src/auth/login.go)
	text := result.Content[0].Text
	// Even if there's no match, the tool should return something (could be "No relevant context")
	if text == "" {
		t.Error("expected non-empty context output")
	}
}

func TestMCP_ToolCall_Context_NoFiles(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_context", map[string]interface{}{})

	if !result.IsError {
		t.Fatal("expected error for missing files")
	}
}

func TestMCP_ToolCall_Nudge(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_nudge", map[string]interface{}{
		"files": "src/auth/login.go",
	})

	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if text == "" {
		t.Error("expected non-empty nudge output")
	}
}

func TestMCP_ToolCall_Nudge_NoFiles(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	result := callTool(t, srv, "hero_nudge", map[string]interface{}{})

	if !result.IsError {
		t.Fatal("expected error for missing files")
	}
}

func TestMCP_ToolCall_UnknownTool(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	argsJSON, _ := json.Marshal(ToolCallParams{Name: "hero_nonexistent", Arguments: nil})
	resp := sendRecv(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      rawID(100),
		Method:  "tools/call",
		Params:  argsJSON,
	})

	if resp.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
	if resp.Error.Code != ErrCodeInvalidParams {
		t.Errorf("error code = %d, want %d", resp.Error.Code, ErrCodeInvalidParams)
	}
}

func TestMCP_ToolCall_InvalidParams(t *testing.T) {
	srv := NewMCPServer(t.TempDir(), t.TempDir(), "1.0.0")

	resp := sendRecv(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      rawID(100),
		Method:  "tools/call",
		Params:  json.RawMessage(`"not an object"`),
	})

	if resp.Error == nil {
		t.Fatal("expected error for invalid params")
	}
	if resp.Error.Code != ErrCodeInvalidParams {
		t.Errorf("error code = %d, want %d", resp.Error.Code, ErrCodeInvalidParams)
	}
}

// ---------------------------------------------------------------------------
// Format helper tests
// ---------------------------------------------------------------------------

func TestFormatSearchResults(t *testing.T) {
	results := []index.SearchResult{
		{Slug: "auth-login", Title: "Auth Login", Type: spec.TypeFeature, Status: spec.StatusDelivering, ClaimedBy: "alice", Snippet: "login flow"},
		{Slug: "csv-export", Title: "CSV Export", Type: spec.TypeFeature, Status: spec.StatusCompleted},
	}

	out := formatSearchResults(results)
	if !strings.Contains(out, "auth-login") {
		t.Errorf("should contain auth-login: %s", out)
	}
	if !strings.Contains(out, "[alice]") {
		t.Errorf("should contain [alice]: %s", out)
	}
	if !strings.Contains(out, "2 result(s)") {
		t.Errorf("should contain '2 result(s)': %s", out)
	}
}

func TestFormatContextBlock_Empty(t *testing.T) {
	cb := &index.ContextBlock{}
	if !cb.IsEmpty() {
		t.Error("empty context block should be empty")
	}
}

func TestFormatStatusOutput(t *testing.T) {
	specs := []*spec.Spec{
		{Slug: "s1", Title: "Feature One", Type: spec.TypeFeature, Status: spec.StatusDelivering},
		{Slug: "s2", Title: "Feature Two", Type: spec.TypeFeature, Status: spec.StatusPlanning},
		{Slug: "c1", Title: "Convention One", Type: spec.TypeConvention, Status: spec.StatusActive},
	}

	out := formatStatusOutput(specs)
	if !strings.Contains(out, "Delivering (1)") {
		t.Errorf("should contain 'Delivering (1)': %s", out)
	}
	if !strings.Contains(out, "Planning (1)") {
		t.Errorf("should contain 'Planning (1)': %s", out)
	}
	if !strings.Contains(out, "Knowledge (1)") {
		t.Errorf("should contain 'Knowledge (1)': %s", out)
	}
	if !strings.Contains(out, "2 in-flight") {
		t.Errorf("should contain '2 in-flight': %s", out)
	}
}

func TestFormatKnowledgeOutput(t *testing.T) {
	entries := []*spec.Spec{
		{Slug: "naming", Title: "Naming Convention", Type: spec.TypeConvention, Status: spec.StatusActive},
		{Slug: "api-note", Title: "API Note", Type: spec.TypeNote, Status: spec.StatusDraft},
	}

	out := formatKnowledgeOutput(entries)
	if !strings.Contains(out, "Conventions (1)") {
		t.Errorf("should contain 'Conventions (1)': %s", out)
	}
	if !strings.Contains(out, "Notes (1)") {
		t.Errorf("should contain 'Notes (1)': %s", out)
	}
	if !strings.Contains(out, "Total: 2") {
		t.Errorf("should contain 'Total: 2': %s", out)
	}
}

// ---------------------------------------------------------------------------
// hero_kickoff / hero_queue / hero_list (kickoff-prompts-queue feature)
// ---------------------------------------------------------------------------

// addKickoffSpec writes a spec with a `## Kickoff` body into the workspace
// at .hero/specs/<slug>/spec.md.
func addKickoffSpec(t *testing.T, heroDir, slug, title, kickoffBody string, status spec.Status, pinned bool) {
	t.Helper()
	dir := filepath.Join(heroDir, "specs", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	pinLine := ""
	if pinned {
		pinLine = "pinned: true\n"
	}
	body := fmt.Sprintf(`---
title: %s
type: feature
status: %s
%s---

## Kickoff

%s

## Goal

Body content.
`, title, status, pinLine, kickoffBody)
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestMCP_ToolCall_Kickoff_ReturnsBody(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	addKickoffSpec(t, heroDir, "demo-spec", "Demo", "Paste me into a fresh session.", spec.StatusPlanning, false)

	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")
	result := callTool(t, srv, "hero_kickoff", map[string]interface{}{"slug": "demo-spec"})

	if result.IsError {
		t.Fatalf("hero_kickoff error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "Paste me into a fresh session.") {
		t.Errorf("expected kickoff body, got: %s", text)
	}
	if !strings.Contains(text, "demo-spec") {
		t.Errorf("expected slug header, got: %s", text)
	}
}

func TestMCP_ToolCall_Kickoff_MissingSlug(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")
	result := callTool(t, srv, "hero_kickoff", map[string]interface{}{})
	if !result.IsError {
		t.Fatal("expected error when slug is missing")
	}
}

func TestMCP_ToolCall_Kickoff_NotFound(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")
	result := callTool(t, srv, "hero_kickoff", map[string]interface{}{"slug": "nope-not-here"})
	if !result.IsError {
		t.Fatal("expected error when slug doesn't exist")
	}
}

func TestMCP_ToolCall_Kickoff_NoSection(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	// auth-login from setupTestWorkspace has no `## Kickoff` section.
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")
	result := callTool(t, srv, "hero_kickoff", map[string]interface{}{"slug": "auth-login"})
	if result.IsError {
		t.Fatalf("missing kickoff should not be an error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "no `## Kickoff` section") {
		t.Errorf("expected hint about missing section, got: %s", result.Content[0].Text)
	}
}

func TestMCP_ToolCall_Queue_PrioritizesPinned(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	addKickoffSpec(t, heroDir, "regular-spec", "Regular", "regular kickoff", spec.StatusPlanning, false)
	addKickoffSpec(t, heroDir, "pinned-spec", "Pinned", "pinned kickoff", spec.StatusPlanning, true)

	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")
	result := callTool(t, srv, "hero_queue", map[string]interface{}{"format": "text"})

	if result.IsError {
		t.Fatalf("hero_queue error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	pinIdx := strings.Index(text, "pinned-spec")
	regIdx := strings.Index(text, "regular-spec")
	if pinIdx < 0 || regIdx < 0 {
		t.Fatalf("missing slugs in output:\n%s", text)
	}
	if pinIdx > regIdx {
		t.Errorf("pinned should rank ahead of regular:\n%s", text)
	}
}

func TestMCP_ToolCall_Queue_DefaultFormatKickoff(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	addKickoffSpec(t, heroDir, "demo", "Demo", "kickoff body content here", spec.StatusPlanning, false)

	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")
	result := callTool(t, srv, "hero_queue", nil)

	if result.IsError {
		t.Fatalf("hero_queue error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "kickoff body content here") {
		t.Errorf("default format should be kickoff and render body, got:\n%s", result.Content[0].Text)
	}
}

func TestMCP_ToolCall_List_RichFilters(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	addKickoffSpec(t, heroDir, "kickoff-spec", "Kickoff Demo", "kickoff body", spec.StatusPlanning, false)

	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")
	result := callTool(t, srv, "hero_list", map[string]interface{}{
		"type":   "feature",
		"ready":  "true",
		"format": "kickoff",
	})

	if result.IsError {
		t.Fatalf("hero_list error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "kickoff-spec") {
		t.Errorf("expected kickoff-spec in ready output, got:\n%s", text)
	}
	if !strings.Contains(text, "kickoff body") {
		t.Errorf("expected kickoff body when format=kickoff, got:\n%s", text)
	}
}

// ---------------------------------------------------------------------------
// hero_pulse / hero_kickoff ambient size-drift surface
// (roadmap-review-ambient-surfacing)
// ---------------------------------------------------------------------------

// writeDriftedLeafSpec writes a minimal feature spec under
// .hero/planning/features/<slug>/ whose declared `size: trivial` will
// not match its computed bucket (lots of files), triggering leaf drift.
func writeDriftedLeafSpec(t *testing.T, heroDir, slug string) {
	t.Helper()
	dir := filepath.Join(heroDir, "planning", "features", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	files := make([]string, 10)
	for i := range files {
		files[i] = fmt.Sprintf("    - `path/to/file_%d.go`", i)
	}
	body := "---\ntitle: Drifted\ntype: feature\nstatus: planning\nsize: trivial\n---\n\n## Kickoff\n\nKick off content.\n\n## Changes\n\n" + strings.Join(files, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// TestMCP_ToolCall_Pulse_SizeDrift_PresentWhenActive covers the
// non-quiet path: with a drifted spec under planning/, the pulse JSON
// response includes a top-level `size_drift` object with count + hint.
// We force the active-spec rule by calling toolKickoff first against
// the drifted slug — but pulse itself doesn't accept an active-spec
// arg, so we rely on rule 3 (initiative without size) instead.
func TestMCP_ToolCall_Pulse_SizeDrift_PresentWhenInitiativeUnsized(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	// Initiative with horizon=now and no declared size, plus a sized
	// child so the container rollup is determinate.
	initDir := filepath.Join(heroDir, "planning", "initiatives", "big-thing")
	if err := os.MkdirAll(initDir, 0o755); err != nil {
		t.Fatal(err)
	}
	initBody := "---\ntitle: Big thing\ntype: initiative\nstatus: planning\nhorizon: now\n---\n\n## Goal\n\nBig.\n"
	if err := os.WriteFile(filepath.Join(initDir, "spec.md"), []byte(initBody), 0o644); err != nil {
		t.Fatal(err)
	}
	childDir := filepath.Join(heroDir, "planning", "features", "child-a")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := make([]string, 8)
	for i := range files {
		files[i] = fmt.Sprintf("    - `child/file_%d.go`", i)
	}
	childBody := "---\ntitle: Child A\ntype: feature\nstatus: planning\nsize: medium\nparent: big-thing\n---\n\n## Changes\n\n" + strings.Join(files, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(childDir, "spec.md"), []byte(childBody), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")
	result := callTool(t, srv, "hero_pulse", map[string]interface{}{
		"format": "json",
	})
	if result.IsError {
		t.Fatalf("hero_pulse error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, `"size_drift"`) {
		t.Errorf("expected size_drift field in JSON response, got:\n%s", text)
	}
	if !strings.Contains(text, "/roadmap-review") {
		t.Errorf("expected /roadmap-review CTA in size_drift hint, got:\n%s", text)
	}
}

// TestMCP_ToolCall_Pulse_SizeDrift_AbsentWhenQuiet covers the quiet
// path: with no drift, the JSON response omits the size_drift field
// (`omitempty`). The auth-login spec from setupTestWorkspace has no
// declared size, so no leaf drift fires; no initiatives present.
func TestMCP_ToolCall_Pulse_SizeDrift_AbsentWhenQuiet(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")
	result := callTool(t, srv, "hero_pulse", map[string]interface{}{
		"format": "json",
	})
	if result.IsError {
		t.Fatalf("hero_pulse error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if strings.Contains(text, `"size_drift"`) {
		t.Errorf("expected no size_drift field when quiet, got:\n%s", text)
	}
}

// TestMCP_ToolCall_Kickoff_SizeDriftPrefix covers the active-spec rule
// in hero_kickoff: a drifted spec passed as the slug triggers the
// ambient hint as a leading line above the kickoff body.
func TestMCP_ToolCall_Kickoff_SizeDriftPrefix(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	writeDriftedLeafSpec(t, heroDir, "drifted-feature")

	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")
	result := callTool(t, srv, "hero_kickoff", map[string]interface{}{
		"slug": "drifted-feature",
	})
	if result.IsError {
		t.Fatalf("hero_kickoff error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "size drift") {
		t.Errorf("expected ambient size-drift hint in kickoff output, got:\n%s", text)
	}
	if !strings.Contains(text, "/roadmap-review") {
		t.Errorf("expected /roadmap-review CTA, got:\n%s", text)
	}
	if !strings.Contains(text, "Kick off content.") {
		t.Errorf("expected kickoff body to follow, got:\n%s", text)
	}
	// Ordering: hint appears BEFORE the kickoff header.
	hintIdx := strings.Index(text, "size drift")
	headerIdx := strings.Index(text, "## drifted-feature")
	if hintIdx < 0 || headerIdx < 0 || hintIdx > headerIdx {
		t.Errorf("expected ambient hint before kickoff header (hint=%d header=%d), got:\n%s", hintIdx, headerIdx, text)
	}
}

// TestMCP_ToolCall_Kickoff_NoPrefixWhenQuiet covers the kickoff path
// when no ambient drift surfaces — the body is unchanged.
func TestMCP_ToolCall_Kickoff_NoPrefixWhenQuiet(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	addKickoffSpec(t, heroDir, "clean-spec", "Clean", "Clean kickoff body.", spec.StatusPlanning, false)

	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")
	result := callTool(t, srv, "hero_kickoff", map[string]interface{}{
		"slug": "clean-spec",
	})
	if result.IsError {
		t.Fatalf("hero_kickoff error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if strings.Contains(text, "size drift") {
		t.Errorf("expected no size-drift line when quiet, got:\n%s", text)
	}
	if !strings.Contains(text, "Clean kickoff body.") {
		t.Errorf("expected kickoff body in output, got:\n%s", text)
	}
}

func TestMCP_ToolCall_List_ReadyAndBlockedExclusive(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")
	result := callTool(t, srv, "hero_list", map[string]interface{}{
		"ready":   "true",
		"blocked": "true",
	})
	if !result.IsError {
		t.Fatal("expected error when both ready and blocked are true")
	}
}
