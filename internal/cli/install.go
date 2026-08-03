package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/cli/prompt"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/workspace"
	"github.com/spf13/cobra"
)

// emitJSON marshals the given payload to stdout as a JSON object. If
// originalErr is non-nil, returns it unchanged so the CLI propagates
// the proper exit status to the shell. Marshal errors are returned
// directly.
func emitJSON(payload any, originalErr error) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return err
	}
	return originalErr
}

// emitInstallJSON runs fn with stdout silenced, then emits exactly one
// InstallJSONOutput on stdout with the error field populated from fn's return
// under the stable code. It honors the --json stdout contract for the install
// short-circuits (satellite, repair, migrate) that return before the main
// install body's JSON handling — without it, any early-returning path prints
// human text and no parseable object. fn's error is returned unchanged so the
// CLI still exits nonzero on failure.
func emitInstallJSON(mode install.Mode, targetDir, version, code string, fn func() error) error {
	start := time.Now()
	var err error
	silenceStdout(func() { err = fn() })
	return emitJSON(install.InstallJSONOutput{
		Target:     installTarget,
		Mode:       string(mode),
		TargetDir:  targetDir,
		Version:    version,
		DurationMs: time.Since(start).Milliseconds(),
		Error:      install.NewJSONError(code, err),
	}, err)
}

// silenceStdout redirects os.Stdout for the duration of fn, discarding
// anything written. Used by --json modes to ensure ONLY the structured
// JSON payload reaches the caller, even if a deep-helper still uses
// fmt.Printf instead of the opts.Quiet-aware progressf.
func silenceStdout(fn func()) {
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		fn()
		return
	}
	os.Stdout = w
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := r.Read(buf); err != nil {
				break
			}
		}
		close(done)
	}()
	defer func() {
		w.Close()
		<-done
		os.Stdout = orig
	}()
	fn()
}

var installCmd = &cobra.Command{
	Use:   "install <project|global> [path]",
	Short: "Install hero agents, commands, and skills to a target tool",
	Long: `Copies hero content into the target tool's expected directory structure.

Install is harness-native: each target gets only the root instruction file it
natively reads — CLAUDE.md for claude, AGENTS.md for every other target
(codex, opencode, cursor, copilot, generic). Installing multiple targets that
include claude produces both files with the same Hero-managed body. The
installed target set is recorded in .hero/install-state.json so 'hero upgrade'
stays faithful to what was installed.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runInstall,
}

var (
	installTarget          string
	installOnlyTarget      bool
	installForce           bool
	installDryRun          bool
	installWorkspace       string
	installDomain          string
	installForceRoot       bool
	installRepair          bool
	installNoTouchClaudeMd bool
	installMigrate         bool
	installJSON            bool
	installNoHooks         bool
	installPruneOrphans    bool
)

// installTargets is the canonical set of harness targets `hero install`
// accepts, in the order they are offered.
//
// The --target flag help and the interactive picker are both built from it, so
// the advertised set, the validated set, and the offered set cannot drift
// apart — a drift that has bitten this surface before, when `hero uninstall`
// accepted only four of the six.
var installTargets = []string{
	string(install.TargetOpenCode),
	string(install.TargetCursor),
	string(install.TargetClaude),
	string(install.TargetCopilot),
	string(install.TargetCodex),
	string(install.TargetGeneric),
}

func init() {
	installCmd.Flags().StringVar(&installTarget, "target", "", "target tool ("+strings.Join(installTargets, "|")+")")
	installCmd.Flags().BoolVar(&installOnlyTarget, "only-target", false, "install ONLY the named --target; skip auto-sync of any other detected harnesses in the same project")
	installCmd.Flags().BoolVar(&installForce, "force", false, "overwrite existing files")
	installCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "show what would be copied")
	installCmd.Flags().StringVar(&installWorkspace, "workspace", "", "sub-folder workspace path — writes MCP config there pointing at the project root (e.g. services/auth)")
	installCmd.Flags().StringVar(&installDomain, "domain", "", "domain pack to install (default: from hero.json or engineering)")
	installCmd.Flags().BoolVar(&installForceRoot, "root", false, "force root install even when an ancestor already has .hero/ (creates a nested workspace; rare)")
	installCmd.Flags().BoolVar(&installRepair, "repair", false, "skip install; verify and repair existing satellite symlinks/markers and reconcile against subprojects.json")
	installCmd.Flags().BoolVar(&installNoTouchClaudeMd, "no-touch-claude-md", false, "skip CLAUDE.md handling entirely (Claude Code won't see Hero content via CLAUDE.md, but other harnesses still get it via AGENTS.md)")
	installCmd.Flags().BoolVar(&installMigrate, "migrate", false, "auto-detect installed harness targets, reconcile drifted copies (newest mtime wins), promote to canonical, and re-install each target as symlinks pointing at canonical")
	installCmd.Flags().BoolVar(&installJSON, "json", false, "emit a single JSON result object on stdout instead of human-readable progress output (for programmatic consumers like a Hero-native client)")
	installCmd.Flags().BoolVar(&installNoHooks, "no-hooks", false, "skip installing the pre-commit hook (the hook is otherwise self-installed on first install when no managed block exists)")
	installCmd.Flags().BoolVar(&installPruneOrphans, "prune-orphaned-instruction-files", false, "after install, delete a root instruction file (AGENTS.md/CLAUDE.md) whose target is not installed AND whose entire content is Hero-managed; files with any user content are never deleted")
}

func runInstall(cmd *cobra.Command, args []string) error {
	// In --json mode a machine consumer parses stdout; cobra's usage
	// dump on stderr is noise — the JSON error object carries the
	// failure.
	if installJSON {
		cmd.SilenceUsage = true
	}

	mode := install.Mode(args[0])
	if mode != install.ModeProject && mode != install.ModeGlobal {
		return fmt.Errorf("mode must be 'project' or 'global', got %q", args[0])
	}

	var targetDir string
	if mode == install.ModeProject {
		if len(args) < 2 {
			return fmt.Errorf("project mode requires a target path")
		}
		targetDir = args[1]
	}

	binaryVersion := rootCmd.Version
	if binaryVersion == "" {
		binaryVersion = "dev"
	}

	// --migrate is preserved as a backward-compatibility entry point.
	// Under render-direct-install, every regular install already runs
	// legacy cleanup (removes `.hero/{agents,commands,skills}/` mirror,
	// removes harness symlinks pointing at it). So --migrate just runs
	// install with auto-sync — picking up every detected target — and
	// the cleanup happens transparently.
	if installMigrate {
		if mode != install.ModeProject || targetDir == "" {
			modeErr := fmt.Errorf("--migrate requires project mode with an explicit target path")
			if installJSON {
				return emitInstallJSON(mode, targetDir, binaryVersion, "migrate_failed",
					func() error { return modeErr })
			}
			return modeErr
		}
		// The note is human progress; in --json mode it would corrupt the
		// single-object stdout contract, so route it to stderr instead.
		if installJSON {
			fmt.Fprintln(os.Stderr, "Note: `--migrate` is now equivalent to a regular install — legacy")
			fmt.Fprintln(os.Stderr, "symlink/canonical cleanup runs automatically. Continuing.")
		} else {
			fmt.Println("Note: `--migrate` is now equivalent to a regular install — legacy")
			fmt.Println("symlink/canonical cleanup runs automatically. Continuing.")
		}
		// Fall through to the regular install body below by treating
		// the target as required. If no --target flag was passed, detect
		// the first installed harness; auto-sync will refresh the rest.
		if installTarget == "" {
			if detected, derr := install.DetectFirstInstalledTarget(targetDir); derr == nil && detected != "" {
				installTarget = string(detected)
			} else {
				detectErr := fmt.Errorf("--migrate requires either a --target flag or a previously-installed harness in %s", targetDir)
				if installJSON {
					return emitInstallJSON(mode, targetDir, binaryVersion, "migrate_failed",
						func() error { return detectErr })
				}
				return detectErr
			}
		}
	}

	// --repair short-circuits the install body and just runs satellite
	// reconciliation against the workspace containing targetDir.
	if installRepair {
		probeDir := targetDir
		if probeDir == "" {
			probeDir, _ = os.Getwd()
		}
		ws, err := workspace.Locate(probeDir)
		if err != nil {
			locErr := fmt.Errorf("--repair requires an existing workspace: %w", err)
			if installJSON {
				return emitInstallJSON(mode, targetDir, binaryVersion, "repair_failed",
					func() error { return locErr })
			}
			return locErr
		}
		if installJSON {
			return emitInstallJSON(mode, targetDir, binaryVersion, "repair_failed",
				func() error { return runSatelliteRepair(ws, binaryVersion, false) })
		}
		fmt.Printf("Repairing satellites for workspace at %s\n", ws.Root)
		return runSatelliteRepair(ws, binaryVersion, false)
	}

	// Detect whether the requested install location is a subfolder of an
	// existing workspace; if so, satellite-mode unless --root.
	if mode == install.ModeProject && !installForceRoot {
		if ws, err := workspace.Locate(targetDir); err == nil {
			absTarget, _ := filepath.Abs(targetDir)
			if ws.Root != absTarget {
				if installJSON {
					// The satellite path predates --json and prints
					// human progress; wrap it so the contract holds:
					// exactly one JSON object on stdout, error field
					// set, nonzero exit on failure.
					return emitInstallJSON(mode, targetDir, binaryVersion, "install_failed",
						func() error { return runSatelliteInstall(cmd, ws, absTarget, binaryVersion) })
				}
				return runSatelliteInstall(cmd, ws, absTarget, binaryVersion)
			}
		}
	}
	if installForceRoot {
		fmt.Println("Warning: --root forces a root install at this location.")
		fmt.Println("If an ancestor directory already has a Hero workspace, this will create a nested workspace.")
		fmt.Println()
	}

	// Prompt for target if not specified
	target := install.Target(installTarget)
	if target == "" {
		chosen, err := promptTarget(cmd.InOrStdin(), cmd.OutOrStdout(), installJSON)
		if err != nil {
			return err
		}
		target = chosen
	}

	// Resolve domain: flag > hero.json > default
	domain := installDomain
	if domain == "" && mode == install.ModeProject && targetDir != "" {
		if cfg, cfgErr := config.Load(targetDir); cfgErr == nil && cfg.Domain != "" {
			domain = cfg.Domain
		}
	}

	// Resolve the domain FS (defaulting to engineering) and overlay it
	// on top of the universal core layer. Domain wins on file-level
	// path conflicts. Mirrors the precedence in
	// internal/spectypes/loader.go: core first, domain overrides.
	resolvedDomain := domain
	if resolvedDomain == "" {
		resolvedDomain = "engineering"
	}
	domainFS, domainErr := hero.DomainFS(resolvedDomain)
	if domainErr != nil {
		return domainErr
	}
	contentFS := hero.OverlayFS(domainFS, hero.CoreFS())

	opts := install.Options{
		ContentFS:       contentFS,
		Target:          target,
		Mode:            mode,
		TargetDir:       targetDir,
		Force:           installForce,
		DryRun:          installDryRun,
		Version:         binaryVersion,
		Domain:          domain,
		NoTouchClaudeMd: installNoTouchClaudeMd,
		Quiet:           installJSON,
		// Auto-sync detected sibling harnesses so adding one target to
		// an existing multi-harness project refreshes the others too —
		// prevents drift between install moments. Suppressed in
		// --only-target mode (future flag), global mode, and dry-run.
		AutoSyncTargets: mode == install.ModeProject && !installOnlyTarget,
	}

	if opts.DryRun && !installJSON {
		fmt.Printf("target:      %s\n", opts.Target)
		fmt.Printf("mode:        %s\n", opts.Mode)
		fmt.Println()
	}

	start := time.Now()
	var result *install.Result
	var err error
	runOnce := func() {
		result, err = install.Run(opts)
	}

	if installJSON {
		silenceStdout(runOnce)
		return emitJSON(install.InstallJSONOutput{
			Target:     string(opts.Target),
			Mode:       string(opts.Mode),
			TargetDir:  opts.TargetDir,
			Version:    opts.Version,
			Result:     result,
			DurationMs: time.Since(start).Milliseconds(),
			Error:      install.NewJSONError("install_failed", err),
		}, err)
	}
	runOnce()

	if err != nil {
		return err
	}

	if opts.DryRun {
		fmt.Printf("\ndry run complete — no files were written\n")
	} else {
		fmt.Printf("Installed %d files", len(result.Copied))
		if len(result.Merged) > 0 {
			fmt.Printf(", merged %d config(s)", len(result.Merged))
		}
		fmt.Println()
		printHandoffHint(target)
	}

	// Opt-in orphan pruning: after a harness-native install, a legacy
	// Model-B phantom (e.g. an AGENTS.md left by an older Claude-only
	// install) can be removed when its target isn't installed and its whole
	// content is Hero-managed. Never deletes a file with user content.
	if installPruneOrphans && mode == install.ModeProject && targetDir != "" {
		resolved := unionTargets(
			install.PreviouslyInstalledTargets(targetDir),
			install.DetectInstalledTargets(targetDir),
			[]install.Target{target},
		)
		handleOrphanedInstructionFiles(targetDir, resolved, contentFS, binaryVersion, true, installDryRun)
	}

	// Self-heal pre-commit hook install on first run when no managed
	// block exists and the user hasn't opted out. Mirrors the pattern
	// in `hero init` — `hero install` is the more-discoverable setup
	// path for teammates who clone an existing repo. Skips silently
	// outside project mode, in dry-run, when --no-hooks is set, when
	// the managed block is already present, when the user has placed
	// a `.hero/.no-hooks` opt-out sentinel, or when the target dir
	// isn't a git repo. Best-effort: a failure doesn't fail the
	// install.
	if mode == install.ModeProject && !installDryRun && !installNoHooks && targetDir != "" {
		if !preCommitHookInstalled(targetDir) && !hookInstallOptedOut(targetDir) {
			if _, gerr := resolveGitDir(targetDir); gerr == nil {
				if herr := installNextHooksQuiet(targetDir); herr != nil {
					fmt.Fprintf(os.Stderr, "  warning: pre-commit hook install failed: %v\n", herr)
				} else {
					fmt.Println()
					fmt.Println("  Installed pre-commit hook (projected NEXT files will travel with commits).")
					fmt.Println("  Pass --no-hooks next time to skip; to opt out permanently, `touch .hero/.no-hooks`.")
				}
			}
		}
	}

	// After a successful root project install, offer subproject walkthrough.
	if mode == install.ModeProject && !installDryRun && err == nil {
		if ws, locErr := workspace.Locate(targetDir); locErr == nil {
			if walkErr := postRootInstallSubprojectWalk(cmd, ws, binaryVersion); walkErr != nil {
				fmt.Printf("  warning: subproject walkthrough error: %v\n", walkErr)
			}
		}
	}

	// Workspace mode: write MCP config into the sub-folder
	if installWorkspace != "" && mode == install.ModeProject {
		wsPath := installWorkspace
		if !filepath.IsAbs(wsPath) {
			wsPath = filepath.Join(targetDir, wsPath)
		}
		if _, err := os.Stat(wsPath); os.IsNotExist(err) {
			return fmt.Errorf("workspace path does not exist: %s", wsPath)
		}

		wsOpts := install.Options{
			Target:      target,
			Mode:        install.ModeProject,
			TargetDir:   wsPath,
			DryRun:      installDryRun,
			Version:     binaryVersion,
			ProjectRoot: targetDir,
		}
		if err := install.RegisterMCP(target, wsOpts); err != nil {
			return fmt.Errorf("registering MCP in workspace: %w", err)
		}
		if !installDryRun {
			fmt.Printf("Workspace MCP config written to %s (pointing at %s)\n", wsPath, targetDir)
		}
	}

	return nil
}

// printHandoffHint tells the user how cross-session handoff stays fresh on
// their target. Claude Code gets it for free via the Stop/PreCompact hooks
// we just wired into settings.json. Other targets fall back to git
// post-commit, which only fires when a commit happens — so we suggest
// running `hero hooks install` to enable that fallback.
func printHandoffHint(target install.Target) {
	switch target {
	case install.TargetClaude:
		fmt.Println()
		fmt.Println("NEXT.md handoff: wired into Stop + PreCompact hooks in settings.json.")
		fmt.Println("It refreshes automatically every turn — no further setup needed.")
	case install.TargetCodex:
		fmt.Println()
		fmt.Println("NEXT.md handoff: wired into Stop hook in .codex/hooks.json.")
		fmt.Println("It refreshes automatically after every session — no further setup needed.")
		printCodexTrustHint()
	case install.TargetOpenCode:
		fmt.Println()
		fmt.Println("NEXT.md handoff: opencode's hook system needs a TS plugin (deferred).")
		fmt.Println("Run `hero hooks install` to enable the git post-commit fallback,")
		fmt.Println("or use `/handoff` manually before switching tools.")
	default:
		fmt.Println()
		fmt.Println("NEXT.md handoff: this tool has no native end-of-turn hook.")
		fmt.Println("Run `hero hooks install` to enable the git post-commit fallback,")
		fmt.Println("or use `/handoff` manually before switching tools.")
	}
}

// promptTarget resolves the install target when --target was not supplied.
//
// Two deliberate changes from the previous implementation, both of which turn
// a silent wrong answer into a loud failure:
//
//   - It used to return install.TargetOpenCode whenever stdin was not a
//     terminal. So `hero install project .` in CI installed opencode, exited
//     0, and said nothing — the wrong harness, with no signal that a choice
//     had even been made. It now fails fast. Hard constraint 3 ("non-TTY must
//     fail fast, never silently succeed") outranks "do not break what works"
//     here, because this did not work; it quietly did the wrong thing.
//
//   - It used to return install.Target(input) unvalidated, so a typo at the
//     prompt produced a bogus target that only surfaced later and deeper.
//     prompt.Choice rejects anything outside installTargets at entry.
//
// See docs/release-notes/ — this is one of exactly two sanctioned behavior
// changes in this child, and TestSanctionedBreakInstallTargetFailsOnNonTTY
// asserts it positively so it cannot be quietly reverted.
func promptTarget(in io.Reader, out io.Writer, jsonMode bool) (install.Target, error) {
	// --json never prompts: stdout carries a machine-readable result object
	// and a prompt would corrupt it. This generalizes the guard that
	// previously existed only at the subproject-add confirm.
	if jsonMode {
		return "", fmt.Errorf("--target is required with --json: pass --target (%s)",
			strings.Join(installTargets, "|"))
	}
	if !prompt.IsInputTTY(in) {
		return "", fmt.Errorf("no --target given and no terminal available to ask for one: pass --target (%s)",
			strings.Join(installTargets, "|"))
	}

	choice, err := prompt.Choice(in, out, "Install target (default: opencode)", installTargets)
	if err != nil {
		return "", err
	}
	if choice == "" {
		return install.TargetOpenCode, nil
	}
	return install.Target(choice), nil
}

// runSatelliteInstall handles the case where targetDir is inside an
// existing Hero workspace. It materializes a satellite and offers to
// add the subproject to subprojects.json.
func runSatelliteInstall(cmd *cobra.Command, ws *workspace.Workspace, satAbs, version string) error {
	// Compute scope (path relative to root, forward-slash).
	rel, err := filepath.Rel(ws.Root, satAbs)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if rel == "" || rel == "." {
		return fmt.Errorf("satellite path equals workspace root: %s", satAbs)
	}

	fmt.Printf("Detected Hero workspace at %s\n", ws.Root)
	fmt.Printf("Installing as satellite: %s\n\n", rel)

	subs, err := install.LoadSubprojects(ws.HeroDir)
	if err != nil {
		return err
	}

	scope := rel
	if !subs.IsDeclared(rel) {
		// Never prompt in --json mode — the caller is a program, and
		// stdout is reserved for the JSON result object.
		in := cmd.InOrStdin()
		if prompt.IsInputTTY(in) && !installDryRun && !installJSON {
			fmt.Printf("This subfolder is not declared in %s/%s.\n", workspace.HeroDir, install.SubprojectsFile)
			yes, err := prompt.Confirm(in, cmd.OutOrStdout(),
				"Add it as a subproject so teammates pick it up automatically? [y/N] ", false)
			if err != nil {
				return err
			}
			if yes {
				subs.AddSubproject(install.Subproject{Path: rel, Scope: rel})
				if err := install.SaveSubprojects(ws.HeroDir, subs); err != nil {
					return fmt.Errorf("save subprojects.json: %w", err)
				}
				fmt.Printf("Added to %s/%s — commit it to share with your team.\n", workspace.HeroDir, install.SubprojectsFile)
			}
		}
	} else {
		// Declared — use the canonical scope from the manifest.
		for _, sp := range subs.Subprojects {
			if sp.Path == rel {
				if sp.Scope != "" {
					scope = sp.Scope
				}
				break
			}
		}
	}

	if installDryRun {
		fmt.Println("dry run — would materialize satellite for installed targets at root.")
		return nil
	}

	res, err := install.Materialize(install.SatelliteOptions{
		RootDir:      ws.Root,
		SatelliteDir: satAbs,
		Scope:        scope,
		Version:      version,
		Force:        installForce,
	})
	if err != nil {
		return fmt.Errorf("materialize satellite: %w", err)
	}
	if res.Degraded {
		fmt.Println("Symlinks unavailable on this machine — wrote marker only.")
		fmt.Println("Open the workspace root directly for full Hero support, or enable")
		fmt.Println("symlink support and run `hero install --repair`.")
	} else {
		fmt.Printf("Satellite materialized for targets: %v\n", res.Targets)
	}
	entry := install.SatelliteEntry{
		Path:     rel,
		Targets:  targetsAsStrings(res.Targets),
		Degraded: res.Degraded,
	}
	if err := install.RecordSatellite(ws.HeroDir, entry); err != nil {
		return fmt.Errorf("record satellite: %w", err)
	}
	return nil
}

// postRootInstallSubprojectWalk runs after a successful root install to
// offer the candidate-subproject walkthrough. It is best-effort — any
// error is surfaced but does not fail the install itself.
func postRootInstallSubprojectWalk(cmd *cobra.Command, ws *workspace.Workspace, version string) error {
	in := cmd.InOrStdin()
	// The --json guard is new here. This site had no such guard, so a
	// candidate walk could interleave a [y/N] prompt with the single JSON
	// result object a programmatic caller is parsing. AC-10 makes the rule
	// general rather than a per-command accident.
	if installJSON || !prompt.IsInputTTY(in) {
		return nil
	}
	subs, err := install.LoadSubprojects(ws.HeroDir)
	if err != nil {
		return err
	}
	candidates, err := install.DetectCandidates(ws.Root, subs, 4)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return nil
	}
	fmt.Println()
	out := cmd.OutOrStdout()
	yes, err := prompt.Confirm(in, out,
		fmt.Sprintf("Detected %d subproject candidate(s). Walk through them now? [y/N] ", len(candidates)), false)
	if err != nil {
		return err
	}
	if !yes {
		fmt.Println("Skipped — run `hero install satellites` later to walk through them.")
		return nil
	}
	return walkCandidates(ws, subs, candidates, version, in, out)
}
