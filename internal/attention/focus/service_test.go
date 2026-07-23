package focus

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
)

type fakeResolver struct{ resolved map[string]ResolvedProject }

func (f fakeResolver) ResolveReference(ref *attention.ProjectReference) ResolvedProject {
	if ref == nil {
		return ResolvedProject{Availability: ProjectAvailable}
	}
	if got, ok := f.resolved[ref.PeerID]; ok {
		return got
	}
	return ResolvedProject{Reference: ref, Availability: ProjectMissing}
}
func (fakeResolver) ResolveInput(string) (*attention.ProjectReference, error) { return nil, nil }
func (fakeResolver) ResolveCurrent() (*attention.ProjectReference, error)     { return nil, nil }

func testService(t *testing.T, resolver ProjectResolver) *Service {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(store, resolver)
	now := time.Date(2026, 7, 22, 18, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	n := 0
	s.newID = func() (string, error) { n++; return "focus_test" + string(rune('0'+n)), nil }
	return s
}

func TestServiceLifecycleReplayAndListing(t *testing.T) {
	s := testService(t, fakeResolver{})
	first, err := s.Create(CreateRequest{Title: "One", Prompt: " exact prompt \n", Lifecycle: attention.FocusInbox})
	if err != nil {
		t.Fatal(err)
	}
	if first.Prompt != " exact prompt \n" || first.CreatedAt != first.UpdatedAt {
		t.Fatalf("first = %#v", first)
	}
	s.now = func() time.Time { return time.Date(2026, 7, 22, 19, 0, 0, 0, time.UTC) }
	moved, err := s.Move(first.ID, attention.FocusToday, first.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if moved.Revision == first.Revision || moved.UpdatedAt == first.UpdatedAt {
		t.Fatalf("moved = %#v", moved)
	}
	replayed, err := s.Move(first.ID, attention.FocusToday, moved.Revision)
	if err != nil || replayed.Revision != moved.Revision || replayed.UpdatedAt != moved.UpdatedAt {
		t.Fatalf("replay = %#v, %v", replayed, err)
	}
	if _, err := s.Move(first.ID, attention.FocusDone, first.Revision); !errors.Is(err, ErrStale) {
		t.Fatalf("stale err = %v", err)
	}
	done, err := s.Move(first.ID, attention.FocusDone, moved.Revision)
	if err != nil {
		t.Fatal(err)
	}
	active, _ := s.List("")
	if len(active) != 0 {
		t.Fatalf("default list = %#v", active)
	}
	doneItems, _ := s.List(attention.FocusDone)
	if len(doneItems) != 1 {
		t.Fatalf("done list = %#v", doneItems)
	}
	reopened, err := s.Move(first.ID, attention.FocusLater, done.Revision)
	if err != nil || reopened.Lifecycle != attention.FocusLater {
		t.Fatalf("reopen = %#v, %v", reopened, err)
	}
}

func TestServiceCreateOrGetRequiresSourceAndIsExact(t *testing.T) {
	s := testService(t, fakeResolver{})
	req := CreateRequest{Title: "From mail", Prompt: "Pick this up", Lifecycle: attention.FocusToday, Origin: &attention.ProvenanceReference{Kind: "mail", SourceID: "mail_1"}, OriginKey: "mail_1:today"}
	first, created, err := s.CreateOrGet(req)
	if err != nil || !created {
		t.Fatalf("first = %#v, %v, %v", first, created, err)
	}
	replay, created, err := s.CreateOrGet(req)
	if err != nil || created || replay.ID != first.ID {
		t.Fatalf("replay = %#v, %v, %v", replay, created, err)
	}
	req.Title = "Changed"
	if _, _, err := s.CreateOrGet(req); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("conflict = %v", err)
	}
	if _, _, err := s.CreateOrGet(CreateRequest{Title: "x", Prompt: "y", OriginKey: "key"}); err == nil {
		t.Fatal("missing provenance accepted")
	}
}

func TestServiceRejectsInvalidBoundaryValues(t *testing.T) {
	s := testService(t, fakeResolver{})
	if _, err := s.Create(CreateRequest{Title: "", Prompt: "valid"}); err == nil {
		t.Fatal("empty title accepted")
	}
	if _, err := s.Create(CreateRequest{Title: "valid", Prompt: ""}); err == nil {
		t.Fatal("empty prompt accepted")
	}
	if _, err := s.Create(CreateRequest{Title: "valid", Prompt: "valid", Project: &attention.ProjectReference{DisplayName: "No ID"}}); err == nil {
		t.Fatal("invalid project accepted")
	}
	if _, err := s.Move("focus_test", attention.FocusToday, 0); err == nil {
		t.Fatal("zero revision accepted")
	}
}

func TestServiceLaunchIntentRequiresResolvedProjectAndDoesNotMutate(t *testing.T) {
	ref := &attention.ProjectReference{PeerID: "peer-1", RegistrySlug: "demo", DisplayName: "Demo"}
	target := filepath.Join(t.TempDir(), "demo")
	resolver := fakeResolver{resolved: map[string]ResolvedProject{"peer-1": {Reference: ref, Availability: ProjectAvailable, Path: target}}}
	s := testService(t, resolver)
	item, err := s.Create(CreateRequest{Title: "Launch", Prompt: "Do precisely this\n", Project: ref})
	if err != nil {
		t.Fatal(err)
	}
	intent, err := s.LaunchIntent(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if intent.Prompt != item.Prompt || intent.Path != target || intent.Revision != item.Revision {
		t.Fatalf("intent = %#v", intent)
	}
	after, _ := s.Get(item.ID)
	if after.Revision != item.Revision || after.Lifecycle != item.Lifecycle {
		t.Fatalf("launch mutated item: %#v", after)
	}

	missing := testService(t, fakeResolver{})
	missingItem, _ := missing.Create(CreateRequest{Title: "Missing", Prompt: "Do not fall back", Project: ref})
	shown, _ := missing.Get(missingItem.ID)
	if shown.Availability != ProjectMissing {
		t.Fatalf("availability = %q", shown.Availability)
	}
	var missingErr *MissingProjectError
	if _, err := missing.LaunchIntent(missingItem.ID); !errors.As(err, &missingErr) {
		t.Fatalf("launch err = %v", err)
	}

	unbound := testService(t, fakeResolver{})
	unboundItem, _ := unbound.Create(CreateRequest{Title: "Unbound", Prompt: "Choose later"})
	if _, err := unbound.LaunchIntent(unboundItem.ID); !errors.As(err, &missingErr) {
		t.Fatalf("unbound launch err = %v", err)
	}
}

func TestServiceListIsDeterministic(t *testing.T) {
	s := testService(t, fakeResolver{})
	first, _ := s.Create(CreateRequest{Title: "First", Prompt: "one"})
	s.now = func() time.Time { return time.Date(2026, 7, 22, 19, 0, 0, 0, time.UTC) }
	second, _ := s.Create(CreateRequest{Title: "Second", Prompt: "two"})
	items, err := s.List("all")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != second.ID || items[1].ID != first.ID {
		t.Fatalf("order = %#v", items)
	}
}
