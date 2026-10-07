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
	c.write("planning/initiatives/init/done", fm("done", "decision", "accepted", "parent: init\n")+"\n## Decision\n\nYes.\n")
	signed := "\n## Completion Ledger\n\n### Acceptance Criteria\n\n| # | C | Status | Note |\n|---|---|---|---|\n| 1 | AC-1 | DONE | ok |\n\n### Changes\n\n| # | I | Status | Note |\n|---|---|---|---|\n| 1 | x | SKIPPED | [signed-off] chet-bellows — out of scope |\n"
	sp := c.write("specs/signed", fm("signed", "feature", "completed", "completed_at: 2026-10-02T00:00:00Z\n")+designedBody+signed)
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(filepath.Join(filepath.Dir(sp), "delivery-audit.md"), []byte("# Delivery audit — signed\n\n**Verdict:** SHIP\n"), 0o644)

	specs, _ := c.build()
	items := Build(specs, Options{Now: testNow, Root: c.root, Signers: map[string]bool{"chet-bellows": true}})
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

// Audit round 3: sign-offs resolve against Gate 1's known signers (a
// free-text "signer" never verifies), reverse parent edges make an
// initiative's children, and two previously untested verify paths.
func TestRound3AuditCases(t *testing.T) {
	c := newCorpus(t)
	seed(c)
	ledger := func(note string) string {
		return "\n## Completion Ledger\n\n### Acceptance Criteria\n\n| # | C | Status | Note |\n|---|---|---|---|\n| 1 | AC-1 | DONE | ok |\n\n### Changes\n\n| # | I | Status | Note |\n|---|---|---|---|\n| 1 | x | SKIPPED | " + note + " |\n"
	}
	shipped := func(slug, body string) {
		p := c.write("specs/"+slug, fm(slug, "feature", "completed", "completed_at: 2026-10-02T00:00:00Z\n")+designedBody+body)
		time.Sleep(10 * time.Millisecond)
		os.WriteFile(filepath.Join(filepath.Dir(p), "delivery-audit.md"), []byte("# Delivery audit — "+slug+"\n\n**Verdict:** SHIP\n"), 0o644)
	}
	shipped("known", ledger("[signed-off] chet-bellows — out of scope"))
	shipped("freetext", ledger("[signed-off] the team agreed — later"))
	shipped("undone", ledger("not done yet"))
	stale := c.write("planning/features/stale", fm("stale", "feature", "delivering", "")+designedBody)
	os.WriteFile(filepath.Join(filepath.Dir(stale), "delivery-audit.md"), []byte("# Delivery audit — stale\n\n**Verdict:** SHIP\n"), 0o644)
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(stale, []byte(fm("stale", "feature", "delivering", "")+designedBody+"\nEdited after the audit.\n"), 0o644)
	c.write("planning/initiatives/umbrella", fm("umbrella", "initiative", "planning", ""))
	c.write("planning/features/kid1", fm("kid1", "feature", "completed", "parent: umbrella\ncompleted_at: 2026-06-01T00:00:00Z\n")+designedBody)
	c.write("planning/features/kid2", fm("kid2", "feature", "planning", "parent: umbrella\n")+designedBody)
	c.write("knowledge/notes/aside", fm("aside", "note", "active", "parent: umbrella\n"))

	specs, _ := c.build()
	items := Build(specs, Options{Now: testNow, Root: c.root, Signers: map[string]bool{"chet-bellows": true}})
	ApplyNext(items, specs)
	by := map[string]Item{}
	for _, it := range items {
		by[it.Slug] = it
	}
	for slug, want := range map[string]string{"known": VerifyPassed, "freetext": VerifyPartial, "undone": VerifyPartial, "stale": VerifyNotRun} {
		if v := by[slug].Verify; v == nil || v.State != want {
			t.Errorf("%s verify = %+v, want %s", slug, v, want)
		}
	}
	if by["stale"].Verify.Audit != nil {
		t.Errorf("an unfinished spec newer than its audit must not report it: %v", *by["stale"].Verify.Audit)
	}
	if n := by["freetext"].Next; n == nil || n.Action != ActionVerify {
		t.Errorf("free-text signer must leave Verify offered, got %+v", n)
	}
	// No signer set at all fails closed.
	bare := Build(specs, Options{Now: testNow, Root: c.root})
	for _, it := range bare {
		if it.Slug == "known" && it.Verify.State != VerifyPartial {
			t.Errorf("nil Signers must not accept sign-offs, got %s", it.Verify.State)
		}
	}
	u := by["umbrella"]
	if u.Progress == nil || u.Progress.Total != 2 || u.Progress.Done != 1 || u.Lane != LaneInProgress || u.Next == nil || u.Next.Label != "Drive" {
		t.Errorf("reverse-parent children not counted: lane %s progress %+v next %+v", u.Lane, u.Progress, u.Next)
	}
}

// followup-epic-parity: an epic is modelled exactly like an initiative, and
// containers (initiative/epic) carry verify null, so a recently finished one
// never yields an action-less weak_verify polish entry.
func TestEpicParityAndContainerVerify(t *testing.T) {
	c := newCorpus(t)
	seed(c)
	c.write("planning/epics/big", fm("big", "epic", "planning", "child:\n  - stub\n"))
	c.write("planning/features/epickid", fm("epickid", "feature", "completed", "parent: big\ncompleted_at: 2026-06-01T00:00:00Z\n")+designedBody)
	c.write("planning/epics/big/pick", fm("pick", "decision", "proposed", "parent: big\n")+"\n## Decision\n\nYes.\n")
	c.write("planning/epics/empty", fm("empty-epic", "epic", "planning", ""))
	c.write("specs/doneinit", fm("doneinit", "initiative", "completed", "completed_at: 2026-10-01T00:00:00Z\nchild:\n  - recent\n"))

	specs, _ := c.build()
	items := Build(specs, Options{Now: testNow, Root: c.root})
	ApplyNext(items, specs)
	by := map[string]Item{}
	for _, it := range items {
		by[it.Slug] = it
	}
	big := by["big"]
	if big.Progress == nil || big.Progress.Total != 3 || big.Progress.Done != 1 || big.Lane != LaneInProgress || big.Next == nil || big.Next.Label != "Drive" {
		t.Errorf("epic not modelled like an initiative: lane %s progress %+v next %+v", big.Lane, big.Progress, big.Next)
	}
	if _, ok := by["pick"]; !ok {
		t.Error("decision under an epic must be a work item")
	}
	if n := by["empty-epic"].Next; n == nil || n.Label != "Compose" || by["empty-epic"].Lane != LaneNone {
		t.Errorf("childless epic = lane %s next %+v", by["empty-epic"].Lane, n)
	}
	for _, slug := range []string{"big", "init", "doneinit"} {
		if by[slug].Verify != nil {
			t.Errorf("%s is a container: verify must be null, got %+v", slug, by[slug].Verify)
		}
	}
	for _, p := range Polish(items, specs) {
		if p.Next == nil {
			t.Errorf("polish entry %s/%s has no action", p.Slug, p.Kind)
		}
	}
}
