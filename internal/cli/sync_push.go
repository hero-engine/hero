package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var (
	syncPushFields      []string
	syncPushDryRun      bool
	syncPushJSON        bool
	syncPushFieldSource string
)

// newTrackerForPush builds the tracker adapter for the field-level push path.
// It is a package var (mirroring newTrackerForPull) so tests can inject a
// no-network mock. Production wiring is tracker.NewWithJiraConfig — the same
// constructor the rest of the sync commands use.
var newTrackerForPush = func(cfg config.Config, projectRoot string) (tracker.Tracker, error) {
	return tracker.NewWithJiraConfig(cfg.Tracker, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
}

var syncPushCmd = &cobra.Command{
	Use:   "push <slug>",
	Short: "Push spec field changes to the tracker (field-level diff)",
	Long: `Pushes content fields (title, description, points, priority, labels) from
a local spec to its tracker issue.

Two modes:
  diff  (default)  Diff the local spec against the tracker and push every
                   content field that differs. Idempotent — a second run
                   with no local changes makes no write call.
  patch            Push exactly the fields named with --field, skipping the
                   diff fetch. The hot path for tooling that already knows
                   what changed (e.g. hero-code's subprocess wrapper).

Only content-classified fields push. org-state fields (tracker_id, created,
reporter, <provider>_status) are tracker-owned and refused.

Examples:
  hero sync push demo-story --field title="New title"
  hero sync push demo-story
  hero sync push demo-story --dry-run
  hero sync push demo-story --field points=5 --field-source patch --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSyncPush,
}

func init() {
	syncPushCmd.Flags().StringArrayVar(&syncPushFields, "field", nil, "field to push as name=value (repeatable)")
	syncPushCmd.Flags().BoolVar(&syncPushDryRun, "dry-run", false, "compute and print the patch without writing to the tracker")
	syncPushCmd.Flags().BoolVar(&syncPushJSON, "json", false, "emit a machine-readable JSON envelope on stdout")
	syncPushCmd.Flags().StringVar(&syncPushFieldSource, "field-source", "diff", "patch (push --field exactly) or diff (diff local vs tracker)")

	syncCmd.AddCommand(syncPushCmd)
}

// pushEnvelope is the stable JSON contract consumed by hero-code's
// subprocess wrapper (hero-pm-sync-native). Field names and the status
// vocabulary must not change without updating that consumer.
type pushEnvelope struct {
	Slug          string   `json:"slug"`
	Tracker       string   `json:"tracker"`
	TrackerID     string   `json:"tracker_id"`
	Status        string   `json:"status"` // synced | pushed | failed | dry-run
	PushedFields  []string `json:"pushed_fields"`
	SkippedFields []string `json:"skipped_fields"`
	Error         *string  `json:"error"`
	DurationMs    int64    `json:"duration_ms"`
}

func runSyncPush(cmd *cobra.Command, args []string) error {
	start := time.Now()
	slug := args[0]

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Resolve slug → spec.
	s, err := resolveSpecBySlug(heroDir, slug)
	if err != nil {
		return err
	}

	// No tracker_id → success no-op (AC: local-only specs are valid).
	if s.TrackerID == "" {
		env := pushEnvelope{
			Slug:          slug,
			TrackerID:     "",
			Status:        "synced",
			PushedFields:  []string{},
			SkippedFields: []string{},
			DurationMs:    time.Since(start).Milliseconds(),
		}
		if cfg.Tracker != nil {
			env.Tracker = cfg.Tracker.Type
		}
		emitPush(env, "no tracker_id; skipping")
		return nil
	}

	if cfg.Tracker == nil || cfg.Tracker.Type == "none" {
		return fmt.Errorf("no tracker configured — set tracker.type in hero.json")
	}

	t, err := newTrackerForPush(cfg, projectRoot)
	if err != nil {
		return fmt.Errorf("initializing tracker: %w", err)
	}

	env := pushEnvelope{
		Slug:          slug,
		Tracker:       t.Name(),
		TrackerID:     s.TrackerID,
		PushedFields:  []string{},
		SkippedFields: []string{},
	}

	// Build the patch — either from --field (patch source) or by diffing.
	// mergeCommit, when non-nil, applies the shared-field merge's local
	// write-back + conflict note + baseline advance AFTER the network push
	// succeeds, so a push failure never leaves a half-write. BOTH the diff path
	// and the --field patch path set it — the app pushes via --field, so the
	// patch path must protect shared fields too.
	var patch map[string]tracker.Value
	var mergeCommit func() error
	switch syncPushFieldSource {
	case "patch":
		patch, mergeCommit, err = patchPush(heroDir, t, s, syncPushFields, &env)
		if err != nil {
			return err
		}
	case "diff":
		if len(syncPushFields) > 0 {
			// Explicit --field flags imply the patch path even under the
			// diff default — pushing exactly what was named.
			patch, mergeCommit, err = patchPush(heroDir, t, s, syncPushFields, &env)
			if err != nil {
				return err
			}
		} else {
			patch, mergeCommit, err = diffPatch(heroDir, t, s, &env)
			if err != nil {
				return finishWithError(env, start, err)
			}
		}
	default:
		return fmt.Errorf("invalid --field-source %q (want patch or diff)", syncPushFieldSource)
	}

	env.PushedFields = sortedPatchKeys(patch)

	// Empty push patch → nothing to write to the tracker. But a shared-field
	// merge may still have local write-backs (only-remote-changed: pull the
	// upstream value into the spec) + a baseline advance to commit. Under
	// dry-run we skip the commit; otherwise we commit and report in-sync.
	if len(patch) == 0 {
		if !syncPushDryRun && mergeCommit != nil {
			if err := mergeCommit(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: shared-field merge write-back failed for %s: %v\n", slug, err)
			}
		}
		env.Status = "synced"
		env.DurationMs = time.Since(start).Milliseconds()
		emitPush(env, fmt.Sprintf("%s in sync — nothing to push", slug))
		return nil
	}

	// Dry-run → print the patch, make no write (AC: dry-run).
	if syncPushDryRun {
		env.Status = "dry-run"
		env.DurationMs = time.Since(start).Milliseconds()
		emitPushDryRun(env, patch)
		return nil
	}

	// Perform the write.
	if err := t.UpdateFields(s.TrackerID, patch); err != nil {
		return finishWithError(env, start, err)
	}

	// Push succeeded → now (and only now) apply the shared-field merge's local
	// write-back + baseline advance. Ordering after the network write keeps the
	// "never half-write" guarantee: a push failure above leaves the spec and
	// baseline at their pre-sync state and the merge retries next run.
	if mergeCommit != nil {
		if err := mergeCommit(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: shared-field merge write-back failed for %s: %v\n", slug, err)
		}
	}

	env.Status = "pushed"
	env.DurationMs = time.Since(start).Milliseconds()
	emitPush(env, fmt.Sprintf("Pushed %d field(s) to %s issue %s: %s",
		len(env.PushedFields), t.Name(), s.TrackerID, strings.Join(env.PushedFields, ", ")))
	return nil
}

// patchFromFlags parses --field name=value flags into a patch map,
// classifying each field and refusing org-state fields (AC: org-state
// refusal → non-zero exit). Unknown fields are skipped with a warning.
func patchFromFlags(flags []string, env *pushEnvelope) (map[string]tracker.Value, error) {
	patch := map[string]tracker.Value{}
	for _, raw := range flags {
		name, value, ok := strings.Cut(raw, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --field %q (want name=value)", raw)
		}
		name = strings.TrimSpace(name)

		cf, known := classifyField(name)
		if known && cf.Class == classOrgState {
			return nil, fmt.Errorf("field %q is org-state (tracker-owned) and cannot be pushed", name)
		}
		if !known {
			fmt.Fprintf(os.Stderr, "Warning: field %q is not a known content field; skipping\n", name)
			env.SkippedFields = append(env.SkippedFields, name)
			continue
		}
		patch[name] = tracker.ParseScalar(value, cf.Hint)
	}
	return patch, nil
}

// patchPush builds the push patch for the --field path. It parses the flags
// (org-state refusal + unknown-field skipping unchanged), then splits them:
//
//   - non-shared fields (hero-owned status/size, priority, points, …) → direct
//     patch-push, exactly as before. Those are hero-authoritative; no 3-way
//     merge, and no remote fetch when NO shared field is named (the hot path
//     the app uses for status stays a single write, no extra round-trip).
//   - shared fields (title/body/tags) → routed through the SAME 3-way merge the
//     diff path uses (runSharedMerge), with the --field value as "local" and
//     the fetched remote as the upstream side. Never a blind push over a
//     concurrent upstream edit. This closes the gap where an app push via
//     `--field title=…` clobbered upstream.
//
// A GetFields fetch failure (only reached when a shared field is named) leaves
// both sides untouched — the whole shared-field merge is skipped and the shared
// --field values are NOT pushed (fail-safe: retry next run, no half-write). The
// returned mergeCommit is nil unless the shared merge ran, and it is applied by
// the caller only after the network push succeeds.
func patchPush(heroDir string, t tracker.Tracker, s *spec.Spec, flags []string, env *pushEnvelope) (map[string]tracker.Value, func() error, error) {
	patch, err := patchFromFlags(flags, env)
	if err != nil {
		return nil, nil, err
	}

	// Which named fields are shared? Split them out; leave non-shared in patch.
	sharedLocal := map[string]tracker.Value{}
	restrict := map[string]bool{}
	for name, v := range patch {
		if _, shared := syncSharedByCanonical(name); shared {
			sharedLocal[name] = v
			restrict[name] = true
			delete(patch, name) // shared values come from the merge, not the raw flag
		}
	}

	// No shared field named → nothing to merge; the direct patch stands as-is.
	if len(restrict) == 0 {
		return patch, nil, nil
	}

	remote, ferr := t.GetFields(s.TrackerID)
	if ferr != nil {
		// Fail-safe: leave both sides untouched, drop the shared --field values
		// (don't blind-push them), retry next run. Non-shared fields still push.
		fmt.Fprintf(os.Stderr, "Warning: could not fetch remote for shared-field merge (%v) — skipping shared fields this run\n", ferr)
		return patch, nil, nil
	}

	sharedPatch, commit, merr := runSharedMerge(heroDir, s, sharedLocal, remote, restrict)
	if merr != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v — skipping shared-field merge this run\n", merr)
		return patch, nil, nil
	}
	for k, v := range sharedPatch {
		patch[k] = v
	}
	return patch, commit, nil
}

// diffPatch fetches the tracker's current content fields once, then builds the
// push patch from two paths:
//
//   - shared fields (title/description/labels) → 3-way merge against the
//     persisted baseline (mergeSharedFields). Never a blind push; upstream is
//     never lost.
//   - non-shared content fields (priority, points, …) → the existing 2-way diff
//     (unchanged: those are tracker/hero-owned and already safe).
//
// It returns the combined push patch plus a mergeCommit closure the caller runs
// AFTER a successful push to apply the merge's local spec write-back and advance
// the baseline. A fetch failure returns the error and leaves both sides
// untouched (never stuck, never half-write).
func diffPatch(heroDir string, t tracker.Tracker, s *spec.Spec, env *pushEnvelope) (map[string]tracker.Value, func() error, error) {
	remote, err := t.GetFields(s.TrackerID)
	if err != nil {
		return nil, nil, err
	}
	local := localFields(s)

	// Non-shared content fields keep the 2-way diff. Shared fields are excluded
	// here and handled by the 3-way merge (restrict=nil: reconcile all shared).
	patch := Diff(local, remote, nonSharedPushFields())

	sharedPatch, commit, err := runSharedMerge(heroDir, s, local, remote, nil)
	if err != nil {
		// A corrupt/unreadable baseline degrades to no shared-field merge this
		// run (leave shared fields as-is), rather than merging against a bad
		// ancestor. Non-shared fields still push.
		fmt.Fprintf(os.Stderr, "Warning: %v — skipping shared-field merge this run\n", err)
		return patch, nil, nil
	}
	for k, v := range sharedPatch {
		patch[k] = v
	}
	return patch, commit, nil
}

// runSharedMerge runs the shared-field 3-way merge and returns the shared-field
// push patch plus a commit closure (local write-back + conflict note + baseline
// advance) the caller runs AFTER a successful push. It is the single merge entry
// point used by BOTH the diff path (restrict=nil, local from the spec) and the
// --field patch path (restrict = the named shared fields, local from the flags).
func runSharedMerge(
	heroDir string,
	s *spec.Spec,
	local, remote map[string]tracker.Value,
	restrict map[string]bool,
) (map[string]tracker.Value, func() error, error) {
	sharedPatch, writeback, updatedBase, conflictNote, err := mergeSharedFields(heroDir, s, local, remote, restrict)
	if err != nil {
		return nil, nil, err
	}
	commit := func() error {
		if werr := applyLocalWriteback(s.Path, writeback); werr != nil {
			return werr
		}
		if werr := applyConflictNote(s.Path, conflictNote); werr != nil {
			return werr
		}
		return advanceBaseline(heroDir, s, remote, updatedBase)
	}
	return sharedPatch, commit, nil
}

// nonSharedPushFields is pushFields minus the shared set — the fields that keep
// the 2-way diff. Shared fields (title/description/labels) are handled by the
// 3-way merge instead of a blind push.
func nonSharedPushFields() []ClassifiedField {
	out := make([]ClassifiedField, 0, len(pushFields))
	for _, f := range pushFields {
		if _, shared := syncSharedByCanonical(f.Name); shared {
			continue
		}
		out = append(out, f)
	}
	return out
}

// localFields builds the canonical content-field map from a spec's
// typed fields. Only fields the spec actually carries are populated;
// absent fields are omitted so the diff never tries to clear a tracker
// field the spec doesn't mention.
func localFields(s *spec.Spec) map[string]tracker.Value {
	out := map[string]tracker.Value{}
	if s.Title != "" {
		out["title"] = tracker.StringValue(s.Title)
	}
	if s.Description != "" {
		out["description"] = tracker.StringValue(s.Description)
	}
	if s.Priority != "" {
		out["priority"] = tracker.StringValue(s.Priority)
	}
	if len(s.Tags) > 0 {
		out["labels"] = tracker.StringsValue(s.Tags)
	}
	return out
}

// resolveSpecBySlug walks the workspace for a spec whose slug matches.
func resolveSpecBySlug(heroDir, slug string) (*spec.Spec, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}
	for _, s := range specs {
		if s.Slug == slug {
			return s, nil
		}
	}
	return nil, fmt.Errorf("spec %q not found", slug)
}

// finishWithError emits the failure envelope and exits with the
// AC-mandated code: 2 for auth (401/403), 1 otherwise. The envelope is
// printed before exit so the subprocess consumer always gets it.
func finishWithError(env pushEnvelope, start time.Time, err error) error {
	env.Status = "failed"
	msg := err.Error()
	env.Error = &msg
	env.DurationMs = time.Since(start).Milliseconds()
	emitPush(env, "")

	var fe *tracker.FieldError
	if errors.As(err, &fe) && fe.Kind == tracker.FieldErrorAuth {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
		os.Exit(2)
	}
	// Returning the error lets cobra/main exit 1 with the message.
	return err
}

// emitPush prints either the JSON envelope (--json) or a human line.
func emitPush(env pushEnvelope, human string) {
	if syncPushJSON {
		printPushJSON(env, nil)
		return
	}
	if human != "" {
		fmt.Println(human)
	}
}

// emitPushDryRun prints the dry-run patch — JSON envelope plus a
// per-field patch block under --json, or a human-readable patch list.
func emitPushDryRun(env pushEnvelope, patch map[string]tracker.Value) {
	if syncPushJSON {
		printPushJSON(env, patch)
		return
	}
	fmt.Printf("[dry-run] %s — would push to %s issue %s:\n", env.Slug, env.Tracker, env.TrackerID)
	for _, name := range sortedPatchKeys(patch) {
		fmt.Printf("  %s = %s\n", name, patch[name].String())
	}
}

// printPushJSON marshals the envelope (optionally with a patch block for
// dry-run) and prints it on a single line to stdout.
func printPushJSON(env pushEnvelope, patch map[string]tracker.Value) {
	if patch == nil {
		data, _ := json.Marshal(env)
		fmt.Println(string(data))
		return
	}
	// Dry-run: include the concrete patch values alongside the envelope.
	patchJSON := map[string]interface{}{}
	for k, v := range patch {
		patchJSON[k] = v.JSON()
	}
	combined := struct {
		pushEnvelope
		Patch map[string]interface{} `json:"patch"`
	}{pushEnvelope: env, Patch: patchJSON}
	data, _ := json.Marshal(combined)
	fmt.Println(string(data))
}

func sortedPatchKeys(m map[string]tracker.Value) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
