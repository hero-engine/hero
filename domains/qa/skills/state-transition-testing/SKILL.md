---
name: state-transition-testing
description: Derive valid, invalid, repeated, guarded, and terminal-state tests from a state model.
---
# State transition testing

Inventory states, events, guards, side effects, and terminal behavior. Cover each
valid transition, important invalid transitions, self-loops, repeated events,
guard boundaries, and recovery after failure. For history-sensitive systems, add
sequences that reach the same state through different paths. Assert both the new
state and observable side effects. Treat an undocumented transition as a
requirement question, not permission to invent behavior.

