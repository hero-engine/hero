// Package herotest provides pluggable test generation from Hero spec acceptance criteria.
package herotest

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// TestFramework defines the interface for pluggable test framework adapters.
type TestFramework interface {
	// Name returns the framework identifier (e.g. "playwright", "cypress").
	Name() string

	// GenerateAssisted produces a scaffolded test file with TODO placeholders for each criterion.
	GenerateAssisted(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error)

	// GenerateAutonomous produces a complete, runnable test file from criteria.
	GenerateAutonomous(s *spec.Spec, criteria []string, cfg *config.TestingConfig) (string, error)

	// AgentContext returns formatted context for an agent to write tests during delivery.
	AgentContext(s *spec.Spec, criteria []string, cfg *config.TestingConfig) string

	// TestFilePath returns the path to the generated test file for a given slug.
	TestFilePath(slug string, cfg *config.TestingConfig) string

	// RunCommand returns the executable and args to run a specific test file.
	RunCommand(testFile string, cfg *config.TestingConfig) (string, []string)
}

// registry holds registered test framework adapters.
var registry = map[string]TestFramework{}

// Register adds a test framework adapter to the global registry.
func Register(f TestFramework) {
	registry[f.Name()] = f
}

// Get returns the test framework adapter for the given name.
func Get(name string) (TestFramework, error) {
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown test framework %q (available: %s)", name, availableFrameworks())
	}
	return f, nil
}

func availableFrameworks() string {
	var names []string
	for k := range registry {
		names = append(names, k)
	}
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

// ExtractCriteria parses the "acceptance criteria" section of a spec into individual criterion strings.
func ExtractCriteria(s *spec.Spec) []string {
	section, ok := s.Sections["acceptance criteria"]
	if !ok {
		return nil
	}

	var criteria []string
	scanner := bufio.NewScanner(strings.NewReader(section))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "- ") {
			criteria = append(criteria, strings.TrimPrefix(line, "- "))
		} else if strings.HasPrefix(line, "* ") {
			criteria = append(criteria, strings.TrimPrefix(line, "* "))
		}
	}
	return criteria
}

// NameStyle controls how test function names are formatted per framework convention.
type NameStyle int

const (
	// NameStyleRaw lowercases and strips special chars (JS/Playwright convention).
	NameStyleRaw NameStyle = iota
	// NameStylePascal produces TestUserCanLogIn (Go convention).
	NameStylePascal
	// NameStyleCamel produces testUserCanLogIn (Swift/XCTest convention).
	NameStyleCamel
	// NameStyleSnake produces test_user_can_log_in (Python/pytest convention).
	NameStyleSnake
)

// FormatTestName converts a criterion string to a test function name in the given style.
func FormatTestName(criterion string, style NameStyle) string {
	if style == NameStyleRaw {
		return CriterionToTestName(criterion)
	}

	// Clean the criterion: remove backticks, quotes, and non-alphanumeric/space chars.
	cleaned := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ' ' {
			return r
		}
		return ' '
	}, criterion)

	// Split into words, dropping empty entries from consecutive spaces.
	words := strings.Fields(cleaned)
	if len(words) == 0 {
		switch style {
		case NameStylePascal:
			return "Test"
		case NameStyleSnake:
			return "test"
		default:
			return "test"
		}
	}

	switch style {
	case NameStylePascal:
		// "user can log in" -> "TestUserCanLogIn"
		var b strings.Builder
		b.WriteString("Test")
		for _, w := range words {
			b.WriteString(titleWord(w))
		}
		return b.String()

	case NameStyleCamel:
		// "user can log in" -> "testUserCanLogIn"
		var b strings.Builder
		b.WriteString("test")
		for _, w := range words {
			b.WriteString(titleWord(w))
		}
		return b.String()

	case NameStyleSnake:
		// "user can log in" -> "test_user_can_log_in"
		var lower []string
		for _, w := range words {
			lower = append(lower, strings.ToLower(w))
		}
		return "test_" + strings.Join(lower, "_")

	default:
		return CriterionToTestName(criterion)
	}
}

// titleWord capitalises the first letter of a word, lowercases the rest.
func titleWord(w string) string {
	if w == "" {
		return ""
	}
	return strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
}

// CriterionToTestName converts a criterion string to a concise test name.
func CriterionToTestName(criterion string) string {
	// Lowercase, trim, limit length
	name := strings.ToLower(criterion)
	// Remove backticks and special chars that break test names
	name = strings.ReplaceAll(name, "`", "")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.ReplaceAll(name, "\"", "")
	// Truncate to reasonable length
	if len(name) > 80 {
		name = name[:80]
	}
	return strings.TrimSpace(name)
}

func init() {
	// Register built-in frameworks
	Register(&PlaywrightFramework{})
}
