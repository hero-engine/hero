// Package mailquery provides the registry-backed Mail-read v1 authority used
// by Hero Serve. Mail services remain the sole owners of envelope and receipt
// reads and mutations.
package mailquery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailread"
	"github.com/hero-engine/hero/contracts/attention/mailthread"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/projectregistry"
)

type mailService interface {
	Inbox(project string, unread bool) ([]mail.ListedMessage, error)
	AllMessages(projectPeerID string) ([]mail.ListedMessage, error)
	Show(id string, markRead bool) (mail.ListedMessage, error)
	Thread(projectPeerID, threadID string) (mailthread.ThreadView, bool, error)
	ThreadAction(mailthread.ActionRequest) (mailthread.ThreadView, error)
	Action(mail.ActionRequest) (mail.ActionResult, error)
	Reply(mail.ReplyRequest) (attention.MailDelivery, error)
}

func (s *Service) ThreadAction(request mailthread.ActionRequest) mailthread.ActionResponse {
	fail := func(err *attention.ContractError) mailthread.ActionResponse {
		return mailthread.ActionResponse{SchemaVersion: mailthread.SchemaVersion, Error: err}
	}
	if contractErr := mailthread.ValidateActionRequest(request); contractErr != nil {
		return fail(contractErr)
	}
	if !validTargetID(request.Identity.ProjectPeerID) || !validTargetID(request.Identity.ThreadID) {
		return fail(validation("thread identity is invalid", "identity"))
	}
	source, contractErr := s.resolveOne(request.Identity.ProjectPeerID)
	if contractErr != nil {
		return fail(contractErr)
	}
	view, err := source.mail.ThreadAction(request)
	if err != nil {
		return fail(translateSourceError(err, "thread_revision"))
	}
	return mailthread.ActionResponse{SchemaVersion: mailthread.SchemaVersion, Thread: &view}
}

type projectSource struct {
	project       attention.ProjectReference
	canonicalPath string
	mail          mailService
}

const maxRegistryIssueDiagnostics = 8

type registryIssue struct {
	slug     string
	category string
}

type registryResolution struct {
	sources      []projectSource
	issues       []registryIssue
	skippedCount int
}

func (r *registryResolution) addIssue(slug, category string) {
	r.skippedCount++
	if len(r.issues) >= maxRegistryIssueDiagnostics {
		return
	}
	r.issues = append(r.issues, registryIssue{
		slug:     safeRegistrySlug(slug),
		category: category,
	})
}

func (r registryResolution) unavailable() *attention.ContractError {
	details := map[string]string{
		"category":      "registry",
		"skipped_count": strconv.Itoa(r.skippedCount),
	}
	if len(r.issues) > 0 {
		values := make([]string, len(r.issues))
		for i, issue := range r.issues {
			values[i] = issue.slug + ":" + issue.category
		}
		details["skipped_entries"] = strings.Join(values, ",")
	}
	return &attention.ContractError{
		Code:    attention.ErrorUnavailable,
		Message: "registered Mail projects could not be resolved",
		Field:   "project_peer_id",
		Details: details,
	}
}

// Service resolves the current machine registry and delegates to Mail's
// source-owned services. It stores no projection or pagination state.
type Service struct {
	store    *mail.Store
	registry *projectregistry.Registry
}

func NewService(stateRoot string, registry *projectregistry.Registry) (*Service, error) {
	if registry == nil {
		return nil, errors.New("project registry is unavailable")
	}
	store, err := mail.NewStore(stateRoot)
	if err != nil {
		return nil, err
	}
	return &Service{store: store, registry: registry}, nil
}

func (s *Service) List(request mailread.ListRequest) mailread.ListResponse {
	fail := func(err *attention.ContractError) mailread.ListResponse {
		return mailread.ListResponse{SchemaVersion: mailread.SchemaVersion, Error: err}
	}
	if contractErr := mailread.ValidateListRequest(request); contractErr != nil {
		return fail(contractErr)
	}
	if request.ProjectPeerID != "" && !validTargetID(request.ProjectPeerID) {
		return fail(validation("project_peer_id is invalid", "project_peer_id"))
	}
	limit := request.Limit
	if limit == 0 {
		limit = mailread.DefaultListLimit
	}
	resolution, contractErr := s.resolveRegistry(request.ProjectPeerID)
	if contractErr != nil {
		return fail(contractErr)
	}
	sources := resolution.sources

	type orderedSummary struct {
		summary mailread.MessageSummary
		instant time.Time
	}
	ordered := make([]orderedSummary, 0)
	for _, source := range sources {
		items, err := source.mail.Inbox("", false)
		if err != nil {
			return fail(unavailable(fmt.Errorf("read registered project %q mailbox: %w", source.project.RegistrySlug, err)))
		}
		for _, item := range items {
			if item.Recipient.PeerID != source.project.PeerID {
				return fail(unavailable(fmt.Errorf("mailbox %q contains recipient %q", source.project.PeerID, item.Recipient.PeerID)))
			}
			if request.ThreadID != "" && item.ThreadID != request.ThreadID {
				continue
			}
			summary, instant, err := summarize(source.project, item)
			if err != nil {
				return fail(unavailable(err))
			}
			if request.UnreadOnly != nil && *request.UnreadOnly && !summary.Unread {
				continue
			}
			ordered = append(ordered, orderedSummary{summary: summary, instant: instant})
		}
	}
	sort.Slice(ordered, func(i, j int) bool {
		if !ordered[i].instant.Equal(ordered[j].instant) {
			return ordered[i].instant.After(ordered[j].instant)
		}
		if ordered[i].summary.Project.PeerID != ordered[j].summary.Project.PeerID {
			return ordered[i].summary.Project.PeerID < ordered[j].summary.Project.PeerID
		}
		return ordered[i].summary.MessageID < ordered[j].summary.MessageID
	})
	summaries := make([]mailread.MessageSummary, len(ordered))
	unreadCount := 0
	for i := range ordered {
		summaries[i] = ordered[i].summary
		if summaries[i].Unread {
			unreadCount++
		}
	}
	revision := summaryRevision(summaries)
	filters := cursorFilters{ProjectPeerID: request.ProjectPeerID, ThreadID: request.ThreadID, UnreadOnly: request.UnreadOnly}
	start := 0
	if request.Cursor != "" {
		cursor, err := decodeCursor(request.Cursor)
		if err != nil {
			return fail(validation(err.Error(), "cursor"))
		}
		if !sameFilters(cursor.Filters, filters) || cursor.Revision != revision {
			return fail(stale("cursor does not match the current filters and Mail revision", "cursor"))
		}
		start = cursorPosition(summaries, cursor)
		if start < 0 {
			return fail(stale("cursor position is no longer available", "cursor"))
		}
	}
	end := start + limit
	if end > len(summaries) {
		end = len(summaries)
	}
	items := make([]mailread.MessageSummary, end-start)
	copy(items, summaries[start:end])
	response := mailread.ListResponse{
		SchemaVersion: mailread.SchemaVersion,
		Revision:      revision,
		TotalCount:    len(summaries),
		UnreadCount:   unreadCount,
		Items:         items,
		Page:          &mailread.PageMetadata{Limit: limit, Returned: len(items), HasMore: end < len(summaries)},
	}
	if request.ProjectPeerID == "" && resolution.skippedCount > 0 {
		response.Diagnostics = []attention.ContractError{*resolution.unavailable()}
	}
	if response.Page.HasMore {
		last := items[len(items)-1]
		cursor, err := encodeCursor(messageCursor{
			Filters: filters, Revision: revision, ActivityAt: last.ActivityAt,
			PeerID: last.Project.PeerID, MessageID: last.MessageID,
		})
		if err != nil {
			return fail(unavailable(err))
		}
		response.NextCursor = cursor
	}
	return response
}

func (s *Service) Detail(projectPeerID, messageID string) mailread.DetailResponse {
	fail := func(err *attention.ContractError) mailread.DetailResponse {
		return mailread.DetailResponse{SchemaVersion: mailread.SchemaVersion, Error: err}
	}
	if strings.TrimSpace(projectPeerID) == "" {
		return fail(validation("project_peer_id is required", "project_peer_id"))
	}
	if !validTargetID(projectPeerID) {
		return fail(validation("project_peer_id is invalid", "project_peer_id"))
	}
	if strings.TrimSpace(messageID) == "" {
		return fail(validation("message_id is required", "message_id"))
	}
	if !validMessageID(messageID) {
		return fail(validation("message_id is invalid", "message_id"))
	}
	source, contractErr := s.resolveOne(projectPeerID)
	if contractErr != nil {
		return fail(contractErr)
	}
	item, err := source.mail.Show(messageID, false)
	if err != nil {
		return fail(translateSourceError(err, "message_id"))
	}
	if item.Recipient.PeerID != projectPeerID {
		return fail(unavailable(errors.New("mail recipient does not match its registered mailbox")))
	}
	summary, _, err := summarize(source.project, item)
	if err != nil {
		return fail(unavailable(err))
	}
	envelope := item.MailEnvelope
	project := source.project
	unread := summary.Unread
	receipt := summary.Receipt
	return mailread.DetailResponse{
		SchemaVersion: mailread.SchemaVersion, Project: &project, Envelope: &envelope,
		ActivityAt: summary.ActivityAt, Unread: &unread, Receipt: &receipt, Actions: summary.Actions,
	}
}

func (s *Service) Action(request mailread.ActionRequest) mailread.ActionResponse {
	fail := func(err *attention.ContractError) mailread.ActionResponse {
		return mailread.ActionResponse{SchemaVersion: mailread.SchemaVersion, Error: err}
	}
	if contractErr := mailread.ValidateActionRequest(request); contractErr != nil {
		return fail(contractErr)
	}
	if !validTargetID(request.ProjectPeerID) {
		return fail(validation("project_peer_id is invalid", "project_peer_id"))
	}
	if !validMessageID(request.MessageID) {
		return fail(validation("message_id is invalid", "message_id"))
	}
	source, contractErr := s.resolveOne(request.ProjectPeerID)
	if contractErr != nil {
		return fail(contractErr)
	}
	sourceAction, ok := mail.SourceActionID(request.ActionID)
	if !ok {
		return fail(&attention.ContractError{Code: attention.ErrorUnsupported, Message: "action is not advertised for Mail", Field: "action_id"})
	}
	input, inputErr := actionInput(request.ActionID, request.Input)
	if inputErr != nil {
		return fail(inputErr)
	}
	result, err := source.mail.Action(mail.ActionRequest{
		MessageID: request.MessageID, Action: sourceAction, ExpectedRevision: request.ReceiptRevision,
		IdempotencyKey: request.IdempotencyKey, Note: input.Note, ArtifactType: input.ArtifactType,
	})
	if err != nil {
		return fail(translateSourceError(err, "receipt_revision"))
	}
	receipt := normalizeReceipt(&result.Receipt)
	project := source.project
	response := mailread.ActionResponse{
		SchemaVersion: mailread.SchemaVersion, Project: &project, MessageID: result.MessageID,
		Receipt: &receipt, Actions: mail.Capabilities(receipt.Revision),
	}
	if result.Navigation != nil {
		response.Navigation = &attention.NavigationReference{Project: result.Project, Target: result.Navigation.Slug}
	}
	return response
}

func (s *Service) Reply(request mailread.ReplyRequest) mailread.ReplyResponse {
	fail := func(err *attention.ContractError) mailread.ReplyResponse {
		return mailread.ReplyResponse{SchemaVersion: mailread.SchemaVersion, Error: err}
	}
	if contractErr := mailread.ValidateReplyRequest(request); contractErr != nil {
		return fail(contractErr)
	}
	if !validTargetID(request.ProjectPeerID) {
		return fail(validation("project_peer_id is invalid", "project_peer_id"))
	}
	if !validMessageID(request.MessageID) {
		return fail(validation("message_id is invalid", "message_id"))
	}
	source, contractErr := s.resolveOne(request.ProjectPeerID)
	if contractErr != nil {
		return fail(contractErr)
	}
	delivery, err := source.mail.Reply(mail.ReplyRequest{
		MessageID: request.MessageID, ExpectedThread: request.ThreadID, Body: request.Body,
		Subject: request.Subject, Kind: request.Kind, IdempotencyKey: request.IdempotencyKey,
	})
	if err != nil {
		return fail(translateSourceError(err, "thread_id"))
	}
	return mailread.ReplyResponse{SchemaVersion: mailread.SchemaVersion, Delivery: &delivery}
}

func (s *Service) resolveOne(peerID string) (projectSource, *attention.ContractError) {
	sources, contractErr := s.resolve(peerID)
	if contractErr != nil {
		return projectSource{}, contractErr
	}
	return sources[0], nil
}

func (s *Service) resolve(peerID string) ([]projectSource, *attention.ContractError) {
	resolution, contractErr := s.resolveRegistry(peerID)
	return resolution.sources, contractErr
}

func (s *Service) resolveRegistry(peerID string) (registryResolution, *attention.ContractError) {
	var resolution registryResolution
	if s == nil || s.registry == nil || s.store == nil {
		return resolution, unavailable(errors.New("Mail query service is unavailable"))
	}
	entries := s.registry.List()
	slugs := make([]string, 0, len(entries))
	for slug := range entries {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	byPeer := make(map[string]projectSource, len(slugs))
	for _, slug := range slugs {
		entry := entries[slug]
		canonicalPath, err := canonicalProjectPath(entry.Path)
		if err != nil {
			resolution.addIssue(slug, "path")
			continue
		}
		cfg, err := config.Load(canonicalPath)
		if err != nil {
			resolution.addIssue(slug, "config")
			continue
		}
		if !validTargetID(cfg.PeerID) {
			resolution.addIssue(slug, "identity")
			continue
		}
		displayName := filepath.Base(canonicalPath)
		if cfg.Peering != nil && strings.TrimSpace(cfg.Peering.Display) != "" {
			displayName = cfg.Peering.Display
		}
		candidate := projectSource{
			project:       attention.ProjectReference{PeerID: cfg.PeerID, RegistrySlug: slug, DisplayName: displayName},
			canonicalPath: canonicalPath,
			mail:          mail.NewService(s.store, canonicalPath, cfg),
		}
		if existing, ok := byPeer[cfg.PeerID]; ok {
			if existing.canonicalPath != canonicalPath {
				return resolution, &attention.ContractError{
					Code: attention.ErrorUnavailable, Message: "multiple registered project paths claim one peer_id",
					Field: "project_peer_id", Details: map[string]string{"project_peer_id": cfg.PeerID},
				}
			}
			continue // sorted slugs preserve the lexicographically smallest alias
		}
		byPeer[cfg.PeerID] = candidate
	}
	if peerID != "" {
		source, ok := byPeer[peerID]
		if !ok {
			if resolution.skippedCount > 0 {
				return resolution, resolution.unavailable()
			}
			return resolution, &attention.ContractError{Code: attention.ErrorMissing, Message: "registered project peer_id was not found", Field: "project_peer_id"}
		}
		resolution.sources = []projectSource{source}
		return resolution, nil
	}
	if len(byPeer) == 0 && resolution.skippedCount > 0 {
		return resolution, resolution.unavailable()
	}
	peerIDs := make([]string, 0, len(byPeer))
	for id := range byPeer {
		peerIDs = append(peerIDs, id)
	}
	sort.Strings(peerIDs)
	resolution.sources = make([]projectSource, 0, len(peerIDs))
	for _, id := range peerIDs {
		resolution.sources = append(resolution.sources, byPeer[id])
	}
	return resolution, nil
}

func safeRegistrySlug(slug string) string {
	if len(slug) == 0 || len(slug) > 64 || !validTargetID(slug) {
		return "invalid-slug"
	}
	return slug
}

func canonicalProjectPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(filepath.Clean(absolute))
}

func validMessageID(value string) bool {
	if !strings.HasPrefix(value, "mail_") || len(value) == len("mail_") {
		return false
	}
	for _, r := range value[len("mail_"):] {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

func validTargetID(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

func summarize(project attention.ProjectReference, item mail.ListedMessage) (mailread.MessageSummary, time.Time, error) {
	instant, err := time.Parse(time.RFC3339Nano, item.CreatedAt)
	if err != nil {
		return mailread.MessageSummary{}, time.Time{}, fmt.Errorf("parse Mail created_at for %s/%s: %w", project.PeerID, item.ID, err)
	}
	receipt := normalizeReceipt(item.Receipt)
	return mailread.MessageSummary{
		Project: project, MessageID: item.ID, ThreadID: item.ThreadID, InReplyTo: item.InReplyTo,
		Sender: item.Sender, Recipient: item.Recipient, Subject: item.Subject, Kind: item.Kind,
		CreatedAt: item.CreatedAt, ActivityAt: item.CreatedAt, Unread: receipt.Unread,
		Receipt: receipt, Actions: mail.Capabilities(receipt.Revision),
	}, instant, nil
}

func normalizeReceipt(receipt *attention.MailReceipt) mailread.ReceiptView {
	if receipt == nil {
		return mailread.ReceiptView{Unread: true}
	}
	return mailread.ReceiptView{
		Revision: receipt.Revision, Unread: receipt.ReadAt == "", ReadAt: receipt.ReadAt,
		AcknowledgedAt: receipt.AcknowledgedAt, DismissedAt: receipt.DismissedAt,
		PromotedArtifact: receipt.PromotedArtifact, FocusItemID: receipt.FocusItemID,
	}
}

func summaryRevision(summaries []mailread.MessageSummary) string {
	hash := sha256.New()
	encoder := json.NewEncoder(hash)
	encoder.SetEscapeHTML(false)
	for _, summary := range summaries {
		actionIDs := make([]string, len(summary.Actions))
		for i := range summary.Actions {
			actionIDs[i] = summary.Actions[i].ID
		}
		_ = encoder.Encode(struct {
			PeerID          string   `json:"peer_id"`
			MessageID       string   `json:"message_id"`
			ActivityAt      string   `json:"activity_at"`
			ReceiptRevision int64    `json:"receipt_revision"`
			Unread          bool     `json:"unread"`
			Actions         []string `json:"actions"`
		}{summary.Project.PeerID, summary.MessageID, summary.ActivityAt, summary.Receipt.Revision, summary.Unread, actionIDs})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func cursorPosition(summaries []mailread.MessageSummary, cursor messageCursor) int {
	for i := range summaries {
		if summaries[i].ActivityAt == cursor.ActivityAt && summaries[i].Project.PeerID == cursor.PeerID && summaries[i].MessageID == cursor.MessageID {
			return i + 1
		}
	}
	return -1
}

type decodedActionInput struct {
	Note         string
	ArtifactType string
}

func actionInput(actionID string, raw json.RawMessage) (decodedActionInput, *attention.ContractError) {
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return decodedActionInput{}, validation("input must be a JSON object", "input")
	}
	allowed := map[string]bool{}
	if actionID == mailread.ActionAcknowledge {
		allowed["note"] = true
	}
	if actionID == mailread.ActionPromote {
		allowed["artifact_type"] = true
	}
	for key := range values {
		if !allowed[key] {
			return decodedActionInput{}, validation("input field is not allowed", "input."+key)
		}
	}
	var input decodedActionInput
	if rawNote, ok := values["note"]; ok && json.Unmarshal(rawNote, &input.Note) != nil {
		return decodedActionInput{}, validation("note must be a string", "input.note")
	}
	if rawArtifact, ok := values["artifact_type"]; ok && json.Unmarshal(rawArtifact, &input.ArtifactType) != nil {
		return decodedActionInput{}, validation("artifact_type must be a string", "input.artifact_type")
	}
	if actionID == mailread.ActionPromote && strings.TrimSpace(input.ArtifactType) == "" {
		return decodedActionInput{}, validation("artifact_type is required", "input.artifact_type")
	}
	return input, nil
}

func translateSourceError(err error, staleField string) *attention.ContractError {
	var contractErr *attention.ContractError
	if errors.As(err, &contractErr) {
		return contractErr
	}
	switch {
	case errors.Is(err, mail.ErrStale), errors.Is(err, mail.ErrThreadMismatch):
		return stale(err.Error(), staleField)
	case errors.Is(err, mail.ErrNotFound), errors.Is(err, mail.ErrRecipientMissing):
		return &attention.ContractError{Code: attention.ErrorMissing, Message: err.Error(), Field: "message_id"}
	case errors.Is(err, mail.ErrUnsupportedAction):
		return &attention.ContractError{Code: attention.ErrorUnsupported, Message: err.Error(), Field: "action_id"}
	case errors.Is(err, mail.ErrIdempotencyConflict):
		return &attention.ContractError{Code: attention.ErrorIdempotencyConflict, Message: err.Error(), Field: "idempotency_key"}
	default:
		return unavailable(err)
	}
}

func validation(message, field string) *attention.ContractError {
	return &attention.ContractError{Code: attention.ErrorValidation, Message: message, Field: field}
}

func stale(message, field string) *attention.ContractError {
	return &attention.ContractError{Code: attention.ErrorStale, Message: message, Field: field}
}

func unavailable(err error) *attention.ContractError {
	return &attention.ContractError{Code: attention.ErrorUnavailable, Message: err.Error()}
}
