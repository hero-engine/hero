package data

import "testing"

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

func TestKindFromEventType(t *testing.T) {
	cases := map[string]string{
		"spec_created":         "spec",
		"spec_updated":         "spec",
		"delivery_complete":    "spec",
		"decision_made":        "knowledge",
		"blocker_hit":          "drift",
		"peer.handoff.sent":    "convention",
		"random_event":         "commit",
	}
	for in, want := range cases {
		if got := kindFromEventType(in); got != want {
			t.Errorf("kindFromEventType(%q) = %q, want %q", in, got, want)
		}
	}
}
