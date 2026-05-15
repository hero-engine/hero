// Package peering defines the wire and on-disk shapes shared between
// hero workspaces that peer with each other — peer manifests, handoff
// records, peer call envelopes, and peer-related event payloads.
//
// Like its parent contracts package, peering is a leaf: nothing under
// contracts/peering/... may import from anywhere else inside this
// repository. The CLI and a future cloud server both consume identical
// types from here.
package peering

// PeeringContractsVersion is the current shape version of the peering
// contracts. It evolves INDEPENDENTLY of the main ContractsVersion in
// contracts/version.go.
//
// Bump rule: every breaking change to any exported type, field, or
// method signature under contracts/peering/... increments this constant
// by one. Adding a new field with a zero-value-safe default is not a
// breaking change. Removing or renaming a field is.
//
// The independence is deliberate — peering shapes evolve on their own
// cadence (handoff trail format, peer manifest schema) without
// requiring a main-contracts bump, and main-contracts bumps don't
// force peer manifests to be regenerated.
const PeeringContractsVersion = 1
