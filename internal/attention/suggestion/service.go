package suggestion

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
)

const (
	DecisionToday   = "today"
	DecisionLater   = "later"
	DecisionDoNext  = "do_next"
	DecisionDismiss = "dismiss"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func (e *Error) Error() string { return e.Message }

type CreateRequest struct {
	Kind           string
	Title          string
	Reason         string
	Prompt         string
	Project        *attention.ProjectReference
	Provenance     *attention.ProvenanceReference
	IdempotencyKey string
}

type Presented struct {
	Item
	Actions []attention.ActionDescriptor `json:"actions"`
}

type LaunchIntent struct {
	Prompt  string                     `json:"prompt"`
	Project attention.ProjectReference `json:"project"`
	Path    string                     `json:"path"`
}

type ActionResult struct {
	SchemaVersion int           `json:"schema_version"`
	Suggestion    Presented     `json:"suggestion"`
	Focus         *focus.Item   `json:"focus,omitempty"`
	Launch        *LaunchIntent `json:"launch,omitempty"`
}

type Service struct {
	store    *Store
	focus    *focus.Service
	resolver focus.ProjectResolver
	now      func() time.Time
	newID    func() (string, error)
}

func NewService(store *Store, focusService *focus.Service, resolver focus.ProjectResolver) *Service {
	return &Service{store: store, focus: focusService, resolver: resolver, now: time.Now, newID: randomID}
}

func (s *Service) Create(req CreateRequest) (Presented, error) {
	if err := s.store.Cleanup(s.now()); err != nil {
		return Presented{}, err
	}
	item, err := s.newItem(req)
	if err != nil {
		return Presented{}, err
	}
	created, _, err := s.store.CreateOrGet(item)
	if err != nil {
		if errors.Is(err, ErrIdempotencyConflict) {
			return Presented{}, opError(attention.ErrorValidation, err.Error(), "idempotency_key")
		}
		return Presented{}, err
	}
	return present(created), nil
}

func (s *Service) Get(id string) (Presented, error) {
	if err := s.store.Cleanup(s.now()); err != nil {
		return Presented{}, err
	}
	item, err := s.store.Get(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Presented{}, opError(attention.ErrorMissing, err.Error(), "suggestion_id")
		}
		return Presented{}, err
	}
	return present(item), nil
}

func (s *Service) List(pendingOnly bool) ([]Presented, error) {
	if err := s.store.Cleanup(s.now()); err != nil {
		return nil, err
	}
	items, err := s.store.List()
	if err != nil {
		return nil, err
	}
	out := make([]Presented, 0, len(items))
	for _, item := range items {
		if pendingOnly && item.State != StatePending {
			continue
		}
		out = append(out, present(item))
	}
	return out, nil
}

func (s *Service) Act(id, decision string, expected int64, actionKey string) (ActionResult, error) {
	decision = strings.ReplaceAll(strings.TrimSpace(decision), "-", "_")
	if !validDecision(decision) {
		return ActionResult{}, opError(attention.ErrorUnsupported, "unsupported suggestion action", "action")
	}
	if expected < 1 {
		return ActionResult{}, opError(attention.ErrorValidation, "revision must be positive", "revision")
	}
	if strings.TrimSpace(actionKey) == "" || !utf8.ValidString(actionKey) || len(actionKey) > 512 {
		return ActionResult{}, opError(attention.ErrorValidation, "idempotency key is required and must be at most 512 bytes", "idempotency_key")
	}
	if err := s.store.Cleanup(s.now()); err != nil {
		return ActionResult{}, err
	}
	item, err := s.store.Get(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ActionResult{}, opError(attention.ErrorMissing, err.Error(), "suggestion_id")
		}
		return ActionResult{}, err
	}
	// Exact action replay is authoritative even though the persisted receipt
	// necessarily has a newer revision than the original request.
	if item.ActionKey == actionKey {
		if item.Decision != decision {
			return ActionResult{}, opError(attention.ErrorValidation, "action idempotency conflict", "idempotency_key")
		}
		return s.result(item, decision)
	}
	if item.Revision != expected {
		return ActionResult{}, opError(attention.ErrorStale, ErrStale.Error(), "revision")
	}
	if item.State == StateExpired {
		return ActionResult{}, opError(attention.ErrorUnsupported, "suggestion has expired", "suggestion_id")
	}
	if item.State != StatePending {
		return ActionResult{}, opError(attention.ErrorUnsupported, "suggestion is no longer pending", "suggestion_id")
	}

	if decision == DecisionDismiss {
		updated, err := s.store.Replace(id, expected, func(v *Item) (bool, error) {
			v.State, v.Decision, v.ActionKey, v.UpdatedAt = StateDismissed, decision, actionKey, utcNow(s.now())
			return true, nil
		})
		if err != nil {
			return ActionResult{}, translateStoreError(err)
		}
		return s.result(updated, decision)
	}

	if decision == DecisionDoNext {
		resolved := s.resolver.ResolveReference(item.Project)
		if item.Project == nil || resolved.Reference == nil || resolved.Availability != focus.ProjectAvailable || resolved.Path == "" {
			return ActionResult{}, opError(attention.ErrorMissing, "suggestion project is unavailable for launch", "project")
		}
	}
	lifecycle := attention.FocusToday
	if decision == DecisionLater {
		lifecycle = attention.FocusLater
	}
	focusItem, _, err := s.focus.CreateOrGet(focus.CreateRequest{
		Title: item.Title, Prompt: item.Prompt, Lifecycle: lifecycle, Project: item.Project,
		Origin:    &attention.ProvenanceReference{Kind: "deferred_suggestion", SourceID: item.ID},
		OriginKey: "deferred_suggestion:" + item.ID,
	})
	if err != nil {
		return ActionResult{}, err
	}
	updated, err := s.store.Replace(id, expected, func(v *Item) (bool, error) {
		v.State, v.Decision, v.ActionKey, v.UpdatedAt = StateAccepted, decision, actionKey, utcNow(s.now())
		v.FocusID, v.FocusRevision = focusItem.ID, focusItem.Revision
		return true, nil
	})
	if err != nil {
		return ActionResult{}, translateStoreError(err)
	}
	return s.result(updated, decision)
}

func (s *Service) result(item Item, decision string) (ActionResult, error) {
	result := ActionResult{SchemaVersion: attention.SchemaVersion, Suggestion: present(item)}
	if item.FocusID != "" {
		listed, err := s.focus.Get(item.FocusID)
		if err != nil {
			return ActionResult{}, err
		}
		focusItem := listed.Item
		result.Focus = &focusItem
	}
	if decision == DecisionDoNext {
		intent, err := s.focus.LaunchIntent(item.FocusID)
		if err != nil {
			return ActionResult{}, err
		}
		result.Launch = &LaunchIntent{Prompt: intent.Prompt, Project: intent.Project, Path: intent.Path}
	}
	return result, nil
}

func (s *Service) newItem(req CreateRequest) (Item, error) {
	req.Kind = strings.TrimSpace(req.Kind)
	if req.Kind == "" {
		req.Kind = "deferred_work"
	}
	if !utf8.ValidString(req.Kind) || len(req.Kind) > 64 {
		return Item{}, opError(attention.ErrorValidation, "kind must be valid UTF-8 and at most 64 bytes", "kind")
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Title == "" || !utf8.ValidString(req.Title) || utf8.RuneCountInString(req.Title) > attention.MaxSubjectCharacters {
		return Item{}, opError(attention.ErrorValidation, "title is required, valid UTF-8, and at most 200 characters", "title")
	}
	if req.Reason == "" || !utf8.ValidString(req.Reason) || len(req.Reason) > attention.MaxBodyBytes {
		return Item{}, opError(attention.ErrorValidation, "reason is required, valid UTF-8, and at most 65536 bytes", "reason")
	}
	if strings.TrimSpace(req.Prompt) == "" || !utf8.ValidString(req.Prompt) || len(req.Prompt) > attention.MaxFocusPromptBytes {
		return Item{}, opError(attention.ErrorValidation, "prompt is required, valid UTF-8, and at most 65536 bytes", "prompt")
	}
	if req.Project != nil && (strings.TrimSpace(req.Project.PeerID) == "" || strings.TrimSpace(req.Project.DisplayName) == "") {
		return Item{}, opError(attention.ErrorValidation, "project peer_id and display_name are required", "project")
	}
	if req.Provenance == nil || strings.TrimSpace(req.Provenance.Kind) == "" || strings.TrimSpace(req.Provenance.SourceID) == "" {
		return Item{}, opError(attention.ErrorValidation, "typed run or session provenance is required", "provenance")
	}
	if req.Provenance.Kind != "run" && req.Provenance.Kind != "session" && req.Provenance.Kind != "conversation" {
		return Item{}, opError(attention.ErrorValidation, "provenance kind must be run, session, or conversation", "provenance.kind")
	}
	if !utf8.ValidString(req.Provenance.SourceID) || len(req.Provenance.SourceID) > 512 {
		return Item{}, opError(attention.ErrorValidation, "provenance source_id must be valid UTF-8 and at most 512 bytes", "provenance.source_id")
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" || !utf8.ValidString(req.IdempotencyKey) || len(req.IdempotencyKey) > 512 {
		return Item{}, opError(attention.ErrorValidation, "idempotency key is required and must be at most 512 bytes", "idempotency_key")
	}
	id, err := s.newID()
	if err != nil {
		return Item{}, err
	}
	now := s.now().UTC()
	return Item{SchemaVersion: attention.SchemaVersion, ID: id, Kind: req.Kind, Title: req.Title, Reason: req.Reason, Prompt: req.Prompt, Project: req.Project, Provenance: req.Provenance, IdempotencyKey: strings.TrimSpace(req.IdempotencyKey), CreatedAt: utcNow(now), UpdatedAt: utcNow(now), ExpiresAt: utcNow(now.Add(7 * 24 * time.Hour)), RetainUntil: utcNow(now.Add(30 * 24 * time.Hour)), State: StatePending}, nil
}

func present(item Item) Presented {
	result := Presented{Item: item}
	if item.State != StatePending {
		return result
	}
	for _, action := range []struct{ operationID, id, label string }{
		{attention.OperationSuggestionToday, DecisionToday, "Accept for Today"},
		{attention.OperationSuggestionLater, DecisionLater, "Accept for Later"},
		{attention.OperationSuggestionDoNext, DecisionDoNext, "Accept and Do Next"},
		{attention.OperationSuggestionDismiss, DecisionDismiss, "Dismiss"},
	} {
		descriptor, ok := attention.AnnotateActionDescriptor(attention.ActionDescriptor{ID: action.id, Label: action.label, RequiredRowRevision: item.Revision, RequiresIdempotency: true}, action.operationID)
		if !ok {
			panic("unknown Attention operation policy: " + action.operationID)
		}
		result.Actions = append(result.Actions, descriptor)
	}
	return result
}

func validDecision(v string) bool {
	return v == DecisionToday || v == DecisionLater || v == DecisionDoNext || v == DecisionDismiss
}

func translateStoreError(err error) error {
	if errors.Is(err, ErrStale) {
		return opError(attention.ErrorStale, err.Error(), "revision")
	}
	if errors.Is(err, ErrNotFound) {
		return opError(attention.ErrorMissing, err.Error(), "suggestion_id")
	}
	return err
}

func opError(code, message, field string) error {
	return &Error{Code: code, Message: message, Field: field}
}

func randomID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate suggestion id: %w", err)
	}
	return "suggestion_" + hex.EncodeToString(b), nil
}
