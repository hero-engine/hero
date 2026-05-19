package codescan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
)

// WriteGraph writes a Result into the unified knowledge graph.
//
// Idempotent: re-running on the same Result produces no new history rows.
// Updating a package re-asserts its files, symbols, and edges; stale edges
// from the prior version are invalidated transparently by graph.UpsertNode.
//
// Node types written: Repo, Package, File, Symbol.
// Edge types written: belongs_to, defines, imports.
func WriteGraph(result *Result, store *graph.Store) (*GraphWriteSummary, error) {
	if result == nil || store == nil {
		return nil, fmt.Errorf("codescan: WriteGraph requires non-nil Result and Store")
	}

	repoKey := repoKeyFor(result.ProjectRoot)
	source := map[string]any{"kind": "codescan"}

	// Code is intrinsically engineering content — see DSKG spec write-path
	// rules. Repo is in globalNodeTypes so its Domain stays empty; every
	// other node codescan writes (Package, File, Symbol) carries the
	// engineering tag, and edges inherit from the from-node.
	const codeDomain = "engineering"

	summary := &GraphWriteSummary{}

	repoID, err := store.UpsertNode(&graph.Node{
		Type:        "Repo",
		Key:         repoKey,
		Props:       map[string]any{"root": result.ProjectRoot},
		Repo:        repoKey,
		ContentHash: hashAny(repoKey, result.ProjectRoot),
		Source:      source,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert Repo: %w", err)
	}
	summary.Repos++

	// Track package id by path for import-edge resolution.
	packageIDs := make(map[string]int64, len(result.Packages))

	for i := range result.Packages {
		pkg := &result.Packages[i]
		pkgKey := nodeKey(repoKey, pkg.Path)

		pkgProps := map[string]any{
			"name":       pkg.Name,
			"path":       pkg.Path,
			"language":   pkg.Language,
			"file_count": pkg.FileCount,
			"line_count": pkg.LineCount,
		}
		if pkg.Doc != "" {
			pkgProps["doc"] = pkg.Doc
		}

		pkgID, err := store.UpsertNode(&graph.Node{
			Type:        "Package",
			Domain:      codeDomain,
			Key:         pkgKey,
			Props:       pkgProps,
			Repo:        repoKey,
			ContentHash: hashPackage(pkg),
			Source:      source,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert Package %s: %w", pkg.Path, err)
		}
		packageIDs[pkg.Path] = pkgID
		summary.Packages++

		if _, err := store.UpsertEdge(&graph.Edge{
			FromID: pkgID, ToID: repoID, Type: "belongs_to", Repo: repoKey, Source: source,
		}); err != nil {
			return nil, fmt.Errorf("edge Package→Repo: %w", err)
		}
		summary.Edges++

		// Files in this package.
		fileIDs := make(map[string]int64, len(pkg.Files))
		for _, fp := range pkg.Files {
			fileKey := nodeKey(repoKey, fp)
			fileID, err := store.UpsertNode(&graph.Node{
				Type:        "File",
				Domain:      codeDomain,
				Key:         fileKey,
				Props:       map[string]any{"path": fp, "language": pkg.Language},
				Repo:        repoKey,
				ContentHash: hashAny("file", fileKey, pkg.Language),
				Source:      source,
			})
			if err != nil {
				return nil, fmt.Errorf("upsert File %s: %w", fp, err)
			}
			fileIDs[fp] = fileID
			summary.Files++

			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: fileID, ToID: pkgID, Type: "belongs_to", Repo: repoKey, Source: source,
			}); err != nil {
				return nil, fmt.Errorf("edge File→Package: %w", err)
			}
			summary.Edges++
		}

		// Symbols defined in this package.
		for _, sym := range pkg.Symbols {
			symKey := symbolKey(repoKey, pkg.Path, sym)
			symProps := map[string]any{
				"name":     sym.Name,
				"kind":     string(sym.Kind),
				"exported": sym.Exported,
				"line":     sym.Line,
			}
			if sym.Signature != "" {
				symProps["signature"] = sym.Signature
			}
			if sym.Receiver != "" {
				symProps["receiver"] = sym.Receiver
			}
			if sym.File != "" {
				symProps["file"] = sym.File
			}
			if sym.Doc != "" {
				symProps["doc"] = sym.Doc
			}

			symID, err := store.UpsertNode(&graph.Node{
				Type:        "Symbol",
				Domain:      codeDomain,
				Key:         symKey,
				Props:       symProps,
				Repo:        repoKey,
				ContentHash: hashAny("sym", symKey, sym.Signature, sym.Doc),
				Source:      source,
			})
			if err != nil {
				return nil, fmt.Errorf("upsert Symbol %s: %w", symKey, err)
			}
			summary.Symbols++

			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: symID, ToID: pkgID, Type: "belongs_to", Repo: repoKey, Source: source,
			}); err != nil {
				return nil, fmt.Errorf("edge Symbol→Package: %w", err)
			}
			summary.Edges++

			if fileID, ok := fileIDs[sym.File]; ok && sym.File != "" {
				if _, err := store.UpsertEdge(&graph.Edge{
					FromID: fileID, ToID: symID, Type: "defines",
					Props:  map[string]any{"line": sym.Line},
					Repo:   repoKey, Source: source,
				}); err != nil {
					return nil, fmt.Errorf("edge File→Symbol defines: %w", err)
				}
				summary.Edges++
			}
		}
	}

	// imports edges (Package → Package). Only resolved within this Result;
	// cross-repo imports become first-class once adjacent-repo scan lands.
	for _, dep := range result.DepGraph {
		fromID, fromOK := packageIDs[dep.From]
		toID, toOK := packageIDs[dep.To]
		if !fromOK || !toOK {
			continue
		}
		if _, err := store.UpsertEdge(&graph.Edge{
			FromID: fromID, ToID: toID, Type: "imports", Repo: repoKey, Source: source,
		}); err != nil {
			return nil, fmt.Errorf("edge imports %s→%s: %w", dep.From, dep.To, err)
		}
		summary.Edges++
	}

	return summary, nil
}

// GraphWriteSummary reports counts written by WriteGraph for diagnostics.
type GraphWriteSummary struct {
	Repos    int
	Packages int
	Files    int
	Symbols  int
	Edges    int
}

func repoKeyFor(projectRoot string) string {
	if projectRoot == "" {
		return "."
	}
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		return filepath.Base(projectRoot)
	}
	return gitutil.RepoKey(abs)
}

func nodeKey(repoKey, path string) string {
	return repoKey + ":" + filepath.ToSlash(path)
}

// symbolKey produces a stable identifier for a symbol within a package.
// Methods include the receiver so overloads on different types don't
// collide.
func symbolKey(repoKey, pkgPath string, sym Symbol) string {
	name := sym.Name
	if sym.Receiver != "" {
		name = "(" + sym.Receiver + ")." + name
	}
	return strings.Join([]string{repoKey, filepath.ToSlash(pkgPath), name}, ":")
}

// hashPackage hashes the structural content of a package (paths, file
// list, symbol signatures) so unchanged packages produce the same hash
// across runs and re-ingest is a no-op.
func hashPackage(p *Package) string {
	type sigSym struct {
		Name, Kind, Receiver, Signature string
		Line                            int
		Exported                        bool
	}
	syms := make([]sigSym, len(p.Symbols))
	for i, s := range p.Symbols {
		syms[i] = sigSym{
			Name:      s.Name,
			Kind:      string(s.Kind),
			Receiver:  s.Receiver,
			Signature: s.Signature,
			Line:      s.Line,
			Exported:  s.Exported,
		}
	}
	payload := struct {
		Path, Language string
		Files          []string
		Symbols        []sigSym
		LineCount      int
	}{p.Path, p.Language, p.Files, syms, p.LineCount}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hashAny(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
