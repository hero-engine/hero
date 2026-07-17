---
name: story-mapping
description: Jeff Patton's story map — a horizontal backbone of user activities with tasks hanging below, a walking-skeleton first slice, and release slices cut across the map.
metadata:
  audience: story-writer, pm-delivery-lead, roadmap-curator
  purpose: framework-guidance
---

## What I do

Give agents Jeff Patton's user story mapping (*User Story Mapping*, 2014) — the technique that turns a flat backlog into a two-dimensional map of the user's whole experience, so you can see the forest before you argue about the trees. A story map arranges work along the **narrative flow of what the user does** (left to right) and by **priority within each step** (top to bottom), then slices horizontally into releases. It's how you find the thinnest end-to-end slice worth shipping instead of building one corner perfectly and nothing else.

## When to use me

- breaking a big feature or initiative into shippable stories without losing the whole (`story-writer`, `pm-delivery-lead`)
- planning a first release and needing to know what's *minimum* and still *viable* (`roadmap-curator`)
- a backlog is a flat list and nobody can see how the pieces form a coherent user experience
- the team is about to build the settings screen to perfection before the core flow exists

## The map's structure

```
BACKBONE →   Sign up      Compose        Send        Track
             (activity)   (activity)    (activity)   (activity)
             ─────────────────────────────────────────────────
  tasks ↓    email/pass   plain draft    to one      delivered?
             SSO          attach file    to many      opened?
             invite team  templates      schedule     bounced?
             ─────────────────────────────────────────────────
```

- **Backbone** — the horizontal spine: the sequence of **user activities** that tell the story of using the product, in the order the user does them. Coarse-grained, verb-shaped, technology-free. "Compose," not "the editor component."
- **Tasks** — under each activity, the specific things a user does to accomplish it, ordered top (essential) to bottom (nice-to-have). These become candidate stories.
- **Priority is vertical.** The higher a task sits under its activity, the more essential it is. This is what lets you slice.

## The walking skeleton — the first slice

The **walking skeleton** is the thinnest possible path that goes all the way *across* the backbone: the minimum task under each activity so a real user can complete the entire journey end to end, even if crudely. It "walks" (works end to end) and it's a "skeleton" (no meat yet).

For the map above, the walking skeleton is: email/password signup → plain draft → send to one → delivered receipt. Ugly, but a user can actually get from start to finish. **You build the skeleton first**, then add flesh. The alternative — building "Compose" to perfection while "Send" doesn't exist yet — produces something that demos one screen and does nothing.

## Release slicing

Once the skeleton walks, you slice the rest of the map into releases with **horizontal cuts**:

- **Release 1 (walking skeleton):** the top task under every activity. End-to-end, thin.
- **Release 2:** the next band down — attachments, send-to-many, open tracking. Each still spans the backbone.
- **Release 3+:** the lower-priority tasks — SSO, templates, scheduling, bounce diagnostics.

Every release cuts *across* all activities, so every release delivers a more capable version of the *whole* journey — never one complete activity and three empty ones.

## Running a mapping session

The map is built, not written. A working session:

1. **Frame the goal.** One user, one goal, start to finish. "A team member submits an expense and gets reimbursed."
2. **Lay the backbone.** Write each user activity on a card, left to right in the order the user does them. Argue about *order and completeness*, not detail, until the story reads coherently across the top.
3. **Fill tasks downward.** Under each activity, brainstorm the tasks a user might do. Don't filter yet — get them all up.
4. **Rank each column vertically.** Drag the essential tasks to the top of each activity, the optional ones down. This vertical sort *is* the prioritization.
5. **Draw the first slice.** Cut a horizontal line just below the top task in every column — that's your walking skeleton. Everything above the line is release 1.
6. **Draw the next slices.** Successive horizontal cuts become release 2, 3, … each spanning the whole backbone.

The physical act of ranking tasks *within an activity* is what surfaces disagreement about what's actually essential — disagreement a flat backlog hides.

## Worked example — release 1 vs the trap

A team maps an expense-reporting tool: backbone = Capture receipt → Categorize → Submit → Approve → Reimburse. The tempting mistake is to build Capture (OCR, multi-format, auto-crop) to a polish before Submit exists. The map makes the error obvious: a beautiful Capture with no Submit reimburses nobody. Release 1 is the skeleton — photo upload → pick a category from a list → submit → one-click approve → mark paid. It's thin at every step but a dollar can actually flow. Release 2 adds OCR, approval routing, and export.

## How the map connects to the rest

Map cells are candidate stories, not finished ones. Each task you commit to a release gets authored as an INVEST story — see `story-writing-invest` — with its own acceptance criteria; the map gives you the *slicing*, INVEST gives you the *shape*. The backbone activities often map upward to initiatives on the roadmap — see `roadmap-framing` — so the map is also how a committed initiative decomposes into a release-sequenced set of specs.

## Why the second dimension matters

A flat backlog answers "what's next?" but never "does this add up to something usable?" The story map's second dimension — the horizontal narrative — is what makes gaps visible. When the backbone reads left to right and a slice cuts across it, you can *see* whether release 1 lets a real user finish the journey or strands them halfway. A prioritized list can rank "attachments" above "send" and no one notices the absurdity until build time; the map makes it obvious that shipping attachments before send delivers nothing. That shared, visible picture is also the map's quiet superpower in a room: it turns "what should we build" from a list-argument into a conversation everyone can point at.

## Anti-patterns

- **Mapping features instead of activities.** A backbone of "the dashboard," "the settings page," "the API" is an architecture diagram, not a story map. The backbone is what the *user does*, in narrative order — verbs, not components.
- **No walking skeleton.** Building one activity to completion before the others exist. You get a polished corner and no usable journey. Slice the thin end-to-end path first.
- **Slicing by layer.** "Release 1 = backend, release 2 = frontend." That's not a release — nobody can use a backend alone. Every slice spans the whole backbone so it delivers a usable journey.
- **A map that's really a backlog.** Cards with no left-to-right narrative and no vertical priority. If you can't read the user's journey across the top, it isn't a map.
- **Perfecting the top row.** Piling every enhancement into release 1 until the "minimum" is the whole product. The skeleton is deliberately crude; polish is release 2.
- **Ownerless activities.** Backbone steps nobody actually does. If no user performs an activity, it doesn't belong on the spine.

## Cross-references

- `story-writing-invest` — map cells graduate into INVEST stories with acceptance criteria; the map slices, INVEST shapes.
- `roadmap-framing` — backbone activities often ladder up to initiatives; the map decomposes a committed initiative into release-sequenced specs.
- `personas-and-journey-maps` — the journey map traces *current* experience; the story map plans the *future* build. Complementary, not the same artifact.
- `cycle-planning` — release slices become the cycles/appetites the team commits to.
- Prior art: Jeff Patton, *User Story Mapping* (2014); the walking-skeleton concept from Alistair Cockburn.
