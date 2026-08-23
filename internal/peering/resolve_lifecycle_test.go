package peering

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/attention/mailthread"
	contractpeering "github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

func TestAuthoritativeHandoffTerminalStatusesResolveExactMailThread(t *testing.T) {
	tests := []struct {
		status  string
		outcome string
	}{
		{status: "completed", outcome: mailthread.OutcomeCompleted},
		{status: "rejected", outcome: mailthread.OutcomeRejected},
		{status: "superseded", outcome: mailthread.OutcomeCancelled},
	}
	for _, test := range tests {
		t.Run(test.status, func(t *testing.T) {
			origin, peer, state := peerMailFixture(t)
			originCfg, _ := config.Load(origin)
			peerCfg, _ := config.Load(peer)
			originSvc, err := projectMailService(origin, state, originCfg)
			if err != nil {
				t.Fatal(err)
			}
			peerSvc, err := projectMailService(peer, state, peerCfg)
			if err != nil {
				t.Fatal(err)
			}
			delivery, err := originSvc.Send(mail.SendRequest{RecipientAlias: "app", Subject: "work", Body: "typed transfer", Kind: "peer.work_transfer", IdempotencyKey: "handoff-" + test.status})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := peerSvc.Reply(mail.ReplyRequest{MessageID: delivery.MessageID, Body: "accepted", Kind: "peer.work_transfer.response", IdempotencyKey: "accept-" + test.status}); err != nil {
				t.Fatal(err)
			}

			originSpec := `---
title: Origin work
slug: origin-work
type: feature
status: awaiting_peer
---
# Origin work
`
			originSpec = AppendTrailToContent(originSpec, contractpeering.TrailEntry{
				At: time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC), Direction: contractpeering.DirectionOut,
				PeerAliasDisplay: "app", PeerID: peerCfg.PeerID, Mode: contractpeering.ModeAsyncDrop,
				OriginatingSpec: "origin-work", PeerSpec: "app/peer-work", ThreadID: delivery.ThreadID,
			})
			writeLifecycleSpec(t, filepath.Join(origin, ".hero", "planning", "features", "origin-work", "spec.md"), originSpec)
			peerSpec := "---\ntitle: Peer work\nslug: peer-work\ntype: feature\nstatus: " + test.status + "\n---\n# Peer work\n"
			writeLifecycleSpec(t, filepath.Join(peer, ".hero", "planning", "features", "peer-work", "spec.md"), peerSpec)

			originalNow := nowFn
			nowFn = func() time.Time { return time.Date(2027, 8, 22, 11, 0, 0, 0, time.UTC) }
			t.Cleanup(func() { nowFn = originalNow })
			transitioned, err := reconcileAwaitingPeer(origin, state, t.Logf)
			if err != nil || len(transitioned) != 1 || transitioned[0] != "origin-work" {
				t.Fatalf("handoff reconciliation = %#v, %v", transitioned, err)
			}
			parsed, err := spec.ParseFile(filepath.Join(origin, ".hero", "planning", "features", "origin-work", "spec.md"))
			if err != nil || parsed.Status != spec.StatusHandedBack {
				t.Fatalf("origin status = %#v, %v", parsed, err)
			}
			thread, _, err := originSvc.Thread(originCfg.PeerID, delivery.ThreadID)
			if err != nil || thread.State.Lifecycle != mailthread.LifecycleResolved || thread.State.Resolution == nil || thread.State.Resolution.Outcome != test.outcome || thread.State.Resolution.SourceID != "app/peer-work" {
				t.Fatalf("linked lifecycle %s = %#v, %v", test.status, thread, err)
			}
		})
	}
}

func writeLifecycleSpec(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
