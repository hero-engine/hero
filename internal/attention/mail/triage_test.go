package mail

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/traversal"
)

func triageService(t *testing.T, id string) (*Service, *Store, attention.MailEnvelope, string) {
	t.Helper()
	state := t.TempDir()
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(state)
	if err != nil {
		t.Fatal(err)
	}
	env := testEnvelope(id)
	env.Body = "Please investigate this request."
	env.Subject = "Investigate request"
	if _, _, err := store.Deliver(env, testDelivery(env)); err != nil {
		t.Fatal(err)
	}
	service := NewService(store, project, config.Config{Folder: ".hero", PeerID: "peer_b", Domain: "engineering"})
	service.now = func() time.Time { return time.Date(2026, 7, 22, 19, 0, 0, 0, time.UTC) }
	focusStore, _ := focus.NewStore(state)
	service.focus = focus.NewService(focusStore, nil)
	return service, store, env, project
}

func TestTriageReceiptActionsAreRevisionedIdempotentAndOrthogonal(t *testing.T) {
	service, store, env, _ := triageService(t, "mail_triage")
	before, _ := store.Get("peer_b", env.ID)
	read, err := service.Action(ActionRequest{MessageID: env.ID, Action: ActionRead, ExpectedRevision: 0, IdempotencyKey: "read-1"})
	if err != nil || read.Receipt.ReadAt == "" || read.Receipt.Revision == 0 {
		t.Fatalf("read = %#v, %v", read, err)
	}
	replay, err := service.Action(ActionRequest{MessageID: env.ID, Action: ActionRead, ExpectedRevision: 0, IdempotencyKey: "read-1"})
	if err != nil || replay.Receipt.Revision != read.Receipt.Revision {
		t.Fatalf("read replay = %#v, %v", replay, err)
	}
	if _, err := service.Action(ActionRequest{MessageID: env.ID, Action: ActionDismiss, ExpectedRevision: 0, IdempotencyKey: "dismiss-1"}); !errors.Is(err, ErrStale) {
		t.Fatalf("stale error = %v", err)
	}
	dismissed, err := service.Action(ActionRequest{MessageID: env.ID, Action: ActionDismiss, ExpectedRevision: read.Receipt.Revision, IdempotencyKey: "dismiss-1"})
	if err != nil || dismissed.Receipt.DismissedAt == "" || dismissed.Receipt.ReadAt == "" || dismissed.Receipt.AcknowledgedAt != "" {
		t.Fatalf("dismiss = %#v, %v", dismissed, err)
	}
	items, _ := service.Inbox("", false)
	if len(items) != 0 {
		t.Fatalf("dismissed message remained active: %#v", items)
	}
	after, _ := store.Get("peer_b", env.ID)
	if before.Body != after.Body || before.Revision != after.Revision {
		t.Fatal("triage mutated immutable envelope")
	}
}

func TestAddToTodayIsIdempotentAndReadsPriorInbound(t *testing.T) {
	service, _, env, _ := triageService(t, "mail_today")
	first, err := service.Action(ActionRequest{MessageID: env.ID, Action: ActionAddToToday, ExpectedRevision: 0, IdempotencyKey: "today-1"})
	if err != nil || first.FocusItemID == "" || first.Receipt.ReadAt == "" || first.Receipt.DismissedAt != "" || first.Receipt.PromotedArtifact != nil {
		t.Fatalf("today = %#v, %v", first, err)
	}
	replay, err := service.Action(ActionRequest{MessageID: env.ID, Action: ActionAddToToday, ExpectedRevision: 0, IdempotencyKey: "today-1"})
	if err != nil || replay.FocusItemID != first.FocusItemID {
		t.Fatalf("today replay = %#v, %v", replay, err)
	}
}

func TestExactReplayCompletesPendingFeedEmission(t *testing.T) {
	service, store, env, project := triageService(t, "mail_event_retry")
	logPath := filepath.Join(project, ".hero", "events.log")
	if err := os.Mkdir(logPath, 0o755); err != nil {
		t.Fatal(err)
	}
	req := ActionRequest{MessageID: env.ID, Action: ActionRead, ExpectedRevision: 0, IdempotencyKey: "read-event"}
	if _, err := service.Action(req); err == nil {
		t.Fatal("expected feed append failure")
	}
	pending, err := store.Receipt("peer_b", env.ID)
	if err != nil || len(pending.Actions) != 1 || pending.Actions[0].EventEmitted {
		t.Fatalf("pending action = %#v, %v", pending, err)
	}
	if err := os.Remove(logPath); err != nil {
		t.Fatal(err)
	}
	replayed, err := service.Action(req)
	if err != nil || len(replayed.Receipt.Actions) != 1 || !replayed.Receipt.Actions[0].EventEmitted {
		t.Fatalf("replay = %#v, %v", replayed, err)
	}
	content, _ := os.ReadFile(logPath)
	if strings.Count(string(content), "mail.read") != 1 {
		t.Fatalf("event not recovered exactly once: %s", content)
	}
}

func TestUnreadSummaryIsBoundedOldestFirst(t *testing.T) {
	service, store, _, _ := triageService(t, "mail_seed")
	for i := 0; i < 6; i++ {
		env := testEnvelope(fmt.Sprintf("mail_summary_%d", i))
		env.CreatedAt = fmt.Sprintf("2026-07-22T18:%02d:00Z", i)
		env.IdempotencyKey = env.ID
		if _, _, err := store.Deliver(env, testDelivery(env)); err != nil {
			t.Fatal(err)
		}
	}
	summary, err := service.UnreadSummary(5)
	if err != nil || summary.Count != 7 || len(summary.Items) != 5 {
		t.Fatalf("summary = %#v, %v", summary, err)
	}
	if summary.Items[0].ID != "mail_seed" || summary.Items[1].ID != "mail_summary_0" {
		t.Fatalf("not oldest-first: %#v", summary.Items)
	}
}

func TestPromotionSlugRetainsUniquenessWhenLongIDsSharePrefix(t *testing.T) {
	prefix := "mail_" + strings.Repeat("a", 100)
	first := promotionSlug(prefix + "one")
	second := promotionSlug(prefix + "two")
	if first == second || len(first) > 72 || len(second) > 72 {
		t.Fatalf("unsafe promotion slugs: %q %q", first, second)
	}
}

func TestPromotionResumesAfterEveryStepAndWritesBodyFreeProvenance(t *testing.T) {
	for _, step := range []string{"reserve", "capture", "promote", "provenance", "receipt"} {
		t.Run(step, func(t *testing.T) {
			service, store, env, project := triageService(t, "mail_"+step)
			failed := false
			service.SetFailureInjector(func(got string) error {
				if got == step && !failed {
					failed = true
					return errors.New("injected " + step)
				}
				return nil
			})
			req := ActionRequest{MessageID: env.ID, Action: ActionPromote, ArtifactType: "feature", ExpectedRevision: 0, IdempotencyKey: "promote-1"}
			if _, err := service.Action(req); err == nil {
				t.Fatalf("expected injected %s failure", step)
			}
			service.SetFailureInjector(nil)
			result, err := service.Action(req)
			if err != nil || result.Artifact == nil || result.Artifact.Slug == "" || result.Navigation == nil || result.MessageID != env.ID || result.ThreadID != env.ThreadID {
				t.Fatalf("retry = %#v, %v", result, err)
			}
			intakeDir := filepath.Join(project, ".hero", "planning", "intake")
			entries, _ := os.ReadDir(intakeDir)
			if len(entries) != 1 {
				t.Fatalf("intakes = %d, want 1", len(entries))
			}
			featureDir := filepath.Join(project, ".hero", "planning", "features")
			features, _ := os.ReadDir(featureDir)
			if len(features) != 1 {
				t.Fatalf("features = %d, want 1", len(features))
			}
			r, _ := store.Receipt("peer_b", env.ID)
			if r.PromotedArtifact == nil || r.PromotedArtifact.Slug != result.Artifact.Slug {
				t.Fatalf("receipt artifact = %#v", r)
			}
			graphBytes, _ := os.ReadFile(filepath.Join(project, ".hero", "events.log"))
			if !strings.Contains(string(graphBytes), "mail.promote") {
				t.Fatalf("promotion event missing: %s", graphBytes)
			}
			graphStore, err := graph.Open(filepath.Join(project, ".hero"))
			if err != nil {
				t.Fatal(err)
			}
			defer graphStore.Close()
			node, err := graphStore.GetNode("MailSource", env.ID, "")
			if err != nil {
				t.Fatal(err)
			}
			props := fmt.Sprint(node.Props)
			if strings.Contains(props, env.Body) || !strings.Contains(props, env.ThreadID) {
				t.Fatalf("unsafe/incomplete graph props: %s", props)
			}
			// Spec provenance nodes live in the repo partition every graph
			// reader filters on (gitutil.RepoKey), NOT the peer id. This
			// asserted "peer_b" while writeMailProvenance keyed by
			// cfg.PeerID — a partition `hero why` never queries, so the
			// provenance chain was unreachable in production. Now that node
			// identity is (type, key, repo), a mis-keyed write no longer
			// even clobbers the correctly-keyed one; it just sits in a
			// partition nobody reads.
			trace, err := traversal.Why(graphStore, gitutil.RepoKey(project), result.Artifact.Slug, 4)
			if err != nil {
				t.Fatal(err)
			}
			var derived, mailSource bool
			for _, hop := range trace.Chains {
				derived = derived || hop.EdgeType == "derived_from"
				mailSource = mailSource || hop.EdgeType == "mail_source"
			}
			if !derived || !mailSource {
				t.Fatalf("why chain missing provenance hops: %#v", trace.Chains)
			}
		})
	}
}
