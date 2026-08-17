---
title: Hero Code Mail Reads Use Typed Hero Serve HTTP
type: decision
status: accepted
domain: engineering
tags: [mail, hero-code, http, mcp, contracts]
created: 2026-08-16
related:
  - cross-project-mail-read-contract
---

## Decision

Hero Code consumes cross-project Mail list, detail, action, and reply through a versioned Hero Serve HTTP contract. Legacy `hero_mail_list`, `hero_mail_show`, and `hero_mail_action` remain model-facing MCP compatibility surfaces and are not the native desktop application-service boundary.

## Context

Hero permits a 65,536-byte Mail body, while Hero Code's generic MCP result normalization caps text near 50 KB and can truncate otherwise valid JSON before typed decoding. Hero Serve HTTP is already the canonical native transport for Attention, can return typed responses without MCP text normalization, and can share the same registry-backed Mail services. Cross-project requests use `(project_peer_id, message_id)` because Mail storage does not establish message IDs as globally unique.

## Alternatives Considered

Changing the released MCP list/show shapes was rejected as a breaking consumer change. Chunking bodies through MCP was rejected because it would add a second paging protocol to solve a transport mismatch. Reading Mail files from Swift was rejected because it would duplicate storage, identity, receipt, and authorization rules outside Hero.
