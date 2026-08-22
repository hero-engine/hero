---
name: step-by-step-authoring
description: Write executable manual test cases with precise actions, data, and one observable expected result per step.
---
# Step-by-step authoring

Start with purpose, source criterion, preconditions, environment, and named test
data. Each step uses a concrete action and a directly observable expected result.
Prefer one assertion per step; move setup shared across cases into preconditions.
Avoid vague phrases such as "verify it works," hidden dependencies, and exact UI
mechanics that are irrelevant to behavior. End with cleanup or persistence
expectations when execution changes durable state.

