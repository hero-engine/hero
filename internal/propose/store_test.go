package propose

import (
	"strings"
	"testing"
)

func envelopeAt(proposalID, batchID, sessionID, agent, specSlug, anchorValue string) *Envelope {
	return &Envelope{
		SchemaVersion: SchemaVersion,
		ProposalID:    proposalID,
		BatchID:       batchID,
		SessionID:     sessionID,
		Agent:         agent,
		Target: Target{
			SpecSlug: specSlug,
			Anchor: Anchor{
				Kind:     AnchorSection,
				Value:    anchorValue,
				Position: PositionAppend,
			},
		},
		Content: Content{Format: FormatMarkdown, Body: "body"},
	}
}

func TestStore_IngestAndList(t *testing.T) {
	s := NewStore()
	res, err := s.Ingest(envelopeAt("p-1", "b-1", "sess", "story-writer", "spec-a", "ac"))
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if res.ReplacedID != "" {
		t.Errorf("first ingest should not replace; got replaced=%q", res.ReplacedID)
	}
	if res.Envelope.EmittedAt.IsZero() {
		t.Error("ingest should stamp emitted_at")
	}

	got := s.List("sess", "", "", "")
	if len(got) != 1 {
		t.Fatalf("list len = %d, want 1", len(got))
	}
	if s.PendingCount("sess") != 1 {
		t.Errorf("pending count = %d, want 1", s.PendingCount("sess"))
	}
}

func TestStore_PerAnchorReplacement_SameAgent(t *testing.T) {
	s := NewStore()
	s.Ingest(envelopeAt("p-1", "b-1", "sess", "story-writer", "spec-a", "ac"))

	res, err := s.Ingest(envelopeAt("p-2", "b-2", "sess", "story-writer", "spec-a", "ac"))
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if res.ReplacedID != "p-1" {
		t.Errorf("expected replacement of p-1, got %q", res.ReplacedID)
	}
	if s.PendingCount("sess") != 1 {
		t.Errorf("pending count = %d, want 1 (replacement, not stack)", s.PendingCount("sess"))
	}
	if _, ok := s.Get("sess", "p-1"); ok {
		t.Error("p-1 should be evicted after replacement")
	}
	if _, ok := s.Get("sess", "p-2"); !ok {
		t.Error("p-2 should be present after replacement")
	}
}

func TestStore_PerAnchorReplacement_DifferentAgentStacks(t *testing.T) {
	s := NewStore()
	s.Ingest(envelopeAt("p-1", "b-1", "sess", "story-writer", "spec-a", "ac"))
	s.Ingest(envelopeAt("p-2", "b-2", "sess", "prd-author", "spec-a", "ac"))

	if s.PendingCount("sess") != 2 {
		t.Errorf("two agents on same anchor should stack; got pending=%d", s.PendingCount("sess"))
	}
}

func TestStore_CloseLifecycle(t *testing.T) {
	s := NewStore()
	s.Ingest(envelopeAt("p-1", "b-1", "sess", "story-writer", "spec-a", "ac"))

	rec, summary, err := s.Close("sess", "p-1", ActionAccepted, "user", "", "")
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if rec.Action != ActionAccepted {
		t.Errorf("action = %q, want accepted", rec.Action)
	}
	if rec.By != "user" {
		t.Errorf("by = %q, want user", rec.By)
	}
	if summary == nil {
		t.Fatal("single-proposal batch should produce a summary on close")
	}
	if summary.Accepted != 1 || summary.Total != 1 {
		t.Errorf("summary = %+v; want Total=1, Accepted=1", summary)
	}
	if s.PendingCount("sess") != 0 {
		t.Errorf("pending count after close = %d, want 0", s.PendingCount("sess"))
	}
}

func TestStore_EditAcceptCarriesBody(t *testing.T) {
	s := NewStore()
	s.Ingest(envelopeAt("p-1", "b-1", "sess", "story-writer", "spec-a", "ac"))

	rec, _, err := s.Close("sess", "p-1", ActionEdited, "user", "", "edited body text")
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if rec.EditedBody != "edited body text" {
		t.Errorf("edited_body = %q, want %q", rec.EditedBody, "edited body text")
	}
}

func TestStore_RejectCarriesReason(t *testing.T) {
	s := NewStore()
	s.Ingest(envelopeAt("p-1", "b-1", "sess", "story-writer", "spec-a", "ac"))

	rec, _, err := s.Close("sess", "p-1", ActionRejected, "user", "not relevant", "")
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if rec.Reason != "not relevant" {
		t.Errorf("reason = %q", rec.Reason)
	}
}

func TestStore_BulkAcceptOnBatch(t *testing.T) {
	s := NewStore()
	for i, anchor := range []string{"ac1", "ac2", "ac3", "ac4"} {
		_, err := s.Ingest(envelopeAt(
			"p-"+string(rune('a'+i)), "b-shared", "sess", "story-writer", "spec-a", anchor))
		if err != nil {
			t.Fatalf("ingest %d: %v", i, err)
		}
	}
	if s.PendingCount("sess") != 4 {
		t.Fatalf("pending = %d, want 4", s.PendingCount("sess"))
	}

	records, summary, err := s.CloseBatch("sess", "b-shared", ActionAccepted, "user")
	if err != nil {
		t.Fatalf("close batch: %v", err)
	}
	if len(records) != 4 {
		t.Errorf("records len = %d, want 4", len(records))
	}
	if summary == nil {
		t.Fatal("bulk close of full batch should produce summary")
	}
	if summary.Total != 4 || summary.Accepted != 4 {
		t.Errorf("summary = %+v", summary)
	}
	if s.PendingCount("sess") != 0 {
		t.Errorf("pending after bulk = %d, want 0", s.PendingCount("sess"))
	}
}

func TestStore_MixedBatchLifecycle(t *testing.T) {
	s := NewStore()
	for i, anchor := range []string{"ac1", "ac2", "ac3", "ac4"} {
		s.Ingest(envelopeAt("p-"+string(rune('a'+i)), "b-mix", "sess", "story-writer", "spec-a", anchor))
	}
	s.Close("sess", "p-a", ActionAccepted, "user", "", "")
	s.Close("sess", "p-b", ActionAccepted, "user", "", "")
	s.Close("sess", "p-c", ActionEdited, "user", "", "new body")
	_, summary, _ := s.Close("sess", "p-d", ActionRejected, "user", "", "")

	if summary == nil {
		t.Fatal("expected summary on final close")
	}
	if summary.Accepted != 2 || summary.Edited != 1 || summary.Rejected != 1 {
		t.Errorf("summary = %+v", summary)
	}

	got := summary.LogLine()
	want := "story-writer drafted 4 proposals → 2 accepted, 1 edited, 1 rejected, 0 dismissed"
	if got != want {
		t.Errorf("log line = %q\nwant       = %q", got, want)
	}
}

func TestStore_BulkEditAcceptRejected(t *testing.T) {
	s := NewStore()
	s.Ingest(envelopeAt("p-1", "b-1", "sess", "story-writer", "spec-a", "ac"))

	_, _, err := s.CloseBatch("sess", "b-1", ActionEdited, "user")
	if err == nil {
		t.Fatal("bulk edit should be rejected")
	}
	if !strings.Contains(err.Error(), "edit is per-proposal") {
		t.Errorf("error = %v", err)
	}
}

func TestStore_ReplacedProposalNotInBatchTotal(t *testing.T) {
	// A replaced proposal should not inflate the batch summary's total.
	s := NewStore()
	s.Ingest(envelopeAt("p-1", "b-1", "sess", "story-writer", "spec-a", "ac"))
	s.Ingest(envelopeAt("p-2", "b-1", "sess", "story-writer", "spec-a", "ac")) // replaces p-1

	_, summary, _ := s.Close("sess", "p-2", ActionAccepted, "user", "", "")
	if summary == nil {
		t.Fatal("expected summary")
	}
	if summary.Total != 1 {
		t.Errorf("total = %d, want 1 (replaced proposal must not count)", summary.Total)
	}
}

func TestStore_CloseUnknownProposal(t *testing.T) {
	s := NewStore()
	_, _, err := s.Close("sess", "nope", ActionAccepted, "user", "", "")
	if err == nil {
		t.Fatal("expected error closing unknown proposal")
	}
}
