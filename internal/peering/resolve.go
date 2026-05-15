package peering

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	contractpeering "github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// ReadPeerManifest reads and parses a peer's manifest from
// `<peerPath>/<folder>/peer-manifest.yaml`. Returns a clear error when
// the file is missing or unreadable — callers should surface that to
// the user with a `hero index` hint.
func ReadPeerManifest(peerPath, folder string) (*contractpeering.PeerManifest, error) {
	if folder == "" {
		folder = config.DefaultFolder
	}
	manifestPath := filepath.Join(peerPath, folder, PeerManifestFileName)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("peer manifest missing at %s — run `hero index` in %s", manifestPath, peerPath)
		}
		return nil, fmt.Errorf("read peer manifest %s: %w", manifestPath, err)
	}
	var m contractpeering.PeerManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse peer manifest %s: %w", manifestPath, err)
	}
	return &m, nil
}

// FilterConventionsBySurface returns the subset of entries whose
// Surface slice contains the given surface tag. Empty surface returns
// all entries. Case-insensitive match.
func FilterConventionsBySurface(entries []contractpeering.ConventionEntry, surface string) []contractpeering.ConventionEntry {
	if surface == "" {
		return entries
	}
	want := strings.ToLower(surface)
	var out []contractpeering.ConventionEntry
	for _, e := range entries {
		for _, s := range e.Surface {
			if strings.ToLower(s) == want {
				out = append(out, e)
				break
			}
		}
	}
	return out
}

// PeerSpecStatus reports the current status of a peer-side spec by
// reading its frontmatter directly. Used by the auto-fire reconciler
// to decide whether an awaiting_peer originator should be moved to
// handed_back.
//
// peerSpec is the "<alias>/<slug>" form recorded in trail entries.
// Returns the parsed status string and the absolute path to the spec
// file, or "", "", nil when the peer spec cannot be found. Errors are
// returned only for unexpected I/O failures.
func PeerSpecStatus(projectRoot string, cfg config.Config, peerSpec string) (status, path string, err error) {
	if peerSpec == "" {
		return "", "", nil
	}
	alias, slug, ok := strings.Cut(peerSpec, "/")
	if !ok || slug == "" {
		return "", "", nil
	}
	peerPath, err := cfg.ResolveRepoPath(projectRoot, alias)
	if err != nil {
		// Unconfigured alias — silent miss, not an error worth
		// surfacing during a render-time reconciliation.
		return "", "", nil
	}
	peerHeroDir := filepath.Join(peerPath, cfg.Folder)
	if _, err := os.Stat(peerHeroDir); err != nil {
		return "", "", nil
	}

	// Search the peer's planning + specs buckets for the slug.
	candidates := []string{
		filepath.Join(peerHeroDir, "planning"),
		filepath.Join(peerHeroDir, "specs"),
	}
	for _, root := range candidates {
		hit := findSpecBySlug(root, slug)
		if hit != "" {
			s, parseErr := spec.ParseFile(hit)
			if parseErr != nil {
				return "", hit, parseErr
			}
			return string(s.Status), hit, nil
		}
	}
	return "", "", nil
}

// findSpecBySlug walks a planning- or specs-root looking for a
// `<slug>/spec.md`. Returns the matched absolute path or "".
func findSpecBySlug(root, slug string) string {
	if root == "" {
		return ""
	}
	// Common shape: <root>/<bucket>/<slug>/spec.md. Walk one level of
	// bucket directories so we don't scan deeply.
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		candidate := filepath.Join(root, e.Name(), slug, "spec.md")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// ReconcileAwaitingPeer walks every spec in awaiting_peer status,
// reads the peer counterpart from the latest spec-out / async-drop
// trail entry, and flips local status to handed_back when the peer
// spec has reached completed. Best-effort: errors per-spec are logged
// to the caller-supplied logger (nil → silent) but do not abort the
// reconciliation of other specs.
//
// Returns the slugs that transitioned. Used by `hero status` and
// `hero queue` render paths so the user sees handed-back work without
// running a separate command.
func ReconcileAwaitingPeer(projectRoot string, log func(format string, args ...any)) ([]string, error) {
	if log == nil {
		log = func(string, ...any) {}
	}
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discover specs: %w", err)
	}

	var transitioned []string
	for _, s := range specs {
		if s.Status != spec.StatusAwaitingPeer {
			continue
		}
		entries, err := ReadTrail(s.Path)
		if err != nil {
			log("reconcile: read trail for %s: %v", s.Slug, err)
			continue
		}
		// Look at the most recent outgoing trail entry that references
		// a peer spec.
		var latest *contractpeering.TrailEntry
		for i := range entries {
			e := entries[i]
			if e.Direction != contractpeering.DirectionOut {
				continue
			}
			if e.PeerSpec == "" {
				continue
			}
			if latest == nil || e.At.After(latest.At) {
				cp := e
				latest = &cp
			}
		}
		if latest == nil {
			continue
		}
		peerStatus, peerPath, err := PeerSpecStatus(projectRoot, cfg, latest.PeerSpec)
		if err != nil {
			log("reconcile: peer status for %s: %v", latest.PeerSpec, err)
			continue
		}
		if peerStatus != string(spec.StatusCompleted) {
			continue
		}
		// Move originator to handed_back and append a handed-back
		// trail entry pointing back at the peer's completed spec.
		entry := contractpeering.TrailEntry{
			At:               nowFn().UTC(),
			Direction:        contractpeering.DirectionIn,
			PeerAliasDisplay: latest.PeerAliasDisplay,
			PeerID:           latest.PeerID,
			Mode:             contractpeering.ModeHandedBack,
			OriginatingSpec:  s.Slug,
			PeerSpec:         latest.PeerSpec,
			PeerStatus:       string(spec.StatusCompleted),
			ResultRef:        peerPath,
			Reason:           "peer-side spec reached completed",
		}
		data, err := os.ReadFile(s.Path)
		if err != nil {
			log("reconcile: read %s: %v", s.Slug, err)
			continue
		}
		updated := spec.SetFrontmatterField(string(data), "status", string(spec.StatusHandedBack))
		updated = AppendTrailToContent(updated, entry)
		if err := os.WriteFile(s.Path, []byte(updated), 0o644); err != nil {
			log("reconcile: write %s: %v", s.Slug, err)
			continue
		}
		transitioned = append(transitioned, s.Slug)
	}
	return transitioned, nil
}

// nowFn is a test seam for time.Now in reconciliation. Production
// callers don't touch it; tests can override.
var nowFn = func() time.Time { return time.Now() }
