package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
	"github.com/hero-engine/hero/internal/projection"
)

func TestWriteFileIfChangedSkipsIdenticalContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "NEXT.md")
	content := []byte("same\n")

	changed, err := writeFileIfChanged(path, content, 0o644)
	if err != nil {
		t.Fatalf("writeFileIfChanged first write: %v", err)
	}
	if !changed {
		t.Fatal("first write should report changed")
	}
	before := mustStatModTime(t, path)
	time.Sleep(20 * time.Millisecond)

	changed, err = writeFileIfChanged(path, content, 0o644)
	if err != nil {
		t.Fatalf("writeFileIfChanged second write: %v", err)
	}
	if changed {
		t.Fatal("identical write should report unchanged")
	}
	after := mustStatModTime(t, path)
	if !after.Equal(before) {
		t.Fatalf("mtime advanced on no-op write: before=%s after=%s", before, after)
	}
}

func TestWriteProjectedFilePreservesUpdatedOnlyChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "NEXT.md")
	original := []byte(`---
updated: 2026-05-04T10:00:00Z
repo: hero
---

## Next

same
`)
	proposed := []byte(`---
updated: 2026-05-04T11:00:00Z
repo: hero
---

## Next

same
`)

	if changed, err := writeProjectedFileIfSemanticChanged(path, original, 0o644); err != nil {
		t.Fatalf("initial projected write: %v", err)
	} else if !changed {
		t.Fatal("initial projected write should report changed")
	}
	before := mustStatModTime(t, path)
	time.Sleep(20 * time.Millisecond)

	changed, err := writeProjectedFileIfSemanticChanged(path, proposed, 0o644)
	if err != nil {
		t.Fatalf("second projected write: %v", err)
	}
	if changed {
		t.Fatal("updated-only projected write should report unchanged")
	}
	after := mustStatModTime(t, path)
	if !after.Equal(before) {
		t.Fatalf("mtime advanced on updated-only projection: before=%s after=%s", before, after)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "2026-05-04T11:00:00Z") {
		t.Fatalf("updated timestamp should have been preserved, got:\n%s", data)
	}
	if !strings.Contains(string(data), "2026-05-04T10:00:00Z") {
		t.Fatalf("original updated timestamp missing, got:\n%s", data)
	}
}

func TestWriteProjectedFileUpdatesOnSemanticChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "NEXT.md")
	original := []byte(`---
updated: 2026-05-04T10:00:00Z
repo: hero
---

## Next

old
`)
	proposed := []byte(`---
updated: 2026-05-04T11:00:00Z
repo: hero
---

## Next

new
`)

	if _, err := writeProjectedFileIfSemanticChanged(path, original, 0o644); err != nil {
		t.Fatalf("initial projected write: %v", err)
	}
	changed, err := writeProjectedFileIfSemanticChanged(path, proposed, 0o644)
	if err != nil {
		t.Fatalf("semantic projected write: %v", err)
	}
	if !changed {
		t.Fatal("semantic projected write should report changed")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "2026-05-04T11:00:00Z") || !strings.Contains(string(data), "new") {
		t.Fatalf("semantic change was not written, got:\n%s", data)
	}
}

func TestWriteProjectedNextMDSkipsUpdatedOnlyChange(t *testing.T) {
	env := newTestEnv(t)
	nextPath := filepath.Join(env.heroDir, "NEXT.md")

	if err := writeProjectedNextMD(nextPath, env.dir, env.heroDir); err != nil {
		t.Fatalf("writeProjectedNextMD first write: %v", err)
	}
	before := mustStatModTime(t, nextPath)
	firstBody, err := os.ReadFile(nextPath)
	if err != nil {
		t.Fatalf("ReadFile first body: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	if err := writeProjectedNextMD(nextPath, env.dir, env.heroDir); err != nil {
		t.Fatalf("writeProjectedNextMD second write: %v", err)
	}
	after := mustStatModTime(t, nextPath)
	if !after.Equal(before) {
		t.Fatalf("projected NEXT.md mtime advanced on updated-only change: before=%s after=%s", before, after)
	}
	secondBody, err := os.ReadFile(nextPath)
	if err != nil {
		t.Fatalf("ReadFile second body: %v", err)
	}
	if string(secondBody) != string(firstBody) {
		t.Fatalf("projected NEXT.md changed on updated-only render:\nfirst:\n%s\nsecond:\n%s", firstBody, secondBody)
	}
}

func TestWriteUserHandoffFileSkipsUpdatedOnlyChange(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/tester"}
	userPath := filepath.Join(env.heroDir, nextDirName, "tester.md")

	if err := writeUserHandoffFile(env.dir, env.heroDir, cfg); err != nil {
		t.Fatalf("writeUserHandoffFile first write: %v", err)
	}
	before := mustStatModTime(t, userPath)
	firstBody, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("ReadFile first body: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	if err := writeUserHandoffFile(env.dir, env.heroDir, cfg); err != nil {
		t.Fatalf("writeUserHandoffFile second write: %v", err)
	}
	after := mustStatModTime(t, userPath)
	if !after.Equal(before) {
		t.Fatalf("user handoff mtime advanced on updated-only change: before=%s after=%s", before, after)
	}
	secondBody, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("ReadFile second body: %v", err)
	}
	if string(secondBody) != string(firstBody) {
		t.Fatalf("user handoff changed on updated-only render:\nfirst:\n%s\nsecond:\n%s", firstBody, secondBody)
	}
}

// TestWriteUserHandoffFileTeamModeProjectsPerUserFile is the canonical
// regression for the primary bug (Change 1): in team mode,
// writeUserHandoffFile used to early-return without writing anything,
// so the per-user .hero/next/<user>.md was never projected from the
// graph. After the fix it must project the file in BOTH modes. This
// test FAILS before Change 1 (the early return writes nothing) and
// passes after.
func TestWriteUserHandoffFileTeamModeProjectsPerUserFile(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Next = &config.NextConfig{Mode: "team"}
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/alice"}

	repoKey := gitutil.RepoKey(env.dir)
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := handoff.RecordAsk(store, repoKey, handoff.UserAsk{
		User: "alice", Text: "wire up the team-mode handoff",
	}); err != nil {
		t.Fatal(err)
	}
	if err := handoff.RecordSuggestion(store, repoKey, handoff.NextSuggestion{
		User: "alice", Text: "land the per-user projection next",
	}); err != nil {
		t.Fatal(err)
	}
	store.Close()

	if err := writeUserHandoffFile(env.dir, env.heroDir, cfg); err != nil {
		t.Fatalf("writeUserHandoffFile (team): %v", err)
	}

	userPath := filepath.Join(env.heroDir, nextDirName, "alice.md")
	body, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("team-mode per-user file not written: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, "wire up the team-mode handoff") {
		t.Errorf("per-user file missing recorded ask:\n%s", got)
	}
	if !strings.Contains(got, "land the per-user projection next") {
		t.Errorf("per-user file missing recorded suggestion:\n%s", got)
	}
	// Personal-briefing shape — not the project-shape NEXT.md render.
	if !strings.Contains(got, "## Last user ask") || !strings.Contains(got, "## Suggested next prompt") {
		t.Errorf("per-user file is not the personal-briefing render:\n%s", got)
	}
}

// Test_writeCheckpoint_TeamMode_WritesBothFiles guards Change 3: a full
// checkpoint in team mode must write BOTH the gitignored machine-state
// file (<user>.local.md) AND the durable per-user briefing (<user>.md),
// and the durable file must be the personal-briefing render
// (## Last user ask / ## Suggested next prompt), NOT the project-shape
// NEXT.md render (## Just finished). The project-shape projection must
// land on the shared .hero/NEXT.md, not clobber the per-user file.
func Test_writeCheckpoint_TeamMode_WritesBothFiles(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Next = &config.NextConfig{Mode: "team", Projected: true}
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/alice"}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	repoKey := gitutil.RepoKey(env.dir)
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := handoff.RecordAsk(store, repoKey, handoff.UserAsk{
		User: "alice", Text: "team-mode durable briefing should persist",
	}); err != nil {
		t.Fatal(err)
	}
	store.Close()

	if _, err := writeCheckpoint(); err != nil {
		t.Fatalf("writeCheckpoint (team): %v", err)
	}

	localPath := filepath.Join(env.heroDir, nextDirName, "alice"+localStateSuffix)
	if _, err := os.Stat(localPath); err != nil {
		t.Fatalf("machine-state file %s not written: %v", localPath, err)
	}

	userPath := filepath.Join(env.heroDir, nextDirName, "alice.md")
	body, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("durable per-user file not written: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, "## Last user ask") || !strings.Contains(got, "## Suggested next prompt") {
		t.Errorf("durable per-user file is not the personal-briefing render:\n%s", got)
	}
	if strings.Contains(got, "## Just finished") {
		t.Errorf("durable per-user file got project-shape NEXT.md content (double-write contamination):\n%s", got)
	}
	if !strings.Contains(got, "team-mode durable briefing should persist") {
		t.Errorf("durable per-user file missing recorded ask:\n%s", got)
	}
}

// Test_writeCheckpoint_TeamMode_NonFatalOnProjectionError pins the
// warn-and-continue contract for the per-user projection: if the
// user-handoff projection fails, writeCheckpoint must still return nil.
// We force a failure by making .hero/next/<user>.md a directory, so the
// underlying file write errors out — the checkpoint must swallow it.
func Test_writeCheckpoint_TeamMode_NonFatalOnProjectionError(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Next = &config.NextConfig{Mode: "team", Projected: true}
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/alice"}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	// Make the per-user path a directory so the file write fails.
	userPath := filepath.Join(env.heroDir, nextDirName, "alice.md")
	if err := os.MkdirAll(userPath, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := writeCheckpoint(); err != nil {
		t.Fatalf("writeCheckpoint must stay non-fatal on per-user projection failure, got: %v", err)
	}
}

// Test_writeCheckpoint_TeamMode_RoundTripIsIdempotent is the spec's
// Test Plan #5: with Phase 1 now projecting the per-user
// .hero/next/<user>.md in team mode, that file becomes a live ingest
// target on other machines. The project → ingest → re-project cycle
// must be idempotent — re-ingesting a freshly-projected file and
// re-projecting must reproduce the same semantic content (the ingest
// dedupes reflections on text and skips auto-derived suggestions, so
// nothing should duplicate or get corrupted).
//
// We compare via normalizeUpdatedFrontmatter — the same semantic
// equality the projector uses to suppress updated-only churn — so the
// unavoidable wall-clock `updated:` timestamp delta between two
// renders doesn't mask the real concern: duplicated reflections or
// auto-derived-suggestion corruption.
func Test_writeCheckpoint_TeamMode_RoundTripIsIdempotent(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Next = &config.NextConfig{Mode: "team"}
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/alice"}

	repoKey := gitutil.RepoKey(env.dir)
	domain := graph.DomainFor(cfg, graph.IntrinsicActive)

	// Seed the user-graph with all three handoff node types. The
	// suggestion is a real agent emission (no auto-derived footer) so
	// it must survive the round-trip; the reflection is the dedupe
	// target.
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := handoff.RecordAsk(store, repoKey, handoff.UserAsk{
		User: "alice", Domain: domain, Text: "where did we leave off on team-mode handoff?",
	}); err != nil {
		t.Fatal(err)
	}
	if err := handoff.RecordSuggestion(store, repoKey, handoff.NextSuggestion{
		User: "alice", Domain: domain, Text: "land the per-user projection idempotence test",
		Rationale: "closes the last open item on the spec",
	}); err != nil {
		t.Fatal(err)
	}
	if err := handoff.RecordReflection(store, repoKey, handoff.SessionReflection{
		User: "alice", Domain: domain, Text: "team-mode reuses the solo render — only the gate changed",
	}); err != nil {
		t.Fatal(err)
	}
	store.Close()

	userPath := filepath.Join(env.heroDir, nextDirName, "alice.md")

	// 1. First projection.
	if err := writeUserHandoffFile(env.dir, env.heroDir, cfg); err != nil {
		t.Fatalf("first projection: %v", err)
	}
	first, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("read first projection: %v", err)
	}

	// 2. Ingest it back via the real `hero next ingest` entry point.
	store, err = graph.Open(env.heroDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := handoff.IngestUserFile(store, repoKey, domain, userPath); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	store.Close()

	// 3. Re-project after the ingest round-trip.
	if err := writeUserHandoffFile(env.dir, env.heroDir, cfg); err != nil {
		t.Fatalf("second projection: %v", err)
	}
	second, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("read second projection: %v", err)
	}

	// 4. Assert byte-stable modulo the `updated:` timestamp. Any drift
	// here means the ingest duplicated a reflection or mangled the
	// suggestion — a real non-idempotence finding, not a test flake.
	firstNorm := normalizeUpdatedFrontmatter(string(first))
	secondNorm := normalizeUpdatedFrontmatter(string(second))
	if firstNorm != secondNorm {
		t.Fatalf("project→ingest→re-project not idempotent.\nfirst:\n%s\n\nsecond:\n%s", first, second)
	}

	// Belt-and-suspenders: the suggestion survived (not dropped as
	// auto-derived) and the reflection appears exactly once.
	if !strings.Contains(secondNorm, "land the per-user projection idempotence test") {
		t.Errorf("agent suggestion lost across round-trip:\n%s", second)
	}
	if n := strings.Count(secondNorm, "team-mode reuses the solo render"); n != 1 {
		t.Errorf("reflection appears %d times after round-trip, want exactly 1:\n%s", n, second)
	}
}

func TestNextCheckpointQuietIsSilent(t *testing.T) {
	_ = newTestEnv(t)

	out, err := runCmd("next", "checkpoint", "-q")
	if err != nil {
		t.Fatalf("next checkpoint -q returned error: %v", err)
	}
	if out != "" {
		t.Fatalf("quiet checkpoint should not print stdout, got %q", out)
	}
}

// Test_rebuildLocalState_DiscardsHandContent pins the primary fix:
// even when the existing .local.md content contains a stale narrative
// section outside the marker block, rebuildLocalState must return
// only the fresh machine block — total rewrite, no preservation.
func Test_rebuildLocalState_DiscardsHandContent(t *testing.T) {
	existing := `<!-- BEGIN HERO MACHINE STATE -->
## Machine state
old machine state
<!-- END HERO MACHINE STATE -->

## Just finished
something from another repo entirely

## Next
do another thing
`
	fresh := "<!-- BEGIN HERO MACHINE STATE -->\nfresh\n<!-- END HERO MACHINE STATE -->"
	out := rebuildLocalState(existing, fresh)

	if !strings.Contains(out, "fresh") {
		t.Errorf("output missing fresh machine block: %q", out)
	}
	if strings.Contains(out, "Just finished") {
		t.Errorf("output preserved hand-content section %q (full output: %q)", "Just finished", out)
	}
	if strings.Contains(out, "another repo") {
		t.Errorf("output preserved cross-repo narrative: %q", out)
	}
	if strings.Contains(out, "old machine state") {
		t.Errorf("output preserved old machine state: %q", out)
	}
}

func Test_writeCheckpoint_BacksUpPreExistingHandContent(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/tester"}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	localPath := filepath.Join(env.heroDir, nextDirName, "tester"+localStateSuffix)
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		t.Fatal(err)
	}
	prePopulated := `<!-- BEGIN HERO MACHINE STATE -->
old machine
<!-- END HERO MACHINE STATE -->

## Just finished
cross-repo narrative pollution

## Next
go elsewhere
`
	if err := os.WriteFile(localPath, []byte(prePopulated), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := writeCheckpoint(); err != nil {
		t.Fatalf("writeCheckpoint: %v", err)
	}

	// Rebuilt file: marker block only, no leaked narrative.
	got, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(got), "Just finished") || strings.Contains(string(got), "cross-repo narrative") {
		t.Errorf("rebuilt .local.md still contains hand-content:\n%s", got)
	}
	if !strings.Contains(string(got), "BEGIN HERO MACHINE STATE") {
		t.Errorf("rebuilt .local.md missing machine block:\n%s", got)
	}

	// Backup file present alongside.
	entries, _ := os.ReadDir(filepath.Dir(localPath))
	foundBackup := ""
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), filepath.Base(localPath)+".bak.") {
			foundBackup = e.Name()
			break
		}
	}
	if foundBackup == "" {
		t.Fatalf("no .bak.<ts> file written. entries=%v", entries)
	}
	backupBytes, _ := os.ReadFile(filepath.Join(filepath.Dir(localPath), foundBackup))
	if !strings.Contains(string(backupBytes), "cross-repo narrative") {
		t.Errorf("backup missing hand-content: %s", backupBytes)
	}
	if strings.Contains(string(backupBytes), "BEGIN HERO MACHINE STATE") {
		t.Errorf("backup should not contain machine block: %s", backupBytes)
	}
}

func Test_writeCheckpoint_NoBackupWhenAlreadyClean(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/tester"}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	// First run: no existing .local.md → no backup.
	if _, err := writeCheckpoint(); err != nil {
		t.Fatalf("writeCheckpoint #1: %v", err)
	}
	// Second run: existing .local.md is machine-only → no backup.
	if _, err := writeCheckpoint(); err != nil {
		t.Fatalf("writeCheckpoint #2: %v", err)
	}

	localPath := filepath.Join(env.heroDir, nextDirName, "tester"+localStateSuffix)
	entries, _ := os.ReadDir(filepath.Dir(localPath))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), filepath.Base(localPath)+".bak.") {
			t.Errorf("unexpected backup file written when content was already clean: %s", e.Name())
		}
	}
}

// Test_writeCheckpoint_BackupIdempotentOnRerun pins the
// idempotency contract: if a polluted .local.md is left in place
// (e.g. a script keeps re-writing it), the second checkpoint must
// not create a duplicate backup with identical content.
func Test_writeCheckpoint_BackupIdempotentOnRerun(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/tester"}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	localPath := filepath.Join(env.heroDir, nextDirName, "tester"+localStateSuffix)
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "<!-- BEGIN HERO MACHINE STATE -->\nx\n<!-- END HERO MACHINE STATE -->\n\n## Just finished\nstale\n"
	if err := os.WriteFile(localPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := writeCheckpoint(); err != nil {
		t.Fatalf("checkpoint #1: %v", err)
	}
	// Re-pollute with identical content and rerun.
	if err := os.WriteFile(localPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := writeCheckpoint(); err != nil {
		t.Fatalf("checkpoint #2: %v", err)
	}

	entries, _ := os.ReadDir(filepath.Dir(localPath))
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), filepath.Base(localPath)+".bak.") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly one backup, got %d", count)
	}
}

// Test_writeCheckpoint_CrossRepoAskDoesNotLeakIntoUserHandoff is the
// end-to-end pin: record an ask against repo A's graph, then run
// the user-handoff projection scoped to repo B, and verify the
// rendered <user>.md does not contain the repo-A ask.
//
// The checkpoint command itself reads RepoKey from gitutil and the
// test environment isn't a git repo, so we drive the projection
// directly with explicit repo keys — same code path that
// writeUserHandoffFile calls.
func Test_writeCheckpoint_CrossRepoAskDoesNotLeakIntoUserHandoff(t *testing.T) {
	env := newTestEnv(t)
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := handoff.RecordAsk(store, "repo-a", handoff.UserAsk{
		User: "tester", Text: "A-context-secret",
	}); err != nil {
		t.Fatal(err)
	}

	body, err := projection.UserHandoffMD(store, projection.UserHandoffOptions{
		User:    "tester",
		RepoKey: "repo-b",
	})
	if err != nil {
		t.Fatalf("UserHandoffMD repo-b: %v", err)
	}
	if strings.Contains(body, "A-context-secret") {
		t.Errorf("repo-b handoff leaked repo-a ask:\n%s", body)
	}
}

// Test_CrossMachineRoundTrip_FullLoop pins next-as-projection AC-6
// and AC-11. The full sequence:
//
//  1. Machine A's graph records a NextSuggestion via the field-grab
//     CLI's underlying handoff.RecordSuggestion.
//  2. projection.UserHandoffMD renders A's user-graph into a
//     .hero/next/<user>.md markdown file (the cross-machine medium).
//  3. The file travels via git (here, just written to disk).
//  4. Machine B (a fresh, empty graph) calls handoff.IngestUserFile
//     on the same file — this is what the SessionStart hook fires
//     on a new session opening after `git pull`.
//  5. Machine B's queries return A's recorded suggestion verbatim.
//
// Two ephemeral graph DBs in one test process simulate the
// two-machine boundary cleanly without spinning up actual git or
// separate filesystems.
func Test_CrossMachineRoundTrip_FullLoop(t *testing.T) {
	user := "alice"
	repoKey := "repo-x"
	suggestion := "let's tackle the phase-6 git hooks tomorrow"
	rationale := "phase 5 ingest lands today, hooks are the next contiguous chunk"
	ask := "where did we leave off on next-as-projection?"
	reflection := "the merge-driver is registered by hero install, not by the repo"

	// --- Machine A: write to local graph + project to handoff file ---

	storeA, err := graph.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graph.Open A: %v", err)
	}
	defer storeA.Close()

	if err := handoff.RecordSuggestion(storeA, repoKey, handoff.NextSuggestion{
		User:      user,
		Text:      suggestion,
		Rationale: rationale,
	}); err != nil {
		t.Fatalf("RecordSuggestion A: %v", err)
	}
	if err := handoff.RecordAsk(storeA, repoKey, handoff.UserAsk{
		User: user,
		Text: ask,
	}); err != nil {
		t.Fatalf("RecordAsk A: %v", err)
	}
	if err := handoff.RecordReflection(storeA, repoKey, handoff.SessionReflection{
		User: user,
		Text: reflection,
	}); err != nil {
		t.Fatalf("RecordReflection A: %v", err)
	}

	body, err := projection.UserHandoffMD(storeA, projection.UserHandoffOptions{
		User:    user,
		RepoKey: repoKey,
	})
	if err != nil {
		t.Fatalf("UserHandoffMD A: %v", err)
	}

	// --- The "git boundary": file lands on disk, no other state crosses. ---

	handoffPath := filepath.Join(t.TempDir(), "alice.md")
	if err := os.WriteFile(handoffPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write handoff file: %v", err)
	}

	// --- Machine B: clean graph, run ingest, query. ---

	storeB, err := graph.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graph.Open B: %v", err)
	}
	defer storeB.Close()

	// Pre-condition: B's graph has no record of any of A's fields.
	if got, _ := handoff.LatestSuggestion(storeB, user, repoKey, "engineering"); got != nil {
		t.Fatalf("machine B has stale suggestion before ingest: %+v", got)
	}

	if err := handoff.IngestUserFile(storeB, repoKey, "engineering", handoffPath); err != nil {
		t.Fatalf("IngestUserFile B: %v", err)
	}

	// Post-condition: B sees A's text verbatim across all three fields.
	gotSug, err := handoff.LatestSuggestion(storeB, user, repoKey, "engineering")
	if err != nil {
		t.Fatalf("LatestSuggestion B: %v", err)
	}
	if gotSug == nil || gotSug.Text != suggestion {
		t.Errorf("Suggestion did not round-trip: got=%+v want=%q", gotSug, suggestion)
	}
	gotAsk, _ := handoff.LatestAsk(storeB, user, repoKey, "engineering")
	if gotAsk == nil || gotAsk.Text != ask {
		t.Errorf("Ask did not round-trip: got=%+v want=%q", gotAsk, ask)
	}
	gotRefs, _ := handoff.RecentReflections(storeB, user, repoKey, "engineering", 10)
	if len(gotRefs) == 0 || gotRefs[0].Text != reflection {
		t.Errorf("Reflection did not round-trip: got=%+v want=%q", gotRefs, reflection)
	}

	// --- Idempotency: a second SessionStart ingest must not duplicate. ---

	if err := handoff.IngestUserFile(storeB, repoKey, "engineering", handoffPath); err != nil {
		t.Fatalf("second IngestUserFile B: %v", err)
	}
	refsAfter, _ := handoff.RecentReflections(storeB, user, repoKey, "engineering", 10)
	if len(refsAfter) != len(gotRefs) {
		t.Errorf("re-ingest duplicated reflections: %d → %d", len(gotRefs), len(refsAfter))
	}
}

// Test_writeCheckpoint_PreFlightGate_RefusesLegacyMarkers pins AC-14
// of next-as-projection: when the repo hasn't been migrated and the
// existing NEXT.md still carries pre-projection markers, the
// checkpoint write must refuse rather than silently rewrite over
// hand-authored sections that migration would otherwise ingest as
// graph nodes.
func Test_writeCheckpoint_PreFlightGate_RefusesLegacyMarkers(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/tester"}
	// Explicitly do NOT set next.projected — this is the unmigrated state.
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	nextPath := filepath.Join(env.heroDir, "NEXT.md")
	legacy := `# Where we are

<!-- BEGIN HERO MACHINE STATE -->
Branch: main
<!-- END HERO MACHINE STATE -->

Hand-written stuff that mattered.
`
	if err := os.WriteFile(nextPath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := writeCheckpoint(); err == nil {
		t.Fatal("writeCheckpoint should refuse with unmigrated legacy markers present")
	} else if !strings.Contains(err.Error(), "hero next migrate-to-projection") {
		t.Errorf("error should direct user to migrate; got: %v", err)
	}

	// File must be untouched.
	got, _ := os.ReadFile(nextPath)
	if string(got) != legacy {
		t.Errorf("NEXT.md was modified despite gate; got:\n%s", got)
	}
}

// Test_writeCheckpoint_PreFlightGate_RefusesLegacyHeaders covers the
// other unmigrated signal: section headers from the hand-authored
// era. Same rule applies when next.projected = false.
func Test_writeCheckpoint_PreFlightGate_RefusesLegacyHeaders(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/tester"}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	nextPath := filepath.Join(env.heroDir, "NEXT.md")
	legacy := `# Where we are

## Just finished

Big refactor of the projection layer.

## Next

Wire the new gate.
`
	if err := os.WriteFile(nextPath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := writeCheckpoint(); err == nil {
		t.Fatal("writeCheckpoint should refuse with unmigrated legacy headers present")
	} else if !strings.Contains(err.Error(), "Just finished") {
		t.Errorf("error should name the offending header; got: %v", err)
	}
}

// Test_writeCheckpoint_PreFlightGate_AllowsWhenMigrated confirms the
// gate stays out of the way once next.projected = true. The
// projection path owns the file from that point and may legitimately
// rewrite content the gate's heuristics would otherwise flag.
func Test_writeCheckpoint_PreFlightGate_AllowsWhenMigrated(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/tester"}
	cfg.Next = &config.NextConfig{Projected: true}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	nextPath := filepath.Join(env.heroDir, "NEXT.md")
	// Even content that *looks* legacy is fine once we're migrated —
	// the projection will replace it.
	if err := os.WriteFile(nextPath, []byte("## Just finished\nstale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := writeCheckpoint(); err != nil {
		t.Fatalf("writeCheckpoint should succeed when migrated: %v", err)
	}
}

// Test_writeCheckpoint_PreFlightGate_AllowsCleanUnmigrated covers
// the fresh-repo case: no NEXT.md or an empty file is not "legacy
// content," so the gate must let the legacy write path run and
// produce a placeholder NEXT.md.
func Test_writeCheckpoint_PreFlightGate_AllowsCleanUnmigrated(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/tester"}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	// No NEXT.md on disk.
	if _, err := writeCheckpoint(); err != nil {
		t.Fatalf("writeCheckpoint should succeed when no legacy content: %v", err)
	}
}

func mustStatModTime(t *testing.T, path string) time.Time {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	return info.ModTime()
}
