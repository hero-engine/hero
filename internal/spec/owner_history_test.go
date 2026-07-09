package spec

import (
	"strings"
	"testing"
	"time"
)

func mustTime(t *testing.T, v string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, v)
	if err != nil {
		t.Fatalf("parsing time %q: %v", v, err)
	}
	return parsed
}

func TestIsValidOwner(t *testing.T) {
	cases := []struct {
		owner string
		want  bool
	}{
		{"pm", true},
		{"engineering", true},
		{"qa", true},
		{"devops", true},
		{"design", true},
		{"docs", true},
		{"", false},
		{"Engineering", false}, // case-sensitive
		{"product", false},
		{"eng", false},
	}
	for _, c := range cases {
		if got := IsValidOwner(c.owner); got != c.want {
			t.Errorf("IsValidOwner(%q) = %v, want %v", c.owner, got, c.want)
		}
	}
}

func TestSynthesizeHistory(t *testing.T) {
	mtime := mustTime(t, "2026-05-20T14:33:00Z")

	cases := []struct {
		name       string
		owner      string
		wantLen    int
		wantOwner  string
		wantActive bool
	}{
		{"synthesizes from owner", "pm", 1, "pm", true},
		{"empty owner yields nil", "", 0, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SynthesizeHistory(c.owner, mtime)
			if len(got) != c.wantLen {
				t.Fatalf("len = %d, want %d", len(got), c.wantLen)
			}
			if c.wantLen == 0 {
				return
			}
			e := got[0]
			if e.Owner != c.wantOwner {
				t.Errorf("owner = %q, want %q", e.Owner, c.wantOwner)
			}
			if e.From == nil || !e.From.Equal(mtime) {
				t.Errorf("from = %v, want %v (file mtime)", e.From, mtime)
			}
			if (e.To == nil) != c.wantActive {
				t.Errorf("active(to==nil) = %v, want %v", e.To == nil, c.wantActive)
			}
		})
	}
}

func TestAppendOwnerFlip(t *testing.T) {
	from := mustTime(t, "2026-05-20T14:33:00Z")
	at := mustTime(t, "2026-05-24T09:12:00Z")

	t.Run("closes active entry and appends", func(t *testing.T) {
		history := []OwnerHistoryEntry{{Owner: "pm", From: &from, To: nil}}
		got := AppendOwnerFlip(history, "engineering", at)
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		// Prior entry closed.
		if got[0].To == nil || !got[0].To.Equal(at) {
			t.Errorf("prior entry to = %v, want %v", got[0].To, at)
		}
		// New active entry.
		if got[1].Owner != "engineering" {
			t.Errorf("new owner = %q, want engineering", got[1].Owner)
		}
		if got[1].From == nil || !got[1].From.Equal(at) {
			t.Errorf("new from = %v, want %v", got[1].From, at)
		}
		if got[1].To != nil {
			t.Errorf("new entry should be active (to == nil), got %v", got[1].To)
		}
	})

	t.Run("does not mutate input", func(t *testing.T) {
		history := []OwnerHistoryEntry{{Owner: "pm", From: &from, To: nil}}
		_ = AppendOwnerFlip(history, "engineering", at)
		if history[0].To != nil {
			t.Errorf("input was mutated: to = %v", history[0].To)
		}
	})

	t.Run("empty history appends single active entry", func(t *testing.T) {
		got := AppendOwnerFlip(nil, "qa", at)
		if len(got) != 1 || got[0].Owner != "qa" || got[0].To != nil {
			t.Errorf("got %+v, want single active qa entry", got)
		}
	})
}

func TestActiveOwner(t *testing.T) {
	from := mustTime(t, "2026-05-20T14:33:00Z")
	to := mustTime(t, "2026-05-24T09:12:00Z")
	history := []OwnerHistoryEntry{
		{Owner: "pm", From: &from, To: &to},
		{Owner: "engineering", From: &to, To: nil},
	}
	if got := ActiveOwner(history); got != "engineering" {
		t.Errorf("ActiveOwner = %q, want engineering", got)
	}
	if got := ActiveOwner(nil); got != "" {
		t.Errorf("ActiveOwner(nil) = %q, want empty", got)
	}
}

func TestOwnerHistoryRoundTrip(t *testing.T) {
	content := `---
title: Demo
owner: engineering
owner_history:
  - owner: pm
    from: 2026-05-20T14:33:00Z
    to: 2026-05-24T09:12:00Z
    note: initial scoping
  - owner: engineering
    from: 2026-05-24T09:12:00Z
    to: null
domain: pm
---

# Demo

Body text.
`

	s, err := Parse(content, "/x/demo/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.Owner != "engineering" {
		t.Fatalf("owner = %q, want engineering", s.Owner)
	}
	if len(s.OwnerHistory) != 2 {
		t.Fatalf("history len = %d, want 2", len(s.OwnerHistory))
	}

	// First entry: closed, with a preserved unknown field.
	first := s.OwnerHistory[0]
	if first.Owner != "pm" || first.To == nil {
		t.Errorf("first entry = %+v, want closed pm", first)
	}
	if len(first.Extra) != 1 || first.Extra[0].Key != "note" || first.Extra[0].Value != "initial scoping" {
		t.Errorf("first.Extra = %+v, want preserved note", first.Extra)
	}

	// Second entry: active.
	if s.OwnerHistory[1].To != nil {
		t.Errorf("second entry should be active, got to=%v", s.OwnerHistory[1].To)
	}

	// Round-trip: rewrite the block and re-parse — unknown field survives.
	rewritten := SetOwnerHistoryBlock(content, s.OwnerHistory)
	s2, err := Parse(rewritten, "/x/demo/spec.md", time.Now())
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if len(s2.OwnerHistory) != 2 {
		t.Fatalf("re-parsed history len = %d, want 2", len(s2.OwnerHistory))
	}
	if len(s2.OwnerHistory[0].Extra) != 1 || s2.OwnerHistory[0].Extra[0].Value != "initial scoping" {
		t.Errorf("unknown field lost on round-trip: %+v", s2.OwnerHistory[0].Extra)
	}
	// The body and other frontmatter keys survive.
	if !strings.Contains(rewritten, "domain: pm") || !strings.Contains(rewritten, "Body text.") {
		t.Errorf("round-trip dropped unrelated content:\n%s", rewritten)
	}
}

func TestSetOwnerHistoryBlockInsert(t *testing.T) {
	// A spec with owner but no owner_history block yet.
	content := `---
title: Demo
owner: pm
---

# Demo
`
	from := mustTime(t, "2026-05-20T14:33:00Z")
	history := []OwnerHistoryEntry{{Owner: "pm", From: &from, To: nil}}

	got := SetOwnerHistoryBlock(content, history)
	s, err := Parse(got, "/x/demo/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(s.OwnerHistory) != 1 || s.OwnerHistory[0].Owner != "pm" {
		t.Fatalf("inserted history = %+v, want single pm entry", s.OwnerHistory)
	}
	if s.Title != "Demo" || s.Owner != "pm" {
		t.Errorf("insert clobbered other fields: title=%q owner=%q", s.Title, s.Owner)
	}
}
