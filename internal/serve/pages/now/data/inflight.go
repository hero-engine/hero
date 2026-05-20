package data

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
)

// InflightInputs is the per-request input bundle for the in-flight
// strip on the Now page.
type InflightInputs struct {
	ProjectRoot string
	HeroDir     string
	// Aggregate, when non-empty, fans out across projects and tags
	// each row with the project slug.
	Aggregate []ActivityProject
	// Limit caps the row count. Zero leaves the default (8). The
	// strip is meant to fit on one row of cards; deeper inspection
	// lives on the Work page.
	Limit int
}

// Inflight is the strip payload.
type Inflight struct {
	Rows      []InflightRow
	Total     int
	Aggregate bool
}

// InflightRow is one card in the strip.
type InflightRow struct {
	Slug         string
	Title        string
	Status       string
	StatusLabel  string
	Project      string
	LastTouched  string // relative-time chip, e.g. "2h"
	Href         string
}

// LoadInflight returns specs currently in `planning`, `delivering`, or
// `in-review` ordered by last-touched timestamp (newest first). Specs
// without a real status — drafts of conventions/notes/contexts — are
// filtered out.
func LoadInflight(in InflightInputs) Inflight {
	limit := in.Limit
	if limit <= 0 {
		limit = 8
	}
	if in.Aggregate != nil {
		var rows []InflightRow
		for _, p := range in.Aggregate {
			rows = append(rows, inflightRowsFor(p.HeroDir, p.Slug, "/p/"+p.Slug)...)
		}
		sort.SliceStable(rows, func(i, j int) bool {
			return rows[i].LastTouched < rows[j].LastTouched // not perfect but stable
		})
		// Sort properly by timestamp via slug-side data: we kept
		// LastTouched as a pretty string already. Re-derive a better
		// order from the raw modified time below.
		return packInflight(sortInflightByMTime(rows, in), limit, true)
	}
	urlPrefix := ""
	if in.HeroDir != "" {
		urlPrefix = "/p/" + filepath.Base(filepath.Dir(in.HeroDir))
	}
	rows := inflightRowsFor(in.HeroDir, "", urlPrefix)
	return packInflight(rows, limit, false)
}

func inflightRowsFor(heroDir, projectSlug, urlPrefix string) []InflightRow {
	if heroDir == "" {
		return nil
	}
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil
	}
	rows := make([]InflightRow, 0, len(specs))
	for _, s := range specs {
		if s == nil {
			continue
		}
		if !isInflightStatus(s.Status) {
			continue
		}
		// Skip non-work types — conventions, contexts, notes have
		// their own pages and don't belong in this strip.
		switch s.Type {
		case spec.TypeContext, spec.TypeNote, spec.TypeExternal,
			spec.TypeConvention, spec.TypeRule, spec.TypeTripwire:
			continue
		}
		href := "/work/spec/" + s.Slug
		if urlPrefix != "" {
			href = urlPrefix + "/work/spec/" + s.Slug
		}
		title := strings.TrimSpace(s.Title)
		if title == "" {
			title = s.Slug
		}
		rows = append(rows, InflightRow{
			Slug:        s.Slug,
			Title:       title,
			Status:      string(s.Status),
			StatusLabel: statusLabelFor(s.Status),
			Project:     projectSlug,
			LastTouched: prettyAge(s.ModifiedAt),
			Href:        href,
		})
	}
	// Sort newest-touched first by stashing mtime into a parallel
	// slice. The caller may re-sort across aggregate projects.
	sortInflight(rows, specs)
	return rows
}

// sortInflight re-orders rows newest-touched-first using each spec's
// ModifiedAt. The rows and specs slices are correlated: rows[i] does
// NOT correspond to specs[i] one-to-one (we filtered), so we look up
// by slug.
func sortInflight(rows []InflightRow, specs []*spec.Spec) {
	mt := map[string]int64{}
	for _, s := range specs {
		if s == nil {
			continue
		}
		mt[s.Slug] = s.ModifiedAt.Unix()
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return mt[rows[i].Slug] > mt[rows[j].Slug]
	})
}

// sortInflightByMTime is the aggregate-mode resort that walks every
// project to recover each row's ModifiedAt timestamp.
func sortInflightByMTime(rows []InflightRow, in InflightInputs) []InflightRow {
	mt := map[string]int64{}
	for _, p := range in.Aggregate {
		specs, err := spec.Discover(p.HeroDir)
		if err != nil {
			continue
		}
		for _, s := range specs {
			if s == nil {
				continue
			}
			mt[p.Slug+"|"+s.Slug] = s.ModifiedAt.Unix()
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return mt[rows[i].Project+"|"+rows[i].Slug] > mt[rows[j].Project+"|"+rows[j].Slug]
	})
	return rows
}

func packInflight(rows []InflightRow, limit int, agg bool) Inflight {
	total := len(rows)
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return Inflight{Rows: rows, Total: total, Aggregate: agg}
}

func isInflightStatus(st spec.Status) bool {
	switch st {
	case spec.StatusPlanning, spec.StatusDelivering, spec.StatusInReview:
		return true
	}
	return false
}

func statusLabelFor(st spec.Status) string {
	switch st {
	case spec.StatusPlanning:
		return "Planning"
	case spec.StatusDelivering:
		return "Delivering"
	case spec.StatusInReview:
		return "In review"
	}
	return string(st)
}
