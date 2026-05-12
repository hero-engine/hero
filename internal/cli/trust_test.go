package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrustCodex(t *testing.T) {
	_ = newTestEnvEmpty(t)

	out, err := runCmd("trust", "codex")
	if err != nil {
		t.Fatalf("trust codex returned error: %v", err)
	}

	assertCodexTrustHint(t, out)
}

func TestTrustUnsupportedTarget(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("trust", "claude")
	if err == nil {
		t.Fatal("trust claude should fail")
	}
	if !strings.Contains(err.Error(), `unsupported trust target "claude"; supported targets: codex`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallCodexPrintsTrustHint(t *testing.T) {
	_ = newTestEnvEmpty(t)
	targetDir := t.TempDir()

	out, err := runCmd("install", "project", targetDir, "--target", "codex")
	if err != nil {
		t.Fatalf("install codex returned error: %v", err)
	}

	assertCodexTrustHint(t, out)
	if _, err := os.Stat(filepath.Join(targetDir, ".codex", "hooks.json")); err != nil {
		t.Fatalf("expected codex hooks to be installed: %v", err)
	}
}

func assertCodexTrustHint(t *testing.T, out string) {
	t.Helper()

	want := []string{
		"Codex permissions: optional one-time setup",
		"Hero cannot grant Codex permissions itself; Codex owns the approval.",
		"Please run `hero status` and request persistent approval for the `hero` command prefix.",
		"You can show this again with `hero trust codex`.",
	}
	for _, s := range want {
		if !strings.Contains(out, s) {
			t.Fatalf("expected output to contain %q, got:\n%s", s, out)
		}
	}
}
