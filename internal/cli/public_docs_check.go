package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/serve"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type mcpProfileCount struct {
	Name  string
	Count int
}

type mcpInventory struct {
	Total    int
	Default  int
	Profiles []mcpProfileCount
}

func canonicalMCPInventory(projectRoot string) (mcpInventory, error) {
	definitions := serve.MCPToolDefinitions()
	inventory := mcpInventory{Total: len(definitions), Default: len(definitions)}
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return inventory, fmt.Errorf("loading MCP tool filter for documentation inventory: %w", err)
	}
	if cfg.Serve == nil || cfg.Serve.ToolFilter == nil {
		return inventory, nil
	}
	filter := serve.NewToolFilter(cfg.Serve.ToolFilter)
	inventory.Default = len(filter.FilterTools(definitions, ""))
	profileNames := make([]string, 0, len(cfg.Serve.ToolFilter.Profiles))
	for name := range cfg.Serve.ToolFilter.Profiles {
		profileNames = append(profileNames, name)
	}
	sort.Strings(profileNames)
	for _, name := range profileNames {
		inventory.Profiles = append(inventory.Profiles, mcpProfileCount{
			Name: name, Count: len(filter.FilterTools(definitions, name)),
		})
	}
	return inventory, nil
}

type publicNarrativeRule struct {
	pattern *regexp.Regexp
	reason  string
}

var publicNarrativeRules = []publicNarrativeRule{
	{regexp.MustCompile(`(?i)\bv0\.9(?:\.\d+)?\b`), "stale v0.9 product copy"},
	{regexp.MustCompile(`(?i)\bhero\s+is\s+(?:an?\s+)?open[ -]source\b`), "Hero's private source has not crossed the public visibility gate"},
	{regexp.MustCompile(`(?i)\bhero\s+is\s+(?:licensed|available)\s+under\s+(?:the\s+)?MIT\b`), "Hero is Apache-2.0 licensed, not MIT"},
	{regexp.MustCompile(`(?i)\bhero(?:\s|-)(?:code|cloud)\s+(?:is|are)\s+open[ -]source\b`), "Hero Code and Hero Cloud are proprietary"},
	{regexp.MustCompile(`(?i)\bhero(?:\s|-)(?:code|cloud).{0,80}\bApache-2\.0\b`), "Hero Code and Hero Cloud are outside Hero's grant"},
	{regexp.MustCompile(`(?i)--auth-token\b|\bHERO_TEAM_TOKEN\b`), "secret-bearing or stale configuration guidance"},
	{regexp.MustCompile(`(?i)hero\s*[·-]\s*sidekick brain`), "superseded positioning"},
	{regexp.MustCompile(`(?i)approval-aware agent jobs`), "unsupported public capability claim"},
	{regexp.MustCompile(`(?i)\bhero installs (?:all )?(?:workflows|commands) as slash commands\b`), "workflows are harness-native, not universal slash commands"},
	{regexp.MustCompile(`(?i)\bhero uses one (?:shared|global) graph across (?:all )?(?:projects|repositories|repos)\b`), "each Hero project has its own graph; peering crosses project boundaries"},
	{regexp.MustCompile(`(?i)\bhero spec complete is the (?:normal|recommended) delivery close\b`), "verified delivery closes through hero spec verify"},
	{regexp.MustCompile(`(?i)\bhero upgrade (?:updates|replaces) the (?:hero )?binary\b`), "hero upgrade refreshes workspace files, not the binary"},
	{regexp.MustCompile(`(?i)\bhero(?:\s|-)(?:code|cloud) is included in this repository\b`), "Hero Code and Hero Cloud are separate proprietary repositories"},
	{regexp.MustCompile(`(?i)\bsprout is (?:included in|part of) Hero's (?:future )?Apache-2\.0 grant\b`), "Sprout is a separate MIT-licensed project outside Hero's grant"},
	{regexp.MustCompile(`(?i)\bsprout is (?:an? )?(?:Apache-2\.0 licensed|proprietary)\b`), "Sprout is a separate public MIT-licensed project"},
	{regexp.MustCompile(`(?i)\bsprout is included in this repository\b`), "Sprout is a separate dependency, not part of this repository"},
}

var inventedContinuityRules = []publicNarrativeRule{
	{regexp.MustCompile(`(?i)\b(?:preview outcome|continuity (?:demonstration|proof)|still being proven)\b`), "invented continuity-proof qualifier is not public product truth"},
	{regexp.MustCompile(`(?i)\bdoes not promise that every tool or session\b`), "invented perfection disclaimer is not public product truth"},
}

var publicTruthAuthorityPaths = []string{
	".hero/marketing/positioning.md",
	".hero/specs/hero-public-truth-baseline/public-claim-registry.yaml",
	"docs/releases/v0.34.0-candidate.md",
}

var heroConfigBlock = regexp.MustCompile(`(?s)<!--\s*hero-config\s*-->\s*` + "```json" + `\s*(.*?)\s*` + "```")
var heroQuickstartBlock = regexp.MustCompile(`(?s)<!--\s*hero-quickstart\s*-->\s*` + "```(?:bash|sh)" + `\s*(.*?)\s*` + "```")
var executableShellBlock = regexp.MustCompile(`(?s)` + "```(?:bash|sh|console)" + `\s*(.*?)\s*` + "```")
var privateSourceLink = regexp.MustCompile(`(?i)https://github\.com/hero-engine/hero(?:["'/#?)]|$)`)

func publicDocsIssues(projectRoot string) []string {
	var issues []string
	surfaces := publicNarrativeSurfaces(projectRoot)
	for path, content := range surfaces {
		issues = append(issues, publicNarrativeIssues(path, content)...)
		issues = append(issues, publicConfigExampleIssues(path, content)...)
	}
	issues = append(issues, publicExecutableInvocationIssues(surfaces)...)
	issues = append(issues, publicTruthAuthorityIssues(projectRoot)...)
	issues = append(issues, repositoryBoundaryIssues(surfaces)...)
	issues = append(issues, repositoryLicenseIssues(projectRoot)...)
	issues = append(issues, docsDependencyIssues(filepath.Join(projectRoot, "requirements-docs.txt"))...)
	issues = append(issues, revisionTemplateIssues(projectRoot)...)
	sort.Strings(issues)
	return issues
}

func publicExecutableInvocationIssues(surfaces map[string]string) []string {
	var issues []string
	for path, content := range surfaces {
		for _, block := range executableShellBlock.FindAllStringSubmatch(content, -1) {
			for _, line := range logicalShellLines(block[1]) {
				trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "$ "))
				if !strings.HasPrefix(trimmed, "hero ") {
					continue
				}
				tokens, err := splitShellWords(trimmed)
				if err != nil {
					issues = append(issues, fmt.Sprintf("%s: executable `%s`: %v", path, trimmed, err))
					continue
				}
				if len(tokens) < 2 {
					issues = append(issues, fmt.Sprintf("%s: executable `%s` has no command", path, trimmed))
					continue
				}
				arguments := tokens[1:]
				for index, token := range arguments {
					if token == "<" || token == ">" || token == "|" || token == "&&" || token == "||" {
						arguments = arguments[:index]
						break
					}
				}
				if err := validateExecutableInvocation(rootCmd, arguments); err != nil {
					issues = append(issues, fmt.Sprintf("%s: executable `%s`: %v", path, trimmed, err))
				}
			}
		}
	}
	return issues
}

func logicalShellLines(block string) []string {
	var lines []string
	current := ""
	for _, raw := range strings.Split(block, "\n") {
		line := strings.TrimSpace(raw)
		if current != "" {
			current += " "
		}
		if strings.HasSuffix(line, "\\") {
			current += strings.TrimSpace(strings.TrimSuffix(line, "\\"))
			continue
		}
		current += line
		if strings.TrimSpace(current) != "" {
			lines = append(lines, current)
		}
		current = ""
	}
	if strings.TrimSpace(current) != "" {
		lines = append(lines, current)
	}
	return lines
}

func splitShellWords(line string) ([]string, error) {
	var words []string
	var word strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if word.Len() > 0 {
			words = append(words, word.String())
			word.Reset()
		}
	}
	for _, character := range line {
		if escaped {
			word.WriteRune(character)
			escaped = false
			continue
		}
		if quote != 0 {
			if character == quote {
				quote = 0
			} else if character == '\\' && quote == '"' {
				escaped = true
			} else {
				word.WriteRune(character)
			}
			continue
		}
		switch character {
		case '\\':
			escaped = true
		case '\'', '"':
			quote = character
		case '#':
			if word.Len() == 0 {
				flush()
				return words, nil
			}
			word.WriteRune(character)
		case ' ', '\t', '\r', '\n':
			flush()
		default:
			word.WriteRune(character)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	if escaped {
		return nil, fmt.Errorf("unterminated escape")
	}
	flush()
	return words, nil
}

func validateExecutableInvocation(root *cobra.Command, arguments []string) error {
	if len(arguments) == 0 {
		return fmt.Errorf("empty command")
	}
	cmd := root
	consumed := 0
	for consumed < len(arguments) {
		token := arguments[consumed]
		if strings.HasPrefix(token, "-") {
			break
		}
		var child *cobra.Command
		for _, candidate := range cmd.Commands() {
			if candidate.Name() == token || candidate.HasAlias(token) {
				child = candidate
				break
			}
		}
		if child == nil {
			break
		}
		cmd = child
		consumed++
	}
	if cmd == root && !strings.HasPrefix(arguments[0], "-") {
		return fmt.Errorf("unknown subcommand %q", arguments[0])
	}

	provided := make(map[string]bool)
	positionals := make([]string, 0, len(arguments)-consumed)
	for index := consumed; index < len(arguments); index++ {
		token := arguments[index]
		if token == "--help" || token == "-h" || token == "--version" {
			continue
		}
		if strings.HasPrefix(token, "--") {
			nameValue := strings.TrimPrefix(token, "--")
			name, value, hasValue := strings.Cut(nameValue, "=")
			flag := executableFlag(cmd, name, "")
			if flag == nil {
				return fmt.Errorf("flag --%s not found on %s", name, cmd.CommandPath())
			}
			provided[name] = true
			if hasValue {
				if value == "" && flag.NoOptDefVal == "" {
					return fmt.Errorf("flag --%s requires a value", name)
				}
				continue
			}
			if flag.NoOptDefVal == "" {
				if index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "-") {
					return fmt.Errorf("flag --%s requires a value", name)
				}
				index++
			}
			continue
		}
		if strings.HasPrefix(token, "-") && len(token) == 2 {
			flag := executableFlag(cmd, "", token[1:])
			if flag == nil {
				return fmt.Errorf("flag %s not found on %s", token, cmd.CommandPath())
			}
			provided[flag.Name] = true
			if flag.NoOptDefVal == "" {
				if index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "-") {
					return fmt.Errorf("flag %s requires a value", token)
				}
				index++
			}
			continue
		}
		positionals = append(positionals, token)
	}
	if cmd.Args != nil && !(cmd == searchCmd && provided["list"] && len(positionals) == 0) {
		if err := cmd.Args(cmd, positionals); err != nil {
			return err
		}
	}
	var requiredError error
	visitRequired := func(flag *pflag.Flag) {
		if requiredError != nil || provided[flag.Name] {
			return
		}
		if values := flag.Annotations[cobra.BashCompOneRequiredFlag]; len(values) > 0 && values[0] == "true" {
			requiredError = fmt.Errorf("required flag --%s missing", flag.Name)
		}
	}
	cmd.Flags().VisitAll(visitRequired)
	cmd.InheritedFlags().VisitAll(visitRequired)
	return requiredError
}

func executableFlag(cmd *cobra.Command, name, shorthand string) *pflag.Flag {
	if name != "" {
		if flag := cmd.Flags().Lookup(name); flag != nil {
			return flag
		}
		return cmd.InheritedFlags().Lookup(name)
	}
	if flag := cmd.Flags().ShorthandLookup(shorthand); flag != nil {
		return flag
	}
	return cmd.InheritedFlags().ShorthandLookup(shorthand)
}

func publicNarrativeSurfaces(projectRoot string) map[string]string {
	surfaces := make(map[string]string)
	for _, name := range []string{
		"README.md", "GETTING-STARTED.md", "MCP-SETUP.md",
		"CROSS-REPO-PEERING.md", "TEAM-SERVER.md", "web/landing/site/index.html",
	} {
		if data, err := os.ReadFile(filepath.Join(projectRoot, name)); err == nil {
			surfaces[name] = string(data)
		}
	}
	addTextTree := func(relativeRoot string) {
		root := filepath.Join(projectRoot, filepath.FromSlash(relativeRoot))
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || !publicTextExtension(filepath.Ext(path)) {
				return walkErr
			}
			rel, err := filepath.Rel(projectRoot, path)
			if err != nil {
				return nil
			}
			if data, err := os.ReadFile(path); err == nil {
				surfaces[filepath.ToSlash(rel)] = string(data)
			}
			return nil
		})
	}
	addTextTree("web/docs/src")
	addTextTree("web/landing/site")
	for _, name := range []string{"web/docs/mkdocs.yml"} {
		data, err := os.ReadFile(filepath.Join(projectRoot, name))
		if err == nil {
			surfaces[name] = string(data)
		}
	}
	surfaces["<rendered:internal/install/agents_md.go>"] = string(install.RenderAgentsMdBodyForDriftTest())
	return surfaces
}

func publicTextExtension(extension string) bool {
	switch strings.ToLower(extension) {
	case ".css", ".html", ".js", ".json", ".md", ".svg", ".txt", ".xml", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func repositoryBoundaryIssues(surfaces map[string]string) []string {
	checks := []struct {
		path    string
		pattern *regexp.Regexp
		reason  string
	}{
		{"README.md", regexp.MustCompile(`(?is)Hero Code.{0,80}separate proprietary product`), "must identify Hero Code as a separate proprietary product"},
		{"README.md", regexp.MustCompile(`(?is)Hero Cloud.{0,80}separate proprietary product`), "must identify Hero Cloud as a separate proprietary product"},
		{"README.md", regexp.MustCompile(`(?is)Sprout.{0,120}separate public MIT-licensed dependency`), "must identify Sprout as a separate public MIT-licensed dependency"},
		{"README.md", regexp.MustCompile(`(?is)this .hero. repository.{0,100}licensed\s+under the Apache License 2\.0`), "must identify this Hero repository as Apache-2.0 licensed"},
		{"web/docs/src/index.md", regexp.MustCompile(`(?is)Sprout.{0,120}separate MIT-licensed project.{0,120}not covered by Hero's Apache-2\.0 grant`), "must keep Sprout outside Hero's Apache-2.0 grant"},
		{"web/docs/src/index.md", regexp.MustCompile(`(?is)this .hero. repository.{0,100}licensed\s+under the Apache License 2\.0`), "must identify this Hero repository as Apache-2.0 licensed"},
	}
	var issues []string
	for _, check := range checks {
		if !check.pattern.MatchString(surfaces[check.path]) {
			issues = append(issues, fmt.Sprintf("%s: %s", check.path, check.reason))
		}
	}
	return issues
}

const apacheLicenseSHA256 = "cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30"

func repositoryLicenseIssues(projectRoot string) []string {
	licenseText, err := os.ReadFile(filepath.Join(projectRoot, "LICENSE"))
	if err != nil {
		return []string{fmt.Sprintf("LICENSE: read canonical Apache-2.0 text: %v", err)}
	}
	digest := sha256.Sum256(licenseText)
	if fmt.Sprintf("%x", digest) != apacheLicenseSHA256 {
		return []string{"LICENSE: does not match the canonical Apache License 2.0 text"}
	}

	notices, err := os.ReadFile(filepath.Join(projectRoot, "THIRD_PARTY_NOTICES.txt"))
	if err != nil {
		return []string{fmt.Sprintf("THIRD_PARTY_NOTICES.txt: read release notices: %v", err)}
	}
	for _, required := range []string{
		"Hero — third-party notices",
		"Go runtime and standard library",
		"Embedded hero-embed-v1 model lineage",
		"modernc.org/libc v1.73.4",
	} {
		if !bytes.Contains(notices, []byte(required)) {
			return []string{fmt.Sprintf("THIRD_PARTY_NOTICES.txt: missing %q", required)}
		}
	}
	return nil
}

func publicQuickstartIssues(binary, projectRoot string) []string {
	var issues []string
	blocks := 0
	for _, path := range []string{"README.md", "GETTING-STARTED.md", "web/docs/src/index.md"} {
		data, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(path)))
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: read quickstart: %v", path, err))
			continue
		}
		for index, match := range heroQuickstartBlock.FindAllStringSubmatch(string(data), -1) {
			blocks++
			if err := exercisePublicQuickstart(binary, match[1]); err != nil {
				issues = append(issues, fmt.Sprintf("%s: quickstart %d: %v", path, index+1, err))
			}
		}
	}
	if blocks != 3 {
		issues = append(issues, fmt.Sprintf("public quickstart markers: found %d, want 3", blocks))
	}
	return issues
}

func exercisePublicQuickstart(binary, block string) error {
	root, err := os.MkdirTemp("", "hero-public-quickstart-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	gitInit := exec.Command("git", "init", "-q")
	gitInit.Dir = root
	if output, err := gitInit.CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %w: %s", err, strings.TrimSpace(string(output)))
	}
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		return err
	}
	environment := append(os.Environ(),
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_STATE_HOME="+filepath.Join(home, ".local", "state"),
	)
	executed := 0
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "hero ") {
			continue
		}
		arguments := strings.Fields(strings.TrimPrefix(line, "hero "))
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		command := exec.CommandContext(ctx, binary, arguments...)
		command.Dir = root
		command.Env = environment
		output, runErr := command.CombinedOutput()
		cancel()
		if runErr != nil {
			return fmt.Errorf("`%s`: %w: %s", line, runErr, strings.TrimSpace(string(output)))
		}
		executed++
	}
	if executed < 3 {
		return fmt.Errorf("executed %d Hero commands, want at least 3", executed)
	}
	for _, expected := range []string{
		".hero/hero.json",
		".agents/skills/command-deliver/SKILL.md",
		".codex/agents/engineer.toml",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(expected))); err != nil {
			return fmt.Errorf("expected install artifact %s: %w", expected, err)
		}
	}
	return nil
}

func publicNarrativeIssues(path, content string) []string {
	issues := narrativeRuleIssues(path, content, publicNarrativeRules)
	issues = append(issues, narrativeRuleIssues(path, content, inventedContinuityRules)...)
	if privateSourceLink.MatchString(content) {
		issues = append(issues, fmt.Sprintf("%s: public source link is enabled before the visibility gate", path))
	}
	return issues
}

func narrativeRuleIssues(path, content string, rules []publicNarrativeRule) []string {
	var issues []string
	for _, rule := range rules {
		if path == "web/docs/src/releases/index.md" && rule.reason == "stale v0.9 product copy" {
			continue
		}
		matches := rule.pattern.FindAllStringIndex(content, -1)
		for _, match := range matches {
			line := 1 + strings.Count(content[:match[0]], "\n")
			issues = append(issues, fmt.Sprintf("%s:%d: %s", path, line, rule.reason))
		}
	}
	return issues
}

func publicTruthAuthorityIssues(projectRoot string) []string {
	var issues []string
	for _, path := range publicTruthAuthorityPaths {
		data, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(path)))
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		issues = append(issues, narrativeRuleIssues(path, string(data), inventedContinuityRules)...)
	}
	return issues
}

func publicConfigExampleIssues(path, content string) []string {
	var issues []string
	for index, match := range heroConfigBlock.FindAllStringSubmatch(content, -1) {
		root, err := os.MkdirTemp("", "hero-public-config-")
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: create config validation workspace: %v", path, err))
			continue
		}
		heroDir := filepath.Join(root, config.DefaultFolder)
		err = os.MkdirAll(heroDir, 0o755)
		if err == nil {
			err = os.WriteFile(filepath.Join(heroDir, config.ConfigFileName), []byte(match[1]), 0o644)
		}
		if err == nil {
			_, err = config.Load(root)
		}
		_ = os.RemoveAll(root)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: hero-config example %d fails config.Load: %v", path, index+1, err))
		}
	}
	return issues
}

func docsDependencyIssues(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("requirements-docs.txt: %v", err)}
	}
	versions := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "==", 2)
		if len(parts) == 2 {
			versions[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	var issues []string
	for name, allowedMajor := range map[string]string{"mkdocs": "1", "mkdocs-material": "9"} {
		version, ok := versions[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("requirements-docs.txt: %s must be exactly pinned", name))
			continue
		}
		if !strings.HasPrefix(version, allowedMajor+".") {
			issues = append(issues, fmt.Sprintf("requirements-docs.txt: %s %s crosses supported major %s", name, version, allowedMajor))
		}
	}
	return issues
}

func revisionTemplateIssues(projectRoot string) []string {
	type markerContract struct {
		path string
		keys []string
	}
	contracts := []markerContract{
		{"web/docs/src/revision.json", []string{"source_revision", "current_release", "generated_at"}},
		{"web/landing/site/revision.json", []string{"source_revision", "source_commit", "source_digest", "source_dirty", "generated_at", "canonical_url"}},
	}
	var issues []string
	for _, contract := range contracts {
		data, err := os.ReadFile(filepath.Join(projectRoot, contract.path))
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", contract.path, err))
			continue
		}
		var marker map[string]interface{}
		if err := json.Unmarshal(data, &marker); err != nil {
			issues = append(issues, fmt.Sprintf("%s: invalid JSON: %v", contract.path, err))
			continue
		}
		for _, key := range contract.keys {
			if _, ok := marker[key]; !ok {
				issues = append(issues, fmt.Sprintf("%s: missing %s", contract.path, key))
			}
		}
	}
	landing, err := os.ReadFile(filepath.Join(projectRoot, "web/landing/site/index.html"))
	if err != nil || !strings.Contains(string(landing), `name="hero-source-revision" content="BUILD_TIME_SOURCE_REVISION"`) {
		issues = append(issues, "web/landing/site/index.html: missing build-time source revision metadata")
	} else {
		landingSource := strings.ToLower(string(landing))
		for phrase, reason := range map[string]string{
			"repository boundary:":    "internal repository control is rendered as marketing copy",
			"artifact revision":       "build provenance is rendered as marketing copy",
			"build_time_generated_at": "unresolved build timestamp is rendered as marketing copy",
			"data-source-revision":    "build provenance is rendered as marketing copy",
		} {
			if strings.Contains(landingSource, phrase) {
				issues = append(issues, fmt.Sprintf("web/landing/site/index.html: %s", reason))
			}
		}
	}
	return issues
}

func productionBaseURLs() map[string]string {
	return map[string]string{
		"docs":    "https://docs.heroengine.ai",
		"landing": "https://heroengine.ai",
	}
}

func productionPublicIssues(client *http.Client, surface, expectedRevision string, bases map[string]string) []string {
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(expectedRevision) {
		return []string{"--expected-revision must be an exact 40-character lowercase Git SHA with --production"}
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	wanted := []string{surface}
	if surface == "all" {
		wanted = []string{"docs", "landing"}
	}
	var issues []string
	for _, name := range wanted {
		base, ok := bases[name]
		if !ok || (name != "docs" && name != "landing") {
			issues = append(issues, fmt.Sprintf("unsupported production surface %q; use docs, landing, or all", surface))
			continue
		}
		paths := []string{"/"}
		if name == "docs" {
			paths = append(paths, "/what-is-hero/", "/getting-started/project-setup/", "/about/build/")
		}
		for _, path := range paths {
			if err := fetchProductionURL(client, strings.TrimRight(base, "/")+path, nil); err != nil {
				issues = append(issues, err.Error())
			}
		}
		var marker map[string]interface{}
		markerURL := strings.TrimRight(base, "/") + "/revision.json"
		if err := fetchProductionURL(client, markerURL, &marker); err != nil {
			issues = append(issues, err.Error())
			continue
		}
		if revision, _ := marker["source_revision"].(string); revision != expectedRevision {
			issues = append(issues, fmt.Sprintf("%s: source_revision %q does not match %q", markerURL, revision, expectedRevision))
		}
		generatedAt, _ := marker["generated_at"].(string)
		if _, err := time.Parse(time.RFC3339, generatedAt); err != nil {
			issues = append(issues, fmt.Sprintf("%s: generated_at is not RFC3339", markerURL))
		}
	}
	return issues
}

func fetchProductionURL(client *http.Client, url string, destination interface{}) error {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", url, err)
	}
	request.Header.Set("User-Agent", "hero-public-docs-check/1.0")
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%s: %w", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s: returned HTTP %d", url, response.StatusCode)
	}
	if destination == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 2<<20))
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(destination); err != nil {
		return fmt.Errorf("%s: invalid revision JSON: %w", url, err)
	}
	return nil
}
