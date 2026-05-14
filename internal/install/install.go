// Package install handles materializing Hero's agents, commands, skills, and
// instruction files into a target harness's expected layout.
//
// The package is decomposed into:
//
//	install.go        — Run dispatcher + Options/Result types (this file)
//	target_<name>.go  — per-harness installer (claude, opencode, cursor, codex,
//	                    copilot, generic)
//	agents_md.go      — AGENTS.md generation + marker-based merge
//	claude_md.go      — CLAUDE.md generation + marker-based merge + Claude paths
//	content.go        — installFlat + installSkillsNested + legacy-flat cleanup
//	files.go          — file copy + JSON merge primitives
//	state.go          — version stamping + install-state.json scaffolding
//
// The single-source-install initiative
// (`.hero/planning/initiatives/single-source-install/`) extends this package
// to a canonical `.hero/{agents,commands,skills}/` tree plus mode-aware
// per-target install (config-redirect / symlink / rendered). The
// decomposition above is the prerequisite refactor for that work.
package install

import (
	"fmt"
	"io/fs"
	"os"
)

// Target represents a supported installation target tool.
type Target string

const (
	TargetOpenCode Target = "opencode"
	TargetCursor   Target = "cursor"
	TargetClaude   Target = "claude"
	TargetCopilot  Target = "copilot"
	TargetCodex    Target = "codex"
	TargetGeneric  Target = "generic"
)

// Mode represents whether installation is project-local or global.
type Mode string

const (
	ModeProject Mode = "project"
	ModeGlobal  Mode = "global"
)

// Options holds all installation parameters.
type Options struct {
	SourceDir   string // path to hero repository root (deprecated, use ContentFS)
	ContentFS   fs.FS  // embedded filesystem with agents/, commands/, skills/
	Target      Target
	Mode        Mode
	TargetDir   string // for project mode: the project path
	Force       bool
	DryRun      bool
	Version     string // hero binary version (for version stamping)
	ProjectRoot string // for workspace mode: the actual project root (where .hero/ lives)
	Domain      string // domain pack to install (default: engineering)

	// NoTouchClaudeMd skips CLAUDE.md handling entirely. User accepts that
	// Claude Code won't see Hero content via CLAUDE.md (other harnesses
	// still get it via AGENTS.md). Niche; for users who want absolute
	// file-immutability semantics on a specific CLAUDE.md.
	NoTouchClaudeMd bool

	// SkipCanonicalRender skips the embedded-source → canonical
	// materialization step. Used by `hero install --migrate`, which has
	// already written disk-detected winner content to canonical and
	// doesn't want subsequent per-target installs to clobber it with
	// freshly-rendered embedded content.
	SkipCanonicalRender bool

	// Quiet suppresses per-file progress prints. Used by --json output
	// modes so the structured result is the only thing on stdout.
	Quiet bool

	// TrustedChecksums maps destination path (relative to TargetDir,
	// using forward slashes) to its SHA-256 checksum from a prior Hero
	// install. When the install pipeline encounters a destination file
	// whose current bytes match the entry here, it treats the file as
	// "Hero-installed at a previous version, safe to update" — even
	// without --force. Used by `hero upgrade` to migrate users from
	// legacy install layouts to the current one without losing
	// user-edited files (those won't be in the trust map; their
	// destination drift is preserved).
	TrustedChecksums map[string]string

	// AutoSyncTargets, when true, makes `hero install --target X` also
	// refresh any other installed harness targets detected in the same
	// project. Prevents drift between harnesses when the binary version
	// changes between install moments. Set by the install CLI command
	// by default; recursive auto-sync calls clear this flag so we don't
	// loop.
	AutoSyncTargets bool
}

// sourceFS returns the filesystem to read content from.
// Prefers ContentFS if set, falls back to os.DirFS(SourceDir).
func (o Options) sourceFS() fs.FS {
	if o.ContentFS != nil {
		return o.ContentFS
	}
	if o.SourceDir != "" {
		return os.DirFS(o.SourceDir)
	}
	return nil
}

// Result tracks what was done during installation.
type Result struct {
	Copied  []CopyAction `json:"copied"`
	Merged  []string     `json:"merged"`
	Skipped []string     `json:"skipped"`
}

// CopyAction records a single file copy.
type CopyAction struct {
	Source string `json:"source"`
	Dest   string `json:"dest"`
}

// Run performs the installation. Each per-target installer renders
// content directly from the embedded source to the harness's documented
// destination paths — no canonical-on-disk mirror, no symlinks.
//
// Project mode (the common case) also cleans up legacy install artifacts
// from earlier Hero architectures: `.hero/{agents,commands,skills}/`
// canonical dirs and harness-dir symlinks pointing at them are removed
// when their content is detectably Hero-authored.
func Run(opts Options) (*Result, error) {
	// Legacy migration: remove `.hero/{agents,commands,skills}/` canonical
	// mirror and any harness symlinks pointing at it. Idempotent —
	// no-op after the first install on the new architecture.
	if opts.Mode == ModeProject && opts.TargetDir != "" && !opts.SkipCanonicalRender {
		if err := cleanupLegacyCanonicalSymlinks(opts, opts.TargetDir); err != nil {
			fmt.Printf("  warning: legacy canonical/symlink cleanup: %v\n", err)
		}
	}

	var result *Result
	var err error

	switch opts.Target {
	case TargetOpenCode:
		result, err = runOpenCode(opts)
	case TargetCursor:
		result, err = runCursor(opts)
	case TargetClaude:
		result, err = runClaude(opts)
	case TargetCopilot:
		result, err = runCopilot(opts)
	case TargetCodex:
		result, err = runCodex(opts)
	case TargetGeneric:
		result, err = runGeneric(opts)
	default:
		return nil, fmt.Errorf("unknown target %q; supported targets: opencode, cursor, claude, copilot, codex, generic", opts.Target)
	}

	if err != nil {
		return result, err
	}

	if result == nil {
		result = &Result{}
	}

	if mcpErr := RegisterMCP(opts.Target, opts); mcpErr != nil {
		fmt.Printf("  warning: could not register MCP server: %v\n", mcpErr)
	}

	if opts.Mode == ModeProject && opts.TargetDir != "" {
		if regErr := RegisterProject(opts.TargetDir, opts.DryRun); regErr != nil {
			fmt.Printf("  warning: could not register project in daemon registry: %v\n", regErr)
		}
	}

	if opts.Mode == ModeProject && opts.TargetDir != "" && !opts.DryRun {
		StampInstallVersion(opts, result)
		RecordTargetInstall(opts, "rendered")
	}

	// Auto-sync: refresh other installed harnesses so they stay at the
	// same binary version (drift prevention). Suppressed when the caller
	// is itself an auto-sync recursive call.
	if opts.AutoSyncTargets && opts.Mode == ModeProject && opts.TargetDir != "" && !opts.DryRun {
		if err := autoSyncSiblings(opts, result); err != nil {
			fmt.Printf("  warning: auto-sync sibling targets: %v\n", err)
		}
	}

	return result, nil
}
