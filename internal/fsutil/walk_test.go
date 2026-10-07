package fsutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
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

// fakeInfo is a FileInfo with a chosen mode, for reproducing what Windows
// reports for reparse points.
type fakeInfo struct {
	name string
	mode os.FileMode
}

func (f fakeInfo) Name() string       { return f.name }
func (f fakeInfo) Size() int64        { return 0 }
func (f fakeInfo) Mode() os.FileMode  { return f.mode }
func (f fakeInfo) ModTime() time.Time { return time.Time{} }
func (f fakeInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeInfo) Sys() any           { return nil }

// stubRoot makes root report lstatMode, stat as target, and Readlink to
// target, as Windows does for a reparse point; other paths are real.
func stubRoot(t *testing.T, root, target string, lstatMode os.FileMode) {
	t.Helper()
	origLstat, origStat, origReadlink := lstat, stat, readlink
	t.Cleanup(func() { lstat, stat, readlink = origLstat, origStat, origReadlink })
	lstat = func(p string) (os.FileInfo, error) {
		if p == root {
			return fakeInfo{name: filepath.Base(root), mode: lstatMode}, nil
		}
		return origLstat(p)
	}
	stat = func(p string) (os.FileInfo, error) {
		if p == root {
			return origStat(target)
		}
		return origStat(p)
	}
	readlink = func(p string) (string, error) {
		if p == root {
			return target, nil
		}
		return origReadlink(p)
	}
}

// symlinked-hero-dir: a Windows directory junction (Lstat: ModeIrregular,
// no ModeDir; EvalSymlinks does not follow it) is resolved with Readlink
// and walked, with paths under the junction.
func TestWalkDescendsJunctionRoot(t *testing.T) {
	_, real := symlinkedTree(t)
	junction := filepath.Join(t.TempDir(), ".hero")
	stubRoot(t, junction, real, os.ModeIrregular)
	if got, linked := resolveRoot(junction); !linked {
		t.Fatalf("resolveRoot(junction) = %q, not linked", got)
	}
	var files []string
	if err := Walk(junction, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			files = append(files, path)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != filepath.Join(junction, "a", "b.md") {
		t.Fatalf("files = %v, want [%s]", files, filepath.Join(junction, "a", "b.md"))
	}
}

// A real directory carrying a non-link reparse tag (e.g. a OneDrive
// placeholder: ModeDir|ModeIrregular) and an irregular non-directory are
// not treated as links.
func TestResolveRootIgnoresNonLinkReparsePoints(t *testing.T) {
	_, real := symlinkedTree(t)
	for name, mode := range map[string]os.FileMode{
		"cloud placeholder dir": os.ModeDir | os.ModeIrregular,
		"irregular file":        os.ModeIrregular,
	} {
		root := filepath.Join(t.TempDir(), "root")
		target := real
		if !mode.IsDir() {
			target = filepath.Join(real, "a", "b.md")
		}
		stubRoot(t, root, target, mode)
		if got, linked := resolveRoot(root); linked || got != root {
			t.Errorf("%s: resolveRoot = %q, %v; want %q, false", name, got, linked, root)
		}
	}
}
