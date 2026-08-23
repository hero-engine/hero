# Mail thread lifecycle v1 handoff

Consume this bundle in addition to Mail-read v1. Hero remains authoritative for
thread identity, lifecycle state, revisions, migration, and advertised actions.
Receipt revisions and thread revisions are distinct and must never be exchanged.
Unknown additive fields and resolution reasons are inert; unknown action IDs are
not executable.

Use the thread list/detail projection for Needs Attention, Updates, History,
badge counts, pagination, and lifecycle actions. Treat `(project_peer_id,
thread_id)` as the only row identity. `actionable`, `unread`, `lifecycle`, and
`bucket` are independent authoritative fields; do not regroup messages or infer
classification in Hero Code. The existing Mail-read v1 message routes remain
the compatibility path for pinned clients.
