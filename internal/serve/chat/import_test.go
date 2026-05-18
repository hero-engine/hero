package chat

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoRunnerImports enforces the build-time boundary called out in
// the spec: internal/serve/chat/* MUST NOT import internal/runner/*.
// hero serve is the dispatcher; inference and runner code live in
// hero-code.
func TestNoRunnerImports(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		// Test files are allowed to import anything; this rule is
		// about production code shipping a dispatcher with no
		// inference paths.
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(".", name)
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, imp := range f.Imports {
			val := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(val, "internal/runner") {
				t.Errorf("%s imports forbidden package %s — chat must not depend on internal/runner/*", path, val)
			}
		}
	}
}
