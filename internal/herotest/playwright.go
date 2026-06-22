package herotest

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// PlaywrightFramework implements TestFramework for Playwright Test.
type PlaywrightFramework struct{}

func (p *PlaywrightFramework) Name() string { return "playwright" }

func (p *PlaywrightFramework) TestFilePath(slug string, cfg *config.TestingConfig) string {
	dir := "e2e"
	if cfg != nil && cfg.TestDir != "" {
		dir = cfg.TestDir
	}
	return filepath.Join(dir, slug+".spec.ts")
}

func (p *PlaywrightFramework) RunCommand(testFile string, cfg *config.TestingConfig) (string, []string) {
	runner := "npx"
	args := []string{"playwright", "test", testFile}
	if cfg != nil && cfg.RunnerCommand != "" {
		parts := strings.Fields(cfg.RunnerCommand)
		if len(parts) > 0 {
			runner = parts[0]
			args = append(parts[1:], testFile)
		}
	}
	if cfg != nil && cfg.ConfigPath != "" {
		args = append(args, "--config", cfg.ConfigPath)
	}
	return runner, args
}

func (p *PlaywrightFramework) GenerateAssisted(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error) {
	if len(criteria) == 0 {
		return "", fmt.Errorf("spec %q has no acceptance criteria", s.Slug)
	}

	var b strings.Builder
	p.writeHeader(&b, s, "assisted")
	p.writeVideoConditional(&b)

	b.WriteString(fmt.Sprintf("test.describe('%s', () => {\n", s.Slug))

	for _, criterion := range criteria {
		testName := CriterionToTestName(criterion)
		b.WriteString(fmt.Sprintf("  test('%s', async ({ page }) => {\n", escapeTS(testName)))
		baseURL := resolveBaseURL(cfg)
		if baseURL != "" {
			b.WriteString(fmt.Sprintf("    await page.goto('%s');\n", baseURL))
		}
		b.WriteString(fmt.Sprintf("    // TODO: implement — %s\n", criterion))
		b.WriteString("    test.skip();\n")
		b.WriteString("  });\n\n")
	}

	b.WriteString("});\n")
	return b.String(), nil
}

func (p *PlaywrightFramework) GenerateAutonomous(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error) {
	if len(criteria) == 0 {
		return "", fmt.Errorf("spec %q has no acceptance criteria", s.Slug)
	}

	var b strings.Builder
	p.writeHeader(&b, s, "autonomous")
	p.writeVideoConditional(&b)

	b.WriteString(fmt.Sprintf("test.describe('%s', () => {\n", s.Slug))

	baseURL := resolveBaseURL(cfg)

	for _, raw := range criteria {
		c := spec.ClassifyCriterion(raw)
		testName := CriterionToTestName(raw)
		b.WriteString(fmt.Sprintf("  test('%s', async ({ page }) => {\n", escapeTS(testName)))

		if baseURL != "" {
			b.WriteString(fmt.Sprintf("    await page.goto('%s');\n", baseURL))
		}

		b.WriteString(assertionForCriterion(c, baseURL))

		b.WriteString("  });\n\n")
	}

	b.WriteString("});\n")
	return b.String(), nil
}

// assertionForCriterion picks a structured path when the criterion is EARS,
// otherwise falls back to the legacy whole-string keyword heuristic so that
// specs predating EARS still generate the same code as before.
func assertionForCriterion(c spec.Criterion, baseURL string) string {
	if !c.Kind.IsEARS() {
		return mapCriterionToAssertion(c.Raw, baseURL)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("    // EARS:%s — %s\n", c.Kind, c.Raw))

	// Event/unwanted patterns carry a trigger that usually maps to a user action.
	if c.Kind == spec.CriterionEvent || c.Kind == spec.CriterionUnwanted {
		if setup := triggerSetup(c.Trigger); setup != "" {
			b.WriteString(setup)
		}
	}

	// Behavior text drives the assertion — run the existing keyword mapper
	// against just the behavior clause rather than the whole bullet.
	b.WriteString(mapCriterionToAssertion(c.Behavior, baseURL))
	return b.String()
}

// triggerSetup produces Playwright setup code from the trigger clause of an
// event or unwanted criterion. Returns empty string when no mapping applies.
func triggerSetup(trigger string) string {
	lower := strings.ToLower(trigger)
	switch {
	case containsAny(lower, "click", "press"):
		return "    // Trigger: " + trigger + "\n    // await page.click('button');\n"
	case containsAny(lower, "submit"):
		return "    // Trigger: " + trigger + "\n    // await page.click('button[type=\"submit\"]');\n"
	case containsAny(lower, "fill", "type", "enter"):
		return "    // Trigger: " + trigger + "\n    // await page.fill('input[name=\"field\"]', 'value');\n"
	case containsAny(lower, "navigate", "visit", "open"):
		return "    // Trigger: " + trigger + "\n    // await page.goto('/some-path');\n"
	default:
		return ""
	}
}

func (p *PlaywrightFramework) AgentContext(s *spec.Spec, criteria []string, cfg *config.TestingConfig) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Test Requirements for %s\n\n", s.Slug))
	b.WriteString(fmt.Sprintf("Generate Playwright tests in: `%s`\n\n", p.TestFilePath(s.Slug, cfg)))
	b.WriteString("### Acceptance Criteria to Cover\n\n")
	for i, c := range criteria {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, c))
	}
	baseURL := resolveBaseURL(cfg)
	if baseURL != "" {
		b.WriteString(fmt.Sprintf("\n### Base URL\n\n`%s`\n", baseURL))
	}
	b.WriteString("\n### Framework\n\nPlaywright Test (`@playwright/test`)\n")
	b.WriteString("Use `import { test, expect } from '@playwright/test';`\n")
	return b.String()
}

// writeHeader writes the standard test file header.
func (p *PlaywrightFramework) writeHeader(b *strings.Builder, s *spec.Spec, mode string) {
	b.WriteString("import { test, expect } from '@playwright/test';\n\n")
	b.WriteString(fmt.Sprintf("// Auto-generated from Hero spec: %s\n", s.Slug))
	if s.Title != "" {
		b.WriteString(fmt.Sprintf("// Spec: %s\n", s.Title))
	}
	b.WriteString(fmt.Sprintf("// Mode: %s\n", mode))
	b.WriteString(fmt.Sprintf("// Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339)))
}

// writeVideoConditional writes the conditional video recording block.
func (p *PlaywrightFramework) writeVideoConditional(b *strings.Builder) {
	b.WriteString("// Enable video recording when PWVIDEO=1 is set\n")
	b.WriteString("if (process.env.PWVIDEO === '1') {\n")
	b.WriteString("  test.use({ video: 'on' });\n")
	b.WriteString("}\n\n")
}

func resolveBaseURL(cfg *config.TestingConfig) string {
	if cfg != nil && cfg.BaseURL != "" {
		return cfg.BaseURL
	}
	return ""
}

// mapCriterionToAssertion maps a natural-language criterion to Playwright assertion code.
// It delegates keyword classification to the shared ClassifyAssertion and then renders
// Playwright-specific code from the returned hint.
func mapCriterionToAssertion(criterion string, baseURL string) string {
	hint := ClassifyAssertion(criterion)
	return renderPlaywrightAssertion(hint, baseURL)
}

// renderPlaywrightAssertion emits Playwright assertion code from an AssertionHint.
func renderPlaywrightAssertion(hint AssertionHint, baseURL string) string {
	var b strings.Builder

	switch hint.Kind {
	case "url":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		if baseURL != "" {
			b.WriteString(fmt.Sprintf("    await expect(page).toHaveURL(/%s/);\n", escapeRegex(baseURL)))
		} else {
			b.WriteString("    // TODO: specify expected URL pattern\n")
			b.WriteString("    // await expect(page).toHaveURL(/expected-path/);\n")
		}

	case "title":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    // TODO: specify expected title\n")
		b.WriteString("    // await expect(page).toHaveTitle(/Expected Title/);\n")

	case "visible":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		if hint.Selector != "" {
			b.WriteString(fmt.Sprintf("    await expect(page.locator('%s')).toBeVisible();\n", hint.Selector))
		} else {
			b.WriteString("    // TODO: specify element selector\n")
			b.WriteString("    // await expect(page.locator('.element')).toBeVisible();\n")
		}

	case "text_contains":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		if hint.QuotedText != "" {
			b.WriteString(fmt.Sprintf("    await expect(page.locator('body')).toContainText('%s');\n", escapeTS(hint.QuotedText)))
		} else {
			b.WriteString("    // TODO: specify expected text\n")
			b.WriteString("    // await expect(page.locator('.element')).toContainText('expected');\n")
		}

	case "count":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    // TODO: specify selector and expected count\n")
		b.WriteString("    // await expect(page.locator('.item')).toHaveCount(N);\n")

	case "input":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    // TODO: specify input selector and expected value\n")
		b.WriteString("    // await expect(page.locator('input[name=\"field\"]')).toHaveValue('expected');\n")

	case "click":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    // TODO: specify button/element to click and expected result\n")
		b.WriteString("    // await page.click('button[type=\"submit\"]');\n")
		b.WriteString("    // await expect(page).toHaveURL(/success/);\n")

	case "error":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    // TODO: trigger error condition and verify error message\n")
		b.WriteString("    // await expect(page.locator('.error')).toBeVisible();\n")

	case "output":
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    // TODO: verify output/response content\n")
		b.WriteString("    // await expect(page.locator('[data-testid=\"output\"]')).toContainText('expected');\n")

	default:
		b.WriteString(fmt.Sprintf("    // Criterion: %s\n", hint.Criterion))
		b.WriteString("    // TODO: implement assertion for this criterion\n")
		b.WriteString("    test.skip();\n")
	}

	return b.String()
}


// escapeTS escapes characters that would break a TypeScript string literal.
func escapeTS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

// escapeRegex escapes characters that would break a regex literal.
func escapeRegex(s string) string {
	special := []string{".", "+", "*", "?", "(", ")", "[", "]", "{", "}", "|", "^", "$"}
	for _, ch := range special {
		s = strings.ReplaceAll(s, ch, "\\"+ch)
	}
	return s
}
