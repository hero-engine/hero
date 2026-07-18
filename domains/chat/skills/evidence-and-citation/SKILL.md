---
name: evidence-and-citation
description: How to assemble evaluated sources into claims a reader can trust — inline citation on every non-obvious claim, contradictions surfaced rather than resolved away, and a `Sources:` register that lets anyone check the work. The shared citation contract for research and analysis output.
metadata:
  audience: researcher, document-analyst, data-analyst
  purpose: evidence-synthesis
---

## What I do

I define how evaluated evidence becomes a defensible answer: which claims need
citations, what a citation looks like, how to handle sources that disagree, and
how to register the sources so a reader can verify every claim. Deciding whether
a source is trustworthy is `source-evaluation`'s job; I take the evaluated set and
turn it into cited prose.

## When to use me

Load me whenever you are writing an answer that draws on sources — a research
report, a document analysis, a data readout. All three chat specialist agents
load me so the citation contract is identical across their outputs.

## What needs a citation

Every **non-obvious claim** carries an inline citation to the specific source it
came from. A non-obvious claim is any statement a reasonable reader might want to
check: a fact, a number, a quote, an attribution, a conclusion drawn from
evidence. What does *not* need a citation: common knowledge, the user's own
stated premises, and your own reasoning steps (which should be visible as
reasoning, not dressed up as sourced fact).

The test: if a reader asked "how do you know that?", would you point at a source?
If yes, cite it. If your honest answer is "I inferred it," mark it as inference,
not fact.

## Citation format

Cite inline, at the point of the claim, pointing at the *specific* source — not a
vague gesture at "research":

- **Corpus item:** the relative path, e.g. `(.hero/knowledge/decisions/x.md)`.
- **Web source:** the URL, or a short label bound to the URL in the register,
  e.g. `(Smith 2025)` where the register resolves the label to the link.
- **Provided document:** the section or locator within it, e.g. `(§3.2)` or
  `(p. 14)`, so the reader can find the exact spot.

Then close with a `Sources:` register listing every source cited, each with
enough detail to locate it (full path or URL, title, date where relevant). A
claim whose source is not in the register is an uncited claim — fix one or the
other.

## Surfacing contradictions

When two evaluated sources disagree, **surface the disagreement — do not silently
pick a winner.** State that the sources conflict, attribute each position to its
source, and give the reader your weighted read (per `source-evaluation`) of which
is better supported and why. The reader is entitled to know the evidence was
mixed; sanding a contradiction into a single confident claim hides exactly the
uncertainty they most need.

If your evaluation makes one side clearly stronger, say so and say why. If it does
not, present both and say the question is unresolved within the current sources.

## Claims, not source dumps

A synthesis assembles *claims* — statements that answer a sub-question — each
backed by its evaluated evidence. It is not a list of what each source said. The
unit is the claim; sources are what support it. Multiple sources may back one
claim (corroboration, cited together); one source may back several claims.

Confidence travels with the claim. A claim resting on one caveated source is
stated more tentatively than one resting on two independent primary sources. Match
the language to the strength of the evidence — plain assertion for solid claims,
explicit hedging for thin ones.

## The `Sources:` register

Close every cited output with:

```
Sources:
- <path or URL> — <title / description>, <date if relevant>
- ...
```

The register is what makes the work checkable. It is not optional decoration; it
is the difference between an answer a reader can audit and one they must take on
faith.

## Anti-patterns

- **The confident uncited claim.** A fact stated flatly with no source behind it,
  indistinguishable from one that has three. The reader cannot tell your
  well-sourced claims from your guesses.
- **Inference wearing the costume of fact.** Presenting your own reasoning as if
  a source said it. Mark inference as inference.
- **Contradiction laundering.** Picking one side of a source disagreement and
  presenting it as settled, hiding that the evidence was mixed.
- **Source dump instead of claims.** Summarizing each source in turn and leaving
  the reader to assemble the answer. Synthesis is your job, not theirs.
- **The vague gesture.** "Studies show," "research indicates," "sources say" with
  no specific, locatable citation. Point at the exact source or do not make the
  claim.
- **Register drift.** A claim cited to a source that never appears in the
  `Sources:` register, or a register listing sources no claim used.
