---
name: document-analysis
description: How to deep-read a single document or corpus item the user points at — map its structure, extract claims and the evidence behind them, answer strictly from the source with locators, and say plainly when the answer is not in the document. Grounded reading, not generation.
metadata:
  audience: document-analyst
  purpose: grounded-document-reading
---

## What I do

I give the discipline for analyzing **one specific document** — a file, a PDF, a
corpus entry, a pasted text — the user has pointed at. The job is grounded
reading: every answer traces to a spot in the document, and "the document does not
say" is a first-class answer. This is the opposite of open-ended research; the
source set is exactly one thing, and the honesty bar is that you never fill a gap
in the document with outside knowledge unless the user explicitly asks you to.

Citation format and the `Sources:` register come from `evidence-and-citation`;
here the "source" is a locator *within* the document.

## When to use me

Load me when the user says "read this and tell me…", "what does this document say
about…", "summarize section 4", or points at a single file or corpus item and
asks a question grounded in it. The `document-analyst` agent loads me
automatically.

## Workflow

### 1. Map the structure first

Before answering anything, build a quick map of the document: its sections, its
apparent purpose, its shape (argument, reference, narrative, data table, mixed).
The map is what lets you locate answers precisely and tell the user when their
question addresses a part the document does not cover.

### 2. Extract claims and their evidence

For an analytical read, separate what the document *claims* from what it offers as
*evidence* for those claims. A document asserting a conclusion with no supporting
evidence is a weaker source than one that shows its work — note that. Map each
significant claim to the evidence (or absence of evidence) behind it.

### 3. Answer with locators

Every answer points at where in the document it came from — a section number, a
heading, a page, a paragraph. The locator is what makes a grounded read
verifiable: the user can jump to the spot and confirm. Quote the document's own
words for anything contested or precise; paraphrase only where paraphrase is
safe.

### 4. Say when it is not there

If the document does not answer the question, **say so plainly**: "This document
does not address X." Do not reach for general knowledge to paper over the gap.
You may then *offer* — as a clearly separate, clearly labeled move — to answer
from outside the document if the user wants that, but the grounded answer and the
outside answer never blur together. The user pointed at this document for a
reason; substituting your own knowledge silently betrays that.

## The honesty rules

- **Grounded means grounded.** Answers come from the document. Outside knowledge
  enters only when the user asks for it, and only clearly marked as outside.
- **Absence is an answer.** "Not stated in this document" is precise and useful.
  A confident answer to a question the document does not address is a
  fabrication.
- **Quote for precision.** When wording matters — a definition, a commitment, a
  number — use the document's exact words with a locator, not a paraphrase that
  might drift.
- **Do not smooth over contradictions.** If the document contradicts itself,
  surface it with both locators rather than resolving it for the author.

## Anti-patterns

- **Answering from training instead of the text.** The failure mode that makes
  document analysis untrustworthy: giving a plausible general answer the document
  does not actually support.
- **Locatorless claims.** "The document says X" with no pointer to where. If you
  cannot locate it, you cannot be sure the document says it.
- **Paraphrase drift.** Rewording a precise commitment into something subtly
  different. Quote when precision matters.
- **Filling gaps silently.** Covering what the document omits with outside
  knowledge, with no signal to the user that they have left the document.
