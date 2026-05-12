package watch

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestScan_empty(t *testing.T) {
	dir := t.TempDir()
	snap, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(snap) != 0 {
		t.Errorf("expected empty snapshot, got %d entries", len(snap))
	}
}

func TestScan_findsMarkdown(t *testing.T) {
	dir := t.TempDir()

	// Create a spec file
	specDir := filepath.Join(dir, "specs", "my-feature")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte("# Title"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a non-markdown file (should be ignored)
	if err := os.WriteFile(filepath.Join(dir, "index.db"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create hero.json (should be tracked)
	if err := os.WriteFile(filepath.Join(dir, "hero.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	snap, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(snap) != 2 {
		t.Errorf("expected 2 entries (spec.md + hero.json), got %d", len(snap))
		for p := range snap {
			t.Logf("  %s", p)
		}
	}

	// Verify spec.md is in snapshot
	specPath := filepath.Join(specDir, "spec.md")
	if _, ok := snap[specPath]; !ok {
		t.Errorf("expected spec.md in snapshot")
	}
}

func TestScan_skipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()

	// Create a hidden directory with a markdown file
	hiddenDir := filepath.Join(dir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "test.md"), []byte("hidden"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a normal markdown file
	if err := os.WriteFile(filepath.Join(dir, "visible.md"), []byte("visible"), 0o644); err != nil {
		t.Fatal(err)
	}

	snap, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(snap) != 1 {
		t.Errorf("expected 1 entry (visible.md only), got %d", len(snap))
		for p := range snap {
			t.Logf("  %s", p)
		}
	}
}

func TestDiff_noChanges(t *testing.T) {
	now := time.Now()
	snap := map[string]time.Time{
		"/a.md": now,
		"/b.md": now,
	}
	events := Diff(snap, snap)
	if len(events) != 0 {
		t.Errorf("expected no events, got %d", len(events))
	}
}

func TestDiff_created(t *testing.T) {
	now := time.Now()
	old := map[string]time.Time{
		"/a.md": now,
	}
	new := map[string]time.Time{
		"/a.md": now,
		"/b.md": now,
	}

	events := Diff(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventCreated {
		t.Errorf("expected created, got %s", events[0].Kind)
	}
	if events[0].Path != "/b.md" {
		t.Errorf("expected /b.md, got %s", events[0].Path)
	}
}

func TestDiff_modified(t *testing.T) {
	t1 := time.Now()
	t2 := t1.Add(time.Second)

	old := map[string]time.Time{"/a.md": t1}
	new := map[string]time.Time{"/a.md": t2}

	events := Diff(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventModified {
		t.Errorf("expected modified, got %s", events[0].Kind)
	}
}

func TestDiff_deleted(t *testing.T) {
	now := time.Now()
	old := map[string]time.Time{"/a.md": now, "/b.md": now}
	new := map[string]time.Time{"/a.md": now}

	events := Diff(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventDeleted {
		t.Errorf("expected deleted, got %s", events[0].Kind)
	}
	if events[0].Path != "/b.md" {
		t.Errorf("expected /b.md, got %s", events[0].Path)
	}
}

func TestDiff_mixed(t *testing.T) {
	t1 := time.Now()
	t2 := t1.Add(time.Second)

	old := map[string]time.Time{
		"/a.md": t1,
		"/b.md": t1,
	}
	new := map[string]time.Time{
		"/a.md": t2, // modified
		"/c.md": t1, // created
		// /b.md deleted
	}

	events := Diff(old, new)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	kinds := make(map[EventKind]int)
	for _, e := range events {
		kinds[e.Kind]++
	}

	if kinds[EventCreated] != 1 {
		t.Errorf("expected 1 created, got %d", kinds[EventCreated])
	}
	if kinds[EventModified] != 1 {
		t.Errorf("expected 1 modified, got %d", kinds[EventModified])
	}
	if kinds[EventDeleted] != 1 {
		t.Errorf("expected 1 deleted, got %d", kinds[EventDeleted])
	}
}

func TestEvent_IsSpec(t *testing.T) {
	tests := []struct {
		path   string
		isSpec bool
	}{
		{"/hero/specs/my-feature/spec.md", true},
		{"/hero/knowledge/conventions/naming/spec.md", true},
		{"/hero/knowledge/notes/brainstorm/spec.md", true},
		{"/hero/hero.json", false},
		{"/hero/README.md", false},
	}

	for _, tt := range tests {
		e := Event{Path: tt.path, Kind: EventModified}
		if got := e.IsSpec(); got != tt.isSpec {
			t.Errorf("Event{Path: %q}.IsSpec() = %v, want %v", tt.path, got, tt.isSpec)
		}
	}
}

func TestEvent_String(t *testing.T) {
	e := Event{Path: "/hero/specs/auth/spec.md", Kind: EventCreated}
	s := e.String()
	if s != "[created] /hero/specs/auth/spec.md" {
		t.Errorf("unexpected String: %q", s)
	}
}

func TestSpecEvents_filters(t *testing.T) {
	events := []Event{
		{Path: "/hero/specs/a/spec.md", Kind: EventCreated},
		{Path: "/hero/hero.json", Kind: EventModified},
		{Path: "/hero/specs/b/spec.md", Kind: EventModified},
		{Path: "/hero/README.md", Kind: EventDeleted},
	}

	specs := SpecEvents(events)
	if len(specs) != 2 {
		t.Errorf("expected 2 spec events, got %d", len(specs))
	}
}

func TestSummary_noChanges(t *testing.T) {
	s := Summary(nil)
	if s != "No changes detected." {
		t.Errorf("unexpected summary: %q", s)
	}
}

func TestSummary_mixed(t *testing.T) {
	events := []Event{
		{Path: "/a.md", Kind: EventCreated},
		{Path: "/b.md", Kind: EventCreated},
		{Path: "/c.md", Kind: EventModified},
		{Path: "/d.md", Kind: EventDeleted},
	}

	s := Summary(events)
	expected := "4 change(s): 2 created, 1 modified, 1 deleted"
	if s != expected {
		t.Errorf("got %q, want %q", s, expected)
	}
}

func TestRunOnce_detectsNewFiles(t *testing.T) {
	dir := t.TempDir()

	var captured []Event
	w := New(dir, time.Second, func(events []Event) {
		captured = events
	})

	// First scan — empty
	events, err := w.RunOnce()
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events on first scan, got %d", len(events))
	}

	// Create a file
	specDir := filepath.Join(dir, "specs", "new-feature")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte("# New Feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second scan — should detect creation
	events, err = w.RunOnce()
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventCreated {
		t.Errorf("expected created, got %s", events[0].Kind)
	}
	if len(captured) != 1 {
		t.Errorf("handler should have been called with 1 event, got %d", len(captured))
	}
}

func TestRunOnce_detectsModifiedFiles(t *testing.T) {
	dir := t.TempDir()

	specDir := filepath.Join(dir, "specs", "feature")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(specDir, "spec.md")
	if err := os.WriteFile(specPath, []byte("# Original\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := New(dir, time.Second, nil)

	// First scan
	if _, err := w.RunOnce(); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	// Modify the file (ensure mtime changes)
	time.Sleep(50 * time.Millisecond)
	now := time.Now().Add(time.Second)
	if err := os.WriteFile(specPath, []byte("# Modified\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(specPath, now, now)

	// Second scan
	events, err := w.RunOnce()
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventModified {
		t.Errorf("expected modified, got %s", events[0].Kind)
	}
}

func TestRunOnce_detectsDeletedFiles(t *testing.T) {
	dir := t.TempDir()

	specDir := filepath.Join(dir, "specs", "feature")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(specDir, "spec.md")
	if err := os.WriteFile(specPath, []byte("# Feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := New(dir, time.Second, nil)

	// First scan
	if _, err := w.RunOnce(); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	// Delete the file
	if err := os.Remove(specPath); err != nil {
		t.Fatal(err)
	}

	// Second scan
	events, err := w.RunOnce()
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventDeleted {
		t.Errorf("expected deleted, got %s", events[0].Kind)
	}
}

func TestRun_stopsOnStop(t *testing.T) {
	dir := t.TempDir()

	var mu sync.Mutex
	pollCount := 0

	w := New(dir, 50*time.Millisecond, func(events []Event) {
		mu.Lock()
		defer mu.Unlock()
		pollCount++
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Run()
	}()

	// Let it poll a few times
	time.Sleep(200 * time.Millisecond)
	w.Stop()

	err := <-errCh
	if err != nil {
		t.Errorf("Run returned error: %v", err)
	}
}

func TestNew_defaults(t *testing.T) {
	dir := t.TempDir()
	w := New(dir, 2*time.Second, nil)
	if w.heroDir != dir {
		t.Errorf("heroDir = %q, want %q", w.heroDir, dir)
	}
	if w.interval != 2*time.Second {
		t.Errorf("interval = %v, want %v", w.interval, 2*time.Second)
	}
}

func TestDiff_emptySnapshots(t *testing.T) {
	events := Diff(map[string]time.Time{}, map[string]time.Time{})
	if len(events) != 0 {
		t.Errorf("expected no events, got %d", len(events))
	}
}

func TestScan_deeplyNested(t *testing.T) {
	dir := t.TempDir()

	// Create deeply nested spec
	deep := filepath.Join(dir, "knowledge", "conventions", "naming")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "spec.md"), []byte("# Naming"), 0o644); err != nil {
		t.Fatal(err)
	}

	snap, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(snap) != 1 {
		t.Errorf("expected 1 entry, got %d", len(snap))
	}
}
