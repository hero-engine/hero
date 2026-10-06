package workmodel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// next-step-engine AC-1/AC-2: every row of read-contract-v1's table.
func TestNextStepTable(t *testing.T) {
	c := newCorpus(t)
	seed(c)
	ledger := "\n## Completion Ledger\n\n### Acceptance Criteria\n\n| # | C | Status | Note |\n|---|---|---|---|\n| 1 | AC-1 | DONE | ok |\n\n### Changes\n\n| # | I | Status | Note |\n|---|---|---|---|\n| 1 | x | DONE | ok |\n"
	verified := c.write("specs/verified", fm("verified", "feature", "completed", "completed_at: 2026-10-02T00:00:00Z\n")+designedBody+ledger)
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(filepath.Join(filepath.Dir(verified), "delivery-audit.md"), []byte("# Delivery audit — verified\n\n**Verdict:** SHIP\n"), 0o644)
	held := c.write("planning/features/held", fm("held", "feature", "delivering", "")+designedBody)
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(filepath.Join(filepath.Dir(held), "delivery-audit.md"), []byte("# Delivery audit — held\n\n**Verdict:** HOLD\n"), 0o644)
	c.write("planning/features/regressed", fm("regressed", "feature", "regressed", "")+designedBody)
	c.write("planning/features/handed", fm("handed", "feature", "handed_off", "")+designedBody)
	c.write("planning/initiatives/empty", fm("empty", "initiative", "planning", ""))
	c.write("planning/bugs/blockedbug", fm("blockedbug", "bug", "planning", "depends-on: [active]\n")+"\n## Root Cause\n\nX.\n\n## Changes\n\nY.\n")

	specs, _ := c.build()
	items := Build(specs, Options{Now: testNow, Root: c.root})
	ApplyNext(items, specs)
	by := map[string]Item{}
	for _, it := range items {
		by[it.Slug] = it
	}

	type want struct {
		action, label, command, phaseLabel, phaseState string
		enabled                                        bool
		reason                                         string
	}
	cases := map[string]*want{
		"stub":        {ActionDesign, "Design", "/design stub", "Planning", PhaseReady, true, ""},
		"ready":       {ActionDeliver, "Deliver", "/deliver ready", "Ready", PhaseReady, true, ""},
		"blocked":     {ActionDeliver, "Deliver", "/deliver blocked", "Planning", PhaseWaiting, false, "waits on ready"},
		"active":      {ActionDeliver, "Continue", "/deliver active", "Delivering", PhaseActive, true, ""},
		"held":        {ActionDeliver, "Continue", "/deliver held", "Delivering", PhaseAttention, true, ""},
		"undiagnosed": {ActionDiagnose, "Diagnose", "/diagnose undiagnosed", "Reported", PhaseReady, true, ""},
		"diagnosed":   {ActionDeliver, "Fix", "/deliver diagnosed", "Diagnosed", PhaseReady, true, ""},
		"blockedbug":  {ActionDeliver, "Fix", "/deliver blockedbug", "Diagnosed", PhaseWaiting, false, "waits on active"},
		"regressed":   {ActionDiagnose, "Diagnose", "/diagnose regressed", "Regressed", PhaseAttention, true, ""},
		"handed":      {ActionDeliver, "Deliver", "/deliver handed", "With peer", PhaseWaiting, false, "handed off to a peer"},
		"init":        {ActionDrive, "Drive", "/drive init", "Driving", PhaseActive, true, ""},
		"empty":       {ActionDesign, "Compose", "/compose empty", "Planning", PhaseReady, true, ""},
		"choice":      {ActionDesign, "Decide", "/decide choice", "Proposed", PhaseReady, true, ""},
		"recent":      {ActionVerify, "Verify", "/verify recent", "Delivered?", PhaseAttention, true, ""},
		"old":         nil, // finished before the recently_done window: no action
		"verified":    nil,
		"gone":        nil,
	}
	for slug, w := range cases {
		got := by[slug].Next
		if w == nil {
			if got != nil {
				t.Errorf("%s next = %+v, want null", slug, got)
			}
			continue
		}
		if got == nil {
			t.Errorf("%s next = null, want %s", slug, w.action)
			continue
		}
		reason := ""
		if got.Reason != nil {
			reason = *got.Reason
		}
		if got.Action != w.action || got.Label != w.label || got.Command != w.command || got.Phase.Label != w.phaseLabel || got.Phase.State != w.phaseState || got.Enabled != w.enabled || reason != w.reason {
			t.Errorf("%s next = %+v (reason %q), want %+v", slug, *got, reason, *w)
		}
	}
}

// next-step-engine AC-3: requester's rules hold for every item.
func TestNextStepInvariants(t *testing.T) {
	c := newCorpus(t)
	seed(c)
	specs, _ := c.build()
	items := Build(specs, Options{Now: testNow, Root: c.root})
	ApplyNext(items, specs)
	for _, it := range items {
		n := it.Next
		if n == nil {
			continue
		}
		if !strings.HasPrefix(n.Command, "/") || strings.ContainsAny(n.Command, "|&;$`") {
			t.Errorf("%s command %q is not a chat-sendable slash command", it.Slug, n.Command)
		}
		acts := map[string]bool{n.Action: true}
		for _, e := range n.Extras {
			if !strings.HasPrefix(e.Command, "/") {
				t.Errorf("%s extra command %q", it.Slug, e.Command)
			}
			if e.Action == ActionDeliver || e.Action == ActionDiagnose {
				acts[e.Action] = true
			}
		}
		if acts[ActionDiagnose] && acts[ActionDeliver] {
			t.Errorf("%s offers both Diagnose and Deliver", it.Slug)
		}
		finished := it.Lane == LaneRecentlyDone || (it.CompletedAt != nil)
		if finished && n.Action == ActionDeliver {
			t.Errorf("%s is finished but offers Deliver", it.Slug)
		}
		if n.Phase.Label == "Delivered" && (it.Verify == nil || it.Verify.State != VerifyPassed) {
			t.Errorf("%s shows Delivered without a passed verify", it.Slug)
		}
		if n.Extras == nil {
			t.Errorf("%s extras must be [] not null", it.Slug)
		}
	}
}

// Audit round 2: a promoted intake sharing a slug must not shadow the live
// spec; signed-off skips count as verified like `hero spec verify`; and the
// table rows the first pass left untested.
func TestRound2AuditCases(t *testing.T) {
	c := newCorpus(t)
	seed(c)
	// Intake written after the feature so a naive last-wins index picks it.
	c.write("planning/features/mailfeat", fm("mailfeat", "feature", "delivering", "")+designedBody)
	c.write("planning/intake/mailfeat", fm("mailfeat", "intake", "promoted", ""))
	// A knowledge explainer sharing a work spec's slug, discovered first.
	c.write("knowledge/explainers/active", fm("active", "explainer", "active", ""))
	c.write("planning/features/watcher", fm("watcher", "feature", "planning", "depends-on: [mailfeat]\n")+designedBody)
	c.write("planning/features/review", fm("review", "feature", "in-review", "")+designedBody)
	c.write("planning/features/waiting", fm("waiting", "feature", "awaiting_peer", "")+designedBody)
	c.write("planning/features/back", fm("back", "feature", "handed_back", "")+designedBody)
	c.write("planning/features/nope", fm("nope", "feature", "rejected", "")+designedBody)
	c.write("planning/features/merged", fm("merged", "feature", "merged", "")+designedBody)
	c.write("planning/initiatives/fresh", fm("fresh", "initiative", "planning", "child:\n  - stub\n"))
	c.write("planning/initiatives/fresh/done", fm("done", "decision", "accepted", "parent: fresh\n")+"\n## Decision\n\nYes.\n")
	signed := "\n## Completion Ledger\n\n### Acceptance Criteria\n\n| # | C | Status | Note |\n|---|---|---|---|\n| 1 | AC-1 | DONE | ok |\n\n### Changes\n\n| # | I | Status | Note |\n|---|---|---|---|\n| 1 | x | SKIPPED | [signed-off] chet-bellows — out of scope |\n"
	sp := c.write("specs/signed", fm("signed", "feature", "completed", "completed_at: 2026-10-02T00:00:00Z\n")+designedBody+signed)
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(filepath.Join(filepath.Dir(sp), "delivery-audit.md"), []byte("# Delivery audit — signed\n\n**Verdict:** SHIP\n"), 0o644)

	specs, _ := c.build()
	items := Build(specs, Options{Now: testNow, Root: c.root})
	ApplyNext(items, specs)
	by := map[string]Item{}
	for _, it := range items {
		if _, dup := by[it.Slug]; dup {
			t.Errorf("slug %s emitted twice", it.Slug)
		}
		by[it.Slug] = it
	}
	if n := by["mailfeat"].Next; n == nil || n.Label != "Continue" || by["mailfeat"].Lane != LaneInProgress {
		t.Errorf("live feature shadowed by its intake: lane %s next %+v", by["mailfeat"].Lane, n)
	}
	if n := by["active"].Next; n == nil || n.Label != "Continue" {
		t.Errorf("work spec shadowed by a same-slug explainer: next %+v", n)
	}
	if by["watcher"].Lane != LaneDesigned {
		t.Errorf("dependency on the live feature must resolve to it (unfinished): lane %s", by["watcher"].Lane)
	}
	if v := by["signed"].Verify; v == nil || v.State != VerifyPassed || by["signed"].Next != nil {
		t.Errorf("signed-off skip should pass verify like hero spec verify: %+v next %+v", v, by["signed"].Next)
	}
	for slug, label := range map[string]string{"review": "Continue", "back": "Continue"} {
		if n := by[slug].Next; n == nil || n.Label != label {
			t.Errorf("%s next = %+v, want %s", slug, n, label)
		}
	}
	if n := by["waiting"].Next; n == nil || n.Enabled || n.Phase.Label != "With peer" {
		t.Errorf("awaiting_peer next = %+v", n)
	}
	for _, slug := range []string{"nope", "merged", "done"} {
		if by[slug].Next != nil {
			t.Errorf("%s should have no next step, got %+v", slug, by[slug].Next)
		}
	}
	if n := by["fresh"].Next; n == nil || n.Label != "Drive" || n.Phase.Label != "Planning" || n.Phase.State != PhaseReady {
		t.Errorf("unstarted initiative with children = %+v", n)
	}
	if n := by["diagnosed"].Next; n == nil {
		t.Fatal("diagnosed bug lost its next step")
	} else {
		for _, e := range n.Extras {
			if e.Action == ActionDiagnose {
				t.Errorf("diagnosed bug offers Deliver plus a diagnose-action extra: %+v", e)
			}
		}
	}
}
