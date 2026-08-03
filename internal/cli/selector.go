package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/cli/prompt"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/skills"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

// pickerMax is the largest choice list rendered at once. Larger local corpora
// are narrowed by a selector-local substring filter before Choice is called.
const pickerMax = 25

const selectorFilterAttempts = 3

var (
	// ErrSelectorCancelled lets a command distinguish an intentional empty
	// selector response from its normal missing-argument error.
	ErrSelectorCancelled = errors.New("selector cancelled")
	ErrSelectorNoMatch   = errors.New("selector filter did not match a candidate")
)

// selectorArgs relaxes exact command arity only for missing values supplied
// interactively. JSON and non-terminal paths retain Cobra's original error.
type selectorArgs struct {
	need   int
	strict cobra.PositionalArgs
}

func exactSelector(n int) selectorArgs {
	return selectorArgs{need: n, strict: cobra.ExactArgs(n)}
}

var (
	selectorOneArg  = exactSelector(1)
	selectorTwoArgs = exactSelector(2)
)

func (a selectorArgs) rule() cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if jsonModeOn(cmd) || !prompt.IsInputTTY(cmd.InOrStdin()) || len(args) >= a.need {
			return a.strict(cmd, args)
		}
		return nil
	}
}

func (a selectorArgs) missing(cmd *cobra.Command, args []string) error {
	return a.strict(cmd, args)
}

func jsonModeOn(cmd *cobra.Command) bool {
	f := cmd.Flags().Lookup("json")
	return f != nil && f.Value.String() == "true"
}

// pickFromCorpus selects a local candidate without ever rendering more than
// pickerMax options. A large corpus is narrowed by stable, case-insensitive
// substring filtering; exact filters select immediately.
func pickFromCorpus(cmd *cobra.Command, label string, candidates []string, missing error) (string, error) {
	if len(candidates) == 0 {
		return "", missing
	}
	if len(candidates) <= pickerMax {
		return chooseCandidate(cmd, label, candidates)
	}

	for attempt := 0; attempt < selectorFilterAttempts; attempt++ {
		filter, err := prompt.Prompt(cmd.InOrStdin(), cmd.OutOrStdout(), "Filter "+label+": ")
		if err != nil {
			return "", err
		}
		if filter == "" {
			return "", ErrSelectorCancelled
		}
		for _, candidate := range candidates {
			if strings.EqualFold(filter, candidate) {
				return candidate, nil
			}
		}

		matches := filterCandidates(candidates, filter)
		if len(matches) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "No %s candidates match %q. Try again.\n", strings.ToLower(label), filter)
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%d %s candidate(s) match %q.\n", len(matches), strings.ToLower(label), filter)
		if len(matches) <= pickerMax {
			return chooseCandidate(cmd, label, matches)
		}
	}
	return "", ErrSelectorNoMatch
}

func chooseCandidate(cmd *cobra.Command, label string, candidates []string) (string, error) {
	answer, err := prompt.Choice(cmd.InOrStdin(), cmd.OutOrStdout(), label, candidates)
	if err != nil {
		return "", err
	}
	if answer == "" {
		return "", ErrSelectorCancelled
	}
	return answer, nil
}

func filterCandidates(candidates []string, filter string) []string {
	needle := strings.ToLower(filter)
	matches := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate), needle) {
			matches = append(matches, candidate)
		}
	}
	return matches
}

func specSlugCandidates(specs []*spec.Spec) []string {
	sel := spec.Selector{
		Filter: spec.Filter{ExcludeClosedDefault: true, Subproject: resolveSubprojectFilter("")},
		Sort:   spec.SortRecency,
	}
	ranked := sel.Apply(specs)
	slugs := make([]string, 0, len(ranked))
	for _, s := range ranked {
		slugs = append(slugs, s.Slug)
	}
	return slugs
}

func pickSpecSlugFrom(cmd *cobra.Command, specs []*spec.Spec, missing error) (string, error) {
	return pickFromCorpus(cmd, "Spec", specSlugCandidates(specs), missing)
}

func pickSpecSlug(cmd *cobra.Command, missing error) (string, error) {
	specs, err := loadAllSpecs()
	if err != nil {
		return "", err
	}
	return pickSpecSlugFrom(cmd, specs, missing)
}

func handedBackSlugCandidates(specs []*spec.Spec) []string {
	pending := make(map[string]bool, len(specs))
	for _, s := range specs {
		if s.Status == spec.StatusHandedBack {
			pending[s.Slug] = true
		}
	}
	out := make([]string, 0, len(pending))
	for _, slug := range specSlugCandidates(specs) {
		if pending[slug] {
			out = append(out, slug)
		}
	}
	return out
}

func skillNameCandidates(cfg config.Config, projectRoot string) ([]string, error) {
	all, err := skills.Discover(skillsDir(cfg, projectRoot))
	if err != nil {
		return nil, fmt.Errorf("discovering skills: %w", err)
	}
	names := make([]string, 0, len(all))
	for _, s := range all {
		names = append(names, s.Slug)
	}
	return names, nil
}

func skillTarget(cmd *cobra.Command, args []string, cfg config.Config, projectRoot string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	names, err := skillNameCandidates(cfg, projectRoot)
	if err != nil {
		return "", err
	}
	return pickFromCorpus(cmd, "Skill", names, selectorOneArg.missing(cmd, args))
}

func withoutSlug(slugs []string, drop string) []string {
	out := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		if slug != drop {
			out = append(out, slug)
		}
	}
	return out
}

func sortedPeerAliases(cfg config.Config) []string {
	aliases := make([]string, 0, len(cfg.Repos))
	for alias := range cfg.Repos {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}
