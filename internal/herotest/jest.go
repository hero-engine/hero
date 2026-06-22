package herotest

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// JestFramework implements TestFramework for Jest.
type JestFramework struct{}

func (j *JestFramework) Name() string { return "jest" }

func (j *JestFramework) TestFilePath(slug string, cfg *config.TestingConfig) string {
	dir := "__tests__"
	if cfg != nil && cfg.TestDir != "" {
		dir = cfg.TestDir
	}
	return filepath.Join(dir, slug+".test.ts")
}

func (j *JestFramework) RunCommand(testFile string, cfg *config.TestingConfig) (string, []string) {
	if cfg != nil && cfg.RunnerCommand != "" {
		parts := strings.Fields(cfg.RunnerCommand)
		if len(parts) > 0 {
			return parts[0], append(parts[1:], testFile)
		}
	}
	return "npx", []string{"jest", testFile, "--no-coverage"}
}

func (j *JestFramework) GenerateAssisted(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error) {
	if len(criteria) == 0 {
		return "", fmt.Errorf("spec %q has no acceptance criteria", s.Slug)
	}

	var b strings.Builder
	j.writeHeader(&b, s, "assisted")

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

func (j *JestFramework) GenerateAutonomous(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error) {
	if len(criteria) == 0 {
		return "", fmt.Errorf("spec %q has no acceptance criteria", s.Slug)
	}

	var b strings.Builder
	j.writeHeader(&b, s, "autonomous")

	b.WriteString(fmt.Sprintf("describe('%s', () => {\n", escapeJS(s.Slug)))

	for _, raw := range criteria {
		c := spec.ClassifyCriterion(raw)
		testName := FormatTestName(raw, NameStyleRaw)
		b.WriteString(fmt.Sprintf("  it('%s', () => {\n", escapeJS(testName)))

		if c.Kind.IsEARS() {
			b.WriteString(fmt.Sprintf("    // EARS:%s — %s\n", c.Kind, c.Raw))
			if c.Kind == spec.CriterionEvent || c.Kind == spec.CriterionUnwanted {
				if setup := jestTriggerSetup(c.Trigger); setup != "" {
					b.WriteString(setup)
				}
			}
			b.WriteString(jestAssertionForBehavior(c))
		} else {
			hint := ClassifyAssertion(raw)
			b.WriteString(renderJestAssertion(hint))
		}

		b.WriteString("  });\n\n")
	}

	b.WriteString("});\n")
	return b.String(), nil
}

func (j *JestFramework) AgentContext(s *spec.Spec, criteria []string, cfg *config.TestingConfig) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Test Requirements for %s\n\n", s.Slug))
	b.WriteString(fmt.Sprintf("Generate Jest tests in: `%s`\n\n", j.TestFilePath(s.Slug, cfg)))
	b.WriteString("### Acceptance Criteria to Cover\n\n")
	for i, c := range criteria {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, c))
	}
	b.WriteString("\n### Framework\n\n")
	b.WriteString("Jest (`jest`)\n\n")
	b.WriteString("#### Conventions\n\n")
	b.WriteString("- Jest globals are auto-available (no import needed): `describe`, `it`, `expect`, `beforeEach`, `afterEach`\n")
	b.WriteString("- Mocking: `jest.mock()`, `jest.spyOn()`, `jest.fn()`\n")
	b.WriteString("- Lifecycle: `beforeEach`, `afterEach`, `beforeAll`, `afterAll`\n")
	b.WriteString("- Supports both CommonJS (`require`) and ESM (`import`) — check project config\n")
	b.WriteString("- Configuration: `jest.config.js` or `jest.config.ts`\n")
	b.WriteString("- Default test directory: `__tests__/`\n")
	return b.String()
}

// writeHeader writes the standard test file header for Jest.
func (j *JestFramework) writeHeader(b *strings.Builder, s *spec.Spec, mode string) {
	b.WriteString("// Jest test file\n\n")
	b.WriteString(fmt.Sprintf("// Auto-generated from Hero spec: %s\n", s.Slug))
	if s.Title != "" {
		b.WriteString(fmt.Sprintf("// Spec: %s\n", s.Title))
	}
	b.WriteString(fmt.Sprintf("// Mode: %s\n", mode))
	b.WriteString(fmt.Sprintf("// Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339)))
}

// jestTriggerSetup produces Jest setup comments from an EARS trigger clause.
func jestTriggerSetup(trigger string) string {
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

// jestAssertionForBehavior generates Jest assertions from an EARS criterion's behavior.
func jestAssertionForBehavior(c spec.Criterion) string {
	if c.Kind == spec.CriterionUnwanted {
		return "    // Verify error handling for unwanted behavior\n" +
			"    expect(() => {\n" +
			fmt.Sprintf("      // TODO: trigger — %s\n", c.Behavior) +
			"    }).toThrow();\n"
	}

	hint := ClassifyAssertion(c.Behavior)
	return renderJestAssertion(hint)
}

// renderJestAssertion emits Jest assertion code from an AssertionHint.
func renderJestAssertion(hint AssertionHint) string {
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

func init() {
	Register(&JestFramework{})
}
