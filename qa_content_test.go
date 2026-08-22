package hero

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
)

var qaAgentInventory = []string{
	"coverage-curator", "coverage-strategist", "dead-regression-scrubber",
	"decision-table-author", "duplicate-detector", "exploratory-charter-author",
	"gherkin-author", "handoff-coordinator", "plan-author", "pm-rejection-router",
	"qa-delivery-lead", "qa-flake-curator", "qa-investigator", "qa-reject-story",
	"qa-reviewer", "qa-strategist", "regression-curator", "release-gate-reviewer",
	"release-readiness-strategist", "seam-requester", "stale-case-scrubber",
	"test-author", "test-issue-triager",
}

var qaCommandInventory = []string{
	"author-cases", "author-charter", "author-plan", "coverage", "deliver",
	"design", "diagnose", "plan-coverage", "promote-to-regression", "reject-story",
	"release-gate", "request-seam", "review", "scrub-qa", "search",
	"triage-flaky", "triage-test-issue", "why",
}

var qaSkillInventory = []string{
	"agile-testing-quadrants", "blocker-policy-evaluation", "boundary-value-analysis",
	"context-driven-principles", "coverage-budgeting", "coverage-gap-detection",
	"cross-pack-body-augmentation", "data-driven-authoring", "decision-table-authoring",
	"ears-test-derivation", "equivalence-partitioning", "exploratory-charter",
	"flake-triage", "flake-verdict-classification", "gherkin-authoring",
	"integration-run-state-reader", "lifecycle-overlay-awareness", "normalization-mapping",
	"qa-preset-detection", "regression-scoring", "release-readiness-framing",
	"risk-based-testing", "seam-request-shaping", "stability-scoring",
	"state-transition-testing", "step-by-step-authoring", "test-issue-triage",
	"three-action-rejection", "use-case-derivation", "verdict-output", "whole-team-quality",
}

var qaSpecTypeInventory = []string{
	"defect", "regression-suite", "release-gate", "test-case", "test-plan",
}

func TestQACapabilityPackInventory(t *testing.T) {
	pack, err := DomainFS("qa")
	if err != nil {
		t.Fatal(err)
	}
	required := []string{"AGENTS.md", "mission.md"}
	for _, name := range qaAgentInventory {
		required = append(required, path.Join("agents", name+".md"))
	}
	for _, name := range qaCommandInventory {
		required = append(required, path.Join("commands", name+".md"))
	}
	for _, name := range qaSkillInventory {
		required = append(required, path.Join("skills", name, "SKILL.md"))
	}
	for _, name := range qaSpecTypeInventory {
		required = append(required, path.Join("spec-types", name+".md"))
	}
	if err := validateRequiredPackFiles(pack, required); err != nil {
		t.Fatal(err)
	}
	if err := validateAgentSkillReferences(pack, CoreFS()); err != nil {
		t.Fatal(err)
	}
	if err := validatePackCrossReferences(pack, CoreFS()); err != nil {
		t.Fatal(err)
	}
	if err := validateAllArtifactReferences(pack, CoreFS(), map[string]bool{"completed": true}); err != nil {
		t.Fatal(err)
	}

	assertExactQAInventory(t, pack, "agents", ".md", qaAgentInventory)
	assertExactQAInventory(t, pack, "commands", ".md", qaCommandInventory)
	assertExactQASkillInventory(t, pack, qaSkillInventory)
	assertExactQAInventory(t, pack, "spec-types", ".md", qaSpecTypeInventory)
}

func TestQAReferenceValidationRejectsFreeformAndSpecTypeTargets(t *testing.T) {
	pack := fstest.MapFS{
		"agents/lead.md":     {Data: []byte("---\nname: lead\ndescription: lead\n---\nRoute gaps to `missing-specialist`.\n")},
		"spec-types/case.md": {Data: []byte("---\ntitle: Case\ntype: case\ndomain: qa\ncategory: work\nfrontmatter:\n  optional:\n    - { name: source, type: \"ref(missing-source-type)\" }\n---\n# Case\n")},
	}
	err := validateAllArtifactReferences(pack, fstest.MapFS{}, nil)
	if err == nil {
		t.Fatal("expected unresolved references")
	}
	for _, want := range []string{"agents/lead.md", "missing-specialist", "spec-types/case.md", "missing-source-type"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}

func TestQADomainSpecTypesPresent(t *testing.T) {
	typeFS := DomainSpecTypesFS("qa")
	if typeFS == nil {
		t.Fatal("DomainSpecTypesFS(qa) returned nil")
	}
	for _, name := range qaSpecTypeInventory {
		if _, err := fs.Stat(typeFS, name+".md"); err != nil {
			t.Errorf("QA spec type %q is not loadable: %v", name, err)
		}
	}
}

func assertExactQAInventory(t *testing.T, pack fs.FS, dir, suffix string, want []string) {
	t.Helper()
	entries, err := fs.ReadDir(pack, dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var got []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) || strings.EqualFold(entry.Name(), "README.md") {
			continue
		}
		got = append(got, strings.TrimSuffix(entry.Name(), suffix))
	}
	sort.Strings(got)
	want = append([]string(nil), want...)
	sort.Strings(want)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("%s inventory drifted\n got: %v\nwant: %v", dir, got, want)
	}
}

func assertExactQASkillInventory(t *testing.T, pack fs.FS, want []string) {
	t.Helper()
	entries, err := fs.ReadDir(pack, "skills")
	if err != nil {
		t.Fatalf("read skills: %v", err)
	}
	var got []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := fs.Stat(pack, path.Join("skills", entry.Name(), "SKILL.md")); err == nil {
			got = append(got, entry.Name())
		}
	}
	sort.Strings(got)
	want = append([]string(nil), want...)
	sort.Strings(want)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("skills inventory drifted\n got: %v\nwant: %v", got, want)
	}
}
