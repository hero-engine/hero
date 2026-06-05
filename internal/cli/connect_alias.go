package cli

import "github.com/spf13/cobra"

// Top-level `hero connect` alias.
//
// The canonical command lives at `hero sync connect` (see sync.go),
// but every help/error string and the slash-command docs refer to
// `hero connect`. This thin shim makes both paths work by sharing
// the same RunE and flag pointers as the canonical command.

var connectAliasCmd = &cobra.Command{
	Use:   "connect [type]",
	Short: "Connect hero to an external tracker or wiki service (alias for `hero sync connect`)",
	Long:  connectCmd.Long,
	RunE:  runConnect,
	Args:  cobra.MaximumNArgs(1),
}

var connectTeamAliasCmd = &cobra.Command{
	Use:   "team <url>",
	Short: connectTeamCmd.Short,
	Long:  connectTeamCmd.Long,
	Args:  cobra.ExactArgs(1),
	RunE:  runConnectTeam,
}

var disconnectTeamAliasCmd = &cobra.Command{
	Use:   "disconnect",
	Short: disconnectTeamCmd.Short,
	RunE:  runDisconnectTeam,
}

func init() {
	connectAliasCmd.Flags().BoolVar(&connectList, "list", false, "list saved connections")
	connectAliasCmd.Flags().StringVar(&connectRemove, "remove", "", "remove connection for the given tracker type")
	connectAliasCmd.Flags().BoolVar(&connectGlobal, "global", false, "save token to global credentials (~/.config/hero/credentials.json) instead of hero.local.json")
	connectAliasCmd.Flags().StringVar(&connectProject, "project", "", "project identifier (used with --remove)")

	connectTeamAliasCmd.Flags().StringVar(&connectTeamToken, "token", "", "auth token (if server uses token auth)")

	connectAliasCmd.AddCommand(connectTeamAliasCmd)
	connectAliasCmd.AddCommand(disconnectTeamAliasCmd)
}
