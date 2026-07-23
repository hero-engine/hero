// Package feed provides a cross-session activity feed built on .hero/events.log.
// It extends the existing ClaimEvent format with richer event types while
// remaining backward-compatible with the old format.
package feed

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Valid event types.
//
// Adding a new type: append it here. ReadEvents tolerates unknown
// types in old log lines, but the IsValidType gate on hero event
// blocks the CLI from creating them ad-hoc. Cross-repo peering event
// kinds (peer.*, workspace.peer_id_minted) are listed alongside the
// original feed kinds so the events.log stays consistent.
var ValidTypes = []string{
	"spec_created",
	"spec_updated",
	"spec.status_changed",
	"files_modified",
	"decision_made",
	"blocker_hit",
	"delivery_complete",
	"subproject_changed",
	"drive.pause_outcome",
	"mail.read",
	"mail.acknowledge",
	"mail.dismiss",
	"mail.promote",
	"mail.add_to_today",

	// cross-repo-peering (Phase 0+1)
	"workspace.peer_id_minted",
	"peer.handoff.sent",
	"peer.handoff.received",
	"peer.handoff.bounced",
	"peer.handoff.accepted",
	"peer.call.invoked",
	"peer.call.completed",
}

// FeedEvent is a single entry in the activity feed.
type FeedEvent struct {
	Timestamp  time.Time `json:"ts"`
	Type       string    `json:"type"`
	Agent      string    `json:"agent"`
	Session    string    `json:"session,omitempty"`
	Slug       string    `json:"slug,omitempty"`
	Subproject string    `json:"subproject,omitempty"`
	Message    string    `json:"message"`
}

// AppendEvent appends a feed event to the events log.
func AppendEvent(logPath string, evt FeedEvent) error {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening events log: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

// Filter controls which events are returned by ReadEvents.
type Filter struct {
	Since      time.Time
	Type       string
	Slug       string
	Agent      string
	Subproject string // exact-match filter on event.Subproject; "all" or "" disables
	Limit      int
}

// ReadEvents reads the events log, handling both new FeedEvent format and
// old ClaimEvent format (with "event"/"at" fields).
func ReadEvents(logPath string, filter Filter) ([]FeedEvent, error) {
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening events log: %w", err)
	}
	defer f.Close()

	var all []FeedEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		evt, ok := parseLine(line)
		if !ok {
			continue
		}

		// Apply filters
		if !filter.Since.IsZero() && evt.Timestamp.Before(filter.Since) {
			continue
		}
		if filter.Type != "" && evt.Type != filter.Type {
			continue
		}
		if filter.Slug != "" && evt.Slug != filter.Slug {
			continue
		}
		if filter.Agent != "" && evt.Agent != filter.Agent {
			continue
		}
		if filter.Subproject != "" && filter.Subproject != "all" && evt.Subproject != filter.Subproject {
			continue
		}

		all = append(all, evt)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Reverse to newest-first
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}

	// Apply limit
	if filter.Limit > 0 && len(all) > filter.Limit {
		all = all[:filter.Limit]
	}

	return all, nil
}

// parseLine tries to parse a JSON line as either FeedEvent or old ClaimEvent.
func parseLine(line []byte) (FeedEvent, bool) {
	// Try new format first
	var evt FeedEvent
	if err := json.Unmarshal(line, &evt); err == nil && evt.Type != "" {
		return evt, true
	}

	// Try old ClaimEvent format: {"event":"claim","slug":"x","agent":"y","at":"..."}
	var old struct {
		Event string    `json:"event"`
		Slug  string    `json:"slug"`
		Agent string    `json:"agent"`
		At    time.Time `json:"at"`
	}
	if err := json.Unmarshal(line, &old); err == nil && old.Event != "" {
		return FeedEvent{
			Timestamp: old.At,
			Type:      "spec_updated",
			Agent:     old.Agent,
			Slug:      old.Slug,
			Message:   fmt.Sprintf("claim event: %s", old.Event),
		}, true
	}

	return FeedEvent{}, false
}

// IsValidType checks if a type string is in the valid types list.
func IsValidType(t string) bool {
	for _, valid := range ValidTypes {
		if t == valid {
			return true
		}
	}
	return false
}

// FormatText produces human-readable output for a list of events.
func FormatText(events []FeedEvent) string {
	if len(events) == 0 {
		return "No events found.\n"
	}

	var b strings.Builder
	for _, e := range events {
		ts := e.Timestamp.Format("15:04")
		typeLabel := shortType(e.Type)
		slug := e.Slug
		if slug == "" {
			slug = "—"
		}
		fmt.Fprintf(&b, "%s [%-10s] %-20s %-20s %s\n", ts, typeLabel, e.Agent, slug, e.Message)
	}
	return b.String()
}

// FormatJSON produces JSON output for a list of events.
func FormatJSON(events []FeedEvent) (string, error) {
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func shortType(t string) string {
	switch t {
	case "spec_created":
		return "spec"
	case "spec_updated":
		return "updated"
	case "spec.status_changed":
		return "status"
	case "files_modified":
		return "files"
	case "decision_made":
		return "decision"
	case "blocker_hit":
		return "blocker"
	case "delivery_complete":
		return "delivered"
	default:
		return t
	}
}

// ParseSince parses a --since value as either a duration string ("1h", "30m")
// or an RFC 3339 timestamp.
func ParseSince(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	// Try duration first
	d, err := time.ParseDuration(s)
	if err == nil {
		return time.Now().UTC().Add(-d), nil
	}

	// Try RFC 3339
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	// Try date only
	t, err = time.Parse("2006-01-02", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("cannot parse %q as duration or timestamp", s)
}
