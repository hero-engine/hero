package workmodel

import (
	"strings"

	"github.com/hero-engine/hero/internal/spec"
)

// Next-step actions (read-contract-v1).
const (
	ActionDesign   = "design"
	ActionDeliver  = "deliver"
	ActionDrive    = "drive"
	ActionDiagnose = "diagnose"
	ActionVerify   = "verify"
	ActionReview   = "review"
	ActionPolish   = "polish"
	ActionDiscover = "discover"
	// ActionChallenge is an extra on a diagnosed bug. It is deliberately not
	// "diagnose": a step never offers Diagnose and Deliver together.
	ActionChallenge = "challenge"
)

// Phase states.
const (
	PhaseWaiting   = "waiting"
	PhaseReady     = "ready"
	PhaseActive    = "active"
	PhaseAttention = "attention"
	PhaseDone      = "done"
)

// ApplyNext fills every item's Next from read-contract-v1's table.
func ApplyNext(items []Item, specs []*spec.Spec) {
	c := NewCorpus(specs)
	for i := range items {
		if s := c.Lookup(items[i].Slug); s != nil {
			items[i].Next = NextFor(items[i], s, c)
		}
	}
}

// NextFor returns the item's single primary action, or nil when there is
// nothing left to do. Rules kept from the requester: exactly one primary
// action (never Diagnose and Deliver together); "Delivered" only when
// verify passed; a finished item with weak or missing verify offers Verify,
// never Deliver.
func NextFor(it Item, s *spec.Spec, c *Corpus) *NextStep {
	slug := it.Slug
	typ := string(s.Type)

	switch s.Status {
	case spec.StatusSuperseded, spec.StatusRejected, spec.StatusMerged:
		return nil
	}

	if s.IsFinished() {
		if typ == "decision" || IsContainer(s) {
			return nil
		}
		if it.Verify != nil && it.Verify.State == VerifyPassed {
			return nil
		}
		// Only recent deliveries are worth re-verifying; work finished
		// before the window (often before audits existed) has no action.
		if it.Lane != LaneRecentlyDone {
			return nil
		}
		return step(ActionVerify, "Verify", "/verify "+slug, "Delivered?", PhaseAttention, nil, nil)
	}

	switch s.Status {
	case spec.StatusRegressed:
		return step(ActionDiagnose, "Diagnose", "/diagnose "+slug, "Regressed", PhaseAttention, nil, nil)
	case spec.StatusHandedOff, spec.StatusAwaitingPeer:
		return disabled(step(ActionDeliver, "Deliver", "/deliver "+slug, "With peer", PhaseWaiting, nil, nil), "handed off to a peer")
	case spec.StatusDelivering, spec.StatusInReview, spec.StatusHandedBack:
		state := PhaseActive
		if it.Verify != nil && it.Verify.Audit != nil && *it.Verify.Audit == "hold" {
			state = PhaseAttention
		}
		return step(ActionDeliver, "Continue", "/deliver "+slug, "Delivering", state, nil, []Extra{review(slug)})
	}

	if IsContainer(s) {
		if len(c.Children(s)) == 0 {
			return step(ActionDesign, "Compose", "/compose "+slug, "Planning", PhaseReady, nil, nil)
		}
		label, state := "Planning", PhaseReady
		if it.Lane == LaneInProgress {
			label, state = "Driving", PhaseActive
		}
		return step(ActionDrive, "Drive", "/drive "+slug, label, state, nil, nil)
	}

	switch typ {
	case "decision":
		return step(ActionDesign, "Decide", "/decide "+slug, "Proposed", PhaseReady, nil, nil)
	case "bug":
		if !Designed(s, c) {
			return step(ActionDiagnose, "Diagnose", "/diagnose "+slug, "Reported", PhaseReady, nil, nil)
		}
		extras := []Extra{{Action: ActionChallenge, Label: "Challenge diagnosis", Command: "/challenge " + slug}}
		return gateOnDeps(step(ActionDeliver, "Fix", "/deliver "+slug, "Diagnosed", PhaseReady, nil, extras), s, c)
	default: // feature, enhancement
		if !Designed(s, c) {
			return step(ActionDesign, "Design", "/design "+slug, "Planning", PhaseReady, nil,
				[]Extra{{Action: ActionDesign, Label: "Split", Command: "/split " + slug}})
		}
		label := "Ready"
		next := step(ActionDeliver, "Deliver", "/deliver "+slug, label, PhaseReady, nil, []Extra{review(slug)})
		if len(UnmetDeps(s, c)) > 0 {
			next.Phase.Label = "Planning"
		}
		return gateOnDeps(next, s, c)
	}
}

// gateOnDeps disables a deliver step while depends-on/blocks targets are
// unfinished, naming them.
func gateOnDeps(next *NextStep, s *spec.Spec, c *Corpus) *NextStep {
	if deps := UnmetDeps(s, c); len(deps) > 0 {
		next.Phase.State = PhaseWaiting
		return disabled(next, "waits on "+strings.Join(deps, ", "))
	}
	return next
}

func review(slug string) Extra {
	return Extra{Action: ActionReview, Label: "Check against the code", Command: "/review " + slug}
}

func step(action, label, command, phaseLabel, phaseState string, reason *string, extras []Extra) *NextStep {
	if extras == nil {
		extras = []Extra{}
	}
	return &NextStep{
		Action:  action,
		Label:   label,
		Command: command,
		Phase:   Phase{Label: phaseLabel, State: phaseState},
		Enabled: reason == nil,
		Reason:  reason,
		Extras:  extras,
	}
}

func disabled(n *NextStep, reason string) *NextStep {
	n.Enabled = false
	n.Reason = &reason
	return n
}
