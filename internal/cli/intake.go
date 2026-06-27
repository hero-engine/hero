package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
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
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}
	for _, s := range specs {
		if s.Slug == slug && s.Type == spec.TypeIntake {
			return s, nil
		}
	}
	return nil, fmt.Errorf("intake %q not found", slug)
}

func runIntakeCapture(cmd *cobra.Command, args []string) error {
	_, _, heroDir, err := intakeWorkspace()
	if err != nil {
		return err
	}

	now := time.Now()
	date := now.Format("2006-01-02")

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

	targetDir := filepath.Join(heroDir, "planning", "intake", slug)
	specPath := filepath.Join(targetDir, "spec.md")
	if _, statErr := os.Stat(specPath); statErr == nil {
		return fmt.Errorf("intake already exists: %s", specPath)
	}

	title := slugToTitle(slug)
	if inlineText != "" {
		title = inlineText
		if len(title) > 80 {
			title = title[:77] + "..."
		}
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	if err := os.WriteFile(specPath, []byte(generateIntakeContent(title, slug, date, inlineText)), 0o644); err != nil {
		return fmt.Errorf("writing intake: %w", err)
	}

	fmt.Printf("Created intake: %s\n", specPath)
	return nil
}

// generateIntakeContent scaffolds an intake spec. status:planning is the
// declared initial state in core/spec-types/intake.md.
func generateIntakeContent(title, slug, date, body string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: %s\n", title))
	sb.WriteString(fmt.Sprintf("slug: %s\n", slug))
	sb.WriteString("type: intake\n")
	sb.WriteString("status: planning\n")
	sb.WriteString(fmt.Sprintf("created: %s\n", date))
	sb.WriteString("tags: []\n")
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("# %s\n\n## Signal\n\n", title))
	if body != "" {
		sb.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("<!-- What is the idea or inbound signal? Capture it in its own words. -->\n")
	}
	sb.WriteString("\n## Notes\n\n<!-- Triage thoughts, links, who asked. Promote with `hero intake promote`. -->\n")
	return sb.String()
}

func runIntakePromote(cmd *cobra.Command, args []string) error {
	slug := args[0]
	_, _, heroDir, err := intakeWorkspace()
	if err != nil {
		return err
	}

	in, err := resolveIntakeBySlug(heroDir, slug)
	if err != nil {
		return err
	}
	if in.Status == spec.StatusPromoted {
		return fmt.Errorf("intake %q is already promoted", slug)
	}

	newType := strings.ToLower(intakePromoteType)
	if newType != "feature" && newType != "bug" {
		return fmt.Errorf("--type must be feature or bug, got %q", newType)
	}

	newDir, err := specTargetDir(heroDir, newType, slug)
	if err != nil {
		return err
	}
	newPath := filepath.Join(newDir, "spec.md")
	if _, statErr := os.Stat(newPath); statErr == nil {
		return fmt.Errorf("%s spec already exists: %s", newType, newPath)
	}

	// Create the roadmap spec with a derived_from relation back to the
	// intake — the edge `hero why` walks for provenance.
	content := injectRelationBlock(generateSpecTemplate(slug, newType), "derived_from", slug)
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	if err := spec.AtomicWriteFile(newPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s spec: %w", newType, err)
	}

	// Mark the intake promoted and record where it went.
	data, err := os.ReadFile(in.Path)
	if err != nil {
		return fmt.Errorf("reading intake: %w", err)
	}
	updated := spec.SetFrontmatterField(string(data), "status", string(spec.StatusPromoted))
	updated = spec.SetFrontmatterField(updated, "promoted_to", slug)
	if err := spec.AtomicWriteFile(in.Path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("updating intake: %w", err)
	}

	fmt.Printf("Promoted intake %q → %s spec: %s\n", slug, newType, newPath)
	return nil
}

func runIntakeReject(cmd *cobra.Command, args []string) error {
	slug := args[0]
	_, _, heroDir, err := intakeWorkspace()
	if err != nil {
		return err
	}

	in, err := resolveIntakeBySlug(heroDir, slug)
	if err != nil {
		return err
	}
	if in.Status == spec.StatusRejected {
		fmt.Printf("intake %q is already rejected\n", slug)
		return nil
	}

	data, err := os.ReadFile(in.Path)
	if err != nil {
		return fmt.Errorf("reading intake: %w", err)
	}
	updated := spec.SetFrontmatterField(string(data), "status", string(spec.StatusRejected))
	if err := spec.AtomicWriteFile(in.Path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("updating intake: %w", err)
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
	block := fmt.Sprintf("relations:\n  - target: %s\n    kind: %s", target, kind)
	lines := strings.Split(content, "\n")
	fences := 0
	for i, l := range lines {
		if strings.TrimSpace(l) == "---" {
			fences++
			if fences == 2 {
				out := make([]string, 0, len(lines)+1)
				out = append(out, lines[:i]...)
				out = append(out, block)
				out = append(out, lines[i:]...)
				return strings.Join(out, "\n")
			}
		}
	}
	return content
}
