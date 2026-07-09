package drive

import (
	"strings"
	"testing"
)

func TestComposeMergeStripQuestion(t *testing.T) {
	res := CheckResult{
		Initiative: "drive", NextSpec: "child-a",
		Pause:     &PauseInfo{Category: "DesignFork", Reason: "pick an approach"},
		Completed: []string{"x"}, Remaining: []string{"child-a", "child-b"},
	}
	q := ComposeQuestion("drive", res)
	for _, want := range []string{"Drive paused", "pick an approach", "DesignFork", "child-a", "--answer"} {
		if !strings.Contains(q, want) {
			t.Errorf("question missing %q:\n%s", want, q)
		}
	}

	prior := "## My notes\nkeep me\n"
	merged := MergeQuestion(prior, q)
	if !strings.Contains(merged, "keep me") || !strings.Contains(merged, "Drive paused") {
		t.Errorf("merge should preserve prior + add block:\n%s", merged)
	}
	// Idempotent: merging again replaces rather than duplicates.
	merged2 := MergeQuestion(merged, ComposeQuestion("drive", res))
	if n := strings.Count(merged2, "Drive paused — needs you"); n != 1 {
		t.Errorf("merge not idempotent: %d blocks\n%s", n, merged2)
	}
	// Strip removes the block, keeps the rest.
	stripped := StripQuestion(merged2)
	if strings.Contains(stripped, "Drive paused") {
		t.Errorf("strip left the block:\n%s", stripped)
	}
	if !strings.Contains(stripped, "keep me") {
		t.Errorf("strip dropped prior content:\n%s", stripped)
	}
}
