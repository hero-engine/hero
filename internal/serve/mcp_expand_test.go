package serve

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestMCP_ReadSpec_Compact verifies that compact:true returns an
// envelope-only response while compact:false (or absent) returns the
// full body unchanged.
func TestMCP_ReadSpec_Compact(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	// Default behaviour — full body, no envelope (Phase 1 backwards compat).
	full := callTool(t, srv, "hero_read_spec", map[string]interface{}{
		"slug": "auth-login",
	})
	if full.IsError {
		t.Fatalf("read_spec full: %s", full.Content[0].Text)
	}
	if strings.Contains(full.Content[0].Text, "[hero envelope]") {
		t.Fatalf("default response must not contain envelope; got: %s", full.Content[0].Text)
	}
	if !strings.Contains(full.Content[0].Text, "Implement login flow") {
		t.Fatalf("default response must contain full body; got: %s", full.Content[0].Text)
	}

	// Compact — envelope only, no body.
	compact := callTool(t, srv, "hero_read_spec", map[string]interface{}{
		"slug":    "auth-login",
		"compact": true,
	})
	if compact.IsError {
		t.Fatalf("read_spec compact: %s", compact.Content[0].Text)
	}
	text := compact.Content[0].Text
	if !strings.Contains(text, "[hero envelope]") {
		t.Fatalf("compact response must contain envelope marker; got: %s", text)
	}
	if !strings.Contains(text, "ref_id: spec:auth-login:full") {
		t.Fatalf("compact response must contain stable ref_id for spec; got: %s", text)
	}
	if !strings.Contains(text, "expand_via: hero_expand") {
		t.Fatalf("compact response must point at hero_expand; got: %s", text)
	}
	// The summary may legitimately quote the spec's first prose line,
	// but the full body (frontmatter + "## Changes" section) must be
	// absent — that's what compact actually trims.
	if strings.Contains(text, "## Changes") {
		t.Fatalf("compact response must NOT contain full body sections; got: %s", text)
	}
	if strings.Contains(text, "src/auth/oauth.go") {
		t.Fatalf("compact response must NOT contain full body details; got: %s", text)
	}
}

// TestMCP_Expand_Single verifies that hero_expand resolves a previously-
// registered spec ref to its full content.
func TestMCP_Expand_Single(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	// Register a ref by calling read_spec compact.
	_ = callTool(t, srv, "hero_read_spec", map[string]interface{}{
		"slug":    "auth-login",
		"compact": true,
	})

	// Now expand.
	expanded := callTool(t, srv, "hero_expand", map[string]interface{}{
		"ref_id": "spec:auth-login:full",
	})
	if expanded.IsError {
		t.Fatalf("expand error: %s", expanded.Content[0].Text)
	}
	text := expanded.Content[0].Text
	if !strings.Contains(text, "Implement login flow") {
		t.Fatalf("expanded content missing full body; got: %s", text)
	}
}

// TestMCP_Expand_Batch verifies that hero_expand accepts an array of
// ref ids and returns each in order with delimiters.
func TestMCP_Expand_Batch(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	_ = callTool(t, srv, "hero_read_spec", map[string]interface{}{
		"slug":    "auth-login",
		"compact": true,
	})

	// Two refs; one valid, one unknown — batch should return both.
	expanded := callTool(t, srv, "hero_expand", map[string]interface{}{
		"ref_ids": []interface{}{"spec:auth-login:full", "spec:nonexistent:full"},
	})
	if expanded.IsError {
		t.Fatalf("batch expand error: %s", expanded.Content[0].Text)
	}
	text := expanded.Content[0].Text
	if !strings.Contains(text, "Implement login flow") {
		t.Fatalf("batch should include known content; got: %s", text)
	}
	if !strings.Contains(text, "[hero expand error]") {
		t.Fatalf("batch should include error block for unknown ref; got: %s", text)
	}
	if !strings.Contains(text, "rehydrate_via: hero_read_spec slug=nonexistent") {
		t.Fatalf("error block should point caller at re-fetch; got: %s", text)
	}
	// Both refs delimited.
	if !strings.Contains(text, "[ref 1/2:") || !strings.Contains(text, "[ref 2/2:") {
		t.Fatalf("batch should be delimited; got: %s", text)
	}
}

// TestMCP_Expand_Unknown verifies the error-block shape for an
// unregistered ref id.
func TestMCP_Expand_Unknown(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	expanded := callTool(t, srv, "hero_expand", map[string]interface{}{
		"ref_id": "spec:auth-login:full",
	})
	if expanded.IsError {
		t.Fatalf("expand call failed: %s", expanded.Content[0].Text)
	}
	text := expanded.Content[0].Text
	if !strings.Contains(text, "[hero expand error]") {
		t.Fatalf("unknown ref should produce structured error; got: %s", text)
	}
	if !strings.Contains(text, "unknown or expired ref") {
		t.Fatalf("error message should be descriptive; got: %s", text)
	}
	if !strings.Contains(text, "rehydrate_via: hero_read_spec slug=auth-login") {
		t.Fatalf("error should suggest rehydrate path; got: %s", text)
	}
}

// TestMCP_Expand_NoArgs verifies hero_expand returns a clear error
// when called without ref_id or ref_ids.
func TestMCP_Expand_NoArgs(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	expanded := callTool(t, srv, "hero_expand", map[string]interface{}{})
	if !expanded.IsError {
		t.Fatalf("expected error when no ref ids supplied")
	}
	if !strings.Contains(expanded.Content[0].Text, "ref_id or ref_ids") {
		t.Fatalf("expected helpful error; got: %s", expanded.Content[0].Text)
	}
}

// TestMCP_Search_Compact verifies the search tool emits an envelope
// with a query-scoped ref id when compact:true is set.
func TestMCP_Search_Compact(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	res := callTool(t, srv, "hero_search", map[string]interface{}{
		"query":   "auth",
		"compact": true,
	})
	if res.IsError {
		t.Fatalf("search compact: %s", res.Content[0].Text)
	}
	text := res.Content[0].Text
	if !strings.Contains(text, "[hero envelope]") {
		t.Fatalf("search compact must emit envelope; got: %s", text)
	}
	if !strings.HasPrefix(parseRefIDFromText(text), "search:") {
		t.Fatalf("search compact must emit a search:* ref id; got envelope: %s", text)
	}
	if strings.Contains(text, "kind: spec") {
		t.Fatalf("search ref must not be classified as spec kind; got: %s", text)
	}
}

// TestMCP_ToolsList_IncludesExpand verifies hero_expand is advertised.
func TestMCP_ToolsList_IncludesExpand(t *testing.T) {
	heroDir, _ := setupTestWorkspace(t)
	srv := NewMCPServer(heroDir, filepath.Dir(heroDir), "1.0.0")

	defs := srv.toolDefinitions()
	for _, d := range defs {
		if d.Name == "hero_expand" {
			if !strings.Contains(d.Description, "ref_id") {
				t.Fatalf("hero_expand description should mention ref_id; got: %s", d.Description)
			}
			return
		}
	}
	t.Fatalf("hero_expand missing from tool definitions")
}

// parseRefIDFromText extracts the ref_id field from an envelope block.
// Test-only helper.
func parseRefIDFromText(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ref_id:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "ref_id:"))
		}
	}
	return ""
}
