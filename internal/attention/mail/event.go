package mail

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailthread"
)

var ErrEventOutOfOrder = errors.New("mail thread event is out of order")

const (
	informationalGrace = 7 * 24 * time.Hour
	linkedWorkGrace    = 30 * 24 * time.Hour
)

type ThreadEventResult struct {
	Thread   mailthread.ThreadView `json:"thread"`
	Replayed bool                  `json:"replayed"`
}

func (s *Service) ThreadEvent(event mailthread.Event) (ThreadEventResult, error) {
	if invalid := mailthread.ValidateEvent(event); invalid != nil {
		return ThreadEventResult{}, invalid
	}
	if event.Identity.ProjectPeerID != s.cfg.PeerID {
		return ThreadEventResult{}, ErrRecipientMismatch
	}
	return s.store.ApplyThreadEvent(event)
}

func (s *Store) ApplyThreadEvent(event mailthread.Event) (ThreadEventResult, error) {
	if invalid := mailthread.ValidateEvent(event); invalid != nil {
		return ThreadEventResult{}, invalid
	}
	lock, err := acquireLock(s.lockPath)
	if err != nil {
		return ThreadEventResult{}, err
	}
	defer lock.Close()

	state, _, err := s.ensureThreadLocked(event.Identity.ProjectPeerID, event.Identity.ThreadID)
	if err != nil {
		return ThreadEventResult{}, err
	}
	hash := threadEventHash(event)
	for _, prior := range state.Events {
		if prior.EventID != event.EventID {
			continue
		}
		if prior.RequestHash != hash {
			return ThreadEventResult{}, ErrIdempotencyConflict
		}
		read, readErr := s.threadReadSummaryLocked(event.Identity.ProjectPeerID, event.Identity.ThreadID)
		return ThreadEventResult{Thread: mailthread.ThreadView{State: state, Read: read, Actions: ThreadCapabilities(state, read)}, Replayed: true}, readErr
	}
	if state.Revision != event.ExpectedRevision {
		return ThreadEventResult{}, &ThreadStaleError{Current: state}
	}
	occurredAt, _ := time.Parse(time.RFC3339Nano, event.OccurredAt)
	for _, prior := range state.Events {
		priorAt, parseErr := time.Parse(time.RFC3339Nano, prior.AppliedAt)
		if parseErr != nil {
			return ThreadEventResult{}, parseErr
		}
		if priorAt.After(occurredAt) {
			return ThreadEventResult{}, ErrEventOutOfOrder
		}
	}

	from := state.Lifecycle
	switch event.Kind {
	case mailthread.EventAdvisoryTerminal, mailthread.EventSpecOutTerminal:
		if state.Lifecycle != mailthread.LifecycleOpen {
			return ThreadEventResult{}, ErrUnsupportedAction
		}
		resolveFromEvent(&state, event, mailthread.GraceInformational, informationalGrace)
	case mailthread.EventLinkedTerminal:
		if state.Lifecycle != mailthread.LifecycleOpen {
			return ThreadEventResult{}, ErrUnsupportedAction
		}
		resolveFromEvent(&state, event, mailthread.GraceLinkedWork, linkedWorkGrace)
	case mailthread.EventInboundActivity:
		if state.Lifecycle != mailthread.LifecycleOpen {
			state.Lifecycle = mailthread.LifecycleOpen
			state.Resolution = nil
			state.GraceClass = ""
			state.ResolvedAt = ""
			state.ArchiveEligibleAt = ""
			state.ArchivedAt = ""
		}
	case mailthread.EventForegroundRead, mailthread.EventReplySucceeded, mailthread.EventActionSucceeded:
		// Receipt mutations are performed by the owning service boundary. The
		// reducer records typed provenance without deriving lifecycle from prose.
	default:
		return ThreadEventResult{}, ErrUnsupportedAction
	}
	state.Events = append(state.Events, mailthread.EventRecord{
		EventID: event.EventID, Kind: event.Kind, RequestHash: hash,
		AppliedAt: event.OccurredAt, Source: event.Source, SourceID: event.SourceID,
		Outcome: event.Outcome, FromLifecycle: from, ToLifecycle: state.Lifecycle,
		PriorMessageIDs: append([]string(nil), event.PriorMessageIDs...),
	})
	state.Revision = threadRevision(state)
	if invalid := mailthread.ValidateState(state); invalid != nil {
		return ThreadEventResult{}, invalid
	}
	if err := atomicReplace(s.threadPath(event.Identity.ProjectPeerID, event.Identity.ThreadID), state); err != nil {
		return ThreadEventResult{}, err
	}
	read, err := s.threadReadSummaryLocked(event.Identity.ProjectPeerID, event.Identity.ThreadID)
	return ThreadEventResult{Thread: mailthread.ThreadView{State: state, Read: read, Actions: ThreadCapabilities(state, read)}}, err
}

func eventRecordMessageIDs(state mailthread.State, eventID string, fallback []string) []string {
	for _, event := range state.Events {
		if event.EventID == eventID && len(event.PriorMessageIDs) != 0 {
			return append([]string(nil), event.PriorMessageIDs...)
		}
	}
	return append([]string(nil), fallback...)
}

func (s *Service) applyStoreEventLatest(event mailthread.Event) (ThreadEventResult, error) {
	result, err := s.store.ApplyThreadEvent(event)
	if err == nil {
		return result, nil
	}
	var stale *ThreadStaleError
	if errors.As(err, &stale) {
		event.ExpectedRevision = stale.Current.Revision
		result, err = s.store.ApplyThreadEvent(event)
	}
	if errors.Is(err, ErrEventOutOfOrder) {
		view, _, viewErr := s.store.Thread(event.Identity.ProjectPeerID, event.Identity.ThreadID)
		return ThreadEventResult{Thread: view}, viewErr
	}
	return result, err
}

func (s *Service) threadMessageIDs(threadID string) ([]string, error) {
	messages, err := s.store.List(s.cfg.PeerID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(messages))
	for _, env := range messages {
		if threadIdentity(env) == threadID {
			ids = append(ids, env.ID)
		}
	}
	return ids, nil
}

func (s *Service) markThreadRead(threadID, cause string) error {
	ids, err := s.threadMessageIDs(threadID)
	if err != nil {
		return err
	}
	return s.markMessagesRead(ids, cause)
}

func (s *Service) markMessagesRead(messageIDs []string, cause string) error {
	for _, messageID := range messageIDs {
		current, err := currentReceipt(s.store, s.cfg.PeerID, messageID)
		if err != nil {
			return err
		}
		key := "thread-read:" + cause + ":" + messageID
		req := ActionRequest{MessageID: messageID, Action: ActionRead, IdempotencyKey: key}
		hash := actionHash(req)
		_, err = s.store.MutateReceipt(s.cfg.PeerID, messageID, current.Revision, func(receipt *attention.MailReceipt) error {
			if replay, conflict := actionReplay(*receipt, key, hash); replay {
				return nil
			} else if conflict {
				return ErrIdempotencyConflict
			}
			appliedAt := s.now().UTC().Format(time.RFC3339Nano)
			if receipt.ReadAt == "" {
				receipt.ReadAt = appliedAt
			}
			if receipt.Kind == "" {
				receipt.Kind = attention.ReceiptRead
			}
			receipt.Actions = append(receipt.Actions, attention.MailAction{ID: ActionRead, IdempotencyKey: key, RequestHash: hash, EventEmitted: true, AppliedAt: appliedAt})
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) markMessagesUnread(messageIDs []string, cause string) error {
	for _, messageID := range messageIDs {
		current, err := currentReceipt(s.store, s.cfg.PeerID, messageID)
		if err != nil {
			return err
		}
		key := "thread-unread:" + cause + ":" + messageID
		req := ActionRequest{MessageID: messageID, Action: ActionUnread, IdempotencyKey: key}
		hash := actionHash(req)
		_, err = s.store.MutateReceipt(s.cfg.PeerID, messageID, current.Revision, func(receipt *attention.MailReceipt) error {
			if replay, conflict := actionReplay(*receipt, key, hash); replay {
				return nil
			} else if conflict {
				return ErrIdempotencyConflict
			}
			appliedAt := s.now().UTC().Format(time.RFC3339Nano)
			receipt.ReadAt = ""
			if receipt.Kind == attention.ReceiptRead {
				receipt.Kind = ""
			}
			receipt.Actions = append(receipt.Actions, attention.MailAction{ID: ActionUnread, IdempotencyKey: key, RequestHash: hash, EventEmitted: true, AppliedAt: appliedAt})
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func resolveFromEvent(state *mailthread.State, event mailthread.Event, graceClass mailthread.GraceClass, grace time.Duration) {
	occurred, _ := time.Parse(time.RFC3339Nano, event.OccurredAt)
	state.Lifecycle = mailthread.LifecycleResolved
	state.Resolution = &mailthread.Resolution{
		Reason: string(event.Kind), Source: event.Source,
		Outcome: event.Outcome, SourceID: event.SourceID,
	}
	state.GraceClass = graceClass
	state.ResolvedAt = event.OccurredAt
	state.ArchiveEligibleAt = occurred.Add(grace).UTC().Format(time.RFC3339Nano)
	state.ArchivedAt = ""
}

func (s *Store) ReconcileThread(recipient, threadID string, now time.Time) (mailthread.ThreadView, bool, error) {
	if !validPathID(recipient) || !validPathID(threadID) {
		return mailthread.ThreadView{}, false, errors.New("invalid mail thread storage identifier")
	}
	lock, err := acquireLock(s.lockPath)
	if err != nil {
		return mailthread.ThreadView{}, false, err
	}
	defer lock.Close()
	state, _, err := s.ensureThreadLocked(recipient, threadID)
	if err != nil {
		return mailthread.ThreadView{}, false, err
	}
	changed := false
	if state.Lifecycle == mailthread.LifecycleResolved {
		resolvedAt, parseErr := time.Parse(time.RFC3339Nano, state.ResolvedAt)
		if parseErr != nil {
			return mailthread.ThreadView{}, false, parseErr
		}
		grace := informationalGrace
		if state.GraceClass == mailthread.GraceLinkedWork {
			grace = linkedWorkGrace
		}
		eligible := resolvedAt.Add(grace).UTC()
		eligibleAt := eligible.Format(time.RFC3339Nano)
		if state.ArchiveEligibleAt != eligibleAt {
			state.ArchiveEligibleAt = eligibleAt
			changed = true
		}
		now = now.UTC()
		if !now.Before(eligible) {
			from := state.Lifecycle
			state.Lifecycle = mailthread.LifecycleArchived
			state.ArchivedAt = now.Format(time.RFC3339Nano)
			eventID := fmt.Sprintf("grace:%s:%s", threadID, state.ResolvedAt)
			state.Events = append(state.Events, mailthread.EventRecord{
				EventID: eventID, Kind: mailthread.EventGraceArchive,
				RequestHash: graceArchiveHash(state.Identity, state.ResolvedAt, eligibleAt),
				AppliedAt:   state.ArchivedAt, Source: "mail.reconcile", SourceID: eventID,
				Outcome: string(state.GraceClass), FromLifecycle: from, ToLifecycle: state.Lifecycle,
			})
			changed = true
		}
	}
	if changed {
		state.Revision = threadRevision(state)
		if invalid := mailthread.ValidateState(state); invalid != nil {
			return mailthread.ThreadView{}, false, invalid
		}
		if err := atomicReplace(s.threadPath(recipient, threadID), state); err != nil {
			return mailthread.ThreadView{}, false, err
		}
	}
	read, err := s.threadReadSummaryLocked(recipient, threadID)
	return mailthread.ThreadView{State: state, Read: read, Actions: ThreadCapabilities(state, read)}, changed, err
}

func threadEventHash(event mailthread.Event) string {
	event.ExpectedRevision = 0
	data, _ := json.Marshal(event)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func graceArchiveHash(identity mailthread.Identity, resolvedAt, eligibleAt string) string {
	data, _ := json.Marshal(struct {
		Identity   mailthread.Identity `json:"identity"`
		ResolvedAt string              `json:"resolved_at"`
		EligibleAt string              `json:"eligible_at"`
	}{identity, resolvedAt, eligibleAt})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
