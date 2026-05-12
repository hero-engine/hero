package cli

import "github.com/spf13/cobra"

// publishCmd is the umbrella for one-way output to external surfaces:
// wikis, GitHub Pages, future Confluence/Notion/etc. Distinct from
// `hero sync` (which is bidirectional).
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish hero state to external surfaces (wiki, pages)",
	Long: `One-way output of hero state to external surfaces.

Subverbs:
  publish wiki     push completed specs to GitHub Wiki / Confluence
  publish pages    publish a living team narrative as a GitHub Pages
                   site (phase 9 — graph-projection-driven)

Compare with ` + "`hero sync`" + `, which is bidirectional and tracker-aware.`,
}

func init() {
	publishCmd.AddCommand(wikiSyncCmd)
}
