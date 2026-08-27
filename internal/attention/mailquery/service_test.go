package mailquery

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailread"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/projectregistry"
)

func writeQueryProject(tb testing.TB, root, peerID, display string, repos map[string]string) {
	tb.Helper()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o700); err != nil {
		tb.Fatal(err)
	}
	value := map[string]any{"folder": ".hero", "peer_id": peerID, "peering": map[string]string{"display": display}}
	if repos != nil {
		value["repos"] = repos
	}
	b, err := json.Marshal(value)
	if err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"), b, 0o600); err != nil {
		tb.Fatal(err)
	}
	manifest := "schema: 1\ncontracts_version: 1\nrepo:\n  peer_id: " + peerID + "\n  name: " + display + "\n  display: " + display + "\ngenerated_at: 2026-08-17T10:00:00Z\n"
	if err := os.WriteFile(filepath.Join(heroDir, "peer-manifest.yaml"), []byte(manifest), 0o600); err != nil {
		tb.Fatal(err)
	}
}

func queryRegistry(projects map[string]string) *projectregistry.Registry {
	entries := make(map[string]*projectregistry.ProjectEntry, len(projects))
	for slug, path := range projects {
		entries[slug] = &projectregistry.ProjectEntry{Path: path}
	}
	return &projectregistry.Registry{Projects: entries}
}

func runQueryGit(tb testing.TB, path string, args ...string) string {
	tb.Helper()
	command := exec.Command("git", append([]string{"-C", path}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		tb.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func initQueryGitRepository(tb testing.TB, root, peerID, display string) {
	tb.Helper()
	if err := os.MkdirAll(root, 0o700); err != nil {
		tb.Fatal(err)
	}
	runQueryGit(tb, root, "init")
	runQueryGit(tb, root, "config", "user.email", "mailquery@example.com")
	runQueryGit(tb, root, "config", "user.name", "Mail Query Tests")
	writeQueryProject(tb, root, peerID, display, nil)
	runQueryGit(tb, root, "add", ".hero")
	runQueryGit(tb, root, "commit", "-m", "seed project identity")
}

func deliverQueryMessage(tb testing.TB, store *mail.Store, recipientPeer, senderPeer, id, thread, createdAt, body string) attention.MailEnvelope {
	tb.Helper()
	envelope := attention.MailEnvelope{
		SchemaVersion: attention.SchemaVersion, ID: id,
		Recipient: attention.ProjectReference{PeerID: recipientPeer, DisplayName: recipientPeer},
		Sender:    attention.ProjectReference{PeerID: senderPeer, DisplayName: senderPeer},
		Subject:   "Subject " + id, Body: body, Kind: attention.MailKindRequest,
		ThreadID: thread, IdempotencyKey: recipientPeer + "_" + id, CreatedAt: createdAt,
	}
	delivery := attention.MailDelivery{
		SchemaVersion: attention.SchemaVersion, MessageID: id, ThreadID: thread,
		Sender: envelope.Sender, Recipient: envelope.Recipient,
		IdempotencyKey: envelope.IdempotencyKey, DeliveredAt: createdAt,
	}
	if _, _, err := store.Deliver(envelope, delivery); err != nil {
		tb.Fatal(err)
	}
	return envelope
}

func TestServiceListSortsFiltersPagesAndInvalidatesCursor(t *testing.T) {
	state := t.TempDir()
	projectA, projectB := t.TempDir(), t.TempDir()
	writeQueryProject(t, projectA, "peer_a", "Alpha", nil)
	writeQueryProject(t, projectB, "peer_b", "Beta", nil)
	store, err := mail.NewStore(state)
	if err != nil {
		t.Fatal(err)
	}
	deliverQueryMessage(t, store, "peer_a", "sender_a", "mail_same", "mail_thread", "2026-08-17T10:00:00Z", "older")
	deliverQueryMessage(t, store, "peer_a", "sender_a", "mail_a", "mail_thread", "2026-08-17T10:00:00.9Z", "new A")
	deliverQueryMessage(t, store, "peer_b", "sender_b", "mail_same", "mail_other", "2026-08-17T10:00:00.9Z", "new B")
	service, err := NewService(state, queryRegistry(map[string]string{"beta": projectB, "alpha": projectA}))
	if err != nil {
		t.Fatal(err)
	}

	page := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, Limit: 2})
	if page.Error != nil || page.TotalCount != 3 || page.UnreadCount != 3 || len(page.Items) != 2 || page.NextCursor == "" {
		t.Fatalf("first page = %#v", page)
	}
	if page.Items[0].MessageID != "mail_a" || page.Items[0].Project.PeerID != "peer_a" || page.Items[1].Project.PeerID != "peer_b" {
		t.Fatalf("parsed ordering = %#v", page.Items)
	}
	for _, item := range page.Items {
		if len(item.Actions) != 6 || item.Actions[0].ID != mailread.ActionMarkRead || item.Actions[5].ID != mailread.ActionReply {
			t.Fatalf("canonical actions = %#v", item.Actions)
		}
		if strings.Contains(string(mustJSON(t, item)), "new A") || strings.Contains(string(mustJSON(t, item)), "new B") {
			t.Fatalf("summary exposed a body: %#v", item)
		}
	}
	last := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, Limit: 2, Cursor: page.NextCursor})
	if last.Error != nil || len(last.Items) != 1 || last.Items[0].MessageID != "mail_same" || last.Items[0].Project.PeerID != "peer_a" {
		t.Fatalf("last page = %#v", last)
	}

	thread := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, ProjectPeerID: "peer_a", ThreadID: "mail_thread"})
	if thread.Error != nil || thread.TotalCount != 2 {
		t.Fatalf("thread page = %#v", thread)
	}
	missing := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, ProjectPeerID: "peer_missing"})
	if missing.Error == nil || missing.Error.Code != attention.ErrorMissing || missing.Page != nil {
		t.Fatalf("unregistered result = %#v", missing)
	}

	action := service.Action(mailread.ActionRequest{
		SchemaVersion: mailread.SchemaVersion, ProjectPeerID: "peer_a", MessageID: "mail_a",
		ActionID: mailread.ActionMarkRead, ReceiptRevision: 0, IdempotencyKey: "read_mail_a",
	})
	if action.Error != nil {
		t.Fatalf("mark read = %#v", action)
	}
	stalePage := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, Limit: 2, Cursor: page.NextCursor})
	if stalePage.Error == nil || stalePage.Error.Code != attention.ErrorStale {
		t.Fatalf("stale continuation = %#v", stalePage)
	}
	unreadOnly := true
	unread := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, UnreadOnly: &unreadOnly})
	if unread.Error != nil || unread.TotalCount != 2 || unread.UnreadCount != 2 {
		t.Fatalf("unread scope = %#v", unread)
	}
}

func TestServiceDetailIsCompositeExactAndNonMutating(t *testing.T) {
	state := t.TempDir()
	projectA, projectB := t.TempDir(), t.TempDir()
	writeQueryProject(t, projectA, "peer_a", "Alpha", nil)
	writeQueryProject(t, projectB, "peer_b", "Beta", nil)
	store, _ := mail.NewStore(state)
	deliverQueryMessage(t, store, "peer_a", "sender_a", "mail_duplicate", "mail_thread_a", "2026-08-17T10:00:00Z", "A body")
	deliverQueryMessage(t, store, "peer_b", "sender_b", "mail_duplicate", "mail_thread_b", "2026-08-17T10:00:01Z", "B body")
	service, _ := NewService(state, queryRegistry(map[string]string{"a": projectA, "b": projectB}))

	listed := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion})
	detail := service.Detail("peer_a", "mail_duplicate")
	if listed.Error != nil || detail.Error != nil || detail.Envelope == nil || detail.Envelope.Body != "A body" || detail.Project == nil || detail.Project.PeerID != "peer_a" {
		t.Fatalf("list/detail = %#v / %#v", listed, detail)
	}
	if _, err := store.Receipt("peer_a", "mail_duplicate"); !errors.Is(err, mail.ErrNotFound) {
		t.Fatalf("read created receipt: %v", err)
	}
	wrong := service.Detail("peer_b", "mail_missing")
	if wrong.Error == nil || wrong.Error.Code != attention.ErrorMissing {
		t.Fatalf("wrong project detail = %#v", wrong)
	}
	malformed := service.Detail("peer_a", "../mail_duplicate")
	if malformed.Error == nil || malformed.Error.Code != attention.ErrorValidation {
		t.Fatalf("malformed identity = %#v", malformed)
	}
}

func TestServiceRegistryDeduplicatesCanonicalPathAndRejectsPeerConflict(t *testing.T) {
	state := t.TempDir()
	project := t.TempDir()
	writeQueryProject(t, project, "peer_shared", "Shared", nil)
	store, _ := mail.NewStore(state)
	deliverQueryMessage(t, store, "peer_shared", "sender", "mail_one", "mail_one", "2026-08-17T10:00:00Z", "body")
	projectLink := filepath.Join(t.TempDir(), "project-link")
	if err := os.Symlink(project, projectLink); err != nil {
		t.Fatal(err)
	}
	service, _ := NewService(state, queryRegistry(map[string]string{"zeta": project, "omega": projectLink, "alpha": project}))
	result := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion})
	if result.Error != nil || len(result.Items) != 1 || result.Items[0].Project.RegistrySlug != "alpha" {
		t.Fatalf("canonical duplicate = %#v", result)
	}

	other := t.TempDir()
	writeQueryProject(t, other, "peer_shared", "Copied", nil)
	conflicted, _ := NewService(state, queryRegistry(map[string]string{"one": project, "two": other}))
	conflict := conflicted.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, ProjectPeerID: "peer_shared"})
	if conflict.Error == nil || conflict.Error.Code != attention.ErrorUnavailable || conflict.Error.Details["category"] != "registry_identity_conflict" || conflict.Error.Details["project_peer_id"] != "peer_shared" {
		t.Fatalf("peer conflict = %#v", conflict)
	}
}

func TestServiceRegistryCollapsesLinkedWorktreeToPrimaryCheckout(t *testing.T) {
	state := t.TempDir()
	repositories := t.TempDir()
	primary := filepath.Join(repositories, "primary repository")
	worktree := filepath.Join(repositories, "linked worktree")
	initQueryGitRepository(t, primary, "peer_shared", "Shared")
	runQueryGit(t, primary, "worktree", "add", "--detach", worktree, "HEAD")
	store, _ := mail.NewStore(state)
	deliverQueryMessage(t, store, "peer_shared", "sender", "mail_one", "mail_one", "2026-08-17T10:00:00Z", "body")
	service, _ := NewService(state, queryRegistry(map[string]string{"alpha_alias": worktree, "zeta_primary": primary}))

	exact := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, ProjectPeerID: "peer_shared"})
	if exact.Error != nil || len(exact.Items) != 1 || exact.Items[0].Project.RegistrySlug != "zeta_primary" || len(exact.Diagnostics) != 0 {
		t.Fatalf("exact worktree result = %#v", exact)
	}
	aggregate := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion})
	if aggregate.Error != nil || len(aggregate.Items) != 1 || len(aggregate.Diagnostics) != 1 || aggregate.Diagnostics[0].Details["skipped_entries"] != "alpha_alias:worktree-alias" {
		t.Fatalf("aggregate worktree result = %#v", aggregate)
	}
}

func TestServiceExactTargetIgnoresUnrelatedPeerConflict(t *testing.T) {
	state, healthy := t.TempDir(), t.TempDir()
	writeQueryProject(t, healthy, "peer_healthy", "Healthy", nil)
	conflictA, conflictB := t.TempDir(), t.TempDir()
	writeQueryProject(t, conflictA, "peer_conflict", "Conflict A", nil)
	writeQueryProject(t, conflictB, "peer_conflict", "Conflict B", nil)
	store, _ := mail.NewStore(state)
	deliverQueryMessage(t, store, "peer_healthy", "sender", "mail_healthy", "mail_healthy", "2026-08-17T10:00:00Z", "body")
	service, _ := NewService(state, queryRegistry(map[string]string{
		"conflict_a": conflictA, "conflict_b": conflictB, "healthy": healthy,
	}))

	result := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, ProjectPeerID: "peer_healthy"})
	if result.Error != nil || len(result.Items) != 1 || result.Items[0].MessageID != "mail_healthy" {
		t.Fatalf("exact healthy result = %#v", result)
	}
}

func TestServiceIndependentGitRepositoriesFailWithBoundedIdentityConflict(t *testing.T) {
	state := t.TempDir()
	repositories := t.TempDir()
	first := filepath.Join(repositories, "first repository")
	second := filepath.Join(repositories, "second repository")
	initQueryGitRepository(t, first, "peer_conflict", "First")
	initQueryGitRepository(t, second, "peer_conflict", "Second")
	service, _ := NewService(state, queryRegistry(map[string]string{"first": first, "second": second}))

	result := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, ProjectPeerID: "peer_conflict"})
	if result.Error == nil || result.Error.Code != attention.ErrorUnavailable || result.Error.Details["category"] != "registry_identity_conflict" || result.Error.Details["project_peer_id"] != "peer_conflict" {
		t.Fatalf("identity conflict = %#v", result)
	}
	encoded := string(mustJSON(t, result))
	if strings.Contains(encoded, first) || strings.Contains(encoded, second) {
		t.Fatalf("identity conflict exposed a path: %s", encoded)
	}
}

func TestServiceAggregateRetainsHealthyProjectsAcrossAliasesAndConflicts(t *testing.T) {
	state := t.TempDir()
	repositories := t.TempDir()
	primary := filepath.Join(repositories, "alias primary")
	worktree := filepath.Join(repositories, "alias worktree")
	initQueryGitRepository(t, primary, "peer_alias", "Alias")
	runQueryGit(t, primary, "worktree", "add", "--detach", worktree, "HEAD")
	healthy := t.TempDir()
	writeQueryProject(t, healthy, "peer_healthy", "Healthy", nil)
	conflictA, conflictB := t.TempDir(), t.TempDir()
	writeQueryProject(t, conflictA, "peer_conflict", "Conflict A", nil)
	writeQueryProject(t, conflictB, "peer_conflict", "Conflict B", nil)
	store, _ := mail.NewStore(state)
	deliverQueryMessage(t, store, "peer_alias", "sender", "mail_alias", "mail_alias", "2026-08-17T10:00:00Z", "alias")
	deliverQueryMessage(t, store, "peer_healthy", "sender", "mail_healthy", "mail_healthy", "2026-08-17T10:00:01Z", "healthy")
	service, _ := NewService(state, queryRegistry(map[string]string{
		"alias_primary": primary, "alias_worktree": worktree, "conflict_a": conflictA,
		"conflict_b": conflictB, "healthy": healthy,
	}))

	result := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion})
	if result.Error != nil || len(result.Items) != 2 || len(result.Diagnostics) != 1 {
		t.Fatalf("aggregate result = %#v", result)
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Details["skipped_count"] != "2" || !strings.Contains(diagnostic.Details["skipped_entries"], "alias_worktree:worktree-alias") || !strings.Contains(diagnostic.Details["skipped_entries"], "conflict_a:identity-conflict") {
		t.Fatalf("aggregate diagnostic = %#v", diagnostic)
	}
}

func TestServiceRepositoryIdentityProbeFailureFallsBackToCanonicalPath(t *testing.T) {
	state, first, second := t.TempDir(), t.TempDir(), t.TempDir()
	writeQueryProject(t, first, "peer_conflict", "First", nil)
	writeQueryProject(t, second, "peer_conflict", "Second", nil)
	service, _ := NewService(state, queryRegistry(map[string]string{"first": first, "second": second}))
	service.resolveRepositoryIdentity = func(string) (string, error) {
		return "", errors.New("probe failed")
	}

	result := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, ProjectPeerID: "peer_conflict"})
	if result.Error == nil || result.Error.Details["category"] != "registry_identity_conflict" {
		t.Fatalf("probe failure result = %#v", result)
	}
}

func TestServiceActionReplayConflictAndReplyDelegateToSource(t *testing.T) {
	state := t.TempDir()
	projectA, projectB := t.TempDir(), t.TempDir()
	writeQueryProject(t, projectA, "peer_a", "Alpha", map[string]string{"b": projectB})
	writeQueryProject(t, projectB, "peer_b", "Beta", map[string]string{"a": projectA})
	store, _ := mail.NewStore(state)
	deliverQueryMessage(t, store, "peer_a", "peer_b", "mail_request", "mail_thread", "2026-08-17T10:00:00Z", "Question")
	service, _ := NewService(state, queryRegistry(map[string]string{"a": projectA, "b": projectB}))
	request := mailread.ActionRequest{
		SchemaVersion: mailread.SchemaVersion, ProjectPeerID: "peer_a", MessageID: "mail_request",
		ActionID: mailread.ActionMarkRead, ReceiptRevision: 0, IdempotencyKey: "operation_read",
	}
	first := service.Action(request)
	replay := service.Action(request)
	if first.Error != nil || replay.Error != nil || first.Receipt == nil || replay.Receipt == nil || first.Receipt.Revision != replay.Receipt.Revision {
		t.Fatalf("idempotent action = %#v / %#v", first, replay)
	}
	conflicting := request
	conflicting.ActionID = mailread.ActionAcknowledge
	conflicting.Input = json.RawMessage(`{"note":"different"}`)
	conflict := service.Action(conflicting)
	if conflict.Error == nil || conflict.Error.Code != attention.ErrorIdempotencyConflict {
		t.Fatalf("action conflict = %#v", conflict)
	}
	staleRequest := request
	staleRequest.IdempotencyKey = "operation_stale"
	stale := service.Action(staleRequest)
	if stale.Error == nil || stale.Error.Code != attention.ErrorStale {
		t.Fatalf("stale action = %#v", stale)
	}

	replyRequest := mailread.ReplyRequest{
		SchemaVersion: mailread.SchemaVersion, ProjectPeerID: "peer_a", MessageID: "mail_request",
		ThreadID: "mail_thread", Body: "Answer", IdempotencyKey: "operation_reply",
	}
	reply := service.Reply(replyRequest)
	replayed := service.Reply(replyRequest)
	if reply.Error != nil || replayed.Error != nil || reply.Delivery == nil || replayed.Delivery == nil || reply.Delivery.MessageID != replayed.Delivery.MessageID {
		t.Fatalf("idempotent reply = %#v / %#v", reply, replayed)
	}
	conflictingReply := replyRequest
	conflictingReply.Body = "changed answer"
	replyConflict := service.Reply(conflictingReply)
	if replyConflict.Error == nil || replyConflict.Error.Code != attention.ErrorIdempotencyConflict {
		t.Fatalf("reply conflict = %#v", replyConflict)
	}
	stored, err := store.Get("peer_b", reply.Delivery.MessageID)
	if err != nil || stored.InReplyTo != "mail_request" || stored.ThreadID != "mail_thread" {
		t.Fatalf("stored reply = %#v, %v", stored, err)
	}
}

func TestServiceExactTargetIgnoresUnrelatedStaleRegistryEntries(t *testing.T) {
	state, project := t.TempDir(), t.TempDir()
	writeQueryProject(t, project, "peer_healthy", "Healthy", nil)
	store, _ := mail.NewStore(state)
	deliverQueryMessage(t, store, "peer_healthy", "sender", "mail_healthy", "mail_thread", "2026-08-17T10:00:00Z", "body")
	invalid := t.TempDir()
	missing := filepath.Join(t.TempDir(), "missing")
	service, err := NewService(state, queryRegistry(map[string]string{
		"001_gone":    missing,
		"002_invalid": invalid,
		"healthy":     project,
	}))
	if err != nil {
		t.Fatal(err)
	}
	result := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, ProjectPeerID: "peer_healthy"})
	if result.Error != nil || len(result.Diagnostics) != 0 || len(result.Items) != 1 || result.Items[0].MessageID != "mail_healthy" {
		t.Fatalf("exact healthy project = %#v", result)
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unrelated registry entry was unexpectedly repaired: %v", err)
	}
}

func TestServiceAggregateRetainsBoundedRegistryDiagnostics(t *testing.T) {
	state, project := t.TempDir(), t.TempDir()
	writeQueryProject(t, project, "peer_healthy", "Healthy", nil)
	store, _ := mail.NewStore(state)
	deliverQueryMessage(t, store, "peer_healthy", "sender", "mail_healthy", "mail_thread", "2026-08-17T10:00:00Z", "body")
	projects := map[string]string{"healthy": project}
	for i := 0; i < maxRegistryIssueDiagnostics+3; i++ {
		projects["gone_"+leftPad3(i)] = filepath.Join(t.TempDir(), "missing")
	}
	service, err := NewService(state, queryRegistry(projects))
	if err != nil {
		t.Fatal(err)
	}
	result := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion})
	if result.Error != nil || len(result.Items) != 1 || result.Items[0].MessageID != "mail_healthy" || len(result.Diagnostics) != 1 {
		t.Fatalf("partial aggregate = %#v", result)
	}
	diagnostic := result.Diagnostics[0]
	entries := strings.Split(diagnostic.Details["skipped_entries"], ",")
	if diagnostic.Code != attention.ErrorUnavailable || diagnostic.Details["category"] != "registry" || diagnostic.Details["skipped_count"] != "11" || len(entries) != maxRegistryIssueDiagnostics {
		t.Fatalf("aggregate diagnostics = %#v", result.Diagnostics)
	}
	encoded := string(mustJSON(t, result))
	if strings.Contains(encoded, string(filepath.Separator)) || !strings.Contains(encoded, `"diagnostics"`) {
		t.Fatalf("unsafe or missing public aggregate diagnostics = %s", encoded)
	}
}

func TestServiceUnavailableRegistryIsNotFalseEmpty(t *testing.T) {
	service, err := NewService(t.TempDir(), queryRegistry(map[string]string{
		"gone": filepath.Join(t.TempDir(), "missing"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []mailread.ListRequest{
		{SchemaVersion: mailread.SchemaVersion},
		{SchemaVersion: mailread.SchemaVersion, ProjectPeerID: "peer_gone"},
	} {
		result := service.List(request)
		if result.Error == nil || result.Error.Code != attention.ErrorUnavailable || result.Page != nil || result.Error.Details["category"] != "registry" || result.Error.Details["skipped_entries"] != "gone:path" {
			t.Fatalf("unavailable registry = %#v", result)
		}
	}
}

func TestServiceHealthyEmptyRegistryRemainsAuthoritativeEmpty(t *testing.T) {
	service, err := NewService(t.TempDir(), queryRegistry(map[string]string{}))
	if err != nil {
		t.Fatal(err)
	}
	result := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion})
	if result.Error != nil || result.Page == nil || result.TotalCount != 0 || len(result.Items) != 0 {
		t.Fatalf("healthy empty registry = %#v", result)
	}
}

func TestServiceMailboxFailureIsUnavailableNotPartial(t *testing.T) {
	state, project := t.TempDir(), t.TempDir()
	writeQueryProject(t, project, "peer_bad", "Bad", nil)
	messageDir := filepath.Join(state, "mail", "boxes", "peer_bad", "messages")
	if err := os.MkdirAll(messageDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(messageDir, "mail_bad.json"), []byte(`{"broken":`), 0o600); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(state, queryRegistry(map[string]string{"bad": project}))
	if err != nil {
		t.Fatal(err)
	}
	result := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion})
	if result.Error == nil || result.Error.Code != attention.ErrorUnavailable || result.Page != nil || len(result.Items) != 0 {
		t.Fatalf("mailbox failure = %#v", result)
	}
}

func BenchmarkServiceListFullScan(b *testing.B) {
	state := b.TempDir()
	projects := map[string]string{}
	store, _ := mail.NewStore(state)
	for p := 0; p < 3; p++ {
		root := b.TempDir()
		peerID := "peer_" + string(rune('a'+p))
		slug := "project_" + string(rune('a'+p))
		writeQueryProject(b, root, peerID, slug, nil)
		projects[slug] = root
		for i := 0; i < 100; i++ {
			id := "mail_" + string(rune('a'+p)) + "_" + leftPad3(i)
			deliverQueryMessage(b, store, peerID, "sender_"+string(rune('a'+p)), id, id, "2026-08-17T10:00:00Z", "body")
		}
	}
	service, _ := NewService(state, queryRegistry(projects))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := service.List(mailread.ListRequest{SchemaVersion: mailread.SchemaVersion, Limit: 100})
		if result.Error != nil || result.TotalCount != 300 {
			b.Fatalf("full scan = %#v", result)
		}
	}
}

func leftPad3(value int) string {
	return string([]byte{'0' + byte(value/100), '0' + byte(value/10%10), '0' + byte(value%10)})
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
