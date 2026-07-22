package tracker

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

type adfNode struct {
	Type    string                     `json:"type"`
	Text    string                     `json:"text,omitempty"`
	Attrs   map[string]json.RawMessage `json:"attrs,omitempty"`
	Marks   []adfMark                  `json:"marks,omitempty"`
	Content []adfNode                  `json:"content,omitempty"`
}

// UnmarshalJSON deliberately decodes recursive children and marks one at a
// time. ADF is an extensible provider document; one malformed child must not
// discard its valid siblings or the rest of the issue description.
func (node *adfNode) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	_ = json.Unmarshal(raw["type"], &node.Type)
	_ = json.Unmarshal(raw["text"], &node.Text)
	_ = json.Unmarshal(raw["attrs"], &node.Attrs)

	var children []json.RawMessage
	if json.Unmarshal(raw["content"], &children) == nil {
		for _, childRaw := range children {
			var child adfNode
			if json.Unmarshal(childRaw, &child) == nil {
				node.Content = append(node.Content, child)
			}
		}
	}
	var marks []json.RawMessage
	if json.Unmarshal(raw["marks"], &marks) == nil {
		for _, markRaw := range marks {
			var mark adfMark
			if json.Unmarshal(markRaw, &mark) == nil && mark.Type != "" {
				node.Marks = append(node.Marks, mark)
			}
		}
	}
	return nil
}

type adfMark struct {
	Type  string                     `json:"type"`
	Attrs map[string]json.RawMessage `json:"attrs,omitempty"`
}

var markdownSpecial = regexp.MustCompile(`([\\` + "`" + `*_[\]<>])`)

// jiraADFToMarkdown converts Jira Cloud's recursive Atlassian Document Format to
// one deterministic Markdown representation. Jira Server and Data Center can
// still return a plain JSON string; that value passes through byte-for-byte.
func jiraADFToMarkdown(raw json.RawMessage) string {
	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return plain
	}

	var root adfNode
	if err := json.Unmarshal(raw, &root); err != nil {
		return ""
	}
	return strings.TrimSpace(renderADFBlock(root, 0))
}

func renderADFBlock(node adfNode, listDepth int) string {
	switch node.Type {
	case "doc":
		return renderADFBlocks(node.Content, listDepth)
	case "paragraph":
		return renderADFInlineNodes(node.Content)
	case "heading":
		level := adfIntAttr(node.Attrs, "level", 1)
		if level < 1 || level > 6 {
			level = 1
		}
		return strings.Repeat("#", level) + " " + renderADFInlineNodes(node.Content)
	case "orderedList":
		return renderADFList(node, listDepth, true)
	case "bulletList":
		return renderADFList(node, listDepth, false)
	case "taskList":
		return renderADFTaskList(node, listDepth)
	case "codeBlock":
		body := adfDescendantText(node)
		fence := strings.Repeat("`", longestBacktickRun(body)+1)
		if len(fence) < 3 {
			fence = "```"
		}
		language := sanitizeFenceInfo(adfStringAttr(node.Attrs, "language"))
		return fence + language + "\n" + body + "\n" + fence
	case "blockquote":
		body := renderADFBlocks(node.Content, listDepth)
		if body == "" {
			return ""
		}
		lines := strings.Split(body, "\n")
		for i := range lines {
			lines[i] = "> " + lines[i]
		}
		return strings.Join(lines, "\n")
	case "panel":
		panelType := strings.ToLower(adfStringAttr(node.Attrs, "panelType"))
		if panelType == "" {
			panelType = strings.ToLower(adfStringAttr(node.Attrs, "type"))
		}
		switch panelType {
		case "info", "note", "warning", "error", "success":
			panelType = strings.ToUpper(panelType[:1]) + panelType[1:]
		default:
			panelType = "Panel"
		}
		body := renderADFBlocks(node.Content, listDepth)
		result := "> **" + escapeMarkdown(panelType) + ":**"
		if body == "" {
			return result
		}
		lines := strings.Split(body, "\n")
		for i := range lines {
			lines[i] = "> " + lines[i]
		}
		return result + "\n" + strings.Join(lines, "\n")
	case "rule":
		return "---"
	case "listItem", "taskItem":
		return renderADFBlocks(node.Content, listDepth)
	default:
		// ADF grows new node types over time. Unknown containers must not make
		// their recognized descendants disappear.
		if len(node.Content) > 0 {
			return renderADFBlocks(node.Content, listDepth)
		}
		return renderADFInline(node)
	}
}

func renderADFBlocks(nodes []adfNode, listDepth int) string {
	parts := make([]string, 0, len(nodes))
	for _, child := range nodes {
		part := strings.TrimSpace(renderADFBlock(child, listDepth))
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "\n\n")
}

func renderADFList(node adfNode, depth int, ordered bool) string {
	indent := strings.Repeat("    ", depth)
	start := adfIntAttr(node.Attrs, "order", 1)
	if start < 1 {
		start = 1
	}
	var lines []string
	itemIndex := 0
	for _, item := range node.Content {
		if item.Type != "listItem" {
			fallback := strings.TrimSpace(renderADFBlock(item, depth+1))
			if fallback != "" {
				lines = append(lines, indent+fallback)
			}
			continue
		}
		marker := "- "
		if ordered {
			marker = strconv.Itoa(start+itemIndex) + ". "
		}
		itemIndex++
		lines = append(lines, renderADFListItem(item, indent, marker, depth))
	}
	return strings.Join(lines, "\n")
}

func renderADFListItem(item adfNode, indent, marker string, depth int) string {
	var bodyParts []string
	var nested []string
	for _, child := range item.Content {
		switch child.Type {
		case "orderedList", "bulletList", "taskList":
			if part := renderADFBlock(child, depth+1); strings.TrimSpace(part) != "" {
				nested = append(nested, strings.TrimRight(part, " \n"))
			}
		default:
			if part := strings.TrimSpace(renderADFBlock(child, depth)); part != "" {
				bodyParts = append(bodyParts, part)
			}
		}
	}
	body := strings.Join(bodyParts, "\n\n")
	if body == "" {
		body = " "
	}
	continuation := indent + strings.Repeat(" ", len(marker))
	bodyLines := strings.Split(body, "\n")
	for i := 1; i < len(bodyLines); i++ {
		bodyLines[i] = continuation + bodyLines[i]
	}
	result := indent + marker + strings.Join(bodyLines, "\n")
	if len(nested) > 0 {
		result += "\n" + strings.Join(nested, "\n")
	}
	return result
}

func renderADFTaskList(node adfNode, depth int) string {
	indent := strings.Repeat("    ", depth)
	var lines []string
	for _, item := range node.Content {
		if item.Type != "taskItem" {
			continue
		}
		marker := "- [ ] "
		state := strings.ToLower(adfStringAttr(item.Attrs, "state"))
		if state == "done" || state == "complete" || state == "checked" {
			marker = "- [x] "
		}
		lines = append(lines, renderADFListItem(item, indent, marker, depth))
	}
	return strings.Join(lines, "\n")
}

func renderADFInlineNodes(nodes []adfNode) string {
	var b strings.Builder
	for _, child := range nodes {
		b.WriteString(renderADFInline(child))
	}
	return b.String()
}

func renderADFInline(node adfNode) string {
	var value string
	switch node.Type {
	case "text":
		value = escapeMarkdown(node.Text)
	case "hardBreak":
		return "<br>\n"
	case "emoji":
		value = adfStringAttr(node.Attrs, "text")
		if value == "" {
			value = adfStringAttr(node.Attrs, "shortName")
			if value != "" && !strings.HasPrefix(value, ":") {
				value = ":" + strings.Trim(value, ":") + ":"
			}
		}
	case "mention":
		value = adfStringAttr(node.Attrs, "text")
		if value == "" {
			value = adfStringAttr(node.Attrs, "displayName")
			if value != "" && !strings.HasPrefix(value, "@") {
				value = "@" + value
			}
		}
		if value == "" {
			value = adfStringAttr(node.Attrs, "id")
			if value != "" && !strings.HasPrefix(value, "@") {
				value = "@" + value
			}
		}
		if value == "" {
			value = "@mention"
		}
	case "status":
		value = adfStringAttr(node.Attrs, "text")
		if value == "" {
			value = "status"
		}
		value = "[" + escapeMarkdown(value) + "]"
	case "inlineCard", "blockCard":
		label := adfStringAttr(node.Attrs, "title")
		if label == "" {
			label = adfStringAttr(node.Attrs, "label")
		}
		target := adfStringAttr(node.Attrs, "url")
		if target != "" && label != "" {
			value = markdownLink(escapeMarkdown(label), target, "")
		} else if target != "" {
			value = "<" + strings.ReplaceAll(target, ">", "%3E") + ">"
		} else if label != "" {
			value = escapeMarkdown(label)
		} else {
			value = "[card]"
		}
	case "date":
		value = adfStringAttr(node.Attrs, "timestamp")
	case "media":
		label := adfStringAttr(node.Attrs, "alt")
		if label == "" {
			label = adfStringAttr(node.Attrs, "title")
		}
		if label == "" {
			label = adfStringAttr(node.Attrs, "filename")
		}
		if label == "" {
			label = "attachment"
		}
		if target := adfStringAttr(node.Attrs, "url"); target != "" {
			value = "![" + escapeMarkdown(label) + "](" + escapeMarkdownTarget(target) + ")"
		} else {
			value = "[media: " + escapeMarkdown(label) + "]"
		}
	default:
		if len(node.Content) > 0 {
			value = renderADFInlineNodes(node.Content)
		} else if node.Text != "" {
			value = escapeMarkdown(node.Text)
		}
	}
	if value == "" {
		return ""
	}
	return applyADFMarks(value, node.Text, node.Marks)
}

func applyADFMarks(value, rawText string, marks []adfMark) string {
	has := func(kind string) (adfMark, bool) {
		for _, mark := range marks {
			if mark.Type == kind {
				return mark, true
			}
		}
		return adfMark{}, false
	}
	if _, ok := has("code"); ok {
		fence := strings.Repeat("`", longestBacktickRun(rawText)+1)
		if fence == "" {
			fence = "`"
		}
		padding := ""
		if strings.HasPrefix(rawText, " ") || strings.HasSuffix(rawText, " ") || strings.HasPrefix(rawText, "`") || strings.HasSuffix(rawText, "`") {
			padding = " "
		}
		value = fence + padding + rawText + padding + fence
	}
	if _, ok := has("strong"); ok {
		value = "**" + value + "**"
	}
	if _, ok := has("em"); ok {
		value = "_" + value + "_"
	}
	if _, ok := has("strike"); ok {
		value = "~~" + value + "~~"
	}
	if mark, ok := has("link"); ok {
		if target := adfStringAttr(mark.Attrs, "href"); target != "" {
			value = markdownLink(value, target, adfStringAttr(mark.Attrs, "title"))
		}
	}
	return value
}

func markdownLink(label, target, title string) string {
	result := "[" + label + "](" + escapeMarkdownTarget(target)
	if title != "" {
		result += ` "` + strings.ReplaceAll(title, `"`, `\"`) + `"`
	}
	return result + ")"
}

func escapeMarkdownTarget(target string) string {
	return strings.NewReplacer(" ", "%20", "(", "%28", ")", "%29").Replace(target)
}

func adfDescendantText(node adfNode) string {
	if node.Text != "" {
		return node.Text
	}
	var b strings.Builder
	for _, child := range node.Content {
		if child.Type == "hardBreak" {
			b.WriteByte('\n')
			continue
		}
		b.WriteString(adfDescendantText(child))
	}
	return b.String()
}

func adfStringAttr(attrs map[string]json.RawMessage, key string) string {
	raw, ok := attrs[key]
	if !ok {
		return ""
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return value
	}
	return ""
}

func adfIntAttr(attrs map[string]json.RawMessage, key string, fallback int) int {
	raw, ok := attrs[key]
	if !ok {
		return fallback
	}
	var value int
	if json.Unmarshal(raw, &value) == nil {
		return value
	}
	return fallback
}

func escapeMarkdown(value string) string {
	return markdownSpecial.ReplaceAllString(value, `\$1`)
}

func sanitizeFenceInfo(value string) string {
	value = strings.TrimSpace(value)
	for _, r := range value {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && !strings.ContainsRune("_+-#.", r) {
			return ""
		}
	}
	if value == "" {
		return ""
	}
	return value
}

func longestBacktickRun(value string) int {
	longest, current := 0, 0
	for _, r := range value {
		if r == '`' {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	return longest
}
