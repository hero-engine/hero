package index

import (
	"fmt"
	"os"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// RefreshStats summarizes one RefreshIfStale call.
type RefreshStats struct {
	// Indexed counts specs that were on disk but absent from the
	// index and got newly inserted.
	Indexed int
	// Updated counts specs that were already indexed but had a newer
	// disk mtime than the stored modified_at and got re-indexed.
	Updated int
	// Removed counts orphan slugs — present in the index but with no
	// corresponding spec.md on disk — that got deleted.
	Removed int
	// Scanned is the total number of specs walked on disk.
	Scanned int
	// DurationMS is wall-clock time of the refresh.
	DurationMS int64
}

// IsClean reports whether the refresh found nothing to do.
func (r RefreshStats) IsClean() bool {
	return r.Indexed == 0 && r.Updated == 0 && r.Removed == 0
}

// RefreshIfStale diffs the index against disk truth and applies
// surgical updates: indexes specs that aren't in the index yet,
// re-indexes specs whose disk mtime is newer than the stored
// modified_at, and removes orphan slugs whose spec.md no longer
// exists on disk.
//
// The function is the staleness-self-healing primitive that lets
// read-side tools (`hero search`, `hero_list`, etc.) self-heal
// before querying. Steady-state cost when nothing's changed: one
// SELECT and one stat per spec.md on disk — microseconds total.
//
// Spec: index-staleness-auto-refresh.
func RefreshIfStale(heroDir string) (RefreshStats, error) {
	start := time.Now()
	var stats RefreshStats

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return stats, fmt.Errorf("discovering specs: %w", err)
	}
	stats.Scanned = len(specs)

	idx, err := Open(heroDir)
	if err != nil {
		return stats, fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	// Snapshot the indexed view: slug -> stored modified_at.
	indexed, err := indexedModifiedAt(idx)
	if err != nil {
		return stats, fmt.Errorf("reading index: %w", err)
	}

	// Walk disk and apply differences.
	seenSlugs := make(map[string]struct{}, len(specs))
	for _, s := range specs {
		seenSlugs[s.Slug] = struct{}{}
		stored, ok := indexed[s.Slug]
		if !ok {
			if err := indexOne(idx, s); err != nil {
				return stats, fmt.Errorf("indexing %s: %w", s.Slug, err)
			}
			stats.Indexed++
			continue
		}
		// Compare at second precision: stored values use RFC3339
		// (seconds), so the disk mtime's sub-second portion would
		// otherwise spuriously trigger re-index every call.
		if s.ModifiedAt.Truncate(time.Second).After(stored) {
			if err := indexOne(idx, s); err != nil {
				return stats, fmt.Errorf("re-indexing %s: %w", s.Slug, err)
			}
			stats.Updated++
		}
	}

	// Remove orphans — slugs in the index whose spec.md no longer
	// exists on disk (deleted by hand, lost in a git operation,
	// moved by a tool that bypassed the index).
	for slug := range indexed {
		if _, ok := seenSlugs[slug]; !ok {
			if err := idx.RemoveSpec(slug); err != nil {
				return stats, fmt.Errorf("removing orphan %s: %w", slug, err)
			}
			stats.Removed++
		}
	}

	stats.DurationMS = time.Since(start).Milliseconds()
	return stats, nil
}

// indexedModifiedAt returns slug -> stored modified_at across all
// rows in the specs table. RFC3339 strings are parsed back into
// time.Time so callers can use Go's time comparison.
func indexedModifiedAt(idx *DB) (map[string]time.Time, error) {
	rows, err := idx.db.Query(`SELECT slug, modified_at FROM specs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]time.Time)
	for rows.Next() {
		var slug, raw string
		if err := rows.Scan(&slug, &raw); err != nil {
			return nil, err
		}
		t, parseErr := time.Parse(time.RFC3339, raw)
		if parseErr != nil {
			// Bad timestamp in DB — treat as epoch so any disk
			// version with a real mtime wins on the next compare.
			t = time.Time{}
		}
		out[slug] = t
	}
	return out, rows.Err()
}

// indexOne re-reads the spec file from disk and runs IndexSpec.
// Centralized so the read-error handling stays consistent across
// the new-and-changed paths.
func indexOne(idx *DB, s *spec.Spec) error {
	content, err := os.ReadFile(s.Path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", s.Path, err)
	}
	return idx.IndexSpec(s, string(content))
}
