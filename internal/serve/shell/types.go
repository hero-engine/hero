package shell

import (
	"html/template"
	"io"
	"net/http"
)

// Home describes a top-nav home page registered with the router.
//
// The router uses Home to build the top-nav tab list, mount the home's
// root and child routes, and (optionally) fetch a badge count.
type Home struct {
	// Slug uniquely identifies the home (e.g. "now", "work"). Also
	// used as the URL path segment.
	Slug string
	// Label is the human-facing tab text.
	Label string
	// Href is the URL the tab links to (e.g. "/now").
	Href string
	// Count is an optional badge fetcher. Invoked with a 50ms
	// deadline; on timeout the tab renders unbadged.
	Count func(*http.Request) (int, bool)
	// Render handles the home root URL (Href).
	Render http.HandlerFunc
	// Items lists per-item child routes under this home (e.g.
	// "GET /work/spec/{slug}"). They share the home's edition gate.
	Items []ItemRoute
	// Editions lists the edition slugs this home is allowed in. Empty
	// means "all editions".
	Editions []string
}

// ItemRoute is a per-item child route under a home.
type ItemRoute struct {
	// Pattern is a Go 1.22+ http.ServeMux pattern, e.g.
	// "GET /work/spec/{slug}".
	Pattern string
	// Render handles the request.
	Render http.HandlerFunc
}

// Page is the per-render envelope a home passes to RenderPage.
type Page struct {
	// ActiveHome matches the Slug of the home rendering this page;
	// used to highlight the corresponding top-nav tab.
	ActiveHome string
	// PageTitle is the HTML <title>.
	PageTitle string
	// SubNav, if non-nil, renders a sub-nav row below the top nav.
	SubNav *SubNav
	// Content writes the body of the page into w.
	Content func(io.Writer) error
	// HeadExtra is appended inside <head>. Use for home-specific
	// <link>/<script> tags.
	HeadExtra template.HTML
}

// ----- Top-nav chrome (consumed by top-nav.html) -----------------------

// Chrome is the data the top-nav template needs to render. Built by
// the router on every request.
type Chrome struct {
	Workspace    string
	Branch       string
	UserName     string
	UserInitials string
	Tabs         []ChromeTab
}

// ChromeTab is one top-nav tab.
type ChromeTab struct {
	Slug     string
	Label    string
	Href     string
	Active   bool
	HasCount bool
	Count    int
}

// ----- Footer (consumed by footer.html) --------------------------------

// Footer is the data the footer template needs.
type Footer struct {
	Workspace string
	Version   string
	Edition   string
}

// ----- Page hero (consumed by page-hero.html) --------------------------

// PageHero is the eyebrow + title + subhead + action row at the top of
// most home pages.
type PageHero struct {
	Eyebrow template.HTML
	Title   string
	Subhead template.HTML
	Actions []PageHeroAction
}

// PageHeroAction is a single action button or chip in a page hero.
//
// Kind: "primary" | "ghost" | "chip"
type PageHeroAction struct {
	Kind  string
	Label string
	Href  string
	Icon  template.HTML
	Chip  string
}

// ----- Tabbed metric strip (consumed by tabbed-metric-strip.html) ------

// MetricStrip is a row of text-link tabs above a swappable tile row.
type MetricStrip struct {
	Tabs    []MetricTab
	AllLink string
}

// MetricTab is one tab in the strip.
type MetricTab struct {
	Slug   string
	Label  string
	Active bool
	Tiles  []MetricTile
}

// MetricTile is one tile in a tab's pane.
type MetricTile struct {
	Value  template.HTML
	Label  string
	Footer template.HTML
	Accent string // "" | "warn"
}

// ----- Sub-nav (consumed by sub-nav.html) ------------------------------

// SubNav is an optional second row of text-link tabs.
type SubNav struct {
	Tabs []SubNavTab
}

// SubNavTab is one tab in a sub-nav row.
//
// Variant: "" | "amber" | "locked"
type SubNavTab struct {
	Label    string
	Href     string
	Active   bool
	Badge    string
	Variant  string
	LockMeta string
}

// ----- Chat input (consumed by chat-input.html) ------------------------

// ChatInput parameterizes the chat-input fragment.
//
// Variant: "hero" | "overlay" | "inline"
type ChatInput struct {
	Variant     string
	Placeholder string
	Context     []ChatContextChip
}

// ChatContextChip is one chip rendered below the chat input.
type ChatContextChip struct {
	Kind  string
	Label string
}

// ----- Empty-state notice (consumed by empty-state-notice.html) --------

// EmptyState parameterizes the soft notice block used as a no-data CTA.
type EmptyState struct {
	Headline      string
	Body          template.HTML
	PrimaryAction EmptyStateAction
	GhostAction   EmptyStateAction
	FootNote      string
}

// EmptyStateAction is a label+href button used by EmptyState.
type EmptyStateAction struct {
	Label string
	Href  string
}
