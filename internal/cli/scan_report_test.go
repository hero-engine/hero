package cli

import (
	"errors"
	"strings"
	"testing"
)

// TestIngestReport_RendersAllOutcomes confirms the summary block
// surfaces all three outcomes without one masking another. Exercises
// master-ingest-restore AC-8 — per-step failure isolation: a failure
// in one step still lets every other step's outcome be visible in the
// summary block.
func TestIngestReport_RendersAllOutcomes(t *testing.T) {
	r := &ingestReport{}
	r.add(stepResult{name: "code", ok: true, detail: "1 pkg"})
	r.add(stepResult{name: "tier-2", skipped: true, reason: "no key"})
	r.add(stepResult{name: "tracker", failed: true, err: errors.New("boom")})
	r.add(stepResult{name: "git", ok: true, detail: "12 commits"})

	out := captureStdout(r.print)

	for _, want := range []string{
		"Graph ingest summary",
		"✅ code:",
		"⊘  tier-2:",
		"❌ tracker:",
		"✅ git:",
		"boom", // failure surfaces inline
		"no key",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q\n%s", want, out)
		}
	}
}

func TestIngestReport_EmptyPrintsNothing(t *testing.T) {
	r := &ingestReport{}
	out := captureStdout(r.print)
	if out != "" {
		t.Errorf("empty report printed %q", out)
	}
}
