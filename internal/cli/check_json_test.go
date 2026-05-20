package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCheckJSON_WritesCacheArtifact verifies that `hero check --json`
// produces a parseable JSON file at the expected on-disk location
// (the contract the dashboard health cache reads from).
func TestCheckJSON_WritesCacheArtifact(t *testing.T) {
	dir := t.TempDir()

	// Initialize a git repo + hero workspace.
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init")
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# x\n"), 0o644)
	git("add", ".")
	git("commit", "-m", "initial")

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	if _, err := runCmd("init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	if _, err := runCmd("check", "--json"); err != nil {
		t.Fatalf("check --json: %v", err)
	}

	artifact := filepath.Join(dir, ".hero", "cache", "health.json")
	data, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}

	var doc struct {
		CapturedAt string `json:"captured_at"`
		Rows       []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse artifact: %v\n%s", err, data)
	}
	if doc.CapturedAt == "" {
		t.Error("expected captured_at populated")
	}
	if len(doc.Rows) == 0 {
		t.Fatal("expected at least one row in health.json")
	}
	for _, r := range doc.Rows {
		if r.Name == "" {
			t.Errorf("row missing name: %+v", r)
		}
		switch r.Status {
		case "pass", "warn", "fail":
		default:
			t.Errorf("row %q has unexpected status %q", r.Name, r.Status)
		}
	}
}
