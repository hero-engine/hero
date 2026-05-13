package install

import (
	"os"
	"testing"
)

// TestEveryInstalledKindHasContract asserts that for every (target,
// kind) cell where the target actually renders files into the
// destination tree, a HarnessContract is declared in
// internal/install/contracts.go.
//
// Cells where the target installs files but no contract is declared
// yet record a t.Skip naming the responsible child spec of the
// install-upgrade-contract-coverage initiative. This keeps CI green
// while making the gap loud and traceable rather than silently
// tolerated.
//
// When children #2 and #3 land their contracts, the corresponding
// Skip messages will go away and the meta-test will assert real
// coverage on those cells.
func TestEveryInstalledKindHasContract(t *testing.T) {
	cells := []struct {
		target Target
		kind   ContentKind
		// childSpec names the install-upgrade-contract-coverage child
		// responsible for landing this cell's contract. Empty string
		// means "should be declared today; failure here is a real
		// regression".
		childSpec string
	}{
		// claude — covered by install-contract-registry-foundation (this spec).
		{TargetClaude, KindAgents, ""},
		{TargetClaude, KindCommands, ""},
		{TargetClaude, KindSkills, ""},

		// opencode + cursor — child #3 install-contract-coverage-opencode-cursor.
		{TargetOpenCode, KindAgents, "install-contract-coverage-opencode-cursor"},
		{TargetOpenCode, KindCommands, "install-contract-coverage-opencode-cursor"},
		{TargetOpenCode, KindSkills, "install-contract-coverage-opencode-cursor"},
		{TargetCursor, KindAgents, "install-contract-coverage-opencode-cursor"},
		{TargetCursor, KindCommands, "install-contract-coverage-opencode-cursor"},
		{TargetCursor, KindSkills, "install-contract-coverage-opencode-cursor"},

		// codex + copilot + generic — child #2 install-smoke-coverage-codex-copilot-generic.
		{TargetCodex, KindAgents, "install-smoke-coverage-codex-copilot-generic"},
		{TargetCodex, KindCommands, "install-smoke-coverage-codex-copilot-generic"},
		{TargetCodex, KindSkills, "install-smoke-coverage-codex-copilot-generic"},
		{TargetCopilot, KindAgents, "install-smoke-coverage-codex-copilot-generic"},
		{TargetCopilot, KindCommands, "install-smoke-coverage-codex-copilot-generic"},
		{TargetCopilot, KindSkills, "install-smoke-coverage-codex-copilot-generic"},
		{TargetGeneric, KindAgents, "install-smoke-coverage-codex-copilot-generic"},
		{TargetGeneric, KindCommands, "install-smoke-coverage-codex-copilot-generic"},
		{TargetGeneric, KindSkills, "install-smoke-coverage-codex-copilot-generic"},
	}

	for _, cell := range cells {
		name := string(cell.target) + "/" + string(cell.kind)
		t.Run(name, func(t *testing.T) {
			h := newInstallHarness(t)
			h.Run(cell.target, nil)

			dir := h.harnessDirFor(cell.target, cell.kind)
			if !dirHasContent(dir) {
				// Target doesn't actually install this kind.
				return
			}
			if _, ok := ContractsFor(cell.target, cell.kind); ok {
				return // contract declared, all good
			}
			if cell.childSpec != "" {
				t.Skipf("(%s, %s) installs files at %s but no contract is declared yet — landing in %s",
					cell.target, cell.kind, dir, cell.childSpec)
				return
			}
			t.Fatalf("(%s, %s) installs files at %s but no HarnessContract is declared — add one to internal/install/contracts.go",
				cell.target, cell.kind, dir)
		})
	}
}

// dirHasContent returns true if dir exists and contains at least one
// entry. Used by TestEveryInstalledKindHasContract to skip cells the
// target does not actually populate.
func dirHasContent(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	return len(entries) > 0
}
