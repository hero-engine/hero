package gitutil

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hero-engine/hero/internal/graph"
)

// WriteGitLogGraph walks the most recent `limit` commits on the
// current branch and writes them into the graph as Commit nodes.
//
// Each commit emits:
//   - a Commit node keyed by its full SHA
//   - a Person node keyed by the author email, plus
//     Commit -> authored_by -> Person edge
//   - File touches: for each path changed in the commit, if a File
//     node exists in the graph (typically created by codescan), a
//     Commit -> touches -> File edge
//
// Idempotent: re-running on the same commit history adds no history
// rows. limit <= 0 means "default" (200).
func WriteGitLogGraph(repoDir, repoKey string, limit int, store *graph.Store) (*GitGraphSummary, error) {
	if store == nil {
		return nil, fmt.Errorf("gitutil: WriteGitLogGraph requires non-nil Store")
	}
	if !IsRepo(repoDir) {
		return &GitGraphSummary{}, nil // non-git project — nothing to ingest
	}
	if limit <= 0 {
		limit = 200
	}

	source := map[string]any{"kind": "git-log"}
	summary := &GitGraphSummary{}

	// Format: each record begins with a marker line, followed by the
	// file list (one path per line, possibly empty). Using a marker
	// line on its own — rather than \x1e between records — means we
	// can robustly attribute files to the correct commit when scanning
	// line-by-line.
	const sep = "\x1f"
	const marker = "__HERO_COMMIT__"
	format := marker + sep + strings.Join([]string{"%H", "%an", "%ae", "%aI", "%s"}, sep)

	cmd := git(repoDir, "log", fmt.Sprintf("-%d", limit),
		"--pretty=format:"+format, "--name-only")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	// Parse: for each marker line, collect subsequent non-empty,
	// non-marker lines as the files of that commit.
	type record struct {
		sha, authorName, authorEmail, date, subject string
		files                                       []string
	}
	var (
		records []record
		cur     *record
	)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, marker+sep) {
			fields := strings.SplitN(line, sep, 6)
			if len(fields) != 6 {
				cur = nil
				continue
			}
			records = append(records, record{
				sha:         fields[1],
				authorName:  fields[2],
				authorEmail: fields[3],
				date:        fields[4],
				subject:     fields[5],
			})
			cur = &records[len(records)-1]
			continue
		}
		if cur == nil || line == "" {
			continue
		}
		cur.files = append(cur.files, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan git log: %w", err)
	}

	for _, rec := range records {
		sha, authorName, authorEmail, date, subject, files :=
			rec.sha, rec.authorName, rec.authorEmail, rec.date, rec.subject, rec.files
		if sha == "" {
			continue
		}

		// Author -> Person node. Person identity is GLOBAL across repos
		// (lowercase email is the canonical key) — deliberately no Repo
		// stamp so the same person resolves to one node everywhere.
		var authorID int64
		if authorEmail != "" {
			id, err := store.UpsertNode(&graph.Node{
				// Person is in globalNodeTypes — Domain stays empty
				// so the same identity resolves cross-domain.
				Type: "Person",
				Key:  strings.ToLower(authorEmail),
				Props: map[string]any{
					"email": authorEmail,
					"name":  authorName,
				},
				ContentHash: shortHash("person", authorEmail, authorName),
				Source:      source,
			})
			if err != nil {
				return nil, fmt.Errorf("upsert Person %s: %w", authorEmail, err)
			}
			authorID = id
			summary.Persons++
		}

		// Commit node. SHAs are globally unique across repos but each
		// commit belongs to a specific repo, so we stamp Repo here.
		// Commits are intrinsically engineering content per the DSKG
		// write-path rules.
		commitID, err := store.UpsertNode(&graph.Node{
			Type:   "Commit",
			Domain: "engineering",
			Key:    sha,
			Props: map[string]any{
				"sha":          sha,
				"subject":      subject,
				"date":         date,
				"author_name":  authorName,
				"author_email": authorEmail,
				"file_count":   len(files),
			},
			Repo:        repoKey,
			ContentHash: shortHash("commit", sha, subject, date),
			Source:      source,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert Commit %s: %w", sha, err)
		}
		summary.Commits++

		if authorID != 0 {
			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: commitID, ToID: authorID, Type: "authored_by",
				Repo:   repoKey,
				Source: source,
			}); err != nil {
				return nil, fmt.Errorf("upsert authored_by edge: %w", err)
			}
			summary.Edges++
		}

		// touches edges: only emit if the corresponding File node already
		// exists (created by codescan). Avoids creating partial File
		// nodes that would confuse the code subgraph.
		for _, fp := range files {
			fileKey := repoKey + ":" + filepath.ToSlash(fp)
			fileID, err := store.GetNodeID("File", fileKey)
			if err != nil {
				continue
			}
			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: commitID, ToID: fileID, Type: "touches",
				Repo:   repoKey,
				Source: source,
			}); err != nil {
				return nil, fmt.Errorf("upsert touches edge: %w", err)
			}
			summary.Edges++
		}

		// Issue references in the subject line. Each reference becomes
		// (or upserts) an Issue node and an edge from the Commit. The
		// Issue node is intentionally light — Jira ingest (phase 6)
		// fleshes out props later without losing this commit linkage.
		for _, ref := range parseIssueRefs(subject) {
			issueID, err := store.UpsertNode(&graph.Node{
				Type: "Issue",
				// Stub Issue created from a commit subject is
				// engineering-default. Tracker ingest (active-domain)
				// claims authoritative ownership when it lands; first
				// writer wins per DSKG invariant. In a PM workspace
				// the tracker would ingest before scan runs, so the
				// PM stamp arrives first.
				Domain:      "engineering",
				Key:         ref.key,
				Props:       map[string]any{"key": ref.key, "tracker": ref.tracker},
				ContentHash: shortHash("issue-stub", ref.key),
				Source:      map[string]any{"kind": "git-log-ref"},
			})
			if err != nil {
				return nil, fmt.Errorf("upsert Issue %s: %w", ref.key, err)
			}
			summary.Issues++

			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: commitID, ToID: issueID, Type: ref.edgeType,
				Props:  map[string]any{"verb": ref.verb},
				Repo:   repoKey,
				Source: source,
			}); err != nil {
				return nil, fmt.Errorf("upsert %s edge: %w", ref.edgeType, err)
			}
			summary.Edges++
		}
	}

	return summary, nil
}

type issueRef struct {
	key      string // canonical issue id, e.g. "GH#123" or "PROJ-456"
	tracker  string // "github" | "jira"
	verb     string // "fixes" | "closes" | "resolves" | "breaks" | "mentions"
	edgeType string // "fixes" | "closes" | "breaks" | "mentions"
}

// closingVerbRE matches GitHub-style closing keywords immediately
// followed by a reference (whitespace tolerant). Case-insensitive.
var closingVerbRE = regexp.MustCompile(`(?i)\b(fix(?:es|ed)?|close[sd]?|resolve[sd]?|break(?:s|ing)?)\b\s*[: ]?\s*(#\d+|[A-Z][A-Z0-9]+-\d+)`)

// bareRefRE matches a reference that isn't preceded by a closing verb.
var bareRefRE = regexp.MustCompile(`(#\d+|[A-Z][A-Z0-9]+-\d+)`)

// parseIssueRefs extracts issue references from a commit subject.
// Closing-verb references (fixes #1, closes PROJ-2) get the matching
// edge type. Bare references become "mentions". Duplicates collapse
// — the first match for a given key wins.
func parseIssueRefs(subject string) []issueRef {
	seen := map[string]bool{}
	var out []issueRef

	for _, m := range closingVerbRE.FindAllStringSubmatch(subject, -1) {
		key := canonIssueKey(m[2])
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, issueRef{
			key:      key,
			tracker:  trackerFor(key),
			verb:     strings.ToLower(m[1]),
			edgeType: edgeTypeFor(m[1]),
		})
	}
	for _, m := range bareRefRE.FindAllStringSubmatch(subject, -1) {
		key := canonIssueKey(m[1])
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, issueRef{
			key:      key,
			tracker:  trackerFor(key),
			verb:     "mentions",
			edgeType: "mentions",
		})
	}
	return out
}

func canonIssueKey(raw string) string {
	if strings.HasPrefix(raw, "#") {
		return "GH" + raw // "GH#123" — clearly distinct from Jira keys
	}
	return strings.ToUpper(raw)
}

func trackerFor(key string) string {
	if strings.HasPrefix(key, "GH#") {
		return "github"
	}
	return "jira"
}

func edgeTypeFor(verb string) string {
	v := strings.ToLower(verb)
	switch {
	case strings.HasPrefix(v, "fix"):
		return "fixes"
	case strings.HasPrefix(v, "clos"):
		return "closes"
	case strings.HasPrefix(v, "resolv"):
		return "closes"
	case strings.HasPrefix(v, "break"):
		return "breaks"
	default:
		return "mentions"
	}
}

// GitGraphSummary reports counts written by WriteGitLogGraph.
type GitGraphSummary struct {
	Commits int
	Persons int
	Issues  int
	Edges   int
}

func shortHash(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
