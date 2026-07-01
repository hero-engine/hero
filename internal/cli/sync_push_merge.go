package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
	syncpkg "github.com/hero-engine/hero/internal/sync"
	"github.com/hero-engine/hero/internal/tracker"
)

// mergeSharedFields runs the 3-way shared-field merge, shared by both push
// entry points (the DIFF path and the --field PATCH path).
//
// For each shared field (title/body-as-description/tags-as-labels) it computes
// base (from the persisted baseline), local (the caller-supplied value), remote
// (fetched tracker value) and resolves deterministically via the merge package.
// The result:
//   - a patch of the shared values that must be pushed UP to the tracker (the
//     merged value differs from remote),
//   - local write-backs applied to the spec.md so the spec converges to the
//     merged value (a taken-remote value, or a preserved-local marker),
//   - an updated baseline written to .hero/sync-state/<slug>.json at the
//     converged value,
//   - a terse title-conflict note for the local-only sync_conflict field.
//
// `local` is where the two callers differ: the DIFF path passes the spec's
// fields (localFields); the --field PATCH path passes the flag values. Both run
// the exact same merge.
//
// `restrict`, when non-nil, limits the merge to the shared fields whose
// canonical name is a key (the PATCH path only touches the fields the caller
// named). When nil, all shared fields are reconciled (the DIFF path).
//
// Non-shared fields are NOT handled here — the caller diffs/patches them.
//
// First-run / no-baseline: when the baseline file is absent (a spec imported
// before this feature), remote is adopted as the base for THIS sync, so the
// first sync can never lose an upstream value; subsequent syncs are true 3-way.
//
// Failure isolation (never stuck / never half-write): the remote fetch happens
// once, up front, in the caller; if it fails the whole shared-field merge is
// skipped and both sides are left unchanged. Local write-backs and the baseline
// advance are applied by the caller only after the network push succeeds — so a
// push failure leaves the baseline at the pre-push state and the merge simply
// retries next run (idempotent).
func mergeSharedFields(
	heroDir string,
	s *spec.Spec,
	local, remote map[string]tracker.Value,
	restrict map[string]bool,
) (pushPatch map[string]tracker.Value, localWriteback map[string]tracker.Value, updatedBase map[string]syncpkg.Base, conflictNote string, err error) {
	pushPatch = map[string]tracker.Value{}
	localWriteback = map[string]tracker.Value{}
	updatedBase = map[string]syncpkg.Base{}

	baseline, rerr := syncpkg.ReadBaseline(heroDir, s.Slug)
	if rerr != nil {
		// A corrupt baseline degrades to no-op rather than merging against a
		// bad ancestor. The caller leaves shared fields untouched this run.
		return nil, nil, nil, "", fmt.Errorf("reading sync baseline for %s: %w", s.Slug, rerr)
	}
	baseMap := map[string]syncpkg.Base{}
	if baseline != nil {
		baseMap = baseline.Base
	}

	for _, f := range syncpkg.SharedFields {
		// PATCH path: only reconcile the shared fields the caller named.
		if restrict != nil && !restrict[f.Canonical] {
			continue
		}

		lv, hasLocal := local[f.Canonical]
		rv, hasRemote := remote[f.Canonical]

		// The tracker didn't return this field (adapter doesn't read it, or
		// it's genuinely empty) — nothing to merge; leave it alone.
		if !hasRemote {
			continue
		}

		switch f.Kind {
		case syncpkg.KindTags:
			localTags := valueTags(lv, hasLocal)
			remoteTags := rv.Strings
			baseTags := baseTagsOrRemote(baseMap, f.BaselineKey, remoteTags)

			res := syncpkg.MergeTags(baseTags, localTags, remoteTags)
			if res.PushLocal {
				pushPatch[f.Canonical] = tracker.StringsValue(res.Merged)
			}
			// Converge the spec to the merged set (unless it already matches
			// what the spec has).
			if !tracker.StringsValue(res.Merged).Equal(lv) {
				localWriteback[f.Canonical] = tracker.StringsValue(res.Merged)
			}
			updatedBase[f.BaselineKey] = syncpkg.TagsBase(res.Merged)

		default: // KindTitle, KindBody
			localStr := valueString(lv, hasLocal)
			remoteStr := rv.Str
			baseStr := baseTextOrRemote(baseMap, f.BaselineKey, remoteStr)

			res := syncpkg.MergeText(f.Kind, baseStr, localStr, remoteStr)
			if res.PushLocal {
				pushPatch[f.Canonical] = tracker.StringValue(res.Merged)
			}
			if res.Merged != localStr {
				localWriteback[f.Canonical] = tracker.StringValue(res.Merged)
			}
			// A title both-changed conflict records a terse note for the
			// local-only `sync_conflict` field. It is NOT written to the body —
			// the body is never mutated by a title conflict. Overwritten each
			// sync (last-writer), so notes never accumulate.
			if res.ConflictNote != "" {
				conflictNote = res.ConflictNote
			}
			updatedBase[f.BaselineKey] = syncpkg.TextBase(res.Merged)
		}
	}

	return pushPatch, localWriteback, updatedBase, conflictNote, nil
}

// applyLocalWriteback writes merged shared-field values back into the spec.md
// frontmatter so the local spec converges to the merged value. Only the three
// shared frontmatter fields are touched (title, description, tags). A write
// failure is non-fatal to the push (returned as a warning) — the spec is stale
// but the tracker converged; the next sync reconciles.
func applyLocalWriteback(path string, writeback map[string]tracker.Value) error {
	if len(writeback) == 0 {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	// Canonical field → frontmatter key.
	fmKey := map[string]string{
		"title":       "title",
		"description": "description",
		"labels":      "tags",
	}
	for canonical, v := range writeback {
		key, ok := fmKey[canonical]
		if !ok {
			continue
		}
		content = spec.SetFrontmatterField(content, key, frontmatterScalar(v))
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// applyConflictNote records a title both-changed conflict in the local-only
// `sync_conflict` frontmatter field. It is OVERWRITTEN each sync (never
// appended), so notes never accumulate, and it is Hero-local — not in the
// shared/pushable set — so it never reaches the tracker and never re-syncs. An
// empty note clears a stale record (a re-sync that no longer conflicts drops
// the note). A note is written only when the field already exists or a conflict
// is present, so a clean spec that never conflicted stays untouched.
func applyConflictNote(path, note string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	if note == "" && !strings.Contains(content, "\nsync_conflict:") && !strings.HasPrefix(content, "sync_conflict:") {
		// No conflict this run and no stale note to clear — leave the spec alone.
		return nil
	}
	content = spec.SetFrontmatterField(content, "sync_conflict", quoteScalar(note))
	return os.WriteFile(path, []byte(content), 0o644)
}

// quoteScalar wraps a frontmatter string value in double quotes (escaping any
// embedded quotes), so a note containing colons or quotes stays valid YAML.
func quoteScalar(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

// frontmatterScalar renders a Value as a frontmatter scalar: strings are
// emitted raw, string arrays as an inline `[a, b]` list (which parseList reads
// back). Multi-line strings are collapsed defensively — the shared frontmatter
// fields (title, description) are single-line by contract.
func frontmatterScalar(v tracker.Value) string {
	switch v.Kind {
	case tracker.ValueStrings:
		return "[" + strings.Join(v.Strings, ", ") + "]"
	default:
		return strings.ReplaceAll(v.Str, "\n", " ")
	}
}

// valueString returns a Value's string payload, or "" when the field is absent
// locally (an absent shared field is treated as empty for the merge — the merge
// then takes remote or unions in remote, never inventing a lossy push).
func valueString(v tracker.Value, has bool) string {
	if !has {
		return ""
	}
	return v.Str
}

func valueTags(v tracker.Value, has bool) []string {
	if !has {
		return nil
	}
	return v.Strings
}

// baseTextOrRemote returns the persisted text base for a field, or remote when
// no baseline exists (first-run adoption: adopt upstream so the first sync
// cannot lose it).
func baseTextOrRemote(baseMap map[string]syncpkg.Base, key, remote string) string {
	if b, ok := baseMap[key]; ok && !b.IsTags() {
		return b.Text
	}
	return remote
}

func baseTagsOrRemote(baseMap map[string]syncpkg.Base, key string, remote []string) []string {
	if b, ok := baseMap[key]; ok && b.IsTags() {
		return b.Tags
	}
	return remote
}

// syncSharedByCanonical exposes the merge package's shared-field lookup to the
// push command's field-filtering (nonSharedPushFields).
func syncSharedByCanonical(canonical string) (syncpkg.SharedField, bool) {
	return syncpkg.SharedByCanonical(canonical)
}

// advanceBaseline persists the converged baseline after a successful sync. The
// merge supplies the new values for the shared fields it touched (updatedBase);
// any shared field the merge did not touch (remote absent) is seeded from the
// remote value so the baseline stays complete for next run. Non-shared fields
// are not part of the baseline.
func advanceBaseline(heroDir string, s *spec.Spec, remote map[string]tracker.Value, updatedBase map[string]syncpkg.Base) error {
	baseline := &syncpkg.Baseline{
		TrackerID: s.TrackerID,
		Base:      map[string]syncpkg.Base{},
	}
	for _, f := range syncpkg.SharedFields {
		if b, ok := updatedBase[f.BaselineKey]; ok {
			baseline.Base[f.BaselineKey] = b
			continue
		}
		// Untouched shared field → seed from remote so the base is complete.
		if rv, ok := remote[f.Canonical]; ok {
			if f.Kind == syncpkg.KindTags {
				baseline.Base[f.BaselineKey] = syncpkg.TagsBase(rv.Strings)
			} else {
				baseline.Base[f.BaselineKey] = syncpkg.TextBase(rv.Str)
			}
		}
	}
	return syncpkg.WriteBaseline(heroDir, s.Slug, baseline)
}

// seedBaselineFromIssue writes the last-synced baseline for a spec directly from
// a fetched tracker Issue. Used by the IMPORT and PULL paths, where the tracker
// value has just become the local value (import creates the spec FROM the issue;
// pull converges the spec toward the issue) — so tracker == local for the shared
// fields and that converged value IS the new common ancestor.
//
// Establishing the baseline here (not only on push) is what makes the app's
// pull/import-then-edit flow safe: a base exists BEFORE any local/upstream
// divergence, so the first conflicting push is a true 3-way merge rather than
// the first-run adopt-remote fallback. A write failure is non-fatal to the
// caller (baseline seeding is best-effort; the next push seeds/advances it).
func seedBaselineFromIssue(heroDir, slug string, issue *tracker.Issue) error {
	if issue == nil {
		return nil
	}
	baseline := &syncpkg.Baseline{
		TrackerID: issue.ID,
		Base: map[string]syncpkg.Base{
			"title": syncpkg.TextBase(issue.Title),
			"body":  syncpkg.TextBase(issue.Description),
			"tags":  syncpkg.TagsBase(issue.Labels),
		},
	}
	return syncpkg.WriteBaseline(heroDir, slug, baseline)
}

// advanceBaselineOnPull updates the baseline on `hero sync pull`, per shared
// field, ONLY where the local spec value already equals the tracker value —
// i.e. the two sides are converged and that value is a genuine common ancestor.
// A shared field that differs (a pending local or upstream edit) is left as-is
// so a real ancestor is never clobbered by one side's value; the next push's
// 3-way merge reconciles it. Best-effort: existing baseline fields for
// non-converged shared fields are preserved. No-op when there's nothing to
// converge (keeps pull idempotent and avoids a needless write).
func advanceBaselineOnPull(heroDir string, s *spec.Spec, issue *tracker.Issue) error {
	if issue == nil {
		return nil
	}
	existing, rerr := syncpkg.ReadBaseline(heroDir, s.Slug)
	if rerr != nil {
		return rerr
	}
	baseline := &syncpkg.Baseline{TrackerID: issue.ID, Base: map[string]syncpkg.Base{}}
	if existing != nil {
		if existing.TrackerID != "" {
			baseline.TrackerID = existing.TrackerID
		}
		for k, v := range existing.Base {
			baseline.Base[k] = v
		}
	}

	local := localFields(s)
	changed := false
	for _, f := range syncpkg.SharedFields {
		switch f.Kind {
		case syncpkg.KindTags:
			remote := tracker.StringsValue(issue.Labels)
			if lv, ok := local[f.Canonical]; ok && lv.Equal(remote) {
				baseline.Base[f.BaselineKey] = syncpkg.TagsBase(issue.Labels)
				changed = true
			}
		default:
			var remoteStr string
			if f.BaselineKey == "title" {
				remoteStr = issue.Title
			} else {
				remoteStr = issue.Description
			}
			if lv, ok := local[f.Canonical]; ok && lv.Str == remoteStr {
				baseline.Base[f.BaselineKey] = syncpkg.TextBase(remoteStr)
				changed = true
			}
		}
	}
	if !changed && existing != nil {
		// Nothing newly converged and a baseline already exists — leave it.
		return nil
	}
	return syncpkg.WriteBaseline(heroDir, s.Slug, baseline)
}
