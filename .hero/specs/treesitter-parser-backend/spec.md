---
title: Tree-sitter CLI Parser Backend
slug: treesitter-parser-backend
type: feature
status: completed
priority: low
tags: [code-intelligence, codescan, treesitter, parsing]
created: 2026-04-18
horizon: now
---

## Goal

Wire up the tree-sitter CLI as an actual parsing backend for codescan. The `code_scan.parser: "treesitter"` config and CLI auto-detection (`internal/cli/scan.go`) already exist, but the parser itself falls through to heuristic parsing. Implementing this provides higher accuracy symbol extraction than regex-based heuristics.

## Problem

Heuristic regex parsers work well for common patterns but break on edge cases: multi-line signatures, nested generics, unusual formatting, macro-generated code. Tree-sitter provides a proper AST, eliminating these issues. The infrastructure to select tree-sitter is already in place — only the actual parsing integration is missing.

## Design

### Architecture

Add a `parse_treesitter.go` (or similar) in `internal/codescan/` that:

1. Invokes the `tree-sitter` CLI to parse a source file into an S-expression or JSON AST.
2. Walks the AST to extract the same symbol types the heuristic parsers produce (functions, types, methods, constants).
3. Returns `[]Symbol` matching the existing `types.go` interface.

### CLI Invocation

The tree-sitter CLI supports `tree-sitter parse <file>` which outputs an S-expression AST. It also supports `tree-sitter dump-languages` to list installed grammars. The parser should:

- Check which grammars are installed for the target language.
- Fall back to heuristic parsing for languages without an installed grammar.
- Parse the S-expression output to extract symbol nodes.

### Language Mapping

Map codescan's language detection to tree-sitter grammar names:

| Codescan language | Tree-sitter grammar |
|---|---|
| go | go |
| javascript | javascript |
| typescript | typescript |
| python | python |
| ruby | ruby |
| rust | rust |
| c, cpp | c, cpp |
| java | java |

### Integration

The `scanner.go` dispatch logic already selects a parser. Add a branch that routes to the tree-sitter parser when the resolved parser mode is `"treesitter"`. The tree-sitter parser should implement the same interface as the heuristic parsers.

## Boundaries

- Requires the `tree-sitter` CLI and language grammars to be installed by the user. Hero does not install them.
- If a grammar isn't available for a language, fall back to heuristic parsing per-language (not all-or-nothing).
- V1 parses one file at a time via CLI invocation. Batch parsing or library embedding (via CGo or WASM) is future optimization.
- S-expression parsing can be done with a simple recursive descent parser — no need for a full S-expression library.
- Performance will be slower than heuristic parsing due to CLI overhead. This is acceptable for scan-time use.
