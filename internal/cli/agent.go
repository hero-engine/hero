package cli

import "github.com/spf13/cobra"

// agentCmd is the umbrella for the async-runtime subsystem: headless
// agent execution, job queue, event-driven automations.
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Async runtime: headless agent execution + automations",
	Long: `Subverbs for running agent work outside the interactive
session — fire-and-forget jobs, scheduled automations, approval
gates for work that needs human oversight before merging.

Subverbs:
  agent run       run agent work headlessly via Claude/OpenAI API
  agent jobs      list / inspect / cancel async jobs
  agent automate  set up event-driven automations
  agent approve   approve a gated job
  agent events    log / inspect cross-session events`,
}

func init() {
	agentCmd.AddCommand(heroRunCmd)
	agentCmd.AddCommand(jobsCmd)
	agentCmd.AddCommand(automationsCmd)
	agentCmd.AddCommand(approveCmd)
	agentCmd.AddCommand(eventCmd)
}
