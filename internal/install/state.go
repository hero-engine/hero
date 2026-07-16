package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/hero-engine/hero/internal/managed"
	"github.com/hero-engine/hero/internal/version"
)

// state.go — install-state tracking.
//
// `.hero/install-state.json` records what Hero did during install/upgrade for
// each target, so subsequent runs can produce idempotent no-ops, detect
// capability changes (e.g. user enabled Windows Developer Mode), drive drift
// detection in rendered-copy mode, and inform `hero install --migrate`.
//
// Today this file is mostly forward-looking — single-source-install P1/P2
// will write real per-target install modes here. The scaffolding lands now
// so the format stabilizes early and future work just adds fields.

// InstallState is the on-disk shape of `.hero/install-state.json`.
type InstallState struct {
	// SchemaVersion lets us evolve the file format compatibly. Bump only on
	// breaking changes; additive changes leave it unchanged.
	SchemaVersion int `json:"schema_version"`

	// HeroVersion is the binary version that wrote this state file.
	HeroVersion string `json:"hero_version"`

	// UpdatedAt is when state was last written (RFC3339).
	UpdatedAt string `json:"updated_at"`

	// HostCapabilities records filesystem and OS capabilities relevant to
	// install mode selection (P2). Populated by the symlink-capability probe.
	HostCapabilities HostCapabilities `json:"host_capabilities"`

	// Targets maps target name → per-target state. One entry per harness
	// that has been installed into this workspace. Removal of a target
	// clears its entry.
	Targets map[string]TargetState `json:"targets"`
}

// HostCapabilities is what `hero install --probe` (P2) writes. Stub today.
type HostCapabilities struct {
	SymlinksSupported bool   `json:"symlinks_supported"`
	OS                string `json:"os,omitempty"`
	Arch              string `json:"arch,omitempty"`
	ProbedAt          string `json:"probed_at,omitempty"`
}

// TargetState is per-target install metadata.
type TargetState struct {
	// Mode is how this target's content is materialized:
	//   "rendered"        — physical copies (legacy / Cline / no-symlinks)
	//   "symlink"         — directory symlinks into .hero/  (P2 default for most targets)
	//   "config-redirect" — harness config points at .hero/ (P2 default for opencode/codex/aider)
	Mode string `json:"mode"`

	// InstalledAt is when this target was first installed (RFC3339).
	InstalledAt string `json:"installed_at"`

	// LastUpdatedAt is the most recent install/upgrade time (RFC3339).
	LastUpdatedAt string `json:"last_updated_at"`

	// HeroVersion is the binary version of the most recent install for
	// this target.
	HeroVersion string `json:"hero_version"`

	// SkillDirs is the set of skill directory names the most recent
	// install wrote at this target's nested-skills destination. The next
	// install prunes entries recorded here that it no longer writes — the
	// proof that a leftover dir is Hero's to remove rather than the
	// user's to keep. See prune.go. Absent for targets with no nested
	// skills dest (cursor, copilot, generic).
	SkillDirs []string `json:"skill_dirs,omitempty"`
}

const installStateSchemaVersion = 1

// InstallStatePath returns the canonical path for the install-state file.
// Returns "" if there's no .hero/ workspace at projectRoot.
func InstallStatePath(projectRoot string) string {
	if projectRoot == "" {
		return ""
	}
	heroDir := filepath.Join(projectRoot, ".hero")
	if _, err := os.Stat(heroDir); err != nil {
		return ""
	}
	return filepath.Join(heroDir, "install-state.json")
}

// ReadInstallState loads the install-state file. Returns a zero-valued state
// (not an error) if the file doesn't exist — that's the legitimate
// "uninstalled" / "pre-state-file" condition.
func ReadInstallState(projectRoot string) (*InstallState, error) {
	path := InstallStatePath(projectRoot)
	if path == "" {
		return &InstallState{SchemaVersion: installStateSchemaVersion, Targets: map[string]TargetState{}}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &InstallState{SchemaVersion: installStateSchemaVersion, Targets: map[string]TargetState{}}, nil
		}
		return nil, fmt.Errorf("reading install-state: %w", err)
	}
	var st InstallState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("parsing install-state: %w", err)
	}
	if st.Targets == nil {
		st.Targets = map[string]TargetState{}
	}
	return &st, nil
}

// WriteInstallState persists the install-state file under .hero/. No-op if
// no .hero/ workspace exists (we don't create the directory just to write
// this file).
func WriteInstallState(projectRoot string, st *InstallState) error {
	if st == nil {
		return fmt.Errorf("install state is nil")
	}
	path := InstallStatePath(projectRoot)
	if path == "" {
		return nil
	}
	st.SchemaVersion = installStateSchemaVersion
	st.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// RecordTargetInstall stamps the install-state file with metadata for the
// just-completed install. Best-effort: errors here are non-fatal because the
// install itself succeeded.
func RecordTargetInstall(opts Options, mode string, result *Result) {
	if opts.DryRun || opts.Mode != ModeProject || opts.TargetDir == "" {
		return
	}
	st, err := ReadInstallState(opts.TargetDir)
	if err != nil {
		fmt.Printf("  warning: could not read install-state: %v\n", err)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	target := string(opts.Target)
	prior, hadPrior := st.Targets[target]
	installedAt := now
	if hadPrior && prior.InstalledAt != "" {
		installedAt = prior.InstalledAt
	}
	ver := opts.Version
	if ver == "" {
		ver = "dev"
	}
	// Skill-dir manifest for the next run's prune. Sorted so re-installs
	// produce a stable file. Targets with no nested-skills dest leave it
	// empty and `omitempty` drops the field.
	var skillDirs []string
	if result != nil && len(result.skillDirs) > 0 {
		skillDirs = append(skillDirs, result.skillDirs...)
		sort.Strings(skillDirs)
	}
	st.Targets[target] = TargetState{
		Mode:          mode,
		InstalledAt:   installedAt,
		LastUpdatedAt: now,
		HeroVersion:   ver,
		SkillDirs:     skillDirs,
	}
	st.HeroVersion = ver

	// Update host capabilities. The probe runs lazily inside
	// hostSupportsSymlinks(); we record whatever it returned.
	st.HostCapabilities = HostCapabilities{
		SymlinksSupported: hostSupportsSymlinks(),
		OS:                runtime.GOOS,
		Arch:              runtime.GOARCH,
		ProbedAt:          now,
	}

	if err := WriteInstallState(opts.TargetDir, st); err != nil {
		fmt.Printf("  warning: could not write install-state: %v\n", err)
	}
}

// PreviouslyInstalledTargets returns the set of targets recorded in
// install-state.json `targets` — the authoritative previously-installed set
// for upgrade. A key present in the map means that target was installed and
// its native instruction file must be maintained. Returns nil when no state
// file exists or no targets are recorded. Order is the stable targetLayouts
// order (claude first) so callers get deterministic output.
func PreviouslyInstalledTargets(projectRoot string) []Target {
	st, err := ReadInstallState(projectRoot)
	if err != nil || st == nil || len(st.Targets) == 0 {
		return nil
	}
	var out []Target
	for _, layout := range targetLayouts {
		if _, ok := st.Targets[string(layout.Target)]; ok {
			out = append(out, layout.Target)
		}
	}
	return out
}

// InferInstalledTargets reconstructs the prior installed-target set for a
// repo that has no persisted `targets` (a pre-state install). It combines:
//
//  1. Content-dir probe (DetectInstalledTargets) — AUTHORITATIVE for the
//     SET. A harness content dir (.claude/, .codex/, …) proves that target
//     was installed.
//  2. Instruction-file presence — a SECONDARY signal used only to keep an
//     existing Hero-managed file maintained, never to invent a target. A
//     CLAUDE.md carrying a Hero managed region implies Claude was a target
//     (covers the legacy Hero-managed-stub case with no content dir). A lone
//     AGENTS.md is deliberately NOT adopted as evidence of a specific
//     non-Claude target — the upgrade orphan-maintain path handles it
//     instead, so a phantom Model-B AGENTS.md never conjures a phantom
//     target.
//
// Returns targets in stable targetLayouts order, deduped.
func InferInstalledTargets(projectRoot string) []Target {
	seen := map[Target]bool{}
	var out []Target
	add := func(t Target) {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	for _, t := range DetectInstalledTargets(projectRoot) {
		add(t)
	}
	if claudeMdIsHeroManaged(projectRoot) {
		add(TargetClaude)
	}
	return out
}

// claudeMdIsHeroManaged reports whether a CLAUDE.md at projectRoot exists and
// carries a Hero managed region — the signal that Claude was a Hero target
// even when no .claude/ content dir survives.
func claudeMdIsHeroManaged(projectRoot string) bool {
	data, err := os.ReadFile(filepath.Join(projectRoot, "CLAUDE.md"))
	if err != nil {
		return false
	}
	return managed.FindManagedRegion(string(data)).Present
}

// PersistInferredTargets records a backfilled/inferred target set into
// install-state.json `targets`, so the next upgrade reads a persisted set
// instead of re-inferring. Existing entries are preserved (their
// installed_at and mode are kept); inferred-new entries get installed_at =
// last_updated_at = now and the given hero version. Best-effort: no-op when
// the set is empty or no .hero/ workspace exists.
func PersistInferredTargets(projectRoot string, targets []Target, heroVersion string) error {
	if len(targets) == 0 || InstallStatePath(projectRoot) == "" {
		return nil
	}
	st, err := ReadInstallState(projectRoot)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	ver := heroVersion
	if ver == "" {
		ver = "dev"
	}
	for _, t := range targets {
		key := string(t)
		prior, had := st.Targets[key]
		installedAt := now
		mode := "rendered"
		var skillDirs []string
		if had {
			if prior.InstalledAt != "" {
				installedAt = prior.InstalledAt
			}
			if prior.Mode != "" {
				mode = prior.Mode
			}
			// Backfilling a target set says nothing about its skills; keep
			// the prune manifest an install wrote (see prune.go).
			skillDirs = prior.SkillDirs
		}
		st.Targets[key] = TargetState{
			Mode:          mode,
			InstalledAt:   installedAt,
			LastUpdatedAt: now,
			HeroVersion:   ver,
			SkillDirs:     skillDirs,
		}
	}
	if st.HeroVersion == "" {
		st.HeroVersion = ver
	}
	return WriteInstallState(projectRoot, st)
}

// StampInstallVersion writes version and checksum info to .hero/version.json.
// This is a best-effort operation; errors are logged but don't fail the install.
// Predates install-state.json; the two coexist — version.json carries
// per-file checksum information for drift detection, install-state.json
// carries the install-mode metadata.
func StampInstallVersion(opts Options, result *Result) {
	heroDir := filepath.Join(opts.TargetDir, ".hero")
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return // no .hero directory = not a hero workspace
	}

	checksums := make(map[string]string)
	for _, action := range result.Copied {
		relPath, err := filepath.Rel(opts.TargetDir, action.Dest)
		if err != nil {
			continue
		}
		cs, err := version.FileChecksum(action.Dest)
		if err != nil {
			continue
		}
		checksums[relPath] = cs
	}

	ver := opts.Version
	if ver == "" {
		ver = "dev"
	}

	if err := version.StampInstall(heroDir, ver, string(opts.Target), string(opts.Mode), checksums); err != nil {
		fmt.Printf("  warning: could not write version stamp: %v\n", err)
	}
}
