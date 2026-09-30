package install

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// DeepSeek loads $DSH_HOME/cordis.patch.yml into every profile, including
// the desktop app's (which never passes --patch). Project installs register
// Hero's MCP server there as one marked block per project, the same
// "own only the marked span" contract as the Codex/Grok TOML upsert: bytes
// outside Hero's markers are never rewritten.

const (
	deepseekHomePatchName   = "cordis.patch.yml"
	deepseekHomeCreatedLine = "# Created by hero install. Entries between hero:managed markers belong to Hero; everything else is yours.\n"
	deepseekMCPPluginName   = "@deepseek-ai/dsh-mcp-client"
	// deepseekCommandEnv pins the MCP command, e.g. to a development build.
	deepseekCommandEnv = "HERO_DEEPSEEK_MCP_COMMAND"
)

var deepseekKeyUnsafe = regexp.MustCompile(`[^a-z0-9]+`)

// DeepSeekHomePatchPath is the home-level patch every DeepSeek profile loads.
func DeepSeekHomePatchPath() (string, error) {
	home, err := DeepSeekHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, deepseekHomePatchName), nil
}

// canonicalProjectRoot makes a root stable across "." and symlinked paths so
// install, doctor, and uninstall derive the same key.
func canonicalProjectRoot(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	return abs, nil
}

// DeepSeekServerName is the per-project MCP serverName: "hero-<name>-<hash4>".
// DeepSeek requires unique names of [A-Za-z0-9_-]{1,32}, and the name prefixes
// every tool (mcp__<serverName>__*), so it must never change for a project.
func DeepSeekServerName(root string) (string, error) {
	canonical, err := canonicalProjectRoot(root)
	if err != nil {
		return "", err
	}
	name := strings.Trim(deepseekKeyUnsafe.ReplaceAllString(strings.ToLower(filepath.Base(canonical)), "-"), "-")
	if name == "" {
		name = "project"
	}
	if len(name) > 22 {
		name = strings.TrimRight(name[:22], "-")
	}
	sum := sha256.Sum256([]byte(canonical))
	return "hero-" + name + "-" + hex.EncodeToString(sum[:])[:4], nil
}

// resolveDeepSeekHeroCommand returns an absolute hero path: GUI launches do
// not inherit the shell PATH, so the bare portable name would not resolve.
func resolveDeepSeekHeroCommand() (string, error) {
	if pinned := strings.TrimSpace(os.Getenv(deepseekCommandEnv)); pinned != "" {
		if !filepath.IsAbs(pinned) {
			return "", fmt.Errorf("%s must be an absolute path, got %q", deepseekCommandEnv, pinned)
		}
		return pinned, nil
	}
	// Keep PATH's own (possibly symlinked) path so `make install` or a
	// package-manager upgrade is picked up without reinstalling.
	if found, err := exec.LookPath("hero"); err == nil {
		if abs, err := filepath.Abs(found); err == nil {
			return abs, nil
		}
	}
	if self, err := os.Executable(); err == nil && !transientExecutable(self) {
		return self, nil
	}
	return "", fmt.Errorf("cannot find a stable `hero` binary for DeepSeek: put hero on PATH (e.g. `make install`) or set %s", deepseekCommandEnv)
}

func transientExecutable(path string) bool {
	slash := filepath.ToSlash(path)
	tmp := filepath.ToSlash(os.TempDir())
	return strings.Contains(slash, "/go-build") || (tmp != "" && strings.HasPrefix(slash, strings.TrimRight(tmp, "/")+"/"))
}

type deepseekMCPEntry struct {
	ID     string `yaml:"id"`
	Name   string `yaml:"name"`
	Config struct {
		ServerName string   `yaml:"serverName"`
		Transport  string   `yaml:"transport"`
		Command    string   `yaml:"command"`
		Args       []string `yaml:"args"`
		Cwd        string   `yaml:"cwd,omitempty"`
	} `yaml:"config"`
}

func deepseekMCPPatch(entry deepseekMCPEntry) ([]byte, error) {
	return yaml.Marshal([]struct {
		Insert []deepseekMCPEntry `yaml:"insert"`
	}{{Insert: []deepseekMCPEntry{entry}}})
}

func deepseekHomeMarkers(server string) (string, string) {
	return "# hero:managed deepseek-mcp " + server, "# end:hero:managed deepseek-mcp " + server
}

func deepseekHomeBlock(root, command string) (string, string, error) {
	canonical, err := canonicalProjectRoot(root)
	if err != nil {
		return "", "", err
	}
	server, err := DeepSeekServerName(canonical)
	if err != nil {
		return "", "", err
	}
	var entry deepseekMCPEntry
	entry.ID, entry.Name = server+"-mcp", deepseekMCPPluginName
	entry.Config.ServerName, entry.Config.Transport = server, "stdio"
	entry.Config.Command = command
	entry.Config.Args = []string{"mcp", "--project-root", canonical}
	entry.Config.Cwd = canonical
	data, err := deepseekMCPPatch(entry)
	if err != nil {
		return "", "", err
	}
	start, end := deepseekHomeMarkers(server)
	return server, start + "\n" + string(data) + end + "\n", nil
}

// cutDeepSeekHomeBlock removes server's marked block, reporting whether it
// was present. An unterminated block is refused rather than guessed at.
func cutDeepSeekHomeBlock(content, server string) (string, string, bool, error) {
	start, end := deepseekHomeMarkers(server)
	i := strings.Index(content, start+"\n")
	if i < 0 {
		return content, "", false, nil
	}
	j := strings.Index(content[i:], end)
	if j < 0 {
		return "", "", false, fmt.Errorf("unterminated Hero block %q in DeepSeek home patch", server)
	}
	k := i + j + len(end)
	if k < len(content) && content[k] == '\n' {
		k++
	}
	return content[:i] + content[k:], content[i:k], true, nil
}

// readDeepSeekHomePatch returns the current bytes ("" when absent) after
// refusing anything Hero must not write through.
func readDeepSeekHomePatch(path string) (string, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("DeepSeek home patch is not a regular file (symlink or special file): %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}
	var parsed interface{}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return "", false, fmt.Errorf("DeepSeek home patch %s is not valid YAML: %w", path, err)
	}
	if _, ok := parsed.([]interface{}); !ok && parsed != nil {
		return "", false, fmt.Errorf("DeepSeek home patch %s must be a YAML list of patch entries", path)
	}
	return string(data), true, nil
}

// PreflightDeepSeekHome fails before any install mutation when the home patch
// cannot be safely updated or no stable hero binary exists.
func PreflightDeepSeekHome(root string) error {
	path, err := DeepSeekHomePatchPath()
	if err != nil {
		return err
	}
	content, _, err := readDeepSeekHomePatch(path)
	if err != nil {
		return err
	}
	server, err := DeepSeekServerName(root)
	if err != nil {
		return err
	}
	if _, _, _, err := cutDeepSeekHomeBlock(content, server); err != nil {
		return err
	}
	_, err = resolveDeepSeekHeroCommand()
	return err
}

// UpsertDeepSeekHomeEntry writes (or refreshes) this project's Hero MCP block
// in the DeepSeek home patch. Repeat runs with the same binary are no-ops.
func UpsertDeepSeekHomeEntry(root string, dryRun bool) (path, server string, changed bool, err error) {
	if path, err = DeepSeekHomePatchPath(); err != nil {
		return
	}
	command, err := resolveDeepSeekHeroCommand()
	if err != nil {
		return
	}
	content, exists, err := readDeepSeekHomePatch(path)
	if err != nil {
		return
	}
	server, block, err := deepseekHomeBlock(root, command)
	if err != nil {
		return
	}
	rest, old, found, err := cutDeepSeekHomeBlock(content, server)
	if err != nil {
		return
	}
	if found && old == block {
		return path, server, false, nil
	}
	var next string
	if found {
		// Replace in place so the file's order is stable.
		i := strings.Index(content, old)
		next = content[:i] + block + content[i+len(old):]
	} else {
		if !exists {
			rest = deepseekHomeCreatedLine
		} else if rest != "" && !strings.HasSuffix(rest, "\n") {
			rest += "\n"
		}
		next = rest + block
	}
	if dryRun {
		return path, server, true, nil
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	return path, server, true, os.WriteFile(path, []byte(next), 0o644)
}

// RemoveDeepSeekHomeEntry removes only this project's block. The file itself
// is removed only when Hero created it and nothing else remains.
func RemoveDeepSeekHomeEntry(root string, dryRun bool) (bool, error) {
	path, err := DeepSeekHomePatchPath()
	if err != nil {
		return false, err
	}
	content, exists, err := readDeepSeekHomePatch(path)
	if err != nil || !exists {
		return false, err
	}
	server, err := DeepSeekServerName(root)
	if err != nil {
		return false, err
	}
	rest, _, found, err := cutDeepSeekHomeBlock(content, server)
	if err != nil || !found || dryRun {
		return found, err
	}
	if strings.HasPrefix(content, deepseekHomeCreatedLine) && strings.TrimSpace(strings.TrimPrefix(rest, deepseekHomeCreatedLine)) == "" {
		return true, os.Remove(path)
	}
	return true, os.WriteFile(path, []byte(rest), 0o644)
}

// DeepSeekRegistration describes this project's home-patch MCP entry.
type DeepSeekRegistration struct {
	Path       string
	ServerName string
	Command    string
	Problem    string // "" when registered and runnable
}

// InspectDeepSeekRegistration reports whether this project's Hero MCP entry
// is present, points at an executable hero, and serves this project root.
func InspectDeepSeekRegistration(root string) DeepSeekRegistration {
	var reg DeepSeekRegistration
	var err error
	if reg.Path, err = DeepSeekHomePatchPath(); err != nil {
		reg.Problem = err.Error()
		return reg
	}
	if reg.ServerName, err = DeepSeekServerName(root); err != nil {
		reg.Problem = err.Error()
		return reg
	}
	content, _, err := readDeepSeekHomePatch(reg.Path)
	if err != nil {
		reg.Problem = err.Error()
		return reg
	}
	_, block, found, err := cutDeepSeekHomeBlock(content, reg.ServerName)
	if err != nil {
		reg.Problem = err.Error()
		return reg
	}
	if !found {
		reg.Problem = "no Hero MCP entry for this project — run `hero install project . --target deepseek`"
		return reg
	}
	var patch []struct {
		Insert []deepseekMCPEntry `yaml:"insert"`
	}
	if err := yaml.Unmarshal([]byte(block), &patch); err != nil || len(patch) != 1 || len(patch[0].Insert) != 1 {
		reg.Problem = "Hero MCP entry is malformed — rerun `hero install project . --target deepseek`"
		return reg
	}
	entry := patch[0].Insert[0]
	reg.Command = entry.Config.Command
	canonical, _ := canonicalProjectRoot(root)
	if info, err := os.Stat(reg.Command); err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		reg.Problem = fmt.Sprintf("command %s is not an executable hero binary — rerun install after `make install`", reg.Command)
	} else if entry.Config.Cwd != canonical {
		reg.Problem = fmt.Sprintf("entry serves %s, not this project", entry.Config.Cwd)
	}
	return reg
}
