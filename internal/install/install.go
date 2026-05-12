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

	// ForceManagedRegion overrides the safety check that refuses to
	// regenerate AGENTS.md / config-file managed regions when the user
	// has edited inside the markers. With this flag set, the regenerate
	// happens anyway and the user's in-region edits are lost.
	ForceManagedRegion bool

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

// Run performs the installation.
func Run(opts Options) (*Result, error) {
	// Materialize the canonical .hero/{agents,commands,skills}/ tree first
	// (project mode only). Each harness target's content dirs then
	// symlink-or-render against this canonical tree. Migration callers
	// pass SkipCanonicalRender to preserve disk-detected winner content
	// they just promoted to canonical.
	canonicalResult := &Result{}
	if !opts.SkipCanonicalRender {
		if err := installCanonical(opts, canonicalResult); err != nil {
			return nil, err
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

	// Roll the canonical-install actions into the returned result so
	// callers see the full operation list.
	if result == nil {
		result = &Result{}
	}
	result.Copied = append(canonicalResult.Copied, result.Copied...)
	result.Merged = append(canonicalResult.Merged, result.Merged...)
	result.Skipped = append(canonicalResult.Skipped, result.Skipped...)

	// Register MCP server in agent config
	if mcpErr := RegisterMCP(opts.Target, opts); mcpErr != nil {
		// Non-fatal: log but don't fail the install
		fmt.Printf("  warning: could not register MCP server: %v\n", mcpErr)
	}

	// Register project in the global daemon registry (~/.hero/projects.json)
	if opts.Mode == ModeProject && opts.TargetDir != "" {
		if regErr := RegisterProject(opts.TargetDir, opts.DryRun); regErr != nil {
			// Non-fatal: log but don't fail the install
			fmt.Printf("  warning: could not register project in daemon registry: %v\n", regErr)
		}
	}

	// Stamp version and file checksums
	if opts.Mode == ModeProject && opts.TargetDir != "" && !opts.DryRun {
		StampInstallVersion(opts, result)
		// Record per-target install-state. Under P2, content dirs are
		// symlinks when the host supports them, otherwise rendered copies.
		mode := "rendered"
		if hostSupportsSymlinks() {
			mode = "symlink"
		}
		RecordTargetInstall(opts, mode)
	}

	return result, nil
}
