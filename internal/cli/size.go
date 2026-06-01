package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
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
	sizeCheck bool
)

var sizeCmd = &cobra.Command{
	Use:           "size [spec-slug] [tier]",
	Short:         "Get, set, or check the declared `size:` of specs",
	SilenceUsage:  true,
	SilenceErrors: false,
	Long: `Manage the declared ` + "`size:`" + ` frontmatter field on specs.

The size ladder (shared across feature / bug / enhancement / epic /
initiative specs):

    trivial / small / medium / large / x-large / giant

Magnitudes are roughly: trivial = hours, small = ~1 day, medium = a
few days, large = a week+, x-large = weeks, giant = month+. Promote
one type up at the giant tier.

Modes:

  hero size <slug>           print the current declared size ("(unset)"
                             if absent); exits 0 either way.
  hero size <slug> <tier>    set the declared size; preserves other
                             frontmatter, formatting, and comments.
  hero size --check          scan all specs for declared-vs-computed
                             drift (leaves only). Exits non-zero on any
                             drift found, suitable for CI.
`,
	Args: cobra.MaximumNArgs(2),
	RunE: runSize,
}

func init() {
	sizeCmd.Flags().BoolVar(&sizeCheck, "check", false, "scan all specs for declared-vs-computed drift; exit non-zero if any found")
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

	if sizeCheck {
		return runSizeCheck(heroDir)
	}

	switch len(args) {
	case 1:
		return runSizeGet(heroDir, args[0])
	case 2:
		return runSizeSet(heroDir, args[0], args[1])
	default:
		return fmt.Errorf("usage: hero size <slug> [tier] | hero size --check")
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

// runSizeCheck walks all specs and reports leaf drift. Container
// drift (declared vs aggregated child rollup) is intentionally not
// implemented here — it ships in slice 3 alongside the rollup work.
// Exits non-zero when any drift is found so CI can gate on it.
func runSizeCheck(heroDir string) error {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	calibration := calibrate(specs)

	type drifted struct {
		slug     string
		declared string
		computed string
	}
	var drifts []drifted

	for _, s := range specs {
		// Only check specs that carry a declared size.
		if s.Size == "" {
			continue
		}
		// Skip container types — their drift signal is declared
		// vs aggregated child rollup, deferred to slice 3. Epic is
		// referenced in the spec but not yet a registered Type; if
		// it lands later, extend this guard.
		if s.Type == spec.TypeInitiative {
			continue
		}
		est := estimateSpec(s, calibration)
		if est.Drift {
			drifts = append(drifts, drifted{
				slug:     s.Slug,
				declared: s.Size,
				computed: est.Bucket,
			})
		}
	}

	if len(drifts) == 0 {
		fmt.Println("No size drift detected.")
		return nil
	}

	sort.Slice(drifts, func(i, j int) bool { return drifts[i].slug < drifts[j].slug })
	for _, d := range drifts {
		fmt.Printf("%s  declared: %s  computed: %s\n", d.slug, d.declared, d.computed)
	}
	fmt.Printf("\n%d spec(s) with size drift.\n", len(drifts))
	return fmt.Errorf("size drift found in %d spec(s)", len(drifts))
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
