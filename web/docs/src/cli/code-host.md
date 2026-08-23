# Code-host operations

Hero's optional code-host broker exposes provider-neutral operations without
giving workflow content direct access to credentials.

**Prerequisites:** configure a supported code-host connection, credentials, and
the repository identity. Inspect the current contract with:

```bash
hero code-host contract
```

Machine clients send a versioned broker request on standard input:

```bash
hero code-host broker capabilities < request.json
```

The exact operation registry is derived from
`contracts/codehostbroker/contract.go` and the MCP runtime registry. Narrative
documentation intentionally avoids a mutable operation count.

## Safe action boundary

- Read operations stay bound to the configured repository.
- Write operations require operation-specific semantic consent; configuring a
  credential is not blanket approval.
- The request must resolve provider, repository, operation, and required
  payload fields uniquely.
- Credentials belong in the configured secret source, never the broker payload,
  shell history, committed `hero.json`, or workflow prose.
- A failed or ambiguous write is not retried with modified meaning. Use a stable
  idempotency identity where the operation supports one.

MCP clients should inspect standard tool annotations to distinguish reads from
writes. Tool category alone is not a safety classification.
