package herotest

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// XCTestFramework implements TestFramework for Apple's XCTest.
type XCTestFramework struct{}

func (x *XCTestFramework) Name() string { return "xctest" }

func (x *XCTestFramework) TestFilePath(slug string, cfg *config.TestingConfig) string {
	module := "App"
	if cfg != nil && cfg.TestDir != "" {
		module = cfg.TestDir
	}
	pascalSlug := slugToPascal(slug)
	return filepath.Join("Tests", module+"Tests", pascalSlug+"Tests.swift")
}

func (x *XCTestFramework) RunCommand(testFile string, cfg *config.TestingConfig) (string, []string) {
	if cfg != nil && cfg.RunnerCommand != "" {
		parts := strings.Fields(cfg.RunnerCommand)
		if len(parts) > 0 {
			return parts[0], append(parts[1:], testFile)
		}
	}
	return "swift", []string{"test", "--filter", testFile}
}

func (x *XCTestFramework) GenerateAssisted(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error) {
	if len(criteria) == 0 {
		return "", fmt.Errorf("spec %q has no acceptance criteria", s.Slug)
	}

	var b strings.Builder
	x.writeHeader(&b, s, "assisted")

	className := slugToPascal(s.Slug) + "Tests"
	b.WriteString(fmt.Sprintf("final class %s: XCTestCase {\n", className))

	for _, criterion := range criteria {
		methodName := FormatTestName(criterion, NameStyleCamel)
		b.WriteString(fmt.Sprintf("    func %s() throws {\n", methodName))
		b.WriteString(fmt.Sprintf("        try XCTSkip(\"TODO: implement — %s\")\n", escapeSwift(criterion)))
		b.WriteString("    }\n\n")
	}

	b.WriteString("}\n")
	return b.String(), nil
}

func (x *XCTestFramework) GenerateAutonomous(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error) {
	if len(criteria) == 0 {
		return "", fmt.Errorf("spec %q has no acceptance criteria", s.Slug)
	}

	var b strings.Builder
	x.writeHeader(&b, s, "autonomous")

	className := slugToPascal(s.Slug) + "Tests"
	b.WriteString(fmt.Sprintf("final class %s: XCTestCase {\n", className))

	for _, raw := range criteria {
		c := spec.ClassifyCriterion(raw)
		methodName := FormatTestName(raw, NameStyleCamel)
		b.WriteString(fmt.Sprintf("    func %s() throws {\n", methodName))

		if c.Kind.IsEARS() {
			b.WriteString(fmt.Sprintf("        // EARS:%s — %s\n", c.Kind, c.Raw))
			if c.Kind == spec.CriterionEvent || c.Kind == spec.CriterionUnwanted {
				if setup := xcTestTriggerSetup(c.Trigger); setup != "" {
					b.WriteString(setup)
				}
			}
			b.WriteString(xcTestAssertionForBehavior(c))
		} else {
			hint := ClassifyAssertion(raw)
			b.WriteString(renderXCTestAssertion(hint))
		}

		b.WriteString("    }\n\n")
	}

	b.WriteString("}\n")
	return b.String(), nil
}

func (x *XCTestFramework) AgentContext(s *spec.Spec, criteria []string, cfg *config.TestingConfig) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Test Requirements for %s\n\n", s.Slug))
	b.WriteString(fmt.Sprintf("Generate XCTest tests in: `%s`\n\n", x.TestFilePath(s.Slug, cfg)))
	b.WriteString("### Acceptance Criteria to Cover\n\n")
	for i, c := range criteria {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, c))
	}
	b.WriteString("\n### Framework\n\n")
	b.WriteString("XCTest (`import XCTest`)\n\n")
	b.WriteString("#### Conventions\n\n")
	b.WriteString("- Test class inherits from `XCTestCase`\n")
	b.WriteString("- Test methods are `func testSomething() throws { ... }`\n")
	b.WriteString("- Assertions: `XCTAssertTrue`, `XCTAssertEqual`, `XCTAssertNil`, `XCTAssertThrowsError`\n")
	b.WriteString("- Skip: `try XCTSkip(\"reason\")`\n")
	b.WriteString("- Setup/teardown: `override func setUp()` / `override func tearDown()`\n\n")
	b.WriteString("#### Alternative: Swift Testing\n\n")
	b.WriteString("If the project uses Swift 5.10+ and targets modern platforms, consider Swift Testing:\n")
	b.WriteString("- `import Testing` instead of `import XCTest`\n")
	b.WriteString("- `@Test func someName() { ... }` instead of `func testSomeName()`\n")
	b.WriteString("- `#expect(condition)` instead of `XCTAssertTrue`\n")
	b.WriteString("- `#expect(throws: SomeError.self) { ... }` instead of `XCTAssertThrowsError`\n")
	return b.String()
}

// writeHeader writes the standard test file header for XCTest.
func (x *XCTestFramework) writeHeader(b *strings.Builder, s *spec.Spec, mode string) {
	b.WriteString("import XCTest\n\n")
	b.WriteString(fmt.Sprintf("// Auto-generated from Hero spec: %s\n", s.Slug))
	if s.Title != "" {
		b.WriteString(fmt.Sprintf("// Spec: %s\n", s.Title))
	}
	b.WriteString(fmt.Sprintf("// Mode: %s\n", mode))
	b.WriteString(fmt.Sprintf("// Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339)))
}

// xcTestTriggerSetup produces XCTest setup comments from an EARS trigger clause.
func xcTestTriggerSetup(trigger string) string {
	lower := strings.ToLower(trigger)
	switch {
	case containsAny(lower, "click", "press", "tap"):
		return "        // Trigger: " + trigger + "\n        // Simulate user tap/click action\n"
	case containsAny(lower, "submit"):
		return "        // Trigger: " + trigger + "\n        // Simulate form submission\n"
	case containsAny(lower, "fill", "type", "enter"):
		return "        // Trigger: " + trigger + "\n        // Simulate text input\n"
	case containsAny(lower, "navigate", "visit", "open"):
		return "        // Trigger: " + trigger + "\n        // Simulate navigation\n"
	default:
		return ""
	}
}

// xcTestAssertionForBehavior generates XCTest assertions from an EARS criterion's behavior.
func xcTestAssertionForBehavior(c spec.Criterion) string {
	if c.Kind == spec.CriterionUnwanted {
		return "        // Verify error handling for unwanted behavior\n" +
			"        XCTAssertThrowsError(try performAction()) { error in\n" +
			fmt.Sprintf("            // Expected: %s\n", c.Behavior) +
			"        }\n"
	}

	hint := ClassifyAssertion(c.Behavior)
	return renderXCTestAssertion(hint)
}

// renderXCTestAssertion emits XCTest assertion code from an AssertionHint.
func renderXCTestAssertion(hint AssertionHint) string {
	var b strings.Builder

	switch hint.Kind {
	case "visible":
		b.WriteString(fmt.Sprintf("        // Criterion: %s\n", hint.Criterion))
		b.WriteString("        // TODO: replace with actual element check\n")
		b.WriteString("        XCTAssertTrue(elementIsVisible)\n")

	case "text_contains":
		b.WriteString(fmt.Sprintf("        // Criterion: %s\n", hint.Criterion))
		if hint.QuotedText != "" {
			b.WriteString(fmt.Sprintf("        XCTAssertTrue(result.contains(\"%s\"))\n", escapeSwift(hint.QuotedText)))
		} else {
			b.WriteString("        // TODO: specify expected text\n")
			b.WriteString("        XCTAssertTrue(result.contains(\"expected\"))\n")
		}

	case "error":
		b.WriteString(fmt.Sprintf("        // Criterion: %s\n", hint.Criterion))
		b.WriteString("        XCTAssertThrowsError(try performAction()) { error in\n")
		b.WriteString("            // TODO: verify specific error\n")
		b.WriteString("        }\n")

	case "url":
		b.WriteString(fmt.Sprintf("        // Criterion: %s\n", hint.Criterion))
		b.WriteString("        // TODO: specify expected URL\n")
		b.WriteString("        XCTAssertEqual(url, expectedURL)\n")

	case "count":
		b.WriteString(fmt.Sprintf("        // Criterion: %s\n", hint.Criterion))
		b.WriteString("        // TODO: specify expected count\n")
		b.WriteString("        XCTAssertEqual(items.count, expectedCount)\n")

	default:
		b.WriteString(fmt.Sprintf("        // Criterion: %s\n", hint.Criterion))
		b.WriteString("        // TODO: implement assertion\n")
		b.WriteString("        XCTAssertTrue(false, \"Not yet implemented\")\n")
	}

	return b.String()
}

// slugToPascal converts a hyphenated slug to PascalCase.
// "login-flow" -> "LoginFlow"
func slugToPascal(slug string) string {
	parts := strings.Split(slug, "-")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + strings.ToLower(p[1:]))
	}
	if b.Len() == 0 {
		return "Test"
	}
	return b.String()
}

// escapeSwift escapes characters that would break a Swift string literal.
func escapeSwift(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

func init() {
	Register(&XCTestFramework{})
}
