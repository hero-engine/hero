package scan

import (
	"os"
	"path/filepath"
	"testing"
)

// newScanDir creates a temp directory with the given files.
// Each key is a relative path; value is the file content.
func newScanDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	return dir
}

func TestAnalyzeGoProject(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"go.mod":              "module example.com/myapp\n\ngo 1.21\n",
		"Makefile":            "build:\n\tgo build ./...\n",
		"cmd/server/main.go":  "package main\n\nfunc main() {}\n",
		"internal/db/db.go":   "package db\n",
		"internal/api/api.go": "package api\n",
		"README.md":           "# My App\n",
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	// Should detect Go
	if r.PrimaryLanguage() != "Go" {
		t.Errorf("PrimaryLanguage = %q, want Go", r.PrimaryLanguage())
	}

	// Should detect go.mod
	found := false
	for _, p := range r.PackageMgrs {
		if p.Name == "Go Modules" {
			found = true
		}
	}
	if !found {
		t.Error("Go Modules not detected")
	}

	// Should detect Make
	foundMake := false
	for _, b := range r.BuildTools {
		if b.Name == "Make" {
			foundMake = true
		}
	}
	if !foundMake {
		t.Error("Make not detected")
	}

	// Should find entry point
	foundEntry := false
	for _, ep := range r.Structure.EntryPoints {
		if ep == filepath.Join("cmd", "server", "main.go") {
			foundEntry = true
		}
	}
	if !foundEntry {
		t.Errorf("entry point cmd/server/main.go not found, got %v", r.Structure.EntryPoints)
	}

	// Should detect docs
	foundDoc := false
	for _, d := range r.DocFiles {
		if d == "README.md" {
			foundDoc = true
		}
	}
	if !foundDoc {
		t.Errorf("README.md not detected in docs, got %v", r.DocFiles)
	}

	// Top level dirs
	hasCmdDir := false
	hasInternalDir := false
	for _, d := range r.Structure.TopLevelDirs {
		if d == "cmd" {
			hasCmdDir = true
		}
		if d == "internal" {
			hasInternalDir = true
		}
	}
	if !hasCmdDir || !hasInternalDir {
		t.Errorf("expected cmd and internal in TopLevelDirs, got %v", r.Structure.TopLevelDirs)
	}
}

func TestAnalyzeNodeProject(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"package.json":    `{"name":"myapp","dependencies":{"react":"^18","express":"^4"}}`,
		"tsconfig.json":   `{"compilerOptions":{}}`,
		"yarn.lock":       "# yarn lockfile v1\n",
		"src/index.ts":    "console.log('hello');\n",
		"src/App.tsx":     "export default function App() { return <div/>; }\n",
		"src/utils/fn.ts": "export const add = (a: number, b: number) => a + b;\n",
		".eslintrc.json":  `{"extends":"next"}`,
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	// Should detect TypeScript as primary (more .ts/.tsx files than .json)
	primary := r.PrimaryLanguage()
	if primary != "TypeScript" {
		t.Errorf("PrimaryLanguage = %q, want TypeScript", primary)
	}

	// Should detect React from package.json
	foundReact := false
	for _, fw := range r.Frameworks {
		if fw.Name == "React" {
			foundReact = true
		}
	}
	if !foundReact {
		t.Error("React not detected from package.json")
	}

	// Should detect Express from package.json
	foundExpress := false
	for _, fw := range r.Frameworks {
		if fw.Name == "Express" {
			foundExpress = true
		}
	}
	if !foundExpress {
		t.Error("Express not detected from package.json")
	}

	// Should detect Yarn
	foundYarn := false
	for _, p := range r.PackageMgrs {
		if p.Name == "Yarn" {
			foundYarn = true
		}
	}
	if !foundYarn {
		t.Error("Yarn not detected")
	}

	// Should detect ESLint
	foundESLint := false
	for _, l := range r.Linters {
		if l.Name == "ESLint" {
			foundESLint = true
		}
	}
	if !foundESLint {
		t.Error("ESLint not detected")
	}

	// Should detect TypeScript framework
	foundTS := false
	for _, fw := range r.Frameworks {
		if fw.Name == "TypeScript" {
			foundTS = true
		}
	}
	if !foundTS {
		t.Error("TypeScript framework not detected from tsconfig.json")
	}
}

func TestAnalyzeRustProject(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"Cargo.toml":  `[package]\nname = "myapp"\nversion = "0.1.0"\n`,
		"src/main.rs": "fn main() {}\n",
		"src/lib.rs":  "pub fn hello() {}\n",
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if r.PrimaryLanguage() != "Rust" {
		t.Errorf("PrimaryLanguage = %q, want Rust", r.PrimaryLanguage())
	}

	foundCargo := false
	for _, b := range r.BuildTools {
		if b.Name == "Cargo" {
			foundCargo = true
		}
	}
	if !foundCargo {
		t.Error("Cargo not detected")
	}

	foundEntry := false
	for _, ep := range r.Structure.EntryPoints {
		if ep == filepath.Join("src", "main.rs") {
			foundEntry = true
		}
	}
	if !foundEntry {
		t.Errorf("main.rs not in entry points, got %v", r.Structure.EntryPoints)
	}
}

func TestAnalyzePythonProject(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"pyproject.toml":     `[tool.poetry]\nname = "myapp"\n`,
		"poetry.lock":        "# lock\n",
		"src/main.py":        "print('hello')\n",
		"src/utils.py":       "def add(a, b): return a + b\n",
		"conftest.py":        "import pytest\n",
		"tests/test_main.py": "def test_main(): pass\n",
		".flake8":            "[flake8]\nmax-line-length = 100\n",
		"ruff.toml":          "[tool.ruff]\n",
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if r.PrimaryLanguage() != "Python" {
		t.Errorf("PrimaryLanguage = %q, want Python", r.PrimaryLanguage())
	}

	// Should detect Poetry
	foundPoetry := false
	for _, p := range r.PackageMgrs {
		if p.Name == "Poetry" {
			foundPoetry = true
		}
	}
	if !foundPoetry {
		t.Error("Poetry not detected")
	}

	// Should detect pytest
	foundPytest := false
	for _, tf := range r.TestFrames {
		if tf.Name == "pytest" {
			foundPytest = true
		}
	}
	if !foundPytest {
		t.Error("pytest not detected")
	}

	// Should detect Flake8 and Ruff
	foundFlake := false
	foundRuff := false
	for _, l := range r.Linters {
		if l.Name == "Flake8" {
			foundFlake = true
		}
		if l.Name == "Ruff" {
			foundRuff = true
		}
	}
	if !foundFlake {
		t.Error("Flake8 not detected")
	}
	if !foundRuff {
		t.Error("Ruff not detected")
	}
}

func TestAnalyzeMonorepo(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"package.json":              `{"workspaces":["packages/*"]}`,
		"turbo.json":                `{"pipeline":{}}`,
		"packages/web/package.json": `{"name":"web"}`,
		"packages/api/package.json": `{"name":"api"}`,
		"packages/web/src/index.ts": "console.log('web');\n",
		"packages/api/src/index.ts": "console.log('api');\n",
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if !r.Structure.IsMonorepo {
		t.Error("monorepo not detected")
	}
}

func TestAnalyzeMonorepoFromPackages(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"packages/core/index.js": "module.exports = {}\n",
		"packages/cli/index.js":  "module.exports = {}\n",
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if !r.Structure.IsMonorepo {
		t.Error("monorepo not detected from packages/ dir")
	}
}

func TestAnalyzeDocker(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"Dockerfile":         "FROM golang:1.21\n",
		"docker-compose.yml": "version: '3'\nservices:\n  app:\n    build: .\n",
		"main.go":            "package main\nfunc main() {}\n",
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if !r.Structure.HasDocker {
		t.Error("Docker not detected")
	}
	if !r.Structure.HasCompose {
		t.Error("Docker Compose not detected")
	}
}

func TestAnalyzeCIProviders(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		".github/workflows/ci.yml": "name: CI\non: push\n",
		".travis.yml":              "language: go\n",
		"main.go":                  "package main\nfunc main() {}\n",
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	foundGHA := false
	foundTravis := false
	for _, ci := range r.CIProviders {
		if ci.Name == "GitHub Actions" {
			foundGHA = true
		}
		if ci.Name == "Travis CI" {
			foundTravis = true
		}
	}
	if !foundGHA {
		t.Error("GitHub Actions not detected")
	}
	if !foundTravis {
		t.Error("Travis CI not detected")
	}
}

func TestAnalyzeTestFrameworks(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"package.json":    `{"devDependencies":{"jest":"^29"}}`,
		"jest.config.js":  "module.exports = {};\n",
		"src/app.js":      "module.exports = {};\n",
		"src/app.test.js": "test('a', () => {});\n",
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	foundJest := false
	for _, tf := range r.TestFrames {
		if tf.Name == "Jest" {
			foundJest = true
		}
	}
	if !foundJest {
		t.Error("Jest not detected")
	}
}

func TestAnalyzeEmptyDir(t *testing.T) {
	dir := t.TempDir()

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(r.Languages) != 0 {
		t.Errorf("expected no languages, got %v", r.Languages)
	}
	if r.PrimaryLanguage() != "" {
		t.Errorf("PrimaryLanguage should be empty, got %q", r.PrimaryLanguage())
	}
}

func TestAnalyzeNonexistentDir(t *testing.T) {
	_, err := Analyze("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("Analyze should fail on nonexistent path")
	}
}

func TestPrimaryLanguage(t *testing.T) {
	r := &Result{
		Languages: []Language{
			{Name: "Go", FileCount: 100, Confidence: 0.8},
			{Name: "Shell", FileCount: 10, Confidence: 0.08},
		},
	}

	if r.PrimaryLanguage() != "Go" {
		t.Errorf("PrimaryLanguage = %q, want Go", r.PrimaryLanguage())
	}
}

func TestPrimaryLanguageEmpty(t *testing.T) {
	r := &Result{}
	if r.PrimaryLanguage() != "" {
		t.Errorf("PrimaryLanguage should be empty for no languages, got %q", r.PrimaryLanguage())
	}
}

func TestStackSkillsGo(t *testing.T) {
	r := &Result{
		Languages: []Language{
			{Name: "Go", FileCount: 100},
			{Name: "Shell", FileCount: 5},
		},
	}

	skills := r.StackSkills()
	found := false
	for _, s := range skills {
		if s == "go-stack" {
			found = true
		}
	}
	if !found {
		t.Errorf("go-stack not in skills: %v", skills)
	}
}

func TestStackSkillsReact(t *testing.T) {
	r := &Result{
		Languages: []Language{
			{Name: "TypeScript", FileCount: 50},
		},
		Frameworks: []Framework{
			{Name: "React", Language: "JavaScript"},
		},
	}

	skills := r.StackSkills()
	foundJS := false
	foundReact := false
	for _, s := range skills {
		if s == "javascript-stack" {
			foundJS = true
		}
		if s == "react-stack" {
			foundReact = true
		}
	}
	if !foundJS {
		t.Errorf("javascript-stack not in skills: %v", skills)
	}
	if !foundReact {
		t.Errorf("react-stack not in skills: %v", skills)
	}
}

func TestStackSkillsNoDuplicates(t *testing.T) {
	r := &Result{
		Languages: []Language{
			{Name: "JavaScript", FileCount: 50},
			{Name: "TypeScript", FileCount: 30},
		},
	}

	skills := r.StackSkills()
	// Both JS and TS map to javascript-stack — should appear once
	count := 0
	for _, s := range skills {
		if s == "javascript-stack" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("javascript-stack appeared %d times, want 1", count)
	}
}

func TestSummaryOutput(t *testing.T) {
	r := &Result{
		Languages: []Language{
			{Name: "Go", FileCount: 100, Confidence: 0.8},
		},
		BuildTools:  []BuildTool{{Name: "Make", ConfigFile: "Makefile"}},
		PackageMgrs: []PackageMgr{{Name: "Go Modules", ConfigFile: "go.mod", Language: "Go"}},
		CIProviders: []CIProvider{{Name: "GitHub Actions", ConfigFile: ".github/workflows/ci.yml"}},
		Structure: ProjectStructure{
			TopLevelDirs: []string{"cmd", "internal"},
			EntryPoints:  []string{"cmd/main.go"},
		},
		DocFiles: []string{"README.md"},
	}

	summary := r.Summary()

	checks := []string{
		"Go",
		"100 files",
		"Make",
		"Go Modules",
		"GitHub Actions",
		"cmd, internal",
		"cmd/main.go",
		"README.md",
	}

	for _, check := range checks {
		if !contains(summary, check) {
			t.Errorf("summary missing %q:\n%s", check, summary)
		}
	}
}

func TestDetectLanguages(t *testing.T) {
	extCounts := map[string]int{
		".go":  50,
		".py":  30,
		".sh":  5,
		".txt": 10, // non-code extension
	}

	langs := detectLanguages(extCounts)

	if len(langs) != 3 {
		t.Fatalf("expected 3 languages, got %d: %v", len(langs), langs)
	}

	// Should be sorted by file count descending
	if langs[0].Name != "Go" {
		t.Errorf("first language should be Go, got %q", langs[0].Name)
	}
	if langs[1].Name != "Python" {
		t.Errorf("second language should be Python, got %q", langs[1].Name)
	}

	// .txt should NOT be counted
	for _, l := range langs {
		if l.Name == "" {
			t.Error("unknown language from .txt should not appear")
		}
	}
}

func TestDetectFromPackageJSON(t *testing.T) {
	content := `{
		"dependencies": {
			"react": "^18.0.0",
			"express": "^4.18.0",
			"prisma": "^5.0.0"
		}
	}`

	frameworks := detectFromPackageJSON(content)

	names := map[string]bool{}
	for _, fw := range frameworks {
		names[fw.Name] = true
	}

	if !names["React"] {
		t.Error("React not detected")
	}
	if !names["Express"] {
		t.Error("Express not detected")
	}
	if !names["Prisma"] {
		t.Error("Prisma not detected")
	}
}

func TestDetectFromPackageJSONNoDuplicates(t *testing.T) {
	content := `{
		"dependencies": {"react": "^18"},
		"devDependencies": {"react": "^18"}
	}`

	frameworks := detectFromPackageJSON(content)

	reactCount := 0
	for _, fw := range frameworks {
		if fw.Name == "React" {
			reactCount++
		}
	}
	if reactCount != 1 {
		t.Errorf("React appeared %d times, want 1", reactCount)
	}
}

func TestDetectMonorepoLerna(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"lerna.json": `{"version":"0.0.0"}`,
	})

	if !detectMonorepo(dir, nil) {
		t.Error("lerna monorepo not detected")
	}
}

func TestDetectMonorepoNx(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"nx.json": `{}`,
	})

	if !detectMonorepo(dir, nil) {
		t.Error("nx monorepo not detected")
	}
}

func TestDetectMonorepoWorkspaces(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"package.json": `{"workspaces":["packages/*"]}`,
	})

	if !detectMonorepo(dir, nil) {
		t.Error("workspaces monorepo not detected")
	}
}

func TestDetectMonorepoAppsDir(t *testing.T) {
	if !detectMonorepo("", []string{"apps", "libs"}) {
		t.Error("apps dir monorepo not detected")
	}
}

func TestDetectMonorepoFalse(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"main.go": "package main\n",
	})

	if detectMonorepo(dir, []string{"cmd", "internal"}) {
		t.Error("should not detect monorepo for standard Go layout")
	}
}

func TestDetectDocFiles(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"README.md":       "# Hello\n",
		"CONTRIBUTING.md": "## Contributing\n",
		"LICENSE":         "MIT\n",
		"SECURITY.md":     "## Security\n",
		"random-file.md":  "not a doc\n",
	})

	docs := detectDocFiles(dir)

	expected := map[string]bool{
		"README.md":       true,
		"CONTRIBUTING.md": true,
		"LICENSE":         true,
		"SECURITY.md":     true,
	}

	for _, d := range docs {
		if !expected[d] {
			t.Errorf("unexpected doc file: %q", d)
		}
		delete(expected, d)
	}
	for missing := range expected {
		t.Errorf("missing doc file: %q", missing)
	}

	// random-file.md should NOT appear
	for _, d := range docs {
		if d == "random-file.md" {
			t.Error("random-file.md should not be in doc files")
		}
	}
}

func TestSkipsHiddenDirs(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"main.go":           "package main\nfunc main() {}\n",
		".hidden/secret.go": "package hidden\n",
		".git/config":       "[core]\n",
		"node_modules/x.js": "module.exports = {}\n",
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	// Should only find 1 Go file (main.go), not the hidden or git ones
	for _, l := range r.Languages {
		if l.Name == "Go" && l.FileCount != 1 {
			t.Errorf("Go file count = %d, want 1 (should skip hidden dirs)", l.FileCount)
		}
	}
}

func TestFileExists(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"exists.txt": "hello",
	})

	if !fileExists(dir, "exists.txt") {
		t.Error("fileExists should return true for existing file")
	}
	if fileExists(dir, "nope.txt") {
		t.Error("fileExists should return false for missing file")
	}
}

// --- Generate tests ---

func TestGenerateProjectOverview(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	os.MkdirAll(heroDir, 0o755)

	r := &Result{
		Languages:   []Language{{Name: "Go", FileCount: 100, Confidence: 0.9}},
		Frameworks:  []Framework{{Name: "Cobra", Language: "Go", Indicator: "go.mod"}},
		BuildTools:  []BuildTool{{Name: "Make", ConfigFile: "Makefile"}},
		PackageMgrs: []PackageMgr{{Name: "Go Modules", ConfigFile: "go.mod", Language: "Go"}},
		CIProviders: []CIProvider{{Name: "GitHub Actions", ConfigFile: ".github/workflows/ci.yml"}},
		TestFrames:  []TestFramework{{Name: "go test", Language: "Go", Indicator: "built-in"}},
		Linters:     []Linter{{Name: "golangci-lint", ConfigFile: ".golangci.yml", Language: "Go"}},
		Structure: ProjectStructure{
			TopLevelDirs: []string{"cmd", "internal"},
			EntryPoints:  []string{"cmd/main.go"},
			HasDocker:    true,
		},
		DocFiles: []string{"README.md"},
	}

	entries := Generate(r, heroDir)

	// First entry should be project-overview context
	var overview *GeneratedEntry
	for i := range entries {
		if entries[i].Slug == "project-overview" {
			overview = &entries[i]
			break
		}
	}

	if overview == nil {
		t.Fatal("project-overview entry not generated")
	}
	if overview.Type != "context" {
		t.Errorf("type = %q, want context", overview.Type)
	}
	if !contains(overview.Content, "type: context") {
		t.Error("content missing type: context")
	}
	if !contains(overview.Content, "Go") {
		t.Error("content missing Go language")
	}
	if !contains(overview.Content, "Make") {
		t.Error("content missing Make build tool")
	}
	if !contains(overview.Content, "Docker") {
		t.Error("content missing Docker")
	}
}

func TestGenerateLinterConventions(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	os.MkdirAll(heroDir, 0o755)

	r := &Result{
		Linters: []Linter{
			{Name: "ESLint", ConfigFile: ".eslintrc.json", Language: "JavaScript"},
			{Name: "Prettier", ConfigFile: ".prettierrc", Language: "JavaScript"},
		},
	}

	entries := Generate(r, heroDir)

	// Should have project-overview + 2 linter conventions
	conventionCount := 0
	for _, e := range entries {
		if e.Type == "convention" {
			conventionCount++
			if !contains(e.Content, "type: convention") {
				t.Errorf("convention entry missing type: convention for %s", e.Slug)
			}
			if !contains(e.Content, "auto-generated") {
				t.Errorf("convention entry missing auto-generated tag for %s", e.Slug)
			}
		}
	}

	if conventionCount < 2 {
		t.Errorf("expected at least 2 convention entries, got %d", conventionCount)
	}
}

func TestGenerateCIRules(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	os.MkdirAll(heroDir, 0o755)

	r := &Result{
		CIProviders: []CIProvider{
			{Name: "GitHub Actions", ConfigFile: ".github/workflows/ci.yml"},
		},
	}

	entries := Generate(r, heroDir)

	foundRule := false
	for _, e := range entries {
		if e.Type == "rule" && contains(e.Slug, "github-actions") {
			foundRule = true
			if !contains(e.Content, "type: rule") {
				t.Error("rule entry missing type: rule")
			}
			if !contains(e.Content, "GitHub Actions") {
				t.Error("rule entry missing CI name")
			}
		}
	}

	if !foundRule {
		t.Error("CI rule entry not generated")
	}
}

func TestGenerateTestConventions(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	os.MkdirAll(heroDir, 0o755)

	r := &Result{
		TestFrames: []TestFramework{
			{Name: "Jest", Language: "JavaScript", Indicator: "jest.config.js"},
		},
	}

	entries := Generate(r, heroDir)

	foundTest := false
	for _, e := range entries {
		if e.Type == "convention" && contains(e.Slug, "jest") {
			foundTest = true
			if !contains(e.Content, "Jest") {
				t.Error("test convention missing Jest name")
			}
		}
	}

	if !foundTest {
		t.Error("test convention not generated for Jest")
	}
}

func TestGenerateEmptyResult(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	os.MkdirAll(heroDir, 0o755)

	r := &Result{}

	entries := Generate(r, heroDir)

	// Should still produce project-overview even with empty result
	if len(entries) != 1 {
		t.Errorf("expected 1 entry (project-overview), got %d", len(entries))
	}
	if entries[0].Slug != "project-overview" {
		t.Errorf("first entry slug = %q, want project-overview", entries[0].Slug)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ESLint", "eslint"},
		{"golangci-lint", "golangci-lint"},
		{"GitHub Actions", "github-actions"},
		{"ESLint (flat config)", "eslint-flat-config"},
		{"Prettier", "prettier"},
	}

	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLangGlob(t *testing.T) {
	if langGlob("Go") != "*.go" {
		t.Errorf("langGlob(Go) = %q", langGlob("Go"))
	}
	if langGlob("Python") != "*.py" {
		t.Errorf("langGlob(Python) = %q", langGlob("Python"))
	}
	if langGlob("Unknown") != "*" {
		t.Errorf("langGlob(Unknown) = %q", langGlob("Unknown"))
	}
}

// contains checks if a string contains a substring.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchIn(s, sub)
}

func searchIn(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
