package codescan

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SearchResult holds a single search match.
type SearchResult struct {
	PackageName string `json:"package_name"`
	PackagePath string `json:"package_path"`
	SymbolName  string `json:"symbol_name,omitempty"`
	SymbolKind  string `json:"symbol_kind,omitempty"`
	Line        int    `json:"line,omitempty"`
	Signature   string `json:"signature,omitempty"`
	Doc         string `json:"doc,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
}

// SearchOptions controls what to search for.
type SearchOptions struct {
	Query    string // text to search for (symbol name, package name)
	Kind     string // filter by symbol kind: func, type, interface, struct, class, etc. (empty = all)
	Package  string // filter to specific package path
	Exported bool   // only show exported symbols (default false = show all matched)
}

// Search searches the code knowledge directory for matching symbols and packages.
func Search(codeDir string, opts SearchOptions) ([]SearchResult, error) {
	if _, err := os.Stat(codeDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no code knowledge found (run 'hero scan' first)")
	}

	var results []SearchResult

	// Build regex for flexible matching
	var queryRe *regexp.Regexp
	if opts.Query != "" {
		// Support glob-like patterns: Handle* -> Handle.*
		pattern := strings.ReplaceAll(regexp.QuoteMeta(opts.Query), `\*`, `.*`)
		queryRe, _ = regexp.Compile("(?i)" + pattern)
	}

	// Walk code knowledge directories
	entries, err := os.ReadDir(codeDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		if slug == "index" {
			continue // skip the index entry
		}

		specPath := filepath.Join(codeDir, slug, "spec.md")
		content, err := os.ReadFile(specPath)
		if err != nil {
			continue
		}

		pkgName, pkgPath, pkgLang := parsePackageHeader(string(content))

		// Package-level filter
		if opts.Package != "" {
			if !strings.Contains(strings.ToLower(pkgPath), strings.ToLower(opts.Package)) &&
				!strings.Contains(strings.ToLower(pkgName), strings.ToLower(opts.Package)) {
				continue
			}
		}

		// If no query, return package-level results
		if opts.Query == "" {
			results = append(results, SearchResult{
				PackageName: pkgName,
				PackagePath: pkgPath,
			})
			continue
		}

		if queryRe != nil && queryRe.MatchString(pkgName) {
			results = append(results, SearchResult{
				PackageName: pkgName,
				PackagePath: pkgPath,
			})
		}

		// Search symbols in the spec content
		symbolResults := searchSymbolsInSpec(string(content), pkgName, pkgPath, pkgLang, queryRe, opts)
		results = append(results, symbolResults...)
	}

	return results, nil
}

// GetPackageInfo returns the full spec content for a specific package.
func GetPackageInfo(codeDir, packagePath string) (string, error) {
	slug := strings.ReplaceAll(packagePath, "/", "-")
	slug = strings.ReplaceAll(slug, "\\", "-")
	slug = strings.ReplaceAll(slug, ".", "-")
	slug = strings.ToLower(slug)
	if slug == "" || slug == "-" {
		slug = "root"
	}

	specPath := filepath.Join(codeDir, slug, "spec.md")
	content, err := os.ReadFile(specPath)
	if err != nil {
		return "", fmt.Errorf("package %q not found in code knowledge", packagePath)
	}

	// Strip YAML frontmatter
	s := string(content)
	if idx := strings.Index(s, "---\n"); idx >= 0 {
		if end := strings.Index(s[idx+4:], "---\n"); end >= 0 {
			s = strings.TrimSpace(s[idx+4+end+4:])
		}
	}

	return s, nil
}

// GetDependencyGraph returns the dependency graph from the index.
func GetDependencyGraph(codeDir string) (string, error) {
	indexPath := filepath.Join(codeDir, "index", "spec.md")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return "", fmt.Errorf("no code index found (run 'hero scan' first)")
	}

	s := string(content)
	// Extract the Internal Dependencies section
	depStart := strings.Index(s, "## Internal Dependencies")
	if depStart < 0 {
		return "No internal dependencies found.", nil
	}

	// Find the next section
	rest := s[depStart:]
	nextSection := strings.Index(rest[1:], "\n## ")
	if nextSection >= 0 {
		rest = rest[:nextSection+1]
	}

	return rest, nil
}

// GetHotFiles returns the hot files section from the index.
func GetHotFiles(codeDir string) (string, error) {
	indexPath := filepath.Join(codeDir, "index", "spec.md")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return "", fmt.Errorf("no code index found (run 'hero scan' first)")
	}

	s := string(content)
	hotStart := strings.Index(s, "## Hot Files")
	if hotStart < 0 {
		return "No hot files data available.", nil
	}

	rest := s[hotStart:]
	nextSection := strings.Index(rest[1:], "\n## ")
	if nextSection >= 0 {
		rest = rest[:nextSection+1]
	}

	return rest, nil
}

// GetConfigVars returns the environment variables section from the index.
func GetConfigVars(codeDir string) (string, error) {
	indexPath := filepath.Join(codeDir, "index", "spec.md")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return "", fmt.Errorf("no code index found (run 'hero scan' first)")
	}

	s := string(content)
	start := strings.Index(s, "## Environment Variables")
	if start < 0 {
		return "No environment variables detected.", nil
	}

	rest := s[start:]
	nextSection := strings.Index(rest[1:], "\n## ")
	if nextSection >= 0 {
		rest = rest[:nextSection+1]
	}

	return rest, nil
}

// GetOverview returns a summary of all packages.
func GetOverview(codeDir string) (string, error) {
	indexPath := filepath.Join(codeDir, "index", "spec.md")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return "", fmt.Errorf("no code index found (run 'hero scan' first)")
	}

	s := string(content)
	// Strip frontmatter
	if idx := strings.Index(s, "---\n"); idx >= 0 {
		if end := strings.Index(s[idx+4:], "---\n"); end >= 0 {
			s = strings.TrimSpace(s[idx+4+end+4:])
		}
	}

	return s, nil
}

// GetEndpoints returns the API endpoints section from the index.
func GetEndpoints(codeDir string) (string, error) {
	indexPath := filepath.Join(codeDir, "index", "spec.md")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return "", fmt.Errorf("no code index found (run 'hero scan' first)")
	}

	s := string(content)
	start := strings.Index(s, "## API Endpoints")
	if start < 0 {
		return "No API endpoints detected.", nil
	}

	rest := s[start:]
	nextSection := strings.Index(rest[1:], "\n## ")
	if nextSection >= 0 {
		rest = rest[:nextSection+1]
	}

	return rest, nil
}

func parsePackageHeader(content string) (name, path, lang string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") && name == "" {
			name = strings.TrimPrefix(line, "# ")
		}
		if strings.HasPrefix(line, "**Path:**") {
			path = extractBacktick(line)
		}
		if strings.HasPrefix(line, "**Language:**") {
			lang = strings.TrimSpace(strings.TrimPrefix(line, "**Language:**"))
		}
	}
	return
}

func extractBacktick(s string) string {
	start := strings.Index(s, "`")
	if start < 0 {
		return ""
	}
	end := strings.Index(s[start+1:], "`")
	if end < 0 {
		return ""
	}
	return s[start+1 : start+1+end]
}

// searchSymbolsInSpec extracts symbol matches from a package spec.
func searchSymbolsInSpec(content, pkgName, pkgPath, pkgLang string, queryRe *regexp.Regexp, opts SearchOptions) []SearchResult {
	var results []SearchResult

	// Parse symbol lines: "- `name` (line N)" or "- `func (Recv) Name(...)` (line N)"
	symbolLineRe := regexp.MustCompile("`([^`]+)`\\s*\\(line\\s+(\\d+)\\)(?:\\s*—\\s*(.*))?")

	lines := strings.Split(content, "\n")
	var currentKind string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track section headers for kind
		if strings.HasPrefix(trimmed, "### ") {
			section := strings.TrimPrefix(trimmed, "### ")
			currentKind = sectionToKind(section)
			continue
		}

		if !strings.HasPrefix(trimmed, "- `") {
			continue
		}

		m := symbolLineRe.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}

		sig := m[1]
		doc := ""
		if len(m) > 3 {
			doc = m[3]
		}

		// Extract symbol name from signature
		symName := extractSymbolName(sig)
		if symName == "" {
			continue
		}

		// Apply kind filter
		if opts.Kind != "" && currentKind != "" {
			if !strings.EqualFold(opts.Kind, currentKind) {
				continue
			}
		}

		// Apply query filter
		if queryRe != nil && !queryRe.MatchString(symName) && !queryRe.MatchString(sig) {
			continue
		}

		results = append(results, SearchResult{
			PackageName: pkgName,
			PackagePath: pkgPath,
			SymbolName:  symName,
			SymbolKind:  currentKind,
			Signature:   sig,
			Doc:         doc,
		})
	}

	return results
}

func sectionToKind(section string) string {
	switch strings.ToLower(section) {
	case "functions":
		return "func"
	case "methods":
		return "method"
	case "interfaces":
		return "interface"
	case "structs":
		return "struct"
	case "classes":
		return "class"
	case "types":
		return "type"
	case "constants":
		return "const"
	case "variables":
		return "var"
	case "enums":
		return "enum"
	case "traits":
		return "trait"
	default:
		return strings.ToLower(section)
	}
}

func extractSymbolName(sig string) string {
	// Handle Go function signatures: "func (Recv) Name(...)"
	if strings.HasPrefix(sig, "func") {
		parts := strings.Fields(sig)
		for i, p := range parts {
			if i == 0 {
				continue // skip "func"
			}
			if strings.HasPrefix(p, "(") && !strings.Contains(p, ")") {
				continue // skip receiver "(Type)"
			}
			if strings.HasPrefix(p, "(") && strings.Contains(p, ")") {
				continue // skip receiver "(Type)"
			}
			// Extract name from "Name(" or "Name"
			name := strings.TrimRight(p, "(")
			if name != "" && name[0] >= 'A' && name[0] <= 'z' {
				return name
			}
		}
	}
	// Simple name: just the text before any punctuation
	name := strings.FieldsFunc(sig, func(r rune) bool {
		return r == '(' || r == '[' || r == '<' || r == ' '
	})
	if len(name) > 0 {
		return name[0]
	}
	return sig
}
