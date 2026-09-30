package install

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DeepSeekHome follows dsh's DSH_HOME expansion, relative to invocation cwd.
func DeepSeekHome() (string, error) {
	value := os.Getenv("DSH_HOME")
	if strings.TrimSpace(value) == "" {
		value = ""
	}
	if value == "" || value == "~" || strings.HasPrefix(value, "~/") || strings.HasPrefix(value, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if value == "" {
			value = filepath.Join(home, ".dsh")
		} else if value == "~" {
			value = home
		} else {
			value = filepath.Join(home, value[2:])
		}
	}
	return filepath.Abs(value)
}

func deepseekBase(opts Options) (string, error) {
	if opts.Mode == ModeGlobal {
		return DeepSeekHome()
	}
	if opts.Mode != ModeProject || opts.TargetDir == "" {
		return "", fmt.Errorf("deepseek requires project directory or global mode")
	}
	info, err := os.Stat(opts.TargetDir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("target directory does not exist: %s", opts.TargetDir)
	}
	return filepath.Abs(filepath.Join(opts.TargetDir, ".dsh"))
}

type deepseekFile struct {
	data   []byte
	source string
}

func deepseekFiles(opts Options) (map[string]deepseekFile, []string, error) {
	files := map[string]deepseekFile{}
	names := map[string]bool{}
	add := func(name, source string, data []byte) error {
		if names[name] {
			return fmt.Errorf("deepseek skill namespace collision: %s", name)
		}
		names[name] = true
		files["skills/"+name+"/SKILL.md"] = deepseekFile{data, source}
		return nil
	}
	skills, err := selectSkillContent(opts.sourceFS())
	if err != nil {
		return nil, nil, err
	}
	for _, s := range skills {
		data, err := fs.ReadFile(opts.sourceFS(), s.SourcePath)
		if err != nil {
			return nil, nil, err
		}
		if err = add(s.Name, s.SourcePath, data); err != nil {
			return nil, nil, err
		}
	}
	domain := opts.Domain
	if domain == "" {
		domain = "engineering"
	}
	for _, kind := range []string{"commands", "agents"} {
		entries, err := selectFlatContent(opts.sourceFS(), kind, domain)
		if err != nil {
			return nil, nil, err
		}
		for _, name := range entries {
			src := kind + "/" + name
			raw, err := fs.ReadFile(opts.sourceFS(), src)
			if err != nil {
				return nil, nil, err
			}
			fm, body := parseSimpleFrontmatter(raw)
			entry := canonicalEntry{Name: strings.TrimSuffix(name, ".md"), SourcePath: src, Frontmatter: fm, Body: body, Raw: raw}
			var dest string
			var data []byte
			if kind == "agents" {
				dest, data, err = renderDeepSeekRoleSkill(entry)
			} else {
				dest, data, err = commandAsSkillRenderer("DeepSeek")(entry)
			}
			if err != nil {
				return nil, nil, err
			}
			if err = add(strings.TrimSuffix(dest, "/SKILL.md"), src, data); err != nil {
				return nil, nil, err
			}
		}
	}
	overlay, err := deepseekOverlay(opts)
	if err != nil {
		return nil, nil, err
	}
	files[deepseekOverlayName] = deepseekFile{overlay, "Hero MCP overlay"}
	dirs := make([]string, 0, len(names))
	for name := range names {
		dirs = append(dirs, name)
	}
	sort.Strings(dirs)
	return files, dirs, nil
}

// deepseekPlan is the rendered, collision-checked DeepSeek file set.
type deepseekPlan struct {
	base  string
	files map[string]deepseekFile
	dirs  []string
	prior map[string]string
}

// planDeepSeek renders the owned file set and fails before any mutation when
// a destination is unknown or modified.
func planDeepSeek(opts Options) (*deepseekPlan, error) {
	base, err := deepseekBase(opts)
	if err != nil {
		return nil, err
	}
	files, dirs, err := deepseekFiles(opts)
	if err != nil {
		return nil, err
	}
	prior, err := deepseekChecksums(opts, base)
	if err != nil {
		return nil, err
	}
	if err = preflightDeepSeek(opts, base, files, prior); err != nil {
		return nil, err
	}
	return &deepseekPlan{base: base, files: files, dirs: dirs, prior: prior}, nil
}

func runDeepSeek(opts Options, plan *deepseekPlan) (*Result, error) {
	base, files, dirs, prior := plan.base, plan.files, plan.dirs, plan.prior
	var err error
	result := &Result{skillDirs: dirs}
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		file := files[key]
		dest := filepath.Join(base, filepath.FromSlash(key))
		result.deepseekOwned = append(result.deepseekOwned, dest)
		old, _ := os.ReadFile(dest)
		if bytes.Equal(old, file.data) {
			continue
		}
		result.Copied = append(result.Copied, CopyAction{Source: file.source, Dest: dest})
		if opts.DryRun {
			continue
		}
		if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return result, err
		}
		if err = os.WriteFile(dest, file.data, 0644); err != nil {
			return result, err
		}
	}
	if err = pruneDeepSeek(opts, base, files, prior); err != nil {
		return result, err
	}
	if err = installNativeInstructionFile(opts, result); err != nil {
		return result, err
	}
	if opts.Mode == ModeGlobal && !opts.DryRun {
		if err = writeDeepSeekManifest(base, files); err != nil {
			return result, err
		}
	}
	if !opts.Quiet {
		fmt.Printf("  DeepSeek overlay generated; activation required. Launch from the intended workspace:\n  %s\n", DeepSeekLaunchCommand(filepath.Join(base, deepseekOverlayName)))
	}
	return result, nil
}
