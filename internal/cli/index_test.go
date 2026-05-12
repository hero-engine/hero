package cli

import (
	"strings"
	"testing"
)

func TestIndexRebuild(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-one/spec.md", `---
title: Feature One
type: feature
status: planning
---
# Feature One

## Changes
- Update `+"`src/main.go`"+`
`)

	env.addSpec("conventions/error-handling/spec.md", `---
title: Error Handling
type: convention
status: active
scope: [*.go]
---
# Error Handling

## Rule
Always wrap errors.
`)

	output, err := runCmd("index")
	if err != nil {
		t.Fatalf("index returned error: %v", err)
	}

	if !strings.Contains(output, "Indexed 2 specs") {
		t.Errorf("index output should mention 2 specs: %q", output)
	}

	if !strings.Contains(output, "1 features") {
		t.Errorf("index output should mention 1 feature: %q", output)
	}

	if !strings.Contains(output, "1 conventions") {
		t.Errorf("index output should mention 1 convention: %q", output)
	}
}

func TestIndexNoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("index")
	if err == nil {
		t.Fatal("index should error without workspace")
	}
}

func TestIndexEmpty(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("index")
	if err != nil {
		t.Fatalf("index returned error: %v", err)
	}

	if !strings.Contains(output, "Indexed 0 specs") {
		t.Errorf("index output should mention 0 specs: %q", output)
	}
}
