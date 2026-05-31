// Package workspace resolves the active Hero workspace root and scope from
// any directory. It centralizes the walk-up logic so satellite folders,
// subfolders of a workspace, and direct workspace dirs all behave the same
// way for downstream commands.
package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// SatelliteMarker is the filename of the satellite marker file written
// inside satellite folders.
const SatelliteMarker = ".hero-satellite"

// HeroDir is the workspace state directory at the root.
const HeroDir = ".hero"

// SatelliteInfo is the on-disk shape of a .hero-satellite marker file.
type SatelliteInfo struct {
	// Root is the relative path from the satellite folder to the workspace
	// root (e.g. "../.."). Stored as relative so checkouts move portably.
	Root string `json:"root"`
	// Scope is the canonical scope identifier this satellite represents.
	Scope string `json:"scope"`
	// Version is the hero version string that materialized the satellite.
	Version string `json:"version,omitempty"`
}

// Workspace is the resolved view of where Hero state lives for a given cwd.
type Workspace struct {
	// Root is the absolute path of the workspace root (the directory that
	// contains .hero/).
	Root string
	// HeroDir is filepath.Join(Root, ".hero").
	HeroDir string
	// CWD is the directory the resolution started from (absolute).
	CWD string
	// IsSatellite is true when CWD is inside a satellite folder (a folder
	// containing a .hero-satellite marker), as opposed to being directly
	// at Root or in some unmarked subfolder.
	IsSatellite bool
	// SatellitePath, when IsSatellite is true, is the absolute path to the
	// satellite folder (the one containing the marker).
	SatellitePath string
	// MarkerScope, when IsSatellite is true, is the scope value read from
	// the marker file. Use Scope() to get the effective scope (which falls
	// back to subprojects.json resolution if no marker is present).
	MarkerScope string
}

// LocateOption configures Locate behavior.
type LocateOption func(*locateOptions)

type locateOptions struct {
	stopAt string
}

// WithStopAt bounds the parent walk so Locate will not ascend above the
// given directory. The boundary directory itself is still checked; its
// parent is not. Useful in tests to isolate the walk from stray .hero/
// directories that may exist in shared ancestors like /tmp.
func WithStopAt(dir string) LocateOption {
	return func(o *locateOptions) {
		o.stopAt = dir
	}
}

// Locate resolves the workspace from the given starting directory.
//
// Resolution order:
//  1. If startDir contains .hero/, that is Root (CWD == Root).
//  2. If startDir contains .hero-satellite, read Root from the marker.
//  3. Otherwise walk up the parent chain. The first ancestor with a
//     .hero-satellite (closer wins, more specific) marks a satellite
//     subtree. The first ancestor with .hero/ marks Root. If both are
//     found, the satellite-relative root from the marker takes priority
//     so an explicitly-marked satellite is never overridden by a more
//     distant ancestor.
//
// Returns an error if no workspace can be located.
func Locate(startDir string, opts ...LocateOption) (*Workspace, error) {
	var cfg locateOptions
	for _, opt := range opts {
		opt(&cfg)
	}

	abs, err := filepath.Abs(startDir)
	if err != nil {
		return nil, fmt.Errorf("resolve start dir: %w", err)
	}

	var stopAt string
	if cfg.stopAt != "" {
		stopAt, err = filepath.Abs(cfg.stopAt)
		if err != nil {
			return nil, fmt.Errorf("resolve stop-at dir: %w", err)
		}
	}

	// Direct workspace check at startDir.
	if isHeroRoot(abs) {
		return &Workspace{
			Root:    abs,
			HeroDir: filepath.Join(abs, HeroDir),
			CWD:     abs,
		}, nil
	}

	// Walk up looking for either a satellite marker or .hero/.
	dir := abs
	for {
		// Check for satellite marker first — closer marker wins.
		markerPath := filepath.Join(dir, SatelliteMarker)
		if info, err := readMarker(markerPath); err == nil {
			rootRel := info.Root
			if rootRel == "" {
				return nil, fmt.Errorf("satellite marker at %s has no root", markerPath)
			}
			rootAbs := filepath.Clean(filepath.Join(dir, rootRel))
			if !isHeroRoot(rootAbs) {
				return nil, fmt.Errorf("satellite marker at %s points to %s, which is not a Hero workspace", markerPath, rootAbs)
			}
			return &Workspace{
				Root:          rootAbs,
				HeroDir:       filepath.Join(rootAbs, HeroDir),
				CWD:           abs,
				IsSatellite:   true,
				SatellitePath: dir,
				MarkerScope:   info.Scope,
			}, nil
		}

		if isHeroRoot(dir) {
			return &Workspace{
				Root:    dir,
				HeroDir: filepath.Join(dir, HeroDir),
				CWD:     abs,
			}, nil
		}

		if stopAt != "" && dir == stopAt {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return nil, ErrNotFound
}

// LocateFromCWD is a convenience that resolves from os.Getwd.
func LocateFromCWD() (*Workspace, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getwd: %w", err)
	}
	return Locate(cwd)
}

// ErrNotFound indicates no workspace was found by walking up from the
// starting directory.
var ErrNotFound = errors.New("no Hero workspace found")

// isHeroRoot reports whether dir contains a .hero/ directory.
func isHeroRoot(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, HeroDir))
	return err == nil && info.IsDir()
}

// readMarker reads and parses a .hero-satellite marker file.
func readMarker(path string) (*SatelliteInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var info SatelliteInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &info, nil
}

// WriteMarker writes a satellite marker file at the given satellite folder.
// rootAbs is the absolute path to the workspace root; the function stores
// it as a path relative to satelliteDir so the marker survives checkout
// at a different absolute location.
func WriteMarker(satelliteDir, rootAbs, scope, version string) error {
	rel, err := filepath.Rel(satelliteDir, rootAbs)
	if err != nil {
		return fmt.Errorf("compute relative root: %w", err)
	}
	info := SatelliteInfo{
		Root:    rel,
		Scope:   scope,
		Version: version,
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(satelliteDir, SatelliteMarker), data, 0o644)
}

// RemoveMarker removes the satellite marker file from a folder. It is a
// no-op if no marker exists.
func RemoveMarker(satelliteDir string) error {
	err := os.Remove(filepath.Join(satelliteDir, SatelliteMarker))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
