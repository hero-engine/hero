package cli

import (
	"github.com/hero-engine/hero/internal/cli/prompt"
	"github.com/spf13/cobra"
)

// promptableArgs relaxes a positional-argument rule by exactly one condition:
// too FEW arguments, at a terminal, where the handler can ask for the rest.
//
// # Why the gate lives here and not in RunE
//
// cobra rejects a short invocation during argument validation, which runs
// BEFORE the root PersistentPreRun. Moving the decision into RunE would keep
// the error text identical but change the bytes around it: a scripted
// `hero admin repos add` with no arguments would start emitting whatever
// PersistentPreRun writes to stderr (peer_id migration notices, version
// mismatch warnings) ahead of the same failure. Rejecting here keeps every
// non-TTY invocation byte-identical, which is the additive guarantee.
//
// It is also the whole TTY gate for the commands that use it. There is
// deliberately no second IsInputTTY check inside their prompt helpers: a
// second gate that can never fire is a check no test can falsify, and this one
// is per-command — swapping any single command's Args wiring back to a plain
// cobra rule turns that command's own non-TTY test red and nothing else.
//
// Only a shortfall is relaxed. Too MANY arguments stays an error even at a
// terminal: no prompt can repair an extra argument, and quietly ignoring it is
// how a typo becomes a wrong write.
func promptableArgs(need int, strict cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		err := strict(cmd, args)
		if err == nil {
			return nil
		}
		if len(args) < need && prompt.IsInputTTY(cmd.InOrStdin()) {
			return nil
		}
		return err
	}
}
