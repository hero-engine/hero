package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var (
	syncPullFields []string
	syncPullJSON   bool
)

// newTrackerForPull builds the tracker adapter for the field-level pull
// path. It is a package var so tests can inject a no-network mock
// without standing up an httptest server. Production wiring is
// tracker.NewWithJiraConfig (the same constructor the rest of the sync
// commands use); see runSyncPullField.
var newTrackerForPull = func(cfg config.Config, projectRoot string) (tracker.Tracker, error) {
	return tracker.NewWithJiraConfig(cfg.Tracker, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
}

// osExitPull is the process-exit hook for the auth-failure path
// (401/403 → exit 2). It is a package var so tests can capture the code
// instead of terminating the test binary. Defaults to os.Exit.
var osExitPull = os.Exit

var pullCmd = &cobra.Command{
	Use:   "pull <spec-path>",
	Short: "Sync tracker status back to spec frontmatter",
	Long: `Fetches the current status of the linked tracker issue and updates
the spec's frontmatter to match.

Requires the spec to have a tracker_id in its frontmatter and a tracker
to be configured in hero.json.

With --field, fetches the CURRENT tracker-side value of a single field
for a tracker-backed spec and emits it as a JSON envelope on stdout. This
mode is READ-ONLY: it never writes the tracker and never writes the local
spec — the caller applies the value locally. One --field per invocation:

  hero sync pull <slug> --field priority --json`,
	Args: cobra.ExactArgs(1),
	RunE: runPull,
}

func init() {
	pullCmd.Flags().StringArrayVar(&syncPullFields, "field", nil, "fetch the current tracker value of this field (read-only; repeatable)")
	pullCmd.Flags().BoolVar(&syncPullJSON, "json", false, "emit a machine-readable JSON envelope on stdout")
}

func runPull(cmd *cobra.Command, args []string) error {
	// Field-level pull (tracker → caller). Read-only: never writes the
	// tracker, never writes the local spec. The Swift PM-sync puller
	// shells `sync pull <slug> --field <name> --json` and applies the
	// returned value itself.
	if len(syncPullFields) > 0 {
		return runSyncPullField(args[0])
	}
	return runStatusPull(args[0])
}

func runStatusPull(specPath string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	if cfg.Tracker == nil || cfg.Tracker.Type == "" || cfg.Tracker.Type == "none" {
		return fmt.Errorf("no tracker configured — set tracker.type in hero.json")
	}

	s, err := spec.ParseFile(specPath)
	if err != nil {
		return fmt.Errorf("parsing spec %s: %w", specPath, err)
	}

	if s.TrackerID == "" {
		return fmt.Errorf("spec %s has no tracker_id — use 'hero link' or 'hero sync' first", s.Slug)
	}

	t, err := tracker.NewWithJiraConfig(cfg.Tracker, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
	if err != nil {
		return fmt.Errorf("initializing tracker: %w", err)
	}

	issue, err := t.GetIssue(s.TrackerID)
	if err != nil {
		return fmt.Errorf("fetching issue %s: %w", s.TrackerID, err)
	}

	fmt.Printf("Tracker issue %s: %s\n", issue.ID, issue.Title)
	fmt.Printf("  Tracker status: %s\n", issue.Status)
	fmt.Printf("  Spec status:    %s\n", s.Status)

	// Size mapping sync (tracker → local). Non-destructive: seeds
	// local `size:` only when local is unset; surfaces conflicts as
	// warnings without auto-resolving. No-op when size_mapping is
	// absent (no tracker mapping → never touched).
	sizePlan := tracker.PlanSizePull(t, issue, s.Size)
	switch sizePlan.Action {
	case tracker.SizeSyncSeedLocal:
		content := readSpecContent(specPath)
		content = spec.SetFrontmatterField(content, "size", sizePlan.WriteValue)
		if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not seed local size: %v\n", err)
		} else {
			fmt.Printf("  size: (unset) → %s  (seeded from tracker value %q)\n", sizePlan.WriteValue, sizePlan.TrackerValue)
		}
	case tracker.SizeSyncConflict:
		fmt.Fprintf(os.Stderr, "  Warning: %s\n", sizePlan.Message)
	}

	// Update tracker-prefixed fields if the spec uses them
	if prefix := detectTrackerPrefix(readSpecContent(specPath)); prefix != "" {
		content := readSpecContent(specPath)
		if issue.Status != "" {
			content = spec.SetFrontmatterField(content, prefix+"_status", issue.Status)
		}
		if issue.Priority != "" {
			content = spec.SetFrontmatterField(content, prefix+"_priority", issue.Priority)
		}
		if issue.Severity != "" {
			content = spec.SetFrontmatterField(content, prefix+"_severity", issue.Severity)
		}
		if issue.Assignee != "" {
			content = spec.SetFrontmatterField(content, prefix+"_assignee", issue.Assignee)
		}
		if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not update tracker fields: %v\n", err)
		}
	}

	// Map tracker status to spec status
	newStatus := mapTrackerStatus(issue.Status, t.Name())
	if newStatus == "" {
		fmt.Printf("  No mapping for tracker status %q — spec unchanged\n", issue.Status)
		return nil
	}

	if spec.Status(newStatus) == s.Status {
		fmt.Println("  Already in sync")
		return nil
	}

	// Update spec frontmatter
	if err := updateFrontmatterStatus(specPath, newStatus); err != nil {
		return fmt.Errorf("updating spec status: %w", err)
	}

	fmt.Printf("  Updated spec status: %s → %s\n", s.Status, newStatus)
	return nil
}

// pullEnvelope is the stable JSON contract consumed by hero-code's
// SyncPuller (hero-pm-sync-pull-fields). `slug`/`field`/`value` are the
// load-bearing keys the Swift SyncPullEnvelope decodes; `value` is
// nullable (a null/absent value means the tracker has no value for the
// field — the Swift side treats that as "skip, no write"). `status` and
// `error` mirror the push envelope for consistency but are not decoded
// by the puller. Field names must not change without updating that
// consumer.
type pullEnvelope struct {
	Slug   string  `json:"slug"`
	Field  string  `json:"field"`
	Value  *string `json:"value"`            // null when the tracker has no value
	Status string  `json:"status,omitempty"` // ok | no-tracker | not-found | failed
	Error  *string `json:"error,omitempty"`
}

// runSyncPullField implements `sync pull <slug> --field <name> [--json]`.
// It GETs the current tracker-side value of the named field(s) for a
// tracker-backed spec and emits one {slug, field, value} envelope per
// field. READ-ONLY: it never writes the tracker and never writes the
// local spec.md — the caller (hero-code's SyncPuller) applies the value
// locally. The Swift caller sends exactly one --field per invocation;
// multiple --field flags emit one envelope per field, one per line.
func runSyncPullField(slug string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Resolve slug → spec. A missing spec is a hard error (the caller
	// named something that doesn't exist locally).
	s, err := resolveSpecBySlug(heroDir, slug)
	if err != nil {
		return err
	}

	// No tracker_id, or no tracker configured → graceful null-value
	// envelope(s), no crash. The caller reads null and skips the field.
	if s.TrackerID == "" || cfg.Tracker == nil || cfg.Tracker.Type == "" || cfg.Tracker.Type == "none" {
		for _, field := range syncPullFields {
			emitPull(pullEnvelope{Slug: slug, Field: field, Value: nil, Status: "no-tracker"})
		}
		return nil
	}

	t, err := newTrackerForPull(cfg, projectRoot)
	if err != nil {
		return fmt.Errorf("initializing tracker: %w", err)
	}

	// Read the current tracker values once; pull each named field out of
	// the returned map. GetFields is the read side already used by the
	// push diff path (internal/cli/sync_push.go diffPatch).
	remote, err := t.GetFields(s.TrackerID)
	if err != nil {
		return finishPullWithError(slug, err)
	}

	for _, field := range syncPullFields {
		env := pullEnvelope{Slug: slug, Field: field, Status: "ok"}
		if v, ok := remote[field]; ok && v.Kind != tracker.ValueNull {
			str := v.String()
			env.Value = &str
		} else {
			// Tracker has no value for this field → null (skip).
			env.Value = nil
			env.Status = "not-found"
		}
		emitPull(env)
	}
	return nil
}

// finishPullWithError emits a failure envelope for each requested field
// and maps the error to the AC-mandated exit code: 401/403 → exit 2,
// everything else → exit 1 (via the returned error). The 429 retry is
// already applied inside the adapter's GetFields (FieldError surfaces a
// rate-limit only after the single retry is exhausted). Mirrors
// finishWithError on the push side.
func finishPullWithError(slug string, err error) error {
	msg := err.Error()
	for _, field := range syncPullFields {
		emitPull(pullEnvelope{Slug: slug, Field: field, Value: nil, Status: "failed", Error: &msg})
	}

	var fe *tracker.FieldError
	if errors.As(err, &fe) && fe.Kind == tracker.FieldErrorAuth {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
		osExitPull(2)
		return err
	}
	// Returning the error lets cobra/main exit 1 with the message.
	return err
}

// emitPull prints the pull envelope as a single JSON line on stdout
// (always — the caller is a subprocess that decodes JSON). A human form
// is printed only when --json is absent.
func emitPull(env pullEnvelope) {
	if syncPullJSON {
		data, _ := json.Marshal(env)
		fmt.Println(string(data))
		return
	}
	if env.Value != nil {
		fmt.Printf("%s.%s = %s\n", env.Slug, env.Field, *env.Value)
	} else {
		fmt.Printf("%s.%s = (no tracker value)\n", env.Slug, env.Field)
	}
}

// mapTrackerStatus attempts to map a tracker-native status string to a spec Status string.
// Returns empty string if no mapping is found.
func mapTrackerStatus(trackerStatus, trackerType string) string {
	normalized := strings.ToLower(strings.TrimSpace(trackerStatus))

	// Common mappings across tracker types
	switch normalized {
	case "open", "to do", "todo", "backlog", "new":
		return string(spec.StatusPlanning)
	case "in progress", "in_progress", "started", "doing":
		return string(spec.StatusDelivering)
	case "in review", "in_review", "review":
		return string(spec.StatusInReview)
	case "closed", "done", "resolved", "completed", "complete":
		return string(spec.StatusCompleted)
	case "cancelled", "canceled", "rejected", "won't do", "wont do", "won't fix", "wontfix", "duplicate":
		return string(spec.StatusSuperseded)
	}

	// GitHub-specific
	if trackerType == "github" {
		switch normalized {
		case "open":
			return string(spec.StatusPlanning)
		case "closed":
			return string(spec.StatusCompleted)
		}
	}

	return ""
}
