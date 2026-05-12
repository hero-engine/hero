package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/hero-engine/hero/internal/install"
	"github.com/spf13/cobra"
)

var verifyInstallJSON bool

var verifyInstallCmd = &cobra.Command{
	Use:   "verify-install [path]",
	Short: "Audit the install state and report drift, broken symlinks, and inconsistencies",
	Long: `Verify the install state at the given path (default: current directory).

Checks each detected harness target's content directories against the
resolved canonical paths, reports:

  - broken symlinks
  - symlinks that escape the project root (security)
  - regular directories where a symlink to canonical is expected
  - rendered-copy mode files that drift from canonical
  - mixed install modes across targets (info only)

Read-only: makes no filesystem changes. Use 'hero install --migrate'
to recover from reported issues.

Exit codes:
  0  no errors (warnings allowed)
  1  one or more error-severity issues
  2  argument/usage error

The --json flag emits a single structured object on stdout instead
of human-readable text, for consumption by programmatic clients
(e.g. a Hero-native client).
`,
	Args: cobra.MaximumNArgs(1),
	RunE: runVerifyInstall,
}

func init() {
	verifyInstallCmd.Flags().BoolVar(&verifyInstallJSON, "json", false, "emit the verification report as a single JSON object on stdout")
	rootCmd.AddCommand(verifyInstallCmd)
}

func runVerifyInstall(cmd *cobra.Command, args []string) error {
	targetDir := "."
	if len(args) == 1 {
		targetDir = args[0]
	}

	start := time.Now()
	report, err := install.RunVerify(targetDir)

	if verifyInstallJSON {
		if jsonErr := emitJSON(install.VerifyJSONOutput{
			Report:     report,
			DurationMs: time.Since(start).Milliseconds(),
			Error:      install.NewJSONError("verify_failed", err),
		}, nil); jsonErr != nil {
			return jsonErr
		}
		if report != nil && report.HasErrors() {
			os.Exit(1)
		}
		return err
	}

	if err != nil {
		return err
	}

	fmt.Print(report.StringReport())

	if report.HasErrors() {
		os.Exit(1)
	}
	return nil
}
