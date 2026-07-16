package serve

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/serve/chat"
)

// Run starts the MCP server, reading from input and writing to output.
// It blocks until the input is closed.
func (s *MCPServer) Run() error {
	// Enable debug logging if HERO_MCP_DEBUG is set
	if os.Getenv("HERO_MCP_DEBUG") != "" {
		logPath := filepath.Join(s.heroDir, "mcp-debug.log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "hero mcp: cannot open debug log %s: %v\n", logPath, err)
		} else {
			s.debugLog = f
			defer f.Close()
			s.logDebug("=== MCP session started (hero %s) ===", s.version)
		}
	}

	// Parent-liveness backstop. Stdin EOF is the fast, correct shutdown
	// path, but the client controls EOF and may die without closing the
	// pipe (crash, SIGKILL, fd handed to a survivor), leaving us blocked
	// in Scan() forever and reparented to launchd/init. The watchdog
	// notices the reparent and exits. Gated to real stdio mode so tests
	// that drive Run() with a bytes.Buffer never start it. Stopped when
	// Run() returns via the done channel.
	//
	// The singleton lock is an additional, orthogonal guard: it reaps a
	// redundant, still-connected daemon left behind when a live client
	// reconnects — the live-duplicate case the orphan watchdog cannot
	// cover. Same real-stdio gate so unit tests never touch the pidfile.
	if s.input == os.Stdin {
		ppid := os.Getppid()
		var release func()
		if r, err := acquireMCPSingleton(mcpPIDFilePath(s.heroDir, ppid), os.Getpid(), ppid); err != nil {
			// Non-fatal: a pidfile problem must never stop us serving.
			// Degrade to the pre-fix behavior (no dedup) rather than refuse.
			fmt.Fprintf(os.Stderr, "hero mcp: singleton lock: %v\n", err)
		} else {
			release = r
			defer release()
		}

		// A superseding daemon or the OS sends SIGTERM (SIGINT under a
		// terminal). The default disposition is immediate death, which
		// skips the deferred release() — leaking .hero/mcp-<ppid>.pid and,
		// worse, abandoning any in-flight index transaction as a hot
		// journal/WAL lock. Install a handler that runs release() before
		// exiting so shutdown is clean on every death path.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			<-sigCh
			if release != nil {
				release()
			}
			os.Exit(0)
		}()

		done := make(chan struct{})
		defer close(done)
		// The goroutine dies with the process; the join channel is only
		// needed by tests that restore the watchdog's seam vars.
		_ = startParentWatchdog(done)
	}

	scanner := bufio.NewScanner(s.input)
	// Allow large messages (1MB)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.logDebug("→ PARSE ERROR: %s", line)
			s.sendError(nil, ErrCodeParse, "Parse error")
			continue
		}

		s.logDebug("→ %s (id=%s)", req.Method, string(req.ID))
		if req.Params != nil {
			s.logDebug("  params: %s", string(req.Params))
		}

		s.handleRequest(&req)
	}

	return scanner.Err()
}

func (s *MCPServer) handleRequest(req *JSONRPCRequest) {
	// Defense in depth: a panic in any tool handler would otherwise
	// unwind through Run()'s scan loop and crash the whole process —
	// a "transport closed" for every tool, not just the failing call.
	// Recover turns it into one JSON-RPC error and keeps serving. No
	// panic trigger is known today; this is cheap insurance.
	defer func() {
		if r := recover(); r != nil {
			s.logDebug("PANIC in %s: %v", req.Method, r)
			s.sendError(req.ID, ErrCodeInternal, fmt.Sprintf("internal error: %v", r))
		}
	}()
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "notifications/initialized":
		// Notification — no response needed
	case "prompts/list":
		// Hero doesn't support prompts; return empty list
		s.sendResult(req.ID, map[string]interface{}{"prompts": []interface{}{}})
	case "resources/list":
		// Hero doesn't support resources; return empty list
		s.sendResult(req.ID, map[string]interface{}{"resources": []interface{}{}})
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	case "ping":
		s.sendResult(req.ID, map[string]interface{}{})
	default:
		s.sendError(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *MCPServer) handleInitialize(req *JSONRPCRequest) {
	// Extract optional hero_profile from client meta (non-standard extension)
	if req.Params != nil {
		var params struct {
			Meta map[string]interface{} `json:"_meta"`
		}
		if err := json.Unmarshal(req.Params, &params); err == nil {
			if profile, ok := params.Meta["hero_profile"].(string); ok {
				s.profile = profile
			}
		}
	}

	// Parse the client's capabilities; if hero_dispatch is declared
	// and a chat registry is attached, register the client as an
	// adapter for the lifetime of this MCP session.
	s.tryRegisterDispatchClient(req)

	// Build dynamic instructions via auto-prime if configured
	instructions := "Hero provides spec-driven AI engineering workflow tools. Use these tools to query project knowledge, search specs, check workspace health, and get context-aware nudges."

	cfg, cfgErr := config.Load(s.projectRoot)
	if cfgErr == nil && cfg.Prime.AutoEnabled() {
		if primeCtx, err := BuildPrimeContext(s.heroDir, s.projectRoot, cfg.Prime.KnowledgeEnabled()); err == nil && primeCtx != "" {
			instructions = primeCtx
		}
	}

	result := InitializeResult{
		ProtocolVersion: "2025-03-26",
		Capabilities: MCPCapabilities{
			Tools: &MCPToolsCapability{},
		},
		ServerInfo: MCPServerInfo{
			Name:        "hero",
			Version:     s.version,
			Schema:      graph.CompiledSchemaVersion(),
			GraphSchema: s.graphSchema,
		},
		Instructions: instructions,
	}
	s.sendResult(req.ID, result)
}

func (s *MCPServer) handleToolsList(req *JSONRPCRequest) {
	tools := s.toolDefinitions()
	if s.filter != nil {
		tools = s.filter.FilterTools(tools, s.profile)
	}
	result := ToolsListResult{
		Tools: tools,
	}
	s.sendResult(req.ID, result)
}

// finishToolCall wraps a tool's (string, error) result in the
// appropriate ToolCallResult shape and emits it to the client.
// Consolidated here so dispatch sites stay tiny.
func (s *MCPServer) finishToolCall(reqID json.RawMessage, result string, toolErr error) {
	if toolErr != nil {
		s.sendResult(reqID, ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error: %v", toolErr)}},
			IsError: true,
		})
		return
	}
	s.sendResult(reqID, ToolCallResult{
		Content: []ToolContent{{Type: "text", Text: result}},
	})
}

func (s *MCPServer) sendResult(id json.RawMessage, result interface{}) {
	s.send(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (s *MCPServer) sendError(id json.RawMessage, code int, message string) {
	s.send(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &JSONRPCError{Code: code, Message: message},
	})
}

func (s *MCPServer) send(resp JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hero mcp: marshal error: %v\n", err)
		return
	}
	s.logDebug("← %s", string(data))
	fmt.Fprintf(s.output, "%s\n", data)
}

// tryRegisterDispatchClient parses the client's hero_dispatch
// capability (if any) and registers an mcpClientAdapter wrapper in
// the chat registry. The actual server-initiated dispatch path is
// stubbed today — Stream returns a "not yet wired" error until
// hero-code's adapter side ships. This preserves the chat registry as
// the source of truth for which adapters are present.
func (s *MCPServer) tryRegisterDispatchClient(req *JSONRPCRequest) {
	if s.chatRegistry == nil || req.Params == nil {
		return
	}
	var params struct {
		Capabilities MCPCapabilities `json:"capabilities"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return
	}
	if params.Capabilities.HeroDispatch == nil {
		return
	}
	declared := params.Capabilities.HeroDispatch
	if declared.Adapter == "" {
		return
	}
	kinds := make([]chat.Kind, 0, len(declared.Kinds))
	for _, k := range declared.Kinds {
		kinds = append(kinds, chat.Kind(k))
	}
	id := declared.SessionID
	if id == "" {
		id = s.sessionID
	}
	adapter := &mcpClientAdapter{
		name:    declared.Adapter,
		version: declared.Version,
		kinds:   kinds,
	}
	if err := s.chatRegistry.Register(id, adapter); err != nil {
		s.logDebug("chat: register adapter %s: %v", id, err)
	}
}

// mcpClientAdapter is a placeholder adapter for MCP clients that
// declared hero_dispatch on initialize. The server-initiated dispatch
// path that actually calls back into the client's hero_chat tool is
// still being built on hero-code's side (see the cross-repo peering
// trail on hero-chat-and-model). Until that lands, this adapter
// shows up in capability listings but Stream returns an error.
type mcpClientAdapter struct {
	name    string
	version string
	kinds   []chat.Kind
}

func (a *mcpClientAdapter) Name() string       { return a.name }
func (a *mcpClientAdapter) Version() string    { return a.version }
func (a *mcpClientAdapter) Kinds() []chat.Kind { return a.kinds }
func (a *mcpClientAdapter) Close() error       { return nil }
func (a *mcpClientAdapter) Stream(ctx context.Context, req chat.DispatchRequest) (<-chan chat.Event, error) {
	out := make(chan chat.Event, 2)
	go func() {
		defer close(out)
		out <- chat.ErrorEvent(
			"adapter_not_wired",
			fmt.Sprintf("adapter %q registered but server-initiated dispatch is not yet implemented", a.name),
			"",
		)
		out <- chat.DoneEvent(0, nil)
	}()
	return out, nil
}

// logDebug writes a timestamped line to the debug log if enabled.
func (s *MCPServer) logDebug(format string, args ...interface{}) {
	if s.debugLog == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(s.debugLog, "%s  %s\n", time.Now().Format("15:04:05.000"), msg)
}
