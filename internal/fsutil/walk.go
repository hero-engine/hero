// Package fsutil holds filesystem helpers shared across Hero packages.
package fsutil

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Walk is filepath.Walk that also descends when root is itself a symlink
// or junction to a directory (filepath.Walk never follows its root). Paths passed to fn
// stay under root, so callers that compare or relativize against root are
// unaffected.
func Walk(root string, fn filepath.WalkFunc) error {
	real, linked := resolveRoot(root)
	if !linked {
		return filepath.Walk(root, fn)
	}
	return filepath.Walk(real, func(path string, info os.FileInfo, err error) error {
		return fn(underRoot(root, real, path), info, err)
	})
}

// WalkDir is filepath.WalkDir with Walk's symlinked-root handling.
func WalkDir(root string, fn fs.WalkDirFunc) error {
	real, linked := resolveRoot(root)
	if !linked {
		return filepath.WalkDir(root, fn)
	}
	return filepath.WalkDir(real, func(path string, d fs.DirEntry, err error) error {
		return fn(underRoot(root, real, path), d, err)
	})
}

// Filesystem calls, swappable so tests can reproduce what Windows reports
// for a directory junction.
var (
	lstat        = os.Lstat
	stat         = os.Stat
	readlink     = os.Readlink
	evalSymlinks = filepath.EvalSymlinks
)

// resolveRoot returns root's target when root links to a directory: a
// symlink, or a Windows directory junction (Go 1.23+ reports junctions as
// ModeIrregular without ModeDir, and EvalSymlinks does not follow them, so
// they are resolved with Readlink).
func resolveRoot(root string) (string, bool) {
	info, err := lstat(root)
	if err != nil || info.IsDir() || info.Mode()&(os.ModeSymlink|os.ModeIrregular) == 0 {
		return root, false
	}
	if target, err := stat(root); err != nil || !target.IsDir() {
		return root, false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if real, err := evalSymlinks(root); err == nil {
			return real, true
		}
		return root, false
	}
	real, err := readlink(root)
	if err != nil {
		return root, false
	}
	if resolved, err := evalSymlinks(real); err == nil {
		real = resolved
	}
	return real, true
}

// underRoot maps a path under real back under root.
func underRoot(root, real, path string) string {
	rel, err := filepath.Rel(real, path)
	if err != nil || rel == "." {
		return root
	}
	return filepath.Join(root, rel)
}
