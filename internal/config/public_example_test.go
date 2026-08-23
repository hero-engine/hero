package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublicHeroConfigFixtureLoadsThroughProductionDecoder(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "public-hero.json"))
	if err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	heroDir := filepath.Join(root, DefaultFolder)
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), fixture, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("public config fixture must load through config.Load: %v", err)
	}
	if cfg.Team == nil {
		t.Fatal("public config fixture did not decode team settings")
	}
	if cfg.Team.NudgeLevel != "assertive" {
		t.Fatalf("team.nudge_level = %q, want assertive", cfg.Team.NudgeLevel)
	}
	if cfg.Integrations == nil || cfg.Integrations.Default != "jira-delivery" {
		t.Fatal("public config fixture did not retain the default integration")
	}
	if cfg.Serve == nil || cfg.Serve.ToolFilter == nil || len(cfg.Serve.ToolFilter.Profiles["minimal"]) != 2 {
		t.Fatal("public config fixture did not retain the minimal MCP profile")
	}
	if cfg.Verify == nil || cfg.Verify.TestCommand != "go test ./..." {
		t.Fatal("public config fixture did not retain the verify test command")
	}
}
