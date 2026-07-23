package suggestion

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
)

type testResolver struct {
	ref  *attention.ProjectReference
	path string
}

func (r testResolver) ResolveReference(ref *attention.ProjectReference) focus.ResolvedProject {
	if ref != nil && r.ref != nil && ref.PeerID == r.ref.PeerID {
		return focus.ResolvedProject{Reference: ref, Availability: focus.ProjectAvailable, Path: r.path}
	}
	return focus.ResolvedProject{Reference: ref, Availability: focus.ProjectMissing}
}
func (testResolver) ResolveInput(string) (*attention.ProjectReference, error) { return nil, nil }
func (testResolver) ResolveCurrent() (*attention.ProjectReference, error)     { return nil, nil }

type testRig struct {
	service    *Service
	store      *Store
	focusStore *focus.Store
	now        time.Time
}

func newTestRig(t *testing.T, resolver focus.ProjectResolver) *testRig {
	t.Helper()
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	focusStore, err := focus.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	focusService := focus.NewService(focusStore, resolver)
	rig := &testRig{store: store, focusStore: focusStore, now: time.Date(2026, 7, 22, 18, 0, 0, 0, time.UTC)}
	rig.service = NewService(store, focusService, resolver)
	rig.service.now = func() time.Time { return rig.now }
	n := 0
	rig.service.newID = func() (string, error) { n++; return "suggestion_test" + string(rune('0'+n)), nil }
	return rig
}

func proposal(ref *attention.ProjectReference) CreateRequest {
	return CreateRequest{Kind: "follow_up", Title: "Investigate cache", Reason: "A separate optimization surfaced", Prompt: "Measure cache misses and design a bounded fix.\n", Project: ref, Provenance: &attention.ProvenanceReference{Kind: "run", SourceID: "run_1"}, IdempotencyKey: "run_1:cache"}
}

func TestProposalPersistsPrivatelyWithoutCreatingFocusAndReplays(t *testing.T) {
	rig := newTestRig(t, testResolver{})
	first, err := rig.service.Create(proposal(nil))
	if err != nil {
		t.Fatal(err)
	}
	if first.State != StatePending || len(first.Actions) != 4 {
		t.Fatalf("proposal = %#v", first)
	}
	items, _ := rig.focusStore.List()
	if len(items) != 0 {
		t.Fatalf("proposal created Focus: %#v", items)
	}
	replay, err := rig.service.Create(proposal(nil))
	if err != nil || replay.ID != first.ID {
		t.Fatalf("replay = %#v, %v", replay, err)
	}
	changed := proposal(nil)
	changed.Prompt = "different"
	if _, err := rig.service.Create(changed); err == nil {
		t.Fatal("conflicting replay accepted")
	}
	info, err := os.Stat(filepath.Join(filepath.Dir(rig.focusStorePath()), "suggestions", first.ID+".json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("private file = %v, %v", info, err)
	}
}

func (r *testRig) focusStorePath() string {
	// Both stores share <root>/focus; infer it from the suggestion directory.
	return filepath.Join(filepath.Dir(r.store.dir), "items")
}

func TestAcceptTodayLaterDoNextAndDismiss(t *testing.T) {
	ref := &attention.ProjectReference{PeerID: "peer-demo", RegistrySlug: "demo", DisplayName: "Demo"}
	resolver := testResolver{ref: ref, path: filepath.Join(t.TempDir(), "demo")}
	for _, tc := range []struct{ decision, lifecycle string }{{DecisionToday, attention.FocusToday}, {DecisionLater, attention.FocusLater}, {DecisionDoNext, attention.FocusToday}} {
		t.Run(tc.decision, func(t *testing.T) {
			rig := newTestRig(t, resolver)
			created, _ := rig.service.Create(proposal(ref))
			result, err := rig.service.Act(created.ID, tc.decision, created.Revision, "action-1")
			if err != nil {
				t.Fatal(err)
			}
			if result.Suggestion.State != StateAccepted || result.Focus == nil || result.Focus.Lifecycle != tc.lifecycle {
				t.Fatalf("result = %#v", result)
			}
			if tc.decision == DecisionDoNext && (result.Launch == nil || result.Launch.Prompt != created.Prompt || result.Launch.Path != resolver.path) {
				t.Fatalf("launch = %#v", result.Launch)
			}
			replay, err := rig.service.Act(created.ID, tc.decision, created.Revision, "action-1")
			if err != nil || replay.Focus.ID != result.Focus.ID {
				t.Fatalf("replay = %#v, %v", replay, err)
			}
			focusItems, _ := rig.focusStore.List()
			if len(focusItems) != 1 {
				t.Fatalf("Focus count = %d", len(focusItems))
			}
		})
	}
	t.Run("dismiss", func(t *testing.T) {
		rig := newTestRig(t, resolver)
		created, _ := rig.service.Create(proposal(ref))
		result, err := rig.service.Act(created.ID, DecisionDismiss, created.Revision, "dismiss-1")
		if err != nil || result.Suggestion.State != StateDismissed || result.Focus != nil {
			t.Fatalf("dismiss = %#v, %v", result, err)
		}
		focusItems, _ := rig.focusStore.List()
		if len(focusItems) != 0 {
			t.Fatalf("dismiss created Focus: %#v", focusItems)
		}
	})
}

func TestActionErrorsMakeNoCommitment(t *testing.T) {
	ref := &attention.ProjectReference{PeerID: "missing", DisplayName: "Missing"}
	rig := newTestRig(t, testResolver{})
	created, _ := rig.service.Create(proposal(ref))
	tests := []struct {
		name, id, decision, key string
		revision                int64
	}{
		{"missing", "suggestion_absent", DecisionToday, "a", created.Revision},
		{"stale", created.ID, DecisionToday, "b", created.Revision + 1},
		{"unsupported", created.ID, "tomorrow", "c", created.Revision},
		{"invalid", created.ID, DecisionToday, "", created.Revision},
		{"missing project", created.ID, DecisionDoNext, "d", created.Revision},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := rig.service.Act(tc.id, tc.decision, tc.revision, tc.key); err == nil {
				t.Fatal("action succeeded")
			}
			focusItems, _ := rig.focusStore.List()
			if len(focusItems) != 0 {
				t.Fatalf("invalid action created Focus: %#v", focusItems)
			}
		})
	}
	rig.now = rig.now.Add(8 * 24 * time.Hour)
	if _, err := rig.service.Act(created.ID, DecisionToday, created.Revision, "expired"); err == nil {
		t.Fatal("expired action succeeded")
	}
	focusItems, _ := rig.focusStore.List()
	if len(focusItems) != 0 {
		t.Fatalf("expired action created Focus: %#v", focusItems)
	}
}

func TestExpiryAndThirtyDayCleanup(t *testing.T) {
	rig := newTestRig(t, testResolver{})
	created, _ := rig.service.Create(proposal(nil))
	rig.now = rig.now.Add(8 * 24 * time.Hour)
	all, err := rig.service.List(false)
	if err != nil || len(all) != 1 || all[0].State != StateExpired {
		t.Fatalf("expired = %#v, %v", all, err)
	}
	pending, _ := rig.service.List(true)
	if len(pending) != 0 {
		t.Fatalf("pending = %#v", pending)
	}
	if _, err := rig.service.Get(created.ID); err != nil {
		t.Fatalf("expired not inspectable: %v", err)
	}
	rig.now = rig.now.Add(23 * 24 * time.Hour)
	all, err = rig.service.List(false)
	if err != nil || len(all) != 0 {
		t.Fatalf("cleanup = %#v, %v", all, err)
	}
}

func TestReceiptWriteFailureRecoversIdempotently(t *testing.T) {
	ref := &attention.ProjectReference{PeerID: "peer-demo", DisplayName: "Demo"}
	rig := newTestRig(t, testResolver{ref: ref, path: t.TempDir()})
	created, _ := rig.service.Create(proposal(ref))
	failOnce := true
	rig.store.beforeWrite = func(item Item) error {
		if item.State == StateAccepted && failOnce {
			failOnce = false
			return errors.New("injected receipt failure")
		}
		return nil
	}
	if _, err := rig.service.Act(created.ID, DecisionToday, created.Revision, "accept-1"); err == nil {
		t.Fatal("injected failure did not fire")
	}
	focusItems, _ := rig.focusStore.List()
	if len(focusItems) != 1 {
		t.Fatalf("Focus after failure = %#v", focusItems)
	}
	result, err := rig.service.Act(created.ID, DecisionToday, created.Revision, "accept-1")
	if err != nil {
		t.Fatal(err)
	}
	focusItems, _ = rig.focusStore.List()
	if len(focusItems) != 1 || result.Focus == nil || result.Focus.ID != focusItems[0].ID {
		t.Fatalf("recovery = %#v, Focus %#v", result, focusItems)
	}
}
