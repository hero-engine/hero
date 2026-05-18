package data

import (
	"fmt"
	"html/template"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// InboxInputs is the per-request input bundle for the Needs-your-input
// section. Proposals is passed in (rather than discovered) because the
// proposal store is owned by the serve package.
type InboxInputs struct {
	ProjectRoot string
	HeroDir     string
	Edition     string // "local" | "team" | ...
	Proposals   []*ProposalRow
}

// LoadInbox composes the Needs-your-input rows from proposals, inbound
// handoffs, and (under team+) PR review mentions / import drafts. Never
// returns nil — empty slices are fine.
func LoadInbox(in InboxInputs) Inbox {
	rows := make([]InboxRow, 0)
	for _, p := range in.Proposals {
		rows = append(rows, proposalRow(p))
	}
	rows = append(rows, inboundHandoffRows(in)...)

	// PR review mentions and import drafts are tracker-integrated and
	// surface only under team+ today. The placeholder row keeps the
	// payload shape stable; the team-mode wiring lives in a sibling
	// spec.
	_ = in.Edition

	return Inbox{Rows: rows, Total: len(rows)}
}

// proposalRow converts one ProposalRow into a displayable inbox row.
// Inline actions point at the canonical proposal endpoints
// (/api/{project}/sessions/{sid}/proposals/{pid}/{action}).
func proposalRow(p *ProposalRow) InboxRow {
	summary := fmt.Sprintf(`<a href="#">%s proposes change to <code style="font-size:13px;">%s</code></a>`,
		template.HTMLEscapeString(displayAgent(p.Agent)),
		template.HTMLEscapeString(p.AnchorValue))

	metaParts := []string{
		"Proposal",
	}
	if p.SpecSlug != "" {
		metaParts = append(metaParts, "spec "+template.HTMLEscapeString(p.SpecSlug))
	}
	metaParts = append(metaParts, prettyAge(p.EmittedAt))
	meta := joinSep(metaParts)

	return InboxRow{
		Kind:    "proposal",
		Summary: template.HTML(summary),
		Meta:    template.HTML(meta),
		Actions: []InboxAction{
			{Label: "Approve", Href: "#"},
			{Label: "View diff", Href: "#"},
			{Label: "Reject", Href: "#", Variant: "danger"},
		},
	}
}

// inboundHandoffRows scans local specs for `ReceivedFrom` blocks (the
// scaffolds peer handoffs drop) and surfaces them as Accept-handoff
// rows. Failure to read specs returns an empty slice.
func inboundHandoffRows(in InboxInputs) []InboxRow {
	if in.HeroDir == "" {
		return nil
	}
	specs, err := spec.Discover(in.HeroDir)
	if err != nil {
		return nil
	}
	var rows []InboxRow
	for _, s := range specs {
		if s.ReceivedFrom == nil {
			continue
		}
		// Only surface still-pending inbound handoffs.
		if s.Status != spec.StatusPlanning && s.Status != "handed_off" {
			continue
		}
		peer := s.ReceivedFrom.PeerAliasDisplay
		if peer == "" {
			peer = "peer"
		}
		summary := fmt.Sprintf(`<a href="#">%s handed back <code style="font-size:13px;">%s</code></a>`,
			template.HTMLEscapeString(peer),
			template.HTMLEscapeString(s.Slug))
		meta := joinSep([]string{
			"Handoff",
			"peer: " + template.HTMLEscapeString(peer),
			prettyAge(s.ReceivedFrom.HandedOffAt),
		})
		rows = append(rows, InboxRow{
			Kind:    "handoff",
			Summary: template.HTML(summary),
			Meta:    template.HTML(meta),
			Actions: []InboxAction{
				{Label: "Accept handoff", Href: "#"},
				{Label: "Open", Href: "#"},
			},
		})
	}
	return rows
}

// displayAgent collapses long agent identifiers to a readable label.
func displayAgent(agent string) string {
	if agent == "" {
		return "agent"
	}
	return agent
}

// prettyAge returns a Linear-style relative time string like "14m" or
// "2h" or "3d". Empty time returns "just now".
func prettyAge(t time.Time) string {
	if t.IsZero() {
		return "just now"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// joinSep joins a list of HTML-safe strings with the Linear-style
// middle-dot separator.
func joinSep(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ` <span class="sep">·</span> `
		}
		out += p
	}
	return out
}
