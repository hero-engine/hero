package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/intake"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var intakePromoteType string

var intakeCmd = &cobra.Command{
	Use:   "intake [slug] [inline text...]",
	Short: "Capture a pre-commitment idea or signal",
	Long: `Captures an intake — a pre-commitment idea or inbound signal that lives
in the spec graph (searchable, provenance-linked) but is deliberately kept
out of committed-work rollups (status, queue, velocity, snapshot) until it
is promoted to a roadmap spec.

Usage:
  hero intake "let users export to CSV"   # inline text becomes title and signal
  hero intake csv-export                   # create an empty intake with a slug
  hero intake promote csv-export           # promote to a feature spec (+ provenance)
  hero intake promote a-bug --type bug     # promote to a bug spec
  hero intake reject stale-idea            # terminal: reject
  hero intake list                         # list intakes by status

If no slug is given, one is derived from the inline text or the current time.`,
	RunE: runIntakeCapture,
}

var intakePromoteCmd = &cobra.Command{
	Use:   "promote <slug>",
	Short: "Promote an intake to a roadmap spec, recording provenance",
	Long: `Creates a roadmap spec (feature by default, or bug with --type bug) from
the intake, writes a derived_from relation on the new spec so 'hero why'
traverses back to the intake, and marks the intake promoted.`,
	Args: cobra.ExactArgs(1),
	RunE: runIntakePromote,
}

var intakeRejectCmd = &cobra.Command{
	Use:   "reject <slug>",
	Short: "Reject an intake (terminal)",
	Args:  cobra.ExactArgs(1),
	RunE:  runIntakeReject,
}

var intakeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List intakes grouped by status",
	Args:  cobra.NoArgs,
	RunE:  runIntakeList,
}

func init() {
	intakePromoteCmd.Flags().StringVar(&intakePromoteType, "type", "feature", "roadmap spec type: feature or bug")
	intakeCmd.AddCommand(intakePromoteCmd, intakeRejectCmd, intakeListCmd)
}

// intakeWorkspace loads config and returns the resolved hero dir, erroring
// if no workspace exists. Shared by every intake subcommand.
func intakeWorkspace() (cfg config.Config, projectRoot, heroDir string, err error) {
	projectRoot = findProjectRoot()
	cfg, err = config.Load(projectRoot)
	if err != nil {
		return cfg, "", "", fmt.Errorf("loading config: %w", err)
	}
	heroDir = cfg.HeroDir(projectRoot)
	if _, statErr := os.Stat(heroDir); os.IsNotExist(statErr) {
		return cfg, "", "", fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}
	return cfg, projectRoot, heroDir, nil
}

// resolveIntakeBySlug finds an intake-typed spec by slug. Filtering to
// TypeIntake keeps promote/reject unambiguous even after a promotion
// leaves a roadmap spec sharing the same slug.
func resolveIntakeBySlug(heroDir, slug string) (*spec.Spec, error) {
	return intake.NewService(heroDir).Resolve(slug)
}

func runIntakeCapture(cmd *cobra.Command, args []string) error {
	_, _, heroDir, err := intakeWorkspace()
	if err != nil {
		return err
	}

	now := time.Now()

	// Slug / inline-text derivation mirrors `hero note`.
	var slug, inlineText string
	switch {
	case len(args) == 0:
		slug = now.Format("2006-01-02-1504")
	case len(args) == 1 && strings.Contains(args[0], " "):
		inlineText = args[0]
		slug = textToSlug(args[0])
	case len(args) == 1:
		slug = args[0]
	default:
		inlineText = strings.Join(args, " ")
		slug = textToSlug(inlineText)
	}

	if slug == "" || strings.Contains(slug, "/") {
		return fmt.Errorf("invalid slug %q — use lowercase-kebab-case without slashes", slug)
	}

	title := slugToTitle(slug)
	if inlineText != "" {
		title = inlineText
		if len(title) > 80 {
			title = title[:77] + "..."
		}
	}

	service := intake.NewService(heroDir)
	specPath, err := service.Capture(intake.CaptureRequest{Slug: slug, Title: title, Body: inlineText})
	if err != nil {
		return err
	}

	fmt.Printf("Created intake: %s\n", specPath)
	return nil
}

// generateIntakeContent scaffolds an intake spec. status:planning is the
// declared initial state in core/spec-types/intake.md.
func generateIntakeContent(title, slug, date, body string) string {
	return intake.GenerateContent(title, slug, date, body, nil, intake.SourceMetadata{})
}

func runIntakePromote(cmd *cobra.Command, args []string) error {
	slug := args[0]
	_, _, heroDir, err := intakeWorkspace()
	if err != nil {
		return err
	}

	newType := strings.ToLower(intakePromoteType)
	result, err := intake.NewService(heroDir).Promote(intake.PromoteRequest{Slug: slug, Type: newType, GenerateTemplate: generateSpecTemplate})
	if err != nil {
		return err
	}
	fmt.Printf("Promoted intake %q → %s spec: %s\n", slug, newType, result.Path)
	return nil
}

func runIntakeReject(cmd *cobra.Command, args []string) error {
	slug := args[0]
	_, _, heroDir, err := intakeWorkspace()
	if err != nil {
		return err
	}

	changed, err := intake.NewService(heroDir).Reject(slug)
	if err != nil {
		return err
	}
	if !changed {
		fmt.Printf("intake %q is already rejected\n", slug)
		return nil
	}

	fmt.Printf("Rejected intake %q\n", slug)
	return nil
}

func runIntakeList(cmd *cobra.Command, args []string) error {
	_, _, heroDir, err := intakeWorkspace()
	if err != nil {
		return err
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	var intakes []*spec.Spec
	for _, s := range specs {
		if s.Type == spec.TypeIntake {
			intakes = append(intakes, s)
		}
	}
	if len(intakes) == 0 {
		fmt.Println("No intakes. Capture one with `hero intake \"<text>\"`.")
		return nil
	}

	fmt.Printf("Intakes (%d):\n", len(intakes))
	for _, s := range intakes {
		fmt.Printf("  %-30s  %-8s  %s\n", s.Slug, string(s.Status), s.Title)
	}
	return nil
}

// injectRelationBlock inserts a single-relation `relations:` block into a
// spec's frontmatter, immediately before the closing `---`. The block
// shape matches parseRelationsBlock (target:/kind: lines).
func injectRelationBlock(content, kind, target string) string {
	return intake.InjectRelation(content, kind, target)
}
