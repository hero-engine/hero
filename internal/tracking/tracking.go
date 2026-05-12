package tracking

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
	"time"
)

// ClaimEvent represents an event in the events log.
type ClaimEvent struct {
	Event           string    `json:"event"`
	Slug            string    `json:"slug"`
	Agent           string    `json:"agent"`
	At              time.Time `json:"at"`
	DurationMinutes int       `json:"duration_minutes,omitempty"`
}

// AppendEvent appends an event to .hero/events.log.
func AppendEvent(eventsLogPath string, event ClaimEvent) error {
	f, err := os.OpenFile(eventsLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening events log: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

// ReadEvents reads all events from .hero/events.log.
func ReadEvents(eventsLogPath string) ([]ClaimEvent, error) {
	f, err := os.Open(eventsLogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening events log: %w", err)
	}
	defer f.Close()

	var events []ClaimEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var evt ClaimEvent
		if err := json.Unmarshal(line, &evt); err != nil {
			continue // skip malformed lines
		}
		events = append(events, evt)
	}
	return events, scanner.Err()
}

// ActiveClaims returns currently active claims (claimed but not released/completed).
func ActiveClaims(events []ClaimEvent) []ClaimEvent {
	// Track state per slug: last event wins
	type state struct {
		event  ClaimEvent
		active bool
	}
	latest := make(map[string]state)

	for _, evt := range events {
		switch evt.Event {
		case "claimed":
			latest[evt.Slug] = state{event: evt, active: true}
		case "released", "completed":
			latest[evt.Slug] = state{event: evt, active: false}
		}
	}

	var active []ClaimEvent
	for _, s := range latest {
		if s.active {
			active = append(active, s.event)
		}
	}
	return active
}

// StaleClaims returns claims with no activity for more than staleDays.
func StaleClaims(events []ClaimEvent, staleDays int) []ClaimEvent {
	cutoff := time.Now().AddDate(0, 0, -staleDays)

	// Find all active claims
	active := ActiveClaims(events)

	var stale []ClaimEvent
	for _, evt := range active {
		if evt.At.Before(cutoff) {
			stale = append(stale, evt)
		}
	}
	return stale
}

// UpdateSpecFrontmatter reads a spec file and sets/clears claimed_by and claimed_at.
// action: "claim", "release", "complete"
func UpdateSpecFrontmatter(specPath, action, agent string, claimedAt time.Time) error {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("reading spec: %w", err)
	}
	content := string(data)

	switch action {
	case "claim":
		content = spec.SetFrontmatterField(content, "claimed_by", agent)
		content = spec.SetFrontmatterField(content, "claimed_at", claimedAt.Format(time.RFC3339))
	case "release":
		content = removeFrontmatterField(content, "claimed_by")
		content = removeFrontmatterField(content, "claimed_at")
	case "complete":
		content = spec.SetFrontmatterField(content, "status", "completed")
		content = removeFrontmatterField(content, "claimed_by")
		content = removeFrontmatterField(content, "claimed_at")
	default:
		return fmt.Errorf("unknown action %q", action)
	}

	return os.WriteFile(specPath, []byte(content), 0o644)
}

// removeFrontmatterField removes a key line from YAML frontmatter.
func removeFrontmatterField(content, key string) string {
	lines := strings.Split(content, "\n")

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return content
	}

	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closeIdx = i
			break
		}
	}
	if closeIdx == -1 {
		return content
	}

	var newLines []string
	for i, line := range lines {
		if i >= 1 && i < closeIdx && strings.HasPrefix(line, key+":") {
			continue
		}
		newLines = append(newLines, line)
	}
	return strings.Join(newLines, "\n")
}
