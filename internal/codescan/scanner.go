package codescan

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/scan"
)

// Scanner orchestrates code analysis across a project.
type Scanner struct {
	cfg         *config.CodeScanConfig
	projectRoot string
	parsers     map[string]Parser // language -> parser
	extToLang   map[string]string // extension -> language
}

// NewScanner creates a scanner with heuristic parsers (default behavior).
func NewScanner(cfg *config.CodeScanConfig, projectRoot string) *Scanner {
	return NewScannerWithMode(cfg, projectRoot, "heuristic")
}

// NewScannerWithMode creates a scanner with the specified parser mode.
// Supported modes: "heuristic" (default), "treesitter", "auto".
// When mode is "treesitter", the tree-sitter CLI is used for languages with
// installed grammars, with heuristic parsers as fallback for unsupported languages.
func NewScannerWithMode(cfg *config.CodeScanConfig, projectRoot, mode string) *Scanner {
	if cfg == nil {
		cfg = config.DefaultCodeScanConfig()
	}

	s := &Scanner{
		cfg:         cfg,
		projectRoot: projectRoot,
		parsers:     make(map[string]Parser),
		extToLang:   make(map[string]string),
	}

	// All heuristic parsers for fallback
	heuristicParsers := []Parser{
		NewGoParser(),
		NewJSTSParser(),
		NewPythonParser(),
		NewJavaParser(),
		NewRustParser(),
		NewCParser(),
		NewRubyParser(),
		NewGroovyParser(),
	}

	if mode == "treesitter" {
		// Use tree-sitter for languages with installed grammars
		ts := NewTreeSitterParser()
		tsLangs := make(map[string]bool)
		for _, lang := range ts.Languages() {
			tsLangs[lang] = true
		}

		// Register tree-sitter for supported languages
		if len(tsLangs) > 0 {
			s.registerParser(ts)
		}

		// Register heuristic parsers for unsupported languages only
		for _, hp := range heuristicParsers {
			for _, lang := range hp.Languages() {
				if !tsLangs[lang] {
					s.parsers[lang] = hp
				}
			}
		}
	} else {
		// Heuristic mode (default)
		for _, hp := range heuristicParsers {
			s.registerParser(hp)
		}
	}

	return s
}

func (s *Scanner) registerParser(p Parser) {
	for _, lang := range p.Languages() {
		s.parsers[lang] = p
	}
}

// RegisterExtension maps a file extension to a language name.
func (s *Scanner) RegisterExtension(ext, lang string) {
	s.extToLang[ext] = lang
}

// languageForExt returns the language for a file extension.
//
// .gsp (Groovy Server Pages) maps to "groovy" so the heuristic Groovy
// parser picks up any `<%= %>` scriptlets, controller/action references,
// and inline class declarations — far from a full template parser, but
// enough to keep GSP files visible in the code graph.
func languageForExt(ext string) string {
	m := map[string]string{
		".go":     "go",
		".js":     "javascript",
		".jsx":    "javascript",
		".mjs":    "javascript",
		".cjs":    "javascript",
		".ts":     "typescript",
		".tsx":    "typescript",
		".mts":    "typescript",
		".py":     "python",
		".pyw":    "python",
		".java":   "java",
		".rs":     "rust",
		".c":      "c",
		".h":      "c",
		".cpp":    "cpp",
		".cc":     "cpp",
		".cxx":    "cpp",
		".hpp":    "cpp",
		".hxx":    "cpp",
		".rb":     "ruby",
		".groovy": "groovy",
		".gvy":    "groovy",
		".gsp":    "groovy",
	}
	return m[ext]
}

// isEndpointOnlyExt returns true for file extensions that have no language parser
// but can still yield API endpoint information (e.g. .proto, .graphql).
func isEndpointOnlyExt(ext string) bool {
	switch ext {
	case ".proto", ".graphql", ".gql":
		return true
	}
	return false
}

// Scan walks the project, parses source files, and returns the result.
// If prevChecksums is provided, only changed files are re-parsed; unchanged
// files are carried forward from prevCache so the merged Result stays complete.
// A nil prevCache (missing/corrupt/legacy) degrades gracefully to a full re-parse.
func (s *Scanner) Scan(prevChecksums map[string]string, prevCache *ScanCache) (*Result, error) {
	return s.ScanContext(context.Background(), prevChecksums, prevCache)
}

// ScanContext is Scan with cancellation propagated through inventory and parse
// boundaries. A returned incomplete Result is diagnostic-only and must not be
// used as an authoritative deletion keep-set.
func (s *Scanner) ScanContext(ctx context.Context, prevChecksums map[string]string, prevCache *ScanCache) (*Result, error) {
	start := time.Now()
	result := &Result{
		ProjectRoot: s.projectRoot,
		Checksums:   make(map[string]string),
		Complete:    true,
	}

	var files []FileInfo

	err := filepath.WalkDir(s.projectRoot, func(path string, d fs.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			result.Complete = false
			result.Diagnostics = append(result.Diagnostics, ScanDiagnostic{
				Path: path, Phase: "walk", Error: err.Error(),
			})
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			if s.cfg.ShouldExclude(name) {
				return filepath.SkipDir
			}
			// Skip hidden dirs (except project root)
			if strings.HasPrefix(name, ".") && path != s.projectRoot {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		lang := languageForExt(ext)
		endpointOnly := lang == "" && isEndpointOnlyExt(ext)
		if lang == "" && !endpointOnly {
			return nil
		}
		if lang != "" {
			if _, ok := s.parsers[lang]; !ok {
				return nil
			}
		}

		relPath, err := filepath.Rel(s.projectRoot, path)
		if err != nil {
			result.Complete = false
			result.Diagnostics = append(result.Diagnostics, ScanDiagnostic{
				Path: path, Phase: "path", Error: err.Error(),
			})
			return nil
		}
		result.Stats.FilesInventoried++

		content, err := os.ReadFile(path)
		if err != nil {
			result.Complete = false
			result.Diagnostics = append(result.Diagnostics, ScanDiagnostic{
				Path: relPath, Phase: "read", Error: err.Error(),
			})
			return nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		sum := fmt.Sprintf("%x", sha256.Sum256(content))
		result.Checksums[relPath] = sum

		if prevChecksums != nil {
			if prev, ok := prevChecksums[relPath]; ok && prev == sum {
				if prevCache != nil {
					if endpointOnly {
						result.ConfigVars = append(result.ConfigVars, prevCache.ConfigVars[relPath]...)
						result.Endpoints = append(result.Endpoints, prevCache.Endpoints[relPath]...)
						result.Stats.Unchanged++
						return nil
					}
					if fi, ok := prevCache.Files[relPath]; ok {
						files = append(files, fi)
						result.ConfigVars = append(result.ConfigVars, prevCache.ConfigVars[relPath]...)
						result.Endpoints = append(result.Endpoints, prevCache.Endpoints[relPath]...)
						result.Stats.Unchanged++
						return nil
					}
				}
			}
		}

		result.Stats.Reparsed++
		if endpointOnly {
			result.Endpoints = append(result.Endpoints, ExtractEndpoints(relPath, content, "")...)
			return nil
		}

		parser := s.parsers[lang]
		fi, err := parser.ParseFile(relPath, content)
		if err != nil {
			result.Complete = false
			result.Diagnostics = append(result.Diagnostics, ScanDiagnostic{
				Path: relPath, Phase: "parse", Error: err.Error(),
			})
			return nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		files = append(files, *fi)

		// Extract config/env vars
		configVars := ExtractConfigVars(relPath, content, lang)
		if len(configVars) > 0 {
			result.ConfigVars = append(result.ConfigVars, configVars...)
		}

		// Extract API endpoints
		endpoints := ExtractEndpoints(relPath, content, lang)
		if len(endpoints) > 0 {
			result.Endpoints = append(result.Endpoints, endpoints...)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking project: %w", err)
	}

	for path, sum := range result.Checksums {
		prev, existed := prevChecksums[path]
		switch {
		case !existed:
			result.Stats.Added++
		case prev != sum:
			result.Stats.Changed++
		}
	}
	for path := range prevChecksums {
		if _, exists := result.Checksums[path]; !exists {
			result.Stats.Deleted++
		}
	}

	result.Files = files

	// Aggregate files into packages
	result.Packages = s.aggregatePackages(files)

	// Build dependency graph
	result.DepGraph = s.buildDepGraph(result.Packages)

	// Compute imported_by
	s.computeImportedBy(result)

	// Detect multi-module structure
	if mm := scan.DetectMultiModule(s.projectRoot); mm != nil {
		result.Modules = s.mapModules(mm, result.Packages)
	}
	result.Stats.Elapsed = time.Since(start)

	return result, nil
}

// aggregatePackages groups files by directory into packages.
func (s *Scanner) aggregatePackages(files []FileInfo) []Package {
	pkgMap := make(map[string]*Package)

	for _, f := range files {
		dir := filepath.Dir(f.Path)
		if dir == "." {
			dir = "."
		}

		pkg, ok := pkgMap[dir]
		if !ok {
			pkg = &Package{
				Path:     dir,
				Language: f.Language,
			}
			pkgMap[dir] = pkg
		}

		// Use the declared package name if available
		if f.Package != "" && pkg.Name == "" {
			pkg.Name = f.Package
		}

		pkg.Files = append(pkg.Files, f.Path)
		pkg.LineCount += f.LineCount
		pkg.FileCount++

		// Collect exported symbols (preserve file path)
		for _, sym := range f.Symbols {
			if sym.Exported {
				sym.File = f.Path
				pkg.Symbols = append(pkg.Symbols, sym)
			}
		}

		// Collect unique imports
		for _, imp := range f.Imports {
			found := false
			for _, existing := range pkg.Imports {
				if existing == imp {
					found = true
					break
				}
			}
			if !found {
				pkg.Imports = append(pkg.Imports, imp)
			}
		}
	}

	// If package has no declared name, use directory basename
	for dir, pkg := range pkgMap {
		if pkg.Name == "" {
			pkg.Name = filepath.Base(dir)
			if pkg.Name == "." {
				pkg.Name = filepath.Base(s.projectRoot)
			}
		}
	}

	// Convert to slice and sort by path
	var packages []Package
	for _, pkg := range pkgMap {
		packages = append(packages, *pkg)
	}
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Path < packages[j].Path
	})

	return packages
}

// buildDepGraph creates edges from package imports.
func (s *Scanner) buildDepGraph(packages []Package) []DepEdge {
	// Build a lookup: package import path -> our package path
	// For Go, this is the module path + dir. For others, it's the import string.
	var edges []DepEdge
	pkgPaths := make(map[string]string) // normalized import -> our dir path

	for _, pkg := range packages {
		pkgPaths[pkg.Path] = pkg.Path
		// Also index by package name for simple matching
		pkgPaths[pkg.Name] = pkg.Path
	}

	for _, pkg := range packages {
		for _, imp := range pkg.Imports {
			// Check if this import refers to an internal package
			for _, target := range packages {
				if strings.HasSuffix(imp, "/"+target.Name) || imp == target.Name ||
					strings.HasSuffix(imp, "/"+target.Path) || imp == target.Path {
					edges = append(edges, DepEdge{
						From: pkg.Path,
						To:   target.Path,
					})
					break
				}
			}
		}
	}

	return edges
}

// mapModules maps detected subprojects to their packages.
func (s *Scanner) mapModules(mm *scan.MultiModuleInfo, packages []Package) []Module {
	var modules []Module
	for _, sp := range mm.Subprojects {
		mod := Module{
			Name: sp.Name,
			Dir:  sp.Dir,
			Kind: sp.Kind,
		}
		for _, pkg := range packages {
			if pkg.Path == sp.Dir || strings.HasPrefix(pkg.Path, sp.Dir+"/") {
				mod.Packages = append(mod.Packages, pkg.Path)
			}
		}
		if len(mod.Packages) > 0 {
			modules = append(modules, mod)
		}
	}
	return modules
}

// computeImportedBy fills ImportedBy for each package based on the dep graph.
func (s *Scanner) computeImportedBy(result *Result) {
	importedBy := make(map[string][]string)
	for _, edge := range result.DepGraph {
		importedBy[edge.To] = append(importedBy[edge.To], edge.From)
	}
	for i := range result.Packages {
		if by, ok := importedBy[result.Packages[i].Path]; ok {
			result.Packages[i].ImportedBy = by
		}
	}
}

// LoadChecksums reads the checksums file from the code knowledge directory.
func LoadChecksums(codeDir string) (map[string]string, error) {
	path := filepath.Join(codeDir, ".checksums.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var checksums map[string]string
	if err := json.Unmarshal(data, &checksums); err != nil {
		return nil, err
	}
	return checksums, nil
}

// SaveChecksums writes the checksums file to the code knowledge directory.
func SaveChecksums(codeDir string, checksums map[string]string) error {
	if err := os.MkdirAll(codeDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(checksums, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(filepath.Join(codeDir, ".checksums.json"), data, 0o644)
}
