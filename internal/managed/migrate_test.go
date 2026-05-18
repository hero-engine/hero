package managed

import (
	"strings"
	"testing"
)

func TestDetectLegacySnapshotBlock(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantFound bool
	}{
		{
			name:      "no legacy block",
			input:     "# Project\n\nJust user content.\n",
			wantFound: false,
		},
		{
			name: "complete legacy pair",
			input: "# Project\n\n" +
				legacySnapshotStart + "\n" +
				"Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md).\n" +
				legacySnapshotEnd + "\n",
			wantFound: true,
		},
		{
			name:      "start marker only — leave alone",
			input:     "# Project\n\n" + legacySnapshotStart + "\nincomplete\n",
			wantFound: false,
		},
		{
			name:      "end marker only — no start, no detection",
			input:     "# Project\n\n" + legacySnapshotEnd + "\n",
			wantFound: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, found := detectLegacySnapshotBlock(c.input)
			if found != c.wantFound {
				t.Errorf("found: got %v want %v", found, c.wantFound)
			}
		})
	}
}

func TestStripLegacySnapshotBlock_RemovesBlockPreservesNeighbours(t *testing.T) {
	input := "# AGENTS.md\n\nHeader stuff.\n\n" +
		legacySnapshotStart + "\n" +
		"Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md).\n" +
		legacySnapshotEnd + "\n\n" +
		"Footer stuff.\n"
	out := stripLegacySnapshotBlock(input)

	if strings.Contains(out, legacySnapshotStart) {
		t.Errorf("start marker should have been removed:\n%s", out)
	}
	if strings.Contains(out, legacySnapshotEnd) {
		t.Errorf("end marker should have been removed:\n%s", out)
	}
	if !strings.Contains(out, "Header stuff.") {
		t.Errorf("header lost:\n%s", out)
	}
	if !strings.Contains(out, "Footer stuff.") {
		t.Errorf("footer lost:\n%s", out)
	}
}

func TestStripLegacySnapshotBlock_Idempotent(t *testing.T) {
	input := "# AGENTS.md\n\n" +
		legacySnapshotStart + "\n" +
		"Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md).\n" +
		legacySnapshotEnd + "\n"
	once := stripLegacySnapshotBlock(input)
	twice := stripLegacySnapshotBlock(once)
	if once != twice {
		t.Errorf("not idempotent:\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}

func TestStripLegacySnapshotBlock_NoBlockIsNoOp(t *testing.T) {
	input := "# AGENTS.md\n\nNo block here.\n"
	out := stripLegacySnapshotBlock(input)
	if out != input {
		t.Errorf("expected input unchanged, got:\n%s", out)
	}
}

func TestStripLegacySnapshotBlock_FileEntirelyTheBlock(t *testing.T) {
	input := legacySnapshotStart + "\n" + PointerLineDummy() + "\n" + legacySnapshotEnd + "\n"
	out := stripLegacySnapshotBlock(input)
	if out != "" {
		t.Errorf("expected empty result, got %q", out)
	}
}

// PointerLineDummy is a local helper so this test file doesn't depend
// on the snapshot package (which would create a cycle managed →
// snapshot).
func PointerLineDummy() string {
	return "Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md)."
}
