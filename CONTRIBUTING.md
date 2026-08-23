# Contributing to Hero

Thank you for helping improve Hero. This repository is still private and has
no root open-source license. External contributions are not accepted until the
Apache-2.0 license and public-visibility gates have both completed. Preparing
these instructions does not grant permission to copy, modify, or redistribute
the repository today.

Once those gates complete, start with the repository's New Issue flow. Use a
bug report for reproducible failures and a feature request for new behavior.
For substantial changes, agree on the problem and intended outcome with a
maintainer before investing in an implementation.

## Development setup

Hero requires the Go version declared in `go.mod`.

```bash
go build ./...
go test ./...
```

To exercise the command locally:

```bash
go run ./cmd/hero --help
```

The full project checks are:

```bash
go vet ./...
go test -race -count=1 ./...
go build ./...
```

Changes to the landing page and hosted documentation have focused validators:

```bash
python3 web/landing/scripts/test_landing_build.py
python3 web/docs/scripts/test_docs_build.py
```

## Pull requests

Keep each pull request focused on one outcome. Include:

- the issue or Hero spec it addresses;
- a concise explanation of the behavior change;
- tests or an explanation of why no automated test applies;
- documentation updates when commands, configuration, or public behavior
  change; and
- confirmation that logs, fixtures, screenshots, and examples contain no
  credentials or private data.

Do not include source, binaries, internal documentation, or brand assets from
Hero Code or Hero Cloud. They are separate proprietary products. Sprout is a
separate MIT-licensed project and is not governed by this repository.

Before submitting, read [the Code of Conduct](CODE_OF_CONDUCT.md),
[support boundaries](SUPPORT.md), and [security policy](SECURITY.md). A root
license must exist before an external contribution can be accepted; when it
does, contributors must have the right to submit their work under that license.

