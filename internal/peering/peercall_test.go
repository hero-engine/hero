package peering

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/attention/mailthread"
	contractpeering "github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/config"
)

// AC-1, AC-3, AC-9
func TestCallDeliversTypedMailWithoutReceiverTreeWrite(t *testing.T) {
	origin, peer, state := peerMailFixture(t)
	before := treeSnapshot(t, peer)
	res, err := Call(origin, CallOptions{
		PeerAlias: "app", Mode: contractpeering.PeerCallAdvisory,
		Prompt: "What is the envelope?", StateRoot: state, IdempotencyKey: "call-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.MessageID == "" || res.ThreadID == "" || res.Status != "queued" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if after := treeSnapshot(t, peer); after != before {
		t.Fatalf("peer checkout mutated by send\nbefore=%s\nafter=%s", before, after)
	}
	peerCfg, _ := config.Load(peer)
	svc, err := projectMailService(peer, state, peerCfg)
	if err != nil {
		t.Fatal(err)
	}
	item, err := svc.Show(res.MessageID, false)
	if err != nil {
		t.Fatal(err)
	}
	if item.Kind != "peer.advisory" || !strings.Contains(item.Body, "What is the envelope?") {
		t.Fatalf("wrong envelope: %+v", item)
	}
}

// AC-2, AC-5
func TestCallIdempotentAndWaitsForSameThreadReply(t *testing.T) {
	origin, peer, state := peerMailFixture(t)
	first, err := Call(origin, CallOptions{PeerAlias: "app", Mode: contractpeering.PeerCallSpecOut, Prompt: "design it", StateRoot: state, IdempotencyKey: "same"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Call(origin, CallOptions{PeerAlias: "app", Mode: contractpeering.PeerCallSpecOut, Prompt: "design it", StateRoot: state, IdempotencyKey: "same"})
	if err != nil {
		t.Fatal(err)
	}
	if first.MessageID != second.MessageID {
		t.Fatalf("replay duplicated mail: %s != %s", first.MessageID, second.MessageID)
	}
	peerCfg, _ := config.Load(peer)
	peerSvc, _ := projectMailService(peer, state, peerCfg)
	replied := make(chan struct{})
	go func() {
		defer close(replied)
		time.Sleep(20 * time.Millisecond)
		_, _ = peerSvc.Reply(mail.ReplyRequest{MessageID: first.MessageID, Body: "done", IdempotencyKey: "reply-1"})
	}()
	waited, err := Call(origin, CallOptions{
		PeerAlias: "app", Mode: contractpeering.PeerCallSpecOut, Prompt: "design it",
		StateRoot: state, IdempotencyKey: "same", Wait: time.Second, PollInterval: 5 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if waited.Status != "responded" || waited.Response == nil || waited.Response.Body != "done" {
		t.Fatalf("did not return response: %+v", waited)
	}
	<-replied
	originCfg, _ := config.Load(origin)
	originSvc, _ := projectMailService(origin, state, originCfg)
	thread, _, err := originSvc.Thread(originCfg.PeerID, first.ThreadID)
	if err != nil || thread.State.Lifecycle != mailthread.LifecycleResolved || thread.State.Resolution == nil || thread.State.Resolution.Source != "peer.spec_out" {
		t.Fatalf("typed spec-out response did not resolve requester thread: %#v, %v", thread, err)
	}
}

// AC-2
func TestCallTimeoutIsStructuredPending(t *testing.T) {
	origin, _, state := peerMailFixture(t)
	res, err := Call(origin, CallOptions{
		PeerAlias: "app", Mode: contractpeering.PeerCallAdvisory, Prompt: "anyone?",
		StateRoot: state, Wait: 15 * time.Millisecond, PollInterval: 2 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "pending" || res.Response != nil {
		t.Fatalf("want pending timeout, got %+v", res)
	}
}

// AC-8
func TestDeprecatedSubagentConfigIsIgnored(t *testing.T) {
	origin, _, state := peerMailFixture(t)
	cfg, _ := config.Load(origin)
	cfg.Peering = &config.PeeringConfig{Subagent: &config.SubagentConfig{Command: filepath.Join(t.TempDir(), "must-not-run")}}
	if err := cfg.Save(origin); err != nil {
		t.Fatal(err)
	}
	res, err := Call(origin, CallOptions{PeerAlias: "app", Mode: contractpeering.PeerCallAdvisory, Prompt: "hi", StateRoot: state})
	if err != nil {
		t.Fatal(err)
	}
	if !res.DeprecatedConfigIgnored || deprecatedSubagentWarning(cfg) == "" {
		t.Fatalf("deprecated config not surfaced: %+v", res)
	}
}

func peerMailFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	origin := filepath.Join(root, "client")
	peer := filepath.Join(root, "app")
	setupWorkspace(t, origin, "11111111-1111-4111-8111-111111111111")
	setupWorkspace(t, peer, "22222222-2222-4222-8222-222222222222")
	writeMinimalPeerManifest(t, origin, "11111111-1111-4111-8111-111111111111", "client")
	writeMinimalPeerManifest(t, peer, "22222222-2222-4222-8222-222222222222", "app")
	originCfg, _ := config.Load(origin)
	originCfg.Repos = map[string]string{"app": peer}
	originCfg.RepoMeta = map[string]config.RepoMetaEntry{"app": {PeerID: "22222222-2222-4222-8222-222222222222"}}
	if err := originCfg.Save(origin); err != nil {
		t.Fatal(err)
	}
	peerCfg, _ := config.Load(peer)
	peerCfg.Repos = map[string]string{"client": origin}
	peerCfg.RepoMeta = map[string]config.RepoMetaEntry{"client": {PeerID: "11111111-1111-4111-8111-111111111111"}}
	if err := peerCfg.Save(peer); err != nil {
		t.Fatal(err)
	}
	return origin, peer, filepath.Join(root, "attention-state")
}

func treeSnapshot(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(root, path)
			data, _ := os.ReadFile(path)
			lines = append(lines, rel+":"+string(data))
		}
		return nil
	})
	return strings.Join(lines, "\n")
}
