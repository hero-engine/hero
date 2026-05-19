package data

import (
	"fmt"
	"html/template"
	"strings"
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

// LoadInbox composes the Needs-your-input rows from every source the
// solo-mode dashboard knows about: proposals, inbound handoffs, recent
// blocker events, peer-call findings awaiting acknowledgement, and
// specs sitting at status: in-review. Never returns nil — empty slices
// are fine.
//
// Spec: dashboard-inbox-misses-most-activity-sources. Previously only
// proposals + inbound handoffs were surfaced, and proposals were
// hardcoded nil — the inbox read empty even when other actionable
// signals were piling up.
func LoadInbox(in InboxInputs) Inbox {
	rows := make([]InboxRow, 0)
	for _, p := range in.Proposals {
		rows = append(rows, proposalRow(p))
	}
	rows = append(rows, inboundHandoffRows(in)...)
	rows = append(rows, blockerEventRows(in)...)
	rows = append(rows, peerFindingsRows(in)...)
	rows = append(rows, pendingReviewRows(in)...)

	// PR review mentions and import drafts are tracker-integrated and
	// surface only under team+ today. The placeholder row keeps the
	// payload shape stable; the team-mode wiring lives in a sibling
	// spec (see Boundaries in the spec).
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

// blockerEventRows surfaces recent `blocker_hit` events from the
// trailing 7-day window. Each blocker becomes one row pointing at the
// associated spec (when one is named). Hardening the inbox against
// silent regressions — a blocker_hit logged by `hero check` or an
// agent now lands in the user's eye line.
func blockerEventRows(in InboxInputs) []InboxRow {
	if in.HeroDir == "" {
		return nil
	}
	const window = 7 * 24 * time.Hour
	evts := readEventsBest(in.HeroDir, time.Now().Add(-window), 0)
	var rows []InboxRow
	seen := map[string]bool{}
	for _, e := range evts {
		if e.Type != "blocker_hit" {
			continue
		}
		// Dedup on slug+message — if a check loop emits the same blocker
		// repeatedly we still surface one row.
		key := e.Slug + "|" + e.Message
		if seen[key] {
			continue
		}
		seen[key] = true

		summaryParts := []string{`<a href="#">`}
		if e.Slug != "" {
			summaryParts = append(summaryParts,
				"Blocker on <code style=\"font-size:13px;\">",
				template.HTMLEscapeString(e.Slug),
				"</code>")
		} else {
			summaryParts = append(summaryParts, "Blocker")
		}
		if e.Message != "" {
			summaryParts = append(summaryParts, ": ", template.HTMLEscapeString(e.Message))
		}
		summaryParts = append(summaryParts, `</a>`)

		metaParts := []string{"Blocker"}
		if e.Agent != "" {
			metaParts = append(metaParts, "from "+template.HTMLEscapeString(displayAgent(e.Agent)))
		}
		metaParts = append(metaParts, prettyAge(e.Timestamp))

		rows = append(rows, InboxRow{
			Kind:    "blocker",
			Summary: template.HTML(strings.Join(summaryParts, "")),
			Meta:    template.HTML(joinSep(metaParts)),
			Actions: []InboxAction{
				{Label: "Open", Href: "#"},
				{Label: "Dismiss", Href: "#", Variant: "muted"},
			},
		})
	}
	return rows
}

// peerFindingsRows surfaces `peer.call.completed` events whose message
// declares findings (kind=findings). A peer asked to investigate has
// returned an answer and is awaiting acknowledgement. Reads from the
// trailing 7-day window and dedups per call_id.
func peerFindingsRows(in InboxInputs) []InboxRow {
	if in.HeroDir == "" {
		return nil
	}
	const window = 7 * 24 * time.Hour
	evts := readEventsBest(in.HeroDir, time.Now().Add(-window), 0)
	var rows []InboxRow
	seen := map[string]bool{}
	for _, e := range evts {
		if e.Type != "peer.call.completed" {
			continue
		}
		// Only surface findings-class results — other call kinds are
		// already-applied side effects (advisory acked, spec-out
		// returned). Findings need a human follow-up.
		if !strings.Contains(e.Message, "kind=findings") {
			continue
		}
		callID := extractCallID(e.Message)
		key := callID
		if key == "" {
			key = e.Timestamp.Format(time.RFC3339Nano) + "|" + e.Slug
		}
		if seen[key] {
			continue
		}
		seen[key] = true

		summary := fmt.Sprintf(`<a href="#">Peer returned findings`)
		if e.Slug != "" {
			summary += fmt.Sprintf(` on <code style="font-size:13px;">%s</code>`, template.HTMLEscapeString(e.Slug))
		}
		summary += `</a>`

		metaParts := []string{"Peer findings"}
		if target := extractField(e.Message, "target="); target != "" {
			metaParts = append(metaParts, "from "+template.HTMLEscapeString(target))
		}
		metaParts = append(metaParts, prettyAge(e.Timestamp))

		rows = append(rows, InboxRow{
			Kind:    "review",
			Summary: template.HTML(summary),
			Meta:    template.HTML(joinSep(metaParts)),
			Actions: []InboxAction{
				{Label: "Review findings", Href: "#"},
				{Label: "Dismiss", Href: "#", Variant: "muted"},
			},
		})
	}
	return rows
}

// pendingReviewRows surfaces specs that currently sit at
// status: in-review. Once a spec lands in this state the author (or
// reviewer in team mode) needs to take action. Solo mode treats every
// in-review spec as the user's responsibility.
func pendingReviewRows(in InboxInputs) []InboxRow {
	if in.HeroDir == "" {
		return nil
	}
	specs, err := spec.Discover(in.HeroDir)
	if err != nil {
		return nil
	}
	var rows []InboxRow
	for _, s := range specs {
		if s == nil || s.Status != spec.StatusInReview {
			continue
		}
		title := s.Title
		if title == "" {
			title = s.Slug
		}
		summary := fmt.Sprintf(`<a href="#">Review pending: <code style="font-size:13px;">%s</code></a>`,
			template.HTMLEscapeString(s.Slug))
		metaParts := []string{
			"In review",
			template.HTMLEscapeString(title),
		}
		if !s.ModifiedAt.IsZero() {
			metaParts = append(metaParts, prettyAge(s.ModifiedAt))
		}
		rows = append(rows, InboxRow{
			Kind:    "review",
			Summary: template.HTML(summary),
			Meta:    template.HTML(joinSep(metaParts)),
			Actions: []InboxAction{
				{Label: "Open review", Href: "#"},
			},
		})
	}
	return rows
}

// extractCallID parses "call_id=<id>" out of a peer-call message.
// Returns empty when no call_id is present.
func extractCallID(msg string) string {
	return extractField(msg, "call_id=")
}

// extractField returns the value of a "<key><sep>" run in msg, stopping
// at the next whitespace. Used to pull call_id, target, etc. out of the
// space-separated peer-call message payload.
func extractField(msg, prefix string) string {
	idx := strings.Index(msg, prefix)
	if idx < 0 {
		return ""
	}
	rest := msg[idx+len(prefix):]
	if end := strings.IndexAny(rest, " \t"); end >= 0 {
		rest = rest[:end]
	}
	return rest
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
