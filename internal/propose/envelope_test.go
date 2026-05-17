package propose

import (
	"strings"
	"testing"
)

func validEnvelope() *Envelope {
	return &Envelope{
		SchemaVersion: SchemaVersion,
		ProposalID:    "p-1",
		BatchID:       "b-1",
		SessionID:     "sess-1",
		Agent:         "story-writer",
		Target: Target{
			SpecSlug: "csv-export",
			Anchor: Anchor{
				Kind:     AnchorSection,
				Value:    "acceptance_criteria",
				Position: PositionAppend,
			},
		},
		Content: Content{
			Format: FormatMarkdown,
			Body:   "- THE SYSTEM SHALL emit a BOM when --bom is passed.",
		},
	}
}

func TestEnvelopeValidate_OK(t *testing.T) {
	e := validEnvelope()
	if err := e.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestEnvelopeValidate_MissingFields(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Envelope)
		wantSub string
	}{
		{"no schema_version", func(e *Envelope) { e.SchemaVersion = "" }, "schema_version"},
		{"no proposal_id", func(e *Envelope) { e.ProposalID = "" }, "proposal_id"},
		{"no batch_id", func(e *Envelope) { e.BatchID = "" }, "batch_id"},
		{"no session_id", func(e *Envelope) { e.SessionID = "" }, "session_id"},
		{"no agent", func(e *Envelope) { e.Agent = "" }, "agent"},
		{"no spec_slug", func(e *Envelope) { e.Target.SpecSlug = "" }, "spec_slug"},
		{"no anchor.kind", func(e *Envelope) { e.Target.Anchor.Kind = "" }, "anchor.kind"},
		{"bad anchor.kind", func(e *Envelope) { e.Target.Anchor.Kind = "nonsense" }, "not a known kind"},
		{"no anchor.value", func(e *Envelope) { e.Target.Anchor.Value = "" }, "anchor.value"},
		{"bad anchor.position", func(e *Envelope) { e.Target.Anchor.Position = "sideways" }, "not a known position"},
		{"no content.format", func(e *Envelope) { e.Content.Format = "" }, "content.format"},
		{"bad content.format", func(e *Envelope) { e.Content.Format = "binary" }, "not a known format"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := validEnvelope()
			tc.mutate(e)
			err := e.Validate()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestParseEnvelope_RoundTrip(t *testing.T) {
	data := []byte(`{
	  "schema_version": "1.0",
	  "proposal_id": "p-abc",
	  "batch_id": "b-1",
	  "session_id": "sess-1",
	  "agent": "story-writer",
	  "target": {
	    "spec_slug": "csv-export",
	    "anchor": {"kind": "section", "value": "acceptance_criteria", "position": "append"}
	  },
	  "content": {"format": "markdown", "body": "hello"}
	}`)
	e, err := ParseEnvelope(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if e.ProposalID != "p-abc" {
		t.Fatalf("proposal_id = %q, want p-abc", e.ProposalID)
	}
	if e.Target.Anchor.Position != PositionAppend {
		t.Fatalf("position = %q, want %q", e.Target.Anchor.Position, PositionAppend)
	}
}

func TestParseEnvelope_BadJSON(t *testing.T) {
	_, err := ParseEnvelope([]byte("not json"))
	if err == nil {
		t.Fatal("expected error on malformed JSON")
	}
}

func TestParseEnvelope_FailsValidation(t *testing.T) {
	// Missing required fields.
	_, err := ParseEnvelope([]byte(`{"schema_version":"1.0"}`))
	if err == nil {
		t.Fatal("expected validation error on empty envelope")
	}
}

func TestHeroProposalPrefix(t *testing.T) {
	// Pin the wire prefix — changing this would break the cross-language contract.
	if HeroProposalPrefix != "HERO-PROPOSAL: " {
		t.Fatalf("prefix = %q, want %q", HeroProposalPrefix, "HERO-PROPOSAL: ")
	}
}
