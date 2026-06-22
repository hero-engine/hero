package herotest

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// VitestFramework implements TestFramework for Vitest.
type VitestFramework struct{}

func (v *VitestFramework) Name() string { return "vitest" }

func (v *VitestFramework) TestFilePath(slug string, cfg *config.TestingConfig) string {
	dir := "tests"
	if cfg != nil && cfg.TestDir != "" {
		dir = cfg.TestDir
	}
	return filepath.Join(dir, slug+".test.ts")
}

func (v *VitestFramework) RunCommand(testFile string, cfg *config.TestingConfig) (string, []string) {
	if cfg != nil && cfg.RunnerCommand != "" {
		parts := strings.Fields(cfg.RunnerCommand)
		if len(parts) > 0 {
			return parts[0], append(parts[1:], testFile)
		}
	}
	return "npx", []string{"vitest", "run", testFile}
}

func (v *VitestFramework) GenerateAssisted(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error) {
	if len(criteria) == 0 {
		return "", fmt.Errorf("spec %q has no acceptance criteria", s.Slug)
	}

	var b strings.Builder
	v.writeHeader(&b, s, "assisted")

	b.WriteString(fmt.Sprintf("describe('%s', () => {\n", escapeJS(s.Slug)))

	for _, criterion := range criteria {
		testName := FormatTestName(criterion, NameStyleRaw)
		b.WriteString(fmt.Sprintf("  it.skip('%s', () => {\n", escapeJS(testName)))
		b.WriteString(fmt.Sprintf("    // TODO: implement — %s\n", criterion))
		b.WriteString("  });\n\n")
	}

	b.WriteString("});\n")
	return b.String(), nil
}

func (v *VitestFramework) GenerateAutonomous(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error) {
	if len(criteria) == 0 {
		return "", fmt.Errorf("spec %q has no acceptance criteria", s.Slug)
	}

	var b strings.Builder
	v.writeHeader(&b, s, "autonomous")

	b.WriteString(fmt.Sprintf("describe('%s', () => {\n", escapeJS(s.Slug)))

	for _, raw := range criteria {
		c := spec.ClassifyCriterion(raw)
		testName := FormatTestName(raw, NameStyleRaw)
		b.WriteString(fmt.Sprintf("  it('%s', () => {\n", escapeJS(testName)))

		if c.Kind.IsEARS() {
			b.WriteString(fmt.Sprintf("    // EARS:%s — %s\n", c.Kind, c.Raw))
			if c.Kind == spec.CriterionEvent || c.Kind == spec.CriterionUnwanted {
				if setup := vitestTriggerSetup(c.Trigger); setup != "" {
					b.WriteString(setup)
				}
			}
			b.WriteString(vitestAssertionForBehavior(c))
		} else {
			hint := ClassifyAssertion(raw)
			b.WriteString(renderVitestAssertion(hint))
		}

		b.WriteString("  });\n\n")
	}

	b.WriteString("});\n")
	return b.String(), nil
}

func (v *VitestFramework) AgentContext(s *spec.Spec, criteria []string, cfg *config.TestingConfig) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Test Requirements for %s\n\n", s.Slug))
	b.WriteString(fmt.Sprintf("Generate Vitest tests in: `%s`\n\n", v.TestFilePath(s.Slug, cfg)))
	b.WriteString("### Acceptance Criteria to Cover\n\n")
	for i, c := range criteria {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, c))
	}
	b.WriteString("\n### Framework\n\n")
	b.WriteString("Vitest (`vitest`)\n\n")
	b.WriteString("#### Conventions\n\n")
	b.WriteString("- Import: `import { describe, it, expect } from 'vitest'`\n")
	b.WriteString("- Test structure: `describe` / `it` / `expect`\n")
	b.WriteString("- Mocking: `vi.mock()`, `vi.spyOn()`, `vi.fn()`\n")
	b.WriteString("- Lifecycle: `beforeEach`, `afterEach`, `beforeAll`, `afterAll`\n")
	b.WriteString("- ESM imports are native — no CommonJS require()\n")
	b.WriteString("- Configuration: `vitest.config.ts`\n")
	return b.String()
}

// writeHeader writes the standard test file header for Vitest.
func (v *VitestFramework) writeHeader(b *strings.Builder, s *spec.Spec, mode string) {
	b.WriteString("import { describe, it, expect } from 'vitest'\n\n")
	b.WriteString(fmt.Sprintf("// Auto-generated from Hero spec: %s\n", s.Slug))
	if s.Title != "" {
		b.WriteString(fmt.Sprintf("// Spec: %s\n", s.Title))
	}
	b.WriteString(fmt.Sprintf("// Mode: %s\n", mode))
	b.WriteString(fmt.Sprintf("// Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339)))
}

// vitestTriggerSetup produces Vitest setup comments from an EARS trigger clause.
func vitestTriggerSetup(trigger string) string {
	lower := strings.ToLower(trigger)
	switch {
	case containsAny(lower, "click", "press", "tap"):
		return "    // Trigger: " + trigger + "\n    // Simulate user click action\n"
	case containsAny(lower, "submit"):
		return "    // Trigger: " + trigger + "\n    // Simulate form submission\n"
	case containsAny(lower, "fill", "type", "enter"):
		return "    // Trigger: " + trigger + "\n    // Simulate text input\n"
	case containsAny(lower, "navigate", "visit", "open"):
		return "    // Trigger: " + trigger + "\n    // Simulate navigation\n"
	default:
		return ""
	}
}

// vitestAssertionForBehavior generates Vitest assertions from an EARS criterion's behavior.
func vitestAssertionForBehavior(c spec.Criterion) string {
	if c.Kind == spec.CriterionUnwanted {
		return "    // Verify error handling for unwanted behavior\n" +
			"    expect(() => {\n" +
			fmt.Sprintf("      // TODO: trigger — %s\n", c.Behavior) +
			"    }).toThrow();\n"
	}

	hint := ClassifyAssertion(c.Behavior)
	return renderVitestAssertion(hint)
}

// renderVitestAssertion emits Vitest assertion code from an AssertionHint.
func renderVitestAssertion(hint AssertionHint) string {
	var b strings.Builder

	switch hint.Kind {
	case "visible":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    // TODO: replace with actual element check\n")
		b.WriteString("    expect(result).toBeTruthy();\n")

	case "text_contains":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		if hint.QuotedText != "" {
			b.WriteString(fmt.Sprintf("    expect(text).toContain('%s');\n", escapeJS(hint.QuotedText)))
		} else {
			b.WriteString("    // TODO: specify expected text\n")
			b.WriteString("    expect(text).toContain('expected');\n")
		}

	case "error":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    expect(() => {\n")
		b.WriteString("      // TODO: trigger error condition\n")
		b.WriteString("    }).toThrow();\n")

	case "url":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    // TODO: specify expected URL\n")
		b.WriteString("    expect(url).toBe('expected-url');\n")

	case "count":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    // TODO: specify expected count\n")
		b.WriteString("    expect(items).toHaveLength(0);\n")

	case "click":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    // TODO: simulate click and verify result\n")
		b.WriteString("    expect(result).toBeTruthy();\n")

	case "input":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    // TODO: verify input/form value\n")
		b.WriteString("    expect(value).toBe('expected');\n")

	default:
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    // TODO: implement assertion\n")
		b.WriteString("    expect(result).toBeTruthy();\n")
	}

	return b.String()
}

// escapeJS escapes characters that would break a JavaScript string literal.
func escapeJS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

func init() {
	Register(&VitestFramework{})
}
