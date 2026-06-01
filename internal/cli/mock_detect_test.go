package cli

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// withSwiftc temporarily overrides the lookSwiftc seam for the duration
// of the test. Cleans up after itself even on test failure.
func withSwiftc(t *testing.T, present bool) {
	t.Helper()
	orig := lookSwiftc
	if present {
		lookSwiftc = func() (string, error) { return "/fake/usr/bin/swiftc", nil }
	} else {
		lookSwiftc = func() (string, error) { return "", errors.New("swiftc: not found") }
	}
	t.Cleanup(func() { lookSwiftc = orig })
}

// runDetect invokes `hero spec mock detect` and parses the JSON line.
// Returns the parsed output plus the raw stdout for failure messages.
func runDetect(t *testing.T, args ...string) (detectOutput, string) {
	t.Helper()
	all := append([]string{"spec", "mock", "detect"}, args...)
	stdout, err := runCmd(all...)
	if err != nil {
		t.Fatalf("runCmd(%v) error: %v\noutput: %s", all, err, stdout)
	}
	stdout = strings.TrimSpace(stdout)
	var out detectOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("parse JSON: %v\nraw: %q", err, stdout)
	}
	return out, stdout
}

// writeFile is a small helper for fixture setup.
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func TestMockDetect_PureGoProject(t *testing.T) {
	_ = newTestEnv(t)
	withSwiftc(t, true) // toolchain availability doesn't matter for HTML

	out, raw := runDetect(t)
	if out.Renderer != "html" {
		t.Errorf("renderer = %q, want %q (raw: %s)", out.Renderer, "html", raw)
	}
	if out.Reason == "" {
		t.Errorf("reason should be set, got empty (raw: %s)", raw)
	}
	if out.Conflict != "" {
		t.Errorf("conflict should be empty for pure-Go, got %q", out.Conflict)
	}
	if out.ExplicitFlag != "" {
		t.Errorf("explicit_flag should be empty, got %q", out.ExplicitFlag)
	}
	if out.ConfigOverride != "" {
		t.Errorf("config_override should be empty, got %q", out.ConfigOverride)
	}
}

func TestMockDetect_PackageSwiftAtRoot(t *testing.T) {
	env := newTestEnv(t)
	withSwiftc(t, true)

	writeFile(t, filepath.Join(env.dir, "Package.swift"), "// swift-tools-version:5.5\n")

	out, raw := runDetect(t)
	if out.Renderer != "swiftui" {
		t.Errorf("renderer = %q, want %q (raw: %s)", out.Renderer, "swiftui", raw)
	}
	if !out.ToolchainOK {
		t.Errorf("toolchain_ok should be true (raw: %s)", raw)
	}
	if out.ToolchainPath == "" {
		t.Errorf("toolchain_path should be set when toolchain_ok, raw: %s", raw)
	}
	foundPkg := false
	for _, s := range out.Signals {
		if s == "Package.swift" {
			foundPkg = true
			break
		}
	}
	if !foundPkg {
		t.Errorf("signals should include Package.swift, got %v", out.Signals)
	}
}

func TestMockDetect_XcodeprojAtRoot(t *testing.T) {
	env := newTestEnv(t)
	withSwiftc(t, true)

	if err := os.MkdirAll(filepath.Join(env.dir, "MyApp.xcodeproj"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, raw := runDetect(t)
	if out.Renderer != "swiftui" {
		t.Errorf("renderer = %q, want %q (raw: %s)", out.Renderer, "swiftui", raw)
	}
	hasXc := false
	for _, s := range out.Signals {
		if s == "MyApp.xcodeproj" {
			hasXc = true
		}
	}
	if !hasXc {
		t.Errorf("signals should mention MyApp.xcodeproj, got %v", out.Signals)
	}
}

func TestMockDetect_OnlySwiftFiles_WeakSignal(t *testing.T) {
	env := newTestEnv(t)
	withSwiftc(t, true)

	writeFile(t, filepath.Join(env.dir, "Main.swift"), "// swift\n")
	writeFile(t, filepath.Join(env.dir, "Helpers.swift"), "// swift\n")

	out, raw := runDetect(t)
	if out.Renderer != "swiftui" {
		t.Errorf("renderer = %q, want %q (raw: %s)", out.Renderer, "swiftui", raw)
	}
	// Signals should include the count
	found := false
	for _, s := range out.Signals {
		if strings.Contains(s, ".swift files at root") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("signals should include .swift file count, got %v", out.Signals)
	}
}

func TestMockDetect_ConfigOverrideHtmlOnSwiftProject(t *testing.T) {
	env := newTestEnv(t)
	withSwiftc(t, true)

	writeFile(t, filepath.Join(env.dir, "Package.swift"), "// swift\n")
	// Overwrite hero.json with mockups.renderer = html
	cfgJSON := `{"folder":".hero","mockups":{"renderer":"html"}}`
	writeFile(t, filepath.Join(env.dir, ".hero", "hero.json"), cfgJSON)

	out, raw := runDetect(t)
	if out.Renderer != "html" {
		t.Errorf("renderer = %q, want %q (config override) (raw: %s)", out.Renderer, "html", raw)
	}
	if out.ConfigOverride != "html" {
		t.Errorf("config_override = %q, want %q", out.ConfigOverride, "html")
	}
	if !strings.Contains(out.Reason, "hero.json") {
		t.Errorf("reason should mention hero.json, got %q", out.Reason)
	}
}

func TestMockDetect_ConfigOverrideSwiftuiWithoutToolchain_FallsBackToHTML(t *testing.T) {
	env := newTestEnv(t)
	withSwiftc(t, false) // no swiftc

	cfgJSON := `{"folder":".hero","mockups":{"renderer":"swiftui"}}`
	writeFile(t, filepath.Join(env.dir, ".hero", "hero.json"), cfgJSON)

	out, raw := runDetect(t)
	if out.Renderer != "html" {
		t.Errorf("renderer = %q, want fallback %q (raw: %s)", out.Renderer, "html", raw)
	}
	if out.ConfigOverride != "swiftui" {
		t.Errorf("config_override = %q, want %q (we record the requested override even on fallback)", out.ConfigOverride, "swiftui")
	}
	if !strings.Contains(out.Reason, "swiftc not found") {
		t.Errorf("reason should mention swiftc, got %q", out.Reason)
	}
}

func TestMockDetect_ExplicitHtmlOnSwift_PopulatesConflict(t *testing.T) {
	env := newTestEnv(t)
	withSwiftc(t, true)

	writeFile(t, filepath.Join(env.dir, "Package.swift"), "// swift\n")

	out, raw := runDetect(t, "--renderer=html")
	if out.Renderer != "html" {
		t.Errorf("renderer = %q, want %q (explicit) (raw: %s)", out.Renderer, "html", raw)
	}
	if out.ExplicitFlag != "html" {
		t.Errorf("explicit_flag = %q, want %q", out.ExplicitFlag, "html")
	}
	if out.Conflict == "" {
		t.Errorf("conflict should be populated when --renderer=html on Swift project (raw: %s)", raw)
	}
	if !strings.Contains(out.Conflict, "html") || !strings.Contains(out.Conflict, "SwiftUI") {
		t.Errorf("conflict text should mention both renderers, got %q", out.Conflict)
	}
}

func TestMockDetect_ExplicitSwiftui_NoConflict(t *testing.T) {
	env := newTestEnv(t)
	withSwiftc(t, true)

	writeFile(t, filepath.Join(env.dir, "Package.swift"), "// swift\n")

	out, _ := runDetect(t, "--renderer=swiftui")
	if out.Renderer != "swiftui" {
		t.Errorf("renderer = %q, want %q", out.Renderer, "swiftui")
	}
	if out.ExplicitFlag != "swiftui" {
		t.Errorf("explicit_flag = %q, want %q", out.ExplicitFlag, "swiftui")
	}
	if out.Conflict != "" {
		t.Errorf("conflict should be empty, got %q", out.Conflict)
	}
}

func TestMockDetect_ExplicitSwiftuiWithoutToolchain_PopulatesConflict(t *testing.T) {
	_ = newTestEnv(t)
	withSwiftc(t, false)

	out, raw := runDetect(t, "--renderer=swiftui")
	if out.Renderer != "swiftui" {
		t.Errorf("renderer = %q, want %q (we honor explicit flag) (raw: %s)", out.Renderer, "swiftui", raw)
	}
	if out.Conflict == "" {
		t.Errorf("conflict should be populated when --renderer=swiftui and no swiftc, raw: %s", raw)
	}
	if !strings.Contains(out.Conflict, "swiftc") {
		t.Errorf("conflict should mention swiftc, got %q", out.Conflict)
	}
	if out.ToolchainOK {
		t.Errorf("toolchain_ok should be false")
	}
}

func TestMockDetect_SwiftSignalsWithoutToolchain_FallsBack(t *testing.T) {
	env := newTestEnv(t)
	withSwiftc(t, false)

	writeFile(t, filepath.Join(env.dir, "Package.swift"), "// swift\n")

	out, raw := runDetect(t)
	if out.Renderer != "html" {
		t.Errorf("renderer = %q, want fallback %q (raw: %s)", out.Renderer, "html", raw)
	}
	if !strings.Contains(out.Reason, "swiftc not found") {
		t.Errorf("reason should explain the fallback, got %q", out.Reason)
	}
	if out.ToolchainOK {
		t.Errorf("toolchain_ok should be false")
	}
}

func TestMockDetect_MonorepoSwiftInAppsIOS(t *testing.T) {
	env := newTestEnv(t)
	withSwiftc(t, true)

	// Simulate apps/ios/Package.swift — a depth-1 monorepo container
	// that ScanRepo walks.
	writeFile(t, filepath.Join(env.dir, "apps", "ios", "Package.swift"), "// swift\n")

	out, raw := runDetect(t)
	if out.Renderer != "swiftui" {
		t.Errorf("renderer = %q, want %q for monorepo Swift (raw: %s)", out.Renderer, "swiftui", raw)
	}
	// Verify signals carry the location so the announce step is unambiguous.
	hasLocation := false
	for _, s := range out.Signals {
		if strings.Contains(s, "ios") {
			hasLocation = true
			break
		}
	}
	if !hasLocation {
		t.Errorf("signals should mention monorepo location, got %v", out.Signals)
	}
}

func TestMockDetect_JSONIsSingleLine(t *testing.T) {
	_ = newTestEnv(t)
	withSwiftc(t, true)

	stdout, err := runCmd("spec", "mock", "detect")
	if err != nil {
		t.Fatalf("runCmd: %v", err)
	}
	// Strip the trailing newline from Fprintln, then ensure no
	// interior newlines.
	body := strings.TrimRight(stdout, "\n")
	if strings.Contains(body, "\n") {
		t.Errorf("detect output must be one JSON line, got:\n%s", stdout)
	}
}

// realSwiftcAvailable is a small probe for use in the "production
// default" sanity test below — we don't require swiftc on CI, so the
// assertion is gated on its presence.
func realSwiftcAvailable() bool {
	_, err := exec.LookPath("swiftc")
	return err == nil
}

func TestMockDetect_NoFlagsNoOverride_RealEnv(t *testing.T) {
	// This test runs against the real exec.LookPath to confirm the
	// production seam is wired correctly. The assertion only fires
	// when swiftc presence flips a behavior, so it's CI-safe.
	_ = newTestEnv(t)
	// Reset to real seam for this case.
	orig := lookSwiftc
	lookSwiftc = func() (string, error) { return exec.LookPath("swiftc") }
	t.Cleanup(func() { lookSwiftc = orig })

	out, raw := runDetect(t)
	if out.Renderer != "html" {
		t.Errorf("renderer = %q, want %q for empty test env (raw: %s)", out.Renderer, "html", raw)
	}
	if realSwiftcAvailable() && !out.ToolchainOK {
		t.Errorf("toolchain_ok should reflect real swiftc availability")
	}
	if !realSwiftcAvailable() && out.ToolchainOK {
		t.Errorf("toolchain_ok should be false when real swiftc absent")
	}
}
