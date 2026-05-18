package now

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoForbiddenImports asserts that the Now home package does NOT
// depend on chat dispatch or agent-runner internals. Chat dispatch is
// owned by hero-chat-and-model and reached via the shell's shared
// chat-input fragment; the runner is owned by hero-agents-home. See
// the hero-now-home spec's "Constraints" section.
func TestNoForbiddenImports(t *testing.T) {
	forbidden := []string{
		"github.com/hero-engine/hero/internal/serve/chat",
		"github.com/hero-engine/hero/internal/runner",
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	fset := token.NewFileSet()
	entries, err := os.ReadDir(wd)
	if err != nil {
		t.Fatalf("read now dir %s: %v", wd, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(wd, entry.Name())
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, imp := range file.Imports {
			val := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range forbidden {
				if val == bad || strings.HasPrefix(val, bad+"/") {
					t.Errorf("%s: forbidden import %q (now must not depend on chat/runner)", entry.Name(), val)
				}
			}
		}
	}
}
