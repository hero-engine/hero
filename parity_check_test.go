package hero

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestRootEngineeringParity asserts that for the legacy-fallback removal
// cutover, the root-level agents/, commands/, skills/ directories are in
// bit-for-bit parity with their domains/engineering/ counterparts. After
// ContentFS() is repointed at domains/engineering/, install output must
// be byte-for-byte identical, which can only hold if these surfaces match.
//
// Per the contentfs-legacy-fallback-removal spec, root wins on conflict.
// This test fails loudly with per-file detail when parity breaks so the
// sync step can be re-run.
//
// Run with: go test -run TestRootEngineeringParity ./...
//
// Once the cutover lands and the root-level dirs are removed, this test
// becomes obsolete and should be deleted alongside them.
func TestRootEngineeringParity(t *testing.T) {
	for _, sub := range []string{"agents", "commands", "skills"} {
		t.Run(sub, func(t *testing.T) {
			rootDir := sub
			mirrorDir := filepath.Join("domains", "engineering", sub)

			rootFiles, err := walkRel(rootDir)
			if err != nil {
				t.Fatalf("walk root %s: %v", rootDir, err)
			}
			mirrorFiles, err := walkRel(mirrorDir)
			if err != nil {
				t.Fatalf("walk mirror %s: %v", mirrorDir, err)
			}

			rootSet := toSet(rootFiles)
			mirrorSet := toSet(mirrorFiles)

			var onlyRoot, onlyMirror []string
			for f := range rootSet {
				if !mirrorSet[f] {
					onlyRoot = append(onlyRoot, f)
				}
			}
			for f := range mirrorSet {
				if !rootSet[f] {
					onlyMirror = append(onlyMirror, f)
				}
			}
			sort.Strings(onlyRoot)
			sort.Strings(onlyMirror)

			if len(onlyRoot) > 0 {
				t.Errorf("%s: %d file(s) in root not in mirror:\n  %s", sub, len(onlyRoot), strings.Join(onlyRoot, "\n  "))
			}
			if len(onlyMirror) > 0 {
				t.Errorf("%s: %d file(s) in mirror not in root:\n  %s", sub, len(onlyMirror), strings.Join(onlyMirror, "\n  "))
			}

			var diffBytes []string
			for f := range rootSet {
				if !mirrorSet[f] {
					continue
				}
				a, err := os.ReadFile(filepath.Join(rootDir, f))
				if err != nil {
					t.Errorf("read root %s: %v", f, err)
					continue
				}
				b, err := os.ReadFile(filepath.Join(mirrorDir, f))
				if err != nil {
					t.Errorf("read mirror %s: %v", f, err)
					continue
				}
				if !bytes.Equal(a, b) {
					diffBytes = append(diffBytes, f)
				}
			}
			sort.Strings(diffBytes)
			if len(diffBytes) > 0 {
				t.Errorf("%s: %d file(s) differ in bytes:\n  %s", sub, len(diffBytes), strings.Join(diffBytes, "\n  "))
			}
		})
	}
}

// walkRel returns all regular file paths under dir, expressed relative
// to dir, using forward slashes. Returns an empty slice (not an error)
// if dir does not exist — that's a parity failure surfaced by the set
// comparison rather than a fatal.
func walkRel(dir string) ([]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	var out []string
	err = filepath.Walk(dir, func(p string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fi.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	return out, err
}

func toSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}
