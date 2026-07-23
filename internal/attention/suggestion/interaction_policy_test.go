package suggestion

import (
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
)

func TestPresentedSuggestionActionsCarryCanonicalInteractionPolicy(t *testing.T) {
	presented := present(Item{State: StatePending, Revision: 7})
	if len(presented.Actions) != 4 {
		t.Fatalf("action count = %d", len(presented.Actions))
	}
	for _, descriptor := range presented.Actions {
		policy, ok := attention.OperationPolicyByID(descriptor.OperationID)
		if !ok {
			t.Fatalf("%s: missing policy for operation %q", descriptor.ID, descriptor.OperationID)
		}
		if descriptor.ID != policy.ActionID || descriptor.Effect != string(policy.Effect) || descriptor.Consent != string(policy.Consent) {
			t.Fatalf("%s: descriptor does not match policy %#v", descriptor.ID, policy)
		}
	}
}
