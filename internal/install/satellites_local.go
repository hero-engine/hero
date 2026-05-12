package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// SatellitesLocalFile is the gitignored per-machine manifest path under
// .hero/.
const SatellitesLocalFile = "satellites.local.json"

// SatelliteEntry is one materialized satellite folder on this machine.
type SatelliteEntry struct {
	// Path is forward-slash, relative to workspace root.
	Path string `json:"path"`
	// Targets is the harness targets (e.g. "claude", "codex") covered by
	// this satellite — one symlink tree per target.
	Targets []string `json:"targets"`
	// InstalledAt is the timestamp the satellite was last materialized
	// or repaired.
	InstalledAt time.Time `json:"installed_at"`
	// Degraded is true when the satellite was created in marker-only
	// fallback mode (e.g. Windows without symlink support). Repair will
	// re-attempt full materialization.
	Degraded bool `json:"degraded,omitempty"`
}

// SatellitesLocal is the on-disk shape of .hero/satellites.local.json.
//
// This file is gitignored. It records which satellites have been
// materialized on the current machine, with which harness targets, so
// repair and uninstall know where to act without walking the tree.
//
// AcknowledgedHints stores per-scope acknowledgment of one-time UI
// hints (e.g. the "showing scope: X" nudge surfaced on first use of a
// satellite). Per-machine because a teammate cloning the repo deserves
// to see the hint themselves on their first scoped command.
type SatellitesLocal struct {
	Version           int              `json:"version"`
	Satellites        []SatelliteEntry `json:"satellites"`
	AcknowledgedHints []string         `json:"acknowledged_hints,omitempty"`
}

// HasHintAck reports whether the given hint key has been acknowledged
// for this machine.
func (s *SatellitesLocal) HasHintAck(key string) bool {
	if s == nil {
		return false
	}
	for _, ack := range s.AcknowledgedHints {
		if ack == key {
			return true
		}
	}
	return false
}

// AddHintAck appends an acknowledgment if it isn't already present.
func (s *SatellitesLocal) AddHintAck(key string) {
	if s.HasHintAck(key) {
		return
	}
	s.AcknowledgedHints = append(s.AcknowledgedHints, key)
}

const satellitesLocalVersion = 1

// LoadSatellitesLocal reads the local manifest. Returns an empty
// (versioned) manifest if the file does not exist.
func LoadSatellitesLocal(heroDir string) (*SatellitesLocal, error) {
	path := filepath.Join(heroDir, SatellitesLocalFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SatellitesLocal{Version: satellitesLocalVersion}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var s SatellitesLocal
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if s.Version == 0 {
		s.Version = satellitesLocalVersion
	}
	return &s, nil
}

// SaveSatellitesLocal writes the local manifest to heroDir.
func SaveSatellitesLocal(heroDir string, s *SatellitesLocal) error {
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", heroDir, err)
	}
	out := &SatellitesLocal{
		Version:    satellitesLocalVersion,
		Satellites: append([]SatelliteEntry(nil), s.Satellites...),
	}
	sort.Slice(out.Satellites, func(i, j int) bool {
		return out.Satellites[i].Path < out.Satellites[j].Path
	})
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(heroDir, SatellitesLocalFile), data, 0o644)
}

// Find returns the entry for the given relative path, or nil if not present.
func (s *SatellitesLocal) Find(relPath string) *SatelliteEntry {
	if s == nil {
		return nil
	}
	norm := normalizeRelPath(relPath)
	for i := range s.Satellites {
		if normalizeRelPath(s.Satellites[i].Path) == norm {
			return &s.Satellites[i]
		}
	}
	return nil
}

// Upsert inserts or replaces an entry by path.
func (s *SatellitesLocal) Upsert(e SatelliteEntry) {
	e.Path = normalizeRelPath(e.Path)
	if e.InstalledAt.IsZero() {
		e.InstalledAt = time.Now().UTC()
	}
	for i := range s.Satellites {
		if normalizeRelPath(s.Satellites[i].Path) == e.Path {
			s.Satellites[i] = e
			return
		}
	}
	s.Satellites = append(s.Satellites, e)
}

// Remove drops an entry by path. Returns true if removed.
func (s *SatellitesLocal) Remove(relPath string) bool {
	norm := normalizeRelPath(relPath)
	for i := range s.Satellites {
		if normalizeRelPath(s.Satellites[i].Path) == norm {
			s.Satellites = append(s.Satellites[:i], s.Satellites[i+1:]...)
			return true
		}
	}
	return false
}
