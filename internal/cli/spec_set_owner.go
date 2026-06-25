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
	"github.com/spf13/cobra"
)

var (
	setOwnerFrom        string
	setOwnerTrackerPush bool
	setOwnerJSON        bool
)

var specSetOwnerCmd = &cobra.Command{
	Use:   "set-owner <slug> <new-owner>",
	Short: "Flip a spec's owner, recording the change in owner_history",
	Long: `Flips a spec's canonical owner and appends the transition to its
owner_history timeline. The current active entry's to-timestamp is closed
to now; a fresh active entry is appended for the new owner; the top-level
owner: field is updated. spec.md is rewritten atomically.

If the spec has no owner_history yet, a single-entry history is
synthesized from the current owner: field using the file mtime as that
entry's from — the same rule the workspace loader uses at read time.

new-owner must be one of the canonical owner roles:
  pm | engineering | qa | devops | design | docs

With --tracker-push, and only when the spec has a tracker_id and the
active vocabulary declares owner as a tracker-visible content field, the
flip is also pushed to the tracker via 'hero sync push'. Otherwise the
push is skipped with a one-line note.

Examples:
  hero spec set-owner my-story engineering
  hero spec set-owner my-story engineering --tracker-push --json`,
	Args: cobra.ExactArgs(2),
	RunE: runSpecSetOwnerCmd,
}

// errInvalidOwner is returned by the testable core when new-owner is not
// in the canonical enum. The cobra RunE wrapper maps it to exit code 2
// (AC: refuse with exit code 2 and a clear error). Keeping the core
// exit-free lets the table-driven test assert on the error directly.
type invalidOwnerError struct{ owner string }

func (e *invalidOwnerError) Error() string {
	owners := spec.CanonicalOwners()
	sort.Strings(owners)
	return fmt.Sprintf("%q is not a valid owner (want one of: %s)",
		e.owner, strings.Join(owners, ", "))
}

// runSpecSetOwnerCmd is the cobra entry point. It delegates to the
// exit-free core and translates the invalid-owner sentinel into the
// AC-mandated exit code 2.
func runSpecSetOwnerCmd(cmd *cobra.Command, args []string) error {
	err := runSpecSetOwner(cmd, args)
	var ioe *invalidOwnerError
	if errors.As(err, &ioe) {
		fmt.Fprintf(os.Stderr, "Error: %s\n", ioe.Error())
		os.Exit(2)
	}
	return err
}

func init() {
	specSetOwnerCmd.Flags().StringVar(&setOwnerFrom, "from", "", "expected current owner (validated when set)")
	specSetOwnerCmd.Flags().BoolVar(&setOwnerTrackerPush, "tracker-push", false, "also push owner to the tracker when the vocabulary allows it")
	specSetOwnerCmd.Flags().BoolVar(&setOwnerJSON, "json", false, "emit a machine-readable JSON envelope on stdout")

	specCmd.AddCommand(specSetOwnerCmd)
}

// ownerSyncPusher invokes `hero sync push <slug> --field owner=<value>`.
// It is a package-level seam so tests can substitute a fake that records
// the call without spawning a subprocess or touching the network. The
// real implementation runs the push in-process via runSyncPush so the
// emitted envelope is identical to a standalone `hero sync push`.
var ownerSyncPusher = realOwnerSyncPush

// realOwnerSyncPush performs the tracker push by reusing the existing
// sync-push command path with the patch field source. Returns the list
// of pushed fields (for the envelope) or an error.
func realOwnerSyncPush(slug, newOwner string) ([]string, error) {
	// Reuse the existing patch path: push exactly owner=<newOwner>.
	prevFields := syncPushFields
	prevSource := syncPushFieldSource
	prevJSON := syncPushJSON
	defer func() {
		syncPushFields = prevFields
		syncPushFieldSource = prevSource
		syncPushJSON = prevJSON
	}()
	syncPushFields = []string{"owner=" + newOwner}
	syncPushFieldSource = "patch"
	syncPushJSON = false
	if err := runSyncPush(syncPushCmd, []string{slug}); err != nil {
		return nil, err
	}
	return []string{"owner"}, nil
}

func runSpecSetOwner(cmd *cobra.Command, args []string) error {
	start := time.Now()
	slug := args[0]
	newOwner := args[1]

	// Validate the requested owner against the canonical enum. The
	// cobra wrapper maps this sentinel to exit code 2 (AC).
	if !spec.IsValidOwner(newOwner) {
		return &invalidOwnerError{owner: newOwner}
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	s, err := resolveSpecBySlug(heroDir, slug)
	if err != nil {
		return err
	}

	// Optional --from guard: refuse if the recorded owner doesn't match
	// the caller's expectation (lost-update protection for the UI).
	if setOwnerFrom != "" {
		current := s.Owner
		if active := spec.ActiveOwner(s.OwnerHistory); active != "" {
			current = active
		}
		if current != setOwnerFrom {
			return fmt.Errorf("--from %q does not match current owner %q", setOwnerFrom, current)
		}
	}

	// Read or synthesize the history, then append the flip.
	history := s.OwnerHistory
	if len(history) == 0 {
		history = spec.SynthesizeHistory(s.Owner, s.ModifiedAt)
	}
	now := time.Now().UTC()
	history = spec.AppendOwnerFlip(history, newOwner, now)

	// Rewrite the frontmatter: owner_history block + top-level owner.
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return fmt.Errorf("reading spec: %w", err)
	}
	updated := spec.SetOwnerHistoryBlock(string(data), history)
	updated = spec.SetFrontmatterField(updated, "owner", newOwner)

	if err := spec.AtomicWriteFile(s.Path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("writing spec: %w", err)
	}

	env := ownerFlipEnvelope{
		Slug:          slug,
		TrackerID:     s.TrackerID,
		Status:        "pushed",
		PushedFields:  []string{},
		SkippedFields: []string{},
	}
	if cfg.Tracker != nil {
		env.Tracker = cfg.Tracker.Type
	}

	// Tracker push: only when requested, the spec has a tracker_id, and
	// the active vocabulary declares owner as a content field.
	var note string
	switch {
	case !setOwnerTrackerPush:
		env.SkippedFields = append(env.SkippedFields, "owner")
		note = "skipped tracker push (--tracker-push not set)"
	case s.TrackerID == "":
		env.SkippedFields = append(env.SkippedFields, "owner")
		note = "skipped tracker push (spec has no tracker_id)"
	case !ownerIsTrackerContentField():
		env.SkippedFields = append(env.SkippedFields, "owner")
		note = "skipped tracker push (owner is not a tracker-visible content field)"
	default:
		pushed, perr := ownerSyncPusher(slug, newOwner)
		if perr != nil {
			env.Status = "failed"
			msg := perr.Error()
			env.Error = &msg
			env.DurationMs = time.Since(start).Milliseconds()
			emitOwnerFlip(env, "")
			return perr
		}
		env.PushedFields = pushed
	}

	env.DurationMs = time.Since(start).Milliseconds()
	emitOwnerFlip(env, fmt.Sprintf("Flipped %s owner → %s%s", slug, newOwner, noteSuffix(note)))
	return nil
}

// ownerIsTrackerContentField reports whether the active vocabulary
// classifies `owner` as a tracker-visible content field. The default
// push table (sync_push_diff.go) does not list owner — owner defaults to
// org-state per the spec's "Tracker round-trip" section — so this is
// false unless a vocabulary explicitly promotes it. Wired through
// classifyField so a future vocabulary that adds owner as content
// enables push without a code change here.
func ownerIsTrackerContentField() bool {
	cf, known := classifyField("owner")
	return known && cf.Class == classContent
}

func noteSuffix(note string) string {
	if note == "" {
		return ""
	}
	return " — " + note
}

// ownerFlipEnvelope mirrors pushEnvelope (sync_push.go) field-for-field
// so the Swift caller (runStoryHandoff / hero-pm-sync-native) parses the
// same shape from either command.
type ownerFlipEnvelope struct {
	Slug          string   `json:"slug"`
	Tracker       string   `json:"tracker"`
	TrackerID     string   `json:"tracker_id"`
	Status        string   `json:"status"` // pushed | synced | failed
	PushedFields  []string `json:"pushed_fields"`
	SkippedFields []string `json:"skipped_fields"`
	Error         *string  `json:"error"`
	DurationMs    int64    `json:"duration_ms"`
}

func emitOwnerFlip(env ownerFlipEnvelope, human string) {
	if setOwnerJSON {
		data, _ := json.Marshal(env)
		fmt.Println(string(data))
		return
	}
	if human != "" {
		fmt.Println(human)
	}
}
