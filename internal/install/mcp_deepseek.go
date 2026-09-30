package install

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const deepseekOverlayName = "hero.cordis.patch.yml"

// DeepSeekMCPConfigPath is where install registers Hero's MCP server: the
// home patch for project installs, the --patch overlay for global installs.
func DeepSeekMCPConfigPath(opts Options) (string, error) {
	if opts.Mode == ModeProject {
		return DeepSeekHomePatchPath()
	}
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

// deepseekOverlay is the global-mode overlay, activated with `dsh --patch`.
// It keeps the portable command because it is not bound to one project.
func deepseekOverlay(opts Options) ([]byte, error) {
	var entry deepseekMCPEntry
	entry.ID, entry.Name = "hero-mcp", deepseekMCPPluginName
	entry.Config.ServerName, entry.Config.Transport = "hero", "stdio"
	entry.Config.Command = heroCommand
	entry.Config.Args = []string{"mcp"}
	if opts.ProjectRoot != "" {
		entry.Config.Args = append(entry.Config.Args, "--project-root", opts.ProjectRoot)
	}
	return deepseekMCPPatch(entry)
}

func registerMCPDeepSeek(opts Options) error {
	if opts.Mode == ModeProject {
		root := opts.ProjectRoot
		if root == "" {
			root = opts.TargetDir
		}
		path, server, changed, err := UpsertDeepSeekHomeEntry(root, opts.DryRun)
		if err != nil {
			return err
		}
		if !opts.Quiet && changed {
			verb := "registered"
			if opts.DryRun {
				verb = "would register"
			}
			fmt.Printf("  DeepSeek MCP server %s %s in %s\n", server, verb, path)
		}
		return nil
	}
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
