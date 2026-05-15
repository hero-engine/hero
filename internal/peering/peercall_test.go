package peering

import (
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
  tokens: 1842
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
