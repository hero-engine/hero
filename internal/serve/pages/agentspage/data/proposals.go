package data

import (
	"fmt"
	"html/template"
	"time"
)

// ProposalsInputs is the per-request input bundle for the proposals
// section / sub-page. Proposals is dependency-injected; it returns the
// flat list of pending proposal rows grouped by batch.
type ProposalsInputs struct {
	ProjectRoot string
	HeroDir     string
	Edition     string

	// Proposals returns the canonical pending-proposal snapshot. The
	// shape mirrors the now-home loader so the same server-side
	// snapshotter can feed both pages.
	Proposals func() []ProposalRow
}

// ProposalRow is one pending proposal envelope, ledger-shape. Kept
// here so the page can build approval rows without importing the
// propose store directly.
type ProposalRow struct {
	ProposalID  string
	BatchID     string
	SessionID   string
	SpecSlug    string
	Agent       string
	AnchorValue string
	EmittedAt   time.Time
	FilesTouched []string
	Adds        int
	Dels        int
}

// Proposals is the /agents/proposals payload.
type Proposals struct {
	Rows     []ApprovalRow
	Awaiting int
}

// LoadProposals builds the approval-row list for the proposals page
// (and for the awaiting-approval section on the sessions view). Nil-
// safe: missing fetcher → empty rows.
func LoadProposals(in ProposalsInputs) Proposals {
	out := Proposals{Rows: []ApprovalRow{}}
	if in.Proposals == nil {
		return out
	}
	rows := in.Proposals()
	for _, r := range rows {
		out.Rows = append(out.Rows, proposalToApprovalRow(r))
	}
	out.Awaiting = len(out.Rows)
	return out
}

// proposalToApprovalRow renders one envelope to the flat approval-row
// shape shared with the awaiting-approval list on the sessions view.
func proposalToApprovalRow(p ProposalRow) ApprovalRow {
	summary := fmt.Sprintf(`<a href="/agents/proposals/%s">%s proposes change to <code style="font-size:13px;">%s</code></a>`,
		template.HTMLEscapeString(p.ProposalID),
		template.HTMLEscapeString(displayAgent(p.Agent)),
		template.HTMLEscapeString(p.AnchorValue))

	meta := approvalMetaLine(p)

	return ApprovalRow{
		Summary: template.HTML(summary),
		Meta:    template.HTML(meta),
		Actions: []SessionAction{
			{Label: "Approve", Href: "#"},
			{Label: "View diff", Href: "/agents/proposals/" + p.ProposalID},
			{Label: "Reject", Href: "#", Variant: "danger"},
		},
	}
}

// approvalMetaLine assembles the "n files · +adds / −dels · age · from /deliver on spec" line.
func approvalMetaLine(p ProposalRow) string {
	files := len(p.FilesTouched)
	if files == 0 {
		files = 1
	}
	noun := "file"
	if files != 1 {
		noun = "files"
	}
	out := fmt.Sprintf(`<span>%d %s</span>`, files, noun)
	if p.Adds != 0 || p.Dels != 0 {
		out += fmt.Sprintf(`<span class="sep">·</span><span class="add">+%d</span> / <span class="rem">−%d</span>`, p.Adds, p.Dels)
	}
	out += fmt.Sprintf(`<span class="sep">·</span><span>%s</span>`, prettyAge(p.EmittedAt))
	if p.SpecSlug != "" {
		out += fmt.Sprintf(`<span class="sep">·</span><span>from /deliver on %s</span>`,
			template.HTMLEscapeString(p.SpecSlug))
	}
	return out
}

// displayAgent collapses long agent identifiers to a readable label.
func displayAgent(agent string) string {
	if agent == "" {
		return "agent"
	}
	return agent
}
