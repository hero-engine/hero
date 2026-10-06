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

// TestFindAuditReport_FlatSiblingCollision reproduces the directory-sharing
// false positive: two flat-named specs (initiative children, not yet given
// their own subdirectory) share a folder. A generic delivery-audit.md left
// over from spec-a's delivery must NOT satisfy Gate 2 for spec-b just
// because it happens to sit at the conventional filename in the same dir.
func TestFindAuditReport_FlatSiblingCollision(t *testing.T) {
	dir := t.TempDir()
	initDir := filepath.Join(dir, "planning", "initiatives", "my-initiative")
	if err := os.MkdirAll(initDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Leftover generic report from a different, already-completed sibling.
	auditContent := `# Delivery audit — spec-a

**Verdict:** SHIP
**Surface:** clean
`
	if err := os.WriteFile(filepath.Join(initDir, "delivery-audit.md"), []byte(auditContent), 0o644); err != nil {
		t.Fatal(err)
	}

	specB := &Spec{
		Slug: "spec-b",
		Path: filepath.Join(initDir, "spec-b.md"),
	}
	result := FindAuditReport(specB)

	if result.Found {
		t.Fatal("expected spec-a's leftover report to NOT satisfy spec-b's gate")
	}
	if !result.SlugMismatch {
		t.Error("expected SlugMismatch = true")
	}
	if result.ReportSlug != "spec-a" {
		t.Errorf("ReportSlug = %q, want spec-a", result.ReportSlug)
	}
}

// TestFindAuditReport_FlatSiblings_EachOwnReport is the positive case for
// the same layout: each flat sibling has its own slug-scoped report, and
// both resolve correctly despite sharing a directory.
func TestFindAuditReport_FlatSiblings_EachOwnReport(t *testing.T) {
	dir := t.TempDir()
	initDir := filepath.Join(dir, "planning", "initiatives", "my-initiative")
	if err := os.MkdirAll(initDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeAudit := func(name, slug, verdict string) {
		content := "# Delivery audit — " + slug + "\n\n**Verdict:** " + verdict + "\n**Surface:** clean\n"
		if err := os.WriteFile(filepath.Join(initDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeAudit("spec-a-delivery-audit.md", "spec-a", "SHIP")
	writeAudit("spec-b-delivery-audit.md", "spec-b", "HOLD")

	specA := &Spec{Slug: "spec-a", Path: filepath.Join(initDir, "spec-a.md")}
	specB := &Spec{Slug: "spec-b", Path: filepath.Join(initDir, "spec-b.md")}

	resultA := FindAuditReport(specA)
	if !resultA.Found || resultA.Verdict != "SHIP" {
		t.Errorf("spec-a: Found=%v Verdict=%q, want Found=true Verdict=SHIP", resultA.Found, resultA.Verdict)
	}
	if resultA.Path != filepath.Join(initDir, "spec-a-delivery-audit.md") {
		t.Errorf("spec-a: Path = %q, want the slug-scoped report", resultA.Path)
	}

	resultB := FindAuditReport(specB)
	if !resultB.Found || resultB.Verdict != "HOLD" {
		t.Errorf("spec-b: Found=%v Verdict=%q, want Found=true Verdict=HOLD", resultB.Found, resultB.Verdict)
	}
	if resultB.Path != filepath.Join(initDir, "spec-b-delivery-audit.md") {
		t.Errorf("spec-b: Path = %q, want the slug-scoped report", resultB.Path)
	}
}

// TestFindAuditReport_FlatSpec_FallsBackToGenericFilename covers a flat
// spec that is the sole occupant of its initiative folder so far (no
// sibling has claimed the generic filename yet) — an older report written
// before the slug-scoped convention still satisfies the gate as long as
// its own title line matches.
func TestFindAuditReport_FlatSpec_FallsBackToGenericFilename(t *testing.T) {
	dir := t.TempDir()
	initDir := filepath.Join(dir, "planning", "initiatives", "my-initiative")
	if err := os.MkdirAll(initDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "# Delivery audit — spec-a\n\n**Verdict:** SHIP\n**Surface:** clean\n"
	if err := os.WriteFile(filepath.Join(initDir, "delivery-audit.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	specA := &Spec{Slug: "spec-a", Path: filepath.Join(initDir, "spec-a.md")}
	result := FindAuditReport(specA)
	if !result.Found {
		t.Fatal("expected fallback to the generic filename to find the matching report")
	}
	if result.Verdict != "SHIP" {
		t.Errorf("Verdict = %q, want SHIP", result.Verdict)
	}
}

// TestFindAuditReport_Stale reproduces the second collision class: an
// audit report that is genuinely for this spec but was written before the
// spec's most recent change, so it cannot speak to the current delivery.
func TestFindAuditReport_Stale(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "planning", "features", "test-slug")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}

	auditPath := filepath.Join(specDir, "delivery-audit.md")
	content := "# Delivery audit — test-slug\n\n**Verdict:** SHIP\n**Surface:** clean\n"
	if err := os.WriteFile(auditPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	old := time.Now().Add(-1 * time.Hour)
	recent := time.Now()
	if err := os.Chtimes(auditPath, old, old); err != nil {
		t.Fatal(err)
	}

	s := &Spec{
		Slug:       "test-slug",
		Path:       filepath.Join(specDir, "spec.md"),
		ModifiedAt: recent, // spec changed after the audit report was written
	}
	result := FindAuditReport(s)

	if result.Found {
		t.Fatal("expected a report predating the spec's last change to NOT satisfy the gate")
	}
	if !result.Stale {
		t.Error("expected Stale = true")
	}
}

// TestFindAuditReport_NotStaleWhenReportIsNewer is the companion positive
// case: a report written after the spec's last modification satisfies the
// gate as before.
func TestFindAuditReport_NotStaleWhenReportIsNewer(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "planning", "features", "test-slug")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}

	auditPath := filepath.Join(specDir, "delivery-audit.md")
	content := "# Delivery audit — test-slug\n\n**Verdict:** SHIP\n**Surface:** clean\n"
	if err := os.WriteFile(auditPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	older := time.Now().Add(-1 * time.Hour)

	s := &Spec{
		Slug:       "test-slug",
		Path:       filepath.Join(specDir, "spec.md"),
		ModifiedAt: older, // spec's last change predates the audit report
	}
	result := FindAuditReport(s)

	if !result.Found {
		t.Fatal("expected a report newer than the spec's last change to satisfy the gate")
	}
	if result.Stale {
		t.Error("expected Stale = false")
	}
}

func TestParseAuditSlug(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"em dash", "# Delivery audit — my-slug\n", "my-slug"},
		{"em dash, capital Audit", "# Delivery Audit — my-slug\n", "my-slug"},
		{"plain hyphen fallback", "# Delivery audit - my-slug\n", "my-slug"},
		{"no title line", "Some text without a title.\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseAuditSlug(tc.content); got != tc.want {
				t.Errorf("parseAuditSlug(%q) = %q, want %q", tc.content, got, tc.want)
			}
		})
	}
}
