package serve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

// TestUpdateSpecFrontmatterField_StampsCompletedAtOnStatusComplete covers the
// auto-resolve writer site at refresh.go:136. When a tracker auto-resolves to
// Done and we flip status to "completed" here, the same write must stamp
// completed_at: so the peer contract — every Go writer that flips status to
// completed produces a parseable completed_at in the same write — holds at
// this site too. The auto-archive safety net catches missed stamps later,
// but the contract is "same write."
func TestUpdateSpecFrontmatterField_StampsCompletedAtOnStatusComplete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	original := "---\ntitle: t\nslug: t\ntype: feature\nstatus: delivering\n---\n# body\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	updateSpecFrontmatterField(path, "status", "completed")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "status: completed") {
		t.Errorf("expected status: completed in frontmatter, got: %s", got)
	}
	if !strings.Contains(got, "completed_at:") {
		t.Errorf("expected completed_at: to be stamped in same write, got: %s", got)
	}
}

// TestUpdateSpecFrontmatterField_NoStampOnOtherFields confirms the stamping
// is gated on the status→completed transition and does not fire on unrelated
// writes (e.g. claimed_by, tags, or a status flip to anything other than
// completed). Without this, the auto-resolve refresher would over-stamp.
func TestUpdateSpecFrontmatterField_NoStampOnOtherFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	original := "---\ntitle: t\nslug: t\ntype: feature\nstatus: planning\n---\n# body\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	updateSpecFrontmatterField(path, "claimed_by", "agent-x")
	updateSpecFrontmatterField(path, "status", "delivering")

	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "completed_at:") {
		t.Errorf("completed_at: should not be stamped on non-complete writes, got: %s", string(data))
	}
}

// TestUpdateSpecFrontmatterField_IdempotentOnAlreadyStamped confirms that
// re-flipping an already-stamped spec leaves the original timestamp alone.
// Same idempotency contract as spec.StampCompletedAt itself; verified at
// this writer site because the helper short-circuits on the existing field.
func TestUpdateSpecFrontmatterField_IdempotentOnAlreadyStamped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	original := "---\ntitle: t\nslug: t\ntype: feature\nstatus: completed\ncompleted_at: 2025-01-01T00:00:00Z\n---\n# body\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	updateSpecFrontmatterField(path, "status", "completed")

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "completed_at: 2025-01-01T00:00:00Z") {
		t.Errorf("expected original completed_at preserved, got: %s", string(data))
	}
}

// silence unused-import false alarm for the spec helper used elsewhere in
// this package's tests via the production code under test.
var _ = spec.StampCompletedAt
