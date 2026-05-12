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
