package codescan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// GenerateKnowledge writes code intelligence results to .hero/knowledge/code/.
// It removes stale package directories from previous runs.
func GenerateKnowledge(result *Result, codeDir string) error {
	if err := GenerateKnowledgeContext(context.Background(), result, codeDir); err != nil {
		return err
	}
	return CommitScanState(result, codeDir, "")
}

// GenerateKnowledgeContext writes generated knowledge without advancing scan
// state. Coordinators call CommitScanState only after every required phase.
func GenerateKnowledgeContext(ctx context.Context, result *Result, codeDir string) error {
	if result == nil || !result.Complete {
		return fmt.Errorf("generating code knowledge requires a complete scan result")
	}
	if err := os.MkdirAll(codeDir, 0o755); err != nil {
		return fmt.Errorf("creating code dir: %w", err)
	}

	// Track which slug directories we write this run
	writtenSlugs := make(map[string]bool)

	// Write per-package files
	for _, pkg := range result.Packages {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := writePackageSpec(pkg, codeDir); err != nil {
			return fmt.Errorf("writing package %s: %w", pkg.Name, err)
		}
		writtenSlugs[slugify(pkg.Path)] = true
	}

	// Write the index/overview
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := writeCodeIndex(result, codeDir); err != nil {
		return fmt.Errorf("writing index: %w", err)
	}
	writtenSlugs["index"] = true

	// Build the prune keep-set from the COMPLETE current file set. On an
	// incremental scan result.Packages holds only changed packages, so
	// writtenSlugs alone would drop every unchanged package and prune its
	// still-valid directory. result.Checksums is recorded for every current
	// file on both full and incremental scans, so slugging each file's
	// directory reconstructs the full set of live package dirs. Genuinely
	// deleted packages are absent from result.Checksums and are still pruned.
	keep := make(map[string]bool, len(writtenSlugs)+len(result.Checksums))
	for slug := range writtenSlugs {
		keep[slug] = true
	}
	for relPath := range result.Checksums {
		keep[slugify(filepath.Dir(relPath))] = true
	}

	// Remove stale directories from previous runs
	if err := pruneStaleDirectoriesContext(ctx, codeDir, keep); err != nil {
		return err
	}

	return nil
}

// pruneStaleDirectories removes subdirectories in codeDir that were not written
// in the current run. Preserves top-level files like .checksums.json.
func pruneStaleDirectoriesContext(ctx context.Context, codeDir string, keep map[string]bool) error {
	entries, err := os.ReadDir(codeDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !e.IsDir() {
			continue
		}
		if !keep[e.Name()] {
			if err := os.RemoveAll(filepath.Join(codeDir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func writePackageSpec(pkg Package, codeDir string) error {
	slug := slugify(pkg.Path)
	dir := filepath.Join(codeDir, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var b strings.Builder

	// Frontmatter
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: \"Package: %s\"\n", pkg.Name))
	b.WriteString("type: context\n")
	b.WriteString("status: active\n")
	b.WriteString(fmt.Sprintf("created: %s\n", time.Now().Format("2006-01-02")))
	b.WriteString("tags: [auto-generated, code-scan]\n")
	b.WriteString("---\n\n")

	// Header
	b.WriteString(fmt.Sprintf("# %s\n\n", pkg.Name))
	b.WriteString(fmt.Sprintf("**Path:** `%s`  \n", pkg.Path))
	b.WriteString(fmt.Sprintf("**Language:** %s  \n", pkg.Language))
	b.WriteString(fmt.Sprintf("**Files:** %d (%d lines)  \n\n", pkg.FileCount, pkg.LineCount))

	if pkg.Doc != "" {
		b.WriteString(pkg.Doc + "\n\n")
	}
	if pkg.AIDesc != "" {
		b.WriteString(pkg.AIDesc + "\n\n")
	}

	// Exported symbols (filtered for signal)
	filtered := filterSymbols(pkg.Symbols)
	if len(filtered) > 0 {
		b.WriteString("## Exported Symbols\n\n")

		// Group by kind
		byKind := make(map[SymbolKind][]Symbol)
		for _, s := range filtered {
			byKind[s.Kind] = append(byKind[s.Kind], s)
		}

		kindOrder := []SymbolKind{SymInterface, SymStruct, SymClass, SymType, SymTrait, SymEnum, SymFunc, SymMethod, SymConst, SymVar}
		for _, kind := range kindOrder {
			syms, ok := byKind[kind]
			if !ok {
				continue
			}
			b.WriteString(fmt.Sprintf("### %s\n\n", kindLabel(kind)))
			for _, s := range syms {
				label := s.Name
				if s.Signature != "" {
					label = s.Signature
				}
				if s.File != "" {
					b.WriteString(fmt.Sprintf("- `%s` in `%s` (line %d)", label, s.File, s.Line))
				} else {
					b.WriteString(fmt.Sprintf("- `%s` (line %d)", label, s.Line))
				}
				if s.Doc != "" {
					b.WriteString(fmt.Sprintf(" — %s", s.Doc))
				}
				if s.AIDesc != "" {
					b.WriteString(fmt.Sprintf(" — %s", s.AIDesc))
				}
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}

	// Imports
	if len(pkg.Imports) > 0 {
		b.WriteString("## Imports\n\n")
		for _, imp := range pkg.Imports {
			b.WriteString(fmt.Sprintf("- `%s`\n", imp))
		}
		b.WriteString("\n")
	}

	// Imported by
	if len(pkg.ImportedBy) > 0 {
		b.WriteString("## Imported By\n\n")
		for _, by := range pkg.ImportedBy {
			b.WriteString(fmt.Sprintf("- `%s`\n", by))
		}
		b.WriteString("\n")
	}

	// Files
	b.WriteString("## Files\n\n")
	for _, f := range pkg.Files {
		b.WriteString(fmt.Sprintf("- `%s`\n", f))
	}
	b.WriteString("\n")

	return os.WriteFile(filepath.Join(dir, "spec.md"), []byte(b.String()), 0o644)
}

func writeCodeIndex(result *Result, codeDir string) error {
	dir := filepath.Join(codeDir, "index")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString("title: Code Structure Index\n")
	b.WriteString("type: context\n")
	b.WriteString("status: active\n")
	b.WriteString(fmt.Sprintf("created: %s\n", time.Now().Format("2006-01-02")))
	b.WriteString("tags: [auto-generated, code-scan, index]\n")
	b.WriteString("---\n\n")

	b.WriteString("# Code Structure Index\n\n")

	// Module overview (if multi-module project)
	if len(result.Modules) > 0 {
		b.WriteString("## Modules\n\n")
		for _, mod := range result.Modules {
			b.WriteString(fmt.Sprintf("- **%s** (`%s`) — %s, %d packages\n", mod.Name, mod.Dir, mod.Kind, len(mod.Packages)))
		}
		b.WriteString("\n")
	}

	// Package summary
	b.WriteString("## Packages\n\n")
	b.WriteString("| Package | Path | Language | Files | Lines | Exported Symbols |\n")
	b.WriteString("|---------|------|----------|-------|-------|------------------|\n")
	for _, pkg := range result.Packages {
		b.WriteString(fmt.Sprintf("| %s | `%s` | %s | %d | %d | %d |\n",
			pkg.Name, pkg.Path, pkg.Language, pkg.FileCount, pkg.LineCount, len(pkg.Symbols)))
	}
	b.WriteString("\n")

	// Dependency graph
	if len(result.DepGraph) > 0 {
		b.WriteString("## Internal Dependencies\n\n")
		b.WriteString("```\n")
		for _, edge := range result.DepGraph {
			b.WriteString(fmt.Sprintf("%s -> %s\n", edge.From, edge.To))
		}
		b.WriteString("```\n\n")
	}

	// Hot files
	if len(result.HotFiles) > 0 {
		b.WriteString("## Hot Files\n\n")
		b.WriteString("Most important files by edit frequency and import count:\n\n")
		for i, hf := range result.HotFiles {
			if i >= 30 {
				break
			}
			b.WriteString(fmt.Sprintf("- `%s` (score: %.1f", hf.Path, hf.Score))
			if hf.Reason != "" {
				b.WriteString(fmt.Sprintf(", %s", hf.Reason))
			}
			b.WriteString(")\n")
		}
		b.WriteString("\n")
	}

	// Environment variables
	if len(result.ConfigVars) > 0 {
		b.WriteString("## Environment Variables\n\n")
		// Group by name
		grouped := make(map[string][]ConfigVar)
		var order []string
		for _, cv := range result.ConfigVars {
			if _, exists := grouped[cv.Name]; !exists {
				order = append(order, cv.Name)
			}
			grouped[cv.Name] = append(grouped[cv.Name], cv)
		}
		sort.Strings(order)
		for _, name := range order {
			refs := grouped[name]
			req := "optional"
			for _, r := range refs {
				if r.Required {
					req = "required"
					break
				}
			}
			b.WriteString(fmt.Sprintf("- `%s` (%s)", name, req))
			if len(refs) == 1 {
				b.WriteString(fmt.Sprintf(" — `%s`:%d", refs[0].File, refs[0].Line))
			} else {
				b.WriteString(fmt.Sprintf(" — %d references", len(refs)))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// API Endpoints
	if len(result.Endpoints) > 0 {
		b.WriteString("## API Endpoints\n\n")

		// Group by protocol
		byProtocol := make(map[string][]Endpoint)
		for _, ep := range result.Endpoints {
			byProtocol[ep.Protocol] = append(byProtocol[ep.Protocol], ep)
		}

		protocolOrder := []string{"rest", "grpc", "graphql", "websocket"}
		protocolLabels := map[string]string{
			"rest":      "REST",
			"grpc":      "gRPC",
			"graphql":   "GraphQL",
			"websocket": "WebSocket",
		}

		for _, proto := range protocolOrder {
			eps, ok := byProtocol[proto]
			if !ok {
				continue
			}
			label := protocolLabels[proto]
			if label == "" {
				label = proto
			}
			b.WriteString(fmt.Sprintf("### %s (%d)\n\n", label, len(eps)))
			for _, ep := range eps {
				b.WriteString(fmt.Sprintf("- `%s %s`", ep.Method, ep.Path))
				if ep.Handler != "" {
					b.WriteString(fmt.Sprintf(" → `%s`", ep.Handler))
				}
				b.WriteString(fmt.Sprintf(" — `%s`:%d\n", ep.File, ep.Line))
			}
			b.WriteString("\n")
		}
	}

	return os.WriteFile(filepath.Join(dir, "spec.md"), []byte(b.String()), 0o644)
}

// trivialNames are method/function names that provide no useful signal.
var trivialNames = map[string]bool{
	"get": true, "set": true, "toString": true, "valueOf": true,
	"constructor": true, "render": true, "componentDidMount": true,
	"componentDidUpdate": true, "componentWillUnmount": true,
	"shouldComponentUpdate": true, "getInitialState": true,
	"getDefaultProps": true, "hashCode": true, "equals": true,
	"clone": true, "finalize": true, "init": true, "close": true,
	"destroy": true, "setUp": true, "tearDown": true, "setup": true,
	"teardown": true, "before": true, "after": true,
}

// symbolKindPriority returns a priority score for sorting (higher = more important).
func symbolKindPriority(k SymbolKind) int {
	switch k {
	case SymInterface, SymTrait:
		return 6
	case SymType, SymStruct, SymClass:
		return 5
	case SymEnum:
		return 4
	case SymFunc:
		return 3
	case SymMethod:
		return 2
	case SymConst:
		return 1
	default:
		return 0
	}
}

// filterSymbols removes trivial symbols and caps output at maxSymbolsPerPackage.
func filterSymbols(syms []Symbol) []Symbol {
	const maxSymbolsPerPackage = 50

	// Remove trivial names (only for methods/functions, not types/classes)
	var kept []Symbol
	for _, s := range syms {
		if (s.Kind == SymFunc || s.Kind == SymMethod) && trivialNames[s.Name] {
			continue
		}
		kept = append(kept, s)
	}

	// If under the cap, return as-is
	if len(kept) <= maxSymbolsPerPackage {
		return kept
	}

	// Sort by priority (types first, then functions, then vars)
	sort.SliceStable(kept, func(i, j int) bool {
		pi, pj := symbolKindPriority(kept[i].Kind), symbolKindPriority(kept[j].Kind)
		return pi > pj
	})

	return kept[:maxSymbolsPerPackage]
}

func kindLabel(k SymbolKind) string {
	labels := map[SymbolKind]string{
		SymFunc:      "Functions",
		SymMethod:    "Methods",
		SymType:      "Types",
		SymInterface: "Interfaces",
		SymStruct:    "Structs",
		SymClass:     "Classes",
		SymConst:     "Constants",
		SymVar:       "Variables",
		SymEnum:      "Enums",
		SymTrait:     "Traits",
	}
	if l, ok := labels[k]; ok {
		return l
	}
	return string(k)
}

func slugify(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ToLower(s)
	if s == "" || s == "-" {
		s = "root"
	}
	return s
}

// DetectHotFiles ranks files by git edit frequency and import count.
func DetectHotFiles(result *Result, projectRoot string) []HotFile {
	// Count how many times each file is imported by others
	importCount := make(map[string]int)
	for _, fi := range result.Files {
		importCount[fi.Path]++
	}

	// Build a map of file paths to their import counts
	fileImports := make(map[string]int)
	for _, pkg := range result.Packages {
		for _, f := range pkg.Files {
			// Count how many other packages import this package
			fileImports[f] = len(pkg.ImportedBy)
		}
	}

	var hotFiles []HotFile
	seen := make(map[string]bool)
	for _, fi := range result.Files {
		if seen[fi.Path] {
			continue
		}
		seen[fi.Path] = true

		impCount := fileImports[fi.Path]
		symCount := 0
		for _, s := range fi.Symbols {
			if s.Exported {
				symCount++
			}
		}

		// Score: weighted combination
		score := float64(impCount)*3.0 + float64(symCount)*1.0 + float64(fi.LineCount)*0.01
		if score < 1.0 {
			continue
		}

		reason := fmt.Sprintf("imported by %d pkgs, %d exported symbols", impCount, symCount)
		hotFiles = append(hotFiles, HotFile{
			Path:        fi.Path,
			Score:       score,
			ImportCount: impCount,
			Reason:      reason,
		})
	}

	sort.Slice(hotFiles, func(i, j int) bool {
		return hotFiles[i].Score > hotFiles[j].Score
	})

	// Keep top 30
	if len(hotFiles) > 30 {
		hotFiles = hotFiles[:30]
	}

	return hotFiles
}
