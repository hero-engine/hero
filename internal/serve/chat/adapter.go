// Package chat implements the hero serve chat dispatcher.
//
// hero serve does not run inference. Every chat turn either resolves
// to a runner-free slash (executed inline here) or dispatches to a
// connected Hero adapter that streams events back. This package owns:
//
//   - the HeroAdapter interface (any process that can pick up a
//     dispatch and run the agent loop)
//   - the Registry of currently connected adapters
//   - the Capability resolver (which adapter handles which kind)
//   - the wire protocol (DispatchRequest + Event)
//   - the runner-free slash handlers (/ask, /note, /scheduled)
//   - SQLite-backed conversation persistence
//   - the HTTP handlers under /api/chat/*
//
// By design this package must NOT import internal/runner/*. A
// build-time test enforces the boundary.
package chat

import (
	"context"
	"time"
)

// Kind names a dispatch class. Adapters declare which kinds they
// support; the resolver picks one per dispatch.
type Kind string

const (
	// KindInteractive is a user-driven chat turn that expects
	// streamed tokens back into a UI.
	KindInteractive Kind = "interactive"
	// KindHeadless is a scheduled or automation-driven run that may
	// fire when no human is watching.
	KindHeadless Kind = "headless"
)

// HeroAdapter is any process that can pick up a dispatch from hero
// serve and run the agent loop. Implemented by hero-code (canonical)
// and any future in-IDE bridge (claude-code-bridge, cursor-bridge,
// codex-bridge, ...).
//
// Stream is invoked synchronously by the dispatcher and returns a
// channel of events. The adapter MUST close the channel after
// emitting a chat.done event (or a terminal chat.error). The ctx is
// cancelled when the client disconnects or the turn is explicitly
// aborted; adapters MUST respect cancellation and stop emitting.
type HeroAdapter interface {
	// Name is the adapter type identifier ("hero-code",
	// "claude-code-bridge", ...). Used for resolver tiebreaks and
	// display.
	Name() string
	// Version is free-form; surfaced in capability JSON.
	Version() string
	// Kinds reports which dispatch kinds this adapter accepts.
	Kinds() []Kind
	// Stream dispatches a turn and returns a channel of events.
	Stream(ctx context.Context, req DispatchRequest) (<-chan Event, error)
	// Close releases adapter resources. Called by the registry on
	// deregister.
	Close() error
}

// AdapterInfo is the public, JSON-serializable view of a connected
// adapter, returned by the capability endpoint.
type AdapterInfo struct {
	ID          string    `json:"id"`
	Adapter     string    `json:"adapter"`
	Version     string    `json:"version"`
	Kinds       []Kind    `json:"kinds"`
	ConnectedAt time.Time `json:"connected_at"`
	LastSeen    time.Time `json:"last_seen"`
}
