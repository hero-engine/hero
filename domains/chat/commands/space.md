---
description: Promote the current thread into a new Space for ongoing related work.
---
Create a new free-floating Space that wraps the current conversation's
context, so the user can return to this topic across multiple future
sessions with the right grounding pre-loaded.

Steps:

1. Derive a short name from the thread (Title Case, ≤ 40 chars).
2. Compose a system addendum that captures the topic, key constraints,
   and the user's persona in ≤ 5 sentences. The addendum is what every
   future chat in this Space sees prepended.
3. Use the `SpaceStore` API (or surface a brief summary the user can
   paste into the New-Space dialog if running outside the GPUI shell).
4. Confirm the new Space name back to the user and suggest they open a
   fresh session inside it for the next turn.

Don't carry the literal chat transcript into the addendum — distill
it. The addendum should read like a project brief, not a log.

Request: $ARGUMENTS
