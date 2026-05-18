package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	hero "github.com/hero-engine/hero"
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
	Long:  "Copies hero content into the target tool's expected directory structure.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runInstall,
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
)

func init() {
	installCmd.Flags().StringVar(&installTarget, "target", "", "target tool (opencode|cursor|claude|copilot|codex|generic)")
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
}

func runInstall(cmd *cobra.Command, args []string) error {
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
			return fmt.Errorf("--migrate requires project mode with an explicit target path")
		}
		fmt.Println("Note: `--migrate` is now equivalent to a regular install — legacy")
		fmt.Println("symlink/canonical cleanup runs automatically. Continuing.")
		// Fall through to the regular install body below by treating
		// the target as required. If no --target flag was passed, detect
		// the first installed harness; auto-sync will refresh the rest.
		if installTarget == "" {
			if detected, derr := install.DetectFirstInstalledTarget(targetDir); derr == nil && detected != "" {
				installTarget = string(detected)
			} else {
				return fmt.Errorf("--migrate requires either a --target flag or a previously-installed harness in %s", targetDir)
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
			return fmt.Errorf("--repair requires an existing workspace: %w", err)
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
				return runSatelliteInstall(ws, absTarget, binaryVersion)
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
		target = promptTarget()
	}

	// Resolve domain: flag > hero.json > default
	domain := installDomain
	if domain == "" && mode == install.ModeProject && targetDir != "" {
		if cfg, cfgErr := config.Load(targetDir); cfgErr == nil && cfg.Domain != "" {
			domain = cfg.Domain
		}
	}

	contentFS := hero.ContentFS()
	if domain != "" {
		domainFS, domainErr := hero.DomainFS(domain)
		if domainErr != nil {
			return domainErr
		}
		contentFS = domainFS
	}

	opts := install.Options{
		ContentFS:          contentFS,
		Target:             target,
		Mode:               mode,
		TargetDir:          targetDir,
		Force:              installForce,
		DryRun:             installDryRun,
		Version:            binaryVersion,
		Domain:             domain,
		NoTouchClaudeMd:    installNoTouchClaudeMd,
		Quiet:              installJSON,
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
			if walkErr := postRootInstallSubprojectWalk(ws, binaryVersion); walkErr != nil {
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

func promptTarget() install.Target {
	if !isTerminal() {
		return install.TargetOpenCode
	}

	fmt.Print("Install target [opencode|cursor|claude|copilot|codex|generic] (default: opencode): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return install.TargetOpenCode
	}

	return install.Target(input)
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// runSatelliteInstall handles the case where targetDir is inside an
// existing Hero workspace. It materializes a satellite and offers to
// add the subproject to subprojects.json.
func runSatelliteInstall(ws *workspace.Workspace, satAbs, version string) error {
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
		add := isTerminal() && !installDryRun
		if add {
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("This subfolder is not declared in %s/%s.\n", workspace.HeroDir, install.SubprojectsFile)
			fmt.Print("Add it as a subproject so teammates pick it up automatically? [y/N] ")
			line, _ := reader.ReadString('\n')
			ans := strings.TrimSpace(strings.ToLower(line))
			if ans == "y" || ans == "yes" {
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
func postRootInstallSubprojectWalk(ws *workspace.Workspace, version string) error {
	if !isTerminal() {
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
	fmt.Printf("Detected %d subproject candidate(s). Walk through them now? [y/N] ", len(candidates))
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	ans := strings.TrimSpace(strings.ToLower(line))
	if ans != "y" && ans != "yes" {
		fmt.Println("Skipped — run `hero install satellites` later to walk through them.")
		return nil
	}
	return walkCandidates(ws, subs, candidates, version, os.Stdin, os.Stdout)
}
