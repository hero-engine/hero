package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Codex CLI hooks.json schema for the Stop event:
//
//	{
//	  "hooks": {
//	    "Stop": [ {"hooks": [ {"type": "command", "command": "..."} ]} ]
//	  }
//	}
//
// Unlike Claude Code, Codex hook entries have no "matcher" field.
// Hero-managed entries are identified by the command string starting
// with "hero next checkpoint".

// wireCodexHooks merges a hero-managed Stop hook into .codex/hooks.json.
// Idempotent: existing hero entries are replaced before writing new ones.
func wireCodexHooks(opts Options, result *Result) error {
	hooksPath, err := codexHooksPath(opts)
	if err != nil {
		return err
	}

	if opts.DryRun {
		progressf(opts, "  hooks.json -> %s (wire Stop hook)\n", hooksPath)
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

	hooks["Stop"] = upsertCodexHeroEntry(hooks["Stop"])

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

	entries, _ := hooks["Stop"].([]interface{})
	stripped := stripCodexHeroEntries(entries)
	if len(stripped) == 0 {
		delete(hooks, "Stop")
	} else {
		hooks["Stop"] = stripped
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
func upsertCodexHeroEntry(existing interface{}) []interface{} {
	entries, _ := existing.([]interface{})
	stripped := stripCodexHeroEntries(entries)
	hero := map[string]interface{}{
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": heroCheckpointCmd,
			},
		},
	}
	return append([]interface{}{hero}, stripped...)
}

// stripCodexHeroEntries removes entries whose inner hooks contain a
// "hero next checkpoint" command.
func stripCodexHeroEntries(entries []interface{}) []interface{} {
	var out []interface{}
	for _, e := range entries {
		em, ok := e.(map[string]interface{})
		if !ok {
			out = append(out, e)
			continue
		}
		if codexEntryIsHero(em) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func codexEntryIsHero(entry map[string]interface{}) bool {
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
