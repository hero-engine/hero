package spec

import (
	"testing"
	"time"
)

func TestParseAcceptanceCriteria_BoldParagraphForm(t *testing.T) {
	s := mustParse(t, `---
title: T
type: feature
---
# T

## Acceptance criteria

**AC-1:** First criterion text spans
multiple lines until the next entry.

**AC-2:** Second criterion is short.
`)
	acs := s.ParseAcceptanceCriteria()
	if len(acs) != 2 {
		t.Fatalf("got %d ACs, want 2: %#v", len(acs), acs)
	}
	if acs[0].ID != "AC-1" {
		t.Errorf("acs[0].ID = %q, want AC-1", acs[0].ID)
	}
	if acs[0].Statement == "" || !contains(acs[0].Statement, "First criterion") {
		t.Errorf("acs[0].Statement = %q", acs[0].Statement)
	}
	if !contains(acs[0].Statement, "next entry") {
		t.Errorf("expected continuation lines joined: %q", acs[0].Statement)
	}
	if acs[1].ID != "AC-2" {
		t.Errorf("acs[1].ID = %q, want AC-2", acs[1].ID)
	}
}

func TestParseAcceptanceCriteria_BulletForm(t *testing.T) {
	s := mustParse(t, `---
title: T
type: feature
---
# T

## Acceptance Criteria

- **AC-1:** Bullet bold form
- AC-2: Bullet plain form
`)
	acs := s.ParseAcceptanceCriteria()
	if len(acs) != 2 {
		t.Fatalf("got %d ACs, want 2: %#v", len(acs), acs)
	}
	if acs[0].Statement != "Bullet bold form" {
		t.Errorf("acs[0].Statement = %q", acs[0].Statement)
	}
	if acs[1].Statement != "Bullet plain form" {
		t.Errorf("acs[1].Statement = %q", acs[1].Statement)
	}
}

func TestParseAcceptanceCriteria_HeadingWithSuffix(t *testing.T) {
	s := mustParse(t, `---
title: T
type: feature
---
# T

## Acceptance criteria (build-out-as-we-go set)

**AC-1:** Statement.
`)
	acs := s.ParseAcceptanceCriteria()
	if len(acs) != 1 {
		t.Fatalf("got %d ACs, want 1", len(acs))
	}
}

func TestParseAcceptanceCriteria_NoSection(t *testing.T) {
	s := mustParse(t, `---
title: T
type: feature
---
# T

## Goal

Nothing here.
`)
	if acs := s.ParseAcceptanceCriteria(); len(acs) != 0 {
		t.Errorf("expected 0 ACs, got %d", len(acs))
	}
}

func mustParse(t *testing.T, content string) *Spec {
	t.Helper()
	s, err := Parse(content, "/tmp/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return s
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestAcceptanceCriteria_LabeledEARSSatisfiesBothConsumers guards the
// invariant that the two acceptance-criteria consumers agree on one bullet
// form. ParseAcceptanceCriteria only makes an entry addressable when it
// carries an AC-N label; ClassifyCriterion must still see the EARS keywords
// underneath that label. Before the label strip these were mutually
// exclusive: labeled ACs were addressable but always classified freeform,
// and unlabeled EARS classified correctly but produced zero addressable
// criteria.
func TestAcceptanceCriteria_LabeledEARSSatisfiesBothConsumers(t *testing.T) {
	s := mustParse(t, `---
title: T
type: feature
---
# T

## Acceptance criteria

- **AC-1:** WHEN a user clicks export THE SYSTEM SHALL enqueue a job
- **AC-2:** THE SYSTEM SHALL log every failed login attempt
`)

	acs := s.ParseAcceptanceCriteria()
	if len(acs) != 2 {
		t.Fatalf("got %d addressable ACs, want 2: %#v", len(acs), acs)
	}

	criteria := s.AcceptanceCriteria()
	if len(criteria) != 2 {
		t.Fatalf("got %d classified criteria, want 2: %#v", len(criteria), criteria)
	}
	for i, c := range criteria {
		if !c.Kind.IsEARS() {
			t.Errorf("criteria[%d].Kind = %v, want an EARS kind (raw=%q)", i, c.Kind, c.Raw)
		}
	}

	// Raw keeps the label so lint and contract output still show the AC id.
	if !contains(criteria[0].Raw, "AC-1") {
		t.Errorf("criteria[0].Raw = %q, want it to preserve the AC-1 label", criteria[0].Raw)
	}
}
