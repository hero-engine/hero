package sync

import (
	"strings"
	"testing"
)

// The truth table below is the acceptance gate for the 3-way shared-field
// merge. For each shared field (title, body, tags) it covers all four cases:
// neither changed, only-local, only-remote, and both-changed — and asserts the
// core guarantee that a both-changed merge NEVER drops the remote value.

// --- title (KindTitle) -------------------------------------------------------

func TestMergeTitle_TruthTable(t *testing.T) {
	const base = "Fix login bug"

	// only-remote changed → take remote (never lose upstream), no push.
	t.Run("only_remote", func(t *testing.T) {
		r := MergeText(KindTitle, base, base, "Fix login bug UPSTREAM")
		if r.Merged != "Fix login bug UPSTREAM" {
			t.Fatalf("merged = %q, want remote value", r.Merged)
		}
		if r.PushLocal {
			t.Fatal("must not push local when only remote changed")
		}
	})

	// only-local changed → push local.
	t.Run("only_local", func(t *testing.T) {
		r := MergeText(KindTitle, base, "Fix login bug LOCAL", base)
		if r.Merged != "Fix login bug LOCAL" || !r.PushLocal {
			t.Fatalf("merged=%q push=%v, want local push", r.Merged, r.PushLocal)
		}
	})

	// neither / already converged → no-op.
	t.Run("converged", func(t *testing.T) {
		r := MergeText(KindTitle, base, base, base)
		if r.Merged != base || r.PushLocal {
			t.Fatalf("merged=%q push=%v, want no-op", r.Merged, r.PushLocal)
		}
	})

	// both changed → keep REMOTE (upstream is truth), never push local over
	// it, and record the dropped local title in a terse conflict note (for the
	// local-only sync_conflict field — NOT the body). This is the drift-test
	// guarantee.
	t.Run("both_changed_keeps_remote", func(t *testing.T) {
		local := "Fix login bug LOCAL"
		remote := "Fix login bug UPSTREAM"
		r := MergeText(KindTitle, base, local, remote)
		if r.Merged != remote {
			t.Fatalf("merged = %q, want remote %q (must not drop upstream)", r.Merged, remote)
		}
		if r.PushLocal {
			t.Fatal("must NOT push local title over a concurrent upstream edit")
		}
		if !strings.Contains(r.ConflictNote, local) || !strings.Contains(r.ConflictNote, remote) {
			t.Fatalf("conflict note %q must reference both local %q and remote %q", r.ConflictNote, local, remote)
		}
		// The note is a terse frontmatter record, not an HTML body marker.
		if strings.Contains(r.ConflictNote, LocalEditMarkerPrefix) {
			t.Fatalf("title conflict must not produce a body marker: %q", r.ConflictNote)
		}
	})
}

// --- body (KindBody) ---------------------------------------------------------

func TestMergeBody_TruthTable(t *testing.T) {
	const base = "line one\nline two\nline three"

	t.Run("only_remote", func(t *testing.T) {
		remote := "line one\nline two CHANGED\nline three"
		r := MergeText(KindBody, base, base, remote)
		if r.Merged != remote || r.PushLocal {
			t.Fatalf("merged=%q push=%v, want take-remote no-push", r.Merged, r.PushLocal)
		}
	})

	t.Run("only_local", func(t *testing.T) {
		local := "line one\nline two LOCAL\nline three"
		r := MergeText(KindBody, base, local, base)
		if r.Merged != local || !r.PushLocal {
			t.Fatalf("merged=%q push=%v, want push-local", r.Merged, r.PushLocal)
		}
	})

	t.Run("converged", func(t *testing.T) {
		r := MergeText(KindBody, base, base, base)
		if r.Merged != base || r.PushLocal {
			t.Fatalf("merged=%q push=%v, want no-op", r.Merged, r.PushLocal)
		}
	})

	// both changed, NON-overlapping hunks → clean diff3 merge; both edits
	// present.
	t.Run("both_changed_non_overlapping", func(t *testing.T) {
		local := "line one LOCAL\nline two\nline three"       // edits line 1
		remote := "line one\nline two\nline three REMOTE"     // edits line 3
		r := MergeText(KindBody, base, local, remote)
		if !strings.Contains(r.Merged, "line one LOCAL") {
			t.Errorf("lost local hunk: %q", r.Merged)
		}
		if !strings.Contains(r.Merged, "line three REMOTE") {
			t.Errorf("lost remote hunk (never lose upstream!): %q", r.Merged)
		}
	})

	// both changed, OVERLAPPING hunk → remote kept, local preserved in marker.
	t.Run("both_changed_overlapping_keeps_remote", func(t *testing.T) {
		local := "line one\nline two LOCAL\nline three"
		remote := "line one\nline two REMOTE\nline three"
		r := MergeText(KindBody, base, local, remote)
		if !strings.Contains(r.Merged, "line two REMOTE") {
			t.Fatalf("remote hunk dropped (never lose upstream!): %q", r.Merged)
		}
		if !strings.Contains(r.Merged, "line two LOCAL") {
			t.Fatalf("local hunk not preserved: %q", r.Merged)
		}
		if !strings.Contains(r.Merged, LocalEditMarkerPrefix) {
			t.Fatalf("local edit not wrapped in informational marker: %q", r.Merged)
		}
		// The marker is informational, never a blocking git-style marker.
		if strings.Contains(r.Merged, "<<<<<<<") || strings.Contains(r.Merged, ">>>>>>>") {
			t.Fatalf("must not emit blocking conflict markers: %q", r.Merged)
		}
	})
}

// --- tags (KindTags) ---------------------------------------------------------

func TestMergeTags_TruthTable(t *testing.T) {
	base := []string{"bug", "sync"}

	t.Run("only_remote", func(t *testing.T) {
		remote := []string{"bug", "sync", "upstream"}
		r := MergeTags(base, base, remote)
		if !tagsEqual(r.Merged, remote) || r.PushLocal {
			t.Fatalf("merged=%v push=%v, want take-remote", r.Merged, r.PushLocal)
		}
	})

	t.Run("only_local", func(t *testing.T) {
		local := []string{"bug", "sync", "local"}
		r := MergeTags(base, local, base)
		if !tagsEqual(r.Merged, local) || !r.PushLocal {
			t.Fatalf("merged=%v push=%v, want push-local", r.Merged, r.PushLocal)
		}
	})

	t.Run("converged", func(t *testing.T) {
		r := MergeTags(base, base, base)
		if !tagsEqual(r.Merged, base) || r.PushLocal {
			t.Fatalf("merged=%v push=%v, want no-op", r.Merged, r.PushLocal)
		}
	})

	// both changed → set-union; neither side's tag dropped, and every remote
	// tag survives.
	t.Run("both_changed_set_union", func(t *testing.T) {
		local := []string{"bug", "sync", "local-tag"}     // +local-tag
		remote := []string{"bug", "sync", "remote-tag"}   // +remote-tag
		r := MergeTags(base, local, remote)
		want := map[string]bool{"bug": true, "sync": true, "local-tag": true, "remote-tag": true}
		if len(r.Merged) != len(want) {
			t.Fatalf("merged=%v, want union of size %d", r.Merged, len(want))
		}
		for _, tag := range r.Merged {
			if !want[tag] {
				t.Errorf("unexpected tag %q in %v", tag, r.Merged)
			}
		}
		for _, remoteTag := range remote {
			if !contains(r.Merged, remoteTag) {
				t.Fatalf("dropped remote tag %q (never lose upstream!): %v", remoteTag, r.Merged)
			}
		}
	})

	// A tag removed by ONE side while the other keeps/adds → tag survives
	// (never drop a tag on a lone removal).
	t.Run("lone_removal_keeps_tag", func(t *testing.T) {
		local := []string{"bug"}                 // removed "sync"
		remote := []string{"bug", "sync", "new"} // kept "sync", added "new"
		r := MergeTags(base, local, remote)
		if !contains(r.Merged, "sync") {
			t.Fatalf("dropped tag on a lone removal: %v", r.Merged)
		}
		if !contains(r.Merged, "new") {
			t.Fatalf("dropped remote addition: %v", r.Merged)
		}
	})

	// A tag removed by BOTH sides → dropped.
	t.Run("both_removed_drops_tag", func(t *testing.T) {
		local := []string{"bug"}
		remote := []string{"bug"}
		r := MergeTags(base, local, remote)
		if contains(r.Merged, "sync") {
			t.Fatalf("kept a tag both sides removed: %v", r.Merged)
		}
	})
}

// TestMerge_Idempotent asserts a no-edit re-sync is a clean no-op: feeding the
// converged value back as base/local/remote never pushes.
func TestMerge_Idempotent(t *testing.T) {
	title := MergeText(KindTitle, "T", "T", "T")
	if title.PushLocal {
		t.Error("title re-sync pushed on no edit")
	}
	body := MergeText(KindBody, "B", "B", "B")
	if body.PushLocal {
		t.Error("body re-sync pushed on no edit")
	}
	tags := MergeTags([]string{"a"}, []string{"a"}, []string{"a"})
	if tags.PushLocal {
		t.Error("tags re-sync pushed on no edit")
	}
}

// TestSharedFields_MatchSwift documents the shared-field set matches
// FieldOwnership.swift: exactly title/body/tags, mapped to the Go canonical
// title/description/labels.
func TestSharedFields_MatchSwift(t *testing.T) {
	want := map[string]string{"title": "title", "body": "description", "tags": "labels"}
	if len(SharedFields) != len(want) {
		t.Fatalf("shared field count = %d, want %d", len(SharedFields), len(want))
	}
	for _, f := range SharedFields {
		canon, ok := want[f.BaselineKey]
		if !ok {
			t.Errorf("unexpected shared field %q", f.BaselineKey)
			continue
		}
		if f.Canonical != canon {
			t.Errorf("shared %q maps to canonical %q, want %q", f.BaselineKey, f.Canonical, canon)
		}
	}
	if _, ok := SharedByCanonical("priority"); ok {
		t.Error("priority must NOT be shared (tracker-owned)")
	}
	if _, ok := SharedByCanonical("status"); ok {
		t.Error("status must NOT be shared (hero-owned)")
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
