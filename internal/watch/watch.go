// Package watch provides polling-based file change detection for the hero workspace.
// It monitors .hero/ directory for spec changes, triggering reindex and health checks.
package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EventKind describes what happened to a file.
type EventKind string

const (
	EventCreated  EventKind = "created"
	EventModified EventKind = "modified"
	EventDeleted  EventKind = "deleted"
)

// Event represents a single file change detected by the watcher.
type Event struct {
	Path string
	Kind EventKind
}

// String returns a human-readable description of the event.
func (e Event) String() string {
	return fmt.Sprintf("[%s] %s", e.Kind, e.Path)
}

// IsSpec returns true if the event is for a spec.md file.
func (e Event) IsSpec() bool {
	return strings.HasSuffix(e.Path, "/spec.md") || filepath.Base(e.Path) == "spec.md"
}

// Handler is called when file changes are detected.
// It receives the list of events since the last poll.
type Handler func(events []Event)

// Watcher polls the hero directory for file changes.
type Watcher struct {
	heroDir  string
	interval time.Duration
	handler  Handler
	snapshot map[string]time.Time // path -> mtime
	stopCh   chan struct{}
}

// New creates a watcher for the given hero directory.
func New(heroDir string, interval time.Duration, handler Handler) *Watcher {
	return &Watcher{
		heroDir:  heroDir,
		interval: interval,
		handler:  handler,
		snapshot: make(map[string]time.Time),
		stopCh:   make(chan struct{}),
	}
}

// Scan walks the hero directory and builds a map of file paths to modification times.
// Only .md files are tracked.
func Scan(heroDir string) (map[string]time.Time, error) {
	result := make(map[string]time.Time)

	err := filepath.Walk(heroDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip files we can't read
		}
		if info.IsDir() {
			// Skip hidden directories inside heroDir (e.g. .git inside .hero is unlikely but safe)
			base := filepath.Base(path)
			if base != filepath.Base(heroDir) && strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only track markdown files and hero.json
		if strings.HasSuffix(path, ".md") || filepath.Base(path) == "hero.json" {
			result[path] = info.ModTime()
		}

		return nil
	})

	return result, err
}

// Diff compares two snapshots and returns the list of changes.
func Diff(oldSnap, newSnap map[string]time.Time) []Event {
	var events []Event

	// Check for new or modified files
	for path, newMtime := range newSnap {
		oldMtime, exists := oldSnap[path]
		if !exists {
			events = append(events, Event{Path: path, Kind: EventCreated})
		} else if !newMtime.Equal(oldMtime) {
			events = append(events, Event{Path: path, Kind: EventModified})
		}
	}

	// Check for deleted files
	for path := range oldSnap {
		if _, exists := newSnap[path]; !exists {
			events = append(events, Event{Path: path, Kind: EventDeleted})
		}
	}

	return events
}

// Run starts the polling loop. It blocks until Stop is called or the context
// signals cancellation. Returns nil when stopped gracefully.
func (w *Watcher) Run() error {
	// Build initial snapshot
	snap, err := Scan(w.heroDir)
	if err != nil {
		return fmt.Errorf("initial scan: %w", err)
	}
	w.snapshot = snap

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return nil
		case <-ticker.C:
			w.poll()
		}
	}
}

// poll performs one scan and fires the handler if changes were detected.
func (w *Watcher) poll() {
	newSnap, err := Scan(w.heroDir)
	if err != nil {
		return // silently skip failed polls
	}

	events := Diff(w.snapshot, newSnap)
	if len(events) > 0 {
		w.snapshot = newSnap
		w.handler(events)
	} else {
		// Update snapshot even if no events — in case mtime precision matters
		w.snapshot = newSnap
	}
}

// Stop signals the watcher to stop.
func (w *Watcher) Stop() {
	close(w.stopCh)
}

// RunOnce performs a single scan, compares to any previous state, and fires the handler.
// This is the CI mode entry point — scan once, report, exit.
func (w *Watcher) RunOnce() ([]Event, error) {
	snap, err := Scan(w.heroDir)
	if err != nil {
		return nil, fmt.Errorf("scanning hero directory: %w", err)
	}

	events := Diff(w.snapshot, snap)
	w.snapshot = snap

	if len(events) > 0 && w.handler != nil {
		w.handler(events)
	}

	return events, nil
}

// SpecEvents filters events to only include spec.md files.
func SpecEvents(events []Event) []Event {
	var result []Event
	for _, e := range events {
		if e.IsSpec() {
			result = append(result, e)
		}
	}
	return result
}

// Summary returns a human-readable summary of the events.
func Summary(events []Event) string {
	if len(events) == 0 {
		return "No changes detected."
	}

	var created, modified, deleted int
	for _, e := range events {
		switch e.Kind {
		case EventCreated:
			created++
		case EventModified:
			modified++
		case EventDeleted:
			deleted++
		}
	}

	var parts []string
	if created > 0 {
		parts = append(parts, fmt.Sprintf("%d created", created))
	}
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", modified))
	}
	if deleted > 0 {
		parts = append(parts, fmt.Sprintf("%d deleted", deleted))
	}

	return fmt.Sprintf("%d change(s): %s", len(events), strings.Join(parts, ", "))
}
