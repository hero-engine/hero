package data

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// Default column cap and per-page size for the expanded ?all=1 view.
const (
	defaultColumnCap = 10
	pageSize         = 50
)

// RoadmapInputs is the per-request input bundle for the Horizons
// centerpiece.
type RoadmapInputs struct {
	ProjectRoot string
	HeroDir     string
	// Filters / pagination — populated from the request query string.
	TypeFilter string // "all" | "feature" | "bug" | "initiative" | ""
	AgeFilter  string // "all" | "active-7d" | ""
	ShowAll    bool   // ?all=1
	Page       int    // 1-indexed; defaults to 1
}

// LoadRoadmap composes the Horizons centerpiece by walking the specs
// index and grouping by `horizon`. Initiatives nest their child specs
// inside the parent card's mini list; children that appear as a child
// of ANY initiative are deduped from the top-level card list so they
// don't render twice. Honors the type/age filter row and the ?all=1 +
// page=N query params per spec. Never returns nil — empty columns are
// fine.
func LoadRoadmap(in RoadmapInputs) Roadmap {
	typeF := normalizeTypeFilter(in.TypeFilter)
	ageF := normalizeAgeFilter(in.AgeFilter)
	page := in.Page
	if page < 1 {
		page = 1
	}

	rm := Roadmap{
		Now:     RoadmapColumn{Label: "Now", Pulse: true},
		Next:    RoadmapColumn{Label: "Next"},
		Later:   RoadmapColumn{Label: "Later"},
		Filters: RoadmapFilters{Type: typeF, Age: ageF},
		ShowAll: in.ShowAll,
		Page:    page,
	}

	specs := loadSpecsBest(in.HeroDir)
	if len(specs) == 0 {
		return rm
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
				if rel.Kind == "parent" {
					childrenOf[rel.Target] = append(childrenOf[rel.Target], s)
				} else {
					childrenOf[s.Slug] = append(childrenOf[s.Slug], bySlug[rel.Target])
				}
			}
		}
	}
	for k, kids := range childrenOf {
		filtered := kids[:0]
		for _, c := range kids {
			if c != nil {
				filtered = append(filtered, c)
			}
		}
		childrenOf[k] = filtered
	}

	// Build the dedupe set: any spec slug that appears as a child of an
	// initiative (regardless of horizon) is dropped from the top-level
	// card list. It renders only as a child row inside the parent's
	// initiative card. Spec calls this out as "initiative-child dedupe."
	nested := map[string]bool{}
	for _, s := range specs {
		if s.Type != spec.TypeInitiative {
			continue
		}
		for _, c := range childrenOf[s.Slug] {
			if c != nil {
				nested[c.Slug] = true
			}
		}
	}

	var initiativeSlugs []string
	for _, s := range specs {
		if s.Type == spec.TypeInitiative && len(childrenOf[s.Slug]) > 0 {
			initiativeSlugs = append(initiativeSlugs, s.Slug)
		}
	}
	sort.Strings(initiativeSlugs)

	// Render initiative cards into their horizon columns.
	for _, parentSlug := range initiativeSlugs {
		p := bySlug[parentSlug]
		if p == nil {
			continue
		}
		if !passesFilter(p, typeF, ageF) {
			continue
		}
		col := columnFor(horizonOf(p))
		card := initiativeCard(p, childrenOf[parentSlug])
		appendToRoadmap(&rm, col, card)
	}

	// Regular specs (non-initiative, non-nested).
	for _, s := range specs {
		if s.Type == spec.TypeInitiative {
			continue
		}
		if nested[s.Slug] {
			continue
		}
		switch s.Type {
		case spec.TypeContext, spec.TypeNote, spec.TypeExternal,
			spec.TypeConvention, spec.TypeRule, spec.TypeTripwire,
			spec.TypeExplainer:
			continue
		}
		if !passesFilter(s, typeF, ageF) {
			continue
		}
		col := columnFor(horizonOf(s))
		card := standardCard(s)
		appendToRoadmap(&rm, col, card)
	}

	// Sort each column by LastTouched desc (most recently touched first),
	// then cap or paginate per the query params.
	sortByLastTouchedDesc(&rm.Now)
	sortByLastTouchedDesc(&rm.Next)
	sortByLastTouchedDesc(&rm.Later)

	rm.Now.Count = len(rm.Now.Cards)
	rm.Next.Count = len(rm.Next.Cards)
	rm.Later.Count = len(rm.Later.Cards)

	applyCapOrPaginate(&rm.Now, in, typeF, ageF, "now")
	applyCapOrPaginate(&rm.Next, in, typeF, ageF, "next")
	applyCapOrPaginate(&rm.Later, in, typeF, ageF, "later")

	// Compute blocked-count for the view toolbar badge while we have
	// the specs loaded — unfiltered (the badge always shows total).
	for _, s := range specs {
		if isBlocked(s) {
			rm.BlockedCount++
		}
	}

	return rm
}

// normalizeTypeFilter maps an unknown / empty filter value to "all".
func normalizeTypeFilter(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "feature":
		return "feature"
	case "bug":
		return "bug"
	case "initiative":
		return "initiative"
	}
	return "all"
}

// normalizeAgeFilter maps an unknown / empty filter value to "all".
func normalizeAgeFilter(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "active-7d", "7d", "active":
		return "active-7d"
	}
	return "all"
}

// passesFilter returns true when a spec passes both the type and age
// filters. Empty / "all" filters pass everything.
func passesFilter(s *spec.Spec, typeF, ageF string) bool {
	if s == nil {
		return false
	}
	if typeF != "all" {
		if typeKey(s) != typeF {
			return false
		}
	}
	if ageF == "active-7d" {
		if s.ModifiedAt.IsZero() || time.Since(s.ModifiedAt) > 7*24*time.Hour {
			return false
		}
	}
	return true
}

// sortByLastTouchedDesc orders a column's cards by LastTouched descending,
// then by slug for stable tie-breaking.
func sortByLastTouchedDesc(col *RoadmapColumn) {
	sort.SliceStable(col.Cards, func(i, j int) bool {
		ti, tj := col.Cards[i].LastTouched, col.Cards[j].LastTouched
		if ti.Equal(tj) {
			return col.Cards[i].Slug < col.Cards[j].Slug
		}
		return ti.After(tj)
	})
}

// applyCapOrPaginate trims the column to the default cap (when
// !ShowAll) or slices it into a pageSize page (when ShowAll). Sets the
// Capped / ShowAllHref / PageInfo fields accordingly.
func applyCapOrPaginate(col *RoadmapColumn, in RoadmapInputs, typeF, ageF string, _ string) {
	total := len(col.Cards)
	if !in.ShowAll {
		if total > defaultColumnCap {
			col.Cards = col.Cards[:defaultColumnCap]
			col.Capped = true
			col.ShowAllHref = buildHref(typeF, ageF, true, 1)
		}
		return
	}

	// ?all=1 — paginate at pageSize per column.
	page := in.Page
	if page < 1 {
		page = 1
	}
	pages := 1
	if total > 0 {
		pages = (total + pageSize - 1) / pageSize
	}
	if page > pages {
		page = pages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	if start < total {
		col.Cards = col.Cards[start:end]
	}
	col.PageInfo = &ColumnPage{Page: page, Pages: pages}
	if page > 1 {
		col.PageInfo.PrevHref = buildHref(typeF, ageF, true, page-1)
	}
	if page < pages {
		col.PageInfo.NextHref = buildHref(typeF, ageF, true, page+1)
	}
}

// buildHref constructs a /work URL preserving the active filter +
// pagination state.
func buildHref(typeF, ageF string, all bool, page int) string {
	q := url.Values{}
	if typeF != "" && typeF != "all" {
		q.Set("type", typeF)
	}
	if ageF != "" && ageF != "all" {
		q.Set("age", ageF)
	}
	if all {
		q.Set("all", "1")
	}
	if page > 1 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	if len(q) == 0 {
		return "/work"
	}
	return "/work?" + q.Encode()
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
		LastTouched: s.ModifiedAt,
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
		LastTouched:  s.ModifiedAt,
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
