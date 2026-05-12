package codescan

import (
	"regexp"
	"strings"
)

// configPattern holds a compiled regex and metadata for env var detection.
type configPattern struct {
	re       *regexp.Regexp
	langs    []string // languages this pattern applies to
	required bool     // whether this access pattern implies required (no default)
}

var configPatterns = []configPattern{
	// Go: os.Getenv("FOO")
	{re: regexp.MustCompile(`os\.Getenv\(\s*"([^"]+)"\s*\)`), langs: []string{"go"}, required: true},
	// Go: os.LookupEnv("FOO")
	{re: regexp.MustCompile(`os\.LookupEnv\(\s*"([^"]+)"\s*\)`), langs: []string{"go"}, required: false},

	// JS/TS: process.env.FOO
	{re: regexp.MustCompile(`process\.env\.([A-Za-z_][A-Za-z0-9_]*)`), langs: []string{"javascript", "typescript"}, required: true},
	// JS/TS: process.env["FOO"]
	{re: regexp.MustCompile(`process\.env\[\s*["']([^"']+)["']\s*\]`), langs: []string{"javascript", "typescript"}, required: true},

	// Python: os.environ["FOO"]
	{re: regexp.MustCompile(`os\.environ\[\s*["']([^"']+)["']\s*\]`), langs: []string{"python"}, required: true},
	// Python: os.getenv("FOO")
	{re: regexp.MustCompile(`os\.getenv\(\s*["']([^"']+)["']`), langs: []string{"python"}, required: false},
	// Python: os.environ.get("FOO")
	{re: regexp.MustCompile(`os\.environ\.get\(\s*["']([^"']+)["']`), langs: []string{"python"}, required: false},

	// Ruby: ENV["FOO"]
	{re: regexp.MustCompile(`ENV\[\s*["']([^"']+)["']\s*\]`), langs: []string{"ruby"}, required: true},
	// Ruby: ENV.fetch("FOO")
	{re: regexp.MustCompile(`ENV\.fetch\(\s*["']([^"']+)["']`), langs: []string{"ruby"}, required: true},
}

// ExtractConfigVars detects environment variable reads in source code.
func ExtractConfigVars(path string, content []byte, language string) []ConfigVar {
	lines := strings.Split(string(content), "\n")
	var vars []ConfigVar
	seen := make(map[string]bool) // dedupe by name+line

	for _, cp := range configPatterns {
		if !langMatches(cp.langs, language) {
			continue
		}
		for lineNum, line := range lines {
			matches := cp.re.FindAllStringSubmatchIndex(line, -1)
			for _, loc := range matches {
				name := line[loc[2]:loc[3]]
				key := name + ":" + string(rune(lineNum))
				if seen[key] {
					continue
				}
				seen[key] = true
				vars = append(vars, ConfigVar{
					Name:     name,
					Source:   "env",
					File:     path,
					Line:     lineNum + 1,
					Required: cp.required,
				})
			}
		}
	}

	return vars
}

func langMatches(langs []string, language string) bool {
	for _, l := range langs {
		if l == language {
			return true
		}
	}
	return false
}
