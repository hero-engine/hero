package codescan

import (
	"regexp"
	"strings"
)

// RustParser extracts symbols from Rust files using regex heuristics.
type RustParser struct{}

func NewRustParser() *RustParser { return &RustParser{} }

func (p *RustParser) Languages() []string { return []string{"rust"} }

var (
	rustFn     = regexp.MustCompile(`^(?:pub(?:\([\w:]+\))?\s+)?(?:async\s+)?fn\s+(\w+)`)
	rustStruct = regexp.MustCompile(`^(?:pub(?:\([\w:]+\))?\s+)?struct\s+(\w+)`)
	rustEnum   = regexp.MustCompile(`^(?:pub(?:\([\w:]+\))?\s+)?enum\s+(\w+)`)
	rustTrait  = regexp.MustCompile(`^(?:pub(?:\([\w:]+\))?\s+)?trait\s+(\w+)`)
	rustType   = regexp.MustCompile(`^(?:pub(?:\([\w:]+\))?\s+)?type\s+(\w+)`)
	rustConst  = regexp.MustCompile(`^(?:pub(?:\([\w:]+\))?\s+)?const\s+(\w+)`)
	rustUse    = regexp.MustCompile(`^use\s+([\w:]+)`)
	rustMod    = regexp.MustCompile(`^(?:pub\s+)?mod\s+(\w+)\s*;`)
)

func (p *RustParser) ParseFile(path string, content []byte) (*FileInfo, error) {
	lines := strings.Split(string(content), "\n")

	fi := &FileInfo{
		Path:      path,
		Language:  "rust",
		LineCount: len(lines),
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := i + 1
		isPub := strings.HasPrefix(trimmed, "pub ")

		if m := rustUse.FindStringSubmatch(trimmed); m != nil {
			fi.Imports = append(fi.Imports, m[1])
			continue
		}
		if m := rustMod.FindStringSubmatch(trimmed); m != nil {
			fi.Imports = append(fi.Imports, m[1])
			continue
		}
		if m := rustFn.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymFunc, Exported: isPub, Line: lineNum,
			})
			continue
		}
		if m := rustStruct.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymStruct, Exported: isPub, Line: lineNum,
			})
			continue
		}
		if m := rustEnum.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymEnum, Exported: isPub, Line: lineNum,
			})
			continue
		}
		if m := rustTrait.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymTrait, Exported: isPub, Line: lineNum,
			})
			continue
		}
		if m := rustType.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymType, Exported: isPub, Line: lineNum,
			})
			continue
		}
		if m := rustConst.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymConst, Exported: isPub, Line: lineNum,
			})
			continue
		}
	}

	return fi, nil
}
