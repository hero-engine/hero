package codescan

import (
	"regexp"
	"strings"
)

// CParser extracts symbols from C and C++ files using regex heuristics.
type CParser struct{}

func NewCParser() *CParser { return &CParser{} }

func (p *CParser) Languages() []string { return []string{"c", "cpp"} }

var (
	cInclude  = regexp.MustCompile(`^#include\s+[<"]([^>"]+)[>"]`)
	cFunc     = regexp.MustCompile(`^(?:static\s+|extern\s+|inline\s+)*(?:[\w*]+\s+)+(\w+)\s*\([^;]*\)\s*\{?\s*$`)
	cStruct   = regexp.MustCompile(`^(?:typedef\s+)?struct\s+(\w+)`)
	cEnum     = regexp.MustCompile(`^(?:typedef\s+)?enum\s+(\w+)`)
	cClass    = regexp.MustCompile(`^class\s+(\w+)`)
	cTypedef  = regexp.MustCompile(`^typedef\s+.*\s+(\w+)\s*;`)
)

func (p *CParser) ParseFile(path string, content []byte) (*FileInfo, error) {
	lines := strings.Split(string(content), "\n")
	lang := "c"
	if strings.HasSuffix(path, ".cpp") || strings.HasSuffix(path, ".cc") ||
		strings.HasSuffix(path, ".cxx") || strings.HasSuffix(path, ".hpp") || strings.HasSuffix(path, ".hxx") {
		lang = "cpp"
	}

	fi := &FileInfo{
		Path:      path,
		Language:  lang,
		LineCount: len(lines),
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := i + 1

		if m := cInclude.FindStringSubmatch(trimmed); m != nil {
			fi.Imports = append(fi.Imports, m[1])
			continue
		}
		// Static means not exported in C
		isStatic := strings.HasPrefix(trimmed, "static ")

		if m := cClass.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymClass, Exported: true, Line: lineNum,
			})
			continue
		}
		if m := cStruct.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymStruct, Exported: !isStatic, Line: lineNum,
			})
			continue
		}
		if m := cEnum.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymEnum, Exported: !isStatic, Line: lineNum,
			})
			continue
		}
		if m := cFunc.FindStringSubmatch(trimmed); m != nil {
			name := m[1]
			// Skip common false positives
			if name == "if" || name == "for" || name == "while" || name == "switch" || name == "return" {
				continue
			}
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: name, Kind: SymFunc, Exported: !isStatic, Line: lineNum,
			})
			continue
		}
		if m := cTypedef.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymType, Exported: true, Line: lineNum,
			})
			continue
		}
	}

	return fi, nil
}
