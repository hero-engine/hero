package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	goalAnswer string
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
	goalCmd.Flags().StringVar(&goalAnswer, "answer", "", "clear the open pause with your decision, so the run resumes")
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
	case goalAnswer != "":
		led, lerr := drive.LoadLedger(heroDir, init.Slug)
		if lerr != nil {
			return lerr
		}
		paused, ok := led.RecordAnswer(goalAnswer)
		if !ok {
			return fmt.Errorf("no open Drive pause for %q to answer", init.Slug)
		}
		if err := led.Save(); err != nil {
			return err
		}
		if err := clearDriveQuestion(heroDir, cfg); err != nil {
			return err
		}
		fmt.Fprintf(w, "Recorded your answer for %s — re-run `/drive %s` (or `hero goal %s --check`) to resume.\n", paused, init.Slug, init.Slug)
		return nil
	case goalCheck:
		res := drive.Check(init, all)
		if err := reconcilePause(heroDir, cfg, init.Slug, all, &res); err != nil {
			return err
		}
		return goalEmitJSON(w, res)
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

// reconcilePause persists a Drive run's pause/resume state against the
// verdict. On a pause it either resumes (if the human already answered this
// transition) or writes the question to the handoff file; on continue/done it
// clears any stale pause + question. Mutates res when resuming.
func reconcilePause(heroDir string, cfg config.Config, initSlug string, all []*spec.Spec, res *drive.CheckResult) error {
	led, err := drive.LoadLedger(heroDir, initSlug)
	if err != nil {
		return err
	}

	if res.Verdict == "pause" && res.Pause != nil {
		// Resume: the human already cleared this exact transition.
		if led.IsAnswered(res.NextSpec) {
			res.Verdict = "continue"
			res.Pause = nil
			if next := specBySlug(all, res.NextSpec); next != nil {
				res.Kickoff = next.Kickoff()
			}
			led.ClearPause()
			if err := led.Save(); err != nil {
				return err
			}
			return clearDriveQuestion(heroDir, cfg)
		}
		// Otherwise record the open question and surface it.
		led.SetPause(&drive.PendingPause{Spec: res.NextSpec, Category: res.Pause.Category, Reason: res.Pause.Reason})
		if err := led.Save(); err != nil {
			return err
		}
		return writeDriveQuestion(heroDir, cfg, initSlug, *res)
	}

	// continue / done — clear any stale pause + lingering question.
	if led.Pause != nil {
		led.ClearPause()
		if err := led.Save(); err != nil {
			return err
		}
	}
	return clearDriveQuestion(heroDir, cfg)
}

func specBySlug(all []*spec.Spec, slug string) *spec.Spec {
	for _, s := range all {
		if s.Slug == slug {
			return s
		}
	}
	return nil
}

// writeDriveQuestion merges the pause question into the team-aware handoff
// file (NEXT.md solo, .hero/next/<user>.md in team mode).
func writeDriveQuestion(heroDir string, cfg config.Config, initSlug string, res drive.CheckResult) error {
	path := resolveNextPath(heroDir, cfg)
	prior, _ := os.ReadFile(path)
	merged := drive.MergeQuestion(string(prior), drive.ComposeQuestion(initSlug, res))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(merged), 0o644)
}

// clearDriveQuestion strips any drive-pause block from the handoff file.
func clearDriveQuestion(heroDir string, cfg config.Config) error {
	path := resolveNextPath(heroDir, cfg)
	prior, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	stripped := drive.StripQuestion(string(prior))
	if stripped == string(prior) {
		return nil
	}
	return os.WriteFile(path, []byte(stripped), 0o644)
}
