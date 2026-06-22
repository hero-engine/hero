package herotest

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// PytestFramework implements TestFramework for Python's pytest.
type PytestFramework struct{}

func (p *PytestFramework) Name() string { return "pytest" }

func (p *PytestFramework) TestFilePath(slug string, cfg *config.TestingConfig) string {
	dir := "tests"
	if cfg != nil && cfg.TestDir != "" {
		dir = cfg.TestDir
	}
	// Python file names use underscores, not hyphens.
	pySlug := strings.ReplaceAll(slug, "-", "_")
	return filepath.Join(dir, "test_"+pySlug+".py")
}

func (p *PytestFramework) RunCommand(testFile string, cfg *config.TestingConfig) (string, []string) {
	if cfg != nil && cfg.RunnerCommand != "" {
		parts := strings.Fields(cfg.RunnerCommand)
		if len(parts) > 0 {
			return parts[0], append(parts[1:], testFile)
		}
	}
	return "pytest", []string{testFile, "-v"}
}

func (p *PytestFramework) GenerateAssisted(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error) {
	if len(criteria) == 0 {
		return "", fmt.Errorf("spec %q has no acceptance criteria", s.Slug)
	}

	var b strings.Builder
	p.writeHeader(&b, s, "assisted")

	for _, criterion := range criteria {
		funcName := FormatTestName(criterion, NameStyleSnake)
		b.WriteString(fmt.Sprintf("\n@pytest.mark.skip(reason=\"TODO: implement — %s\")\n", escapePython(criterion)))
		b.WriteString(fmt.Sprintf("def %s():\n", funcName))
		b.WriteString("    pass\n")
	}

	return b.String(), nil
}

func (p *PytestFramework) GenerateAutonomous(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error) {
	if len(criteria) == 0 {
		return "", fmt.Errorf("spec %q has no acceptance criteria", s.Slug)
	}

	var b strings.Builder
	p.writeHeader(&b, s, "autonomous")

	for _, raw := range criteria {
		c := spec.ClassifyCriterion(raw)
		funcName := FormatTestName(raw, NameStyleSnake)
		b.WriteString(fmt.Sprintf("\ndef %s():\n", funcName))

		if c.Kind.IsEARS() {
			b.WriteString(fmt.Sprintf("    # EARS:%s — %s\n", c.Kind, c.Raw))
			if c.Kind == spec.CriterionEvent || c.Kind == spec.CriterionUnwanted {
				if setup := pytestTriggerSetup(c.Trigger); setup != "" {
					b.WriteString(setup)
				}
			}
			b.WriteString(pytestAssertionForBehavior(c))
		} else {
			hint := ClassifyAssertion(raw)
			b.WriteString(renderPytestAssertion(hint))
		}
	}

	return b.String(), nil
}

func (p *PytestFramework) AgentContext(s *spec.Spec, criteria []string, cfg *config.TestingConfig) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Test Requirements for %s\n\n", s.Slug))
	b.WriteString(fmt.Sprintf("Generate pytest tests in: `%s`\n\n", p.TestFilePath(s.Slug, cfg)))
	b.WriteString("### Acceptance Criteria to Cover\n\n")
	for i, c := range criteria {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, c))
	}
	b.WriteString("\n### Framework\n\n")
	b.WriteString("pytest (`import pytest`)\n\n")
	b.WriteString("#### Conventions\n\n")
	b.WriteString("- Test functions: `def test_something():` (no class needed)\n")
	b.WriteString("- Assertions: plain `assert` statements — `assert result == expected`\n")
	b.WriteString("- Fixtures: `@pytest.fixture` for setup/teardown, injected via function parameters\n")
	b.WriteString("- Parametrize: `@pytest.mark.parametrize('input,expected', [...])` for table-driven tests\n")
	b.WriteString("- Error testing: `with pytest.raises(ExceptionType):` block\n")
	b.WriteString("- Shared fixtures: place in `conftest.py` (auto-discovered by pytest)\n")
	b.WriteString("- Skip: `pytest.skip('reason')` or `@pytest.mark.skip(reason='...')`\n")
	return b.String()
}

// writeHeader writes the standard test file header for pytest.
func (p *PytestFramework) writeHeader(b *strings.Builder, s *spec.Spec, mode string) {
	b.WriteString(fmt.Sprintf("\"\"\"Tests for %s.\"\"\"\n", s.Slug))
	b.WriteString("import pytest\n\n")
	b.WriteString(fmt.Sprintf("# Auto-generated from Hero spec: %s\n", s.Slug))
	if s.Title != "" {
		b.WriteString(fmt.Sprintf("# Spec: %s\n", s.Title))
	}
	b.WriteString(fmt.Sprintf("# Mode: %s\n", mode))
	b.WriteString(fmt.Sprintf("# Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
}

// pytestTriggerSetup produces pytest setup comments from an EARS trigger clause.
func pytestTriggerSetup(trigger string) string {
	lower := strings.ToLower(trigger)
	switch {
	case containsAny(lower, "click", "press", "tap"):
		return "    # Trigger: " + trigger + "\n    # Simulate user click action\n"
	case containsAny(lower, "submit"):
		return "    # Trigger: " + trigger + "\n    # Simulate form submission\n"
	case containsAny(lower, "fill", "type", "enter"):
		return "    # Trigger: " + trigger + "\n    # Simulate text input\n"
	case containsAny(lower, "navigate", "visit", "open"):
		return "    # Trigger: " + trigger + "\n    # Simulate navigation\n"
	default:
		return ""
	}
}

// pytestAssertionForBehavior generates pytest assertions from an EARS criterion's behavior.
func pytestAssertionForBehavior(c spec.Criterion) string {
	if c.Kind == spec.CriterionUnwanted {
		return "    # Verify error handling for unwanted behavior\n" +
			"    with pytest.raises(Exception):\n" +
			fmt.Sprintf("        pass  # TODO: trigger — %s\n", c.Behavior)
	}

	hint := ClassifyAssertion(c.Behavior)
	return renderPytestAssertion(hint)
}

// renderPytestAssertion emits pytest assertion code from an AssertionHint.
func renderPytestAssertion(hint AssertionHint) string {
	var b strings.Builder

	switch hint.Kind {
	case "visible":
		b.WriteString(fmt.Sprintf("    # Criterion: %s\n", hint.Criterion))
		b.WriteString("    # TODO: replace with actual element check\n")
		b.WriteString("    assert result is not None\n")

	case "text_contains":
		b.WriteString(fmt.Sprintf("    # Criterion: %s\n", hint.Criterion))
		if hint.QuotedText != "" {
			b.WriteString(fmt.Sprintf("    assert '%s' in text\n", escapePython(hint.QuotedText)))
		} else {
			b.WriteString("    # TODO: specify expected text\n")
			b.WriteString("    assert 'expected' in text\n")
		}

	case "error":
		b.WriteString(fmt.Sprintf("    # Criterion: %s\n", hint.Criterion))
		b.WriteString("    with pytest.raises(Exception):\n")
		b.WriteString("        pass  # TODO: trigger error condition\n")

	case "url":
		b.WriteString(fmt.Sprintf("    # Criterion: %s\n", hint.Criterion))
		b.WriteString("    # TODO: specify expected URL\n")
		b.WriteString("    assert url == 'expected-url'\n")

	case "count":
		b.WriteString(fmt.Sprintf("    # Criterion: %s\n", hint.Criterion))
		b.WriteString("    # TODO: specify expected count\n")
		b.WriteString("    assert len(items) == 0\n")

	case "click":
		b.WriteString(fmt.Sprintf("    # Criterion: %s\n", hint.Criterion))
		b.WriteString("    # TODO: simulate click and verify result\n")
		b.WriteString("    assert result\n")

	case "input":
		b.WriteString(fmt.Sprintf("    # Criterion: %s\n", hint.Criterion))
		b.WriteString("    # TODO: verify input/form value\n")
		b.WriteString("    assert value == 'expected'\n")

	default:
		b.WriteString(fmt.Sprintf("    # Criterion: %s\n", hint.Criterion))
		b.WriteString("    # TODO: implement assertion\n")
		b.WriteString("    assert result\n")
	}

	return b.String()
}

// escapePython escapes characters that would break a Python string literal.
func escapePython(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

func init() {
	Register(&PytestFramework{})
}
