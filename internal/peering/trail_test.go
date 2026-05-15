package peering

import (
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/peering"
)

// TestTrailRoundTrip confirms that an entry can be rendered, parsed
// back, and produce an equivalent struct (modulo formatting).
func TestTrailRoundTrip(t *testing.T) {
	entry := peering.TrailEntry{
		At:               time.Date(2026, 5, 15, 14, 0, 0, 0, time.UTC),
		Direction:        peering.DirectionOut,
		PeerAliasDisplay: "app",
		PeerID:           "9c1c2f3e-4a8b-4f9d-9a0e-7e1f0d8c3a55",
		Mode:             peering.ModeAsyncDrop,
		OriginatingSpec:  "order-failure-error-display",
		PeerSpec:         "app/error-envelope-mismatch",
		AtCommit:         "3176736",
		Reason:           "Symptom is in the client, root cause is the API response shape.",
	}

	rendered := RenderTrailSection([]peering.TrailEntry{entry})
	parsed := ParseTrail(rendered)
	if len(parsed) != 1 {
		t.Fatalf("expected 1 parsed entry, got %d (rendered=%q)", len(parsed), rendered)
	}
	got := parsed[0]
	if !got.At.Equal(entry.At) {
		t.Errorf("At: want %v, got %v", entry.At, got.At)
	}
	if got.Direction != entry.Direction {
		t.Errorf("Direction: want %q, got %q", entry.Direction, got.Direction)
	}
	if got.PeerAliasDisplay != entry.PeerAliasDisplay {
		t.Errorf("PeerAliasDisplay: want %q, got %q", entry.PeerAliasDisplay, got.PeerAliasDisplay)
	}
	if got.PeerID != entry.PeerID {
		t.Errorf("PeerID: want %q, got %q", entry.PeerID, got.PeerID)
	}
	if got.Mode != entry.Mode {
		t.Errorf("Mode: want %q, got %q", entry.Mode, got.Mode)
	}
	if got.OriginatingSpec != entry.OriginatingSpec {
		t.Errorf("OriginatingSpec: want %q, got %q", entry.OriginatingSpec, got.OriginatingSpec)
	}
	if got.PeerSpec != entry.PeerSpec {
		t.Errorf("PeerSpec: want %q, got %q", entry.PeerSpec, got.PeerSpec)
	}
	if got.AtCommit != entry.AtCommit {
		t.Errorf("AtCommit: want %q, got %q", entry.AtCommit, got.AtCommit)
	}
	if got.Reason != entry.Reason {
		t.Errorf("Reason: want %q, got %q", entry.Reason, got.Reason)
	}
}

// TestAppendTrailToContent inserts a trail entry into a spec body
// that has no prior trail section, then a second entry, and verifies
// chronological ordering.
func TestAppendTrailToContent(t *testing.T) {
	body := `---
title: T
type: feature
status: planning
---

# T

## Goal

Do the thing.
`

	first := peering.TrailEntry{
		At:               time.Date(2026, 5, 15, 14, 0, 0, 0, time.UTC),
		Direction:        peering.DirectionOut,
		PeerAliasDisplay: "app",
		PeerID:           "9c1c2f3e-4a8b-4f9d-9a0e-7e1f0d8c3a55",
		Mode:             peering.ModeAsyncDrop,
	}
	second := peering.TrailEntry{
		At:               time.Date(2026, 5, 15, 16, 23, 0, 0, time.UTC),
		Direction:        peering.DirectionIn,
		PeerAliasDisplay: "app",
		PeerID:           "9c1c2f3e-4a8b-4f9d-9a0e-7e1f0d8c3a55",
		Mode:             peering.ModeHandedBack,
		ResultRef:        "commit 4427cec",
	}

	withFirst := AppendTrailToContent(body, first)
	if !strings.Contains(withFirst, "## Handoff Trail") {
		t.Fatal("section header missing after first append")
	}
	withBoth := AppendTrailToContent(withFirst, second)
	entries := ParseTrail(withBoth)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if !entries[0].At.Before(entries[1].At) {
		t.Errorf("entries not chronological: %v then %v", entries[0].At, entries[1].At)
	}
	// Idempotence check: rendering the parsed list and parsing it
	// back yields the same count.
	again := RenderTrailSection(entries)
	if got := ParseTrail(again); len(got) != 2 {
		t.Errorf("round-trip count: got %d", len(got))
	}
}
