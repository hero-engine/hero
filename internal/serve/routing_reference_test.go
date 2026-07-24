package serve

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
)

var routingCodeSpanPattern = regexp.MustCompile("`([^`\\n]+)`")

type routingReferences struct {
	workflows []string
	mcpTools  []string
	skills    []string
}

type routingReferenceInventory struct {
	workflows map[string]bool
	mcpTools  map[string]bool
	skills    map[string]bool
}

func TestCanonicalRoutingReferencesResolveAgainstRealSurfaces(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "domains", "engineering", "routing.md"))
	if err != nil {
		t.Fatal(err)
	}

	references := extractRoutingReferences(string(data))
	if len(references.workflows) == 0 {
		t.Fatal("canonical routing source contained no workflow references")
	}

	inventory := routingInventory(t, root)
	if errors := validateRoutingReferences(references, inventory); len(errors) != 0 {
		t.Fatalf("invalid canonical routing references:\n%s", strings.Join(errors, "\n"))
	}
}

func TestRoutingReferenceValidationRejectsUnknownSurfaceNames(t *testing.T) {
	inventory := routingReferenceInventory{
		workflows: map[string]bool{"deliver": true},
		mcpTools:  map[string]bool{"hero_attention_snapshot": true},
		skills:    map[string]bool{"command-deliver": true},
	}

	for _, testCase := range []struct {
		name       string
		references routingReferences
		want       string
	}{
		{
			name:       "workflow",
			references: routingReferences{workflows: []string{"invented-workflow"}},
			want:       "workflow /invented-workflow",
		},
		{
			name:       "MCP tool",
			references: routingReferences{mcpTools: []string{"hero_invented_tool"}},
			want:       "MCP tool hero_invented_tool",
		},
		{
			name:       "installed skill",
			references: routingReferences{skills: []string{"command-invented"}},
			want:       "installed skill command-invented",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			errors := validateRoutingReferences(testCase.references, inventory)
			if len(errors) != 1 || !strings.Contains(errors[0], testCase.want) {
				t.Fatalf("errors = %#v; want one error containing %q", errors, testCase.want)
			}
		})
	}
}

func TestConversationalRouteCorpusUsesAdvertisedMCPTools(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "contracts", "attention", "testdata", "v1", "conversational-routes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture attention.ConversationalRouteFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if contractErr := attention.ValidateConversationalRouteFixture(fixture); contractErr != nil {
		t.Fatalf("fixture validation: %v", contractErr)
	}

	advertised := make(map[string]bool)
	server := NewMCPServer(filepath.Join(t.TempDir(), ".hero"), root, "test")
	for _, definition := range server.toolDefinitions() {
		advertised[definition.Name] = true
	}
	for _, routeCase := range fixture.Cases {
		if (routeCase.ExpectedSurface == attention.DispatchSurfaceMCPTool ||
			routeCase.ExpectedSurface == attention.DispatchSurfaceAdvertisedAction) &&
			!advertised[routeCase.ExpectedTool] {
			t.Errorf("%s: MCP tool %q is not advertised", routeCase.ID, routeCase.ExpectedTool)
		}
	}
}

func extractRoutingReferences(content string) routingReferences {
	var references routingReferences
	for _, match := range routingCodeSpanPattern.FindAllStringSubmatch(content, -1) {
		token := strings.TrimSpace(match[1])
		switch {
		case strings.HasPrefix(token, "/"):
			name := strings.TrimPrefix(strings.Fields(token)[0], "/")
			references.workflows = append(references.workflows, name)
		case strings.HasPrefix(token, "hero_") && !strings.ContainsAny(token, " \t"):
			references.mcpTools = append(references.mcpTools, token)
		case strings.HasPrefix(token, "command-") && !strings.ContainsAny(token, " \t/"):
			references.skills = append(references.skills, token)
		}
	}
	return references
}

func validateRoutingReferences(references routingReferences, inventory routingReferenceInventory) []string {
	var errors []string
	for _, name := range references.workflows {
		if !inventory.workflows[name] {
			errors = append(errors, fmt.Sprintf("workflow /%s does not exist", name))
		}
	}
	for _, name := range references.mcpTools {
		if !inventory.mcpTools[name] {
			errors = append(errors, fmt.Sprintf("MCP tool %s does not exist", name))
		}
	}
	for _, name := range references.skills {
		if !inventory.skills[name] {
			errors = append(errors, fmt.Sprintf("installed skill %s does not exist", name))
		}
	}
	sort.Strings(errors)
	return errors
}

func routingInventory(t *testing.T, root string) routingReferenceInventory {
	t.Helper()
	workflows := make(map[string]bool)
	for _, dir := range []string{
		filepath.Join(root, "core", "commands"),
		filepath.Join(root, "domains", "engineering", "commands"),
	} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
				workflows[strings.TrimSuffix(entry.Name(), ".md")] = true
			}
		}
	}

	skills := make(map[string]bool)
	for _, dir := range []string{
		filepath.Join(root, "core", "skills"),
		filepath.Join(root, "domains", "engineering", "skills"),
	} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				skills[entry.Name()] = true
			}
		}
	}
	for workflow := range workflows {
		skills["command-"+workflow] = true
	}

	mcpTools := make(map[string]bool)
	server := NewMCPServer(filepath.Join(t.TempDir(), ".hero"), root, "test")
	for _, definition := range server.toolDefinitions() {
		mcpTools[definition.Name] = true
	}

	return routingReferenceInventory{
		workflows: workflows,
		mcpTools:  mcpTools,
		skills:    skills,
	}
}
