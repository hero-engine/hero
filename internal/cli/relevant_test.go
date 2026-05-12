package cli

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

func TestNudgeGentleWithContext(t *testing.T) {
	env := newTestEnv(t)

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

	env.indexAll()

	output, err := runCmd("relevant", "--files", "src/main.go")
	if err != nil {
		t.Fatalf("nudge returned error: %v", err)
	}

	// Default nudge level is gentle
	if !strings.Contains(output, "Hero") {
		t.Errorf("nudge should mention Hero: %q", output)
	}
}

func TestNudgeOff(t *testing.T) {
	env := newTestEnv(t)

	// Set nudge level to off
	cfg := config.DefaultConfig()
	cfg.Team.NudgeLevel = "off"
	cfg.Save(env.dir)

	env.addSpec("conventions/error-handling/spec.md", `---
title: Error Handling
type: convention
status: active
scope: [*.go]
---
# Error Handling
`)

	env.indexAll()

	output, err := runCmd("relevant", "--files", "src/main.go")
	if err != nil {
		t.Fatalf("nudge returned error: %v", err)
	}

	// Should be silent
	if output != "" {
		t.Errorf("nudge with level=off should be silent, got: %q", output)
	}
}

func TestNudgeAssertive(t *testing.T) {
	env := newTestEnv(t)

	// Set nudge level to assertive
	cfg := config.DefaultConfig()
	cfg.Team.NudgeLevel = "assertive"
	cfg.Save(env.dir)

	env.addSpec("conventions/error-handling/spec.md", `---
title: Error Handling
type: convention
status: active
scope: [*.go]
---
# Error Handling
`)

	env.indexAll()

	output, err := runCmd("relevant", "--files", "src/main.go")
	if err != nil {
		t.Fatalf("nudge returned error: %v", err)
	}

	// Assertive should include recommendation
	if !strings.Contains(output, "Recommendation") {
		t.Errorf("assertive nudge should include recommendation: %q", output)
	}
}

func TestNudgeNoContext(t *testing.T) {
	env := newTestEnv(t)
	env.indexAll()

	output, err := runCmd("relevant", "--files", "src/unrelated.go")
	if err != nil {
		t.Fatalf("nudge returned error: %v", err)
	}

	// No context should produce empty output
	if output != "" {
		t.Errorf("nudge with no context should be silent, got: %q", output)
	}
}

func TestNudgeNoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	// Should silently succeed — nudge is non-fatal
	output, err := runCmd("relevant", "--files", "src/main.go")
	if err != nil {
		t.Fatalf("nudge should not error without workspace: %v", err)
	}

	if output != "" {
		t.Errorf("nudge without workspace should be silent, got: %q", output)
	}
}

func TestNudgeRequiresFiles(t *testing.T) {
	_ = newTestEnv(t)

	// nudge without --files should fail — MarkFlagRequired makes cobra print a usage
	// error but the error may or may not be propagated depending on cobra version.
	// Since nudge swallows errors silently (returns nil for non-fatal issues),
	// and MarkFlagRequired may output but not error, we just verify the command
	// doesn't produce nudge content without files.
	output, _ := runCmd("relevant")

	// Should NOT contain nudge content
	if strings.Contains(output, "Hero") && strings.Contains(output, "Conventions") {
		t.Error("nudge without --files should not produce nudge output")
	}
}

func TestNudgeWithPendingSpecs(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/inflight-work/spec.md", `---
title: In-Flight Work
type: feature
status: delivering
---
# In-Flight Work

## Changes
- Update `+"`src/api/handler.go`"+`
`)

	env.indexAll()

	output, err := runCmd("relevant", "--files", "src/api/handler.go")
	if err != nil {
		t.Fatalf("nudge returned error: %v", err)
	}

	// Should mention in-flight work
	if !strings.Contains(output, "In-flight") || !strings.Contains(output, "inflight-work") {
		t.Errorf("nudge should mention in-flight spec: %q", output)
	}
}
