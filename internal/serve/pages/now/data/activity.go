package data

import (
	"fmt"
	"html/template"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/feed"
)

// Window is the activity-feed time filter. Empty / unknown falls back
// to WindowWeek which is the default 7-day pane.
type Window string

const (
	WindowToday Window = "today"
	WindowWeek  Window = "week"
	WindowMonth Window = "month"
	WindowAll   Window = "all"
)

// WindowFromString resolves a user-supplied filter slug to a Window
// value. Unknown values default to WindowWeek so a stray query string
// never empties the feed.
func WindowFromString(s string) Window {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "today":
		return WindowToday
	case "week", "":
		return WindowWeek
	case "month":
		return WindowMonth
	case "all":
		return WindowAll
	}
	return WindowWeek
}

// Since returns the lower-bound timestamp for events the window
// considers. WindowAll returns the zero time (no lower bound).
func (w Window) Since(now time.Time) time.Time {
	switch w {
	case WindowToday:
		return startOfDay(now)
	case WindowMonth:
		return now.Add(-30 * 24 * time.Hour)
	case WindowAll:
		return time.Time{}
	default:
		return now.Add(-7 * 24 * time.Hour)
	}
}

// Label returns the pill copy for the filter row.
func (w Window) Label() string {
	switch w {
	case WindowToday:
		return "Today"
	case WindowMonth:
		return "This month"
	case WindowAll:
		return "All"
	default:
		return "This week"
	}
}

// ActivityInputs is the per-request input bundle for the activity
// feed. ProjectRoot is currently unused (the feed reads only the
// events log under HeroDir) but is kept on the struct so future
// signals like local file mtimes can hang off without a contract
// change.
type ActivityInputs struct {
	ProjectRoot string
	HeroDir     string
	UserName    string
	Window      Window
	// Aggregate, when non-empty, fans out across the listed project
	// contexts and merges their feeds into one chronological stream.
	// Each entry carries the project slug so the UI can tag rows.
	// Empty == single-project mode (use ProjectRoot/HeroDir).
	Aggregate []ActivityProject
}

// ActivityProject is one project contributing to an aggregate feed.
// Kept here (rather than importing serve.ProjectContext) so this
// package stays import-leaf-free.
type ActivityProject struct {
	Slug    string
	Path    string
	HeroDir string
}

// Activity is the page-section payload.
type Activity struct {
	Window   Window
	Filters  []ActivityFilter
	Entries  []ActivityEntry
	Total    int
	Empty    bool
	// Aggregate is true when the feed was assembled across multiple
	// projects — the template uses this to show per-row project tags.
	Aggregate bool
}

// ActivityFilter is one pill in the filter row.
type ActivityFilter struct {
	Window Window
	Label  string
	Slug   string
	Active bool
	Href   string
}

// ActivityEntry is one row in the chronological feed.
//
// Kind drives the icon ("spec" | "decision" | "knowledge" | "handoff"
// | "commit" | "agent" | "pulse"). GroupCount > 1 indicates several
// events of the same kind on the same parent slug collapsed into one
// row, matching the existing Changes-section pattern.
type ActivityEntry struct {
	Kind       string
	Title      template.HTML
	Subtitle   string
	Link       string
	Project    string
	Timestamp  time.Time
	TimeChip   string
	GroupCount int
}

// LoadActivity composes the Now page activity feed for the given
// window. Reads .hero/events.log (the same source as the Since-you-
// were-here section) but exposes every event-kind the spec lists —
// spec status transitions, decisions, notes, conventions, peer calls,
// agent sessions, and commits — under one feed.
//
// Empty event logs produce an empty Activity rather than a stub. The
// template renders an honest empty state for genuinely-empty projects.
func LoadActivity(in ActivityInputs) Activity {
	win := in.Window
	if win == "" {
		win = WindowWeek
	}
	since := win.Since(time.Now())

	urlPrefix := ""
	if in.Aggregate != nil {
		urlPrefix = "/p/all"
	} else if in.HeroDir != "" {
		// Single-project mode: derive the project slug from the path
		// so deep-links resolve under /p/<slug>/. The slug is what the
		// router uses; the data layer doesn't need to know more.
		urlPrefix = "/p/" + filepath.Base(filepath.Dir(in.HeroDir))
	}

	var raw []rawEvent
	if in.Aggregate != nil {
		for _, p := range in.Aggregate {
			for _, e := range readEventsBest(p.HeroDir, since, 0) {
				raw = append(raw, rawEvent{
					evt:     e,
					project: p.Slug,
				})
			}
		}
		sort.SliceStable(raw, func(i, j int) bool {
			return raw[i].evt.Timestamp.After(raw[j].evt.Timestamp)
		})
	} else {
		for _, e := range readEventsBest(in.HeroDir, since, 0) {
			raw = append(raw, rawEvent{evt: e})
		}
	}

	entries := make([]ActivityEntry, 0, len(raw))
	for _, r := range raw {
		entry := activityEntryFor(r.evt, r.project, urlPrefix)
		if entry.Kind == "" {
			continue
		}
		entries = append(entries, entry)
	}
	entries = groupSameSpecKind(entries)

	out := Activity{
		Window:    win,
		Filters:   buildFilters(win, urlPrefix),
		Entries:   entries,
		Total:     len(entries),
		Empty:     len(entries) == 0,
		Aggregate: in.Aggregate != nil,
	}
	return out
}

type rawEvent struct {
	evt     feed.FeedEvent
	project string
}

func activityEntryFor(e feed.FeedEvent, project, urlPrefix string) ActivityEntry {
	kind, title, sub := classifyEvent(e)
	if kind == "" {
		return ActivityEntry{}
	}
	link := linkForEvent(e, urlPrefix)
	return ActivityEntry{
		Kind:       kind,
		Title:      template.HTML(title),
		Subtitle:   sub,
		Link:       link,
		Project:    project,
		Timestamp:  e.Timestamp,
		TimeChip:   prettyAge(e.Timestamp),
		GroupCount: 1,
	}
}

// classifyEvent returns (kind, sentence-form HTML title, subtitle) for
// one feed event. Returning kind=="" suppresses the row — used for
// events that don't belong in the user-facing feed (raw heartbeat
// types, internal claim_acquired bookkeeping when not from an agent).
//
// Kind values map to the icon set in templates/activity.html.
func classifyEvent(e feed.FeedEvent) (kind, title, sub string) {
	slug := template.HTMLEscapeString(e.Slug)
	agent := template.HTMLEscapeString(displayAgent(e.Agent))
	msg := template.HTMLEscapeString(e.Message)
	switch {
	case e.Type == "delivery_complete":
		return "spec", "Spec <strong>" + slug + "</strong> completed", agent
	case e.Type == "delivery_start":
		return "spec", "Started <strong>" + slug + "</strong>", agent
	case e.Type == "spec_created":
		return "spec", "Spec <strong>" + slug + "</strong> created", agent
	case e.Type == "spec_updated":
		return "spec", "Spec <strong>" + slug + "</strong> updated", agent
	case e.Type == "spec.status_changed":
		return "spec", "Spec <strong>" + slug + "</strong> status changed", agent
	case e.Type == "decision_made":
		title = "Decision recorded"
		if slug != "" {
			title += " on <strong>" + slug + "</strong>"
		}
		return "decision", title, agent
	case e.Type == "knowledge.captured", e.Type == "note.captured":
		title = "Knowledge captured"
		if slug != "" {
			title += ": <strong>" + slug + "</strong>"
		}
		return "knowledge", title, agent
	case e.Type == "convention.detected":
		return "knowledge", "Convention detected: <strong>" + slug + "</strong>", agent
	case strings.HasPrefix(e.Type, "peer.call."):
		title = "Peer call"
		if strings.HasSuffix(e.Type, "completed") {
			title = "Peer call completed"
		}
		if slug != "" {
			title += " on <strong>" + slug + "</strong>"
		}
		return "handoff", title, agent
	case strings.HasPrefix(e.Type, "peer.handoff."):
		title = "Peer handoff"
		if strings.HasSuffix(e.Type, "received") {
			title = "Peer handoff received"
		} else if strings.HasSuffix(e.Type, "accepted") {
			title = "Peer handoff accepted"
		} else if strings.HasSuffix(e.Type, "sent") {
			title = "Peer handoff sent"
		}
		if slug != "" {
			title += ": <strong>" + slug + "</strong>"
		}
		return "handoff", title, agent
	case e.Type == "agent_session_started":
		title = "Agent session started"
		if slug != "" {
			title += " on <strong>" + slug + "</strong>"
		}
		return "agent", title, agent
	case e.Type == "agent_session_ended":
		title = "Agent session ended"
		if slug != "" {
			title += " on <strong>" + slug + "</strong>"
		}
		return "agent", title, agent
	case e.Type == "commit", e.Type == "files_modified":
		title = "Commit"
		if msg != "" {
			title = "<strong>Commit</strong> " + msg
		}
		return "commit", title, agent
	case e.Type == "blocker_hit":
		title = "Blocker"
		if slug != "" {
			title += " on <strong>" + slug + "</strong>"
		}
		if msg != "" {
			title += ": " + msg
		}
		return "pulse", title, agent
	}
	return "", "", ""
}

// linkForEvent picks a deep-link target for an event. We default to
// the project-scoped spec page when the event names one; events
// without a spec target leave Link empty so the template renders a
// non-clickable row.
func linkForEvent(e feed.FeedEvent, urlPrefix string) string {
	if e.Slug == "" {
		return ""
	}
	if urlPrefix == "" {
		return "/work/spec/" + e.Slug
	}
	return urlPrefix + "/work/spec/" + e.Slug
}

// groupSameSpecKind collapses consecutive entries that share both
// Kind and Project+Slug into a single row, matching the spec's "When
// multiple commits land on the same spec within a single feed window
// THE SYSTEM SHALL collapse them into a single grouped entry showing
// the count" requirement. Items without a slug are passed through.
func groupSameSpecKind(entries []ActivityEntry) []ActivityEntry {
	if len(entries) == 0 {
		return entries
	}
	out := make([]ActivityEntry, 0, len(entries))
	type key struct {
		kind    string
		slug    string
		project string
	}
	lastIdx := -1
	var lastKey key
	for _, e := range entries {
		// Pull slug from the link rather than re-parsing the title.
		slug := slugFromLink(e.Link)
		k := key{kind: e.Kind, slug: slug, project: e.Project}
		if slug != "" && lastIdx >= 0 && k == lastKey {
			out[lastIdx].GroupCount++
			continue
		}
		out = append(out, e)
		lastIdx = len(out) - 1
		lastKey = k
	}
	return out
}

func slugFromLink(link string) string {
	if link == "" {
		return ""
	}
	const marker = "/work/spec/"
	if i := strings.Index(link, marker); i >= 0 {
		return link[i+len(marker):]
	}
	return ""
}

func buildFilters(active Window, urlPrefix string) []ActivityFilter {
	all := []Window{WindowToday, WindowWeek, WindowMonth, WindowAll}
	out := make([]ActivityFilter, 0, len(all))
	pageHref := urlPrefix
	if pageHref == "" {
		pageHref = "/now"
	} else {
		pageHref += "/now"
	}
	for _, w := range all {
		out = append(out, ActivityFilter{
			Window: w,
			Label:  w.Label(),
			Slug:   string(w),
			Active: w == active,
			Href:   fmt.Sprintf("%s?window=%s", pageHref, w),
		})
	}
	return out
}

// startOfDay returns the local midnight cut for the given timestamp.
func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
