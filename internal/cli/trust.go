package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/cli/prompt"
	"github.com/hero-engine/hero/internal/install"
	"github.com/spf13/cobra"
)

var trustTargets = []string{"codex", "claude"}

var trustCmd = &cobra.Command{
	Use:   "trust <codex|claude> [project|global]",
	Short: "Apply or show one-time harness permission setup for Hero",
	Args:  promptableArgs(1, cobra.RangeArgs(1, 2)),
	RunE:  runTrust,
}

func promptTrustTarget(in io.Reader, out io.Writer) (string, error) {
	choice, err := prompt.Choice(in, out, "Trust target", trustTargets)
	if err != nil {
		return "", err
	}
	if choice == "" {
		return "", fmt.Errorf("trust target is required; supported targets: %s", strings.Join(trustTargets, ", "))
	}
	return choice, nil
}

func runTrust(cmd *cobra.Command, args []string) error {
	target := ""
	if len(args) > 0 {
		target = args[0]
	} else {
		var err error
		target, err = promptTrustTarget(cmd.InOrStdin(), cmd.OutOrStdout())
		if err != nil {
			return err
		}
	}
	mode := install.ModeProject
	scopeExplicit := false
	if len(args) == 2 {
		scopeExplicit = true
		mode = install.Mode(args[1])
		if mode != install.ModeProject && mode != install.ModeGlobal {
			return fmt.Errorf("scope must be 'project' or 'global', got %q", args[1])
		}
	}

	switch target {
	case "codex":
		printCodexTrustHint()
		if scopeExplicit {
			fmt.Println()
			fmt.Printf("Note: scope `%s` has no effect for Codex; Codex owns its own approval state.\n", mode)
		}
		return nil
	case "claude":
		return applyClaudeTrust(mode)
	default:
		return fmt.Errorf("unsupported trust target %q; supported targets: codex, claude", target)
	}
}

func printCodexTrustHint() {
	fmt.Println()
	fmt.Println("Codex permissions: optional one-time setup")
	fmt.Println("Hero cannot grant Codex permissions itself; Codex owns the approval.")
	fmt.Println("To avoid repeated Hero CLI prompts, ask Codex:")
	fmt.Println()
	fmt.Println("  Please run `hero status` and request persistent approval for the `hero` command prefix.")
	fmt.Println()
	fmt.Println("You can show this again with `hero trust codex`.")
}

func applyClaudeTrust(mode install.Mode) error {
	var projectRoot string
	if mode == install.ModeProject {
		projectRoot = findProjectRoot()
		if projectRoot == "" {
			return fmt.Errorf("could not resolve project root; run from inside a hero workspace")
		}
	}

	added, path, err := install.EnsureClaudeHeroAllowlist(mode, projectRoot)
	if err != nil {
		return fmt.Errorf("applying claude allowlist: %w", err)
	}

	displayPath := prettyHomePath(path)
	label := "Claude Code"
	if mode == install.ModeGlobal {
		label = "Claude Code (global)"
	}

	fmt.Println()
	if added {
		fmt.Printf("%s: added %s to %s\n", label, "Bash(hero:*)", displayPath)
		fmt.Println("Claude Code will stop prompting for `hero` commands after it reloads settings.")
	} else {
		fmt.Printf("%s: Bash(hero:*) already present in %s.\n", label, displayPath)
		fmt.Println("No change needed.")
	}
	return nil
}

// prettyHomePath rewrites an absolute path under the user's home
// directory to a leading `~/` form for nicer terminal display. Returns
// the original path unchanged when it isn't under $HOME or when $HOME
// can't be resolved.
func prettyHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	if abs == home {
		return "~"
	}
	prefix := home + string(filepath.Separator)
	if strings.HasPrefix(abs, prefix) {
		return "~" + string(filepath.Separator) + strings.TrimPrefix(abs, prefix)
	}
	return path
}
