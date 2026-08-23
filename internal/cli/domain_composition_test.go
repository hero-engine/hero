package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/domains"
	"github.com/hero-engine/hero/internal/install"
)

func installCompositionForCLI(t *testing.T, root string, targets ...install.Target) {
	t.Helper()
	content, _, err := hero.ComposeContent(hero.DomainComposition{Primary: "engineering"})
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		if _, err := install.Run(install.Options{ContentFS: content, Target: target, Mode: install.ModeProject, TargetDir: root, Force: true, Domain: "engineering"}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDomainLifecycleReinstallsAndPreservesWorkspaceData(t *testing.T) {
	env := newTestEnv(t)
	installCompositionForCLI(t, env.dir, install.TargetCodex)
	history := filepath.Join(env.heroDir, "specs", "historical-qa.md")
	if err := os.WriteFile(history, []byte("domain: qa\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := runDomainEnable(nil, []string{"pm"}); err != nil {
		t.Fatal(err)
	}
	if err := runDomainEnable(nil, []string{"qa"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(env.dir)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := cfg.ResolveDomains()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resolved.Extensions, []domains.DomainID{domains.DomainPM, domains.DomainQA}; !reflect.DeepEqual(got, want) {
		t.Fatalf("extensions = %v, want %v", got, want)
	}
	qaAgent := filepath.Join(env.dir, ".codex", "agents", "qa-delivery-lead.toml")
	if _, err := os.Stat(qaAgent); err != nil {
		t.Fatalf("QA extension was not installed: %v", err)
	}

	if err := runDomainDisable(nil, []string{"qa"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(qaAgent); !os.IsNotExist(err) {
		t.Fatalf("QA extension file survived disable: %v", err)
	}
	if data, err := os.ReadFile(history); err != nil || string(data) != "domain: qa\n" {
		t.Fatalf("historical QA artifact changed: %q, %v", data, err)
	}

	if err := runDomainSwitch(nil, []string{"qa"}); err != nil {
		t.Fatal(err)
	}
	cfg, err = config.Load(env.dir)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err = cfg.ResolveDomains()
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Primary != domains.DomainQA || !reflect.DeepEqual(resolved.Extensions, []domains.DomainID{domains.DomainPM}) {
		t.Fatalf("switched composition = %#v", resolved)
	}
}

func TestDomainLifecycleRollbackKeepsConfigAndRenderedFiles(t *testing.T) {
	env := newTestEnv(t)
	installCompositionForCLI(t, env.dir, install.TargetClaude, install.TargetCodex)
	configPath := filepath.Join(env.heroDir, config.ConfigFileName)
	beforeConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeAgent, err := os.ReadFile(filepath.Join(env.dir, ".codex", "agents", "feature-delivery-lead.toml"))
	if err != nil {
		t.Fatal(err)
	}

	originalRunner := domainInstallRunner
	t.Cleanup(func() { domainInstallRunner = originalRunner })
	calls := 0
	domainInstallRunner = func(opts install.Options) (*install.Result, error) {
		calls++
		if calls == 2 {
			return nil, fmt.Errorf("injected target failure")
		}
		return install.Run(opts)
	}
	if err := runDomainEnable(nil, []string{"qa"}); err == nil || !strings.Contains(err.Error(), "prior composition restored") {
		t.Fatalf("enable error = %v", err)
	}
	afterConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterConfig) != string(beforeConfig) {
		t.Fatal("configuration changed after failed lifecycle mutation")
	}
	afterAgent, err := os.ReadFile(filepath.Join(env.dir, ".codex", "agents", "feature-delivery-lead.toml"))
	if err != nil || string(afterAgent) != string(beforeAgent) {
		t.Fatalf("rendered state changed after rollback: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.dir, ".codex", "agents", "qa-delivery-lead.toml")); !os.IsNotExist(err) {
		t.Fatalf("QA file survived rollback: %v", err)
	}
}

func TestDomainShowAndListExposeCompositionAndRoles(t *testing.T) {
	env := newTestEnv(t)
	cfg, err := config.Load(env.dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetDomainComposition(domains.DomainEngineering, []domains.DomainID{domains.DomainPM, domains.DomainQA}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatal(err)
	}
	show := captureStdout(func() { err = runDomainShow(nil, nil) })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Primary domain: engineering", "Extensions: pm, qa", "ready (bundled, local)"} {
		if !strings.Contains(show, want) {
			t.Errorf("show missing %q: %s", want, show)
		}
	}
	list := captureStdout(func() { err = runDomainList(nil, nil) })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"engineering", "state=primary", "pm", "qa", "state=extension", "roles=primary,extension", "ready=true", "bundled=true"} {
		if !strings.Contains(list, want) {
			t.Errorf("list missing %q: %s", want, list)
		}
	}
}

func TestInitCompositionAndTargets(t *testing.T) {
	env := newTestEnvEmpty(t)
	if _, err := runCmd("init", "--domain", "engineering", "--with", "pm,qa", "--target", "codex,claude", "--no-hooks", "--no-agents"); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(env.dir)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := cfg.ResolveDomains()
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Primary != domains.DomainEngineering || !reflect.DeepEqual(resolved.Extensions, []domains.DomainID{domains.DomainPM, domains.DomainQA}) {
		t.Fatalf("init composition = %#v", resolved)
	}
	for _, name := range []string{
		".codex/agents/pm-delivery-lead.toml", ".codex/agents/qa-delivery-lead.toml",
		".claude/agents/pm-delivery-lead.md", ".claude/agents/qa-delivery-lead.md",
	} {
		if _, err := os.Stat(filepath.Join(env.dir, name)); err != nil {
			t.Errorf("init target missing %s: %v", name, err)
		}
	}
}

func TestInitStandaloneBundledPacksAndNoTargetGuidance(t *testing.T) {
	for _, primary := range []string{"pm", "qa"} {
		t.Run(primary, func(t *testing.T) {
			env := newTestEnvEmpty(t)
			output, err := runCmd("init", "--domain", primary, "--no-hooks", "--no-agents")
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := config.Load(env.dir)
			if err != nil {
				t.Fatal(err)
			}
			if got := cfg.PrimaryDomain(); got != primary {
				t.Fatalf("primary = %q", got)
			}
			if !strings.Contains(output, "hero install project . --target codex") {
				t.Fatalf("no-target guidance missing exact command: %s", output)
			}
		})
	}
}
