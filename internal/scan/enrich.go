package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Enrichment holds deep project context extracted by reading source files.
// It augments a scan Result with content that static marker detection cannot produce.
type Enrichment struct {
	ModulePath   string            // Go module path from go.mod (e.g. "github.com/foo/bar")
	ModuleGoVer  string            // Go version from go.mod
	GoRequires   []GoRequire       // direct dependencies from go.mod
	PackageDocs  []PackageDoc      // per-package doc comments
	PackageDirs  []PackageDir      // per-directory package summaries
	ReadmeText   string            // README content (first 3000 chars)
	NPMName      string            // name from package.json
	NPMVersion   string            // version from package.json
	NPMDeps      []NPMDep          // dependencies from package.json
	NPMDevDeps   []NPMDep          // devDependencies from package.json
	NPMScripts   map[string]string // scripts from package.json
	GoTestCount  int               // number of _test.go files found
	GoPackages   []string          // list of unique Go package names
	CargoName    string            // crate name from Cargo.toml
	CargoVersion string            // crate version from Cargo.toml
}

// GoRequire is a direct dependency from go.mod.
type GoRequire struct {
	Path     string
	Version  string
	Indirect bool
}

// PackageDoc is the doc comment for a Go package.
type PackageDoc struct {
	ImportPath string // e.g. "internal/api"
	Dir        string // relative directory
	Doc        string // extracted package-level comment
}

// PackageDir is a summary of a directory's role, inferred from its contents.
type PackageDir struct {
	Dir       string // relative path
	Label     string // short description inferred from contents
	FileCount int
}

// NPMDep is a single npm dependency entry.
type NPMDep struct {
	Name    string
	Version string
}

// Enrich performs a deep read of source files in the project root and returns
// an Enrichment that can be used to produce a richer project overview.
// It is called after Analyze and uses the Result to guide what to read.
// All errors are handled gracefully — partial results are always returned.
func Enrich(root string, r *Result) *Enrichment {
	e := &Enrichment{
		NPMScripts: map[string]string{},
	}

	// Read README
	for _, docFile := range r.DocFiles {
		if strings.HasPrefix(strings.ToLower(docFile), "readme") {
			e.ReadmeText = truncate(readFileStr(root, docFile), 3000)
			break
		}
	}

	// Parse go.mod
	if fileExists(root, "go.mod") {
		parseGoMod(readFileStr(root, "go.mod"), e)
	}

	// Parse package.json
	if fileExists(root, "package.json") {
		parsePackageJSON(readFileStr(root, "package.json"), e)
	}

	// Parse Cargo.toml
	if fileExists(root, "Cargo.toml") {
		parseCargoToml(readFileStr(root, "Cargo.toml"), e)
	}

	// Walk Go source for package doc comments, test count, and package names
	if r.PrimaryLanguage() == "Go" || hasLanguage(r, "Go") {
		walkGoPackages(root, r, e)
	}

	// Summarize top-level directories
	e.PackageDirs = summarizeDirs(root, r.Structure.TopLevelDirs)

	return e
}

// parseGoMod extracts module path, Go version, and direct dependencies from go.mod content.
func parseGoMod(content string, e *Enrichment) {
	lines := strings.Split(content, "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "module ") {
			e.ModulePath = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			continue
		}
		if strings.HasPrefix(line, "go ") && e.ModuleGoVer == "" {
			e.ModuleGoVer = strings.TrimSpace(strings.TrimPrefix(line, "go "))
			continue
		}

		// require block
		if line == "require (" {
			inRequire = true
			continue
		}
		if line == ")" && inRequire {
			inRequire = false
			continue
		}
		// single-line require
		if strings.HasPrefix(line, "require ") {
			dep := strings.TrimPrefix(line, "require ")
			if req, ok := parseRequireLine(dep); ok {
				e.GoRequires = append(e.GoRequires, req)
			}
			continue
		}
		if inRequire && line != "" && !strings.HasPrefix(line, "//") {
			if req, ok := parseRequireLine(line); ok {
				e.GoRequires = append(e.GoRequires, req)
			}
		}
	}

	// Sort: direct first, then alphabetical
	sort.Slice(e.GoRequires, func(i, j int) bool {
		if e.GoRequires[i].Indirect != e.GoRequires[j].Indirect {
			return !e.GoRequires[i].Indirect
		}
		return e.GoRequires[i].Path < e.GoRequires[j].Path
	})
}

var reRequireLine = regexp.MustCompile(`^(\S+)\s+(\S+)(\s+//\s*indirect)?`)

func parseRequireLine(line string) (GoRequire, bool) {
	m := reRequireLine.FindStringSubmatch(line)
	if m == nil {
		return GoRequire{}, false
	}
	return GoRequire{
		Path:     m[1],
		Version:  m[2],
		Indirect: strings.Contains(line, "// indirect"),
	}, true
}

// parsePackageJSON extracts name, version, deps, devDeps, and scripts from package.json content.
func parsePackageJSON(content string, e *Enrichment) {
	// name
	if m := regexp.MustCompile(`"name"\s*:\s*"([^"]+)"`).FindStringSubmatch(content); len(m) == 2 {
		e.NPMName = m[1]
	}
	// version
	if m := regexp.MustCompile(`"version"\s*:\s*"([^"]+)"`).FindStringSubmatch(content); len(m) == 2 {
		e.NPMVersion = m[1]
	}
	// dependencies block
	e.NPMDeps = extractNPMDeps(content, "dependencies")
	e.NPMDevDeps = extractNPMDeps(content, "devDependencies")
	// scripts
	e.NPMScripts = extractNPMScripts(content)
}

// extractNPMDeps extracts name→version pairs from a named JSON object block.
// Uses simple line-by-line parsing, not full JSON, to avoid importing encoding/json.
func extractNPMDeps(content, blockName string) []NPMDep {
	// Find the block: "dependencies": {
	startMarker := fmt.Sprintf(`"%s"`, blockName)
	idx := strings.Index(content, startMarker)
	if idx == -1 {
		return nil
	}
	// Find opening brace after marker
	rest := content[idx+len(startMarker):]
	braceIdx := strings.Index(rest, "{")
	if braceIdx == -1 {
		return nil
	}
	rest = rest[braceIdx+1:]
	// Read until closing brace (first unmatched })
	depth := 1
	end := 0
	for i, ch := range rest {
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				end = i
				break
			}
		}
	}
	block := rest[:end]

	// Extract "name": "version" pairs
	var deps []NPMDep
	re := regexp.MustCompile(`"([^"]+)"\s*:\s*"([^"]+)"`)
	for _, m := range re.FindAllStringSubmatch(block, -1) {
		deps = append(deps, NPMDep{Name: m[1], Version: m[2]})
	}
	return deps
}

// extractNPMScripts extracts the scripts block from package.json.
func extractNPMScripts(content string) map[string]string {
	scripts := map[string]string{}
	idx := strings.Index(content, `"scripts"`)
	if idx == -1 {
		return scripts
	}
	rest := content[idx+len(`"scripts"`):]
	braceIdx := strings.Index(rest, "{")
	if braceIdx == -1 {
		return scripts
	}
	rest = rest[braceIdx+1:]
	depth := 1
	end := 0
	for i, ch := range rest {
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				end = i
				break
			}
		}
	}
	block := rest[:end]
	re := regexp.MustCompile(`"([^"]+)"\s*:\s*"([^"]+)"`)
	for _, m := range re.FindAllStringSubmatch(block, -1) {
		scripts[m[1]] = m[2]
	}
	return scripts
}

// parseCargoToml extracts name and version from Cargo.toml content.
func parseCargoToml(content string, e *Enrichment) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name") {
			if m := regexp.MustCompile(`name\s*=\s*"([^"]+)"`).FindStringSubmatch(line); len(m) == 2 {
				e.CargoName = m[1]
			}
		}
		if strings.HasPrefix(line, "version") {
			if m := regexp.MustCompile(`version\s*=\s*"([^"]+)"`).FindStringSubmatch(line); len(m) == 2 {
				e.CargoVersion = m[1]
			}
		}
	}
}

// walkGoPackages walks the project to find package doc comments, count test files,
// and collect unique package names. Reads only the first .go file per directory.
func walkGoPackages(root string, r *Result, e *Enrichment) {
	pkgNames := map[string]bool{}

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		base := filepath.Base(path)
		// Skip hidden dirs, vendor, generated dirs
		if info.IsDir() {
			if base == ".git" || base == ".hero" || base == "node_modules" ||
				base == "vendor" || base == "__pycache__" || base == ".next" ||
				base == "dist" || base == "build" || base == "target" ||
				strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		dir := filepath.Dir(rel)

		// Count test files
		if strings.HasSuffix(path, "_test.go") {
			e.GoTestCount++
			return nil
		}

		// Read first non-test .go file per directory for package doc + name
		alreadyRead := false
		for _, pd := range e.PackageDocs {
			if pd.Dir == dir {
				alreadyRead = true
				break
			}
		}

		content := readFileStr(root, rel)
		if content == "" {
			return nil
		}

		// Extract package name
		pkgName := extractGoPackageName(content)
		if pkgName != "" && pkgName != "main" {
			pkgNames[pkgName] = true
		}

		if !alreadyRead {
			doc := extractGoPackageDoc(content)
			importPath := ""
			if e.ModulePath != "" && dir != "." {
				importPath = e.ModulePath + "/" + filepath.ToSlash(dir)
			}
			e.PackageDocs = append(e.PackageDocs, PackageDoc{
				ImportPath: importPath,
				Dir:        dir,
				Doc:        doc,
			})
		}

		return nil
	})

	for name := range pkgNames {
		e.GoPackages = append(e.GoPackages, name)
	}
	sort.Strings(e.GoPackages)
}

var rePackageLine = regexp.MustCompile(`(?m)^package\s+(\w+)`)

// extractGoPackageName extracts the package name from a Go source file.
func extractGoPackageName(content string) string {
	m := rePackageLine.FindStringSubmatch(content)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

// extractGoPackageDoc extracts the package-level doc comment from a Go source file.
// Returns the first block comment or line comment sequence before the package declaration.
func extractGoPackageDoc(content string) string {
	lines := strings.Split(content, "\n")
	var commentLines []string
	inBlockComment := false
	var blockLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for end of block comment
		if inBlockComment {
			if idx := strings.Index(trimmed, "*/"); idx >= 0 {
				part := strings.TrimSpace(trimmed[:idx])
				if part != "" {
					blockLines = append(blockLines, part)
				}
				inBlockComment = false
				commentLines = blockLines
			} else {
				// Strip leading * if present
				cleaned := strings.TrimPrefix(trimmed, "* ")
				cleaned = strings.TrimPrefix(cleaned, "*")
				if cleaned != "" {
					blockLines = append(blockLines, cleaned)
				}
			}
			continue
		}

		// Start of block comment
		if strings.HasPrefix(trimmed, "/*") {
			inBlockComment = true
			blockLines = nil
			rest := strings.TrimPrefix(trimmed, "/*")
			if endIdx := strings.Index(rest, "*/"); endIdx >= 0 {
				// Single-line block comment
				part := strings.TrimSpace(rest[:endIdx])
				if part != "" {
					commentLines = []string{part}
				}
				inBlockComment = false
			} else {
				rest = strings.TrimSpace(rest)
				if rest != "" {
					blockLines = append(blockLines, rest)
				}
			}
			continue
		}

		// Line comment
		if strings.HasPrefix(trimmed, "//") {
			cleaned := strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
			commentLines = append(commentLines, cleaned)
			continue
		}

		// Package declaration
		if strings.HasPrefix(trimmed, "package ") {
			break
		}

		// Blank line resets line comment accumulation (but not after we've collected something)
		if trimmed == "" && len(commentLines) > 0 && !inBlockComment {
			// If the next non-blank line is the package declaration, keep these comments
			// Otherwise reset — this handles build tags etc.
			commentLines = nil
		}
	}

	if len(commentLines) == 0 {
		return ""
	}

	// Trim "Package <name>" prefix if present (Go convention)
	result := strings.Join(commentLines, " ")
	if m := regexp.MustCompile(`(?i)^package \w+\s*[–-]?\s*`).FindString(result); m != "" {
		result = strings.TrimSpace(result[len(m):])
	}

	return truncate(result, 200)
}

// summarizeDirs builds PackageDir summaries for the given top-level directories.
func summarizeDirs(root string, topDirs []string) []PackageDir {
	var dirs []PackageDir

	for _, d := range topDirs {
		full := filepath.Join(root, d)
		info, err := os.Stat(full)
		if err != nil || !info.IsDir() {
			continue
		}

		entries, err := os.ReadDir(full)
		if err != nil {
			continue
		}

		fileCount := 0
		var subDirs []string
		var goFiles []string

		for _, e := range entries {
			if e.IsDir() {
				subDirs = append(subDirs, e.Name())
			} else {
				fileCount++
				if strings.HasSuffix(e.Name(), ".go") {
					goFiles = append(goFiles, e.Name())
				}
			}
		}

		label := inferDirLabel(d, subDirs, goFiles)
		dirs = append(dirs, PackageDir{
			Dir:       d,
			Label:     label,
			FileCount: fileCount + len(subDirs),
		})
	}

	return dirs
}

// inferDirLabel infers a short description for a directory from its name and contents.
func inferDirLabel(name string, subDirs, goFiles []string) string {
	// Common Go project layout names
	known := map[string]string{
		"cmd":        "CLI entry points (main packages)",
		"internal":   "internal packages (not exported)",
		"pkg":        "exported library packages",
		"api":        "API definitions (OpenAPI, protobuf, etc.)",
		"web":        "web assets and frontend code",
		"frontend":   "frontend application code",
		"ui":         "user interface components",
		"docs":       "documentation",
		"scripts":    "build and utility scripts",
		"deploy":     "deployment configuration",
		"config":     "application configuration",
		"test":       "integration and end-to-end tests",
		"tests":      "integration and end-to-end tests",
		"migrations": "database migrations",
		"proto":      "protobuf definitions",
		"build":      "build output",
		"dist":       "distribution output",
		"bin":        "compiled binaries",
		"lib":        "library code",
		"src":        "application source code",
	}

	if label, ok := known[strings.ToLower(name)]; ok {
		if len(subDirs) > 0 {
			return fmt.Sprintf("%s (%s)", label, strings.Join(limitSlice(subDirs, 4), ", "))
		}
		return label
	}

	// If it has subdirectories, list them
	if len(subDirs) > 0 {
		return fmt.Sprintf("contains: %s", strings.Join(limitSlice(subDirs, 4), ", "))
	}

	return ""
}

func limitSlice(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return append(s[:n], fmt.Sprintf("+%d more", len(s)-n))
}

func hasLanguage(r *Result, name string) bool {
	for _, l := range r.Languages {
		if l.Name == name {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Try to truncate at a word boundary
	sub := s[:n]
	if idx := strings.LastIndexAny(sub, " \n\t"); idx > n/2 {
		sub = sub[:idx]
	}
	return sub + "…"
}

// GenerateRichProjectOverview creates a richer project-overview context entry
// using both the scan Result and the Enrichment.
func GenerateRichProjectOverview(r *Result, e *Enrichment, heroDir, date string) GeneratedEntry {
	slug := "project-overview"
	path := filepath.Join(heroDir, "knowledge", "context", slug, "spec.md")

	var sb strings.Builder

	// Frontmatter
	sb.WriteString("---\n")
	sb.WriteString("title: Project Overview\n")
	sb.WriteString("type: context\n")
	sb.WriteString("status: active\n")
	sb.WriteString(fmt.Sprintf("created: %s\n", date))
	sb.WriteString("tags: [auto-generated, project-scan]\n")
	sb.WriteString("---\n\n")

	// Project identity from README headline or module path
	if e.ReadmeText != "" {
		headline := extractReadmeHeadline(e.ReadmeText)
		if headline != "" {
			sb.WriteString(fmt.Sprintf("## What is %s\n\n", headline))
			desc := extractReadmeDescription(e.ReadmeText)
			if desc != "" {
				sb.WriteString(desc + "\n\n")
			}
		}
	} else if e.ModulePath != "" {
		sb.WriteString(fmt.Sprintf("## Project: `%s`\n\n", e.ModulePath))
		sb.WriteString("Auto-generated by `hero scan`. Add a README or enrich this section with project context.\n\n")
	} else if e.NPMName != "" {
		sb.WriteString(fmt.Sprintf("## Project: `%s`", e.NPMName))
		if e.NPMVersion != "" {
			sb.WriteString(fmt.Sprintf(" v%s", e.NPMVersion))
		}
		sb.WriteString("\n\n")
		sb.WriteString("Auto-generated by `hero scan`. Add a README or enrich this section with project context.\n\n")
	} else {
		sb.WriteString("Auto-generated by `hero scan`. Edit and enrich with project-specific details.\n\n")
	}

	// Tech stack table
	sb.WriteString("## Tech Stack\n\n")
	sb.WriteString("| Layer | Technology |\n")
	sb.WriteString("|-------|------------|\n")

	// Primary language(s)
	if len(r.Languages) > 0 {
		primary := r.Languages[0]
		goVersion := ""
		if primary.Name == "Go" && e.ModuleGoVer != "" {
			goVersion = fmt.Sprintf(" %s+", e.ModuleGoVer)
		}
		sb.WriteString(fmt.Sprintf("| Language | %s%s |\n", primary.Name, goVersion))
		// Secondary languages (if significant)
		for _, l := range r.Languages[1:] {
			if l.Confidence >= 0.10 {
				sb.WriteString(fmt.Sprintf("| Language (secondary) | %s |\n", l.Name))
			}
		}
	}

	// Frameworks
	for _, f := range r.Frameworks {
		sb.WriteString(fmt.Sprintf("| Framework | %s |\n", f.Name))
	}

	// Build tools
	for _, b := range r.BuildTools {
		sb.WriteString(fmt.Sprintf("| Build | %s (`%s`) |\n", b.Name, b.ConfigFile))
	}

	// Package managers
	for _, p := range r.PackageMgrs {
		sb.WriteString(fmt.Sprintf("| Package manager | %s |\n", p.Name))
	}

	// Test frameworks
	for _, tf := range r.TestFrames {
		sb.WriteString(fmt.Sprintf("| Testing | %s |\n", tf.Name))
	}
	if e.GoTestCount > 0 && len(r.TestFrames) == 0 {
		sb.WriteString(fmt.Sprintf("| Testing | `go test` (%d test files) |\n", e.GoTestCount))
	}

	// Linters
	for _, l := range r.Linters {
		sb.WriteString(fmt.Sprintf("| Linter/Formatter | %s (`%s`) |\n", l.Name, l.ConfigFile))
	}

	// CI
	for _, ci := range r.CIProviders {
		sb.WriteString(fmt.Sprintf("| CI/CD | %s |\n", ci.Name))
	}

	// Docker
	if r.Structure.HasDocker {
		docker := "Docker"
		if r.Structure.HasCompose {
			docker += " + Compose"
		}
		sb.WriteString(fmt.Sprintf("| Container | %s |\n", docker))
	}

	sb.WriteString("\n")

	// Go dependencies
	directDeps := filterGoRequires(e.GoRequires, false)
	if len(directDeps) > 0 {
		sb.WriteString("## Key Dependencies\n\n")
		for _, dep := range directDeps {
			// Short name for display
			short := shortDepName(dep.Path)
			sb.WriteString(fmt.Sprintf("- `%s` %s\n", short, dep.Version))
		}
		sb.WriteString("\n")
	}

	// NPM dependencies
	if len(e.NPMDeps) > 0 {
		sb.WriteString("## Key Dependencies\n\n")
		for _, dep := range limitNPMDeps(e.NPMDeps, 12) {
			sb.WriteString(fmt.Sprintf("- `%s` %s\n", dep.Name, dep.Version))
		}
		sb.WriteString("\n")
	}
	if len(e.NPMDevDeps) > 0 {
		sb.WriteString("### Dev Dependencies\n\n")
		for _, dep := range limitNPMDeps(e.NPMDevDeps, 8) {
			sb.WriteString(fmt.Sprintf("- `%s` %s\n", dep.Name, dep.Version))
		}
		sb.WriteString("\n")
	}

	// Package organization
	if len(e.PackageDirs) > 0 || len(e.PackageDocs) > 0 {
		sb.WriteString("## Package Organization\n\n")

		// Top-level directory table
		if len(e.PackageDirs) > 0 {
			for _, pd := range e.PackageDirs {
				if pd.Label != "" {
					sb.WriteString(fmt.Sprintf("- `%s/` — %s\n", pd.Dir, pd.Label))
				} else {
					sb.WriteString(fmt.Sprintf("- `%s/`\n", pd.Dir))
				}
			}
			sb.WriteString("\n")
		}

		// Internal package docs (if any have real doc comments)
		docEntries := filterPackageDocs(e.PackageDocs)
		if len(docEntries) > 0 {
			sb.WriteString("### Internal Packages\n\n")
			for _, pd := range docEntries {
				relDir := filepath.ToSlash(pd.Dir)
				if pd.Doc != "" {
					sb.WriteString(fmt.Sprintf("- `%s/` — %s\n", relDir, pd.Doc))
				}
			}
			sb.WriteString("\n")
		}
	}

	// Project structure
	sb.WriteString("## Project Structure\n\n")

	if r.Structure.IsMonorepo {
		sb.WriteString("This is a **monorepo**.\n\n")
	}

	if len(r.Structure.EntryPoints) > 0 {
		sb.WriteString("Entry points:\n")
		for _, ep := range r.Structure.EntryPoints {
			sb.WriteString(fmt.Sprintf("- `%s`\n", ep))
		}
		sb.WriteString("\n")
	}

	// Module / project identity
	if e.ModulePath != "" {
		sb.WriteString(fmt.Sprintf("Go module: `%s`\n\n", e.ModulePath))
	}
	if e.CargoName != "" {
		sb.WriteString(fmt.Sprintf("Crate: `%s` v%s\n\n", e.CargoName, e.CargoVersion))
	}

	// NPM scripts
	if len(e.NPMScripts) > 0 {
		sb.WriteString("### NPM Scripts\n\n")
		for name, cmd := range e.NPMScripts {
			sb.WriteString(fmt.Sprintf("- `npm run %s` — `%s`\n", name, truncate(cmd, 80)))
		}
		sb.WriteString("\n")
	}

	// Documentation files
	if len(r.DocFiles) > 0 {
		sb.WriteString("## Documentation\n\n")
		for _, d := range r.DocFiles {
			sb.WriteString(fmt.Sprintf("- `%s`\n", d))
		}
		sb.WriteString("\n")
	}

	// Gaps / enrichment prompt
	sb.WriteString("## Current Gaps\n\n")
	gaps := detectGaps(r, e)
	if len(gaps) > 0 {
		for _, g := range gaps {
			sb.WriteString(fmt.Sprintf("- **%s** — %s\n", g.name, g.desc))
		}
	} else {
		sb.WriteString("No obvious gaps detected.\n")
	}
	sb.WriteString("\n")

	sb.WriteString("<!-- Add project-specific context here:\n")
	sb.WriteString("- Architecture overview and key design patterns\n")
	sb.WriteString("- Deployment topology (cloud provider, regions, etc.)\n")
	sb.WriteString("- Important environment variables\n")
	sb.WriteString("- Third-party service dependencies\n")
	sb.WriteString("-->\n")

	return GeneratedEntry{
		Type:    "context",
		Slug:    slug,
		Path:    path,
		Content: sb.String(),
	}
}

type gap struct {
	name string
	desc string
}

func detectGaps(r *Result, e *Enrichment) []gap {
	var gaps []gap

	if e.GoTestCount == 0 && len(r.TestFrames) == 0 {
		gaps = append(gaps, gap{"No tests", "no test files or test framework detected"})
	}
	if len(r.CIProviders) == 0 {
		gaps = append(gaps, gap{"No CI/CD", "no CI provider detected"})
	}
	if len(r.Linters) == 0 {
		gaps = append(gaps, gap{"No linters", "no linter or formatter configuration detected"})
	}
	if !r.Structure.HasDocker {
		// Only flag Docker absence for backend-looking projects
		if r.PrimaryLanguage() == "Go" || r.PrimaryLanguage() == "Rust" || r.PrimaryLanguage() == "Java" {
			gaps = append(gaps, gap{"No Dockerfile", "no containerized build/deploy configuration"})
		}
	}
	hasReadme := false
	for _, d := range r.DocFiles {
		if strings.HasPrefix(strings.ToLower(d), "readme") {
			hasReadme = true
		}
	}
	if !hasReadme {
		gaps = append(gaps, gap{"No README", "add a README to describe the project"})
	}

	return gaps
}

// extractReadmeHeadline returns the first H1 heading text from README content.
func extractReadmeHeadline(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

// extractReadmeDescription returns the first non-heading paragraph from README content.
func extractReadmeDescription(text string) string {
	lines := strings.Split(text, "\n")
	var para []string
	inPara := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip headings, badges, empty lines before content
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "![") || strings.HasPrefix(trimmed, "[![") {
			if inPara {
				break
			}
			continue
		}
		if trimmed == "" {
			if inPara {
				break
			}
			continue
		}
		inPara = true
		para = append(para, trimmed)
		if len(para) >= 4 { // cap at ~4 lines
			break
		}
	}

	result := strings.Join(para, " ")
	return truncate(result, 400)
}

func filterGoRequires(requires []GoRequire, indirect bool) []GoRequire {
	var result []GoRequire
	for _, r := range requires {
		if r.Indirect == indirect {
			result = append(result, r)
		}
	}
	return result
}

func shortDepName(path string) string {
	// Return the last two path segments for readability
	// e.g. "github.com/spf13/cobra" → "spf13/cobra"
	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return path
}

func limitNPMDeps(deps []NPMDep, n int) []NPMDep {
	if len(deps) <= n {
		return deps
	}
	return deps[:n]
}

func filterPackageDocs(docs []PackageDoc) []PackageDoc {
	var result []PackageDoc
	for _, d := range docs {
		if d.Doc != "" && strings.Contains(d.Dir, "/") {
			result = append(result, d)
		}
	}
	// Sort by dir
	sort.Slice(result, func(i, j int) bool {
		return result[i].Dir < result[j].Dir
	})
	return result
}
