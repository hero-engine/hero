package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/attention/projection"
	"github.com/hero-engine/hero/internal/attention/suggestion"
)

type attentionCLIMail struct{ items []mail.ListedMessage }

func (f *attentionCLIMail) Inbox(string, bool) ([]mail.ListedMessage, error) {
	return f.items, nil
}
func (f *attentionCLIMail) Action(mail.ActionRequest) (mail.ActionResult, error) {
	return mail.ActionResult{}, nil
}

type attentionCLIFocus struct{}

func (*attentionCLIFocus) List(string) ([]focus.ListedItem, error) { return nil, nil }
func (*attentionCLIFocus) Get(string) (focus.ListedItem, error) {
	return focus.ListedItem{}, focus.ErrNotFound
}
func (*attentionCLIFocus) Move(string, string, int64) (focus.Item, error) {
	return focus.Item{}, nil
}
func (*attentionCLIFocus) LaunchIntent(string) (focus.LaunchIntent, error) {
	return focus.LaunchIntent{}, nil
}

type attentionCLISuggestions struct{}

func (*attentionCLISuggestions) List(bool) ([]suggestion.Presented, error) { return nil, nil }
func (*attentionCLISuggestions) Act(string, string, int64, string) (suggestion.ActionResult, error) {
	return suggestion.ActionResult{}, nil
}

func TestAttentionTodayJSONMatchesProjectionService(t *testing.T) {
	project := attention.ProjectReference{PeerID: "peer", DisplayName: "Project"}
	service := projection.NewService(
		&attentionCLIMail{items: []mail.ListedMessage{{MailEnvelope: attention.MailEnvelope{
			ID: "mail_cli", Recipient: project, Subject: "CLI parity", Body: "Same projection record",
			CreatedAt: "2026-07-22T18:00:00Z",
		}}}},
		&attentionCLIFocus{},
		&attentionCLISuggestions{},
	)
	oldLoader := attentionProjectionServiceLoader
	attentionProjectionServiceLoader = func() (*projection.Service, error) { return service, nil }
	t.Cleanup(func() { attentionProjectionServiceLoader = oldLoader })

	command := newAttentionCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"today", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	var cliSnapshot attention.AttentionSnapshot
	if err := json.Unmarshal(output.Bytes(), &cliSnapshot); err != nil {
		t.Fatal(err)
	}
	direct, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	cliSnapshot.GeneratedAt = ""
	direct.GeneratedAt = ""
	if cliSnapshot.Revision != direct.Revision || len(cliSnapshot.Rows) != 1 ||
		cliSnapshot.Rows[0].ID != direct.Rows[0].ID ||
		cliSnapshot.Rows[0].Summary != direct.Rows[0].Summary {
		t.Fatalf("CLI = %#v\ndirect = %#v", cliSnapshot, direct)
	}
}
