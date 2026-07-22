package cli

import (
	"context"
	"encoding/json"
	"fmt"

	evidencecontract "github.com/hero-engine/hero/contracts/trackerevidence"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var (
	evidenceNoAttachments bool
	evidenceStatusOnly    bool
	evidenceForce         bool
)

type evidenceLoadService interface {
	Load(context.Context, evidencecontract.Request) evidencecontract.Status
	ReadSnapshot(evidencecontract.Status) (*tracker.IssueEvidence, error)
}

var newEvidenceLoadService = func(projectRoot string) evidenceLoadService {
	return tracker.NewEvidenceLoader(projectRoot)
}

var syncEvidenceCmd = &cobra.Command{
	Use:   "evidence <spec-slug>",
	Short: "Fetch the complete tracker ticket for diagnosis",
	Long: `Fetches the full tracker issue through Hero's configured credential and writes
a private adjacent snapshot plus a compact tracker-evidence/v1 manifest. Jira
comments are paginated to exhaustion. Repeated explicit loads reuse a snapshot
only when provider identity, native update time, and the whole-snapshot hash
prove it current.`,
	Args: cobra.ExactArgs(1),
	RunE: runSyncEvidence,
}

func init() {
	syncEvidenceCmd.Flags().BoolVar(&evidenceNoAttachments, "no-attachments", false, "return attachment metadata without downloading files")
	syncEvidenceCmd.Flags().BoolVar(&evidenceStatusOnly, "status", false, "emit bounded tracker-evidence/v1 status instead of the private evidence body")
	syncEvidenceCmd.Flags().BoolVar(&evidenceForce, "force", false, "force an explicit full refresh even when the local snapshot is current")
	syncCmd.AddCommand(syncEvidenceCmd)
}

func runSyncEvidence(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	includeAttachments := !evidenceNoAttachments
	service := newEvidenceLoadService(projectRoot)
	status := service.Load(cmd.Context(), evidencecontract.Request{
		SpecSlug:           args[0],
		ConnectionID:       syncIntegration,
		IncludeAttachments: &includeAttachments,
		ForceRefresh:       evidenceForce,
	})
	if evidenceStatusOnly {
		return printEvidenceJSON(status)
	}
	if status.Error != nil || (status.Status != evidencecontract.StateFetched && status.Status != evidencecontract.StateRefreshed && status.Status != evidencecontract.StateCurrent) {
		if status.Error != nil {
			return fmt.Errorf("%s: %s", status.Error.Code, status.Error.Message)
		}
		return fmt.Errorf("evidence snapshot is %s", status.Status)
	}
	evidence, err := service.ReadSnapshot(status)
	if err != nil {
		return fmt.Errorf("reading validated evidence snapshot: %w", err)
	}
	return printEvidenceJSON(evidence)
}

func printEvidenceJSON(value any) error {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
