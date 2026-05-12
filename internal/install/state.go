package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

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
func RecordTargetInstall(opts Options, mode string) {
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
	st.Targets[target] = TargetState{
		Mode:          mode,
		InstalledAt:   installedAt,
		LastUpdatedAt: now,
		HeroVersion:   ver,
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
