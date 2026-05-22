package cli

import (
	"fmt"

	"github.com/hero-engine/hero/internal/hooks"
	"github.com/spf13/cobra"
)

// host_hooks.go — `--host=claude|codex|all` flag plumbing for
// `hero hooks install/uninstall/status`. The git-hook installer in
// hooks.go is untouched; this file adds the host-tool path that wires
// the SessionStart{compact} entry into Claude Code's settings.json
// (and Codex's equivalent, once the Codex installer ships).
//
// Wiring strategy: rather than fork the existing runHooksInstall et
// al, the cobra flags + dispatch live here and the existing
// {hooks*Cmd} have a PreRunE / RunE wrapper added in init() so that
// `hero hooks install` (no flag) keeps today's git-hook-only behavior.

const (
	hostFlagNone   = ""
	hostFlagClaude = "claude"
	hostFlagCodex  = "codex"
	hostFlagAll    = "all"
)

var hostHookHostFlag string

func init() {
	for _, c := range []*cobra.Command{hooksInstallCmd, hooksUninstallCmd, hooksStatusCmd} {
		c.Flags().StringVar(&hostHookHostFlag, "host", hostFlagNone,
			"target host tool: claude | codex | all (default: git hooks only)")
	}
	// Wrap the existing RunE so `--host` triggers the host-tool path.
	// We can't easily compose two RunE on a single command in cobra, so
	// rebuild the RunE to dispatch by flag value, falling through to
	// the original git-hook behavior when --host is empty.
	hooksInstallCmd.RunE = wrapWithHostHook(runHooksInstall, runHostHooksInstall)
	hooksUninstallCmd.RunE = wrapWithHostHook(runHooksUninstall, runHostHooksUninstall)
	hooksStatusCmd.RunE = wrapWithHostHook(runHooksStatus, runHostHooksStatus)
}

// wrapWithHostHook returns a RunE that calls the original git-hook
// runner unless --host is set, in which case it routes to the host-
// hook runner. When --host=all, both run sequentially.
func wrapWithHostHook(gitRun, hostRun func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		switch hostHookHostFlag {
		case hostFlagNone:
			return gitRun(cmd, args)
		case hostFlagAll:
			if err := gitRun(cmd, args); err != nil {
				return err
			}
			return hostRun(cmd, args)
		default:
			return hostRun(cmd, args)
		}
	}
}

func runHostHooksInstall(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	w := cmd.OutOrStdout()
	switch hostHookHostFlag {
	case hostFlagClaude, hostFlagAll:
		installed, err := hooks.InstallClaudeCompactHandoff(projectRoot)
		if err != nil {
			return fmt.Errorf("claude: %w", err)
		}
		if installed {
			fmt.Fprintf(w, "  installed  claude SessionStart{compact}\n")
		} else {
			fmt.Fprintf(w, "  skipped    claude SessionStart{compact} (already installed)\n")
		}
		if hostHookHostFlag != hostFlagAll {
			return nil
		}
		fallthrough
	case hostFlagCodex:
		installed, err := hooks.InstallCodexCompactHandoff(projectRoot)
		if err != nil {
			return fmt.Errorf("codex: %w", err)
		}
		if installed {
			fmt.Fprintf(w, "  installed  codex SessionStart{compact}\n")
		} else {
			fmt.Fprintf(w, "  skipped    codex SessionStart{compact} (already installed)\n")
		}
		if !hooks.CodexFeatureFlagEnabled() {
			fmt.Fprintf(w,
				"  warning    codex hooks are off by default — add `codex_hooks = true` under [features] in ~/.codex/config.toml to enable\n")
		}
		fmt.Fprintf(w,
			"  note       codex will prompt you to trust this project's .codex/ config on first run\n")
	}
	return nil
}

func runHostHooksUninstall(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	w := cmd.OutOrStdout()
	switch hostHookHostFlag {
	case hostFlagClaude, hostFlagAll:
		removed, err := hooks.UninstallClaudeCompactHandoff(projectRoot)
		if err != nil {
			return fmt.Errorf("claude: %w", err)
		}
		if removed {
			fmt.Fprintf(w, "  removed    claude SessionStart{compact}\n")
		} else {
			fmt.Fprintf(w, "  skipped    claude SessionStart{compact} (not installed)\n")
		}
		if hostHookHostFlag != hostFlagAll {
			return nil
		}
		fallthrough
	case hostFlagCodex:
		removed, err := hooks.UninstallCodexCompactHandoff(projectRoot)
		if err != nil {
			return fmt.Errorf("codex: %w", err)
		}
		if removed {
			fmt.Fprintf(w, "  removed    codex SessionStart{compact}\n")
		} else {
			fmt.Fprintf(w, "  skipped    codex SessionStart{compact} (not installed)\n")
		}
	}
	return nil
}

func runHostHooksStatus(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	w := cmd.OutOrStdout()
	switch hostHookHostFlag {
	case hostFlagClaude, hostFlagAll:
		ok, err := hooks.ClaudeCompactHandoffStatus(projectRoot)
		if err != nil {
			return fmt.Errorf("claude: %w", err)
		}
		state := "no"
		if ok {
			state = "yes"
		}
		fmt.Fprintf(w, "  claude SessionStart{compact}: %s\n", state)
		if hostHookHostFlag != hostFlagAll {
			return nil
		}
		fallthrough
	case hostFlagCodex:
		ok, err := hooks.CodexCompactHandoffStatus(projectRoot)
		if err != nil {
			return fmt.Errorf("codex: %w", err)
		}
		state := "no"
		if ok {
			state = "yes"
		}
		fmt.Fprintf(w, "  codex SessionStart{compact}: %s", state)
		if ok && !hooks.CodexFeatureFlagEnabled() {
			fmt.Fprintf(w, " (warning: codex_hooks feature flag not enabled in ~/.codex/config.toml)")
		}
		fmt.Fprintln(w)
	}
	return nil
}
