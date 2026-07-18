# Hero Research — Dormant Seed

This directory is **preserved seed material for a possible future Hero Research
product**. It is **not** a live pack. Nothing loads it, nothing stages it, and no
client serves it today. It exists so the research doctrine authored for Hero Chat
is not lost — only parked.

## Provenance

The content here was authored by the **`chat-canonical-research`** feature
(completed 2026-07-18), which extended the baseline Hero Chat pack with a guided
research workflow: a `/research` command, three specialist agents
(`researcher`, `document-analyst`, `data-analyst`), five research/analysis skills
(`research-workflow`, `source-evaluation`, `evidence-and-citation`,
`document-analysis`, `data-analysis`), and a client-agnostic
`plan → round → evaluation → synthesis → report` checkpoint/interrupt contract.

The **`chat-sheds-research-to-seed`** decision (accepted 2026-07-18) reversed that
boundary: **basic Hero Chat does not own research.** The apparatus was removed from
`domains/chat` and preserved here instead. These nine files are recovered
**byte-for-byte from commit `3a09d27`** (the last commit before the removal in
`04a0b5d`) — they are the reviewed-and-shipped doctrine, frozen, not re-authored.

## Dormancy — why this is safe to leave here

This tree lives at repo root under `seeds/`, **outside `domains/`**, on purpose:

- **Not staged by any client.** hero-code's content extraction reads only
  `domains/`; a tree outside `domains/` is invisible to it. Placing this content
  anywhere under `domains/` (even a nested subdir) risks it being staged and
  re-exposed to a client — the exact trap the governing decision names.
- **Not embedded.** Go's `go:embed` set in `content.go` enumerates specific
  `domains/*` paths only; nothing here is embedded into the `hero` binary.
- **Not a domain.** `seeds/hero-research` is not in `AvailableDomains()`, has no
  `DomainFS` case, and is not a registered pack. Nothing resolves or serves it.

Its frontmatter is nonetheless validated by `TestChatPack` (in the repo root Go
package) so the dormant seed cannot quietly rot into malformed frontmatter while
it waits.

## Reviving it is a new decision — not a silent restore

If Hero ever wants real research capability, the revival is a **deliberate,
recorded decision**, e.g.:

- a `research` domain that `Extends: chat` (a served, switchable pack), or
- a standalone **Hero Research** application.

Either path can lift this doctrine directly. But **do not move this content back
under `domains/`** as an ad-hoc change — that would re-bake research into a client
without the product decision behind it. Start from a new decision spec, then port
from here.

## Layout

```
seeds/hero-research/
  README.md                              (this file)
  commands/research.md                   (the /research workflow command)
  agents/researcher.md                   (runs the workflow end to end)
  agents/document-analyst.md             (grounded single-document analysis)
  agents/data-analyst.md                 (structured-data analysis)
  skills/research-workflow/SKILL.md      (the checkpoint/interrupt contract)
  skills/source-evaluation/SKILL.md      (source triage)
  skills/evidence-and-citation/SKILL.md  (cited synthesis, contradiction-surfacing)
  skills/document-analysis/SKILL.md      (grounded deep-read doctrine)
  skills/data-analysis/SKILL.md          (honest data-analysis doctrine)
```
