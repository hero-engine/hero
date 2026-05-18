---
description: Capture a freeform note into the knowledge base.
---
Save the user's current thought as a standalone note in
`<workspace>/.hero/knowledge/notes/`. Notes are reference material —
short, self-contained, and discoverable later via the corpus pill and
the assistant's tool calls.

Steps:

1. Read the user's intent from `$ARGUMENTS` or the recent conversation.
2. Choose a short kebab-case slug derived from the first few meaningful
   words (no leading dot, no extension).
3. Write `<workspace>/.hero/knowledge/notes/<slug>.md` with frontmatter:
   - `title`: concise human title
   - `created`: today's ISO date
   - `source`: `chat`
   - `tags`: optional, when an obvious one-word topic is present
4. Body is the note content as the user dictated, lightly cleaned up
   for legibility (no editorializing, no extra commentary).
5. Confirm the path back to the user as a linkified markdown.

Don't create folders other than `notes/` unless the user asks. Don't
prepend hero-style EARS bullets — this is freeform reference material,
not a spec.

Request: $ARGUMENTS
