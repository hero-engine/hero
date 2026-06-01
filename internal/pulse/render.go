package pulse

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RenderText renders a PulseData as human-readable plaintext.
func RenderText(p *PulseData) string {
	var sb strings.Builder

	weekOf := p.Period.From.Format("Jan 2, 2006")
	fmt.Fprintf(&sb, "Sprint Pulse — Week of %s\n\n", weekOf)

	// Done
	fmt.Fprintf(&sb, "Done this sprint (%d):\n", len(p.Done))
	if len(p.Done) == 0 {
		sb.WriteString("  (none)\n")
	} else {
		for _, s := range p.Done {
			fmt.Fprintf(&sb, "  ✓ %s — %s\n", s.Slug, s.Title)
		}
	}
	sb.WriteString("\n")

	// In flight
	fmt.Fprintf(&sb, "In flight (%d):\n", len(p.InFlight))
	if len(p.InFlight) == 0 {
		sb.WriteString("  (none)\n")
	} else {
		for _, s := range p.InFlight {
			fmt.Fprintf(&sb, "  ↻ %s — %s\n", s.Slug, s.Title)
		}
	}
	sb.WriteString("\n")

	// At risk
	fmt.Fprintf(&sb, "At risk (%d):\n", len(p.AtRisk))
	if len(p.AtRisk) == 0 {
		sb.WriteString("  (none)\n")
	} else {
		for _, s := range p.AtRisk {
			if s.DaysStale > 0 {
				fmt.Fprintf(&sb, "  ⚠ %s — %s (stale: %d days)\n", s.Slug, s.Title, s.DaysStale)
			} else {
				fmt.Fprintf(&sb, "  ⚠ %s — %s\n", s.Slug, s.Title)
			}
		}
	}
	sb.WriteString("\n")

	// Ambient size-drift (count + hint only). Rendered before the
	// existing detailed `Drift detected` block so the workspace-wide
	// summary shows first; the per-spec list still follows when the
	// warning pipeline has flagged individual specs.
	if p.SizeDrift != nil {
		fmt.Fprintf(&sb, "Size drift: %s\n\n", p.SizeDrift.Hint)
	}

	// Drift
	if len(p.Drift) > 0 {
		fmt.Fprintf(&sb, "Drift detected (%d):\n", len(p.Drift))
		for _, d := range p.Drift {
			icon := "⚠"
			if d.HasViolation {
				icon = "✗"
			}
			fmt.Fprintf(&sb, "  %s %s — %s (%d warnings)\n", icon, d.Slug, d.Title, d.Warnings)
		}
		sb.WriteString("\n")
	}

	// Knowledge updates
	fmt.Fprintf(&sb, "Knowledge updates (%d):\n", len(p.KnowledgeUpdates))
	if len(p.KnowledgeUpdates) == 0 {
		sb.WriteString("  (none)\n")
	} else {
		for _, u := range p.KnowledgeUpdates {
			fmt.Fprintf(&sb, "  + %s updated %s\n", u.Slug, u.ModifiedAt.Format("Jan 2"))
		}
	}
	sb.WriteString("\n")

	// Blockers
	if len(p.Blockers) == 0 {
		sb.WriteString("No blockers detected.\n")
	} else {
		fmt.Fprintf(&sb, "Blockers (%d):\n", len(p.Blockers))
		for _, b := range p.Blockers {
			fmt.Fprintf(&sb, "  ! %s\n", b)
		}
	}

	return sb.String()
}

// RenderMarkdown renders a PulseData as GitHub-flavored markdown.
func RenderMarkdown(p *PulseData) string {
	var sb strings.Builder

	weekOf := p.Period.From.Format("Jan 2, 2006")
	to := p.Period.To.Format("Jan 2, 2006")
	fmt.Fprintf(&sb, "# Sprint Pulse — %s to %s\n\n", weekOf, to)

	// Done
	fmt.Fprintf(&sb, "## Done this sprint (%d)\n\n", len(p.Done))
	if len(p.Done) == 0 {
		sb.WriteString("*(none)*\n\n")
	} else {
		for _, s := range p.Done {
			trackerSuffix := ""
			if s.TrackerID != "" {
				trackerSuffix = fmt.Sprintf(" `%s`", s.TrackerID)
			}
			fmt.Fprintf(&sb, "- ✅ **%s** — %s%s\n", s.Slug, s.Title, trackerSuffix)
		}
		sb.WriteString("\n")
	}

	// In flight
	fmt.Fprintf(&sb, "## In flight (%d)\n\n", len(p.InFlight))
	if len(p.InFlight) == 0 {
		sb.WriteString("*(none)*\n\n")
	} else {
		for _, s := range p.InFlight {
			claimedSuffix := ""
			if s.ClaimedBy != "" {
				claimedSuffix = fmt.Sprintf(" *(claimed: %s)*", s.ClaimedBy)
			}
			fmt.Fprintf(&sb, "- 🔄 **%s** — %s%s\n", s.Slug, s.Title, claimedSuffix)
		}
		sb.WriteString("\n")
	}

	// At risk
	fmt.Fprintf(&sb, "## At risk (%d)\n\n", len(p.AtRisk))
	if len(p.AtRisk) == 0 {
		sb.WriteString("*(none)*\n\n")
	} else {
		for _, s := range p.AtRisk {
			staleSuffix := ""
			if s.DaysStale > 0 {
				staleSuffix = fmt.Sprintf(" *(stale: %d days)*", s.DaysStale)
			}
			fmt.Fprintf(&sb, "- ⚠️ **%s** — %s%s\n", s.Slug, s.Title, staleSuffix)
		}
		sb.WriteString("\n")
	}

	// Ambient size-drift (workspace-wide, count + hint only).
	if p.SizeDrift != nil {
		fmt.Fprintf(&sb, "## Size drift\n\n%s\n\n", p.SizeDrift.Hint)
	}

	// Drift
	if len(p.Drift) > 0 {
		fmt.Fprintf(&sb, "## Drift detected (%d)\n\n", len(p.Drift))
		for _, d := range p.Drift {
			severity := "warning"
			if d.HasViolation {
				severity = "violation"
			}
			fmt.Fprintf(&sb, "- ⚠️ **%s** — %s *(%d warnings, %s)*\n", d.Slug, d.Title, d.Warnings, severity)
		}
		sb.WriteString("\n")
	}

	// Knowledge updates
	fmt.Fprintf(&sb, "## Knowledge updates (%d)\n\n", len(p.KnowledgeUpdates))
	if len(p.KnowledgeUpdates) == 0 {
		sb.WriteString("*(none)*\n\n")
	} else {
		for _, u := range p.KnowledgeUpdates {
			fmt.Fprintf(&sb, "- **%s** — %s *(updated %s)*\n", u.Slug, u.Title, u.ModifiedAt.Format("Jan 2"))
		}
		sb.WriteString("\n")
	}

	// Blockers
	fmt.Fprintf(&sb, "## Blockers\n\n")
	if len(p.Blockers) == 0 {
		sb.WriteString("No blockers detected.\n")
	} else {
		for _, b := range p.Blockers {
			fmt.Fprintf(&sb, "- 🚫 %s\n", b)
		}
	}

	return sb.String()
}

// RenderJSON renders a PulseData as JSON.
func RenderJSON(p *PulseData) (string, error) {
	type jsonPeriod struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	type jsonSpec struct {
		Slug       string `json:"slug"`
		Title      string `json:"title"`
		Status     string `json:"status"`
		ClaimedBy  string `json:"claimed_by,omitempty"`
		LastCommit string `json:"last_commit,omitempty"`
		DaysStale  int    `json:"days_stale,omitempty"`
		TrackerID  string `json:"tracker_id,omitempty"`
	}
	type jsonKnowledge struct {
		Slug       string `json:"slug"`
		Title      string `json:"title"`
		ModifiedAt string `json:"modified_at"`
	}
	type jsonDrift struct {
		Slug         string `json:"slug"`
		Title        string `json:"title"`
		Warnings     int    `json:"warnings"`
		HasViolation bool   `json:"has_violation"`
	}
	type jsonSizeDrift struct {
		Count int    `json:"count"`
		Hint  string `json:"hint"`
	}
	type jsonOutput struct {
		Period           jsonPeriod      `json:"period"`
		Done             []jsonSpec      `json:"done"`
		InFlight         []jsonSpec      `json:"in_flight"`
		AtRisk           []jsonSpec      `json:"at_risk"`
		Drift            []jsonDrift     `json:"drift,omitempty"`
		SizeDrift        *jsonSizeDrift  `json:"size_drift,omitempty"`
		KnowledgeUpdates []jsonKnowledge `json:"knowledge_updates"`
		Blockers         []string        `json:"blockers"`
	}

	toJSONSpec := func(s SpecSummary) jsonSpec {
		js := jsonSpec{
			Slug:      s.Slug,
			Title:     s.Title,
			Status:    s.Status,
			ClaimedBy: s.ClaimedBy,
			DaysStale: s.DaysStale,
			TrackerID: s.TrackerID,
		}
		if !s.LastCommit.IsZero() {
			js.LastCommit = s.LastCommit.Format(time.RFC3339)
		}
		return js
	}

	out := jsonOutput{
		Period: jsonPeriod{
			From: p.Period.From.Format(time.RFC3339),
			To:   p.Period.To.Format(time.RFC3339),
		},
		Done:             make([]jsonSpec, 0, len(p.Done)),
		InFlight:         make([]jsonSpec, 0, len(p.InFlight)),
		AtRisk:           make([]jsonSpec, 0, len(p.AtRisk)),
		KnowledgeUpdates: make([]jsonKnowledge, 0, len(p.KnowledgeUpdates)),
		Blockers:         p.Blockers,
	}

	if out.Blockers == nil {
		out.Blockers = []string{}
	}

	for _, s := range p.Done {
		out.Done = append(out.Done, toJSONSpec(s))
	}
	for _, s := range p.InFlight {
		out.InFlight = append(out.InFlight, toJSONSpec(s))
	}
	for _, s := range p.AtRisk {
		out.AtRisk = append(out.AtRisk, toJSONSpec(s))
	}
	for _, d := range p.Drift {
		out.Drift = append(out.Drift, jsonDrift{
			Slug:         d.Slug,
			Title:        d.Title,
			Warnings:     d.Warnings,
			HasViolation: d.HasViolation,
		})
	}
	if p.SizeDrift != nil {
		out.SizeDrift = &jsonSizeDrift{
			Count: p.SizeDrift.Count,
			Hint:  p.SizeDrift.Hint,
		}
	}
	for _, u := range p.KnowledgeUpdates {
		out.KnowledgeUpdates = append(out.KnowledgeUpdates, jsonKnowledge{
			Slug:       u.Slug,
			Title:      u.Title,
			ModifiedAt: u.ModifiedAt.Format(time.RFC3339),
		})
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
