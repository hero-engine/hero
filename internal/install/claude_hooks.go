package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Claude Code's settings.json schema for the hook events we manage:
//
//   {
//     "hooks": {
//       "Stop":       [ {"matcher": "", "hooks": [ {"type": "command", "command": "..."} ]} ],
//       "PreCompact": [ {"matcher": "", "hooks": [ {"type": "command", "command": "..."} ]} ]
//     }
//   }
//
// We identify hero-managed hook entries by the command string itself —
// any inner hook whose command begins with `hero next checkpoint` is
// considered hero-managed and gets stripped on re-install / uninstall.
const heroCheckpointCmd = "hero next checkpoint --quiet"

// claudeHookEvents are the host-tool events we wire `hero next checkpoint` into.
//
//   Stop       — fires after every assistant turn ends; keeps NEXT.md fresh continuously
//   PreCompact — fires before context compaction; the most dangerous moment for losing state
var claudeHookEvents = []string{"Stop", "PreCompact"}

// heroAllowlistEntry is the permissions.allow entry that tells Claude
// Code to auto-approve any Bash tool call starting with `hero`. Without
// it, Claude prompts on every hero invocation. With it, the user grants
// persistent approval for the command prefix.
const heroAllowlistEntry = "Bash(hero:*)"

// wireClaudeHooks merges hero-managed Stop and PreCompact hooks into
// .claude/settings.json (or ~/.claude/settings.json in global mode).
// Idempotent: existing hero entries are removed before fresh ones are
// added, so re-running install never duplicates.
func wireClaudeHooks(opts Options, result *Result) error {
	settingsPath, err := claudeSettingsPath(opts)
	if err != nil {
		return err
	}

	if opts.DryRun {
		progressf(opts, "  settings.json -> %s (wire Stop + PreCompact hooks)\n", settingsPath)
		return nil
	}

	settings := map[string]interface{}{}
	if data, readErr := os.ReadFile(settingsPath); readErr == nil {
		if jsonErr := json.Unmarshal(data, &settings); jsonErr != nil {
			return fmt.Errorf("parsing %s: %w", settingsPath, jsonErr)
		}
	} else if !os.IsNotExist(readErr) {
		return fmt.Errorf("reading %s: %w", settingsPath, readErr)
	}

	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = map[string]interface{}{}
	}

	for _, event := range claudeHookEvents {
		hooks[event] = upsertHeroEntry(hooks[event])
	}
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	out = append(out, '\n')

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return fmt.Errorf("creating dir for %s: %w", settingsPath, err)
	}
	if err := os.WriteFile(settingsPath, out, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", settingsPath, err)
	}

	result.Merged = append(result.Merged, settingsPath)
	return nil
}

// wireClaudePermissions ensures `permissions.allow` in
// .claude/settings.json contains `Bash(hero:*)` so Claude Code stops
// prompting on every `hero` invocation. Idempotent: if the entry is
// already present, the file is left untouched and `added` is false.
// Other allowlist entries and unrelated settings keys are preserved.
func wireClaudePermissions(opts Options, result *Result) (added bool, err error) {
	settingsPath, err := claudeSettingsPath(opts)
	if err != nil {
		return false, err
	}

	if opts.DryRun {
		progressf(opts, "  settings.json -> %s (wire Bash(hero:*) permission)\n", settingsPath)
		return false, nil
	}

	settings := map[string]interface{}{}
	if data, readErr := os.ReadFile(settingsPath); readErr == nil {
		if jsonErr := json.Unmarshal(data, &settings); jsonErr != nil {
			return false, fmt.Errorf("parsing %s: %w", settingsPath, jsonErr)
		}
	} else if !os.IsNotExist(readErr) {
		return false, fmt.Errorf("reading %s: %w", settingsPath, readErr)
	}

	permissions, _ := settings["permissions"].(map[string]interface{})
	if permissions == nil {
		permissions = map[string]interface{}{}
	}
	allow, _ := permissions["allow"].([]interface{})

	for _, entry := range allow {
		if s, ok := entry.(string); ok && s == heroAllowlistEntry {
			// Already present — nothing to do.
			return false, nil
		}
	}

	allow = append(allow, heroAllowlistEntry)
	permissions["allow"] = allow
	settings["permissions"] = permissions

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshaling settings: %w", err)
	}
	out = append(out, '\n')

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return false, fmt.Errorf("creating dir for %s: %w", settingsPath, err)
	}
	if err := os.WriteFile(settingsPath, out, 0o644); err != nil {
		return false, fmt.Errorf("writing %s: %w", settingsPath, err)
	}

	if result != nil {
		result.Merged = append(result.Merged, settingsPath)
	}
	return true, nil
}

// EnsureClaudeHeroAllowlist applies the Bash(hero:*) permission to a
// Claude Code settings.json. Used by `hero trust claude` to re-apply
// the entry on demand without going through the full installer.
//
// In ModeProject the entry lands in <projectDir>/.claude/settings.json;
// projectDir must be non-empty.
// In ModeGlobal the entry lands in ~/.claude/settings.json; projectDir
// is ignored.
//
// Returns whether the entry was newly added and the absolute settings
// path it was written to.
func EnsureClaudeHeroAllowlist(mode Mode, projectDir string) (added bool, path string, err error) {
	if mode == ModeProject && projectDir == "" {
		return false, "", fmt.Errorf("project mode requires a target directory")
	}
	opts := Options{Mode: mode, TargetDir: projectDir}
	settingsPath, err := claudeSettingsPath(opts)
	if err != nil {
		return false, "", err
	}
	added, err = wireClaudePermissions(opts, nil)
	return added, settingsPath, err
}

// UnwireClaudeHooks removes hero-managed Stop / PreCompact entries from
// settings.json. If, after removal, a hook event has no remaining
// entries, the event key is deleted. If `hooks` itself ends up empty,
// it's deleted too.
func UnwireClaudeHooks(opts Options) error {
	settingsPath, err := claudeSettingsPath(opts)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	settings := map[string]interface{}{}
	if jsonErr := json.Unmarshal(data, &settings); jsonErr != nil {
		return fmt.Errorf("parsing %s: %w", settingsPath, jsonErr)
	}

	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		return nil
	}

	for _, event := range claudeHookEvents {
		entries, _ := hooks[event].([]interface{})
		stripped := stripHeroEntries(entries)
		if len(stripped) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = stripped
		}
	}

	if len(hooks) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = hooks
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(settingsPath, out, 0o644)
}

// upsertHeroEntry returns a fresh entry list with any prior hero entries
// removed and one fresh hero entry appended at the front. Other entries
// (user's own hooks) are preserved untouched.
func upsertHeroEntry(existing interface{}) []interface{} {
	entries, _ := existing.([]interface{})
	stripped := stripHeroEntries(entries)
	hero := map[string]interface{}{
		"matcher": "",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": heroCheckpointCmd,
			},
		},
	}
	return append([]interface{}{hero}, stripped...)
}

// stripHeroEntries removes any entry whose inner hooks contain a
// `hero next checkpoint` command. Returns the remaining (user-owned)
// entries, in original order.
func stripHeroEntries(entries []interface{}) []interface{} {
	var out []interface{}
	for _, e := range entries {
		em, ok := e.(map[string]interface{})
		if !ok {
			out = append(out, e)
			continue
		}
		if entryIsHero(em) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func entryIsHero(entry map[string]interface{}) bool {
	inner, _ := entry["hooks"].([]interface{})
	for _, h := range inner {
		hm, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		cmd, _ := hm["command"].(string)
		if strings.HasPrefix(cmd, "hero next checkpoint") {
			return true
		}
	}
	return false
}

// claudeSettingsPath returns the path to settings.json based on install mode.
//
//	project mode: <project>/.claude/settings.json
//	global mode:  ~/.claude/settings.json
func claudeSettingsPath(opts Options) (string, error) {
	switch opts.Mode {
	case ModeProject:
		if opts.TargetDir == "" {
			return "", fmt.Errorf("project mode requires a target directory")
		}
		return filepath.Join(opts.TargetDir, ".claude", "settings.json"), nil
	case ModeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".claude", "settings.json"), nil
	default:
		return "", fmt.Errorf("unknown mode: %s", opts.Mode)
	}
}
