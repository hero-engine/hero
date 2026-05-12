package codescan

import (
	"regexp"
	"strings"
	"unicode"
)

// PythonParser extracts symbols from Python files using regex heuristics.
type PythonParser struct{}

func NewPythonParser() *PythonParser { return &PythonParser{} }

func (p *PythonParser) Languages() []string { return []string{"python"} }

var (
	pyFunc    = regexp.MustCompile(`^(\s*)(?:async\s+)?def\s+(\w+)\s*\(`)
	pyClass   = regexp.MustCompile(`^class\s+(\w+)`)
	pyImport  = regexp.MustCompile(`^import\s+(\S+)`)
	pyFromImp = regexp.MustCompile(`^from\s+(\S+)\s+import`)
	pyAssign  = regexp.MustCompile(`^([A-Z][A-Z_0-9]+)\s*=`)
)

func (p *PythonParser) ParseFile(path string, content []byte) (*FileInfo, error) {
	lines := strings.Split(string(content), "\n")

	fi := &FileInfo{
		Path:      path,
		Language:  "python",
		LineCount: len(lines),
	}

	// Detect package from __init__.py
	if strings.HasSuffix(path, "__init__.py") {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			fi.Package = parts[len(parts)-2]
		}
	}

	var prevDoc string
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := i + 1

		// Track docstrings for next symbol
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
			doc := strings.TrimPrefix(trimmed, `"""`)
			doc = strings.TrimPrefix(doc, `'''`)
			doc = strings.TrimSuffix(doc, `"""`)
			doc = strings.TrimSuffix(doc, `'''`)
			if doc != "" {
				prevDoc = strings.TrimSpace(doc)
			}
		}

		// Imports
		if m := pyImport.FindStringSubmatch(trimmed); m != nil {
			fi.Imports = append(fi.Imports, m[1])
			continue
		}
		if m := pyFromImp.FindStringSubmatch(trimmed); m != nil {
			fi.Imports = append(fi.Imports, m[1])
			continue
		}

		// Functions (top-level only — no leading whitespace)
		if m := pyFunc.FindStringSubmatch(line); m != nil {
			indent := m[1]
			name := m[2]
			isMethod := len(indent) > 0
			exported := !strings.HasPrefix(name, "_")

			kind := SymFunc
			if isMethod {
				kind = SymMethod
			}
			sym := Symbol{
				Name: name, Kind: kind, Exported: exported, Line: lineNum,
			}
			if !isMethod {
				sym.Doc = prevDoc
			}
			fi.Symbols = append(fi.Symbols, sym)
			prevDoc = ""
			continue
		}

		// Classes
		if m := pyClass.FindStringSubmatch(trimmed); m != nil {
			exported := unicode.IsUpper(rune(m[1][0]))
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymClass, Exported: exported, Line: lineNum,
				Doc: prevDoc,
			})
			prevDoc = ""
			continue
		}

		// Module-level constants (ALL_CAPS)
		if m := pyAssign.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymConst, Exported: true, Line: lineNum,
			})
			continue
		}

		prevDoc = ""
	}

	return fi, nil
}
