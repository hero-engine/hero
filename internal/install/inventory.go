package install

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	hero "github.com/hero-engine/hero"
)

// inventory.go — the per-target install introspection `hero doctor` renders
// as its "Installed harness targets" table.
//
// The whole correctness surface lives here, next to the per-target installer
// functions (target_*.go) it must not drift from. Expected counts come from
// the same EnumerateContent selectors install uses; the codex skills rollup
// mirrors codexSkillDirNames arithmetic (guarded by inventory_test.go). It is
// READ-ONLY: it counts files on disk against what install would materialize,
// and never writes.

// KindCount is the expected-vs-actual file count for one content kind on one
// installed target.
//
// NotApplicable is how codex's commands cell renders as an em dash rather than
// a number: codex has no command loader (SlashCommand is a built-in enum), so
// "0 of N commands" would read as a broken install. It is modeled in the type
// so the renderer can never accidentally print 0 for a kind the harness simply
// does not load. Expected still carries the true command count on such a cell
// (it is used to explain codex's commands-as-skills rollup), but NotApplicable
// — not Expected == 0 — drives the rendering.
type KindCount struct {
	Expected      int
	Actual        int
	NotApplicable bool
}

// TargetInventory is one row of the installed-harness-target table: a target,
// its native root instruction file, and the expected/actual counts for each
// content kind.
type TargetInventory struct {
	Target   Target
	RootFile string
	Agents   KindCount
	Commands KindCount
	Skills   KindCount
	// Missing lists canonical generated artifacts absent from their native
	// harness destinations. Counts alone are insufficient: one stale or user
	// artifact can numerically mask a missing required workflow.
	Missing []string
}

// inventoryTargets is the canonical target sweep order (matching
// targetLayouts / PreviouslyInstalledTargets) used for on-disk detection so
// detected rows come out deterministically.
var inventoryTargets = []Target{
	TargetClaude, TargetCodex, TargetOpenCode, TargetCursor, TargetCopilot, TargetGeneric, TargetGrok, TargetDeepSeek,
}

// Inventory returns one TargetInventory per installed harness target in the
// project at projectRoot, for the given domain (empty resolves to
// "engineering"). Expected counts come from the running binary's embedded
// content FS via EnumerateContent, so they equal what `hero install`
// materializes; actual counts come from the files on disk at each target's
// real destination paths.
//
// The row set is the union of PreviouslyInstalledTargets (persisted, machine-
// local, gitignored) and on-disk detection, so a fresh clone still lists
// targets and a persisted target whose tree is missing renders as a flagged
// 0/N row rather than vanishing.
func Inventory(projectRoot, domain string) ([]TargetInventory, error) {
	if domain == "" {
		domain = "engineering"
	}
	domainFS, err := hero.DomainFS(domain)
	if err != nil {
		return nil, fmt.Errorf("resolving domain %q content: %w", domain, err)
	}
	return inventoryFromFS(projectRoot, hero.OverlayFS(domainFS, hero.CoreFS()), domain)
}

// InventoryForTargets returns install inventory for the exact target set the
// caller intends to reconcile. Upgrade uses this before its version no-op so a
// missing generated artifact can trigger a same-version repair. Keeping the
// content FS explicit ensures the check measures the same composed content the
// subsequent install run will materialize.
func InventoryForTargets(projectRoot string, contentFS fs.FS, domain string, targets []Target) ([]TargetInventory, error) {
	if projectRoot == "" || len(targets) == 0 {
		return nil, nil
	}
	manifest, err := EnumerateContent(contentFS, domain)
	if err != nil {
		return nil, fmt.Errorf("enumerating canonical install set: %w", err)
	}
	unique := UnionTargets(targets)
	out := make([]TargetInventory, 0, len(unique))
	for _, target := range unique {
		out = append(out, buildTargetInventory(target, projectRoot, manifest))
	}
	return out, nil
}

// Incomplete reports whether any applicable generated content kind is short
// of what the current content manifest would install. Extra files are not a
// shortfall; normal install pruning handles obsolete generated artifacts.
func (inv TargetInventory) Incomplete() bool {
	return len(inv.Missing) > 0 || kindCountIncomplete(inv.Agents) || kindCountIncomplete(inv.Commands) || kindCountIncomplete(inv.Skills)
}

func kindCountIncomplete(count KindCount) bool {
	return !count.NotApplicable && count.Actual < count.Expected
}

// inventoryFromFS is the testable core of Inventory: it takes the content FS
// explicitly rather than resolving the embedded one, so tests can drive it
// from a seeded content tree.
func inventoryFromFS(projectRoot string, contentFS fs.FS, domain string) ([]TargetInventory, error) {
	if projectRoot == "" {
		return nil, nil
	}
	manifest, err := EnumerateContent(contentFS, domain)
	if err != nil {
		return nil, fmt.Errorf("enumerating canonical install set: %w", err)
	}
	targets := UnionTargets(PreviouslyInstalledTargets(projectRoot), detectedTargets(projectRoot))
	var out []TargetInventory
	for _, t := range targets {
		out = append(out, buildTargetInventory(t, projectRoot, manifest))
	}
	return out, nil
}

// buildTargetInventory fills one row: expected counts derived per-target from
// the manifest, actual counts read from the target's real dest paths.
func buildTargetInventory(t Target, projectRoot string, m ContentManifest) TargetInventory {
	agentsPath, commandsPath, skillsPath := targetInstallPaths(t, projectRoot)
	inv := TargetInventory{
		Target:   t,
		RootFile: nativeInstructionFile(t),
		Agents:   KindCount{Expected: len(m.Agents), Actual: countInstalled(agentsPath)},
	}
	if t == TargetCodex || t == TargetGrok || t == TargetDeepSeek {
		// Codex and Grok have no Hero-owned command loader — commands install
		// as command-* skills under their respective native skill roots.
		// Commands are NotApplicable and skills rolls both sets together.
		inv.Commands = KindCount{Expected: len(m.Commands), NotApplicable: true}
		inv.Skills = KindCount{Expected: len(m.Skills) + len(m.Commands), Actual: countInstalled(skillsPath)}
	} else {
		inv.Commands = KindCount{Expected: len(m.Commands), Actual: countInstalled(commandsPath)}
		inv.Skills = KindCount{Expected: len(m.Skills), Actual: countInstalled(skillsPath)}
	}
	if t == TargetDeepSeek {
		inv.Agents.NotApplicable = true
		inv.Skills.Expected += len(m.Agents)
	}
	inv.Missing = missingGeneratedArtifacts(t, projectRoot, m)
	return inv
}

func missingGeneratedArtifacts(target Target, projectRoot string, manifest ContentManifest) []string {
	var expected []string
	addFlat := func(base string, names []string, suffix string) {
		for _, name := range names {
			expected = append(expected, filepath.Join(base, name+suffix))
		}
	}
	addNested := func(base string, names []string, prefix string) {
		for _, name := range names {
			expected = append(expected, filepath.Join(base, prefix+name, "SKILL.md"))
		}
	}

	switch target {
	case TargetDeepSeek:
		base := filepath.Join(projectRoot, ".dsh")
		addNested(filepath.Join(base, "skills"), manifest.Skills, "")
		addNested(filepath.Join(base, "skills"), manifest.Commands, commandSkillPrefix)
		addNested(filepath.Join(base, "skills"), manifest.Agents, roleSkillPrefix)
		expected = append(expected, filepath.Join(base, deepseekOverlayName))
	case TargetClaude:
		base := filepath.Join(projectRoot, ".claude")
		addFlat(filepath.Join(base, "agents"), manifest.Agents, ".md")
		addFlat(filepath.Join(base, "commands"), manifest.Commands, ".md")
		addNested(filepath.Join(base, "skills"), manifest.Skills, "")
	case TargetOpenCode:
		base := filepath.Join(projectRoot, ".opencode")
		addFlat(filepath.Join(base, "agents"), manifest.Agents, ".md")
		addFlat(filepath.Join(base, "commands"), manifest.Commands, ".md")
		addNested(filepath.Join(base, "skills"), manifest.Skills, "")
	case TargetCursor:
		base := filepath.Join(projectRoot, ".cursor", "rules")
		addFlat(filepath.Join(base, "agents"), manifest.Agents, ".md")
		addFlat(filepath.Join(base, "commands"), manifest.Commands, ".md")
		addFlat(filepath.Join(base, "skills"), manifest.Skills, ".md")
	case TargetCodex:
		addFlat(filepath.Join(projectRoot, ".codex", "agents"), manifest.Agents, ".toml")
		skillsBase := filepath.Join(projectRoot, ".agents", "skills")
		addNested(skillsBase, manifest.Skills, "")
		addNested(skillsBase, manifest.Commands, commandSkillPrefix)
	case TargetCopilot:
		prompts := filepath.Join(projectRoot, ".github", "prompts")
		addFlat(filepath.Join(prompts, "agents"), manifest.Agents, ".prompt.md")
		addFlat(filepath.Join(prompts, "commands"), manifest.Commands, ".prompt.md")
		addNested(filepath.Join(projectRoot, ".github", "skills"), manifest.Skills, "")
	case TargetGeneric:
		base := filepath.Join(projectRoot, ".ai")
		addFlat(filepath.Join(base, "agents"), manifest.Agents, ".md")
		addFlat(filepath.Join(base, "commands"), manifest.Commands, ".md")
		addNested(filepath.Join(base, "skills"), manifest.Skills, "")
	case TargetGrok:
		base := filepath.Join(projectRoot, ".grok")
		addFlat(filepath.Join(base, "agents"), manifest.Agents, ".md")
		addNested(filepath.Join(base, "skills"), manifest.Skills, "")
		addNested(filepath.Join(base, "skills"), manifest.Commands, commandSkillPrefix)
	}

	missing := make([]string, 0)
	for _, path := range expected {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			rel, relErr := filepath.Rel(projectRoot, path)
			if relErr != nil {
				rel = path
			}
			missing = append(missing, filepath.ToSlash(rel))
		}
	}
	return missing
}

// countMode names how a destination directory's installed content is counted.
type countMode int

const (
	countFlatMD       countMode = iota // *.md files, directory READMEs excluded
	countFlatTOML                      // *.toml files (codex agents)
	countFlatPromptMD                  // *.prompt.md files (copilot agents/commands)
	countNestedSkill                   // <name>/SKILL.md directories
)

// kindPath pairs a destination directory with how to count its content. A
// zero-value kindPath (empty dir) counts as 0 — used for codex commands, which
// has no destination.
type kindPath struct {
	dir  string
	mode countMode
}

// targetInstallPaths returns the per-kind destination directory and counting
// mode for a target, derived from the per-target installer functions in
// target_*.go. Keep these in lockstep with those functions.
func targetInstallPaths(t Target, root string) (agents, commands, skills kindPath) {
	switch t {
	case TargetDeepSeek:
		return kindPath{}, kindPath{}, kindPath{filepath.Join(root, ".dsh", "skills"), countNestedSkill}
	case TargetClaude:
		base := filepath.Join(root, ".claude")
		return kindPath{filepath.Join(base, "agents"), countFlatMD},
			kindPath{filepath.Join(base, "commands"), countFlatMD},
			kindPath{filepath.Join(base, "skills"), countNestedSkill}
	case TargetOpenCode:
		base := filepath.Join(root, ".opencode")
		return kindPath{filepath.Join(base, "agents"), countFlatMD},
			kindPath{filepath.Join(base, "commands"), countFlatMD},
			kindPath{filepath.Join(base, "skills"), countNestedSkill}
	case TargetCursor:
		base := filepath.Join(root, ".cursor", "rules")
		// Cursor skills are flat <name>.md files (installSkillsFlat), not
		// nested SKILL.md dirs.
		return kindPath{filepath.Join(base, "agents"), countFlatMD},
			kindPath{filepath.Join(base, "commands"), countFlatMD},
			kindPath{filepath.Join(base, "skills"), countFlatMD}
	case TargetCodex:
		// Codex agents are TOML at .codex/agents; skills (and commands-as-
		// skills) live at .agents/skills; there is no command destination.
		return kindPath{filepath.Join(root, ".codex", "agents"), countFlatTOML},
			kindPath{},
			kindPath{filepath.Join(root, ".agents", "skills"), countNestedSkill}
	case TargetCopilot:
		prompts := filepath.Join(root, ".github", "prompts")
		return kindPath{filepath.Join(prompts, "agents"), countFlatPromptMD},
			kindPath{filepath.Join(prompts, "commands"), countFlatPromptMD},
			kindPath{filepath.Join(root, ".github", "skills"), countNestedSkill}
	case TargetGeneric:
		base := filepath.Join(root, ".ai")
		return kindPath{filepath.Join(base, "agents"), countFlatMD},
			kindPath{filepath.Join(base, "commands"), countFlatMD},
			kindPath{filepath.Join(base, "skills"), countNestedSkill}
	case TargetGrok:
		base := filepath.Join(root, ".grok")
		return kindPath{filepath.Join(base, "agents"), countFlatMD},
			kindPath{},
			kindPath{filepath.Join(base, "skills"), countNestedSkill}
	}
	return kindPath{}, kindPath{}, kindPath{}
}

// countInstalled counts the installed content at a destination per its mode.
// An empty or missing directory counts as 0.
func countInstalled(kp kindPath) int {
	if kp.dir == "" {
		return 0
	}
	if kp.mode == countNestedSkill {
		return countNestedSkillDirs(kp.dir)
	}
	entries, err := os.ReadDir(kp.dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		switch kp.mode {
		case countFlatTOML:
			if strings.HasSuffix(name, ".toml") {
				count++
			}
		case countFlatPromptMD:
			if strings.HasSuffix(name, ".prompt.md") {
				count++
			}
		default: // countFlatMD
			if strings.HasSuffix(name, ".md") && !isContentReadme(name) {
				count++
			}
		}
	}
	return count
}

// countNestedSkillDirs counts subdirectories of dir that contain a SKILL.md —
// the Anthropic nested-skill layout install writes.
func countNestedSkillDirs(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "SKILL.md")); err == nil {
			count++
		}
	}
	return count
}

// detectedTargets returns the targets whose install is present on disk, in
// canonical sweep order.
func detectedTargets(projectRoot string) []Target {
	var out []Target
	for _, t := range inventoryTargets {
		if targetInstalledOnDisk(t, projectRoot) {
			out = append(out, t)
		}
	}
	return out
}

// targetInstalledOnDisk reports whether a target's install is present on disk,
// using the same destination paths inventory counts from — so detection and
// counting can never disagree. Copilot additionally recognizes its file
// marker (.github/copilot-instructions.md), because copilot cannot be inferred
// through the shared targetLayouts registry, which probes the legacy
// .github/copilot/ directory the modern install deletes.
func targetInstalledOnDisk(t Target, projectRoot string) bool {
	agents, commands, skills := targetInstallPaths(t, projectRoot)
	for _, kp := range []kindPath{agents, commands, skills} {
		if kp.dir == "" {
			continue
		}
		if info, err := os.Stat(kp.dir); err == nil && info.IsDir() {
			return true
		}
	}
	if t == TargetCopilot {
		marker := filepath.Join(projectRoot, ".github", "copilot-instructions.md")
		if info, err := os.Stat(marker); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}
