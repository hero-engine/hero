// Package scan provides codebase analysis for project onboarding.
// It detects the technology stack, project structure, build tools,
// CI/CD configuration, and common patterns to generate initial
// Hero knowledge base entries.
package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Result holds the complete analysis of a project.
type Result struct {
	ProjectRoot string
	Languages   []Language
	Frameworks  []Framework
	BuildTools  []BuildTool
	CIProviders []CIProvider
	PackageMgrs []PackageMgr
	Structure   ProjectStructure
	Linters     []Linter
	TestFrames  []TestFramework
	DocFiles    []string // README, CONTRIBUTING, etc.

	// Deep analysis results
	MultiModule      *MultiModuleInfo // nil if single-module project
	FrameworkDetails []FrameworkDetail // deep framework-specific info
	ImportSources    []ImportSource    // existing knowledge files found
}

// Language represents a detected programming language.
type Language struct {
	Name       string
	Extensions []string
	FileCount  int
	Confidence float64 // 0.0 to 1.0
}

// Framework represents a detected framework or library.
type Framework struct {
	Name       string
	Language   string
	Indicator  string // what file/pattern detected it
	Confidence float64
}

// BuildTool represents a detected build tool.
type BuildTool struct {
	Name       string
	ConfigFile string
}

// CIProvider represents a detected CI/CD provider.
type CIProvider struct {
	Name       string
	ConfigFile string
}

// PackageMgr represents a detected package manager.
type PackageMgr struct {
	Name       string
	ConfigFile string
	Language   string
}

// Linter represents a detected linter or code quality tool.
type Linter struct {
	Name       string
	ConfigFile string
	Language   string
}

// TestFramework represents a detected testing framework.
type TestFramework struct {
	Name      string
	Language  string
	Indicator string
}

// ProjectStructure describes the top-level project layout.
type ProjectStructure struct {
	IsMonorepo   bool
	TopLevelDirs []string
	EntryPoints  []string // main.go, index.ts, etc.
	HasDocker    bool
	HasCompose   bool
}

// Analyze scans the project root and returns a Result.
func Analyze(projectRoot string) (*Result, error) {
	if _, err := os.Stat(projectRoot); err != nil {
		return nil, fmt.Errorf("project root not found: %w", err)
	}

	r := &Result{
		ProjectRoot: projectRoot,
	}

	// Collect file extension counts (skip hidden dirs, vendors, node_modules)
	extCounts := map[string]int{}
	var topDirs []string

	entries, err := os.ReadDir(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("reading project root: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			name := e.Name()
			if !strings.HasPrefix(name, ".") && name != "node_modules" && name != "vendor" && name != "__pycache__" {
				topDirs = append(topDirs, name)
			}
		}
	}
	r.Structure.TopLevelDirs = topDirs

	// Walk the tree counting extensions (limit depth to avoid huge repos)
	maxFiles := 10000
	fileCount := 0
	filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || fileCount >= maxFiles {
			return filepath.SkipDir
		}

		rel, _ := filepath.Rel(projectRoot, path)

		// Skip hidden dirs, vendor, node_modules, build output
		if info.IsDir() {
			base := info.Name()
			skip := base == ".git" || base == ".hero" || base == "node_modules" ||
				base == "vendor" || base == "__pycache__" || base == ".next" ||
				base == "dist" || base == "build" || base == "target" ||
				base == ".gradle" || base == ".idea" || base == ".vscode" ||
				strings.HasPrefix(base, ".")
			if skip && rel != "." {
				return filepath.SkipDir
			}
			return nil
		}

		fileCount++
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if ext != "" {
			extCounts[ext]++
		}

		// Check for entry points
		name := info.Name()
		if name == "main.go" || name == "index.ts" || name == "index.js" ||
			name == "main.py" || name == "main.rs" || name == "App.tsx" ||
			name == "App.jsx" {
			r.Structure.EntryPoints = append(r.Structure.EntryPoints, rel)
		}

		return nil
	})

	// Detect languages from extensions
	r.Languages = detectLanguages(extCounts)

	// Detect root marker files
	r.Frameworks, r.BuildTools, r.PackageMgrs, r.Linters, r.TestFrames = detectFromMarkers(projectRoot)

	// Detect CI
	r.CIProviders = detectCI(projectRoot)

	// Detect Docker
	r.Structure.HasDocker = fileExists(projectRoot, "Dockerfile") ||
		fileExists(projectRoot, "dockerfile") ||
		dirContainsGlob(projectRoot, "*.dockerfile") ||
		dirContainsGlob(projectRoot, "Dockerfile.*")
	r.Structure.HasCompose = fileExists(projectRoot, "docker-compose.yml") ||
		fileExists(projectRoot, "docker-compose.yaml") ||
		fileExists(projectRoot, "compose.yml") ||
		fileExists(projectRoot, "compose.yaml")

	// Detect monorepo
	r.Structure.IsMonorepo = detectMonorepo(projectRoot, topDirs)

	// Detect doc files
	r.DocFiles = detectDocFiles(projectRoot)

	// Deep analysis: multi-module structure
	r.MultiModule = DetectMultiModule(projectRoot)

	// Deep analysis: framework-specific details
	r.FrameworkDetails = DetectFrameworkDetails(projectRoot, r)

	// Detect importable existing knowledge files
	r.ImportSources = DetectImportSources(projectRoot)

	return r, nil
}

// langDef maps extensions to language names.
var langDef = map[string]string{
	".go":     "Go",
	".java":   "Java",
	".groovy": "Groovy",
	".kt":     "Kotlin",
	".kts":    "Kotlin",
	".rs":     "Rust",
	".py":     "Python",
	".pyi":    "Python",
	".js":     "JavaScript",
	".mjs":    "JavaScript",
	".cjs":    "JavaScript",
	".ts":     "TypeScript",
	".mts":    "TypeScript",
	".tsx":    "TypeScript",
	".jsx":    "JavaScript",
	".rb":     "Ruby",
	".php":    "PHP",
	".cs":     "C#",
	".c":      "C",
	".cpp":    "C++",
	".h":      "C",
	".hpp":    "C++",
	".swift":  "Swift",
	".scala":  "Scala",
	".ex":     "Elixir",
	".exs":    "Elixir",
	".clj":    "Clojure",
	".lua":    "Lua",
	".zig":    "Zig",
	".dart":   "Dart",
	".sh":     "Shell",
	".bash":   "Shell",
	".zsh":    "Shell",
}

func detectLanguages(extCounts map[string]int) []Language {
	// Group by language
	langFiles := map[string][]string{}
	langCounts := map[string]int{}

	for ext, count := range extCounts {
		if lang, ok := langDef[ext]; ok {
			langFiles[lang] = append(langFiles[lang], ext)
			langCounts[lang] += count
		}
	}

	totalCode := 0
	for _, c := range langCounts {
		totalCode += c
	}

	var langs []Language
	for name, count := range langCounts {
		conf := float64(count) / float64(max(totalCode, 1))
		langs = append(langs, Language{
			Name:       name,
			Extensions: langFiles[name],
			FileCount:  count,
			Confidence: conf,
		})
	}

	// Sort by file count descending
	sort.Slice(langs, func(i, j int) bool {
		return langs[i].FileCount > langs[j].FileCount
	})

	return langs
}

func detectFromMarkers(root string) ([]Framework, []BuildTool, []PackageMgr, []Linter, []TestFramework) {
	var frameworks []Framework
	var buildTools []BuildTool
	var pkgMgrs []PackageMgr
	var linters []Linter
	var testFrames []TestFramework

	type marker struct {
		file   string
		action func()
	}

	markers := []marker{
		// Go
		{"go.mod", func() {
			pkgMgrs = append(pkgMgrs, PackageMgr{"Go Modules", "go.mod", "Go"})
			buildTools = append(buildTools, BuildTool{"go build", "go.mod"})
		}},
		{"Makefile", func() {
			buildTools = append(buildTools, BuildTool{"Make", "Makefile"})
		}},

		// Java/JVM
		{"pom.xml", func() {
			buildTools = append(buildTools, BuildTool{"Maven", "pom.xml"})
			pkgMgrs = append(pkgMgrs, PackageMgr{"Maven", "pom.xml", "Java"})
		}},
		{"build.gradle", func() {
			buildTools = append(buildTools, BuildTool{"Gradle", "build.gradle"})
			pkgMgrs = append(pkgMgrs, PackageMgr{"Gradle", "build.gradle", "Java"})
		}},
		{"build.gradle.kts", func() {
			buildTools = append(buildTools, BuildTool{"Gradle (Kotlin DSL)", "build.gradle.kts"})
			pkgMgrs = append(pkgMgrs, PackageMgr{"Gradle", "build.gradle.kts", "Kotlin"})
		}},

		// Rust
		{"Cargo.toml", func() {
			buildTools = append(buildTools, BuildTool{"Cargo", "Cargo.toml"})
			pkgMgrs = append(pkgMgrs, PackageMgr{"Cargo", "Cargo.toml", "Rust"})
		}},

		// Python
		{"pyproject.toml", func() {
			pkgMgrs = append(pkgMgrs, PackageMgr{"pyproject", "pyproject.toml", "Python"})
		}},
		{"setup.py", func() {
			pkgMgrs = append(pkgMgrs, PackageMgr{"setuptools", "setup.py", "Python"})
		}},
		{"requirements.txt", func() {
			pkgMgrs = append(pkgMgrs, PackageMgr{"pip", "requirements.txt", "Python"})
		}},
		{"Pipfile", func() {
			pkgMgrs = append(pkgMgrs, PackageMgr{"Pipenv", "Pipfile", "Python"})
		}},
		{"poetry.lock", func() {
			pkgMgrs = append(pkgMgrs, PackageMgr{"Poetry", "poetry.lock", "Python"})
		}},

		// JavaScript/TypeScript
		{"package.json", func() {
			pkgMgrs = append(pkgMgrs, PackageMgr{"npm", "package.json", "JavaScript"})
		}},
		{"yarn.lock", func() {
			pkgMgrs = append(pkgMgrs, PackageMgr{"Yarn", "yarn.lock", "JavaScript"})
		}},
		{"pnpm-lock.yaml", func() {
			pkgMgrs = append(pkgMgrs, PackageMgr{"pnpm", "pnpm-lock.yaml", "JavaScript"})
		}},
		{"bun.lockb", func() {
			pkgMgrs = append(pkgMgrs, PackageMgr{"Bun", "bun.lockb", "JavaScript"})
		}},
		{"tsconfig.json", func() {
			frameworks = append(frameworks, Framework{"TypeScript", "JavaScript", "tsconfig.json", 1.0})
		}},

		// Ruby
		{"Gemfile", func() {
			pkgMgrs = append(pkgMgrs, PackageMgr{"Bundler", "Gemfile", "Ruby"})
		}},

		// PHP
		{"composer.json", func() {
			pkgMgrs = append(pkgMgrs, PackageMgr{"Composer", "composer.json", "PHP"})
		}},

		// Elixir
		{"mix.exs", func() {
			buildTools = append(buildTools, BuildTool{"Mix", "mix.exs"})
			pkgMgrs = append(pkgMgrs, PackageMgr{"Hex", "mix.exs", "Elixir"})
		}},

		// Linters
		{".eslintrc.json", func() {
			linters = append(linters, Linter{"ESLint", ".eslintrc.json", "JavaScript"})
		}},
		{".eslintrc.js", func() {
			linters = append(linters, Linter{"ESLint", ".eslintrc.js", "JavaScript"})
		}},
		{".eslintrc.yml", func() {
			linters = append(linters, Linter{"ESLint", ".eslintrc.yml", "JavaScript"})
		}},
		{"eslint.config.js", func() {
			linters = append(linters, Linter{"ESLint (flat config)", "eslint.config.js", "JavaScript"})
		}},
		{"eslint.config.mjs", func() {
			linters = append(linters, Linter{"ESLint (flat config)", "eslint.config.mjs", "JavaScript"})
		}},
		{".golangci.yml", func() {
			linters = append(linters, Linter{"golangci-lint", ".golangci.yml", "Go"})
		}},
		{".golangci.yaml", func() {
			linters = append(linters, Linter{"golangci-lint", ".golangci.yaml", "Go"})
		}},
		{".pylintrc", func() {
			linters = append(linters, Linter{"Pylint", ".pylintrc", "Python"})
		}},
		{".flake8", func() {
			linters = append(linters, Linter{"Flake8", ".flake8", "Python"})
		}},
		{"ruff.toml", func() {
			linters = append(linters, Linter{"Ruff", "ruff.toml", "Python"})
		}},
		{".prettierrc", func() {
			linters = append(linters, Linter{"Prettier", ".prettierrc", "JavaScript"})
		}},
		{".prettierrc.json", func() {
			linters = append(linters, Linter{"Prettier", ".prettierrc.json", "JavaScript"})
		}},
		{"rustfmt.toml", func() {
			linters = append(linters, Linter{"rustfmt", "rustfmt.toml", "Rust"})
		}},
		{".rubocop.yml", func() {
			linters = append(linters, Linter{"RuboCop", ".rubocop.yml", "Ruby"})
		}},
		{".editorconfig", func() {
			linters = append(linters, Linter{"EditorConfig", ".editorconfig", ""})
		}},

		// Frameworks (detected from config files)
		{"next.config.js", func() {
			frameworks = append(frameworks, Framework{"Next.js", "JavaScript", "next.config.js", 1.0})
		}},
		{"next.config.mjs", func() {
			frameworks = append(frameworks, Framework{"Next.js", "JavaScript", "next.config.mjs", 1.0})
		}},
		{"next.config.ts", func() {
			frameworks = append(frameworks, Framework{"Next.js", "TypeScript", "next.config.ts", 1.0})
		}},
		{"nuxt.config.js", func() {
			frameworks = append(frameworks, Framework{"Nuxt.js", "JavaScript", "nuxt.config.js", 1.0})
		}},
		{"nuxt.config.ts", func() {
			frameworks = append(frameworks, Framework{"Nuxt.js", "TypeScript", "nuxt.config.ts", 1.0})
		}},
		{"angular.json", func() {
			frameworks = append(frameworks, Framework{"Angular", "TypeScript", "angular.json", 1.0})
		}},
		{"svelte.config.js", func() {
			frameworks = append(frameworks, Framework{"SvelteKit", "JavaScript", "svelte.config.js", 1.0})
		}},
		{"vite.config.ts", func() {
			frameworks = append(frameworks, Framework{"Vite", "TypeScript", "vite.config.ts", 0.8})
		}},
		{"vite.config.js", func() {
			frameworks = append(frameworks, Framework{"Vite", "JavaScript", "vite.config.js", 0.8})
		}},
		{"webpack.config.js", func() {
			buildTools = append(buildTools, BuildTool{"Webpack", "webpack.config.js"})
		}},
		{"tailwind.config.js", func() {
			frameworks = append(frameworks, Framework{"Tailwind CSS", "JavaScript", "tailwind.config.js", 0.9})
		}},
		{"tailwind.config.ts", func() {
			frameworks = append(frameworks, Framework{"Tailwind CSS", "TypeScript", "tailwind.config.ts", 0.9})
		}},
		{".goreleaser.yaml", func() {
			buildTools = append(buildTools, BuildTool{"GoReleaser", ".goreleaser.yaml"})
		}},
		{".goreleaser.yml", func() {
			buildTools = append(buildTools, BuildTool{"GoReleaser", ".goreleaser.yml"})
		}},

		// Test frameworks
		{"jest.config.js", func() {
			testFrames = append(testFrames, TestFramework{"Jest", "JavaScript", "jest.config.js"})
		}},
		{"jest.config.ts", func() {
			testFrames = append(testFrames, TestFramework{"Jest", "TypeScript", "jest.config.ts"})
		}},
		{"vitest.config.ts", func() {
			testFrames = append(testFrames, TestFramework{"Vitest", "TypeScript", "vitest.config.ts"})
		}},
		{"vitest.config.js", func() {
			testFrames = append(testFrames, TestFramework{"Vitest", "JavaScript", "vitest.config.js"})
		}},
		{"pytest.ini", func() {
			testFrames = append(testFrames, TestFramework{"pytest", "Python", "pytest.ini"})
		}},
		{"conftest.py", func() {
			testFrames = append(testFrames, TestFramework{"pytest", "Python", "conftest.py"})
		}},
		{".rspec", func() {
			testFrames = append(testFrames, TestFramework{"RSpec", "Ruby", ".rspec"})
		}},
		{"phpunit.xml", func() {
			testFrames = append(testFrames, TestFramework{"PHPUnit", "PHP", "phpunit.xml"})
		}},
		{"playwright.config.ts", func() {
			testFrames = append(testFrames, TestFramework{"Playwright", "TypeScript", "playwright.config.ts"})
		}},
		{"playwright.config.js", func() {
			testFrames = append(testFrames, TestFramework{"Playwright", "JavaScript", "playwright.config.js"})
		}},
		{"playwright.config.mts", func() {
			testFrames = append(testFrames, TestFramework{"Playwright", "TypeScript", "playwright.config.mts"})
		}},
		{"vitest.config.mts", func() {
			testFrames = append(testFrames, TestFramework{"Vitest", "TypeScript", "vitest.config.mts"})
		}},
		{"cypress.config.ts", func() {
			testFrames = append(testFrames, TestFramework{"Cypress", "TypeScript", "cypress.config.ts"})
		}},
		{"cypress.config.js", func() {
			testFrames = append(testFrames, TestFramework{"Cypress", "JavaScript", "cypress.config.js"})
		}},
	}

	for _, m := range markers {
		if fileExists(root, m.file) {
			m.action()
		}
	}

	// Detect frameworks from package.json dependencies
	if fileExists(root, "package.json") {
		pkgJSON := readFileStr(root, "package.json")
		if pkgJSON != "" {
			pkgFrameworks := detectFromPackageJSON(pkgJSON)
			frameworks = append(frameworks, pkgFrameworks...)
			testFrames = append(testFrames, detectTestFromPackageJSON(pkgJSON, testFrames)...)
		}
	}

	// Detect JVM test frameworks from Gradle / Maven build files.
	testFrames = append(testFrames, detectJVMTestFrameworks(root, testFrames)...)

	return frameworks, buildTools, pkgMgrs, linters, testFrames
}

// detectTestFromPackageJSON catches test frameworks that live as
// package.json dependencies without a root-level config file (Vitest
// configured inline in vite.config, Playwright configured via project
// dirs, Jest configured under "jest" key, etc.). Avoids duplicating
// already-detected frames.
func detectTestFromPackageJSON(content string, existing []TestFramework) []TestFramework {
	already := make(map[string]bool, len(existing))
	for _, tf := range existing {
		already[tf.Name] = true
	}
	checks := []struct {
		dep, name, lang string
	}{
		{`"vitest"`, "Vitest", "JavaScript"},
		{`"@playwright/test"`, "Playwright", "JavaScript"},
		{`"jest"`, "Jest", "JavaScript"},
		{`"@jest/core"`, "Jest", "JavaScript"},
		{`"cypress"`, "Cypress", "JavaScript"},
		{`"mocha"`, "Mocha", "JavaScript"},
		{`"ava"`, "AVA", "JavaScript"},
	}
	var out []TestFramework
	for _, c := range checks {
		if !already[c.name] && strings.Contains(content, c.dep) {
			out = append(out, TestFramework{c.name, c.lang, "package.json"})
			already[c.name] = true
		}
	}
	return out
}

// detectJVMTestFrameworks looks at Gradle/Maven build files plus the
// conventional src/test/{groovy,kotlin,java} directories to pick up
// Spock, JUnit, Kotest, and TestNG without depending on a root-level
// config file (these frameworks don't use one).
func detectJVMTestFrameworks(root string, existing []TestFramework) []TestFramework {
	already := make(map[string]bool, len(existing))
	for _, tf := range existing {
		already[tf.Name] = true
	}

	var buildContent string
	for _, name := range []string{"build.gradle", "build.gradle.kts", "pom.xml"} {
		if fileExists(root, name) {
			buildContent += readFileStr(root, name) + "\n"
		}
	}

	add := func(name, lang, indicator string) []TestFramework {
		if already[name] {
			return nil
		}
		already[name] = true
		return []TestFramework{{name, lang, indicator}}
	}

	var out []TestFramework
	if strings.Contains(buildContent, "spock-core") || strings.Contains(buildContent, "org.spockframework") {
		out = append(out, add("Spock", "Groovy", "build.gradle")...)
	} else if dirExists(root, "src/test/groovy") {
		out = append(out, add("Spock", "Groovy", "src/test/groovy")...)
	}
	if strings.Contains(buildContent, "io.kotest") {
		out = append(out, add("Kotest", "Kotlin", "build.gradle")...)
	}
	if strings.Contains(buildContent, "org.testng") || strings.Contains(buildContent, "<artifactId>testng</artifactId>") {
		out = append(out, add("TestNG", "Java", "build.gradle/pom.xml")...)
	}
	if strings.Contains(buildContent, "junit-jupiter") || strings.Contains(buildContent, "<artifactId>junit") {
		out = append(out, add("JUnit", "Java", "build.gradle/pom.xml")...)
	} else if dirExists(root, "src/test/java") || dirExists(root, "src/test/kotlin") {
		out = append(out, add("JUnit", "Java", "src/test")...)
	}
	return out
}

func detectFromPackageJSON(content string) []Framework {
	var frameworks []Framework

	// Simple substring matching — no JSON parsing needed for detection
	checks := []struct {
		dep  string
		name string
		lang string
	}{
		{`"react"`, "React", "JavaScript"},
		{`"vue"`, "Vue.js", "JavaScript"},
		{`"@angular/core"`, "Angular", "TypeScript"},
		{`"svelte"`, "Svelte", "JavaScript"},
		{`"express"`, "Express", "JavaScript"},
		{`"fastify"`, "Fastify", "JavaScript"},
		{`"koa"`, "Koa", "JavaScript"},
		{`"nestjs"`, "NestJS", "TypeScript"},
		{`"@nestjs/core"`, "NestJS", "TypeScript"},
		{`"django"`, "Django", "Python"},
		{`"flask"`, "Flask", "Python"},
		{`"prisma"`, "Prisma", "TypeScript"},
		{`"@prisma/client"`, "Prisma", "TypeScript"},
		{`"drizzle-orm"`, "Drizzle", "TypeScript"},
		{`"sequelize"`, "Sequelize", "JavaScript"},
		{`"typeorm"`, "TypeORM", "TypeScript"},
		{`"electron"`, "Electron", "JavaScript"},
		{`"react-native"`, "React Native", "JavaScript"},
		{`"expo"`, "Expo", "JavaScript"},
		{`"gatsby"`, "Gatsby", "JavaScript"},
		{`"remix"`, "Remix", "JavaScript"},
		{`"@remix-run"`, "Remix", "JavaScript"},
		{`"astro"`, "Astro", "JavaScript"},
		{`"hono"`, "Hono", "TypeScript"},
	}

	seen := map[string]bool{}
	for _, c := range checks {
		if strings.Contains(content, c.dep) && !seen[c.name] {
			frameworks = append(frameworks, Framework{
				Name:       c.name,
				Language:   c.lang,
				Indicator:  "package.json",
				Confidence: 0.95,
			})
			seen[c.name] = true
		}
	}

	return frameworks
}

func detectCI(root string) []CIProvider {
	var providers []CIProvider

	checks := []struct {
		path string
		name string
	}{
		{".github/workflows", "GitHub Actions"},
		{".gitlab-ci.yml", "GitLab CI"},
		{".circleci/config.yml", "CircleCI"},
		{"Jenkinsfile", "Jenkins"},
		{".travis.yml", "Travis CI"},
		{"bitbucket-pipelines.yml", "Bitbucket Pipelines"},
		{"azure-pipelines.yml", "Azure Pipelines"},
		{"buildkite.yml", "Buildkite"},
		{".buildkite/pipeline.yml", "Buildkite"},
		{"cloudbuild.yaml", "Google Cloud Build"},
		{"appveyor.yml", "AppVeyor"},
	}

	for _, c := range checks {
		full := filepath.Join(root, c.path)
		if info, err := os.Stat(full); err == nil {
			configFile := c.path
			if info.IsDir() {
				// For GitHub Actions, list the workflow files
				entries, _ := os.ReadDir(full)
				for _, e := range entries {
					if strings.HasSuffix(e.Name(), ".yml") || strings.HasSuffix(e.Name(), ".yaml") {
						configFile = filepath.Join(c.path, e.Name())
						break
					}
				}
			}
			providers = append(providers, CIProvider{Name: c.name, ConfigFile: configFile})
		}
	}

	return providers
}

func detectMonorepo(root string, topDirs []string) bool {
	// Check for monorepo markers
	if fileExists(root, "lerna.json") || fileExists(root, "nx.json") ||
		fileExists(root, "turbo.json") || fileExists(root, "pnpm-workspace.yaml") {
		return true
	}

	// Check for packages/ or apps/ directories
	for _, d := range topDirs {
		if d == "packages" || d == "apps" || d == "services" || d == "libs" {
			return true
		}
	}

	// Check for workspaces in package.json
	if fileExists(root, "package.json") {
		content := readFileStr(root, "package.json")
		if strings.Contains(content, `"workspaces"`) {
			return true
		}
	}

	return false
}

func detectDocFiles(root string) []string {
	var docs []string
	candidates := []string{
		"README.md", "README.rst", "README.txt", "README",
		"CONTRIBUTING.md", "CONTRIBUTING.rst",
		"CHANGELOG.md", "CHANGELOG.rst", "CHANGES.md",
		"CODE_OF_CONDUCT.md",
		"LICENSE", "LICENSE.md", "LICENSE.txt",
		"SECURITY.md",
		"ARCHITECTURE.md",
		"GETTING-STARTED.md",
	}
	for _, c := range candidates {
		if fileExists(root, c) {
			docs = append(docs, c)
		}
	}
	return docs
}

// Summary returns a human-readable summary of the scan result.
func (r *Result) Summary() string {
	var sb strings.Builder

	sb.WriteString("Project Scan Results\n")
	sb.WriteString(strings.Repeat("─", 40) + "\n")

	// Languages
	if len(r.Languages) > 0 {
		sb.WriteString("\nLanguages:\n")
		for _, l := range r.Languages {
			pct := int(l.Confidence * 100)
			sb.WriteString(fmt.Sprintf("  %-20s %d files (%d%%)\n", l.Name, l.FileCount, pct))
		}
	}

	// Frameworks
	if len(r.Frameworks) > 0 {
		sb.WriteString("\nFrameworks:\n")
		for _, f := range r.Frameworks {
			sb.WriteString(fmt.Sprintf("  %-20s (%s, detected from %s)\n", f.Name, f.Language, f.Indicator))
		}
	}

	// Build tools
	if len(r.BuildTools) > 0 {
		sb.WriteString("\nBuild Tools:\n")
		for _, b := range r.BuildTools {
			sb.WriteString(fmt.Sprintf("  %-20s %s\n", b.Name, b.ConfigFile))
		}
	}

	// Package managers
	if len(r.PackageMgrs) > 0 {
		sb.WriteString("\nPackage Managers:\n")
		for _, p := range r.PackageMgrs {
			sb.WriteString(fmt.Sprintf("  %-20s %s (%s)\n", p.Name, p.ConfigFile, p.Language))
		}
	}

	// CI/CD
	if len(r.CIProviders) > 0 {
		sb.WriteString("\nCI/CD:\n")
		for _, c := range r.CIProviders {
			sb.WriteString(fmt.Sprintf("  %-20s %s\n", c.Name, c.ConfigFile))
		}
	}

	// Linters
	if len(r.Linters) > 0 {
		sb.WriteString("\nLinters / Formatters:\n")
		for _, l := range r.Linters {
			sb.WriteString(fmt.Sprintf("  %-20s %s\n", l.Name, l.ConfigFile))
		}
	}

	// Test frameworks
	if len(r.TestFrames) > 0 {
		sb.WriteString("\nTest Frameworks:\n")
		for _, t := range r.TestFrames {
			sb.WriteString(fmt.Sprintf("  %-20s %s\n", t.Name, t.Indicator))
		}
	}

	// Structure
	sb.WriteString("\nProject Structure:\n")
	if r.Structure.IsMonorepo {
		sb.WriteString("  Monorepo: yes\n")
	}
	if r.Structure.HasDocker {
		sb.WriteString("  Docker: yes\n")
	}
	if r.Structure.HasCompose {
		sb.WriteString("  Docker Compose: yes\n")
	}
	if len(r.Structure.TopLevelDirs) > 0 {
		sb.WriteString(fmt.Sprintf("  Top-level dirs: %s\n", strings.Join(r.Structure.TopLevelDirs, ", ")))
	}
	if len(r.Structure.EntryPoints) > 0 {
		sb.WriteString(fmt.Sprintf("  Entry points: %s\n", strings.Join(r.Structure.EntryPoints, ", ")))
	}

	// Docs
	if len(r.DocFiles) > 0 {
		sb.WriteString(fmt.Sprintf("\nDocumentation: %s\n", strings.Join(r.DocFiles, ", ")))
	}

	// Multi-module info
	if r.MultiModule != nil {
		sb.WriteString(fmt.Sprintf("\nMulti-Module Project (%s):\n", r.MultiModule.BuildSystem))
		sb.WriteString(fmt.Sprintf("  Config: %s\n", r.MultiModule.RootConfig))
		sb.WriteString(fmt.Sprintf("  Subprojects: %d\n", len(r.MultiModule.Subprojects)))
		for _, sp := range r.MultiModule.Subprojects {
			lang := ""
			if sp.Language != "" {
				lang = fmt.Sprintf(" [%s]", sp.Language)
			}
			kind := ""
			if sp.Kind != "" && sp.Kind != "gradle-module" {
				kind = fmt.Sprintf(" (%s)", sp.Kind)
			}
			sb.WriteString(fmt.Sprintf("    %s%s%s\n", sp.Name, lang, kind))
		}
	}

	// Framework details
	if len(r.FrameworkDetails) > 0 {
		sb.WriteString("\nFramework Details:\n")
		for _, fd := range r.FrameworkDetails {
			ver := ""
			if fd.Version != "" {
				ver = fmt.Sprintf(" %s", fd.Version)
			}
			sb.WriteString(fmt.Sprintf("  %s%s\n", fd.Name, ver))
			if len(fd.ConfigFiles) > 0 {
				sb.WriteString(fmt.Sprintf("    Config: %s\n", strings.Join(fd.ConfigFiles, ", ")))
			}
		}
	}

	// Import sources
	if len(r.ImportSources) > 0 {
		sb.WriteString("\nExisting Knowledge Files:\n")
		for _, is := range r.ImportSources {
			sb.WriteString(fmt.Sprintf("  [%s] %s\n", is.Kind, is.Path))
		}
	}

	return sb.String()
}

// PrimaryLanguage returns the most-used language, or "" if none detected.
func (r *Result) PrimaryLanguage() string {
	if len(r.Languages) == 0 {
		return ""
	}
	return r.Languages[0].Name
}

// StackSkills returns the Hero skill names that match the detected stack.
func (r *Result) StackSkills() []string {
	skillMap := map[string]string{
		"Go":         "go-stack",
		"Java":       "java-stack",
		"Groovy":     "groovy-stack",
		"Rust":       "rust-stack",
		"Python":     "python-stack",
		"JavaScript": "javascript-stack",
		"TypeScript": "javascript-stack",
	}

	seen := map[string]bool{}
	var skills []string

	for _, lang := range r.Languages {
		if skill, ok := skillMap[lang.Name]; ok && !seen[skill] {
			skills = append(skills, skill)
			seen[skill] = true
		}
	}

	// Check for React
	for _, fw := range r.Frameworks {
		if fw.Name == "React" || fw.Name == "React Native" || fw.Name == "Next.js" {
			if !seen["react-stack"] {
				skills = append(skills, "react-stack")
				seen["react-stack"] = true
			}
		}
	}

	return skills
}

// Helpers

func fileExists(root, name string) bool {
	_, err := os.Stat(filepath.Join(root, name))
	return err == nil
}

func readFileStr(root, name string) string {
	data, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		return ""
	}
	return string(data)
}

func dirContainsGlob(root, pattern string) bool {
	matches, _ := filepath.Glob(filepath.Join(root, pattern))
	return len(matches) > 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
