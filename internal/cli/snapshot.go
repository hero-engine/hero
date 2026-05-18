package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/snapshot"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var (
	snapshotJSON     bool
	snapshotSection  string
	snapshotProjectFlag  bool
	snapshotLabel    string
	snapshotExplain  bool
	snapshotJSONList bool
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Print the project-shape rollup (surfaces, stages, initiatives, blockers).",
	Long: `Renders .hero/SNAPSHOT.md from the current graph. Default output is
markdown to stdout; --json emits the same data structured. --project
total-rewrites .hero/SNAPSHOT.md from the live graph.

Surfaces are inferred from repo shape; .hero/surfaces.yaml is an
optional override layer. Snapshot is discoverable via a one-line
pointer in NEXT.md and AGENTS.md — never auto-injected into a
session.

Archive subcommands manage the .hero/snapshots/ trajectory trail.
Archives are excluded from default search and cold-start bundles;
they're reachable only through 'history', 'show', and 'diff'.`,
	RunE: runSnapshot,
}

var snapshotDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Print the inferred surface list with rationale.",
	Long: `Shows which surfaces Hero detected in this repo and (with
--explain) why each was detected. Useful when an inferred surface
list looks wrong — the rationale tells you which override (rename,
ignore, addition) to add to .hero/surfaces.yaml.`,
	RunE: runSnapshotDetect,
}

var snapshotAssignCmd = &cobra.Command{
	Use:   "assign",
	Short: "Walk unassigned specs and prompt for a surface for each.",
	RunE:  runSnapshotAssign,
}

var snapshotArchiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Write an immediate archive of the current snapshot.",
	Long: `Forces a manual archive at .hero/snapshots/<today>[--<slug>].md
regardless of staleness-cutoff or last-archive timestamp. Use
--label to add a slug to the filename and frontmatter.`,
	RunE: runSnapshotArchive,
}

var snapshotHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "List archived snapshots newest-first.",
	RunE:  runSnapshotHistory,
}

var snapshotShowCmd = &cobra.Command{
	Use:   "show <date>",
	Short: "Render one archived snapshot to stdout.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSnapshotShow,
}

var snapshotDiffCmd = &cobra.Command{
	Use:   "diff <date-a> <date-b>",
	Short: "Print a text diff between two archives (or vs. 'live').",
	Args:  cobra.ExactArgs(2),
	RunE:  runSnapshotDiff,
}

func init() {
	snapshotCmd.Flags().BoolVar(&snapshotJSON, "json", false, "emit structured JSON instead of markdown")
	snapshotCmd.Flags().StringVar(&snapshotSection, "section", "", "print one section only (surfaces, initiatives, recent, next, risks, all)")
	snapshotCmd.Flags().BoolVar(&snapshotProjectFlag, "project", false, "total-rewrite .hero/SNAPSHOT.md and the NEXT/AGENTS pointer")

	snapshotDetectCmd.Flags().BoolVar(&snapshotExplain, "explain", false, "show which signals fired for each detected surface")

	snapshotArchiveCmd.Flags().StringVar(&snapshotLabel, "label", "", "label slug appended to the archive filename and frontmatter")

	snapshotHistoryCmd.Flags().BoolVar(&snapshotJSONList, "json", false, "emit structured JSON instead of human text")

	snapshotCmd.AddCommand(snapshotDetectCmd)
	snapshotCmd.AddCommand(snapshotAssignCmd)
	snapshotCmd.AddCommand(snapshotArchiveCmd)
	snapshotCmd.AddCommand(snapshotHistoryCmd)
	snapshotCmd.AddCommand(snapshotShowCmd)
	snapshotCmd.AddCommand(snapshotDiffCmd)
}

func runSnapshot(cmd *cobra.Command, args []string) error {
	root := findProjectRoot()
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	heroDir := cfg.HeroDir(root)

	if snapshotProjectFlag {
		// Project to file, mirrors the integration in checkpoint.
		nextPath := resolveNextPath(heroDir, cfg)
		agentsMD := filepath.Join(root, "AGENTS.md")
		if _, err := os.Stat(agentsMD); err != nil {
			agentsMD = ""
		}
		archive := cfg.SnapshotArchive()
		_, err := snapshot.Project(snapshot.ProjectOptions{
			ProjectRoot:  root,
			HeroDir:      heroDir,
			ProjectName:  filepath.Base(root),
			Mission:      readMissionOneLiner(filepath.Join(heroDir, "mission.md")),
			NextMDPath:   nextPath,
			AgentsMDPath: agentsMD,
			ArchiveConfig: snapshot.ArchiveConfig{
				StalenessCutoff:   archive.StalenessCutoff,
				MilestonesEnabled: cfg.SnapshotMilestonesEnabled(),
				ReleaseTagPattern: archive.ReleaseTagPattern,
				Retention:         archive.Retention,
				RetentionCount:    archive.RetentionCount,
			},
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "snapshot written → %s\n", filepath.Join(heroDir, snapshot.SnapshotFileName))
		return nil
	}

	snap, err := buildSnapshotForCLI(root, heroDir)
	if err != nil {
		return err
	}

	if snapshotJSON {
		out, err := snapshot.Render(snap, snapshot.FormatJSON)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return nil
	}

	if snapshotSection != "" && snapshotSection != "all" {
		out, err := renderSnapshotSection(snap, snapshotSection)
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), out)
		return nil
	}

	out, err := snapshot.Render(snap, snapshot.FormatMarkdown)
	if err != nil {
		return err
	}
	fmt.Fprint(cmd.OutOrStdout(), string(out))
	return nil
}

func buildSnapshotForCLI(root, heroDir string) (*snapshot.Snapshot, error) {
	allSpecs, _ := spec.Discover(heroDir)
	override, err := snapshot.LoadOverride(heroDir)
	if err != nil {
		return nil, fmt.Errorf("load surfaces.yaml: %w", err)
	}
	return snapshot.Build(snapshot.BuildOptions{
		ProjectRoot: root,
		HeroDir:     heroDir,
		ProjectName: filepath.Base(root),
		Mission:     readMissionOneLiner(filepath.Join(heroDir, "mission.md")),
	}, allSpecs, override, nil)
}

func renderSnapshotSection(s *snapshot.Snapshot, section string) (string, error) {
	full, err := snapshot.Render(s, snapshot.FormatMarkdown)
	if err != nil {
		return "", err
	}
	body := string(full)
	headers := map[string]string{
		"surfaces":    "## Surfaces\n",
		"initiatives": "## Active initiatives\n",
		"recent":      "## Recently completed",
		"next":        "## Next up across surfaces\n",
		"risks":       "## Open risks & blockers\n",
		"health":      "## Snapshot health\n",
	}
	start, ok := headers[strings.ToLower(section)]
	if !ok {
		return "", fmt.Errorf("unknown section %q (try: surfaces, initiatives, recent, next, risks, health)", section)
	}
	idx := strings.Index(body, start)
	if idx < 0 {
		return "", fmt.Errorf("section %q not present in current render", section)
	}
	rest := body[idx:]
	// Stop at the next ## header.
	endIdx := strings.Index(rest[len(start):], "\n## ")
	if endIdx < 0 {
		return rest, nil
	}
	return rest[:len(start)+endIdx+1], nil
}

func runSnapshotDetect(cmd *cobra.Command, args []string) error {
	root := findProjectRoot()
	cfg, _ := config.Load(root)
	heroDir := cfg.HeroDir(root)

	rs, err := snapshot.ScanRepo(root)
	if err != nil {
		return fmt.Errorf("scan repo: %w", err)
	}
	detected := snapshot.Detect(rs)
	override, _ := snapshot.LoadOverride(heroDir)
	merged := snapshot.Merge(detected, override)

	w := cmd.OutOrStdout()
	for _, s := range merged {
		fmt.Fprintf(w, "%s\n", s.ID)
		if s.Name != "" && s.Name != s.ID {
			fmt.Fprintf(w, "  name: %s\n", s.Name)
		}
		if len(s.Paths) > 0 {
			fmt.Fprintf(w, "  paths: %s\n", strings.Join(s.Paths, ", "))
		}
		fmt.Fprintf(w, "  source: %s (confidence %.2f)\n", s.Source, s.Confidence)
		if snapshotExplain && len(s.Signals) > 0 {
			fmt.Fprintf(w, "  signals:\n")
			for _, sig := range s.Signals {
				fmt.Fprintf(w, "    - %s\n", sig)
			}
		}
	}
	return nil
}

func runSnapshotAssign(cmd *cobra.Command, args []string) error {
	root := findProjectRoot()
	cfg, _ := config.Load(root)
	heroDir := cfg.HeroDir(root)

	snap, err := buildSnapshotForCLI(root, heroDir)
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	count := 0
	for _, a := range snap.Assignments {
		if a.SurfaceID == snapshot.UnassignedSurfaceID {
			count++
		}
	}
	if count == 0 {
		fmt.Fprintln(w, "No unassigned specs.")
		return nil
	}
	fmt.Fprintf(w, "%d unassigned specs:\n", count)
	for _, a := range snap.Assignments {
		if a.SurfaceID != snapshot.UnassignedSurfaceID {
			continue
		}
		fmt.Fprintf(w, "  %s — %s (%s)\n", a.Spec.Slug, a.Spec.Title, a.Spec.Type)
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Interactive prompting is not yet wired in this build — edit the")
	fmt.Fprintln(w, "spec frontmatter manually (add `surface: <id>`) or update")
	fmt.Fprintln(w, ".hero/surfaces.yaml additions/overrides to expand coverage.")
	return nil
}

func runSnapshotArchive(cmd *cobra.Command, args []string) error {
	root := findProjectRoot()
	cfg, _ := config.Load(root)
	heroDir := cfg.HeroDir(root)

	archive := cfg.SnapshotArchive()
	result, err := snapshot.Project(snapshot.ProjectOptions{
		ProjectRoot:        root,
		HeroDir:            heroDir,
		ProjectName:        filepath.Base(root),
		Mission:            readMissionOneLiner(filepath.Join(heroDir, "mission.md")),
		NextMDPath:         resolveNextPath(heroDir, cfg),
		AgentsMDPath:       agentsMDPathOrEmpty(root),
		ManualArchive:      true,
		ManualArchiveLabel: snapshotLabel,
		ArchiveConfig: snapshot.ArchiveConfig{
			StalenessCutoff:   archive.StalenessCutoff,
			MilestonesEnabled: cfg.SnapshotMilestonesEnabled(),
			ReleaseTagPattern: archive.ReleaseTagPattern,
			Retention:         archive.Retention,
			RetentionCount:    archive.RetentionCount,
		},
	})
	if err != nil {
		return err
	}
	for _, a := range result.Archives {
		fmt.Fprintf(cmd.OutOrStdout(), "archived → %s\n", a.Path)
	}
	if len(result.Archives) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no archive written (same-day idempotent — use --label to override)")
	}
	return nil
}

func agentsMDPathOrEmpty(root string) string {
	p := filepath.Join(root, "AGENTS.md")
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

func runSnapshotHistory(cmd *cobra.Command, args []string) error {
	root := findProjectRoot()
	cfg, _ := config.Load(root)
	heroDir := cfg.HeroDir(root)

	archives, err := snapshot.List(heroDir)
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	if snapshotJSONList {
		type row struct {
			Date      string `json:"date"`
			Trigger   string `json:"trigger"`
			Label     string `json:"label,omitempty"`
			GitCommit string `json:"git_commit,omitempty"`
			Path      string `json:"path"`
		}
		rows := make([]row, 0, len(archives))
		for _, a := range archives {
			rows = append(rows, row{
				Date:      a.Date,
				Trigger:   string(a.Trigger),
				Label:     a.Label,
				GitCommit: a.GitCommit,
				Path:      a.Path,
			})
		}
		data, _ := json.MarshalIndent(rows, "", "  ")
		fmt.Fprintln(w, string(data))
		return nil
	}
	if len(archives) == 0 {
		fmt.Fprintln(w, "No archives yet.")
		return nil
	}
	for _, a := range archives {
		label := a.Label
		if label == "" {
			label = "(no label)"
		}
		fmt.Fprintf(w, "%s  %-10s  %-30s  %s\n", a.Date, a.Trigger, label, filepath.Base(a.Path))
	}
	return nil
}

func runSnapshotShow(cmd *cobra.Command, args []string) error {
	root := findProjectRoot()
	cfg, _ := config.Load(root)
	heroDir := cfg.HeroDir(root)

	rec, err := snapshot.FindArchive(heroDir, args[0])
	if err != nil {
		return err
	}
	if rec == nil {
		return fmt.Errorf("no archive found for %q", args[0])
	}
	data, err := os.ReadFile(rec.Path)
	if err != nil {
		return err
	}
	fmt.Fprint(cmd.OutOrStdout(), string(data))
	return nil
}

func runSnapshotDiff(cmd *cobra.Command, args []string) error {
	root := findProjectRoot()
	cfg, _ := config.Load(root)
	heroDir := cfg.HeroDir(root)

	left, err := resolveSnapshotForDiff(args[0], heroDir)
	if err != nil {
		return err
	}
	right, err := resolveSnapshotForDiff(args[1], heroDir)
	if err != nil {
		return err
	}
	fmt.Fprint(cmd.OutOrStdout(), snapshot.Diff(left, right))
	return nil
}

func resolveSnapshotForDiff(token, heroDir string) (string, error) {
	switch strings.ToLower(token) {
	case "live", "current":
		data, err := os.ReadFile(filepath.Join(heroDir, snapshot.SnapshotFileName))
		if err != nil {
			return "", fmt.Errorf("live snapshot: %w", err)
		}
		return string(data), nil
	case "latest":
		archives, err := snapshot.List(heroDir)
		if err != nil || len(archives) == 0 {
			return "", fmt.Errorf("no archives to use as 'latest'")
		}
		return archives[0].Body, nil
	}
	rec, err := snapshot.FindArchive(heroDir, token)
	if err != nil {
		return "", err
	}
	if rec == nil {
		return "", fmt.Errorf("no archive found for %q", token)
	}
	return rec.Body, nil
}
