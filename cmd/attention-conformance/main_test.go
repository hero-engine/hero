package main

import (
	"slices"
	"testing"
)

func TestValidateReleaseStatusRejectsDirtyContractOutput(t *testing.T) {
	if err := validateReleaseStatus(""); err != nil {
		t.Fatalf("clean status: %v", err)
	}
	if err := validateReleaseStatus(" M contracts/attention/conformance/v1/manifest.json\n"); err == nil {
		t.Fatal("expected dirty contract output to block readiness")
	}
}

func TestReleaseReadinessCoversEveryPublicationInput(t *testing.T) {
	for _, path := range []string{
		"cmd/attention-conformance",
		"internal/attention/conformance",
		"internal/serve/mcp_tools_def.go",
		"internal/serve/mcp_dispatch.go",
		"internal/serve/mcp_tools_attention_contract.go",
		"internal/serve/api_attention.go",
	} {
		if !slices.Contains(releasePaths, path) {
			t.Errorf("release readiness does not cover %s", path)
		}
	}
}
