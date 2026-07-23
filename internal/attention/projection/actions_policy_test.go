package projection

import (
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
)

func TestAdvertisedActionsCarryCanonicalInteractionPolicy(t *testing.T) {
	groups := [][]attention.ActionDescriptor{
		mailActions(3),
		focusActions(4, true),
	}
	for _, actions := range groups {
		for _, descriptor := range actions {
			policy, ok := attention.OperationPolicyByID(descriptor.OperationID)
			if !ok {
				t.Fatalf("%s: missing policy for operation %q", descriptor.ID, descriptor.OperationID)
			}
			if descriptor.Effect != string(policy.Effect) || descriptor.Consent != string(policy.Consent) {
				t.Fatalf("%s: descriptor policy = %q/%q, want %q/%q", descriptor.ID, descriptor.Effect, descriptor.Consent, policy.Effect, policy.Consent)
			}
			if descriptor.ID != policy.ActionID {
				t.Fatalf("%s: policy action = %q", descriptor.ID, policy.ActionID)
			}
		}
	}
}
