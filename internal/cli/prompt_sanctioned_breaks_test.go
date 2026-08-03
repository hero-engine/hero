//go:build unix

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This file holds the POSITIVE assertions for the two — and only two —
// deliberate behavior changes in the `cli-prompt-package-core` child.
//
// Both are silent-wrong-behavior bugs rather than crashes: today they exit
// zero and do the wrong thing. A test that merely passed after the fix would
// prove nothing, because it would have passed before it too. Each assertion
// below was therefore first confirmed to FAIL against the pre-fix code.
//
// They are written as positive assertions ("the system MUST refuse", "the
// system MUST exit non-zero") rather than as golden-file diffs, so that a
// future reviewer cannot quietly "restore backward compatibility" and still
// have the suite go green. Restoring either old behavior fails these tests
// loudly and by name.
//
// Both changes are documented in docs/release-notes/. Both are also visible
// as intentional diffs in the AC-1 baseline fixtures.

// TestSanctionedBreakSecretRefusesUnprotectedStream asserts the security fix.
//
// Before: `hero admin users passwd <user>` fell through to fmt.Scanln when
// there was no TTY, accepting a password from whatever was on stdin — a pipe,
// a here-doc, a file, a CI log. The password was read successfully and the
// command proceeded to the user lookup.
//
// After: the read is refused outright and the error names how to proceed.
//
// The discriminator is precise. If the command reports that the user does not
// exist, it got PAST the password read, which means it accepted a password
// off the pipe — the exact defect. That is asserted against explicitly rather
// than merely checking for "an error", because the pre-fix code also errored,
// just for the wrong reason and only after the damage was done.
func TestSanctionedBreakSecretRefusesUnprotectedStream(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exitCode, stdout, stderr := runHero(t, bin, base, root,
		[]string{"admin", "users", "passwd", "alice"}, condPipe, "hunter2\nhunter2\n")
	combined := stdout + stderr

	if exitCode == 0 {
		t.Errorf("exit = 0, want non-zero: a password read from a non-TTY stream must fail\n%s", combined)
	}
	if strings.Contains(combined, `user "alice" not found`) {
		t.Errorf("command reached the user lookup, which means it ACCEPTED the password from the pipe.\n"+
			"This is the echoed-password fallback the fix removes.\n%s", combined)
	}
	if !mentionsTerminalRefusal(combined) {
		t.Errorf("error does not explain that a terminal is required:\n%s", combined)
	}
	// The refusal must also name a way forward, per AC-5.
	if !strings.Contains(combined, "--password") {
		t.Errorf("refusal does not name the non-interactive alternative (--password):\n%s", combined)
	}
}

// TestSanctionedBreakSecretRefusesClosedStdin covers the same refusal under the
// other non-TTY condition.
//
// This case took a different code path before the fix: with stdin closed, the
// Go runtime backfills fd 0 with /dev/null, which IS a character device, so
// the old ModeCharDevice check believed it had a terminal and called
// term.ReadPassword on it — producing a bare "operation not supported by
// device" with no guidance. Both paths must now land on the same clear
// refusal.
func TestSanctionedBreakSecretRefusesClosedStdin(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newSanctionedWorkspace(t)

	exitCode, stdout, stderr := runHero(t, bin, base, root,
		[]string{"admin", "users", "passwd", "alice"}, condClosed, "")
	combined := stdout + stderr

	if exitCode == 0 {
		t.Errorf("exit = 0, want non-zero\n%s", combined)
	}
	if strings.Contains(combined, "operation not supported by device") {
		t.Errorf("still surfacing the raw ioctl failure instead of a clear refusal:\n%s", combined)
	}
	if !mentionsTerminalRefusal(combined) {
		t.Errorf("error does not explain that a terminal is required:\n%s", combined)
	}
}

// TestSanctionedBreakInstallTargetFailsOnNonTTY asserts the CI-correctness fix.
//
// Before: `hero install project <dir>` with no --target and no TTY returned
// install.TargetOpenCode and exited 0, so CI silently installed the wrong
// harness. Nothing in the output said so.
//
// After: it exits non-zero.
//
// Checking the exit code alone is not enough — the old behavior also exited 0
// only because it had chosen a target, so this asserts that opencode was NOT
// installed as well.
func TestSanctionedBreakInstallTargetFailsOnNonTTY(t *testing.T) {
	for _, cond := range []string{condClosed, condPipe} {
		t.Run(cond, func(t *testing.T) {
			bin := baselineBinary(t)
			base, root := newBareInstallTarget(t)

			exitCode, stdout, stderr := runHero(t, bin, base, root,
				[]string{"install", "project", "proj"}, cond, "")
			combined := stdout + stderr
			errLine := errorLine(combined)

			if exitCode == 0 {
				t.Errorf("exit = 0, want non-zero: `hero install project` with no TTY and no --target "+
					"must fail rather than silently choosing a harness\n%s", combined)
			}
			if strings.Contains(combined, "target:      opencode") {
				t.Errorf("silently defaulted to opencode — this is the bug the fix removes\n%s", combined)
			}
			// Assert on the error line, not the whole output: cobra's usage
			// dump lists every flag including --target, so a substring check
			// over the combined streams passes even when the command failed
			// for an entirely unrelated reason.
			if !strings.Contains(errLine, "--target") {
				t.Errorf("error line does not name --target as the way to proceed.\ngot error line: %q\nfull output:\n%s", errLine, combined)
			}
			if !mentionsTerminalRefusal(errLine) {
				t.Errorf("error line does not explain that no terminal was available.\ngot error line: %q\nfull output:\n%s", errLine, combined)
			}
			// Nothing should have been written for the defaulted harness.
			if _, err := os.Stat(filepath.Join(root, "proj", ".opencode")); err == nil {
				t.Error("an opencode harness was installed despite the failure")
			}
		})
	}
}

// TestSanctionedBreakInstallTargetRejectsUnknownValue asserts AC-7: a typo at
// the target prompt must be rejected rather than turned into install.Target
// of arbitrary text.
//
// This is driven through the flag rather than the prompt because the prompt
// only fires on a TTY; the flag path shares the same validation set, and
// TestInstallTargetValidationSharedByFlagAndPrompt pins that they cannot
// drift apart.
func TestSanctionedBreakInstallTargetRejectsUnknownValue(t *testing.T) {
	bin := baselineBinary(t)
	base, root := newBareInstallTarget(t)

	exitCode, stdout, stderr := runHero(t, bin, base, root,
		[]string{"install", "project", "proj", "--target", "clade"}, condPipe, "")
	combined := stdout + stderr
	errLine := errorLine(combined)

	if exitCode == 0 {
		t.Errorf("exit = 0, want non-zero for an unknown target\n%s", combined)
	}
	if !strings.Contains(errLine, "clade") {
		t.Errorf("error line does not quote the rejected value.\ngot error line: %q\nfull output:\n%s", errLine, combined)
	}
	if _, err := os.Stat(filepath.Join(root, "proj", ".clade")); err == nil {
		t.Error("an unvalidated target directory was created")
	}
}

func mentionsTerminalRefusal(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "terminal") || strings.Contains(lower, "tty")
}

// errorLine returns cobra's "Error: ..." line, or "" when the command
// succeeded.
//
// Assertions about *why* a command failed must look here rather than at the
// combined streams. Cobra dumps full usage on error, and that dump names every
// flag the command has — so a naive `strings.Contains(output, "--target")`
// passes for any failure whatsoever. That produced a false pass during
// authoring of this file.
func errorLine(combined string) string {
	for _, line := range strings.Split(combined, "\n") {
		if strings.HasPrefix(line, "Error: ") {
			return line
		}
	}
	return ""
}

func newSanctionedWorkspace(t *testing.T) (base, root string) {
	t.Helper()
	base = t.TempDir()
	root = filepath.Join(base, "work")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir work root: %v", err)
	}
	writeWorkspace(t, root)
	return base, root
}

// newBareInstallTarget creates a `proj` directory with NO Hero workspace at or
// above it.
//
// This matters: if the root is already a workspace, `hero install project proj`
// takes the satellite path, which never calls promptTarget at all. The command
// then fails for an unrelated reason and a loosely-written assertion passes
// against a code path it never touched.
func newBareInstallTarget(t *testing.T) (base, root string) {
	t.Helper()
	base = t.TempDir()
	root = filepath.Join(base, "work")
	if err := os.MkdirAll(filepath.Join(root, "proj"), 0o755); err != nil {
		t.Fatalf("mkdir proj: %v", err)
	}
	return base, root
}
