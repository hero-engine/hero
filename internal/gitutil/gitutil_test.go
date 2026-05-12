package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initGitRepo creates a temporary git repo with an initial commit.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")

	// Create initial commit
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "initial commit")

	return dir
}

func TestIsRepo(t *testing.T) {
	dir := initGitRepo(t)
	if !IsRepo(dir) {
		t.Error("should detect git repo")
	}

	nonGit := t.TempDir()
	if IsRepo(nonGit) {
		t.Error("should not detect non-git directory")
	}
}

func TestDefaultBranch(t *testing.T) {
	dir := initGitRepo(t)
	branch := DefaultBranch(dir)
	if branch != "main" {
		t.Errorf("expected main, got %q", branch)
	}
}

func TestCurrentBranch(t *testing.T) {
	dir := initGitRepo(t)

	branch := CurrentBranch(dir)
	if branch != "main" {
		t.Errorf("expected main, got %q", branch)
	}

	// Create and switch to a feature branch
	run(t, dir, "checkout", "-b", "feature/test")

	branch = CurrentBranch(dir)
	if branch != "feature/test" {
		t.Errorf("expected feature/test, got %q", branch)
	}
}

func TestFilesChangedOnBranch(t *testing.T) {
	dir := initGitRepo(t)

	// Create a feature branch and make changes
	run(t, dir, "checkout", "-b", "feature/new-stuff")

	if err := os.WriteFile(filepath.Join(dir, "new-file.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "add", ".")
	run(t, dir, "commit", "-m", "add new file")

	files := FilesChangedOnBranch(dir, "main")
	if len(files) != 1 || files[0] != "new-file.go" {
		t.Errorf("expected [new-file.go], got %v", files)
	}
}

func TestFilesChangedOnBranch_NoChanges(t *testing.T) {
	dir := initGitRepo(t)

	// Stay on main — no divergence
	files := FilesChangedOnBranch(dir, "main")
	if len(files) != 0 {
		t.Errorf("expected empty, got %v", files)
	}
}

func TestFilesChangedUncommitted(t *testing.T) {
	dir := initGitRepo(t)

	// Create an unstaged file
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := FilesChangedUncommitted(dir)
	found := false
	for _, f := range files {
		if f == "dirty.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected dirty.txt in uncommitted files, got %v", files)
	}
}

func TestAllChangedFiles(t *testing.T) {
	dir := initGitRepo(t)

	// Create a feature branch with a committed change
	run(t, dir, "checkout", "-b", "feature/combo")

	if err := os.WriteFile(filepath.Join(dir, "committed.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "add", ".")
	run(t, dir, "commit", "-m", "committed change")

	// Also create an uncommitted file
	if err := os.WriteFile(filepath.Join(dir, "uncommitted.txt"), []byte("wip\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := AllChangedFiles(dir)
	hasCommitted := false
	hasUncommitted := false
	for _, f := range files {
		if f == "committed.go" {
			hasCommitted = true
		}
		if f == "uncommitted.txt" {
			hasUncommitted = true
		}
	}

	if !hasCommitted {
		t.Errorf("should include committed.go, got %v", files)
	}
	if !hasUncommitted {
		t.Errorf("should include uncommitted.txt, got %v", files)
	}
}

func TestNormalizeFilePath(t *testing.T) {
	tests := []struct {
		root string
		path string
		want string
	}{
		{"/home/user/project", "./src/main.go", "src/main.go"},
		{"/home/user/project", "src/main.go", "src/main.go"},
		{"/home/user/project", "/home/user/project/src/main.go", "src/main.go"},
	}

	for _, tt := range tests {
		got := NormalizeFilePath(tt.root, tt.path)
		if got != tt.want {
			t.Errorf("NormalizeFilePath(%q, %q) = %q, want %q", tt.root, tt.path, got, tt.want)
		}
	}
}

func TestIsRepo_NonGitDir(t *testing.T) {
	dir := t.TempDir()
	if IsRepo(dir) {
		t.Error("non-git directory should return false")
	}

	// All git functions should gracefully handle non-git dirs
	if branch := DefaultBranch(dir); branch != "main" {
		t.Errorf("non-git dir should fall back to main, got %q", branch)
	}
	if branch := CurrentBranch(dir); branch != "" {
		t.Errorf("non-git dir should return empty current branch, got %q", branch)
	}
	if files := FilesChangedOnBranch(dir, "main"); len(files) != 0 {
		t.Errorf("non-git dir should return empty files, got %v", files)
	}
	if files := FilesChangedUncommitted(dir); len(files) != 0 {
		t.Errorf("non-git dir should return empty files, got %v", files)
	}
}

// run is a test helper that executes a git command.
func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
