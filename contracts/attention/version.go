// Package attention defines the portable v1 contracts for Hero's durable
// attention surfaces. It deliberately contains DTOs only and remains a leaf.
package attention

const SchemaVersion = 1

// V1 compatibility is additive: consumers must ignore unknown object fields
// and preserve raw string values such as source kinds and action IDs. Removing
// or renaming fields, narrowing accepted values, or changing field meaning
// requires a new attention schema version.
