package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/install"
)

// TestInstallJSON_SatelliteFailureEmitsJSONAndErrors is a regression
// test for the satellite short-circuit in runInstall swallowing the
// --json contract: running `hero install project . --target opencode
// --json` from a directory whose ancestor holds a stray .hero
// workspace with no harness targets used to print human progress to
// stdout, emit no JSON at all, and (in interactive shells) prompt on
// stdin. The contract is: exactly one JSON result object on stdout
// with the error field set, and a non-nil error returned so the CLI
// exits nonzero.
func TestInstallJSON_SatelliteFailureEmitsJSONAndErrors(t *testing.T) {
	root := t.TempDir()

	// Stray ancestor workspace: .hero exists but no harness targets
	// are installed at root, so satellite materialize must fail.
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".hero", "hero.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(sub); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	output, cmdErr := runCmd("install", "project", ".", "--target", "opencode", "--json")

	if cmdErr == nil {
		t.Fatal("expected install to fail (no harness targets at root), got nil error — CLI would exit 0")
	}

	var out install.InstallJSONOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("stdout is not a single JSON result object: %v\nstdout:\n%s", err, output)
	}
	if out.Error == nil {
		t.Fatalf("JSON output has no error field despite failure\nstdout:\n%s", output)
	}
	if out.Error.Code != "install_failed" {
		t.Errorf("error code = %q, want %q", out.Error.Code, "install_failed")
	}
	if out.Error.Message == "" {
		t.Error("error message is empty")
	}
}
