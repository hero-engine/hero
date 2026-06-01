package sizing

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// AmbientDriftReport summarises the workspace-wide drift count after
// applying the noise threshold (active spec, recently-changed specs,
// horizon: now initiatives without declared size) and the
// 24-hour stop-nagging window. Surfaces (NEXT.md, hero_pulse,
// hero_kickoff, delivery-lead pre-flight) consume this verbatim — the
// Hint is the canonical phrasing and must not be re-composed at the
// call site. See spec roadmap-review-ambient-surfacing.
type AmbientDriftReport struct {
	// Count is the number of drift entries that passed the noise
	// threshold. Always 0 when Quiet is true.
	Count int
	// Hint is the paste-ready one-line message. Empty when Quiet or
	// Count == 0. Example:
	//   "3 specs have size drift — run /roadmap-review to triage"
	Hint string
	// Quiet is true when no surface should fire (either the filtered
	// count was zero, or the stop-nagging window suppresses).
	Quiet bool
	// Reason is a short diagnostic explaining the Quiet flag:
	//   "no drift"          — nothing passed the filter
	//   "recently triaged"  — stop-nagging window active
	// Empty when Quiet is false.
	Reason string
	// LegacyExcluded is the number of `(unset)`-predating-the-field
	// containers excluded from the count because they didn't meet
	// rule 3 (horizon: now). Diagnostic only — surfaces ignore it.
	LegacyExcluded int
}

// AmbientDriftSummary is the count+hint pair surfaces emit when not
// quiet. Sliced out of AmbientDriftReport so pulse-style responses can
// embed it without leaking diagnostic fields. Nil/zero is the quiet
// state on the wire.
type AmbientDriftSummary struct {
	Count int    `json:"count"`
	Hint  string `json:"hint"`
}

// AmbientDriftOpts tunes the AmbientDrift call.
type AmbientDriftOpts struct {
	// ActiveSpec is the slug the session is currently touching, when
	// known. Empty (commit-time NEXT.md regeneration) skips rule 1.
	ActiveSpec string
	// RecencyDays is the rule-2 window. <=0 → default 7.
	RecencyDays int
	// StopNaggingHours is the suppression window after the newest
	// session record. <=0 → default 24.
	StopNaggingHours int
	// Now is injectable for tests. Zero → time.Now().
	Now time.Time
}

// AmbientDrift returns the filtered drift count + paste-ready hint
// that all three ambient surfaces share. heroDir is the workspace's
// `.hero/` directory (used to read specs and the session-record
// directory); projectRoot is its parent (used for git mtime). Returns
// a zero-value report on hard errors (the surfacing layer should fail
// quietly — drift is informational, not a gate).
func AmbientDrift(heroDir, projectRoot string, opts AmbientDriftOpts) AmbientDriftReport {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	recencyDays := opts.RecencyDays
	if recencyDays <= 0 {
		recencyDays = 7
	}
	stopNaggingHours := opts.StopNaggingHours
	if stopNaggingHours <= 0 {
		stopNaggingHours = 24
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return AmbientDriftReport{Quiet: true, Reason: "no drift"}
	}

	leaf, rawContainer := CollectDrift(specs)
	container := make([]containerLike, 0, len(rawContainer))
	for _, c := range rawContainer {
		container = append(container, containerDriftAdapter{slug: c.Slug, declared: c.Declared})
	}
	bySlug := make(map[string]*spec.Spec, len(specs))
	for _, s := range specs {
		if s == nil {
			continue
		}
		bySlug[s.Slug] = s
	}

	recencyCutoff := now.Add(-time.Duration(recencyDays) * 24 * time.Hour)
	filtered, legacyExcluded := filterAmbientDrift(leaf, container, bySlug, opts.ActiveSpec, projectRoot, recencyCutoff)

	if filtered == 0 {
		return AmbientDriftReport{Quiet: true, Reason: "no drift", LegacyExcluded: legacyExcluded}
	}

	// Stop-nagging: consult the newest session record under
	// `.hero/knowledge/roadmap-review-sessions/`.
	suppress, recordedCount, ok := checkStopNagging(heroDir, now, time.Duration(stopNaggingHours)*time.Hour)
	if suppress {
		// Exception: lift suppression if the filtered count has grown
		// since the recorded `drift_count_at_exit`. Missing field
		// (ok=false) is fully suppressive per the spec contract.
		if ok && filtered > recordedCount {
			// Fall through — count grew.
		} else {
			return AmbientDriftReport{Quiet: true, Reason: "recently triaged", LegacyExcluded: legacyExcluded}
		}
	}

	return AmbientDriftReport{
		Count:          filtered,
		Hint:           formatAmbientHint(filtered),
		LegacyExcluded: legacyExcluded,
	}
}

// formatAmbientHint composes the canonical lens-agnostic phrasing.
// Singular vs plural grammar is handled here so all three surfaces
// quote the exact same string. Lens label is "size drift" — never
// "sizing-lens drift" or "roadmap-shape concern" (locked by the spec).
func formatAmbientHint(count int) string {
	if count == 1 {
		return "1 spec has size drift — run /roadmap-review to triage"
	}
	return fmt.Sprintf("%d specs have size drift — run /roadmap-review to triage", count)
}

// filterAmbientDrift applies the OR-joined noise threshold to the
// drift set and returns (filteredCount, legacyExcluded). The three
// rules are:
//
//  1. active spec — slug matches opts.ActiveSpec
//  2. recency — spec.md committed within RecencyDays
//  3. high-impact — initiative with `horizon: now` and no declared size
//
// Container drift entries from initiatives that don't meet rule 3 and
// are unset-predating-the-field are tallied into legacyExcluded for
// diagnostic surfaces; they do NOT count toward filteredCount.
func filterAmbientDrift(leaf []Estimate, container []containerLike, bySlug map[string]*spec.Spec, activeSpec, projectRoot string, recencyCutoff time.Time) (int, int) {
	count := 0
	legacy := 0

	// Cache git mtimes per spec path so we don't shell out repeatedly
	// for the same spec across leaf+container or duplicate calls.
	mtimes := make(map[string]time.Time)
	getMtime := func(path string) time.Time {
		if path == "" {
			return time.Time{}
		}
		if t, ok := mtimes[path]; ok {
			return t
		}
		t := gitMtime(projectRoot, path)
		mtimes[path] = t
		return t
	}

	for _, est := range leaf {
		s := bySlug[est.Slug]
		if matchesAmbientRules(s, est.Slug, activeSpec, recencyCutoff, getMtime) {
			count++
		}
	}
	for _, c := range container {
		s := bySlug[c.GetSlug()]
		matched := matchesAmbientRules(s, c.GetSlug(), activeSpec, recencyCutoff, getMtime)
		if matched {
			count++
			continue
		}
		// Track legacy `(unset)` containers separately — these are the
		// ~10 initiatives that predate the `size:` field. They're
		// excluded from the surfaced count when they don't meet any
		// rule (rule 3 already would have matched above).
		if c.GetDeclared() == "" && !matched {
			legacy++
		}
	}
	return count, legacy
}

// matchesAmbientRules evaluates the three OR-joined rules. A nil spec
// pointer is treated as "no metadata available" — only the active-spec
// slug match can fire.
func matchesAmbientRules(s *spec.Spec, slug, activeSpec string, recencyCutoff time.Time, getMtime func(string) time.Time) bool {
	// Rule 1 — active spec.
	if activeSpec != "" && slug == activeSpec {
		return true
	}
	if s == nil {
		return false
	}
	// Rule 3 — high-impact unsized horizon-now initiative.
	if s.Type == spec.TypeInitiative && s.EffectiveHorizon() == spec.HorizonNow && s.Size == "" {
		return true
	}
	// Rule 2 — recency window via git mtime of the spec file.
	mt := getMtime(s.Path)
	if !mt.IsZero() && mt.After(recencyCutoff) {
		return true
	}
	return false
}

// containerLike is the subset of snapshot.ContainerDriftReport that
// filterAmbientDrift needs. Declared as an interface to keep this file
// from importing snapshot directly (sizing already imports snapshot,
// so this is purely a future-proofing accommodation — see the adapter
// in CollectDrift's caller).
type containerLike interface {
	GetSlug() string
	GetDeclared() string
}

// containerDriftAdapter wraps snapshot.ContainerDriftReport into the
// minimal containerLike shape filterAmbientDrift needs.
type containerDriftAdapter struct {
	slug, declared string
}

func (c containerDriftAdapter) GetSlug() string     { return c.slug }
func (c containerDriftAdapter) GetDeclared() string { return c.declared }

// checkStopNagging inspects `.hero/knowledge/roadmap-review-sessions/`
// for the newest file, parses its frontmatter, and reports:
//   - suppress: true when newest mtime is within the window
//   - recordedCount: the parsed `drift_count_at_exit:` value, when present
//   - ok: true when the field was present in the newest record
//
// When the directory doesn't exist or is empty, suppress=false (no
// session record → never suppress). When the field is missing
// (forward-compatibility), suppress=true with ok=false — fully
// suppressive, per the spec's missing-field fallback rule.
func checkStopNagging(heroDir string, now time.Time, window time.Duration) (suppress bool, recordedCount int, ok bool) {
	dir := filepath.Join(heroDir, "knowledge", "roadmap-review-sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, 0, false
	}
	type sessFile struct {
		path string
		mod  time.Time
	}
	var files []sessFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, sessFile{
			path: filepath.Join(dir, e.Name()),
			mod:  info.ModTime(),
		})
	}
	if len(files) == 0 {
		return false, 0, false
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].mod.After(files[j].mod)
	})
	newest := files[0]
	if now.Sub(newest.mod) > window {
		return false, 0, false
	}
	// Within the suppression window — parse the frontmatter to extract
	// drift_count_at_exit. Missing field is fully suppressive.
	count, fieldOK := readDriftCountAtExit(newest.path)
	return true, count, fieldOK
}

// readDriftCountAtExit parses the frontmatter block of a session
// record and returns the `drift_count_at_exit:` value (when present).
// Best-effort: any parse error or missing field returns (0, false).
func readDriftCountAtExit(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return 0, false
	}
	// Trim leading "---" line.
	rest := content[3:]
	// Drop the leading newline after "---".
	if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	} else if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	}
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return 0, false
	}
	frontmatter := rest[:end]
	for _, line := range strings.Split(frontmatter, "\n") {
		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if key != "drift_count_at_exit" {
			continue
		}
		val := strings.TrimSpace(line[idx+1:])
		// Strip trailing inline comments.
		if hash := strings.Index(val, "#"); hash >= 0 {
			val = strings.TrimSpace(val[:hash])
		}
		n, err := strconv.Atoi(val)
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}

// gitMtime returns the timestamp of the most recent commit touching
// the given absolute file path inside projectRoot. Zero time on any
// error (no git, file untracked, etc) — callers treat that as
// "outside recency window."
func gitMtime(projectRoot, absPath string) time.Time {
	if projectRoot == "" || absPath == "" {
		return time.Time{}
	}
	rel, err := filepath.Rel(projectRoot, absPath)
	if err != nil {
		rel = absPath
	}
	cmd := exec.Command("git", "-C", projectRoot, "log", "-1", "--pretty=format:%aI", "--", rel)
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return time.Time{}
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, line); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05 -0700", line); err == nil {
		return t
	}
	return time.Time{}
}
