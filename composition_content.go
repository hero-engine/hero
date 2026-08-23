package hero

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/domains"
	"github.com/hero-engine/hero/internal/graph"
)

// ContentEntry is one resolved, installable artifact in a composed workspace.
// ID is stable and owner-namespaced; Path is the single harness-neutral output
// path that target renderers consume.
type ContentEntry struct {
	ID    string `json:"id"`
	Owner string `json:"owner"`
	Role  string `json:"role"`
	Kind  string `json:"kind"`
	Path  string `json:"path"`
}

// ContentPackActivation describes one pack's role in a resolved install.
type ContentPackActivation struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Bundled bool   `json:"bundled"`
}

// DomainComposition is the public, dependency-neutral composition contract.
type DomainComposition struct {
	Primary    string   `json:"primary"`
	Extensions []string `json:"extensions,omitempty"`
}

// SpecTypeAmendment declares a compatible, owner-namespaced extension to a
// canonical spec type. Amendments are manifest metadata, never shadow files.
type SpecTypeAmendment struct {
	ID             string   `json:"id"`
	Owner          string   `json:"owner"`
	TargetType     string   `json:"target_type"`
	ExtensionPoint string   `json:"extension_point"`
	Values         []string `json:"values"`
}

// ContentManifest is the validated canonical input to every harness renderer.
type ContentManifest struct {
	Primary            string                     `json:"primary"`
	Packs              []ContentPackActivation    `json:"packs"`
	Entries            []ContentEntry             `json:"entries"`
	DeferredEntries    []ContentEntry             `json:"deferred_entries,omitempty"`
	CommandHandlers    []CommandHandlerDescriptor `json:"command_handlers,omitempty"`
	SpecTypeAmendments []SpecTypeAmendment        `json:"spec_type_amendments,omitempty"`
}

// CommandIntentPredicate is an inspectable routing condition. Contains is
// matched case-insensitively against the request intent.
type CommandIntentPredicate struct {
	Contains string `json:"contains"`
}

// CommandHandlerDescriptor is the complete, data-level routing contract for
// one owner of a shared or projected command.
type CommandHandlerDescriptor struct {
	ID               string                   `json:"id"`
	Command          string                   `json:"command"`
	Owner            string                   `json:"owner"`
	Role             string                   `json:"role"`
	TargetAgent      string                   `json:"target_agent"`
	ArtifactTypes    []string                 `json:"artifact_types,omitempty"`
	LifecycleStates  []string                 `json:"lifecycle_states,omitempty"`
	IntentPredicates []CommandIntentPredicate `json:"intent_predicates,omitempty"`
	Priority         int                      `json:"priority"`
}

// CommandRouteRequest contains the explicit routing signals available at
// command dispatch time.
type CommandRouteRequest struct {
	Command      string `json:"command"`
	HandlerID    string `json:"handler_id,omitempty"`
	ArtifactType string `json:"artifact_type,omitempty"`
	Lifecycle    string `json:"lifecycle,omitempty"`
	Intent       string `json:"intent,omitempty"`
}

// CommandHandlerSelection is the executable routing result. ArtifactDomain is
// validated against the enabled pack stack through graph.DomainForHandler.
type CommandHandlerSelection struct {
	Handler        CommandHandlerDescriptor `json:"handler"`
	ArtifactDomain string                   `json:"artifact_domain"`
}

// CommandRoutingAmbiguityError prevents pack order from deciding equal matches.
type CommandRoutingAmbiguityError struct {
	Command    string
	HandlerIDs []string
}

func (e *CommandRoutingAmbiguityError) Error() string {
	return fmt.Sprintf("ambiguous command routing for %q between %s", e.Command, strings.Join(e.HandlerIDs, ", "))
}

// ContentCollisionError reports an undeclared output collision. Composition
// never resolves these by pack order.
type ContentCollisionError struct {
	ID          string
	Path        string
	FirstPath   string
	SecondPath  string
	FirstOwner  string
	SecondOwner string
}

func (e *ContentCollisionError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("content identity collision at %q claimed by %q (%s) and %q (%s); assign each contribution a unique stable ID", e.ID, e.FirstOwner, e.FirstPath, e.SecondOwner, e.SecondPath)
	}
	return fmt.Sprintf("content collision at %q claimed by %q and %q; namespace the artifact or declare a shared command router", e.Path, e.FirstOwner, e.SecondOwner)
}

type contentClaim struct {
	owner domains.DomainID
	role  domains.ActivationRole
	path  string
	data  []byte
}

type commandHandler struct {
	descriptor CommandHandlerDescriptor
	body       []byte
}

var extensionAgentEntrypoints = map[domains.DomainID]map[string]bool{
	domains.DomainPM: {
		"discovery-researcher": true, "intake-triager": true,
		"pm-delivery-lead": true, "pm-reviewer": true,
		"prd-author": true, "prioritization-strategist": true,
		"product-strategist": true, "roadmap-curator": true,
		"story-writer": true,
	},
	domains.DomainQA: {
		"coverage-strategist": true, "plan-author": true,
		"qa-delivery-lead": true, "qa-flake-curator": true,
		"qa-reviewer": true, "qa-strategist": true,
		"regression-curator": true, "release-readiness-strategist": true,
		"test-author": true, "test-issue-triager": true,
	},
}

// extensionCommandEntrypoints is deliberately smaller than each standalone
// pack's command surface. Every projected command is closed over the bounded
// extensionAgentEntrypoints roster: commands that mention a deep-pack agent
// stay available only when that pack is primary.
var extensionCommandEntrypoints = map[domains.DomainID]map[string]bool{
	domains.DomainPM: {
		"discover": true, "interview": true, "prd": true,
		"prioritize": true, "roadmap": true,
	},
	domains.DomainQA: {
		"author-cases": true, "author-plan": true, "coverage": true,
		"deliver": true, "diagnose": true, "plan-coverage": true,
		"promote-to-regression": true, "triage-flaky": true,
		"triage-test-issue": true,
	},
}

var declaredCommandHandlers = map[domains.DomainID]map[string]CommandHandlerDescriptor{
	domains.DomainCore: {
		"discover": {
			TargetAgent: "product-ideator", Priority: 0,
		},
	},
	domains.DomainEngineering: {
		"deliver": {
			TargetAgent: "feature-delivery-lead", Priority: 0,
		},
		"diagnose": {
			TargetAgent: "debug-investigator", Priority: 0,
		},
	},
	domains.DomainPM: {
		"discover": {
			TargetAgent: "discovery-researcher", Priority: 100,
			ArtifactTypes: []string{"intake", "opportunity", "roadmap-item"},
			IntentPredicates: []CommandIntentPredicate{
				{Contains: "customer"}, {Contains: "discovery"},
				{Contains: "interview"}, {Contains: "opportunity"},
			},
		},
		"interview":  {TargetAgent: "discovery-researcher", Priority: 100},
		"prd":        {TargetAgent: "prd-author", Priority: 100},
		"prioritize": {TargetAgent: "prioritization-strategist", Priority: 100},
		"roadmap":    {TargetAgent: "roadmap-curator", Priority: 100},
	},
	domains.DomainQA: {
		"author-cases": {TargetAgent: "test-author", Priority: 100},
		"author-plan":  {TargetAgent: "plan-author", Priority: 100},
		"coverage":     {TargetAgent: "coverage-strategist", Priority: 100},
		"deliver": {
			TargetAgent: "qa-delivery-lead", Priority: 100,
			ArtifactTypes:   []string{"test-plan", "test-case", "test-suite", "qa-finding"},
			LifecycleStates: []string{"qa-ready", "qa-rejected"},
			IntentPredicates: []CommandIntentPredicate{
				{Contains: "qa"}, {Contains: "test"}, {Contains: "coverage"},
			},
		},
		"diagnose": {
			TargetAgent: "test-issue-triager", Priority: 100,
			ArtifactTypes:   []string{"test-issue", "test-case", "test-run"},
			LifecycleStates: []string{"failing", "flaky"},
			IntentPredicates: []CommandIntentPredicate{
				{Contains: "test failure"}, {Contains: "flaky"}, {Contains: "qa"},
			},
		},
		"plan-coverage":         {TargetAgent: "qa-strategist", Priority: 100},
		"promote-to-regression": {TargetAgent: "regression-curator", Priority: 100},
		"triage-flaky":          {TargetAgent: "qa-flake-curator", Priority: 100},
		"triage-test-issue":     {TargetAgent: "test-issue-triager", Priority: 100},
	},
}

// ComposeContent resolves Core, one full primary pack, and bounded PM/QA
// extension projections into one deterministic filesystem. The result is
// entirely local and is validated before an installer receives it.
func ComposeContent(composition DomainComposition) (fs.FS, ContentManifest, error) {
	return composeContentWithSources(composition, CoreFS(), DomainFS)
}

func composeContentWithSources(
	composition DomainComposition,
	core fs.FS,
	domainSource func(string) (fs.FS, error),
) (fs.FS, ContentManifest, error) {
	declared := &domains.Composition{Primary: domains.DomainID(composition.Primary)}
	for _, extension := range composition.Extensions {
		declared.Extensions = append(declared.Extensions, domains.DomainID(extension))
	}
	validated, err := domains.ResolveComposition(&domains.Composition{
		Primary: declared.Primary, Extensions: declared.Extensions,
	}, "")
	if err != nil {
		return nil, ContentManifest{}, err
	}

	manifest := ContentManifest{Primary: string(validated.Primary)}
	files := newMemoryContentFS()
	claims := map[string]contentClaim{}
	identities := map[string]contentClaim{}
	handlers := map[string][]commandHandler{}

	add := func(id domains.DomainID, role domains.ActivationRole, source fs.FS, bounded bool) error {
		if id != domains.DomainCore {
			manifest.Packs = append(manifest.Packs, ContentPackActivation{ID: string(id), Role: string(role), Bundled: true})
		}
		return fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || name == "." || (bounded && extensionPathIgnored(name)) {
				return nil
			}
			kind, artifactName := contentIdentity(name)
			stableID := fmt.Sprintf("%s.%s.%s", id, kind, artifactName)
			if previous, exists := identities[stableID]; exists {
				return &ContentCollisionError{
					ID: stableID, FirstPath: previous.path, SecondPath: name,
					FirstOwner: string(previous.owner), SecondOwner: string(id),
				}
			}
			identities[stableID] = contentClaim{owner: id, role: role, path: name}
			if bounded && !extensionPathAllowed(id, name) {
				manifest.DeferredEntries = append(manifest.DeferredEntries, ContentEntry{
					ID: stableID, Owner: string(id), Role: string(role), Kind: kind, Path: name,
				})
				return nil
			}
			data, readErr := fs.ReadFile(source, name)
			if readErr != nil {
				return readErr
			}
			if bounded && strings.HasPrefix(name, "agents/") {
				data = advertiseComposedAgent(data)
			}
			descriptor, declaredHandler := commandHandlerDescriptor(id, role, artifactName)
			if previous, exists := claims[name]; exists {
				// Preserve legacy primary-only semantics exactly: a primary pack
				// shadows Core at the same path. Routers are introduced only when
				// an enabled extension adds another handler.
				if previous.owner == domains.DomainCore && role == domains.RolePrimary {
					copied := append([]byte(nil), data...)
					claims[name] = contentClaim{owner: id, role: role, path: name, data: copied}
					files.set(name, copied)
					manifest.Entries = append(manifest.Entries, ContentEntry{ID: stableID, Owner: string(id), Role: string(role), Kind: kind, Path: name})
					return nil
				}
				previousDescriptor, previousDeclared := commandHandlerDescriptor(previous.owner, previous.role, artifactName)
				if kind == "command" && declaredHandler && previousDeclared {
					if len(handlers[name]) == 0 {
						handlers[name] = append(handlers[name], commandHandler{
							descriptor: previousDescriptor, body: previous.data,
						})
						appendCommandHandler(&manifest, previousDescriptor)
					}
					handlers[name] = append(handlers[name], commandHandler{
						descriptor: descriptor, body: data,
					})
					appendCommandHandler(&manifest, descriptor)
					claims[name] = contentClaim{owner: domains.DomainCore, role: domains.RolePrimary, path: name}
					files.set(name, renderCommandRouter(artifactName, handlers[name]))
					manifest.Entries = append(manifest.Entries, ContentEntry{ID: stableID, Owner: string(id), Role: string(role), Kind: kind, Path: name})
					return nil
				}
				return &ContentCollisionError{Path: name, FirstOwner: string(previous.owner), SecondOwner: string(id)}
			}
			copied := append([]byte(nil), data...)
			claims[name] = contentClaim{owner: id, role: role, path: name, data: copied}
			files.set(name, copied)
			manifest.Entries = append(manifest.Entries, ContentEntry{ID: stableID, Owner: string(id), Role: string(role), Kind: kind, Path: name})
			if role == domains.RoleExtension && kind == "command" {
				if !declaredHandler {
					return fmt.Errorf("extension command %q has no declared handler descriptor", name)
				}
				appendCommandHandler(&manifest, descriptor)
			}
			return nil
		})
	}

	if err := add(domains.DomainCore, domains.RolePrimary, core, false); err != nil {
		return nil, ContentManifest{}, err
	}
	primaryFS, err := domainSource(string(validated.Primary))
	if err != nil {
		return nil, ContentManifest{}, err
	}
	if err := add(validated.Primary, domains.RolePrimary, primaryFS, false); err != nil {
		return nil, ContentManifest{}, err
	}
	for _, extension := range validated.Extensions {
		extensionFS, extensionErr := domainSource(string(extension))
		if extensionErr != nil {
			return nil, ContentManifest{}, extensionErr
		}
		if err := add(extension, domains.RoleExtension, extensionFS, true); err != nil {
			return nil, ContentManifest{}, err
		}
	}
	manifest.SpecTypeAmendments = resolveSpecTypeAmendments(validated)
	files.append("AGENTS.md", renderExtensionRouting(validated.Extensions))

	sort.Slice(manifest.Entries, func(i, j int) bool {
		if manifest.Entries[i].Path == manifest.Entries[j].Path {
			return manifest.Entries[i].ID < manifest.Entries[j].ID
		}
		return manifest.Entries[i].Path < manifest.Entries[j].Path
	})
	sort.Slice(manifest.DeferredEntries, func(i, j int) bool {
		return manifest.DeferredEntries[i].ID < manifest.DeferredEntries[j].ID
	})
	sort.Slice(manifest.CommandHandlers, func(i, j int) bool {
		return manifest.CommandHandlers[i].ID < manifest.CommandHandlers[j].ID
	})
	return files, manifest, nil
}

func commandHandlerDescriptor(id domains.DomainID, role domains.ActivationRole, command string) (CommandHandlerDescriptor, bool) {
	declared, ok := declaredCommandHandlers[id][command]
	if !ok {
		return CommandHandlerDescriptor{}, false
	}
	declared.ID = commandHandlerID(id, command)
	declared.Command = command
	declared.Owner = string(id)
	declared.Role = string(role)
	return declared, true
}

func appendCommandHandler(manifest *ContentManifest, descriptor CommandHandlerDescriptor) {
	for _, existing := range manifest.CommandHandlers {
		if existing.ID == descriptor.ID {
			return
		}
	}
	manifest.CommandHandlers = append(manifest.CommandHandlers, descriptor)
}

func advertiseComposedAgent(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "domains:") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			lines[i] = indent + `domains: ["*"]`
			break
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func resolveSpecTypeAmendments(composition domains.ResolvedComposition) []SpecTypeAmendment {
	if !composition.Contains(domains.DomainQA) {
		return nil
	}
	return []SpecTypeAmendment{{
		ID: "qa.spec-type.feature.lifecycle", Owner: "qa", TargetType: "feature",
		ExtensionPoint: "lifecycle", Values: []string{"qa-ready", "qa-rejected"},
	}}
}

func renderExtensionRouting(extensions []domains.DomainID) []byte {
	if len(extensions) == 0 {
		return nil
	}
	var out strings.Builder
	out.WriteString("\n\n## Enabled capability-pack entry points\n\n")
	out.WriteString("Engineering remains the fallback. Route specialist intent to these bounded entry points. Deeper pack content remains bundled locally rather than installed into the roster. List it with `hero domain content`; load one item with `hero domain content <owner>.<kind>.<name>` (for example, `hero domain content pm.agent.metrics-analyst`).\n")
	for _, extension := range extensions {
		fmt.Fprintf(&out, "\n### %s extension\n\nAgents: ", strings.ToUpper(string(extension)))
		var agents []string
		for agent := range extensionAgentEntrypoints[extension] {
			agents = append(agents, agent)
		}
		sort.Strings(agents)
		for i, agent := range agents {
			if i > 0 {
				out.WriteString(", ")
			}
			fmt.Fprintf(&out, "`%s`", agent)
		}
		out.WriteString(". Shared verbs use the installed composed router; pack-specific commands remain available by name.\n")
	}
	return []byte(out.String())
}

func extensionPathAllowed(id domains.DomainID, name string) bool {
	if strings.HasPrefix(name, "agents/") {
		stem := strings.TrimSuffix(path.Base(name), path.Ext(name))
		return extensionAgentEntrypoints[id][stem]
	}
	if strings.HasPrefix(name, "commands/") {
		stem := strings.TrimSuffix(path.Base(name), path.Ext(name))
		return extensionCommandEntrypoints[id][stem]
	}
	return strings.HasPrefix(name, "spec-types/")
}

func extensionPathIgnored(name string) bool {
	return strings.EqualFold(path.Base(name), "README.md") || name == "AGENTS.md" || name == "mission.md"
}

// ResolveDeferredContent returns one locally embedded contribution from an
// enabled pack without installing the pack's full roster. Stable IDs come from
// ContentManifest.DeferredEntries and resolve to the same source bytes used by
// a standalone primary-pack install.
func ResolveDeferredContent(manifest ContentManifest, stableID string) (ContentEntry, []byte, error) {
	stableID = strings.TrimSpace(stableID)
	owner := stableContentOwner(stableID)
	if owner != "" && !manifestPackEnabled(manifest, owner) {
		return ContentEntry{}, nil, fmt.Errorf("content %q belongs to disabled pack %q; enable it before loading", stableID, owner)
	}

	var matched *ContentEntry
	for i := range manifest.DeferredEntries {
		entry := manifest.DeferredEntries[i]
		if entry.ID != stableID {
			continue
		}
		if matched != nil {
			return ContentEntry{}, nil, &ContentCollisionError{
				ID: stableID, FirstPath: matched.Path, SecondPath: entry.Path,
				FirstOwner: matched.Owner, SecondOwner: entry.Owner,
			}
		}
		matched = &entry
	}
	if matched == nil {
		return ContentEntry{}, nil, fmt.Errorf("deferred content %q is not available in the enabled composition", stableID)
	}
	if !manifestPackEnabled(manifest, matched.Owner) {
		return ContentEntry{}, nil, fmt.Errorf("content %q belongs to disabled pack %q; enable it before loading", stableID, matched.Owner)
	}
	source, err := DomainFS(matched.Owner)
	if err != nil {
		return ContentEntry{}, nil, err
	}
	data, err := fs.ReadFile(source, matched.Path)
	if err != nil {
		return ContentEntry{}, nil, fmt.Errorf("loading bundled content %q: %w", stableID, err)
	}
	return *matched, data, nil
}

func stableContentOwner(stableID string) string {
	owner, _, ok := strings.Cut(stableID, ".")
	if !ok {
		return ""
	}
	return owner
}

func manifestPackEnabled(manifest ContentManifest, owner string) bool {
	if owner == manifest.Primary || owner == string(domains.DomainCore) {
		return true
	}
	for _, pack := range manifest.Packs {
		if pack.ID == owner {
			return true
		}
	}
	return false
}

func contentIdentity(name string) (string, string) {
	parts := strings.Split(name, "/")
	kind := strings.TrimSuffix(parts[0], "s")
	artifact := strings.TrimSuffix(path.Base(name), path.Ext(name))
	if parts[0] == "skills" && len(parts) > 1 {
		artifact = parts[1]
	}
	return kind, artifact
}

func commandHandlerID(owner domains.DomainID, name string) string {
	return fmt.Sprintf("%s.command.%s", owner, name)
}

// SelectCommandHandler executes the manifest routing contract. It never uses
// pack order: explicit handler IDs win, otherwise specificity and then
// declared priority are compared, with exact ties rejected as ambiguous.
func SelectCommandHandler(manifest ContentManifest, request CommandRouteRequest) (CommandHandlerSelection, error) {
	command := strings.TrimPrefix(strings.TrimSpace(request.Command), "/")
	type candidate struct {
		descriptor CommandHandlerDescriptor
		score      int
	}
	var candidates []candidate
	for _, descriptor := range manifest.CommandHandlers {
		if descriptor.Command != command {
			continue
		}
		if request.HandlerID != "" {
			if descriptor.ID == request.HandlerID {
				return selectedCommandHandler(manifest, descriptor)
			}
			continue
		}
		score, matches := commandHandlerSpecificity(descriptor, request)
		if matches {
			candidates = append(candidates, candidate{descriptor: descriptor, score: score})
		}
	}
	if len(candidates) == 0 {
		if request.HandlerID != "" {
			return CommandHandlerSelection{}, fmt.Errorf("command handler %q is not installed for /%s", request.HandlerID, command)
		}
		return CommandHandlerSelection{}, fmt.Errorf("no command handler matches /%s", command)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].descriptor.Priority != candidates[j].descriptor.Priority {
			return candidates[i].descriptor.Priority > candidates[j].descriptor.Priority
		}
		return candidates[i].descriptor.ID < candidates[j].descriptor.ID
	})
	best := candidates[0]
	var tied []string
	for _, current := range candidates {
		if current.score == best.score && current.descriptor.Priority == best.descriptor.Priority {
			tied = append(tied, current.descriptor.ID)
		}
	}
	if len(tied) > 1 {
		sort.Strings(tied)
		return CommandHandlerSelection{}, &CommandRoutingAmbiguityError{Command: command, HandlerIDs: tied}
	}
	return selectedCommandHandler(manifest, best.descriptor)
}

func commandHandlerSpecificity(descriptor CommandHandlerDescriptor, request CommandRouteRequest) (int, bool) {
	constrained := len(descriptor.ArtifactTypes)+len(descriptor.LifecycleStates)+len(descriptor.IntentPredicates) > 0
	score := 0
	if containsFold(descriptor.ArtifactTypes, request.ArtifactType) {
		score += 100
	}
	if containsFold(descriptor.LifecycleStates, request.Lifecycle) {
		score += 100
	}
	intent := strings.ToLower(request.Intent)
	for _, predicate := range descriptor.IntentPredicates {
		if predicate.Contains != "" && strings.Contains(intent, strings.ToLower(predicate.Contains)) {
			score += 10
		}
	}
	return score, !constrained || score > 0
}

func containsFold(values []string, value string) bool {
	if value == "" {
		return false
	}
	for _, candidate := range values {
		if strings.EqualFold(candidate, value) {
			return true
		}
	}
	return false
}

func selectedCommandHandler(manifest ContentManifest, descriptor CommandHandlerDescriptor) (CommandHandlerSelection, error) {
	composition := &domains.Composition{Primary: domains.DomainID(manifest.Primary)}
	for _, pack := range manifest.Packs {
		if pack.Role == string(domains.RoleExtension) {
			composition.Extensions = append(composition.Extensions, domains.DomainID(pack.ID))
		}
	}
	cfg := config.Config{Domains: composition}
	owner, err := graph.DomainForHandler(cfg, descriptor.Owner)
	if err != nil {
		return CommandHandlerSelection{}, err
	}
	return CommandHandlerSelection{Handler: descriptor, ArtifactDomain: owner}, nil
}

func renderCommandRouter(name string, handlers []commandHandler) []byte {
	var out strings.Builder
	fmt.Fprintf(&out, "---\ndescription: Deterministic composed router for /%s across the enabled Hero pack stack.\nhero_router: true\nhandlers:\n", name)
	for _, handler := range handlers {
		d := handler.descriptor
		fmt.Fprintf(&out, "  - id: %s\n    owner: %s\n    role: %s\n    target_agent: %s\n    priority: %d\n", d.ID, d.Owner, d.Role, d.TargetAgent, d.Priority)
		renderYAMLList(&out, "    artifact_types", d.ArtifactTypes)
		renderYAMLList(&out, "    lifecycle_states", d.LifecycleStates)
		if len(d.IntentPredicates) == 0 {
			out.WriteString("    intent_predicates: []\n")
		} else {
			out.WriteString("    intent_predicates:\n")
			for _, predicate := range d.IntentPredicates {
				fmt.Fprintf(&out, "      - contains: %q\n", predicate.Contains)
			}
		}
	}
	out.WriteString("---\n")
	fmt.Fprintf(&out, "# Composed /%s router\n\n", name)
	out.WriteString("Select exactly one handler from the enabled list below. Pack order is not a routing rule.\n\n")
	out.WriteString("## Selection contract\n\n")
	out.WriteString("1. Prefer a handler whose declared domain owns the explicit artifact type or whose specialist intent directly matches the request.\n")
	out.WriteString("2. Otherwise use the primary-domain handler as the general/default route. Core is the final fallback when no primary handler exists.\n")
	out.WriteString("3. If two handlers match with equal specificity, stop with `ambiguous command routing` and name both handler IDs; never choose by list or pack order.\n")
	out.WriteString("4. Stamp artifacts created by the selected handler with that handler's domain, independently of the workspace primary.\n\n")
	for _, handler := range handlers {
		d := handler.descriptor
		fmt.Fprintf(&out, "## Handler `%s`\n\nOwner: `%s`; activation role: `%s`; target agent: `%s`; priority: `%d`.\n\n", d.ID, d.Owner, d.Role, d.TargetAgent, d.Priority)
		out.Write(handler.body)
		out.WriteString("\n\n")
	}
	return []byte(out.String())
}

func renderYAMLList(out *strings.Builder, key string, values []string) {
	if len(values) == 0 {
		fmt.Fprintf(out, "%s: []\n", key)
		return
	}
	fmt.Fprintf(out, "%s:\n", key)
	for _, value := range values {
		fmt.Fprintf(out, "      - %q\n", value)
	}
}

// memoryContentFS is a small immutable-at-return filesystem used to hand one
// validated manifest to every installer without exposing testing-only types.
type memoryContentFS struct {
	files map[string][]byte
}

func newMemoryContentFS() *memoryContentFS {
	return &memoryContentFS{files: map[string][]byte{}}
}

func (m *memoryContentFS) set(name string, data []byte) {
	m.files[name] = append([]byte(nil), data...)
}

func (m *memoryContentFS) append(name string, data []byte) {
	if len(data) == 0 {
		return
	}
	m.files[name] = append(m.files[name], data...)
}

func (m *memoryContentFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	if data, ok := m.files[name]; ok {
		return &memoryFile{Reader: bytes.NewReader(data), info: memoryInfo{name: path.Base(name), size: int64(len(data))}}, nil
	}
	entries, err := m.ReadDir(name)
	if err != nil {
		return nil, err
	}
	return &memoryDir{entries: entries, info: memoryInfo{name: path.Base(name), dir: true}}, nil
}

func (m *memoryContentFS) ReadFile(name string) ([]byte, error) {
	data, ok := m.files[name]
	if !ok {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
	}
	return append([]byte(nil), data...), nil
}

func (m *memoryContentFS) Stat(name string) (fs.FileInfo, error) {
	if data, ok := m.files[name]; ok {
		return memoryInfo{name: path.Base(name), size: int64(len(data))}, nil
	}
	if _, err := m.ReadDir(name); err == nil {
		return memoryInfo{name: path.Base(name), dir: true}, nil
	}
	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

func (m *memoryContentFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == "" {
		name = "."
	}
	prefix := ""
	if name != "." {
		prefix = strings.TrimSuffix(name, "/") + "/"
	}
	children := map[string]memoryInfo{}
	for filename, data := range m.files {
		if !strings.HasPrefix(filename, prefix) {
			continue
		}
		remainder := strings.TrimPrefix(filename, prefix)
		if remainder == "" {
			continue
		}
		part, rest, _ := strings.Cut(remainder, "/")
		if rest != "" {
			children[part] = memoryInfo{name: part, dir: true}
		} else {
			children[part] = memoryInfo{name: part, size: int64(len(data))}
		}
	}
	if len(children) == 0 {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	entries := make([]fs.DirEntry, 0, len(children))
	for _, child := range children {
		entries = append(entries, child)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

type memoryInfo struct {
	name string
	size int64
	dir  bool
}

func (i memoryInfo) Name() string { return i.name }
func (i memoryInfo) Size() int64  { return i.size }
func (i memoryInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o555
	}
	return 0o444
}
func (i memoryInfo) ModTime() time.Time         { return time.Time{} }
func (i memoryInfo) IsDir() bool                { return i.dir }
func (i memoryInfo) Sys() any                   { return nil }
func (i memoryInfo) Type() fs.FileMode          { return i.Mode().Type() }
func (i memoryInfo) Info() (fs.FileInfo, error) { return i, nil }

type memoryFile struct {
	*bytes.Reader
	info memoryInfo
}

func (f *memoryFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *memoryFile) Close() error               { return nil }

type memoryDir struct {
	entries []fs.DirEntry
	index   int
	info    memoryInfo
}

func (d *memoryDir) Stat() (fs.FileInfo, error) { return d.info, nil }
func (d *memoryDir) Close() error               { return nil }
func (d *memoryDir) Read([]byte) (int, error)   { return 0, io.EOF }
func (d *memoryDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.index >= len(d.entries) && n > 0 {
		return nil, io.EOF
	}
	end := len(d.entries)
	if n > 0 && d.index+n < end {
		end = d.index + n
	}
	out := append([]fs.DirEntry(nil), d.entries[d.index:end]...)
	d.index = end
	return out, nil
}
