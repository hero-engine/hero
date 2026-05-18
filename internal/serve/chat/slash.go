package chat

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// SlashHandler is the signature every runner-free slash implements.
// Handlers MUST emit events to out and MUST close-by-convention by
// emitting a chat.done event (or a terminal chat.error). Handlers are
// invoked from a goroutine the caller owns; ctx cancellation should
// be honored.
type SlashHandler func(ctx context.Context, req DispatchRequest, out chan<- Event) error

// Slash describes one registered slash command.
type Slash struct {
	// Name is the slash identifier without the leading "/".
	Name string
	// Help is a one-line description for the palette.
	Help string
	// RunnerFree is true when the slash executes inside hero serve
	// directly (no adapter needed). RunnerFree slashes MUST set
	// Handler; adapter-dispatched slashes leave Handler nil.
	RunnerFree bool
	// Handler is the synchronous in-process implementation, set only
	// when RunnerFree is true.
	Handler SlashHandler
}

// slashes is the global slash registry. Pack-prefixed slashes (e.g.
// `pm:design-story`) are added via Register at startup.
var (
	slashesMu sync.RWMutex
	slashes   = defaultSlashes()
)

func defaultSlashes() map[string]Slash {
	return map[string]Slash{
		"ask":       {Name: "ask", RunnerFree: true, Handler: askHandler, Help: "Read-only Q&A over indexed corpus"},
		"note":      {Name: "note", RunnerFree: true, Handler: noteHandler, Help: "Capture a note to .hero/knowledge/notes/"},
		"scheduled": {Name: "scheduled", RunnerFree: true, Handler: scheduledHandler, Help: "Convert input into a scheduled-agent YAML"},
		"design":    {Name: "design", Help: "Design a feature or change"},
		"deliver":   {Name: "deliver", Help: "Deliver a spec"},
		"diagnose":  {Name: "diagnose", Help: "Diagnose a bug"},
	}
}

// Lookup returns the slash registered under name, if any.
func Lookup(name string) (Slash, bool) {
	slashesMu.RLock()
	defer slashesMu.RUnlock()
	s, ok := slashes[name]
	return s, ok
}

// All returns every registered slash, sorted by name.
func All() []Slash {
	slashesMu.RLock()
	defer slashesMu.RUnlock()
	out := make([]Slash, 0, len(slashes))
	for _, s := range slashes {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Register adds a new slash to the registry. Returns an error if the
// name is empty, already registered, or marked RunnerFree without a
// Handler. Use for pack-prefixed slashes only; collisions are a
// startup-time bug.
func Register(s Slash) error {
	if s.Name == "" {
		return fmt.Errorf("slash: name required")
	}
	if s.RunnerFree && s.Handler == nil {
		return fmt.Errorf("slash %q: runner-free slash needs Handler", s.Name)
	}
	slashesMu.Lock()
	defer slashesMu.Unlock()
	if _, ok := slashes[s.Name]; ok {
		return fmt.Errorf("slash %q: already registered", s.Name)
	}
	slashes[s.Name] = s
	return nil
}

// resetSlashesForTest restores the default slash registry. Test-only.
func resetSlashesForTest() {
	slashesMu.Lock()
	defer slashesMu.Unlock()
	slashes = defaultSlashes()
}

// ParseSlash extracts a slash invocation from the prompt. Returns
// (nil, prompt) when the prompt does not begin with /<word>.
//
// Recognized shape: "/<name>[ <args...>]" — name is alphanumeric +
// "-" + "_" + ":" (for pack prefixes like "pm:design-story").
func ParseSlash(prompt string) (*SlashInvoc, string) {
	trimmed := strings.TrimLeft(prompt, " \t")
	if !strings.HasPrefix(trimmed, "/") {
		return nil, prompt
	}
	rest := trimmed[1:]
	var name string
	for i, r := range rest {
		if isSlashNameChar(r) {
			continue
		}
		name = rest[:i]
		rest = rest[i:]
		break
	}
	if name == "" && rest != "" && !strings.ContainsAny(rest, " \t") {
		// whole remainder is a name (no args)
		name = rest
		rest = ""
	}
	if name == "" {
		return nil, prompt
	}
	args := strings.TrimSpace(rest)
	return &SlashInvoc{Name: name, Args: args}, prompt
}

func isSlashNameChar(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	case r == '-' || r == '_' || r == ':':
		return true
	}
	return false
}
