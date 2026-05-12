package cli

import (
	"strings"
	"testing"
)

func TestGraphWithRelations(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/initiatives/q3-auth/spec.md", `---
title: Q3 Auth Overhaul
type: initiative
status: planning
---
# Q3 Auth Overhaul
`)

	env.addSpec("planning/features/add-mfa/spec.md", `---
title: Add MFA
type: feature
status: planning
parent: q3-auth
depends-on: session-mgmt
---
# Add MFA
`)

	env.addSpec("planning/features/session-mgmt/spec.md", `---
title: Session Management
type: feature
status: delivering
---
# Session Management
`)

	env.indexAll()

	output, err := runCmd("graph", "add-mfa")
	if err != nil {
		t.Fatalf("graph returned error: %v", err)
	}

	if !strings.Contains(output, "Relationships for add-mfa") {
		t.Errorf("graph should show header: %q", output)
	}

	// Should show parent relation
	if !strings.Contains(output, "parent") {
		t.Errorf("graph should show parent relation: %q", output)
	}

	// Should show depends-on relation
	if !strings.Contains(output, "depends-on") {
		t.Errorf("graph should show depends-on relation: %q", output)
	}
}

func TestGraphNoRelations(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/standalone/spec.md", `---
title: Standalone
type: feature
status: planning
---
# Standalone
`)

	env.indexAll()

	output, err := runCmd("graph", "standalone")
	if err != nil {
		t.Fatalf("graph returned error: %v", err)
	}

	if !strings.Contains(output, "No relationships found") {
		t.Errorf("graph should show no relationships: %q", output)
	}
}

// TestGraphNoArgRunsStats covers the "be useful, not strict" design
// choice in runGraph: bare `hero graph` falls through to `graph stats`
// rather than erroring on missing slug. See the comment in graph.go.
func TestGraphNoArgRunsStats(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("graph")
	if err != nil {
		t.Fatalf("bare 'hero graph' should fall through to graph stats, got error: %v", err)
	}
}

func TestGraphNoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("graph", "some-spec")
	if err == nil {
		t.Fatal("graph should error without workspace")
	}
}

func TestGraphMermaid(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/initiatives/q3-auth/spec.md", `---
title: Q3 Auth Overhaul
type: initiative
status: planning
---
# Q3 Auth Overhaul
`)

	env.addSpec("planning/features/add-mfa/spec.md", `---
title: Add MFA
type: feature
status: planning
parent: q3-auth
depends-on: session-mgmt
---
# Add MFA
`)

	env.addSpec("planning/features/session-mgmt/spec.md", `---
title: Session Management
type: feature
status: delivering
---
# Session Management
`)

	env.indexAll()

	output, err := runCmd("graph", "add-mfa", "--format", "mermaid")
	if err != nil {
		t.Fatalf("graph mermaid returned error: %v", err)
	}

	// Should contain mermaid markers
	if !strings.Contains(output, "```mermaid") {
		t.Errorf("should contain mermaid code fence, got: %q", output)
	}
	if !strings.Contains(output, "graph TD") {
		t.Errorf("should contain graph TD, got: %q", output)
	}

	// Should contain the center node
	if !strings.Contains(output, "add_mfa") {
		t.Errorf("should contain center node add_mfa, got: %q", output)
	}

	// Should contain edge labels
	if !strings.Contains(output, "parent") {
		t.Errorf("should contain parent edge, got: %q", output)
	}
	if !strings.Contains(output, "depends-on") {
		t.Errorf("should contain depends-on edge, got: %q", output)
	}

	// Should contain style directive
	if !strings.Contains(output, "style add_mfa") {
		t.Errorf("should contain style directive, got: %q", output)
	}
}

func TestGraphMermaidNoRelations(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/alone/spec.md", `---
title: Alone
type: feature
status: planning
---
# Alone
`)

	env.indexAll()

	output, err := runCmd("graph", "alone", "--format", "mermaid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should show no relationships message, not mermaid output
	if !strings.Contains(output, "No relationships found") {
		t.Errorf("should show no relationships, got: %q", output)
	}
	if strings.Contains(output, "```mermaid") {
		t.Error("should not show mermaid when no relationships")
	}
}

func TestGraphInvalidFormat(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/test-fmt/spec.md", `---
title: Test
type: feature
status: planning
parent: other
---
# Test
`)

	env.addSpec("planning/features/other/spec.md", `---
title: Other
type: feature
status: planning
---
# Other
`)

	env.indexAll()

	_, err := runCmd("graph", "test-fmt", "--format", "xml")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("unexpected error: %v", err)
	}
}
