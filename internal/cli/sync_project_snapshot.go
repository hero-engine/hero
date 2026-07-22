package cli

import (
	"encoding/json"
	"fmt"
	"os"

	projectcontract "github.com/hero-engine/hero/contracts/trackerproject"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var projectSnapshotBoard string

var newProjectSnapshotLoader = func(cfg *config.TrackerConfig, jiraCfg *config.JiraConfig, trackerKnowledgeDir string) (tracker.ProjectSnapshotLoader, error) {
	return tracker.NewProjectSnapshotLoader(cfg, jiraCfg, trackerKnowledgeDir)
}

var syncProjectSnapshotCmd = &cobra.Command{
	Use:   "project-snapshot",
	Short: "Read tracker-native project scheduling truth",
	Long: `Emits a bounded tracker-project-snapshot/v1 document containing the
configured project, selected board, active/future iterations, and lightweight
item membership. Descriptions, comments, changelogs, and attachments are not
loaded. Existing local specs are joined by tracker ID without mutation.`,
	Args: cobra.NoArgs,
	RunE: runSyncProjectSnapshot,
}

func init() {
	syncProjectSnapshotCmd.Flags().StringVar(&projectSnapshotBoard, "board", "", "board ID or name (overrides tracker board setting)")
	syncCmd.AddCommand(syncProjectSnapshotCmd)
}

func runSyncProjectSnapshot(cmd *cobra.Command, _ []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := selectSyncIntegration(&cfg); err != nil {
		return err
	}
	if cfg.Tracker == nil || cfg.Tracker.Type == "" || cfg.Tracker.Type == "none" {
		return fmt.Errorf("no tracker configured")
	}
	loader, err := newProjectSnapshotLoader(cfg.Tracker, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
	if err != nil {
		return fmt.Errorf("initializing project snapshot: %w", err)
	}
	boardRef := effectiveProjectSnapshotBoard(cfg.Tracker.Board, projectSnapshotBoard, cmd.Flags().Changed("board"))
	snapshot, err := loader.LoadProjectSnapshot(boardRef)
	if err != nil {
		return fmt.Errorf("loading project snapshot: %w", err)
	}
	snapshot.Version = projectcontract.Version
	snapshot.ConnectionID = syncIntegration
	joinProjectSnapshotLocalSlugs(snapshot, cfg.HeroDir(projectRoot))

	out, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding project snapshot: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(out))
	return nil
}

func effectiveProjectSnapshotBoard(configured, explicit string, explicitSet bool) string {
	if explicitSet {
		return explicit
	}
	return configured
}

func joinProjectSnapshotLocalSlugs(snapshot *projectcontract.Snapshot, heroDir string) {
	if snapshot == nil {
		return
	}
	discovered, err := spec.Discover(heroDir)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	localByTrackerID := make(map[string]string, len(discovered))
	for _, local := range discovered {
		if local.TrackerID != "" {
			localByTrackerID[local.TrackerID] = local.Slug
		}
	}
	for index := range snapshot.Items {
		snapshot.Items[index].LocalSlug = localByTrackerID[snapshot.Items[index].TrackerID]
	}
}
