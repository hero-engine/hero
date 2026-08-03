package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/cli/prompt"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/skills"
	"github.com/spf13/cobra"
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage and run reusable skill workflows",
	Long:  `Skills are reusable step-by-step workflows stored as .hero/skills/<name>.md files.`,
}

var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all skills",
	RunE:  runSkillList,
}

var skillShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Print the skill markdown file",
	Args:  selectorOneArg.rule(),
	RunE:  runSkillShow,
}

var skillRunCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Run a skill",
	Args:  selectorOneArg.rule(),
	RunE:  runSkillRun,
}

var skillSaveCmd = &cobra.Command{
	Use:   "save",
	Short: "Create a new skill template and open it in $EDITOR",
	RunE:  runSkillSave,
}

var skillEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Open a skill in $EDITOR",
	Args:  selectorOneArg.rule(),
	RunE:  runSkillEdit,
}

var skillRmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a skill file",
	Args:  selectorOneArg.rule(),
	RunE:  runSkillRm,
}

var skillLogCmd = &cobra.Command{
	Use:   "log <name>",
	Short: "Show git log for a skill file",
	Args:  selectorOneArg.rule(),
	RunE:  runSkillLog,
}

var skillRunParams []string

func init() {
	skillRunCmd.Flags().StringSliceVar(&skillRunParams, "param", nil, "parameter values as key=val (repeatable)")

	skillCmd.AddCommand(skillListCmd)
	skillCmd.AddCommand(skillShowCmd)
	skillCmd.AddCommand(skillRunCmd)
	skillCmd.AddCommand(skillSaveCmd)
	skillCmd.AddCommand(skillEditCmd)
	skillCmd.AddCommand(skillRmCmd)
	skillCmd.AddCommand(skillLogCmd)
}

// skillsDir returns the path to the .hero/skills directory.
func skillsDir(cfg config.Config, projectRoot string) string {
	return filepath.Join(cfg.HeroDir(projectRoot), "skills")
}

// skillFilePath returns the path to a specific skill file.
func skillFilePath(cfg config.Config, projectRoot, name string) string {
	return filepath.Join(skillsDir(cfg, projectRoot), name+".md")
}

func runSkillList(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	dir := skillsDir(cfg, projectRoot)
	all, err := skills.Discover(dir)
	if err != nil {
		return fmt.Errorf("discovering skills: %w", err)
	}

	if len(all) == 0 {
		fmt.Println("No skills found. Create one with: hero skill save")
		return nil
	}

	fmt.Printf("Skills (%d):\n", len(all))
	for _, s := range all {
		title := s.Title
		if title == "" {
			title = s.Slug
		}
		stepCount := len(s.Steps)
		fmt.Printf("  %-30s  %s  (%d steps)\n", s.Slug, title, stepCount)
	}
	return nil
}

func runSkillShow(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	name, err := skillTarget(cmd, args, cfg, projectRoot)
	if err != nil {
		return err
	}
	path := skillFilePath(cfg, projectRoot, name)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("skill %q not found (looked in %s)", name, path)
		}
		return err
	}

	fmt.Print(string(data))
	return nil
}

func runSkillRun(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	name, err := skillTarget(cmd, args, cfg, projectRoot)
	if err != nil {
		return err
	}
	path := skillFilePath(cfg, projectRoot, name)

	skill, err := skills.ParseSkillFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("skill %q not found (looked in %s)", name, path)
		}
		return fmt.Errorf("parsing skill %q: %w", name, err)
	}

	// Build params map from --param flags
	params := make(map[string]string)
	for _, kv := range skillRunParams {
		eqIdx := strings.Index(kv, "=")
		if eqIdx < 0 {
			return fmt.Errorf("invalid --param %q: expected key=val", kv)
		}
		params[kv[:eqIdx]] = kv[eqIdx+1:]
	}

	// Find missing params and prompt interactively if TTY available
	if len(skill.Params) > 0 {
		in := cmd.InOrStdin()
		out := cmd.OutOrStdout()
		isTTY := prompt.IsInputTTY(in)
		for _, p := range skill.Params {
			if _, ok := params[p.Name]; ok {
				continue
			}
			if isTTY {
				val, err := promptParam(in, out, p.Name, p.Description)
				if err != nil {
					return fmt.Errorf("reading param %q: %w", p.Name, err)
				}
				params[p.Name] = val
			} else {
				return fmt.Errorf("missing required parameter %q (use --param %s=<value>)", p.Name, p.Name)
			}
		}
	}

	runner := &skills.Runner{Params: params}
	return runner.Run(skill)
}

func runSkillSave(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	dir := skillsDir(cfg, projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating skills directory: %w", err)
	}

	// Two fields read in sequence, not a form engine. A descriptor would need
	// a third consumer to earn itself, and there isn't one — see the hard stop
	// in the cli-prompt-package-adoption spec.
	//
	in := cmd.InOrStdin()
	out := cmd.OutOrStdout()
	if !prompt.IsInputTTY(in) {
		return fmt.Errorf("skill save requires an attached terminal")
	}

	// promptLineStrict, not prompt.Prompt: these two reads act on the error, so
	// they need bufio's end-of-stream contract. See prompt_line.go.
	name, err := promptLineStrict(in, out, "Skill name (slug, e.g. my-workflow): ")
	if err != nil {
		return fmt.Errorf("reading name: %w", err)
	}
	if name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	title, err := promptLineStrict(in, out, "Skill title: ")
	if err != nil {
		return fmt.Errorf("reading title: %w", err)
	}

	path := filepath.Join(dir, name+".md")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("skill %q already exists at %s", name, path)
	}

	template := skills.CaptureTemplate(name, title, nil)
	if err := os.WriteFile(path, []byte(template), 0o644); err != nil {
		return fmt.Errorf("writing skill file: %w", err)
	}

	fmt.Printf("Created %s\n", path)
	return openEditor(path)
}

func runSkillEdit(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	name, err := skillTarget(cmd, args, cfg, projectRoot)
	if err != nil {
		return err
	}
	path := skillFilePath(cfg, projectRoot, name)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("skill %q not found (looked in %s)", name, path)
	}

	return openEditor(path)
}

func runSkillRm(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	name, err := skillTarget(cmd, args, cfg, projectRoot)
	if err != nil {
		return err
	}
	path := skillFilePath(cfg, projectRoot, name)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("skill %q not found (looked in %s)", name, path)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing skill: %w", err)
	}

	fmt.Printf("Removed %s\n", path)
	return nil
}

func runSkillLog(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	name, err := skillTarget(cmd, args, cfg, projectRoot)
	if err != nil {
		return err
	}
	path := skillFilePath(cfg, projectRoot, name)

	// Use a path relative to project root for git log
	relPath, err := filepath.Rel(projectRoot, path)
	if err != nil {
		relPath = path
	}

	c := exec.Command("git", "-C", projectRoot, "log", "--oneline", "--", relPath)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("git log: %w", err)
	}
	return nil
}

// openEditor opens a file in $EDITOR (or vi as fallback).
func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// promptParam interactively prompts the user for a parameter value.
//
// The caller has already established that in is a terminal; this only reads.
func promptParam(in io.Reader, out io.Writer, name, description string) (string, error) {
	label := fmt.Sprintf("  %s: ", name)
	if description != "" {
		label = fmt.Sprintf("  %s (%s): ", name, description)
	}
	// promptLineStrict, not prompt.Prompt: the caller wraps this error into
	// `reading param %q`, so a stream that ends mid-answer must stay an error.
	val, err := promptLineStrict(in, out, label)
	if err != nil {
		return "", err
	}
	return val, nil
}
