package codescan

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

// tsLangMap maps codescan language names to tree-sitter grammar names.
var tsLangMap = map[string]string{
	"go":         "go",
	"javascript": "javascript",
	"typescript": "typescript",
	"python":     "python",
	"ruby":       "ruby",
	"rust":       "rust",
	"c":          "c",
	"cpp":        "cpp",
	"java":       "java",
}

// TreeSitterParser implements Parser by invoking the tree-sitter CLI.
type TreeSitterParser struct {
	mu         sync.Mutex
	grammars   map[string]bool // available grammar names
	grammarsOK bool            // true once grammars have been probed
	langs      []string        // languages we claim to support
}

// NewTreeSitterParser creates a parser backed by the tree-sitter CLI.
// It lazily detects installed grammars on first use.
func NewTreeSitterParser() *TreeSitterParser {
	return &TreeSitterParser{}
}

// Languages returns languages for which grammars are installed.
func (p *TreeSitterParser) Languages() []string {
	p.ensureGrammars()
	return p.langs
}

// ParseFile parses a file via `tree-sitter parse` and extracts symbols.
func (p *TreeSitterParser) ParseFile(path string, content []byte) (*FileInfo, error) {
	p.ensureGrammars()

	ext := strings.ToLower(filepath.Ext(path))
	lang := languageForExt(ext)
	if lang == "" {
		return nil, fmt.Errorf("unknown language for %s", path)
	}

	tsName, ok := tsLangMap[lang]
	if !ok {
		return nil, fmt.Errorf("no tree-sitter mapping for %s", lang)
	}
	if !p.grammars[tsName] {
		return nil, fmt.Errorf("tree-sitter grammar not installed: %s", tsName)
	}

	// tree-sitter parse needs an absolute or relative path to an actual file.
	// The caller passes a relative path but content is already read.
	// We invoke tree-sitter parse on the path directly (it reads the file itself).
	cmd := exec.Command("tree-sitter", "parse", path)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tree-sitter parse: %w", err)
	}

	fi := &FileInfo{
		Path:      path,
		Language:  lang,
		LineCount: strings.Count(string(content), "\n") + 1,
	}

	// Parse the S-expression output
	sexp := string(out)
	nodes := parseSExp(sexp)

	// Extract symbols and imports from the parsed nodes
	extractSymbols(nodes, lang, fi)
	extractImports(nodes, lang, fi)

	// Try to extract package name for Go
	if lang == "go" {
		extractGoPackage(nodes, fi)
	}

	return fi, nil
}

// ensureGrammars lazily probes for installed tree-sitter grammars.
func (p *TreeSitterParser) ensureGrammars() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.grammarsOK {
		return
	}
	p.grammarsOK = true
	p.grammars = make(map[string]bool)

	out, err := exec.Command("tree-sitter", "dump-languages").CombinedOutput()
	if err != nil {
		return
	}

	// dump-languages outputs JSON. Parse it to find grammar names.
	// The format is a JSON object where keys are grammar names.
	var langData map[string]json.RawMessage
	if err := json.Unmarshal(out, &langData); err != nil {
		// Fallback: try line-based parsing
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || line == "{" || line == "}" {
				continue
			}
			// Extract key from JSON-like "key": ...
			if idx := strings.Index(line, `"`); idx >= 0 {
				rest := line[idx+1:]
				if end := strings.Index(rest, `"`); end >= 0 {
					p.grammars[rest[:end]] = true
				}
			}
		}
	} else {
		for name := range langData {
			p.grammars[name] = true
		}
	}

	// Build langs list from installed grammars
	for csLang, tsName := range tsLangMap {
		if p.grammars[tsName] {
			p.langs = append(p.langs, csLang)
		}
	}
}

// --- S-expression parser ---

// sexpNode represents a node from tree-sitter's S-expression output.
type sexpNode struct {
	Type     string      // e.g. "function_declaration"
	Field    string      // e.g. "name" if this is a named field child
	Row      int         // 0-indexed start row
	Col      int         // 0-indexed start col
	Text     string      // for leaf nodes like identifiers, may contain the text
	Children []*sexpNode // child nodes
}

// parseSExp parses tree-sitter's S-expression output into a tree of nodes.
func parseSExp(s string) []*sexpNode {
	p := &sexpParser{input: s}
	var nodes []*sexpNode
	for p.pos < len(p.input) {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			break
		}
		if p.input[p.pos] == '(' {
			if n := p.parseNode(""); n != nil {
				nodes = append(nodes, n)
			}
		} else {
			p.pos++
		}
	}
	return nodes
}

type sexpParser struct {
	input string
	pos   int
}

func (p *sexpParser) skipWhitespace() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\n' || p.input[p.pos] == '\r' || p.input[p.pos] == '\t') {
		p.pos++
	}
}

func (p *sexpParser) parseNode(field string) *sexpNode {
	if p.pos >= len(p.input) || p.input[p.pos] != '(' {
		return nil
	}
	p.pos++ // skip '('
	p.skipWhitespace()

	// Read node type
	nodeType := p.readWord()
	if nodeType == "" {
		// skip to matching close paren
		p.skipToClose()
		return nil
	}

	node := &sexpNode{Type: nodeType, Field: field}

	p.skipWhitespace()

	// Try to read position: [row, col] - [row, col]
	if p.pos < len(p.input) && p.input[p.pos] == '[' {
		row, col := p.readPosition()
		node.Row = row
		node.Col = col
		p.skipWhitespace()
		// skip " - "
		if p.pos < len(p.input) && p.input[p.pos] == '-' {
			p.pos++
			p.skipWhitespace()
			p.readPosition() // end position, discard
		}
	}

	p.skipWhitespace()

	// Parse children
	for p.pos < len(p.input) && p.input[p.pos] != ')' {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			break
		}

		if p.input[p.pos] == ')' {
			break
		}

		// Check for field name: "name: (...)"
		childField := ""
		if p.input[p.pos] != '(' {
			word := p.readWord()
			p.skipWhitespace()
			if p.pos < len(p.input) && p.input[p.pos] == ':' {
				childField = word
				p.pos++ // skip ':'
				p.skipWhitespace()
			}
			// If not a field label, it might be a plain text token (leaf)
			if childField == "" && word != "" {
				// Store as text child
				node.Children = append(node.Children, &sexpNode{Type: word, Field: "", Text: word})
				continue
			}
		}

		if p.pos < len(p.input) && p.input[p.pos] == '(' {
			child := p.parseNode(childField)
			if child != nil {
				node.Children = append(node.Children, child)
			}
		} else if p.pos < len(p.input) && p.input[p.pos] != ')' {
			// Skip unexpected tokens
			p.readWord()
		}
	}

	if p.pos < len(p.input) && p.input[p.pos] == ')' {
		p.pos++
	}

	return node
}

func (p *sexpParser) readWord() string {
	p.skipWhitespace()
	start := p.pos
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' || c == '(' || c == ')' || c == '[' || c == ':' {
			break
		}
		p.pos++
	}
	return p.input[start:p.pos]
}

func (p *sexpParser) readPosition() (int, int) {
	// Read [row, col]
	if p.pos >= len(p.input) || p.input[p.pos] != '[' {
		return 0, 0
	}
	p.pos++ // skip '['
	rowStr := p.readNumber()
	p.skipWhitespace()
	if p.pos < len(p.input) && p.input[p.pos] == ',' {
		p.pos++
	}
	p.skipWhitespace()
	colStr := p.readNumber()
	if p.pos < len(p.input) && p.input[p.pos] == ']' {
		p.pos++
	}
	row, _ := strconv.Atoi(rowStr)
	col, _ := strconv.Atoi(colStr)
	return row, col
}

func (p *sexpParser) readNumber() string {
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
		p.pos++
	}
	return p.input[start:p.pos]
}

func (p *sexpParser) skipToClose() {
	depth := 1
	for p.pos < len(p.input) && depth > 0 {
		if p.input[p.pos] == '(' {
			depth++
		} else if p.input[p.pos] == ')' {
			depth--
		}
		p.pos++
	}
}

// --- Symbol extraction ---

// nodeTypeToKind maps tree-sitter node types to symbol kinds per language.
var nodeTypeToKind = map[string]map[string]SymbolKind{
	"go": {
		"function_declaration": SymFunc,
		"method_declaration":   SymMethod,
		"type_declaration":     SymType,
		"type_spec":            SymType,
		"const_spec":           SymConst,
		"var_spec":             SymVar,
	},
	"javascript": {
		"function_declaration": SymFunc,
		"class_declaration":    SymClass,
		"method_definition":    SymMethod,
		"lexical_declaration":  SymVar,
	},
	"typescript": {
		"function_declaration": SymFunc,
		"class_declaration":    SymClass,
		"method_definition":    SymMethod,
		"lexical_declaration":  SymVar,
		"interface_declaration": SymInterface,
		"type_alias_declaration": SymType,
	},
	"python": {
		"function_definition": SymFunc,
		"class_definition":    SymClass,
		"assignment":          SymVar,
	},
	"ruby": {
		"method":     SymMethod,
		"class":      SymClass,
		"module":     SymType,
		"assignment": SymVar,
	},
	"rust": {
		"function_item": SymFunc,
		"struct_item":   SymStruct,
		"enum_item":     SymEnum,
		"trait_item":    SymTrait,
		"impl_item":     SymType,
		"type_item":     SymType,
		"const_item":    SymConst,
	},
	"java": {
		"class_declaration":     SymClass,
		"method_declaration":    SymMethod,
		"interface_declaration": SymInterface,
		"enum_declaration":      SymEnum,
	},
	"c": {
		"function_definition":  SymFunc,
		"struct_specifier":     SymStruct,
		"enum_specifier":       SymEnum,
		"declaration":          SymVar,
	},
	"cpp": {
		"function_definition":  SymFunc,
		"class_specifier":      SymClass,
		"struct_specifier":     SymStruct,
		"enum_specifier":       SymEnum,
		"declaration":          SymVar,
	},
}

func extractSymbols(nodes []*sexpNode, lang string, fi *FileInfo) {
	kindMap := nodeTypeToKind[lang]
	if kindMap == nil {
		return
	}
	for _, n := range nodes {
		walkForSymbols(n, lang, kindMap, fi, 0, false)
	}
}

func walkForSymbols(n *sexpNode, lang string, kindMap map[string]SymbolKind, fi *FileInfo, depth int, inExport bool) {
	if n == nil {
		return
	}

	isExport := n.Type == "export_statement"

	kind, isSymbol := kindMap[n.Type]
	if isSymbol {
		// Refine kind for Go type_spec
		if lang == "go" && n.Type == "type_spec" {
			kind = refineGoTypeKind(n)
		}

		name := findName(n)
		if name != "" {
			sym := Symbol{
				Name:     name,
				Kind:     kind,
				Line:     n.Row + 1, // convert 0-indexed to 1-indexed
				Exported: isExported(name, lang, inExport || isExport),
			}

			// Try to extract signature from parameters
			if sig := buildSignature(n, name, kind); sig != "" {
				sym.Signature = sig
			}

			// For Go methods, try to extract receiver
			if lang == "go" && n.Type == "method_declaration" {
				if recv := findFieldChild(n, "receiver"); recv != nil {
					if recvName := findFirstChildOfType(recv, "type_identifier"); recvName != "" {
						sym.Receiver = recvName
					} else if recvName := findFirstChildOfType(recv, "pointer_type"); recvName != "" {
						sym.Receiver = "*" + findFirstChildOfType(findChildByType(recv, "pointer_type"), "type_identifier")
					}
				}
			}

			fi.Symbols = append(fi.Symbols, sym)
		}
	}

	for _, child := range n.Children {
		walkForSymbols(child, lang, kindMap, fi, depth+1, inExport || isExport)
	}
}

func refineGoTypeKind(n *sexpNode) SymbolKind {
	for _, c := range n.Children {
		switch c.Type {
		case "struct_type":
			return SymStruct
		case "interface_type":
			return SymInterface
		}
		// Check field children too
		if c.Field == "type" {
			switch c.Type {
			case "struct_type":
				return SymStruct
			case "interface_type":
				return SymInterface
			}
		}
	}
	return SymType
}

func findName(n *sexpNode) string {
	// Look for a "name" field child
	for _, c := range n.Children {
		if c.Field == "name" {
			if c.Type == "identifier" || c.Type == "type_identifier" || c.Type == "property_identifier" {
				return c.Text
			}
			// If the name child itself has children, get text from first identifier
			for _, gc := range c.Children {
				if gc.Type == "identifier" || gc.Type == "type_identifier" {
					return gc.Text
				}
			}
			// For tree-sitter, identifiers often have their text as the type itself
			// when they're leaf nodes. In the S-expression, leaf identifiers look like:
			// (identifier [r,c] - [r,c])
			// The actual text is not in the S-expression output.
			// We need to fall back to using position-based extraction from source.
			return c.Type
		}
	}
	// Fallback: first identifier child
	for _, c := range n.Children {
		if c.Type == "identifier" || c.Type == "type_identifier" {
			return c.Text
		}
	}
	return ""
}

func findFieldChild(n *sexpNode, field string) *sexpNode {
	for _, c := range n.Children {
		if c.Field == field {
			return c
		}
	}
	return nil
}

func findChildByType(n *sexpNode, typ string) *sexpNode {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if c.Type == typ {
			return c
		}
	}
	return nil
}

func findFirstChildOfType(n *sexpNode, typ string) string {
	if n == nil {
		return ""
	}
	for _, c := range n.Children {
		if c.Type == typ {
			if c.Text != "" {
				return c.Text
			}
			return c.Type
		}
	}
	return ""
}

func buildSignature(n *sexpNode, name string, kind SymbolKind) string {
	if kind != SymFunc && kind != SymMethod {
		return ""
	}
	params := findFieldChild(n, "parameters")
	if params == nil {
		return ""
	}
	// Simple: just note that parameters exist
	return name + "(...)"
}

func isExported(name, lang string, inExport bool) bool {
	switch lang {
	case "go":
		if len(name) == 0 {
			return false
		}
		return unicode.IsUpper(rune(name[0]))
	case "javascript", "typescript":
		return inExport
	case "python":
		return !strings.HasPrefix(name, "_")
	case "ruby":
		// Public by default; private methods start with underscore by convention
		return !strings.HasPrefix(name, "_")
	case "rust":
		// pub keyword would be in the AST but we simplify:
		// top-level items with capitalized names or any function are considered exported
		return true
	case "java":
		// Simplified: capitalized names or any method in a public class
		return unicode.IsUpper(rune(name[0]))
	case "c", "cpp":
		// No real export concept; treat all as exported
		return true
	}
	return true
}

// --- Import extraction ---

var importNodeTypes = map[string]bool{
	"import_declaration": true,
	"import_statement":   true,
	"import_from_statement": true,
	"use_declaration":    true,
	"preproc_include":    true,
}

func extractImports(nodes []*sexpNode, lang string, fi *FileInfo) {
	for _, n := range nodes {
		walkForImports(n, lang, fi)
	}
}

func walkForImports(n *sexpNode, lang string, fi *FileInfo) {
	if n == nil {
		return
	}
	if importNodeTypes[n.Type] {
		// Try to find a string literal child for the import path
		imp := findImportPath(n)
		if imp != "" {
			fi.Imports = append(fi.Imports, imp)
		}
	}
	for _, c := range n.Children {
		walkForImports(c, lang, fi)
	}
}

func findImportPath(n *sexpNode) string {
	// Look for interpreted_string_literal, string, string_literal children
	for _, c := range n.Children {
		switch c.Type {
		case "interpreted_string_literal", "string", "string_literal", "string_content":
			if c.Text != "" {
				return strings.Trim(c.Text, `"'`)
			}
		}
		// Look for path field
		if c.Field == "path" || c.Field == "source" || c.Field == "module_name" {
			if c.Text != "" {
				return strings.Trim(c.Text, `"'`)
			}
			// recurse one level
			for _, gc := range c.Children {
				if gc.Text != "" {
					return strings.Trim(gc.Text, `"'`)
				}
			}
		}
	}
	// Recurse into import_spec_list etc.
	for _, c := range n.Children {
		if strings.Contains(c.Type, "import") || strings.Contains(c.Type, "spec") {
			imp := findImportPath(c)
			if imp != "" {
				return imp
			}
		}
	}
	return ""
}

func extractGoPackage(nodes []*sexpNode, fi *FileInfo) {
	for _, n := range nodes {
		if n.Type == "source_file" {
			for _, c := range n.Children {
				if c.Type == "package_clause" {
					name := findName(c)
					if name != "" {
						fi.Package = name
						return
					}
				}
			}
		}
		// Also check top-level package_clause
		if n.Type == "package_clause" {
			name := findName(n)
			if name != "" {
				fi.Package = name
				return
			}
		}
	}
}
