package peering

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	contractpeering "github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// TestCallDryRunAdvisory verifies the dispatcher's dry-run path:
// builds an envelope, never exec's a subagent, returns synthetic
// findings result. No state on the peer side, no trail on the
// originator side.
func TestCallDryRunAdvisory(t *testing.T) {
	root := t.TempDir()
	originRoot := filepath.Join(root, "client")
	peerRoot := filepath.Join(root, "app")
	setupWorkspace(t, originRoot, "11111111-1111-4111-8111-111111111111")
	setupWorkspace(t, peerRoot, "22222222-2222-4222-8222-222222222222")

	cfg, _ := config.Load(originRoot)
	cfg.Repos = map[string]string{"app": peerRoot}
	_ = cfg.Save(originRoot)

	res, err := Call(originRoot, CallOptions{
		PeerAlias: "app",
		Mode:      contractpeering.PeerCallAdvisory,
		Prompt:    "What's your error envelope shape?",
		DryRun:    true,
		Stdout:    io.Discard,
	})
	if err != nil {
		t.Fatalf("Call dry-run: %v", err)
	}
	if res.PeerID != "22222222-2222-4222-8222-222222222222" {
		t.Errorf("peer_id: %q", res.PeerID)
	}
	if !strings.Contains(res.EnvelopeJSON, "Advisory mode.") {
		t.Errorf("envelope missing advisory instructions:\n%s", res.EnvelopeJSON)
	}
	if !strings.Contains(res.EnvelopeJSON, "What's your error envelope shape?") {
		t.Errorf("envelope missing prompt:\n%s", res.EnvelopeJSON)
	}
	if res.Result.Kind != contractpeering.ResultFindings {
		t.Errorf("dry-run kind: %q", res.Result.Kind)
	}
}

// TestCallRejectsUnknownPeer fails fast when the alias isn't configured.
func TestCallRejectsUnknownPeer(t *testing.T) {
	root := t.TempDir()
	originRoot := filepath.Join(root, "client")
	setupWorkspace(t, originRoot, "11111111-1111-4111-8111-111111111111")

	_, err := Call(originRoot, CallOptions{
		PeerAlias: "missing",
		Mode:      contractpeering.PeerCallAdvisory,
		Prompt:    "hi",
		DryRun:    true,
		Stdout:    io.Discard,
	})
	if err == nil {
		t.Fatal("expected error for unknown peer")
	}
}

// TestCallRejectsFullMode locks the v2 deferral: full delivery is not
// shipped in Phase 2 and must surface a clear error rather than be
// silently treated as advisory.
func TestCallRejectsFullMode(t *testing.T) {
	root := t.TempDir()
	originRoot := filepath.Join(root, "client")
	peerRoot := filepath.Join(root, "app")
	setupWorkspace(t, originRoot, "11111111-1111-4111-8111-111111111111")
	setupWorkspace(t, peerRoot, "22222222-2222-4222-8222-222222222222")
	cfg, _ := config.Load(originRoot)
	cfg.Repos = map[string]string{"app": peerRoot}
	_ = cfg.Save(originRoot)

	_, err := Call(originRoot, CallOptions{
		PeerAlias: "app",
		Mode:      contractpeering.PeerCallFull,
		Prompt:    "do everything",
		DryRun:    true,
		Stdout:    io.Discard,
	})
	if err == nil {
		t.Fatal("expected mode=full to be rejected")
	}
}

// TestParseResultBlock verifies the structured-result extractor for
// both well-formed and malformed subagent output.
func TestParseResultBlock(t *testing.T) {
	t.Run("findings", func(t *testing.T) {
		stdout := `Some preamble chatter from the subagent...

<peer-call-result>
kind: findings
findings: |
  Error envelopes follow rfc7807-lite — code,
  message, details map.
budget_consumed:
  turns: 4
  tokens: ~1842
</peer-call-result>

Trailing chatter ignored.
`
		r, err := parseResultBlock(stdout)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if r.Kind != contractpeering.ResultFindings {
			t.Errorf("kind: %q", r.Kind)
		}
		if !strings.Contains(r.Findings, "rfc7807-lite") {
			t.Errorf("findings: %q", r.Findings)
		}
		if r.BudgetConsumed.Turns != 4 || r.BudgetConsumed.Tokens != 1842 {
			t.Errorf("budget: %+v", r.BudgetConsumed)
		}
	})

	t.Run("tolerant budget forms", func(t *testing.T) {
		cases := []struct {
			name   string
			line   string
			wantOK bool
			want   contractpeering.ApproxInt
		}{
			{"plain int", "tokens: 22000", true, 22000},
			{"tilde-prefixed", "tokens: ~22000", true, 22000},
			{"float", "tokens: 22000.0", true, 22000},
			{"float truncates", "tokens: 22000.7", true, 22000},
			{"quoted int", `tokens: "22000"`, true, 22000},
			{"quoted tilde", `tokens: "~22000"`, true, 22000},
			{"zero", "tokens: 0", true, 0},
			{"missing key", "", true, 0},
			{"negative rejected", "tokens: -1", false, 0},
			{"garbage rejected", "tokens: lots", false, 0},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				stdout := "<peer-call-result>\nkind: findings\nbudget_consumed:\n  turns: 1\n  " + tc.line + "\n</peer-call-result>\n"
				r, err := parseResultBlock(stdout)
				if tc.wantOK {
					if err != nil {
						t.Fatalf("parse: %v", err)
					}
					if r.BudgetConsumed.Tokens != tc.want {
						t.Errorf("tokens: got %d, want %d", r.BudgetConsumed.Tokens, tc.want)
					}
				} else if err == nil {
					t.Fatalf("expected parse error, got tokens=%d", r.BudgetConsumed.Tokens)
				}
			})
		}
	})

	t.Run("spec-ref", func(t *testing.T) {
		stdout := `<peer-call-result>
kind: spec-ref
spec_slug: error-envelope-mismatch
peer_status: planning
budget_consumed:
  turns: 18
  tokens: 47000
</peer-call-result>
`
		r, err := parseResultBlock(stdout)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if r.Kind != contractpeering.ResultSpecRef {
			t.Errorf("kind: %q", r.Kind)
		}
		if r.SpecSlug != "error-envelope-mismatch" {
			t.Errorf("slug: %q", r.SpecSlug)
		}
	})

	t.Run("missing fence", func(t *testing.T) {
		_, err := parseResultBlock("just some text with no fence")
		if err == nil {
			t.Fatal("expected error for missing fence")
		}
	})
}

// TestBudgetDefaults locks the budget numbers documented in the
// kickoff. Changing these is a behavior change that should require a
// spec amendment.
func TestBudgetDefaults(t *testing.T) {
	a := applyBudgetDefaults(contractpeering.PeerCallAdvisory, contractpeering.BudgetSpec{})
	if a.Turns != DefaultAdvisoryTurns || a.Tokens != DefaultAdvisoryTokens {
		t.Errorf("advisory defaults drifted: %+v", a)
	}
	s := applyBudgetDefaults(contractpeering.PeerCallSpecOut, contractpeering.BudgetSpec{})
	if s.Turns != DefaultSpecOutTurns || s.Tokens != DefaultSpecOutTokens {
		t.Errorf("spec-out defaults drifted: %+v", s)
	}

	// User-supplied values must not be overridden.
	override := applyBudgetDefaults(contractpeering.PeerCallAdvisory, contractpeering.BudgetSpec{Turns: 5, Tokens: 100})
	if override.Turns != 5 || override.Tokens != 100 {
		t.Errorf("user budget got clobbered: %+v", override)
	}
}

// TestReconcileAwaitingPeerFlipsHandedBack verifies the auto-fire
// reconciler: an awaiting_peer originator with a peer counterpart that
// has reached completed gets moved to handed_back with a trail entry.
func TestReconcileAwaitingPeerFlipsHandedBack(t *testing.T) {
	root := t.TempDir()
	originRoot := filepath.Join(root, "client")
	peerRoot := filepath.Join(root, "app")
	setupWorkspace(t, originRoot, "11111111-1111-4111-8111-111111111111")
	setupWorkspace(t, peerRoot, "22222222-2222-4222-8222-222222222222")

	cfg, _ := config.Load(originRoot)
	cfg.Repos = map[string]string{"app": peerRoot}
	_ = cfg.Save(originRoot)

	// Originator spec in awaiting_peer with a trail entry pointing at
	// a peer spec.
	origPath := writeFeatureSpec(t, originRoot, "order-failure", "Order failure")
	origData, _ := os.ReadFile(origPath)
	updated := spec.SetFrontmatterField(string(origData), "status", string(spec.StatusAwaitingPeer))
	updated = AppendTrailToContent(updated, contractpeering.TrailEntry{
		At:               time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC),
		Direction:        contractpeering.DirectionOut,
		PeerAliasDisplay: "app",
		PeerID:           "22222222-2222-4222-8222-222222222222",
		Mode:             contractpeering.ModeSpecOut,
		OriginatingSpec:  "order-failure",
		PeerSpec:         "app/error-envelope-mismatch",
	})
	if err := os.WriteFile(origPath, []byte(updated), 0o644); err != nil {
		t.Fatalf("write origin: %v", err)
	}

	// Peer spec in completed status.
	peerSpecDir := filepath.Join(peerRoot, ".hero", "planning", "features", "error-envelope-mismatch")
	if err := os.MkdirAll(peerSpecDir, 0o755); err != nil {
		t.Fatalf("mkdir peer: %v", err)
	}
	peerContent := "---\ntitle: \"Error envelope mismatch\"\ntype: feature\nstatus: completed\n---\n\n# Error envelope mismatch\n"
	if err := os.WriteFile(filepath.Join(peerSpecDir, "spec.md"), []byte(peerContent), 0o644); err != nil {
		t.Fatalf("write peer spec: %v", err)
	}

	transitioned, err := ReconcileAwaitingPeer(originRoot, nil)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(transitioned) != 1 || transitioned[0] != "order-failure" {
		t.Fatalf("expected [order-failure], got %v", transitioned)
	}

	final, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("re-read origin: %v", err)
	}
	finalSpec, err := spec.Parse(string(final), origPath, time.Now())
	if err != nil {
		t.Fatalf("re-parse origin: %v", err)
	}
	if finalSpec.Status != spec.StatusHandedBack {
		t.Errorf("post-reconcile status: %q", finalSpec.Status)
	}
	entries, _ := ReadTrail(origPath)
	if len(entries) != 2 {
		t.Fatalf("trail length: %d", len(entries))
	}
	last := entries[len(entries)-1]
	if last.Mode != contractpeering.ModeHandedBack {
		t.Errorf("last trail mode: %q", last.Mode)
	}
	if last.Direction != contractpeering.DirectionIn {
		t.Errorf("last trail direction: %q", last.Direction)
	}
}

// TestReconcileAwaitingPeerNoOpWhenPeerStillWorking confirms that an
// awaiting_peer spec whose counterpart is still in planning does NOT
// get flipped.
func TestReconcileAwaitingPeerNoOpWhenPeerStillWorking(t *testing.T) {
	root := t.TempDir()
	originRoot := filepath.Join(root, "client")
	peerRoot := filepath.Join(root, "app")
	setupWorkspace(t, originRoot, "11111111-1111-4111-8111-111111111111")
	setupWorkspace(t, peerRoot, "22222222-2222-4222-8222-222222222222")
	cfg, _ := config.Load(originRoot)
	cfg.Repos = map[string]string{"app": peerRoot}
	_ = cfg.Save(originRoot)

	origPath := writeFeatureSpec(t, originRoot, "x", "X")
	origData, _ := os.ReadFile(origPath)
	updated := spec.SetFrontmatterField(string(origData), "status", string(spec.StatusAwaitingPeer))
	updated = AppendTrailToContent(updated, contractpeering.TrailEntry{
		At:               time.Now().UTC(),
		Direction:        contractpeering.DirectionOut,
		PeerAliasDisplay: "app",
		PeerID:           "22222222-2222-4222-8222-222222222222",
		Mode:             contractpeering.ModeSpecOut,
		PeerSpec:         "app/peer-x",
	})
	_ = os.WriteFile(origPath, []byte(updated), 0o644)

	peerSpecDir := filepath.Join(peerRoot, ".hero", "planning", "features", "peer-x")
	_ = os.MkdirAll(peerSpecDir, 0o755)
	_ = os.WriteFile(filepath.Join(peerSpecDir, "spec.md"),
		[]byte("---\ntitle: Y\ntype: feature\nstatus: planning\n---\n"), 0o644)

	transitioned, err := ReconcileAwaitingPeer(originRoot, nil)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(transitioned) != 0 {
		t.Errorf("expected no transitions, got %v", transitioned)
	}
}

// TestReadPeerManifest verifies the manifest reader returns a clear
// error pointing at `hero index` when the file is missing.
func TestReadPeerManifestMissing(t *testing.T) {
	root := t.TempDir()
	_, err := ReadPeerManifest(root, ".hero")
	if err == nil {
		t.Fatal("expected error for missing manifest")
	}
	if !strings.Contains(err.Error(), "hero index") {
		t.Errorf("error should hint `hero index`: %v", err)
	}
}

// TestWritePeerCallArtifact_FullFindings verifies the per-call
// artifact captures the entire findings string verbatim — no
// truncation — and lives at .hero/peer-calls/<call_id>.md regardless
// of related_spec. This is the regression test for the 400-char
// stdout cap bug: the artifact must always carry the full text.
func TestWritePeerCallArtifact_FullFindings(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	longFindings := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 80)
	if len(longFindings) <= 400 {
		t.Fatalf("test fixture must exceed 400 chars, got %d", len(longFindings))
	}

	req := contractpeering.PeerCallRequest{
		CallID:       "01HZTESTCALLID0000000000",
		Mode:         contractpeering.PeerCallAdvisory,
		OriginPeerID: "11111111-1111-4111-8111-111111111111",
		TargetPeerID: "22222222-2222-4222-8222-222222222222",
		Prompt:       "What's your error envelope shape?",
		Reason:       "regression check",
		At:           time.Date(2026, 5, 19, 23, 0, 0, 0, time.UTC),
	}
	res := contractpeering.PeerCallResult{
		Kind:     contractpeering.ResultFindings,
		Findings: longFindings,
		BudgetConsumed: contractpeering.BudgetConsumed{
			Turns: 7, Tokens: 1842,
		},
	}

	rel, err := writePeerCallArtifact(root, heroDir, "app", req, res)
	if err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	wantRel := filepath.Join(".hero", "peer-calls", req.CallID+".md")
	if rel != wantRel {
		t.Errorf("rel path: got %q, want %q", rel, wantRel)
	}

	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, longFindings) {
		t.Errorf("artifact missing full findings text")
	}
	if !strings.Contains(got, "call_id: "+req.CallID) {
		t.Errorf("artifact missing call_id frontmatter:\n%s", got)
	}
	if !strings.Contains(got, "mode: advisory") {
		t.Errorf("artifact missing mode frontmatter")
	}
	if !strings.Contains(got, "peer_alias: app") {
		t.Errorf("artifact missing peer_alias frontmatter")
	}
	if !strings.Contains(got, "result_kind: findings") {
		t.Errorf("artifact missing result_kind frontmatter")
	}
	if !strings.Contains(got, "## Prompt") || !strings.Contains(got, "## Findings") {
		t.Errorf("artifact missing expected sections")
	}
	if !strings.Contains(got, "What's your error envelope shape?") {
		t.Errorf("artifact missing prompt body")
	}
	if !strings.Contains(got, "turns: 7") || !strings.Contains(got, "tokens: 1842") {
		t.Errorf("artifact missing budget_consumed")
	}
}

// TestWritePeerCallArtifact_SpecRefKind verifies the spec-out
// rendering path: the artifact records the produced peer spec slug
// and peer_status rather than synthesizing a Findings section from
// empty text.
func TestWritePeerCallArtifact_SpecRefKind(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	req := contractpeering.PeerCallRequest{
		CallID:       "01HZTESTSPECREF0000000000",
		Mode:         contractpeering.PeerCallSpecOut,
		OriginPeerID: "11111111-1111-4111-8111-111111111111",
		TargetPeerID: "22222222-2222-4222-8222-222222222222",
		Prompt:       "Design the error envelope fix.",
		At:           time.Date(2026, 5, 19, 23, 0, 0, 0, time.UTC),
	}
	res := contractpeering.PeerCallResult{
		Kind:       contractpeering.ResultSpecRef,
		SpecSlug:   "error-envelope-mismatch",
		PeerStatus: "planning",
	}
	rel, err := writePeerCallArtifact(root, heroDir, "app", req, res)
	if err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "## Spec produced") {
		t.Errorf("expected spec-produced section:\n%s", got)
	}
	if !strings.Contains(got, "spec_slug: error-envelope-mismatch") {
		t.Errorf("expected spec_slug")
	}
	if !strings.Contains(got, "peer_spec: app/error-envelope-mismatch") {
		t.Errorf("expected peer_spec frontmatter")
	}
}

// TestRecordOriginatorSideUsesArtifactPathAsResultRef locks the
// trail-entry wiring: when an artifact path is supplied, the trail
// entry's ResultRef points at it rather than the bare call_id.
func TestRecordOriginatorSideUsesArtifactPathAsResultRef(t *testing.T) {
	root := t.TempDir()
	originRoot := filepath.Join(root, "client")
	setupWorkspace(t, originRoot, "11111111-1111-4111-8111-111111111111")
	cfg, _ := config.Load(originRoot)

	specPath := writeFeatureSpec(t, originRoot, "order-failure", "Order failure")

	req := contractpeering.PeerCallRequest{
		CallID:       "01HZTRAILREFTEST00000000",
		Mode:         contractpeering.PeerCallAdvisory,
		OriginPeerID: cfg.PeerID,
		TargetPeerID: "22222222-2222-4222-8222-222222222222",
		At:           time.Now().UTC(),
	}
	result := contractpeering.PeerCallResult{
		CallID:   req.CallID,
		Kind:     contractpeering.ResultFindings,
		Findings: "ok",
		At:       req.At,
	}
	opts := CallOptions{
		PeerAlias:   "app",
		Mode:        contractpeering.PeerCallAdvisory,
		Prompt:      "hi",
		RelatedSpec: "order-failure",
	}
	artifactRel := filepath.Join(".hero", "peer-calls", req.CallID+".md")
	if err := recordOriginatorSide(originRoot, cfg, opts, req, result, "22222222-2222-4222-8222-222222222222", "", artifactRel); err != nil {
		t.Fatalf("recordOriginatorSide: %v", err)
	}
	entries, err := ReadTrail(specPath)
	if err != nil {
		t.Fatalf("read trail: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 trail entry, got %d", len(entries))
	}
	if entries[0].ResultRef != artifactRel {
		t.Errorf("trail ResultRef: got %q, want %q", entries[0].ResultRef, artifactRel)
	}
}

// TestRecordOriginatorSideFallsBackToCallIDWhenNoArtifact preserves
// the prior behavior when the artifact path is empty (e.g., artifact
// write failed). ResultRef should still hold the bare call_id so the
// trail entry is not silently empty.
func TestRecordOriginatorSideFallsBackToCallIDWhenNoArtifact(t *testing.T) {
	root := t.TempDir()
	originRoot := filepath.Join(root, "client")
	setupWorkspace(t, originRoot, "11111111-1111-4111-8111-111111111111")
	cfg, _ := config.Load(originRoot)

	specPath := writeFeatureSpec(t, originRoot, "x", "X")

	req := contractpeering.PeerCallRequest{
		CallID:       "01HZFALLBACKCALLID000000",
		Mode:         contractpeering.PeerCallAdvisory,
		OriginPeerID: cfg.PeerID,
		TargetPeerID: "22222222-2222-4222-8222-222222222222",
		At:           time.Now().UTC(),
	}
	result := contractpeering.PeerCallResult{
		CallID: req.CallID,
		Kind:   contractpeering.ResultFindings,
		At:     req.At,
	}
	opts := CallOptions{
		PeerAlias:   "app",
		Mode:        contractpeering.PeerCallAdvisory,
		Prompt:      "hi",
		RelatedSpec: "x",
	}
	if err := recordOriginatorSide(originRoot, cfg, opts, req, result, "22222222-2222-4222-8222-222222222222", "", ""); err != nil {
		t.Fatalf("recordOriginatorSide: %v", err)
	}
	entries, err := ReadTrail(specPath)
	if err != nil {
		t.Fatalf("read trail: %v", err)
	}
	if len(entries) != 1 || entries[0].ResultRef != req.CallID {
		t.Fatalf("expected ResultRef==call_id fallback, got %+v", entries)
	}
}

// TestSubagentRunError_DetectsLoggedOut verifies the regression for the
// swallowed-login-message bug: claude prints "Not logged in · Please run
// /login" to stdout (stderr empty), and the old code surfaced only
// stderr — yielding an opaque "exit status 1 (stderr: )". The error must
// now surface that stdout AND give an actionable claude login hint.
func TestSubagentRunError_DetectsLoggedOut(t *testing.T) {
	err := subagentRunError("claude", errors.New("exit status 1"),
		"Not logged in · Please run /login", "", false)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "not logged in") {
		t.Errorf("want login guidance, got: %s", msg)
	}
	if !strings.Contains(msg, "claude") || !strings.Contains(msg, "ANTHROPIC_API_KEY") {
		t.Errorf("want actionable claude hint, got: %s", msg)
	}
	if !strings.Contains(msg, "Please run /login") {
		t.Errorf("swallowed stdout must now be surfaced, got: %s", msg)
	}
	if strings.Contains(msg, "(stderr: )") {
		t.Errorf("must not regress to the opaque empty-stderr form: %s", msg)
	}
}

// TestSubagentRunError_SurfacesStdoutOnGenericFailure confirms a
// non-auth failure still surfaces whatever the CLI wrote to stdout.
func TestSubagentRunError_SurfacesStdoutOnGenericFailure(t *testing.T) {
	err := subagentRunError("claude", errors.New("exit status 2"),
		"panic: boom on stdout", "", false)
	if !strings.Contains(err.Error(), "boom on stdout") {
		t.Errorf("stdout should be surfaced in error, got: %s", err.Error())
	}
}

// TestSubagentRunError_CustomCLINoClaudeHint verifies a non-claude CLI
// gets a generic "authenticate it" message, never claude-specific advice
// that would be wrong for it — the seam the multi-CLI spec builds on.
func TestSubagentRunError_CustomCLINoClaudeHint(t *testing.T) {
	err := subagentRunError("codex", errors.New("exit status 1"),
		"", "Error: unauthorized", false)
	msg := err.Error()
	if !strings.Contains(msg, `"codex"`) {
		t.Errorf("want command name in message, got: %s", msg)
	}
	if strings.Contains(msg, "ANTHROPIC_API_KEY") || strings.Contains(msg, "/login") {
		t.Errorf("must not give claude-specific hint for codex: %s", msg)
	}
}

// TestFilterConventionsBySurface checks the surface filter is
// case-insensitive and matches when any one Surface tag matches.
func TestFilterConventionsBySurface(t *testing.T) {
	entries := []contractpeering.ConventionEntry{
		{Slug: "a", Surface: []string{"http-response"}},
		{Slug: "b", Surface: []string{"sse-event", "HTTP-Response"}},
		{Slug: "c", Surface: []string{"internal"}},
	}
	got := FilterConventionsBySurface(entries, "http-response")
	if len(got) != 2 {
		t.Fatalf("want 2, got %d (%v)", len(got), got)
	}
	if got[0].Slug != "a" || got[1].Slug != "b" {
		t.Errorf("unexpected entries: %v", got)
	}
	all := FilterConventionsBySurface(entries, "")
	if len(all) != 3 {
		t.Errorf("empty surface should return all, got %d", len(all))
	}
}
