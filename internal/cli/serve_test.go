package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeCmd_Help(t *testing.T) {
	output, err := runCmd("serve", "--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Starts a local daemon") {
		t.Errorf("help should mention daemon: %s", output)
	}
	if !strings.Contains(output, "--port") {
		t.Errorf("help should show --port flag: %s", output)
	}
	if !strings.Contains(output, "--no-watch") {
		t.Errorf("help should show --no-watch flag: %s", output)
	}
	if !strings.Contains(output, "--no-ui") {
		t.Errorf("help should show --no-ui flag: %s", output)
	}
	if !strings.Contains(output, "--add") {
		t.Errorf("help should show --add flag: %s", output)
	}
	if !strings.Contains(output, "--remove") {
		t.Errorf("help should show --remove flag: %s", output)
	}
	if !strings.Contains(output, "--list") {
		t.Errorf("help should show --list flag: %s", output)
	}
}

func TestServeCmd_NoWorkspace(t *testing.T) {
	env := newTestEnvEmpty(t)

	// Verify we're actually in the empty dir
	heroDir := filepath.Join(env.dir, ".hero")
	if _, err := os.Stat(heroDir); err == nil {
		t.Skip("unexpected .hero directory in test env")
	}

	// Reset Cobra command state to avoid interference from other tests
	serveCmd.ResetFlags()
	serveCmd.Flags().IntVar(&servePort, "port", 0, "HTTP port (default 7437)")
	serveCmd.Flags().BoolVar(&serveNoWatch, "no-watch", false, "disable automatic file watching")
	serveCmd.Flags().BoolVar(&serveNoUI, "no-ui", false, "disable the embedded dashboard UI")
	serveCmd.Flags().StringVar(&serveAdd, "add", "", "register a project directory")
	serveCmd.Flags().StringVar(&serveRemove, "remove", "", "unregister a project by slug")
	serveCmd.Flags().BoolVar(&serveList, "list", false, "list all registered projects")

	// hero serve with no workspace should fail
	_, err := runCmd("serve", "--port", "0")
	if err == nil {
		t.Fatal("expected error when no workspace exists")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "hero workspace not found") {
		t.Errorf("error should mention missing workspace, got: %v", err)
	}
}

func TestMCPCmd_NoWorkspace(t *testing.T) {
	env := newTestEnvEmpty(t)

	heroDir := filepath.Join(env.dir, ".hero")
	if _, err := os.Stat(heroDir); err == nil {
		t.Skip("unexpected .hero directory in test env")
	}

	_, err := runCmd("mcp")
	if err == nil {
		t.Fatal("expected error when no workspace exists")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "hero workspace not found") {
		t.Errorf("error should mention missing workspace, got: %v", err)
	}
}

func TestMCPCmd_WithWorkspace(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/test-feature/spec.md", `---
title: Test Feature
type: feature
status: planning
---

# Test Feature

A test feature for MCP testing.
`)
	env.indexAll()

	// MCP mode reads from stdin. With empty stdin, it should return immediately.
	output, err := runCmd("mcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// MCP mode with empty stdin produces no output (stdin closes immediately)
	_ = output
}

func TestServeCmd_List(t *testing.T) {
	_ = newTestEnvEmpty(t) // just need a clean working dir

	// Override HOME so we don't read the real ~/.hero/projects.json
	t.Setenv("HOME", t.TempDir())

	output, err := runCmd("serve", "--list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With no projects registered, should show empty message
	if !strings.Contains(output, "No projects registered") && !strings.Contains(output, "Registered projects") {
		t.Errorf("expected project list output, got: %s", output)
	}
}
