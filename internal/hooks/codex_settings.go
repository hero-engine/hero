// codex_settings.go — Codex CLI host-tool hook installer.
//
// Mirrors claude_settings.go but writes to the project-local Codex
// hooks file: `<projectRoot>/.codex/hooks.json`. The JSON shape is
// effectively identical to Claude Code's `hooks` block, plus an
// optional `timeout` field on each command entry which we set to 30s.
//
// Design constraints (from next-compact-handoff spec, identical to
// Claude installer):
//
//   - Preserve any pre-existing user-authored entries verbatim.
//   - Idempotent install: re-running install is a no-op when our
//     entry is already present (same `command` + Hero marker).
//   - Uninstall removes only Hero-marked entries. User entries survive.
//
// Feature-flag note: Codex's hooks system is off by default. The
// user must opt in via `~/.codex/config.toml`:
//
//	[features]
//	codex_hooks = true
//
// We do NOT mutate `~/.codex/config.toml`. On install we READ it
// (naive line scan, no TOML parser) and surface a warning via the
// returned `CodexInstallNotice` when the flag isn't set. The install
// itself still succeeds — the hook just won't fire until the user
// flips the flag.
//
// Trust prompt: Codex prompts the user to "trust" project-local
// config on first run. Hero does not handle this; it's informational.
//
// Source references (verified May 2026):
//   - Schema input:  https://github.com/openai/codex/blob/main/codex-rs/hooks/schema/generated/session-start.command.input.schema.json
//   - Schema output: https://github.com/openai/codex/blob/main/codex-rs/hooks/schema/generated/session-start.command.output.schema.json
//   - SessionStartSource enum: https://github.com/openai/codex/blob/main/codex-rs/hooks/src/events/session_start.rs
package hooks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// codexCommandTimeoutSeconds is the timeout we set on every Hero-
// installed Codex hook command. Generous enough that a slow graph
// query doesn't abort; short enough that a hung Hero process doesn't
// stall the user's session indefinitely.
const codexCommandTimeoutSeconds = 30

// CodexSettingsPath returns the conventional path to Codex's
// per-project hooks file inside projectRoot.
func CodexSettingsPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".codex", "hooks.json")
}

// CodexCompactHandoffSupported reports whether the Hero binary knows
// how to wire the compact-handoff hook on Codex's side. True now that
// the Codex hooks schema is verified and the installer is implemented.
func CodexCompactHandoffSupported() bool {
	return true
}

// InstallCodexCompactHandoff installs the SessionStart{matcher:"compact"}
// hook into `<projectRoot>/.codex/hooks.json`.
//
// Returns (false, nil) if the Hero entry is already present. Returns
// (true, nil) on a fresh install. Preserves any other hook entries
// the user has authored.
//
// Independent of the install result, callers should consult
// CodexFeatureFlagEnabled to decide whether to print the
// "enable codex_hooks" warning. We deliberately keep that check
// out-of-band so install/uninstall logic stays pure.
func InstallCodexCompactHandoff(projectRoot string) (bool, error) {
	return installCodexHook(projectRoot, "SessionStart", ClaudeHookSpec{
		Matcher: "compact",
		Command: "hero next compact-handoff --json",
	})
}

// UninstallCodexCompactHandoff removes the Hero-marked
// SessionStart{matcher:"compact"} entry. Returns (true, nil) if a
// removal happened, (false, nil) when nothing matched. Preserves
// user-authored entries in the same hook array. Cleans up empty
// containers (matcher entry → SessionStart array → hooks object →
// the file itself, and the `.codex/` dir if empty).
func UninstallCodexCompactHandoff(projectRoot string) (bool, error) {
	return uninstallCodexHook(projectRoot, "SessionStart", "compact")
}

// CodexCompactHandoffStatus returns true when a Hero-marked
// SessionStart{compact} entry is present in `.codex/hooks.json`.
// Returns false (no error) when the file doesn't exist or doesn't
// contain the entry.
func CodexCompactHandoffStatus(projectRoot string) (bool, error) {
	settings, err := readCodexSettings(CodexSettingsPath(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	entries := claudeHookEntries(settings, "SessionStart")
	for _, e := range entries {
		if matcherOf(e) != "compact" {
			continue
		}
		if codexEntryHasHeroHook(e) {
			return true, nil
		}
	}
	return false, nil
}

// CodexFeatureFlagEnabled reports whether the user's global
// `~/.codex/config.toml` contains `codex_hooks = true` under any
// section. Naive line-scan — no TOML parser. Returns false if the
// file doesn't exist or can't be read. Errors are swallowed because
// "flag not detected" is an acceptable degraded result (the caller
// will print the warning either way).
func CodexFeatureFlagEnabled() bool {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return false
	}
	path := filepath.Join(home, ".codex", "config.toml")
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip inline comments. TOML uses `#`; we accept anything
		// before the first `#` not inside a quoted string. The flag
		// value is a bare bool so quotes don't apply.
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if !strings.HasPrefix(line, "codex_hooks") {
			continue
		}
		// Match `codex_hooks = true` (any whitespace, any case for
		// the value to be lenient).
		rest := strings.TrimSpace(strings.TrimPrefix(line, "codex_hooks"))
		if !strings.HasPrefix(rest, "=") {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(rest, "="))
		if strings.EqualFold(val, "true") {
			return true
		}
	}
	return false
}

// installCodexHook is the shared install path. Reads (or initialises)
// hooks.json, inserts a Hero-marked hook entry into the named hook
// array's matcher, and writes the result.
//
// The Codex/Claude shape: each matcher entry's `hooks` array contains
// individual command objects. The `added_by_hero` marker lives on the
// *command* entry (not the matcher entry), so that user entries
// sharing the same matcher survive uninstall cleanly.
func installCodexHook(projectRoot, hookEvent string, spec ClaudeHookSpec) (bool, error) {
	path := CodexSettingsPath(projectRoot)
	settings, err := readCodexSettings(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if settings == nil {
		settings = map[string]any{}
	}

	hooks := getOrCreateMap(settings, "hooks")
	entries := claudeHookEntries(settings, hookEvent)

	// Find an existing matcher entry to attach to, if any. We attach
	// alongside user entries rather than creating a parallel matcher
	// entry — that mirrors how Claude Code parses identical matchers.
	matcherIdx := -1
	for i, e := range entries {
		if matcherOf(e) == spec.Matcher {
			matcherIdx = i
			break
		}
	}

	heroCmd := map[string]any{
		"type":          "command",
		"command":       spec.Command,
		"timeout":       codexCommandTimeoutSeconds,
		HeroMarkerField: true,
	}

	if matcherIdx >= 0 {
		entry, _ := entries[matcherIdx].(map[string]any)
		if entry == nil {
			entry = map[string]any{"matcher": spec.Matcher}
			entries[matcherIdx] = entry
		}
		inner, _ := entry["hooks"].([]any)
		// Idempotency: if a hero-marked entry with the same command is
		// already present, no-op.
		for _, h := range inner {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if isHeroEntry(hm) && cmdOf(hm) == spec.Command {
				return false, nil
			}
		}
		inner = append(inner, heroCmd)
		entry["hooks"] = inner
		hooks[hookEvent] = entries
		return true, writeCodexSettings(path, settings)
	}

	// Create a new matcher entry.
	newEntry := map[string]any{
		"matcher": spec.Matcher,
		"hooks":   []any{heroCmd},
	}
	entries = append(entries, newEntry)
	hooks[hookEvent] = entries
	return true, writeCodexSettings(path, settings)
}

// uninstallCodexHook removes Hero-marked command entries from the
// matcher group inside the named hook array. Empty containers are
// pruned: matcher entry → SessionStart array → hooks object → file
// (and the `.codex/` dir when also empty).
func uninstallCodexHook(projectRoot, hookEvent, matcher string) (bool, error) {
	path := CodexSettingsPath(projectRoot)
	settings, err := readCodexSettings(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	hooksMap, _ := settings["hooks"].(map[string]any)
	if hooksMap == nil {
		return false, nil
	}
	entries := claudeHookEntries(settings, hookEvent)
	removed := false
	var keptEntries []any
	for _, e := range entries {
		em, ok := e.(map[string]any)
		if !ok || matcherOf(em) != matcher {
			keptEntries = append(keptEntries, e)
			continue
		}
		inner, _ := em["hooks"].([]any)
		var keptInner []any
		for _, h := range inner {
			hm, ok := h.(map[string]any)
			if !ok {
				keptInner = append(keptInner, h)
				continue
			}
			if isHeroEntry(hm) {
				removed = true
				continue
			}
			keptInner = append(keptInner, h)
		}
		if len(keptInner) == 0 {
			// Drop the matcher entry entirely.
			continue
		}
		em["hooks"] = keptInner
		keptEntries = append(keptEntries, em)
	}
	if !removed {
		return false, nil
	}
	if len(keptEntries) == 0 {
		delete(hooksMap, hookEvent)
		if len(hooksMap) == 0 {
			delete(settings, "hooks")
		}
	} else {
		hooksMap[hookEvent] = keptEntries
	}
	if len(settings) == 0 {
		// Empty JSON object — remove the file (and dir if empty).
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return true, err
		}
		_ = os.Remove(filepath.Dir(path)) // best-effort; fails harmlessly when not empty
		return true, nil
	}
	return true, writeCodexSettings(path, settings)
}

// readCodexSettings parses `.codex/hooks.json` into a permissive map.
// Returns os.ErrNotExist (wrapped via os.IsNotExist) when the file is
// missing so callers can treat that as "fresh install."
func readCodexSettings(path string) (map[string]any, error) {
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

// writeCodexSettings writes hooks.json with 2-space indentation.
// Creates parent directories as needed.
func writeCodexSettings(path string, settings map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}

// codexEntryHasHeroHook reports whether the given matcher entry's
// `hooks` array contains at least one Hero-marked command entry.
func codexEntryHasHeroHook(entry any) bool {
	em, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	inner, _ := em["hooks"].([]any)
	for _, h := range inner {
		hm, ok := h.(map[string]any)
		if !ok {
			continue
		}
		if isHeroEntry(hm) {
			return true
		}
	}
	return false
}

func cmdOf(entry map[string]any) string {
	s, _ := entry["command"].(string)
	return s
}
