package cli

import (
	"os"
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
