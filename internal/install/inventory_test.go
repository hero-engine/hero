package install

import (
	"os"
	"path/filepath"
	"testing"
)

// inventory_test.go — Inventory coverage, table-driven over all six targets
// per the harness-changes-cover-all-targets tripwire. Installs via the shared
// harness, then asserts the introspection Inventory reports.
//
// The seeded source (harness_test.go) ships 2 agents, 2 commands, 2 skills, so
// a healthy single-target install expects 2/2/2 (codex skills roll up to 4:
// 2 canonical + 2 commands-as-skills).

// harnessInventory runs Inventory's testable core against the harness's seeded
// content source, so expected counts derive from the same content the install
// used.
func harnessInventory(t *testing.T, h *installHarness) []TargetInventory {
	t.Helper()
	invs, err := inventoryFromFS(h.TargetDir, os.DirFS(h.SourceDir), "engineering")
	if err != nil {
		t.Fatalf("inventory: %v", err)
	}
	return invs
}

// findRow returns the inventory row for target, or fails.
func findRow(t *testing.T, invs []TargetInventory, target Target) TargetInventory {
	t.Helper()
	for _, inv := range invs {
		if inv.Target == target {
			return inv
		}
	}
	t.Fatalf("no inventory row for %s (got %d rows)", target, len(invs))
	return TargetInventory{}
}

func hasRow(invs []TargetInventory, target Target) bool {
	for _, inv := range invs {
		if inv.Target == target {
			return true
		}
	}
	return false
}

// TestInventory_AllSixTargets is the six-target gate: every target installs and
// reports a correct, complete row from its real destination paths. A t.Skip on
// any target here is a failed delivery, not a green one.
func TestInventory_AllSixTargets(t *testing.T) {
	for _, tc := range integrityTargets {
		t.Run(tc.name, func(t *testing.T) {
			h := newInstallHarness(t)
			h.Run(tc.target, nil)

			invs := harnessInventory(t, h)
			if len(invs) != 1 {
				t.Fatalf("expected exactly one installed target row, got %d: %+v", len(invs), invs)
			}
			row := findRow(t, invs, tc.target)

			if row.RootFile != tc.rootFile {
				t.Errorf("root file = %q, want %q", row.RootFile, tc.rootFile)
			}
			if row.Agents.Expected != 2 || row.Agents.Actual != 2 {
				t.Errorf("agents = %d/%d, want 2/2", row.Agents.Actual, row.Agents.Expected)
			}

			if tc.target == TargetCodex {
				if !row.Commands.NotApplicable {
					t.Errorf("codex commands must be NotApplicable")
				}
				// 2 canonical skills + 2 commands-as-skills.
				if row.Skills.Expected != 4 || row.Skills.Actual != 4 {
					t.Errorf("codex skills = %d/%d, want 4/4", row.Skills.Actual, row.Skills.Expected)
				}
			} else {
				if row.Commands.NotApplicable {
					t.Errorf("%s commands must not be NotApplicable", tc.target)
				}
				if row.Commands.Expected != 2 || row.Commands.Actual != 2 {
					t.Errorf("%s commands = %d/%d, want 2/2", tc.target, row.Commands.Actual, row.Commands.Expected)
				}
				if row.Skills.Expected != 2 || row.Skills.Actual != 2 {
					t.Errorf("%s skills = %d/%d, want 2/2", tc.target, row.Skills.Actual, row.Skills.Expected)
				}
			}
		})
	}
}

// TestInventory_CodexCommandsNotApplicable — codex's commands cell is modeled as
// NotApplicable, never as a numeric 0 that would read as a broken install.
func TestInventory_CodexCommandsNotApplicable(t *testing.T) {
	h := newInstallHarness(t)
	h.Run(TargetCodex, nil)
	row := findRow(t, harnessInventory(t, h), TargetCodex)
	if !row.Commands.NotApplicable {
		t.Fatalf("codex commands must be NotApplicable, got %+v", row.Commands)
	}
}

// TestInventory_CodexSkillsMatchInstalledDirs is the pivotal drift guard:
// inventory's expected codex skills must equal len(codexSkillDirNames(opts)) —
// the exact set the installer materializes — so the rollup can never silently
// drift from the installer. Asserts against the FUNCTION, not a literal.
func TestInventory_CodexSkillsMatchInstalledDirs(t *testing.T) {
	h := newInstallHarness(t)
	h.Run(TargetCodex, nil)

	opts := Options{SourceDir: h.SourceDir, Domain: "engineering"}
	want, err := codexSkillDirNames(opts)
	if err != nil {
		t.Fatalf("codexSkillDirNames: %v", err)
	}

	row := findRow(t, harnessInventory(t, h), TargetCodex)
	if row.Skills.Expected != len(want) {
		t.Errorf("codex skills expected = %d, want len(codexSkillDirNames) = %d", row.Skills.Expected, len(want))
	}
	if row.Skills.Actual != len(want) {
		t.Errorf("codex skills actual = %d, want %d installed dirs", row.Skills.Actual, len(want))
	}
}

// TestInventory_CopilotDetectedFromInstructionsFile guards the DetectInstalledTargets
// trap: copilot must appear in the inventory from a real runCopilot install, and
// its file marker (.github/copilot-instructions.md) must be sufficient for
// detection on its own. The existing integrity check skips copilot for exactly
// this probe gap (integrity_test.go); inventory must not.
func TestInventory_CopilotDetectedFromInstructionsFile(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir) // so copilot-instructions.md is written
	h.Run(TargetCopilot, nil)

	if !hasRow(harnessInventory(t, h), TargetCopilot) {
		t.Fatal("copilot must appear in inventory after a real install (DetectInstalledTargets trap)")
	}

	// File-marker branch in isolation: a directory with only the instructions
	// file must still detect copilot.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".github"), 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, ".github", "copilot-instructions.md")
	if err := os.WriteFile(marker, []byte("# copilot"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !targetInstalledOnDisk(TargetCopilot, dir) {
		t.Fatal("copilot must be detected via .github/copilot-instructions.md alone")
	}
}

// TestInventory_UnionSurvivesMissingInstallState — install-state.json is
// gitignored, so on a fresh clone PreviouslyInstalledTargets returns nil and
// on-disk detection must carry the whole result.
func TestInventory_UnionSurvivesMissingInstallState(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	h.Run(TargetClaude, nil)

	statePath := filepath.Join(h.TargetDir, ".hero", "install-state.json")
	if err := os.Remove(statePath); err != nil {
		t.Fatalf("remove install-state.json: %v", err)
	}
	if got := PreviouslyInstalledTargets(h.TargetDir); len(got) != 0 {
		t.Fatalf("fixture broken: persisted targets should be gone, got %v", got)
	}

	invs := harnessInventory(t, h)
	if !hasRow(invs, TargetClaude) {
		t.Fatalf("claude must still resolve from disk without install-state.json, got %d rows", len(invs))
	}
}

// TestInventory_PersistedTargetWithMissingTreeIsZero — the other union
// direction: a target recorded in install-state.json whose tree is gone must
// render as a flagged 0/N row rather than vanishing.
func TestInventory_PersistedTargetWithMissingTreeIsZero(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	h.Run(TargetClaude, nil)

	if got := PreviouslyInstalledTargets(h.TargetDir); len(got) == 0 {
		t.Fatalf("fixture broken: claude should be persisted, got %v", got)
	}
	if err := os.RemoveAll(filepath.Join(h.TargetDir, ".claude")); err != nil {
		t.Fatalf("remove .claude: %v", err)
	}

	row := findRow(t, harnessInventory(t, h), TargetClaude)
	if row.Agents.Actual != 0 {
		t.Errorf("agents actual = %d, want 0 (tree deleted)", row.Agents.Actual)
	}
	if row.Agents.Expected == 0 {
		t.Errorf("agents expected must be > 0 to flag the shortfall, got 0")
	}
	if row.Skills.Actual != 0 {
		t.Errorf("skills actual = %d, want 0 (tree deleted)", row.Skills.Actual)
	}
}
