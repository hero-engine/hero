package tracker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	evidencecontract "github.com/hero-engine/hero/contracts/trackerevidence"
	"github.com/hero-engine/hero/internal/config"
	herospec "github.com/hero-engine/hero/internal/spec"
)

const (
	evidenceManifestName = "tracker-evidence.json"
	evidencePrivateName  = ".tracker-evidence"
	evidencePayloadName  = "evidence.json"
)

var evidenceLocks sync.Map

type evidenceLock struct{ sync.Mutex }

// EvidenceLoader is the one in-process implementation behind CLI and MCP
// full-details loads. Provider credentials stay inside adapter construction;
// only the bounded tracker-evidence/v1 status crosses this boundary.
type EvidenceLoader struct {
	projectRoot   string
	loadConfig    func(string) (config.Config, error)
	newTracker    func(config.TrackerConnection, config.Config, config.Secret, string) (Tracker, error)
	writeSnapshot func(context.Context, string, string, string, string, string, bool, EvidenceTracker, *IssueEvidence) (evidencecontract.Manifest, error)
}

func NewEvidenceLoader(projectRoot string) *EvidenceLoader {
	return &EvidenceLoader{
		projectRoot: projectRoot,
		loadConfig:  config.Load,
		newTracker: func(connection config.TrackerConnection, cfg config.Config, token config.Secret, knowledgeDir string) (Tracker, error) {
			tc := connection.TrackerConfig()
			tc.Token = token.Reveal()
			return NewWithJiraConfig(tc, cfg.Jira, knowledgeDir)
		},
		writeSnapshot: writeEvidenceSnapshot,
	}
}

// Load explicitly materializes or validates full evidence for one linked spec.
// It never runs from import, refresh, discovery, or background code.
func (loader *EvidenceLoader) Load(ctx context.Context, request evidencecontract.Request) evidencecontract.Status {
	status := evidencecontract.Status{Version: evidencecontract.Version, SpecSlug: strings.TrimSpace(request.SpecSlug)}
	if err := ctx.Err(); err != nil {
		return failEvidenceStatus(status, evidencecontract.StateUnavailable, evidencecontract.ErrorCancelled, "evidence load cancelled", false)
	}

	cfg, err := loader.loadConfig(loader.projectRoot)
	if err != nil {
		return failEvidenceStatus(status, evidencecontract.StateUnavailable, evidencecontract.ErrorProviderUnavailable, "tracker configuration is unavailable", true)
	}
	heroDir := cfg.HeroDir(loader.projectRoot)
	linkedSpec, err := findEvidenceSpec(heroDir, status.SpecSlug)
	if err != nil {
		return failEvidenceStatus(status, evidencecontract.StateUnavailable, evidencecontract.ErrorSpecNotFound, "spec was not found", false)
	}
	if linkedSpec.TrackerID == "" {
		return failEvidenceStatus(status, evidencecontract.StateUnavailable, evidencecontract.ErrorTrackerUnlinked, "spec is not linked to a tracker issue", false)
	}
	status.IssueID = linkedSpec.TrackerID
	specDir := filepath.Dir(linkedSpec.Path)
	status.ManifestPath = relativeEvidencePath(loader.projectRoot, filepath.Join(specDir, evidenceManifestName))
	status.EvidencePath = relativeEvidencePath(loader.projectRoot, filepath.Join(specDir, evidencePrivateName, evidencePayloadName))

	connection, err := cfg.ResolveTrackerConnection(request.ConnectionID)
	if err != nil {
		code := evidencecontract.ErrorProviderUnavailable
		if strings.Contains(err.Error(), "connection_id is required") {
			code = evidencecontract.ErrorAmbiguousConnection
		}
		return failEvidenceStatus(status, evidencecontract.StateUnavailable, code, "tracker connection could not be selected", false)
	}
	status.Provider, status.ConnectionID = connection.Provider, connection.ID
	if connection.Provider != "jira" {
		return failEvidenceStatus(status, evidencecontract.StateUnsupported, evidencecontract.ErrorUnsupportedProvider, "tracker provider does not support full evidence", false)
	}
	token, err := connection.ResolveToken()
	if err != nil {
		return unavailableWithSnapshot(loader.projectRoot, status, specDir, request.AttachmentsEnabled(), evidencecontract.ErrorProviderUnavailable, "tracker credential is unavailable")
	}
	t, err := loader.newTracker(connection, cfg, token, cfg.TrackerKnowledgeDir(loader.projectRoot))
	if err != nil {
		return unavailableWithSnapshot(loader.projectRoot, status, specDir, request.AttachmentsEnabled(), evidencecontract.ErrorProviderUnavailable, "tracker provider is unavailable")
	}
	provider, supported := t.(EvidenceTracker)
	if !supported {
		return failEvidenceStatus(status, evidencecontract.StateUnsupported, evidencecontract.ErrorUnsupportedProvider, "tracker provider does not support full evidence", false)
	}

	if err := ctx.Err(); err != nil {
		return failEvidenceStatus(status, evidencecontract.StateUnavailable, evidencecontract.ErrorCancelled, "evidence load cancelled", false)
	}
	metadata, err := evidenceIssueMetadata(ctx, t, linkedSpec.TrackerID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return failEvidenceStatus(status, evidencecontract.StateUnavailable, evidencecontract.ErrorCancelled, "evidence load cancelled", false)
		}
		return unavailableWithSnapshot(loader.projectRoot, status, specDir, request.AttachmentsEnabled(), evidencecontract.ErrorProviderUnavailable, "tracker provider is unavailable")
	}
	status.TrackerUpdatedAt = metadata.UpdatedAt

	lock := evidenceLockFor(specDir)
	lock.Lock()
	defer lock.Unlock()
	recoverEvidenceStore(loader.projectRoot, specDir)

	hadManifest := evidenceManifestExists(specDir)
	if !request.ForceRefresh && validTrackerEvidenceTime(metadata.UpdatedAt) {
		manifest, evidence, checkErr := validateEvidenceSnapshot(loader.projectRoot, specDir, connection.Provider, linkedSpec.TrackerID, metadata.UpdatedAt, request.AttachmentsEnabled())
		if checkErr == nil {
			return statusFromSnapshot(status, evidencecontract.StateCurrent, manifest, evidence, true)
		}
	}

	if err := ctx.Err(); err != nil {
		return failEvidenceStatus(status, evidencecontract.StateUnavailable, evidencecontract.ErrorCancelled, "evidence load cancelled", false)
	}
	evidence, err := getIssueEvidenceContext(ctx, provider, linkedSpec.TrackerID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return failEvidenceStatus(status, evidencecontract.StateUnavailable, evidencecontract.ErrorCancelled, "evidence load cancelled", false)
		}
		return unavailableWithSnapshot(loader.projectRoot, status, specDir, request.AttachmentsEnabled(), evidencecontract.ErrorProviderUnavailable, "tracker provider is unavailable")
	}
	if evidence == nil {
		return unavailableWithSnapshot(loader.projectRoot, status, specDir, request.AttachmentsEnabled(), evidencecontract.ErrorProviderUnavailable, "tracker returned no evidence")
	}
	snapshotUpdatedAt := metadata.UpdatedAt
	if evidence.Normalized != nil && evidence.Normalized.UpdatedAt != "" {
		snapshotUpdatedAt = evidence.Normalized.UpdatedAt
	}
	status.TrackerUpdatedAt = snapshotUpdatedAt

	manifest, err := loader.writeSnapshot(ctx, loader.projectRoot, specDir, connection.Provider, linkedSpec.TrackerID, snapshotUpdatedAt, request.AttachmentsEnabled(), provider, evidence)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return failEvidenceStatus(status, evidencecontract.StateUnavailable, evidencecontract.ErrorCancelled, "evidence load cancelled", false)
		}
		return unavailableWithSnapshot(loader.projectRoot, status, specDir, request.AttachmentsEnabled(), evidencecontract.ErrorWriteFailed, "evidence snapshot could not be written")
	}
	state := evidencecontract.StateFetched
	if hadManifest {
		state = evidencecontract.StateRefreshed
	}
	return statusFromSnapshot(status, state, manifest, evidence, false)
}

func evidenceIssueMetadata(ctx context.Context, source Tracker, issueID string) (*Issue, error) {
	if contextual, ok := source.(ContextEvidenceTracker); ok {
		return contextual.GetEvidenceMetadataContext(ctx, issueID)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return source.GetIssue(issueID)
}

func getIssueEvidenceContext(ctx context.Context, source EvidenceTracker, issueID string) (*IssueEvidence, error) {
	if contextual, ok := source.(ContextEvidenceTracker); ok {
		return contextual.GetIssueEvidenceContext(ctx, issueID)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return source.GetIssueEvidence(issueID)
}

func downloadEvidenceAttachmentContext(ctx context.Context, source EvidenceTracker, contentURL string) ([]byte, error) {
	if contextual, ok := source.(ContextEvidenceTracker); ok {
		return contextual.DownloadEvidenceAttachmentContext(ctx, contentURL)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return source.DownloadEvidenceAttachment(contentURL)
}

// ReadSnapshot opens the private evidence body after Load returned a usable
// current/fetched/refreshed status. It is intentionally not part of MCP's
// bounded status response.
func (loader *EvidenceLoader) ReadSnapshot(status evidencecontract.Status) (*IssueEvidence, error) {
	if status.EvidencePath == "" {
		return nil, errors.New("evidence snapshot is unavailable")
	}
	path, err := workspaceEvidencePath(loader.projectRoot, status.EvidencePath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var evidence IssueEvidence
	if err := json.Unmarshal(data, &evidence); err != nil {
		return nil, err
	}
	return &evidence, nil
}

func findEvidenceSpec(heroDir, slug string) (*herospec.Spec, error) {
	if slug == "" {
		return nil, errors.New("spec slug is required")
	}
	specs, err := herospec.Discover(heroDir)
	if err != nil {
		return nil, err
	}
	for _, candidate := range specs {
		if candidate.Slug == slug {
			return candidate, nil
		}
	}
	return nil, errors.New("spec not found")
}

func evidenceLockFor(specDir string) *evidenceLock {
	value, _ := evidenceLocks.LoadOrStore(filepath.Clean(specDir), &evidenceLock{})
	return value.(*evidenceLock)
}

func failEvidenceStatus(status evidencecontract.Status, state evidencecontract.State, code evidencecontract.ErrorCode, message string, retryable bool) evidencecontract.Status {
	status.Status = state
	status.CacheHit = false
	status.Error = &evidencecontract.Error{Code: code, Message: message, Retryable: retryable}
	return status
}

func statusFromSnapshot(status evidencecontract.Status, state evidencecontract.State, manifest evidencecontract.Manifest, evidence *IssueEvidence, cacheHit bool) evidencecontract.Status {
	status.Status = state
	status.TrackerUpdatedAt = manifest.TrackerUpdatedAt
	status.ContentSHA256 = manifest.ContentSHA256
	status.AttachmentCount = manifest.AttachmentCount
	status.OmissionCount = manifest.OmissionCount
	status.CacheHit = cacheHit
	status.Error = nil
	if evidence != nil {
		status.AttachmentCount = len(evidence.Attachments)
		status.OmissionCount = len(evidence.Omissions)
	}
	return status
}

func unavailableWithSnapshot(projectRoot string, status evidencecontract.Status, specDir string, requireAttachments bool, code evidencecontract.ErrorCode, message string) evidencecontract.Status {
	manifest, evidence, err := validateEvidenceSnapshot(projectRoot, specDir, status.Provider, status.IssueID, "", requireAttachments)
	if err == nil {
		status = statusFromSnapshot(status, evidencecontract.StateUnavailable, manifest, evidence, false)
	}
	return failEvidenceStatus(status, evidencecontract.StateUnavailable, code, message, true)
}

func evidenceManifestExists(specDir string) bool {
	_, err := os.Stat(filepath.Join(specDir, evidenceManifestName))
	return err == nil
}

func validTrackerEvidenceTime(value string) bool {
	if value == "" {
		return false
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999-0700"} {
		if _, err := time.Parse(layout, value); err == nil {
			return true
		}
	}
	return false
}

func validateEvidenceSnapshot(projectRoot, specDir, provider, issueID, updatedAt string, requireAttachments bool) (evidencecontract.Manifest, *IssueEvidence, error) {
	manifestPath := filepath.Join(specDir, evidenceManifestName)
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return evidencecontract.Manifest{}, nil, err
	}
	var manifest evidencecontract.Manifest
	if json.Unmarshal(manifestData, &manifest) != nil || manifest.Version != evidencecontract.Version || manifest.Provider != provider || manifest.IssueID != issueID {
		return evidencecontract.Manifest{}, nil, errors.New("invalid evidence manifest")
	}
	if updatedAt != "" && (manifest.TrackerUpdatedAt != updatedAt || !validTrackerEvidenceTime(updatedAt)) {
		return evidencecontract.Manifest{}, nil, errors.New("stale evidence manifest")
	}
	evidencePath, err := workspaceEvidencePath(projectRoot, manifest.EvidencePath)
	if err != nil || evidencePath != filepath.Join(specDir, evidencePrivateName, evidencePayloadName) {
		return evidencecontract.Manifest{}, nil, errors.New("invalid evidence path")
	}
	evidenceData, err := os.ReadFile(evidencePath)
	if err != nil {
		return evidencecontract.Manifest{}, nil, err
	}
	var evidence IssueEvidence
	if json.Unmarshal(evidenceData, &evidence) != nil || evidence.Tracker != provider || evidence.IssueID != issueID {
		return evidencecontract.Manifest{}, nil, errors.New("invalid evidence payload")
	}
	attachmentFiles, err := evidenceAttachmentFiles(projectRoot, specDir, &evidence, requireAttachments)
	if err != nil {
		return evidencecontract.Manifest{}, nil, err
	}
	hash, err := evidenceSnapshotHash(evidenceData, attachmentFiles)
	if err != nil || hash != manifest.ContentSHA256 {
		return evidencecontract.Manifest{}, nil, errors.New("evidence payload hash mismatch")
	}
	return manifest, &evidence, nil
}

func evidenceAttachmentFiles(projectRoot, specDir string, evidence *IssueEvidence, requireAttachments bool) (map[string]string, error) {
	files := map[string]string{}
	privateDir := filepath.Join(specDir, evidencePrivateName)
	for _, attachment := range evidence.Attachments {
		if attachment.LocalPath == "" {
			if requireAttachments && attachment.Content != "" {
				return nil, errors.New("required evidence attachment is missing")
			}
			continue
		}
		path, err := workspaceEvidencePath(projectRoot, attachment.LocalPath)
		if err != nil || !pathWithin(privateDir, path) {
			return nil, errors.New("invalid evidence attachment path")
		}
		if _, err := os.Stat(path); err != nil {
			return nil, err
		}
		files[attachment.LocalPath] = path
	}
	return files, nil
}

func writeEvidenceSnapshot(ctx context.Context, projectRoot, specDir, provider, issueID, updatedAt string, includeAttachments bool, source EvidenceTracker, evidence *IssueEvidence) (evidencecontract.Manifest, error) {
	tempDir, err := os.MkdirTemp(specDir, evidencePrivateName+".tmp-")
	if err != nil {
		return evidencecontract.Manifest{}, err
	}
	if err := os.Chmod(tempDir, 0o700); err != nil {
		_ = os.RemoveAll(tempDir)
		return evidencecontract.Manifest{}, err
	}
	defer os.RemoveAll(tempDir)

	attachmentFiles := map[string]string{}
	if includeAttachments && len(evidence.Attachments) > 0 {
		attachmentsDir := filepath.Join(tempDir, "attachments")
		if err := os.MkdirAll(attachmentsDir, 0o700); err != nil {
			return evidencecontract.Manifest{}, err
		}
		for index := range evidence.Attachments {
			if err := ctx.Err(); err != nil {
				return evidencecontract.Manifest{}, err
			}
			attachment := &evidence.Attachments[index]
			if attachment.Content == "" {
				continue
			}
			data, err := downloadEvidenceAttachmentContext(ctx, source, attachment.Content)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return evidencecontract.Manifest{}, err
				}
				evidence.Omissions = append(evidence.Omissions, "attachment "+attachment.ID+": download unavailable")
				continue
			}
			filename := safeEvidenceAttachmentName(attachment)
			tempPath := filepath.Join(attachmentsDir, filename)
			if err := herospec.AtomicWriteFile(tempPath, data, 0o600); err != nil {
				return evidencecontract.Manifest{}, err
			}
			finalRelative := relativeEvidencePath(projectRoot, filepath.Join(specDir, evidencePrivateName, "attachments", filename))
			attachment.LocalPath = finalRelative
			attachmentFiles[finalRelative] = tempPath
		}
	}
	if err := ctx.Err(); err != nil {
		return evidencecontract.Manifest{}, err
	}
	evidenceData, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return evidencecontract.Manifest{}, err
	}
	evidenceData = append(evidenceData, '\n')
	if err := herospec.AtomicWriteFile(filepath.Join(tempDir, evidencePayloadName), evidenceData, 0o600); err != nil {
		return evidencecontract.Manifest{}, err
	}
	contentHash, err := evidenceSnapshotHash(evidenceData, attachmentFiles)
	if err != nil {
		return evidencecontract.Manifest{}, err
	}
	manifest := evidencecontract.Manifest{
		Version: evidencecontract.Version, Provider: provider, IssueID: issueID,
		TrackerUpdatedAt: updatedAt, ContentSHA256: contentHash,
		EvidencePath:    relativeEvidencePath(projectRoot, filepath.Join(specDir, evidencePrivateName, evidencePayloadName)),
		AttachmentCount: len(evidence.Attachments), OmissionCount: len(evidence.Omissions), RetrievedAt: evidence.RetrievedAt,
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return evidencecontract.Manifest{}, err
	}
	manifestData = append(manifestData, '\n')
	if err := publishEvidenceSnapshot(ctx, specDir, tempDir, manifestData); err != nil {
		return evidencecontract.Manifest{}, err
	}
	return manifest, nil
}

type evidencePublishOps struct {
	stat          func(string) (os.FileInfo, error)
	rename        func(string, string) error
	removeAll     func(string) error
	writeManifest func(string, []byte, os.FileMode) error
}

var defaultEvidencePublishOps = evidencePublishOps{
	stat: os.Stat, rename: os.Rename, removeAll: os.RemoveAll, writeManifest: herospec.AtomicWriteFile,
}

func publishEvidenceSnapshot(ctx context.Context, specDir, candidateDir string, manifestData []byte) error {
	return publishEvidenceSnapshotWithOps(ctx, specDir, candidateDir, manifestData, defaultEvidencePublishOps)
}

func publishEvidenceSnapshotWithOps(ctx context.Context, specDir, candidateDir string, manifestData []byte, ops evidencePublishOps) error {
	privateDir := filepath.Join(specDir, evidencePrivateName)
	backupDir := filepath.Join(specDir, evidencePrivateName+".backup")
	_ = ops.removeAll(backupDir)
	if err := ctx.Err(); err != nil {
		return err
	}
	hadPrivate := false
	if _, err := ops.stat(privateDir); err == nil {
		if err := ops.rename(privateDir, backupDir); err != nil {
			return err
		}
		hadPrivate = true
	}
	rollback := func() {
		_ = ops.removeAll(privateDir)
		if hadPrivate {
			_ = ops.rename(backupDir, privateDir)
		}
	}
	if err := ctx.Err(); err != nil {
		rollback()
		return err
	}
	if err := ops.rename(candidateDir, privateDir); err != nil {
		if hadPrivate {
			_ = ops.rename(backupDir, privateDir)
		}
		return err
	}
	if err := ctx.Err(); err != nil {
		rollback()
		return err
	}
	manifestPath := filepath.Join(specDir, evidenceManifestName)
	if err := ops.writeManifest(manifestPath, manifestData, 0o600); err != nil {
		rollback()
		return err
	}
	_ = ops.removeAll(backupDir)
	return nil
}

func recoverEvidenceStore(projectRoot, specDir string) {
	privateDir := filepath.Join(specDir, evidencePrivateName)
	backupDir := filepath.Join(specDir, evidencePrivateName+".backup")
	if _, backupErr := os.Stat(backupDir); backupErr == nil {
		manifestData, manifestErr := os.ReadFile(filepath.Join(specDir, evidenceManifestName))
		var manifest evidencecontract.Manifest
		currentCommitted := manifestErr == nil && json.Unmarshal(manifestData, &manifest) == nil
		if currentCommitted {
			_, _, currentErr := validateEvidenceSnapshot(projectRoot, specDir, manifest.Provider, manifest.IssueID, "", false)
			currentCommitted = currentErr == nil
		}
		if currentCommitted {
			_ = os.RemoveAll(backupDir)
		} else {
			_ = os.RemoveAll(privateDir)
			_ = os.Rename(backupDir, privateDir)
		}
	}
	matches, _ := filepath.Glob(filepath.Join(specDir, evidencePrivateName+".tmp-*"))
	for _, match := range matches {
		_ = os.RemoveAll(match)
	}
}

func evidenceSnapshotHash(evidenceData []byte, attachments map[string]string) (string, error) {
	hash := sha256.New()
	_, _ = hash.Write([]byte(evidencecontract.Version + "\x00evidence\x00"))
	_, _ = hash.Write(evidenceData)
	paths := make([]string, 0, len(attachments))
	for relative := range attachments {
		paths = append(paths, relative)
	}
	sort.Strings(paths)
	for _, relative := range paths {
		data, err := os.ReadFile(attachments[relative])
		if err != nil {
			return "", err
		}
		_, _ = hash.Write([]byte("\x00attachment\x00" + relative + "\x00"))
		_, _ = hash.Write(data)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func safeEvidenceAttachmentName(attachment *EvidenceAttachment) string {
	sum := sha256.Sum256([]byte(attachment.ID + "\x00" + attachment.Filename))
	extension := strings.ToLower(filepath.Ext(attachment.Filename))
	if len(extension) > 12 || strings.Trim(extension, ".abcdefghijklmnopqrstuvwxyz0123456789") != "" {
		extension = ""
	}
	return "attachment-" + hex.EncodeToString(sum[:8]) + extension
}

func relativeEvidencePath(projectRoot, path string) string {
	relative, err := filepath.Rel(projectRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}

func workspaceEvidencePath(projectRoot, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) {
		return "", errors.New("evidence path must be workspace-relative")
	}
	path := filepath.Clean(filepath.Join(projectRoot, filepath.FromSlash(relative)))
	if !pathWithin(projectRoot, path) {
		return "", errors.New("evidence path escapes workspace")
	}
	return path, nil
}

func pathWithin(parent, child string) bool {
	relative, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
