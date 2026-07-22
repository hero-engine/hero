package tracker

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	evidencecontract "github.com/hero-engine/hero/contracts/trackerevidence"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

type evidenceStoreTracker struct {
	mu            sync.Mutex
	issue         Issue
	evidence      IssueEvidence
	getIssueCalls int
	evidenceCalls int
	downloadCalls int
	getIssueErr   error
	evidenceErr   error
	downloadErr   error
	blockStage    string
	stageStarted  chan struct{}
}

func (mock *evidenceStoreTracker) GetIssue(string) (*Issue, error) {
	mock.mu.Lock()
	defer mock.mu.Unlock()
	mock.getIssueCalls++
	if mock.getIssueErr != nil {
		return nil, mock.getIssueErr
	}
	copy := mock.issue
	return &copy, nil
}

func (mock *evidenceStoreTracker) GetIssueEvidence(string) (*IssueEvidence, error) {
	mock.mu.Lock()
	defer mock.mu.Unlock()
	mock.evidenceCalls++
	if mock.evidenceErr != nil {
		return nil, mock.evidenceErr
	}
	data, _ := json.Marshal(mock.evidence)
	var copy IssueEvidence
	_ = json.Unmarshal(data, &copy)
	return &copy, nil
}

func (mock *evidenceStoreTracker) DownloadEvidenceAttachment(string) ([]byte, error) {
	mock.mu.Lock()
	defer mock.mu.Unlock()
	mock.downloadCalls++
	if mock.downloadErr != nil {
		return nil, mock.downloadErr
	}
	return []byte("private attachment bytes"), nil
}

func (mock *evidenceStoreTracker) GetEvidenceMetadataContext(ctx context.Context, issueID string) (*Issue, error) {
	if mock.blockStage == "metadata" {
		close(mock.stageStarted)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return mock.GetIssue(issueID)
}

func (mock *evidenceStoreTracker) GetIssueEvidenceContext(ctx context.Context, issueID string) (*IssueEvidence, error) {
	if mock.blockStage == "evidence" {
		close(mock.stageStarted)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return mock.GetIssueEvidence(issueID)
}

func (mock *evidenceStoreTracker) DownloadEvidenceAttachmentContext(ctx context.Context, contentURL string) ([]byte, error) {
	if mock.blockStage == "download" {
		close(mock.stageStarted)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return mock.DownloadEvidenceAttachment(contentURL)
}

func (*evidenceStoreTracker) Name() string                                { return "jira" }
func (*evidenceStoreTracker) CreateIssue(*spec.Spec) (string, error)      { return "", nil }
func (*evidenceStoreTracker) UpdateStatus(string, spec.Status) error      { return nil }
func (*evidenceStoreTracker) UpdateSize(string, string) error             { return nil }
func (*evidenceStoreTracker) ListIssues(string, int) ([]Issue, error)     { return nil, nil }
func (*evidenceStoreTracker) Search(SearchQuery) ([]Issue, error)         { return nil, nil }
func (*evidenceStoreTracker) AddComment(string, string) error             { return nil }
func (*evidenceStoreTracker) AttachFile(string, string, string) error     { return nil }
func (*evidenceStoreTracker) SupportsHierarchy() bool                     { return false }
func (*evidenceStoreTracker) MapSize(string) (string, error)              { return "", nil }
func (*evidenceStoreTracker) ReverseMapSize(string) (string, error)       { return "", nil }
func (*evidenceStoreTracker) GetFields(string) (map[string]Value, error)  { return nil, nil }
func (*evidenceStoreTracker) UpdateFields(string, map[string]Value) error { return nil }

func TestEvidenceLoader_FirstLoadCurrentRefreshAndForce(t *testing.T) {
	root, specDir, loader, mock := newEvidenceStoreTest(t, "jira")
	request := evidencecontract.Request{SpecSlug: "morph-297"}

	first := loader.Load(context.Background(), request)
	if first.Status != evidencecontract.StateFetched || first.CacheHit || first.Error != nil {
		t.Fatalf("first status = %+v", first)
	}
	assertEvidenceFilesPrivate(t, root, specDir, first)
	manifestBefore := mustReadEvidenceFile(t, filepath.Join(specDir, evidenceManifestName))
	payloadBefore := mustReadEvidenceFile(t, filepath.Join(specDir, evidencePrivateName, evidencePayloadName))
	manifestInfoBefore, _ := os.Stat(filepath.Join(specDir, evidenceManifestName))
	payloadInfoBefore, _ := os.Stat(filepath.Join(specDir, evidencePrivateName, evidencePayloadName))

	second := loader.Load(context.Background(), request)
	if second.Status != evidencecontract.StateCurrent || !second.CacheHit || mock.evidenceCalls != 1 || mock.downloadCalls != 1 {
		t.Fatalf("cache-hit status=%+v calls evidence=%d download=%d", second, mock.evidenceCalls, mock.downloadCalls)
	}
	manifestInfoAfter, _ := os.Stat(filepath.Join(specDir, evidenceManifestName))
	payloadInfoAfter, _ := os.Stat(filepath.Join(specDir, evidencePrivateName, evidencePayloadName))
	if string(manifestBefore) != string(mustReadEvidenceFile(t, filepath.Join(specDir, evidenceManifestName))) || string(payloadBefore) != string(mustReadEvidenceFile(t, filepath.Join(specDir, evidencePrivateName, evidencePayloadName))) || !manifestInfoBefore.ModTime().Equal(manifestInfoAfter.ModTime()) || !payloadInfoBefore.ModTime().Equal(payloadInfoAfter.ModTime()) {
		t.Fatal("current cache hit rewrote manifest or payload")
	}

	mock.issue.UpdatedAt = "2026-07-21T13:00:00.000-0600"
	mock.evidence.Normalized.UpdatedAt = mock.issue.UpdatedAt
	refreshed := loader.Load(context.Background(), request)
	if refreshed.Status != evidencecontract.StateRefreshed || refreshed.TrackerUpdatedAt != mock.issue.UpdatedAt || mock.evidenceCalls != 2 {
		t.Fatalf("refresh status=%+v evidence calls=%d", refreshed, mock.evidenceCalls)
	}
	request.ForceRefresh = true
	forced := loader.Load(context.Background(), request)
	if forced.Status != evidencecontract.StateRefreshed || mock.evidenceCalls != 3 {
		t.Fatalf("force status=%+v evidence calls=%d", forced, mock.evidenceCalls)
	}
}

func TestEvidenceLoader_CorruptionMissingAttachmentAndUnknownTimestampRefetch(t *testing.T) {
	root, specDir, loader, mock := newEvidenceStoreTest(t, "jira")
	request := evidencecontract.Request{SpecSlug: "morph-297"}
	if got := loader.Load(context.Background(), request); got.Status != evidencecontract.StateFetched {
		t.Fatalf("first load = %+v", got)
	}
	payloadPath := filepath.Join(specDir, evidencePrivateName, evidencePayloadName)
	if err := os.WriteFile(payloadPath, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := loader.Load(context.Background(), request); got.Status != evidencecontract.StateRefreshed || mock.evidenceCalls != 2 {
		t.Fatalf("corrupt refresh = %+v calls=%d", got, mock.evidenceCalls)
	}
	attachmentPath := filepath.Join(root, filepath.FromSlash(mock.evidenceAttachmentPath(t, specDir)))
	if err := os.WriteFile(attachmentPath, []byte("mutated attachment bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := loader.Load(context.Background(), request); got.Status != evidencecontract.StateRefreshed || mock.evidenceCalls != 3 {
		t.Fatalf("mutated attachment refresh = %+v calls=%d", got, mock.evidenceCalls)
	}
	attachmentPath = filepath.Join(root, filepath.FromSlash(mock.evidenceAttachmentPath(t, specDir)))
	if err := os.Remove(attachmentPath); err != nil {
		t.Fatal(err)
	}
	if got := loader.Load(context.Background(), request); got.Status != evidencecontract.StateRefreshed || mock.evidenceCalls != 4 {
		t.Fatalf("missing attachment refresh = %+v calls=%d", got, mock.evidenceCalls)
	}
	mock.issue.UpdatedAt = "not-a-provider-time"
	mock.evidence.Normalized.UpdatedAt = mock.issue.UpdatedAt
	if got := loader.Load(context.Background(), request); got.Status != evidencecontract.StateRefreshed {
		t.Fatalf("malformed time refresh = %+v", got)
	}
	if got := loader.Load(context.Background(), request); got.Status != evidencecontract.StateRefreshed || got.TrackerUpdatedAt != "not-a-provider-time" {
		t.Fatalf("malformed time was treated as current or substituted: %+v", got)
	}
}

func TestEvidenceLoader_DownloadOmissionRemainsPrivateAndNeverCurrent(t *testing.T) {
	_, specDir, loader, mock := newEvidenceStoreTest(t, "jira")
	mock.downloadErr = errors.New("private download URL and credential-canary")
	request := evidencecontract.Request{SpecSlug: "morph-297"}
	first := loader.Load(context.Background(), request)
	if first.Status != evidencecontract.StateFetched || first.OmissionCount != 1 {
		t.Fatalf("first omitted download = %+v", first)
	}
	evidence, err := loader.ReadSnapshot(first)
	if err != nil || len(evidence.Omissions) != 1 || evidence.Attachments[0].LocalPath != "" {
		t.Fatalf("private omission evidence=%+v err=%v", evidence, err)
	}
	manifest := string(mustReadEvidenceFile(t, filepath.Join(specDir, evidenceManifestName)))
	if strings.Contains(manifest, "private download") || strings.Contains(manifest, "credential-canary") {
		t.Fatalf("manifest leaked download failure: %s", manifest)
	}
	second := loader.Load(context.Background(), request)
	if second.Status != evidencecontract.StateRefreshed || mock.evidenceCalls != 2 || mock.downloadCalls != 2 {
		t.Fatalf("omitted attachment was treated as current: %+v calls=%d/%d", second, mock.evidenceCalls, mock.downloadCalls)
	}
}

func TestEvidenceLoader_ManifestIdentityMismatchRefetches(t *testing.T) {
	_, specDir, loader, mock := newEvidenceStoreTest(t, "jira")
	request := evidencecontract.Request{SpecSlug: "morph-297"}
	if got := loader.Load(context.Background(), request); got.Status != evidencecontract.StateFetched {
		t.Fatalf("first load = %+v", got)
	}

	for name, mutate := range map[string]func(*evidencecontract.Manifest){
		"version":  func(manifest *evidencecontract.Manifest) { manifest.Version = "tracker-evidence/v999" },
		"provider": func(manifest *evidencecontract.Manifest) { manifest.Provider = "github" },
		"issue":    func(manifest *evidencecontract.Manifest) { manifest.IssueID = "OTHER-1" },
		"updated":  func(manifest *evidencecontract.Manifest) { manifest.TrackerUpdatedAt = "2026-07-20T00:00:00Z" },
	} {
		t.Run(name, func(t *testing.T) {
			manifestPath := filepath.Join(specDir, evidenceManifestName)
			var manifest evidencecontract.Manifest
			if err := json.Unmarshal(mustReadEvidenceFile(t, manifestPath), &manifest); err != nil {
				t.Fatal(err)
			}
			mutate(&manifest)
			data, _ := json.MarshalIndent(manifest, "", "  ")
			if err := os.WriteFile(manifestPath, append(data, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
			before := mock.evidenceCalls
			if got := loader.Load(context.Background(), request); got.Status != evidencecontract.StateRefreshed || mock.evidenceCalls != before+1 {
				t.Fatalf("status=%+v evidence calls=%d -> %d", got, before, mock.evidenceCalls)
			}
		})
	}
}

func TestEvidenceLoader_UnavailablePreservesValidatedSnapshot(t *testing.T) {
	_, specDir, loader, mock := newEvidenceStoreTest(t, "jira")
	request := evidencecontract.Request{SpecSlug: "morph-297"}
	first := loader.Load(context.Background(), request)
	manifestBefore := mustReadEvidenceFile(t, filepath.Join(specDir, evidenceManifestName))
	payloadBefore := mustReadEvidenceFile(t, filepath.Join(specDir, evidencePrivateName, evidencePayloadName))
	mock.getIssueErr = errors.New("credential-canary https://private.example user@example.com")
	got := loader.Load(context.Background(), request)
	if got.Status != evidencecontract.StateUnavailable || got.Error == nil || got.Error.Code != evidencecontract.ErrorProviderUnavailable || got.ContentSHA256 != first.ContentSHA256 || got.CacheHit {
		t.Fatalf("unavailable status = %+v", got)
	}
	encoded, _ := json.Marshal(got)
	for _, secret := range []string{"credential-canary", "private.example", "user@example.com"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("status leaked %q: %s", secret, encoded)
		}
	}
	if string(manifestBefore) != string(mustReadEvidenceFile(t, filepath.Join(specDir, evidenceManifestName))) || string(payloadBefore) != string(mustReadEvidenceFile(t, filepath.Join(specDir, evidencePrivateName, evidencePayloadName))) {
		t.Fatal("provider failure changed the last committed snapshot")
	}
}

func TestEvidenceLoader_UnsupportedAndCancellationCreateNothing(t *testing.T) {
	_, specDir, loader, mock := newEvidenceStoreTest(t, "github")
	got := loader.Load(context.Background(), evidencecontract.Request{SpecSlug: "morph-297"})
	if got.Status != evidencecontract.StateUnsupported || got.Error == nil || got.Error.Code != evidencecontract.ErrorUnsupportedProvider || mock.getIssueCalls != 0 {
		t.Fatalf("unsupported status = %+v calls=%d", got, mock.getIssueCalls)
	}
	if evidenceManifestExists(specDir) {
		t.Fatal("unsupported provider created a manifest")
	}

	_, cancelledDir, cancelledLoader, cancelledMock := newEvidenceStoreTest(t, "jira")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled := cancelledLoader.Load(ctx, evidencecontract.Request{SpecSlug: "morph-297"})
	if cancelled.Error == nil || cancelled.Error.Code != evidencecontract.ErrorCancelled || cancelledMock.getIssueCalls != 0 || evidenceManifestExists(cancelledDir) {
		t.Fatalf("cancelled status = %+v calls=%d", cancelled, cancelledMock.getIssueCalls)
	}
}

func TestEvidenceLoader_CancelsEveryInFlightProviderStage(t *testing.T) {
	for _, stage := range []string{"metadata", "evidence", "download"} {
		t.Run(stage, func(t *testing.T) {
			_, specDir, loader, mock := newEvidenceStoreTest(t, "jira")
			mock.blockStage = stage
			mock.stageStarted = make(chan struct{})
			ctx, cancel := context.WithCancel(context.Background())
			result := make(chan evidencecontract.Status, 1)
			go func() {
				result <- loader.Load(ctx, evidencecontract.Request{SpecSlug: "morph-297"})
			}()
			<-mock.stageStarted
			cancel()
			got := <-result
			if got.Error == nil || got.Error.Code != evidencecontract.ErrorCancelled || evidenceManifestExists(specDir) {
				t.Fatalf("cancelled %s status=%+v", stage, got)
			}
			if _, err := os.Stat(filepath.Join(specDir, evidencePrivateName)); !os.IsNotExist(err) {
				t.Fatalf("cancelled %s left private state: %v", stage, err)
			}
		})
	}
}

func TestEvidenceLoader_NoAttachmentsDefersDownloads(t *testing.T) {
	_, _, loader, mock := newEvidenceStoreTest(t, "jira")
	includeAttachments := false
	request := evidencecontract.Request{SpecSlug: "morph-297", IncludeAttachments: &includeAttachments}
	if got := loader.Load(context.Background(), request); got.Status != evidencecontract.StateFetched || mock.downloadCalls != 0 {
		t.Fatalf("metadata-only load=%+v downloads=%d", got, mock.downloadCalls)
	}
	if got := loader.Load(context.Background(), evidencecontract.Request{SpecSlug: "morph-297"}); got.Status != evidencecontract.StateRefreshed || mock.downloadCalls != 1 {
		t.Fatalf("attachment load=%+v downloads=%d", got, mock.downloadCalls)
	}
}

func TestEvidenceLoader_ResolutionErrorsDoNotFetchOrWrite(t *testing.T) {
	_, specDir, loader, mock := newEvidenceStoreTest(t, "jira")
	setting := func(value string) json.RawMessage {
		data, _ := json.Marshal(value)
		return data
	}
	loader.loadConfig = func(string) (config.Config, error) {
		return config.Config{Folder: ".hero", Integrations: &config.IntegrationsConfig{Connections: map[string]config.IntegrationConfig{
			"jira-a": {Provider: "jira", Settings: map[string]json.RawMessage{"project": setting("A"), "base_url": setting("https://a.example")}},
			"jira-b": {Provider: "jira", Settings: map[string]json.RawMessage{"project": setting("B"), "base_url": setting("https://b.example")}},
		}}}, nil
	}
	got := loader.Load(context.Background(), evidencecontract.Request{SpecSlug: "morph-297"})
	if got.Error == nil || got.Error.Code != evidencecontract.ErrorAmbiguousConnection || mock.getIssueCalls != 0 || evidenceManifestExists(specDir) {
		t.Fatalf("ambiguous load=%+v calls=%d", got, mock.getIssueCalls)
	}

	missing := loader.Load(context.Background(), evidencecontract.Request{SpecSlug: "missing", ConnectionID: "jira-a"})
	if missing.Error == nil || missing.Error.Code != evidencecontract.ErrorSpecNotFound || mock.getIssueCalls != 0 {
		t.Fatalf("missing load=%+v calls=%d", missing, mock.getIssueCalls)
	}
}

func TestEvidenceLoader_WriteFailurePreservesCommittedSnapshot(t *testing.T) {
	_, specDir, loader, mock := newEvidenceStoreTest(t, "jira")
	request := evidencecontract.Request{SpecSlug: "morph-297"}
	first := loader.Load(context.Background(), request)
	manifestBefore := mustReadEvidenceFile(t, filepath.Join(specDir, evidenceManifestName))
	payloadBefore := mustReadEvidenceFile(t, filepath.Join(specDir, evidencePrivateName, evidencePayloadName))
	mock.issue.UpdatedAt = "2026-07-21T13:00:00.000-0600"
	mock.evidence.Normalized.UpdatedAt = mock.issue.UpdatedAt
	loader.writeSnapshot = func(context.Context, string, string, string, string, string, bool, EvidenceTracker, *IssueEvidence) (evidencecontract.Manifest, error) {
		return evidencecontract.Manifest{}, errors.New("credential-canary write detail")
	}
	got := loader.Load(context.Background(), request)
	if got.Status != evidencecontract.StateUnavailable || got.Error == nil || got.Error.Code != evidencecontract.ErrorWriteFailed || got.ContentSHA256 != first.ContentSHA256 {
		t.Fatalf("write failure = %+v", got)
	}
	encoded, _ := json.Marshal(got)
	if strings.Contains(string(encoded), "credential-canary") {
		t.Fatalf("write error leaked: %s", encoded)
	}
	if string(manifestBefore) != string(mustReadEvidenceFile(t, filepath.Join(specDir, evidenceManifestName))) || string(payloadBefore) != string(mustReadEvidenceFile(t, filepath.Join(specDir, evidencePrivateName, evidencePayloadName))) {
		t.Fatal("write failure changed the committed snapshot")
	}
}

func TestRecoverEvidenceStoreRestoresPreCommitBackup(t *testing.T) {
	root, specDir, loader, _ := newEvidenceStoreTest(t, "jira")
	if got := loader.Load(context.Background(), evidencecontract.Request{SpecSlug: "morph-297"}); got.Status != evidencecontract.StateFetched {
		t.Fatalf("first load = %+v", got)
	}
	privateDir := filepath.Join(specDir, evidencePrivateName)
	backupDir := privateDir + ".backup"
	payloadBefore := mustReadEvidenceFile(t, filepath.Join(privateDir, evidencePayloadName))
	if err := os.Rename(privateDir, backupDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(privateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(privateDir, evidencePayloadName), []byte("partial replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	recoverEvidenceStore(root, specDir)
	if string(mustReadEvidenceFile(t, filepath.Join(privateDir, evidencePayloadName))) != string(payloadBefore) {
		t.Fatal("recovery did not restore the last manifest-backed private snapshot")
	}
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Fatalf("backup still exists after recovery: %v", err)
	}
}

func TestRecoverEvidenceStoreKeepsManifestCommittedCandidate(t *testing.T) {
	root, specDir, loader, _ := newEvidenceStoreTest(t, "jira")
	if got := loader.Load(context.Background(), evidencecontract.Request{SpecSlug: "morph-297"}); got.Status != evidencecontract.StateFetched {
		t.Fatalf("first load = %+v", got)
	}
	privateDir := filepath.Join(specDir, evidencePrivateName)
	backupDir := privateDir + ".backup"
	if err := copyEvidenceDirectory(privateDir, backupDir); err != nil {
		t.Fatal(err)
	}
	payloadBefore := mustReadEvidenceFile(t, filepath.Join(privateDir, evidencePayloadName))
	recoverEvidenceStore(root, specDir)
	if string(mustReadEvidenceFile(t, filepath.Join(privateDir, evidencePayloadName))) != string(payloadBefore) {
		t.Fatal("recovery replaced a manifest-committed candidate")
	}
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Fatalf("backup still exists after committed recovery: %v", err)
	}
}

func TestPublishEvidenceSnapshotFailuresPreservePriorCommit(t *testing.T) {
	for _, stage := range []string{"backup", "candidate", "manifest", "cancel-after-candidate"} {
		t.Run(stage, func(t *testing.T) {
			specDir := t.TempDir()
			privateDir := filepath.Join(specDir, evidencePrivateName)
			candidateDir := filepath.Join(specDir, evidencePrivateName+".tmp-candidate")
			if err := os.Mkdir(privateDir, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(candidateDir, 0o700); err != nil {
				t.Fatal(err)
			}
			oldPayload := []byte("old committed payload")
			oldManifest := []byte("old committed manifest")
			if err := os.WriteFile(filepath.Join(privateDir, evidencePayloadName), oldPayload, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(candidateDir, evidencePayloadName), []byte("new candidate payload"), 0o600); err != nil {
				t.Fatal(err)
			}
			manifestPath := filepath.Join(specDir, evidenceManifestName)
			if err := os.WriteFile(manifestPath, oldManifest, 0o600); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ops := defaultEvidencePublishOps
			rename := ops.rename
			ops.rename = func(oldPath, newPath string) error {
				if stage == "backup" && oldPath == privateDir {
					return errors.New("backup rename failed")
				}
				if stage == "candidate" && oldPath == candidateDir {
					return errors.New("candidate rename failed")
				}
				err := rename(oldPath, newPath)
				if err == nil && stage == "cancel-after-candidate" && oldPath == candidateDir {
					cancel()
				}
				return err
			}
			if stage == "manifest" {
				ops.writeManifest = func(string, []byte, os.FileMode) error { return errors.New("manifest write failed") }
			}
			if err := publishEvidenceSnapshotWithOps(ctx, specDir, candidateDir, []byte("new manifest"), ops); err == nil {
				t.Fatalf("%s failure unexpectedly succeeded", stage)
			}
			if string(mustReadEvidenceFile(t, filepath.Join(privateDir, evidencePayloadName))) != string(oldPayload) {
				t.Fatalf("%s failure changed private snapshot", stage)
			}
			if string(mustReadEvidenceFile(t, manifestPath)) != string(oldManifest) {
				t.Fatalf("%s failure changed manifest", stage)
			}
			if _, err := os.Stat(privateDir + ".backup"); !os.IsNotExist(err) {
				t.Fatalf("%s failure left backup: %v", stage, err)
			}
		})
	}
}

func copyEvidenceDirectory(source, destination string) error {
	if err := os.Mkdir(destination, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(source, entry.Name())
		destinationPath := filepath.Join(destination, entry.Name())
		if entry.IsDir() {
			if err := copyEvidenceDirectory(sourcePath, destinationPath); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(destinationPath, data, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func TestEvidenceLoader_ConcurrentLoadsCollapseFullFetch(t *testing.T) {
	_, _, loader, mock := newEvidenceStoreTest(t, "jira")
	request := evidencecontract.Request{SpecSlug: "morph-297"}
	results := make(chan evidencecontract.Status, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			results <- loader.Load(context.Background(), request)
		}()
	}
	wait.Wait()
	close(results)
	states := map[evidencecontract.State]int{}
	for result := range results {
		states[result.Status]++
	}
	if states[evidencecontract.StateFetched] != 1 || states[evidencecontract.StateCurrent] != 1 || mock.evidenceCalls != 1 || mock.downloadCalls != 1 {
		t.Fatalf("states=%v evidence=%d downloads=%d", states, mock.evidenceCalls, mock.downloadCalls)
	}
}

func newEvidenceStoreTest(t *testing.T, provider string) (string, string, *EvidenceLoader, *evidenceStoreTracker) {
	t.Helper()
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	specDir := filepath.Join(heroDir, "planning", "bugs", "morph-297")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	specBody := "---\ntitle: MORPH-297\nslug: morph-297\ntype: bug\nstatus: planning\ntracker_id: MORPH-297\n---\n# MORPH-297\n"
	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(specBody), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.Folder = ".hero"
	cfg.Tracker = &config.TrackerConfig{Type: provider, Project: "MORPH", BaseURL: "https://private.example", UserEmail: "user@example.com", Token: "credential-canary"}
	updated := "2026-07-21T12:00:00.123-0600"
	mock := &evidenceStoreTracker{
		issue: Issue{ID: "MORPH-297", UpdatedAt: updated},
		evidence: IssueEvidence{
			Tracker: "jira", IssueID: "MORPH-297", RetrievedAt: "2026-07-21T18:00:01Z",
			Normalized:  &Issue{ID: "MORPH-297", UpdatedAt: updated, Description: "private description"},
			RawFields:   map[string]json.RawMessage{"description": json.RawMessage(`{"secret":"raw evidence"}`)},
			Comments:    []EvidenceComment{{ID: "1", Author: "Private Person", Text: "private comment"}},
			Attachments: []EvidenceAttachment{{ID: "../9", Filename: "customer screenshot.png", Content: "https://private.example/attachment/9"}},
		},
	}
	loader := NewEvidenceLoader(root)
	loader.loadConfig = func(string) (config.Config, error) { return cfg, nil }
	loader.newTracker = func(config.TrackerConnection, config.Config, config.Secret, string) (Tracker, error) {
		return mock, nil
	}
	return root, specDir, loader, mock
}

func (mock *evidenceStoreTracker) evidenceAttachmentPath(t *testing.T, specDir string) string {
	t.Helper()
	payload := mustReadEvidenceFile(t, filepath.Join(specDir, evidencePrivateName, evidencePayloadName))
	var evidence IssueEvidence
	if err := json.Unmarshal(payload, &evidence); err != nil {
		t.Fatal(err)
	}
	if len(evidence.Attachments) != 1 || evidence.Attachments[0].LocalPath == "" {
		t.Fatalf("attachment paths = %+v", evidence.Attachments)
	}
	return evidence.Attachments[0].LocalPath
}

func assertEvidenceFilesPrivate(t *testing.T, root, specDir string, status evidencecontract.Status) {
	t.Helper()
	privateInfo, err := os.Stat(filepath.Join(specDir, evidencePrivateName))
	if err != nil || privateInfo.Mode().Perm() != 0o700 {
		t.Fatalf("private dir mode=%v err=%v", privateInfo, err)
	}
	for _, path := range []string{filepath.Join(specDir, evidenceManifestName), filepath.Join(specDir, evidencePrivateName, evidencePayloadName), filepath.Join(root, filepath.FromSlash(status.EvidencePath))} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("private file %s mode=%v err=%v", path, info, err)
		}
	}
	manifest := mustReadEvidenceFile(t, filepath.Join(specDir, evidenceManifestName))
	for _, private := range []string{"credential-canary", "private.example", "user@example.com", "private description", "raw evidence", "private comment", "Private Person", "customer screenshot.png"} {
		if strings.Contains(string(manifest), private) {
			t.Fatalf("manifest leaked %q: %s", private, manifest)
		}
	}
}

func mustReadEvidenceFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
