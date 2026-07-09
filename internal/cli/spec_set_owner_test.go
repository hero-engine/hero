package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

// resetSetOwnerFlags restores set-owner flags to defaults between cases.
func resetSetOwnerFlags() {
	setOwnerFrom = ""
	setOwnerTrackerPush = false
	setOwnerJSON = false
}

func TestSetOwnerSynthesizesHistory(t *testing.T) {
	env := newTestEnv(t)
	resetSetOwnerFlags()

	// Spec with owner but NO owner_history block.
	env.addSpec("planning/features/my-story/spec.md", `---
title: My Story
type: feature
status: delivering
owner: pm
domain: pm
---

# My Story

Body.
`)

	err := runSpecSetOwner(specSetOwnerCmd, []string{"my-story", "engineering"})
	if err != nil {
		t.Fatalf("set-owner: %v", err)
	}

	got := readSpec(t, env, "planning/features/my-story/spec.md")
	if got.Owner != "engineering" {
		t.Errorf("owner = %q, want engineering", got.Owner)
	}
	if len(got.OwnerHistory) != 2 {
		t.Fatalf("history len = %d, want 2 (synthesized pm + appended engineering)", len(got.OwnerHistory))
	}
	if got.OwnerHistory[0].Owner != "pm" || got.OwnerHistory[0].To == nil {
		t.Errorf("synthesized entry = %+v, want closed pm", got.OwnerHistory[0])
	}
	if got.OwnerHistory[1].Owner != "engineering" || got.OwnerHistory[1].To != nil {
		t.Errorf("appended entry = %+v, want active engineering", got.OwnerHistory[1])
	}
}

func TestSetOwnerAppendsToExisting(t *testing.T) {
	env := newTestEnv(t)
	resetSetOwnerFlags()

	env.addSpec("planning/features/with-history/spec.md", `---
title: With History
type: feature
status: delivering
owner: pm
owner_history:
  - owner: pm
    from: 2026-05-20T14:33:00Z
    to: null
---

# With History
`)

	err := runSpecSetOwner(specSetOwnerCmd, []string{"with-history", "qa"})
	if err != nil {
		t.Fatalf("set-owner: %v", err)
	}

	got := readSpec(t, env, "planning/features/with-history/spec.md")
	if len(got.OwnerHistory) != 2 {
		t.Fatalf("history len = %d, want 2", len(got.OwnerHistory))
	}
	// Prior entry closed (to no longer nil).
	if got.OwnerHistory[0].To == nil {
		t.Errorf("prior entry not closed: %+v", got.OwnerHistory[0])
	}
	if got.OwnerHistory[1].Owner != "qa" || got.OwnerHistory[1].To != nil {
		t.Errorf("appended entry = %+v, want active qa", got.OwnerHistory[1])
	}
	if spec.ActiveOwner(got.OwnerHistory) != "qa" {
		t.Errorf("active owner = %q, want qa", spec.ActiveOwner(got.OwnerHistory))
	}
}

func TestSetOwnerRejectsInvalidOwner(t *testing.T) {
	env := newTestEnv(t)
	resetSetOwnerFlags()

	env.addSpec("planning/features/bad/spec.md", `---
title: Bad
type: feature
owner: pm
---

# Bad
`)

	err := runSpecSetOwner(specSetOwnerCmd, []string{"bad", "product"})
	if err == nil {
		t.Fatal("expected an error for invalid owner, got nil")
	}
	var ioe *invalidOwnerError
	if !asInvalidOwner(err, &ioe) {
		t.Fatalf("error %v is not *invalidOwnerError", err)
	}
	// Spec untouched: owner stays pm, no history added.
	got := readSpec(t, env, "planning/features/bad/spec.md")
	if got.Owner != "pm" {
		t.Errorf("owner changed to %q on invalid flip; want unchanged pm", got.Owner)
	}
	if len(got.OwnerHistory) != 0 {
		t.Errorf("history written on invalid flip: %+v", got.OwnerHistory)
	}
}

func TestSetOwnerTrackerPushSkippedByDefault(t *testing.T) {
	env := newTestEnv(t)
	resetSetOwnerFlags()
	setOwnerJSON = true

	// Record any sync push call.
	var called bool
	prev := ownerSyncPusher
	ownerSyncPusher = func(slug, newOwner string) ([]string, error) {
		called = true
		return []string{"owner"}, nil
	}
	defer func() { ownerSyncPusher = prev }()

	env.addSpec("planning/features/no-push/spec.md", `---
title: No Push
type: feature
owner: pm
tracker_id: PROJ-1
---

# No Push
`)

	out := captureStdout(func() {
		if err := runSpecSetOwner(specSetOwnerCmd, []string{"no-push", "engineering"}); err != nil {
			t.Fatalf("set-owner: %v", err)
		}
	})

	// Without --tracker-push the sync invocation must not fire.
	if called {
		t.Error("ownerSyncPusher fired without --tracker-push")
	}
	env.expectEnvelope(t, out, "pushed", nil, []string{"owner"})
}

func TestSetOwnerTrackerPushInvokesSync(t *testing.T) {
	env := newTestEnv(t)
	resetSetOwnerFlags()
	setOwnerTrackerPush = true
	setOwnerJSON = true

	// Promote owner to a content field so the push path is reached,
	// then restore so other tests see the default org-state default.
	pushFields = append(pushFields, ClassifiedField{Name: "owner", Class: classContent, Hint: "string"})
	defer func() {
		pushFields = pushFields[:len(pushFields)-1]
	}()

	var gotSlug, gotOwner string
	prev := ownerSyncPusher
	ownerSyncPusher = func(slug, newOwner string) ([]string, error) {
		gotSlug, gotOwner = slug, newOwner
		return []string{"owner"}, nil
	}
	defer func() { ownerSyncPusher = prev }()

	env.addSpec("planning/features/pushy/spec.md", `---
title: Pushy
type: feature
owner: pm
tracker_id: PROJ-7
---

# Pushy
`)

	out := captureStdout(func() {
		if err := runSpecSetOwner(specSetOwnerCmd, []string{"pushy", "engineering"}); err != nil {
			t.Fatalf("set-owner: %v", err)
		}
	})

	if gotSlug != "pushy" || gotOwner != "engineering" {
		t.Errorf("sync invoked with (%q, %q), want (pushy, engineering)", gotSlug, gotOwner)
	}
	env.expectEnvelope(t, out, "pushed", []string{"owner"}, nil)
}

func TestSetOwnerTrackerPushSkippedWhenOrgState(t *testing.T) {
	env := newTestEnv(t)
	resetSetOwnerFlags()
	setOwnerTrackerPush = true
	setOwnerJSON = true

	// owner is NOT a content field by default → push skipped with a note.
	var called bool
	prev := ownerSyncPusher
	ownerSyncPusher = func(slug, newOwner string) ([]string, error) {
		called = true
		return nil, nil
	}
	defer func() { ownerSyncPusher = prev }()

	env.addSpec("planning/features/orgstate/spec.md", `---
title: Org State
type: feature
owner: pm
tracker_id: PROJ-9
---

# Org State
`)

	out := captureStdout(func() {
		if err := runSpecSetOwner(specSetOwnerCmd, []string{"orgstate", "engineering"}); err != nil {
			t.Fatalf("set-owner: %v", err)
		}
	})

	if called {
		t.Error("sync push fired even though owner is org-state")
	}
	env.expectEnvelope(t, out, "pushed", nil, []string{"owner"})
}

func TestSetOwnerFromGuard(t *testing.T) {
	env := newTestEnv(t)
	resetSetOwnerFlags()
	setOwnerFrom = "qa" // wrong — actual owner is pm

	env.addSpec("planning/features/guarded/spec.md", `---
title: Guarded
type: feature
owner: pm
---

# Guarded
`)

	err := runSpecSetOwner(specSetOwnerCmd, []string{"guarded", "engineering"})
	if err == nil || !strings.Contains(err.Error(), "does not match current owner") {
		t.Fatalf("expected --from mismatch error, got %v", err)
	}
}

// ---- helpers ----------------------------------------------------------

func readSpec(t *testing.T, env *testEnv, rel string) *spec.Spec {
	t.Helper()
	path := env.heroDir + "/" + rel
	s, err := spec.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile %s: %v", path, err)
	}
	return s
}

func asInvalidOwner(err error, target **invalidOwnerError) bool {
	for err != nil {
		if ioe, ok := err.(*invalidOwnerError); ok {
			*target = ioe
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func (e *testEnv) expectEnvelope(t *testing.T, out, wantStatus string, wantPushed, wantSkipped []string) {
	t.Helper()
	line := strings.TrimSpace(out)
	var env ownerFlipEnvelope
	if err := json.Unmarshal([]byte(line), &env); err != nil {
		t.Fatalf("envelope is not valid JSON: %v\n%s", err, line)
	}
	if env.Status != wantStatus {
		t.Errorf("status = %q, want %q", env.Status, wantStatus)
	}
	if !sameStrings(env.PushedFields, wantPushed) {
		t.Errorf("pushed_fields = %v, want %v", env.PushedFields, wantPushed)
	}
	if !sameStrings(env.SkippedFields, wantSkipped) {
		t.Errorf("skipped_fields = %v, want %v", env.SkippedFields, wantSkipped)
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
