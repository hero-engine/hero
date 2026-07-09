---
description: Pose a question scoped strictly to the knowledge corpus.
---
Answer the user's question using *only* content under
`<workspace>/.hero/knowledge/`. Cite every claim back to a source
path. When the corpus doesn't contain the answer, say so explicitly
and offer to run a regular (non-scoped) query.

Steps:

1. Search the corpus first with whatever semantic-search capability the
   session exposes, falling back to a plain text/grep search where it
   doesn't. Read the matching files with the session's file-read
   capability.
2. Synthesize a concise answer (≤ 6 sentences) grounded only in what
   you found.
3. List every source you drew on as a `Sources:` footer with relative
   paths.
4. If no corpus content is relevant: say so plainly. Suggest the user
   either run the question without the `/ask-corpus` scope, or
   `/capture` the missing knowledge first.

Don't speculate. Don't extrapolate beyond what the source files say.
The point of `/ask-corpus` is *grounded* answers — drift defeats it.

Request: $ARGUMENTS
