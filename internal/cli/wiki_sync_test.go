package cli

import (
	"strings"
	"testing"
)

func TestWikiSync_NoWorkspace(t *testing.T) {
	env := newTestEnvEmpty(t)
	_ = env

	_, err := runCmd("publish", "wiki", "some-spec.md")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
}

func TestWikiSync_NoSyncConfig(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("publish", "wiki", "some-spec.md")
	if err == nil {
		t.Fatal("expected error for no sync config")
	}
	if !strings.Contains(err.Error(), "no wiki sync target") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWikiSync_NoArgsNoAll(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	// Even with default config (sync.target: "none"), should fail at sync config check first
	_, err := runCmd("publish", "wiki")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWikiSync_AllNoCompleted(t *testing.T) {
	env := newTestEnv(t)

	// Add a planning spec (not completed)
	env.addSpec("planning/features/test-feat/spec.md", `---
title: Test Feature
type: feature
status: planning
---
# Test Feature
`)

	// wiki-sync --all still fails because sync.target is "none" in default config
	_, err := runCmd("publish", "wiki", "--all")
	if err == nil {
		t.Fatal("expected error for no sync config")
	}
}
