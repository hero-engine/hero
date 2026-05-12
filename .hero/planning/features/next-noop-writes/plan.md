# Delivery Plan

1. Add no-op write helpers in `internal/cli/checkpoint.go`: byte-identical writes skip disk updates, and projected tracked files compare with frontmatter `updated:` normalized.
2. Replace unconditional writes in legacy NEXT, projected project NEXT, user handoff, and local state with the helpers.
3. Add focused CLI tests for byte no-op writes, `updated:`-only projection no-ops, semantic projection changes, and quiet checkpoint behavior.
4. Run `go test ./internal/cli ./internal/projection` and update the spec ACs with evidence.
