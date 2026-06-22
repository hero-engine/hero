package herotest

import (
	"os"
	"path/filepath"
	"strings"
)

// Detect examines the project root for framework marker files and returns the
// best-matching framework name. Returns ("", nil) when no framework can be
// detected. The caller should fall back to its own default in that case.
//
// Priority order (first match wins):
//  1. Package.swift          -> "xctest"
//  2. go.mod                 -> "go"
//  3. jest.config.{js,ts,mjs} -> "jest"
//  4. vitest.config.{ts,js,mjs} -> "vitest"
//  5. package.json with "vitest" -> "vitest"
//  6. package.json with "jest"   -> "jest"
//  7. package.json (bare)        -> "vitest" (modern default)
//  8. pyproject.toml / setup.py / setup.cfg / conftest.py -> "pytest"
//  9. playwright.config.{ts,js}  -> "playwright"
func Detect(projectRoot string) (string, error) {
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(projectRoot, name))
		return err == nil
	}

	// 1. Swift
	if exists("Package.swift") {
		return "xctest", nil
	}

	// 2. Go
	if exists("go.mod") {
		return "go", nil
	}

	// 3. Jest config files (checked before vitest — explicit config wins)
	for _, name := range []string{"jest.config.js", "jest.config.ts", "jest.config.mjs"} {
		if exists(name) {
			return "jest", nil
		}
	}

	// 4. Vitest config files
	for _, name := range []string{"vitest.config.ts", "vitest.config.js", "vitest.config.mjs"} {
		if exists(name) {
			return "vitest", nil
		}
	}

	// 5-7. package.json inspection
	pkgPath := filepath.Join(projectRoot, "package.json")
	if data, err := os.ReadFile(pkgPath); err == nil {
		content := string(data)
		if strings.Contains(content, `"vitest"`) {
			return "vitest", nil
		}
		if strings.Contains(content, `"jest"`) {
			return "jest", nil
		}
		// Bare package.json with no test framework -> vitest (modern default)
		return "vitest", nil
	}

	// 8. Python
	for _, name := range []string{"pyproject.toml", "setup.py", "setup.cfg", "conftest.py"} {
		if exists(name) {
			return "pytest", nil
		}
	}

	// 9. Playwright config
	for _, name := range []string{"playwright.config.ts", "playwright.config.js"} {
		if exists(name) {
			return "playwright", nil
		}
	}

	return "", nil
}
