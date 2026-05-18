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

// LoadChanges returns the top 5-6 entries for the timeline feed.
// Prefers the next-handoff-emit events log; falls back to a commits-
// only view when that pipeline is unavailable.
func LoadChanges(in ChangesInputs) Changes {
	events := readEventsBest(in.HeroDir, time.Time{}, 6)
	if len(events) > 0 {
		rows := make([]ChangeRow, 0, len(events))
		for _, e := range events {
			rows = append(rows, ChangeRow{
				Time: prettyAge(e.Timestamp),
				Kind: kindFromEventType(e.Type),
				HTML: renderEventText(e.Type, e.Slug, e.Message, e.Agent),
			})
		}
		return Changes{Rows: rows, Limited: false}
	}
	return commitsOnly(in.ProjectRoot)
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
	case t == "delivery_complete", t == "spec.complete":
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
	case "delivery_complete", "spec.complete":
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
