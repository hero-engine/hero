package install

import (
	"os"
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
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
