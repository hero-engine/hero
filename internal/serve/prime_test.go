package serve

import (
	"strings"
	"testing"
)

func TestBuildPrimeContext_Basic(t *testing.T) {
	heroDir, projectRoot := setupTestWorkspace(t)

	ctx, err := BuildPrimeContext(heroDir, projectRoot, false)
	if err != nil {
		t.Fatalf("BuildPrimeContext: %v", err)
	}

	if !strings.Contains(ctx, "Hero project:") {
		t.Errorf("expected 'Hero project:' header, got: %s", ctx)
	}
	if !strings.Contains(ctx, "Active Work") {
		t.Errorf("expected 'Active Work' section, got: %s", ctx)
	}
	if !strings.Contains(ctx, "auth-login") {
		t.Errorf("expected spec slug 'auth-login', got: %s", ctx)
	}
}

func TestBuildPrimeContext_WithKnowledge(t *testing.T) {
	heroDir, projectRoot := setupTestWorkspace(t)

	ctx, err := BuildPrimeContext(heroDir, projectRoot, true)
	if err != nil {
		t.Fatalf("BuildPrimeContext: %v", err)
	}

	if !strings.Contains(ctx, "Conventions") {
		t.Errorf("expected 'Conventions' section when includeKnowledge=true, got: %s", ctx)
	}
}

func TestBuildPrimeContext_WithoutKnowledge(t *testing.T) {
	heroDir, projectRoot := setupTestWorkspace(t)

	ctx, err := BuildPrimeContext(heroDir, projectRoot, false)
	if err != nil {
		t.Fatalf("BuildPrimeContext: %v", err)
	}

	if strings.Contains(ctx, "Conventions") {
		t.Errorf("expected no 'Conventions' section when includeKnowledge=false, got: %s", ctx)
	}
}

func TestBuildPrimeContext_InvalidDir(t *testing.T) {
	_, err := BuildPrimeContext("/nonexistent/path/.hero", "/nonexistent/path", true)
	if err == nil {
		t.Fatal("expected error for invalid heroDir, got nil")
	}
}
