package skills

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Runner executes skill steps.
type Runner struct {
	Params   map[string]string
	HeroPath string // path to hero binary (defaults to os.Args[0])
}

// Run executes all steps of a skill in order.
// shell steps: run via $SHELL -c <cmd>
// hero steps: run via exec(HeroPath, args...)
// prompt steps: print to stdout (for agent to pick up)
func (r *Runner) Run(skill *Skill) error {
	heroPath := r.HeroPath
	if heroPath == "" {
		heroPath = os.Args[0]
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	for _, step := range skill.Steps {
		switch step.Kind {
		case "shell":
			cmd, err := InterpolateParams(step.Cmd, r.Params)
			if err != nil {
				fmt.Fprintf(os.Stderr, "skill: step %d: %v\n", step.Index, err)
				continue
			}
			if err := r.runShell(shell, cmd); err != nil {
				fmt.Fprintf(os.Stderr, "skill: step %d failed: %v\n", step.Index, err)
				// continue — don't abort the skill
			}

		case "hero":
			cmdStr, err := InterpolateParams(step.Cmd, r.Params)
			if err != nil {
				fmt.Fprintf(os.Stderr, "skill: step %d: %v\n", step.Index, err)
				continue
			}
			// Strip leading "hero " prefix if present
			cmdStr = strings.TrimPrefix(cmdStr, "hero ")
			args := strings.Fields(cmdStr)
			if err := r.runHero(heroPath, args); err != nil {
				fmt.Fprintf(os.Stderr, "skill: step %d failed: %v\n", step.Index, err)
				// continue — don't abort
			}

		case "prompt":
			text, err := InterpolateParams(step.Text, r.Params)
			if err != nil {
				fmt.Fprintf(os.Stderr, "skill: step %d: %v\n", step.Index, err)
				continue
			}
			fmt.Printf("[Step %d] %s\n", step.Index, text)

		default:
			fmt.Printf("[Step %d] %s\n", step.Index, step.Raw)
		}
	}

	return nil
}

// runShell executes a command via the user's shell.
func (r *Runner) runShell(shell, cmd string) error {
	c := exec.Command(shell, "-c", cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

// runHero executes a hero sub-command via the hero binary.
func (r *Runner) runHero(heroPath string, args []string) error {
	c := exec.Command(heroPath, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}
