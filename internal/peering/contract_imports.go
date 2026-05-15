package peering

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	contractpeering "github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/config"
)

// Phase 3 of cross-repo-peering: contract-import passive surfacing.
//
// The boundary detector. "This file imports a peer-owned contract
// symbol" is the cleanest signal that a change crosses a repo
// boundary — cleaner than path heuristics or commit-message scans.
//
// v1 scope, locked: scan changed files (cheap, signal-rich) — not
// the whole repo. The scanner produces signals; the surfacing layer
// renders them into a one-liner consumed by `/resume` and
// `hero context`. No blocking, no prompts, no auto-trigger.

// ContractImportHit is one (file, contract-symbol, owning-peer)
// match produced by the scanner. The renderer turns this into a
// single human-readable line.
type ContractImportHit struct {
	// File is the path to the file in the consuming repo (absolute
	// or relative — whatever the input gave us, normalized for
	// display).
	File string

	// SymbolPackage is the import path of the contract (e.g.
	// "github.com/hero-engine/hero/contracts/events").
	SymbolPackage string

	// SymbolName is the Go identifier referenced (e.g. "Envelope").
	SymbolName string

	// FullSymbol is the original "<package>.<Name>" string as
	// declared in the peer's manifest (e.g. "contracts/events.Envelope"
	// — note the manifest form is a tail-relative path, not the full
	// module path).
	FullSymbol string

	// PeerAlias is the local alias of the owning peer (display form).
	PeerAlias string

	// PeerID is the canonical peer_id.
	PeerID string

	// Convention is the governing convention slug from the contract
	// entry, or "" if none.
	Convention string

	// LastChangedCommit is the short SHA of the most recent commit
	// that touched the peer-side file defining this symbol. Empty
	// when not resolvable.
	LastChangedCommit string
}

// ScanOptions configures a single scan.
type ScanOptions struct {
	// ChangedFiles is the list of files (in the consuming repo) to
	// scan. Paths can be absolute or relative to ProjectRoot. Empty
	// → no scan.
	ChangedFiles []string

	// Now is a test seam; unused for v1 but reserved.
}

// ScanContractImports walks the changed files for any Go imports of
// peer-owned contract symbols. Each peer is loaded by reading its
// manifest from the resolved repo path; peers with unreachable
// manifests are silently skipped (the passive-surfacing path must
// never error out the caller — it's a context signal, not a
// guardrail).
//
// Filters applied per file:
//   - skip *_test.go
//   - skip vendor/ tree
//   - skip generated files (header `// Code generated ... DO NOT EDIT.`)
//   - file extension must be .go
//
// Match definition: a file matches a peer-owned contract entry when
// (a) one of its import paths contains the symbol's package path
// (suffix match — the manifest stores "contracts/events.Envelope"
// shape, the consumer's import is a full module path ending in
// "contracts/events"), AND (b) the file references the symbol's
// identifier name. Text scan is sufficient for v1 — AST parsing buys
// nothing for the "this changed file touches a peer surface" use
// case and costs the cold-start latency `/resume` runs in.
func ScanContractImports(projectRoot string, opts ScanOptions) ([]ContractImportHit, error) {
	if len(opts.ChangedFiles) == 0 {
		return nil, nil
	}
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	peers := loadPeerContracts(cfg, projectRoot)
	if len(peers) == 0 {
		return nil, nil
	}

	var hits []ContractImportHit
	for _, raw := range opts.ChangedFiles {
		path := raw
		if !filepath.IsAbs(path) {
			path = filepath.Join(projectRoot, path)
		}
		if !shouldScanFile(path) {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			// Missing or unreadable changed file (e.g. deleted in the
			// working tree) — skip silently. Surfacing is best-effort.
			continue
		}
		if isGeneratedGo(data) {
			continue
		}
		imports := extractImports(data)
		if len(imports) == 0 {
			continue
		}
		for _, peer := range peers {
			for _, shape := range peer.Shapes {
				pkg, name := splitGoSymbol(shape.GoSymbol)
				if pkg == "" || name == "" {
					continue
				}
				if !importMatches(imports, pkg) {
					continue
				}
				if !bodyReferences(data, name) {
					continue
				}
				hits = append(hits, ContractImportHit{
					File:              raw,
					SymbolPackage:     pkg,
					SymbolName:        name,
					FullSymbol:        shape.GoSymbol,
					PeerAlias:         peer.Alias,
					PeerID:            peer.PeerID,
					Convention:        shape.Convention,
					LastChangedCommit: lastChangedCommit(peer.Path, pkg, name),
				})
			}
		}
	}
	return hits, nil
}

// RenderContractImportSignal renders a slice of hits as the
// one-line-per-hit passive signal documented in the spec. When hits
// is empty, returns the empty string — callers can print
// unconditionally.
//
// Format per the spec example:
//   You're touching `contracts/events.Envelope` — owned by peer `app`
//   (peer_id: 9c1c2f3e-...). Convention: error-envelope. Last
//   changed: commit 4427cec.
//
// Deduplicated by (file, symbol) so a file that mentions the symbol
// many times still produces one line.
func RenderContractImportSignal(hits []ContractImportHit) string {
	if len(hits) == 0 {
		return ""
	}
	seen := map[string]bool{}
	var b strings.Builder
	b.WriteString("<!-- hero:peer-contract-imports -->\n")
	b.WriteString("**Hero** — you're touching peer-owned contract symbols:\n\n")
	for _, h := range hits {
		key := h.File + "|" + h.FullSymbol
		if seen[key] {
			continue
		}
		seen[key] = true
		shortID := h.PeerID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		line := fmt.Sprintf("- `%s` in `%s` — owned by peer `%s` (peer_id: %s…).",
			h.FullSymbol, h.File, h.PeerAlias, shortID)
		if h.Convention != "" {
			line += fmt.Sprintf(" Convention: %s.", h.Convention)
		}
		if h.LastChangedCommit != "" {
			line += fmt.Sprintf(" Last changed: commit %s.", h.LastChangedCommit)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n_Passive signal — no action required. Run `hero peer call <alias> --mode=advisory ...` if you need a live answer._\n")
	return b.String()
}

// peerContracts is the per-peer slice of contract entries the scanner
// matches against.
type peerContracts struct {
	Alias  string
	PeerID string
	Path   string
	Shapes []contractpeering.ContractEntry
}

// loadPeerContracts reads each configured peer's manifest and
// returns the slice of (alias, peer_id, shapes). Peers without a
// readable manifest, without a contracts section, or without shapes
// are silently dropped — this code path is a passive signal, not a
// guardrail, and a missing peer manifest must never block `/resume`.
func loadPeerContracts(cfg config.Config, projectRoot string) []peerContracts {
	var out []peerContracts
	for alias := range cfg.Repos {
		peerPath, err := cfg.ResolveRepoPath(projectRoot, alias)
		if err != nil {
			continue
		}
		m, err := ReadPeerManifest(peerPath, cfg.Folder)
		if err != nil {
			continue
		}
		if m.Contracts == nil || len(m.Contracts.Shapes) == 0 {
			continue
		}
		out = append(out, peerContracts{
			Alias:  alias,
			PeerID: m.Repo.PeerID,
			Path:   peerPath,
			Shapes: m.Contracts.Shapes,
		})
	}
	return out
}

// shouldScanFile applies the per-file filters before reading any
// content: extension, vendor tree, test files.
func shouldScanFile(absPath string) bool {
	if !strings.HasSuffix(absPath, ".go") {
		return false
	}
	if strings.HasSuffix(absPath, "_test.go") {
		return false
	}
	// Standard Go convention: vendor/ contents are not "this repo's
	// code". A consumer pulling a peer's contract through vendor
	// isn't crossing the boundary in the editing sense.
	if strings.Contains(absPath, string(filepath.Separator)+"vendor"+string(filepath.Separator)) {
		return false
	}
	return true
}

// isGeneratedGo returns true when the Go file's first non-blank
// comment carries the canonical generated-code marker. Cheap header
// scan (first 2 KiB).
func isGeneratedGo(data []byte) bool {
	head := data
	if len(head) > 2048 {
		head = head[:2048]
	}
	scanner := bufio.NewScanner(bytes.NewReader(head))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Match `// Code generated ... DO NOT EDIT.` per the Go spec
		// recommendation in cmd/go/internal/generate.
		if strings.HasPrefix(line, "// Code generated") && strings.Contains(line, "DO NOT EDIT.") {
			return true
		}
		// Stop at the package clause — generated markers must appear
		// above it.
		if strings.HasPrefix(line, "package ") {
			return false
		}
	}
	return false
}

// extractImports returns the import paths declared in a Go file. We
// only need the paths (the strings between quotes inside the import
// declaration) — not the import aliases. Tolerates both the
// `import "foo"` and `import (\n "foo"\n)` shapes.
//
// Text scan rather than go/parser: parsing buys nothing here and
// imposes a real cold-start cost on `/resume`.
func extractImports(data []byte) []string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var imports []string
	inBlock := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !inBlock {
			if line == "import (" {
				inBlock = true
				continue
			}
			if strings.HasPrefix(line, "import ") {
				if p := extractQuoted(line); p != "" {
					imports = append(imports, p)
				}
				continue
			}
			// Once we see a non-import top-level declaration the
			// import section is over — stop scanning the rest of the
			// file.
			if strings.HasPrefix(line, "func ") ||
				strings.HasPrefix(line, "type ") ||
				strings.HasPrefix(line, "var ") ||
				strings.HasPrefix(line, "const ") {
				break
			}
			continue
		}
		// Inside the `import (` block.
		if line == ")" {
			inBlock = false
			continue
		}
		if p := extractQuoted(line); p != "" {
			imports = append(imports, p)
		}
	}
	return imports
}

// extractQuoted returns the substring inside the first pair of
// double quotes on the line, or "" if none. Handles `_ "foo"`,
// `alias "foo"`, and bare `"foo"`.
func extractQuoted(line string) string {
	start := strings.Index(line, `"`)
	if start < 0 {
		return ""
	}
	end := strings.Index(line[start+1:], `"`)
	if end < 0 {
		return ""
	}
	return line[start+1 : start+1+end]
}

// importMatches reports whether any of the file's imports matches
// the peer's contract package, using a path-suffix match. The
// manifest's `go_symbol` field is a tail-relative path (e.g.
// "contracts/events.Envelope") — a consumer's import line carries
// the full module path ending in that suffix. Matching by suffix
// keeps us independent of the consumer's module choice.
//
// To avoid false positives like "contracts/events" matching
// "myorg/contracts/eventsv2", we require a path-segment boundary:
// the suffix must either be the entire import path or be preceded
// by a "/".
func importMatches(imports []string, pkgSuffix string) bool {
	for _, imp := range imports {
		if imp == pkgSuffix {
			return true
		}
		if strings.HasSuffix(imp, "/"+pkgSuffix) {
			return true
		}
	}
	return false
}

// splitGoSymbol splits a `contracts/events.Envelope` shaped string
// into ("contracts/events", "Envelope"). The split is on the last
// "." — package paths don't contain dots in their leaf segment, so
// this is unambiguous for v1's tail-relative form.
func splitGoSymbol(sym string) (pkg, name string) {
	i := strings.LastIndex(sym, ".")
	if i < 0 {
		return "", ""
	}
	return sym[:i], sym[i+1:]
}

// bodyReferences reports whether the file body mentions the symbol
// identifier as a token. Using word-boundary cues so a symbol named
// "Envelope" doesn't match "EnvelopeBuilder" or "myEnvelope". This
// is a heuristic — false positives in comments are acceptable for a
// passive signal — but false negatives are not, so we err on the
// side of matching.
//
// Token boundaries: the byte before is not a letter/digit/"_", the
// byte after is not a letter/digit/"_". A package-qualified
// reference (e.g. "events.Envelope") matches because "." is not a
// word character.
func bodyReferences(data []byte, name string) bool {
	if name == "" {
		return false
	}
	src := data
	target := []byte(name)
	for {
		idx := bytes.Index(src, target)
		if idx < 0 {
			return false
		}
		// Boundary check.
		preOK := idx == 0 || !isWordByte(src[idx-1])
		afterIdx := idx + len(target)
		postOK := afterIdx >= len(src) || !isWordByte(src[afterIdx])
		if preOK && postOK {
			return true
		}
		src = src[idx+1:]
	}
}

func isWordByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z':
		return true
	case b >= 'A' && b <= 'Z':
		return true
	case b >= '0' && b <= '9':
		return true
	case b == '_':
		return true
	}
	return false
}

// lastChangedCommit returns the short SHA of the most recent commit
// in the peer repo that touched the file defining the given symbol.
// Best-effort: returns "" when git is missing, when the peer isn't
// a git repo, or when the symbol's file can't be located.
//
// Resolution strategy: locate the .go file in peerPath whose path
// ends with the contract package suffix AND that contains the
// symbol's declaration. We walk only files matching the package
// suffix to keep this cheap.
func lastChangedCommit(peerPath, pkgSuffix, symbolName string) string {
	file := locateSymbolFile(peerPath, pkgSuffix, symbolName)
	if file == "" {
		return ""
	}
	relFile, err := filepath.Rel(peerPath, file)
	if err != nil {
		relFile = file
	}
	out, err := exec.Command("git", "-C", peerPath, "log", "-1", "--format=%h", "--", relFile).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// locateSymbolFile walks the peer repo looking for a .go file whose
// path matches the contract package suffix and which declares the
// symbol. Cheap-ish: we filter by path suffix before reading, then
// scan for a declaration token. Returns absolute path or "".
func locateSymbolFile(peerPath, pkgSuffix, symbolName string) string {
	var found string
	// Use the last segment of the suffix as a directory hint so we
	// don't walk the whole tree pointlessly.
	dirHint := pkgSuffix
	if i := strings.LastIndex(pkgSuffix, "/"); i >= 0 {
		dirHint = pkgSuffix[i+1:]
	}
	_ = filepath.WalkDir(peerPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Must live under a directory whose path ends with the
		// contract package suffix.
		dir := filepath.Dir(path)
		if !strings.HasSuffix(dir, string(filepath.Separator)+pkgSuffix) &&
			!strings.HasSuffix(filepath.Base(dir), dirHint) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		// Look for a top-level declaration of the symbol.
		if declares(data, symbolName) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// declares reports whether the file appears to declare a top-level
// Go identifier with the given name. Heuristic: matches `type Name`,
// `func Name`, `var Name`, `const Name`. Cheap token scan; the
// passive signal tolerates the occasional miss.
func declares(data []byte, name string) bool {
	prefixes := [][]byte{
		[]byte("type " + name + " "),
		[]byte("type " + name + "\t"),
		[]byte("type " + name + "("),
		[]byte("type " + name + "{"),
		[]byte("type " + name + " ="),
		[]byte("func " + name + "("),
		[]byte("var " + name + " "),
		[]byte("var " + name + "\t"),
		[]byte("var " + name + " ="),
		[]byte("const " + name + " "),
		[]byte("const " + name + "\t"),
		[]byte("const " + name + " ="),
	}
	for _, p := range prefixes {
		if bytes.Contains(data, p) {
			return true
		}
	}
	return false
}
