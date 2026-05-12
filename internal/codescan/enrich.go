package codescan

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// EnrichmentCache manages LLM-generated descriptions cached by content hash.
type EnrichmentCache struct {
	Entries map[string]EnrichmentEntry `json:"entries"` // key: "pkg:symbol:contentHash"
}

// EnrichmentEntry holds a single cached description.
type EnrichmentEntry struct {
	Description string `json:"description"`
	UpdatedAt   string `json:"updated_at"` // RFC3339
}

// UnenrichedSymbol represents a symbol that lacks an AI description.
type UnenrichedSymbol struct {
	Package       string `json:"package"`
	Symbol        string `json:"symbol"`
	Kind          string `json:"kind"`
	Signature     string `json:"signature"`
	File          string `json:"file"`
	Line          int    `json:"line"`
	SourceContext string `json:"source_context"` // ~20 lines around the symbol from the actual source file
}

// LoadEnrichmentCache reads the enrichment cache from .hero/knowledge/code/.enrichments.json.
func LoadEnrichmentCache(codeDir string) (*EnrichmentCache, error) {
	path := filepath.Join(codeDir, ".enrichments.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &EnrichmentCache{Entries: make(map[string]EnrichmentEntry)}, nil
		}
		return nil, fmt.Errorf("reading enrichment cache: %w", err)
	}
	var cache EnrichmentCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parsing enrichment cache: %w", err)
	}
	if cache.Entries == nil {
		cache.Entries = make(map[string]EnrichmentEntry)
	}
	return &cache, nil
}

// Save writes the enrichment cache to .hero/knowledge/code/.enrichments.json.
func (c *EnrichmentCache) Save(codeDir string) error {
	if err := os.MkdirAll(codeDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(codeDir, ".enrichments.json"), data, 0o644)
}

// Get looks up a description by cache key.
func (c *EnrichmentCache) Get(key string) (string, bool) {
	entry, ok := c.Entries[key]
	if !ok {
		return "", false
	}
	return entry.Description, true
}

// Set stores a description with the current timestamp.
func (c *EnrichmentCache) Set(key, description string) {
	c.Entries[key] = EnrichmentEntry{
		Description: description,
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}
}

// EnrichmentKey generates a cache key as "pkgPath:symbolName:contentHash".
func EnrichmentKey(pkgPath, symbolName, contentHash string) string {
	return pkgPath + ":" + symbolName + ":" + contentHash
}

// GetUnenrichedSymbols scans the knowledge directory for symbols without AI descriptions
// in the enrichment cache. Returns up to limit symbols with their source context.
func GetUnenrichedSymbols(codeDir string, limit int) ([]UnenrichedSymbol, error) {
	if limit <= 0 {
		limit = 20
	}

	cache, err := LoadEnrichmentCache(codeDir)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(codeDir)
	if err != nil {
		return nil, fmt.Errorf("reading code knowledge dir: %w", err)
	}

	// Determine project root from codeDir: strip .hero/knowledge/code
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(codeDir)))

	symbolLineRe := regexp.MustCompile("`([^`]+)`\\s*\\(line\\s+(\\d+)\\)")

	var results []UnenrichedSymbol

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "index" {
			continue
		}

		specPath := filepath.Join(codeDir, entry.Name(), "spec.md")
		content, err := os.ReadFile(specPath)
		if err != nil {
			continue
		}

		pkgName, pkgPath, _ := parsePackageHeader(string(content))
		_ = pkgName

		lines := strings.Split(string(content), "\n")
		var currentKind string
		var currentFiles []string

		// Collect file list from the spec
		inFiles := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "## Files" {
				inFiles = true
				continue
			}
			if inFiles && strings.HasPrefix(trimmed, "- `") {
				f := extractBacktick(trimmed)
				if f != "" {
					currentFiles = append(currentFiles, f)
				}
				continue
			}
			if inFiles && strings.HasPrefix(trimmed, "##") {
				inFiles = false
			}
		}

		// Parse symbols
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			if strings.HasPrefix(trimmed, "### ") {
				currentKind = sectionToKind(strings.TrimPrefix(trimmed, "### "))
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
			var lineNum int
			fmt.Sscanf(m[2], "%d", &lineNum)

			symName := extractSymbolName(sig)
			if symName == "" {
				continue
			}

			// Check if already enriched (use simple key without content hash)
			simpleKey := EnrichmentKey(pkgPath, symName, "")
			if _, ok := cache.Get(simpleKey); ok {
				continue
			}

			// Also check with any content hash prefix match
			enriched := false
			prefix := pkgPath + ":" + symName + ":"
			for k := range cache.Entries {
				if strings.HasPrefix(k, prefix) {
					enriched = true
					break
				}
			}
			if enriched {
				continue
			}

			// Find source file for this symbol
			sourceContext := ""
			if lineNum > 0 && len(currentFiles) > 0 {
				sourceContext = extractSourceContext(projectRoot, currentFiles, lineNum)
			}

			results = append(results, UnenrichedSymbol{
				Package:       pkgPath,
				Symbol:        symName,
				Kind:          currentKind,
				Signature:     sig,
				File:          findFileForLine(currentFiles, lineNum),
				Line:          lineNum,
				SourceContext: sourceContext,
			})

			if len(results) >= limit {
				return results, nil
			}
		}
	}

	return results, nil
}

// extractSourceContext reads ~20 lines around the given line number from source files.
func extractSourceContext(projectRoot string, files []string, lineNum int) string {
	for _, f := range files {
		absPath := filepath.Join(projectRoot, f)
		file, err := os.Open(absPath)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		var allLines []string
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}
		file.Close()

		if lineNum <= 0 || lineNum > len(allLines) {
			continue
		}

		start := lineNum - 10
		if start < 0 {
			start = 0
		}
		end := lineNum + 10
		if end > len(allLines) {
			end = len(allLines)
		}

		return strings.Join(allLines[start:end], "\n")
	}
	return ""
}

// findFileForLine returns the first file from the list (simple heuristic).
func findFileForLine(files []string, lineNum int) string {
	if len(files) > 0 {
		return files[0]
	}
	return ""
}

// GenerateKnowledgeWithEnrichments writes code intelligence with AI descriptions applied.
func GenerateKnowledgeWithEnrichments(result *Result, codeDir string, cache *EnrichmentCache) error {
	if cache == nil {
		return GenerateKnowledge(result, codeDir)
	}

	// Apply enrichments to the result before generating
	for i := range result.Packages {
		pkg := &result.Packages[i]
		for j := range pkg.Symbols {
			sym := &pkg.Symbols[j]
			// Try simple key first (no content hash)
			key := EnrichmentKey(pkg.Path, sym.Name, "")
			if desc, ok := cache.Get(key); ok {
				sym.AIDesc = desc
				continue
			}
			// Try prefix match
			prefix := pkg.Path + ":" + sym.Name + ":"
			for k, entry := range cache.Entries {
				if strings.HasPrefix(k, prefix) {
					sym.AIDesc = entry.Description
					break
				}
			}
		}
	}

	return GenerateKnowledge(result, codeDir)
}
