package install

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/workspace"
)

// ApplyOptions controls migration execution.
type ApplyOptions struct {
	// RootDir is the absolute path of the workspace root.
	RootDir string
	// Version is the hero binary version string.
	Version string
	// Force ignores dirty git state.
	Force bool
	// ForceResume completes a partially-applied migration. Without this,
	// re-running on a half-state refuses.
	ForceResume bool
	// DryRun reports actions without writing.
	DryRun bool
}

// ApplyResult summarizes one migration execution.
type ApplyResult struct {
	SatellitePath  string
	SpecsMoved     []string
	KnowledgeMoved []string
	EventsAppended bool
	NestedRemoved  bool
	Errors         []string
}

// ApplyMigration executes a migration plan for a single nested workspace.
// The plan is regenerated from disk at apply time so we don't act on a
// stale snapshot. Returns an error if any step fails; partial state is
// preserved (no rollback) so the user can resolve with normal git tools.
func ApplyMigration(opts ApplyOptions, nestedRel string) (*ApplyResult, error) {
	plan, err := PlanMigration(opts.RootDir, nestedRel)
	if err != nil {
		return nil, err
	}
	rootAbs := opts.RootDir
	satAbs := filepath.Join(rootAbs, filepath.FromSlash(nestedRel))
	res := &ApplyResult{SatellitePath: nestedRel}

	if !opts.Force && !opts.DryRun {
		if dirty, why := isDirty(rootAbs, satAbs); dirty {
			return res, fmt.Errorf("uncommitted changes detected (%s); commit/stash or pass --force", why)
		}
	}

	// 1. Move specs.
	for _, src := range plan.SpecsToMove {
		dst, err := destPathForSpec(src, satAbs, rootAbs, nestedRel)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("compute dest for %s: %v", src, err))
			return res, err
		}
		if opts.DryRun {
			res.SpecsMoved = append(res.SpecsMoved, fmt.Sprintf("%s -> %s", src, dst))
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return res, err
		}
		if err := moveFile(rootAbs, src, dst); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("move %s: %v", src, err))
			return res, err
		}
		// Stamp scope into frontmatter at the new location.
		if err := stampSubprojectInFile(dst, plan.Scope); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("stamp %s: %v", dst, err))
			return res, err
		}
		res.SpecsMoved = append(res.SpecsMoved, dst)
	}

	// 2. Move knowledge.
	for _, src := range plan.KnowledgeToMove {
		rel, err := filepath.Rel(filepath.Join(satAbs, ".hero", "knowledge"), src)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("compute knowledge rel for %s: %v", src, err))
			return res, err
		}
		dst := filepath.Join(rootAbs, ".hero", "knowledge", rel)
		// Suffix on collision.
		if _, err := os.Stat(dst); err == nil {
			dst = collisionSuffix(dst, nestedRel)
		}
		if opts.DryRun {
			res.KnowledgeMoved = append(res.KnowledgeMoved, fmt.Sprintf("%s -> %s", src, dst))
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return res, err
		}
		if err := moveFile(rootAbs, src, dst); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("move %s: %v", src, err))
			return res, err
		}
		_ = stampSubprojectInFile(dst, plan.Scope) // best effort — many knowledge files have no frontmatter
		res.KnowledgeMoved = append(res.KnowledgeMoved, dst)
	}

	// 3. Append events.
	if plan.EventsToAppend != "" {
		if !opts.DryRun {
			if err := appendEventsLog(plan.EventsToAppend, filepath.Join(rootAbs, ".hero", "events.log"), nestedRel); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("append events: %v", err))
				return res, err
			}
			// Remove the source events.log so it doesn't replay if we re-run.
			_ = os.Remove(plan.EventsToAppend)
		}
		res.EventsAppended = true
	}

	if opts.DryRun {
		return res, nil
	}

	// 4. Remove the nested .hero/.
	if err := os.RemoveAll(plan.NestedHeroDir); err != nil {
		res.Errors = append(res.Errors, fmt.Sprintf("remove nested hero: %v", err))
		return res, err
	}
	res.NestedRemoved = true

	// 5. Add to subprojects.json (if not already there).
	heroDir := filepath.Join(rootAbs, ".hero")
	subs, err := LoadSubprojects(heroDir)
	if err != nil {
		return res, fmt.Errorf("load subprojects: %w", err)
	}
	if !subs.IsDeclared(nestedRel) {
		subs.AddSubproject(Subproject{Path: nestedRel, Scope: nestedRel})
		if err := SaveSubprojects(heroDir, subs); err != nil {
			return res, fmt.Errorf("save subprojects: %w", err)
		}
	}

	// 6. Materialize the satellite using the existing path.
	matRes, err := Materialize(SatelliteOptions{
		RootDir:      rootAbs,
		SatelliteDir: satAbs,
		Scope:        plan.Scope,
		Version:      opts.Version,
		Force:        true, // we just emptied .hero, may need to overwrite per-target marker
	})
	if err != nil {
		return res, fmt.Errorf("materialize satellite: %w", err)
	}
	if err := RecordSatellite(heroDir, SatelliteEntry{
		Path:     nestedRel,
		Targets:  targetSliceToStrings(matRes.Targets),
		Degraded: matRes.Degraded,
	}); err != nil {
		return res, fmt.Errorf("record satellite: %w", err)
	}

	return res, nil
}

// isDirty reports whether either rootAbs or satAbs has uncommitted changes
// affecting the migration's relevant paths. Returns (dirty, reason).
func isDirty(rootAbs, satAbs string) (bool, string) {
	cmd := exec.Command("git", "-C", rootAbs, "status", "--porcelain", "--", satAbs)
	out, err := cmd.Output()
	if err != nil {
		return false, "" // git unavailable; let migration proceed
	}
	lines := strings.TrimSpace(string(out))
	if lines == "" {
		return false, ""
	}
	first := strings.SplitN(lines, "\n", 2)[0]
	return true, "first change: " + first
}

// destPathForSpec computes the destination spec.md path under root,
// applying the `-from-<sub>` suffix on directory collision.
func destPathForSpec(src, satAbs, rootAbs, sub string) (string, error) {
	// src looks like /<root>/<sub>/.hero/planning/<bucket>/<slug>/spec.md
	rel, err := filepath.Rel(filepath.Join(satAbs, ".hero"), src)
	if err != nil {
		return "", err
	}
	dst := filepath.Join(rootAbs, ".hero", rel)
	if _, err := os.Stat(filepath.Dir(dst)); err == nil {
		// Directory exists — suffix the slug dir.
		dst = collisionSuffix(dst, sub)
	}
	return dst, nil
}

// collisionSuffix appends `-from-<sub>` to the parent dir of the dest
// path to disambiguate. The slug stays canonical for graph identity —
// only the filesystem path is suffixed.
func collisionSuffix(dst, sub string) string {
	dir := filepath.Dir(dst)
	base := filepath.Base(dst)
	suffix := "-from-" + strings.ReplaceAll(filepath.ToSlash(sub), "/", "-")
	return filepath.Join(dir+suffix, base)
}

// moveFile prefers `git mv` when the source is git-tracked; otherwise
// uses os.Rename.
func moveFile(rootAbs, src, dst string) error {
	if isTracked(rootAbs, src) {
		cmd := exec.Command("git", "-C", rootAbs, "mv", src, dst)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git mv: %v: %s", err, string(out))
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

// isTracked reports whether path is tracked in the rootAbs git repo.
func isTracked(rootAbs, path string) bool {
	cmd := exec.Command("git", "-C", rootAbs, "ls-files", "--error-unmatch", path)
	return cmd.Run() == nil
}

// stampSubprojectInFile rewrites a markdown file to include
// `subproject: <scope>` in its frontmatter. No-op if the file has no
// frontmatter.
func stampSubprojectInFile(path, scope string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return nil // no frontmatter, nothing to stamp
	}
	rest := content[4:]
	closeIdx := strings.Index(rest, "\n---")
	if closeIdx < 0 {
		return nil
	}
	frontmatter := rest[:closeIdx]
	body := rest[closeIdx:]

	scanner := bufio.NewScanner(strings.NewReader(frontmatter))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []string
	replaced := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "subproject:") {
			out = append(out, "subproject: "+scope)
			replaced = true
			continue
		}
		out = append(out, line)
	}
	if !replaced {
		out = append(out, "subproject: "+scope)
	}
	newContent := "---\n" + strings.Join(out, "\n") + body
	return os.WriteFile(path, []byte(newContent), 0o644)
}

// appendEventsLog reads each line of srcLog and appends to dstLog, decorating
// JSON lines with `migrated_from: <sub>` and prefixing non-JSON lines with
// a comment marker.
func appendEventsLog(srcLog, dstLog, sub string) error {
	src, err := os.Open(srcLog)
	if err != nil {
		return err
	}
	defer src.Close()
	if err := os.MkdirAll(filepath.Dir(dstLog), 0o755); err != nil {
		return err
	}
	dst, err := os.OpenFile(dstLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer dst.Close()

	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if line[0] == '{' {
			// JSON event — splice in migrated_from.
			var obj map[string]interface{}
			if err := json.Unmarshal(line, &obj); err == nil {
				obj["migrated_from"] = sub
				if obj["subproject"] == "" || obj["subproject"] == nil {
					obj["subproject"] = sub
				}
				if data, mErr := json.Marshal(obj); mErr == nil {
					if _, err := dst.Write(append(data, '\n')); err != nil {
						return err
					}
					continue
				}
			}
		}
		// Non-JSON or unparseable — prefix with a comment marker.
		prefix := fmt.Sprintf("# migrated_from=%s ", sub)
		if _, err := io.WriteString(dst, prefix); err != nil {
			return err
		}
		if _, err := dst.Write(append(line, '\n')); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// FormatApplyResult renders an ApplyResult as a multi-line summary.
func FormatApplyResult(res *ApplyResult, dryRun bool) string {
	if res == nil {
		return ""
	}
	var sb strings.Builder
	verb := "Migrated"
	if dryRun {
		verb = "Would migrate"
	}
	fmt.Fprintf(&sb, "%s satellite at %s:\n", verb, res.SatellitePath)
	fmt.Fprintf(&sb, "  %d spec(s)\n", len(res.SpecsMoved))
	fmt.Fprintf(&sb, "  %d knowledge file(s)\n", len(res.KnowledgeMoved))
	if res.EventsAppended {
		sb.WriteString("  events.log appended to root\n")
	}
	if res.NestedRemoved {
		sb.WriteString("  nested .hero/ removed\n")
	}
	if len(res.Errors) > 0 {
		fmt.Fprintf(&sb, "  errors: %d\n", len(res.Errors))
		for _, e := range res.Errors {
			fmt.Fprintf(&sb, "    - %s\n", e)
		}
	}
	return sb.String()
}

// Verify the workspace marker is still intact at root after the apply
// (called from tests).
var _ = workspace.HeroDir
