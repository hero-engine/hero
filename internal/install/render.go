package install

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// render.go — per-harness format rendering of canonical content.
//
// Most install targets symlink (or render copy) the canonical
// agents/commands/skills tree directly because their consuming
// harness reads the same file format Hero authors. Codex agents
// (TOML) and Copilot agents/commands (.prompt.md) are exceptions:
// the consuming format genuinely differs from canonical markdown.
// This file provides the small helpers those targets use to render
// rather than symlink.
//
// Single canonical source is preserved — these helpers READ
// canonical bytes from opts.sourceFS() and emit a different shape
// at the destination. They never modify canonical content.

// canonicalEntry is a single canonical file from the embedded source
// FS, ready for rendering.
type canonicalEntry struct {
	Name        string // filename stem (e.g. "engineer" for "engineer.md")
	SourcePath  string // path within sourceFS (e.g. "agents/engineer.md")
	Frontmatter map[string]string // parsed YAML frontmatter, key→trimmed-value
	Body        []byte // markdown body after the closing `---`
	Raw         []byte // full file bytes
}

// renderToFile renders entries from a canonical kind dir into destDir
// via fn(entry) → (destFilename, renderedBytes). Skips non-.md
// entries. Honors opts.DryRun. Records each write under result.Copied.
func renderToFile(opts Options, result *Result, kind, destDir string, fn func(canonicalEntry) (string, []byte, error)) error {
	srcFS := opts.sourceFS()
	if srcFS == nil {
		return fmt.Errorf("no content source available")
	}
	entries, err := fs.ReadDir(srcFS, kind)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		srcPath := path.Join(kind, e.Name())
		raw, err := fs.ReadFile(srcFS, srcPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", srcPath, err)
		}
		entry := canonicalEntry{
			Name:       strings.TrimSuffix(e.Name(), ".md"),
			SourcePath: srcPath,
			Raw:        raw,
		}
		entry.Frontmatter, entry.Body = parseSimpleFrontmatter(raw)
		destName, rendered, err := fn(entry)
		if err != nil {
			return fmt.Errorf("render %s: %w", srcPath, err)
		}
		if destName == "" {
			continue // renderer chose to skip this entry
		}
		dst := filepath.Join(destDir, destName)
		if opts.DryRun {
			logRendered(opts, dst, kind)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, rendered, 0o644); err != nil {
			return err
		}
		result.Copied = append(result.Copied, CopyAction{Source: srcPath, Dest: dst})
		logRendered(opts, dst, kind)
	}
	return nil
}

func logRendered(opts Options, dst, kind string) {
	if opts.Quiet {
		return
	}
	fmt.Printf("  %s -> %s (rendered)\n", kind, dst)
}

// parseSimpleFrontmatter returns (frontmatterFields, body) from a
// markdown file with `---\n...\n---\n` YAML frontmatter. Only handles
// flat string values like `key: value` — sufficient for Hero's
// canonical agents/commands/skills shape. Returns (empty map, raw)
// if the file has no frontmatter.
func parseSimpleFrontmatter(raw []byte) (map[string]string, []byte) {
	out := map[string]string{}
	const marker = "---\n"
	s := string(raw)
	if !strings.HasPrefix(s, marker) {
		return out, raw
	}
	rest := s[len(marker):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return out, raw
	}
	fm := rest[:end]
	body := rest[end+len("\n---"):]
	body = strings.TrimPrefix(body, "\n")
	for _, line := range strings.Split(fm, "\n") {
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// Trim YAML quoting if present.
		val = strings.TrimPrefix(val, "\"")
		val = strings.TrimSuffix(val, "\"")
		val = strings.TrimPrefix(val, "'")
		val = strings.TrimSuffix(val, "'")
		out[key] = val
	}
	return out, []byte(body)
}

// renderCodexAgentToml emits a Codex .toml subagent definition from a
// canonical agent markdown entry. Codex requires `developer_instructions`;
// `name` and `description` are optional but Hero populates them.
//
// Output shape:
//
//	name = "<entry.Name>"
//	description = "<entry.Frontmatter[description]>"
//	developer_instructions = """
//	<entry.Body>
//	"""
//
// Returns the destination filename ("<entry.Name>.toml") and the
// rendered bytes. If the body contains a literal `"""`, the renderer
// replaces it with `\"\"\"` to keep TOML parsable — this matches
// go-toml's escape conventions.
func renderCodexAgentToml(entry canonicalEntry) (string, []byte, error) {
	name := entry.Frontmatter["name"]
	if name == "" {
		name = entry.Name
	}
	desc := entry.Frontmatter["description"]
	body := strings.TrimSpace(string(entry.Body))
	body = strings.ReplaceAll(body, `"""`, `\"\"\"`)
	var out bytes.Buffer
	fmt.Fprintf(&out, "name = %q\n", name)
	if desc != "" {
		fmt.Fprintf(&out, "description = %q\n", desc)
	}
	out.WriteString("developer_instructions = \"\"\"\n")
	out.WriteString(body)
	out.WriteString("\n\"\"\"\n")
	return entry.Name + ".toml", out.Bytes(), nil
}

// renderCommandAsCodexSkill renders a canonical command markdown file as
// a Codex-loadable skill at command-<name>/SKILL.md. The rendered skill
// includes an execution preamble so Codex treats the file as a workflow
// to execute rather than documentation to summarize.
func renderCommandAsCodexSkill(entry canonicalEntry) (string, []byte, error) {
	desc := entry.Frontmatter["description"]
	if desc == "" {
		desc = "Hero /" + entry.Name + " workflow — follow these steps to execute the command"
	}

	var out bytes.Buffer
	fmt.Fprintf(&out, "---\nname: command-%s\ndescription: %s\nmetadata:\n  purpose: command-workflow\n---\n\n", entry.Name, desc)
	out.WriteString("> **This is a Hero workflow for Codex.** Read each step below and execute it in sequence.\n")
	out.WriteString("> Do NOT summarize or treat these steps as documentation.\n")
	out.WriteString("> Do NOT update spec frontmatter as a substitute for doing the actual work described.\n\n")
	body := bytes.TrimLeft(entry.Body, "\n")
	out.Write(body)
	if len(body) > 0 && body[len(body)-1] != '\n' {
		out.WriteByte('\n')
	}
	return "command-" + entry.Name + "/SKILL.md", out.Bytes(), nil
}

// renderCopilotPromptFile emits a Copilot .prompt.md file from a
// canonical agent or command markdown entry. Copilot prompt files are
// markdown with optional YAML frontmatter; the renderer writes a
// minimal frontmatter (description only) plus the canonical body.
//
// Returns destination filename ("<entry.Name>.prompt.md") and bytes.
func renderCopilotPromptFile(entry canonicalEntry) (string, []byte, error) {
	desc := entry.Frontmatter["description"]
	body := strings.TrimLeft(string(entry.Body), "\n")
	var out bytes.Buffer
	out.WriteString("---\n")
	if desc != "" {
		fmt.Fprintf(&out, "description: %s\n", desc)
	}
	out.WriteString("---\n")
	out.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		out.WriteString("\n")
	}
	return entry.Name + ".prompt.md", out.Bytes(), nil
}

