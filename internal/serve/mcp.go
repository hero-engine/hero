// Package serve provides the hero daemon: MCP server, HTTP API, file watcher, and event stream.
package serve

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/attention/mailquery"
	"github.com/hero-engine/hero/internal/attention/projection"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/refs"
	"github.com/hero-engine/hero/internal/serve/chat"
)

// This file is the slim public surface for the Hero MCP server. The
// implementation is split across:
//
//   mcp_protocol.go      JSON-RPC + MCP protocol types and error codes
//   mcp_lifecycle.go     Run, request routing, initialize, tools/list,
//                        sendResult/sendError/send, logDebug,
//                        finishToolCall result wrapping
//   mcp_dispatch.go      tools/call dispatch — table-driven handler map
//   mcp_tools_def.go     toolDefinitions() — the canonical tool list
//   mcp_tools.go         all tool handler implementations and helpers
//                        (read / mutate / analyze, with shared formatters)
//   mcp_expand.go        hero_expand and ref-store wiring
//   mcp_resolvers.go     ref-store resolvers per kind
//   mcp_summaries.go     compact-mode summary builders
//   envelope.go          [hero envelope] text helper + compact arg

// MCPServer handles the MCP protocol over stdio.
type MCPServer struct {
	heroDir     string
	projectRoot string
	version     string
	graphSchema string // workspace graph schema, read at construction; "" when no graph
	input       io.Reader
	output      io.Writer
	filter      *ToolFilter // optional tool filter; nil = allow all
	profile     string      // active tool profile (set during initialize)
	debugLog    *os.File    // optional debug log file; nil = no logging
	ctx         context.Context
	sendMu      sync.Mutex
	debugMu     sync.Mutex
	requestMu   sync.Mutex
	requests    map[string]context.CancelFunc
	requestWG   sync.WaitGroup

	// Two-tier MCP responses (spec: two-tier-mcp-responses).
	// refsStore is opened lazily on first use; refsRegistry holds
	// resolvers for each ref kind so hero_expand can rehydrate.
	refsStore    *refs.Store
	refsRegistry *refs.Registry
	sessionID    string

	// chatRegistry, when non-nil, receives adapter registrations for
	// clients that declare a hero_dispatch capability on initialize.
	// Optional — leaving it nil disables the dispatch surface and
	// MCP behaves exactly as it did before the chat spec landed.
	chatRegistry *chat.Registry

	// attentionStateRoot and attentionResolver are test seams for private
	// user-state tools. Production resolves the standard global state root.
	attentionStateRoot string
	attentionResolver  focus.ProjectResolver
	attentionService   func() (*projection.Service, error)
	mailQueryService   func() (*mailquery.Service, error)
}

// NewMCPServer creates an MCP server for the given hero workspace.
func NewMCPServer(heroDir, projectRoot, version string) *MCPServer {
	// Read the graph schema without migrating so a stale binary can still
	// report the graph's schema on initialize. Best-effort: leaving it
	// "" (no graph, or unreadable) simply omits the field.
	graphSchema, _ := graph.ReadSchemaVersion(heroDir)
	s := &MCPServer{
		heroDir:      heroDir,
		projectRoot:  projectRoot,
		version:      version,
		graphSchema:  graphSchema,
		input:        os.Stdin,
		output:       os.Stdout,
		ctx:          context.Background(),
		requests:     make(map[string]context.CancelFunc),
		refsRegistry: refs.NewRegistry(),
		sessionID:    refs.SessionID(projectRoot, os.Getpid()),
	}
	s.setupResolvers()
	return s
}

// SetContext supplies the MCP process lifetime context. Foreground provider
// work uses it so process cancellation stops in-flight network requests.
func (s *MCPServer) SetContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.ctx = ctx
}

// NewMCPServerWithFilter creates an MCP server with a tool filter.
func NewMCPServerWithFilter(heroDir, projectRoot, version string, filter *ToolFilter) *MCPServer {
	s := NewMCPServer(heroDir, projectRoot, version)
	s.filter = filter
	return s
}

// SetChatRegistry attaches a chat adapter registry. When set, clients
// that declare a hero_dispatch capability on initialize are
// registered as Hero adapters. Pass nil to disable the integration.
func (s *MCPServer) SetChatRegistry(r *chat.Registry) {
	s.chatRegistry = r
}

// SetIO overrides the default stdin/stdout (for testing).
func (s *MCPServer) SetIO(in io.Reader, out io.Writer) {
	s.input = in
	s.output = out
}
