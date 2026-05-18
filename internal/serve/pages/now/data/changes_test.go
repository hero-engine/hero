package data

import (
	"testing"
	"time"
)

func TestLoadChanges_EmptyFallsBackToLimited(t *testing.T) {
	got := LoadChanges(ChangesInputs{})
	if !got.Limited {
		t.Errorf("expected Limited=true on empty input")
	}
}

func TestPrettyAgeShort(t *testing.T) {
	cases := map[string]string{
		"":              "",
		"5 minutes ago": "5m",
		"2 hours ago":   "2h",
		"3 days ago":    "3d",
		"1 week ago":    "1w",
		"2 months ago":  "2mo",
		"4 years ago":   "4y",
	}
	for in, want := range cases {
		if got := prettyAgeShort(in); got != want {
			t.Errorf("prettyAgeShort(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDedupeWithinWindow_PeerCallsCollapse(t *testing.T) {
	// 6 peer.call events within 30 minutes — should collapse to a
	// single row with Count=6. We mix invoked/completed to verify
	// the display-group mapping (both → "peer-call").
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	rows := []ChangeRow{}
	types := []string{"peer.call.invoked", "peer.call.completed", "peer.call.invoked", "peer.call.completed", "peer.call.invoked", "peer.call.completed"}
	for i, typ := range types {
		// Newest first: subtract i*5 minutes from now.
		rows = append(rows, ChangeRow{
			TimeAt:       now.Add(-time.Duration(i*5) * time.Minute),
			DisplayGroup: displayGroupFor(typ),
			GroupLabel:   groupLabelFor(typ),
			Count:        1,
		})
	}
	out := dedupeWithinWindow(rows, time.Hour)
	if len(out) != 1 {
		t.Fatalf("expected 1 collapsed row, got %d: %+v", len(out), out)
	}
	if out[0].Count != 6 {
		t.Errorf("expected Count=6, got %d", out[0].Count)
	}
	if len(out[0].CollapsedRows) != 5 {
		t.Errorf("expected 5 collapsed originals (kept row holds the newest), got %d", len(out[0].CollapsedRows))
	}
}

func TestDedupeWithinWindow_OutsideWindowDoesNotCollapse(t *testing.T) {
	// Two peer.call events 90 minutes apart — should NOT collapse.
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	rows := []ChangeRow{
		{TimeAt: now, DisplayGroup: "peer-call", Count: 1},
		{TimeAt: now.Add(-90 * time.Minute), DisplayGroup: "peer-call", Count: 1},
	}
	out := dedupeWithinWindow(rows, time.Hour)
	if len(out) != 2 {
		t.Errorf("expected 2 rows (no collapse outside window), got %d", len(out))
	}
	for _, r := range out {
		if r.Count != 1 {
			t.Errorf("expected Count=1, got %d", r.Count)
		}
	}
}

func TestDedupeWithinWindow_DifferentGroupsDoNotCollapse(t *testing.T) {
	// peer-call and decision events at the same time — should stay
	// as separate rows; groups are not combined.
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	rows := []ChangeRow{
		{TimeAt: now, DisplayGroup: "peer-call", Count: 1},
		{TimeAt: now.Add(-1 * time.Minute), DisplayGroup: "decision", Count: 1},
		{TimeAt: now.Add(-2 * time.Minute), DisplayGroup: "peer-call", Count: 1},
	}
	out := dedupeWithinWindow(rows, time.Hour)
	if len(out) != 3 {
		t.Errorf("expected 3 rows (different groups stay separate), got %d: %+v", len(out), out)
	}
}

func TestDisplayGroupFor(t *testing.T) {
	cases := map[string]string{
		"peer.call.invoked":     "peer-call",
		"peer.call.completed":   "peer-call",
		"peer.handoff.sent":     "peer-handoff",
		"spec_created":          "spec-change",
		"spec_updated":          "spec-change",
		"spec.status_changed":   "spec-change",
		"spec.complete":         "spec-complete",
		"spec.complete.shipped": "spec-complete",
		"claim_acquired":        "claim-acquired",
		"peer_id.minted":        "peer-id-minted",
		"handoff.dropped":       "handoff",
		"handoff.accepted":      "handoff",
		"check.passed":          "check",
		"check.failed":          "check",
		"commit":                "commit",
		"files_modified":        "commit",
		"decision_made":         "decision",
		"delivery_complete":     "delivery",
		"knowledge.captured":    "knowledge",
		"note.captured":         "knowledge",
		"random_event":          "",
	}
	for in, want := range cases {
		if got := displayGroupFor(in); got != want {
			t.Errorf("displayGroupFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGroupLabelFor_NewGroups(t *testing.T) {
	cases := map[string]string{
		"spec.complete":   "specs completed",
		"claim_acquired":  "claims acquired",
		"peer_id.minted":  "peer ids minted",
		"handoff.dropped": "handoffs",
		"check.passed":    "checks",
		"commit":          "commits",
	}
	for in, want := range cases {
		if got := groupLabelFor(in); got != want {
			t.Errorf("groupLabelFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDedupeWithinWindow_SpecCompleteCollapses(t *testing.T) {
	// Three spec.complete events within the dedup window — collapse to
	// a single row with Count=3.
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	rows := []ChangeRow{}
	for i := 0; i < 3; i++ {
		rows = append(rows, ChangeRow{
			TimeAt:       now.Add(-time.Duration(i*5) * time.Minute),
			DisplayGroup: displayGroupFor("spec.complete"),
			GroupLabel:   groupLabelFor("spec.complete"),
			Count:        1,
		})
	}
	out := dedupeWithinWindow(rows, time.Hour)
	if len(out) != 1 {
		t.Fatalf("expected 1 collapsed row, got %d", len(out))
	}
	if out[0].Count != 3 {
		t.Errorf("expected Count=3, got %d", out[0].Count)
	}
	if out[0].GroupLabel != "specs completed" {
		t.Errorf("expected GroupLabel=%q, got %q", "specs completed", out[0].GroupLabel)
	}
}

func TestDedupeWithinWindow_DistinctNewGroupsDoNotCollapse(t *testing.T) {
	// claim_acquired and peer_id.minted are different display groups —
	// they should not collapse together.
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	rows := []ChangeRow{
		{TimeAt: now, DisplayGroup: displayGroupFor("claim_acquired"), Count: 1},
		{TimeAt: now.Add(-1 * time.Minute), DisplayGroup: displayGroupFor("peer_id.minted"), Count: 1},
	}
	out := dedupeWithinWindow(rows, time.Hour)
	if len(out) != 2 {
		t.Errorf("expected 2 rows (different new groups stay separate), got %d", len(out))
	}
}

func TestKindFromEventType(t *testing.T) {
	cases := map[string]string{
		"spec_created":          "spec",
		"spec_updated":          "spec",
		"spec.status_changed":   "spec",
		"delivery_complete":     "check",
		// spec.complete falls through to the "pulse" default — it was
		// a draft verb that never landed (see polish-v2 Fix 4).
		"spec.complete":         "pulse",
		"decision_made":         "decision",
		"blocker_hit":           "drift",
		"peer.handoff.sent":     "handoff",
		"peer.call.invoked":     "handoff",
		"peer.call.completed":   "handoff",
		"knowledge.captured":    "knowledge",
		"note.captured":         "knowledge",
		"files_modified":        "commit",
		"commit":                "commit",
		"random_event":          "pulse",
	}
	for in, want := range cases {
		if got := kindFromEventType(in); got != want {
			t.Errorf("kindFromEventType(%q) = %q, want %q", in, got, want)
		}
	}
}
