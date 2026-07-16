package cli

import (
	"os"
	"testing"
)

// TestMain isolates HOME for the entire package. CLI tests run real
// `hero install` flows in-process, and codex MCP wiring writes the
// machine-local User layer (~/.codex/config.toml) even on project installs
// (codex-mcp-binary-path-resolution) — without this backstop, any test
// driving a codex install would rewrite the developer's real user config
// to point at a transient go-test binary. Tests that need an observable
// HOME still call t.Setenv("HOME", t.TempDir()) themselves.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "hero-cli-test-home-*")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
