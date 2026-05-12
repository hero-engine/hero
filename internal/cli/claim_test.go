package cli

import (
	"strings"
	"testing"
)

func TestClaim(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-claim/spec.md", `---
title: Feature Claim
type: feature
status: planning
---
# Feature Claim
`)

	env.indexAll()

	output, err := runCmd("spec", "claim", "feat-claim", "--agent", "alice")
	if err != nil {
		t.Fatalf("claim returned error: %v", err)
	}

	if !strings.Contains(output, "Claimed feat-claim for alice") {
		t.Errorf("claim output unexpected: %q", output)
	}
}

func TestClaimAlreadyClaimed(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-taken/spec.md", `---
title: Feature Taken
type: feature
status: planning
---
# Feature Taken
`)

	env.indexAll()

	// First claim succeeds
	_, err := runCmd("spec", "claim", "feat-taken", "--agent", "alice")
	if err != nil {
		t.Fatalf("first claim returned error: %v", err)
	}

	// Second claim by different user should fail
	_, err = runCmd("spec", "claim", "feat-taken", "--agent", "bob")
	if err == nil {
		t.Fatal("claiming already-claimed spec should fail")
	}

	if !strings.Contains(err.Error(), "already claimed") {
		t.Errorf("error should mention 'already claimed': %v", err)
	}
}

func TestClaimNonexistent(t *testing.T) {
	env := newTestEnv(t)
	env.indexAll()

	_, err := runCmd("spec", "claim", "nonexistent-spec", "--agent", "alice")
	if err == nil {
		t.Fatal("claiming nonexistent spec should fail")
	}
}

func TestUnclaim(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-unclaim/spec.md", `---
title: Feature Unclaim
type: feature
status: planning
---
# Feature Unclaim
`)

	env.indexAll()

	// Claim first
	_, err := runCmd("spec", "claim", "feat-unclaim", "--agent", "alice")
	if err != nil {
		t.Fatalf("claim returned error: %v", err)
	}

	// Unclaim
	output, err := runCmd("spec", "unclaim", "feat-unclaim")
	if err != nil {
		t.Fatalf("unclaim returned error: %v", err)
	}

	if !strings.Contains(output, "Released claim on feat-unclaim") {
		t.Errorf("unclaim output unexpected: %q", output)
	}

	// Should be claimable again
	_, err = runCmd("spec", "claim", "feat-unclaim", "--agent", "bob")
	if err != nil {
		t.Fatalf("re-claim after unclaim returned error: %v", err)
	}
}

func TestClaimRequiresArgs(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("spec", "claim")
	if err == nil {
		t.Fatal("claim without args should fail")
	}
}

func TestUnclaimRequiresArgs(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("spec", "unclaim")
	if err == nil {
		t.Fatal("unclaim without args should fail")
	}
}

func TestClaimNoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("spec", "claim", "some-spec", "--agent", "alice")
	if err == nil {
		t.Fatal("claim should error without workspace")
	}
}
