package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/sizing"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

// sizeLadder is the canonical 6-tier ladder shared between the
// declared `size:` frontmatter field (internal/spec/spec.go) and
// the computed effort buckets (internal/cli/cost.go). Kept here as
// a small, local copy so the CLI command can format error messages
// without reaching into the spec package's unexported state.
var sizeLadder = []string{
	effortTrivial,
	effortSmall,
	effortMedium,
	effortLarge,
	effortXLarge,
	effortGiant,
}

var (
	sizeCheck   bool
	sizeAck     string
	sizeSummary bool
)

var sizeCmd = &cobra.Command{
	Use:          "size [spec-slug] [tier]",
	Short:        "Get, set, ack, or check the declared `size:` of specs",
	SilenceUsage: true,
	// Silence cobra's auto-error print so non-zero exits don't print
	// the error twice. cmd/hero/main.go already prints err to stderr
	// before os.Exit(1); cobra's "Error: <err>" prefix is the duplicate.
	// Spec size-drift-actionable-output Option C: silence locally on
	// this command, leave main.go and other commands untouched.
	SilenceErrors: true,
	Long: `Manage the declared ` + "`size:`" + ` frontmatter field on specs.

The size ladder (shared across feature / bug / enhancement / epic /
initiative specs):

    trivial / small / medium / large / x-large / giant

Magnitudes are roughly: trivial = hours, small = ~1 day, medium = a
few days, large = a week+, x-large = weeks, giant = month+. Promote
one type up at the giant tier.

Modes:

  hero size <slug>            print the current declared size ("(unset)"
                              if absent); exits 0 either way.
  hero size <slug> <tier>     set the declared size; preserves other
                              frontmatter, formatting, and comments.
  hero size --ack <tier> <slug>
                              stamp ` + "`size_ack: <tier>`" + ` on the spec
                              (one-shot acknowledgement that suppresses
                              the design-time nudge). Today only ` + "`giant`" + `
                              is consumed by the nudge regime, but any
                              ladder tier is accepted.
  hero size --check           scan all specs for declared-vs-computed
                              drift (leaves only). Exits non-zero on any
                              drift found, suitable for CI.
`,
	Args: cobra.MaximumNArgs(2),
	RunE: runSize,
}

func init() {
	sizeCmd.Flags().BoolVar(&sizeCheck, "check", false, "scan all specs for declared-vs-computed drift; exit non-zero if any found")
	sizeCmd.Flags().StringVar(&sizeAck, "ack", "", "stamp `size_ack: <tier>` on <slug> (acknowledges an oversized spec; suppresses the design-time nudge)")
	sizeCmd.Flags().BoolVar(&sizeSummary, "summary", false, "with --check, emit only the workspace-wide AmbientDrift count + hint (one line) instead of per-spec rows; intended for delivery-lead pre-flight surfaces")
}

func runSize(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	if sizeCheck && sizeAck != "" {
		return fmt.Errorf("--check and --ack are mutually exclusive")
	}
	if sizeSummary && !sizeCheck {
		return fmt.Errorf("--summary requires --check")
	}
	if sizeCheck {
		if len(args) > 0 {
			return fmt.Errorf("--check takes no positional arguments")
		}
		return runSizeCheck(heroDir, &cfg)
	}
	if sizeAck != "" {
		if len(args) != 1 {
			return fmt.Errorf("usage: hero size --ack <tier> <slug>")
		}
		return runSizeAck(heroDir, args[0], sizeAck)
	}

	switch len(args) {
	case 1:
		return runSizeGet(heroDir, args[0])
	case 2:
		return runSizeSet(heroDir, args[0], args[1])
	default:
		return fmt.Errorf("usage: hero size <slug> [tier] | hero size --ack <tier> <slug> | hero size --check")
	}
}

// runSizeGet prints the declared size for the given slug. Unset is
// rendered as "(unset)" and is a normal (non-error) state.
func runSizeGet(heroDir, slug string) error {
	target, err := loadSpecBySlug(heroDir, slug)
	if err != nil {
		return err
	}
	if target.Size == "" {
		fmt.Println("(unset)")
		return nil
	}
	fmt.Println(target.Size)
	return nil
}

// runSizeSet validates the tier and writes the frontmatter field
// non-destructively via spec.SetFrontmatterField.
func runSizeSet(heroDir, slug, tier string) error {
	tier = spec.NormalizeSize(tier)
	if err := validateSizeTier(tier); err != nil {
		return err
	}
	target, err := loadSpecBySlug(heroDir, slug)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(target.Path)
	if err != nil {
		return fmt.Errorf("reading spec: %w", err)
	}
	updated := spec.SetFrontmatterField(string(data), "size", tier)
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("writing spec: %w", err)
	}

	prev := target.Size
	if prev == "" {
		prev = "(unset)"
	}
	fmt.Printf("size: %s → %s  (%s)\n", prev, tier, slug)
	return nil
}

// runSizeAck stamps `size_ack: <tier>` on the spec's frontmatter.
// Used to acknowledge an oversized spec (today only `giant` is
// consumed by the nudge regime; the ladder is accepted for forward
// compatibility). Idempotent — re-stamping the same tier is a no-op
// from the user's perspective.
func runSizeAck(heroDir, slug, tier string) error {
	tier = spec.NormalizeSize(tier)
	if err := validateSizeTier(tier); err != nil {
		return err
	}
	target, err := loadSpecBySlug(heroDir, slug)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(target.Path)
	if err != nil {
		return fmt.Errorf("reading spec: %w", err)
	}
	updated := spec.SetFrontmatterField(string(data), "size_ack", tier)
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("writing spec: %w", err)
	}

	fmt.Printf("size_ack: %s  (%s)\n", tier, slug)
	return nil
}

// runSizeCheck walks all specs and reports both leaf drift
// (declared vs computed bucket on feature/bug/enhancement specs) and
// container drift (declared vs aggregated child rollup on
// initiatives). Output rows are prefixed with the drift kind so the
// two flavors are visually distinct. Exits non-zero when any drift
// is found so CI can gate on it.
func runSizeCheck(heroDir string, cfg *config.Config) error {
	// --summary path: emit only the AmbientDrift count + hint as a
	// single line, suitable for delivery-lead pre-flight surfaces.
	// Quiet → emit nothing and exit 0. Non-empty → emit the hint and
	// exit 0 (no CI gate on the ambient summary; that's --check's job).
	if sizeSummary {
		projectRoot := findProjectRoot()
		ambientOpts := sizing.AmbientDriftOpts{}
		if cfg != nil && cfg.Roadmap != nil {
			ambientOpts.RecencyDays = cfg.Roadmap.AmbientRecencyDaysOrDefault()
			ambientOpts.StopNaggingHours = cfg.Roadmap.StopNaggingHoursOrDefault()
		}
		rep := sizing.AmbientDrift(heroDir, projectRoot, ambientOpts)
		if !rep.Quiet && rep.Count > 0 {
			fmt.Println(rep.Hint)
		}
		return nil
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	// Print the tracker-capability header so the operator (and the
	// spec-sizing skill, when run by an agent) knows which nudge
	// regime applies before they read drift rows. One line, no
	// special casing — keeps the output cheap on no-tracker
	// workspaces.
	printTrackerCapability(cfg)

	leafDrifts, containerDrifts := sizing.CollectDrift(specs)

	if len(leafDrifts) == 0 && len(containerDrifts) == 0 {
		fmt.Println("No size drift detected.")
		return nil
	}

	sort.Slice(leafDrifts, func(i, j int) bool { return leafDrifts[i].Slug < leafDrifts[j].Slug })
	sort.Slice(containerDrifts, func(i, j int) bool { return containerDrifts[i].Slug < containerDrifts[j].Slug })

	for _, d := range leafDrifts {
		fmt.Printf("[leaf]      %s  declared: %s  computed: %s\n",
			d.Slug, d.Declared, d.Bucket)
		kind := sizing.ClassifyLeafDriftKind(d.Declared, d.Bucket)
		primary, alternative := sizing.SuggestedAction(d.Slug, d.Declared, d.Bucket, kind)
		fmt.Printf("  → %s  or  %s\n", primary, alternative)
	}
	for _, d := range containerDrifts {
		declared := d.Declared
		if declared == "" {
			declared = "(unset)"
		}
		if d.Indeterminate {
			fmt.Printf("[container] %s  declared: %s  rollup: (indeterminate, %d child(ren) missing size)\n",
				d.Slug, declared, d.ChildCount)
			// No actionable suggestion fits "we don't know the rollup."
			continue
		}
		fmt.Printf("[container] %s  declared: %s  rollup: %s  (%d child(ren))\n",
			d.Slug, declared, d.Rollup, d.ChildCount)
		kind := sizing.DriftKindContainerLow
		if d.Declared == "" {
			kind = sizing.DriftKindContainerUnset
		}
		primary, alternative := sizing.SuggestedAction(d.Slug, d.Declared, d.Rollup, kind)
		fmt.Printf("  → %s  or  %s\n", primary, alternative)
	}

	total := len(leafDrifts) + len(containerDrifts)
	fmt.Printf("\n%d spec(s) with size drift  (%d leaf, %d container).\n",
		total, len(leafDrifts), len(containerDrifts))
	fmt.Println("Run '/roadmap-review' to triage interactively.")
	return fmt.Errorf("size drift found in %d spec(s)", total)
}

// printTrackerCapability emits the one-line tracker-regime header at
// the top of `hero size --check` output. Mirrors the projection
// returned by sizing.TrackerCapability so the agent-side surface and
// CLI surface read identically.
func printTrackerCapability(cfg *config.Config) {
	cap := WorkspaceTrackerCapability(cfg)
	if !cap.Configured {
		fmt.Printf("tracker: none — nudge regime: %s\n", cap.NudgeRegime())
		return
	}
	fmt.Printf("tracker: %s (supports_hierarchy: %t) — nudge regime: %s\n",
		cap.Type, cap.SupportsHierarchy, cap.NudgeRegime())
}

// WorkspaceTrackerCapability projects the configured tracker into the
// sizing.TrackerCapability shape. Exported so callers outside this
// file (and tests) can reuse the projection without re-implementing
// the "tracker.type != none" check.
func WorkspaceTrackerCapability(cfg *config.Config) sizing.TrackerCapability {
	cap := sizing.TrackerCapability{}
	if cfg == nil || cfg.Tracker == nil || cfg.Tracker.Type == "" || cfg.Tracker.Type == "none" {
		return cap
	}
	cap.Configured = true
	cap.Type = cfg.Tracker.Type
	cap.SupportsHierarchy = tracker.TypeSupportsHierarchy(cfg.Tracker.Type)
	return cap
}

// reportSizeDriftSummary is the helper consumed by `hero check`. It
// runs the same collector as `hero size --check` but returns only the
// counts — callers print their own rate-limited summary lines rather
// than dumping per-spec rows into the health output. Errors during
// discovery are non-fatal: the caller skips the rows.
func reportSizeDriftSummary(heroDir string) (leafCount, containerCount int) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return 0, 0
	}
	leaf, container := sizing.CollectDrift(specs)
	return len(leaf), len(container)
}

// validateSizeTier mirrors the spec package's validateSize but for
// the CLI surface. It rejects empty (the caller must distinguish
// get-vs-set), unlike the spec loader which treats empty as unset.
func validateSizeTier(tier string) error {
	for _, t := range sizeLadder {
		if t == tier {
			return nil
		}
	}
	return fmt.Errorf("invalid size %q: must be one of %v", tier, sizeLadder)
}

// loadSpecBySlug returns the spec matching the given slug, or a
// not-found error. Loaded once per call — fine for an interactive
// CLI surface.
func loadSpecBySlug(heroDir, slug string) (*spec.Spec, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}
	for _, s := range specs {
		if s.Slug == slug {
			return s, nil
		}
	}
	return nil, fmt.Errorf("spec %q not found", slug)
}
