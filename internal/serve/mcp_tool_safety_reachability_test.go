package serve

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// mcp_tool_safety_reachability_test.go — the STRUCTURAL half of the safety
// guard. TestConditionalWritersAreNotReadOnly pins the five known writers by
// name; this test catches a *future* one nobody remembered to list.
//
// It builds a name-resolved call graph over internal/ and asserts that no tool
// classed safetyRead/safetyAnalyze can transitively reach a filesystem write
// primitive (os.WriteFile / Create / Remove / RemoveAll / Rename). Every Hero
// tool that writes user-authored state funnels through os.WriteFile
// (active.Register, contract.Link, errpattern.SavePattern, hero_plan,
// snapshot archive, tracking.UpdateSpecFrontmatter — all os.WriteFile), while
// the index/graph cache the auditor flagged writes via MkdirAll + the SQLite
// driver and never touches os.WriteFile. So "reaches os.WriteFile" is a clean
// proxy for "mutates user state," with no cache false-positives.
//
// Precision vs the pinned test: this walks direct calls, same-package
// calls/methods, and package-function calls (the idiom every current write
// chain uses) transitively. Its one blind spot is a write reached through a
// method on a NON-receiver type (e.g. someField.Save()); those are not
// resolved without full type info, and are why TestConditionalWritersAreNotReadOnly
// stays as a belt-and-suspenders net. A new read-classed tool that writes via a
// plain os.WriteFile — anywhere in its transitive package-function reach — fails
// here automatically.

const repoModulePath = "github.com/hero-engine/hero"

// writeSelectors are the os.* functions that mutate the filesystem. Read-side
// helpers (os.Open, os.OpenFile, os.ReadFile, os.Stat) are deliberately absent
// so a read tool that opens a file for reading is not flagged.
var writeSelectors = map[string]bool{
	"WriteFile": true, "Create": true, "Remove": true, "RemoveAll": true, "Rename": true,
}

func TestReadClassedToolsDoNotReachWritePrimitives(t *testing.T) {
	root := repoRootFromTest(t)
	idx := buildFuncIndex(t, root)

	handlerMethods := parseToolHandlerMethods(t, filepath.Join(root, "internal", "serve", "mcp_dispatch.go"))
	servePkg := repoModulePath + "/internal/serve"

	checked := 0
	for name, class := range toolSafetyClasses() {
		if class == safetyMutate {
			continue
		}
		method, ok := handlerMethods[name]
		if !ok {
			t.Errorf("%s: classed read/analyze but no handler found in toolHandlers() "+
				"(renamed? the mapping this guard relies on is stale)", name)
			continue
		}
		checked++
		if path := idx.reachesWrite(servePkg + "." + method); path != "" {
			t.Errorf("%s (%s) is classed read/analyze but its handler can write to disk: %s\n"+
				"    A tool that can write must be safetyMutate — readOnlyHint:true tells a "+
				"client it is safe to call unprompted.", name, method, path)
		}
	}
	// Anti-vacuity: if the handler mapping broke, we would check nothing and
	// pass silently. Require a plausible number of read/analyze tools.
	if checked < 20 {
		t.Fatalf("only %d read/analyze tools checked; expected 20+ — the handler "+
			"mapping or safety map likely failed to parse, making this guard vacuous", checked)
	}
	t.Logf("verified %d read/analyze tools reach no write primitive", checked)
}

// --- call-graph machinery ---------------------------------------------------

type funcInfo struct {
	decl    *ast.FuncDecl
	imports map[string]string // local name -> import path
	pkgPath string
}

type funcIndex struct {
	byKey map[string][]funcInfo // "<pkgPath>.<name>" -> decls (func or method)
	memo  map[string]string     // key -> write path ("" = none), for BFS caching
}

// buildFuncIndex parses every non-test .go file in the first-party module
// (rooted at root) and indexes each top-level function and method by
// "<importPath>.<name>". Indexing the whole module — not just internal/ —
// means a write helper added anywhere in first-party code is still on the
// graph, so a handler that delegates to it is caught.
func buildFuncIndex(t *testing.T, root string) *funcIndex {
	t.Helper()
	idx := &funcIndex{byKey: map[string][]funcInfo{}, memo: map[string]string{}}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "testdata", "vendor", "node_modules", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return nil // unparsable file: skip, not this guard's concern
		}
		rel, _ := filepath.Rel(root, filepath.Dir(path)) // e.g. "internal/active"
		pkgPath := repoModulePath
		if rel != "." {
			pkgPath = repoModulePath + "/" + filepath.ToSlash(rel)
		}
		imports := map[string]string{}
		for _, imp := range file.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			name := p[strings.LastIndexByte(p, '/')+1:]
			if imp.Name != nil {
				name = imp.Name.Name
			}
			imports[name] = p
		}
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			key := pkgPath + "." + fd.Name.Name
			idx.byKey[key] = append(idx.byKey[key], funcInfo{decl: fd, imports: imports, pkgPath: pkgPath})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("indexing %s: %v", root, err)
	}
	if len(idx.byKey) == 0 {
		t.Fatalf("indexed 0 functions under %s — parse failure would make this guard vacuous", root)
	}
	return idx
}

// reachesWrite returns a non-empty call path ("a -> b -> os.WriteFile") if the
// function(s) named by key can transitively reach a write primitive.
func (idx *funcIndex) reachesWrite(key string) string {
	if cached, ok := idx.memo[key]; ok {
		return cached
	}
	idx.memo[key] = "" // break cycles: assume no-write until proven otherwise
	infos, ok := idx.byKey[key]
	if !ok {
		return "" // outside internal/ (stdlib, external dep) and not a write leaf
	}
	shortKey := key[strings.LastIndexByte(key, '/')+1:]
	for _, fi := range infos {
		var found string
		ast.Inspect(fi.decl.Body, func(n ast.Node) bool {
			if found != "" {
				return false
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			leaf, callee := fi.resolveCall(call)
			if leaf != "" {
				found = shortKey + " -> " + leaf
				return false
			}
			if callee != "" {
				if sub := idx.reachesWrite(callee); sub != "" {
					found = shortKey + " -> " + sub
					return false
				}
			}
			return true
		})
		if found != "" {
			idx.memo[key] = found
			return found
		}
	}
	return ""
}

// resolveCall classifies one call expression. It returns either a non-empty
// write-leaf label (e.g. "os.WriteFile") OR a callee key to recurse into.
func (fi funcInfo) resolveCall(call *ast.CallExpr) (leaf, callee string) {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		x, ok := fun.X.(*ast.Ident)
		if !ok {
			return "", "" // chained/complex receiver — not resolved
		}
		if impPath, isPkg := fi.imports[x.Name]; isPkg {
			if impPath == "os" && writeSelectors[fun.Sel.Name] {
				return "os." + fun.Sel.Name, ""
			}
			return "", impPath + "." + fun.Sel.Name
		}
		// x is a value (receiver / local): treat as a same-package method.
		return "", fi.pkgPath + "." + fun.Sel.Name
	case *ast.Ident:
		return "", fi.pkgPath + "." + fun.Name
	}
	return "", ""
}

// parseToolHandlerMethods extracts the "hero_x" -> "toolY" mapping from the
// map literal in toolHandlers(), so the guard checks the actual handler each
// tool dispatches to.
func parseToolHandlerMethods(t *testing.T, dispatchPath string) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, dispatchPath, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", dispatchPath, err)
	}
	out := map[string]string{}
	ast.Inspect(file, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "toolHandlers" {
			return true
		}
		ast.Inspect(fd.Body, func(m ast.Node) bool {
			kv, ok := m.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			keyLit, ok := kv.Key.(*ast.BasicLit)
			if !ok || keyLit.Kind != token.STRING {
				return true
			}
			sel, ok := kv.Value.(*ast.SelectorExpr) // s.toolX
			if !ok {
				return true
			}
			name, _ := strconv.Unquote(keyLit.Value)
			out[name] = sel.Sel.Name
			return true
		})
		return false
	})
	if len(out) == 0 {
		t.Fatalf("parsed 0 handler methods from %s — the map shape changed", dispatchPath)
	}
	return out
}

func repoRootFromTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repo root (go.mod) from test file")
		}
		dir = parent
	}
}
