// Package projection builds the user-global, read-only Attention view and
// delegates advertised actions to the services that own each source record.
package projection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/attention/suggestion"
)

type MailSource interface {
	Inbox(project string, unread bool) ([]mail.ListedMessage, error)
	Action(mail.ActionRequest) (mail.ActionResult, error)
}

type FocusSource interface {
	List(state string) ([]focus.ListedItem, error)
	Get(id string) (focus.ListedItem, error)
	Move(id, lifecycle string, expected int64) (focus.Item, error)
	LaunchIntent(id string) (focus.LaunchIntent, error)
}

type SuggestionSource interface {
	List(pendingOnly bool) ([]suggestion.Presented, error)
	Act(id, decision string, expected int64, actionKey string) (suggestion.ActionResult, error)
}

type Service struct {
	mail        MailSource
	focus       FocusSource
	suggestions SuggestionSource
	now         func() time.Time
}

func NewService(mailSource MailSource, focusSource FocusSource, suggestionSource SuggestionSource) *Service {
	return &Service{mail: mailSource, focus: focusSource, suggestions: suggestionSource, now: time.Now}
}

func (s *Service) Snapshot() (attention.AttentionSnapshot, error) {
	if s.mail == nil || s.focus == nil || s.suggestions == nil {
		return attention.AttentionSnapshot{}, contractError(attention.ErrorUnavailable, "attention sources are unavailable", "")
	}
	mails, err := s.mail.Inbox("", true)
	if err != nil {
		return attention.AttentionSnapshot{}, unavailable(err)
	}
	foci, err := s.focus.List(attention.FocusToday)
	if err != nil {
		return attention.AttentionSnapshot{}, unavailable(err)
	}
	suggestions, err := s.suggestions.List(true)
	if err != nil {
		return attention.AttentionSnapshot{}, unavailable(err)
	}
	rows := make([]attention.AttentionRow, 0, len(mails)+len(foci)+len(suggestions))
	for _, item := range mails {
		rows = append(rows, projectMail(item))
	}
	for _, item := range foci {
		rows = append(rows, projectFocus(item))
	}
	for _, item := range suggestions {
		rows = append(rows, s.projectSuggestion(item))
	}
	orderRows(rows)
	return s.finish(rows), nil
}

func (s *Service) finish(rows []attention.AttentionRow) attention.AttentionSnapshot {
	counts := attention.AttentionCounts{Total: len(rows)}
	for _, row := range rows {
		switch row.Group {
		case "mail":
			counts.Mail++
		case "focus":
			counts.Focus++
		case "suggestion":
			counts.Suggestion++
		}
	}
	revision := snapshotRevision(rows)
	return attention.AttentionSnapshot{
		SchemaVersion: attention.SchemaVersion,
		GeneratedAt:   s.now().UTC().Format(time.RFC3339Nano),
		Revision:      revision, RefreshToken: revision, Counts: counts, Rows: rows,
	}
}

func projectMail(item mail.ListedMessage) attention.AttentionRow {
	// Mail mutations compare against the receipt revision. A missing receipt is
	// revision zero even when the immutable envelope has its own content hash.
	revision := int64(0)
	activity := item.CreatedAt
	if item.Receipt != nil {
		revision = item.Receipt.Revision
		for _, value := range []string{item.Receipt.AcknowledgedAt, item.Receipt.ReadAt} {
			if value != "" {
				activity = value
			}
		}
	}
	return attention.AttentionRow{
		SchemaVersion: attention.SchemaVersion, ID: "mail:" + item.ID,
		SourceKind: "mail", SourceID: item.ID, Project: item.Recipient,
		Title: item.Subject, Summary: item.Body, Body: item.Body, Timestamp: activity,
		CreatedAt: item.CreatedAt, ActivityAt: activity, Group: "mail",
		Unread: true, Availability: "available", Revision: revision,
		Actions: mailActions(revision), Provenance: item.Provenance,
	}
}

func projectFocus(item focus.ListedItem) attention.AttentionRow {
	var project attention.ProjectReference
	if item.Project != nil {
		project = *item.Project
	}
	provenance := []attention.ProvenanceReference{}
	if item.Origin != nil {
		provenance = append(provenance, *item.Origin)
	}
	return attention.AttentionRow{
		SchemaVersion: attention.SchemaVersion, ID: "focus:" + item.ID,
		SourceKind: "focus", SourceID: item.ID, Project: project,
		Title: item.Title, Summary: item.Prompt, Body: item.Prompt, Timestamp: item.UpdatedAt,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, ActivityAt: item.UpdatedAt,
		Group: "focus", Today: true, Availability: item.Availability,
		Revision: item.Revision, Actions: focusActions(item.Revision, item.Project != nil && item.Availability == focus.ProjectAvailable),
		Provenance: provenance,
	}
}

func (s *Service) projectSuggestion(item suggestion.Presented) attention.AttentionRow {
	var project attention.ProjectReference
	if item.Project != nil {
		project = *item.Project
	}
	provenance := []attention.ProvenanceReference{}
	if item.Provenance != nil {
		provenance = append(provenance, *item.Provenance)
	}
	actions := append([]attention.ActionDescriptor(nil), item.Actions...)
	for i := range actions {
		if len(actions[i].InputSchema) == 0 {
			actions[i].InputSchema = noInput
		}
	}
	availability := focus.ProjectAvailable
	if source, ok := s.focus.(interface {
		ProjectAvailability(*attention.ProjectReference) string
	}); ok {
		availability = source.ProjectAvailability(item.Project)
	}
	if item.Project == nil || availability != focus.ProjectAvailable {
		filtered := actions[:0]
		for _, descriptor := range actions {
			if descriptor.ID != suggestion.DecisionDoNext {
				filtered = append(filtered, descriptor)
			}
		}
		actions = filtered
	}
	return attention.AttentionRow{
		SchemaVersion: attention.SchemaVersion, ID: "suggestion:" + item.ID,
		SourceKind: "suggestion", SourceID: item.ID, Project: project,
		Title: item.Title, Summary: item.Reason, Body: item.Reason, Timestamp: item.UpdatedAt,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, ActivityAt: item.UpdatedAt,
		Group: "suggestion", Availability: availability, Revision: item.Revision,
		Actions: actions, Provenance: provenance,
	}
}

func snapshotRevision(rows []attention.AttentionRow) string {
	hash := sha256.New()
	encoder := json.NewEncoder(hash)
	encoder.SetEscapeHTML(false)
	for _, row := range rows {
		ids := make([]string, len(row.Actions))
		for i := range row.Actions {
			ids[i] = row.Actions[i].ID
		}
		_ = encoder.Encode(struct {
			Kind     string   `json:"kind"`
			ID       string   `json:"id"`
			Revision int64    `json:"revision"`
			Actions  []string `json:"actions"`
		}{row.SourceKind, row.SourceID, row.Revision, ids})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func (s *Service) Dispatch(request attention.ActionRequest) attention.ActionResult {
	fail := func(err *attention.ContractError, row *attention.AttentionRow) attention.ActionResult {
		return attention.ActionResult{SchemaVersion: attention.SchemaVersion, Row: row, Error: err}
	}
	if request.SchemaVersion != attention.SchemaVersion {
		return fail(contractError(attention.ErrorIncompatibleVersion, "unsupported attention schema version", "schema_version"), nil)
	}
	snapshot, err := s.Snapshot()
	if err != nil {
		var ce *attention.ContractError
		if errors.As(err, &ce) {
			return fail(ce, nil)
		}
		return fail(contractError(attention.ErrorUnavailable, err.Error(), ""), nil)
	}
	var row *attention.AttentionRow
	for i := range snapshot.Rows {
		if snapshot.Rows[i].ID == request.RowID {
			row = &snapshot.Rows[i]
			break
		}
	}
	if row == nil {
		return fail(contractError(attention.ErrorMissing, "attention row was not found", "row_id"), nil)
	}
	if request.RowID != row.SourceKind+":"+row.SourceID {
		return fail(contractError(attention.ErrorValidation, "row identity does not match its source", "row_id"), row)
	}
	var descriptor *attention.ActionDescriptor
	for i := range row.Actions {
		if row.Actions[i].ID == request.ActionID {
			descriptor = &row.Actions[i]
			break
		}
	}
	if descriptor == nil {
		return fail(contractError(attention.ErrorUnsupported, "action is not advertised for this row", "action_id"), row)
	}
	if validation := attention.ValidateActionRequest(request, *descriptor); validation != nil {
		return fail(validation, row)
	}
	if validation := validateInput(descriptor.InputSchema, request.Input); validation != nil {
		return fail(validation, row)
	}
	result, dispatchErr := s.dispatchSource(*row, request)
	if dispatchErr != nil {
		return fail(translateError(dispatchErr), row)
	}
	updated, refreshErr := s.Snapshot()
	if refreshErr != nil {
		return fail(contractError(attention.ErrorUnavailable, refreshErr.Error(), ""), nil)
	}
	result.SchemaVersion = attention.SchemaVersion
	result.SnapshotRevision = updated.Revision
	for i := range updated.Rows {
		if updated.Rows[i].ID == request.RowID {
			result.Row = &updated.Rows[i]
			return result
		}
	}
	result.RemovedRowID = request.RowID
	return result
}

func (s *Service) dispatchSource(row attention.AttentionRow, request attention.ActionRequest) (attention.ActionResult, error) {
	switch row.SourceKind {
	case "mail":
		actionID := request.ActionID
		if actionID == "mark_read" {
			actionID = mail.ActionRead
		}
		var input struct {
			Note         string `json:"note"`
			ArtifactType string `json:"artifact_type"`
		}
		_ = json.Unmarshal(request.Input, &input)
		source, err := s.mail.Action(mail.ActionRequest{MessageID: row.SourceID, Action: actionID, ExpectedRevision: request.RowRevision, IdempotencyKey: request.IdempotencyKey, Note: input.Note, ArtifactType: input.ArtifactType})
		if err != nil {
			return attention.ActionResult{}, err
		}
		raw, _ := json.Marshal(source)
		result := attention.ActionResult{Source: raw}
		if source.Navigation != nil {
			result.Navigation = &attention.NavigationReference{Project: source.Project, Target: source.Navigation.Slug}
		}
		return result, nil
	case "focus":
		if request.ActionID == "launch" {
			intent, err := s.focus.LaunchIntent(row.SourceID)
			if err != nil {
				return attention.ActionResult{}, err
			}
			raw, _ := json.Marshal(intent)
			return attention.ActionResult{Source: raw, Launch: &attention.LaunchIntent{Project: intent.Project, Prompt: intent.Prompt}}, nil
		}
		state := map[string]string{"move_inbox": attention.FocusInbox, "move_later": attention.FocusLater, "complete": attention.FocusDone}[request.ActionID]
		source, err := s.focus.Move(row.SourceID, state, request.RowRevision)
		if err != nil {
			return attention.ActionResult{}, err
		}
		raw, _ := json.Marshal(source)
		return attention.ActionResult{Source: raw}, nil
	case "suggestion":
		source, err := s.suggestions.Act(row.SourceID, request.ActionID, request.RowRevision, request.IdempotencyKey)
		if err != nil {
			return attention.ActionResult{}, err
		}
		raw, _ := json.Marshal(source)
		result := attention.ActionResult{Source: raw}
		if source.Launch != nil {
			result.Launch = &attention.LaunchIntent{Project: source.Launch.Project, Prompt: source.Launch.Prompt}
		}
		return result, nil
	default:
		return attention.ActionResult{}, contractError(attention.ErrorUnsupported, "unknown attention source kind", "row_id")
	}
}

func unavailable(err error) error {
	return contractError(attention.ErrorUnavailable, fmt.Sprintf("attention state unavailable: %v", err), "")
}

func translateError(err error) *attention.ContractError {
	var ce *attention.ContractError
	if errors.As(err, &ce) {
		return ce
	}
	var se *suggestion.Error
	if errors.As(err, &se) {
		return contractError(se.Code, se.Message, se.Field)
	}
	switch {
	case errors.Is(err, focus.ErrStale), errors.Is(err, mail.ErrStale):
		return contractError(attention.ErrorStale, err.Error(), "row_revision")
	case errors.Is(err, focus.ErrNotFound), errors.Is(err, mail.ErrNotFound):
		return contractError(attention.ErrorMissing, err.Error(), "row_id")
	case errors.Is(err, mail.ErrUnsupportedAction):
		return contractError(attention.ErrorUnsupported, err.Error(), "action_id")
	default:
		code := attention.ErrorValidation
		if strings.Contains(strings.ToLower(err.Error()), "unavailable") {
			code = attention.ErrorUnavailable
		}
		return contractError(code, err.Error(), "")
	}
}
