// Package peering implements the cross-repo peering primitives —
// peer identity (UUID minting + migration), manifest generation,
// legacy handoff lifecycle, trail read/write, and Project Mail routing.
//
// Wire shapes live in contracts/peering. This package is the CLI-side
// implementation that produces and consumes those shapes against the
// local filesystem.
package peering

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/feed"
)

// MintPeerID generates a new RFC 4122 v4 UUID, written as a lowercase
// canonical string. Used at `hero init` and during migration of a
// pre-peer_id workspace on its first invocation.
func MintPeerID() string {
	return uuid.NewString()
}

// EnsurePeerID returns the workspace's PeerID, minting and persisting
// a new one if absent. Safe to call on any hero invocation — it's a
// no-op when peer_id is already set. The mint, when it happens, is
// recorded in the workspace events log so the moment of identity
// assignment is recoverable.
//
// projectRoot is the workspace project root (the directory that
// contains the .hero/ folder).
//
// trigger is "init" when called from `hero init` and "migration"
// when called from a generic hero invocation against an existing
// workspace that predates peer_id.
//
// Returns (peerID, minted) where minted is true iff this call wrote
// a new peer_id.
func EnsurePeerID(projectRoot string, trigger string) (string, bool, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return "", false, fmt.Errorf("load config: %w", err)
	}
	if cfg.PeerID != "" {
		return cfg.PeerID, false, nil
	}

	cfg.PeerID = MintPeerID()
	if err := cfg.Save(projectRoot); err != nil {
		return "", false, fmt.Errorf("save config: %w", err)
	}

	// Best-effort mint event. A failure here does NOT roll back the
	// peer_id write — the identity is the durable artifact; the event
	// is audit metadata.
	logPath := filepath.Join(cfg.HeroDir(projectRoot), "events.log")
	_ = feed.AppendEvent(logPath, feed.FeedEvent{
		Timestamp: time.Now().UTC(),
		Type:      string(peering.EventPeerIDMinted),
		Agent:     "hero",
		Message:   fmt.Sprintf("peer_id minted (%s): %s", trigger, cfg.PeerID),
	})

	return cfg.PeerID, true, nil
}

// RecordPeerIDMintEvent appends a workspace.peer_id_minted event to
// the workspace events log. Best-effort: errors are swallowed because
// the durable artifact is the peer_id in hero.json, not this event.
//
// Use this when you've minted a peer_id outside of EnsurePeerID (for
// example, in `hero init` where the mint happens as part of writing
// the initial config). EnsurePeerID handles the event write itself.
func RecordPeerIDMintEvent(heroDir, peerID, trigger string) {
	if heroDir == "" || peerID == "" {
		return
	}
	logPath := filepath.Join(heroDir, "events.log")
	_ = feed.AppendEvent(logPath, feed.FeedEvent{
		Timestamp: time.Now().UTC(),
		Type:      string(peering.EventPeerIDMinted),
		Agent:     "hero",
		Message:   fmt.Sprintf("peer_id minted (%s): %s", trigger, peerID),
	})
}

// EnsurePeerIDOnHeroDir is a convenience variant that takes a
// workspace .hero/ directory path instead of the project root.
func EnsurePeerIDOnHeroDir(heroDir string, trigger string) (string, bool, error) {
	if heroDir == "" {
		return "", false, errors.New("hero directory required")
	}
	projectRoot := filepath.Dir(heroDir)
	if _, err := os.Stat(heroDir); err != nil {
		return "", false, fmt.Errorf("hero workspace not found at %s: %w", heroDir, err)
	}
	return EnsurePeerID(projectRoot, trigger)
}
