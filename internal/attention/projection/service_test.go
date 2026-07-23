package projection

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/attention/suggestion"
)

type fakeMail struct {
	items []mail.ListedMessage
	acted mail.ActionRequest
	err   error
}

func (f *fakeMail) Inbox(string, bool) ([]mail.ListedMessage, error) { return f.items, f.err }
func (f *fakeMail) Action(request mail.ActionRequest) (mail.ActionResult, error) {
	f.acted = request
	if f.err != nil {
		return mail.ActionResult{}, f.err
	}
	for i := range f.items {
		if f.items[i].ID == request.MessageID {
			f.items = append(f.items[:i], f.items[i+1:]...)
			break
		}
	}
	return mail.ActionResult{MessageID: request.MessageID}, nil
}

type fakeFocus struct {
	items []focus.ListedItem
	moved string
	err   error
}

func (f *fakeFocus) List(string) ([]focus.ListedItem, error) { return f.items, f.err }
func (f *fakeFocus) Get(id string) (focus.ListedItem, error) {
	for _, item := range f.items {
		if item.ID == id {
			return item, nil
		}
	}
	return focus.ListedItem{}, focus.ErrNotFound
}
func (f *fakeFocus) Move(id, lifecycle string, expected int64) (focus.Item, error) {
	f.moved = lifecycle
	if f.err != nil {
		return focus.Item{}, f.err
	}
	for i := range f.items {
		if f.items[i].ID == id {
			f.items[i].Lifecycle = lifecycle
			result := f.items[i].Item
			if lifecycle != attention.FocusToday {
				f.items = append(f.items[:i], f.items[i+1:]...)
			}
			return result, nil
		}
	}
	return focus.Item{}, focus.ErrNotFound
}
func (f *fakeFocus) LaunchIntent(id string) (focus.LaunchIntent, error) {
	item, err := f.Get(id)
	if err != nil {
		return focus.LaunchIntent{}, err
	}
	return focus.LaunchIntent{ID: id, Revision: item.Revision, Prompt: item.Prompt, Project: *item.Project}, nil
}

type fakeSuggestions struct {
	items []suggestion.Presented
	err   error
}

func (f *fakeSuggestions) List(bool) ([]suggestion.Presented, error) { return f.items, f.err }
func (f *fakeSuggestions) Act(id, decision string, expected int64, key string) (suggestion.ActionResult, error) {
	if f.err != nil {
		return suggestion.ActionResult{}, f.err
	}
	for i := range f.items {
		if f.items[i].ID == id {
			item := f.items[i]
			f.items = append(f.items[:i], f.items[i+1:]...)
			return suggestion.ActionResult{SchemaVersion: 1, Suggestion: item}, nil
		}
	}
	return suggestion.ActionResult{}, &suggestion.Error{Code: attention.ErrorMissing, Message: "missing"}
}

func TestSnapshotOrderingRevisionAndCounts(t *testing.T) {
	project := attention.ProjectReference{PeerID: "peer", DisplayName: "Project"}
	mails := &fakeMail{items: []mail.ListedMessage{
		{MailEnvelope: attention.MailEnvelope{ID: "mail_b", Recipient: project, Subject: "B", CreatedAt: "2026-07-22T12:00:00Z", Revision: 2}},
		{MailEnvelope: attention.MailEnvelope{ID: "mail_a", Recipient: project, Subject: "A", CreatedAt: "2026-07-22T11:00:00Z", Revision: 1}},
	}}
	foci := &fakeFocus{items: []focus.ListedItem{
		{Item: focus.Item{ID: "focus_old", Project: &project, Title: "Old", Prompt: "old", Lifecycle: attention.FocusToday, Revision: 3, CreatedAt: "2026-07-21T10:00:00Z", UpdatedAt: "2026-07-22T10:00:00Z"}, Availability: focus.ProjectAvailable},
		{Item: focus.Item{ID: "focus_new", Project: &project, Title: "New", Prompt: "new", Lifecycle: attention.FocusToday, Revision: 4, CreatedAt: "2026-07-21T10:00:00Z", UpdatedAt: "2026-07-22T13:00:00Z"}, Availability: focus.ProjectAvailable},
	}}
	suggestions := &fakeSuggestions{items: []suggestion.Presented{{
		Item:    suggestion.Item{ID: "suggestion_1", Project: &project, Title: "Later", Reason: "reason", CreatedAt: "2026-07-22T09:00:00Z", UpdatedAt: "2026-07-22T09:00:00Z", Revision: 5},
		Actions: []attention.ActionDescriptor{{ID: suggestion.DecisionToday, Label: "Today", RequiredRowRevision: 5, RequiresIdempotency: true}},
	}}}
	service := NewService(mails, foci, suggestions)
	service.now = func() time.Time { return time.Date(2026, 7, 22, 15, 0, 0, 0, time.UTC) }

	first, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, 7, 22, 16, 0, 0, 0, time.UTC) }
	second, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision != second.Revision || first.RefreshToken != first.Revision {
		t.Fatalf("unstable revision: %q %q", first.Revision, second.Revision)
	}
	want := []string{"mail:mail_a", "mail:mail_b", "focus:focus_new", "focus:focus_old", "suggestion:suggestion_1"}
	for i := range want {
		if first.Rows[i].ID != want[i] {
			t.Fatalf("row %d = %q, want %q", i, first.Rows[i].ID, want[i])
		}
	}
	if first.Counts.Mail != 2 || first.Counts.Focus != 2 || first.Counts.Suggestion != 1 || first.Counts.Total != 5 {
		t.Fatalf("counts = %#v", first.Counts)
	}
}

func TestDispatchValidatesAndReturnsAuthoritativeInvalidation(t *testing.T) {
	project := attention.ProjectReference{PeerID: "peer", DisplayName: "Project"}
	mails := &fakeMail{items: []mail.ListedMessage{{MailEnvelope: attention.MailEnvelope{
		ID: "mail_1", Recipient: project, Subject: "Read me", CreatedAt: "2026-07-22T11:00:00Z", Revision: 7,
	}}}}
	service := NewService(mails, &fakeFocus{}, &fakeSuggestions{})

	stale := service.Dispatch(attention.ActionRequest{SchemaVersion: 1, RowID: "mail:mail_1", ActionID: "mark_read", RowRevision: 6, IdempotencyKey: "key"})
	if stale.Error == nil || stale.Error.Code != attention.ErrorStale || stale.Row == nil {
		t.Fatalf("stale result = %#v", stale)
	}
	unsupported := service.Dispatch(attention.ActionRequest{SchemaVersion: 1, RowID: "mail:mail_1", ActionID: "reply", RowRevision: 7, IdempotencyKey: "key"})
	if unsupported.Error == nil || unsupported.Error.Code != attention.ErrorUnsupported {
		t.Fatalf("unsupported result = %#v", unsupported)
	}
	success := service.Dispatch(attention.ActionRequest{
		SchemaVersion: 1, RowID: "mail:mail_1", ActionID: "mark_read", RowRevision: 0,
		IdempotencyKey: "key", Input: json.RawMessage(`{}`),
	})
	if success.Error != nil || success.RemovedRowID != "mail:mail_1" || success.SnapshotRevision == "" {
		t.Fatalf("success = %#v", success)
	}
	if mails.acted.Action != mail.ActionRead || mails.acted.MessageID != "mail_1" {
		t.Fatalf("delegated request = %#v", mails.acted)
	}
}

func TestSnapshotUnavailableIsNotEmpty(t *testing.T) {
	service := NewService(&fakeMail{err: errors.New("disk failed")}, &fakeFocus{}, &fakeSuggestions{})
	_, err := service.Snapshot()
	var contractErr *attention.ContractError
	if !errors.As(err, &contractErr) || contractErr.Code != attention.ErrorUnavailable {
		t.Fatalf("error = %#v", err)
	}
}

func TestDispatchDelegatesFocusAndSuggestionCapabilities(t *testing.T) {
	project := attention.ProjectReference{PeerID: "peer", DisplayName: "Project"}
	foci := &fakeFocus{items: []focus.ListedItem{{
		Item:         focus.Item{ID: "focus_1", Project: &project, Title: "Finish", Prompt: "finish", Lifecycle: attention.FocusToday, Revision: 2, CreatedAt: "2026-07-22T10:00:00Z", UpdatedAt: "2026-07-22T11:00:00Z"},
		Availability: focus.ProjectAvailable,
	}}}
	focusService := NewService(&fakeMail{}, foci, &fakeSuggestions{})
	focusResult := focusService.Dispatch(attention.ActionRequest{
		SchemaVersion: 1, RowID: "focus:focus_1", ActionID: "complete",
		RowRevision: 2, IdempotencyKey: "focus-key", Input: json.RawMessage(`{}`),
	})
	if focusResult.Error != nil || focusResult.RemovedRowID != "focus:focus_1" || foci.moved != attention.FocusDone {
		t.Fatalf("focus result = %#v, moved = %q", focusResult, foci.moved)
	}

	suggestions := &fakeSuggestions{items: []suggestion.Presented{{
		Item:    suggestion.Item{ID: "suggestion_1", Project: &project, Title: "Later", Reason: "reason", CreatedAt: "2026-07-22T09:00:00Z", UpdatedAt: "2026-07-22T09:00:00Z", Revision: 5},
		Actions: []attention.ActionDescriptor{{ID: suggestion.DecisionToday, Label: "Today", RequiredRowRevision: 5, RequiresIdempotency: true}},
	}}}
	suggestionService := NewService(&fakeMail{}, &fakeFocus{}, suggestions)
	suggestionResult := suggestionService.Dispatch(attention.ActionRequest{
		SchemaVersion: 1, RowID: "suggestion:suggestion_1", ActionID: suggestion.DecisionToday,
		RowRevision: 5, IdempotencyKey: "suggestion-key", Input: json.RawMessage(`{}`),
	})
	if suggestionResult.Error != nil || suggestionResult.RemovedRowID != "suggestion:suggestion_1" {
		t.Fatalf("suggestion result = %#v", suggestionResult)
	}
}

func TestDispatchRejectsIncompatibleVersionAndUnknownInput(t *testing.T) {
	project := attention.ProjectReference{PeerID: "peer", DisplayName: "Project"}
	mails := &fakeMail{items: []mail.ListedMessage{{MailEnvelope: attention.MailEnvelope{
		ID: "mail_1", Recipient: project, Subject: "Mail", CreatedAt: "2026-07-22T11:00:00Z",
	}}}}
	service := NewService(mails, &fakeFocus{}, &fakeSuggestions{})
	incompatible := service.Dispatch(attention.ActionRequest{SchemaVersion: 2})
	if incompatible.Error == nil || incompatible.Error.Code != attention.ErrorIncompatibleVersion {
		t.Fatalf("incompatible = %#v", incompatible)
	}
	invalid := service.Dispatch(attention.ActionRequest{
		SchemaVersion: 1, RowID: "mail:mail_1", ActionID: "mark_read",
		RowRevision: 0, IdempotencyKey: "key", Input: json.RawMessage(`{"invented":true}`),
	})
	if invalid.Error == nil || invalid.Error.Code != attention.ErrorValidation || mails.acted.Action != "" {
		t.Fatalf("invalid = %#v, delegated = %#v", invalid, mails.acted)
	}
}

func TestSnapshotDoesNotAdvertiseProjectBoundActionsForUnboundRows(t *testing.T) {
	foci := &fakeFocus{items: []focus.ListedItem{{
		Item: focus.Item{
			ID: "focus_unbound", Title: "Private", Prompt: "Continue privately",
			Lifecycle: attention.FocusToday, Revision: 1,
			CreatedAt: "2026-07-22T10:00:00Z", UpdatedAt: "2026-07-22T10:00:00Z",
		},
		Availability: focus.ProjectAvailable,
	}}}
	suggestions := &fakeSuggestions{items: []suggestion.Presented{{
		Item: suggestion.Item{
			ID: "suggestion_unbound", Title: "Maybe", Reason: "No project",
			CreatedAt: "2026-07-22T09:00:00Z", UpdatedAt: "2026-07-22T09:00:00Z",
			Revision: 1,
		},
		Actions: []attention.ActionDescriptor{
			{ID: suggestion.DecisionDoNext, Label: "Do Next", RequiredRowRevision: 1, RequiresIdempotency: true},
			{ID: suggestion.DecisionToday, Label: "Today", RequiredRowRevision: 1, RequiresIdempotency: true},
		},
	}}}
	snapshot, err := NewService(&fakeMail{}, foci, suggestions).Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range snapshot.Rows {
		for _, descriptor := range row.Actions {
			if row.SourceKind == "focus" && descriptor.ID == "launch" {
				t.Fatalf("unbound Focus advertised launch: %#v", row.Actions)
			}
			if row.SourceKind == "suggestion" && descriptor.ID == suggestion.DecisionDoNext {
				t.Fatalf("unbound suggestion advertised do_next: %#v", row.Actions)
			}
		}
	}
}
