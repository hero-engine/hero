// codex_settings.go — Codex CLI host-tool hook installer.
//
// Status: stub for the next-compact-handoff MVP. Codex's
// SessionStartSource::Compact is implemented in the Codex source tree
// but the public hooks documentation page does not yet describe the
// config path or schema as of 2026-05. Rather than guess at a path we
// can't verify, the installer reports a "not yet wired" status and
// callers route Codex requests to a no-op. Claude Code support ships
// first; Codex follows when the docs land or we verify the schema by
// inspection of a working install.
//
// When Codex docs land, replace the stub implementations below with
// the real read/write of (likely) `~/.codex/config.toml` or whatever
// path Codex documents, using the same marker convention (an explicit
// `added_by_hero = true` field on each Hero-installed hook entry).
package hooks

// CodexCompactHandoffSupported reports whether the Hero binary knows
// how to wire the compact-handoff hook on Codex's side. Today: false.
// `hero hooks install --host=codex` should print an informational
// "Codex installer not yet wired — see issue/spec for status" instead
// of failing the broader install.
func CodexCompactHandoffSupported() bool {
	return false
}

// InstallCodexCompactHandoff is a no-op stub. Returns (false, nil) so
// callers report "nothing installed" without failing.
func InstallCodexCompactHandoff(projectRoot string) (bool, error) {
	return false, nil
}

// UninstallCodexCompactHandoff is a no-op stub.
func UninstallCodexCompactHandoff(projectRoot string) (bool, error) {
	return false, nil
}

// CodexCompactHandoffStatus is a no-op stub; always false until the
// Codex installer ships.
func CodexCompactHandoffStatus(projectRoot string) (bool, error) {
	return false, nil
}
