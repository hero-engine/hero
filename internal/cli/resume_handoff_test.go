package cli

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
	"github.com/spf13/cobra"
)

// Test_Resume_SurfacesHandoffAsk is Test Plan #5 — the `hero resume`
// integration. Seed a UserAsk through the same (user, repo, domain)
// triple the resume path resolves, run the real resume command, and
// assert the rendered markdown carries the last ask in the handoff
// section. This proves the wiring end-to-end: brief.go populates
// User/Domain, digest.handoffSection reads them, the brief renders it.
func Test_Resume_SurfacesHandoffAsk(t *testing.T) {
	const askText = "RESUME_ASK_TEXT what was I doing before the session reset?"
	const sugText = "RESUME_SUGGEST_TEXT finish wiring the handoff section into resume"

	cfg := teamCfg() // DefaultAgent human/alice → slug "alice"
	env := newTestEnv(t)
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	// Seed via the exact keying the resume path will use: repoKey from
	// the project root, domain from the active config.
	repoKey := gitutil.RepoKey(env.dir)
	domain := graph.DomainFor(cfg, graph.IntrinsicActive)

	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("open graph: %v", err)
	}
	if err := handoff.RecordAsk(store, repoKey, handoff.UserAsk{
		User: "alice", Domain: domain, Text: askText,
	}); err != nil {
		store.Close()
		t.Fatalf("record ask: %v", err)
	}
	if err := handoff.RecordSuggestion(store, repoKey, handoff.NextSuggestion{
		User: "alice", Domain: domain, Text: sugText, Rationale: "test wiring",
	}); err != nil {
		store.Close()
		t.Fatalf("record suggestion: %v", err)
	}
	store.Close()

	// Drive the real resume command. captureStdout grabs what the model
	// would see.
	resetFlags()
	resumeBudget = 3000
	resumeJSON = false
	resumeEmail = "alice@example.com"
	resumeFocus = nil
	resumeAuto = false

	var runErr error
	out := captureStdout(func() {
		cmd := &cobra.Command{RunE: runResume}
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("hero resume failed: %v", runErr)
	}

	if !strings.Contains(out, "## Where you left off") {
		t.Errorf("resume brief missing handoff section:\n%s", out)
	}
	if !strings.Contains(out, askText) {
		t.Errorf("resume brief missing the last ask %q:\n%s", askText, out)
	}
	if !strings.Contains(out, sugText) {
		t.Errorf("resume brief missing the suggested-next %q:\n%s", sugText, out)
	}
}
