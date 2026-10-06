---
description: Verify a delivered spec — run the cold delivery audit if it is missing or stale, then `hero spec verify`.
---
Verify that a spec's delivery is complete. This is the closing gate on its own: use it when a spec was delivered but its verification is missing, weak, or stale. Hero's next step shows **Verify** for such specs.

1. **Resolve the spec** from the argument (a slug). Read it with `hero_read_spec` or `hero spec show <slug>`. If its status is not `completed`, `delivering`, or `in-review`, stop and say what to run instead: `/deliver <slug>` for unfinished work, or nothing for superseded work.
2. **Check the Completion Ledger.** It must exist, and every row must be `DONE` or a SKIPPED/BLOCKED row with a structured sign-off (`[signed-off] <who> — <why>`). These are the same rows `hero spec verify` accepts; see the `completion-ledger` skill. If other rows remain, stop and report them. Finishing the work is `/deliver`'s job, not this workflow's.
3. **Check the audit report.** `hero spec verify` requires a `delivery-audit.md` beside the spec. It must name this spec in its title (`# Delivery audit — <slug>`) and be newer than the spec file. If it is missing, names another spec, or is older than the spec:
   - Spawn a **fresh** reviewer with the `delivery-audit` skill. Hand it only on-disk artifacts: the spec path, the diff of the delivering commits, the ledger, and the test evidence. The verifier must not grade its own work. If the harness cannot spawn an independent reviewer, stop at this gate and report that the audit needs an independent run.
   - On **HOLD**, report the concerns and stop. Fixing them is `/deliver`'s job.
4. **Run the gate:** `hero spec verify <slug>`. On PASS the spec is completed and archived. If the spec was already completed and archived, the command only reports that. The audit you just produced is then what lifts the spec's verify state to **passed** in Hero's work view, so say so. Report the gate results. On FAIL, report each failing gate and its remedy. Never edit `status:` by hand, and never pass `--force` unless the user explicitly asks for that override.
5. **Report** the verdict in one line, plus the audit headline when the audit ran.
