package codescan

import (
	"regexp"
	"strings"
)

// RubyParser extracts symbols from Ruby files using regex heuristics.
type RubyParser struct{}

func NewRubyParser() *RubyParser { return &RubyParser{} }

func (p *RubyParser) Languages() []string { return []string{"ruby"} }

var (
	rbClass   = regexp.MustCompile(`^(\s*)class\s+(\w+)`)
	rbModule  = regexp.MustCompile(`^(\s*)module\s+(\w+)`)
	rbDef     = regexp.MustCompile(`^(\s*)def\s+(self\.)?(\w+[?!=]?)`)
	rbRequire = regexp.MustCompile(`^require(?:_relative)?\s+['"]([^'"]+)['"]`)
	rbConst   = regexp.MustCompile(`^\s*([A-Z][A-Z_0-9]+)\s*=`)
)

func (p *RubyParser) ParseFile(path string, content []byte) (*FileInfo, error) {
	lines := strings.Split(string(content), "\n")

	fi := &FileInfo{
		Path:      path,
		Language:  "ruby",
		LineCount: len(lines),
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := i + 1

		if m := rbRequire.FindStringSubmatch(trimmed); m != nil {
			fi.Imports = append(fi.Imports, m[1])
			continue
		}
		if m := rbClass.FindStringSubmatch(line); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[2], Kind: SymClass, Exported: true, Line: lineNum,
			})
			continue
		}
		if m := rbModule.FindStringSubmatch(line); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[2], Kind: SymType, Exported: true, Line: lineNum,
			})
			continue
		}
		if m := rbDef.FindStringSubmatch(line); m != nil {
			indent := m[1]
			isSelf := m[2] != ""
			name := m[3]
			exported := isSelf || len(indent) <= 2 // top-level or class methods
			kind := SymFunc
			if len(indent) > 0 && !isSelf {
				kind = SymMethod
			}
			// Private methods start with underscore by convention (not always)
			if strings.HasPrefix(name, "_") {
				exported = false
			}
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: name, Kind: kind, Exported: exported, Line: lineNum,
			})
			continue
		}
		if m := rbConst.FindStringSubmatch(line); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymConst, Exported: true, Line: lineNum,
			})
			continue
		}
	}

	return fi, nil
}
