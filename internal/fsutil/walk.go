// Package fsutil holds filesystem helpers shared across Hero packages.
package fsutil

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Walk is filepath.Walk that also descends when root is itself a symlink
// to a directory (filepath.Walk never follows its root). Paths passed to fn
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

// resolveRoot returns root's target when root is a symlink that resolves.
func resolveRoot(root string) (string, bool) {
	info, err := os.Lstat(root)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return root, false
	}
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		return root, false
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
