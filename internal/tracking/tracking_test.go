package tracking

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// TestUpdateSpecFrontmatter_CompleteStampsTimestamp confirms the
// "complete" action (used by claim/release/complete bookkeeping)
// flips status, stamps completed_at, and clears the active-claim
// fields in a single write.
func TestUpdateSpecFrontmatter_CompleteStampsTimestamp(t *testing.T) {
	fixed := time.Date(2026, 5, 31, 19, 42, 8, 0, time.UTC)
	prev := spec.SwapNowFnForTest(func() time.Time { return fixed })
	t.Cleanup(func() { spec.SwapNowFnForTest(prev) })

	tmp := t.TempDir()
	specPath := filepath.Join(tmp, "spec.md")
	content := `---
title: CSV Export
type: feature
status: delivering
claimed_by: agent-a
claimed_at: 2026-05-30T08:00:00Z
---
# CSV Export
`
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := UpdateSpecFrontmatter(specPath, "complete", "agent-a", time.Time{}); err != nil {
		t.Fatalf("UpdateSpecFrontmatter: %v", err)
	}

	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "status: completed") {
		t.Errorf("missing status: completed\n%s", body)
	}
	if !strings.Contains(body, "completed_at: 2026-05-31T19:42:08Z") {
		t.Errorf("missing completed_at stamp\n%s", body)
	}
	if strings.Contains(body, "claimed_by:") {
		t.Errorf("claimed_by should be removed\n%s", body)
	}
	if strings.Contains(body, "claimed_at:") {
		t.Errorf("claimed_at should be removed\n%s", body)
	}
}
