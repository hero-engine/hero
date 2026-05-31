package snapshot

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// Format is the choice of renderer output.
type Format string

const (
	FormatMarkdown Format = "markdown"
	FormatJSON     Format = "json"
)

// Render returns the snapshot in the requested format. Markdown is
// the default; JSON is structured for scripts / the MCP shape.
func Render(s *Snapshot, format Format) ([]byte, error) {
	switch format {
	case FormatJSON:
		return renderJSON(s)
	case "", FormatMarkdown:
		return renderMarkdown(s), nil
	}
	return nil, fmt.Errorf("snapshot: unknown format %q", format)
}

func renderMarkdown(s *Snapshot) []byte {
	if s == nil {
		return []byte("# Project Snapshot\n\n_No data._\n")
	}
	var b strings.Builder
	title := s.ProjectName
	if title == "" {
		title = "(project)"
	}
	fmt.Fprintf(&b, "# Project Snapshot — %s\n\n", title)
	if s.Mission != "" {
		fmt.Fprintf(&b, "> %s\n\n", strings.TrimSpace(s.Mission))
	}
	fmt.Fprintf(&b, "_Last refreshed: %s · projected from %d source nodes_\n\n",
		s.GeneratedAt.UTC().Format(time.RFC3339), s.SourceNodes)

	writeSurfacesTable(&b, s)
	writeInitiativesSection(&b, s)
	writeRecentSection(&b, s)
	writeNextSection(&b, s)
	writeRisksSection(&b, s)
	writeHealthSection(&b, s)

	return []byte(b.String())
}

func writeSurfacesTable(b *strings.Builder, s *Snapshot) {
	b.WriteString("## Surfaces\n\n")
	if len(s.Surfaces) == 0 && s.UnassignedCount == 0 {
		b.WriteString("_No surfaces detected. Run `hero snapshot detect --explain` to see why._\n\n")
		return
	}
	if s.HasReleaseSignal {
		b.WriteString("| Surface | Stage | Path(s) | Initial-release | Last touched | Driver spec |\n")
		b.WriteString("|---|---|---|---|---|---|\n")
	} else {
		b.WriteString("| Surface | Stage | Path(s) | Last touched | Driver spec |\n")
		b.WriteString("|---|---|---|---|---|\n")
	}

	// Count assignments per surface so we can show the driver spec.
	bySurface := map[string][]SpecAssignment{}
	for _, a := range s.Assignments {
		bySurface[a.SurfaceID] = append(bySurface[a.SurfaceID], a)
	}

	for _, srf := range s.Surfaces {
		assignments := bySurface[srf.ID]
		stage := string(srf.Stage)
		if srf.StagePinned {
			stage += " (pinned)"
		}
		paths := strings.Join(srf.Paths, ", ")
		if paths == "" {
			paths = "—"
		}
		lastTouched := "—"
		var newest time.Time
		for _, a := range assignments {
			if a.Spec != nil && a.Spec.ModifiedAt.After(newest) {
				newest = a.Spec.ModifiedAt
			}
		}
		if !newest.IsZero() {
			lastTouched = humanDuration(time.Since(newest)) + " ago"
		}
		// Pick driver: in-flight spec preferred; else newest planning.
		driver := pickDriverSpec(assignments)

		if s.HasReleaseSignal {
			rel := renderReleaseCell(assignments)
			fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %s |\n",
				srf.ID, stage, paths, rel, lastTouched, driver)
		} else {
			fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n",
				srf.ID, stage, paths, lastTouched, driver)
		}
	}

	if s.UnassignedCount > 0 {
		if s.HasReleaseSignal {
			fmt.Fprintf(b, "| (unassigned) | — | — | — | — | %d specs without surface |\n",
				s.UnassignedCount)
		} else {
			fmt.Fprintf(b, "| (unassigned) | — | — | — | %d specs without surface |\n",
				s.UnassignedCount)
		}
		b.WriteString("\n_Run `hero snapshot assign` to bucket unassigned specs._\n")
	}
	b.WriteString("\n")
	if !s.HasReleaseSignal {
		b.WriteString("> **No release model declared.** Add `release_target:` to a spec or initiative, or configure tracker integration, to enable the initial-release rollup column.\n\n")
	}
}

func pickDriverSpec(as []SpecAssignment) string {
	for _, a := range as {
		if a.Spec != nil && a.Spec.Status == spec.StatusDelivering {
			return a.Spec.Slug
		}
	}
	for _, a := range as {
		if a.Spec != nil && a.Spec.Status == spec.StatusInReview {
			return a.Spec.Slug
		}
	}
	for _, a := range as {
		if a.Spec != nil && a.Spec.Status == spec.StatusPlanning {
			return a.Spec.Slug
		}
	}
	return "—"
}

func renderReleaseCell(as []SpecAssignment) string {
	total := 0
	done := 0
	for _, a := range as {
		if a.ReleaseTarget == "" {
			continue
		}
		total++
		if a.Spec.Status == spec.StatusCompleted {
			done++
		}
	}
	if total == 0 {
		return "—"
	}
	pct := done * 100 / total
	return fmt.Sprintf("%d/%d (%d%%)", done, total, pct)
}

func writeInitiativesSection(b *strings.Builder, s *Snapshot) {
	b.WriteString("## Active initiatives\n\n")
	active := []Initiative{}
	completed := []Initiative{}
	for _, init := range s.Initiatives {
		if init.Status == spec.StatusCompleted {
			completed = append(completed, init)
		} else {
			active = append(active, init)
		}
	}
	if len(active) == 0 && len(completed) == 0 {
		b.WriteString("_None._\n\n")
		return
	}
	if len(active) == 0 {
		b.WriteString("_None._\n\n")
	}
	for _, init := range active {
		surfaces := "—"
		if len(init.Surfaces) > 0 {
			surfaces = strings.Join(init.Surfaces, ", ")
		}
		inFlightNote := ""
		if len(init.InFlight) > 0 {
			inFlightNote = "; in flight: " + strings.Join(init.InFlight, ", ")
		}
		fmt.Fprintf(b, "- **%s** (surface: %s) — %d/%d specs done%s\n",
			init.Title, surfaces, init.Done, init.Total, inFlightNote)
	}
	// Show last 3 completed initiatives below active ones, for context.
	if len(completed) > 0 {
		// Sort by completion (newest first).
		sort.Slice(completed, func(i, j int) bool {
			return completed[i].CompletedAt.After(completed[j].CompletedAt)
		})
		max := 3
		if len(completed) < max {
			max = len(completed)
		}
		b.WriteString("\n### Recently completed initiatives\n\n")
		for _, init := range completed[:max] {
			surfaces := "—"
			if len(init.Surfaces) > 0 {
				surfaces = strings.Join(init.Surfaces, ", ")
			}
			fmt.Fprintf(b, "- **%s** (surface: %s) — %d/%d specs done · COMPLETED",
				init.Title, surfaces, init.Done, init.Total)
			if !init.CompletedAt.IsZero() {
				fmt.Fprintf(b, " %s", init.CompletedAt.UTC().Format("2006-01-02"))
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
}

func writeRecentSection(b *strings.Builder, s *Snapshot) {
	b.WriteString("## Recently completed (last 14 days)\n\n")
	if len(s.RecentlyDone) == 0 {
		b.WriteString("_Nothing recent._\n\n")
		return
	}
	// Group by surface, then alpha-sort surfaces.
	bySurf := map[string][]RecentItem{}
	for _, r := range s.RecentlyDone {
		bySurf[r.SurfaceID] = append(bySurf[r.SurfaceID], r)
	}
	keys := make([]string, 0, len(bySurf))
	for k := range bySurf {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		items := bySurf[k]
		titles := []string{}
		for _, it := range items {
			titles = append(titles, it.Slug)
		}
		fmt.Fprintf(b, "- **%s** — %s\n", k, strings.Join(titles, ", "))
	}
	b.WriteString("\n")
}

func writeNextSection(b *strings.Builder, s *Snapshot) {
	b.WriteString("## Next up across surfaces\n\n")
	if len(s.NextUp) == 0 {
		b.WriteString("_None._\n\n")
		return
	}
	for i, n := range s.NextUp {
		prio := n.Priority
		if prio == "" {
			prio = "—"
		}
		fmt.Fprintf(b, "%d. **%s** — `%s` (%s, %s)\n",
			i+1, n.SurfaceID, n.Slug, prio, n.Status)
	}
	b.WriteString("\n")
}

func writeRisksSection(b *strings.Builder, s *Snapshot) {
	b.WriteString("## Open risks & blockers\n\n")
	if len(s.Blockers) == 0 && len(s.StaleInFlight) == 0 && len(s.AgedBugs) == 0 && s.UnassignedCount == 0 {
		b.WriteString("_None._\n\n")
		return
	}
	if len(s.Blockers) > 0 {
		parts := []string{}
		for _, bl := range s.Blockers {
			parts = append(parts, fmt.Sprintf("`%s` (waits on %s)", bl.Slug, strings.Join(bl.WaitsOn, ", ")))
		}
		fmt.Fprintf(b, "- **Blocked specs (%d):** %s.\n", len(s.Blockers), strings.Join(parts, "; "))
	}
	if len(s.StaleInFlight) > 0 {
		parts := []string{}
		for _, st := range s.StaleInFlight {
			parts = append(parts, fmt.Sprintf("`%s` (%dd)", st.Slug, st.StaleDays))
		}
		fmt.Fprintf(b, "- **Stale-in-flight (%d):** %s.\n", len(s.StaleInFlight), strings.Join(parts, ", "))
	}
	if len(s.AgedBugs) > 0 {
		parts := []string{}
		for _, ag := range s.AgedBugs {
			parts = append(parts, fmt.Sprintf("`%s` (open %dd)", ag.Slug, ag.AgeDays))
		}
		fmt.Fprintf(b, "- **Aged open bugs (%d):** %s.\n", len(s.AgedBugs), strings.Join(parts, ", "))
	}
	if s.UnassignedCount > 0 {
		fmt.Fprintf(b, "- **Unassigned specs (%d) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.\n", s.UnassignedCount)
	}
	b.WriteString("\n")
}

func writeHealthSection(b *strings.Builder, s *Snapshot) {
	b.WriteString("## Snapshot health\n\n")
	totalSurfaces := s.InferredCount + s.OverrideAppliedCount
	covered := len(s.Assignments) - s.UnassignedCount
	coveragePct := 0
	if len(s.Assignments) > 0 {
		coveragePct = covered * 100 / len(s.Assignments)
	}
	fmt.Fprintf(b, "- Surfaces detected: %d (inferred: %d · overrides applied: %d)\n",
		totalSurfaces, s.InferredCount, s.OverrideAppliedCount)
	fmt.Fprintf(b, "- Specs covered: %d/%d (%d%%)\n",
		covered, len(s.Assignments), coveragePct)
	if !s.OverrideEditedAt.IsZero() {
		fmt.Fprintf(b, "- Last surfaces.yaml edit: %s\n", s.OverrideEditedAt.UTC().Format("2006-01-02"))
	}
	fmt.Fprintf(b, "- Projection generation: %dms · Source nodes: %d\n",
		s.GenerationMillis, s.SourceNodes)
	b.WriteString("\n")
}

// humanDuration formats a duration as "1d", "3h", "12m", or "<1m".
func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return "<1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
