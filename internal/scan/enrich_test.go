package scan

import (
	"strings"
	"testing"
)

func TestParseGoMod(t *testing.T) {
	content := `module github.com/foo/bar

go 1.21

require (
	github.com/spf13/cobra v1.10.2
	github.com/pkg/errors v0.9.1 // indirect
)

require github.com/google/uuid v1.6.0
`
	e := &Enrichment{}
	parseGoMod(content, e)

	if e.ModulePath != "github.com/foo/bar" {
		t.Errorf("ModulePath = %q, want github.com/foo/bar", e.ModulePath)
	}
	if e.ModuleGoVer != "1.21" {
		t.Errorf("ModuleGoVer = %q, want 1.21", e.ModuleGoVer)
	}
	if len(e.GoRequires) != 3 {
		t.Errorf("len(GoRequires) = %d, want 3", len(e.GoRequires))
	}

	// Direct deps first
	if e.GoRequires[0].Indirect || e.GoRequires[1].Indirect {
		t.Error("direct deps should come before indirect")
	}

	// Check indirect flag
	found := false
	for _, r := range e.GoRequires {
		if r.Path == "github.com/pkg/errors" && r.Indirect {
			found = true
		}
	}
	if !found {
		t.Error("pkg/errors should be indirect")
	}
}

func TestParseGoModEmpty(t *testing.T) {
	e := &Enrichment{}
	parseGoMod("", e)
	if e.ModulePath != "" {
		t.Errorf("expected empty ModulePath for empty content, got %q", e.ModulePath)
	}
}

func TestParsePackageJSON(t *testing.T) {
	content := `{
  "name": "my-app",
  "version": "2.1.0",
  "dependencies": {
    "react": "^18.0.0",
    "express": "^4.18.0"
  },
  "devDependencies": {
    "jest": "^29.0.0"
  },
  "scripts": {
    "build": "tsc --build",
    "test": "jest"
  }
}`
	e := &Enrichment{NPMScripts: map[string]string{}}
	parsePackageJSON(content, e)

	if e.NPMName != "my-app" {
		t.Errorf("NPMName = %q, want my-app", e.NPMName)
	}
	if e.NPMVersion != "2.1.0" {
		t.Errorf("NPMVersion = %q, want 2.1.0", e.NPMVersion)
	}

	foundReact := false
	for _, d := range e.NPMDeps {
		if d.Name == "react" {
			foundReact = true
		}
	}
	if !foundReact {
		t.Error("react not in NPMDeps")
	}

	foundJest := false
	for _, d := range e.NPMDevDeps {
		if d.Name == "jest" {
			foundJest = true
		}
	}
	if !foundJest {
		t.Error("jest not in NPMDevDeps")
	}

	if e.NPMScripts["build"] != "tsc --build" {
		t.Errorf("scripts[build] = %q, want tsc --build", e.NPMScripts["build"])
	}
}

func TestParseCargoToml(t *testing.T) {
	content := `[package]
name = "myapp"
version = "0.5.2"
edition = "2021"
`
	e := &Enrichment{}
	parseCargoToml(content, e)

	if e.CargoName != "myapp" {
		t.Errorf("CargoName = %q, want myapp", e.CargoName)
	}
	if e.CargoVersion != "0.5.2" {
		t.Errorf("CargoVersion = %q, want 0.5.2", e.CargoVersion)
	}
}

func TestExtractGoPackageName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"package main\n\nfunc main() {}\n", "main"},
		{"package api\n\n// stuff\n", "api"},
		{"// Package scan provides codebase analysis.\npackage scan\n", "scan"},
		{"", ""},
	}

	for _, tt := range tests {
		got := extractGoPackageName(tt.input)
		if got != tt.want {
			t.Errorf("extractGoPackageName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractGoPackageDoc(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantSub string // substring that should appear in the result
	}{
		{
			name: "line comment doc",
			input: `// Package api provides HTTP handlers for the REST API.
// It uses the Gin framework.
package api
`,
			wantSub: "HTTP handlers",
		},
		{
			name: "block comment doc",
			input: `/*
Package db provides database access using BadgerDB.
*/
package db
`,
			wantSub: "database access",
		},
		{
			name: "no doc comment",
			input: `package main

func main() {}
`,
			wantSub: "",
		},
		{
			name: "package prefix stripped",
			input: `// Package scan provides codebase analysis for project onboarding.
package scan
`,
			wantSub: "codebase analysis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractGoPackageDoc(tt.input)
			if tt.wantSub == "" {
				if got != "" {
					t.Errorf("expected empty doc, got %q", got)
				}
			} else {
				if !strings.Contains(got, tt.wantSub) {
					t.Errorf("extractGoPackageDoc() = %q, want substring %q", got, tt.wantSub)
				}
			}
		})
	}
}

func TestExtractReadmeHeadline(t *testing.T) {
	text := "# Hero\n\nA spec-driven AI engineering workflow.\n\n## Usage\n..."
	got := extractReadmeHeadline(text)
	if got != "Hero" {
		t.Errorf("extractReadmeHeadline = %q, want Hero", got)
	}
}

func TestExtractReadmeHeadlineNone(t *testing.T) {
	got := extractReadmeHeadline("Just some text\nno headings here\n")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtractReadmeDescription(t *testing.T) {
	text := "# Hero\n\n[![badge](img)](link)\n\nHero is a spec-driven AI engineering workflow tool.\nDesign before you build.\n\n## Usage\n"
	got := extractReadmeDescription(text)
	if !strings.Contains(got, "spec-driven") {
		t.Errorf("description missing expected text, got %q", got)
	}
}

func TestExtractNPMDeps(t *testing.T) {
	content := `{
  "dependencies": {
    "react": "^18.0.0",
    "express": "^4.18.0",
    "lodash": "^4.17.21"
  }
}`
	deps := extractNPMDeps(content, "dependencies")
	if len(deps) != 3 {
		t.Errorf("len(deps) = %d, want 3", len(deps))
	}
	names := map[string]bool{}
	for _, d := range deps {
		names[d.Name] = true
	}
	if !names["react"] || !names["express"] || !names["lodash"] {
		t.Errorf("missing expected deps, got: %v", deps)
	}
}

func TestExtractNPMDepsEmpty(t *testing.T) {
	deps := extractNPMDeps(`{"name":"foo"}`, "dependencies")
	if len(deps) != 0 {
		t.Errorf("expected 0 deps, got %d", len(deps))
	}
}

func TestEnrichGoProject(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"go.mod": `module github.com/test/myapp

go 1.22

require (
	github.com/spf13/cobra v1.10.2
	github.com/pkg/errors v0.9.1 // indirect
)
`,
		"README.md":                "# MyApp\n\nMyApp is a test application.\n",
		"cmd/main.go":              "// Package main is the entry point.\npackage main\n\nfunc main() {}\n",
		"internal/api/api.go":      "// Package api provides HTTP handlers.\npackage api\n",
		"internal/api/api_test.go": "package api\n\nimport \"testing\"\n\nfunc TestFoo(t *testing.T) {}\n",
		"internal/db/db.go":        "// Package db provides database access.\npackage db\n",
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	e := Enrich(dir, r)

	if e.ModulePath != "github.com/test/myapp" {
		t.Errorf("ModulePath = %q, want github.com/test/myapp", e.ModulePath)
	}
	if e.ModuleGoVer != "1.22" {
		t.Errorf("ModuleGoVer = %q, want 1.22", e.ModuleGoVer)
	}
	if e.GoTestCount != 1 {
		t.Errorf("GoTestCount = %d, want 1", e.GoTestCount)
	}
	if !strings.Contains(e.ReadmeText, "MyApp") {
		t.Errorf("ReadmeText missing MyApp, got %q", e.ReadmeText)
	}
	// Should have at least one direct dep
	direct := filterGoRequires(e.GoRequires, false)
	if len(direct) == 0 {
		t.Error("expected at least one direct Go dependency")
	}
	// Should have package docs for internal packages
	found := false
	for _, pd := range e.PackageDocs {
		if strings.Contains(pd.Dir, "api") && strings.Contains(pd.Doc, "HTTP") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected package doc for internal/api, got: %+v", e.PackageDocs)
	}
}

func TestEnrichNPMProject(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"package.json": `{
  "name": "my-frontend",
  "version": "1.2.3",
  "dependencies": {
    "react": "^18.0.0"
  },
  "devDependencies": {
    "jest": "^29.0.0"
  },
  "scripts": {
    "build": "vite build",
    "test": "jest"
  }
}`,
		"src/index.ts": "console.log('hello');\n",
		"README.md":    "# My Frontend\n\nA React application.\n",
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	e := Enrich(dir, r)

	if e.NPMName != "my-frontend" {
		t.Errorf("NPMName = %q, want my-frontend", e.NPMName)
	}
	if e.NPMVersion != "1.2.3" {
		t.Errorf("NPMVersion = %q, want 1.2.3", e.NPMVersion)
	}
	if len(e.NPMDeps) == 0 {
		t.Error("expected NPM deps")
	}
	if e.NPMScripts["build"] != "vite build" {
		t.Errorf("scripts[build] = %q", e.NPMScripts["build"])
	}
}

func TestGenerateRichProjectOverview(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"go.mod": `module github.com/test/myapp

go 1.22

require (
	github.com/spf13/cobra v1.10.2
)
`,
		"README.md":           "# MyApp\n\nMyApp is a spec-driven tool for testing.\n",
		"cmd/main.go":         "package main\nfunc main() {}\n",
		"internal/api/api.go": "// Package api provides HTTP handlers.\npackage api\n",
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	e := Enrich(dir, r)

	heroDir := dir + "/.hero"
	entry := GenerateRichProjectOverview(r, e, heroDir, "2026-01-01")

	if entry.Type != "context" {
		t.Errorf("Type = %q, want context", entry.Type)
	}
	if entry.Slug != "project-overview" {
		t.Errorf("Slug = %q, want project-overview", entry.Slug)
	}

	body := entry.Content

	// Should contain module info
	if !strings.Contains(body, "github.com/test/myapp") {
		t.Error("content missing module path")
	}
	// Should contain Go version
	if !strings.Contains(body, "1.22") {
		t.Error("content missing Go version")
	}
	// Should contain README headline
	if !strings.Contains(body, "MyApp") {
		t.Error("content missing README headline")
	}
	// Should contain cobra dependency
	if !strings.Contains(body, "cobra") {
		t.Error("content missing spf13/cobra dependency")
	}
	// Should contain package doc
	if !strings.Contains(body, "HTTP handlers") {
		t.Error("content missing package doc for internal/api")
	}
	// Should contain gaps section
	if !strings.Contains(body, "Current Gaps") {
		t.Error("content missing Current Gaps section")
	}
}

func TestGenerateRichProjectOverviewEmpty(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"main.go": "package main\nfunc main() {}\n",
	})

	r, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	e := Enrich(dir, r)

	heroDir := dir + "/.hero"
	entry := GenerateRichProjectOverview(r, e, heroDir, "2026-01-01")

	// Should not panic, should produce valid content
	if entry.Slug != "project-overview" {
		t.Errorf("Slug = %q", entry.Slug)
	}
	if entry.Content == "" {
		t.Error("content should not be empty")
	}
}

func TestInferDirLabel(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"cmd", "CLI entry points (main packages)"},
		{"internal", "internal packages (not exported)"},
		{"src", "application source code"},
		{"docs", "documentation"},
		{"unknown-dir", ""},
	}

	for _, tt := range tests {
		got := inferDirLabel(tt.name, nil, nil)
		if tt.want == "" {
			// Just check it doesn't panic
			continue
		}
		if !strings.Contains(got, tt.want[:10]) {
			t.Errorf("inferDirLabel(%q) = %q, want prefix of %q", tt.name, got, tt.want)
		}
	}
}

func TestInferDirLabelWithSubdirs(t *testing.T) {
	got := inferDirLabel("cmd", []string{"server", "cli"}, nil)
	if !strings.Contains(got, "server") {
		t.Errorf("inferDirLabel with subdirs should list them, got %q", got)
	}
}

func TestTruncate(t *testing.T) {
	short := "hello world"
	if truncate(short, 100) != short {
		t.Errorf("short string should not be truncated")
	}

	long := strings.Repeat("hello world ", 100)
	result := truncate(long, 50)
	if len(result) > 55 { // allow a few extra for the ellipsis
		t.Errorf("truncate result too long: %d chars", len(result))
	}
	if !strings.HasSuffix(result, "…") {
		t.Errorf("truncated string should end with ellipsis, got %q", result)
	}
}

func TestShortDepName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"github.com/spf13/cobra", "spf13/cobra"},
		{"modernc.org/sqlite", "modernc.org/sqlite"},
		{"gopkg.in/yaml.v3", "gopkg.in/yaml.v3"},
		{"singlepart", "singlepart"},
	}

	for _, tt := range tests {
		got := shortDepName(tt.path)
		if got != tt.want {
			t.Errorf("shortDepName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestDetectGaps(t *testing.T) {
	r := &Result{
		Languages: []Language{{Name: "Go", FileCount: 50}},
		// No CI, no linters, no Docker, no tests
	}
	e := &Enrichment{GoTestCount: 0}

	gaps := detectGaps(r, e)

	names := map[string]bool{}
	for _, g := range gaps {
		names[g.name] = true
	}

	if !names["No tests"] {
		t.Error("should detect No tests gap")
	}
	if !names["No CI/CD"] {
		t.Error("should detect No CI/CD gap")
	}
	if !names["No linters"] {
		t.Error("should detect No linters gap")
	}
}

func TestDetectGapsNone(t *testing.T) {
	r := &Result{
		Languages:   []Language{{Name: "Go", FileCount: 50}},
		CIProviders: []CIProvider{{Name: "GitHub Actions"}},
		Linters:     []Linter{{Name: "golangci-lint"}},
		TestFrames:  []TestFramework{{Name: "go test"}},
		Structure:   ProjectStructure{HasDocker: true},
		DocFiles:    []string{"README.md"},
	}
	e := &Enrichment{GoTestCount: 5}

	gaps := detectGaps(r, e)
	if len(gaps) != 0 {
		t.Errorf("expected no gaps, got: %v", gaps)
	}
}
