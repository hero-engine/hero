package mail

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/attention/mailthread"
	"github.com/hero-engine/hero/internal/config"
)

func TestTypedTerminalEventsAreRevisionedIdempotentAndClosedWorld(t *testing.T) {
	sender, store, _, b := testService(t)
	delivery, err := sender.Send(SendRequest{RecipientAlias: "b", Subject: "question", Body: "content is inert", IdempotencyKey: "terminal-root"})
	if err != nil {
		t.Fatal(err)
	}
	receiver := NewService(store, b, config.Config{Folder: ".hero", PeerID: "peer_b"})
	view, _, err := receiver.Thread("peer_b", delivery.ThreadID)
	if err != nil {
		t.Fatal(err)
	}
	event := mailthread.Event{
		SchemaVersion: mailthread.SchemaVersion,
		Identity:      view.State.Identity, Kind: mailthread.EventAdvisoryTerminal,
		EventID: "peer-call:call-1:completed", ExpectedRevision: view.State.Revision,
		OccurredAt: "2026-07-22T19:00:00Z", Source: "peer.advisory", SourceID: "call-1",
		Outcome: mailthread.OutcomeAnswered,
	}
	resolved, err := receiver.ThreadEvent(event)
	if err != nil || resolved.Thread.State.Lifecycle != mailthread.LifecycleResolved || resolved.Thread.State.GraceClass != mailthread.GraceInformational {
		t.Fatalf("advisory terminal = %#v, %v", resolved, err)
	}
	if resolved.Thread.State.Resolution.Source != "peer.advisory" || resolved.Thread.State.Resolution.SourceID != "call-1" || resolved.Thread.State.ArchiveEligibleAt != "2026-07-29T19:00:00Z" {
		t.Fatalf("resolution provenance = %#v", resolved.Thread.State)
	}
	replay, err := receiver.ThreadEvent(event)
	if err != nil || !replay.Replayed || replay.Thread.State.Revision != resolved.Thread.State.Revision {
		t.Fatalf("event replay = %#v, %v", replay, err)
	}
	conflict := event
	conflict.Outcome = mailthread.OutcomeRejected
	if _, err := receiver.ThreadEvent(conflict); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("event conflict = %v", err)
	}
	stale := event
	stale.EventID = "peer-call:call-2:completed"
	stale.SourceID = "call-2"
	if _, err := receiver.ThreadEvent(stale); !errors.Is(err, ErrStale) {
		t.Fatalf("stale event = %v", err)
	}
	outOfOrder := stale
	outOfOrder.ExpectedRevision = resolved.Thread.State.Revision
	outOfOrder.OccurredAt = "2026-07-22T17:59:59Z"
	if _, err := receiver.ThreadEvent(outOfOrder); !errors.Is(err, ErrEventOutOfOrder) {
		t.Fatalf("out-of-order event = %v", err)
	}
	unregistered := event
	unregistered.EventID = "unregistered"
	unregistered.ExpectedRevision = resolved.Thread.State.Revision
	unregistered.Source = "tracker.guess"
	if _, err := receiver.ThreadEvent(unregistered); err == nil {
		t.Fatal("unregistered typed source was accepted")
	}
}

func TestEventOrderingUsesChronologicalRFC3339NanoTime(t *testing.T) {
	sender, store, _, b := testService(t)
	delivery, err := sender.Send(SendRequest{RecipientAlias: "b", Subject: "ordering", Body: "ignored", IdempotencyKey: "ordering-root"})
	if err != nil {
		t.Fatal(err)
	}
	receiver := NewService(store, b, config.Config{Folder: ".hero", PeerID: "peer_b"})
	view, _, _ := store.Thread("peer_b", delivery.ThreadID)
	first := mailthread.Event{SchemaVersion: 1, Identity: view.State.Identity, Kind: mailthread.EventActionSucceeded, EventID: "ordering-newer", ExpectedRevision: view.State.Revision, OccurredAt: "2026-07-22T19:00:00.1Z", Source: "mail.action", SourceID: delivery.MessageID}
	newer, err := receiver.ThreadEvent(first)
	if err != nil {
		t.Fatal(err)
	}
	older := first
	older.EventID = "ordering-older"
	older.ExpectedRevision = newer.Thread.State.Revision
	older.OccurredAt = "2026-07-22T19:00:00Z"
	if _, err := receiver.ThreadEvent(older); !errors.Is(err, ErrEventOutOfOrder) {
		t.Fatalf("chronologically older fractional timestamp was accepted: %v", err)
	}
}

func TestDeterministicGraceReconciliationUsesExactSevenAndThirtyDayBoundaries(t *testing.T) {
	sender, store, _, b := testService(t)
	receiver := NewService(store, b, config.Config{Folder: ".hero", PeerID: "peer_b"})
	base := time.Date(2026, 7, 22, 19, 0, 0, 0, time.UTC)

	makeThread := func(key string) mailthread.ThreadView {
		delivery, err := sender.Send(SendRequest{RecipientAlias: "b", Subject: key, Body: "ignored", IdempotencyKey: key})
		if err != nil {
			t.Fatal(err)
		}
		view, _, err := store.Thread("peer_b", delivery.ThreadID)
		if err != nil {
			t.Fatal(err)
		}
		return view
	}

	advisory := makeThread("grace-advisory")
	advisoryEvent := mailthread.Event{SchemaVersion: 1, Identity: advisory.State.Identity, Kind: mailthread.EventAdvisoryTerminal, EventID: "advisory-done", ExpectedRevision: advisory.State.Revision, OccurredAt: base.Format(time.RFC3339Nano), Source: "peer.advisory", SourceID: "advisory-1", Outcome: mailthread.OutcomeAnswered}
	advisoryResolved, err := receiver.ThreadEvent(advisoryEvent)
	if err != nil {
		t.Fatal(err)
	}
	before, changed, err := store.ReconcileThread("peer_b", advisory.State.Identity.ThreadID, base.Add(informationalGrace-time.Nanosecond))
	if err != nil || changed || before.State.Lifecycle != mailthread.LifecycleResolved {
		t.Fatalf("seven-day minus = %#v changed=%v err=%v", before, changed, err)
	}
	exact, changed, err := store.ReconcileThread("peer_b", advisory.State.Identity.ThreadID, base.Add(informationalGrace))
	if err != nil || !changed || exact.State.Lifecycle != mailthread.LifecycleArchived || exact.State.ArchiveEligibleAt != advisoryResolved.Thread.State.ArchiveEligibleAt {
		t.Fatalf("seven-day exact = %#v changed=%v err=%v", exact, changed, err)
	}

	linked := makeThread("grace-linked")
	linkedEvent := mailthread.Event{SchemaVersion: 1, Identity: linked.State.Identity, Kind: mailthread.EventLinkedTerminal, EventID: "handoff-done", ExpectedRevision: linked.State.Revision, OccurredAt: base.Format(time.RFC3339Nano), Source: "peer.handoff", SourceID: "handoff-1", Outcome: mailthread.OutcomeCompleted}
	linkedResolved, err := receiver.ThreadEvent(linkedEvent)
	if err != nil || linkedResolved.Thread.State.ArchiveEligibleAt != base.Add(linkedWorkGrace).Format(time.RFC3339Nano) {
		t.Fatalf("linked terminal = %#v, %v", linkedResolved, err)
	}
	minus, changed, err := store.ReconcileThread("peer_b", linked.State.Identity.ThreadID, base.Add(linkedWorkGrace-time.Nanosecond))
	if err != nil || changed || minus.State.Lifecycle != mailthread.LifecycleResolved {
		t.Fatalf("thirty-day minus = %#v changed=%v err=%v", minus, changed, err)
	}
	plus, changed, err := store.ReconcileThread("peer_b", linked.State.Identity.ThreadID, base.Add(linkedWorkGrace+time.Nanosecond))
	if err != nil || !changed || plus.State.Lifecycle != mailthread.LifecycleArchived {
		t.Fatalf("thirty-day plus = %#v changed=%v err=%v", plus, changed, err)
	}

	open := makeThread("open-forever")
	old, changed, err := store.ReconcileThread("peer_b", open.State.Identity.ThreadID, base.Add(10*365*24*time.Hour))
	if err != nil || changed || old.State.Lifecycle != mailthread.LifecycleOpen || old.State.ArchiveEligibleAt != "" {
		t.Fatalf("open thread expired = %#v changed=%v err=%v", old, changed, err)
	}
}

func TestLinkedWorkTerminalOutcomesResolveWithoutContentInference(t *testing.T) {
	sender, store, _, b := testService(t)
	receiver := NewService(store, b, config.Config{Folder: ".hero", PeerID: "peer_b"})
	for index, outcome := range []string{mailthread.OutcomeCompleted, mailthread.OutcomeRejected, mailthread.OutcomeCancelled} {
		delivery, err := sender.Send(SendRequest{RecipientAlias: "b", Subject: "words do not matter", Body: "done maybe perhaps", IdempotencyKey: "linked-outcome-" + outcome})
		if err != nil {
			t.Fatal(err)
		}
		view, _, _ := store.Thread("peer_b", delivery.ThreadID)
		event := mailthread.Event{SchemaVersion: 1, Identity: view.State.Identity, Kind: mailthread.EventLinkedTerminal, EventID: "handoff:" + outcome, ExpectedRevision: view.State.Revision, OccurredAt: time.Date(2026, 7, 22, 20, index, 0, 0, time.UTC).Format(time.RFC3339Nano), Source: "peer.handoff", SourceID: "peer/spec-" + outcome, Outcome: outcome}
		result, err := receiver.ThreadEvent(event)
		if err != nil || result.Thread.State.Lifecycle != mailthread.LifecycleResolved || result.Thread.State.Resolution.Outcome != outcome || result.Thread.State.GraceClass != mailthread.GraceLinkedWork {
			t.Fatalf("linked outcome %q = %#v, %v", outcome, result, err)
		}
	}
}

func TestReplyReadsPriorInboundAndNewInboundReopensWithHistory(t *testing.T) {
	sender, store, a, b := testService(t)
	writePeer(t, a, "peer_a", "A")
	root, err := sender.Send(SendRequest{RecipientAlias: "b", Subject: "request", Body: "please inspect", Kind: "request", IdempotencyKey: "reply-root"})
	if err != nil {
		t.Fatal(err)
	}
	receiver := NewService(store, b, config.Config{Folder: ".hero", PeerID: "peer_b", Repos: map[string]string{"a": a}, RepoMeta: map[string]config.RepoMetaEntry{"a": {PeerID: "peer_a"}}})
	receiver.now = func() time.Time { return time.Date(2026, 7, 22, 18, 1, 0, 0, time.UTC) }
	replyRequest := ReplyRequest{MessageID: root.MessageID, ExpectedThread: root.ThreadID, Body: "answer", IdempotencyKey: "reply-1"}
	reply, err := receiver.ReplyAndReconcile(replyRequest)
	if err != nil || reply.Thread.Read.UnreadCount != 0 || reply.Thread.State.Lifecycle != mailthread.LifecycleOpen {
		t.Fatalf("reply reconciliation = %#v, %v", reply, err)
	}

	receiver.now = func() time.Time { return time.Date(2026, 7, 22, 18, 2, 0, 0, time.UTC) }
	current, _, _ := receiver.Thread("peer_b", root.ThreadID)
	terminal := mailthread.Event{SchemaVersion: 1, Identity: current.State.Identity, Kind: mailthread.EventSpecOutTerminal, EventID: "spec-out:done", ExpectedRevision: current.State.Revision, OccurredAt: receiver.now().Format(time.RFC3339Nano), Source: "peer.spec_out", SourceID: "call-2", Outcome: mailthread.OutcomeCompleted}
	resolved, err := receiver.ThreadEvent(terminal)
	if err != nil || resolved.Thread.State.Lifecycle != mailthread.LifecycleResolved {
		t.Fatalf("resolve before reopen = %#v, %v", resolved, err)
	}

	sender.now = func() time.Time { return time.Date(2026, 7, 22, 18, 3, 0, 0, time.UTC) }
	if _, err := sender.Reply(ReplyRequest{MessageID: reply.Delivery.MessageID, ExpectedThread: root.ThreadID, Body: "follow-up", IdempotencyKey: "reply-2"}); err != nil {
		t.Fatal(err)
	}
	reopened, _, err := store.Thread("peer_b", root.ThreadID)
	if err != nil || reopened.State.Lifecycle != mailthread.LifecycleOpen || reopened.Read.UnreadCount != 1 || reopened.State.Resolution != nil {
		t.Fatalf("inbound reopen = %#v, %v", reopened, err)
	}
	last := reopened.State.Events[len(reopened.State.Events)-1]
	if last.Kind != mailthread.EventInboundActivity || last.FromLifecycle != mailthread.LifecycleResolved || last.ToLifecycle != mailthread.LifecycleOpen {
		t.Fatalf("reopen history = %#v", reopened.State.Events)
	}
	replayed, err := receiver.ReplyAndReconcile(replyRequest)
	if err != nil || replayed.Thread.Read.UnreadCount != 1 {
		t.Fatalf("reply replay read later inbound content: %#v, %v", replayed, err)
	}
}

func TestSemanticActionReplayDoesNotReadLaterInboundContent(t *testing.T) {
	service, store, env, _ := triageService(t, "mail_action_replay_root")
	request := ActionRequest{MessageID: env.ID, Action: ActionAddToToday, ExpectedRevision: 0, IdempotencyKey: "today-replay-snapshot"}
	first, err := service.Action(request)
	if err != nil || first.Receipt.ReadAt == "" {
		t.Fatalf("first semantic action = %#v, %v", first, err)
	}
	late := testEnvelope("mail_action_replay_late")
	late.ThreadID = env.ThreadID
	late.IdempotencyKey = late.ID
	if _, _, err := store.Deliver(late, testDelivery(late)); err != nil {
		t.Fatal(err)
	}
	replay, err := service.Action(request)
	if err != nil || replay.Thread == nil || replay.Thread.Read.UnreadCount != 1 {
		t.Fatalf("semantic action replay read later inbound content: %#v, %v", replay, err)
	}
	if _, err := store.Receipt("peer_b", late.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("later inbound receipt mutated on replay: %v", err)
	}
}

func TestLifecycleActionReadsOnlyMessagesPresentAtInvocation(t *testing.T) {
	service, store, env, _ := triageService(t, "mail_action_reads")
	open, _, _ := store.Thread("peer_b", env.ThreadID)
	request := mailthread.ActionRequest{SchemaVersion: 1, Identity: open.State.Identity, ActionID: mailthread.ActionResolve, ThreadRevision: open.State.Revision, IdempotencyKey: "resolve-and-read", Input: json.RawMessage(`{"reason":"completed","source":"user","outcome":"done"}`)}
	resolved, err := service.ThreadAction(request)
	if err != nil || resolved.State.Lifecycle != mailthread.LifecycleResolved || resolved.Read.UnreadCount != 0 {
		t.Fatalf("lifecycle action reconciliation = %#v, %v", resolved, err)
	}
}
