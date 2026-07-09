---
title: Filesystem Path Conflict Errors
type: context
status: active
tags: [filesystem, export, testing]
created: 2026-06-24
---

## Context

When walking or exporting a directory tree, a destination file that blocks a source directory's child path may surface as `ENOTDIR` from `os.Lstat(childPath)` rather than as a normal file/directory mismatch on the parent path. Treat `ENOTDIR` as a path-specific conflict instead of returning the raw filesystem error.

This came up while delivering `knowledge-export-cli`: the destination had `context/a` as a regular file while the source had `context/a/spec.md`. The exporter now maps that case to a clear conflict so merge/fail behavior stays deterministic and user-readable.
