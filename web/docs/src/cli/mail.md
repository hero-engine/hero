# Project Mail

Project Mail is a private, local-first mailbox for asynchronous messages between
configured Hero projects. Delivery uses the recipient project's peer manifest
for identity, then writes only to the current user's global attention state. It
does not invoke an agent, require Hero Serve, or modify either project checkout.

```bash
printf 'Please review the contract.\n' | \
  hero mail send app --subject "Contract review" --body-file - --kind request

hero mail inbox
hero mail inbox --unread --json
hero mail show mail_abcd
hero mail show mail_abcd --no-mark-read --json
printf 'Reviewed; looks good.\n' | hero mail reply mail_abcd --body-file -
hero mail ack mail_abcd --note "I will handle this"
```

Bodies are accepted only through `--body-file path` or `--body-file -` for
standard input; they are never positional arguments. `send` accepts an optional
`--idempotency-key` and `--message-id` for safe retries. Reusing either identity
with different normalized content returns `idempotency_conflict` without
overwriting the original message.

`show` marks a message read unless `--no-mark-read` is present. `ack` records
the first acknowledgement timestamp and optional note idempotently. Read and
acknowledgement state is stored separately from the immutable envelope.

All commands support `--json`. Successful output uses the versioned attention
contract shapes; failures emit a JSON object with stable `code` and `message`
fields and retain a non-zero command result.

Shell completion is generated dynamically by Cobra from the registered command
tree. There are no checked-in completion snapshot files; adding the `mail`
command and its five registered children automatically exposes them to the
generated Bash, Zsh, Fish, and PowerShell completion paths.
