package mail

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailthread"
)

type ThreadStaleError struct {
	Current mailthread.State
}

func (e *ThreadStaleError) Error() string { return ErrStale.Error() }
func (e *ThreadStaleError) Unwrap() error { return ErrStale }

func threadIdentity(env attention.MailEnvelope) string {
	if env.ThreadID != "" {
		return env.ThreadID
	}
	return env.ID
}

func (s *Store) Thread(recipient, threadID string) (mailthread.ThreadView, bool, error) {
	if !validPathID(recipient) || !validPathID(threadID) {
		return mailthread.ThreadView{}, false, errors.New("invalid mail thread storage identifier")
	}
	lock, err := acquireLock(s.lockPath)
	if err != nil {
		return mailthread.ThreadView{}, false, err
	}
	defer lock.Close()
	state, created, err := s.ensureThreadLocked(recipient, threadID)
	if err != nil {
		return mailthread.ThreadView{}, false, err
	}
	read, err := s.threadReadSummaryLocked(recipient, threadID)
	if err != nil {
		return mailthread.ThreadView{}, false, err
	}
	return mailthread.ThreadView{State: state, Read: read, Actions: ThreadCapabilities(state)}, created, nil
}

func (s *Store) ensureThreadLocked(recipient, threadID string) (mailthread.State, bool, error) {
	path := s.threadPath(recipient, threadID)
	if data, err := os.ReadFile(path); err == nil {
		var state mailthread.State
		if err := json.Unmarshal(data, &state); err != nil {
			return state, false, fmt.Errorf("decode mail thread state: %w", err)
		}
		if invalid := mailthread.ValidateState(state); invalid != nil {
			return state, false, invalid
		}
		return state, false, nil
	} else if !os.IsNotExist(err) {
		return mailthread.State{}, false, err
	}

	messages, err := s.threadMessagesLocked(recipient, threadID)
	if err != nil {
		return mailthread.State{}, false, err
	}
	if len(messages) == 0 {
		return mailthread.State{}, false, ErrNotFound
	}
	state := mailthread.State{
		SchemaVersion: mailthread.SchemaVersion,
		Identity:      mailthread.Identity{ProjectPeerID: recipient, ThreadID: threadID},
		Lifecycle:     mailthread.LifecycleOpen,
	}
	allDismissed := true
	latestDismissed := ""
	for _, env := range messages {
		receipt, receiptErr := s.Receipt(recipient, env.ID)
		if errors.Is(receiptErr, ErrNotFound) {
			allDismissed = false
			continue
		}
		if receiptErr != nil {
			return state, false, receiptErr
		}
		if receipt.DismissedAt == "" {
			allDismissed = false
			continue
		}
		if receipt.DismissedAt > latestDismissed {
			latestDismissed = receipt.DismissedAt
		}
	}
	if allDismissed {
		state.Lifecycle = mailthread.LifecycleArchived
		state.Resolution = &mailthread.Resolution{Reason: "migrated_dismissed", Source: "mailread_v1", Outcome: "dismissed"}
		state.GraceClass = mailthread.GraceInformational
		state.ResolvedAt = latestDismissed
		state.ArchivedAt = latestDismissed
	}
	state.Revision = threadRevision(state)
	if invalid := mailthread.ValidateState(state); invalid != nil {
		return state, false, invalid
	}
	if err := secureDir(filepath.Dir(path)); err != nil {
		return state, false, err
	}
	if err := atomicCreate(path, state); err != nil {
		return state, false, err
	}
	return state, true, nil
}

func (s *Store) MutateThread(recipient, threadID string, expected int64, actionID, key, requestHash, appliedAt string, change func(*mailthread.State) error) (mailthread.ThreadView, error) {
	if !validPathID(recipient) || !validPathID(threadID) {
		return mailthread.ThreadView{}, errors.New("invalid mail thread storage identifier")
	}
	lock, err := acquireLock(s.lockPath)
	if err != nil {
		return mailthread.ThreadView{}, err
	}
	defer lock.Close()
	state, _, err := s.ensureThreadLocked(recipient, threadID)
	if err != nil {
		return mailthread.ThreadView{}, err
	}
	for _, prior := range state.Actions {
		if prior.IdempotencyKey == key {
			if prior.RequestHash != requestHash {
				return mailthread.ThreadView{}, ErrIdempotencyConflict
			}
			read, err := s.threadReadSummaryLocked(recipient, threadID)
			return mailthread.ThreadView{State: state, Read: read, Actions: ThreadCapabilities(state)}, err
		}
	}
	if state.Revision != expected {
		return mailthread.ThreadView{}, &ThreadStaleError{Current: state}
	}
	if err := change(&state); err != nil {
		return mailthread.ThreadView{}, err
	}
	state.Actions = append(state.Actions, mailthread.ActionRecord{
		ActionID: actionID, IdempotencyKey: key,
		RequestHash: requestHash, AppliedAt: appliedAt,
	})
	state.Revision = threadRevision(state)
	if invalid := mailthread.ValidateState(state); invalid != nil {
		return mailthread.ThreadView{}, invalid
	}
	if err := atomicReplace(s.threadPath(recipient, threadID), state); err != nil {
		return mailthread.ThreadView{}, err
	}
	read, err := s.threadReadSummaryLocked(recipient, threadID)
	return mailthread.ThreadView{State: state, Read: read, Actions: ThreadCapabilities(state)}, err
}

func (s *Store) threadMessagesLocked(recipient, threadID string) ([]attention.MailEnvelope, error) {
	messages, err := s.List(recipient)
	if err != nil {
		return nil, err
	}
	filtered := messages[:0]
	for _, env := range messages {
		if threadIdentity(env) == threadID {
			filtered = append(filtered, env)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].ID < filtered[j].ID })
	return filtered, nil
}

func (s *Store) threadReadSummaryLocked(recipient, threadID string) (mailthread.ReadSummary, error) {
	messages, err := s.threadMessagesLocked(recipient, threadID)
	if err != nil {
		return mailthread.ReadSummary{}, err
	}
	result := mailthread.ReadSummary{MessageCount: len(messages)}
	for _, env := range messages {
		receipt, err := s.Receipt(recipient, env.ID)
		if errors.Is(err, ErrNotFound) {
			result.UnreadCount++
			continue
		}
		if err != nil {
			return result, err
		}
		if receipt.ReadAt == "" {
			result.UnreadCount++
		}
	}
	return result, nil
}

func (s *Store) threadPath(recipient, threadID string) string {
	return filepath.Join(s.boxes, recipient, "threads", threadID+".json")
}

func threadRevision(state mailthread.State) int64 {
	state.Revision = 0
	data, _ := json.Marshal(state)
	sum := sha256.Sum256(data)
	value := int64(binary.BigEndian.Uint64(sum[:8]) & (1<<63 - 1))
	if value == 0 {
		return 1
	}
	return value
}

func threadActionHash(request mailthread.ActionRequest) string {
	request.ThreadRevision = 0
	data, _ := json.Marshal(request)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (s *Service) Thread(projectPeerID, threadID string) (mailthread.ThreadView, bool, error) {
	if projectPeerID != s.cfg.PeerID {
		return mailthread.ThreadView{}, false, ErrRecipientMismatch
	}
	view, changed, err := s.store.ReconcileThread(projectPeerID, threadID, s.now())
	return view, changed, err
}

func (s *Service) ThreadAction(request mailthread.ActionRequest) (mailthread.ThreadView, error) {
	if invalid := mailthread.ValidateActionRequest(request); invalid != nil {
		return mailthread.ThreadView{}, invalid
	}
	if request.Identity.ProjectPeerID != s.cfg.PeerID {
		return mailthread.ThreadView{}, ErrRecipientMismatch
	}
	messageIDs, err := s.threadMessageIDs(request.Identity.ThreadID)
	if err != nil {
		return mailthread.ThreadView{}, err
	}
	if _, _, err := s.store.ReconcileThread(s.cfg.PeerID, request.Identity.ThreadID, s.now()); err != nil {
		return mailthread.ThreadView{}, err
	}
	var input mailthread.ActionInput
	if len(request.Input) != 0 {
		if err := json.Unmarshal(request.Input, &input); err != nil {
			return mailthread.ThreadView{}, err
		}
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	hash := threadActionHash(request)
	view, err := s.store.MutateThread(s.cfg.PeerID, request.Identity.ThreadID, request.ThreadRevision, request.ActionID, request.IdempotencyKey, hash, now, func(state *mailthread.State) error {
		from := state.Lifecycle
		switch request.ActionID {
		case mailthread.ActionResolve:
			if state.Lifecycle != mailthread.LifecycleOpen {
				return ErrUnsupportedAction
			}
			state.Lifecycle = mailthread.LifecycleResolved
			state.Resolution = &mailthread.Resolution{Reason: input.Reason, Source: input.Source, Outcome: input.Outcome, SourceID: input.SourceID}
			state.GraceClass = input.GraceClass
			if state.GraceClass == "" {
				state.GraceClass = mailthread.GraceInformational
			}
			state.ResolvedAt = now
			resolvedAt, _ := time.Parse(time.RFC3339Nano, now)
			grace := informationalGrace
			if state.GraceClass == mailthread.GraceLinkedWork {
				grace = linkedWorkGrace
			}
			state.ArchiveEligibleAt = resolvedAt.Add(grace).UTC().Format(time.RFC3339Nano)
			state.ArchivedAt = ""
		case mailthread.ActionReopen, mailthread.ActionRestore:
			if request.ActionID == mailthread.ActionReopen && state.Lifecycle != mailthread.LifecycleResolved {
				return ErrUnsupportedAction
			}
			if request.ActionID == mailthread.ActionRestore && state.Lifecycle != mailthread.LifecycleArchived {
				return ErrUnsupportedAction
			}
			state.Lifecycle = mailthread.LifecycleOpen
			state.Resolution = nil
			state.GraceClass = ""
			state.ResolvedAt = ""
			state.ArchiveEligibleAt = ""
			state.ArchivedAt = ""
		case mailthread.ActionArchive:
			if state.Lifecycle == mailthread.LifecycleArchived {
				return ErrUnsupportedAction
			}
			state.Lifecycle = mailthread.LifecycleArchived
			state.ArchivedAt = now
		default:
			return ErrUnsupportedAction
		}
		source := input.Source
		if source == "" {
			source = "mail.lifecycle"
		}
		sourceID := input.SourceID
		if sourceID == "" {
			sourceID = request.IdempotencyKey
		}
		state.Events = append(state.Events, mailthread.EventRecord{
			EventID: "lifecycle:" + request.IdempotencyKey, Kind: mailthread.EventActionSucceeded,
			RequestHash: hash, AppliedAt: now, Source: source, SourceID: sourceID,
			Outcome: input.Outcome, FromLifecycle: from, ToLifecycle: state.Lifecycle,
			PriorMessageIDs: append([]string(nil), messageIDs...),
		})
		return nil
	})
	if err != nil {
		return mailthread.ThreadView{}, err
	}
	messageIDs = eventRecordMessageIDs(view.State, "lifecycle:"+request.IdempotencyKey, messageIDs)
	if err := s.markMessagesRead(messageIDs, "lifecycle:"+request.IdempotencyKey); err != nil {
		return mailthread.ThreadView{}, err
	}
	view, _, err = s.store.ReconcileThread(s.cfg.PeerID, request.Identity.ThreadID, s.now())
	return view, err
}

var lifecycleNoInput = json.RawMessage(`{"type":"object","additionalProperties":false}`)
var lifecycleResolveInput = json.RawMessage(`{"type":"object","required":["reason","source"],"properties":{"reason":{"type":"string"},"source":{"type":"string"},"outcome":{"type":"string"},"source_id":{"type":"string"},"grace_class":{"enum":["informational","linked_work"]}},"additionalProperties":false}`)

func ThreadCapabilities(state mailthread.State) []attention.ActionDescriptor {
	type definition struct {
		operation, id, label string
		input                json.RawMessage
	}
	var definitions []definition
	switch state.Lifecycle {
	case mailthread.LifecycleOpen:
		definitions = []definition{{"mail.resolve", mailthread.ActionResolve, "Resolve", lifecycleResolveInput}, {"mail.archive", mailthread.ActionArchive, "Archive", lifecycleNoInput}}
	case mailthread.LifecycleResolved:
		definitions = []definition{{"mail.reopen", mailthread.ActionReopen, "Reopen", lifecycleNoInput}, {"mail.archive", mailthread.ActionArchive, "Archive", lifecycleNoInput}}
	case mailthread.LifecycleArchived:
		definitions = []definition{{"mail.restore", mailthread.ActionRestore, "Restore", lifecycleNoInput}}
	}
	result := make([]attention.ActionDescriptor, 0, len(definitions))
	for _, definition := range definitions {
		descriptor := attention.ActionDescriptor{ID: definition.id, Label: definition.label, OperationID: definition.operation, Effect: string(attention.EffectStateWrite), Consent: string(attention.ConsentExplicitUser), RequiredRowRevision: state.Revision, RequiresIdempotency: true, InputSchema: definition.input}
		result = append(result, descriptor)
	}
	return result
}
