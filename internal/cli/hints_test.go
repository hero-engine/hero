package cli

import (
	"strings"
	"testing"
)

// TestCliHintsResolveAgainstRootCmd walks every registered hint against
// the cobra command tree to ensure the command path and any --flags
// still exist. This is the drift-prevention test: if a subcommand is
// renamed or a flag moved, the hint that names the old form fails
// loudly here rather than rotting in production output.
func TestCliHintsResolveAgainstRootCmd(t *testing.T) {
	if len(cliHints) == 0 {
		t.Fatal("cliHints registry is empty; expected at least one entry")
	}
	for _, h := range cliHints {
		cmd, _, err := rootCmd.Traverse(h.Args)
		if err != nil {
			t.Errorf("hint %q (id=%s, args=%v) fails to traverse: %v", h.Hint, h.ID, h.Args, err)
			continue
		}
		// Traverse resolves the deepest subcommand but does not validate
		// trailing flags on the leaf. Verify each --flag exists on the
		// resolved command or any inherited flag set.
		for _, a := range h.Args {
			if !strings.HasPrefix(a, "--") {
				continue
			}
			flagName := strings.TrimPrefix(a, "--")
			// Strip =value form (e.g. --files=foo).
			if eq := strings.IndexByte(flagName, '='); eq >= 0 {
				flagName = flagName[:eq]
			}
			if cmd.Flags().Lookup(flagName) == nil && cmd.InheritedFlags().Lookup(flagName) == nil {
				t.Errorf("hint %q (id=%s): flag --%s not found on %s", h.Hint, h.ID, flagName, cmd.CommandPath())
			}
		}
	}
}

// TestCliHintByIDPanicsOnUnknown ensures the lookup helper fails loudly
// when a call site asks for an ID that doesn't exist in the registry —
// safer than returning an empty string that would silently produce
// "Run `` <paths>" output.
func TestCliHintByIDPanicsOnUnknown(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on unknown hint id, got none")
		}
	}()
	_ = cliHintByID("definitely-not-a-real-hint-id")
}
