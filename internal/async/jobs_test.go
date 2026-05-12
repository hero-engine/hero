package async

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJobStore_AddAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := NewJobStore(dir)

	job := Job{
		ID:         "test-001",
		Slug:       "my-feature",
		SpecPath:   "/path/to/spec.md",
		Branch:     "hero/deliver/my-feature",
		BaseBranch: "main",
		Status:     StatusPending,
	}

	if err := store.Add(job); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	jobs, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].ID != "test-001" {
		t.Errorf("expected ID test-001, got %s", jobs[0].ID)
	}
	if jobs[0].Slug != "my-feature" {
		t.Errorf("expected slug my-feature, got %s", jobs[0].Slug)
	}
}

func TestJobStore_Update(t *testing.T) {
	dir := t.TempDir()
	store := NewJobStore(dir)

	store.Add(Job{ID: "j1", Slug: "feat-1", Status: StatusPending})

	err := store.Update("j1", func(j *Job) {
		j.Status = StatusRunning
		j.StartedAt = time.Now()
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	j, err := store.Get("j1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if j.Status != StatusRunning {
		t.Errorf("expected running, got %s", j.Status)
	}
}

func TestJobStore_Active(t *testing.T) {
	dir := t.TempDir()
	store := NewJobStore(dir)

	store.Add(Job{ID: "j1", Status: StatusPending})
	store.Add(Job{ID: "j2", Status: StatusRunning})
	store.Add(Job{ID: "j3", Status: StatusCompleted})
	store.Add(Job{ID: "j4", Status: StatusFailed})

	active, err := store.Active()
	if err != nil {
		t.Fatalf("Active failed: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("expected 2 active, got %d", len(active))
	}
}

func TestJobStore_GetBySlug(t *testing.T) {
	dir := t.TempDir()
	store := NewJobStore(dir)

	store.Add(Job{ID: "j1", Slug: "feat-a", Status: StatusFailed})
	store.Add(Job{ID: "j2", Slug: "feat-a", Status: StatusRunning})

	j, err := store.GetBySlug("feat-a")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	// Should return the most recent (j2)
	if j.ID != "j2" {
		t.Errorf("expected j2, got %s", j.ID)
	}
}

func TestJobStore_LoadEmpty(t *testing.T) {
	dir := t.TempDir()
	store := NewJobStore(dir)

	jobs, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestBranchName(t *testing.T) {
	tests := []struct {
		slug     string
		expected string
	}{
		{"my-feature", "hero/deliver/my-feature"},
		{"cloud-auth", "hero/deliver/cloud-auth"},
		{"fix_bug_123", "hero/deliver/fix_bug_123"},
	}
	for _, tt := range tests {
		got := BranchName(tt.slug)
		if got != tt.expected {
			t.Errorf("BranchName(%q) = %q, want %q", tt.slug, got, tt.expected)
		}
	}
}

func TestJobStore_FilePersistence(t *testing.T) {
	dir := t.TempDir()
	store := NewJobStore(dir)

	store.Add(Job{ID: "persist-1", Slug: "test", Status: StatusPending})

	// Verify file exists
	path := filepath.Join(dir, "async-jobs.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("expected jobs file to exist")
	}

	// Create a new store instance pointing to same file
	store2 := NewJobStore(dir)
	jobs, err := store2.Load()
	if err != nil {
		t.Fatalf("Load from new store failed: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != "persist-1" {
		t.Errorf("expected persist-1 from new store instance")
	}
}
