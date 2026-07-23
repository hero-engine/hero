package contracts_test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// TestContractsImportBoundary walks every package under contracts/... and
// confirms none of them, including the portable attention contract, import
// from anywhere else inside this repository.
// The contracts package must remain a leaf so a future external consumer
// (notably the hero-cloud repo) can depend on it without dragging the
// rest of hero along.
//
// Adding `import "github.com/hero-engine/hero/internal/foo"` (or any
// non-contracts in-repo path) into contracts/... must fail this test
// with a message naming the offending file and import.
func TestContractsImportBoundary(t *testing.T) {
	const (
		modulePrefix   = "github.com/hero-engine/hero"
		allowedSubtree = "github.com/hero-engine/hero/contracts"
	)

	out, err := exec.Command("go", "list", "-deps", "-json", "./...").Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("go list failed: %v\nstderr:\n%s", err, ee.Stderr)
		}
		t.Fatalf("go list failed: %v", err)
	}

	type pkgInfo struct {
		ImportPath string   `json:"ImportPath"`
		Dir        string   `json:"Dir"`
		GoFiles    []string `json:"GoFiles"`
		Imports    []string `json:"Imports"`
	}

	dec := json.NewDecoder(strings.NewReader(string(out)))
	var violations []string

	for dec.More() {
		var p pkgInfo
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("decode go list output: %v", err)
		}
		// Only audit packages that belong to the contracts subtree.
		if !strings.HasPrefix(p.ImportPath, allowedSubtree) {
			continue
		}
		for _, imp := range p.Imports {
			if !strings.HasPrefix(imp, modulePrefix) {
				continue // stdlib or third-party — fine
			}
			if strings.HasPrefix(imp, allowedSubtree) {
				continue // other contracts/ paths are fine
			}
			violations = append(violations, p.ImportPath+" (dir "+p.Dir+") imports forbidden in-repo path "+imp)
		}
	}

	if len(violations) > 0 {
		t.Fatalf("contracts/ must be a leaf package — forbidden imports found:\n  %s",
			strings.Join(violations, "\n  "))
	}
}
