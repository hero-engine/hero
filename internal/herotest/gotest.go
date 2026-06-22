package herotest

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// GoTestFramework implements TestFramework for Go's built-in testing package.
type GoTestFramework struct{}

func (g *GoTestFramework) Name() string { return "go" }

func (g *GoTestFramework) TestFilePath(slug string, cfg *config.TestingConfig) string {
	dir := "."
	if cfg != nil && cfg.TestDir != "" {
		dir = cfg.TestDir
	}
	// Go file names use underscores, not hyphens.
	goSlug := strings.ReplaceAll(slug, "-", "_")
	return filepath.Join(dir, goSlug+"_test.go")
}

func (g *GoTestFramework) RunCommand(testFile string, cfg *config.TestingConfig) (string, []string) {
	if cfg != nil && cfg.RunnerCommand != "" {
		parts := strings.Fields(cfg.RunnerCommand)
		if len(parts) > 0 {
			return parts[0], append(parts[1:], testFile)
		}
	}
	// Derive the pascal-case test name prefix from the slug in the file name.
	base := filepath.Base(testFile)
	base = strings.TrimSuffix(base, "_test.go")
	pascal := slugToPascal(strings.ReplaceAll(base, "_", "-"))
	return "go", []string{"test", "-run", "Test" + pascal, "-v", "./..."}
}

func (g *GoTestFramework) GenerateAssisted(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error) {
	if len(criteria) == 0 {
		return "", fmt.Errorf("spec %q has no acceptance criteria", s.Slug)
	}

	pkg := g.detectPackage(cfg)

	var b strings.Builder
	g.writeHeader(&b, s, "assisted", pkg)

	funcName := "Test" + slugToPascal(s.Slug)
	b.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", funcName))

	for _, criterion := range criteria {
		b.WriteString(fmt.Sprintf("\tt.Run(%q, func(t *testing.T) {\n", criterion))
		b.WriteString("\t\tt.Skip(\"TODO: implement\")\n")
		b.WriteString("\t})\n\n")
	}

	b.WriteString("}\n")
	return b.String(), nil
}

func (g *GoTestFramework) GenerateAutonomous(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error) {
	if len(criteria) == 0 {
		return "", fmt.Errorf("spec %q has no acceptance criteria", s.Slug)
	}

	pkg := g.detectPackage(cfg)

	var b strings.Builder
	g.writeHeader(&b, s, "autonomous", pkg)

	funcName := "Test" + slugToPascal(s.Slug)
	b.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", funcName))

	for _, raw := range criteria {
		c := spec.ClassifyCriterion(raw)
		b.WriteString(fmt.Sprintf("\tt.Run(%q, func(t *testing.T) {\n", raw))

		if c.Kind.IsEARS() {
			b.WriteString(fmt.Sprintf("\t\t// EARS:%s — %s\n", c.Kind, c.Raw))
			if c.Kind == spec.CriterionEvent || c.Kind == spec.CriterionUnwanted {
				if setup := goTestTriggerSetup(c.Trigger); setup != "" {
					b.WriteString(setup)
				}
			}
			b.WriteString(goTestAssertionForBehavior(c))
		} else {
			hint := ClassifyAssertion(raw)
			b.WriteString(renderGoTestAssertion(hint))
		}

		b.WriteString("\t})\n\n")
	}

	b.WriteString("}\n")
	return b.String(), nil
}

func (g *GoTestFramework) AgentContext(s *spec.Spec, criteria []string, cfg *config.TestingConfig) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Test Requirements for %s\n\n", s.Slug))
	b.WriteString(fmt.Sprintf("Generate Go tests in: `%s`\n\n", g.TestFilePath(s.Slug, cfg)))
	b.WriteString("### Acceptance Criteria to Cover\n\n")
	for i, c := range criteria {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, c))
	}
	b.WriteString("\n### Framework\n\n")
	b.WriteString("Go `testing` package (`import \"testing\"`)\n\n")
	b.WriteString("#### Conventions\n\n")
	b.WriteString("- Test functions: `func TestXxx(t *testing.T)`\n")
	b.WriteString("- Subtests: `t.Run(\"name\", func(t *testing.T) { ... })`\n")
	b.WriteString("- Table-driven tests: define a `tests` slice of structs, iterate with `t.Run`\n")
	b.WriteString("- Assertions: `t.Errorf`, `t.Fatalf`, `t.Fatal`, `t.Error`\n")
	b.WriteString("- Helper functions: mark with `t.Helper()` for clean stack traces\n")
	b.WriteString("- Test data: place in `testdata/` directory (ignored by `go build`)\n")
	b.WriteString("- Parallel tests: `t.Parallel()` at the start of the test/subtest\n")
	return b.String()
}

// writeHeader writes the standard test file header for Go tests.
func (g *GoTestFramework) writeHeader(b *strings.Builder, s *spec.Spec, mode, pkg string) {
	b.WriteString(fmt.Sprintf("package %s\n\n", pkg))
	b.WriteString("import \"testing\"\n\n")
	b.WriteString(fmt.Sprintf("// Auto-generated from Hero spec: %s\n", s.Slug))
	if s.Title != "" {
		b.WriteString(fmt.Sprintf("// Spec: %s\n", s.Title))
	}
	b.WriteString(fmt.Sprintf("// Mode: %s\n", mode))
	b.WriteString(fmt.Sprintf("// Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339)))
}

// detectPackage determines the Go package name for the test file.
// It reads the first .go file in TestDir to extract the package declaration,
// falling back to the directory name.
func (g *GoTestFramework) detectPackage(cfg *config.TestingConfig) string {
	dir := "."
	if cfg != nil && cfg.TestDir != "" {
		dir = cfg.TestDir
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return packageFromDir(dir)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "package ") {
				pkg := strings.TrimPrefix(line, "package ")
				pkg = strings.TrimSpace(pkg)
				f.Close()
				if pkg != "" {
					return pkg
				}
			}
		}
		f.Close()
	}

	return packageFromDir(dir)
}

// packageFromDir derives a package name from a directory path.
func packageFromDir(dir string) string {
	if dir == "." || dir == "" {
		return "main"
	}
	base := filepath.Base(dir)
	// Go package names are lowercase, no hyphens.
	base = strings.ReplaceAll(base, "-", "")
	if base == "" {
		return "main"
	}
	return strings.ToLower(base)
}

// goTestTriggerSetup produces Go test setup comments from an EARS trigger clause.
func goTestTriggerSetup(trigger string) string {
	lower := strings.ToLower(trigger)
	switch {
	case containsAny(lower, "click", "press", "tap"):
		return "\t\t// Trigger: " + trigger + "\n\t\t// Simulate user action\n"
	case containsAny(lower, "submit"):
		return "\t\t// Trigger: " + trigger + "\n\t\t// Simulate form submission\n"
	case containsAny(lower, "fill", "type", "enter"):
		return "\t\t// Trigger: " + trigger + "\n\t\t// Simulate input\n"
	case containsAny(lower, "navigate", "visit", "open"):
		return "\t\t// Trigger: " + trigger + "\n\t\t// Simulate navigation\n"
	default:
		return ""
	}
}

// goTestAssertionForBehavior generates Go test assertions from an EARS criterion's behavior.
func goTestAssertionForBehavior(c spec.Criterion) string {
	if c.Kind == spec.CriterionUnwanted {
		return "\t\t// Verify error handling for unwanted behavior\n" +
			"\t\terr := performAction()\n" +
			"\t\tif err == nil {\n" +
			fmt.Sprintf("\t\t\tt.Fatal(\"expected error: %s\")\n", c.Behavior) +
			"\t\t}\n"
	}

	hint := ClassifyAssertion(c.Behavior)
	return renderGoTestAssertion(hint)
}

// renderGoTestAssertion emits Go test assertion code from an AssertionHint.
func renderGoTestAssertion(hint AssertionHint) string {
	var b strings.Builder

	switch hint.Kind {
	case "visible":
		b.WriteString(fmt.Sprintf("\t\t// Criterion: %s\n", hint.Criterion))
		b.WriteString("\t\t// TODO: replace with actual visibility check\n")
		b.WriteString("\t\tif !isVisible {\n")
		b.WriteString("\t\t\tt.Errorf(\"expected element to be visible\")\n")
		b.WriteString("\t\t}\n")

	case "text_contains":
		b.WriteString(fmt.Sprintf("\t\t// Criterion: %s\n", hint.Criterion))
		if hint.QuotedText != "" {
			b.WriteString(fmt.Sprintf("\t\tif !strings.Contains(got, %q) {\n", hint.QuotedText))
			b.WriteString(fmt.Sprintf("\t\t\tt.Errorf(\"expected to contain %%q, got %%q\", %q, got)\n", hint.QuotedText))
			b.WriteString("\t\t}\n")
		} else {
			b.WriteString("\t\t// TODO: specify expected text\n")
			b.WriteString("\t\tif !strings.Contains(got, \"expected\") {\n")
			b.WriteString("\t\t\tt.Errorf(\"expected to contain %q, got %q\", \"expected\", got)\n")
			b.WriteString("\t\t}\n")
		}

	case "error":
		b.WriteString(fmt.Sprintf("\t\t// Criterion: %s\n", hint.Criterion))
		b.WriteString("\t\terr := performAction()\n")
		b.WriteString("\t\tif err == nil {\n")
		b.WriteString("\t\t\tt.Fatal(\"expected error\")\n")
		b.WriteString("\t\t}\n")

	case "url":
		b.WriteString(fmt.Sprintf("\t\t// Criterion: %s\n", hint.Criterion))
		b.WriteString("\t\t// TODO: specify expected URL\n")
		b.WriteString("\t\tif got != expectedURL {\n")
		b.WriteString("\t\t\tt.Errorf(\"URL = %q, want %q\", got, expectedURL)\n")
		b.WriteString("\t\t}\n")

	case "count":
		b.WriteString(fmt.Sprintf("\t\t// Criterion: %s\n", hint.Criterion))
		b.WriteString("\t\t// TODO: specify expected count\n")
		b.WriteString("\t\tif len(items) != expectedCount {\n")
		b.WriteString("\t\t\tt.Errorf(\"count = %d, want %d\", len(items), expectedCount)\n")
		b.WriteString("\t\t}\n")

	default:
		b.WriteString(fmt.Sprintf("\t\t// Criterion: %s\n", hint.Criterion))
		b.WriteString("\t\t// TODO: implement assertion\n")
		b.WriteString("\t\tt.Skip(\"not yet implemented\")\n")
	}

	return b.String()
}

func init() {
	Register(&GoTestFramework{})
}
