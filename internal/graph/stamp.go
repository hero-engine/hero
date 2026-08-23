package graph

import (
	"fmt"

	"github.com/hero-engine/hero/internal/config"
)

// NodeHint classifies a node's relationship to the domain partition
// so write-side ingest sites can call DomainFor with a fixed constant
// rather than reasoning about intrinsic-vs-active per call.
//
// The three values map to the three stamping rules in
// `domain-scoped-knowledge-graph/spec.md`:
//
//   - IntrinsicCode  → always engineering (codescan / git ingest)
//   - IntrinsicGlobal → empty string (Mission / Person — shared across domains)
//   - IntrinsicActive → workspace's primary domain from Config.ResolveDomains
//
// Callers pick the hint at the ingest-package level; the hint is a
// package constant, not a per-call decision.
type NodeHint int

const (
	// IntrinsicActive stamps from the resolved primary domain (with "engineering" fallback).
	// Most ingest paths use this — specs, tracker issues, sessions, memory,
	// next-doc, knowledge, tasks.
	IntrinsicActive NodeHint = iota
	// IntrinsicCode stamps "engineering" regardless of the active domain.
	// Code symbols and git commits are intrinsically engineering content.
	IntrinsicCode
	// IntrinsicGlobal stamps "" — the global allow-list domain. Used by
	// Mission (workspace-wide brief) and Person (shared across domains).
	IntrinsicGlobal
)

// DomainFor returns the domain string for an ingest write, given the
// workspace config and the ingest package's intrinsic hint.
//
//   - IntrinsicCode  → "engineering"
//   - IntrinsicGlobal → ""
//   - IntrinsicActive → the resolved primary domain, falling back to "engineering" when
//     the workspace has no domain configured (pre-migration workspaces
//     and engineering-default setups)
//
// The "engineering" fallback for IntrinsicActive matches the v3
// schema's column default and the ResolveDomain precedence chain
// documented in the DSKG spec.
func DomainFor(cfg config.Config, hint NodeHint) string {
	switch hint {
	case IntrinsicCode:
		return "engineering"
	case IntrinsicGlobal:
		return ""
	case IntrinsicActive:
		resolved, err := cfg.ResolveDomains()
		if err == nil {
			return string(resolved.Primary)
		}
		return "engineering"
	default:
		return "engineering"
	}
}

// DomainForHandler returns the durable artifact owner for a routed pack
// handler. Ownership is independent of workspace primary and UI focus, but the
// owner must participate in the enabled stack so stale handlers cannot write.
func DomainForHandler(cfg config.Config, owner string) (string, error) {
	resolved, err := cfg.ResolveDomains()
	if err != nil {
		return "", err
	}
	for _, enabled := range resolved.Stack() {
		if string(enabled) == owner {
			return owner, nil
		}
	}
	return "", fmt.Errorf("handler owner %q is not enabled in this workspace", owner)
}
