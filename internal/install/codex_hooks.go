package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Codex CLI hooks.json schema for the Stop event:
//
//	{
//	  "hooks": {
//	    "Stop":         [ {"hooks": [ {"type": "command", "command": "..."} ]} ],
//	    "SessionStart": [ {"hooks": [ {"type": "command", "command": "..."} ]} ]
//	  }
//	}
//
// Unlike Claude Code, Codex hook entries have no "matcher" field.
// Hero-managed entries are identified by the command string matching
// one of the heroCmdPrefixes (defined in claude_hooks.go) — same set
// applies on both targets, so the SessionStart round-trip-ingest hook
// stays in sync across both surfaces.

// codexHookEvents is the parallel of claudeHookEvents for Codex.
// Codex documents support for both Stop and SessionStart in its
// hook schema; if a Codex client ignores SessionStart, the entry is
// harmlessly inert.
var codexHookEvents = []string{"Stop", "SessionStart"}

// codexHookCommandFor returns the canonical hero command for a
// Codex hook event. Mirror of claudeHookCommandFor.
func codexHookCommandFor(event string) string {
	switch event {
	case "SessionStart":
		return heroIngestCmd
	default:
		return heroCheckpointCmd
	}
}

// wireCodexHooks merges hero-managed Stop and SessionStart hooks
// into .codex/hooks.json. Idempotent: existing hero entries are
// replaced before writing new ones.
func wireCodexHooks(opts Options, result *Result) error {
	hooksPath, err := codexHooksPath(opts)
	if err != nil {
		return err
	}

	if opts.DryRun {
		progressf(opts, "  hooks.json -> %s (wire Stop + SessionStart hooks)\n", hooksPath)
		return nil
	}

	hooks := map[string]interface{}{}
	if data, readErr := os.ReadFile(hooksPath); readErr == nil {
		outer := map[string]interface{}{}
		if jsonErr := json.Unmarshal(data, &outer); jsonErr == nil {
			if h, ok := outer["hooks"].(map[string]interface{}); ok {
				hooks = h
			}
		}
	} else if !os.IsNotExist(readErr) {
		return fmt.Errorf("reading %s: %w", hooksPath, readErr)
	}

	for _, event := range codexHookEvents {
		hooks[event] = upsertCodexHeroEntry(hooks[event], codexHookCommandFor(event))
	}

	out, err := json.MarshalIndent(map[string]interface{}{"hooks": hooks}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling hooks: %w", err)
	}
	out = append(out, '\n')

	if err := os.MkdirAll(filepath.Dir(hooksPath), 0o755); err != nil {
		return fmt.Errorf("creating dir for %s: %w", hooksPath, err)
	}
	if err := os.WriteFile(hooksPath, out, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", hooksPath, err)
	}

	result.Merged = append(result.Merged, hooksPath)
	return nil
}

// UnwireCodexHooks removes the hero-managed Stop entry from .codex/hooks.json.
func UnwireCodexHooks(opts Options) error {
	hooksPath, err := codexHooksPath(opts)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(hooksPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	outer := map[string]interface{}{}
	if jsonErr := json.Unmarshal(data, &outer); jsonErr != nil {
		return fmt.Errorf("parsing %s: %w", hooksPath, jsonErr)
	}

	hooks, _ := outer["hooks"].(map[string]interface{})
	if hooks == nil {
		return nil
	}

	for _, event := range codexHookEvents {
		entries, _ := hooks[event].([]interface{})
		stripped := stripCodexHeroEntries(entries)
		if len(stripped) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = stripped
		}
	}

	if len(hooks) == 0 {
		delete(outer, "hooks")
	} else {
		outer["hooks"] = hooks
	}

	out, err := json.MarshalIndent(outer, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(hooksPath, out, 0o644)
}

// upsertCodexHeroEntry returns a fresh entry list: existing hero entries removed,
// one fresh entry prepended. User entries are preserved.
func upsertCodexHeroEntry(existing interface{}, command string) []interface{} {
	entries, _ := existing.([]interface{})
	stripped := stripCodexHeroEntries(entries)
	hero := map[string]interface{}{
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": command,
			},
		},
	}
	return append([]interface{}{hero}, stripped...)
}

// stripCodexHeroEntries removes entries whose inner hooks contain a
// hero-managed command (matched by heroCmdPrefixes).
func stripCodexHeroEntries(entries []interface{}) []interface{} {
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

// codexHooksPath returns the path to .codex/hooks.json based on install mode.
func codexHooksPath(opts Options) (string, error) {
	switch opts.Mode {
	case ModeProject:
		if opts.TargetDir == "" {
			return "", fmt.Errorf("project mode requires a target directory")
		}
		return filepath.Join(opts.TargetDir, ".codex", "hooks.json"), nil
	case ModeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".codex", "hooks.json"), nil
	default:
		return "", fmt.Errorf("unknown mode: %s", opts.Mode)
	}
}
