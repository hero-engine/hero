package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestConnectHelpSurfacesExposeTheSameFlags(t *testing.T) {
	for _, cmd := range []*cobra.Command{connectCmd, connectAliasCmd} {
		for _, name := range []string{"project", "role", "token-stdin", "local-only", "no-verify", "json"} {
			if cmd.Flags().Lookup(name) == nil {
				t.Errorf("%s is missing --%s", cmd.CommandPath(), name)
			}
		}
		for _, claim := range []string{"--project", "--role", "--token-stdin", "--local-only", "--no-verify", "--json"} {
			if !strings.Contains(cmd.Long, claim) {
				t.Errorf("%s help omits %s", cmd.CommandPath(), claim)
			}
		}
	}
	if connectCmd.Flags().Lookup("project").Usage != connectAliasCmd.Flags().Lookup("project").Usage {
		t.Error("connect surfaces disagree about --project")
	}
}
