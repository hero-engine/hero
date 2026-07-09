package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBaseline_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := &Baseline{
		TrackerID: "PROJ-123",
		Base: map[string]Base{
			"title": TextBase("Fix login bug"),
			"body":  TextBase("line one\nline two"),
			"tags":  TagsBase([]string{"bug", "sync"}),
		},
	}
	if err := WriteBaseline(dir, "fix-login", in); err != nil {
		t.Fatalf("write: %v", err)
	}

	// SyncedAt stamped on write.
	if in.SyncedAt == "" {
		t.Error("SyncedAt not stamped")
	}

	out, err := ReadBaseline(dir, "fix-login")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if out.TrackerID != "PROJ-123" {
		t.Errorf("tracker_id = %q", out.TrackerID)
	}
	if out.Base["title"].Text != "Fix login bug" {
		t.Errorf("title base = %q", out.Base["title"].Text)
	}
	if out.Base["body"].Text != "line one\nline two" {
		t.Errorf("body base = %q", out.Base["body"].Text)
	}
	if !out.Base["tags"].IsTags() {
		t.Fatal("tags base not tag-kinded")
	}
	if len(out.Base["tags"].Tags) != 2 {
		t.Errorf("tags base = %v", out.Base["tags"].Tags)
	}
}

// TestBaseline_ContractShape asserts the on-disk JSON matches the cross-repo
// contract the Swift SyncStateStore reads: a top-level tracker_id/base/synced_at
// with title/body as bare strings and tags as a bare array.
func TestBaseline_ContractShape(t *testing.T) {
	dir := t.TempDir()
	b := &Baseline{
		TrackerID: "ABC-1",
		Base: map[string]Base{
			"title": TextBase("T"),
			"tags":  TagsBase([]string{"x", "y"}),
		},
	}
	if err := WriteBaseline(dir, "s", b); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, StateDirName, "s.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{`"tracker_id": "ABC-1"`, `"title": "T"`, `"tags": [`, `"synced_at"`} {
		if !strings.Contains(s, want) {
			t.Errorf("contract file missing %q; got:\n%s", want, s)
		}
	}
}

func TestBaseline_MissingIsNilNil(t *testing.T) {
	dir := t.TempDir()
	b, err := ReadBaseline(dir, "never-synced")
	if err != nil {
		t.Fatalf("missing baseline should not error: %v", err)
	}
	if b != nil {
		t.Fatalf("missing baseline should be nil, got %+v", b)
	}
}
