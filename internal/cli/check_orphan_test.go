package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDetectOrphanInstructionFiles_FreshCloneHealthyInstallNotOrphaned pins
// the fix for the fresh-clone false positive: install-state.json is
// gitignored, so on a clone it is absent and PreviouslyInstalledTargets
// returns nil. A healthy install must still be recognized via the
// filesystem probe (InferInstalledTargets), so a present-and-managed
// instruction file is NOT reported as orphaned.
func TestDetectOrphanInstructionFiles_FreshCloneHealthyInstallNotOrphaned(t *testing.T) {
	root := t.TempDir()

	// Healthy claude install, filesystem-only (no install-state.json):
	// a .claude/ content dir makes InferInstalledTargets detect claude.
	if err := os.MkdirAll(filepath.Join(root, ".claude", "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# CLAUDE.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Sanity: no persisted state, mirroring a fresh clone.
	if _, err := os.Stat(filepath.Join(root, ".hero", "install-state.json")); err == nil {
		t.Fatal("fixture should have no install-state.json")
	}

	if got := detectOrphanInstructionFiles(root); len(got) != 0 {
		t.Errorf("CLAUDE.md with a healthy on-disk install must not be orphaned, got %+v", got)
	}
}

// TestDetectOrphanInstructionFiles_GenuineOrphanStillFlagged confirms the
// fix did not blunt the check: an instruction file whose target is neither
// recorded nor inferable is still reported.
func TestDetectOrphanInstructionFiles_GenuineOrphanStillFlagged(t *testing.T) {
	root := t.TempDir()

	// AGENTS.md present, but no non-claude content dir, no persisted state —
	// nothing signals that any non-claude target was ever installed.
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# AGENTS.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := detectOrphanInstructionFiles(root)
	if len(got) != 1 || got[0].file != "AGENTS.md" {
		t.Errorf("genuinely orphaned AGENTS.md must still be flagged, got %+v", got)
	}
}
