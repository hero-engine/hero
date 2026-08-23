package hero

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestComposeContentEngineeringPrimaryWithPMAndQA(t *testing.T) {
	content, manifest, err := ComposeContent(DomainComposition{
		Primary: "engineering", Extensions: []string{"pm", "qa"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Primary != "engineering" {
		t.Fatalf("primary = %q", manifest.Primary)
	}
	if got, want := manifest.Packs, []ContentPackActivation{
		{ID: "engineering", Role: "primary", Bundled: true},
		{ID: "pm", Role: "extension", Bundled: true},
		{ID: "qa", Role: "extension", Bundled: true},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("packs = %#v, want %#v", got, want)
	}

	for _, name := range []string{
		"agents/feature-delivery-lead.md", "agents/pm-delivery-lead.md", "agents/qa-delivery-lead.md",
		"spec-types/roadmap-item.md", "spec-types/test-plan.md",
	} {
		if _, err := fs.Stat(content, name); err != nil {
			t.Errorf("expected composed artifact %s: %v", name, err)
		}
	}
	for _, name := range []string{
		"agents/metrics-analyst.md", "agents/decision-table-author.md",
		"skills/prd-structure/SKILL.md", "skills/risk-based-testing/SKILL.md",
		"agents/duplicate-detector.md", "agents/handoff-coordinator.md",
		"agents/pm-roadmap-reviewer.md", "agents/qa-handoff-coordinator.md",
	} {
		if _, err := fs.Stat(content, name); !isNotExist(err) {
			t.Errorf("bounded extension artifact unexpectedly advertised at %s", name)
		}
	}
	routing, err := fs.ReadFile(content, "AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, entrypoint := range []string{"`pm-delivery-lead`", "`qa-delivery-lead`"} {
		if count := strings.Count(string(routing), entrypoint); count != 1 {
			t.Errorf("extension routing advertises %s %d times, want once", entrypoint, count)
		}
	}
	for _, guidance := range []string{"`hero domain content`", "`hero domain content <owner>.<kind>.<name>`"} {
		if !strings.Contains(string(routing), guidance) {
			t.Errorf("extension routing missing on-demand guidance %s", guidance)
		}
	}

	for command, handlers := range map[string][]string{
		"commands/deliver.md":  {"engineering.command.deliver", "qa.command.deliver"},
		"commands/diagnose.md": {"engineering.command.diagnose", "qa.command.diagnose"},
		"commands/discover.md": {"core.command.discover", "pm.command.discover"},
	} {
		data, err := fs.ReadFile(content, command)
		if err != nil {
			t.Fatalf("read %s: %v", command, err)
		}
		body := string(data)
		for _, handler := range handlers {
			if !strings.Contains(body, handler) {
				t.Errorf("%s missing handler %s", command, handler)
			}
		}
		for _, contract := range []string{"Pack order is not a routing rule", "ambiguous command routing", "Stamp artifacts"} {
			if !strings.Contains(body, contract) {
				t.Errorf("%s missing routing contract %q", command, contract)
			}
		}
	}
	for _, command := range []string{
		"commands/design.md", "commands/handoff.md", "commands/review.md",
		"commands/scrub.md", "commands/why.md",
	} {
		data, err := fs.ReadFile(content, command)
		if err != nil {
			t.Fatalf("read primary command %s: %v", command, err)
		}
		if strings.Contains(string(data), "hero_router: true") {
			t.Errorf("deep-pack command unexpectedly projected into router %s", command)
		}
	}
}

func TestComposeContentStandalonePrimaryUsesFullPack(t *testing.T) {
	for _, primary := range []string{"engineering", "sales", "pm", "qa"} {
		t.Run(primary, func(t *testing.T) {
			content, _, err := ComposeContent(DomainComposition{Primary: primary})
			if err != nil {
				t.Fatal(err)
			}
			domainFS, err := DomainFS(primary)
			if err != nil {
				t.Fatal(err)
			}
			legacy := OverlayFS(domainFS, CoreFS())
			if got, want := readFSTree(t, content), readFSTree(t, legacy); !reflect.DeepEqual(got, want) {
				t.Errorf("primary-only composed output drifted from legacy overlay for %s", primary)
			}
		})
	}
}

func readFSTree(t *testing.T, source fs.FS) map[string]string {
	t.Helper()
	out := map[string]string{}
	if err := fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(source, name)
		if err != nil {
			return err
		}
		out[name] = string(data)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestComposeContentDeterministic(t *testing.T) {
	composition := DomainComposition{Primary: "engineering", Extensions: []string{"pm", "qa"}}
	firstFS, firstManifest, err := ComposeContent(composition)
	if err != nil {
		t.Fatal(err)
	}
	secondFS, secondManifest, err := ComposeContent(composition)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstManifest, secondManifest) {
		t.Fatal("manifest changed between identical resolutions")
	}
	for _, entry := range firstManifest.Entries {
		first, err := fs.ReadFile(firstFS, entry.Path)
		if err != nil {
			t.Fatal(err)
		}
		second, err := fs.ReadFile(secondFS, entry.Path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("%s changed between identical resolutions", entry.Path)
		}
	}
}

func TestComposeContentRejectsInvalidComposition(t *testing.T) {
	_, _, err := ComposeContent(DomainComposition{Primary: "pm", Extensions: []string{"engineering"}})
	if err == nil || !strings.Contains(err.Error(), "does not support") {
		t.Fatalf("error = %v", err)
	}
}

func TestComposeContentRejectsUndeclaredCollisionWithBothOwners(t *testing.T) {
	sources := map[string]fs.FS{
		"engineering": fstest.MapFS{"spec-types/clash.md": {Data: []byte("engineering")}},
		"pm":          fstest.MapFS{"spec-types/clash.md": {Data: []byte("pm")}},
	}
	_, _, err := composeContentWithSources(
		DomainComposition{Primary: "engineering", Extensions: []string{"pm"}},
		fstest.MapFS{"agents/core.md": {Data: []byte("core")}},
		func(domain string) (fs.FS, error) { return sources[domain], nil },
	)
	var collision *ContentCollisionError
	if !errors.As(err, &collision) {
		t.Fatalf("error = %v, want ContentCollisionError", err)
	}
	if collision.Path != "spec-types/clash.md" || collision.FirstOwner != "engineering" || collision.SecondOwner != "pm" {
		t.Fatalf("collision = %#v", collision)
	}
}

func TestComposeContentRejectsDuplicateStableIDWithIdentityAndOwners(t *testing.T) {
	sources := map[string]fs.FS{
		"engineering": fstest.MapFS{
			"agents/one/same.md": {Data: []byte("first")},
			"agents/two/same.md": {Data: []byte("second")},
		},
	}
	_, _, err := composeContentWithSources(
		DomainComposition{Primary: "engineering"},
		fstest.MapFS{"agents/core.md": {Data: []byte("core")}},
		func(domain string) (fs.FS, error) { return sources[domain], nil },
	)
	var collision *ContentCollisionError
	if !errors.As(err, &collision) {
		t.Fatalf("error = %v, want ContentCollisionError", err)
	}
	if collision.ID != "engineering.agent.same" || collision.FirstOwner != "engineering" || collision.SecondOwner != "engineering" {
		t.Fatalf("collision = %#v", collision)
	}
	if collision.FirstPath == collision.SecondPath || collision.FirstPath == "" || collision.SecondPath == "" {
		t.Fatalf("collision paths are not actionable: %#v", collision)
	}
	for _, want := range []string{collision.ID, collision.FirstOwner, collision.FirstPath, collision.SecondPath} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("collision error missing %q: %v", want, err)
		}
	}
}

func TestComposeContentRejectsSharedCommandWithoutBothOwnerDescriptors(t *testing.T) {
	sources := map[string]fs.FS{
		"engineering": fstest.MapFS{"commands/prd.md": {Data: []byte("engineering")}},
		"pm":          fstest.MapFS{"commands/prd.md": {Data: []byte("pm")}},
	}
	_, _, err := composeContentWithSources(
		DomainComposition{Primary: "engineering", Extensions: []string{"pm"}},
		fstest.MapFS{"agents/core.md": {Data: []byte("core")}},
		func(domain string) (fs.FS, error) { return sources[domain], nil },
	)
	var collision *ContentCollisionError
	if !errors.As(err, &collision) {
		t.Fatalf("error = %v, want undeclared command ContentCollisionError", err)
	}
	if collision.Path != "commands/prd.md" || collision.FirstOwner != "engineering" || collision.SecondOwner != "pm" {
		t.Fatalf("collision = %#v", collision)
	}
}

func TestComposedExtensionCommandsAreClosedOverAdvertisedAgents(t *testing.T) {
	content, manifest, err := ComposeContent(DomainComposition{Primary: "engineering", Extensions: []string{"pm", "qa"}})
	if err != nil {
		t.Fatal(err)
	}
	advertised := map[string]map[string]bool{}
	for _, entry := range manifest.Entries {
		if entry.Role != "extension" || entry.Kind != "agent" {
			continue
		}
		if advertised[entry.Owner] == nil {
			advertised[entry.Owner] = map[string]bool{}
		}
		advertised[entry.Owner][strings.TrimSuffix(pathBase(entry.Path), ".md")] = true
	}
	handlers := map[string]CommandHandlerDescriptor{}
	for _, handler := range manifest.CommandHandlers {
		handlers[handler.Owner+"/"+handler.Command] = handler
		if handler.Role == "extension" && !advertised[handler.Owner][handler.TargetAgent] {
			t.Errorf("handler %s targets unadvertised agent %s", handler.ID, handler.TargetAgent)
		}
	}
	for _, entry := range manifest.Entries {
		if entry.Role != "extension" || entry.Kind != "command" {
			continue
		}
		command := strings.TrimSuffix(pathBase(entry.Path), ".md")
		handler, ok := handlers[entry.Owner+"/"+command]
		if !ok {
			t.Errorf("extension command %s has no handler descriptor", entry.Path)
			continue
		}
		body, err := fs.ReadFile(content, entry.Path)
		if err != nil {
			t.Fatal(err)
		}
		domainFS, err := DomainFS(entry.Owner)
		if err != nil {
			t.Fatal(err)
		}
		agentEntries, err := fs.ReadDir(domainFS, "agents")
		if err != nil {
			t.Fatal(err)
		}
		for _, agentEntry := range agentEntries {
			agent := strings.TrimSuffix(agentEntry.Name(), ".md")
			if strings.Contains(string(body), "`"+agent+"`") && !advertised[entry.Owner][agent] {
				t.Errorf("%s references missing extension agent %s (declared target %s)", entry.Path, agent, handler.TargetAgent)
			}
		}
	}
}

func TestResolveDeferredContentLoadsEmbeddedBytesForEnabledPack(t *testing.T) {
	content, manifest, err := ComposeContent(DomainComposition{Primary: "engineering", Extensions: []string{"pm", "qa"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(content, "agents/metrics-analyst.md"); !isNotExist(err) {
		t.Fatalf("deep agent was union-installed: %v", err)
	}
	if _, err := fs.Stat(content, "skills/metrics-design/SKILL.md"); !isNotExist(err) {
		t.Fatalf("deep skill was union-installed: %v", err)
	}

	for _, id := range []string{"pm.agent.metrics-analyst", "pm.skill.metrics-design", "qa.agent.decision-table-author", "qa.skill.decision-table-authoring"} {
		entry, got, err := ResolveDeferredContent(manifest, id)
		if err != nil {
			t.Fatalf("resolve %s: %v", id, err)
		}
		source, err := DomainFS(entry.Owner)
		if err != nil {
			t.Fatal(err)
		}
		want, err := fs.ReadFile(source, entry.Path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s did not resolve directly from bundled local content", id)
		}
	}
}

func TestResolveDeferredContentRejectsDisabledOwner(t *testing.T) {
	_, manifest, err := ComposeContent(DomainComposition{Primary: "engineering", Extensions: []string{"qa"}})
	if err != nil {
		t.Fatal(err)
	}
	manifest.Packs = []ContentPackActivation{{ID: "engineering", Role: "primary", Bundled: true}}
	_, _, err = ResolveDeferredContent(manifest, "qa.agent.decision-table-author")
	if err == nil || !strings.Contains(err.Error(), "disabled pack \"qa\"") {
		t.Fatalf("error = %v, want disabled-owner rejection", err)
	}
}

func pathBase(name string) string {
	parts := strings.Split(name, "/")
	return parts[len(parts)-1]
}

func TestSelectCommandHandlerUsesSpecificityAndDurableOwner(t *testing.T) {
	_, manifest, err := ComposeContent(DomainComposition{Primary: "engineering", Extensions: []string{"pm", "qa"}})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		request CommandRouteRequest
		owner   string
	}{
		{name: "QA artifact selects QA delivery", request: CommandRouteRequest{Command: "/deliver", ArtifactType: "test-case"}, owner: "qa"},
		{name: "general delivery falls back to primary", request: CommandRouteRequest{Command: "deliver", ArtifactType: "feature"}, owner: "engineering"},
		{name: "specialist discovery intent selects PM", request: CommandRouteRequest{Command: "discover", Intent: "customer discovery interview"}, owner: "pm"},
		{name: "general discovery falls back to Core", request: CommandRouteRequest{Command: "discover", Intent: "brainstorm platform architecture"}, owner: "core"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selection, err := SelectCommandHandler(manifest, tt.request)
			if err != nil {
				t.Fatal(err)
			}
			if selection.Handler.Owner != tt.owner || selection.ArtifactDomain != tt.owner {
				t.Fatalf("selection = %#v, want durable owner %q", selection, tt.owner)
			}
		})
	}
}

func TestSelectCommandHandlerRejectsEqualSpecificityAmbiguity(t *testing.T) {
	manifest := ContentManifest{
		Primary: "engineering",
		Packs: []ContentPackActivation{
			{ID: "engineering", Role: "primary", Bundled: true},
			{ID: "qa", Role: "extension", Bundled: true},
		},
		CommandHandlers: []CommandHandlerDescriptor{
			{ID: "engineering.command.review", Command: "review", Owner: "engineering", Role: "primary", TargetAgent: "pr-reviewer", Priority: 10},
			{ID: "qa.command.review", Command: "review", Owner: "qa", Role: "extension", TargetAgent: "qa-reviewer", Priority: 10},
		},
	}
	_, err := SelectCommandHandler(manifest, CommandRouteRequest{Command: "review"})
	var ambiguous *CommandRoutingAmbiguityError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("error = %v, want CommandRoutingAmbiguityError", err)
	}
	if got, want := ambiguous.HandlerIDs, []string{"engineering.command.review", "qa.command.review"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler IDs = %v, want %v", got, want)
	}
}

func isNotExist(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "file does not exist") || strings.Contains(err.Error(), "not exist"))
}

func TestComposedContentHasNoLiteTreesOrDuplicateRenderedPaths(t *testing.T) {
	_, manifest, err := ComposeContent(DomainComposition{Primary: "engineering", Extensions: []string{"pm", "qa"}})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, entry := range manifest.Entries {
		if strings.Contains(entry.Path, "lite") || strings.Contains(entry.ID, "lite") {
			t.Errorf("split lite artifact leaked into manifest: %#v", entry)
		}
		if owner, duplicate := seen[entry.Path]; duplicate && entry.Kind != "command" {
			t.Errorf("unexplained rendered collision at %s: %s and %s", entry.Path, owner, entry.ID)
		}
		seen[entry.Path] = fmt.Sprintf("%s/%s", entry.Owner, entry.ID)
	}
}
