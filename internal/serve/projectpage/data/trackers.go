package data

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
)

// TrackersInputs is the per-request input bundle for the Trackers
// section.
type TrackersInputs struct {
	ProjectRoot string
	HeroDir     string
}

// Trackers is what the partial renders.
type Trackers struct {
	// Configured is false when no tracker is configured — the partial
	// shows the opt-in card linking to setup docs.
	Configured bool

	// Type is "github" / "jira" / "linear" / "none" when Configured.
	Type string

	// Project is the tracker-side project / repo identifier.
	Project string

	// LastSyncAt is the last successful sync time pulled from a cached
	// artifact on disk. Zero time renders "as of: never".
	LastSyncAt       time.Time
	LastSyncAtPretty string

	// QueuedImports is the count of import entries waiting to be
	// promoted from a cached queue file. Zero when no queue file
	// exists.
	QueuedImports int

	// LastError, when non-empty, surfaces the most recent sync error
	// summary from disk.
	LastError string
}

// LoadTrackers reads hero.json's tracker block and looks for cached
// sync metadata on disk. Phase 1 never makes a network call.
func LoadTrackers(in TrackersInputs) Trackers {
	if in.ProjectRoot == "" {
		return Trackers{}
	}
	cfg, err := config.Load(in.ProjectRoot)
	if err != nil {
		return Trackers{}
	}
	if cfg.Tracker == nil || cfg.Tracker.Type == "" || strings.EqualFold(cfg.Tracker.Type, "none") {
		return Trackers{}
	}
	out := Trackers{
		Configured: true,
		Type:       cfg.Tracker.Type,
		Project:    cfg.Tracker.Project,
	}
	if in.HeroDir != "" {
		out.LastSyncAt = lastImportTimestamp(in.HeroDir)
		if !out.LastSyncAt.IsZero() {
			out.LastSyncAtPretty = out.LastSyncAt.Format("2006-01-02 15:04")
		}
	}
	return out
}

// lastImportTimestamp returns the most recent file mod time across the
// .hero/imports/ tree (where import artifacts live by convention).
// Returns zero time when the directory is missing.
func lastImportTimestamp(heroDir string) time.Time {
	importsDir := filepath.Join(heroDir, "imports")
	if _, err := os.Stat(importsDir); err != nil {
		return time.Time{}
	}
	var latest time.Time
	_ = filepath.WalkDir(importsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	return latest
}
