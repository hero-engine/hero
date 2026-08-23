package mail

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailthread"
)

const (
	ActionRead        = "read"
	ActionUnread      = "unread"
	ActionAcknowledge = "acknowledge"
	ActionDismiss     = "dismiss"
	ActionPromote     = "promote"
	ActionAddToToday  = "add_to_today"
)

var ErrUnsupportedAction = errors.New("unsupported mail action")

type ActionRequest struct {
	MessageID        string `json:"message_id"`
	Action           string `json:"action"`
	ExpectedRevision int64  `json:"expected_revision"`
	IdempotencyKey   string `json:"idempotency_key"`
	Note             string `json:"note,omitempty"`
	ArtifactType     string `json:"artifact_type,omitempty"`
}

type NavigationReference struct {
	Kind          string `json:"kind"`
	ProjectPeerID string `json:"project_peer_id"`
	Slug          string `json:"slug"`
}

type ActionResult struct {
	Action      string                     `json:"action"`
	Receipt     attention.MailReceipt      `json:"receipt"`
	Artifact    *attention.MailArtifact    `json:"artifact,omitempty"`
	FocusItemID string                     `json:"focus_item_id,omitempty"`
	MessageID   string                     `json:"message_id"`
	ThreadID    string                     `json:"thread_id"`
	Project     attention.ProjectReference `json:"project"`
	Navigation  *NavigationReference       `json:"navigation,omitempty"`
	Thread      *mailthread.ThreadView     `json:"thread,omitempty"`
}

func (s *Service) Action(req ActionRequest) (ActionResult, error) {
	env, err := s.store.Get(s.cfg.PeerID, req.MessageID)
	if err != nil {
		return ActionResult{}, err
	}
	messageIDs, err := s.threadMessageIDs(threadIdentity(env))
	if err != nil {
		return ActionResult{}, err
	}
	result, err := s.actionRaw(req)
	if err != nil {
		return ActionResult{}, err
	}
	if req.Action != ActionRead && req.Action != ActionUnread {
		if err := s.markMessagesRead(messageIDs, "action:"+req.IdempotencyKey); err != nil {
			return ActionResult{}, err
		}
	}
	if authoritative, err := s.store.Receipt(s.cfg.PeerID, req.MessageID); err == nil {
		result.Receipt = authoritative
	} else {
		return ActionResult{}, err
	}
	view, _, err := s.store.ReconcileThread(s.cfg.PeerID, threadIdentity(env), s.now())
	if err != nil {
		return ActionResult{}, err
	}
	if req.Action == ActionRead || req.Action == ActionUnread {
		result.Thread = &view
		return result, nil
	}
	event := mailthread.Event{
		SchemaVersion: mailthread.SchemaVersion,
		Identity:      view.State.Identity, Kind: mailthread.EventActionSucceeded,
		EventID: "action:" + req.IdempotencyKey, ExpectedRevision: view.State.Revision,
		OccurredAt: actionAppliedAt(result.Receipt, req.IdempotencyKey, env.CreatedAt), Source: "mail.action", SourceID: req.MessageID,
	}
	eventResult, err := s.applyStoreEventLatest(event)
	if err != nil {
		return ActionResult{}, err
	}
	result.Thread = &eventResult.Thread
	return result, nil
}

func (s *Service) actionRaw(req ActionRequest) (ActionResult, error) {
	if req.Action == ActionPromote {
		return s.promote(req)
	}
	if req.Action == ActionAddToToday {
		return s.addToToday(req)
	}
	if req.Action != ActionRead && req.Action != ActionUnread && req.Action != ActionAcknowledge && req.Action != ActionDismiss {
		return ActionResult{}, fmt.Errorf("%w: %q", ErrUnsupportedAction, req.Action)
	}
	if req.IdempotencyKey == "" {
		return ActionResult{}, errors.New("idempotency key is required")
	}
	if !utf8.ValidString(req.IdempotencyKey) || len(req.IdempotencyKey) > 512 {
		return ActionResult{}, errors.New("idempotency key must be valid UTF-8 and at most 512 bytes")
	}
	if !utf8.ValidString(req.Note) || utf8.RuneCountInString(req.Note) > 500 {
		return ActionResult{}, errors.New("acknowledgement note must be valid UTF-8 and at most 500 characters")
	}
	env, err := s.store.Get(s.cfg.PeerID, req.MessageID)
	if err != nil {
		return ActionResult{}, err
	}
	hash := actionHash(req)
	if current, err := s.store.Receipt(s.cfg.PeerID, req.MessageID); err == nil {
		if replay, conflict := actionReplay(current, req.IdempotencyKey, hash); replay {
			current, err = s.ensureActionEvent(current, env, req.IdempotencyKey)
			if err != nil {
				return ActionResult{}, err
			}
			if _, _, err := s.store.Thread(s.cfg.PeerID, threadIdentity(env)); err != nil {
				return ActionResult{}, err
			}
			return actionResult(req.Action, env, current), nil
		} else if conflict {
			return ActionResult{}, ErrIdempotencyConflict
		}
	} else if !errors.Is(err, ErrNotFound) {
		return ActionResult{}, err
	}
	receipt, err := s.store.MutateReceipt(s.cfg.PeerID, req.MessageID, req.ExpectedRevision, func(r *attention.MailReceipt) error {
		if replay, conflict := actionReplay(*r, req.IdempotencyKey, hash); replay {
			return nil
		} else if conflict {
			return ErrIdempotencyConflict
		}
		now := s.now().UTC().Format(time.RFC3339Nano)
		switch req.Action {
		case ActionRead:
			if r.ReadAt == "" {
				r.ReadAt = now
			}
			if r.Kind == "" {
				r.Kind = attention.ReceiptRead
			}
		case ActionUnread:
			r.ReadAt = ""
			if r.Kind == attention.ReceiptRead {
				r.Kind = ""
			}
		case ActionAcknowledge:
			if r.AcknowledgedAt == "" {
				r.AcknowledgedAt = now
				r.AcknowledgementNote = req.Note
			}
			r.Kind = attention.ReceiptAcknowledged
		case ActionDismiss:
			if r.DismissedAt == "" {
				r.DismissedAt = now
			}
			r.Kind = attention.ReceiptDismissed
		}
		r.Actions = append(r.Actions, attention.MailAction{ID: req.Action, IdempotencyKey: req.IdempotencyKey, RequestHash: hash, AppliedAt: now})
		return nil
	})
	if err != nil {
		return ActionResult{}, err
	}
	if err := s.emit(req.Action, env, env.ID); err != nil {
		return ActionResult{}, err
	}
	receipt, err = s.markActionEvent(receipt, env.ID, req.IdempotencyKey)
	if err != nil {
		return ActionResult{}, err
	}
	if _, _, err := s.store.Thread(s.cfg.PeerID, threadIdentity(env)); err != nil {
		return ActionResult{}, err
	}
	return actionResult(req.Action, env, receipt), nil
}

func actionAppliedAt(receipt attention.MailReceipt, key, fallback string) string {
	for _, action := range receipt.Actions {
		if action.IdempotencyKey == key && action.AppliedAt != "" {
			return action.AppliedAt
		}
	}
	return fallback
}

func actionResult(action string, env attention.MailEnvelope, receipt attention.MailReceipt) ActionResult {
	return ActionResult{Action: action, Receipt: receipt, MessageID: env.ID, ThreadID: env.ThreadID, Project: env.Recipient}
}

func actionHash(req ActionRequest) string {
	req.ExpectedRevision = 0
	b, _ := json.Marshal(req)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func actionReplay(r attention.MailReceipt, key, hash string) (bool, bool) {
	for _, prior := range r.Actions {
		if prior.IdempotencyKey == key {
			return prior.RequestHash == hash, prior.RequestHash != hash
		}
	}
	return false, false
}

func (s *Service) ensureActionEvent(receipt attention.MailReceipt, env attention.MailEnvelope, key string) (attention.MailReceipt, error) {
	for _, action := range receipt.Actions {
		if action.IdempotencyKey == key {
			if action.EventEmitted {
				return receipt, nil
			}
			if err := s.emit(action.ID, env, eventReference(action.ID, receipt, env)); err != nil {
				return receipt, err
			}
			return s.markActionEvent(receipt, env.ID, key)
		}
	}
	return receipt, nil
}

func (s *Service) markActionEvent(receipt attention.MailReceipt, messageID, key string) (attention.MailReceipt, error) {
	return s.store.MutateReceipt(s.cfg.PeerID, messageID, receipt.Revision, func(r *attention.MailReceipt) error {
		for i := range r.Actions {
			if r.Actions[i].IdempotencyKey == key {
				r.Actions[i].EventEmitted = true
				return nil
			}
		}
		return errors.New("mail action record is missing")
	})
}

func eventReference(action string, receipt attention.MailReceipt, env attention.MailEnvelope) string {
	switch action {
	case ActionPromote:
		if receipt.PromotedArtifact != nil {
			return receipt.PromotedArtifact.Slug
		}
	case ActionAddToToday:
		if receipt.FocusItemID != "" {
			return receipt.FocusItemID
		}
	}
	return env.ID
}
