# Mail thread lifecycle v1 handoff

Consume this bundle in addition to Mail-read v1. Hero remains authoritative for
thread identity, lifecycle state, revisions, migration, and advertised actions.
Receipt revisions and thread revisions are distinct and must never be exchanged.
Unknown additive fields and resolution reasons are inert; unknown action IDs are
not executable.
