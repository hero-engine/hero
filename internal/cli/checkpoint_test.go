package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/config"
)

func TestWriteFileIfChangedSkipsIdenticalContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "NEXT.md")
	content := []byte("same\n")

	changed, err := writeFileIfChanged(path, content, 0o644)
	if err != nil {
		t.Fatalf("writeFileIfChanged first write: %v", err)
	}
	if !changed {
		t.Fatal("first write should report changed")
	}
	before := mustStatModTime(t, path)
	time.Sleep(20 * time.Millisecond)

	changed, err = writeFileIfChanged(path, content, 0o644)
	if err != nil {
		t.Fatalf("writeFileIfChanged second write: %v", err)
	}
	if changed {
		t.Fatal("identical write should report unchanged")
	}
	after := mustStatModTime(t, path)
	if !after.Equal(before) {
		t.Fatalf("mtime advanced on no-op write: before=%s after=%s", before, after)
	}
}

func TestWriteProjectedFilePreservesUpdatedOnlyChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "NEXT.md")
	original := []byte(`---
updated: 2026-05-04T10:00:00Z
repo: hero
---

## Next

same
`)
	proposed := []byte(`---
updated: 2026-05-04T11:00:00Z
repo: hero
---

## Next

same
`)

	if changed, err := writeProjectedFileIfSemanticChanged(path, original, 0o644); err != nil {
		t.Fatalf("initial projected write: %v", err)
	} else if !changed {
		t.Fatal("initial projected write should report changed")
	}
	before := mustStatModTime(t, path)
	time.Sleep(20 * time.Millisecond)

	changed, err := writeProjectedFileIfSemanticChanged(path, proposed, 0o644)
	if err != nil {
		t.Fatalf("second projected write: %v", err)
	}
	if changed {
		t.Fatal("updated-only projected write should report unchanged")
	}
	after := mustStatModTime(t, path)
	if !after.Equal(before) {
		t.Fatalf("mtime advanced on updated-only projection: before=%s after=%s", before, after)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "2026-05-04T11:00:00Z") {
		t.Fatalf("updated timestamp should have been preserved, got:\n%s", data)
	}
	if !strings.Contains(string(data), "2026-05-04T10:00:00Z") {
		t.Fatalf("original updated timestamp missing, got:\n%s", data)
	}
}

func TestWriteProjectedFileUpdatesOnSemanticChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "NEXT.md")
	original := []byte(`---
updated: 2026-05-04T10:00:00Z
repo: hero
---

## Next

old
`)
	proposed := []byte(`---
updated: 2026-05-04T11:00:00Z
repo: hero
---

## Next

new
`)

	if _, err := writeProjectedFileIfSemanticChanged(path, original, 0o644); err != nil {
		t.Fatalf("initial projected write: %v", err)
	}
	changed, err := writeProjectedFileIfSemanticChanged(path, proposed, 0o644)
	if err != nil {
		t.Fatalf("semantic projected write: %v", err)
	}
	if !changed {
		t.Fatal("semantic projected write should report changed")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "2026-05-04T11:00:00Z") || !strings.Contains(string(data), "new") {
		t.Fatalf("semantic change was not written, got:\n%s", data)
	}
}

func TestWriteProjectedNextMDSkipsUpdatedOnlyChange(t *testing.T) {
	env := newTestEnv(t)
	nextPath := filepath.Join(env.heroDir, "NEXT.md")

	if err := writeProjectedNextMD(nextPath, env.dir, env.heroDir); err != nil {
		t.Fatalf("writeProjectedNextMD first write: %v", err)
	}
	before := mustStatModTime(t, nextPath)
	firstBody, err := os.ReadFile(nextPath)
	if err != nil {
		t.Fatalf("ReadFile first body: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	if err := writeProjectedNextMD(nextPath, env.dir, env.heroDir); err != nil {
		t.Fatalf("writeProjectedNextMD second write: %v", err)
	}
	after := mustStatModTime(t, nextPath)
	if !after.Equal(before) {
		t.Fatalf("projected NEXT.md mtime advanced on updated-only change: before=%s after=%s", before, after)
	}
	secondBody, err := os.ReadFile(nextPath)
	if err != nil {
		t.Fatalf("ReadFile second body: %v", err)
	}
	if string(secondBody) != string(firstBody) {
		t.Fatalf("projected NEXT.md changed on updated-only render:\nfirst:\n%s\nsecond:\n%s", firstBody, secondBody)
	}
}

func TestWriteUserHandoffFileSkipsUpdatedOnlyChange(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/tester"}
	userPath := filepath.Join(env.heroDir, nextDirName, "tester.md")

	if err := writeUserHandoffFile(env.dir, env.heroDir, cfg); err != nil {
		t.Fatalf("writeUserHandoffFile first write: %v", err)
	}
	before := mustStatModTime(t, userPath)
	firstBody, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("ReadFile first body: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	if err := writeUserHandoffFile(env.dir, env.heroDir, cfg); err != nil {
		t.Fatalf("writeUserHandoffFile second write: %v", err)
	}
	after := mustStatModTime(t, userPath)
	if !after.Equal(before) {
		t.Fatalf("user handoff mtime advanced on updated-only change: before=%s after=%s", before, after)
	}
	secondBody, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("ReadFile second body: %v", err)
	}
	if string(secondBody) != string(firstBody) {
		t.Fatalf("user handoff changed on updated-only render:\nfirst:\n%s\nsecond:\n%s", firstBody, secondBody)
	}
}

func TestNextCheckpointQuietIsSilent(t *testing.T) {
	_ = newTestEnv(t)

	out, err := runCmd("next", "checkpoint", "-q")
	if err != nil {
		t.Fatalf("next checkpoint -q returned error: %v", err)
	}
	if out != "" {
		t.Fatalf("quiet checkpoint should not print stdout, got %q", out)
	}
}

func mustStatModTime(t *testing.T, path string) time.Time {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	return info.ModTime()
}
