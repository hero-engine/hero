package fsutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// symlinkedTree returns a symlink to a directory holding a/b.md, skipping
// where the filesystem cannot create symlinks.
func symlinkedTree(t *testing.T) (link, real string) {
	t.Helper()
	base := t.TempDir()
	real = filepath.Join(base, "real")
	if err := os.MkdirAll(filepath.Join(real, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "a", "b.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link = filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	return link, real
}

func TestWalkDescendsSymlinkedRoot(t *testing.T) {
	link, _ := symlinkedTree(t)
	var paths []string
	err := Walk(link, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{link, filepath.Join(link, "a"), filepath.Join(link, "a", "b.md")}
	sort.Strings(paths)
	if len(paths) != len(want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("paths = %v, want %v (paths stay under the link)", paths, want)
		}
	}
}

func TestWalkDirDescendsSymlinkedRoot(t *testing.T) {
	link, _ := symlinkedTree(t)
	var files []string
	err := WalkDir(link, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != filepath.Join(link, "a", "b.md") {
		t.Fatalf("files = %v, want [%s]", files, filepath.Join(link, "a", "b.md"))
	}
}

func TestWalkPlainRootUnchanged(t *testing.T) {
	_, real := symlinkedTree(t)
	var files []string
	if err := Walk(real, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			files = append(files, path)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != filepath.Join(real, "a", "b.md") {
		t.Fatalf("files = %v", files)
	}
}

func TestWalkBrokenSymlinkRootReportsLikeFilepathWalk(t *testing.T) {
	base := t.TempDir()
	link := filepath.Join(base, "link")
	if err := os.Symlink(filepath.Join(base, "missing"), link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	var visited []string
	if err := Walk(link, func(path string, info os.FileInfo, err error) error {
		visited = append(visited, path)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(visited) != 1 || visited[0] != link {
		t.Fatalf("visited = %v, want just the link itself", visited)
	}
}
