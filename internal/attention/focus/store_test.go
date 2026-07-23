package focus

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
)

func TestStorePersistsPrivateItemWithContentRevision(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	item := testItem("focus_one")
	created, err := store.Create(item)
	if err != nil {
		t.Fatal(err)
	}
	if created.Revision <= 0 {
		t.Fatalf("revision = %d", created.Revision)
	}
	path := filepath.Join(root, "focus", "items", "focus_one.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	got, err := store.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Prompt != item.Prompt || got.Revision != created.Revision {
		t.Fatalf("got %#v", got)
	}

	other := item
	other.ID = "focus_two"
	other.Revision = 999
	createdOther, err := store.Create(other)
	if err != nil {
		t.Fatal(err)
	}
	// ID participates in persisted canonical content, so different items get
	// independent opaque revisions even with the same user fields.
	if createdOther.Revision == created.Revision {
		t.Fatal("distinct persisted content produced same revision")
	}
}

func TestStoreRejectsPathTraversalIDs(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	item := testItem("focus_../escape")
	if _, err := store.Create(item); err == nil {
		t.Fatal("path traversal ID accepted")
	}
}

func TestStoreCreateOrGetIsIdempotentAndDetectsConflict(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	item := testItem("focus_one")
	item.Origin = &attention.ProvenanceReference{Kind: "mail", SourceID: "mail_one"}
	item.OriginKey = "mail:mail_one:add_to_today"
	created, wasCreated, err := store.CreateOrGet(item)
	if err != nil || !wasCreated {
		t.Fatalf("first = %#v, %v, %v", created, wasCreated, err)
	}
	replay := item
	replay.ID = "focus_replay"
	replay.CreatedAt, replay.UpdatedAt = "later", "later"
	got, wasCreated, err := store.CreateOrGet(replay)
	if err != nil || wasCreated || got.ID != created.ID {
		t.Fatalf("replay = %#v, %v, %v", got, wasCreated, err)
	}
	replay.Prompt = "different"
	_, _, err = store.CreateOrGet(replay)
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("conflict err = %v", err)
	}
	unchanged, _ := store.Get(created.ID)
	if unchanged.Prompt != item.Prompt {
		t.Fatalf("original changed: %#v", unchanged)
	}
}

func TestStoreConcurrentReplaceDoesNotLoseUpdate(t *testing.T) {
	root := t.TempDir()
	a, _ := NewStore(root)
	b, _ := NewStore(root)
	created, err := a.Create(testItem("focus_one"))
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i, store := range []*Store{a, b} {
		wg.Add(1)
		go func(i int, store *Store) {
			defer wg.Done()
			<-start
			_, err := store.Replace(created.ID, created.Revision, func(item *Item) (bool, error) {
				if i == 0 {
					item.Lifecycle = attention.FocusToday
				} else {
					item.Lifecycle = attention.FocusLater
				}
				item.UpdatedAt = "2026-07-22T19:00:00Z"
				return true, nil
			})
			errs <- err
		}(i, store)
	}
	close(start)
	wg.Wait()
	close(errs)
	var successes, stale int
	for err := range errs {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrStale) {
			stale++
		} else {
			t.Fatalf("unexpected err: %v", err)
		}
	}
	if successes != 1 || stale != 1 {
		t.Fatalf("successes=%d stale=%d", successes, stale)
	}
}

func testItem(id string) Item {
	return Item{SchemaVersion: 1, ID: id, Title: "Remember", Prompt: "Continue exactly here.\n", Lifecycle: attention.FocusInbox, CreatedAt: "2026-07-22T18:00:00Z", UpdatedAt: "2026-07-22T18:00:00Z"}
}
