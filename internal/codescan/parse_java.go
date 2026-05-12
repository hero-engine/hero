package codescan

import (
	"regexp"
	"strings"
)

// JavaParser extracts symbols from Java files using regex heuristics.
type JavaParser struct{}

func NewJavaParser() *JavaParser { return &JavaParser{} }

func (p *JavaParser) Languages() []string { return []string{"java"} }

var (
	javaPackage   = regexp.MustCompile(`^package\s+([\w.]+)\s*;`)
	javaImport    = regexp.MustCompile(`^import\s+(?:static\s+)?([\w.]+)\s*;`)
	javaClass     = regexp.MustCompile(`(?:public|protected|private)?\s*(?:abstract\s+)?(?:final\s+)?class\s+(\w+)`)
	javaInterface = regexp.MustCompile(`(?:public\s+)?interface\s+(\w+)`)
	javaEnum      = regexp.MustCompile(`(?:public\s+)?enum\s+(\w+)`)
	javaMethod    = regexp.MustCompile(`^\s+(?:public|protected|private)\s+(?:static\s+)?(?:final\s+)?(?:synchronized\s+)?(?:abstract\s+)?(?:\w+(?:<[^>]+>)?)\s+(\w+)\s*\(`)
)

func (p *JavaParser) ParseFile(path string, content []byte) (*FileInfo, error) {
	lines := strings.Split(string(content), "\n")

	fi := &FileInfo{
		Path:      path,
		Language:  "java",
		LineCount: len(lines),
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := i + 1

		if m := javaPackage.FindStringSubmatch(trimmed); m != nil {
			fi.Package = m[1]
			continue
		}
		if m := javaImport.FindStringSubmatch(trimmed); m != nil {
			fi.Imports = append(fi.Imports, m[1])
			continue
		}

		isPublic := strings.Contains(trimmed, "public ")

		if m := javaClass.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymClass, Exported: isPublic, Line: lineNum,
				Signature: javaSigFromLine(trimmed),
			})
			continue
		}
		if m := javaInterface.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymInterface, Exported: isPublic, Line: lineNum,
				Signature: javaSigFromLine(trimmed),
			})
			continue
		}
		if m := javaEnum.FindStringSubmatch(trimmed); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymEnum, Exported: isPublic, Line: lineNum,
				Signature: javaSigFromLine(trimmed),
			})
			continue
		}
		if m := javaMethod.FindStringSubmatch(line); m != nil {
			fi.Symbols = append(fi.Symbols, Symbol{
				Name: m[1], Kind: SymMethod, Exported: isPublic, Line: lineNum,
				Signature: javaSigFromLine(strings.TrimSpace(line)),
			})
			continue
		}
	}

	return fi, nil
}

func javaSigFromLine(line string) string {
	sig := line
	if idx := strings.Index(sig, "{"); idx > 0 {
		sig = strings.TrimSpace(sig[:idx])
	}
	// Remove trailing semicolons
	sig = strings.TrimRight(sig, ";")
	sig = strings.TrimSpace(sig)
	if len(sig) > 120 {
		sig = sig[:117] + "..."
	}
	return sig
}
