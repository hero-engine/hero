package opsrunner

import (
	"testing"
	"time"
)

func TestRegistry_GetOrCreate_Dedup(t *testing.T) {
	r := newRegistry()
	calls := 0
	factory := func() *Job {
		calls++
		return &Job{ID: "id-1", Slug: "p", Verb: "re-scan", done: make(chan struct{})}
	}
	j1, fresh := r.getOrCreate("p", "re-scan", factory)
	if !fresh {
		t.Fatal("first call should be fresh")
	}
	if j1.ID != "id-1" {
		t.Fatalf("ID = %q", j1.ID)
	}

	// Second call with the same key returns the same job, factory not invoked again.
	j2, fresh2 := r.getOrCreate("p", "re-scan", func() *Job {
		calls++
		return &Job{ID: "id-2"}
	})
	if fresh2 {
		t.Fatal("second call should be dedup")
	}
	if j2 != j1 {
		t.Fatalf("expected same job pointer")
	}
	if calls != 1 {
		t.Fatalf("factory invoked %d times, want 1", calls)
	}
}

func TestRegistry_GetOrCreate_AfterDone(t *testing.T) {
	r := newRegistry()
	first, _ := r.getOrCreate("p", "v", func() *Job {
		return &Job{ID: "id-a", done: make(chan struct{})}
	})
	close(first.done)

	// After done is closed, a fresh getOrCreate should produce a new job.
	second, fresh := r.getOrCreate("p", "v", func() *Job {
		return &Job{ID: "id-b", done: make(chan struct{})}
	})
	if !fresh {
		t.Fatal("expected fresh after first finished")
	}
	if second.ID != "id-b" {
		t.Fatalf("expected new job, got ID %q", second.ID)
	}
}

func TestRegistry_Lookup_ExcludesFinished(t *testing.T) {
	r := newRegistry()
	j, _ := r.getOrCreate("p", "v", func() *Job {
		return &Job{ID: "id", Slug: "p", Verb: "v", done: make(chan struct{})}
	})
	if r.lookup("p", "v") == nil {
		t.Fatal("expected in-flight lookup to succeed")
	}
	close(j.done)
	if r.lookup("p", "v") != nil {
		t.Fatal("expected finished lookup to return nil")
	}
}

func TestRegistry_FindByID_IncludesFinished(t *testing.T) {
	r := newRegistry()
	j, _ := r.getOrCreate("p", "v", func() *Job {
		return &Job{ID: "id", Slug: "p", Verb: "v", done: make(chan struct{})}
	})
	close(j.done)
	if r.findByID("p", "id") != j {
		t.Fatal("findByID should still locate finished jobs")
	}
	if r.findByID("p", "other") != nil {
		t.Fatal("findByID for unknown id should be nil")
	}
}

func TestRingBuffer_Cap(t *testing.T) {
	rb := newRingBuffer(3)
	for i := 0; i < 5; i++ {
		rb.push(outputLine{Text: string(rune('a' + i)), At: time.Now()})
	}
	snap := rb.snapshot()
	if len(snap) != 3 {
		t.Fatalf("len = %d, want 3", len(snap))
	}
	if snap[0].Text != "c" || snap[1].Text != "d" || snap[2].Text != "e" {
		t.Fatalf("unexpected ring contents: %+v", snap)
	}
}

func TestNewJobID_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := newJobID()
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}
