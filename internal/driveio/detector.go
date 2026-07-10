// Package driveio wires the pure /drive judge (internal/drive) to the
// sqlite-backed index (internal/index). It exists solely so internal/drive
// stays free of an internal/index import: drive.Check takes an injected,
// nil-safe detector callback, and this package is the single place that builds
// that callback from index.FindDeliveringConflicts. Both callers
// (hero goal --check and MCP hero_goal) use Detector, so the
// filter+map lives in exactly one place and their detected-gate verdicts are
// parity-identical by construction.
package driveio

import (
	"github.com/hero-engine/hero/internal/drive"
	"github.com/hero-engine/hero/internal/index"
)

// Detector returns the drive.Check detector callback backed by the index's
// delivering-filtered overlap engine. For a candidate slug it reports each
// currently-delivering spec whose files whole-file-overlap the candidate,
// mapped to drive.DetectedConflict (preserving the index's slug/file order).
// A query error yields nil: a detector failure must not crash a /drive run,
// consistent with the best-effort `conflicts, _ := idx.FindConflicts(...)`
// usage elsewhere.
//
// The overlap engine reads files_touched, populated from each spec's
// `## Changes` section — so the detector is effective for designed children
// (which have a file footprint) and weak for bare stubs (which have none yet).
// This is acceptable: by the time the judge routes a child to deliver, it is
// designed and has a footprint.
func Detector(idx *index.DB) func(string) []drive.DetectedConflict {
	return func(slug string) []drive.DetectedConflict {
		conflicts, err := idx.FindDeliveringConflicts(slug)
		if err != nil {
			return nil
		}
		out := make([]drive.DetectedConflict, 0, len(conflicts))
		for _, c := range conflicts {
			out = append(out, drive.DetectedConflict{Slug: c.Slug, Files: c.OverlappingFiles})
		}
		return out
	}
}
