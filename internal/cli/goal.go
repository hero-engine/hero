package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/drive"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var (
	goalCheck  bool
	goalEmit   bool
	goalDryRun int
)

var goalCmd = &cobra.Command{
	Use:   "goal <initiative>",
	Short: "Emit an initiative's run condition, or judge a /drive run turn (--check)",
	Long: `hero goal bridges an initiative to the harness /goal loop for /drive.

  hero goal <init>            emit the paste-ready run condition
  hero goal <init> --check    one-turn verdict as JSON (continue|pause|done)
  hero goal <init> --dry-run N  preview the next N transitions --check would take

It does NOT drive the loop or evaluate completion from a transcript — the
harness /goal owns those. hero goal is the authoritative judge the loop
consults via a Stop hook.`,
	Args: cobra.ExactArgs(1),
	RunE: runGoal,
}

func init() {
	goalCmd.Flags().BoolVar(&goalCheck, "check", false, "emit one-turn verdict as JSON")
	goalCmd.Flags().BoolVar(&goalEmit, "emit", false, "print the run condition (default action)")
	goalCmd.Flags().IntVar(&goalDryRun, "dry-run", 0, "preview the next N transitions (e.g. --dry-run 3)")
}

func runGoal(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, statErr := os.Stat(heroDir); os.IsNotExist(statErr) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}
	all, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discover specs: %w", err)
	}
	init, err := findSpecBySlugOrPath(heroDir, args[0])
	if err != nil {
		return err
	}
	if init.Type != spec.TypeInitiative {
		return fmt.Errorf("%q is a %s, not an initiative — `hero goal` runs initiatives (use `/deliver` for a single spec)", init.Slug, init.Type)
	}

	w := cmd.OutOrStdout()
	switch {
	case goalCheck:
		return goalEmitJSON(w, drive.Check(init, all))
	case goalDryRun > 0:
		return goalEmitJSON(w, drive.DryRun(init, all, goalDryRun))
	default: // emit
		bySlug := make(map[string]*spec.Spec, len(all))
		for _, s := range all {
			bySlug[s.Slug] = s
		}
		if obj := strings.TrimSpace(init.GoalSection()); obj != "" {
			fmt.Fprintln(w, obj)
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, init.RunCondition(bySlug))
		return nil
	}
}

func goalEmitJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
