package tracker

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hero-engine/hero/internal/config"
)

// The 6-tier ladder, duplicated locally to avoid a package import cycle
// with internal/sizing. Strings are load-bearing — must match the
// constants in internal/sizing/sizing.go and internal/cli/cost.go.
var sizeTiers = []string{
	"trivial",
	"small",
	"medium",
	"large",
	"x-large",
	"giant",
}

// defaultJiraSizeMapping is the shipped default for Jira workspaces
// that don't override `tracker.size_mapping`. Story-points midpoints
// pick a single canonical value per tier when MapSize writes out.
func defaultJiraSizeMapping() *config.SizeMappingConfig {
	return &config.SizeMappingConfig{
		Field: "story_points",
		Thresholds: map[string][]*float64{
			"trivial": {floatPtr(0), floatPtr(1)},
			"small":   {floatPtr(2), floatPtr(2)},
			"medium":  {floatPtr(3), floatPtr(5)},
			"large":   {floatPtr(8), floatPtr(8)},
			"x-large": {floatPtr(13), floatPtr(13)},
			"giant":   {floatPtr(20), nil},
		},
		ContainerField: "epic_label",
	}
}

// defaultLinearSizeMapping is the shipped default for Linear workspaces.
func defaultLinearSizeMapping() *config.SizeMappingConfig {
	return &config.SizeMappingConfig{
		Field: "estimate",
		Thresholds: map[string][]*float64{
			"trivial": {floatPtr(0), floatPtr(1)},
			"small":   {floatPtr(2), floatPtr(2)},
			"medium":  {floatPtr(3), floatPtr(5)},
			"large":   {floatPtr(8), floatPtr(8)},
			"x-large": {floatPtr(13), floatPtr(13)},
			"giant":   {floatPtr(20), nil},
		},
	}
}

// defaultGitHubSizeMapping is the shipped default for GitHub. Maps to
// the conventional `size/<tier>` label prefix. GitHub doesn't strongly
// model hierarchy without sub-issues, so SupportsHierarchy is false by
// default for GitHub adapters.
func defaultGitHubSizeMapping() *config.SizeMappingConfig {
	return &config.SizeMappingConfig{
		Field: "size/", // label prefix
		// Thresholds carried for symmetry; numeric bands map to label
		// strings via the tier name itself. We populate the bands so
		// Validate() passes and ReverseMapSize has a list of known
		// tiers to match against.
		Thresholds: map[string][]*float64{
			"trivial": {floatPtr(0), floatPtr(0)},
			"small":   {floatPtr(1), floatPtr(1)},
			"medium":  {floatPtr(2), floatPtr(2)},
			"large":   {floatPtr(3), floatPtr(3)},
			"x-large": {floatPtr(4), floatPtr(4)},
			"giant":   {floatPtr(5), nil},
		},
	}
}

func floatPtr(v float64) *float64 { return &v }

// effectiveMapping picks the configured mapping when present, falling
// back to the named default for the given tracker name. Returns nil
// when neither is available (e.g. unknown tracker with no override).
func effectiveMapping(cfg *config.SizeMappingConfig, trackerName string) *config.SizeMappingConfig {
	if cfg != nil {
		return cfg
	}
	switch trackerName {
	case "jira":
		return defaultJiraSizeMapping()
	case "linear":
		return defaultLinearSizeMapping()
	case "github":
		return defaultGitHubSizeMapping()
	default:
		return nil
	}
}

// mapSizeWith translates a local tier name to the tracker-side value.
// For numeric fields (story_points / estimate) it picks the band's
// lower bound as the canonical write value — predictable and round-
// trips cleanly through ReverseMapSize. For label-style fields (the
// `size/` GitHub prefix) it concatenates prefix + tier.
//
// Returns ("", err) if the tier is unknown to the mapping, leaving the
// caller free to surface the conflict rather than silently writing.
func mapSizeWith(cfg *config.SizeMappingConfig, localTier string) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("no size mapping configured")
	}
	if !isKnownTier(localTier) {
		return "", fmt.Errorf("unknown local tier %q", localTier)
	}
	band, ok := cfg.Thresholds[localTier]
	if !ok || len(band) == 0 || band[0] == nil {
		return "", fmt.Errorf("tier %q is not in size_mapping.thresholds", localTier)
	}
	// Label-prefix fields: tier name is the canonical value, not a number.
	if isLabelPrefixField(cfg.Field) {
		return cfg.Field + localTier, nil
	}
	return strconv.FormatFloat(*band[0], 'f', -1, 64), nil
}

// reverseMapSizeWith translates a tracker-side value back to the local
// tier. For label-style fields the string match is exact (with prefix
// stripping). For numeric fields the value is matched against each
// tier band; the first band whose [min, max] contains the value wins
// (a nil max means "unbounded above").
func reverseMapSizeWith(cfg *config.SizeMappingConfig, trackerValue string) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("no size mapping configured")
	}
	trackerValue = strings.TrimSpace(trackerValue)
	if trackerValue == "" {
		return "", fmt.Errorf("tracker value is empty")
	}
	if isLabelPrefixField(cfg.Field) {
		// Accept either the full label ("size/large") or a bare tier
		// ("large") for flexibility.
		v := strings.TrimPrefix(trackerValue, cfg.Field)
		if isKnownTier(v) {
			return v, nil
		}
		return "", fmt.Errorf("tracker label %q does not match any known tier", trackerValue)
	}
	num, err := strconv.ParseFloat(trackerValue, 64)
	if err != nil {
		return "", fmt.Errorf("tracker value %q is not numeric: %w", trackerValue, err)
	}
	// Walk tiers in ladder order so the lowest-matching tier wins,
	// matching the threshold-config intent (e.g. 0..1 → trivial even
	// though small starts at 2).
	for _, tier := range sizeTiers {
		band, ok := cfg.Thresholds[tier]
		if !ok || len(band) != 2 || band[0] == nil {
			continue
		}
		min := *band[0]
		if num < min {
			continue
		}
		if band[1] == nil { // unbounded above
			return tier, nil
		}
		if num <= *band[1] {
			return tier, nil
		}
	}
	return "", fmt.Errorf("tracker value %q (%.3f) is outside all configured tier bands", trackerValue, num)
}

func isKnownTier(t string) bool {
	for _, s := range sizeTiers {
		if s == t {
			return true
		}
	}
	return false
}

// isLabelPrefixField reports whether the configured Field looks like a
// label-style identifier (ends in "/" — GitHub convention) rather than
// a numeric field. Heuristic but unambiguous for the defaults we ship.
func isLabelPrefixField(field string) bool {
	return strings.HasSuffix(field, "/")
}

// --- Default adapter glue ---
//
// The Tracker interface gains three methods (SupportsHierarchy,
// MapSize, ReverseMapSize). Each concrete adapter implements them by
// delegating to the package-level helpers above with its effective
// mapping (configured ↦ shipped default).
//
// Jira / Linear: SupportsHierarchy() = true.
// GitHub: SupportsHierarchy() = false (sub-issues is opt-in by org
// and not currently autodetected).

// SupportsHierarchy reports whether the tracker natively models
// parent/child relationships (epics, projects, sub-issues, …).
func (j *jira) SupportsHierarchy() bool { return true }

// SupportsHierarchy for Linear: projects/cycles model hierarchy.
func (l *linear) SupportsHierarchy() bool { return true }

// SupportsHierarchy for GitHub: conservative false — basic issues lack
// hierarchy. Teams using GitHub sub-issues can override via a future
// config flag; for now Hero treats GitHub as flat.
func (g *gitHub) SupportsHierarchy() bool { return false }

func (j *jira) MapSize(localTier string) (string, error) {
	return mapSizeWith(j.sizeMapping(), localTier)
}

func (j *jira) ReverseMapSize(trackerValue string) (string, error) {
	return reverseMapSizeWith(j.sizeMapping(), trackerValue)
}

func (l *linear) MapSize(localTier string) (string, error) {
	return mapSizeWith(l.sizeMapping(), localTier)
}

func (l *linear) ReverseMapSize(trackerValue string) (string, error) {
	return reverseMapSizeWith(l.sizeMapping(), trackerValue)
}

func (g *gitHub) MapSize(localTier string) (string, error) {
	return mapSizeWith(g.sizeMapping(), localTier)
}

func (g *gitHub) ReverseMapSize(trackerValue string) (string, error) {
	return reverseMapSizeWith(g.sizeMapping(), trackerValue)
}

// sizeMapping accessors. These return the per-adapter configured
// mapping when set (added in slice 5 — see config.SizeMappingConfig),
// or the shipped default. They are isolated on each adapter rather
// than threaded through the constructors so the existing factories
// stay backward-compatible.

func (j *jira) sizeMapping() *config.SizeMappingConfig {
	if j.configuredSizeMapping != nil {
		return j.configuredSizeMapping
	}
	return defaultJiraSizeMapping()
}

func (l *linear) sizeMapping() *config.SizeMappingConfig {
	if l.configuredSizeMapping != nil {
		return l.configuredSizeMapping
	}
	return defaultLinearSizeMapping()
}

func (g *gitHub) sizeMapping() *config.SizeMappingConfig {
	if g.configuredSizeMapping != nil {
		return g.configuredSizeMapping
	}
	return defaultGitHubSizeMapping()
}

// ExtractTrackerSize reads the tracker-side size value out of an Issue
// using the tracker's effective size mapping. Returns ("", "") when no
// size value is present on the issue (a normal state — most issues
// have no story points set) or when the tracker has no mapping
// (unknown tracker type / no defaults).
//
// The second return is the tier name produced by ReverseMapSize when
// the raw value parses cleanly; empty means the raw value is present
// but doesn't match any known tier (caller should warn rather than
// guess).
//
// Used by hero sync pull / hero sync push to seed or compare local
// `size:` against what the tracker carries.
func ExtractTrackerSize(t Tracker, issue *Issue) (rawValue, mappedTier string) {
	if t == nil || issue == nil {
		return "", ""
	}
	mapping := mappingFromTracker(t)
	if mapping == nil {
		return "", ""
	}

	if isLabelPrefixField(mapping.Field) {
		// GitHub-style: look for a label matching the prefix.
		for _, label := range issue.Labels {
			if strings.HasPrefix(label, mapping.Field) {
				rawValue = label
				if tier, err := reverseMapSizeWith(mapping, label); err == nil {
					mappedTier = tier
				}
				return rawValue, mappedTier
			}
		}
		return "", ""
	}

	// Numeric field: check the CustomFields map. Tracker adapters key
	// CustomFields by lowercase field-name; we accept either the
	// mapping.Field name verbatim or its lowercase form.
	if issue.CustomFields == nil {
		return "", ""
	}
	candidates := []string{mapping.Field, strings.ToLower(mapping.Field)}
	for _, key := range candidates {
		if v, ok := issue.CustomFields[key]; ok && v != "" {
			rawValue = v
			if tier, err := reverseMapSizeWith(mapping, v); err == nil {
				mappedTier = tier
			}
			return rawValue, mappedTier
		}
	}
	return "", ""
}

// mappingFromTracker is the type-assertion shim that fishes the
// effective size mapping back out of a concrete adapter. Exists
// because the public Tracker interface intentionally doesn't expose
// the raw config (callers should use MapSize / ReverseMapSize). The
// extractor needs Field/Thresholds for the label-prefix walk, so we
// allow this one type-assertion seam internally.
func mappingFromTracker(t Tracker) *config.SizeMappingConfig {
	switch tr := t.(type) {
	case *jira:
		return tr.sizeMapping()
	case *linear:
		return tr.sizeMapping()
	case *gitHub:
		return tr.sizeMapping()
	}
	return nil
}

// TypeSupportsHierarchy reports whether the named tracker type
// natively models parent/child relationships, without requiring a
// configured adapter instance (no token needed). Mirrors the per-
// adapter SupportsHierarchy() method. Used by `hero size --check`
// and the spec-sizing skill's capability surface so the nudge regime
// can be reported on workspaces that have no token configured (e.g.
// CI runs, fresh checkouts).
func TypeSupportsHierarchy(trackerType string) bool {
	switch trackerType {
	case "jira", "linear":
		return true
	default:
		return false
	}
}

// SizeSyncAction is the high-level outcome of a size-sync attempt.
type SizeSyncAction string

const (
	// SizeSyncNoop — nothing to do (no mapping, both sides empty, or
	// values already agree).
	SizeSyncNoop SizeSyncAction = "noop"
	// SizeSyncSeedLocal — the tracker has a value, the local spec
	// doesn't, and the pull should write the mapped tier into local
	// frontmatter.
	SizeSyncSeedLocal SizeSyncAction = "seed-local"
	// SizeSyncPushToTracker — local has a value, the tracker doesn't,
	// and the push should write the mapped value back. (The push half
	// is currently advisory — wiring an actual write call is left to
	// the per-tracker push path which already exists for status.)
	SizeSyncPushToTracker SizeSyncAction = "push-to-tracker"
	// SizeSyncConflict — both sides carry values that map to
	// different tiers. Non-destructive: neither side is touched; the
	// caller surfaces a warning so the user resolves manually.
	SizeSyncConflict SizeSyncAction = "conflict"
)

// SizeSyncPlan is the decision the sync layer makes for one spec.
// It is intentionally inert — the caller writes the local frontmatter
// (or makes the tracker call) based on Action. This keeps the helper
// pure and easy to test.
type SizeSyncPlan struct {
	Action       SizeSyncAction
	LocalTier    string // current local `size:` (may be empty)
	TrackerValue string // raw tracker-side value (may be empty)
	TrackerTier  string // tracker value mapped to a local tier, when parseable
	WriteValue   string // for SeedLocal: the local tier to stamp; for
	// PushToTracker: the tracker-side value the mapping would write
	Message string // human-readable summary (used for warnings/logs)
}

// PlanSizePull computes the SizeSyncPlan for the pull direction
// (tracker → local). Returns a noop when no mapping is configured —
// the no-tracker workspace path stays silent.
func PlanSizePull(t Tracker, issue *Issue, localTier string) SizeSyncPlan {
	plan := SizeSyncPlan{LocalTier: localTier, Action: SizeSyncNoop}
	if t == nil || issue == nil {
		return plan
	}
	if mappingFromTracker(t) == nil {
		return plan
	}
	rawValue, mappedTier := ExtractTrackerSize(t, issue)
	plan.TrackerValue = rawValue
	plan.TrackerTier = mappedTier

	switch {
	case rawValue == "":
		// Tracker has no value — nothing to pull.
		return plan
	case localTier == "" && mappedTier != "":
		plan.Action = SizeSyncSeedLocal
		plan.WriteValue = mappedTier
		plan.Message = fmt.Sprintf("seeded local size from tracker: %s → %s", rawValue, mappedTier)
	case localTier == "" && mappedTier == "":
		// Tracker has a value we can't map — surface as a conflict so
		// the user knows their thresholds don't cover it.
		plan.Action = SizeSyncConflict
		plan.Message = fmt.Sprintf("tracker value %q does not map to any configured tier; resolve in hero.json size_mapping.thresholds or set local size manually", rawValue)
	case localTier != "" && mappedTier == "":
		// Tracker carries an unmappable value. Don't touch local;
		// surface so the user knows.
		plan.Action = SizeSyncConflict
		plan.Message = fmt.Sprintf("tracker value %q does not map to any configured tier; local size is %q", rawValue, localTier)
	case localTier == mappedTier:
		// Already in agreement — silent noop.
		return plan
	default:
		plan.Action = SizeSyncConflict
		plan.Message = fmt.Sprintf("size conflict: local=%q  tracker=%q (maps to %q). Run `hero size <slug> <tier>` to set local intentionally, or update tracker.", localTier, rawValue, mappedTier)
	}
	return plan
}

// PlanSizePush computes the SizeSyncPlan for the push direction
// (local → tracker). Returns a noop when no mapping is configured or
// local size is unset. The push itself is the caller's job — this
// helper only decides whether the write is safe.
func PlanSizePush(t Tracker, issue *Issue, localTier string) SizeSyncPlan {
	plan := SizeSyncPlan{LocalTier: localTier, Action: SizeSyncNoop}
	if t == nil {
		return plan
	}
	if mappingFromTracker(t) == nil {
		return plan
	}
	if localTier == "" {
		return plan
	}
	wantValue, err := t.MapSize(localTier)
	if err != nil {
		plan.Action = SizeSyncConflict
		plan.Message = fmt.Sprintf("cannot map local size %q to a tracker value: %v", localTier, err)
		return plan
	}
	plan.WriteValue = wantValue

	var rawValue, mappedTier string
	if issue != nil {
		rawValue, mappedTier = ExtractTrackerSize(t, issue)
	}
	plan.TrackerValue = rawValue
	plan.TrackerTier = mappedTier

	switch {
	case rawValue == "":
		// Tracker has no value — safe to push.
		plan.Action = SizeSyncPushToTracker
		plan.Message = fmt.Sprintf("push local size %q to tracker as %q", localTier, wantValue)
	case mappedTier == localTier:
		// Already in agreement — silent noop, regardless of exact
		// numeric value (a human-set 4 mapped to "small" is fine
		// even if our default canonical write would be 2).
		return plan
	default:
		plan.Action = SizeSyncConflict
		plan.Message = fmt.Sprintf("size conflict: local=%q  tracker=%q (maps to %q). Refusing to overwrite — resolve manually.", localTier, rawValue, mappedTier)
	}
	return plan
}
