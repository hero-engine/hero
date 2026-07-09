package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/acceptance"
	"github.com/hero-engine/hero/internal/async"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/mission"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var (
	deliverManual bool
	deliverAsync  bool
	deliverBatch  bool
)

var deliverCmd = &cobra.Command{
	Use:   "deliver [spec-slug]",
	Short: "Start delivery of a spec",
	Long: `Marks a spec as in-progress for delivery.

Use --manual to indicate you will implement the spec yourself (hand delivery).
Use --async to launch a background agent that delivers the spec, creates a
branch, and opens a PR when done.
Use --batch --async to deliver all approved specs in the pipeline.

After manual implementation, run 'hero verify <slug>' to check your work.

Examples:
  hero deliver --manual my-feature    # hand delivery
  hero deliver --async my-feature     # background agent delivery
  hero deliver --batch --async        # batch agent delivery of all approved specs
  hero status                         # check async progress`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDeliver,
}

func init() {
	deliverCmd.Flags().BoolVar(&deliverManual, "manual", false, "hand delivery — you implement, AI verifies")
	deliverCmd.Flags().BoolVar(&deliverAsync, "async", false, "background agent delivery — creates branch + PR")
	deliverCmd.Flags().BoolVar(&deliverBatch, "batch", false, "deliver all approved specs (requires --async)")
}

func runDeliver(cmd *cobra.Command, args []string) error {
	if deliverBatch {
		if !deliverAsync {
			return fmt.Errorf("--batch requires --async (batch delivery runs in background)")
		}
		if len(args) > 0 {
			return fmt.Errorf("--batch does not take a spec slug — it delivers all approved specs")
		}
		return runDeliverBatch()
	}

	if !deliverManual && !deliverAsync {
		return fmt.Errorf("specify --manual (you implement) or --async (background agent)\n\nFor interactive agent delivery, use the /deliver slash command in your agent.")
	}

	if len(args) == 0 {
		return fmt.Errorf("specify a spec slug (or use --batch --async for all approved specs)")
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	var target *spec.Spec
	for _, s := range specs {
		if s.Slug == args[0] {
			target = s
			break
		}
	}
	if target == nil {
		return fmt.Errorf("spec %q not found", args[0])
	}

	// An initiative is a parent, not a single deliverable. Delivering one
	// child at a time is `/deliver`; running the whole initiative
	// autonomously is `/drive`. Point the user there rather than flipping an
	// initiative to delivering (which would strand its children).
	if target.Type == spec.TypeInitiative {
		return fmt.Errorf("%q is an initiative, not a single spec — run `/drive %s` (or `hero goal %s`) to execute the whole initiative; `/deliver` works one child at a time", target.Slug, target.Slug, target.Slug)
	}

	// Kickoff is a hard precondition for flipping to delivering. Without
	// a `## Kickoff` section the spec can't be picked up from a fresh
	// session and `hero queue` excludes it — flipping it to delivering
	// just strands context. Skip under `go test` so existing fixtures
	// (and runDeliverBatch's throwaway loops) don't need every spec
	// rewritten. Already-delivering specs bypass the check because the
	// transition already happened.
	if !testing.Testing() && target.Status != "delivering" {
		if isKickoffMissing(target.Kickoff()) {
			return fmt.Errorf("spec %q has no `## Kickoff` section — add one before delivery (see skills/kickoff-prompt/SKILL.md)", target.Slug)
		}
	}

	// WIP soft advisory: when the team already has too many specs in
	// flight, recommend finishing one before starting another. This is
	// a warning to stderr — never a block — so an operator who knows
	// what they're doing can proceed. Suppressed under `go test`.
	if !testing.Testing() {
		printWIPAdvisory(specs, cfg, target.Slug)
	}

	// Validate status
	switch target.Status {
	case spec.StatusCompleted:
		return fmt.Errorf("spec %q is already completed", target.Slug)
	case "delivering":
		if deliverAsync {
			store := async.DefaultStore()
			if existing, _ := store.GetBySlug(target.Slug); existing != nil {
				if existing.Status == async.StatusRunning || existing.Status == async.StatusPending {
					fmt.Printf("Spec %q already has an active async delivery (job %s).\n", target.Slug, existing.ID)
					fmt.Println("Run 'hero status' to check progress.")
					return nil
				}
			}
		}
		if target.DeliveryMethod == "manual" {
			fmt.Printf("Spec %q is already in manual delivery.\n", target.Slug)
			fmt.Printf("Run 'hero verify %s' when you're done implementing.\n", target.Slug)
			return nil
		}
		return fmt.Errorf("spec %q is already being delivered (by agent)", target.Slug)
	}

	if m, _ := mission.LoadFile(heroDir); m != nil {
		if line := mission.Preamble(m); line != "" {
			fmt.Println(line)
			fmt.Println()
		}
	}
	printAcceptanceSuccessBar(heroDir, target.Slug)

	if deliverAsync {
		return runAsyncDeliver(projectRoot, heroDir, target)
	}
	return runManualDeliver(heroDir, target)
}

// isKickoffMissing returns true when the spec's `## Kickoff` body is
// empty or only the imported-spec TODO placeholder. The placeholder is
// written by `hero import` so the model has a concrete spot to fill in;
// treating it as "missing" keeps the gate honest — an untouched stub
// shouldn't pass for a real cold-start prompt.
func isKickoffMissing(kickoff string) bool {
	trimmed := strings.TrimSpace(kickoff)
	if trimmed == "" {
		return true
	}
	return strings.Contains(trimmed, "TODO: write a paste-ready cold-start prompt")
}

// printWIPAdvisory warns to stderr when the workspace already has too
// many specs at status: delivering. Soft advisory — never blocks. The
// threshold comes from cfg.Delivery.WIPWarningThreshold and defaults
// to 5 when unset. The currently-targeted slug is excluded so the
// count reflects "other work already in flight," not the spec the
// operator is about to start.
func printWIPAdvisory(specs []*spec.Spec, cfg config.Config, targetSlug string) {
	threshold := 5
	if cfg.Delivery != nil && cfg.Delivery.WIPWarningThreshold > 0 {
		threshold = cfg.Delivery.WIPWarningThreshold
	}
	var inFlight []*spec.Spec
	for _, s := range specs {
		if s.Status != spec.StatusDelivering {
			continue
		}
		if s.Slug == targetSlug {
			continue
		}
		inFlight = append(inFlight, s)
	}
	if len(inFlight) < threshold {
		return
	}
	fmt.Fprintf(os.Stderr, "WARNING: %d specs already in delivery (threshold %d). Consider finishing one before starting another:\n",
		len(inFlight), threshold)
	limit := 10
	if len(inFlight) < limit {
		limit = len(inFlight)
	}
	for i := 0; i < limit; i++ {
		fmt.Fprintf(os.Stderr, "  - %s\n", inFlight[i].Slug)
	}
	if len(inFlight) > limit {
		fmt.Fprintf(os.Stderr, "  ... and %d more\n", len(inFlight)-limit)
	}
}

// printAcceptanceSuccessBar prints the spec's open ACs to stdout when
// the AC graph has any. Best-effort: a graph error or zero criteria
// silently skip — never blocks a delivery start.
//
// The block is the model's "you are being graded on these N criteria"
// signal — it's the user-visible payoff for the AC graph existing.
func printAcceptanceSuccessBar(heroDir, slug string) {
	store, err := graph.Open(heroDir)
	if err != nil {
		return
	}
	defer store.Close()

	criteria, err := acceptance.ListBySpec(store, slug)
	if err != nil || len(criteria) == 0 {
		return
	}
	openCount := 0
	for _, c := range criteria {
		if c.IsOpen() {
			openCount++
		}
	}
	fmt.Println()
	fmt.Printf("Acceptance criteria — graded on %d/%d open\n",
		openCount, len(criteria))
	for _, c := range criteria {
		fmt.Printf("  %s  %s — %s\n",
			statusGlyph(c.Status), c.ACID, summarize(c.Statement, 100))
	}
	fmt.Println()
}

func statusGlyph(s string) string {
	switch s {
	case "passing":
		return "✅"
	case "failing":
		return "❌"
	case "regressed":
		return "⚠️ "
	case "retired":
		return "⊘ "
	default:
		return "◯ "
	}
}

// summarize trims a multi-line statement to a one-liner ellipsis.
func summarize(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	if i := indexOfRune(cut, ' '); i > max/2 {
		return cut[:i] + "…"
	}
	return cut + "…"
}

func indexOfRune(s string, r rune) int {
	last := -1
	for i, c := range s {
		if c == r {
			last = i
		}
	}
	return last
}

func runDeliverBatch() error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	// Find all approved specs (ready for delivery)
	var approved []*spec.Spec
	for _, s := range specs {
		if !s.IsWorkSpec() {
			continue
		}
		if s.Status == spec.StatusCompleted || s.Status == "delivering" {
			continue
		}
		stage := classifyReadiness(s)
		if stage == "approved" || (s.Type == spec.TypeBug && stage == "diagnosed") {
			approved = append(approved, s)
		}
	}

	if len(approved) == 0 {
		fmt.Println("No approved specs ready for delivery.")
		fmt.Println("Run 'hero pipeline' to see spec stages.")
		return nil
	}

	currentBranch, err := gitCurrentBranch(projectRoot)
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	store := async.DefaultStore()
	batchID := generateJobID()

	for _, s := range approved {
		jobID := generateJobID()
		branch := async.BranchName(s.Slug)

		job := async.Job{
			ID:         jobID,
			Type:       async.JobDeliver,
			Slug:       s.Slug,
			SpecPath:   s.Path,
			Branch:     branch,
			BaseBranch: currentBranch,
			Status:     async.StatusPending,
			BatchID:    batchID,
		}

		if err := store.Add(job); err != nil {
			return fmt.Errorf("creating job for %s: %w", s.Slug, err)
		}

		// Update spec status
		data, err := os.ReadFile(s.Path)
		if err != nil {
			continue
		}
		content := string(data)
		content = spec.SetFrontmatterField(content, "status", "delivering")
		content = spec.SetFrontmatterField(content, "delivery_method", "async")
		os.WriteFile(s.Path, []byte(content), 0o644)
	}

	if _, err := index.Rebuild(heroDir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: re-index failed: %v\n", err)
	}

	// Launch a single background process to work through the batch
	if err := launchBackgroundBatch(projectRoot, batchID); err != nil {
		return fmt.Errorf("starting batch process: %w", err)
	}

	fmt.Printf("Enqueued %d specs for async delivery\n", len(approved))
	fmt.Printf("  Batch: %s\n", batchID)
	fmt.Println()
	for _, s := range approved {
		title := s.Title
		if title == "" {
			title = s.Slug
		}
		fmt.Printf("  - %s (%s)\n", title, s.Type)
	}
	fmt.Println()
	fmt.Println("Run 'hero status' to check progress.")
	return nil
}

func runManualDeliver(heroDir string, target *spec.Spec) error {
	data, err := os.ReadFile(target.Path)
	if err != nil {
		return fmt.Errorf("reading spec: %w", err)
	}

	content := string(data)
	content = spec.SetFrontmatterField(content, "status", "delivering")
	content = spec.SetFrontmatterField(content, "delivery_method", "manual")

	if err := os.WriteFile(target.Path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing spec: %w", err)
	}

	if _, err := index.Rebuild(heroDir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: re-index failed: %v\n", err)
	}

	fmt.Printf("Started manual delivery of %q\n", target.Slug)
	fmt.Printf("  Status: delivering (manual)\n")
	fmt.Printf("  Spec:   %s\n", target.Path)
	fmt.Println()
	fmt.Println("Implement the spec, then run:")
	fmt.Printf("  hero verify %s\n", target.Slug)
	return nil
}

func runAsyncDeliver(projectRoot, heroDir string, target *spec.Spec) error {
	currentBranch, err := gitCurrentBranch(projectRoot)
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	jobID := generateJobID()
	branch := async.BranchName(target.Slug)

	store := async.DefaultStore()
	job := async.Job{
		ID:         jobID,
		Type:       async.JobDeliver,
		Slug:       target.Slug,
		SpecPath:   target.Path,
		Branch:     branch,
		BaseBranch: currentBranch,
		Status:     async.StatusPending,
	}

	if err := store.Add(job); err != nil {
		return fmt.Errorf("creating job: %w", err)
	}

	// Update spec status
	data, err := os.ReadFile(target.Path)
	if err != nil {
		return fmt.Errorf("reading spec: %w", err)
	}
	content := string(data)
	content = spec.SetFrontmatterField(content, "status", "delivering")
	content = spec.SetFrontmatterField(content, "delivery_method", "async")
	if err := os.WriteFile(target.Path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing spec: %w", err)
	}

	if _, err := index.Rebuild(heroDir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: re-index failed: %v\n", err)
	}

	// Launch background process
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding hero binary: %w", err)
	}

	bgCmd := exec.Command(exe, "job-run", "--job", jobID)
	bgCmd.Dir = projectRoot
	bgCmd.Stdout = nil
	bgCmd.Stderr = nil
	bgCmd.Stdin = nil

	if err := bgCmd.Start(); err != nil {
		return fmt.Errorf("starting background process: %w", err)
	}
	bgCmd.Process.Release()

	fmt.Printf("Async delivery started for %q\n", target.Slug)
	fmt.Printf("  Job:    %s\n", jobID)
	fmt.Printf("  Branch: %s\n", branch)
	fmt.Printf("  Spec:   %s\n", target.Path)
	fmt.Println()
	fmt.Println("Run 'hero status' to check progress.")
	return nil
}

func gitCurrentBranch(projectDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out[:len(out)-1]), nil
}

func generateJobID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
