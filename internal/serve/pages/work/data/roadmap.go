package data

import (
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
)

// RoadmapInputs is the per-request input bundle for the Horizons
// centerpiece.
type RoadmapInputs struct {
	ProjectRoot string
	HeroDir     string
}

// LoadRoadmap composes the Horizons centerpiece by walking the specs
// index and grouping by `horizon`. Initiatives nest their child specs
// inside the parent card's mini list; children with the same horizon
// as their parent do not also render standalone. Never returns nil —
// empty columns are fine.
func LoadRoadmap(in RoadmapInputs) Roadmap {
	specs := loadSpecsBest(in.HeroDir)
	if len(specs) == 0 {
		return Roadmap{
			Now:   RoadmapColumn{Label: "Now", Pulse: true},
			Next:  RoadmapColumn{Label: "Next"},
			Later: RoadmapColumn{Label: "Later"},
		}
	}

	// Build parent → children map keyed by parent slug. A child here is
	// any spec with a `parent` relation; the initiative card list
	// borrows from the same map.
	bySlug := make(map[string]*spec.Spec, len(specs))
	for _, s := range specs {
		bySlug[s.Slug] = s
	}
	childrenOf := map[string][]*spec.Spec{}
	for _, s := range specs {
		for _, rel := range s.Relations {
			if rel.Kind == "parent" || rel.Kind == "child" {
				// "parent" relation: this spec belongs to <target> as a
				// child. "child" relation: this spec OWNS <target> as
				// a child. Normalize both directions.
				if rel.Kind == "parent" {
					childrenOf[rel.Target] = append(childrenOf[rel.Target], s)
				} else {
					childrenOf[s.Slug] = append(childrenOf[s.Slug], bySlug[rel.Target])
				}
			}
		}
	}

	// Drop nil entries that snuck in via unresolved targets.
	for k, kids := range childrenOf {
		filtered := kids[:0]
		for _, c := range kids {
			if c != nil {
				filtered = append(filtered, c)
			}
		}
		childrenOf[k] = filtered
	}

	// Track which child slugs are already rendered inside an initiative
	// card so we don't double-render them at the column root.
	nested := map[string]bool{}
	var initiativeSlugs []string
	for _, s := range specs {
		if s.Type == spec.TypeInitiative && len(childrenOf[s.Slug]) > 0 {
			initiativeSlugs = append(initiativeSlugs, s.Slug)
		}
	}
	// Stable order so a fixed workspace produces fixed output.
	sort.Strings(initiativeSlugs)

	rm := Roadmap{
		Now:   RoadmapColumn{Label: "Now", Pulse: true},
		Next:  RoadmapColumn{Label: "Next"},
		Later: RoadmapColumn{Label: "Later"},
	}

	for _, parentSlug := range initiativeSlugs {
		p := bySlug[parentSlug]
		if p == nil {
			continue
		}
		col := columnFor(horizonOf(p))
		card := initiativeCard(p, childrenOf[parentSlug])
		// Only nest children that share the parent's horizon.
		parentH := horizonOf(p)
		for _, c := range childrenOf[parentSlug] {
			if horizonOf(c) == parentH {
				nested[c.Slug] = true
			}
		}
		appendToRoadmap(&rm, col, card)
	}

	// Regular specs (non-initiative, non-nested).
	sortedSpecs := append([]*spec.Spec(nil), specs...)
	sort.SliceStable(sortedSpecs, func(i, j int) bool {
		return sortedSpecs[i].Slug < sortedSpecs[j].Slug
	})
	for _, s := range sortedSpecs {
		if s.Type == spec.TypeInitiative {
			continue
		}
		if nested[s.Slug] {
			continue
		}
		// Skip context / note / external — they don't belong on the
		// roadmap. Conventions / rules / tripwires also stay off.
		switch s.Type {
		case spec.TypeContext, spec.TypeNote, spec.TypeExternal,
			spec.TypeConvention, spec.TypeRule, spec.TypeTripwire:
			continue
		}
		col := columnFor(horizonOf(s))
		card := standardCard(s)
		appendToRoadmap(&rm, col, card)
	}

	rm.Now.Count = len(rm.Now.Cards)
	rm.Next.Count = len(rm.Next.Cards)
	rm.Later.Count = len(rm.Later.Cards)

	// Compute blocked-count for the view toolbar badge while we have
	// the specs loaded.
	for _, s := range specs {
		if isBlocked(s) {
			rm.BlockedCount++
		}
	}

	return rm
}

// columnFor returns "now" | "next" | "later" for a spec horizon.
// Empty/unknown → "now" per the spec ("Empty = unset (treated as now in
// default views)"). Quarter-suffixed values group into Later.
func columnFor(h spec.Horizon) string {
	s := strings.ToLower(strings.TrimSpace(string(h)))
	switch s {
	case "", "now":
		return "now"
	case "next":
		return "next"
	case "someday", "parking", "later":
		return "later"
	}
	if strings.HasPrefix(s, "q") {
		return "later"
	}
	return "now"
}

func horizonOf(s *spec.Spec) spec.Horizon {
	if s == nil {
		return spec.HorizonNow
	}
	return s.Horizon
}

func appendToRoadmap(rm *Roadmap, col string, card SpecCard) {
	switch col {
	case "now":
		rm.Now.Cards = append(rm.Now.Cards, card)
	case "next":
		rm.Next.Cards = append(rm.Next.Cards, card)
	default:
		rm.Later.Cards = append(rm.Later.Cards, card)
	}
}

// standardCard converts one regular spec to a roadmap card.
func standardCard(s *spec.Spec) SpecCard {
	statusKey, statusLabel := statusFor(s)
	card := SpecCard{
		Slug:        s.Slug,
		Title:       fallbackTitle(s),
		TypeKey:     typeKey(s),
		TypeLabel:   typeLabel(s),
		StatusKey:   statusKey,
		StatusLabel: statusLabel,
		Owner:       ownerOf(s),
	}
	if s.Status == spec.StatusDelivering || s.Status == spec.StatusInReview {
		// Real coverage/criteria joins are deferred to signals.go in a
		// follow-on; for now omit the bars rather than fake numbers.
		card.Bars = nil
	}
	// Backlog/planning cards without bars get a quiet placeholder.
	if len(card.Bars) == 0 && (s.Status == spec.StatusPlanning || s.Status == "" || s.Status == "backlog") {
		// Leave quiet note empty by default — too noisy to invent one.
	}
	return card
}

// initiativeCard converts an initiative spec + its children.
func initiativeCard(s *spec.Spec, kids []*spec.Spec) SpecCard {
	statusKey, statusLabel := statusFor(s)
	card := SpecCard{
		Slug:         s.Slug,
		Title:        fallbackTitle(s),
		TypeKey:      "initiative",
		TypeLabel:    "Initiative",
		StatusKey:    statusKey,
		StatusLabel:  statusLabel,
		Owner:        ownerOf(s),
		IsInitiative: true,
	}
	for _, c := range kids {
		if c == nil {
			continue
		}
		ck, _ := statusFor(c)
		card.Children = append(card.Children, ChildRow{
			StatusKey: ck,
			Slug:      c.Slug,
			Progress:  "", // AC progress join lands in a follow-on
		})
	}
	// Stable child order for deterministic output.
	sort.SliceStable(card.Children, func(i, j int) bool {
		return card.Children[i].Slug < card.Children[j].Slug
	})
	return card
}

// statusFor returns the (status-key, human-label) pair for a spec.
// Maps spec.Status into the mockup's small set of dot colors.
func statusFor(s *spec.Spec) (string, string) {
	if s == nil {
		return "planning", "planning"
	}
	switch s.Status {
	case spec.StatusDelivering:
		return "delivering", "delivering"
	case spec.StatusInReview:
		return "review", "in-review"
	case spec.StatusCompleted:
		return "done", "completed"
	case spec.StatusRegressed:
		return "blocked", "regressed"
	case spec.StatusHandedOff, spec.StatusAwaitingPeer:
		return "review", "awaiting peer"
	case spec.StatusHandedBack:
		return "review", "handed back"
	case spec.StatusPlanning, "":
		return "planning", "planning"
	}
	return "planning", string(s.Status)
}

func typeKey(s *spec.Spec) string {
	if s == nil {
		return "feature"
	}
	switch s.Type {
	case spec.TypeBug:
		return "bug"
	case spec.TypeInitiative:
		return "initiative"
	case spec.TypeDecision:
		return "decision"
	}
	return "feature"
}

func typeLabel(s *spec.Spec) string {
	switch typeKey(s) {
	case "bug":
		return "Bug"
	case "initiative":
		return "Initiative"
	case "decision":
		return "Decision"
	}
	return "Feature"
}

// ownerOf returns the avatar/name pair. Empty ClaimedBy renders the
// unclaimed placeholder.
func ownerOf(s *spec.Spec) CardOwner {
	name := strings.TrimSpace(s.ClaimedBy)
	if name == "" {
		return CardOwner{Initials: "?", Name: "unclaimed", Unclaimed: true}
	}
	return CardOwner{Initials: initials(name), Name: name}
}

// fallbackTitle prefers Spec.Title but falls back to the slug-as-title
// so the roadmap never shows blank rows.
func fallbackTitle(s *spec.Spec) string {
	if s == nil {
		return ""
	}
	if strings.TrimSpace(s.Title) != "" {
		return s.Title
	}
	return s.Slug
}

// initials returns up to two uppercase letters from a display name.
// "Ben Wheeler" → "BW"; "ben" → "BE".
func initials(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "?"
	}
	fields := strings.Fields(name)
	if len(fields) == 1 {
		s := strings.ToUpper(fields[0])
		if len(s) >= 2 {
			return s[:2]
		}
		return s
	}
	out := []byte{}
	for _, f := range fields {
		if len(f) == 0 {
			continue
		}
		out = append(out, strings.ToUpper(f)[:1]...)
		if len(out) >= 2 {
			break
		}
	}
	return string(out)
}

// isBlocked reports whether a spec should show up in the Blocked
// section. Today the heuristic is: status == regressed OR title /
// status text contains a blocked marker. The richer dependency-walk
// lives in a sibling spec.
func isBlocked(s *spec.Spec) bool {
	if s == nil {
		return false
	}
	if s.Status == spec.StatusRegressed {
		return true
	}
	if strings.Contains(strings.ToLower(string(s.Status)), "block") {
		return true
	}
	return false
}

// loadSpecsBest discovers specs under heroDir, returning a nil slice
// on any failure so the page degrades gracefully to empty columns.
func loadSpecsBest(heroDir string) []*spec.Spec {
	if heroDir == "" {
		return nil
	}
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil
	}
	return specs
}
