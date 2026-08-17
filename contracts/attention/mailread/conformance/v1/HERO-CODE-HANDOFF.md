# Hero Code Mail-read v1 handoff

Hero Code consumes this independently versioned bundle through Hero Serve HTTP.
List responses are bounded metadata only. Detail responses contain the complete
validated Mail envelope and must bypass generic MCP text normalization.

Treat `(project_peer_id, message_id)` as message identity and
`(project_peer_id, thread_id)` as thread scope. Unknown additive fields and raw
identifiers remain inert data; never dispatch an unknown action descriptor.

Vendor this entire directory as one unit and verify every artifact checksum in
`manifest.json` before decoding fixtures. Pin this Mail-read bundle separately
from the existing Attention v1 conformance bundle.
