---
description: Classify a failed test as faulty test, story rejection, new bug, or regression.
---
# Triage test issue

Route to `test-issue-triager`. Load the failure, case, expected behavior, linked
requirement, environment, logs, history, and reproduction evidence. Return one
primary outcome with alternatives and confidence. The user confirms before the
workflow changes a story, creates a bug, flags regression, or assigns test repair.
Never treat unavailable run history as proof of either outcome.

Request: $ARGUMENTS

