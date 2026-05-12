package skills

import (
	"fmt"
	"strings"
	"time"
)

// CaptureTemplate builds a skill markdown template from a slice of commands.
// Each command that starts with "hero" becomes a hero step; others become Run: steps.
func CaptureTemplate(name, title string, commands []string) string {
	if title == "" {
		title = name
	}

	date := time.Now().Format("2006-01-02")

	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "title: %s\n", title)
	fmt.Fprintf(&sb, "slug: %s\n", name)
	sb.WriteString("version: 1\n")
	sb.WriteString("tags: []\n")
	fmt.Fprintf(&sb, "created: %s\n", date)
	sb.WriteString("---\n\n")

	fmt.Fprintf(&sb, "# %s\n\n", title)
	sb.WriteString("<!-- Describe what this skill does. -->\n\n")

	sb.WriteString("## Parameters\n\n")
	sb.WriteString("<!-- Add parameters if needed:\n")
	sb.WriteString("- `param_name` — description of the parameter\n")
	sb.WriteString("-->\n\n")

	sb.WriteString("## Steps\n\n")

	if len(commands) == 0 {
		sb.WriteString("1. Run: echo \"hello\"\n")
	} else {
		for i, cmd := range commands {
			cmd = strings.TrimSpace(cmd)
			if cmd == "" {
				continue
			}
			if strings.HasPrefix(cmd, "hero ") || cmd == "hero" {
				fmt.Fprintf(&sb, "%d. %s\n", i+1, cmd)
			} else {
				fmt.Fprintf(&sb, "%d. Run: %s\n", i+1, cmd)
			}
		}
	}

	sb.WriteString("\n## Notes\n\n")
	sb.WriteString("<!-- Optional notes about this skill. -->\n")

	return sb.String()
}
