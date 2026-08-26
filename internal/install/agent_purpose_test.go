package install

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	hero "github.com/hero-engine/hero"
)

func TestCanonicalAgentPurposesCoverDynamicInventory(t *testing.T) {
	packs := []struct {
		name string
		fs   fs.FS
	}{
		{name: "core", fs: hero.CoreFS()},
	}
	for _, domain := range []string{"engineering", "pm", "sales"} {
		pack, err := hero.DomainFS(domain)
		if err != nil {
			t.Fatalf("DomainFS(%q): %v", domain, err)
		}
		packs = append(packs, struct {
			name string
			fs   fs.FS
		}{name: domain, fs: pack})
	}

	for _, pack := range packs {
		t.Run(pack.name, func(t *testing.T) {
			if err := validateCanonicalAgentPurposes(pack.fs); err != nil {
				t.Fatal(err)
			}
			entries, err := fs.ReadDir(pack.fs, "agents")
			if err != nil {
				t.Fatal(err)
			}
			validated := 0
			for _, entry := range entries {
				if !entry.IsDir() && !isContentReadme(entry.Name()) {
					validated++
				}
			}
			if validated == 0 {
				t.Fatal("expected at least one installable agent descriptor")
			}
		})
	}
}

func TestCanonicalAgentPurposeRejectsMissingDuplicateAndUnknown(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing", body: "---\nname: example\n---\nbody"},
		{name: "empty", body: "---\nname: example\npurpose:\n---\nbody"},
		{name: "duplicate", body: "---\nname: example\npurpose: agent\npurpose: review\n---\nbody"},
		{name: "unknown", body: "---\nname: example\npurpose: chat\n---\nbody"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content := fstest.MapFS{"agents/example.md": {Data: []byte(test.body)}}
			if err := validateCanonicalAgentPurposes(content); err == nil {
				t.Fatalf("expected %s purpose to fail", test.name)
			}
		})
	}
}

func TestCanonicalAgentPurposeRunRejectsEntirelyMissingPackBeforeWrites(t *testing.T) {
	content := fstest.MapFS{
		"agents/example.md": {Data: []byte("---\nname: example\n---\nbody")},
	}
	target := t.TempDir()
	_, err := Run(Options{
		ContentFS: content,
		Target:    TargetGeneric,
		Mode:      ModeProject,
		TargetDir: target,
		Force:     true,
	})
	if err == nil || !strings.Contains(err.Error(), "missing required purpose") {
		t.Fatalf("Run() error = %v, want missing required purpose", err)
	}
	if _, statErr := os.Stat(filepath.Join(target, ".ai")); !os.IsNotExist(statErr) {
		t.Fatalf("canonical validation wrote target content before failing: %v", statErr)
	}
}

func TestCanonicalAgentPurposeAcceptsClosedVocabulary(t *testing.T) {
	for _, purpose := range []AgentPurpose{
		AgentPurposeDesign, AgentPurposeDiagnose, AgentPurposeAgent,
		AgentPurposeDraft, AgentPurposeReview, AgentPurposeAssist,
	} {
		t.Run(string(purpose), func(t *testing.T) {
			raw := []byte("---\nname: example\npurpose: " + purpose + "\n---\nbody")
			if got, err := parseRequiredAgentPurpose(raw); err != nil || got != purpose {
				t.Fatalf("parse purpose: got %q, err %v", got, err)
			}
		})
	}
}

func TestCanonicalAgentPurposeDeliveryLeadsAreDesign(t *testing.T) {
	tests := []struct {
		domain string
		name   string
	}{
		{domain: "engineering", name: "feature-delivery-lead.md"},
		{domain: "engineering", name: "platform-delivery-lead.md"},
		{domain: "pm", name: "pm-delivery-lead.md"},
	}
	for _, test := range tests {
		t.Run(test.domain+"/"+test.name, func(t *testing.T) {
			pack, err := hero.DomainFS(test.domain)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := fs.ReadFile(pack, "agents/"+test.name)
			if err != nil {
				t.Fatal(err)
			}
			purpose, err := parseRequiredAgentPurpose(raw)
			if err != nil {
				t.Fatal(err)
			}
			if purpose != AgentPurposeDesign {
				t.Fatalf("purpose = %q, want %q", purpose, AgentPurposeDesign)
			}
		})
	}
}

func TestCanonicalAgentPurposeInstallContractsAllTargets(t *testing.T) {
	pm, err := hero.DomainFS("pm")
	if err != nil {
		t.Fatal(err)
	}
	content := hero.OverlayFS(pm, hero.CoreFS())
	targets := []struct {
		target Target
		path   string
	}{
		{TargetClaude, ".claude/agents/pm-delivery-lead.md"},
		{TargetOpenCode, ".opencode/agents/pm-delivery-lead.md"},
		{TargetCursor, ".cursor/rules/agents/pm-delivery-lead.md"},
		{TargetCopilot, ".github/prompts/agents/pm-delivery-lead.prompt.md"},
		{TargetCodex, ".codex/agents/pm-delivery-lead.toml"},
		{TargetGeneric, ".ai/agents/pm-delivery-lead.md"},
		{TargetGrok, ".grok/agents/pm-delivery-lead.md"},
	}
	for _, test := range targets {
		t.Run(string(test.target), func(t *testing.T) {
			h := newInstallHarness(t)
			h.Run(test.target, func(opts *Options) {
				opts.ContentFS = content
				opts.Domain = "pm"
			})
			contract, ok := ContractsFor(test.target, KindAgents)
			if !ok {
				t.Fatalf("missing agent contract for %s", test.target)
			}
			h.assertContract(filepath.Join(h.TargetDir, test.path), contract, test.target, KindAgents)
		})
	}
}

func TestCustomSourceAgentsRemainOutsidePurposeValidation(t *testing.T) {
	h := newInstallHarness(t)
	raw, err := fs.ReadFile(os.DirFS(h.SourceDir), "agents/engineer.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, count := readAgentPurposeFrontmatter(raw); count != 0 {
		t.Fatal("legacy SourceDir fixture unexpectedly declares purpose")
	}
	h.Run(TargetGeneric, nil)
	h.mustExist(".ai/agents/engineer.md")
}
