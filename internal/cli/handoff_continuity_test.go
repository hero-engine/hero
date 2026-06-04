package cli

import (
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
// The spec's design sketch names `digest.Generate` (the `hero resume`
// path) as the thing to assert handoff content against. Reading the
// code shows that is not where per-user handoff content surfaces:
// `digest.Generate`'s brief carries Mission / In-flight / Just-changed
// / Tried / Blocked / Nearby — it does NOT query the UserAsk /
// NextSuggestion / SessionReflection singletons. The handoff content a
// fresh session actually consumes surfaces through two real paths:
//
//   1. The graph-query surface the `hero next ask/suggest/reflection`
//      commands read: handoff.LatestAsk / projection.PickUserSuggestion
//      / handoff.RecentReflections. These ARE the functions the CLI
//      calls (see internal/cli/next_handoff.go) — not a reimplementation.
//   2. The re-projected `.hero/next/<user>.md`, rendered from the graph
//      by writeUserHandoffFile → projection.UserHandoffMD. This is the
//      personal briefing a fresh session opens.
//
// So the guardrail asserts handoff reconstruction against BOTH of those
// real surfaces, and ALSO runs `digest.Generate` on the rehydrated B
// graph to prove the actual `hero resume` load path executes end-to-end
// against the traveled-and-ingested state (it must not error, and must
// see B as a populated workspace). Asserting against the genuine
// consumption surfaces — rather than the brief, which structurally
// cannot carry this content today — is what makes the guardrail honest:
// it tracks the path a fresh session takes, and it bites when travel
// breaks (see Test_HandoffContinuity_CrossMachine_GuardrailBites).

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
	if err := handoff.IngestUserFile(storeB, repoKeyB, domainB, bUserPath); err != nil {
		storeB.Close()
		t.Fatalf("ingest on B: %v", err)
	}
	storeB.Close()

	// --- Prove the real `hero resume` load path runs end-to-end on the
	// rehydrated B graph. The brief structurally does not carry handoff
	// singletons (see file header), but it MUST generate without error
	// against the traveled-and-ingested state — that is the load a fresh
	// session performs first.
	storeB, err = graph.Open(envB.heroDir)
	if err != nil {
		t.Fatalf("reopen B graph: %v", err)
	}
	if _, err := digest.Generate(storeB, digest.Options{
		RepoKey:     repoKeyB,
		Branch:      gitutil.CurrentBranch(envB.dir),
		AuthorEmail: "alice@example.com",
	}); err != nil {
		storeB.Close()
		t.Fatalf("hero resume load path (digest.Generate) failed on B: %v", err)
	}
	storeB.Close()

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
		if err := handoff.IngestUserFile(store, repoKeyB, domainB, bUserPath); err != nil {
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
	if err := handoff.IngestUserFile(storeB, repoKeyB, domainB, bUserPath); err != nil {
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
	storeB.Close()
	if err != nil {
		t.Fatalf("re-project B briefing: %v", err)
	}
	if !strings.Contains(briefing, autoAsk) {
		t.Errorf("B re-projected briefing missing auto-emitted ask:\n%s", briefing)
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
