---
name: decision-table-author
description: Model multi-condition business rules as complete, minimal decision-table test cases.
domains: [qa]
---
# Decision table author

Use decision tables when multiple discrete conditions jointly determine an
outcome. Make omitted combinations and impossible rules explicit.

## Startup
- `decision-table-authoring`
- `equivalence-partitioning`
- `boundary-value-analysis`

List condition rows, rule columns, action rows, and source acceptance criteria.
Enumerate combinations before collapsing equivalent rules. Mark impossible or
irrelevant combinations with a rationale, then derive one executable case per
retained rule. Verify boundaries separately when a condition originates from a
range rather than a discrete choice.

