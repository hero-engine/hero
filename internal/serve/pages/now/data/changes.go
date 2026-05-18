package data

import (
	"fmt"
	"html/template"
	"os/exec"
	"strings"
	"time"
)

// ChangesInputs is the per-request input bundle for the Since-you-were-
// here section.
type ChangesInputs struct {
	ProjectRoot string
	HeroDir     string
}

// dedupWindow is the time window within which same-display-group
// events collapse into a single row with a count badge. Per polish-v3
// Fix 3: 1 hour is the default; events further apart render as
// separate rows.
const dedupWindow = time.Hour

// changesDisplayLimit caps the dedup'd row count we surface in the
// section. We over-fetch raw events so dedup has material to collapse,
// then truncate to this limit after dedup.
const changesDisplayLimit = 6

// LoadChanges returns the top 5-6 entries for the timeline feed.
// Prefers the next-handoff-emit events log; falls back to a commits-
// only view when that pipeline is unavailable.
//
// Dedup: consecutive same-display-group rows within a 1-hour window
// collapse into a single row with a count badge + expand affordance.
// We over-fetch raw events (4× the display limit) so the dedup pass
// has material to compress.
func LoadChanges(in ChangesInputs) Changes {
	events := readEventsBest(in.HeroDir, time.Time{}, changesDisplayLimit*4)
	if len(events) > 0 {
		rows := make([]ChangeRow, 0, len(events))
		for _, e := range events {
			rows = append(rows, ChangeRow{
				Time:         prettyAge(e.Timestamp),
				Kind:         kindFromEventType(e.Type),
				HTML:         renderEventText(e.Type, e.Slug, e.Message, e.Agent),
				TimeAt:       e.Timestamp,
				DisplayGroup: displayGroupFor(e.Type),
				GroupLabel:   groupLabelFor(e.Type),
				Count:        1,
			})
		}
		rows = dedupeWithinWindow(rows, dedupWindow)
		if len(rows) > changesDisplayLimit {
			rows = rows[:changesDisplayLimit]
		}
		return Changes{Rows: rows, Limited: false}
	}
	return commitsOnly(in.ProjectRoot)
}

// dedupeWithinWindow walks the input rows (already time-sorted newest-
// first) and collapses runs of consecutive same-display-group rows
// whose timestamps fall within `window`. The kept row inherits the
// most-recent row's metadata (icon, time chip, HTML); Count is the
// run length and CollapsedRows lists the originals (without recursing
// — collapsed rows are flat for the expand affordance).
//
// Rows with an empty TimeAt or empty DisplayGroup are never collapsed
// — they pass through individually so the test fixtures and the
// commits-only fallback keep their old behavior.
func dedupeWithinWindow(rows []ChangeRow, window time.Duration) []ChangeRow {
	if len(rows) == 0 {
		return rows
	}
	out := make([]ChangeRow, 0, len(rows))
	for _, r := range rows {
		if r.DisplayGroup == "" || r.TimeAt.IsZero() {
			out = append(out, r)
			continue
		}
		if len(out) > 0 {
			last := &out[len(out)-1]
			if last.DisplayGroup == r.DisplayGroup && !last.TimeAt.IsZero() {
				// Newest first, so last.TimeAt >= r.TimeAt; use the
				// absolute difference so out-of-order inputs still
				// match within the window.
				delta := last.TimeAt.Sub(r.TimeAt)
				if delta < 0 {
					delta = -delta
				}
				if delta < window {
					last.Count++
					last.CollapsedRows = append(last.CollapsedRows, r)
					last.WindowText = humanWindowText(last.TimeAt.Sub(r.TimeAt))
					continue
				}
			}
		}
		out = append(out, r)
	}
	return out
}

// humanWindowText renders a "within the last <span>" string for the
// count badge subtext. Always uses the larger unit so the badge stays
// short.
func humanWindowText(d time.Duration) string {
	if d < time.Minute {
		return "within the last minute"
	}
	if d < time.Hour {
		mins := int(d / time.Minute)
		if mins <= 1 {
			return "within the last minute"
		}
		return fmt.Sprintf("within the last %dm", mins)
	}
	hours := int(d / time.Hour)
	if hours <= 1 {
		return "within the last hour"
	}
	return fmt.Sprintf("within the last %dh", hours)
}

// displayGroupFor maps a raw event type to its dedup display group.
// Multiple raw types collapse together when they describe variants of
// the same action (e.g. peer.call.invoked and peer.call.completed are
// both "peer-call"). Returning "" disables dedup for that type.
func displayGroupFor(t string) string {
	switch {
	case strings.HasPrefix(t, "peer.call."):
		return "peer-call"
	case strings.HasPrefix(t, "peer.handoff."):
		return "peer-handoff"
	case t == "spec_created", t == "spec_updated", t == "spec.status_changed":
		return "spec-change"
	case t == "knowledge.captured", t == "note.captured":
		return "knowledge"
	case t == "files_modified", t == "commit":
		return "commit"
	case t == "decision_made":
		return "decision"
	case t == "delivery_complete":
		return "delivery"
	case t == "blocker_hit":
		return "blocker"
	default:
		return ""
	}
}

// groupLabelFor returns the plural human label used in the count
// badge for a raw event type (e.g. "peer calls", "spec updates").
func groupLabelFor(t string) string {
	switch displayGroupFor(t) {
	case "peer-call":
		return "peer calls"
	case "peer-handoff":
		return "peer handoffs"
	case "spec-change":
		return "spec updates"
	case "knowledge":
		return "knowledge captures"
	case "commit":
		return "commits"
	case "decision":
		return "decisions"
	case "delivery":
		return "deliveries"
	case "blocker":
		return "blockers"
	default:
		return "events"
	}
}

// commitsOnly is the fallback rendering when the event-log pipeline
// returns nothing — we drop back to a simple git log.
func commitsOnly(projectRoot string) Changes {
	if projectRoot == "" {
		return Changes{Limited: true}
	}
	out, err := exec.Command("git", "-C", projectRoot, "log",
		"-n", "6",
		"--pretty=format:%h|%s|%an|%ar").Output()
	if err != nil {
		return Changes{Limited: true}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	rows := make([]ChangeRow, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		sha, subject, author, when := parts[0], parts[1], parts[2], parts[3]
		text := fmt.Sprintf(`<a href="#" class="now-feed-mono">%s</a> <strong>%s</strong> <span class="now-feed-actor">· %s</span>`,
			template.HTMLEscapeString(sha),
			template.HTMLEscapeString(subject),
			template.HTMLEscapeString(author))
		rows = append(rows, ChangeRow{
			Time: prettyAgeShort(when),
			Kind: "commit",
			HTML: template.HTML(text),
		})
	}
	return Changes{Rows: rows, Limited: true}
}

// kindFromEventType maps a feed event type to a feed-icon kind. The
// mapping drives the colored dot / icon in the Since-you-were-here
// feed. Default fallback is `pulse` (a neutral dot) — we deliberately
// avoid `convention` as a catch-all because nearly every workspace
// event was rendering with the convention icon (the original bug).
func kindFromEventType(t string) string {
	switch {
	case strings.HasPrefix(t, "peer."):
		return "handoff"
	case t == "decision_made":
		return "decision"
	// canonical completion verb is delivery_complete; spec.complete
	// was a draft that never landed.
	case t == "delivery_complete":
		return "check"
	case t == "spec.status_changed":
		return "spec"
	case t == "spec_created", t == "spec_updated":
		return "spec"
	case t == "knowledge.captured", t == "note.captured":
		return "knowledge"
	case t == "blocker_hit":
		return "drift"
	case t == "files_modified":
		return "commit"
	case t == "commit":
		return "commit"
	default:
		return "pulse"
	}
}

// renderEventText produces sentence-form HTML for one timeline row.
func renderEventText(typ, slug, msg, agent string) template.HTML {
	slugSafe := template.HTMLEscapeString(slug)
	msgSafe := template.HTMLEscapeString(msg)
	agentSafe := template.HTMLEscapeString(agent)
	switch typ {
	// canonical completion verb is delivery_complete; spec.complete
	// was a draft that never landed.
	case "delivery_complete":
		return template.HTML(fmt.Sprintf(`Spec <a href="#">%s</a> <strong>completed</strong> <span class="now-feed-actor">· %s</span>`, slugSafe, agentSafe))
	case "spec_updated":
		return template.HTML(fmt.Sprintf(`Spec <a href="#">%s</a> updated <span class="now-feed-actor">· %s</span>`, slugSafe, agentSafe))
	case "spec_created":
		return template.HTML(fmt.Sprintf(`Spec <a href="#">%s</a> created <span class="now-feed-actor">· %s</span>`, slugSafe, agentSafe))
	case "spec.status_changed":
		return template.HTML(fmt.Sprintf(`Spec <a href="#">%s</a> status changed <span class="now-feed-actor">· %s</span>`, slugSafe, agentSafe))
	case "decision_made":
		return template.HTML(fmt.Sprintf(`Decision recorded on <a href="#">%s</a> <span class="now-feed-actor">· %s</span>`, slugSafe, agentSafe))
	case "peer.call.invoked":
		return template.HTML(fmt.Sprintf(`Peer call invoked on <a href="#">%s</a> <span class="now-feed-actor">· %s</span>`, slugSafe, agentSafe))
	case "peer.call.completed":
		return template.HTML(fmt.Sprintf(`Peer call completed on <a href="#">%s</a> <span class="now-feed-actor">· %s</span>`, slugSafe, agentSafe))
	case "knowledge.captured", "note.captured":
		return template.HTML(fmt.Sprintf(`Knowledge captured <a href="#">%s</a> <span class="now-feed-actor">· %s</span>`, slugSafe, agentSafe))
	default:
		if msg == "" {
			msg = typ
			msgSafe = template.HTMLEscapeString(msg)
		}
		if slugSafe != "" {
			return template.HTML(fmt.Sprintf(`<a href="#">%s</a> %s <span class="now-feed-actor">· %s</span>`, slugSafe, msgSafe, agentSafe))
		}
		return template.HTML(fmt.Sprintf(`%s <span class="now-feed-actor">· %s</span>`, msgSafe, agentSafe))
	}
}

// prettyAgeShort compacts "12 minutes ago" style git output into the
// timeline chip format ("12m", "2h", "1d").
func prettyAgeShort(ago string) string {
	ago = strings.TrimSpace(ago)
	if ago == "" {
		return ""
	}
	// "2 minutes ago" → "2m"
	parts := strings.Fields(ago)
	if len(parts) < 2 {
		return ago
	}
	unit := parts[1]
	switch {
	case strings.HasPrefix(unit, "second"):
		return parts[0] + "s"
	case strings.HasPrefix(unit, "minute"):
		return parts[0] + "m"
	case strings.HasPrefix(unit, "hour"):
		return parts[0] + "h"
	case strings.HasPrefix(unit, "day"):
		return parts[0] + "d"
	case strings.HasPrefix(unit, "week"):
		return parts[0] + "w"
	case strings.HasPrefix(unit, "month"):
		return parts[0] + "mo"
	case strings.HasPrefix(unit, "year"):
		return parts[0] + "y"
	default:
		return parts[0]
	}
}
