package install

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
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
	if len(name) > 20 {
		name = strings.TrimRight(name[:20], "-")
	}
	sum := sha256.Sum256([]byte(rootIdentity(canonical)))
	return "hero-" + name + "-" + hex.EncodeToString(sum[:])[:6], nil
}

// rootIdentity folds case on the platforms whose default filesystems are
// case-insensitive, so two spellings of one project never get two servers.
func rootIdentity(root string) string {
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return strings.ToLower(root)
	}
	return root
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
		if abs, err := filepath.Abs(found); err == nil && !transientExecutable(abs) {
			return abs, nil
		}
	}
	if self, err := os.Executable(); err == nil && !transientExecutable(self) {
		return self, nil
	}
	return "", fmt.Errorf("cannot find a stable `hero` binary for DeepSeek: put hero on PATH (e.g. `make install`) or set %s", deepseekCommandEnv)
}

// transientExecutable reports go-build and temp-directory binaries, which
// disappear and must never be written into the DeepSeek home patch. Paths are
// compared both as given and resolved (macOS /tmp and /var are symlinks).
func transientExecutable(path string) bool {
	candidates := []string{path}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		candidates = append(candidates, resolved)
	}
	prefixes := []string{"/tmp/", "/private/tmp/", "/var/folders/", "/private/var/folders/"}
	for _, tmp := range []string{os.TempDir()} {
		prefixes = append(prefixes, strings.TrimRight(filepath.ToSlash(tmp), "/")+"/")
		if resolved, err := filepath.EvalSymlinks(tmp); err == nil {
			prefixes = append(prefixes, strings.TrimRight(filepath.ToSlash(resolved), "/")+"/")
		}
	}
	for _, c := range candidates {
		slash := filepath.ToSlash(c)
		if strings.Contains(slash, "/go-build") {
			return true
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(slash, prefix) {
				return true
			}
		}
	}
	return false
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

// deepseekAddedNewline flags a start marker whose block needed a separating
// newline (the file did not end with one), so removal can restore the exact
// original bytes.
const deepseekAddedNewline = " (added-newline)"

func deepseekHomeEntry(root, command string) (deepseekMCPEntry, error) {
	var entry deepseekMCPEntry
	canonical, err := canonicalProjectRoot(root)
	if err != nil {
		return entry, err
	}
	server, err := DeepSeekServerName(canonical)
	if err != nil {
		return entry, err
	}
	entry.ID, entry.Name = server+"-mcp", deepseekMCPPluginName
	entry.Config.ServerName, entry.Config.Transport = server, "stdio"
	entry.Config.Command = command
	entry.Config.Args = []string{"mcp", "--project-root", canonical}
	entry.Config.Cwd = canonical
	return entry, nil
}

// insertDeepSeekHomeBlock places entry's marked block at pos in rest,
// adding (and flagging) a separating newline only when one is missing.
func insertDeepSeekHomeBlock(rest string, pos int, entry deepseekMCPEntry) (string, error) {
	data, err := deepseekMCPPatch(entry)
	if err != nil {
		return "", err
	}
	start, end := deepseekHomeMarkers(entry.Config.ServerName)
	prefix := ""
	if pos > 0 && rest[pos-1] != '\n' {
		prefix, start = "\n", start+deepseekAddedNewline
	}
	return rest[:pos] + prefix + start + "\n" + string(data) + end + "\n" + rest[pos:], nil
}

// cutDeepSeekHomeBlock removes every block for server, returning the
// remaining bytes, the first block's text, and where it started in rest.
// Markers only count as whole lines (CRLF tolerated), so marker-like text in
// a foreign comment is never matched; an unterminated block is refused.
func cutDeepSeekHomeBlock(content, server string) (rest, block string, pos int, found bool, err error) {
	start, end := deepseekHomeMarkers(server)
	rest, pos = content, -1
	for {
		i, lineEnd, added := -1, 0, false
		for off := 0; off < len(rest); {
			next := strings.IndexByte(rest[off:], '\n')
			stop := len(rest)
			if next >= 0 {
				stop = off + next + 1
			}
			line := strings.TrimRight(rest[off:stop], "\r\n")
			if i < 0 && (line == start || line == start+deepseekAddedNewline) {
				i, added = off, line != start
			} else if i >= 0 && line == end {
				lineEnd = stop
				break
			}
			off = stop
		}
		if i < 0 {
			return rest, block, pos, found, nil
		}
		if lineEnd == 0 {
			return "", "", 0, false, fmt.Errorf("unterminated Hero block %q in DeepSeek home patch", server)
		}
		if added && i > 0 && rest[i-1] == '\n' {
			i--
		}
		if !found {
			block, pos, found = rest[i:lineEnd], i, true
		}
		rest = rest[:i] + rest[lineEnd:]
	}
}

// deepseekPatchList parses a home patch the way DeepSeek requires: a YAML
// list (empty or absent counts as zero entries).
func deepseekPatchList(content string) ([]interface{}, error) {
	var parsed interface{}
	if err := yaml.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, err
	}
	if parsed == nil {
		return nil, nil
	}
	list, ok := parsed.([]interface{})
	if !ok {
		return nil, fmt.Errorf("must be a YAML list of patch entries")
	}
	return list, nil
}

// countDeepSeekEntry counts list elements inserting the given entry id.
func countDeepSeekEntry(list []interface{}, id string) int {
	n := 0
	for _, item := range list {
		m, _ := item.(map[string]interface{})
		inserts, _ := m["insert"].([]interface{})
		for _, ins := range inserts {
			if e, _ := ins.(map[string]interface{}); e != nil && e["id"] == id {
				n++
			}
		}
	}
	return n
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
	if _, err := deepseekPatchList(string(data)); err != nil {
		return "", false, fmt.Errorf("DeepSeek home patch %s: %w", path, err)
	}
	return string(data), true, nil
}

// PreflightDeepSeekHome fails before any install mutation when the home patch
// cannot be safely updated or no stable hero binary exists.
func PreflightDeepSeekHome(root string) error {
	_, _, _, err := UpsertDeepSeekHomeEntry(root, true)
	return err
}

// UpsertDeepSeekHomeEntry writes (or refreshes) this project's Hero MCP block
// in the DeepSeek home patch. Repeat runs with the same binary are no-ops.
// The result is re-parsed before writing: a layout Hero cannot extend as
// text (flow-style list, indented list, explicit document end) is refused,
// because an unparseable home patch stops every DeepSeek profile booting.
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
	entry, err := deepseekHomeEntry(root, command)
	if err != nil {
		return
	}
	server = entry.Config.ServerName
	rest, _, pos, found, err := cutDeepSeekHomeBlock(content, server)
	if err != nil {
		return
	}
	if !found {
		if !exists {
			rest = deepseekHomeCreatedLine
		}
		pos = len(rest)
	}
	next, err := insertDeepSeekHomeBlock(rest, pos, entry)
	if err != nil {
		return
	}
	if next == content {
		return path, server, false, nil
	}
	before, err := deepseekPatchList(rest)
	if err != nil {
		err = fmt.Errorf("DeepSeek home patch %s: %w", path, err)
		return
	}
	after, perr := deepseekPatchList(next)
	if perr != nil || len(after) != len(before)+1 || countDeepSeekEntry(after, entry.ID) != 1 {
		err = fmt.Errorf("cannot safely add Hero to DeepSeek home patch %s: its layout (e.g. flow-style [], an indented list, or a `...` document end) cannot be extended without rewriting your entries; convert it to a block-style YAML list and rerun", path)
		return
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
	rest, _, _, found, err := cutDeepSeekHomeBlock(content, server)
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
	_, block, _, found, err := cutDeepSeekHomeBlock(content, reg.ServerName)
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
	} else if rootIdentity(entry.Config.Cwd) != rootIdentity(canonical) {
		reg.Problem = fmt.Sprintf("entry serves %s, not this project", entry.Config.Cwd)
	}
	return reg
}
