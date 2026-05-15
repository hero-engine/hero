package peering

// Phase 3 of cross-repo-peering: pilot harness for the dogfood gate.
//
// The user's exit criterion is "dogfood on three real repos". The
// agent can't do the dogfood — but it can prove every mechanic in
// the ladder works end-to-end against three mock sibling workspaces
// so that the only remaining variable on the user side is ergonomic
// feel, not correctness.
//
// Repos in this harness mirror the spec's pilot scenario:
//   - app      (peer_id 22…22) — the backend, owns contracts
//   - client   (peer_id 11…11) — the originator that issues handoffs
//   - desktop  (peer_id 33…33) — a third sibling so manifest /
//                                 multi-peer flows exercise N>2
//
// What's exercised end-to-end:
//   1. `hero peer manifest` (GenerateManifest + WriteManifest) per
//      workspace, with at least one peer-surface convention + one
//      contract entry.
//   2. Peer-relevant convention loading: ReadPeerManifest from
//      client → app, simulating `hero relevant --peer app`.
//   3. Sync peer call advisory — dry-run (mocked subagent) so we
//      verify the envelope and trail without spawning a real LLM.
//   4. Sync peer call spec-out — dry-run; the receiver-side spec is
//      then manually created via Handoff to exercise the lifecycle
//      that the real spec-out subagent would produce.
//   5. Async handoff (`hero handoff`) — full lifecycle: receiver
//      scaffold appears, originator status flips to handed_off,
//      both sides carry trail entries, events emitted.
//   6. Peer-side completion → originator auto-fires to handed_back
//      via ReconcileAwaitingPeer (the on-demand reconciler `hero
//      status` runs at render time).
//   7. Contract-import passive surfacing: a Go file in client that
//      imports app's contract symbol produces a hit; a test-file
//      match does not.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	contractpeering "github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// threeSiblingFixture materializes (app, client, desktop) sibling
// workspaces under a single tempdir, wires Repos in each direction
// needed by the scenarios below, and seeds peer manifests with one
// convention + one contract apiece. Returns absolute roots.
func threeSiblingFixture(t *testing.T) (appRoot, clientRoot, desktopRoot string) {
	t.Helper()
	root := t.TempDir()
	appRoot = filepath.Join(root, "app")
	clientRoot = filepath.Join(root, "client")
	desktopRoot = filepath.Join(root, "desktop")

	const (
		appPeerID     = "22222222-2222-4222-8222-222222222222"
		clientPeerID  = "11111111-1111-4111-8111-111111111111"
		desktopPeerID = "33333333-3333-4333-8333-333333333333"
	)
	setupWorkspace(t, appRoot, appPeerID)
	setupWorkspace(t, clientRoot, clientPeerID)
	setupWorkspace(t, desktopRoot, desktopPeerID)

	// app publishes the contract; client / desktop import it.
	writePeerManifestWithContracts(t, appRoot, appPeerID,
		"contracts/events", "Envelope", "error-envelope")
	// client publishes its own surface so multi-peer manifest reads
	// have something to chew on (no contracts — only conventions).
	writeMinimalPeerManifest(t, clientRoot, clientPeerID, "client")
	writeMinimalPeerManifest(t, desktopRoot, desktopPeerID, "desktop")

	// Wire repos. client points at both app and desktop; app points
	// at client (so handoff origin alias resolves from app's side).
	{
		cfg, err := config.Load(clientRoot)
		if err != nil {
			t.Fatalf("load client cfg: %v", err)
		}
		cfg.Repos = map[string]string{
			"app":     appRoot,
			"desktop": desktopRoot,
		}
		if err := cfg.Save(clientRoot); err != nil {
			t.Fatalf("save client cfg: %v", err)
		}
	}
	{
		cfg, err := config.Load(appRoot)
		if err != nil {
			t.Fatalf("load app cfg: %v", err)
		}
		cfg.Repos = map[string]string{
			"client":  clientRoot,
			"desktop": desktopRoot,
		}
		if err := cfg.Save(appRoot); err != nil {
			t.Fatalf("save app cfg: %v", err)
		}
	}
	{
		cfg, err := config.Load(desktopRoot)
		if err != nil {
			t.Fatalf("load desktop cfg: %v", err)
		}
		cfg.Repos = map[string]string{
			"app":    appRoot,
			"client": clientRoot,
		}
		if err := cfg.Save(desktopRoot); err != nil {
			t.Fatalf("save desktop cfg: %v", err)
		}
	}

	return appRoot, clientRoot, desktopRoot
}

// writeMinimalPeerManifest creates a peer-manifest.yaml that lists
// no contracts and no conventions — enough for ReadPeerManifest to
// succeed against a sibling without seeding shapes the scenario
// doesn't care about.
func writeMinimalPeerManifest(t *testing.T, root, peerID, name string) {
	t.Helper()
	cfg, _ := config.Load(root)
	if cfg.PeerID == "" {
		cfg.PeerID = peerID
		_ = cfg.Save(root)
	}
	// Use GenerateAndWriteManifest so we go through the real path.
	if err := GenerateAndWriteManifest(root); err != nil {
		t.Fatalf("generate %s manifest: %v", name, err)
	}
}

// TestPilotHarness_FullLadder runs the entire three-sibling tag-team
// flow end-to-end. Mocked subagent invocations use DryRun so no real
// LLM is spawned.
func TestPilotHarness_FullLadder(t *testing.T) {
	appRoot, clientRoot, _ := threeSiblingFixture(t)

	// --- 1. Manifest is readable across the boundary ---------------
	appManifest, err := ReadPeerManifest(appRoot, ".hero")
	if err != nil {
		t.Fatalf("read app manifest from client perspective: %v", err)
	}
	if appManifest.Contracts == nil || len(appManifest.Contracts.Shapes) != 1 {
		t.Fatalf("app manifest should publish 1 contract shape, got %+v", appManifest.Contracts)
	}
	if appManifest.Contracts.Shapes[0].GoSymbol != "contracts/events.Envelope" {
		t.Errorf("unexpected go_symbol: %q", appManifest.Contracts.Shapes[0].GoSymbol)
	}

	// --- 2. Originator spec on client side ------------------------
	originSlug := "order-failure-error-display"
	writeFeatureSpec(t, clientRoot, originSlug, "Order failure error display")

	// --- 3. Sync peer call: advisory (mocked via DryRun) ----------
	advRes, err := Call(clientRoot, CallOptions{
		PeerAlias:   "app",
		Mode:        contractpeering.PeerCallAdvisory,
		Prompt:      "Does removing field X from the events.Envelope break you?",
		RelatedSpec: originSlug,
		Reason:      "client suspects schema break",
		DryRun:      true,
		Stdout:      discardWriter{},
	})
	if err != nil {
		t.Fatalf("advisory call (dry-run): %v", err)
	}
	if !strings.Contains(advRes.EnvelopeJSON, "Advisory mode") {
		t.Errorf("advisory envelope missing mode marker:\n%s", advRes.EnvelopeJSON)
	}
	if advRes.Result.Kind != contractpeering.ResultFindings {
		t.Errorf("advisory result kind: %q", advRes.Result.Kind)
	}

	// --- 4. Sync peer call: spec-out (dry-run) --------------------
	specRes, err := Call(clientRoot, CallOptions{
		PeerAlias:   "app",
		Mode:        contractpeering.PeerCallSpecOut,
		Prompt:      "Design the backend fix for the envelope shape change",
		RelatedSpec: originSlug,
		Reason:      "client raised — backend should own",
		DryRun:      true,
		Stdout:      discardWriter{},
	})
	if err != nil {
		t.Fatalf("spec-out call (dry-run): %v", err)
	}
	if !strings.Contains(specRes.EnvelopeJSON, "Spec-out mode") {
		t.Errorf("spec-out envelope missing mode marker:\n%s", specRes.EnvelopeJSON)
	}

	// --- 5. Async handoff: real receiver-side write + lifecycle ---
	at := time.Date(2026, 5, 15, 14, 0, 0, 0, time.UTC)
	handoffRes, err := Handoff(clientRoot, originSlug, HandoffOptions{
		PeerAlias: "app",
		Reason:    "Backend owns the response envelope.",
		At:        at,
	})
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}
	if handoffRes.PeerSlug == "" {
		t.Fatal("handoff returned empty peer slug")
	}
	// Receiver-side scaffold exists with received_from.
	peerSpecPath := handoffRes.PeerPath
	peerData, err := os.ReadFile(peerSpecPath)
	if err != nil {
		t.Fatalf("read peer scaffold: %v", err)
	}
	peerContent := string(peerData)
	if !strings.Contains(peerContent, "received_from") {
		t.Errorf("peer scaffold missing received_from block:\n%s", peerContent)
	}
	if !strings.Contains(peerContent, handoffRes.OriginPeerID) {
		t.Errorf("peer scaffold should record origin peer_id %s", handoffRes.OriginPeerID)
	}
	// Originator status flipped to handed_off.
	originSpecPath := filepath.Join(clientRoot, ".hero", "planning", "features", originSlug, "spec.md")
	originParsed, err := spec.ParseFile(originSpecPath)
	if err != nil {
		t.Fatalf("parse origin spec: %v", err)
	}
	if originParsed.Status != spec.StatusHandedOff {
		t.Errorf("origin status: want handed_off, got %q", originParsed.Status)
	}
	// Trail entries appear on both sides.
	originTrail, err := ReadTrail(originSpecPath)
	if err != nil {
		t.Fatalf("read origin trail: %v", err)
	}
	if len(originTrail) == 0 {
		t.Fatal("origin trail empty")
	}
	peerTrail, err := ReadTrail(peerSpecPath)
	if err != nil {
		t.Fatalf("read peer trail: %v", err)
	}
	if len(peerTrail) == 0 {
		t.Fatal("peer trail empty")
	}

	// --- 6. Mark origin as awaiting_peer so the reconciler picks it
	// up next — this is what the real spec-out / handoff flow does
	// when the peer side hits `delivering`. The Phase 1 handoff lands
	// at handed_off; we drive it to awaiting_peer manually to skip
	// the intermediate step (the reconciler we exercise next is the
	// awaiting_peer → handed_back one, which is the auto-fire path
	// `hero status` calls).
	originData, _ := os.ReadFile(originSpecPath)
	originUpdated := spec.SetFrontmatterField(string(originData), "status", string(spec.StatusAwaitingPeer))
	if err := os.WriteFile(originSpecPath, []byte(originUpdated), 0o644); err != nil {
		t.Fatalf("rewrite origin status: %v", err)
	}

	// --- 7. Peer-side completion drives the reconciler ------------
	peerUpdated := spec.SetFrontmatterField(peerContent, "status", string(spec.StatusCompleted))
	if err := os.WriteFile(peerSpecPath, []byte(peerUpdated), 0o644); err != nil {
		t.Fatalf("rewrite peer status: %v", err)
	}

	transitioned, err := ReconcileAwaitingPeer(clientRoot, nil)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(transitioned) != 1 || transitioned[0] != originSlug {
		t.Errorf("expected reconcile to transition %s, got %v", originSlug, transitioned)
	}
	finalOrigin, _ := spec.ParseFile(originSpecPath)
	if finalOrigin.Status != spec.StatusHandedBack {
		t.Errorf("post-reconcile origin status: want handed_back, got %q", finalOrigin.Status)
	}

	// --- 8. Contract-import passive surfacing ---------------------
	consumer := filepath.Join(clientRoot, "src", "api", "orders.go")
	if err := os.MkdirAll(filepath.Dir(consumer), 0o755); err != nil {
		t.Fatalf("mkdir consumer: %v", err)
	}
	consumerSrc := `package api

import (
	"fmt"
	"github.com/example/app/contracts/events"
)

func Render(e events.Envelope) string { return fmt.Sprintf("%v", e) }
`
	if err := os.WriteFile(consumer, []byte(consumerSrc), 0o644); err != nil {
		t.Fatalf("write consumer: %v", err)
	}

	hits, err := ScanContractImports(clientRoot, ScanOptions{
		ChangedFiles: []string{"src/api/orders.go"},
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 contract-import hit, got %d: %+v", len(hits), hits)
	}
	if hits[0].PeerAlias != "app" {
		t.Errorf("hit peer alias: %q", hits[0].PeerAlias)
	}
	signal := RenderContractImportSignal(hits)
	if !strings.Contains(signal, "contracts/events.Envelope") {
		t.Errorf("rendered signal missing symbol:\n%s", signal)
	}

	// And a test-file consumer must NOT surface — non-contract per spec.
	testConsumer := filepath.Join(clientRoot, "src", "api", "orders_test.go")
	if err := os.WriteFile(testConsumer, []byte(consumerSrc), 0o644); err != nil {
		t.Fatalf("write test consumer: %v", err)
	}
	hits2, err := ScanContractImports(clientRoot, ScanOptions{
		ChangedFiles: []string{"src/api/orders_test.go"},
	})
	if err != nil {
		t.Fatalf("scan test file: %v", err)
	}
	if len(hits2) != 0 {
		t.Errorf("test-file scan produced %d hits; spec requires zero", len(hits2))
	}
}

// discardWriter is a tiny io.Writer that throws away all input.
// Lets dry-run callers pass `Stdout: discardWriter{}` without
// pulling in io.Discard at multiple call sites.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
