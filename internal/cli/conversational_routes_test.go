package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
)

func TestConversationalRouteCorpusUsesRealCLICommands(t *testing.T) {
	root := findProjectRoot()
	data, err := os.ReadFile(filepath.Join(root, "contracts", "attention", "testdata", "v1", "conversational-routes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture attention.ConversationalRouteFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if contractErr := attention.ValidateConversationalRouteFixture(fixture); contractErr != nil {
		t.Fatalf("fixture validation: %v", contractErr)
	}

	checked := 0
	for _, routeCase := range fixture.Cases {
		if routeCase.ExpectedSurface != attention.DispatchSurfaceCLIWorkflow {
			continue
		}
		invocations := ExtractInvocations(
			"contracts/attention/testdata/v1/conversational-routes.json",
			[]byte("`"+routeCase.ExpectedCommand+"`"),
		)
		if len(invocations) != 1 {
			t.Errorf("%s: extracted %d CLI invocations from %q", routeCase.ID, len(invocations), routeCase.ExpectedCommand)
			continue
		}
		checked++
		if err := ValidateInvocation(rootCmd, invocations[0]); err != nil {
			t.Errorf("%s: %v", routeCase.ID, err)
		}
	}
	if checked != 3 {
		t.Fatalf("checked %d typed peering CLI routes; want 3", checked)
	}
}
