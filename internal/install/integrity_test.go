package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/managed"
)

// integrity_test.go — CheckIntegrity coverage, table-driven over all seven
// targets per the harness-changes-cover-all-targets tripwire. Mirrors the
// TestHarness_InstalledContentSurvivesOrdinaryCommands shape: install via
// the harness, damage (or don't), then assert what CheckIntegrity reports.

// integrityTargets is the full seven-target table with each target's native
// instruction file per nativeInstructionFile (AC-5).
var integrityTargets = []struct {
	name     string
	target   Target
	rootFile string
}{
	{"claude", TargetClaude, "CLAUDE.md"},
	{"codex", TargetCodex, "AGENTS.md"},
	{"opencode", TargetOpenCode, "AGENTS.md"},
	{"cursor", TargetCursor, "AGENTS.md"},
	{"copilot", TargetCopilot, "AGENTS.md"},
	{"generic", TargetGeneric, "AGENTS.md"},
	{"grok", TargetGrok, "AGENTS.md"},
}

// newIntegrityHarness installs one target with a .hero/ workspace
// pre-created (so install-state.json and the root instruction file land)
// and returns the harness plus the base Options CheckIntegrity callers
// must pass — the same content source the install used, per the
// options-construction-parity rule.
func newIntegrityHarness(t *testing.T, target Target) (*installHarness, Options) {
	t.Helper()
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	h.Run(target, nil)
	return h, Options{SourceDir: h.SourceDir}
}

func mustCheckIntegrity(t *testing.T, projectRoot string, base Options) []IntegrityFinding {
	t.Helper()
	findings, err := CheckIntegrity(projectRoot, base)
	if err != nil {
		t.Fatalf("CheckIntegrity: %v", err)
	}
	return findings
}

// damageRegion replaces the managed region of rootFile with a region whose
// body is newBody, preserving content outside the markers (the shape of
// the original eraser incident).
func damageRegion(t *testing.T, h *installHarness, rootFile, newBody string) {
	t.Helper()
	path := filepath.Join(h.TargetDir, rootFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", rootFile, err)
	}
	region := managed.RenderManagedRegion("dev", newBody)
	if err := os.WriteFile(path, []byte(managed.InsertManagedRegion(string(data), region)), 0o644); err != nil {
		t.Fatalf("write %s: %v", rootFile, err)
	}
}

// TestCheckIntegrity_RenderBodyIsDeterministic guards the oracle's core
// assumption: the managed body renders identically on consecutive calls.
// If any section ever injects a timestamp, absolute path, or map-iteration
// order, every healthy install would report stale — this test names the
// regression directly instead of letting it surface as flaky findings.
func TestCheckIntegrity_RenderBodyIsDeterministic(t *testing.T) {
	for _, tc := range integrityTargets {
		t.Run(tc.name, func(t *testing.T) {
			h := newInstallHarness(t)
			path := filepath.Join(h.TargetDir, tc.rootFile)
			opts := Options{
				SourceDir: h.SourceDir,
				Target:    tc.target,
				Mode:      ModeProject,
				TargetDir: h.TargetDir,
			}
			ctx := managed.Context{File: path, HeroVersion: "dev", ProjectDir: h.TargetDir}
			first, err := (managed.Writer{File: path, Sections: defaultSections(opts, path)}).RenderBody(ctx)
			if err != nil {
				t.Fatalf("first RenderBody: %v", err)
			}
			second, err := (managed.Writer{File: path, Sections: defaultSections(opts, path)}).RenderBody(ctx)
			if err != nil {
				t.Fatalf("second RenderBody: %v", err)
			}
			if first != second {
				t.Errorf("RenderBody is non-deterministic for %s — the integrity oracle cannot work.\n--- first ---\n%s\n--- second ---\n%s",
					tc.target, first, second)
			}
		})
	}
}

// TestCheckIntegrity_CleanInstallIsSilent — a healthy install produces zero
// findings, for every target and its nativeInstructionFile-mapped file.
// (AC-3, AC-5)
func TestCheckIntegrity_CleanInstallIsSilent(t *testing.T) {
	for _, tc := range integrityTargets {
		t.Run(tc.name, func(t *testing.T) {
			h, base := newIntegrityHarness(t, tc.target)
			if findings := mustCheckIntegrity(t, h.TargetDir, base); len(findings) != 0 {
				t.Errorf("expected zero findings on a clean %s install, got %+v", tc.target, findings)
			}
			// Determinism at the CheckIntegrity level: a second run must
			// also be silent.
			if findings := mustCheckIntegrity(t, h.TargetDir, base); len(findings) != 0 {
				t.Errorf("second run not silent: %+v", findings)
			}
		})
	}
}

// TestCheckIntegrity_DetectsGuttedRegion — the original incident: the
// managed region collapses to the pointer-only stub. Must report damaged,
// naming the missing sections and the exact repair command. (AC-1, AC-4,
// AC-5)
func TestCheckIntegrity_DetectsGuttedRegion(t *testing.T) {
	const stub = "## Project snapshot\n\nProject shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md)."
	for _, tc := range integrityTargets {
		t.Run(tc.name, func(t *testing.T) {
			h, base := newIntegrityHarness(t, tc.target)
			damageRegion(t, h, tc.rootFile, stub)

			findings := mustCheckIntegrity(t, h.TargetDir, base)
			if len(findings) != 1 {
				t.Fatalf("expected exactly one finding, got %+v", findings)
			}
			f := findings[0]
			if f.Kind != IntegrityDamaged {
				t.Errorf("kind: got %v want IntegrityDamaged", f.Kind)
			}
			if f.File != tc.rootFile {
				t.Errorf("file: got %q want %q (nativeInstructionFile mapping)", f.File, tc.rootFile)
			}
			if len(f.MissingSections) == 0 {
				t.Fatal("expected MissingSections to name the gutted sections")
			}
			joined := strings.Join(f.MissingSections, ", ")
			if !strings.Contains(joined, "Hero Binary & MCP Surface") {
				t.Errorf("MissingSections should include the operational-guidance section, got: %s", joined)
			}
			if strings.Contains(joined, "Project snapshot") {
				t.Errorf("Project snapshot survived the gutting and must not be reported missing, got: %s", joined)
			}
			wantCmd := "hero install project . --target " + string(f.Target)
			if f.RepairCmd != wantCmd {
				t.Errorf("repair command: got %q want %q", f.RepairCmd, wantCmd)
			}
		})
	}
}

// TestCheckIntegrity_DetectsStaleBody — one mutated line inside the region
// (sections all still present) reports stale, not damaged. (AC-2)
func TestCheckIntegrity_DetectsStaleBody(t *testing.T) {
	const anchor = "Finish the closing gate before yielding"
	for _, tc := range integrityTargets {
		t.Run(tc.name, func(t *testing.T) {
			h, base := newIntegrityHarness(t, tc.target)
			path := filepath.Join(h.TargetDir, tc.rootFile)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), anchor) {
				t.Fatalf("fixture missing anchor line %q", anchor)
			}
			mutated := strings.Replace(string(data), anchor, "Yield whenever you feel like it", 1)
			if err := os.WriteFile(path, []byte(mutated), 0o644); err != nil {
				t.Fatal(err)
			}

			findings := mustCheckIntegrity(t, h.TargetDir, base)
			if len(findings) != 1 {
				t.Fatalf("expected exactly one finding, got %+v", findings)
			}
			f := findings[0]
			if f.Kind != IntegrityStale {
				t.Errorf("kind: got %v want IntegrityStale (all sections present)", f.Kind)
			}
			if f.File != tc.rootFile {
				t.Errorf("file: got %q want %q", f.File, tc.rootFile)
			}
			if len(f.MissingSections) != 0 {
				t.Errorf("stale finding should have no MissingSections, got %v", f.MissingSections)
			}
		})
	}
}

// TestCheckIntegrity_SilentOnNeverInstalledTarget — instruction files for
// targets never installed produce no findings, even when the file exists
// on disk. (AC-6)
func TestCheckIntegrity_SilentOnNeverInstalledTarget(t *testing.T) {
	t.Run("no install at all", func(t *testing.T) {
		h := newInstallHarness(t)
		if err := os.WriteFile(filepath.Join(h.TargetDir, "AGENTS.md"),
			[]byte("# My project\n\nHand-written agent notes, no Hero anywhere.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if findings := mustCheckIntegrity(t, h.TargetDir, Options{SourceDir: h.SourceDir}); len(findings) != 0 {
			t.Errorf("expected zero findings with no installed targets, got %+v", findings)
		}
	})

	t.Run("claude installed, user AGENTS.md untouched", func(t *testing.T) {
		h, base := newIntegrityHarness(t, TargetClaude)
		// A user-authored AGENTS.md with no managed region: no non-claude
		// target is installed, so this file must not be inspected at all —
		// not even for a missing region.
		if err := os.WriteFile(filepath.Join(h.TargetDir, "AGENTS.md"),
			[]byte("# My project\n\nHand-written agent notes, no Hero region.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if findings := mustCheckIntegrity(t, h.TargetDir, base); len(findings) != 0 {
			t.Errorf("expected zero findings (AGENTS.md target never installed), got %+v", findings)
		}
	})
}

// TestCheckIntegrity_IgnoresUserContentOutsideMarkers — prose outside the
// managed markers is never compared, never flagged, never modified. (AC-7)
func TestCheckIntegrity_IgnoresUserContentOutsideMarkers(t *testing.T) {
	for _, tc := range integrityTargets {
		t.Run(tc.name, func(t *testing.T) {
			h, base := newIntegrityHarness(t, tc.target)
			path := filepath.Join(h.TargetDir, tc.rootFile)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			region := managed.FindManagedRegion(string(data))
			if !region.Present {
				t.Fatalf("fixture %s has no managed region", tc.rootFile)
			}
			edited := "User prose ABOVE the markers.\n\n" +
				string(data[:region.StartIdx]) +
				string(data[region.StartIdx:region.EndIdx]) +
				string(data[region.EndIdx:]) +
				"\nUser prose BELOW the markers.\n"
			if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
				t.Fatal(err)
			}

			if findings := mustCheckIntegrity(t, h.TargetDir, base); len(findings) != 0 {
				t.Errorf("user content outside markers must not produce findings, got %+v", findings)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != edited {
				t.Errorf("%s was modified by CheckIntegrity — it must be read-only", tc.rootFile)
			}
		})
	}
}

// TestCheckIntegrity_IgnoresVersionStampDrift — the marker's v= stamp
// records the writing binary's version; a version bump with an identical
// body must be silent. (AC-8)
func TestCheckIntegrity_IgnoresVersionStampDrift(t *testing.T) {
	for _, tc := range integrityTargets {
		t.Run(tc.name, func(t *testing.T) {
			h, base := newIntegrityHarness(t, tc.target)
			path := filepath.Join(h.TargetDir, tc.rootFile)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			const oldMarker = "<!-- hero:managed-start v=dev -->"
			if !strings.Contains(string(data), oldMarker) {
				t.Fatalf("fixture %s lacks the v=dev start marker", tc.rootFile)
			}
			bumped := strings.Replace(string(data), oldMarker, "<!-- hero:managed-start v=v99.0.0 -->", 1)
			if err := os.WriteFile(path, []byte(bumped), 0o644); err != nil {
				t.Fatal(err)
			}

			if findings := mustCheckIntegrity(t, h.TargetDir, base); len(findings) != 0 {
				t.Errorf("v= stamp drift with identical body must be silent, got %+v", findings)
			}
		})
	}
}

// TestCheckIntegrity_MissingRegion — the file exists for an installed
// target but carries no managed region at all → damaged. (AC-10)
func TestCheckIntegrity_MissingRegion(t *testing.T) {
	for _, tc := range integrityTargets {
		t.Run(tc.name, func(t *testing.T) {
			h, base := newIntegrityHarness(t, tc.target)
			path := filepath.Join(h.TargetDir, tc.rootFile)
			if err := os.WriteFile(path,
				[]byte("# "+tc.rootFile+"\n\nAll Hero markers and content stripped.\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			findings := mustCheckIntegrity(t, h.TargetDir, base)
			if len(findings) != 1 {
				t.Fatalf("expected exactly one finding, got %+v", findings)
			}
			f := findings[0]
			if f.Kind != IntegrityDamaged {
				t.Errorf("kind: got %v want IntegrityDamaged", f.Kind)
			}
			if f.File != tc.rootFile {
				t.Errorf("file: got %q want %q", f.File, tc.rootFile)
			}
			if !strings.Contains(f.RepairCmd, "hero install project . --target ") {
				t.Errorf("repair command malformed: %q", f.RepairCmd)
			}
		})
	}
}

// TestCheckIntegrity_WritesNothing — CheckIntegrity must not write a single
// byte anywhere under the project, healthy or damaged. (AC-9)
func TestCheckIntegrity_WritesNothing(t *testing.T) {
	for _, tc := range integrityTargets {
		t.Run(tc.name, func(t *testing.T) {
			h, base := newIntegrityHarness(t, tc.target)
			// Damage the file so the finding-producing path runs too.
			damageRegion(t, h, tc.rootFile, "## Project snapshot\n\nProject shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md).")

			before := snapshotTree(t, h.TargetDir)
			if findings := mustCheckIntegrity(t, h.TargetDir, base); len(findings) != 1 {
				t.Fatalf("fixture should produce one finding, got %+v", findings)
			}
			after := snapshotTree(t, h.TargetDir)

			if len(before) != len(after) {
				t.Fatalf("file count changed: %d -> %d", len(before), len(after))
			}
			for path, b := range before {
				a, ok := after[path]
				if !ok {
					t.Errorf("%s disappeared during CheckIntegrity", path)
					continue
				}
				if a.modTime != b.modTime || a.content != b.content {
					t.Errorf("%s was modified during CheckIntegrity", path)
				}
			}
		})
	}
}

type treeEntry struct {
	modTime int64
	content string
}

// snapshotTree records every regular file's mtime and bytes under root.
func snapshotTree(t *testing.T, root string) map[string]treeEntry {
	t.Helper()
	out := map[string]treeEntry{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[path] = treeEntry{modTime: info.ModTime().UnixNano(), content: string(data)}
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot tree: %v", err)
	}
	return out
}

// TestCheckIntegrity_FreshCloneWithoutInstallState — install-state.json is
// gitignored, so on a fresh clone PreviouslyInstalledTargets returns nil.
// The union with InferInstalledTargets must keep resolving the installed
// set from on-disk content, and a gutted region must still be detected.
// Guards design decision 5's load-bearing union.
func TestCheckIntegrity_FreshCloneWithoutInstallState(t *testing.T) {
	for _, tc := range integrityTargets {
		t.Run(tc.name, func(t *testing.T) {
			if tc.target == TargetCopilot {
				// PRE-EXISTING PROBE GAP, not an integrity-check bug:
				// targetLayouts probes .github/copilot/, a legacy location
				// the modern copilot install deliberately no longer creates
				// (see TestHarness_SmokeCopilot), so InferInstalledTargets —
				// and hero upgrade's own detection, which shares it — cannot
				// infer copilot from the filesystem. On a fresh clone with
				// no install-state.json, copilot resolves as not-installed
				// and the check stays silent. Fixing the probe registry is
				// a separate change (it also drives satellites).
				t.Skipf("copilot cannot be inferred from disk (targetLayouts probes legacy .github/copilot/)")
			}
			h, base := newIntegrityHarness(t, tc.target)
			statePath := filepath.Join(h.TargetDir, ".hero", "install-state.json")
			if err := os.Remove(statePath); err != nil {
				t.Fatalf("remove install-state.json: %v", err)
			}
			if got := PreviouslyInstalledTargets(h.TargetDir); len(got) != 0 {
				t.Fatalf("fixture broken: persisted targets should be gone, got %v", got)
			}

			// Clean install still resolves and stays silent.
			if findings := mustCheckIntegrity(t, h.TargetDir, base); len(findings) != 0 {
				t.Fatalf("fresh-clone healthy install must be silent, got %+v", findings)
			}

			// And a gutted region is still detected via the inferred set.
			damageRegion(t, h, tc.rootFile, "## Project snapshot\n\nProject shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md).")
			findings := mustCheckIntegrity(t, h.TargetDir, base)
			if len(findings) != 1 || findings[0].Kind != IntegrityDamaged {
				t.Errorf("gutted region must be detected without install-state.json, got %+v", findings)
			}
		})
	}
}

// TestCheckIntegrity_MultiTargetSharedAgentsMdIsSilent — several non-claude
// targets share AGENTS.md and installs are last-writer-wins (auto-sync), so
// the on-disk body legitimately matches exactly one of the group's
// renderings (codex appends its workflow subsection; the others don't).
// A healthy multi-target install must be silent regardless of which target
// wrote last — per-target strict equality would false-positive here.
func TestCheckIntegrity_MultiTargetSharedAgentsMdIsSilent(t *testing.T) {
	orders := []struct {
		name  string
		first Target
		last  Target
	}{
		{"codex then opencode", TargetCodex, TargetOpenCode},
		{"opencode then codex", TargetOpenCode, TargetCodex},
	}
	for _, tc := range orders {
		t.Run(tc.name, func(t *testing.T) {
			h := newInstallHarness(t)
			if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
				t.Fatal(err)
			}
			h.Run(tc.first, nil)
			h.Run(tc.last, nil)

			if findings := mustCheckIntegrity(t, h.TargetDir, Options{SourceDir: h.SourceDir}); len(findings) != 0 {
				t.Errorf("healthy %s install must be silent, got %+v", tc.name, findings)
			}
		})
	}
}
