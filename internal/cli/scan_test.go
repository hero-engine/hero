package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanBasic(t *testing.T) {
	env := newTestEnv(t)

	// Create some Go files to detect
	os.WriteFile(filepath.Join(env.dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)
	os.WriteFile(filepath.Join(env.dir, "go.mod"), []byte("module example.com/test\n\ngo 1.21\n"), 0o644)

	output, err := runCmd("scan")
	if err != nil {
		t.Fatalf("scan returned error: %v", err)
	}

	if !strings.Contains(output, "Go") {
		t.Errorf("output missing Go language: %q", output)
	}
	if !strings.Contains(output, "Created:") {
		t.Errorf("output missing Created count: %q", output)
	}
}

func TestScanDryRun(t *testing.T) {
	env := newTestEnv(t)

	os.WriteFile(filepath.Join(env.dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)
	os.WriteFile(filepath.Join(env.dir, "go.mod"), []byte("module example.com/test\n"), 0o644)

	output, err := runCmd("scan", "--dry-run")
	if err != nil {
		t.Fatalf("scan --dry-run returned error: %v", err)
	}

	if !strings.Contains(output, "Dry run") {
		t.Errorf("output missing 'Dry run': %q", output)
	}

	// Should NOT have written any files
	contextDir := filepath.Join(env.heroDir, "knowledge", "context", "project-overview")
	if _, err := os.Stat(contextDir); err == nil {
		t.Error("dry-run should not create files")
	}
}

func TestScanGeneratesProjectOverview(t *testing.T) {
	env := newTestEnv(t)

	os.WriteFile(filepath.Join(env.dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)

	_, err := runCmd("scan")
	if err != nil {
		t.Fatalf("scan returned error: %v", err)
	}

	specPath := filepath.Join(env.heroDir, "knowledge", "context", "project-overview", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("project-overview spec not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type: context") {
		t.Error("project-overview missing type: context")
	}
	if !strings.Contains(content, "Project Overview") {
		t.Error("project-overview missing title")
	}
}

func TestScanGeneratesLinterConvention(t *testing.T) {
	env := newTestEnv(t)

	os.WriteFile(filepath.Join(env.dir, ".eslintrc.json"), []byte(`{"extends":"next"}`), 0o644)
	os.WriteFile(filepath.Join(env.dir, "app.js"), []byte("console.log('hi');\n"), 0o644)

	_, err := runCmd("scan")
	if err != nil {
		t.Fatalf("scan returned error: %v", err)
	}

	specPath := filepath.Join(env.heroDir, "knowledge", "conventions", "use-eslint", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("eslint convention not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type: convention") {
		t.Error("convention missing type")
	}
	if !strings.Contains(content, "ESLint") {
		t.Error("convention missing ESLint")
	}
}

func TestScanGeneratesCIRule(t *testing.T) {
	env := newTestEnv(t)

	ghDir := filepath.Join(env.dir, ".github", "workflows")
	os.MkdirAll(ghDir, 0o755)
	os.WriteFile(filepath.Join(ghDir, "ci.yml"), []byte("name: CI\non: push\n"), 0o644)
	os.WriteFile(filepath.Join(env.dir, "main.go"), []byte("package main\n"), 0o644)

	_, err := runCmd("scan")
	if err != nil {
		t.Fatalf("scan returned error: %v", err)
	}

	specPath := filepath.Join(env.heroDir, "knowledge", "rules", "ci-github-actions", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("CI rule not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type: rule") {
		t.Error("rule missing type")
	}
	if !strings.Contains(content, "GitHub Actions") {
		t.Error("rule missing GitHub Actions")
	}
}

func TestScanSkipsExisting(t *testing.T) {
	env := newTestEnv(t)

	os.WriteFile(filepath.Join(env.dir, "main.go"), []byte("package main\n"), 0o644)

	// Pre-create the project-overview
	overviewDir := filepath.Join(env.heroDir, "knowledge", "context", "project-overview")
	os.MkdirAll(overviewDir, 0o755)
	os.WriteFile(filepath.Join(overviewDir, "spec.md"), []byte("original content"), 0o644)

	output, err := runCmd("scan")
	if err != nil {
		t.Fatalf("scan returned error: %v", err)
	}

	if !strings.Contains(output, "Skipped") {
		t.Errorf("output missing Skipped count: %q", output)
	}

	// Content should be unchanged
	data, _ := os.ReadFile(filepath.Join(overviewDir, "spec.md"))
	if string(data) != "original content" {
		t.Error("existing entry was overwritten without --force")
	}
}

func TestScanForceOverwrites(t *testing.T) {
	env := newTestEnv(t)

	os.WriteFile(filepath.Join(env.dir, "main.go"), []byte("package main\n"), 0o644)

	// Pre-create the project-overview
	overviewDir := filepath.Join(env.heroDir, "knowledge", "context", "project-overview")
	os.MkdirAll(overviewDir, 0o755)
	os.WriteFile(filepath.Join(overviewDir, "spec.md"), []byte("original"), 0o644)

	_, err := runCmd("scan", "--force")
	if err != nil {
		t.Fatalf("scan --force returned error: %v", err)
	}

	// Content should be overwritten
	data, _ := os.ReadFile(filepath.Join(overviewDir, "spec.md"))
	if string(data) == "original" {
		t.Error("--force did not overwrite existing entry")
	}
	if !strings.Contains(string(data), "Project Overview") {
		t.Error("overwritten file missing expected content")
	}
}

func TestScanRequiresWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("scan")
	if err == nil {
		t.Fatal("scan should fail without hero workspace")
	}

	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("error should mention workspace: %v", err)
	}
}

func TestScanShowsSkills(t *testing.T) {
	env := newTestEnv(t)

	os.WriteFile(filepath.Join(env.dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)
	os.WriteFile(filepath.Join(env.dir, "go.mod"), []byte("module test\n"), 0o644)

	output, err := runCmd("scan")
	if err != nil {
		t.Fatalf("scan returned error: %v", err)
	}

	if !strings.Contains(output, "go-stack") {
		t.Errorf("output missing go-stack skill: %q", output)
	}
}

func TestScanEmptyProject(t *testing.T) {
	_ = newTestEnv(t)

	// No source files at all — should still succeed
	output, err := runCmd("scan")
	if err != nil {
		t.Fatalf("scan returned error on empty project: %v", err)
	}

	// Should still generate project-overview
	if !strings.Contains(output, "project-overview") {
		t.Errorf("output missing project-overview entry: %q", output)
	}
}

func TestScanGeneratesTestConvention(t *testing.T) {
	env := newTestEnv(t)

	os.WriteFile(filepath.Join(env.dir, "jest.config.js"), []byte("module.exports = {};\n"), 0o644)
	os.WriteFile(filepath.Join(env.dir, "src/app.js"), []byte("module.exports = {};\n"), 0o644)

	// Need to create src dir first
	os.MkdirAll(filepath.Join(env.dir, "src"), 0o755)
	os.WriteFile(filepath.Join(env.dir, "src", "app.js"), []byte("module.exports = {};\n"), 0o644)

	_, err := runCmd("scan")
	if err != nil {
		t.Fatalf("scan returned error: %v", err)
	}

	specPath := filepath.Join(env.heroDir, "knowledge", "conventions", "testing-with-jest", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("jest convention not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Jest") {
		t.Error("convention missing Jest")
	}
}

func TestScanDoesNotInstallPreCommitHook(t *testing.T) {
	env := newTestEnv(t)
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	// Pre-confirm: no hook installed yet (newTestEnv calls init which
	// might install one — but newTestEnv runs init *before* git init
	// here, so the hook install during init would have no-op'd
	// because there was no git repo. Confirm.)
	hookPath := filepath.Join(env.dir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Skipf("pre-commit hook already exists from earlier setup; can't isolate scan behavior")
	}

	os.WriteFile(filepath.Join(env.dir, "main.go"), []byte("package main\n"), 0o644)
	os.WriteFile(filepath.Join(env.dir, "go.mod"), []byte("module example.com/test\n"), 0o644)

	out, err := runCmd("scan")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if strings.Contains(out, "Installed pre-commit hook") {
		t.Errorf("scan should not print hook install line: %q", out)
	}
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("scan should not create .git/hooks/pre-commit (stat err=%v)", err)
	}
}

func TestScanNoHooksFlagAccepted(t *testing.T) {
	env := newTestEnv(t)
	os.WriteFile(filepath.Join(env.dir, "main.go"), []byte("package main\n"), 0o644)
	os.WriteFile(filepath.Join(env.dir, "go.mod"), []byte("module example.com/test\n"), 0o644)

	if _, err := runCmd("scan", "--no-hooks"); err != nil {
		t.Errorf("scan --no-hooks should be accepted as a no-op flag: %v", err)
	}
}

func TestScanOmitsClaudeMemoryStepWhenAbsent(t *testing.T) {
	env := newTestEnv(t)
	// Point HOME at a temp dir so memory.DirForProject resolves to a
	// non-existent path.
	home := t.TempDir()
	t.Setenv("HOME", home)

	os.WriteFile(filepath.Join(env.dir, "main.go"), []byte("package main\n"), 0o644)
	os.WriteFile(filepath.Join(env.dir, "go.mod"), []byte("module example.com/test\n"), 0o644)

	out, err := runCmd("scan")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if strings.Contains(out, "claude-memory") {
		t.Errorf("scan should not emit a claude-memory line when the dir is absent: %q", out)
	}
	// And explicitly: no stray "memory" line carrying the old label either.
	if strings.Contains(out, "⊘  memory:") || strings.Contains(out, "⊘ memory:") {
		t.Errorf("scan should not emit the old 'memory' label: %q", out)
	}
}

func TestScanEmitsFriendlyClaudeMemoryWhenEmpty(t *testing.T) {
	env := newTestEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create the empty memory dir at the path memory.DirForProject
	// would compute: ~/.claude/projects/<encoded>/memory. The scan
	// resolves symlinks on macOS (/var → /private/var), so do the
	// same before encoding to ensure the paths match.
	resolved, err := filepath.EvalSymlinks(env.dir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	encoded := strings.ReplaceAll(resolved, string(filepath.Separator), "-")
	memDir := filepath.Join(home, ".claude", "projects", encoded, "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatalf("mkdir memDir: %v", err)
	}

	os.WriteFile(filepath.Join(env.dir, "main.go"), []byte("package main\n"), 0o644)
	os.WriteFile(filepath.Join(env.dir, "go.mod"), []byte("module example.com/test\n"), 0o644)

	out, err := runCmd("scan")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if !strings.Contains(out, "claude-memory") {
		t.Errorf("expected claude-memory step when memory dir exists: %q", out)
	}
	if !strings.Contains(out, "Claude Code memory store for this project is empty") {
		t.Errorf("expected friendly empty-memory reason in output: %q", out)
	}
	// No raw path leak.
	if strings.Contains(out, memDir+" not present or empty") {
		t.Errorf("output should not contain raw path skip reason: %q", out)
	}
}

func TestScanExplainsUncustomizedUpdates(t *testing.T) {
	env := newTestEnv(t)
	os.WriteFile(filepath.Join(env.dir, "main.go"), []byte("package main\n"), 0o644)
	os.WriteFile(filepath.Join(env.dir, "go.mod"), []byte("module example.com/test\n"), 0o644)

	// First scan: all entries are created. No explanation expected
	// (Created > 0).
	out1, err := runCmd("scan")
	if err != nil {
		t.Fatalf("first scan: %v", err)
	}
	if strings.Contains(out1, "Updated entries hadn't been customized") {
		t.Errorf("first scan should not print uncustomized explanation: %q", out1)
	}

	// Second scan: entries are now re-updated (still uncustomized).
	// Explanation should appear.
	out2, err := runCmd("scan")
	if err != nil {
		t.Fatalf("second scan: %v", err)
	}
	if !strings.Contains(out2, "Updated entries hadn't been customized — they regenerate cleanly. Hand-edits are preserved on future scans.") {
		t.Errorf("second scan should print uncustomized explanation: %q", out2)
	}
}
