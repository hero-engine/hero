package projection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailthread"
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/attention/suggestion"
	"github.com/hero-engine/hero/internal/projectregistry"
)

// writeRegistryProject registers a minimal Hero project on disk so it can be
// picked up through the machine-global project registry, mirroring the
// fixture mailquery's own tests use.
func writeRegistryProject(tb testing.TB, root, peerID, display string) {
	tb.Helper()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o700); err != nil {
		tb.Fatal(err)
	}
	value := map[string]any{"folder": ".hero", "peer_id": peerID, "peering": map[string]string{"display": display}}
	b, err := json.Marshal(value)
	if err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"), b, 0o600); err != nil {
		tb.Fatal(err)
	}
}

func registrySourceRegistry(projects map[string]string) *projectregistry.Registry {
	entries := make(map[string]*projectregistry.ProjectEntry, len(projects))
	for slug, path := range projects {
		entries[slug] = &projectregistry.ProjectEntry{Path: path}
	}
	return &projectregistry.Registry{Projects: entries}
}

func deliverRegistrySourceMessage(tb testing.TB, store *mail.Store, sender, recipient, id, threadID, kind, createdAt string) {
	tb.Helper()
	envelope := attention.MailEnvelope{
		SchemaVersion: attention.SchemaVersion, ID: id,
		Sender:    attention.ProjectReference{PeerID: sender, DisplayName: sender},
		Recipient: attention.ProjectReference{PeerID: recipient, DisplayName: recipient},
		Subject:   "Subject " + id, Body: "body", Kind: kind, ThreadID: threadID,
		IdempotencyKey: recipient + "-" + id, CreatedAt: createdAt,
	}
	delivery := attention.MailDelivery{
		SchemaVersion: attention.SchemaVersion, MessageID: id, ThreadID: threadID,
		Sender: envelope.Sender, Recipient: envelope.Recipient,
		IdempotencyKey: envelope.IdempotencyKey, DeliveredAt: createdAt,
	}
	if _, _, err := store.Deliver(envelope, delivery); err != nil {
		tb.Fatal(err)
	}
}

// TestRegistryMailSourceThreadsExcludesUninvolvedProjectMail reproduces
// hero-engine/hero#7: an Attention snapshot for one project must never
// surface Mail exchanged privately between two other, unrelated peers that
// merely happen to also be registered on the same machine.
func TestRegistryMailSourceThreadsExcludesUninvolvedProjectMail(t *testing.T) {
	state := t.TempDir()
	projectMail, projectMax, projectMCP := t.TempDir(), t.TempDir(), t.TempDir()
	writeRegistryProject(t, projectMail, "bookwyrm_mail", "bookwyrm-mail")
	writeRegistryProject(t, projectMax, "bookwyrm_max", "bookwyrm-max")
	writeRegistryProject(t, projectMCP, "bookwyrm_mcp", "bookwyrm-mcp")

	store, err := mail.NewStore(state)
	if err != nil {
		t.Fatal(err)
	}
	// A private exchange entirely between bookwyrm-max and bookwyrm-mcp.
	// bookwyrm-mail is neither sender nor recipient on any of these.
	deliverRegistrySourceMessage(t, store, "bookwyrm_mcp", "bookwyrm_max", "mail_1", "peer_thread", attention.MailKindRequest, "2026-08-17T10:00:00Z")
	deliverRegistrySourceMessage(t, store, "bookwyrm_max", "bookwyrm_mcp", "mail_2", "peer_thread_2", attention.MailKindRequest, "2026-08-17T10:01:00Z")

	registry := registrySourceRegistry(map[string]string{
		"bookwyrm-mail": projectMail,
		"bookwyrm-max":  projectMax,
		"bookwyrm-mcp":  projectMCP,
	})

	source, err := NewRegistryMailSource(state, registry, "bookwyrm_mail")
	if err != nil {
		t.Fatal(err)
	}

	// Mirror the exact, unfiltered request Snapshot() issues by default.
	response := source.Threads(mailthread.ThreadListRequest{
		SchemaVersion: mailthread.SchemaVersion, Bucket: mailthread.BucketNeedsAttention, Limit: mailthread.MaxListLimit,
	})
	if response.Error != nil {
		t.Fatalf("unexpected error: %#v", response.Error)
	}
	if len(response.Items) != 0 || response.Counts.Total != 0 {
		t.Fatalf("attention Mail threads for an uninvolved project leaked peer mail: %#v", response)
	}

	// Passing a foreign peer ID explicitly must not widen the view either —
	// this source is pinned to the project it was built for.
	widened := source.Threads(mailthread.ThreadListRequest{
		SchemaVersion: mailthread.SchemaVersion, ProjectPeerID: "bookwyrm_max", Bucket: mailthread.BucketNeedsAttention, Limit: mailthread.MaxListLimit,
	})
	if widened.Error != nil {
		t.Fatalf("unexpected error: %#v", widened.Error)
	}
	if len(widened.Items) != 0 || widened.Counts.Total != 0 {
		t.Fatalf("an explicit foreign project_peer_id widened the Attention Mail view: %#v", widened)
	}
}

// TestSnapshotExcludesUninvolvedProjectMail exercises the same leak through
// the full Attention Service.Snapshot() path used by hero_attention_snapshot,
// hero attention today, and the Serve API.
func TestSnapshotExcludesUninvolvedProjectMail(t *testing.T) {
	state := t.TempDir()
	projectMail, projectMax, projectMCP := t.TempDir(), t.TempDir(), t.TempDir()
	writeRegistryProject(t, projectMail, "bookwyrm_mail", "bookwyrm-mail")
	writeRegistryProject(t, projectMax, "bookwyrm_max", "bookwyrm-max")
	writeRegistryProject(t, projectMCP, "bookwyrm_mcp", "bookwyrm-mcp")

	store, err := mail.NewStore(state)
	if err != nil {
		t.Fatal(err)
	}
	deliverRegistrySourceMessage(t, store, "bookwyrm_mcp", "bookwyrm_max", "mail_1", "peer_thread", attention.MailKindRequest, "2026-08-17T10:00:00Z")
	deliverRegistrySourceMessage(t, store, "bookwyrm_max", "bookwyrm_mcp", "mail_2", "peer_thread_2", attention.MailKindRequest, "2026-08-17T10:01:00Z")

	registry := registrySourceRegistry(map[string]string{
		"bookwyrm-mail": projectMail,
		"bookwyrm-max":  projectMax,
		"bookwyrm-mcp":  projectMCP,
	})

	mailSource, err := NewRegistryMailSource(state, registry, "bookwyrm_mail")
	if err != nil {
		t.Fatal(err)
	}
	focusStore, err := focus.NewStore(state)
	if err != nil {
		t.Fatal(err)
	}
	resolver := focus.NewRegistryResolver(registry)
	focusService := focus.NewService(focusStore, resolver)
	suggestionStore, err := suggestion.NewStore(state)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(mailSource, focusService, suggestion.NewService(suggestionStore, focusService, resolver))

	snapshot, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Counts.Mail != 0 {
		t.Fatalf("bookwyrm-mail's Attention snapshot leaked %d Mail rows exchanged only between other peers: %#v", snapshot.Counts.Mail, snapshot.Rows)
	}
	for _, row := range snapshot.Rows {
		if row.Group == "mail" {
			t.Fatalf("bookwyrm-mail's Attention snapshot leaked a Mail row it is not sender or recipient on: %#v", row)
		}
	}
}
