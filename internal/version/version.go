// Package version manages Hero workspace version tracking, mismatch detection,
// and file checksum tracking for smart upgrades.
package version

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const VersionFileName = "version.json"

// Info holds the version state for a Hero workspace.
type Info struct {
	// HeroVersion is the version of the hero binary that last wrote this file.
	HeroVersion string `json:"hero_version"`

	// InitializedAt is when hero init first created this workspace.
	InitializedAt time.Time `json:"initialized_at"`

	// LastInstall records the most recent hero install invocation.
	LastInstall *InstallRecord `json:"last_install,omitempty"`

	// LastUpgrade records the most recent hero upgrade invocation.
	LastUpgrade *UpgradeRecord `json:"last_upgrade,omitempty"`

	// InstalledFiles maps relative file paths to their SHA-256 checksums
	// as of the last install or upgrade. Used for smart diffing.
	InstalledFiles map[string]string `json:"installed_files,omitempty"`
}

// InstallRecord captures details of the last hero install.
type InstallRecord struct {
	Version   string    `json:"version"`
	Target    string    `json:"target"`
	Mode      string    `json:"mode"`
	Timestamp time.Time `json:"timestamp"`
}

// UpgradeRecord captures details of the last hero upgrade.
type UpgradeRecord struct {
	FromVersion string    `json:"from_version"`
	ToVersion   string    `json:"to_version"`
	Timestamp   time.Time `json:"timestamp"`
	Updated     int       `json:"updated"`
	Skipped     int       `json:"skipped"`
}

// Read loads version.json from the given .hero directory.
// Returns nil, nil if the file does not exist (pre-version workspace).
func Read(heroDir string) (*Info, error) {
	path := filepath.Join(heroDir, VersionFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var info Info
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &info, nil
}

// Write persists version.json to the given .hero directory.
func Write(heroDir string, info *Info) error {
	path := filepath.Join(heroDir, VersionFileName)

	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling version info: %w", err)
	}
	data = append(data, '\n')

	return os.WriteFile(path, data, 0o644)
}

// StampInit creates or updates version.json for a hero init.
func StampInit(heroDir, binaryVersion string) error {
	info, _ := Read(heroDir)
	if info == nil {
		info = &Info{
			InitializedAt: time.Now(),
		}
	}
	info.HeroVersion = binaryVersion
	return Write(heroDir, info)
}

// StampInstall updates version.json after a hero install.
func StampInstall(heroDir, binaryVersion, target, mode string, files map[string]string) error {
	info, _ := Read(heroDir)
	if info == nil {
		info = &Info{
			InitializedAt: time.Now(),
		}
	}
	info.HeroVersion = binaryVersion
	info.LastInstall = &InstallRecord{
		Version:   binaryVersion,
		Target:    target,
		Mode:      mode,
		Timestamp: time.Now(),
	}
	if files != nil {
		if info.InstalledFiles == nil {
			info.InstalledFiles = make(map[string]string)
		}
		for k, v := range files {
			info.InstalledFiles[k] = v
		}
	}
	return Write(heroDir, info)
}

// StampUpgrade updates version.json after a hero upgrade.
func StampUpgrade(heroDir string, fromVersion, toVersion string, updated, skipped int) error {
	info, _ := Read(heroDir)
	if info == nil {
		info = &Info{
			InitializedAt: time.Now(),
		}
	}
	info.HeroVersion = toVersion
	info.LastUpgrade = &UpgradeRecord{
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Timestamp:   time.Now(),
		Updated:     updated,
		Skipped:     skipped,
	}
	return Write(heroDir, info)
}

// CompareVersions performs semantic version comparison.
// Returns -1 if a < b, 0 if equal, 1 if a > b.
// Handles versions with different component counts (e.g. "0.3" vs "0.3.0").
func CompareVersions(a, b string) int {
	aParts := parseVersionParts(a)
	bParts := parseVersionParts(b)

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		ai := 0
		bi := 0
		if i < len(aParts) {
			ai = aParts[i]
		}
		if i < len(bParts) {
			bi = bParts[i]
		}
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}
	return 0
}

func parseVersionParts(v string) []int {
	v = cleanVersion(v)
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if n, err := atoi(p); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// Mismatch compares the workspace version against the running binary version.
// Returns a human-readable warning if they differ, or empty string if matched.
func Mismatch(heroDir, binaryVersion string) string {
	if binaryVersion == "" || binaryVersion == "dev" {
		return "" // dev builds don't warn
	}

	info, err := Read(heroDir)
	if err != nil || info == nil {
		return "" // no version file = pre-version workspace, don't warn
	}

	if info.HeroVersion == "" || info.HeroVersion == "dev" {
		return "" // workspace was created with dev build
	}

	if info.HeroVersion == binaryVersion {
		return "" // versions match
	}

	wsVer := cleanVersion(info.HeroVersion)
	binVer := cleanVersion(binaryVersion)

	cmp := CompareVersions(binaryVersion, info.HeroVersion)
	if cmp > 0 {
		return fmt.Sprintf("workspace was created with v%s, binary is v%s — run 'hero upgrade' to update",
			wsVer, binVer)
	}

	return fmt.Sprintf("workspace was last used with v%s, binary is v%s — downgrade detected (no action needed)",
		wsVer, binVer)
}

// FileChecksum computes the SHA-256 checksum of a file.
func FileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// IsFileModified checks whether a file has been modified since the last
// install/upgrade by comparing its current checksum against the stored one.
// Returns true if modified or if no stored checksum exists.
func IsFileModified(info *Info, relPath, absPath string) bool {
	if info == nil || info.InstalledFiles == nil {
		return true // no tracking data = assume modified
	}

	storedChecksum, ok := info.InstalledFiles[relPath]
	if !ok {
		return true // file not tracked
	}

	currentChecksum, err := FileChecksum(absPath)
	if err != nil {
		return true // can't read = assume modified
	}

	return currentChecksum != storedChecksum
}

// WorkspaceVersion returns the version string from the workspace, or "unknown"
// if no version.json exists.
func WorkspaceVersion(heroDir string) string {
	info, err := Read(heroDir)
	if err != nil || info == nil || info.HeroVersion == "" {
		return "unknown"
	}
	return info.HeroVersion
}

// cleanVersion strips a leading "v" prefix if present.
func cleanVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// atoi is a thin wrapper around strconv.Atoi.
func atoi(s string) (int, error) {
	return strconv.Atoi(s)
}
