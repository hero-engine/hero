package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/version"
)

func homePatch(t *testing.T) string {
	t.Helper()
	path, err := DeepSeekHomePatchPath()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func readHomePatch(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(homePatch(t))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// AC-1/AC-2: first install creates the file with one entry; a repeat is a no-op.
func TestDeepSeekHomeEntryCreateAndNoop(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	h.Run(TargetDeepSeek, nil)
	first := readHomePatch(t)
	server, _ := DeepSeekServerName(h.TargetDir)
	root, _ := canonicalProjectRoot(h.TargetDir)
	for _, want := range []string{deepseekHomeCreatedLine, "id: " + server + "-mcp", "name: '@deepseek-ai/dsh-mcp-client'", "serverName: " + server, "command: " + os.Getenv(deepseekCommandEnv), "cwd: " + root, "- --project-root"} {
		if !strings.Contains(first, want) {
			t.Errorf("home patch missing %q:\n%s", want, first)
		}
	}
	if len(server) > 32 || !strings.HasPrefix(server, "hero-") || strings.Trim(server, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" {
		t.Fatalf("invalid serverName %q", server)
	}
	if _, _, changed, err := UpsertDeepSeekHomeEntry(h.TargetDir, false); err != nil || changed {
		t.Fatalf("repeat upsert changed=%v err=%v", changed, err)
	}
	if again := readHomePatch(t); again != first {
		t.Fatalf("repeat install rewrote the home patch")
	}
}

// AC-2/AC-9: foreign entries and comments survive byte-for-byte, and
// uninstall restores the exact original file.
func TestDeepSeekHomeEntryPreservesForeignContent(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	original := "# my DeepSeek overrides\n- id: agent-default-model\n  config:\n    model: mine # keep\n"
	path := homePatch(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	h.Run(TargetDeepSeek, nil)
	got := readHomePatch(t)
	if !strings.HasPrefix(got, original) || strings.Contains(got, deepseekHomeCreatedLine) {
		t.Fatalf("foreign content not preserved:\n%s", got)
	}
	if removed, err := RemoveDeepSeekHomeEntry(h.TargetDir, true); err != nil || !removed {
		t.Fatalf("dry-run removal: %v %v", removed, err)
	}
	if readHomePatch(t) != got {
		t.Fatal("dry-run removal mutated the file")
	}
	if removed, err := RemoveDeepSeekHomeEntry(h.TargetDir, false); err != nil || !removed {
		t.Fatalf("removal: %v %v", removed, err)
	}
	if after := readHomePatch(t); after != original {
		t.Fatalf("uninstall did not restore the original:\n%q\nwant\n%q", after, original)
	}
}

// AC-3/AC-9: two projects get independent entries; removing one keeps the
// other, and the Hero-created file goes away only when empty.
func TestDeepSeekHomeEntryTwoProjects(t *testing.T) {
	h := newInstallHarness(t)
	other := t.TempDir()
	mkHeroDir(t, h.TargetDir)
	mkHeroDir(t, other)
	h.Run(TargetDeepSeek, nil)
	h.Run(TargetDeepSeek, func(o *Options) { o.TargetDir = other })
	a, _ := DeepSeekServerName(h.TargetDir)
	b, _ := DeepSeekServerName(other)
	if a == b {
		t.Fatalf("server names collide: %s", a)
	}
	both := readHomePatch(t)
	if !strings.Contains(both, "serverName: "+a) || !strings.Contains(both, "serverName: "+b) {
		t.Fatalf("missing an entry:\n%s", both)
	}
	if _, err := RemoveDeepSeekHomeEntry(other, false); err != nil {
		t.Fatal(err)
	}
	if reg := InspectDeepSeekRegistration(h.TargetDir); reg.Problem != "" {
		t.Fatalf("removing one project broke the other: %+v", reg)
	}
	if _, err := RemoveDeepSeekHomeEntry(h.TargetDir, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(homePatch(t)); !os.IsNotExist(err) {
		t.Fatalf("empty Hero-created home patch survived: %v", err)
	}
}

// AC-4: unsafe home patches fail before any project file is written, in
// dry-run too.
func TestDeepSeekHomePatchRefusedBeforeMutation(t *testing.T) {
	cases := map[string]func(t *testing.T, path string){
		"mapping": func(t *testing.T, path string) {
			os.WriteFile(path, []byte("not: a list\n"), 0o644)
		},
		"invalid yaml": func(t *testing.T, path string) {
			os.WriteFile(path, []byte("- [unclosed\n"), 0o644)
		},
		"symlink": func(t *testing.T, path string) {
			target := filepath.Join(t.TempDir(), "real.yml")
			os.WriteFile(target, []byte("[]\n"), 0o644)
			os.Symlink(target, path)
		},
	}
	for name, setup := range cases {
		for _, dry := range []bool{false, true} {
			t.Run(name, func(t *testing.T) {
				h := newInstallHarness(t)
				mkHeroDir(t, h.TargetDir)
				path := homePatch(t)
				os.MkdirAll(filepath.Dir(path), 0o755)
				setup(t, path)
				opts := Options{SourceDir: h.SourceDir, Target: TargetDeepSeek, Mode: ModeProject, TargetDir: h.TargetDir, DryRun: dry, Quiet: true}
				if _, err := Run(opts); err == nil {
					t.Fatal("unsafe home patch accepted")
				}
				h.mustNotExist("AGENTS.md")
				h.mustNotExist(".dsh")
			})
		}
	}
}

// AC-5: the command must be absolute and never a transient build binary.
func TestDeepSeekHeroCommandResolution(t *testing.T) {
	t.Setenv(deepseekCommandEnv, "hero")
	if _, err := resolveDeepSeekHeroCommand(); err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("relative pin accepted: %v", err)
	}
	t.Setenv(deepseekCommandEnv, "")
	bin := t.TempDir()
	fake := filepath.Join(bin, "hero")
	os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755)
	t.Setenv("PATH", bin)
	if got, err := resolveDeepSeekHeroCommand(); err != nil || got != fake {
		t.Fatalf("PATH hero = %q, %v; want %q", got, err, fake)
	}
	for path, want := range map[string]bool{
		"/private/var/folders/x/go-build123/b001/install.test": true,
		filepath.Join(os.TempDir(), "hero"):                    true,
		"/usr/local/bin/hero":                                  false,
	} {
		if transientExecutable(path) != want {
			t.Errorf("transientExecutable(%q) != %v", path, want)
		}
	}
}

// AC-8: inspection names a stale command and an entry for another root.
func TestDeepSeekRegistrationProblems(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	if reg := InspectDeepSeekRegistration(h.TargetDir); !strings.Contains(reg.Problem, "no Hero MCP entry") {
		t.Fatalf("missing entry not reported: %+v", reg)
	}
	h.Run(TargetDeepSeek, nil)
	path := homePatch(t)
	good := readHomePatch(t)
	stale := strings.Replace(good, "command: "+os.Getenv(deepseekCommandEnv), "command: /nonexistent/hero", 1)
	os.WriteFile(path, []byte(stale), 0o644)
	if reg := InspectDeepSeekRegistration(h.TargetDir); !strings.Contains(reg.Problem, "/nonexistent/hero") {
		t.Fatalf("stale command not reported: %+v", reg)
	}
	root, _ := canonicalProjectRoot(h.TargetDir)
	wrong := strings.Replace(good, "cwd: "+root, "cwd: /elsewhere", 1)
	os.WriteFile(path, []byte(wrong), 0o644)
	if reg := InspectDeepSeekRegistration(h.TargetDir); !strings.Contains(reg.Problem, "not this project") {
		t.Fatalf("wrong root not reported: %+v", reg)
	}
}

// AC-6: an unmodified legacy project overlay is pruned on reinstall; a
// modified one is kept.
func TestDeepSeekLegacyOverlayPruning(t *testing.T) {
	for _, modified := range []bool{false, true} {
		t.Run(map[bool]string{false: "unmodified", true: "modified"}[modified], func(t *testing.T) {
			h := newInstallHarness(t)
			mkHeroDir(t, h.TargetDir)
			h.Run(TargetDeepSeek, nil)
			legacy := filepath.Join(h.TargetDir, ".dsh", deepseekOverlayName)
			os.WriteFile(legacy, []byte("- insert: []\n"), 0o644)
			sum, err := version.FileChecksum(legacy)
			if err != nil {
				t.Fatal(err)
			}
			if err := version.StampInstall(filepath.Join(h.TargetDir, ".hero"), "test", string(TargetDeepSeek), string(ModeProject), map[string]string{".dsh/" + deepseekOverlayName: sum}); err != nil {
				t.Fatal(err)
			}
			if modified {
				os.WriteFile(legacy, []byte("- insert: [] # mine\n"), 0o644)
			}
			h.Run(TargetDeepSeek, func(o *Options) { o.Force = false })
			_, err = os.Stat(legacy)
			if modified && err != nil {
				t.Fatalf("modified legacy overlay removed: %v", err)
			}
			if !modified && !os.IsNotExist(err) {
				t.Fatalf("unmodified legacy overlay kept: %v", err)
			}
		})
	}
}
