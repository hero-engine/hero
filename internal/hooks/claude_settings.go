// claude_settings.go — read, mutate, and write `.claude/settings.json`
// host-tool hook entries for Hero's compact-handoff integration.
//
// Design constraints (from next-compact-handoff spec):
//
//   - Preserve any pre-existing user-authored entries verbatim. We
//     parse the file into a permissive map[string]any, mutate only the
//     specific hook arrays we care about, and re-serialize.
//   - Idempotent install: re-running install is a no-op when our entry
//     is already present. We tag every Hero-installed hook entry with
//     `"added_by_hero": true` so we can identify ours unambiguously on
//     subsequent reads without comment-parsing.
//   - Uninstall removes only Hero-marked entries. User entries survive.
package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// HeroMarkerField is the JSON object key Hero adds to every hook
// entry it installs into `.claude/settings.json`. Presence of this
// field (regardless of value) marks the entry as Hero-managed and
// safe to remove on uninstall.
const HeroMarkerField = "added_by_hero"

// ClaudeSettingsPath returns the conventional path to Claude Code's
// per-project settings file inside projectRoot.
func ClaudeSettingsPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".claude", "settings.json")
}

// ClaudeHookSpec describes a single hook entry Hero wants installed
// into a SessionStart / PreCompact / Stop array.
type ClaudeHookSpec struct {
	// Matcher narrows when the hook fires (e.g. "compact" for
	// SessionStart). Empty matcher matches all events of that name.
	Matcher string
	// Command is the shell command to run.
	Command string
}

// InstallClaudeCompactHandoff installs the SessionStart{matcher:"compact"}
// hook that wires `hero next compact-handoff --json` into Claude Code's
// post-compaction context-injection flow.
//
// Idempotent: returns (false, nil) when the Hero-marked entry is
// already present. Returns (true, nil) on a fresh install. Preserves
// any other hook entries the user has authored.
func InstallClaudeCompactHandoff(projectRoot string) (bool, error) {
	return installClaudeHook(projectRoot, "SessionStart", ClaudeHookSpec{
		Matcher: "compact",
		Command: "hero next compact-handoff --json",
	})
}

// UninstallClaudeCompactHandoff removes the SessionStart{matcher:"compact"}
// Hero entry. Returns (true, nil) if a removal happened, (false, nil)
// when nothing matched. Preserves user-authored entries in the same
// hook array.
func UninstallClaudeCompactHandoff(projectRoot string) (bool, error) {
	return uninstallClaudeHook(projectRoot, "SessionStart", "compact")
}

// ClaudeCompactHandoffStatus returns true when a Hero-marked SessionStart
// {compact} entry is present in `.claude/settings.json`. Returns false
// (no error) when the file doesn't exist or doesn't contain the entry.
func ClaudeCompactHandoffStatus(projectRoot string) (bool, error) {
	settings, err := readClaudeSettings(ClaudeSettingsPath(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	entries := claudeHookEntries(settings, "SessionStart")
	for _, e := range entries {
		if isHeroEntry(e) && matcherOf(e) == "compact" {
			return true, nil
		}
	}
	return false, nil
}

// installClaudeHook is the shared install path. Reads (or initialises)
// settings.json, inserts a Hero-marked entry into the named hook array
// if no existing entry with the same matcher + command already exists,
// and writes the result.
func installClaudeHook(projectRoot, hookEvent string, spec ClaudeHookSpec) (bool, error) {
	path := ClaudeSettingsPath(projectRoot)
	settings, err := readClaudeSettings(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if settings == nil {
		settings = map[string]any{}
	}

	hooks := getOrCreateMap(settings, "hooks")
	entries := claudeHookEntries(settings, hookEvent)

	// Idempotency check: if any existing entry (Hero-marked or not)
	// shares our matcher AND points at our command, treat as installed.
	for _, e := range entries {
		if matcherOf(e) != spec.Matcher {
			continue
		}
		if containsCommand(e, spec.Command) {
			return false, nil
		}
	}

	// Build the new entry. Stable shape matches Claude Code's
	// documented hook payload format.
	newEntry := map[string]any{
		"matcher":       spec.Matcher,
		HeroMarkerField: true,
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": spec.Command,
			},
		},
	}
	entries = append(entries, newEntry)
	hooks[hookEvent] = entries

	return true, writeClaudeSettings(path, settings)
}

// uninstallClaudeHook removes Hero-marked entries from the named hook
// array whose matcher matches. Returns whether any entry was removed.
func uninstallClaudeHook(projectRoot, hookEvent, matcher string) (bool, error) {
	path := ClaudeSettingsPath(projectRoot)
	settings, err := readClaudeSettings(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		return false, nil
	}
	entries := claudeHookEntries(settings, hookEvent)
	removed := false
	var kept []any
	for _, e := range entries {
		if isHeroEntry(e) && matcherOf(e) == matcher {
			removed = true
			continue
		}
		kept = append(kept, e)
	}
	if !removed {
		return false, nil
	}
	if len(kept) == 0 {
		delete(hooks, hookEvent)
		if len(hooks) == 0 {
			delete(settings, "hooks")
		}
	} else {
		hooks[hookEvent] = kept
	}
	return true, writeClaudeSettings(path, settings)
}

// readClaudeSettings parses .claude/settings.json into a permissive
// map. Returns os.ErrNotExist (wrapped via os.IsNotExist) when the
// file is missing so callers can treat that as "fresh install."
func readClaudeSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if settings == nil {
		settings = map[string]any{}
	}
	return settings, nil
}

// writeClaudeSettings writes settings.json with 2-space indentation
// matching Claude Code's own convention. Creates parent directories
// as needed.
func writeClaudeSettings(path string, settings map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	// Trailing newline keeps editors happy and matches Claude Code's
	// own output.
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}

// claudeHookEntries returns the array of hook entries under
// settings.hooks.<hookEvent>, or nil when the path doesn't exist.
func claudeHookEntries(settings map[string]any, hookEvent string) []any {
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		return nil
	}
	raw, _ := hooks[hookEvent].([]any)
	return raw
}

// getOrCreateMap fetches m[key] as a map, creating an empty one (and
// storing it in m) if missing or of the wrong type.
func getOrCreateMap(m map[string]any, key string) map[string]any {
	if existing, ok := m[key].(map[string]any); ok {
		return existing
	}
	created := map[string]any{}
	m[key] = created
	return created
}

func isHeroEntry(entry any) bool {
	e, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	v, ok := e[HeroMarkerField]
	if !ok {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return v != nil
}

func matcherOf(entry any) string {
	e, ok := entry.(map[string]any)
	if !ok {
		return ""
	}
	s, _ := e["matcher"].(string)
	return s
}

func containsCommand(entry any, cmd string) bool {
	e, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	hooks, _ := e["hooks"].([]any)
	for _, h := range hooks {
		hm, ok := h.(map[string]any)
		if !ok {
			continue
		}
		if s, _ := hm["command"].(string); s == cmd {
			return true
		}
	}
	return false
}
