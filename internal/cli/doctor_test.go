package cli

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/install"
)

// kc builds a numeric KindCount for the doctor inventory tests.
func kc(actual, expected int) install.KindCount {
	return install.KindCount{Expected: expected, Actual: actual}
}

// healthyClaude / healthyCodex are canonical fully-installed rows.
func healthyClaude() install.TargetInventory {
	return install.TargetInventory{
		Target: install.TargetClaude, RootFile: "CLAUDE.md",
		Agents: kc(35, 35), Commands: kc(29, 29), Skills: kc(55, 55),
	}
}

func healthyCodex() install.TargetInventory {
	return install.TargetInventory{
		Target: install.TargetCodex, RootFile: "AGENTS.md",
		Agents:   kc(35, 35),
		Commands: install.KindCount{Expected: 29, NotApplicable: true},
		Skills:   kc(84, 84),
	}
}

func healthyGrok() install.TargetInventory {
	return install.TargetInventory{
		Target: install.TargetGrok, RootFile: "AGENTS.md",
		Agents: kc(35, 35), Commands: install.KindCount{Expected: 29, NotApplicable: true}, Skills: kc(84, 84),
	}
}

func TestBuildDoctorReport(t *testing.T) {
	base := doctorInfo{
		exe:           "/Users/dev/go/bin/hero",
		exeResolved:   "/Users/dev/go/bin/hero",
		pathHero:      "/Users/dev/go/bin/hero",
		binaryVersion: "0.14.0",
		binarySchema:  "4",
		graphSchema:   "4",
		heroDir:       "/repo/.hero",
	}

	t.Run("reports exe, version, both schemas", func(t *testing.T) {
		report := buildDoctorReport(base)
		for _, want := range []string{
			"/Users/dev/go/bin/hero", // exe path
			"v0.14.0",                // binary version (displayVersion)
			"binary schema:   4",     // binary schema
			"graph schema:    4",     // graph schema
		} {
			if !strings.Contains(report, want) {
				t.Errorf("report missing %q:\n%s", want, report)
			}
		}
	})

	t.Run("agree verdict", func(t *testing.T) {
		report := buildDoctorReport(base)
		if !strings.Contains(report, "Verdict: OK") {
			t.Errorf("expected agree verdict:\n%s", report)
		}
	})

	t.Run("binary older than graph verdict", func(t *testing.T) {
		info := base
		info.binarySchema = "2"
		info.graphSchema = "4"
		report := buildDoctorReport(info)
		if !strings.Contains(report, "OLDER than the workspace graph") {
			t.Errorf("expected binary-older verdict:\n%s", report)
		}
		if !strings.Contains(report, "will NOT help") {
			t.Errorf("binary-older verdict must warn `hero upgrade` won't help:\n%s", report)
		}
	})

	t.Run("binary newer than graph verdict", func(t *testing.T) {
		info := base
		info.binarySchema = "4"
		info.graphSchema = "2"
		report := buildDoctorReport(info)
		if !strings.Contains(report, "NEWER than the workspace graph") {
			t.Errorf("expected binary-newer verdict:\n%s", report)
		}
	})

	t.Run("PATH divergence fires the flag", func(t *testing.T) {
		info := base
		// Simulate a harness resolving `hero` to a different binary than
		// the one running (the Defect-2 signal).
		info.pathHero = "/usr/local/bin/hero"
		info.pathHeroResolved = "/usr/local/bin/hero"
		report := buildDoctorReport(info)
		if !strings.Contains(report, "DIFFERENT binary than the one running") {
			t.Errorf("expected PATH-divergence warning:\n%s", report)
		}
	})

	t.Run("no PATH divergence when same binary", func(t *testing.T) {
		report := buildDoctorReport(base) // pathHero == exe
		if strings.Contains(report, "DIFFERENT binary than the one running") {
			t.Errorf("should not warn when PATH hero matches running binary:\n%s", report)
		}
	})

	t.Run("degrades gracefully outside a workspace", func(t *testing.T) {
		info := base
		info.heroDir = ""
		info.graphSchema = ""
		report := buildDoctorReport(info)
		if !strings.Contains(report, "no hero workspace found") {
			t.Errorf("expected graceful no-workspace message:\n%s", report)
		}
		if !strings.Contains(report, "cannot compare") {
			t.Errorf("expected cannot-compare verdict outside workspace:\n%s", report)
		}
	})

	t.Run("healthy install table renders rows and codex em-dash", func(t *testing.T) {
		info := base
		info.inventory = []install.TargetInventory{healthyClaude(), healthyCodex()}
		report := buildDoctorReport(info)

		for _, want := range []string{
			"Installed harness targets",
			"TARGET", "AGENTS", "COMMANDS", "SKILLS", "ROOT FILE",
			"35/35", "29/29", "55/55", "84/84",
			"—", // codex commands cell (NotApplicable), never "0/29"
			"CLAUDE.md", "AGENTS.md",
			"not installed: copilot, cursor, generic, grok, opencode",
			"codex has no command loader",
			"(55 canonical + 29 commands = 84)",
		} {
			if !strings.Contains(report, want) {
				t.Errorf("healthy table missing %q:\n%s", want, report)
			}
		}
		if strings.Contains(report, "0/29") {
			t.Errorf("codex commands must never render as 0/29:\n%s", report)
		}
		if strings.Contains(report, "WARNING") {
			t.Errorf("healthy install must emit no WARNING:\n%s", report)
		}
		if strings.Contains(report, "!") {
			t.Errorf("healthy install must have no `!` markers:\n%s", report)
		}
		// Section ordering: after Workspace graph, before Verdict.
		gi := strings.Index(report, "Workspace graph")
		si := strings.Index(report, "Installed harness targets")
		vi := strings.Index(report, "Verdict:")
		if !(gi < si && si < vi) {
			t.Errorf("section must sit between Workspace graph and Verdict (gi=%d si=%d vi=%d):\n%s", gi, si, vi, report)
		}
	})

	t.Run("codex footnote omitted when codex absent", func(t *testing.T) {
		info := base
		info.inventory = []install.TargetInventory{healthyClaude()}
		report := buildDoctorReport(info)
		if strings.Contains(report, "codex has no command loader") {
			t.Errorf("codex footnote must be omitted when codex is absent:\n%s", report)
		}
		if !strings.Contains(report, "not installed: codex, copilot, cursor, generic, grok, opencode") {
			t.Errorf("expected codex on the not-installed line:\n%s", report)
		}
	})

	t.Run("grok commands roll into native skills", func(t *testing.T) {
		info := base
		info.inventory = []install.TargetInventory{healthyGrok()}
		report := buildDoctorReport(info)
		for _, want := range []string{"grok", "—", "grok has no standalone command loader", ".grok/skills/command-<name>/", "(55 canonical + 29 commands = 84)"} {
			if !strings.Contains(report, want) {
				t.Errorf("Grok inventory missing %q:\n%s", want, report)
			}
		}
	})

	t.Run("shortfall_recommends_hero_upgrade", func(t *testing.T) {
		claudeShort := healthyClaude()
		claudeShort.Commands = kc(12, 29) // installed but short
		info := base
		info.inventory = []install.TargetInventory{claudeShort, healthyCodex()}
		report := buildDoctorReport(info)

		if !strings.Contains(report, "12/29 !") {
			t.Errorf("expected short cell flagged with `!`:\n%s", report)
		}
		if !strings.Contains(report, "WARNING:") {
			t.Errorf("expected in-section WARNING for the shortfall:\n%s", report)
		}
		if !strings.Contains(report, "hero upgrade") {
			t.Errorf("shortfall WARNING must recommend `hero upgrade`:\n%s", report)
		}
		if strings.Contains(report, "hero install") {
			t.Errorf("shortfall WARNING must NOT mention `hero install`:\n%s", report)
		}
		// Verdict is unchanged by a shortfall.
		if !strings.Contains(report, "Verdict: OK — binary and graph agree on schema 4.") {
			t.Errorf("shortfall must not alter the Verdict line:\n%s", report)
		}
	})

	t.Run("not_installed_target_no_warning", func(t *testing.T) {
		// cursor is absent (only claude + codex installed, both healthy). It
		// must appear only on the not-installed line — no `!`, no WARNING, no
		// upgrade nudge.
		info := base
		info.inventory = []install.TargetInventory{healthyClaude(), healthyCodex()}
		report := buildDoctorReport(info)

		if !strings.Contains(report, "not installed: copilot, cursor, generic, grok, opencode") {
			t.Errorf("expected cursor on the not-installed line:\n%s", report)
		}
		if strings.Contains(report, "!") {
			t.Errorf("a not-installed target must produce no `!`:\n%s", report)
		}
		if strings.Contains(report, "WARNING") {
			t.Errorf("a not-installed target must produce no WARNING:\n%s", report)
		}
		if strings.Contains(report, "hero upgrade") {
			t.Errorf("a not-installed target must not trigger the upgrade nudge:\n%s", report)
		}
	})

	t.Run("over_count_is_not_a_shortfall", func(t *testing.T) {
		// A stale extra file on disk makes actual EXCEED expected (this repo
		// really produces claude 30/29). Over-count is not a shortfall: no
		// `!`, no WARNING, no upgrade nudge. Pins the strict `<` so a future
		// change to `<=`/`!=` can't silently start flagging over-counts.
		claudeOver := healthyClaude()
		claudeOver.Commands = kc(30, 29) // installed, over-count
		info := base
		info.inventory = []install.TargetInventory{claudeOver, healthyCodex()}
		report := buildDoctorReport(info)

		if !strings.Contains(report, "30/29") {
			t.Errorf("expected over-count cell rendered as 30/29:\n%s", report)
		}
		if strings.Contains(report, "!") {
			t.Errorf("over-count must not be flagged with `!`:\n%s", report)
		}
		if strings.Contains(report, "WARNING") {
			t.Errorf("over-count must emit no WARNING:\n%s", report)
		}
		if strings.Contains(report, "hero upgrade") {
			t.Errorf("over-count must not trigger the upgrade nudge:\n%s", report)
		}
	})

	t.Run("verdict_unchanged_under_shortfall", func(t *testing.T) {
		healthy := base
		healthy.inventory = []install.TargetInventory{healthyClaude(), healthyCodex()}

		shortInfo := base
		claudeShort := healthyClaude()
		claudeShort.Skills = kc(0, 55)
		shortInfo.inventory = []install.TargetInventory{claudeShort, healthyCodex()}

		if verdictLine(buildDoctorReport(healthy)) != verdictLine(buildDoctorReport(shortInfo)) {
			t.Errorf("Verdict line must be byte-identical with and without a shortfall")
		}
	})

	t.Run("empty install state renders neutral line", func(t *testing.T) {
		info := base
		info.inventory = nil
		report := buildDoctorReport(info)
		if !strings.Contains(report, "no harness targets installed") {
			t.Errorf("expected neutral empty-state line:\n%s", report)
		}
		if !strings.Contains(report, "hero install --target") {
			t.Errorf("empty state must name `hero install --target`:\n%s", report)
		}
		if strings.Contains(report, "WARNING") {
			t.Errorf("empty state must emit no WARNING:\n%s", report)
		}
	})

	t.Run("section skipped with no graph or workspace", func(t *testing.T) {
		info := base
		info.heroDir = ""
		info.graphSchema = ""
		info.inventory = []install.TargetInventory{healthyClaude()} // populated but must be skipped
		report := buildDoctorReport(info)
		if strings.Contains(report, "Installed harness targets") {
			t.Errorf("section must be skipped on the no-workspace early return:\n%s", report)
		}
	})

	t.Run("introspection error is a note, not a failure", func(t *testing.T) {
		info := base
		info.inventoryErr = "boom"
		report := buildDoctorReport(info)
		if !strings.Contains(report, "install introspection unavailable: boom") {
			t.Errorf("expected non-fatal introspection note:\n%s", report)
		}
		if !strings.Contains(report, "Verdict: OK") {
			t.Errorf("doctor must still reach the Verdict when introspection fails:\n%s", report)
		}
	})
}

// verdictLine extracts the Verdict: line (and its indented continuation) for
// byte-identity comparison.
func verdictLine(report string) string {
	i := strings.Index(report, "Verdict:")
	if i < 0 {
		return ""
	}
	return report[i:]
}
