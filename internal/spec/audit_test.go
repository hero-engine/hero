package spec

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindAuditReport_ShipClean(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "planning", "features", "test-slug")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}

	auditContent := `# Delivery audit — test-slug

**Audited:** git diff main...HEAD
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] Do X — internal/foo.go:42
`
	if err := os.WriteFile(filepath.Join(specDir, "delivery-audit.md"), []byte(auditContent), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Spec{Path: filepath.Join(specDir, "spec.md")}
	result := FindAuditReport(s)

	if !result.Found {
		t.Fatal("expected audit report to be found")
	}
	if result.Verdict != "SHIP" {
		t.Errorf("Verdict = %q, want SHIP", result.Verdict)
	}
	if result.Surface != "clean" {
		t.Errorf("Surface = %q, want clean", result.Surface)
	}
}

func TestFindAuditReport_ShipNoteworthy(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "specs", "test-slug")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}

	auditContent := `# Delivery audit — test-slug

**Audited:** git diff HEAD~3
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] Do X
- [~] Do Y — partial
`
	if err := os.WriteFile(filepath.Join(specDir, "delivery-audit.md"), []byte(auditContent), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Spec{Path: filepath.Join(specDir, "spec.md")}
	result := FindAuditReport(s)

	if !result.Found {
		t.Fatal("expected audit report to be found")
	}
	if result.Verdict != "SHIP" {
		t.Errorf("Verdict = %q, want SHIP", result.Verdict)
	}
	if result.Surface != "noteworthy" {
		t.Errorf("Surface = %q, want noteworthy", result.Surface)
	}
}

func TestFindAuditReport_Hold(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "planning", "bugs", "test-slug")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}

	auditContent := `# Delivery audit — test-slug

**Audited:** git diff main...HEAD
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria
- [✗] Do X — no evidence found
`
	if err := os.WriteFile(filepath.Join(specDir, "delivery-audit.md"), []byte(auditContent), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Spec{Path: filepath.Join(specDir, "spec.md")}
	result := FindAuditReport(s)

	if !result.Found {
		t.Fatal("expected audit report to be found")
	}
	if result.Verdict != "HOLD" {
		t.Errorf("Verdict = %q, want HOLD", result.Verdict)
	}
}

func TestFindAuditReport_Missing(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "planning", "features", "test-slug")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// No audit file written

	s := &Spec{Path: filepath.Join(specDir, "spec.md")}
	result := FindAuditReport(s)

	if result.Found {
		t.Error("expected audit report to not be found")
	}
}

func TestFindAuditReport_NilSpec(t *testing.T) {
	result := FindAuditReport(nil)
	if result.Found {
		t.Error("expected audit report to not be found for nil spec")
	}
}

func TestFindAuditReport_MalformedHeader(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "specs", "test-slug")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Missing verdict line
	auditContent := `# Delivery audit — test-slug

Some text without the standard header format.
`
	if err := os.WriteFile(filepath.Join(specDir, "delivery-audit.md"), []byte(auditContent), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Spec{Path: filepath.Join(specDir, "spec.md")}
	result := FindAuditReport(s)

	if !result.Found {
		t.Fatal("file exists so Found should be true")
	}
	if result.Verdict != "" {
		t.Errorf("Verdict = %q, want empty (malformed)", result.Verdict)
	}
}

func TestFindAuditReportInDir(t *testing.T) {
	dir := t.TempDir()
	auditContent := `# Delivery audit — test

**Verdict:** SHIP
**Surface:** clean
`
	if err := os.WriteFile(filepath.Join(dir, "delivery-audit.md"), []byte(auditContent), 0o644); err != nil {
		t.Fatal(err)
	}

	result := FindAuditReportInDir(dir)
	if !result.Found {
		t.Fatal("expected audit to be found in dir")
	}
	if result.Verdict != "SHIP" {
		t.Errorf("Verdict = %q, want SHIP", result.Verdict)
	}
}

// Silence the unused import warning for time.
var _ = time.Now
