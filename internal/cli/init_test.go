package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

func TestInitCreatesDirectories(t *testing.T) {
	env := newTestEnvEmpty(t)

	output, err := runCmd("init")
	if err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	if !strings.Contains(output, "Initialized hero workspace") {
		t.Errorf("init output missing initialization message: %q", output)
	}

	// Verify all directories were created
	expectedDirs := []string{
		".hero",
		".hero/planning",
		".hero/planning/features",
		".hero/planning/bugs",
		".hero/planning/initiatives",
		".hero/specs",
		".hero/knowledge",
		".hero/knowledge/conventions",
		".hero/knowledge/decisions",
		".hero/knowledge/rules",
		".hero/knowledge/external",
		".hero/knowledge/context",
		".hero/knowledge/templates",
		".hero/knowledge/notes",
	}

	for _, d := range expectedDirs {
		full := filepath.Join(env.dir, d)
		info, err := os.Stat(full)
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", d)
		}
	}
}

func TestInitWritesHeroJSON(t *testing.T) {
	env := newTestEnvEmpty(t)

	_, err := runCmd("init")
	if err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	configPath := filepath.Join(env.dir, ".hero", "hero.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("hero.json not created: %v", err)
	}

	content := string(data)
	// Verify it has the expected default values
	if !strings.Contains(content, `"folder": ".hero"`) {
		t.Error("hero.json missing folder field")
	}
	if !strings.Contains(content, `"nudge_level": "gentle"`) {
		t.Error("hero.json missing nudge_level default")
	}
	if !strings.Contains(content, `"stale_days": 14`) {
		t.Error("hero.json missing stale_days default")
	}
}

// TestInitBornProjected pins AC-B (born-projected): a freshly
// init-created workspace has next.projected == true, so it never
// enters legacy mode and never hits the checkpoint migration gate.
// It also guards that config.DefaultConfig() itself stays unprojected
// — DefaultConfig is the fallback for repos with no hero.json, and
// flipping it there would retroactively migrate existing legacy repos,
// which is the auto-migrate path's job, not init's.
func TestInitBornProjected(t *testing.T) {
	env := newTestEnvEmpty(t)

	if _, err := runCmd("init"); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	cfg, err := config.Load(env.dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.NextProjected() {
		t.Error("freshly init-created workspace should be born next.projected == true")
	}

	// DefaultConfig (the no-hero.json fallback) must NOT be projected,
	// so existing legacy repos aren't retroactively flipped.
	if config.DefaultConfig().NextProjected() {
		t.Error("config.DefaultConfig() must stay next.projected == false to avoid retroactively flipping existing repos")
	}
}

func TestInitRefusesReinit(t *testing.T) {
	_ = newTestEnv(t) // already has .hero/

	_, err := runCmd("init")
	if err == nil {
		t.Fatal("init should return error when workspace already exists")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}
}

func TestInitCustomFolder(t *testing.T) {
	env := newTestEnvEmpty(t)

	output, err := runCmd("init", "--folder", ".custom")
	if err != nil {
		t.Fatalf("init --folder .custom returned error: %v", err)
	}

	if !strings.Contains(output, ".custom") {
		t.Errorf("init output should mention custom folder: %q", output)
	}

	customDir := filepath.Join(env.dir, ".custom")
	if _, err := os.Stat(customDir); err != nil {
		t.Errorf("custom folder not created: %v", err)
	}

	// hero.json should be in the custom folder
	configPath := filepath.Join(customDir, "hero.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("hero.json not in custom folder: %v", err)
	}
}

func TestInitDomainFlag(t *testing.T) {
	env := newTestEnvEmpty(t)

	if _, err := runCmd("init", "--domain", "engineering"); err != nil {
		t.Fatalf("init --domain engineering returned error: %v", err)
	}

	configPath := filepath.Join(env.dir, ".hero", "hero.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("hero.json not created: %v", err)
	}
	if !strings.Contains(string(data), `"domain": "engineering"`) {
		t.Errorf("hero.json missing domain field: %s", data)
	}
}

func TestInitDomainFlagUnknownDomain(t *testing.T) {
	env := newTestEnvEmpty(t)

	_, err := runCmd("init", "--domain", "not-a-real-domain")
	if err == nil {
		t.Fatal("init --domain not-a-real-domain should error")
	}
	if !strings.Contains(err.Error(), "not-a-real-domain") {
		t.Errorf("error should mention the unknown domain, got: %v", err)
	}

	// Workspace must not have been created when validation fails.
	if _, statErr := os.Stat(filepath.Join(env.dir, ".hero")); !os.IsNotExist(statErr) {
		t.Errorf(".hero/ should not exist after a failed init (stat err=%v)", statErr)
	}
}

func TestInitInstallsPreCommitHook(t *testing.T) {
	env := newTestEnvEmpty(t)
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	out, err := runCmd("init")
	if err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	if !strings.Contains(out, "Installed pre-commit hook") {
		t.Errorf("init output missing hook confirmation: %q", out)
	}

	hookPath := filepath.Join(env.dir, ".git", "hooks", "pre-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("pre-commit hook not created: %v", err)
	}
	if !strings.Contains(string(data), "hero next") {
		t.Errorf("hook content missing hero marker: %q", string(data))
	}
}

func TestInitNoHooksFlag(t *testing.T) {
	env := newTestEnvEmpty(t)
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	out, err := runCmd("init", "--no-hooks")
	if err != nil {
		t.Fatalf("init --no-hooks returned error: %v", err)
	}

	if strings.Contains(out, "Installed pre-commit hook") {
		t.Errorf("--no-hooks should suppress hook install message: %q", out)
	}

	hookPath := filepath.Join(env.dir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("pre-commit hook should not exist under --no-hooks (stat err=%v)", err)
	}
}

func TestInitNoGitNoHook(t *testing.T) {
	env := newTestEnvEmpty(t)
	// No `git init` here — init should succeed and not try to install a hook.

	if _, err := runCmd("init"); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	hookPath := filepath.Join(env.dir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("hook should not exist outside a git repo (stat err=%v)", err)
	}
}

func TestInitOutputListsDirectories(t *testing.T) {
	_ = newTestEnvEmpty(t)

	output, err := runCmd("init")
	if err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	expectedParts := []string{
		"planning/features/",
		"planning/bugs/",
		"planning/initiatives/",
		"specs/",
		"knowledge/",
		"conventions/",
		"decisions/",
		"rules/",
		"notes/",
		"hero.json",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("init output missing %q", part)
		}
	}
}
