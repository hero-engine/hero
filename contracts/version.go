// Package contracts defines the wire and graph shapes shared between the
// hero CLI and the hero-cloud server. It is a leaf package: nothing under
// contracts/... may import from anywhere else inside this repository.
// External consumers (most notably the hero-cloud repo) depend only on
// this package.
package contracts

// ContractsVersion is the current shape version of this contracts package.
//
// Bump rule: every breaking change to any exported type, field, or method
// signature under contracts/... increments this constant by one. Adding a
// new field with a zero-value-safe default is not a breaking change.
// Removing or renaming a field is. Changing a wire-level meaning (even
// without a structural change) is.
//
// CLI clients embed ContractsVersion in every event envelope they send.
// Servers compare against ServerMinContractsVersion and reject anything
// below it with a structured upgrade-required error.
const ContractsVersion = 1

// ServerMinContractsVersion is the minimum client ContractsVersion the
// server will accept on the wire.
//
// Bump rule: servers raise ServerMinContractsVersion deliberately when
// they drop support for an older client shape. This is independent of
// ContractsVersion — the server may speak version N while still
// accepting clients at version N-2. Raising this constant is a breaking
// change for any client running below it.
const ServerMinContractsVersion = 1
