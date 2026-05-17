package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tasks"
	"github.com/spf13/cobra"
)

// `hero task` umbrella for work-tracking task operations. Additive
// peer of `hero ac`; mirrors its shape and command surface. AC stays
// untouched.

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Work-tracking task operations",
	Long: `Work-tracking task operations (additive peer of acceptance criteria).

  hero task add <spec-slug> "<text>"   add a new task to a spec's ## Tasks
  hero task list <spec-slug>           list tasks for a spec (markdown or --json)
  hero task start <spec-slug> <T-N>    flip a task to doing (stamps started)
  hero task done <spec-slug> <T-N>     flip a task to done (stamps done)
  hero task history <spec-slug>:<T-N>  timeline of every status flip
  hero task status [--feature X]       completion-rate per spec across the corpus

Tasks are the "next thing to do" sub-element. They flip on human
action; acceptance criteria flip on test evidence. The two
infrastructures live side-by-side.`,
}

var (
	taskListJSON      bool
	taskStatusFeature string
	taskStatusJSON    bool
	taskHistoryJSON   bool

	taskAddKind              string
	taskAddAssignee          string
	taskAddDiscoveredAgainst string
)

var taskAddCmd = &cobra.Command{
	Use:   "add <spec-slug> <text>",
	Short: "Add a new task to a spec's ## Tasks section",
	Long: `Adds a new entry to the spec's ## Tasks section. ID is auto-assigned
as the next T-N; status defaults to "todo".

If the spec has no ## Tasks section, one is created at the end of the
file. Optional metadata flags populate the inline {...} shorthand:
  --kind               task category (e.g. qa-blocker, chore, feature)
  --assignee           handle or display name
  --discovered-against parent spec slug where this issue was found`,
	Args: cobra.MinimumNArgs(2),
	RunE: runTaskAdd,
}

var taskListCmd = &cobra.Command{
	Use:   "list <spec-slug>",
	Short: "List tasks for a spec from the graph",
	Long: `Reads Task nodes for the given spec slug from the graph and prints
them — markdown by default, JSON with --json.

Output (--json):
  [{"key":"<slug>:T-N","task_id":"T-N","text":"...","status":"...","parent":"..."}, ...]`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskList,
}

var taskStartCmd = &cobra.Command{
	Use:   "start <spec-slug> <T-N>",
	Short: "Flip a task to doing (stamps started timestamp)",
	Args:  cobra.ExactArgs(2),
	RunE:  runTaskTransition(tasks.StatusDoing),
}

var taskDoneCmd = &cobra.Command{
	Use:   "done <spec-slug> <T-N>",
	Short: "Flip a task to done (stamps done timestamp)",
	Args:  cobra.ExactArgs(2),
	RunE:  runTaskTransition(tasks.StatusDone),
}

var taskHistoryCmd = &cobra.Command{
	Use:   "history <spec-slug>:<T-N>",
	Short: "Show every recorded status flip for one task",
	Long: `Walks the bitemporal Task rows for the given task key (oldest
first) and prints each [valid_from, valid_to) interval and its
status. The current row has an open-ended interval.

Example:
  hero task history checkout-flow:T-3
  todo    2026-05-15T09:00:00Z → 2026-05-16T11:22:00Z
  doing   2026-05-16T11:22:00Z → 2026-05-17T14:08:00Z
  done    2026-05-17T14:08:00Z → (current)`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskHistory,
}

var taskStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Task completion-rate per spec across the corpus",
	Long: `Reads every Task node and rolls up status counts per parent spec.
Default output is markdown with one row per spec; --json emits the
raw rollup. --feature <slug> narrows to one spec.`,
	RunE: runTaskStatus,
}

func init() {
	taskListCmd.Flags().BoolVar(&taskListJSON, "json", false, "emit JSON array suitable for piping into scripts")
	taskStatusCmd.Flags().StringVar(&taskStatusFeature, "feature", "", "narrow to one spec slug")
	taskStatusCmd.Flags().BoolVar(&taskStatusJSON, "json", false, "emit JSON rollup")
	taskHistoryCmd.Flags().BoolVar(&taskHistoryJSON, "json", false, "emit JSON timeline")
	taskAddCmd.Flags().StringVar(&taskAddKind, "kind", "", "task kind (qa-blocker, chore, feature, ...)")
	taskAddCmd.Flags().StringVar(&taskAddAssignee, "assignee", "", "assignee handle / display name")
	taskAddCmd.Flags().StringVar(&taskAddDiscoveredAgainst, "discovered-against", "", "parent spec slug where this was found")

	taskCmd.AddCommand(taskAddCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskStartCmd)
	taskCmd.AddCommand(taskDoneCmd)
	taskCmd.AddCommand(taskHistoryCmd)
	taskCmd.AddCommand(taskStatusCmd)
	rootCmd.AddCommand(taskCmd)
}

// loadStoreForTask opens the graph store the same way `hero ac`
// commands do. Centralized here to keep each command thin.
func loadStoreForTask() (*graph.Store, string, error) {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, "", fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return nil, "", fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}
	store, err := graph.Open(heroDir)
	if err != nil {
		return nil, "", fmt.Errorf("opening graph: %w", err)
	}
	return store, heroDir, nil
}

// loadSpecForTask resolves slug to its parsed Spec via Discover.
// Returns nil and an error if none exists. Named distinctly from the
// existing string-returning findSpecBySlug (claim.go) to avoid
// collision and clarify intent.
func loadSpecForTask(heroDir, slug string) (*spec.Spec, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}
	for _, s := range specs {
		if s.Slug == slug {
			return s, nil
		}
	}
	return nil, fmt.Errorf("no spec found with slug %q (use `hero list` to see slugs)", slug)
}

func runTaskAdd(cmd *cobra.Command, args []string) error {
	slug := args[0]
	text := strings.TrimSpace(strings.Join(args[1:], " "))
	if text == "" {
		return fmt.Errorf("task text is required")
	}

	store, heroDir, err := loadStoreForTask()
	if err != nil {
		return err
	}
	defer store.Close()

	sp, err := loadSpecForTask(heroDir, slug)
	if err != nil {
		return err
	}

	existing := tasks.ParseTasks(tasks.FindSection(sp.Sections))
	newTask, updated := tasks.AddTask(existing, text, tasks.AddOptions{
		Kind:              taskAddKind,
		Assignee:          taskAddAssignee,
		DiscoveredAgainst: taskAddDiscoveredAgainst,
	})

	body := tasks.Render(updated, tasks.EditOptions{})
	if err := tasks.ApplyToFile(sp.Path, body); err != nil {
		return err
	}

	// Re-scan into the graph so the new task lands as a Task node
	// without waiting for the next full `hero scan`. Best-effort: a
	// write failure on the graph side doesn't undo the file edit.
	if parentID, ok := resolveSpecGraphID(store, sp); ok {
		projectRoot := findProjectRoot()
		repoKey := gitutil.RepoKey(projectRoot)
		_, _ = tasks.Write(parentNodeType(sp.Type), sp.Slug, parentID, updated, repoKey, store)
	}

	fmt.Printf("Added %s — %s\n", newTask.ID, newTask.Text)
	if newTask.Kind != "" || newTask.Assignee != "" || newTask.DiscoveredAgainst != "" {
		var bits []string
		if newTask.Kind != "" {
			bits = append(bits, "kind="+newTask.Kind)
		}
		if newTask.Assignee != "" {
			bits = append(bits, "assignee="+newTask.Assignee)
		}
		if newTask.DiscoveredAgainst != "" {
			bits = append(bits, "discovered_against="+newTask.DiscoveredAgainst)
		}
		fmt.Printf("  {%s}\n", strings.Join(bits, ", "))
	}
	return nil
}

func runTaskList(cmd *cobra.Command, args []string) error {
	store, _, err := loadStoreForTask()
	if err != nil {
		return err
	}
	defer store.Close()

	records, err := tasks.ListBySpec(store, args[0])
	if err != nil {
		return fmt.Errorf("listing tasks: %w", err)
	}
	if len(records) == 0 {
		if taskListJSON {
			fmt.Println("[]")
			return nil
		}
		fmt.Printf("No tasks found for %q.\n", args[0])
		fmt.Println("(`hero task add <slug> \"<text>\"` adds one; `hero scan` ingests Task nodes from `## Tasks` blocks.)")
		return nil
	}

	if taskListJSON {
		fmt.Println("[")
		for i, r := range records {
			sep := ","
			if i == len(records)-1 {
				sep = ""
			}
			data, _ := json.Marshal(struct {
				Key               string `json:"key"`
				TaskID            string `json:"task_id"`
				Text              string `json:"text"`
				Status            string `json:"status"`
				Parent            string `json:"parent"`
				Kind              string `json:"kind,omitempty"`
				Assignee          string `json:"assignee,omitempty"`
				DiscoveredAgainst string `json:"discovered_against,omitempty"`
			}{
				Key: r.Key, TaskID: r.TaskID, Text: r.Text,
				Status: r.Status, Parent: r.Parent,
				Kind: r.Kind, Assignee: r.Assignee,
				DiscoveredAgainst: r.DiscoveredAgainst,
			})
			fmt.Printf("  %s%s\n", string(data), sep)
		}
		fmt.Println("]")
		return nil
	}

	fmt.Printf("Tasks for `%s` (%d):\n\n", args[0], len(records))
	for _, r := range records {
		extra := ""
		if r.Assignee != "" {
			extra = fmt.Sprintf("  @%s", r.Assignee)
		}
		fmt.Printf("  %s  %s — %s%s\n", taskGlyph(r.Status), r.TaskID, summarize(r.Text, 90), extra)
	}
	return nil
}

// runTaskTransition returns a RunE that flips a task to target status,
// rewrites the spec file, and resyncs the graph.
func runTaskTransition(target string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		slug, taskID := args[0], args[1]
		store, heroDir, err := loadStoreForTask()
		if err != nil {
			return err
		}
		defer store.Close()

		sp, err := loadSpecForTask(heroDir, slug)
		if err != nil {
			return err
		}

		existing := tasks.ParseTasks(tasks.FindSection(sp.Sections))
		updated, err := tasks.TransitionTo(existing, taskID, target)
		if err != nil {
			return err
		}
		body := tasks.Render(updated, tasks.EditOptions{})
		if err := tasks.ApplyToFile(sp.Path, body); err != nil {
			return err
		}

		if parentID, ok := resolveSpecGraphID(store, sp); ok {
			projectRoot := findProjectRoot()
			repoKey := gitutil.RepoKey(projectRoot)
			_, _ = tasks.Write(parentNodeType(sp.Type), sp.Slug, parentID, updated, repoKey, store)
		}

		fmt.Printf("%s → %s on %s.\n", taskID, target, slug)
		return nil
	}
}

func runTaskHistory(cmd *cobra.Command, args []string) error {
	store, _, err := loadStoreForTask()
	if err != nil {
		return err
	}
	defer store.Close()

	entries, err := tasks.History(store, args[0])
	if err != nil {
		return fmt.Errorf("history query: %w", err)
	}

	if taskHistoryJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(entries)
	}

	if len(entries) == 0 {
		fmt.Printf("No history for %q. Check the task key — format is <spec-slug>:T-N.\n", args[0])
		return nil
	}

	fmt.Printf("History for %s:\n\n", args[0])
	for _, e := range entries {
		to := "(current)"
		if e.ValidTo != "" {
			to = e.ValidTo
		}
		fmt.Printf("  %s  %s → %s\n", taskGlyph(e.Status), e.ValidFrom, to)
		fmt.Printf("                                         %s\n", e.Status)
	}
	return nil
}

func runTaskStatus(cmd *cobra.Command, args []string) error {
	store, _, err := loadStoreForTask()
	if err != nil {
		return err
	}
	defer store.Close()

	rows, err := tasks.StatusByFeature(store, taskStatusFeature)
	if err != nil {
		return fmt.Errorf("status query: %w", err)
	}

	if taskStatusJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(rows)
	}

	if len(rows) == 0 {
		if taskStatusFeature != "" {
			fmt.Printf("No tasks found for %q.\n", taskStatusFeature)
		} else {
			fmt.Println("No tasks in graph. (Run `hero scan` first, or add some with `hero task add`.)")
		}
		return nil
	}

	fmt.Printf("%-40s  %5s  %4s %4s %4s  %s\n",
		"Spec", "Total", "Todo", "Doing", "Done", "Completion")
	fmt.Println(strings.Repeat("-", 80))
	for _, r := range rows {
		rate := ""
		if r.Total > 0 {
			rate = fmt.Sprintf("%3.0f%%", r.CompletionRate()*100)
		}
		fmt.Printf("%-40s  %5d  %4d %4d %4d  %s\n",
			truncate(r.Parent, 40), r.Total, r.Todo, r.Doing, r.Done, rate)
	}
	return nil
}

// resolveSpecGraphID returns the graph node ID for the spec, looked
// up by the canonical (Type, Slug) pair. If the spec hasn't been
// scanned yet, a minimal parent node is upserted so the task write
// can wire its belongs_to edge. A later full `hero scan` pass
// overwrites the stub with the full spec properties — content hash
// drives idempotency.
func resolveSpecGraphID(store *graph.Store, s *spec.Spec) (int64, bool) {
	nodeType := parentNodeType(s.Type)
	if nodeType == "" {
		return 0, false
	}
	if id, err := store.GetNodeID(nodeType, s.Slug); err == nil {
		return id, true
	}
	// Upsert a minimal parent so tasks can attach. Keeping the stub
	// shape tight (just title + status) avoids racing the full scan's
	// content hash — when scan runs next, its richer hash supersedes.
	id, err := store.UpsertNode(&graph.Node{
		Type: nodeType,
		Key:  s.Slug,
		Props: map[string]any{
			"title":  s.Title,
			"status": string(s.Status),
			"stub":   true,
		},
		ContentHash: "task-parent-stub-" + s.Slug,
		Source:      map[string]any{"kind": "task-cli-stub", "path": s.Path},
	})
	if err != nil {
		return 0, false
	}
	return id, true
}

// parentNodeType maps a spec.Type to the corresponding graph node
// type. Duplicated from internal/spec.graphTypeFor on purpose: we
// don't want internal/tasks → internal/spec import (it'd be circular
// once spec eventually wires into tasks), and we don't want
// internal/cli reaching into spec's unexported helpers. Only the
// work-shaped types are listed; others return "" (no graph write).
func parentNodeType(t spec.Type) string {
	switch t {
	case spec.TypeFeature:
		return "Feature"
	case spec.TypeBug:
		return "Bug"
	case spec.TypeInitiative:
		return "Initiative"
	case spec.TypeConvention:
		return "Convention"
	case spec.TypeDecision:
		return "Decision"
	}
	return ""
}

// taskGlyph returns a status indicator for terminal output. Distinct
// from statusGlyph (which is for AC statuses); using different glyphs
// here makes mixed AC + task printouts scannable.
func taskGlyph(s string) string {
	switch s {
	case tasks.StatusDone:
		return "✅"
	case tasks.StatusDoing:
		return "🔄"
	case tasks.StatusTodo:
		return "◯ "
	default:
		return "?"
	}
}
