package install

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain isolates HOME for the entire package. Codex MCP wiring writes
// the machine-local User layer (~/.codex/config.toml) even on project
// installs (codex-mcp-binary-path-resolution), so ANY test that performs a
// codex install would otherwise rewrite the developer's real user config
// to point at a transient go-test binary. Individual tests that need a
// fresh or observable HOME still call t.Setenv("HOME", t.TempDir()) — this
// is the backstop that makes forgetting that call harmless.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "hero-install-test-home-*")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	// DeepSeek project installs write an absolute hero path into the
	// (isolated) DeepSeek home patch. Pin it to a stable fake executable so
	// results never depend on whether this machine has hero on PATH.
	fakeHero := filepath.Join(home, "bin", "hero")
	if err := os.MkdirAll(filepath.Dir(fakeHero), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(fakeHero, []byte("#!/bin/sh\n"), 0o755); err != nil {
		panic(err)
	}
	os.Setenv("HERO_DEEPSEEK_MCP_COMMAND", fakeHero)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
