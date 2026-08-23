package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	hero "github.com/hero-engine/hero"
)

func TestComposedInstallAllTargets(t *testing.T) {
	tests := []Target{TargetClaude, TargetOpenCode, TargetCursor, TargetCopilot, TargetCodex, TargetGeneric, TargetGrok}
	content, manifest, err := hero.ComposeContent(hero.DomainComposition{Primary: "engineering", Extensions: []string{"pm", "qa"}})
	if err != nil {
		t.Fatal(err)
	}
	shared := map[string][]hero.CommandHandlerDescriptor{}
	for _, handler := range manifest.CommandHandlers {
		shared[handler.Command] = append(shared[handler.Command], handler)
	}
	for command, handlers := range shared {
		if len(handlers) < 2 {
			delete(shared, command)
		}
	}
	for _, target := range tests {
		t.Run(string(target), func(t *testing.T) {
			root := t.TempDir()
			mkHeroDir(t, root)
			if _, err := Run(Options{ContentFS: content, Target: target, Mode: ModeProject, TargetDir: root, Force: true, Domain: "engineering"}); err != nil {
				t.Fatal(err)
			}
			advertisedAgents := map[string]bool{}
			for _, entry := range manifest.Entries {
				if entry.Role != "extension" || entry.Kind != "agent" {
					continue
				}
				name := strings.TrimSuffix(filepath.Base(entry.Path), ".md")
				advertisedAgents[entry.Owner+"/"+name] = true
				installed := composedInstalledPath(target, "agent", name)
				if _, err := os.Stat(filepath.Join(root, installed)); err != nil {
					t.Errorf("missing advertised %s entry point %s: %v", entry.Owner, installed, err)
				}
			}
			for _, handler := range manifest.CommandHandlers {
				if handler.Role != "extension" {
					continue
				}
				if !advertisedAgents[handler.Owner+"/"+handler.TargetAgent] {
					t.Errorf("%s targets unadvertised agent %s", handler.ID, handler.TargetAgent)
				}
				if _, err := os.Stat(filepath.Join(root, composedInstalledPath(target, "agent", handler.TargetAgent))); err != nil {
					t.Errorf("%s target agent was not installed: %v", handler.ID, err)
				}
				if _, err := os.Stat(filepath.Join(root, composedInstalledPath(target, "command", handler.Command))); err != nil {
					t.Errorf("%s command was not installed: %v", handler.ID, err)
				}
			}
			for command, handlers := range shared {
				routerPath := composedInstalledPath(target, "command", command)
				router, err := os.ReadFile(filepath.Join(root, routerPath))
				if err != nil {
					t.Fatalf("read router %s: %v", routerPath, err)
				}
				for _, handler := range handlers {
					for _, metadata := range []string{handler.ID, handler.TargetAgent, "priority"} {
						if !strings.Contains(string(router), metadata) {
							t.Errorf("router %s missing metadata %q", routerPath, metadata)
						}
					}
				}
				if !strings.Contains(string(router), "ambiguous command routing") {
					t.Errorf("router %s missing ambiguity contract", routerPath)
				}
			}
		})
	}
}

func composedInstalledPath(target Target, kind, name string) string {
	switch target {
	case TargetClaude:
		return filepath.Join(".claude", kind+"s", name+".md")
	case TargetOpenCode:
		return filepath.Join(".opencode", kind+"s", name+".md")
	case TargetCursor:
		return filepath.Join(".cursor", "rules", kind+"s", name+".md")
	case TargetCopilot:
		return filepath.Join(".github", "prompts", kind+"s", name+".prompt.md")
	case TargetCodex:
		if kind == "agent" {
			return filepath.Join(".codex", "agents", name+".toml")
		}
		return filepath.Join(".agents", "skills", "command-"+name, "SKILL.md")
	case TargetGeneric:
		return filepath.Join(".ai", kind+"s", name+".md")
	case TargetGrok:
		if kind == "agent" {
			return filepath.Join(".grok", "agents", name+".md")
		}
		return filepath.Join(".grok", "skills", "command-"+name, "SKILL.md")
	default:
		return ""
	}
}

func TestComposedInstallDisablePrunesHeroFilesAndPreservesProjectFiles(t *testing.T) {
	root := t.TempDir()
	mkHeroDir(t, root)
	full, _, err := hero.ComposeContent(hero.DomainComposition{Primary: "engineering", Extensions: []string{"pm", "qa"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{ContentFS: full, Target: TargetCodex, Mode: ModeProject, TargetDir: root, Force: true, Domain: "engineering"}); err != nil {
		t.Fatal(err)
	}
	qaFile := filepath.Join(root, ".codex", "agents", "qa-delivery-lead.toml")
	if _, err := os.Stat(qaFile); err != nil {
		t.Fatalf("QA file was not installed: %v", err)
	}
	projectFile := filepath.Join(root, ".codex", "agents", "project-owned.toml")
	if err := os.WriteFile(projectFile, []byte("project-owned"), 0o644); err != nil {
		t.Fatal(err)
	}
	withoutQA, _, err := hero.ComposeContent(hero.DomainComposition{Primary: "engineering", Extensions: []string{"pm"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{ContentFS: withoutQA, Target: TargetCodex, Mode: ModeProject, TargetDir: root, Force: true, Domain: "engineering"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(qaFile); !os.IsNotExist(err) {
		t.Fatalf("disabled QA file still exists: %v", err)
	}
	if data, err := os.ReadFile(projectFile); err != nil || string(data) != "project-owned" {
		t.Fatalf("project file changed: %q, %v", data, err)
	}
}
