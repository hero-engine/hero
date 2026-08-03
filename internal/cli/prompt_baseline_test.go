//go:build unix

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/ptytest"
)

// This file is the regression baseline for every interactive prompt site in
// the CLI. It is the primary safety property of the `interactive-cli-input`
// initiative: it was captured BEFORE any prompt site was migrated onto the
// shared `internal/cli/prompt` package, so every later claim that a migration
// "behaves identically" is verified as a diff against recorded bytes rather
// than against reviewer judgment.
//
// The behaviour these fixtures record stops existing once the refactor lands,
// so it cannot be reconstructed afterwards. Regenerate them ONLY when a diff
// has been consciously reviewed and accepted:
//
//	go test ./internal/cli/ -run TestPromptSiteBaseline -update-baseline
//
// Each case is driven through a real subprocess rather than in-process cobra,
// for three reasons the in-process harness cannot satisfy:
//
//  1. Exit codes are part of the recorded contract.
//  2. Most pre-migration sites write to `os.Stdout` via `fmt.Print`, not to
//     `cmd.OutOrStdout()`, so an in-process buffer would not see them.
//  3. The TTY predicates under test read the real `os.Stdin` file descriptor.
//     Only a subprocess lets us control what fd 0 actually is.
//
// The child is started with Setsid so it has no controlling terminal. Without
// that, `connect.go`'s `promptSecret` would successfully open /dev/tty on a
// developer machine and block on a real password read, making the fixture
// depend on whether the suite was run from a terminal.

var updateBaseline = flag.Bool("update-baseline", false,
	"regenerate the prompt-site baseline golden fixtures in testdata/prompt_baseline")

// Stdin conditions required by the spec.
//
// Note on condClosed: the Go runtime guarantees fds 0/1/2 are open at process
// start, backfilling any closed one with /dev/null. So "stdin closed" is
// observationally identical to "stdin is /dev/null" for any Go binary, and
// both are character devices. That distinction is load-bearing: the
// pre-migration `os.Stdin.Stat()`/`ModeCharDevice` predicates report *true*
// (i.e. "this is a terminal") for /dev/null, while `term.IsTerminal` reports
// false. The two predicates therefore disagree precisely under this condition,
// which is exactly what these fixtures pin down.
const (
	condFlags  = "flags"  // every required value supplied by flag or argument
	condClosed = "closed" // required input missing, stdin closed
	condPipe   = "pipe"   // required input missing, stdin a non-TTY pipe

	// condTTY is not a baseline condition — it records no golden fixture.
	// It drives a site with a real pseudo-terminal on fd 0 so the
	// TTY-present branch can be exercised through the shipped binary.
	//
	// Note that the child still has no CONTROLLING terminal (it is started
	// with Setsid, and its stdin is dup'd rather than opened), so /dev/tty
	// remains unavailable. Sites that read secrets therefore still refuse
	// under this condition — which is the correct assertion for them anyway.
	condTTY = "tty"
)

// baselineCase describes one prompt site and how to drive it.
type baselineCase struct {
	// name is the golden-file stem.
	name string
	// site names the prompt site from the initiative's inventory table. Every
	// one of the 14 rows must be covered; TestPromptSiteBaselineCoversAllSites
	// enforces that.
	site string
	// setup prepares the workspace for the missing-input conditions.
	setup func(t *testing.T, root string)
	// args is the invocation with required interactive input MISSING.
	args []string
	// pipeStdin is fed on the condPipe run. Empty means an empty pipe. A few
	// sites are only reachable after an earlier prompt has been answered
	// (connect's token prompt sits behind its repository prompt), so those
	// cases prime the pipe with the earlier answer.
	pipeStdin string
	// flagArgs is the invocation with every required value supplied. Empty
	// means the site has no fully-flagged form and condFlags is skipped.
	flagArgs []string
	// flagSetup overrides setup for the condFlags run when the fully-flagged
	// form needs a different fixture.
	flagSetup func(t *testing.T, root string)
	// flagStdin is fed on the condFlags run (e.g. `--token-stdin`).
	flagStdin string
	// reach is a substring that proves the invocation actually got as far as
	// the prompt site, rather than erroring out earlier for an unrelated
	// reason (a missing fixture, a renamed subcommand). It must appear in at
	// least one of the two missing-input fixtures.
	//
	// Without this, a fixture that never reaches its site still records
	// "real" bytes and still passes — a silently empty safety net. That is
	// the single most likely way this baseline rots.
	reach string
	// ttyStdin is fed to the pseudo-terminal in the condTTY run. Empty means
	// "press enter at every prompt".
	ttyStdin string
	// ttySkip, when set, skips the TTY-present assertion with this reason.
	// It is only for sites that read secrets: those go to /dev/tty, and the
	// test child deliberately has no controlling terminal, so they refuse
	// under every condition here. Their refusal is asserted directly in
	// prompt_sanctioned_breaks_test.go instead.
	ttySkip string
}

// baselineCases covers all 14 prompt sites from the initiative inventory.
//
// Two of the fourteen (`new.go:433`, `export.go:121`) are already correct and
// are audit-only for this child — they are baselined anyway, because the
// baseline's job is to prove they did NOT change.
func baselineCases() []baselineCase {
	return []baselineCase{
		{
			name:  "connect_github_repo_prompt",
			reach: "Repository (owner/repo): ",
			site:  "connect.go:766 prompt()",
			setup: setupWorkspace,
			args:  []string{"connect", "github"},
			flagArgs: []string{"connect", "github",
				"--integration-id", "gh-baseline", "--project", "owner/repo",
				"--local-only", "--no-verify", "--token-stdin"},
			flagStdin: "baseline-token\n",
		},
		{
			name:      "connect_github_secret_prompt",
			reach:     "Personal access token (needs 'repo' scope): ",
			ttySkip:   "promptSecret reads /dev/tty; the test child has no controlling terminal",
			site:      "connect.go:778 promptSecret()",
			setup:     setupWorkspace,
			args:      []string{"connect", "github"},
			pipeStdin: "owner/repo\n",
		},
		{
			name:     "install_project_target_prompt",
			reach:    "Install target",
			site:     "install.go:442 promptTarget()",
			setup:    setupBareProjectDir,
			args:     []string{"install", "project", "proj", "--dry-run"},
			flagArgs: []string{"install", "project", "proj", "--dry-run", "--target", "claude"},
		},
		{
			name:     "install_satellite_subproject_add",
			reach:    "Add it as a subproject so teammates pick it up automatically? [y/N]",
			site:     "install.go:489 subproject-add confirm",
			setup:    setupWorkspaceWithUndeclaredSubdir,
			args:     []string{"install", "project", "sub"},
			flagArgs: []string{"install", "project", "sub", "--json"},
		},
		{
			name:     "install_candidate_walk",
			reach:    "subproject candidate(s). Walk through them now? [y/N]",
			site:     "install.go:567 candidate-walk confirm",
			setup:    setupRootInstallCandidates,
			args:     []string{"install", "project", ".", "--target", "claude"},
			flagArgs: []string{"install", "project", ".", "--target", "claude", "--json"},
		},
		{
			name:     "install_satellites_migrate_nested",
			reach:    "Apply these migrations? [y/N]",
			site:     "install_satellites.go:111 migrate-nested confirm",
			setup:    setupNestedWorkspace,
			args:     []string{"install", "satellites", "--migrate-nested", "--apply"},
			flagArgs: []string{"install", "satellites", "--migrate-nested", "--apply", "--yes"},
		},
		{
			name:     "install_satellites_reconcile_declared",
			reach:    "materialize satellite? [Y/n]",
			site:     "install_satellites.go:216 reconcileDeclared",
			setup:    setupDeclaredSubproject,
			args:     []string{"install", "satellites"},
			flagArgs: []string{"install", "satellites", "--no"},
		},
		{
			name:     "install_satellites_walk_candidates",
			reach:    "propose? [y/N/a/s/q/x/X/?]",
			site:     "install_satellites.go:243 walkCandidates",
			setup:    setupSatelliteCandidates,
			args:     []string{"install", "satellites"},
			flagArgs: []string{"install", "satellites", "--no"},
		},
		{
			name:  "skill_save_form",
			reach: "Skill name (slug, e.g. my-workflow): ",
			site:  "skill.go:208 skill save",
			setup: setupWorkspace,
			args:  []string{"skill", "save"},
		},
		{
			name:     "skill_run_param_prompt",
			reach:    "target (the thing to act on): ",
			site:     "skill.go:320 promptParam()",
			setup:    setupSkillWithParam,
			args:     []string{"skill", "run", "baseline-skill"},
			flagArgs: []string{"skill", "run", "baseline-skill", "--param", "target=abc"},
		},
		{
			name:  "new_interactive",
			site:  "new.go:433 collectInteractiveInputs",
			setup: setupWorkspace,
			// `new` is mounted under `hero spec` (spec.go:51).
			args:      []string{"spec", "new", "baseline-slug", "--interactive"},
			flagArgs:  []string{"spec", "new", "baseline-slug", "--title", "Baseline Slug"},
			flagSetup: setupWorkspace,
			reach:     "Title [",
		},
		{
			name:    "users_passwd_password_prompt",
			reach:   "New password: ",
			ttySkip: "prompt.Secret reads /dev/tty; the test child has no controlling terminal",
			site:    "users.go:188 promptPassword()",
			setup:   setupWorkspace,
			// The spec calls this "hero users passwd"; the command is actually
			// mounted under `hero admin` (admin.go:23).
			args:     []string{"admin", "users", "passwd", "alice"},
			flagArgs: nil, // `admin users passwd` has no non-interactive form today.
			// The pipe condition is the security defect: a password is read
			// off an unprotected stream.
			pipeStdin: "hunter2\nhunter2\n",
		},
		{
			name:     "export_knowledge_conflict_prompt",
			reach:    "Conflict: ",
			ttySkip:  "the export gate also requires an output terminal; stdout here is a pipe",
			site:     "export.go:121 promptConflictStrategy()",
			setup:    setupExportConflict,
			args:     []string{"export", "knowledge", "dest", "--conflict", "interactive"},
			flagArgs: []string{"export", "knowledge", "dest", "--conflict", "skip"},
		},
		{
			name:     "handoff_accept_next_status",
			reach:    "Pick the next status for this spec:",
			site:     "handoff.go:326 promptNextStatus()",
			setup:    setupHandedBackSpec,
			args:     []string{"handoff", "accept", "baseline-handed-back"},
			flagArgs: nil, // `handoff accept` has no non-interactive form today.
		},
	}
}

func TestPromptSiteBaseline(t *testing.T) {
	bin := baselineBinary(t)

	for _, tc := range baselineCases() {
		t.Run(tc.name, func(t *testing.T) {
			for _, cond := range []string{condFlags, condClosed, condPipe} {
				if cond == condFlags && tc.flagArgs == nil {
					continue
				}
				t.Run(cond, func(t *testing.T) {
					args := tc.args
					stdin := ""
					setup := tc.setup
					switch cond {
					case condFlags:
						args = tc.flagArgs
						stdin = tc.flagStdin
						if tc.flagSetup != nil {
							setup = tc.flagSetup
						}
					case condPipe:
						stdin = tc.pipeStdin
					}

					// HOME and TMPDIR must live OUTSIDE the workspace root:
					// `install`'s candidate detector walks the root, and a
					// home directory sitting inside it shows up as a
					// subproject candidate and perturbs the fixture.
					base := t.TempDir()
					root := filepath.Join(base, "work")
					if err := os.MkdirAll(root, 0o755); err != nil {
						t.Fatalf("mkdir work root: %v", err)
					}
					if setup != nil {
						setup(t, root)
					}
					got := runBaselineCase(t, bin, base, root, args, cond, stdin)

					golden := filepath.Join("testdata", "prompt_baseline",
						fmt.Sprintf("%s.%s.txt", tc.name, cond))
					if *updateBaseline {
						if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
							t.Fatalf("mkdir golden dir: %v", err)
						}
						if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
							t.Fatalf("write golden: %v", err)
						}
						return
					}
					want, err := os.ReadFile(golden)
					if err != nil {
						t.Fatalf("read golden %s: %v\n\nRegenerate with:\n  go test ./internal/cli/ -run TestPromptSiteBaseline -update-baseline", golden, err)
					}
					if got != string(want) {
						t.Errorf("baseline drift for site %s (%s)\n--- want (%s) ---\n%s\n--- got ---\n%s",
							tc.site, cond, golden, want, got)
					}
				})
			}
		})
	}
}

// TestPromptSiteBaselineCoversAllSites guards the inventory itself. The
// initiative's site table has 14 rows; an earlier prose summary said 13 and was
// wrong. If a future change drops a case, this fails rather than silently
// shrinking the safety net.
func TestPromptSiteBaselineCoversAllSites(t *testing.T) {
	const wantSites = 14
	cases := baselineCases()
	seen := map[string]bool{}
	for _, c := range cases {
		if seen[c.site] {
			t.Errorf("duplicate site %q", c.site)
		}
		seen[c.site] = true
	}
	if len(seen) != wantSites {
		t.Errorf("baseline covers %d prompt sites, want %d (the initiative's site table has %d rows)",
			len(seen), wantSites, wantSites)
	}
}

// TestPromptSitesReturnBeforeLivePipeEOF closes the false assurance in the
// closed-reader fixtures. An empty strings.Reader reaches EOF immediately;
// an open pipe has no bytes and no EOF, so any accidental prompt read blocks.
// Returning before this deadline proves the command declined to read from the
// non-terminal stream at every migrated site.
func TestPromptSitesReturnBeforeLivePipeEOF(t *testing.T) {
	bin := baselineBinary(t)
	for _, tc := range baselineCases() {
		t.Run(tc.name, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, "work")
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatalf("mkdir work root: %v", err)
			}
			if tc.setup != nil {
				tc.setup(t, root)
			}
			if err := runHeroWithOpenPipe(bin, base, root, tc.args); err != nil {
				t.Fatalf("%s read from a live non-TTY pipe: %v", tc.site, err)
			}
		})
	}
}

// TestPromptSitesPromptWhenTTYPresent is the third of AC-13's three cases —
// "required input missing AND stdin is a terminal" — driven through the
// shipped binary with a real pseudo-terminal on fd 0.
//
// It doubles as the proof that each baseline case actually drives the site it
// claims to. That proof used to read the non-TTY fixtures for the prompt text,
// which worked only while the sites still (incorrectly) prompted into
// non-terminals. Now that they correctly stay silent there, absence of the
// prompt is the expected result, so the assertion had to move to where a
// prompt is genuinely expected. It is a stronger check than the one it
// replaces: it exercises the real predicate rather than inspecting a file.
//
// It caught two hollow fixtures in its earlier form — one whose workspace
// fixture routed the command down a different code path entirely, and one
// whose subcommand had been renamed.
func TestPromptSitesPromptWhenTTYPresent(t *testing.T) {
	bin := baselineBinary(t)

	for _, tc := range baselineCases() {
		t.Run(tc.name, func(t *testing.T) {
			if tc.reach == "" {
				t.Fatalf("case %q declares no reach marker", tc.name)
			}
			if tc.ttySkip != "" {
				t.Skip(tc.ttySkip)
			}

			base := t.TempDir()
			root := filepath.Join(base, "work")
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatalf("mkdir work root: %v", err)
			}
			if tc.setup != nil {
				tc.setup(t, root)
			}

			answers := tc.ttyStdin
			if answers == "" {
				// Enough newlines to take the default at each prompt without
				// leaving a later read blocked on an empty terminal.
				answers = strings.Repeat("\n", 12)
			}
			exitCode, stdout, stderr := runHero(t, bin, base, root, tc.args, condTTY, answers)
			combined := stdout + stderr

			if exitCode == -1 {
				t.Fatalf("site %s blocked with a terminal attached; output so far:\n%s", tc.site, combined)
			}
			if !strings.Contains(combined, tc.reach) {
				t.Errorf("site %s did not prompt with a terminal attached.\nwant substring: %q\ngot:\n%s",
					tc.site, tc.reach, combined)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// subprocess runner
// ---------------------------------------------------------------------------

var (
	baselineBinOnce sync.Once
	baselineBinPath string
	baselineBinErr  error
)

// repoRoot resolves the module root from this file's own compile-time path, so
// it is immune to tests that chdir.
func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func baselineBinary(t *testing.T) string {
	t.Helper()
	baselineBinOnce.Do(func() {
		dir, err := os.MkdirTemp("", "hero-baseline-bin-")
		if err != nil {
			baselineBinErr = err
			return
		}
		bin := filepath.Join(dir, "hero")
		build := exec.Command("go", "build", "-o", bin, "./cmd/hero")
		build.Dir = repoRoot()
		if out, err := build.CombinedOutput(); err != nil {
			baselineBinErr = fmt.Errorf("go build ./cmd/hero: %v\n%s", err, out)
			return
		}
		baselineBinPath = bin
	})
	if baselineBinErr != nil {
		t.Fatalf("building hero binary for baseline: %v", baselineBinErr)
	}
	return baselineBinPath
}

func runBaselineCase(t *testing.T, bin, base, root string, args []string, cond, stdin string) string {
	t.Helper()
	home := filepath.Join(base, "home")
	tmp := filepath.Join(base, "tmp")
	exitCode, stdout, stderr := runHero(t, bin, base, root, args, cond, stdin)
	return formatBaseline(base, root, home, tmp, args, cond, stdin, exitCode, stdout, stderr)
}

// runHero invokes the built binary in an isolated workspace under one of the
// three stdin conditions and returns its exit code and streams.
//
// An exit code of -1 means the process was killed by the timeout, i.e. it
// blocked. That is itself a finding: hard constraint 3 says a non-TTY
// invocation must fail fast and never hang.
func runHero(t *testing.T, bin, base, root string, args []string, cond, stdin string) (int, string, string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	home := filepath.Join(base, "home")
	tmp := filepath.Join(base, "tmp")
	for _, d := range []string{home, tmp} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	var cmd *exec.Cmd
	switch cond {
	case condClosed:
		// Close fd 0 in the child. Go's exec cannot express this directly, so
		// go through a shell: `exec "$0" "$@" 0<&-`.
		shArgs := append([]string{"-c", `exec "$0" "$@" 0<&-`, bin}, args...)
		cmd = exec.CommandContext(ctx, "/bin/sh", shArgs...)
	case condTTY:
		master, slave, err := ptytest.Open()
		if err != nil {
			t.Skipf("%v", err)
		}
		defer master.Close()
		defer slave.Close()
		cmd = exec.CommandContext(ctx, bin, args...)
		cmd.Stdin = slave
		// Write from a goroutine: the pty buffer is finite and the child may
		// not have started reading yet.
		go func() { _, _ = io.WriteString(master, stdin) }()
	default:
		cmd = exec.CommandContext(ctx, bin, args...)
		// A non-*os.File reader makes os/exec allocate a real pipe for fd 0.
		cmd.Stdin = strings.NewReader(stdin)
	}

	cmd.Dir = root
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
		"TMPDIR=" + tmp,
		// Keep git deterministic and off the developer's real identity.
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		// No HERO_* variables: the baseline must not depend on ambient state.
	}
	// Detach from any controlling terminal so /dev/tty is unavailable and
	// secret prompts fail deterministically instead of blocking on a real one.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		// A hang is itself a recorded behaviour — hard constraint 3 says
		// non-TTY must never block, so a TIMEOUT in a fixture is a finding.
		exitCode = -1
	case err != nil:
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("running %v: %v", args, err)
		}
	}

	return exitCode, stdout.String(), stderr.String()
}

func runHeroWithOpenPipe(bin, base, root string, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	home := filepath.Join(base, "home")
	tmp := filepath.Join(base, "tmp")
	for _, dir := range []string{home, tmp} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	defer reader.Close()
	defer writer.Close()

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = root
	cmd.Stdin = reader
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
		"TMPDIR=" + tmp,
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("timed out after waiting for input; output:\n%s", output.String())
	}
	return nil
}

func formatBaseline(base, root, home, tmp string, args []string, cond, stdin string, exitCode int, stdout, stderr string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "$ hero %s\n", strings.Join(args, " "))
	fmt.Fprintf(&b, "stdin-condition: %s\n", cond)
	if stdin != "" {
		fmt.Fprintf(&b, "stdin-bytes: %q\n", stdin)
	}
	if exitCode == -1 {
		b.WriteString("exit: TIMEOUT (command blocked — hard constraint 3 violation)\n")
	} else {
		fmt.Fprintf(&b, "exit: %d\n", exitCode)
	}
	b.WriteString("--- stdout ---\n")
	b.WriteString(normalizeBaseline(base, root, home, tmp, stdout))
	b.WriteString("--- stderr ---\n")
	b.WriteString(normalizeBaseline(base, root, home, tmp, stderr))
	return b.String()
}

var (
	// Absolute paths that survive root/home substitution (e.g. the resolved
	// /private prefix macOS adds, or the go build cache).
	reTmpPath   = regexp.MustCompile(`(/private)?/(var|tmp)/[^\s"']*`)
	reTimestamp = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}\S*`)
	reDuration  = regexp.MustCompile(`\b\d+(\.\d+)?(ns|µs|ms|s)\b`)
	reHexID     = regexp.MustCompile(`\b[0-9a-f]{8,}\b`)
	// Wall-clock measurements reported as a JSON number with the unit in the
	// key, e.g. `"duration_ms": 12`.
	reDurationField = regexp.MustCompile(`"duration_(ms|us|ns|s)":\s*\d+`)
)

func normalizeBaseline(base, root, home, tmp string, s string) string {
	if s != "" && !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	// Longest-first, twice over: the /private-prefixed form of a path must be
	// replaced before its bare form, or macOS's /private/var/... resolves to
	// the nonsense "/private<ROOT>". Children before their parent `base`.
	for _, sub := range []struct{ from, to string }{
		{home, "<HOME>"},
		{tmp, "<TMPDIR>"},
		{root, "<ROOT>"},
		{base, "<BASE>"},
	} {
		if sub.from == "" {
			continue
		}
		// macOS resolves /tmp and /var through /private — longest first.
		if resolved, err := filepath.EvalSymlinks(sub.from); err == nil && resolved != sub.from {
			s = strings.ReplaceAll(s, resolved, sub.to)
		}
		s = strings.ReplaceAll(s, "/private"+sub.from, sub.to)
		s = strings.ReplaceAll(s, sub.from, sub.to)
	}
	s = reTimestamp.ReplaceAllString(s, "<TIMESTAMP>")
	s = reDurationField.ReplaceAllString(s, `"duration": <DURATION>`)
	s = reDuration.ReplaceAllString(s, "<DURATION>")
	s = reTmpPath.ReplaceAllString(s, "<TMP>")
	s = reHexID.ReplaceAllString(s, "<HEX>")
	s = elideInstallManifest(s)
	return s
}

// reManifestLine matches one "  src/file.md -> dest/file.md" line from an
// install plan.
var reManifestLine = regexp.MustCompile(`^\s+\S+ -> \S+$`)

// elideInstallManifest collapses runs of install-plan file lines into a single
// marker.
//
// The manifest body enumerates every agent, command, and skill in the domain
// pack, so it churns whenever a pack file is added or renamed — changes that
// have nothing to do with prompting. Left verbatim, this baseline would go red
// on unrelated commits and train people to regenerate it reflexively, which
// would destroy the one property it exists to protect.
//
// The elision is applied identically when capturing and when comparing, and it
// touches only lines produced by `internal/install`'s plan renderer — which no
// child of this initiative modifies. Everything that this baseline is actually
// about (prompt text, prompt ordering, chosen target, error text, exit code)
// is still compared byte-for-byte.
func elideInstallManifest(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	eliding := false
	for _, line := range lines {
		if reManifestLine.MatchString(line) {
			if !eliding {
				out = append(out, "<INSTALL MANIFEST LINES ELIDED>")
				eliding = true
			}
			continue
		}
		eliding = false
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// ---------------------------------------------------------------------------
// fixtures
// ---------------------------------------------------------------------------

func setupWorkspace(t *testing.T, root string) {
	t.Helper()
	writeWorkspace(t, root)
}

func writeWorkspace(t *testing.T, dir string) {
	t.Helper()
	heroDir := filepath.Join(dir, ".hero")
	for _, d := range []string{
		heroDir,
		filepath.Join(heroDir, "planning", "features"),
		filepath.Join(heroDir, "planning", "bugs"),
		filepath.Join(heroDir, "planning", "initiatives"),
		filepath.Join(heroDir, "specs"),
		filepath.Join(heroDir, "knowledge", "notes"),
		filepath.Join(heroDir, "skills"),
		filepath.Join(heroDir, "mocks"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", d, err)
		}
	}
	cfg := config.DefaultConfig()
	if err := cfg.Save(dir); err != nil {
		t.Fatalf("save config in %s: %v", dir, err)
	}
}

// setupBareProjectDir creates a target directory with NO workspace anywhere
// above it, so `hero install project proj` takes the root-install path and
// reaches promptTarget.
func setupBareProjectDir(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "proj"), 0o755); err != nil {
		t.Fatalf("mkdir proj: %v", err)
	}
}

// setupWorkspaceWithUndeclaredSubdir puts a subfolder inside an existing
// workspace and leaves it out of subprojects.json, which is what triggers the
// "Add it as a subproject?" confirm.
func setupWorkspaceWithUndeclaredSubdir(t *testing.T, root string) {
	t.Helper()
	writeWorkspace(t, root)
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
}

// setupRootInstallCandidates creates a root that is already a workspace and
// holds detectable subproject candidates.
//
// The workspace must exist BEFORE the install: the candidate walk at
// install.go:371 is gated on `workspace.Locate(targetDir)` succeeding, and
// `hero install` does not itself create `.hero/`. Without a pre-existing
// workspace the install succeeds, the walk never runs, and this fixture
// records bytes that never touch the prompt site it claims to cover.
func setupRootInstallCandidates(t *testing.T, root string) {
	t.Helper()
	writeWorkspace(t, root)
	writeCandidate(t, filepath.Join(root, "svc-a"))
	writeCandidate(t, filepath.Join(root, "svc-b"))
}

func writeCandidate(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir candidate %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"),
		[]byte("{\"name\":\""+filepath.Base(dir)+"\"}\n"), 0o644); err != nil {
		t.Fatalf("write candidate manifest: %v", err)
	}
}

func setupSatelliteCandidates(t *testing.T, root string) {
	t.Helper()
	writeWorkspace(t, root)
	writeCandidate(t, filepath.Join(root, "svc-a"))
	writeCandidate(t, filepath.Join(root, "svc-b"))
}

// setupDeclaredSubproject declares a subproject in subprojects.json without
// materializing its satellite, which is what reconcileDeclared prompts about.
func setupDeclaredSubproject(t *testing.T, root string) {
	t.Helper()
	writeWorkspace(t, root)
	writeCandidate(t, filepath.Join(root, "declared"))
	heroDir := filepath.Join(root, ".hero")
	subs := &install.SubprojectsManifest{}
	subs.AddSubproject(install.Subproject{Path: "declared", Scope: "declared"})
	if err := install.SaveSubprojects(heroDir, subs); err != nil {
		t.Fatalf("save subprojects: %v", err)
	}
}

func setupNestedWorkspace(t *testing.T, root string) {
	t.Helper()
	writeWorkspace(t, root)
	writeWorkspace(t, filepath.Join(root, "nested"))
}

func setupSkillWithParam(t *testing.T, root string) {
	t.Helper()
	writeWorkspace(t, root)
	skill := `---
title: Baseline Skill
version: 1
---

# Baseline Skill

## Parameters

- ` + "`target`" + ` — the thing to act on

## Steps

1. Prompt agent: do something with {{target}}
`
	path := filepath.Join(root, ".hero", "skills", "baseline-skill.md")
	if err := os.WriteFile(path, []byte(skill), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}

func setupExportConflict(t *testing.T, root string) {
	t.Helper()
	writeWorkspace(t, root)
	src := filepath.Join(root, ".hero", "knowledge", "notes", "baseline.md")
	if err := os.WriteFile(src, []byte("source content\n"), 0o644); err != nil {
		t.Fatalf("write knowledge source: %v", err)
	}
	dst := filepath.Join(root, "dest", "notes")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}
	// Different content at the same relative path => a conflict to resolve.
	if err := os.WriteFile(filepath.Join(dst, "baseline.md"), []byte("destination content\n"), 0o644); err != nil {
		t.Fatalf("write knowledge destination: %v", err)
	}
}

func setupHandedBackSpec(t *testing.T, root string) {
	t.Helper()
	writeWorkspace(t, root)
	body := `---
title: Baseline Handed Back
slug: baseline-handed-back
type: feature
status: handed_back
created: 2026-01-01
---

# Baseline Handed Back

## Context

Fixture for the handoff accept prompt baseline.
`
	path := filepath.Join(root, ".hero", "planning", "features", "baseline-handed-back.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
}
