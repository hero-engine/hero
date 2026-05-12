package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// linking.go — directory-symlink primitives used by per-target installers
// to point harness content directories at the canonical .hero/ tree.
//
// Three modes per harness content dir:
//
//   - **symlink**  (default when host supports it): a directory symlink
//                  from <harness>/<kind> → ../.hero/<kind>. Edits to
//                  .hero/agents/foo.md are visible immediately to every
//                  harness via its symlinked view.
//
//   - **rendered** (fallback when symlinks unavailable, or when the host
//                  is Windows without Developer Mode, or when the harness
//                  is on the symlinks-broken list — Cline): physical
//                  copies in the harness directory. Drift detection lives
//                  in `hero verify-install` (P4 — stubbed via the manifest
//                  written here).
//
//   - **migrate**  (transitional): when the harness dir already exists as
//                  a regular directory from a legacy direct-render
//                  install, the user opts into symlink layout by passing
//                  --force. The dir is removed and replaced with the
//                  symlink.

// hostSymlinkSupport is the cached result of the probe — one boolean per
// process lifetime. Refreshed if the install state file already has a
// recorded probe result (probe is cheap, so we don't bother to skip it
// when state already has the answer; but we cache within a single
// process).
var (
	hostSymlinkProbeOnce sync.Once
	hostSymlinkProbeOK   bool
)

// hostSupportsSymlinks attempts a test symlink in a tempdir and reports
// whether the kernel + filesystem will allow Hero to create directory
// symlinks. The result is cached for the lifetime of the process.
func hostSupportsSymlinks() bool {
	hostSymlinkProbeOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "hero-symlink-probe-")
		if err != nil {
			hostSymlinkProbeOK = false
			return
		}
		defer os.RemoveAll(tmpDir)

		// Create a target file and try to symlink to it.
		target := filepath.Join(tmpDir, "target")
		if err := os.WriteFile(target, []byte("probe"), 0o644); err != nil {
			hostSymlinkProbeOK = false
			return
		}
		link := filepath.Join(tmpDir, "link")
		if err := os.Symlink(target, link); err != nil {
			hostSymlinkProbeOK = false
			return
		}
		hostSymlinkProbeOK = true
	})
	return hostSymlinkProbeOK
}

// linkOrRenderResult describes what linkOrRenderDir did.
type linkOrRenderResult struct {
	Mode string // "symlink", "rendered", "noop"
}

// linkOrRenderDir points a harness content directory at the canonical
// .hero/ tree, preferring a directory symlink and falling back to
// rendered copies when symlinks are unavailable or the harness is on
// the symlinks-broken list.
//
//   - kind:         "agents" / "commands" / "skills"
//   - canonicalDir: absolute path under .hero/ (the symlink target)
//   - harnessDir:   absolute path under the harness root (the symlink path)
//   - nested:       skills layout — when true, content is <name>/SKILL.md
//                   for the rendered fallback. (Symlinks don't care; the
//                   canonical dir already has the right layout.)
//   - forceSymlinkBroken: harness-specific override (Cline) that forces
//                   rendered mode regardless of host capability.
//
// Idempotent: a symlink already pointing at the right canonical path is
// a no-op. A rendered dir at the current version is a no-op (handled by
// copyFileFromFS's skip-when-exists behavior — and on --force, by full
// re-render).
//
// Migration: if harnessDir is a regular directory (legacy direct-render
// install), it gets replaced when --force is set; without --force the
// install errors out with a clear "use --force to migrate to canonical
// layout" message.
func linkOrRenderDir(opts Options, result *Result, kind, canonicalDir, harnessDir string, nested, forceSymlinkBroken bool) (linkOrRenderResult, error) {
	if opts.DryRun {
		progressf(opts, "  link/render %s -> %s\n", kind, harnessDir)
		return linkOrRenderResult{Mode: "dryrun"}, nil
	}

	// If the canonical directory doesn't exist (e.g., project hasn't been
	// `hero init`'d), there's nothing to link to. Fall back to direct
	// rendering from embedded source — preserves legacy behavior for
	// uninitialized projects.
	if _, err := os.Stat(canonicalDir); err != nil {
		if nested {
			return linkOrRenderResult{Mode: "rendered"}, installSkillsNested(opts, result, harnessDir)
		}
		return linkOrRenderResult{Mode: "rendered"}, installFlat(opts, result, kind, harnessDir)
	}

	useSymlink := !forceSymlinkBroken && hostSupportsSymlinks()

	// Inspect what's currently at harnessDir.
	info, statErr := os.Lstat(harnessDir)
	exists := statErr == nil

	if useSymlink {
		// Compute the relative symlink target (preferred — relative makes
		// the workspace portable across paths).
		rel, err := filepath.Rel(filepath.Dir(harnessDir), canonicalDir)
		if err != nil || rel == "" {
			rel = canonicalDir
		}

		// Existing symlink to the right place: no-op.
		if exists && info.Mode()&os.ModeSymlink != 0 {
			currentTarget, err := os.Readlink(harnessDir)
			if err == nil && currentTarget == rel {
				return linkOrRenderResult{Mode: "noop"}, nil
			}
			// Wrong target — replace.
			if err := os.Remove(harnessDir); err != nil {
				return linkOrRenderResult{}, fmt.Errorf("removing stale symlink %s: %w", harnessDir, err)
			}
			exists = false
		}

		// Existing regular directory: only replace if --force.
		if exists && info.IsDir() {
			if !opts.Force {
				return linkOrRenderResult{}, fmt.Errorf("%s is a regular directory from a legacy install; pass --force to migrate to canonical-symlink layout", harnessDir)
			}
			if err := os.RemoveAll(harnessDir); err != nil {
				return linkOrRenderResult{}, fmt.Errorf("removing legacy directory %s: %w", harnessDir, err)
			}
			exists = false
		}

		// Existing non-dir, non-symlink (unexpected): force-remove only
		// with --force.
		if exists {
			if !opts.Force {
				return linkOrRenderResult{}, fmt.Errorf("%s exists and is not a directory or symlink; pass --force to replace", harnessDir)
			}
			if err := os.Remove(harnessDir); err != nil {
				return linkOrRenderResult{}, fmt.Errorf("removing %s: %w", harnessDir, err)
			}
		}

		// Create the parent and the symlink.
		if err := os.MkdirAll(filepath.Dir(harnessDir), 0o755); err != nil {
			return linkOrRenderResult{}, err
		}
		if err := os.Symlink(rel, harnessDir); err != nil {
			// Symlink creation failed despite the probe passing — usually
			// happens when the parent is on a different filesystem with
			// restrictions. Fall through to rendered mode.
			useSymlink = false
		} else {
			result.Copied = append(result.Copied, CopyAction{
				Source: "symlink->" + rel,
				Dest:   harnessDir,
			})
			progressf(opts, "  %s -> %s (symlink -> %s)\n", kind, harnessDir, rel)
			return linkOrRenderResult{Mode: "symlink"}, nil
		}
	}

	// Rendered fallback: copy from embedded source into the harness dir.
	// copyFileFromFS still does the "skip when exists unless --force"
	// dance, which gives reasonable idempotency for repeated re-renders.
	if nested {
		if err := installSkillsNested(opts, result, harnessDir); err != nil {
			return linkOrRenderResult{}, err
		}
	} else {
		if err := installFlat(opts, result, kind, harnessDir); err != nil {
			return linkOrRenderResult{}, err
		}
	}
	return linkOrRenderResult{Mode: "rendered"}, nil
}
