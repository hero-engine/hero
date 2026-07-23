---
title: "Attention Semantic Consent Is Separate from MCP Approval"
type: decision
status: accepted
tags: [attention, mcp, consent, permissions]
created: 2026-07-23
relations:
  - target: attention-interaction-consent-contract
    kind: decided-in
---

## Decision

Hero publishes a versioned Attention operation policy for semantic consent;
standard MCP tool annotations remain risk hints, and a harness or client keeps
authority over its configured execution approval.

## Context

Conversational intent answers whether the model may select an Attention
operation: explicit user imperative, explicit acceptance, clarification, or no
consent. MCP annotations answer different questions such as read-only,
destructive, idempotent, and open-world behavior, and the protocol defines them
as hints rather than trusted authorization. Combining the two would either let
a tool hint bypass user approval or force every client to reverse-engineer
consent from tool names.

## Alternatives Considered

Using MCP annotations as the consent contract was rejected because they cannot
prove where an imperative originated and cannot safely authorize content from
Mail. Relying only on client approval was rejected because six harnesses and
Hero Code would classify the same Attention intent differently. A production
string-matching classifier was rejected because explicitness depends on
resolved conversational context, not phrases alone.
