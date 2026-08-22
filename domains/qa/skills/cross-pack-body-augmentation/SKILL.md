---
name: cross-pack-body-augmentation
description: Append attributable QA findings to another pack's artifact without overwriting its authored content or ownership.
---
# Cross-pack body augmentation

Preserve the existing body byte-for-byte outside the dedicated QA findings block.
Each finding records author or agent, timestamp, source case or evidence, observed
behavior, affected acceptance criterion, and proposed action. Add new entries
idempotently using a stable finding identifier. Treat the block as a proposal and
do not mutate lifecycle state unless the user confirms through the owning workflow.

