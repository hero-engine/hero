package snapshot

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// ApplyRetention enforces the configured retention policy on the
// archive directory. Returns the slugs of removed files. Errors are
// surfaced individually; partial removals proceed.
func ApplyRetention(heroDir string, cfg ArchiveConfig) ([]string, error) {
	policy := strings.ToLower(cfg.Retention)
	if policy == "" || policy == "all" {
		return nil, nil
	}

	archives, err := List(heroDir)
	if err != nil {
		return nil, err
	}

	switch policy {
	case "last-n":
		if cfg.RetentionCount <= 0 {
			return nil, nil
		}
		// archives are already newest-first from List; keep the
		// first N, delete the rest.
		if len(archives) <= cfg.RetentionCount {
			return nil, nil
		}
		toDelete := archives[cfg.RetentionCount:]
		return deleteArchives(toDelete)
	case "none":
		// Keep nothing — delete every archive.
		return deleteArchives(archives)
	}
	return nil, fmt.Errorf("snapshot: unknown retention policy %q", cfg.Retention)
}

func deleteArchives(archives []ArchiveRecord) ([]string, error) {
	// Sort by date asc so we delete oldest first for predictable logs.
	sort.Slice(archives, func(i, j int) bool {
		return archives[i].Date < archives[j].Date
	})
	var removed []string
	for _, a := range archives {
		if err := os.Remove(a.Path); err != nil {
			continue
		}
		removed = append(removed, a.Path)
	}
	return removed, nil
}
