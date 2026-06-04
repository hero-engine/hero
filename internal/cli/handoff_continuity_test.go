package cli

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/digest"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
	"github.com/hero-engine/hero/internal/projection"
	"github.com/spf13/cobra"
)

// handoff_continuity_test.go is the executable guardrail for Hero's
// handoff "magic": finish a turn → context captured; start a fresh
// session on another machine → context already loaded, zero manual
// prep. It protects the e2e-handoff-continuity spec's invariant.
//
// The load-bearing, previously-UNGUARDED case is the cross-machine
// leap: machine B has an EMPTY graph and ONLY the committed
// `.hero/next/<user>.md` (graph.db is gitignored and does NOT
// travel), yet must reconstruct machine A's context. The existing
// Test_writeCheckpoint_TeamMode_RoundTripIsIdempotent
// (checkpoint_test.go) proves project→ingest→re-project is byte-
// stable on the SAME machine with the SAME graph; it never severs
// the graph. These tests do.
//
// --- A note on the real start-of-turn load surface ---------------
//
// The handoff content a fresh session consumes surfaces through THREE
// real paths, and the guardrail now asserts against all three:
//
//   1. The graph-query surface the `hero next ask/suggest/reflection`
//      commands read: handoff.LatestAsk / projection.PickUserSuggestion
//      / handoff.RecentReflections. These ARE the functions the CLI
//      calls (see internal/cli/next_handoff.go) — not a reimplementation.
//   2. The re-projected `.hero/next/<user>.md`, rendered from the graph
//      by writeUserHandoffFile → projection.UserHandoffMD. This is the
//      personal briefing a fresh session opens.
//   3. The `hero resume` brief itself. As of
//      resume-brief-surfaces-handoff, `digest.Generate` carries a
//      "Where you left off" section (digest.handoffSection) that reads
//      the UserAsk / NextSuggestion / SessionReflection singletons keyed
//      by (user, repo, domain). The brief previously could NOT carry
//      this content — that gap is closed, so the guardrail now asserts
//      B's brief CONTAINS A's ask and suggestion, not merely that
//      digest.Generate runs without error.
//
// Asserting against all three genuine consumption surfaces is what makes
// the guardrail honest: it tracks the paths a fresh session takes, and
// it bites when travel breaks (see
// Test_HandoffContinuity_CrossMachine_GuardrailBites).

// seededHandoff is the distinct, greppable text a "machine A" turn
// persists. Each string is unique so an assertion can prove the exact
// value reconstructed on B, not a coincidental substring.
type seededHandoff struct {
	ask        string
	suggestion string
	rationale  string
	reflection string
}

func defaultSeed() seededHandoff {
	return seededHandoff{
		ask:        "MACHINE_A_ASK where did we leave off on cross-machine handoff?",
		suggestion: "MACHINE_A_SUGGESTION land the cross-machine continuity guardrail",
		rationale:  "closes the core AC of e2e-handoff-continuity",
		reflection: "MACHINE_A_REFLECTION graph.db is gitignored so the file is the only medium",
	}
}

// seedMachineA opens the env's graph and records a UserAsk, a REAL
// agent NextSuggestion (carries a Rationale, so the ingest does not
// treat it as auto-derived and drop it), and a SessionReflection,
// then projects `.hero/next/<user>.md` and returns the committed
// bytes — exactly what would be `git add`-ed and travel to B.
func seedMachineA(t *testing.T, env *testEnv, cfg config.Config, s seededHandoff) (committed []byte, userPath string) {
	t.Helper()
	repoKey := gitutil.RepoKey(env.dir)
	domain := graph.DomainFor(cfg, graph.IntrinsicActive)

	// SEAM(next-auto-emit-user-ask): the auto-emit feature has landed.
	// Test_HandoffContinuity_CrossMachine_AutoEmit drives a real Stop
	// checkpoint with a transcript payload (instead of this manual
	// graph seed) and runs the SAME cross-machine reconstruction below,
	// proving auto-capture → travel → reconstruct feeds the magic. This
	// helper keeps the manual seed because it also exercises the
	// NextSuggestion + SessionReflection surfaces, which auto-emit does
	// not (and must not) feed.
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("open A graph: %v", err)
	}
	if err := handoff.RecordAsk(store, repoKey, handoff.UserAsk{
		User: "alice", Domain: domain, Text: s.ask,
	}); err != nil {
		t.Fatalf("record ask: %v", err)
	}
	if err := handoff.RecordSuggestion(store, repoKey, handoff.NextSuggestion{
		User: "alice", Domain: domain, Text: s.suggestion, Rationale: s.rationale,
	}); err != nil {
		t.Fatalf("record suggestion: %v", err)
	}
	if err := handoff.RecordReflection(store, repoKey, handoff.SessionReflection{
		User: "alice", Domain: domain, Text: s.reflection,
	}); err != nil {
		t.Fatalf("record reflection: %v", err)
	}
	store.Close()

	if err := writeUserHandoffFile(env.dir, env.heroDir, cfg); err != nil {
		t.Fatalf("project A handoff file: %v", err)
	}
	userPath = filepath.Join(env.heroDir, nextDirName, "alice.md")
	committed, err = os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("read A committed handoff file: %v", err)
	}
	return committed, userPath
}

// teamCfg builds the team-mode config the per-user projection needs:
// DefaultAgent → user slug "alice", and team mode so the per-user file
// is the live handoff target.
func teamCfg() config.Config {
	cfg := config.DefaultConfig()
	cfg.Next = &config.NextConfig{Mode: "team"}
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/alice"}
	return cfg
}

// assertHandoffReconstructed asserts that the (already-ingested) graph
// behind heroDir surfaces A's ask, suggestion, and reflection through
// the REAL handoff query surfaces the `hero next` commands use, and
// that the personal briefing re-projected from that graph contains all
// three. projectRoot is the repo root used for the repoKey partition.
func assertHandoffReconstructed(t *testing.T, projectRoot, heroDir string, cfg config.Config, s seededHandoff, where string) {
	t.Helper()
	repoKey := gitutil.RepoKey(projectRoot)
	domain := graph.DomainFor(cfg, graph.IntrinsicActive)

	store, err := graph.Open(heroDir)
	if err != nil {
		t.Fatalf("[%s] open graph: %v", where, err)
	}
	defer store.Close()

	// 1. The exact functions `hero next ask` reads.
	ask, err := handoff.LatestAsk(store, "alice", repoKey, domain)
	if err != nil {
		t.Fatalf("[%s] LatestAsk: %v", where, err)
	}
	if ask == nil || !strings.Contains(ask.Text, s.ask) {
		t.Errorf("[%s] ask not reconstructed: got %+v, want contains %q", where, ask, s.ask)
	}

	// 2. The exact function `hero next suggest` reads. A real agent
	// suggestion (with rationale) must survive as SuggestionFromAgent —
	// not be replaced by an auto-derived fallback.
	text, _, source := projection.PickUserSuggestion(store, "alice", repoKey, domain)
	if !strings.Contains(text, s.suggestion) {
		t.Errorf("[%s] suggestion not reconstructed: got %q (source=%s), want contains %q", where, text, source, s.suggestion)
	}
	if source != projection.SuggestionFromAgent {
		t.Errorf("[%s] suggestion source = %q, want %q (agent emission was misclassified as auto-derived)", where, source, projection.SuggestionFromAgent)
	}

	// 3. The exact function `hero next reflection` reads.
	refs, err := handoff.RecentReflections(store, "alice", repoKey, domain, 5)
	if err != nil {
		t.Fatalf("[%s] RecentReflections: %v", where, err)
	}
	foundReflection := false
	for _, r := range refs {
		if strings.Contains(r.Text, s.reflection) {
			foundReflection = true
		}
	}
	if !foundReflection {
		t.Errorf("[%s] reflection not reconstructed: got %+v, want one containing %q", where, refs, s.reflection)
	}

	// 4. The personal briefing a fresh session opens: re-project
	// `.hero/next/<user>.md` from this graph and assert all three
	// strings made it into the rendered handoff document.
	briefing, err := projection.UserHandoffMD(store, projection.UserHandoffOptions{
		User:    "alice",
		RepoKey: repoKey,
	})
	if err != nil {
		t.Fatalf("[%s] re-project briefing: %v", where, err)
	}
	for _, want := range []string{s.ask, s.suggestion, s.reflection} {
		if !strings.Contains(briefing, want) {
			t.Errorf("[%s] re-projected briefing missing %q:\n%s", where, want, briefing)
		}
	}
}

// Test_HandoffContinuity_CrossMachine is AC-1 — the core guardrail.
// Persist handoff state on machine A, commit it (capture the file
// bytes), construct a SEPARATE machine B with its own EMPTY graph and
// ONLY the committed file, run the real ingest, then prove B
// reconstructs A's ask / suggestion / reflection. B never shares A's
// graph.
func Test_HandoffContinuity_CrossMachine(t *testing.T) {
	s := defaultSeed()

	// --- Machine A: seed, project, capture the committed bytes. ------
	cfgA := teamCfg()
	envA := newTestEnv(t)
	if err := cfgA.Save(envA.dir); err != nil {
		t.Fatalf("save A cfg: %v", err)
	}
	committed, _ := seedMachineA(t, envA, cfgA, s)

	// --- Machine B: a brand-new workspace, its own empty graph.db,
	// nothing seeded. Drop in ONLY the committed file — NOT A's graph.
	cfgB := teamCfg()
	envB := newTestEnv(t)
	if err := cfgB.Save(envB.dir); err != nil {
		t.Fatalf("save B cfg: %v", err)
	}
	bUserPath := filepath.Join(envB.heroDir, nextDirName, "alice.md")
	if err := os.MkdirAll(filepath.Dir(bUserPath), 0o755); err != nil {
		t.Fatalf("mkdir B next dir: %v", err)
	}
	if err := os.WriteFile(bUserPath, committed, 0o644); err != nil {
		t.Fatalf("drop committed file on B: %v", err)
	}

	// Sanity: B's graph really is empty before ingest — the magic must
	// come from the file, nothing else.
	repoKeyB := gitutil.RepoKey(envB.dir)
	domainB := graph.DomainFor(cfgB, graph.IntrinsicActive)
	storeB, err := graph.Open(envB.heroDir)
	if err != nil {
		t.Fatalf("open B graph: %v", err)
	}
	if ask, _ := handoff.LatestAsk(storeB, "alice", repoKeyB, domainB); ask != nil && ask.Text != "" {
		storeB.Close()
		t.Fatalf("B graph not empty before ingest: found ask %q — test setup is wrong", ask.Text)
	}

	// --- Rehydrate on B: the real `hero next ingest` entry point. ----
	if err := handoff.IngestUserFile(storeB, repoKeyB, domainB, bUserPath, "", true); err != nil {
		storeB.Close()
		t.Fatalf("ingest on B: %v", err)
	}
	storeB.Close()

	// --- Prove the real `hero resume` load path runs end-to-end on the
	// rehydrated B graph AND now carries A's handoff content. The brief
	// surfaces the per-user handoff singletons through digest's
	// handoffSection (resume-brief-surfaces-handoff), keyed by the same
	// (user, repo, domain) triple — so B's fresh-session brief must
	// contain A's ask and suggestion, not merely generate without error.
	storeB, err = graph.Open(envB.heroDir)
	if err != nil {
		t.Fatalf("reopen B graph: %v", err)
	}
	briefB, err := digest.Generate(storeB, digest.Options{
		RepoKey:     repoKeyB,
		Branch:      gitutil.CurrentBranch(envB.dir),
		AuthorEmail: "alice@example.com",
		User:        "alice",
		Domain:      domainB,
	})
	if err != nil {
		storeB.Close()
		t.Fatalf("hero resume load path (digest.Generate) failed on B: %v", err)
	}
	storeB.Close()
	mdB := briefB.Markdown()
	for _, want := range []string{s.ask, s.suggestion} {
		if !strings.Contains(mdB, want) {
			t.Errorf("B's resume brief did not carry A's handoff content %q:\n%s", want, mdB)
		}
	}

	// --- The assertion that bites: B reconstructed A's context from the
	// committed file alone.
	assertHandoffReconstructed(t, envB.dir, envB.heroDir, cfgB, s, "machine-B")
}

// Test_HandoffContinuity_SameMachine is AC-2 — after a projection on A,
// the start-of-turn load on A's own graph surfaces the last ask and
// suggested-next, and the resume load path runs clean. This is the
// same-machine fresh-session half of the invariant.
func Test_HandoffContinuity_SameMachine(t *testing.T) {
	s := defaultSeed()
	cfg := teamCfg()
	env := newTestEnv(t)
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	seedMachineA(t, env, cfg, s)

	// The real resume load path must run clean on the populated graph.
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("open graph: %v", err)
	}
	if _, err := digest.Generate(store, digest.Options{
		RepoKey:     gitutil.RepoKey(env.dir),
		AuthorEmail: "alice@example.com",
	}); err != nil {
		store.Close()
		t.Fatalf("hero resume load path failed on A: %v", err)
	}
	store.Close()

	assertHandoffReconstructed(t, env.dir, env.heroDir, cfg, s, "machine-A-same")
}

// Test_HandoffContinuity_TravelEligibility is AC-3 — the per-user
// `.hero/next/<user>.md` is travel-eligible (NOT matched by the repo's
// .gitignore), while `.hero/next/<user>.local.md` and `graph.db` ARE
// gitignored. Asserted via the real `git check-ignore` against the
// repo's actual root .gitignore — the same semantics the staging hook
// respects — so a future change that drops the file from tracking or
// moves graph.db out of ignore would turn this red.
func Test_HandoffContinuity_TravelEligibility(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available; AC-3 needs git check-ignore for real .gitignore semantics")
	}
	repoRoot := repoRootForTest(t)

	travels := ".hero/next/alice.md"
	if gitIgnored(t, repoRoot, travels) {
		t.Errorf("AC-3 violated: %q IS gitignored — the federation file would not travel, breaking cross-machine handoff", travels)
	}

	for _, p := range []string{
		".hero/next/alice.local.md",
		".hero/graph.db",
	} {
		if !gitIgnored(t, repoRoot, p) {
			t.Errorf("AC-3 violated: %q is NOT gitignored — it must never travel", p)
		}
	}
}

// Test_HandoffContinuity_CrossMachine_Idempotent is AC-4 — the cross-
// machine reconstruction is idempotent on B: ingest→project→ingest→
// project does not duplicate the reflection or drop the agent
// suggestion. Mirrors the existing same-machine idempotence test's
// normalizeUpdatedFrontmatter byte-stability + count checks, but on a
// B graph that was born empty and rehydrated from the traveled file.
func Test_HandoffContinuity_CrossMachine_Idempotent(t *testing.T) {
	s := defaultSeed()

	// Machine A → committed bytes.
	cfgA := teamCfg()
	envA := newTestEnv(t)
	if err := cfgA.Save(envA.dir); err != nil {
		t.Fatalf("save A cfg: %v", err)
	}
	committed, _ := seedMachineA(t, envA, cfgA, s)

	// Machine B: empty graph, only the committed file.
	cfgB := teamCfg()
	envB := newTestEnv(t)
	if err := cfgB.Save(envB.dir); err != nil {
		t.Fatalf("save B cfg: %v", err)
	}
	bUserPath := filepath.Join(envB.heroDir, nextDirName, "alice.md")
	if err := os.MkdirAll(filepath.Dir(bUserPath), 0o755); err != nil {
		t.Fatalf("mkdir B next dir: %v", err)
	}
	if err := os.WriteFile(bUserPath, committed, 0o644); err != nil {
		t.Fatalf("drop committed file on B: %v", err)
	}

	repoKeyB := gitutil.RepoKey(envB.dir)
	domainB := graph.DomainFor(cfgB, graph.IntrinsicActive)

	ingest := func() {
		store, err := graph.Open(envB.heroDir)
		if err != nil {
			t.Fatalf("open B graph: %v", err)
		}
		if err := handoff.IngestUserFile(store, repoKeyB, domainB, bUserPath, "", true); err != nil {
			store.Close()
			t.Fatalf("ingest on B: %v", err)
		}
		store.Close()
	}
	project := func() []byte {
		if err := writeUserHandoffFile(envB.dir, envB.heroDir, cfgB); err != nil {
			t.Fatalf("project on B: %v", err)
		}
		got, err := os.ReadFile(bUserPath)
		if err != nil {
			t.Fatalf("read B projection: %v", err)
		}
		return got
	}

	// ingest → project → ingest → project.
	ingest()
	first := project()
	ingest()
	second := project()

	firstNorm := normalizeUpdatedFrontmatter(string(first))
	secondNorm := normalizeUpdatedFrontmatter(string(second))
	if firstNorm != secondNorm {
		t.Fatalf("cross-machine round-trip on B not idempotent.\nfirst:\n%s\n\nsecond:\n%s", first, second)
	}
	if !strings.Contains(secondNorm, s.suggestion) {
		t.Errorf("agent suggestion lost across B round-trip:\n%s", second)
	}
	if n := strings.Count(secondNorm, s.reflection); n != 1 {
		t.Errorf("reflection appears %d times after B round-trip, want exactly 1:\n%s", n, second)
	}
}

// Test_HandoffContinuity_CrossMachine_GuardrailBites is Test Plan #2 —
// the mutation check, kept as a permanent regression assertion. A
// guardrail that cannot fail is theater. This reproduces the cross-
// machine setup but SKIPS the ingest on B (the exact failure a broken
// travel/load loop would produce: the file arrives but is never
// rehydrated into the graph). With no ingest, B's handoff query
// surfaces must NOT reconstruct A's context. If a future change made
// reconstruction "succeed" without ingest, that would mean the
// assertion in AC-1 no longer depends on the file→graph path — i.e.
// the guardrail stopped biting — and THIS test goes red to flag it.
func Test_HandoffContinuity_CrossMachine_GuardrailBites(t *testing.T) {
	s := defaultSeed()

	cfgA := teamCfg()
	envA := newTestEnv(t)
	if err := cfgA.Save(envA.dir); err != nil {
		t.Fatalf("save A cfg: %v", err)
	}
	committed, _ := seedMachineA(t, envA, cfgA, s)

	cfgB := teamCfg()
	envB := newTestEnv(t)
	if err := cfgB.Save(envB.dir); err != nil {
		t.Fatalf("save B cfg: %v", err)
	}
	bUserPath := filepath.Join(envB.heroDir, nextDirName, "alice.md")
	if err := os.MkdirAll(filepath.Dir(bUserPath), 0o755); err != nil {
		t.Fatalf("mkdir B next dir: %v", err)
	}
	if err := os.WriteFile(bUserPath, committed, 0o644); err != nil {
		t.Fatalf("drop committed file on B: %v", err)
	}

	// Deliberately DO NOT ingest. The file is present but the graph is
	// empty — the broken-magic state.
	repoKeyB := gitutil.RepoKey(envB.dir)
	domainB := graph.DomainFor(cfgB, graph.IntrinsicActive)
	store, err := graph.Open(envB.heroDir)
	if err != nil {
		t.Fatalf("open B graph: %v", err)
	}
	defer store.Close()

	ask, _ := handoff.LatestAsk(store, "alice", repoKeyB, domainB)
	if ask != nil && strings.Contains(ask.Text, s.ask) {
		t.Fatalf("guardrail does not bite: B reconstructed the ask WITHOUT ingest — "+
			"the AC-1 assertion no longer proves the file→graph→briefing path. got %q", ask.Text)
	}
	text, _, _ := projection.PickUserSuggestion(store, "alice", repoKeyB, domainB)
	if strings.Contains(text, s.suggestion) {
		t.Fatalf("guardrail does not bite: B reconstructed the suggestion WITHOUT ingest. got %q", text)
	}
}

// Test_HandoffContinuity_CrossMachine_AutoEmit closes the SEAM in
// seedMachineA: it proves the auto-emit path (not a manual RecordAsk)
// feeds the cross-machine magic. On machine A it drives a REAL Stop
// checkpoint with a transcript payload whose LAST user message is a
// known string, capturing the committed `.hero/next/<user>.md`. Then it
// runs the SAME cross-machine reconstruction the manual-seed guardrail
// uses: machine B is born empty, ONLY the committed file travels, the
// real ingest runs, and B must surface the auto-emitted ask through the
// real `hero next ask` query surface AND the re-projected briefing.
//
// This upgrades the guardrail from "manual-seed → travel → reconstruct"
// to "auto-capture → travel → reconstruct" — the actual end-to-end loop
// a developer relies on when they never type `hero next ask`.
func Test_HandoffContinuity_CrossMachine_AutoEmit(t *testing.T) {
	const autoAsk = "MACHINE_A_AUTO_ASK what was the last thing the user actually said?"

	// --- Machine A: drive a real checkpoint with a transcript payload.
	cfgA := teamCfg()
	envA := newTestEnv(t)
	if err := cfgA.Save(envA.dir); err != nil {
		t.Fatalf("save A cfg: %v", err)
	}

	tp := writeTranscript(t, envA.dir,
		`{"type":"user","message":{"role":"user","content":"earlier ask that must NOT win"}}`+"\n"+
			`{"type":"assistant","message":{"role":"assistant","content":"working on it"}}`+"\n"+
			`{"type":"user","message":{"role":"user","content":"`+autoAsk+`"}}`+"\n")
	payload := `{"session_id":"sess-A-auto","transcript_path":"` + tp + `"}`

	checkpointQuiet = true
	cmdA := &cobra.Command{RunE: runNextCheckpoint}
	cmdA.SetIn(strings.NewReader(payload))
	cmdA.SetOut(io.Discard)
	cmdA.SetErr(io.Discard)
	if err := cmdA.Execute(); err != nil {
		t.Fatalf("machine-A checkpoint with transcript payload: %v", err)
	}

	// The committed file is exactly what `git add` would stage and
	// travel to B. Auto-emit ran BEFORE the projection, so the file
	// already carries the auto-emitted ask.
	aUserPath := filepath.Join(envA.heroDir, nextDirName, "alice.md")
	committed, err := os.ReadFile(aUserPath)
	if err != nil {
		t.Fatalf("read A committed handoff file: %v", err)
	}
	if !strings.Contains(string(committed), autoAsk) {
		t.Fatalf("auto-emit did not feed A's committed file (same-turn render failed):\n%s", committed)
	}
	if strings.Contains(string(committed), "earlier ask that must NOT win") {
		t.Fatalf("A's file carries the FIRST user message, not the LAST:\n%s", committed)
	}

	// --- Machine B: brand-new workspace, own empty graph, only the file.
	cfgB := teamCfg()
	envB := newTestEnv(t)
	if err := cfgB.Save(envB.dir); err != nil {
		t.Fatalf("save B cfg: %v", err)
	}
	bUserPath := filepath.Join(envB.heroDir, nextDirName, "alice.md")
	if err := os.MkdirAll(filepath.Dir(bUserPath), 0o755); err != nil {
		t.Fatalf("mkdir B next dir: %v", err)
	}
	if err := os.WriteFile(bUserPath, committed, 0o644); err != nil {
		t.Fatalf("drop committed file on B: %v", err)
	}

	repoKeyB := gitutil.RepoKey(envB.dir)
	domainB := graph.DomainFor(cfgB, graph.IntrinsicActive)

	// Sanity: B is genuinely empty before ingest — the magic must come
	// from the traveled file alone.
	storeB, err := graph.Open(envB.heroDir)
	if err != nil {
		t.Fatalf("open B graph: %v", err)
	}
	if ask, _ := handoff.LatestAsk(storeB, "alice", repoKeyB, domainB); ask != nil && ask.Text != "" {
		storeB.Close()
		t.Fatalf("B graph not empty before ingest: found %q", ask.Text)
	}

	// --- Rehydrate on B via the real ingest entry point.
	if err := handoff.IngestUserFile(storeB, repoKeyB, domainB, bUserPath, "", true); err != nil {
		storeB.Close()
		t.Fatalf("ingest on B: %v", err)
	}
	storeB.Close()

	// --- The assertion that bites: B reconstructs the AUTO-EMITTED ask
	// from the committed file alone, through the real query surface.
	storeB, err = graph.Open(envB.heroDir)
	if err != nil {
		t.Fatalf("reopen B graph: %v", err)
	}
	ask, err := handoff.LatestAsk(storeB, "alice", repoKeyB, domainB)
	if err != nil {
		storeB.Close()
		t.Fatalf("LatestAsk on B: %v", err)
	}
	if ask == nil || !strings.Contains(ask.Text, autoAsk) {
		storeB.Close()
		t.Fatalf("B did not reconstruct the auto-emitted ask: got %+v, want contains %q", ask, autoAsk)
	}

	// And the personal briefing a fresh session opens carries it too.
	briefing, err := projection.UserHandoffMD(storeB, projection.UserHandoffOptions{
		User:    "alice",
		RepoKey: repoKeyB,
	})
	if err != nil {
		storeB.Close()
		t.Fatalf("re-project B briefing: %v", err)
	}
	if !strings.Contains(briefing, autoAsk) {
		storeB.Close()
		t.Errorf("B re-projected briefing missing auto-emitted ask:\n%s", briefing)
	}

	// And the real `hero resume` brief — the surface the model actually
	// reads at session start — now carries the auto-emitted ask too
	// (resume-brief-surfaces-handoff). This is the end of the loop:
	// auto-capture → travel → ingest → brief.
	briefB, err := digest.Generate(storeB, digest.Options{
		RepoKey:     repoKeyB,
		Branch:      gitutil.CurrentBranch(envB.dir),
		AuthorEmail: "alice@example.com",
		User:        "alice",
		Domain:      domainB,
	})
	storeB.Close()
	if err != nil {
		t.Fatalf("hero resume load path (digest.Generate) failed on B: %v", err)
	}
	if !strings.Contains(briefB.Markdown(), autoAsk) {
		t.Errorf("B's resume brief missing the auto-emitted ask:\n%s", briefB.Markdown())
	}
}

// divergentTeamCfg builds a team-mode config whose handoff slug is the
// given value — used to give machine B a LOCAL identity that differs
// from the slug baked into the traveled file. This is the exact
// cross-machine condition the previous guardrail could not exercise:
// teamCfg() pins "alice" on both sides, so a slug DIVERGENCE between
// machines never happened.
func divergentTeamCfg(slug string) config.Config {
	cfg := config.DefaultConfig()
	cfg.Next = &config.NextConfig{Mode: "team"}
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/" + slug}
	return cfg
}

// Test_HandoffContinuity_CrossMachine_SlugDivergence is the guardrail
// the spec demands — and the one the legacy cross-machine test
// structurally could not be. Machine A keys handoff under slug "alice"
// and commits .hero/next/alice.md. Machine B has a DIFFERENT local
// slug ("bob") and ONLY the committed alice.md travels (no graph.db).
// After the real ingest (which threads B's local slug, per B-1), B's
// `hero resume` brief — queried under B's "bob" identity — must STILL
// surface A's ask and suggestion. This proves the cross-machine load
// survives a slug divergence, which is the whole bug.
func Test_HandoffContinuity_CrossMachine_SlugDivergence(t *testing.T) {
	s := defaultSeed()

	// --- Machine A: slug "alice", project & capture committed bytes.
	cfgA := teamCfg() // DefaultAgent human/alice → slug "alice"
	envA := newTestEnv(t)
	if err := cfgA.Save(envA.dir); err != nil {
		t.Fatalf("save A cfg: %v", err)
	}
	committed, _ := seedMachineA(t, envA, cfgA, s)

	// --- Machine B: LOCAL slug "bob" (divergent), empty graph, only the
	// committed alice.md.
	const localSlugB = "bob"
	cfgB := divergentTeamCfg(localSlugB)
	envB := newTestEnv(t)
	if err := cfgB.Save(envB.dir); err != nil {
		t.Fatalf("save B cfg: %v", err)
	}
	if got := nextUserSlug(cfgB); got != localSlugB {
		t.Fatalf("B local slug = %q, want %q (test premise broken)", got, localSlugB)
	}
	// The file that traveled is alice.md (A's identity), NOT bob.md.
	bUserPath := filepath.Join(envB.heroDir, nextDirName, "alice.md")
	if err := os.MkdirAll(filepath.Dir(bUserPath), 0o755); err != nil {
		t.Fatalf("mkdir B next dir: %v", err)
	}
	if err := os.WriteFile(bUserPath, committed, 0o644); err != nil {
		t.Fatalf("drop committed file on B: %v", err)
	}

	repoKeyB := gitutil.RepoKey(envB.dir)
	domainB := graph.DomainFor(cfgB, graph.IntrinsicActive)

	// Sanity: B finds NOTHING under its local "bob" slug before ingest —
	// the file is keyed "alice", so a naive read misses (the bug).
	storeB, err := graph.Open(envB.heroDir)
	if err != nil {
		t.Fatalf("open B graph: %v", err)
	}
	if ask, _ := handoff.LatestAsk(storeB, localSlugB, repoKeyB, domainB); ask != nil && ask.Text != "" {
		storeB.Close()
		t.Fatalf("B graph not empty under %q before ingest", localSlugB)
	}

	// --- Rehydrate on B via the REAL ingest, threading B's local slug
	// exactly as runNextIngest does (B-1). This mirrors the singletons
	// under "bob" because "bob" currently has zero handoff nodes.
	if err := handoff.IngestUserFile(storeB, repoKeyB, domainB, bUserPath, nextUserSlug(cfgB), true); err != nil {
		storeB.Close()
		t.Fatalf("ingest on B: %v", err)
	}
	storeB.Close()

	// --- B's resume brief, queried under B's DIVERGENT local slug, must
	// carry A's content. This is the assertion the old guardrail could
	// not make. We replicate runResume's opts + identity reconciliation.
	storeB, err = graph.Open(envB.heroDir)
	if err != nil {
		t.Fatalf("reopen B graph: %v", err)
	}
	opts := digest.Options{
		RepoKey:     repoKeyB,
		Branch:      gitutil.CurrentBranch(envB.dir),
		AuthorEmail: "bob@example.com",
		User:        nextUserSlug(cfgB), // "bob"
		Domain:      domainB,
	}
	resolveHandoffIdentity(storeB, envB.heroDir, &opts, io.Discard)
	briefB, err := digest.Generate(storeB, opts)
	storeB.Close()
	if err != nil {
		t.Fatalf("hero resume load path failed on B: %v", err)
	}
	mdB := briefB.Markdown()
	for _, want := range []string{s.ask, s.suggestion} {
		if !strings.Contains(mdB, want) {
			t.Errorf("B's resume brief (local slug %q) did not carry A's handoff content %q:\n%s",
				localSlugB, want, mdB)
		}
	}

	// --- Prove it BITES: a naive read under B's local slug WITHOUT the
	// fix's mirror/fallback would find nothing. We assert that querying
	// the singleton directly under "bob" WITH the alias mirror in place
	// DOES resolve — and that querying under a third, never-seen slug
	// does NOT (so the alias is identity-scoped, not a blanket leak).
	storeB, err = graph.Open(envB.heroDir)
	if err != nil {
		t.Fatalf("reopen B graph for bite check: %v", err)
	}
	defer storeB.Close()
	if ask, _ := handoff.LatestAsk(storeB, localSlugB, repoKeyB, domainB); ask == nil || !strings.Contains(ask.Text, s.ask) {
		t.Errorf("B-1 mirror missing: no ask under local slug %q after ingest: %+v", localSlugB, ask)
	}
	if ask, _ := handoff.LatestAsk(storeB, "carol", repoKeyB, domainB); ask != nil && ask.Text != "" {
		t.Errorf("alias leaked to an unrelated slug \"carol\": %+v", ask)
	}
}

// Test_HandoffContinuity_SlugDivergence_FallbackWithoutMirror proves
// B-2/B-3: even when ingest did NOT mirror under the local slug (e.g.
// ingest ran on an earlier session before this fix, so nodes exist only
// under the file's "alice" identity), the read-side fallback in
// resolveHandoffIdentity resolves the single on-disk identity so the
// brief still loads — and the bite assertion confirms that without the
// fallback the brief would be empty under "bob".
func Test_HandoffContinuity_SlugDivergence_FallbackWithoutMirror(t *testing.T) {
	s := defaultSeed()

	cfgA := teamCfg()
	envA := newTestEnv(t)
	if err := cfgA.Save(envA.dir); err != nil {
		t.Fatalf("save A cfg: %v", err)
	}
	committed, _ := seedMachineA(t, envA, cfgA, s)

	const localSlugB = "bob"
	cfgB := divergentTeamCfg(localSlugB)
	envB := newTestEnv(t)
	if err := cfgB.Save(envB.dir); err != nil {
		t.Fatalf("save B cfg: %v", err)
	}
	bUserPath := filepath.Join(envB.heroDir, nextDirName, "alice.md")
	if err := os.MkdirAll(filepath.Dir(bUserPath), 0o755); err != nil {
		t.Fatalf("mkdir B next dir: %v", err)
	}
	if err := os.WriteFile(bUserPath, committed, 0o644); err != nil {
		t.Fatalf("drop committed file on B: %v", err)
	}

	repoKeyB := gitutil.RepoKey(envB.dir)
	domainB := graph.DomainFor(cfgB, graph.IntrinsicActive)

	// Ingest WITHOUT the local-slug mirror (localSlug=""), simulating a
	// prior-session ingest that keyed only under "alice".
	storeB, err := graph.Open(envB.heroDir)
	if err != nil {
		t.Fatalf("open B graph: %v", err)
	}
	if err := handoff.IngestUserFile(storeB, repoKeyB, domainB, bUserPath, "", true); err != nil {
		storeB.Close()
		t.Fatalf("ingest on B: %v", err)
	}
	storeB.Close()

	// BITE: without the read-side fallback, a query under "bob" finds
	// nothing (nodes are keyed "alice").
	storeB, err = graph.Open(envB.heroDir)
	if err != nil {
		t.Fatalf("reopen B graph: %v", err)
	}
	if ask, _ := handoff.LatestAsk(storeB, localSlugB, repoKeyB, domainB); ask != nil && ask.Text != "" {
		storeB.Close()
		t.Fatalf("test premise broken: ask present under %q without mirror", localSlugB)
	}

	// With the fallback, resolveHandoffIdentity adopts the single on-disk
	// identity ("alice"), and the brief loads A's content.
	opts := digest.Options{
		RepoKey:     repoKeyB,
		Branch:      gitutil.CurrentBranch(envB.dir),
		AuthorEmail: "bob@example.com",
		User:        localSlugB,
		Domain:      domainB,
	}
	resolveHandoffIdentity(storeB, envB.heroDir, &opts, io.Discard)
	if opts.User != "alice" {
		t.Errorf("read-side fallback did not adopt the single on-disk identity: opts.User=%q, want \"alice\"", opts.User)
	}
	briefB, err := digest.Generate(storeB, opts)
	storeB.Close()
	if err != nil {
		t.Fatalf("digest.Generate on B: %v", err)
	}
	mdB := briefB.Markdown()
	for _, want := range []string{s.ask, s.suggestion} {
		if !strings.Contains(mdB, want) {
			t.Errorf("B's brief missing %q after read-side fallback:\n%s", want, mdB)
		}
	}
}

// Test_HandoffContinuity_UnresolvableIdentity_IsObservable proves B-3:
// when handoff files exist under slugs that match neither the local
// identity nor resolve to a single fallback (two distinct users, none
// matching local "bob"), the read path emits an observable diagnostic
// naming the queried slug vs. the available slugs — NOT a silent empty
// section — and resume still succeeds (non-fatal).
func Test_HandoffContinuity_UnresolvableIdentity_IsObservable(t *testing.T) {
	const localSlugB = "bob"
	cfgB := divergentTeamCfg(localSlugB)
	envB := newTestEnv(t)
	if err := cfgB.Save(envB.dir); err != nil {
		t.Fatalf("save B cfg: %v", err)
	}
	repoKeyB := gitutil.RepoKey(envB.dir)
	domainB := graph.DomainFor(cfgB, graph.IntrinsicActive)

	// Two distinct real users with handoff content in the graph AND on
	// disk — neither is "bob". This is the ambiguous case: no single
	// fallback can be chosen.
	storeB, err := graph.Open(envB.heroDir)
	if err != nil {
		t.Fatalf("open B graph: %v", err)
	}
	for _, u := range []string{"alice", "dustin"} {
		if err := handoff.RecordAsk(storeB, repoKeyB, handoff.UserAsk{
			User: u, Domain: domainB, Text: "ASK_" + u,
		}); err != nil {
			storeB.Close()
			t.Fatalf("seed ask for %s: %v", u, err)
		}
		path := filepath.Join(envB.heroDir, nextDirName, u+".md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			storeB.Close()
			t.Fatalf("mkdir: %v", err)
		}
		md := "---\nuser: " + u + "\n---\n\n# " + u + "'s handoff\n\n## Last user ask\n\n> ASK_" + u + "\n"
		if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
			storeB.Close()
			t.Fatalf("write %s.md: %v", u, err)
		}
	}

	opts := digest.Options{
		RepoKey: repoKeyB,
		User:    localSlugB,
		Domain:  domainB,
	}
	var warn bytes.Buffer
	resolveHandoffIdentity(storeB, envB.heroDir, &opts, &warn)
	storeB.Close()

	// Identity stays "bob" (no single fallback resolvable).
	if opts.User != localSlugB {
		t.Errorf("ambiguous identity should not be auto-resolved: opts.User=%q, want %q", opts.User, localSlugB)
	}
	diag := warn.String()
	if diag == "" {
		t.Fatalf("B-3 violated: unresolvable identity produced NO diagnostic (silent empty handoff)")
	}
	// The hint must name the queried slug and the available slugs.
	if !strings.Contains(diag, localSlugB) {
		t.Errorf("diagnostic does not name the queried slug %q: %q", localSlugB, diag)
	}
	for _, u := range []string{"alice", "dustin"} {
		if !strings.Contains(diag, u) {
			t.Errorf("diagnostic does not name available slug %q: %q", u, diag)
		}
	}
}

// Test_persistDefaultAgentIfUnset_PinsCommittedIdentity proves A-1: when
// the committed hero.json has no tracking.defaultAgent, the stabilizer
// writes the current derived slug there so every clone derives the same
// identity. nextUserSlug already prefers defaultAgent over git config —
// asserted here too (the precedence the spec requires).
func Test_persistDefaultAgentIfUnset_PinsCommittedIdentity(t *testing.T) {
	env := newTestEnv(t)
	// Start from a committed config with NO tracking block.
	cfg := config.DefaultConfig()
	cfg.Tracking = nil
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	// Precondition: gitUserName() must yield a stable, non-"unknown" slug
	// in this environment for the pin to fire. If it can't, skip — the
	// stabilizer is correctly a no-op on an unidentifiable machine.
	slug := gitUserName()
	if slug == "" || slug == "unknown" {
		t.Skipf("no stable local identity (gitUserName=%q); A-1 correctly no-ops", slug)
	}

	merged, err := config.Load(env.dir)
	if err != nil {
		t.Fatalf("load cfg: %v", err)
	}
	updated := persistDefaultAgentIfUnset(env.dir, merged)
	if updated == nil {
		t.Fatalf("expected A-1 to pin defaultAgent, got nil")
	}
	if updated.Tracking == nil || updated.Tracking.DefaultAgent != slug {
		t.Errorf("in-memory config not updated: %+v, want defaultAgent=%q", updated.Tracking, slug)
	}

	// The COMMITTED file must now carry it (not hero.local.json).
	reloaded, err := config.Load(env.dir)
	if err != nil {
		t.Fatalf("reload cfg: %v", err)
	}
	if reloaded.Tracking == nil || reloaded.Tracking.DefaultAgent != slug {
		t.Errorf("committed hero.json did not persist defaultAgent: %+v", reloaded.Tracking)
	}
	// Precedence: nextUserSlug prefers the now-committed defaultAgent.
	if got := nextUserSlug(reloaded); got != slug {
		t.Errorf("nextUserSlug = %q, want pinned defaultAgent %q (precedence violated)", got, slug)
	}

	// Idempotent: a second call is a no-op (already pinned).
	if again := persistDefaultAgentIfUnset(env.dir, reloaded); again != nil {
		t.Errorf("A-1 should be a no-op when defaultAgent is already set, got %+v", again.Tracking)
	}
}

// Test_persistDefaultAgentIfUnset_RespectsExistingPin proves A-1 never
// churns a workspace that already pins identity (committed or via the
// merged local config). The committed file is left byte-for-byte intact.
func Test_persistDefaultAgentIfUnset_RespectsExistingPin(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/preset"}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	before, err := os.ReadFile(filepath.Join(env.heroDir, config.ConfigFileName))
	if err != nil {
		t.Fatalf("read committed cfg: %v", err)
	}

	merged, err := config.Load(env.dir)
	if err != nil {
		t.Fatalf("load cfg: %v", err)
	}
	if got := persistDefaultAgentIfUnset(env.dir, merged); got != nil {
		t.Errorf("A-1 churned an already-pinned workspace: %+v", got.Tracking)
	}
	after, err := os.ReadFile(filepath.Join(env.heroDir, config.ConfigFileName))
	if err != nil {
		t.Fatalf("re-read committed cfg: %v", err)
	}
	if string(before) != string(after) {
		t.Errorf("committed hero.json changed despite existing pin:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// Test_IngestUserFile_TeamModeGate_NoCrossContamination proves the B-1
// anti-corruption gate: when the local slug ALREADY has its own handoff
// nodes (a genuine second team member), ingesting another user's file
// must NOT mirror that user's content under the local slug.
func Test_IngestUserFile_TeamModeGate_NoCrossContamination(t *testing.T) {
	env := newTestEnv(t)
	repoKey := gitutil.RepoKey(env.dir)
	const domain = "engineering"

	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("open graph: %v", err)
	}
	// "bob" is a real local user with his OWN handoff already in the graph.
	if err := handoff.RecordAsk(store, repoKey, handoff.UserAsk{
		User: "bob", Domain: domain, Text: "BOB_OWN_ASK",
	}); err != nil {
		store.Close()
		t.Fatalf("seed bob ask: %v", err)
	}
	store.Close()

	// alice.md travels in. Ingest with localSlug="bob". Because "bob"
	// already has handoff content, the alias gate must NOT fire.
	alicePath := filepath.Join(env.heroDir, nextDirName, "alice.md")
	if err := os.MkdirAll(filepath.Dir(alicePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	aliceMD := "---\nuser: alice\n---\n\n# alice's handoff\n\n## Last user ask\n\n> ALICE_ASK\n"
	if err := os.WriteFile(alicePath, []byte(aliceMD), 0o644); err != nil {
		t.Fatalf("write alice.md: %v", err)
	}

	store, err = graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("reopen graph: %v", err)
	}
	defer store.Close()
	if err := handoff.IngestUserFile(store, repoKey, domain, alicePath, "bob", true); err != nil {
		t.Fatalf("ingest alice.md: %v", err)
	}

	// bob's ask must be UNCHANGED — alice's content must not bleed in.
	bobAsk, _ := handoff.LatestAsk(store, "bob", repoKey, domain)
	if bobAsk == nil || bobAsk.Text != "BOB_OWN_ASK" {
		t.Errorf("bob's handoff was corrupted by the alias: got %+v, want text=BOB_OWN_ASK", bobAsk)
	}
	// alice's own keying must still be present (recorded under her slug).
	aliceAsk, _ := handoff.LatestAsk(store, "alice", repoKey, domain)
	if aliceAsk == nil || aliceAsk.Text != "ALICE_ASK" {
		t.Errorf("alice's own handoff missing after ingest: %+v", aliceAsk)
	}
}

// Test_IngestUserFile_MultiFile_NoAlias closes the empty-graph-teammate
// leak a cold audit flagged. The zero-node alias gate alone is unsafe for
// a BRAND-NEW teammate: their graph is empty, so they too have zero
// handoff nodes — and the old gate would mirror whichever other user's
// .hero/next/*.md sorts first onto their local slug. Asks/suggestions are
// singletons that self-heal next checkpoint, but REFLECTIONS leak and
// persist under the wrong identity, violating the "no cross-contamination"
// guarantee.
//
// Scenario: a repo with TWO travel-eligible files (alice.md, bob.md), a
// local user "carol" with an EMPTY graph, runs ingest exactly as
// runNextIngest does — computing the single-file gate via the real
// nextFileUserSlugs helper. With two distinct identities on disk the gate
// is false, so NO node — especially no reflection — may be mirrored onto
// carol.
//
// Bite proof: drop the single-file condition (pass singleTravelFile=true,
// the pre-this-change behaviour) and carol gets alice's reflection
// mirrored — the very leak the production gate now blocks.
func Test_IngestUserFile_MultiFile_NoAlias(t *testing.T) {
	env := newTestEnv(t)
	repoKey := gitutil.RepoKey(env.dir)
	const domain = "engineering"
	const localSlug = "carol" // brand-new teammate: empty graph, zero nodes

	nextDir := filepath.Join(env.heroDir, nextDirName)
	if err := os.MkdirAll(nextDir, 0o755); err != nil {
		t.Fatalf("mkdir next dir: %v", err)
	}

	// Two distinct travel-eligible files. Each carries a UNIQUE reflection
	// so a leak is unambiguous: if carol ends up with either reflection,
	// it bled across from someone else's file.
	const aliceReflection = "ALICE_REFLECTION only alice should ever own this"
	const bobReflection = "BOB_REFLECTION only bob should ever own this"
	aliceMD := "---\nuser: alice\n---\n\n# alice's handoff\n\n" +
		"## Last user ask\n\n> ALICE_ASK\n\n## Recent reflections\n\n- " + aliceReflection + "\n"
	bobMD := "---\nuser: bob\n---\n\n# bob's handoff\n\n" +
		"## Last user ask\n\n> BOB_ASK\n\n## Recent reflections\n\n- " + bobReflection + "\n"
	alicePath := filepath.Join(nextDir, "alice.md")
	bobPath := filepath.Join(nextDir, "bob.md")
	if err := os.WriteFile(alicePath, []byte(aliceMD), 0o644); err != nil {
		t.Fatalf("write alice.md: %v", err)
	}
	if err := os.WriteFile(bobPath, []byte(bobMD), 0o644); err != nil {
		t.Fatalf("write bob.md: %v", err)
	}

	// The gate exactly as runNextIngest computes it: distinct travel-
	// eligible identities on disk. Two files → false → alias suppressed.
	singleTravelFile := len(nextFileUserSlugs(env.heroDir)) == 1
	if singleTravelFile {
		t.Fatalf("test premise broken: two distinct files present but gate computed single=true (slugs=%v)",
			nextFileUserSlugs(env.heroDir))
	}

	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("open graph: %v", err)
	}
	defer store.Close()

	// Ingest BOTH files under carol's local slug, mirroring runNextIngest's
	// loop. Order is deterministic (sorted), so alice.md is processed first
	// — the file the old zero-node gate would have mirrored onto carol.
	for _, p := range []string{alicePath, bobPath} {
		if err := handoff.IngestUserFile(store, repoKey, domain, p, localSlug, singleTravelFile); err != nil {
			t.Fatalf("ingest %s: %v", filepath.Base(p), err)
		}
	}

	// carol's identity must be PRISTINE — no ask, no reflection mirrored.
	if ask, _ := handoff.LatestAsk(store, localSlug, repoKey, domain); ask != nil && ask.Text != "" {
		t.Errorf("alias leaked an ask onto %q: %+v", localSlug, ask)
	}
	carolRefs, _ := handoff.RecentReflections(store, localSlug, repoKey, domain, 10)
	if len(carolRefs) != 0 {
		t.Errorf("alias leaked %d reflection(s) onto %q: %+v — empty-graph teammate contaminated",
			len(carolRefs), localSlug, carolRefs)
	}

	// And each user's own keying must still be intact under their own slug.
	for slug, want := range map[string]string{"alice": aliceReflection, "bob": bobReflection} {
		refs, _ := handoff.RecentReflections(store, slug, repoKey, domain, 10)
		found := false
		for _, r := range refs {
			if r.Text == want {
				found = true
			}
		}
		if !found {
			t.Errorf("%s's own reflection missing under their slug: %+v", slug, refs)
		}
	}
}

// --- AC-3 helpers ---------------------------------------------------

// repoRootForTest finds the repository root by walking up from this
// source file's directory until it hits the root .gitignore (or the
// git toplevel). The test's working dir is a temp env, so we cannot
// rely on cwd.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed; cannot locate repo root for AC-3")
	}
	dir := filepath.Dir(thisFile)
	for i := 0; i < 20; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".gitignore")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not locate repo root (.gitignore + .git) for AC-3")
	return ""
}

// gitIgnored reports whether the repo's real .gitignore matches relPath,
// using `git check-ignore` for authentic gitignore semantics.
func gitIgnored(t *testing.T, repoRoot, relPath string) bool {
	t.Helper()
	cmd := exec.Command("git", "-C", repoRoot, "check-ignore", relPath)
	err := cmd.Run()
	if err == nil {
		return true // matched → ignored
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false // exit 1 → not ignored
	}
	t.Fatalf("git check-ignore %q failed unexpectedly: %v", relPath, err)
	return false
}
