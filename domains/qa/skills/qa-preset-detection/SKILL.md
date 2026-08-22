---
name: qa-preset-detection
description: Resolve local QA gate style, case format, issue persistence, rejection strictness, and blocker policy with safe defaults.
---
# QA preset detection

Read QA configuration from the project before authoring or proposing transitions.
Resolve gate style, case format, test-issue persistence, rejection strictness, and
blocker policy independently so teams can mix practices. When a value is absent,
use the documented pack default and label it. When a value is invalid, stop the
affected operation with the key and accepted values; never silently coerce it.

