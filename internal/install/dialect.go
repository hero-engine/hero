package install

import (
	"fmt"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/methodology"
	"github.com/hero-engine/hero/internal/vocabulary"
)

// renderActiveDialectBlock returns markdown describing the active
// vocabulary and methodology (read from hero.json at opts.ProjectRoot
// or opts.TargetDir) so the managed AGENTS.md / CLAUDE.md body can teach
// the agent the workspace's dialect. Returns an empty string when no
// vocabulary or methodology is configured — engineering and legacy
// workspaces see no extra content.
//
// The block is appended to generateAgentsMdBody's output so it lives
// inside the same Hero-managed region and is rewritten on every install.
func renderActiveDialectBlock(opts Options) string {
	root := opts.ProjectRoot
	if root == "" {
		root = opts.TargetDir
	}
	if root == "" {
		return ""
	}
	cfg, err := config.Load(root)
	if err != nil {
		return ""
	}
	if cfg.Vocabulary == "" && cfg.Methodology == "" {
		return ""
	}

	vocabs, vErr := vocabulary.Load(vocabulary.CoreFS(), nil)
	methodologies, mErr := methodology.Load(methodology.CoreFS(), nil)
	if vErr != nil && mErr != nil {
		return ""
	}

	var vocab *vocabulary.Vocabulary
	if vErr == nil {
		if v, rErr := vocabulary.Resolve(&cfg, vocabs, methodologies); rErr == nil {
			vocab = v
		}
	}
	var method *methodology.Methodology
	if mErr == nil && cfg.Methodology != "" {
		if m, rErr := methodology.Resolve(&cfg, methodologies); rErr == nil {
			method = m
		}
	}

	if vocab == nil && method == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n### Active workspace dialect\n\n")
	sb.WriteString("This workspace declares the following layers in `hero.json` — speak them when you render output to the user:\n\n")
	if vocab != nil {
		sb.WriteString(fmt.Sprintf("- **Vocabulary:** `%s`", vocab.Name))
		if vocab.DisplayName != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", vocab.DisplayName))
		}
		sb.WriteString("\n")
	}
	if method != nil {
		sb.WriteString(fmt.Sprintf("- **Methodology:** `%s`", method.Name))
		if method.DisplayName != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", method.DisplayName))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	if vocab != nil && len(vocab.Kinds) > 0 {
		sb.WriteString("Render the workspace's canonical types with these display names:\n\n")
		// Stable order: walk a fixed list so the body is deterministic.
		ordered := []string{"spec.feature", "spec.bug", "spec.chore", "epic.theme", "epic.delivery"}
		any := false
		for _, k := range ordered {
			if name, ok := vocab.Kinds[k]; ok && name != "" {
				sb.WriteString(fmt.Sprintf("- `%s` → %s\n", k, name))
				any = true
			}
		}
		if !any {
			// No kinds matched our ordered list — emit at least the
			// top-level type names so the agent has something.
			for _, t := range []string{"spec", "epic", "roadmap-item"} {
				if name, ok := vocab.Types[t]; ok && name != "" {
					sb.WriteString(fmt.Sprintf("- `%s` → %s\n", t, name))
				}
			}
		}
		sb.WriteString("\n")
	}

	if vocab != nil && len(vocab.NLTriggers) > 0 {
		sb.WriteString("Natural-language phrases that map to canonical types under this vocabulary (use these to disambiguate user input):\n\n")
		for _, t := range vocab.NLTriggers {
			if len(t.Phrases) == 0 {
				continue
			}
			canonical := t.Canonical.Type
			if t.Canonical.Kind != "" {
				canonical = t.Canonical.Type + "." + t.Canonical.Kind
			}
			sb.WriteString(fmt.Sprintf("- %q → `%s`\n", strings.Join(t.Phrases, "\", \""), canonical))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("On-disk frontmatter stays canonical (`type: feature`, `type: bug`, …). The dialect above is **render-time only** — translate display terms back to canonical when routing or writing frontmatter.\n")

	return sb.String()
}
