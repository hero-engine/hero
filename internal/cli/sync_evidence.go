package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var evidenceNoAttachments bool

var syncEvidenceCmd = &cobra.Command{
	Use:   "evidence <spec-slug>",
	Short: "Fetch the complete tracker ticket for diagnosis",
	Long: `Fetches the full tracker issue through Hero's configured credential and
emits a structured JSON evidence envelope. Jira comments are paginated to
exhaustion and attachments are downloaded to .hero/cache/tracker-evidence so
agents can inspect screenshots without receiving tracker credentials.`,
	Args: cobra.ExactArgs(1),
	RunE: runSyncEvidence,
}

func init() {
	syncEvidenceCmd.Flags().BoolVar(&evidenceNoAttachments, "no-attachments", false, "return attachment metadata without downloading files")
	syncCmd.AddCommand(syncEvidenceCmd)
}

func runSyncEvidence(_ *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := selectSyncIntegration(&cfg); err != nil {
		return err
	}
	heroDir := cfg.HeroDir(projectRoot)
	s, err := resolveSpecBySlug(heroDir, args[0])
	if err != nil {
		return err
	}
	if s.TrackerID == "" {
		return fmt.Errorf("spec %q has no tracker_id", s.Slug)
	}
	t, err := tracker.NewWithJiraConfig(cfg.Tracker, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
	if err != nil {
		return fmt.Errorf("initializing tracker: %w", err)
	}
	provider, ok := t.(tracker.EvidenceTracker)
	if !ok {
		return fmt.Errorf("%s does not support full-ticket evidence yet", t.Name())
	}
	evidence, err := provider.GetIssueEvidence(s.TrackerID)
	if err != nil {
		return err
	}
	if !evidenceNoAttachments && len(evidence.Attachments) > 0 {
		if err := downloadEvidenceAttachments(heroDir, s.Slug, provider, evidence); err != nil {
			return err
		}
	}
	out, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func downloadEvidenceAttachments(heroDir, slug string, provider tracker.EvidenceTracker, evidence *tracker.IssueEvidence) error {
	dir := filepath.Join(heroDir, "cache", "tracker-evidence", slug, "attachments")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating evidence directory: %w", err)
	}
	for i := range evidence.Attachments {
		a := &evidence.Attachments[i]
		if a.Content == "" {
			continue
		}
		data, err := provider.DownloadEvidenceAttachment(a.Content)
		if err != nil {
			evidence.Omissions = append(evidence.Omissions, fmt.Sprintf("attachment %s: %v", a.Filename, err))
			continue
		}
		name := filepath.Base(strings.ReplaceAll(a.Filename, string(filepath.Separator), "_"))
		path := filepath.Join(dir, a.ID+"-"+name)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return fmt.Errorf("writing evidence attachment: %w", err)
		}
		a.LocalPath = path
	}
	return nil
}
