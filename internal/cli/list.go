package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/spec"
)

// hasScopeHintAck reports whether the per-machine ack for `scope: X`
// has been recorded in satellites.local.json.
func hasScopeHintAck(heroDir, scope string) bool {
	local, err := install.LoadSatellitesLocal(heroDir)
	if err != nil {
		return false
	}
	return local.HasHintAck("scope-hint:" + scope)
}

// recordScopeHintAck stores a per-machine ack so the hint isn't
// repeated. Errors are non-fatal — worst case we re-show the hint.
func recordScopeHintAck(heroDir, scope string) error {
	local, err := install.LoadSatellitesLocal(heroDir)
	if err != nil {
		return err
	}
	local.AddHintAck("scope-hint:" + scope)
	return install.SaveSatellitesLocal(heroDir, local)
}

var (
	listTypes         []string
	listStatuses      []string
	listHorizons      []string
	listTags          []string
	listReady         bool
	listBlocked       bool
	listPinned        bool
	listMine          string
	listStale         int
	listSortKey       string
	listLimit         int
	listFormat        string
	listSubproject    string
	listDomain        string
	listFocusedDomain string
	listAllDomains    bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Filtered listing of specs across types, statuses, and horizons",
	Long: `List specs in the workspace, filtered and sorted by flag-driven criteria.

Defaults: open (non-completed) specs sorted by recency, table format.

Common patterns:
  hero list                              # all open specs
  hero list --type bug --status planning # bugs awaiting work
  hero list --ready --sort priority      # what's pickup-able, ranked
  hero list --blocked                    # specs waiting on something
  hero list --tag agents --horizon now   # tagged work in the now horizon
  hero list --mine chet-bellows             # specs claimed by you
  hero list --stale 30                   # specs untouched for 30 days
  hero list --format json                # machine-readable

` + "`hero queue`" + ` is the curated front door — equivalent to:
  hero list --ready --sort priority --format kickoff
`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringSliceVar(&listTypes, "type", nil, "filter by spec type (feature, bug, convention, decision, initiative, rule, context)")
	listCmd.Flags().StringSliceVar(&listStatuses, "status", nil, "filter by status (planning, in-review, delivering, completed, ...)")
	listCmd.Flags().StringSliceVar(&listHorizons, "horizon", nil, "filter by horizon (now, next, someday, parking)")
	listCmd.Flags().StringSliceVar(&listTags, "tag", nil, "require all of these tags (repeatable)")
	listCmd.Flags().BoolVar(&listReady, "ready", false, "only specs eligible for `hero queue` (open + deps satisfied)")
	listCmd.Flags().BoolVar(&listBlocked, "blocked", false, "only specs with at least one unmet hard dependency")
	listCmd.Flags().BoolVar(&listPinned, "pinned", false, "only specs with `pinned: true` in frontmatter")
	listCmd.Flags().StringVar(&listMine, "mine", "", "filter to specs claimed by this user (claimed_by frontmatter)")
	listCmd.Flags().IntVar(&listStale, "stale", 0, "only specs untouched for at least N days")
	listCmd.Flags().StringVar(&listSortKey, "sort", string(spec.SortRecency), "sort: recency, status, alpha, priority")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "cap result count (0 = unlimited)")
	listCmd.Flags().StringVar(&listFormat, "format", "table", "output format: table, text, json, kickoff")
	listCmd.Flags().StringVar(&listSubproject, "subproject", "", "filter by subproject scope (e.g. engines/mlx); 'all' disables filtering. Default: active scope when run from a satellite/scoped cwd")
	listCmd.Flags().StringVar(&listDomain, "domain", "", "list one domain instead of the enabled workspace stack (\"*\" = all)")
	listCmd.Flags().StringVar(&listFocusedDomain, "focused-domain", "", "rank an enabled domain first without changing workspace configuration")
	listCmd.Flags().BoolVar(&listAllDomains, "all-domains", false, "list specs from every domain")
}

// refreshIndexBeforeRead syncs the index against disk truth so the
// caller's query reflects current state — even when a spec was
// created or modified outside the indexing path. Errors are logged
// but never fatal: stale results beat aborting the command.
//
// Spec: index-staleness-auto-refresh.
func refreshIndexBeforeRead() {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return
	}
	if _, err := index.RefreshIfStale(heroDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: index refresh failed: %v\n", err)
	}
}

func runList(cmd *cobra.Command, args []string) error {
	refreshIndexBeforeRead()
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	specs, err := loadAllSpecs()
	if err != nil {
		return err
	}

	sel, err := buildSelectorFromListFlags()
	if err != nil {
		return err
	}

	maybePrintScopeHint(cmd.ErrOrStderr(), listSubproject, sel.Filter.Subproject)

	override := listDomain
	if listAllDomains {
		override = "*"
	}
	scope := graph.ResolveDomainFocused(cfg, override, listFocusedDomain)
	visible := make([]*spec.Spec, 0, len(specs))
	for _, candidate := range specs {
		domain := candidate.Domain
		if domain == "" {
			domain = cfg.PrimaryDomain()
		}
		if scope.Match(domain) {
			visible = append(visible, candidate)
		}
	}
	limit := sel.Limit
	sel.Limit = 0
	out := sel.Apply(visible)
	sort.SliceStable(out, func(i, j int) bool {
		left, right := out[i].Domain, out[j].Domain
		if left == "" {
			left = cfg.PrimaryDomain()
		}
		if right == "" {
			right = cfg.PrimaryDomain()
		}
		return scope.Rank(left) < scope.Rank(right)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return renderSpecs(cmd.OutOrStdout(), out, listFormat)
}

// loadAllSpecs reads every spec under the project's hero directory.
// Helper shared with `hero queue`.
func loadAllSpecs() ([]*spec.Spec, error) {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}
	return spec.Discover(heroDir)
}

func buildSelectorFromListFlags() (spec.Selector, error) {
	if listReady && listBlocked {
		return spec.Selector{}, fmt.Errorf("--ready and --blocked are mutually exclusive")
	}

	types := make([]spec.Type, 0, len(listTypes))
	for _, t := range listTypes {
		types = append(types, spec.Type(strings.TrimSpace(t)))
	}
	statuses := make([]spec.Status, 0, len(listStatuses))
	for _, s := range listStatuses {
		statuses = append(statuses, spec.Status(strings.TrimSpace(s)))
	}
	horizons := make([]spec.Horizon, 0, len(listHorizons))
	for _, h := range listHorizons {
		horizons = append(horizons, spec.Horizon(strings.TrimSpace(h)))
	}

	sortKey := spec.Sort(strings.TrimSpace(listSortKey))
	switch sortKey {
	case spec.SortRecency, spec.SortStatus, spec.SortAlpha, spec.SortPriority, "":
	default:
		return spec.Selector{}, fmt.Errorf("unknown --sort %q (recency, status, alpha, priority)", listSortKey)
	}

	subproject := resolveSubprojectFilter(listSubproject)

	return spec.Selector{
		Filter: spec.Filter{
			Types:                types,
			Statuses:             statuses,
			Horizons:             horizons,
			Tags:                 listTags,
			Ready:                listReady,
			Blocked:              listBlocked,
			Pinned:               listPinned,
			MineUser:             listMine,
			StaleDays:            listStale,
			ExcludeClosedDefault: true,
			Subproject:           subproject,
		},
		Sort:  sortKey,
		Limit: listLimit,
	}, nil
}

// resolveSubprojectFilter implements the "default to active scope when
// run from a satellite/scoped cwd" rule. Returns:
//   - "all" if the user passed --subproject all (disable filter)
//   - the user-supplied value when --subproject was passed explicitly
//   - the active cwd scope when neither was supplied and one is active
//   - "" otherwise (no filter)
//
// The scope-defaulting hint ("showing scope: X — pass --subproject all
// for the full workspace") is printed by callers via maybePrintScopeHint
// so it isn't repeated on every flow that resolves the filter.
func resolveSubprojectFilter(flag string) string {
	if flag != "" {
		return flag
	}
	projectRoot := findProjectRoot()
	heroDir := filepath.Join(projectRoot, ".hero")
	scope := resolveActiveScope(projectRoot, heroDir)
	if scope == "" {
		return ""
	}
	return scope
}

// maybePrintScopeHint surfaces a one-line nudge when the active scope
// was applied as a default. The acknowledgment is stored per-machine so
// the hint doesn't repeat. Callers pass the resolved subproject string
// (the same value returned by resolveSubprojectFilter) and the user's
// raw --subproject flag value.
func maybePrintScopeHint(out io.Writer, flagValue, resolved string) {
	if flagValue != "" || resolved == "" || resolved == "all" {
		return
	}
	projectRoot := findProjectRoot()
	heroDir := filepath.Join(projectRoot, ".hero")
	if hasScopeHintAck(heroDir, resolved) {
		return
	}
	fmt.Fprintf(out, "showing scope: %s — pass `--subproject all` for the full workspace\n", resolved)
	_ = recordScopeHintAck(heroDir, resolved)
}

// renderSpecs writes the selected specs in the given format. Shared
// with `hero queue`.
func renderSpecs(w io.Writer, specs []*spec.Spec, format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return renderSpecsJSON(w, specs)
	case "kickoff":
		return renderSpecsKickoff(w, specs)
	case "text":
		return renderSpecText(w, specs)
	case "table", "":
		return renderSpecsTable(w, specs)
	}
	return fmt.Errorf("unknown --format %q (table, text, json, kickoff)", format)
}

func renderSpecsTable(w io.Writer, specs []*spec.Spec) error {
	if len(specs) == 0 {
		fmt.Fprintln(w, "No specs match.")
		return nil
	}
	vocab := activeVocab(loadConfigSilent())
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SLUG\tTYPE\tSTATUS\tHORIZON\tPIN\tTITLE")
	for _, s := range specs {
		pin := ""
		if s.Pinned {
			pin = "★"
		}
		horizon := string(s.EffectiveHorizon())
		typeStr := displayType(vocab, string(s.Type))
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			s.Slug, typeStr, s.Status, horizon, pin, s.Title)
	}
	return tw.Flush()
}

func renderSpecText(w io.Writer, specs []*spec.Spec) error {
	if len(specs) == 0 {
		fmt.Fprintln(w, "No specs match.")
		return nil
	}
	vocab := activeVocab(loadConfigSilent())
	for _, s := range specs {
		pin := ""
		if s.Pinned {
			pin = " ★"
		}
		typeStr := displayType(vocab, string(s.Type))
		fmt.Fprintf(w, "- %s — %s (%s/%s)%s\n", s.Slug, s.Title, typeStr, s.Status, pin)
	}
	return nil
}

func renderSpecsJSON(w io.Writer, specs []*spec.Spec) error {
	type jsonRow struct {
		Slug    string   `json:"slug"`
		Title   string   `json:"title"`
		Type    string   `json:"type"`
		Status  string   `json:"status"`
		Horizon string   `json:"horizon"`
		Created string   `json:"created,omitempty"`
		Tags    []string `json:"tags,omitempty"`
		Pinned  bool     `json:"pinned,omitempty"`
		Kickoff string   `json:"kickoff,omitempty"`
		Path    string   `json:"path,omitempty"`
	}
	rows := make([]jsonRow, len(specs))
	for i, s := range specs {
		created := ""
		if !s.CreatedAt.IsZero() {
			created = s.CreatedAt.Format("2006-01-02")
		}
		rows[i] = jsonRow{
			Slug:    s.Slug,
			Title:   s.Title,
			Type:    string(s.Type),
			Status:  string(s.Status),
			Horizon: string(s.EffectiveHorizon()),
			Created: created,
			Tags:    s.Tags,
			Pinned:  s.Pinned,
			Kickoff: s.Kickoff(),
			Path:    s.Path,
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

func renderSpecsKickoff(w io.Writer, specs []*spec.Spec) error {
	if len(specs) == 0 {
		fmt.Fprintln(w, "No specs match.")
		return nil
	}
	vocab := activeVocab(loadConfigSilent())
	for i, s := range specs {
		if i > 0 {
			fmt.Fprintln(w, "\n---")
			fmt.Fprintln(w)
		}
		pin := ""
		if s.Pinned {
			pin = " ★"
		}
		fmt.Fprintf(w, "## %s — %s%s\n", s.Slug, s.Title, pin)
		fmt.Fprintf(w, "_%s · %s · horizon: %s_\n\n", displayType(vocab, string(s.Type)), s.Status, s.EffectiveHorizon())

		// Initiatives surface their `## Goal` run-opener (arm with /drive),
		// not a per-session Kickoff. A Goal opens the loop over the whole
		// initiative where a Kickoff opens one session on one spec.
		if s.Type == spec.TypeInitiative {
			body := strings.TrimSpace(s.GoalSection())
			if body == "" {
				fmt.Fprintf(w, "_(no `## Goal` run opener — hand-edit %s)_\n", s.Path)
				continue
			}
			fmt.Fprintf(w, "_Run opener — arm with `/drive %s`_\n\n", s.Slug)
			fmt.Fprintln(w, body)
			continue
		}

		body := strings.TrimSpace(s.Kickoff())
		if body == "" {
			fmt.Fprintf(w, "_(no `## Kickoff` section — run `/design` or hand-edit %s)_\n", s.Path)
			continue
		}
		fmt.Fprintln(w, body)
	}
	return nil
}
