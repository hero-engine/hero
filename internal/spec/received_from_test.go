package spec

import (
	"testing"
	"time"
)

// TestReceivedFromBlockParse confirms the received_from frontmatter
// block parses cleanly into a ReceivedFromBlock.
func TestReceivedFromBlockParse(t *testing.T) {
	content := `---
title: "Error envelope mismatch"
type: bug
status: planning
received_from:
  peer_id: 9c1c2f3e-4a8b-4f9d-9a0e-7e1f0d8c3a55
  peer_alias_display: client
  originator_slug: order-failure-error-display
  handed_off_at: 2026-05-15T14:00:00Z
  at_commit: 3176736
  reason: "Symptom is in the client, root cause is the API response shape."
---

# Error envelope mismatch

Body.
`
	s, err := Parse(content, "/tmp/x.md", time.Now())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.ReceivedFrom == nil {
		t.Fatal("ReceivedFrom is nil")
	}
	rf := s.ReceivedFrom
	if rf.PeerID != "9c1c2f3e-4a8b-4f9d-9a0e-7e1f0d8c3a55" {
		t.Errorf("peer_id = %q", rf.PeerID)
	}
	if rf.PeerAliasDisplay != "client" {
		t.Errorf("peer_alias_display = %q", rf.PeerAliasDisplay)
	}
	if rf.OriginatorSlug != "order-failure-error-display" {
		t.Errorf("originator_slug = %q", rf.OriginatorSlug)
	}
	if rf.AtCommit != "3176736" {
		t.Errorf("at_commit = %q", rf.AtCommit)
	}
	if rf.Reason == "" {
		t.Errorf("reason missing")
	}
	want := time.Date(2026, 5, 15, 14, 0, 0, 0, time.UTC)
	if !rf.HandedOffAt.Equal(want) {
		t.Errorf("handed_off_at = %v, want %v", rf.HandedOffAt, want)
	}
	// Status should not be clobbered by the received_from block.
	if s.Status != StatusPlanning {
		t.Errorf("status = %q", s.Status)
	}
	if s.Title == "" {
		t.Errorf("title missing")
	}
}

// TestHandoffStatusEnumValues confirms the three new statuses
// roundtrip through Parse without being misinterpreted.
func TestHandoffStatusEnumValues(t *testing.T) {
	for _, st := range []Status{StatusHandedOff, StatusAwaitingPeer, StatusHandedBack} {
		content := "---\nstatus: " + string(st) + "\n---\n\n# T\n"
		s, err := Parse(content, "/tmp/x.md", time.Now())
		if err != nil {
			t.Fatalf("Parse %s: %v", st, err)
		}
		if s.Status != st {
			t.Errorf("status round-trip: want %q, got %q", st, s.Status)
		}
		if !s.IsInFlight() {
			t.Errorf("status %q should be IsInFlight", st)
		}
	}
	// IsHandoffPending: handed_off + awaiting_peer only.
	for _, st := range []Status{StatusHandedOff, StatusAwaitingPeer} {
		s := &Spec{Status: st}
		if !s.IsHandoffPending() {
			t.Errorf("%q should be IsHandoffPending", st)
		}
		if s.IsLocallyDelivering() {
			t.Errorf("%q should not be IsLocallyDelivering", st)
		}
	}
	// handed_back is NOT handoff-pending — it's back on this side.
	if (&Spec{Status: StatusHandedBack}).IsHandoffPending() {
		t.Error("handed_back should not be IsHandoffPending")
	}
}
