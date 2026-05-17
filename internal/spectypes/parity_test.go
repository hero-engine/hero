package spectypes

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/triage"
)

// TestLintParity walks every spec under .hero/ and confirms the
// registry-driven validator (ValidateTypeAndStatus) produces the same
// type+status verdict as the legacy hardcoded validator
// (triage.ValidateStructure).
//
// This is the parity gate called out in §4 of the spec-type-registry
// feature spec. It is the safety net before any call-site migration:
// if the registry diverges from the legacy validator for any spec in
// the workspace, this test fails and the divergence must be resolved
// (either by fixing the registry's lifecycle declaration or by
// updating the spec frontmatter).
func TestLintParity(t *testing.T) {
	root := repoRoot(t)
	heroDir := filepath.Join(root, ".hero")
	if _, err := os.Stat(heroDir); err != nil {
		t.Skipf(".hero dir not present at %s: %v", heroDir, err)
	}

	reg, err := Load("engineering")
	if err != nil {
		t.Fatalf("Load registry: %v", err)
	}

	var checked int
	var divergent []string

	err = fs.WalkDir(os.DirFS(heroDir), ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip unreadable
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		// Only check spec.md files (the spec format).
		if filepath.Base(path) != "spec.md" {
			return nil
		}
		full := filepath.Join(heroDir, path)
		s, err := spec.ParseFile(full)
		if err != nil {
			return nil // unparseable specs are not parity targets
		}

		// Parity is asserted only for types both validators cover.
		// Per the spec, meta/knowledge types (rule, external, context,
		// note, tripwire, plan, index) stay on legacy code paths and
		// are out of scope for the registry rework.
		if !registryCovers(string(s.Type)) {
			return nil
		}
		checked++

		legacy := triage.ValidateStructure(s)
		legacyType, legacyStatus := classifyLegacy(legacy)

		regIssues := ValidateTypeAndStatus(reg, string(s.Type), string(s.Status))
		regType, regStatus := classifyRegistry(regIssues)

		// Compare type verdict universally.
		if legacyType != regType {
			divergent = append(divergent, full+" type-verdict: legacy="+legacyType+" registry="+regType)
		}
		// Compare status verdict only for types the legacy validator
		// constrained (feature, bug, convention, decision). Other
		// registered types (initiative, prd, epic, chore, intake,
		// release, sprint) had no legacy status enum — the registry's
		// new lifecycle is additive, not a parity break.
		if legacyStatusConstrained(string(s.Type)) && legacyStatus != regStatus {
			divergent = append(divergent, full+" status-verdict: legacy="+legacyStatus+" registry="+regStatus)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if checked == 0 {
		t.Fatal("no spec.md files found under .hero/ — corpus is empty?")
	}

	if len(divergent) > 0 {
		t.Errorf("registry diverged from legacy validator on %d/%d specs:", len(divergent), checked)
		// Cap noise to the first 20 entries.
		limit := len(divergent)
		if limit > 20 {
			limit = 20
		}
		for _, d := range divergent[:limit] {
			t.Errorf("  %s", d)
		}
		if len(divergent) > 20 {
			t.Errorf("  ... and %d more", len(divergent)-20)
		}
	}
	t.Logf("TestLintParity checked %d specs; %d divergent", checked, len(divergent))
}

// classifyLegacy reduces a slice of triage issues to two strings: the
// type-related verdict ("ok" | "missing" | "invalid") and the
// status-related verdict. Created-date warnings are ignored — they
// are non-blocking and not relevant to type/status parity.
func classifyLegacy(issues []triage.StructuralIssue) (typeV, statusV string) {
	typeV, statusV = "ok", "ok"
	for _, iss := range issues {
		if !iss.IsError {
			continue
		}
		switch iss.Field {
		case "type":
			if strings.Contains(iss.Message, "required") {
				typeV = "missing"
			} else {
				typeV = "invalid"
			}
		case "status":
			if strings.Contains(iss.Message, "required") {
				statusV = "missing"
			} else {
				statusV = "invalid"
			}
		}
	}
	return
}

func classifyRegistry(issues []LintIssue) (typeV, statusV string) {
	typeV, statusV = "ok", "ok"
	for _, iss := range issues {
		if !iss.IsError {
			continue
		}
		switch iss.Field {
		case "type":
			if strings.Contains(iss.Message, "required") {
				typeV = "missing"
			} else {
				typeV = "invalid"
			}
		case "status":
			if strings.Contains(iss.Message, "required") {
				statusV = "missing"
			} else {
				statusV = "invalid"
			}
		}
	}
	return
}

// registryCovers reports whether the registry intentionally covers
// this type. Per spec-type-registry §13, meta/knowledge types beyond
// decision and convention are out of scope; the legacy validator
// continues to govern them.
func registryCovers(typeName string) bool {
	switch typeName {
	case "feature", "bug", "convention", "decision", "initiative",
		"prd", "epic", "chore", "intake", "release", "sprint":
		return true
	}
	return false
}

// legacyStatusConstrained reports whether the legacy validator
// applied a status enum to this type. Only feature, bug, convention,
// decision had enforced status sets; initiative et al. fell through
// the legacy default branch ("any non-empty status is fine"). The
// registry's added lifecycles for those types are an additive
// enhancement, not a parity-relevant change.
func legacyStatusConstrained(typeName string) bool {
	switch typeName {
	case "feature", "bug", "convention", "decision":
		return true
	}
	return false
}

// repoRoot finds the repository root by walking up from the test's
// working directory looking for go.mod. Tests run with cwd inside the
// package directory; the repo root is two levels up (internal/spectypes/
// → internal/ → repo).
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatalf("could not find repo root walking up from %s", wd)
	return ""
}
