package herotest

import (
	"os"
	"path/filepath"
	"testing"
)

func touchFile(t *testing.T, dir, name string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectEmpty(t *testing.T) {
	dir := t.TempDir()
	fw, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if fw != "" {
		t.Errorf("Detect(empty) = %q, want empty", fw)
	}
}

func TestDetectSwift(t *testing.T) {
	dir := t.TempDir()
	touchFile(t, dir, "Package.swift")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw != "xctest" {
		t.Errorf("Detect(Package.swift) = %q, want %q", fw, "xctest")
	}
}

func TestDetectGo(t *testing.T) {
	dir := t.TempDir()
	touchFile(t, dir, "go.mod")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw != "go" {
		t.Errorf("Detect(go.mod) = %q, want %q", fw, "go")
	}
}

func TestDetectJestConfig(t *testing.T) {
	dir := t.TempDir()
	touchFile(t, dir, "jest.config.js")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw != "jest" {
		t.Errorf("Detect(jest.config.js) = %q, want %q", fw, "jest")
	}
}

func TestDetectVitestConfig(t *testing.T) {
	dir := t.TempDir()
	touchFile(t, dir, "vitest.config.ts")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw != "vitest" {
		t.Errorf("Detect(vitest.config.ts) = %q, want %q", fw, "vitest")
	}
}

func TestDetectPackageJSONVitest(t *testing.T) {
	dir := t.TempDir()
	content := `{"devDependencies": {"vitest": "^1.0.0"}}`
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644)
	fw, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw != "vitest" {
		t.Errorf("Detect(package.json+vitest) = %q, want %q", fw, "vitest")
	}
}

func TestDetectPackageJSONJest(t *testing.T) {
	dir := t.TempDir()
	content := `{"devDependencies": {"jest": "^29.0.0"}}`
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644)
	fw, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw != "jest" {
		t.Errorf("Detect(package.json+jest) = %q, want %q", fw, "jest")
	}
}

func TestDetectPackageJSONBare(t *testing.T) {
	dir := t.TempDir()
	content := `{"name": "my-app"}`
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644)
	fw, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw != "vitest" {
		t.Errorf("Detect(bare package.json) = %q, want %q", fw, "vitest")
	}
}

func TestDetectPython(t *testing.T) {
	dir := t.TempDir()
	touchFile(t, dir, "pyproject.toml")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw != "pytest" {
		t.Errorf("Detect(pyproject.toml) = %q, want %q", fw, "pytest")
	}
}

func TestDetectPlaywright(t *testing.T) {
	dir := t.TempDir()
	touchFile(t, dir, "playwright.config.ts")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw != "playwright" {
		t.Errorf("Detect(playwright.config.ts) = %q, want %q", fw, "playwright")
	}
}

func TestDetectPrioritySwiftOverPackageJSON(t *testing.T) {
	dir := t.TempDir()
	touchFile(t, dir, "Package.swift")
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app"}`), 0o644)
	fw, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw != "xctest" {
		t.Errorf("Detect(Package.swift + package.json) = %q, want %q (Swift wins)", fw, "xctest")
	}
}

func TestDetectPriorityGoOverPython(t *testing.T) {
	dir := t.TempDir()
	touchFile(t, dir, "go.mod")
	touchFile(t, dir, "pyproject.toml")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw != "go" {
		t.Errorf("Detect(go.mod + pyproject.toml) = %q, want %q (Go wins)", fw, "go")
	}
}

func TestDetectPriorityJestConfigOverPackageJSON(t *testing.T) {
	dir := t.TempDir()
	touchFile(t, dir, "jest.config.ts")
	// package.json has vitest, but jest config file should win
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"devDependencies":{"vitest":"1.0"}}`), 0o644)
	fw, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw != "jest" {
		t.Errorf("Detect(jest.config.ts + vitest-in-pkg) = %q, want %q (config wins)", fw, "jest")
	}
}
