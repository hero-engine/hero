package install

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/version"
	"gopkg.in/yaml.v3"
)

const deepseekOverlayName = "hero.cordis.patch.yml"

// DeepSeekOverlayPath identifies the explicitly activated Cordis overlay.
func DeepSeekOverlayPath(opts Options) (string, error) {
	base, err := deepseekBase(opts)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, deepseekOverlayName), nil
}

// DeepSeekLaunchCommand is the interactive (web profile) session launch, with the patch
// path quoted for a POSIX shell. It carries no prompt so it never starts a
// one-shot model run.
func DeepSeekLaunchCommand(path string) string {
	return "dsh --profile web --patch '" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}

func deepseekOverlay(opts Options) ([]byte, error) {
	args := []string{"mcp"}
	if opts.ProjectRoot != "" {
		args = append(args, "--project-root", opts.ProjectRoot)
	}
	type plugin struct {
		ID     string `yaml:"id"`
		Name   string `yaml:"name"`
		Config struct {
			ServerName string   `yaml:"serverName"`
			Transport  string   `yaml:"transport"`
			Command    string   `yaml:"command"`
			Args       []string `yaml:"args"`
		} `yaml:"config"`
	}
	p := plugin{ID: "hero-mcp", Name: "@deepseek-ai/dsh-mcp-client"}
	p.Config.ServerName = "hero"
	p.Config.Transport = "stdio"
	p.Config.Command = heroCommand
	p.Config.Args = args
	return yaml.Marshal([]struct {
		Insert []plugin `yaml:"insert"`
	}{{Insert: []plugin{p}}})
}

func registerMCPDeepSeek(opts Options) error {
	base, err := deepseekBase(opts)
	if err != nil {
		return err
	}
	data, err := deepseekOverlay(opts)
	if err != nil {
		return err
	}
	prior, err := deepseekChecksums(opts, base)
	if err != nil {
		return err
	}
	files := map[string]deepseekFile{deepseekOverlayName: {data, "Hero MCP overlay"}}
	if err = preflightDeepSeek(opts, base, files, prior); err != nil {
		return err
	}
	if opts.DryRun {
		return nil
	}
	if err = os.MkdirAll(base, 0755); err != nil {
		return err
	}
	path := filepath.Join(base, deepseekOverlayName)
	old, _ := os.ReadFile(path)
	if !bytes.Equal(old, data) {
		if err = os.WriteFile(path, data, 0644); err != nil {
			return err
		}
	}
	if opts.Mode == ModeProject && opts.ProjectRoot != "" && InstallStatePath(opts.ProjectRoot) != "" {
		root, err := filepath.Abs(opts.ProjectRoot)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sum, err := version.FileChecksum(path)
		if err != nil {
			return err
		}
		return version.StampInstall(filepath.Join(opts.ProjectRoot, ".hero"), opts.Version, string(TargetDeepSeek), string(ModeProject), map[string]string{filepath.ToSlash(rel): sum})
	}
	return nil
}

// RemoveDeepSeekOverlay removes only checksum-owned overlay bytes, unless forced.
func RemoveDeepSeekOverlay(projectRoot string, dryRun, force bool) (bool, error) {
	opts := Options{Target: TargetDeepSeek, Mode: ModeProject, TargetDir: projectRoot}
	base, err := deepseekBase(opts)
	if err != nil {
		return false, err
	}
	path := filepath.Join(base, deepseekOverlayName)
	if _, err = os.Lstat(path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	prior, err := deepseekChecksums(opts, base)
	if err != nil {
		return false, err
	}
	if deepseekHasSymlinkAncestor(path, base) || (!force && !deepseekUnchanged(path, prior[deepseekOverlayName])) {
		return false, nil
	}
	if !dryRun {
		err = os.Remove(path)
	}
	return err == nil, err
}
