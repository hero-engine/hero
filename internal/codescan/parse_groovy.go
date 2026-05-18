package codescan

import (
	"regexp"
	"strings"
)

// GroovyParser extracts symbols from Groovy/Grails files using regex heuristics.
type GroovyParser struct{}

func NewGroovyParser() *GroovyParser { return &GroovyParser{} }

func (p *GroovyParser) Languages() []string { return []string{"groovy"} }

var (
	groovyPackage   = regexp.MustCompile(`^package\s+([\w.]+)`)
	groovyImport    = regexp.MustCompile(`^import\s+(?:static\s+)?([\w.]+)\s*;?\s*$`)
	groovyClass     = regexp.MustCompile(`(?:public\s+)?(?:abstract\s+)?(?:final\s+)?class\s+(\w+)(?:\s+extends\s+(\w+))?`)
	groovyInterface = regexp.MustCompile(`(?:public\s+)?interface\s+(\w+)`)
	groovyEnum      = regexp.MustCompile(`(?:public\s+)?enum\s+(\w+)`)
	groovyTrait     = regexp.MustCompile(`trait\s+(\w+)`)

	// Methods: def methodName(...), Type methodName(...), public/static/private variants
	groovyMethod = regexp.MustCompile(`^\s+(?:(?:public|protected|private|static|final|synchronized|abstract|def)\s+)*(?:def\s+|(?:\w+(?:<[^>]+>)?\s+))(\w+)\s*\(`)

	// Top-level function (unindented def or typed)
	groovyFunc = regexp.MustCompile(`^(?:(?:public|protected|private|static|final|def)\s+)*(?:def\s+|(?:\w+(?:<[^>]+>)?\s+))(\w+)\s*\(`)

	// Grails constraints/mapping blocks (static)
	groovyStaticBlock = regexp.MustCompile(`^\s+static\s+(\w+)\s*=\s*\{`)
)

func (p *GroovyParser) ParseFile(path string, content []byte) (*FileInfo, error) {
	lines := strings.Split(string(content), "\n")

	fi := &FileInfo{
		Path:      path,
		Language:  "groovy",
		LineCount: len(lines),
	}

	inClass := false
	classIndent := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := i + 1

		// Skip empty/comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		if m := groovyPackage.FindStringSubmatch(trimmed); m != nil {
			fi.Package = m[1]
			continue
		}
		if m := groovyImport.FindStringSubmatch(trimmed); m != nil {
			fi.Imports = append(fi.Imports, m[1])
			continue
		}

		// Detect visibility
		isPublic := !strings.Contains(trimmed, "private ") && !strings.Contains(trimmed, "protected ")

		// Classes
		if m := groovyClass.FindStringSubmatch(trimmed); m != nil {
			sig := trimmed
			if idx := strings.Index(sig, "{"); idx > 0 {
				sig = strings.TrimSpace(sig[:idx])
			}
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymClass, Exported: isPublic, Line: lineNum,
				Signature: sig,
			})
			inClass = true
			classIndent = leadingSpaces(line)
			continue
		}
		if m := groovyInterface.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymInterface, Exported: isPublic, Line: lineNum,
				Signature: groovySigFromLine(trimmed),
			})
			inClass = true
			classIndent = leadingSpaces(line)
			continue
		}
		if m := groovyTrait.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymTrait, Exported: isPublic, Line: lineNum,
				Signature: groovySigFromLine(trimmed),
			})
			inClass = true
			classIndent = leadingSpaces(line)
			continue
		}
		if m := groovyEnum.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymEnum, Exported: isPublic, Line: lineNum,
				Signature: groovySigFromLine(trimmed),
			})
			inClass = true
			classIndent = leadingSpaces(line)
			continue
		}

		// Track class end (simple heuristic: closing brace at class indent level)
		if inClass && trimmed == "}" && leadingSpaces(line) <= classIndent {
			inClass = false
			continue
		}

		// Inside a class: methods
		if inClass {
			if m := groovyMethod.FindStringSubmatch(line); m != nil {
				fi.Symbols = append(fi.Symbols, Symbol{
					Name: m[1], Kind: SymMethod, Exported: isPublic, Line: lineNum,
					Signature: groovySigFromLine(trimmed),
				})
				continue
			}
			// Static blocks (constraints, mapping, etc.) — Grails convention
			if m := groovyStaticBlock.FindStringSubmatch(line); m != nil {
				name := m[1]
				if name == "constraints" || name == "mapping" || name == "hasMany" || name == "belongsTo" || name == "transients" {
					fi.Symbols = append(fi.Symbols, Symbol{
						Name: "static " + name, Kind: SymVar, Exported: true, Line: lineNum,
					})
				}
				continue
			}
			continue
		}

		// Top-level functions (not in class)
		if m := groovyFunc.FindStringSubmatch(line); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymFunc, Exported: isPublic, Line: lineNum,
				Signature: groovySigFromLine(trimmed),
			})
			continue
		}
	}

	return fi, nil
}

func groovySigFromLine(line string) string {
	sig := line
	if idx := strings.Index(sig, "{"); idx > 0 {
		sig = strings.TrimSpace(sig[:idx])
	}
	if len(sig) > 120 {
		sig = sig[:117] + "..."
	}
	return sig
}

func leadingSpaces(line string) int {
	count := 0
	for _, c := range line {
		if c == ' ' {
			count++
		} else if c == '\t' {
			count += 4
		} else {
			break
		}
	}
	return count
}
