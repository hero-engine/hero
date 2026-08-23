package mail

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/feed"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/intake"
	"github.com/hero-engine/hero/internal/spec"
)

func (s *Service) promote(req ActionRequest) (ActionResult, error) {
	if req.IdempotencyKey == "" {
		return ActionResult{}, errors.New("idempotency key is required")
	}
	if !utf8.ValidString(req.IdempotencyKey) || len(req.IdempotencyKey) > 512 {
		return ActionResult{}, errors.New("idempotency key must be valid UTF-8 and at most 512 bytes")
	}
	if req.ArtifactType == "" {
		req.ArtifactType = "intake"
	}
	if req.ArtifactType != "intake" && req.ArtifactType != "feature" && req.ArtifactType != "bug" {
		return ActionResult{}, fmt.Errorf("unsupported promotion type %q", req.ArtifactType)
	}
	env, err := s.store.Get(s.cfg.PeerID, req.MessageID)
	if err != nil {
		return ActionResult{}, err
	}
	hash := actionHash(req)
	current, currentErr := s.store.Receipt(s.cfg.PeerID, req.MessageID)
	if currentErr != nil && !errors.Is(currentErr, ErrNotFound) {
		return ActionResult{}, currentErr
	}
	if currentErr == nil {
		if replay, conflict := actionReplay(current, req.IdempotencyKey, hash); replay {
			return promotedResult(env, current), nil
		} else if conflict {
			return ActionResult{}, ErrIdempotencyConflict
		}
		if current.Promotion != nil && current.Promotion.IdempotencyKey == req.IdempotencyKey {
			if current.Promotion.RequestHash != hash {
				return ActionResult{}, ErrIdempotencyConflict
			}
		} else if current.Promotion != nil {
			return ActionResult{}, ErrIdempotencyConflict
		} else if current.Revision != req.ExpectedRevision {
			return ActionResult{}, &StaleError{Current: current}
		}
	} else if req.ExpectedRevision != 0 {
		return ActionResult{}, &StaleError{}
	}

	slug := promotionSlug(env.ID)
	if current.Promotion == nil {
		current, err = s.store.MutateReceipt(s.cfg.PeerID, env.ID, req.ExpectedRevision, func(r *attention.MailReceipt) error {
			r.Promotion = &attention.MailPromotion{IntakeSlug: slug, IdempotencyKey: req.IdempotencyKey, RequestHash: hash}
			return nil
		})
		if err != nil {
			return ActionResult{}, err
		}
		if err := s.inject("reserve"); err != nil {
			return ActionResult{}, err
		}
	}
	source := &attention.ProvenanceReference{Kind: "mail", SourceID: env.ID, Label: env.Subject, CreatedAt: env.CreatedAt}
	if !current.Promotion.Captured {
		if _, resolveErr := s.intake.Resolve(slug); resolveErr != nil {
			if _, err := s.intake.Capture(intake.CaptureRequest{
				Slug: slug, Title: env.Subject, Body: env.Body, Provenance: source,
				Source: intake.SourceMetadata{SenderPeerID: env.Sender.PeerID, RecipientPeerID: env.Recipient.PeerID, ThreadID: env.ThreadID},
			}); err != nil {
				return ActionResult{}, err
			}
		}
		current, err = s.updatePromotion(env.ID, func(progress *attention.MailPromotion) {
			progress.Captured = true
		})
		if err != nil {
			return ActionResult{}, err
		}
	}
	if err := s.inject("capture"); err != nil {
		return ActionResult{}, err
	}
	if current.Promotion.Artifact == nil {
		artifact := attention.MailArtifact{Slug: slug, Type: "intake", Status: string(spec.StatusPlanning)}
		if req.ArtifactType != "intake" {
			promoted, err := s.intake.Promote(intake.PromoteRequest{Slug: slug, Type: req.ArtifactType, AllowReplay: true, GenerateTemplate: intake.DefaultRoadmapTemplate})
			if err != nil {
				return ActionResult{}, err
			}
			artifact = attention.MailArtifact{Slug: promoted.Slug, Type: promoted.Type, Status: promoted.Status}
		}
		current, err = s.updatePromotion(env.ID, func(progress *attention.MailPromotion) {
			progress.Artifact = &artifact
		})
		if err != nil {
			return ActionResult{}, err
		}
	}
	if err := s.inject("promote"); err != nil {
		return ActionResult{}, err
	}
	if !current.Promotion.ProvenanceWritten {
		if err := s.writeMailProvenance(env, slug); err != nil {
			return ActionResult{}, err
		}
		current, err = s.updatePromotion(env.ID, func(progress *attention.MailPromotion) {
			progress.ProvenanceWritten = true
		})
		if err != nil {
			return ActionResult{}, err
		}
	}
	if err := s.inject("provenance"); err != nil {
		return ActionResult{}, err
	}
	if !current.Promotion.EventEmitted {
		if err := s.emit(ActionPromote, env, current.Promotion.Artifact.Slug); err != nil {
			return ActionResult{}, err
		}
		current, err = s.updatePromotion(env.ID, func(progress *attention.MailPromotion) {
			progress.EventEmitted = true
		})
		if err != nil {
			return ActionResult{}, err
		}
	}
	receipt, err := s.store.MutateReceipt(s.cfg.PeerID, env.ID, current.Revision, func(r *attention.MailReceipt) error {
		r.PromotedArtifact = r.Promotion.Artifact
		r.Kind = attention.ReceiptPromoted
		r.Actions = append(r.Actions, attention.MailAction{ID: ActionPromote, IdempotencyKey: req.IdempotencyKey, RequestHash: hash, EventEmitted: true, AppliedAt: s.now().UTC().Format(time.RFC3339Nano)})
		return nil
	})
	if err != nil {
		return ActionResult{}, err
	}
	if err := s.inject("receipt"); err != nil {
		return ActionResult{}, err
	}
	return promotedResult(env, receipt), nil
}

func promotedResult(env attention.MailEnvelope, r attention.MailReceipt) ActionResult {
	result := actionResult(ActionPromote, env, r)
	result.Artifact = r.PromotedArtifact
	if r.PromotedArtifact != nil {
		result.Navigation = &NavigationReference{Kind: "spec", ProjectPeerID: env.Recipient.PeerID, Slug: r.PromotedArtifact.Slug}
	}
	return result
}

func (s *Service) addToToday(req ActionRequest) (ActionResult, error) {
	if req.IdempotencyKey == "" {
		return ActionResult{}, errors.New("idempotency key is required")
	}
	if !utf8.ValidString(req.IdempotencyKey) || len(req.IdempotencyKey) > 512 {
		return ActionResult{}, errors.New("idempotency key must be valid UTF-8 and at most 512 bytes")
	}
	if s.focus == nil {
		return ActionResult{}, errors.New("focus authority is unavailable")
	}
	env, err := s.store.Get(s.cfg.PeerID, req.MessageID)
	if err != nil {
		return ActionResult{}, err
	}
	hash := actionHash(req)
	current, err := currentReceipt(s.store, s.cfg.PeerID, env.ID)
	if err != nil {
		return ActionResult{}, err
	}
	if replay, conflict := actionReplay(current, req.IdempotencyKey, hash); replay {
		current, err = s.ensureActionEvent(current, env, req.IdempotencyKey)
		if err != nil {
			return ActionResult{}, err
		}
		result := actionResult(ActionAddToToday, env, current)
		result.FocusItemID = current.FocusItemID
		return result, nil
	} else if conflict {
		return ActionResult{}, ErrIdempotencyConflict
	}
	if current.Revision != req.ExpectedRevision {
		return ActionResult{}, &StaleError{Current: current}
	}
	project := env.Recipient
	prompt := env.Body + "\n\nSource: mail:" + env.ID + "\nThread: " + env.ThreadID
	item, _, err := s.focus.CreateOrGet(focus.CreateRequest{
		Title: env.Subject, Prompt: prompt, Lifecycle: attention.FocusToday,
		Project: &project, Origin: &attention.ProvenanceReference{Kind: "mail", SourceID: env.ID, Label: env.Subject, CreatedAt: env.CreatedAt},
		OriginKey: "mail:" + env.Recipient.PeerID + ":" + env.ID,
	})
	if err != nil {
		return ActionResult{}, err
	}
	receipt, err := s.store.MutateReceipt(s.cfg.PeerID, env.ID, current.Revision, func(r *attention.MailReceipt) error {
		r.FocusItemID = item.ID
		r.Actions = append(r.Actions, attention.MailAction{ID: ActionAddToToday, IdempotencyKey: req.IdempotencyKey, RequestHash: hash, AppliedAt: s.now().UTC().Format(time.RFC3339Nano)})
		return nil
	})
	if err != nil {
		return ActionResult{}, err
	}
	if err := s.emit(ActionAddToToday, env, item.ID); err != nil {
		return ActionResult{}, err
	}
	receipt, err = s.markActionEvent(receipt, env.ID, req.IdempotencyKey)
	if err != nil {
		return ActionResult{}, err
	}
	result := actionResult(ActionAddToToday, env, receipt)
	result.FocusItemID = item.ID
	return result, nil
}

func currentReceipt(store *Store, recipient, id string) (attention.MailReceipt, error) {
	r, err := store.Receipt(recipient, id)
	if errors.Is(err, ErrNotFound) {
		env, getErr := store.Get(recipient, id)
		if getErr != nil {
			return r, getErr
		}
		return attention.MailReceipt{SchemaVersion: attention.SchemaVersion, ID: "receipt_" + id, EnvelopeID: id, Recipient: env.Recipient, CreatedAt: env.CreatedAt, EnvelopeRevision: env.Revision}, nil
	}
	return r, err
}

func (s *Service) inject(step string) error {
	if s.failAfter != nil {
		return s.failAfter(step)
	}
	return nil
}

func (s *Service) updatePromotion(messageID string, change func(*attention.MailPromotion)) (attention.MailReceipt, error) {
	current, err := currentReceipt(s.store, s.cfg.PeerID, messageID)
	if err != nil {
		return attention.MailReceipt{}, err
	}
	return s.store.MutateReceipt(s.cfg.PeerID, messageID, current.Revision, func(r *attention.MailReceipt) error {
		if r.Promotion == nil {
			return errors.New("mail promotion progress is missing")
		}
		change(r.Promotion)
		return nil
	})
}

func (s *Service) emit(action string, env attention.MailEnvelope, ref string) error {
	return feed.AppendEvent(filepath.Join(s.cfg.HeroDir(s.root), "events.log"), feed.FeedEvent{
		Timestamp: s.now().UTC(), Type: "mail." + action, Agent: "hero-mail", Slug: ref,
		Message: fmt.Sprintf("%s %s (thread %s)", action, env.ID, env.ThreadID),
	})
}

func (s *Service) writeMailProvenance(env attention.MailEnvelope, intakeSlug string) error {
	heroDir := s.cfg.HeroDir(s.root)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return err
	}
	store, err := graph.Open(heroDir)
	if err != nil {
		return err
	}
	defer store.Close()
	domain := graph.DomainFor(s.cfg, graph.IntrinsicActive)
	// repoKey must be gitutil.RepoKey — the partition every graph reader
	// filters on. This wrote spec nodes under cfg.PeerID (a UUID), so the
	// nodes landed in a partition nothing queries. Same divergence class as
	// the filepath.Base(projectRoot) writers fixed in
	// graph-why-resolution-and-peer-spec-indexing, one derivation over.
	repoKey := gitutil.RepoKey(s.root)
	if _, err := spec.WriteGraph(specs, repoKey, domain, store); err != nil {
		return err
	}
	if _, err := store.GetNode("Intake", intakeSlug, repoKey); err != nil {
		return err
	}
	_, err = store.GetNode("MailSource", env.ID, repoKey)
	return err
}

func promotionSlug(messageID string) string {
	slug := "mail-" + strings.TrimPrefix(messageID, "mail_")
	if len(slug) <= 72 {
		return slug
	}
	sum := sha256.Sum256([]byte(messageID))
	suffix := hex.EncodeToString(sum[:6])
	return slug[:72-len(suffix)-1] + "-" + suffix
}
