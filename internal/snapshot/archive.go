package snapshot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ArchiveTrigger names the cause of a particular archive write.
type ArchiveTrigger string

const (
	TriggerMilestone ArchiveTrigger = "milestone"
	TriggerManual    ArchiveTrigger = "manual"
	TriggerStaleness ArchiveTrigger = "staleness"
)

// ArchiveDirName is the leaf directory under heroDir where archives
// live. The full path is heroDir + "/" + ArchiveDirName.
const ArchiveDirName = "snapshots"

// HistoricalBanner is the body banner prepended to every archive.
// The writer refuses to emit an archive missing this line, and
// containment tests assert it appears at the top of the body.
const HistoricalBanner = "> **Historical archive captured %s.** This is a point-in-time snapshot, not current state. For live state see [SNAPSHOT.md](../SNAPSHOT.md)."

// ArchiveRecord describes one archive file on disk.
type ArchiveRecord struct {
	Date             string         // YYYY-MM-DD
	Label            string         // optional slug suffix (empty for unlabeled)
	Trigger          ArchiveTrigger
	GitCommit        string
	ProjectorVersion int
	Historical       bool
	NotCurrent       bool
	Path             string // absolute
	Body             string // full body including banner
	BodyOffset       int    // byte offset in the file where body (after frontmatter) starts
}

// ArchiveConfig is the user-visible knob set for archive behavior.
// Zero values use the documented defaults.
type ArchiveConfig struct {
	// StalenessCutoff is one of: "weekly", "biweekly", "monthly",
	// "quarterly", "off". Empty defaults to "monthly".
	StalenessCutoff string
	// MilestonesEnabled toggles release-tag / initiative-completion
	// auto-archive. Default true (set by config Load).
	MilestonesEnabled bool
	// ReleaseTagPattern is a regex (default "v[0-9].*") used to test
	// candidate git tag names for milestone archives.
	ReleaseTagPattern string
	// Retention is one of: "all", "last-N", "none". Empty defaults
	// to "all".
	Retention string
	// RetentionCount is the N for last-N. Ignored otherwise.
	RetentionCount int
}

// stalenessDuration returns the cutoff window or zero when the
// cutoff is "off" / unrecognized.
func (c ArchiveConfig) stalenessDuration() time.Duration {
	switch strings.ToLower(c.StalenessCutoff) {
	case "weekly":
		return 7 * 24 * time.Hour
	case "biweekly":
		return 14 * 24 * time.Hour
	case "", "monthly":
		return 30 * 24 * time.Hour
	case "quarterly":
		return 90 * 24 * time.Hour
	case "off":
		return 0
	}
	return 30 * 24 * time.Hour
}

// TriggerHit is one resolved trigger that should fire on this run.
type TriggerHit struct {
	Trigger ArchiveTrigger
	Label   string // may be empty (e.g. unlabeled manual)
}

// EvaluateTriggersInput is the bundle of state the trigger evaluator
// consumes. Pure-function: same input → same output.
type EvaluateTriggersInput struct {
	Now              time.Time
	ExistingArchives []ArchiveRecord
	// NewlyCompletedInitiatives is a list of initiative slugs whose
	// status flipped to completed on this projector run.
	NewlyCompletedInitiatives []string
	// NewReleaseTags is a list of git tag names created since the
	// most-recent archive.
	NewReleaseTags []string
	// ManualLabel, when non-empty, signals an explicit manual
	// archive invocation (CLI or MCP `archive: true`). The label
	// may itself be empty (manual without --label).
	ManualLabel    string
	ManualRequested bool
	Config         ArchiveConfig
}

// EvaluateTriggers returns the trigger hits that should fire this
// run. Same-day milestones suppress staleness; manual always fires.
//
// First-run safety: when there are no existing archives the
// staleness check uses Now as its baseline (last_archive_date = now),
// so the first projector run never retroactively writes a
// safety-net archive. Milestones / manual still fire.
func EvaluateTriggers(in EvaluateTriggersInput) []TriggerHit {
	var hits []TriggerHit

	if in.ManualRequested {
		hits = append(hits, TriggerHit{Trigger: TriggerManual, Label: in.ManualLabel})
	}

	if in.Config.MilestonesEnabled {
		for _, slug := range in.NewlyCompletedInitiatives {
			hits = append(hits, TriggerHit{Trigger: TriggerMilestone, Label: slug})
		}
		tagPattern := in.Config.ReleaseTagPattern
		if tagPattern == "" {
			tagPattern = "v[0-9].*"
		}
		re, err := regexp.Compile(tagPattern)
		if err == nil {
			for _, tag := range in.NewReleaseTags {
				if re.MatchString(tag) {
					hits = append(hits, TriggerHit{Trigger: TriggerMilestone, Label: tag})
				}
			}
		}
	}

	// Staleness — only when no milestone / manual fired on the same run.
	hasOther := false
	for _, h := range hits {
		if h.Trigger != TriggerStaleness {
			hasOther = true
			break
		}
	}
	if !hasOther {
		cutoff := in.Config.stalenessDuration()
		if cutoff > 0 {
			last := mostRecentArchiveTime(in.ExistingArchives, in.Now)
			if !last.IsZero() && in.Now.Sub(last) > cutoff {
				hits = append(hits, TriggerHit{Trigger: TriggerStaleness, Label: "auto-staleness"})
			}
		}
	}
	return hits
}

func mostRecentArchiveTime(archives []ArchiveRecord, fallback time.Time) time.Time {
	// First-run safety: when no archives exist, return the fallback
	// (typically Now) so the cutoff window can't retroactively fire.
	if len(archives) == 0 {
		return fallback
	}
	var newest time.Time
	for _, a := range archives {
		t, err := time.Parse("2006-01-02", a.Date)
		if err != nil {
			continue
		}
		if t.After(newest) {
			newest = t
		}
	}
	if newest.IsZero() {
		return fallback
	}
	return newest
}

// MaybeWriteInput controls one MaybeWrite call.
type MaybeWriteInput struct {
	Rendered  []byte    // bytes of the live SNAPSHOT.md
	HeroDir   string    // absolute path to .hero/
	Now       time.Time // capture time (controls filename + frontmatter)
	GitCommit string    // optional HEAD sha
	Triggers  []TriggerHit
}

// MaybeWriteResult reports archive writes performed during one call.
type MaybeWriteResult struct {
	Written []ArchiveRecord
	Skipped []TriggerHit // triggers suppressed by same-day idempotency
}

// MaybeWrite evaluates the supplied triggers and writes archive
// files for those that fire. Idempotency rules:
//   - Same-day same-label trigger that finds an existing file is a no-op.
//   - Manual + non-empty label always writes; the filename gains "-2",
//     "-3" suffixes when the label-day-pair is reused on the same day.
func MaybeWrite(in MaybeWriteInput) (MaybeWriteResult, error) {
	var result MaybeWriteResult
	if len(in.Triggers) == 0 {
		return result, nil
	}
	if len(in.Rendered) == 0 {
		return result, errors.New("snapshot: archive needs non-empty rendered bytes")
	}
	archiveDir := filepath.Join(in.HeroDir, ArchiveDirName)
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return result, fmt.Errorf("create %s: %w", archiveDir, err)
	}
	date := in.Now.UTC().Format("2006-01-02")

	for _, hit := range in.Triggers {
		filename := dateFilename(date, hit.Label)
		path := filepath.Join(archiveDir, filename)
		if _, err := os.Stat(path); err == nil {
			// Same-day no-op (idempotent) unless a labeled manual.
			if hit.Trigger == TriggerManual && hit.Label != "" {
				path = nextSuffixedPath(archiveDir, date, hit.Label)
			} else {
				result.Skipped = append(result.Skipped, hit)
				continue
			}
		}
		body, err := buildArchiveBody(in.Rendered, date, hit.Trigger, hit.Label, in.GitCommit)
		if err != nil {
			return result, err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return result, fmt.Errorf("write %s: %w", path, err)
		}
		result.Written = append(result.Written, ArchiveRecord{
			Date:             date,
			Label:            hit.Label,
			Trigger:          hit.Trigger,
			GitCommit:        in.GitCommit,
			ProjectorVersion: ProjectorVersion,
			Historical:       true,
			NotCurrent:       true,
			Path:             path,
		})
	}
	return result, nil
}

func dateFilename(date, label string) string {
	if label == "" {
		return date + ".md"
	}
	return date + "--" + safeSlug(label) + ".md"
}

func safeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '.':
			b.WriteRune(r)
		case r == ' ' || r == '_' || r == '/':
			b.WriteRune('-')
		}
	}
	out := b.String()
	out = strings.Trim(out, "-")
	if out == "" {
		out = "archive"
	}
	return out
}

func nextSuffixedPath(archiveDir, date, label string) string {
	base := date + "--" + safeSlug(label)
	for n := 2; n < 100; n++ {
		candidate := filepath.Join(archiveDir, fmt.Sprintf("%s-%d.md", base, n))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return filepath.Join(archiveDir, base+"-conflict.md")
}

// buildArchiveBody prepends the archive frontmatter + historical
// banner to the rendered SNAPSHOT.md bytes. The output is what gets
// written to disk.
//
// Invariants enforced:
//   - frontmatter carries historical: true and not_current: true
//   - banner line follows the frontmatter immediately, before any other body content
//   - projector_version reflects the package constant
func buildArchiveBody(rendered []byte, date string, trigger ArchiveTrigger, label, gitCommit string) (string, error) {
	if len(rendered) == 0 {
		return "", errors.New("snapshot: archive body requires rendered input")
	}
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "snapshot_date: %s\n", date)
	fmt.Fprintf(&b, "trigger: %s\n", trigger)
	if label != "" {
		fmt.Fprintf(&b, "label: %q\n", label)
	}
	if gitCommit != "" {
		fmt.Fprintf(&b, "git_commit: %s\n", gitCommit)
	}
	fmt.Fprintf(&b, "projector_version: %d\n", ProjectorVersion)
	b.WriteString("historical: true\n")
	b.WriteString("not_current: true\n")
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, HistoricalBanner+"\n\n", date)
	b.Write(rendered)
	return b.String(), nil
}

// List enumerates archives in the configured archive directory,
// newest-first by snapshot_date.
func List(heroDir string) ([]ArchiveRecord, error) {
	archiveDir := filepath.Join(heroDir, ArchiveDirName)
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []ArchiveRecord
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		rec, err := Read(filepath.Join(archiveDir, e.Name()))
		if err != nil {
			continue
		}
		out = append(out, *rec)
	}
	sort.Slice(out, func(i, j int) bool {
		// Sort by date desc, then filename desc as tiebreaker.
		if out[i].Date != out[j].Date {
			return out[i].Date > out[j].Date
		}
		return filepath.Base(out[i].Path) > filepath.Base(out[j].Path)
	})
	return out, nil
}

// Read parses one archive file into an ArchiveRecord. The body
// includes the historical banner; the frontmatter is parsed into the
// record's fields.
func Read(path string) (*ArchiveRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	src := string(data)
	if !strings.HasPrefix(src, "---\n") {
		return nil, fmt.Errorf("%s: missing frontmatter", path)
	}
	rest := strings.TrimPrefix(src, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return nil, fmt.Errorf("%s: malformed frontmatter (no closing ---)", path)
	}
	fmBlock := rest[:end]
	body := rest[end+len("\n---\n"):]
	body = strings.TrimLeft(body, "\n")

	rec := &ArchiveRecord{Path: path, Body: body, BodyOffset: len(src) - len(body)}
	for _, line := range strings.Split(fmBlock, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "snapshot_date:"):
			rec.Date = strings.TrimSpace(strings.TrimPrefix(line, "snapshot_date:"))
		case strings.HasPrefix(line, "trigger:"):
			rec.Trigger = ArchiveTrigger(strings.TrimSpace(strings.TrimPrefix(line, "trigger:")))
		case strings.HasPrefix(line, "label:"):
			v := strings.TrimSpace(strings.TrimPrefix(line, "label:"))
			rec.Label = strings.Trim(v, "\"'")
		case strings.HasPrefix(line, "git_commit:"):
			rec.GitCommit = strings.TrimSpace(strings.TrimPrefix(line, "git_commit:"))
		case strings.HasPrefix(line, "projector_version:"):
			v := strings.TrimSpace(strings.TrimPrefix(line, "projector_version:"))
			n, _ := strconv.Atoi(v)
			rec.ProjectorVersion = n
		case strings.HasPrefix(line, "historical:"):
			rec.Historical = strings.Contains(line, "true")
		case strings.HasPrefix(line, "not_current:"):
			rec.NotCurrent = strings.Contains(line, "true")
		}
	}
	if rec.Date == "" {
		// Fall back to filename.
		base := strings.TrimSuffix(filepath.Base(path), ".md")
		if len(base) >= 10 {
			rec.Date = base[:10]
		}
	}
	if !rec.Historical || !rec.NotCurrent {
		return nil, fmt.Errorf("%s: missing isolation flags (historical/not_current)", path)
	}
	if !strings.Contains(body, "Historical archive captured") {
		return nil, fmt.Errorf("%s: missing historical banner line in body", path)
	}
	return rec, nil
}

// FindArchive resolves an archive by its date string (YYYY-MM-DD)
// optionally suffixed with --<slug>. Returns nil if not found.
func FindArchive(heroDir, key string) (*ArchiveRecord, error) {
	archiveDir := filepath.Join(heroDir, ArchiveDirName)
	// Direct filename hit?
	direct := filepath.Join(archiveDir, key+".md")
	if _, err := os.Stat(direct); err == nil {
		return Read(direct)
	}
	// Walk and match by Date.
	archives, err := List(heroDir)
	if err != nil {
		return nil, err
	}
	for _, a := range archives {
		if a.Date == key {
			return &a, nil
		}
		// Match "YYYY-MM-DD--label"
		filename := strings.TrimSuffix(filepath.Base(a.Path), ".md")
		if filename == key {
			return &a, nil
		}
	}
	return nil, nil
}
