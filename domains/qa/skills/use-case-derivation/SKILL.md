---
name: use-case-derivation
description: Turn a user flow into happy-path, alternative-path, exception, permission, and recovery test conditions.
---
# Use-case derivation

Name actor, goal, preconditions, trigger, main flow, postcondition, and business
invariant. Derive the main success case, then one case per meaningful alternate or
exception flow. Add permission, interruption, retry, cancellation, and partial-
completion cases when the flow supports them. Keep UI navigation out of the test
purpose unless navigation itself is the requirement. Trace every derived case to
the flow step or exception that motivated it.

