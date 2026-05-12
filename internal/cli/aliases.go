package cli

import "github.com/spf13/cobra"

// Top-level aliases for the core-loop verbs that live canonically
// under `hero spec` but are common enough to deserve one-keystroke
// access.
//
// Each alias is a thin Cobra wrapper that delegates to the same
// RunE as the canonical command. Both `hero deliver <slug>` and
// `hero spec deliver <slug>` work and produce identical output.
//
// The slash-commands (/deliver, /design, /diagnose) remain the
// natural-language entry points; these CLI aliases are for the
// "I just want to type it" power-user case and for backward
// compatibility with patterns that pre-date the spec subsystem.

var deliverAliasCmd = &cobra.Command{
	Use:   "deliver [spec-slug]",
	Short: "Start delivery of a spec (alias for `hero spec deliver`)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDeliver,
}

var diagnoseAliasCmd = &cobra.Command{
	Use:   "diagnose [spec-slug]",
	Short: "Investigate a bug, classify root cause, produce a fix spec (alias for `hero spec diagnose`)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDiagnose,
}

// `hero design` doesn't have a direct backend — it's a workflow
// shortcut for "create a new feature spec." We map it to runNew
// (the spec scaffold command). The slash-command `/design` remains
// the richer natural-language path.
var designAliasCmd = &cobra.Command{
	Use:   "design <slug>",
	Short: "Scaffold a new feature spec (alias for `hero spec new`)",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runNew,
}
