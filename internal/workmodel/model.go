// Package workmodel turns the spec corpus into the read contract's work
// items (read-contract-v1): one deterministic interpretation of lanes,
// verify state, progress and revisions shared by hero_work, hero_spec and
// the next-step engine.
package workmodel

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// Lane values.
const (
	LaneDesigned     = "designed"
	LaneReady        = "ready"
	LaneInProgress   = "in_progress"
	LaneRecentlyDone = "recently_done"
	LaneNone         = "none"
)

// Verify states.
const (
	VerifyPassed  = "passed"
	VerifyPartial = "partial"
	VerifyFailed  = "failed"
	VerifyNotRun  = "not_run"
)

// DefaultRecentDays is the recently_done window when the caller passes none.
const DefaultRecentDays = 14

// Item is the read contract's WorkItem.
type Item struct {
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Revision    string    `json:"revision"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Priority    *string   `json:"priority"`
	Severity    *string   `json:"severity"`
	Size        *string   `json:"size"`
	Path        string    `json:"path"`
	Parent      *string   `json:"parent"`
	Progress    *Progress `json:"progress"`
	Lane        string    `json:"lane"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
	CompletedAt *string   `json:"completed_at"`
	Tracker     *Tracker  `json:"tracker"`
	Verify      *Verify   `json:"verify"`
	Next        *NextStep `json:"next"`
}

// Progress counts an initiative's finished declared children.
type Progress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

// Tracker is the item's external issue.
type Tracker struct {
	ID  string  `json:"id"`
	URL *string `json:"url"`
}

// Verify is the item's delivery verification state.
type Verify struct {
	State string  `json:"state"`
	Audit *string `json:"audit"`
}

// NextStep is the item's single primary action (filled by the next-step
// engine).
type NextStep struct {
	Action  string  `json:"action"`
	Label   string  `json:"label"`
	Command string  `json:"command"`
	Phase   Phase   `json:"phase"`
	Enabled bool    `json:"enabled"`
	Reason  *string `json:"reason"`
	Extras  []Extra `json:"extras"`
}

// Phase is the item's lifecycle label and state.
type Phase struct {
	Label string `json:"label"`
	State string `json:"state"`
}

// Extra is a secondary action.
type Extra struct {
	Action  string `json:"action"`
	Label   string `json:"label"`
	Command string `json:"command"`
}

// Options controls a build.
type Options struct {
	Now        time.Time
	RecentDays int
	// Root is the project root; item paths are made relative to it.
	Root string
	// Signers are the identities whose ledger sign-offs count, exactly as
	// `hero spec verify` Gate 1 resolves them (spec.KnownSigners). Nil
	// accepts no sign-off: unresolved sign-offs never count as verified.
	Signers map[string]bool
}

// Corpus indexes specs by slug for relation lookups.
type Corpus struct {
	bySlug   map[string]*spec.Spec
	children map[string][]string // parent slug -> work specs declaring it as parent
}

// NewCorpus indexes specs by slug. Slugs can collide across kinds: a
// promoted Mail intake shares its slug with the spec it became, and a
// knowledge entry (e.g. an explainer) can share one with the feature it
// explains. Work specs win, then other specs, then intakes; within a rank
// the first spec seen keeps the slug (Discover order is stable).
func NewCorpus(specs []*spec.Spec) *Corpus {
	c := &Corpus{bySlug: make(map[string]*spec.Spec, len(specs))}
	for _, s := range specs {
		if s.Slug == "" {
			continue
		}
		if prev, ok := c.bySlug[s.Slug]; ok && slugRank(prev) <= slugRank(s) {
			continue
		}
		c.bySlug[s.Slug] = s
	}
	c.children = map[string][]string{}
	for _, s := range specs {
		if slugRank(s) != 0 || c.bySlug[s.Slug] != s {
			continue
		}
		if p := ParentSlug(s); p != "" {
			c.children[p] = append(c.children[p], s.Slug)
		}
	}
	return c
}

// Children returns an initiative's children: the slugs it declares, then
// work specs that name it as their parent, each once, in that order.
func (c *Corpus) Children(s *spec.Spec) []string {
	seen := map[string]bool{s.Slug: true}
	var out []string
	for _, slug := range append(spec.DeclaredChildren(s), c.children[s.Slug]...) {
		if !seen[slug] {
			seen[slug] = true
			out = append(out, slug)
		}
	}
	return out
}

func slugRank(s *spec.Spec) int {
	switch string(s.Type) {
	case "feature", "bug", "enhancement", "initiative", "epic", "decision":
		return 0
	case string(spec.TypeIntake):
		return 2
	default:
		return 1
	}
}

// Lookup returns the spec with the given slug, or nil.
func (c *Corpus) Lookup(slug string) *spec.Spec { return c.bySlug[slug] }

// Build returns one item per work spec, ordered by path.
func Build(specs []*spec.Spec, opts Options) []Item {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	if opts.RecentDays <= 0 {
		opts.RecentDays = DefaultRecentDays
	}
	corpus := NewCorpus(specs)
	var items []Item
	for _, s := range specs {
		if !IsWorkItem(s, corpus) {
			continue
		}
		items = append(items, buildItem(s, corpus, opts))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return items
}

// BuildOne returns the item for one work spec, or false when s is not one.
func BuildOne(s *spec.Spec, specs []*spec.Spec, opts Options) (Item, bool) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	if opts.RecentDays <= 0 {
		opts.RecentDays = DefaultRecentDays
	}
	corpus := NewCorpus(specs)
	if !IsWorkItem(s, corpus) {
		return Item{}, false
	}
	return buildItem(s, corpus, opts), true
}

// IsWorkItem reports whether s is a work spec per read-contract-v1.
func IsWorkItem(s *spec.Spec, c *Corpus) bool {
	switch string(s.Type) {
	case "feature", "bug", "enhancement", "initiative", "epic":
		return true
	case "decision":
		if p := ParentSlug(s); p != "" {
			if ps := c.Lookup(p); ps != nil && IsContainer(ps) {
				return true
			}
		}
	}
	return false
}

func buildItem(s *spec.Spec, c *Corpus, opts Options) Item {
	it := Item{
		Slug:      s.Slug,
		Title:     s.Title,
		Type:      string(s.Type),
		Status:    string(s.Status),
		Priority:  normalizeLevel(s.Priority),
		Severity:  normalizeLevel(s.Severity),
		Size:      optional(s.Size),
		Path:      relPath(opts.Root, s.Path),
		Lane:      Lane(s, c, opts),
		CreatedAt: formatTime(s.CreatedAt),
		UpdatedAt: formatTime(s.ModifiedAt),
	}
	if p := ParentSlug(s); p != "" {
		it.Parent = &p
	}
	if IsContainer(s) {
		it.Progress = initiativeProgress(s, c)
	}
	if !s.CompletedAt.IsZero() {
		t := formatTime(s.CompletedAt)
		it.CompletedAt = &t
	}
	if s.TrackerID != "" {
		it.Tracker = &Tracker{ID: s.TrackerID}
	}
	it.Verify = VerifyOf(s, opts.Signers)
	// Lane and verify state can change with time alone (recently_done ages
	// out), so they are part of the revision too.
	derived := it.Lane
	if it.Verify != nil {
		derived += ":" + it.Verify.State
	}
	it.Revision = Revision(s, c, derived)
	return it
}

// IsContainer reports whether s groups other work (initiative or epic) and
// is therefore modelled by its children.
func IsContainer(s *spec.Spec) bool {
	return s.Type == spec.TypeInitiative || string(s.Type) == "epic"
}

// ParentSlug returns the spec's declared parent, or "".
func ParentSlug(s *spec.Spec) string {
	for _, r := range s.Relations {
		if r.Kind == "parent" || r.Kind == "child-of" {
			return r.Target
		}
	}
	return ""
}

// Designed reports whether s carries a design per read-contract-v1.
func Designed(s *spec.Spec, c *Corpus) bool {
	has := func(names ...string) bool {
		for _, n := range names {
			if strings.TrimSpace(s.Sections[n]) != "" {
				return true
			}
		}
		return false
	}
	switch string(s.Type) {
	case "bug":
		return has("root cause", "root cause analysis") && has("changes", "fix", "suggested fix approach")
	case string(spec.TypeInitiative), "epic": // IsContainer
		return len(c.Children(s)) > 0
	case "decision":
		return has("decision")
	default:
		return has("changes") && len(s.ParseAcceptanceCriteria()) > 0
	}
}

// UnmetDeps returns the targets of s's depends-on/blocks edges that are not
// finished (a missing target counts as unmet), in declaration order.
func UnmetDeps(s *spec.Spec, c *Corpus) []string {
	var out []string
	seen := map[string]bool{}
	for _, r := range s.Relations {
		if r.Kind != "depends-on" && r.Kind != "depends_on" && r.Kind != "blocks" {
			continue
		}
		if seen[r.Target] {
			continue
		}
		seen[r.Target] = true
		if t := c.Lookup(r.Target); t == nil || !t.IsFinished() {
			out = append(out, r.Target)
		}
	}
	return out
}

var inProgressStatuses = map[spec.Status]bool{
	spec.StatusDelivering:   true,
	spec.StatusInReview:     true,
	spec.StatusRegressed:    true,
	spec.StatusHandedOff:    true,
	spec.StatusAwaitingPeer: true,
	spec.StatusHandedBack:   true,
}

// Lane assigns s exactly one lane by read-contract-v1's first-match rules.
func Lane(s *spec.Spec, c *Corpus, opts Options) string {
	if s.IsFinished() {
		if s.Status == spec.StatusSuperseded {
			return LaneNone
		}
		done := s.CompletedAt
		if done.IsZero() {
			done = s.ModifiedAt
		}
		if !done.IsZero() && opts.Now.Sub(done) <= time.Duration(opts.RecentDays)*24*time.Hour {
			return LaneRecentlyDone
		}
		return LaneNone
	}
	if inProgressStatuses[s.Status] {
		return LaneInProgress
	}
	if IsContainer(s) && initiativeStarted(s, c) {
		return LaneInProgress
	}
	if s.Status == spec.StatusPlanning || s.Status == spec.StatusProposed {
		if !Designed(s, c) {
			return LaneNone
		}
		if len(UnmetDeps(s, c)) > 0 {
			return LaneDesigned
		}
		return LaneReady
	}
	return LaneNone
}

func initiativeStarted(s *spec.Spec, c *Corpus) bool {
	for _, slug := range c.Children(s) {
		if ch := c.Lookup(slug); ch != nil && (ch.IsFinished() || inProgressStatuses[ch.Status]) {
			return true
		}
	}
	return false
}

func initiativeProgress(s *spec.Spec, c *Corpus) *Progress {
	children := c.Children(s)
	p := &Progress{Total: len(children)}
	for _, slug := range children {
		if ch := c.Lookup(slug); ch != nil && ch.IsFinished() {
			p.Done++
		}
	}
	return p
}

// VerifyOf derives verify state from files on disk: the spec's own
// validated audit report, its Completion Ledger, and its status. It is nil
// for types that do not verify (decision).
func VerifyOf(s *spec.Spec, signers map[string]bool) *Verify {
	// Decisions are accepted, and initiatives/epics finish through their
	// children; none of them are audited or verified themselves.
	if s.Type == spec.TypeDecision || IsContainer(s) {
		return nil
	}
	v := &Verify{State: VerifyNotRun}
	audit := spec.FindAuditReport(s)
	// `hero spec verify` rewrites a spec's status after its audit, so a
	// finished spec is always newer than its own report: staleness only
	// means something for unfinished work. A slug mismatch never counts.
	if audit.Found || (audit.Stale && !audit.SlugMismatch && s.IsFinished()) {
		verdict := strings.ToLower(audit.Verdict)
		if verdict == "ship" || verdict == "hold" {
			v.Audit = &verdict
		}
	}
	switch {
	case s.Status == spec.StatusRegressed || (v.Audit != nil && *v.Audit == "hold"):
		v.State = VerifyFailed
	case s.IsFinished():
		if v.Audit != nil && *v.Audit == "ship" && ledgerAllDone(s, signers) {
			v.State = VerifyPassed
		} else {
			v.State = VerifyPartial
		}
	}
	return v
}

func ledgerAllDone(s *spec.Spec, signers map[string]bool) bool {
	ledger := spec.ParseLedger(s)
	// Resolve sign-offs exactly as Gate 1 does: an unknown signer clears
	// SignedOff, so free-text "signers" never pass.
	if signers == nil {
		signers = map[string]bool{}
	}
	ledger.ResolveSigners(signers)
	if !ledger.Found || len(ledger.ACRows) == 0 {
		return false
	}
	// Same acceptance as `hero spec verify` Gate 1: DONE, or a structured
	// sign-off on a SKIPPED/BLOCKED row.
	for _, rows := range [][]spec.LedgerRow{ledger.ACRows, ledger.ChangesRows} {
		for _, r := range rows {
			signedSkip := r.SignedOff && (r.Status == spec.LedgerSkipped || r.Status == spec.LedgerBlocked)
			if r.Status != spec.LedgerDone && !signedSkip {
				return false
			}
		}
	}
	return true
}

func normalizeLevel(v string) *string {
	var out string
	// The mapping read-contract-v1 defines, including common tracker
	// synonyms; anything else is null.
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "p0", "critical", "blocker", "highest":
		out = "critical"
	case "p1", "high", "major":
		out = "high"
	case "p2", "medium", "moderate", "normal":
		out = "medium"
	case "p3", "p4", "low", "lowest", "minor", "trivial":
		out = "low"
	default:
		return nil
	}
	return &out
}

func optional(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return &v
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func relPath(root, path string) string {
	if root != "" {
		if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(path)
}
