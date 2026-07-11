package cli

import "github.com/spf13/cobra"

// adminCmd is the umbrella for multi-user / cloud admin operations:
// team server settings, user management, domain switching, repo
// registration. Distinct from per-developer workflow commands.
var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Multi-user / team-server / cloud administration",
	Long: `Subverbs for managing team-scope resources: the team server,
user accounts, business-unit domains, registered repos.

Subverbs:
  admin team    team server status and management
  admin users   manage team server users
  admin domain  show or switch the active business-unit domain
  admin repos   manage registered repos for cross-repo features`,
}

func init() {
	adminCmd.AddCommand(teamCmd)
	adminCmd.AddCommand(usersCmd)
	adminCmd.AddCommand(domainCmd)
	adminCmd.AddCommand(reposCmd)
	adminCmd.AddCommand(backfillCompletedAtCmd)
	adminCmd.AddCommand(backfillCreatedCmd)
}
