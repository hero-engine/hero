---
description: Refine a PM artifact (feature, PRD, epic, initiative) toward delivery readiness.
---
Route this refinement request to `pm-delivery-lead`, who selects the right authoring specialist based on the artifact's `type` (and `kind` where relevant).

## Pre-flight

1. Load the active methodology preset via the `pm-preset-detection` skill — it determines whether refinement uses sprint/flow shape (ten-section PRD, points on specs) or cycle shape (pitch PRD, appetite, hill position).
2. Read the active vocabulary preset so display names in agent output match the workspace's chosen terminology (Story / Scope / Card / Issue / Feature).
3. Read the target spec's frontmatter to confirm its `type` and `kind`.

## Sub-routing by spec type

`pm-delivery-lead` dispatches under the unified type model:

| Spec type | Specialist | Loads skill |
|---|---|---|
| `feature` / `bug` / `chore` | `story-writer` (canonical pack name; vocabulary-aware display) | `story-writing-invest`, `acceptance-criteria-ears` |
| `prd` | `prd-author` | `prd-structure` |
| `epic` | `pm-delivery-lead` direct (`epic-framer` takes over in v1.5) | — |
| `initiative` | `product-strategist` | `roadmap-framing` |
| `intake` | `intake-triager` | `intake-classification`, `duplicate-detection` |

## Flags

- `--section <name>` — refine only one section (e.g. `--section ac` to tighten acceptance criteria, `--section appetite` for cycle pitches, `--section evidence` for initiatives). The specialist edits only that section.
- `--inline-propose` — land refined content as proposed content in the artifact pane (the UI's proposed-content semantics) rather than as a chat-side draft. Default behavior depends on whether the call originated from a contextual button (button → inline-propose; raw slash → chat draft).

## Output

- The spec file on disk is updated.
- A one-line log to chat naming what was refined: `refined cart-abandon-email AC → 4 EARS criteria, 2 freeform`.
- Where the UX supports it, refined sections are marked as proposed content awaiting accept/reject.

The artifact is the deliverable; chat is the trace. Do not paste the refined content into chat.

Request: $ARGUMENTS
