package cloud_test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// TestCloudImportBoundary walks every package under cloud/... and
// cmd/hero-cloud/... and confirms each only imports from:
//
//   - the Go standard library
//   - third-party modules (anything outside github.com/hero-engine/hero/)
//   - github.com/hero-engine/hero/contracts/... (the shared seam)
//
// Any other in-repo import will break the hero-cloud-split Phase 1 cut,
// because anything cloud-side code transitively pulls in must travel
// along with it to the new hero-cloud repo. The contracts/ package is
// the only sanctioned bridge.
//
// This is the symmetric mirror of TestContractsImportBoundary in
// contracts/contracts_boundary_test.go: that one enforces that nothing
// flows out of contracts/ into the rest of the repo; this one enforces
// that nothing flows into cloud-side code from outside contracts/.
func TestCloudImportBoundary(t *testing.T) {
	const (
		modulePrefix     = "github.com/hero-engine/hero"
		cloudSubtree     = "github.com/hero-engine/hero/cloud"
		cmdCloudSubtree  = "github.com/hero-engine/hero/cmd/hero-cloud"
		contractsSubtree = "github.com/hero-engine/hero/contracts"
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

	inCloud := func(p string) bool {
		return strings.HasPrefix(p, cloudSubtree) || strings.HasPrefix(p, cmdCloudSubtree)
	}

	dec := json.NewDecoder(strings.NewReader(string(out)))
	var violations []string

	for dec.More() {
		var p pkgInfo
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("decode go list output: %v", err)
		}
		// Only audit packages under cloud/... and cmd/hero-cloud/...
		if !inCloud(p.ImportPath) {
			continue
		}
		for _, imp := range p.Imports {
			if !strings.HasPrefix(imp, modulePrefix) {
				continue // stdlib or third-party — fine
			}
			if strings.HasPrefix(imp, contractsSubtree) {
				continue // contracts/ is the sanctioned bridge
			}
			if inCloud(imp) {
				continue // cloud-internal imports travel with the cut
			}
			violations = append(violations,
				p.ImportPath+" (dir "+p.Dir+") imports forbidden in-repo path "+imp+
					" — this will break the hero-cloud-split Phase 1 cut; route through contracts/ instead")
		}
	}

	if len(violations) > 0 {
		t.Fatalf("cloud/... and cmd/hero-cloud/... must depend only on stdlib, third-party, and contracts/... — forbidden imports found:\n  %s",
			strings.Join(violations, "\n  "))
	}
}
