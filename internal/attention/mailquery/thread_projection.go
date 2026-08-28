package mailquery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailthread"
	"github.com/hero-engine/hero/internal/attention/mail"
)

type projectedThread struct {
	summary  mailthread.ThreadSummary
	view     mailthread.ThreadView
	messages []mailthread.MessageView
	instant  time.Time
}

func (s *Service) Threads(request mailthread.ThreadListRequest) mailthread.ThreadListResponse {
	fail := func(err *attention.ContractError) mailthread.ThreadListResponse {
		return mailthread.ThreadListResponse{SchemaVersion: mailthread.SchemaVersion, Error: err}
	}
	if contractErr := mailthread.ValidateThreadListRequest(request); contractErr != nil {
		return fail(contractErr)
	}
	if request.ProjectPeerID != "" && !validTargetID(request.ProjectPeerID) {
		return fail(validation("project_peer_id is invalid", "project_peer_id"))
	}
	limit := request.Limit
	if limit == 0 {
		limit = mailthread.DefaultListLimit
	}
	sources, contractErr := s.resolve(request.ProjectPeerID)
	if contractErr != nil {
		return fail(contractErr)
	}
	all, err := projectThreads(sources)
	if err != nil {
		return fail(unavailable(err))
	}
	counts := mailthread.ThreadCounts{Total: len(all)}
	for _, item := range all {
		if item.summary.Actionable {
			counts.Actionable++
			if item.summary.Unread {
				counts.ActionableUnread++
			}
		}
	}
	revision := threadRevision(all)
	filters := threadCursorFilters{ProjectPeerID: request.ProjectPeerID, Bucket: request.Bucket, Lifecycle: request.Lifecycle}
	filtered := make([]projectedThread, 0, len(all))
	for _, item := range all {
		if request.Bucket != "" && item.summary.Bucket != request.Bucket {
			continue
		}
		if request.Lifecycle != "" && item.summary.Lifecycle != request.Lifecycle {
			continue
		}
		filtered = append(filtered, item)
	}
	start := 0
	if request.Cursor != "" {
		cursor, err := decodeThreadCursor(request.Cursor)
		if err != nil {
			return fail(validation(err.Error(), "cursor"))
		}
		if cursor.Filters != filters || cursor.Revision != revision {
			return fail(stale("cursor does not match the current filters and Mail thread revision", "cursor"))
		}
		start = threadCursorPosition(filtered, cursor)
		if start < 0 {
			return fail(stale("cursor position is no longer available", "cursor"))
		}
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	items := make([]mailthread.ThreadSummary, end-start)
	for i := start; i < end; i++ {
		items[i-start] = filtered[i].summary
	}
	response := mailthread.ThreadListResponse{
		SchemaVersion: mailthread.SchemaVersion, Revision: revision, Counts: counts, Items: items,
		Page: &mailthread.PageMetadata{Limit: limit, Returned: len(items), HasMore: end < len(filtered)},
	}
	if response.Page.HasMore {
		last := items[len(items)-1]
		cursor, err := encodeThreadCursor(threadCursor{Filters: filters, Revision: revision, ActivityAt: last.ActivityAt, PeerID: last.Identity.ProjectPeerID, ThreadID: last.Identity.ThreadID})
		if err != nil {
			return fail(unavailable(err))
		}
		response.NextCursor = cursor
	}
	return response
}

func (s *Service) ThreadDetail(projectPeerID, threadID string) mailthread.ThreadDetailResponse {
	fail := func(err *attention.ContractError) mailthread.ThreadDetailResponse {
		return mailthread.ThreadDetailResponse{SchemaVersion: mailthread.SchemaVersion, Error: err}
	}
	if !validTargetID(projectPeerID) {
		return fail(validation("project_peer_id is invalid", "project_peer_id"))
	}
	if !validTargetID(threadID) {
		return fail(validation("thread_id is invalid", "thread_id"))
	}
	source, contractErr := s.resolveOne(projectPeerID)
	if contractErr != nil {
		return fail(contractErr)
	}
	threads, err := projectThreads([]projectSource{source})
	if err != nil {
		return fail(unavailable(err))
	}
	for _, item := range threads {
		if item.summary.Identity.ThreadID == threadID {
			summary, view := item.summary, item.view
			return mailthread.ThreadDetailResponse{SchemaVersion: mailthread.SchemaVersion, Summary: &summary, Thread: &view, Messages: item.messages}
		}
	}
	return fail(&attention.ContractError{Code: attention.ErrorMissing, Message: "Mail thread was not found", Field: "thread_id"})
}

func projectThreads(sources []projectSource) ([]projectedThread, error) {
	result := make([]projectedThread, 0)
	for _, source := range sources {
		items, err := source.mail.AllMessages(source.project.PeerID)
		if err != nil {
			return nil, fmt.Errorf("read registered project %q mailbox: %w", source.project.RegistrySlug, err)
		}
		grouped := map[string][]mail.ListedMessage{}
		for _, item := range items {
			if item.Recipient.PeerID != source.project.PeerID {
				return nil, fmt.Errorf("mailbox %q contains recipient %q", source.project.PeerID, item.Recipient.PeerID)
			}
			threadID := item.ThreadID
			if threadID == "" {
				threadID = item.ID
			}
			grouped[threadID] = append(grouped[threadID], item)
		}
		threadIDs := make([]string, 0, len(grouped))
		for threadID := range grouped {
			threadIDs = append(threadIDs, threadID)
		}
		sort.Strings(threadIDs)
		for _, threadID := range threadIDs {
			view, _, err := source.mail.Thread(source.project.PeerID, threadID)
			if err != nil {
				return nil, fmt.Errorf("read Mail thread %s/%s: %w", source.project.PeerID, threadID, err)
			}
			projected, err := summarizeThread(source.project, view, grouped[threadID])
			if err != nil {
				return nil, err
			}
			result = append(result, projected)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if !result[i].instant.Equal(result[j].instant) {
			return result[i].instant.After(result[j].instant)
		}
		if result[i].summary.Identity.ProjectPeerID != result[j].summary.Identity.ProjectPeerID {
			return result[i].summary.Identity.ProjectPeerID < result[j].summary.Identity.ProjectPeerID
		}
		return result[i].summary.Identity.ThreadID < result[j].summary.Identity.ThreadID
	})
	return result, nil
}

func summarizeThread(project attention.ProjectReference, view mailthread.ThreadView, items []mail.ListedMessage) (projectedThread, error) {
	sort.Slice(items, func(i, j int) bool {
		left, leftErr := time.Parse(time.RFC3339Nano, items[i].CreatedAt)
		right, rightErr := time.Parse(time.RFC3339Nano, items[j].CreatedAt)
		if leftErr == nil && rightErr == nil && !left.Equal(right) {
			return left.Before(right)
		}
		return items[i].ID < items[j].ID
	})
	latest := items[len(items)-1]
	listed := latest
	actionable := false
	for i := len(items) - 1; i >= 0; i-- {
		if actionableKind(items[i].Kind) {
			listed = items[i]
			actionable = true
			break
		}
		if !statusKind(items[i].Kind) {
			listed = items[i]
			break
		}
	}
	activity, err := time.Parse(time.RFC3339Nano, latest.CreatedAt)
	if err != nil {
		return projectedThread{}, fmt.Errorf("parse Mail created_at for %s/%s: %w", project.PeerID, latest.ID, err)
	}
	for _, event := range view.State.Events {
		instant, parseErr := time.Parse(time.RFC3339Nano, event.AppliedAt)
		if parseErr != nil {
			return projectedThread{}, parseErr
		}
		if instant.After(activity) {
			activity = instant
		}
	}
	actionable = view.State.Lifecycle == mailthread.LifecycleOpen && actionable
	bucket := mailthread.BucketUpdates
	if view.State.Lifecycle == mailthread.LifecycleArchived {
		bucket = mailthread.BucketHistory
	} else if actionable {
		bucket = mailthread.BucketNeedsAttention
	}
	messages := make([]mailthread.MessageView, len(items))
	for i := range items {
		messages[i] = mailthread.MessageView{Envelope: items[i].MailEnvelope, Receipt: items[i].Receipt, Unread: items[i].Receipt == nil || items[i].Receipt.ReadAt == ""}
	}
	summary := mailthread.ThreadSummary{
		Identity: view.State.Identity, Project: project, Sender: listed.Sender,
		Subject: listed.Subject, Kind: listed.Kind,
		ActivityAt: activity.UTC().Format(time.RFC3339Nano), Unread: view.Read.UnreadCount > 0,
		Actionable: actionable, Lifecycle: view.State.Lifecycle, Bucket: bucket,
		MessageCount: view.Read.MessageCount, UnreadCount: view.Read.UnreadCount,
		Revision: view.State.Revision, Actions: append([]attention.ActionDescriptor(nil), view.Actions...),
	}
	if invalid := mailthread.ValidateThreadSummary(summary); invalid != nil {
		return projectedThread{}, invalid
	}
	return projectedThread{summary: summary, view: view, messages: messages, instant: activity}, nil
}

func actionableKind(kind string) bool {
	switch kind {
	case attention.MailKindQuestion, attention.MailKindRequest, "peer.advisory", "peer.spec_out", "peer.work_transfer":
		return true
	case attention.MailKindResponse, attention.MailKindNotice:
		return false
	default:
		return false
	}
}

func statusKind(kind string) bool {
	return kind == attention.MailKindResponse || kind == attention.MailKindNotice || strings.HasSuffix(kind, ".response")
}

func threadRevision(items []projectedThread) string {
	hash := sha256.New()
	encoder := json.NewEncoder(hash)
	encoder.SetEscapeHTML(false)
	for _, item := range items {
		_ = encoder.Encode(item.summary)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func threadCursorPosition(items []projectedThread, cursor threadCursor) int {
	for i := range items {
		if items[i].summary.ActivityAt == cursor.ActivityAt && items[i].summary.Identity.ProjectPeerID == cursor.PeerID && items[i].summary.Identity.ThreadID == cursor.ThreadID {
			return i + 1
		}
	}
	return -1
}
