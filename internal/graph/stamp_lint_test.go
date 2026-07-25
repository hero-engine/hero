package graph

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoUnstampedUpsertCalls walks every .go file under internal/ and
// fails if it finds an `UpsertNode(&graph.Node{...})` call whose
// composite literal omits the Domain field AND whose Type is not a
// global-allow-list literal. Runtime checks catch this at upsert
// time, but the lint surfaces the violation at build time so a
// stamping omission in a new ingest path never reaches main.
//
// UpsertEdge is intentionally NOT checked: edges inherit Domain from
// the from-node at write time per design. The trap case (from-node
// is a global type) is enforced via ErrEdgeDomainRequired at runtime.
func TestNoUnstampedUpsertCalls(t *testing.T) {
	// Files allowed to use bare composite literals — they exercise the
	// invariant rather than rely on it.
	allowList := map[string]bool{
		"internal/graph/node.go":                  true, // defines the invariant
		"internal/graph/edge.go":                  true, // defines the invariant
		"internal/graph/sync.go":                  true, // hand-built node from federation payload
		"internal/graph/alias.go":                 true, // edge inherits from from-node by design
		"internal/graph/domain_test.go":           true,
		"internal/graph/write_invariants_test.go": true,
		"internal/graph/graph_test.go":            true,
		"internal/graph/alias_test.go":            true,
		"internal/graph/sync_test.go":             true,
	}

	root := findModuleRoot(t)
	internal := filepath.Join(root, "internal")

	var violations []string
	err := filepath.Walk(internal, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if allowList[rel] {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil // skip files that don't parse; not our concern
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel.Name != "UpsertNode" {
				return true
			}
			if len(call.Args) != 1 {
				return true
			}
			// Look through &x or x for a composite literal.
			lit := compositeLitOf(call.Args[0])
			if lit == nil {
				return true
			}
			var hasDomain, typeIsGlobalLit bool
			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				ident, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				if ident.Name == "Domain" {
					hasDomain = true
				}
				if ident.Name == "Type" {
					if bl, ok := kv.Value.(*ast.BasicLit); ok && bl.Kind == token.STRING {
						s := strings.Trim(bl.Value, `"`)
						if IsGlobalNodeType(s) {
							typeIsGlobalLit = true
						}
					}
				}
			}
			if !hasDomain && !typeIsGlobalLit {
				pos := fset.Position(call.Pos())
				violations = append(violations, formatViolation(rel, pos.Line, sel.Sel.Name))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("found %d unstamped graph upsert call(s) — set Domain explicitly or use graph.DomainFor(cfg, hint):\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}

func compositeLitOf(e ast.Expr) *ast.CompositeLit {
	switch v := e.(type) {
	case *ast.UnaryExpr:
		if v.Op == token.AND {
			if cl, ok := v.X.(*ast.CompositeLit); ok {
				return cl
			}
		}
	case *ast.CompositeLit:
		return v
	}
	return nil
}

func formatViolation(file string, line int, method string) string {
	return file + ":" + itoa(line) + ": " + method + " call without Domain field"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [16]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
