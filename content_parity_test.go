package hero

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"
)

// TestDomainPacks_NoUnannotatedCoreShadows enforces the single-master
// content contract established by the content-dedup-resync spec: a
// domain-pack file that shadows a core path (same relative path under
// agents/, commands/, or skills/) fully replaces it at install time
// (OverlayFS is file-level — domain wins), so every shadow is either
// accidental duplication or a deliberate per-domain rewrite.
//
// Accidental duplication is how core and engineering silently forked
// between v0.8.0 and v0.23 (see the hero-content-audit findings): the
// same file was maintained in two places, fixes landed on one side,
// and each install audience got a different stale copy. This test
// makes that state unrepresentable:
//
//   - A shadow with no `core_fork:` frontmatter key fails — delete the
//     redundant copy (the overlay falls through to core) or annotate
//     the intentional fork with a one-line reason.
//   - A shadow annotated `core_fork:` that is byte-identical to the
//     core file also fails — an identical copy is dead weight, not a
//     fork.
func TestDomainPacks_NoUnannotatedCoreShadows(t *testing.T) {
	core := CoreFS()

	for _, domain := range AvailableDomains() {
		t.Run(domain, func(t *testing.T) {
			domFS, err := DomainFS(domain)
			if err != nil {
				t.Fatalf("DomainFS(%q): %v", domain, err)
			}

			err = fs.WalkDir(domFS, ".", func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				if !strings.HasPrefix(p, "agents/") &&
					!strings.HasPrefix(p, "commands/") &&
					!strings.HasPrefix(p, "skills/") {
					return nil
				}
				coreBytes, err := fs.ReadFile(core, p)
				if err != nil {
					return nil // no core counterpart — pack-native file
				}
				domBytes, err := fs.ReadFile(domFS, p)
				if err != nil {
					return err
				}

				reason, annotated := coreForkReason(domBytes)
				switch {
				case !annotated:
					t.Errorf("domains/%s/%s shadows core/%s with no core_fork: annotation — delete the redundant copy (overlay falls through to core) or annotate the intentional fork", domain, p, p)
				case strings.TrimSpace(reason) == "":
					t.Errorf("domains/%s/%s has an empty core_fork: annotation — state why this fork replaces core/%s", domain, p, p)
				case bytes.Equal(stripCoreForkLine(domBytes), coreBytes):
					t.Errorf("domains/%s/%s is annotated core_fork but content-identical to core/%s — delete the copy", domain, p, p)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("walk domain %q: %v", domain, err)
			}
		})
	}
}

// stripCoreForkLine removes the frontmatter `core_fork:` line so the
// identical-copy check compares content, not annotation: a fork whose
// only difference from core is its own annotation is still dead weight.
func stripCoreForkLine(content []byte) []byte {
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return content
	}
	for i, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		if strings.HasPrefix(line, "core_fork:") {
			return []byte(strings.Join(append(lines[:i+1:i+1], lines[i+2:]...), "\n"))
		}
	}
	return content
}

// coreForkReason extracts the value of a top-level `core_fork:` key
// from the file's leading YAML frontmatter block. Returns ok=false
// when there is no frontmatter or no core_fork key.
func coreForkReason(content []byte) (reason string, ok bool) {
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", false
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return "", false
		}
		if v, found := strings.CutPrefix(line, "core_fork:"); found {
			return strings.TrimSpace(v), true
		}
	}
	return "", false
}
