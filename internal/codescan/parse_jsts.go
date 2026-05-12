package codescan

import (
	"regexp"
	"strings"
)

// JSTSParser extracts symbols from JavaScript and TypeScript files using regex heuristics.
type JSTSParser struct{}

func NewJSTSParser() *JSTSParser { return &JSTSParser{} }

func (p *JSTSParser) Languages() []string { return []string{"javascript", "typescript"} }

var (
	// Functions: export function, export default function, export async function
	jsFunc = regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s+(\w+)`)

	// Arrow function exports: export const X = (...) => or export const X = async (...) =>
	jsArrowExport = regexp.MustCompile(`^export\s+(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(`)

	// Type-annotated arrow exports: export const X: Type = (...) => (common React pattern)
	jsTypedArrowExport = regexp.MustCompile(`^export\s+(?:const|let|var)\s+(\w+)\s*:.*=\s*(?:async\s+)?[\(\{]`)

	// Classes: export class, export default class, export abstract class
	jsClass = regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:abstract\s+)?class\s+(\w+)`)

	// Interfaces (TS)
	jsInterface = regexp.MustCompile(`^(?:export\s+)?interface\s+(\w+)`)

	// Type aliases (TS)
	jsTypeAlias = regexp.MustCompile(`^(?:export\s+)?type\s+(\w+)\s*[=<]`)

	// Enums (TS)
	jsEnum = regexp.MustCompile(`^(?:export\s+)?enum\s+(\w+)`)

	// Exported const/let/var
	jsConstExport = regexp.MustCompile(`^export\s+(?:const|let|var)\s+(\w+)`)

	// Destructured exports: export const { A, B } = ...
	jsDestructuredExport = regexp.MustCompile(`^export\s+(?:const|let|var)\s+\{\s*([^}]+)\}`)

	// Default export at end of file: export default X
	jsDefaultExp = regexp.MustCompile(`^export\s+default\s+(\w+)\s*;?\s*$`)

	// Named export group: export { foo, bar, baz }
	jsNamedExportGroup = regexp.MustCompile(`^export\s+\{\s*([^}]+)\}`)

	// Re-exports: export { ... } from '...', export * from '...'
	jsReExportFrom = regexp.MustCompile(`^export\s+(?:\{[^}]*\}|\*(?:\s+as\s+\w+)?)\s+from\s+['"]([^'"]+)['"]`)

	// CommonJS: module.exports = ...
	jsModuleExports = regexp.MustCompile(`^module\.exports\s*=`)

	// CommonJS: exports.name = ...
	jsExportsProperty = regexp.MustCompile(`^exports\.(\w+)\s*=`)

	// Imports
	jsImportFrom = regexp.MustCompile(`^import\s+.*?from\s+['"]([^'"]+)['"]`)
	jsImportSide = regexp.MustCompile(`^import\s+['"]([^'"]+)['"]`)
	jsRequire    = regexp.MustCompile(`require\(['"]([^'"]+)['"]\)`)
)

func (p *JSTSParser) ParseFile(path string, content []byte) (*FileInfo, error) {
	lines := strings.Split(string(content), "\n")
	lang := "javascript"
	if strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx") {
		lang = "typescript"
	}

	fi := &FileInfo{
		Path:      path,
		Language:  lang,
		LineCount: len(lines),
	}

	// Track all declared names for resolving export default references
	declaredNames := make(map[string]int) // name -> line number
	var defaultExportRef string
	var defaultExportLine int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := i + 1

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		// Imports
		if m := jsImportFrom.FindStringSubmatch(trimmed); m != nil {
			fi.Imports = append(fi.Imports, m[1])
			continue
		}
		if m := jsImportSide.FindStringSubmatch(trimmed); m != nil {
			fi.Imports = append(fi.Imports, m[1])
			continue
		}
		if m := jsRequire.FindStringSubmatch(trimmed); m != nil {
			fi.Imports = append(fi.Imports, m[1])
		}

		// Re-exports: capture as imports, not symbols
		if m := jsReExportFrom.FindStringSubmatch(trimmed); m != nil {
			fi.Imports = append(fi.Imports, m[1])
			// Also extract re-exported names
			if strings.Contains(trimmed, "{") {
				names := extractBracedNames(trimmed)
				for _, name := range names {
					fi.Symbols = append(fi.Symbols, Symbol{
						Name: name, Kind: SymVar, Exported: true, Line: lineNum,
					})
				}
			}
			continue
		}

		exported := strings.HasPrefix(trimmed, "export ")

		// Functions (including export default function)
		if m := jsFunc.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymFunc, Exported: exported, Line: lineNum,
				Signature: jstsSigFromLine(trimmed),
			})
			declaredNames[m[1]] = lineNum
			continue
		}

		// Destructured exports: export const { A, B } = ...
		if m := jsDestructuredExport.FindStringSubmatch(trimmed); m != nil {
			names := splitNames(m[1])
			for _, name := range names {
				fi.Symbols = append(fi.Symbols, Symbol{
					Name: name, Kind: SymConst, Exported: true, Line: lineNum,
				})
			}
			continue
		}

		// Type-annotated arrow exports (must check before jsConstExport)
		if m := jsTypedArrowExport.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymFunc, Exported: true, Line: lineNum,
				Signature: jstsSigFromLine(trimmed),
			})
			declaredNames[m[1]] = lineNum
			continue
		}

		// Arrow function exports
		if m := jsArrowExport.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymFunc, Exported: true, Line: lineNum,
				Signature: jstsSigFromLine(trimmed),
			})
			declaredNames[m[1]] = lineNum
			continue
		}

		// Classes
		if m := jsClass.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymClass, Exported: exported, Line: lineNum,
				Signature: jstsSigFromLine(trimmed),
			})
			declaredNames[m[1]] = lineNum
			continue
		}

		// Interfaces (TS)
		if m := jsInterface.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymInterface, Exported: exported, Line: lineNum,
				Signature: jstsSigFromLine(trimmed),
			})
			declaredNames[m[1]] = lineNum
			continue
		}

		// Type aliases (TS)
		if m := jsTypeAlias.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymType, Exported: exported, Line: lineNum,
				Signature: jstsSigFromLine(trimmed),
			})
			declaredNames[m[1]] = lineNum
			continue
		}

		// Enums (TS)
		if m := jsEnum.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymEnum, Exported: exported, Line: lineNum,
				Signature: jstsSigFromLine(trimmed),
			})
			declaredNames[m[1]] = lineNum
			continue
		}

		// Named export groups: export { foo, bar, baz }
		if m := jsNamedExportGroup.FindStringSubmatch(trimmed); m != nil {
			names := splitNames(m[1])
			for _, name := range names {
				// Mark previously declared names as exported
				fi.Symbols = append(fi.Symbols, Symbol{
					Name: name, Kind: SymVar, Exported: true, Line: lineNum,
				})
			}
			continue
		}

		// Default export reference: export default X (where X was declared earlier)
		if m := jsDefaultExp.FindStringSubmatch(trimmed); m != nil {
			name := m[1]
			// Skip keywords (these are inline expressions, not references)
			if name != "function" && name != "class" && name != "null" && name != "undefined" && name != "true" && name != "false" {
				defaultExportRef = name
				defaultExportLine = lineNum
			}
			continue
		}

		// CommonJS: module.exports = ...
		if jsModuleExports.MatchString(trimmed) {
			// Mark all previously declared top-level symbols as exported
			for j := range fi.Symbols {
				fi.Symbols[j].Exported = true
			}
			continue
		}

		// CommonJS: exports.name = ...
		if m := jsExportsProperty.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymFunc, Exported: true, Line: lineNum,
			})
			continue
		}

		// Exported const/let/var (catch-all for simple exports)
		if m := jsConstExport.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymConst, Exported: true, Line: lineNum,
			})
			declaredNames[m[1]] = lineNum
			continue
		}

		// Track non-exported declarations for resolving default exports
		if !exported {
			// const/let/var declarations
			if strings.HasPrefix(trimmed, "const ") || strings.HasPrefix(trimmed, "let ") || strings.HasPrefix(trimmed, "var ") {
				parts := strings.Fields(trimmed)
				if len(parts) >= 2 {
					name := strings.TrimRight(parts[1], ":=,;")
					if isValidIdentifier(name) {
						declaredNames[name] = lineNum
					}
				}
			}
		}
	}

	// Resolve default export reference — mark the referenced symbol as exported
	if defaultExportRef != "" {
		found := false
		for j := range fi.Symbols {
			if fi.Symbols[j].Name == defaultExportRef {
				fi.Symbols[j].Exported = true
				found = true
				break
			}
		}
		if !found {
			// Symbol wasn't captured by other patterns; add it
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: defaultExportRef, Kind: SymVar, Exported: true, Line: defaultExportLine,
			})
		}
	}

	return fi, nil
}

// splitNames splits "foo, bar as baz, qux" into ["foo", "baz", "qux"]
// handling "as" aliases (keeps the alias name, which is what's exported).
func splitNames(s string) []string {
	var names []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Handle "X as Y" — use Y (the exported name)
		if idx := strings.Index(part, " as "); idx >= 0 {
			name := strings.TrimSpace(part[idx+4:])
			if isValidIdentifier(name) {
				names = append(names, name)
			}
		} else {
			name := strings.TrimSpace(part)
			if isValidIdentifier(name) {
				names = append(names, name)
			}
		}
	}
	return names
}

// extractBracedNames extracts names from "export { A, B as C } from '...'"
func extractBracedNames(line string) []string {
	start := strings.Index(line, "{")
	end := strings.Index(line, "}")
	if start < 0 || end < 0 || end <= start {
		return nil
	}
	return splitNames(line[start+1 : end])
}

func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if i == 0 {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '$') {
				return false
			}
		} else {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '$') {
				return false
			}
		}
	}
	return true
}

// jstsSigFromLine extracts a clean signature from a JS/TS source line.
// It truncates the body (everything after '{') and caps length.
func jstsSigFromLine(line string) string {
	// Remove trailing body opener
	sig := line
	if idx := strings.Index(sig, "{"); idx > 0 {
		sig = strings.TrimSpace(sig[:idx])
	}
	// Remove trailing '=' for arrow functions (keep params)
	if strings.HasSuffix(sig, "=>") {
		sig = strings.TrimSpace(sig[:len(sig)-2])
	}
	// Cap at reasonable length
	if len(sig) > 120 {
		sig = sig[:117] + "..."
	}
	return sig
}
