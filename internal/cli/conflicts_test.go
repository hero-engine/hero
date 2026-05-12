package cli

import (
	"strings"
	"testing"
)

func TestConflictsNone(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-a/spec.md", `---
title: Feature A
type: feature
status: delivering
---
# Feature A

## Changes
- Update `+"`src/alpha.go`"+`
`)

	env.addSpec("planning/features/feat-b/spec.md", `---
title: Feature B
type: feature
status: delivering
---
# Feature B

## Changes
- Update `+"`src/beta.go`"+`
`)

	env.indexAll()

	output, err := runCmd("check", "conflicts", "feat-a")
	if err != nil {
		t.Fatalf("conflicts returned error: %v", err)
	}

	if !strings.Contains(output, "No conflicts found") {
		t.Errorf("should show no conflicts: %q", output)
	}
}

func TestConflictsFound(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-c/spec.md", `---
title: Feature C
type: feature
status: delivering
---
# Feature C

## Changes
- Update `+"`src/shared.go`"+`
- Update `+"`src/alpha.go`"+`
`)

	env.addSpec("planning/features/feat-d/spec.md", `---
title: Feature D
type: feature
status: planning
---
# Feature D

## Changes
- Update `+"`src/shared.go`"+`
- Update `+"`src/beta.go`"+`
`)

	env.indexAll()

	output, err := runCmd("check", "conflicts", "feat-c")
	if err != nil {
		t.Fatalf("conflicts returned error: %v", err)
	}

	if !strings.Contains(output, "feat-d") {
		t.Errorf("should show feat-d as conflict: %q", output)
	}

	if !strings.Contains(output, "src/shared.go") {
		t.Errorf("should mention overlapping file: %q", output)
	}

	if !strings.Contains(output, "1 conflict") {
		t.Errorf("should show 1 conflict: %q", output)
	}
}

func TestConflictsRequiresArg(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("check", "conflicts")
	if err == nil {
		t.Fatal("conflicts without slug should fail")
	}
}

func TestConflictsNoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("check", "conflicts", "some-spec")
	if err == nil {
		t.Fatal("conflicts should error without workspace")
	}
}
