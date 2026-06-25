package knowledge

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"

	"gopkg.in/yaml.v3"
)

type ConflictStrategy string

const (
	ConflictFail        ConflictStrategy = "fail"
	ConflictSkip        ConflictStrategy = "skip"
	ConflictOverwrite   ConflictStrategy = "overwrite"
	ConflictMerge       ConflictStrategy = "merge"
	ConflictInteractive ConflictStrategy = "interactive"
)

type Options struct {
	Strategy ConflictStrategy
	Prompt   func(Conflict) (ConflictStrategy, error)
}

type treeExportOptions struct {
	label       string
	sourceLabel string
	merge       func(exportEntry, []byte, []byte) ([]byte, error)
}

type Summary struct {
	Copied      int
	Skipped     int
	Overwritten int
	Merged      int
	Identical   int
	Conflicts   int
}

type Conflict struct {
	RelPath    string
	SourcePath string
	DestPath   string
	Reason     string
}

type ConflictError struct {
	Label     string
	Conflicts []Conflict
}

func (e *ConflictError) Error() string {
	label := "knowledge export"
	if e.Label != "" {
		label = e.Label
	}
	if len(e.Conflicts) == 0 {
		return label + " conflict"
	}
	paths := make([]string, 0, len(e.Conflicts))
	for _, c := range e.Conflicts {
		if c.Reason == "" {
			paths = append(paths, c.RelPath)
			continue
		}
		paths = append(paths, fmt.Sprintf("%s (%s)", c.RelPath, c.Reason))
	}
	return label + " conflicts: " + strings.Join(paths, ", ")
}

func ExportMocks(srcMocksDir, destination string, opts Options) (*Summary, error) {
	return exportTree(srcMocksDir, destination, opts, treeExportOptions{
		label:       "mock export",
		sourceLabel: "source mocks dir",
		merge: func(entry exportEntry, destBytes, srcBytes []byte) ([]byte, error) {
			return nil, errors.New("merge is not supported for mock artifacts")
		},
	})
}

type exportEntry struct {
	rel string
	src string
	dst string
	dir bool
}

type exportOp int

const (
	opCreateDir exportOp = iota
	opCopy
	opSkip
	opOverwrite
	opMerge
	opIdentical
)

type exportPlan struct {
	entry exportEntry
	op    exportOp
	bytes []byte
}

func Export(srcKnowledgeDir, destination string, opts Options) (*Summary, error) {
	return exportTree(srcKnowledgeDir, destination, opts, treeExportOptions{
		label:       "knowledge export",
		sourceLabel: "source knowledge dir",
		merge:       mergeMarkdown,
	})
}

func exportTree(srcDir, destination string, opts Options, treeOpts treeExportOptions) (*Summary, error) {
	if treeOpts.label == "" {
		treeOpts.label = "export"
	}
	if treeOpts.sourceLabel == "" {
		treeOpts.sourceLabel = "source dir"
	}
	if treeOpts.merge == nil {
		treeOpts.merge = func(entry exportEntry, destBytes, srcBytes []byte) ([]byte, error) {
			return nil, errors.New("merge is not supported for this export")
		}
	}
	strategy := opts.Strategy
	if strategy == "" {
		strategy = ConflictFail
	}
	if !validConflictStrategy(strategy) {
		return nil, fmt.Errorf("invalid conflict strategy %q", strategy)
	}
	if strategy == ConflictInteractive && opts.Prompt == nil {
		return nil, errors.New("interactive conflict strategy requires a prompt callback")
	}

	srcAbs, err := filepath.Abs(srcDir)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", treeOpts.sourceLabel, err)
	}
	dstAbs, err := filepath.Abs(destination)
	if err != nil {
		return nil, fmt.Errorf("resolve destination: %w", err)
	}
	if err := rejectDestinationInsideSource(srcAbs, dstAbs, treeOpts.sourceLabel); err != nil {
		return nil, err
	}

	info, err := os.Stat(srcAbs)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", treeOpts.sourceLabel, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory: %s", treeOpts.sourceLabel, srcAbs)
	}

	entries, err := collectExportEntries(srcAbs, dstAbs)
	if err != nil {
		return nil, labelConflictError(err, treeOpts.label)
	}

	if strategy == ConflictInteractive {
		summary, err := applyInteractive(entries, opts.Prompt, treeOpts.merge)
		return summary, labelConflictError(err, treeOpts.label)
	}

	plans, summary, err := buildPlans(entries, strategy, treeOpts.merge)
	if err != nil {
		return summary, labelConflictError(err, treeOpts.label)
	}
	if err := applyPlans(plans); err != nil {
		return nil, err
	}
	return summary, nil
}

func labelConflictError(err error, label string) error {
	if err == nil {
		return nil
	}
	var conflictErr *ConflictError
	if errors.As(err, &conflictErr) && conflictErr.Label == "" {
		conflictErr.Label = label
	}
	return err
}

func validConflictStrategy(strategy ConflictStrategy) bool {
	switch strategy {
	case ConflictFail, ConflictSkip, ConflictOverwrite, ConflictMerge, ConflictInteractive:
		return true
	default:
		return false
	}
}

func rejectDestinationInsideSource(srcAbs, dstAbs, sourceLabel string) error {
	rel, err := filepath.Rel(srcAbs, dstAbs)
	if err != nil {
		return fmt.Errorf("compare source and destination: %w", err)
	}
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return fmt.Errorf("destination must not be inside %s: %s", sourceLabel, dstAbs)
	}
	return nil
}

func collectExportEntries(srcAbs, dstAbs string) ([]exportEntry, error) {
	var entries []exportEntry
	err := filepath.WalkDir(srcAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			rel, _ := filepath.Rel(srcAbs, path)
			return &ConflictError{Conflicts: []Conflict{{RelPath: filepath.ToSlash(rel), SourcePath: path, Reason: "source symlink is not supported"}}}
		}
		rel, err := filepath.Rel(srcAbs, path)
		if err != nil {
			return fmt.Errorf("rel %s: %w", path, err)
		}
		if rel == "." {
			return nil
		}
		entries = append(entries, exportEntry{
			rel: filepath.ToSlash(rel),
			src: path,
			dst: filepath.Join(dstAbs, rel),
			dir: d.IsDir(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func buildPlans(entries []exportEntry, strategy ConflictStrategy, merge func(exportEntry, []byte, []byte) ([]byte, error)) ([]exportPlan, *Summary, error) {
	summary := &Summary{}
	plans := make([]exportPlan, 0, len(entries))
	var conflicts []Conflict
	for _, entry := range entries {
		plan, conflict, err := planEntry(entry, strategy, merge)
		if err != nil {
			return nil, nil, err
		}
		if conflict != nil {
			conflicts = append(conflicts, *conflict)
			continue
		}
		if plan != nil {
			plans = append(plans, *plan)
			countPlan(summary, plan.op)
		}
	}
	if len(conflicts) > 0 {
		summary.Conflicts = len(conflicts)
		return nil, summary, &ConflictError{Conflicts: conflicts}
	}
	return plans, summary, nil
}

func planEntry(entry exportEntry, strategy ConflictStrategy, merge func(exportEntry, []byte, []byte) ([]byte, error)) (*exportPlan, *Conflict, error) {
	if entry.dir {
		return planDir(entry)
	}

	srcBytes, err := os.ReadFile(entry.src)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", entry.src, err)
	}
	dstInfo, err := os.Lstat(entry.dst)
	if err != nil {
		if os.IsNotExist(err) {
			return &exportPlan{entry: entry, op: opCopy, bytes: srcBytes}, nil, nil
		}
		if errors.Is(err, syscall.ENOTDIR) {
			return nil, conflict(entry, "destination path is blocked by a file"), nil
		}
		return nil, nil, fmt.Errorf("stat destination %s: %w", entry.dst, err)
	}
	if dstInfo.Mode()&os.ModeSymlink != 0 {
		return nil, conflict(entry, "destination symlink is not supported"), nil
	}
	if dstInfo.IsDir() {
		return nil, conflict(entry, "destination is a directory"), nil
	}
	dstBytes, err := os.ReadFile(entry.dst)
	if err != nil {
		return nil, nil, fmt.Errorf("read destination %s: %w", entry.dst, err)
	}
	if bytes.Equal(srcBytes, dstBytes) {
		return &exportPlan{entry: entry, op: opIdentical}, nil, nil
	}

	switch strategy {
	case ConflictFail:
		return nil, conflict(entry, "destination file differs"), nil
	case ConflictSkip:
		return &exportPlan{entry: entry, op: opSkip}, nil, nil
	case ConflictOverwrite:
		return &exportPlan{entry: entry, op: opOverwrite, bytes: srcBytes}, nil, nil
	case ConflictMerge:
		merged, err := merge(entry, dstBytes, srcBytes)
		if err != nil {
			return nil, conflict(entry, err.Error()), nil
		}
		return &exportPlan{entry: entry, op: opMerge, bytes: merged}, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported conflict strategy %q", strategy)
	}
}

func planDir(entry exportEntry) (*exportPlan, *Conflict, error) {
	info, err := os.Lstat(entry.dst)
	if err != nil {
		if os.IsNotExist(err) {
			return &exportPlan{entry: entry, op: opCreateDir}, nil, nil
		}
		return nil, nil, fmt.Errorf("stat destination %s: %w", entry.dst, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, conflict(entry, "destination symlink is not supported"), nil
	}
	if !info.IsDir() {
		return nil, conflict(entry, "destination is a file"), nil
	}
	return nil, nil, nil
}

func applyInteractive(entries []exportEntry, prompt func(Conflict) (ConflictStrategy, error), merge func(exportEntry, []byte, []byte) ([]byte, error)) (*Summary, error) {
	summary := &Summary{}
	for _, entry := range entries {
		strategy := ConflictFail
		for {
			plan, c, err := planEntry(entry, strategy, merge)
			if err != nil {
				return nil, err
			}
			if c == nil {
				if plan != nil {
					if err := applyPlans([]exportPlan{*plan}); err != nil {
						return nil, err
					}
					countPlan(summary, plan.op)
				}
				break
			}
			if strategy != ConflictFail {
				summary.Conflicts++
				return summary, &ConflictError{Conflicts: []Conflict{*c}}
			}

			choice, err := prompt(*c)
			if err != nil {
				return nil, err
			}
			if choice == ConflictInteractive || !validConflictStrategy(choice) {
				return nil, fmt.Errorf("invalid interactive conflict choice %q for %s", choice, entry.rel)
			}
			if choice == ConflictFail {
				summary.Conflicts++
				return summary, &ConflictError{Conflicts: []Conflict{*c}}
			}
			strategy = choice
		}
	}
	return summary, nil
}

func applyPlans(plans []exportPlan) error {
	for _, plan := range plans {
		switch plan.op {
		case opCreateDir:
			if err := os.MkdirAll(plan.entry.dst, 0o755); err != nil {
				return fmt.Errorf("create directory %s: %w", plan.entry.dst, err)
			}
		case opCopy, opOverwrite, opMerge:
			if err := writeFileAtomic(plan.entry.dst, plan.bytes); err != nil {
				return err
			}
		case opSkip, opIdentical:
		}
	}
	return nil
}

func countPlan(summary *Summary, op exportOp) {
	switch op {
	case opCopy:
		summary.Copied++
	case opSkip:
		summary.Skipped++
	case opOverwrite:
		summary.Overwritten++
	case opMerge:
		summary.Merged++
	case opIdentical:
		summary.Identical++
	}
}

func writeFileAtomic(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", filepath.Dir(path), err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".hero-export-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file for %s: %w", path, err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temp file for %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file for %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file for %s: %w", path, err)
	}
	return nil
}

func conflict(entry exportEntry, reason string) *Conflict {
	return &Conflict{RelPath: entry.rel, SourcePath: entry.src, DestPath: entry.dst, Reason: reason}
}

func mergeMarkdown(entry exportEntry, destBytes, srcBytes []byte) ([]byte, error) {
	if !isMarkdownPath(entry.rel) {
		return nil, errors.New("merge supports markdown files only")
	}
	destDoc, err := parseMarkdownDoc(destBytes)
	if err != nil {
		return nil, fmt.Errorf("parse destination markdown: %w", err)
	}
	srcDoc, err := parseMarkdownDoc(srcBytes)
	if err != nil {
		return nil, fmt.Errorf("parse source markdown: %w", err)
	}
	mergedFM, err := mergeFrontmatter(destDoc.frontmatter, srcDoc.frontmatter)
	if err != nil {
		return nil, err
	}
	mergedBody := mergeBodySections(destDoc.body, srcDoc.body)
	return renderMarkdownDoc(mergedFM, mergedBody), nil
}

func isMarkdownPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown"
}

type markdownDoc struct {
	frontmatter map[string]any
	body        string
}

func parseMarkdownDoc(b []byte) (markdownDoc, error) {
	s := string(b)
	if !strings.HasPrefix(s, "---\n") {
		return markdownDoc{frontmatter: map[string]any{}, body: s}, nil
	}
	end := strings.Index(s[4:], "\n---")
	if end < 0 {
		return markdownDoc{}, errors.New("unterminated frontmatter")
	}
	fmRaw := s[4 : 4+end]
	body := s[4+end:]
	if strings.HasPrefix(body, "\n---") {
		body = strings.TrimPrefix(body, "\n---")
	}
	body = strings.TrimPrefix(body, "\n")
	fm := map[string]any{}
	if strings.TrimSpace(fmRaw) != "" {
		if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
			return markdownDoc{}, err
		}
	}
	return markdownDoc{frontmatter: fm, body: body}, nil
}

func mergeFrontmatter(dest, src map[string]any) (map[string]any, error) {
	merged := make(map[string]any, len(dest)+len(src))
	for k, v := range dest {
		merged[k] = normalizeYAMLValue(v)
	}
	for k, srcVal := range src {
		srcVal = normalizeYAMLValue(srcVal)
		destVal, ok := merged[k]
		if !ok {
			merged[k] = srcVal
			continue
		}
		if isUnionListField(k) {
			merged[k] = unionList(destVal, srcVal)
			continue
		}
		if !reflect.DeepEqual(destVal, srcVal) {
			return nil, fmt.Errorf("frontmatter field %q conflicts", k)
		}
	}
	return merged, nil
}

func normalizeYAMLValue(v any) any {
	switch t := v.(type) {
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = normalizeYAMLValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, item := range t {
			out[k] = normalizeYAMLValue(item)
		}
		return out
	default:
		return v
	}
}

func isUnionListField(k string) bool {
	switch k {
	case "tags", "scope", "triggers":
		return true
	default:
		return false
	}
}

func unionList(dest, src any) []any {
	out := []any{}
	seen := map[string]bool{}
	appendOne := func(v any) {
		key := fmt.Sprint(v)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, v)
	}
	for _, v := range asList(dest) {
		appendOne(v)
	}
	for _, v := range asList(src) {
		appendOne(v)
	}
	return out
}

func asList(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case []string:
		out := make([]any, len(t))
		for i, s := range t {
			out[i] = s
		}
		return out
	default:
		return []any{t}
	}
}

func mergeBodySections(dest, src string) string {
	if strings.TrimSpace(dest) == "" {
		return src
	}
	existing := topLevelHeadings(dest)
	result := strings.TrimRight(dest, "\n")
	for _, section := range splitTopLevelSections(src) {
		heading := sectionHeading(section)
		if heading == "" {
			if strings.TrimSpace(section) != "" && !strings.Contains(dest, strings.TrimSpace(section)) {
				result += "\n\n" + strings.TrimSpace(section)
			}
			continue
		}
		if !existing[heading] {
			result += "\n\n" + strings.TrimRight(section, "\n")
			existing[heading] = true
		}
	}
	return result + "\n"
}

func topLevelHeadings(body string) map[string]bool {
	out := map[string]bool{}
	for _, section := range splitTopLevelSections(body) {
		if h := sectionHeading(section); h != "" {
			out[h] = true
		}
	}
	return out
}

func splitTopLevelSections(body string) []string {
	lines := strings.SplitAfter(body, "\n")
	var sections []string
	var current strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") && current.Len() > 0 {
			sections = append(sections, current.String())
			current.Reset()
		}
		current.WriteString(line)
	}
	if current.Len() > 0 {
		sections = append(sections, current.String())
	}
	return sections
}

func sectionHeading(section string) string {
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "## ") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func renderMarkdownDoc(fm map[string]any, body string) []byte {
	var b strings.Builder
	if len(fm) > 0 {
		b.WriteString("---\n")
		for _, k := range sortedFrontmatterKeys(fm) {
			b.WriteString(renderYAMLField(k, fm[k]))
		}
		b.WriteString("---\n\n")
	}
	b.WriteString(strings.TrimLeft(body, "\n"))
	if !strings.HasSuffix(b.String(), "\n") {
		b.WriteString("\n")
	}
	return []byte(b.String())
}

func sortedFrontmatterKeys(fm map[string]any) []string {
	preferred := []string{"title", "slug", "type", "status", "domain", "size", "tags", "scope", "triggers", "created", "source", "raw_path"}
	seen := map[string]bool{}
	keys := make([]string, 0, len(fm))
	for _, k := range preferred {
		if _, ok := fm[k]; ok {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	var rest []string
	for k := range fm {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(keys, rest...)
}

func renderYAMLField(k string, v any) string {
	switch t := v.(type) {
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			parts = append(parts, renderYAMLInline(item))
		}
		return fmt.Sprintf("%s: [%s]\n", k, strings.Join(parts, ", "))
	case []string:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			parts = append(parts, renderYAMLInline(item))
		}
		return fmt.Sprintf("%s: [%s]\n", k, strings.Join(parts, ", "))
	case string:
		return fmt.Sprintf("%s: %s\n", k, renderYAMLInline(t))
	case bool, int, int64, float64:
		return fmt.Sprintf("%s: %v\n", k, t)
	default:
		b, err := yaml.Marshal(map[string]any{k: v})
		if err != nil {
			return fmt.Sprintf("%s: %q\n", k, fmt.Sprint(v))
		}
		return string(b)
	}
}

func renderYAMLInline(v any) string {
	s := fmt.Sprint(v)
	if s == "" || strings.ContainsAny(s, ":#[]{}\n,") || strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
