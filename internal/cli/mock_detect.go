package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/snapshot"
	"github.com/spf13/cobra"
)

// lookSwiftc is a test seam — overridden in unit tests to simulate the
// presence/absence of the swiftc compiler without touching the real
// shell environment. Production callers use exec.LookPath directly.
var lookSwiftc = func() (string, error) { return exec.LookPath("swiftc") }

// detectOutput is the wire-shape of `hero spec mock detect`. It is the
// single source of truth the agent reads to pick a renderer; the
// fields are designed so the agent can quote them verbatim in the
// pre-generation announce step. Field names match the spec body
// (lines 156-181 of mockup-renderer-selection-swiftui-bias/spec.md).
type detectOutput struct {
	Renderer       string   `json:"renderer"`
	Reason         string   `json:"reason"`
	Signals        []string `json:"signals"`
	ToolchainOK    bool     `json:"toolchain_ok"`
	ToolchainPath  string   `json:"toolchain_path,omitempty"`
	ConfigOverride string   `json:"config_override,omitempty"`
	ExplicitFlag   string   `json:"explicit_flag,omitempty"`
	Conflict       string   `json:"conflict,omitempty"`
}

var mockDetectRenderer string

var mockDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Emit the recommended renderer for /mock as one line of JSON.",
	Long: `Decides which renderer /mock should use and prints the result as a
single line of JSON. Walks the repo for Swift signals
(Package.swift, *.xcodeproj, *.xcworkspace, .swift files), reads any
mockups.renderer override from hero.json, gates on swiftc
availability, and applies precedence:

  explicit --renderer flag  >  hero.json mockups.renderer  >
  auto-detect (Swift signals + swiftc)  >  HTML fallback

The agent invokes this as the first step of /mock and uses the
"renderer" field verbatim — no re-derivation, no LLM judgment. If
the "conflict" field is non-null (e.g. user passed --renderer=html
on a Swift project) the agent halts and surfaces the message to the
user before generating anything.

Non-zero exit is reserved for internal errors only — a result of
"no Swift signals detected, use HTML" exits 0 like any other
successful detection.`,
	RunE: runMockDetect,
}

func init() {
	mockDetectCmd.Flags().StringVar(&mockDetectRenderer, "renderer", "", "passthrough of the user's explicit --renderer=<html|swiftui> flag")
	mockCmd.AddCommand(mockDetectCmd)
}

func runMockDetect(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	out := computeMockDetect(projectRoot, cfg, mockDetectRenderer)

	b, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshaling detect output: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return nil
}

// computeMockDetect runs the renderer-selection algorithm. Pure function
// over projectRoot, config, and the user's explicit --renderer flag.
// Extracted so unit tests can drive it directly without invoking Cobra.
func computeMockDetect(projectRoot string, cfg config.Config, explicit string) detectOutput {
	signals, swiftDetected := scanSwiftSignals(projectRoot)

	// Toolchain probe is always emitted as data — never an error.
	swiftcPath, swiftcErr := lookSwiftc()
	toolchainOK := swiftcErr == nil && swiftcPath != ""
	if !toolchainOK {
		swiftcPath = ""
	}

	out := detectOutput{
		Signals:       signals,
		ToolchainOK:   toolchainOK,
		ToolchainPath: swiftcPath,
	}

	// Config override is recorded regardless of whether it wins (so the
	// agent can announce "hero.json says X" when it does).
	configRenderer := ""
	if cfg.Mockups != nil {
		configRenderer = strings.ToLower(strings.TrimSpace(cfg.Mockups.Renderer))
	}
	if configRenderer == "html" || configRenderer == "swiftui" {
		out.ConfigOverride = configRenderer
	}

	explicit = strings.ToLower(strings.TrimSpace(explicit))
	if explicit == "html" || explicit == "swiftui" {
		out.ExplicitFlag = explicit
	}

	// Precedence: explicit flag → config override → auto-detect.
	switch {
	case out.ExplicitFlag != "":
		out.Renderer = out.ExplicitFlag
		out.Reason = "explicit --renderer=" + out.ExplicitFlag
		// Conflict detection on the explicit path.
		if out.ExplicitFlag == "html" && swiftDetected {
			out.Conflict = "explicit flag --renderer=html overrides detected SwiftUI stack — confirm before generating"
		}
		if out.ExplicitFlag == "swiftui" && !toolchainOK {
			out.Conflict = "explicit flag --renderer=swiftui but swiftc is not available on PATH — confirm before generating"
		}
	case out.ConfigOverride != "":
		out.Renderer = out.ConfigOverride
		out.Reason = "hero.json mockups.renderer = " + out.ConfigOverride
		// If the config picked swiftui but swiftc is missing, fall back
		// to HTML and explain — this mirrors the auto-detect fallback
		// (AC #5 in the spec).
		if out.Renderer == "swiftui" && !toolchainOK {
			out.Renderer = "html"
			out.Reason = "hero.json mockups.renderer = swiftui but swiftc not found — falling back to HTML"
		}
	case swiftDetected && toolchainOK:
		out.Renderer = "swiftui"
		out.Reason = swiftReasonFromSignals(signals)
	case swiftDetected && !toolchainOK:
		out.Renderer = "html"
		out.Reason = "Swift signals detected but swiftc not found — falling back to HTML"
	default:
		out.Renderer = "html"
		out.Reason = "no Swift signals detected"
	}

	return out
}

// scanSwiftSignals walks the repo via snapshot.ScanRepo (which covers
// monorepo containers at depth 1) and adds explicit root-level checks
// for Swift package/project markers plus a shallow .swift file count.
//
// Two-tier detection:
//   - Strong signals: Package.swift, *.xcodeproj, *.xcworkspace anywhere
//     ScanRepo can see, OR a .swift file. Any strong signal flips
//     swiftDetected=true.
//   - Weak-signal note: bare .swift file count is included in the
//     signals slice so the announce step exposes the volume to the user
//     ("12 .swift files at root"). The next person can revisit if a
//     small apps/ios/ tree under a primarily-Go monorepo causes false
//     positives — see the Risks section of the spec.
//
// Returns (signals, swiftDetected).
func scanSwiftSignals(root string) ([]string, bool) {
	var signals []string
	swiftFound := false

	// Root-level structural markers carry the most weight.
	entries, err := os.ReadDir(root)
	if err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			if !e.IsDir() && name == "Package.swift" {
				signals = append(signals, "Package.swift")
				swiftFound = true
				continue
			}
			if e.IsDir() && strings.HasSuffix(name, ".xcodeproj") {
				signals = append(signals, name)
				swiftFound = true
				continue
			}
			if e.IsDir() && strings.HasSuffix(name, ".xcworkspace") {
				signals = append(signals, name)
				swiftFound = true
				continue
			}
		}
	}

	// Shallow .swift file count — only at repo root to keep latency
	// bounded. Sufficient to flip the weak signal; ScanRepo covers
	// monorepo containers separately.
	swiftCount := 0
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".swift") && e.Name() != "Package.swift" {
				swiftCount++
			}
		}
	}
	if swiftCount > 0 {
		signals = append(signals, fmt.Sprintf("%d .swift files at root", swiftCount))
		swiftFound = true
	}

	// Monorepo containers via ScanRepo. We re-walk one level deep into
	// the same allowlisted containers (apps/, packages/, …) so an
	// iOS-app-in-apps/ios/ layout fires Swift detection.
	rs, scanErr := snapshot.ScanRepo(root)
	if scanErr == nil {
		seenContainers := map[string]struct{}{}
		for _, dir := range rs.Dirs {
			if !strings.Contains(dir, "/") {
				continue
			}
			full := filepath.Join(root, dir)
			containerSignals := swiftMarkersInDir(full)
			if len(containerSignals) == 0 {
				// Cheap .swift sniff at depth 1 — bounded walk, only
				// the immediate dir (no recursion).
				if hasSwiftFile(full) {
					containerSignals = append(containerSignals, dir+"/ (contains .swift files)")
				}
			} else {
				// Prefix the strong-signal name with its container path
				// so the announce step is unambiguous about location.
				for i, sig := range containerSignals {
					containerSignals[i] = filepath.Join(dir, sig)
				}
			}
			if len(containerSignals) > 0 {
				if _, dup := seenContainers[dir]; !dup {
					signals = append(signals, containerSignals...)
					seenContainers[dir] = struct{}{}
				}
				swiftFound = true
			}
		}
	}

	sort.Strings(signals)
	return signals, swiftFound
}

// swiftMarkersInDir reports Package.swift / *.xcodeproj / *.xcworkspace
// found directly inside dir. Names returned are bare (no path prefix).
// Returns empty slice on read errors.
func swiftMarkersInDir(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && name == "Package.swift" {
			out = append(out, "Package.swift")
		}
		if e.IsDir() && strings.HasSuffix(name, ".xcodeproj") {
			out = append(out, name)
		}
		if e.IsDir() && strings.HasSuffix(name, ".xcworkspace") {
			out = append(out, name)
		}
	}
	return out
}

// hasSwiftFile reports whether dir contains at least one .swift file
// at its immediate level. No recursion — bounded so detect stays fast
// on large monorepos.
func hasSwiftFile(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".swift") {
			return true
		}
	}
	return false
}

// swiftReasonFromSignals composes a one-sentence rationale the agent
// can paste into its announce step. Preference order:
//   1. Package.swift (most authoritative)
//   2. *.xcodeproj / *.xcworkspace
//   3. raw .swift file count
//   4. monorepo container hits
func swiftReasonFromSignals(signals []string) string {
	for _, s := range signals {
		if s == "Package.swift" || strings.HasSuffix(s, "/Package.swift") {
			return "detected Package.swift"
		}
	}
	for _, s := range signals {
		if strings.HasSuffix(s, ".xcodeproj") {
			return "detected " + s
		}
		if strings.HasSuffix(s, ".xcworkspace") {
			return "detected " + s
		}
	}
	for _, s := range signals {
		if strings.HasSuffix(s, "files at root") {
			return "detected " + s
		}
	}
	if len(signals) > 0 {
		return "detected " + signals[0]
	}
	return "Swift stack detected"
}
