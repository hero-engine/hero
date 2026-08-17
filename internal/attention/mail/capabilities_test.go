package mail

import (
	"reflect"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
)

func TestCapabilitiesCarryCanonicalPolicyAndSourceMapping(t *testing.T) {
	capabilities := Capabilities(17)
	wantIDs := []string{"mark_read", "acknowledge", "dismiss", "promote", "add_to_today", "reply"}
	gotIDs := make([]string, len(capabilities))
	for i, descriptor := range capabilities {
		gotIDs[i] = descriptor.ID
		policy, ok := attention.OperationPolicyByID(descriptor.OperationID)
		if !ok {
			t.Fatalf("%s: unknown operation %q", descriptor.ID, descriptor.OperationID)
		}
		if descriptor.Effect != string(policy.Effect) || descriptor.Consent != string(policy.Consent) {
			t.Fatalf("%s policy = %q/%q, want %q/%q", descriptor.ID, descriptor.Effect, descriptor.Consent, policy.Effect, policy.Consent)
		}
		if descriptor.RequiredRowRevision != 17 || !descriptor.RequiresIdempotency || len(descriptor.InputSchema) == 0 {
			t.Fatalf("incomplete descriptor: %#v", descriptor)
		}
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("capability IDs = %v, want %v", gotIDs, wantIDs)
	}

	wantSource := map[string]string{
		"mark_read": ActionRead, "acknowledge": ActionAcknowledge, "dismiss": ActionDismiss,
		"promote": ActionPromote, "add_to_today": ActionAddToToday,
	}
	for canonical, want := range wantSource {
		got, ok := SourceActionID(canonical)
		if !ok || got != want {
			t.Fatalf("SourceActionID(%q) = %q, %v; want %q, true", canonical, got, ok, want)
		}
	}
	for _, inert := range []string{"reply", "read", "future_action"} {
		if got, ok := SourceActionID(inert); ok {
			t.Fatalf("SourceActionID(%q) = %q, true; want inert", inert, got)
		}
	}
}

func TestRowCapabilitiesPreserveProjectionActionSet(t *testing.T) {
	capabilities := RowCapabilities(3)
	if len(capabilities) != 5 || capabilities[0].ID != "mark_read" || capabilities[len(capabilities)-1].ID != "add_to_today" {
		t.Fatalf("row capabilities = %#v", capabilities)
	}
}
