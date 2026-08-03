# Donor branch disposition — `design/interactive-cli-input`

This ledger prevents two opposite failures: merging every audit side quest into
the interactive-input initiative, or abandoning valid fixes because the donor
branch became too broad. The donor branch remains intact until the closing gate
confirms every group below has a durable disposition.

Disposition meanings:

- **Port** — selectively reproduce the owned behavior in one successor child.
- **Extract** — retain through the named independent spec/workstream, after
  revalidating against current `main`; never cherry-pick blindly.
- **Evidence only** — keep as investigation/history, with no code port.
- **Replaced** — the clean successor contains the current artifact.

## Commit groups

| Donor commits | What they contain | Disposition | Durable destination |
|---|---|---|---|
| `f4169ed`, `596dbfb`, `d6af840`, `5088536`, `b6d6a55`, `3e17542`, `3e497d0` | Original initiative orchestration and progress edits | Evidence only | This successor initiative and git history |
| `12a6775` | Six-target uninstall parity | Port | `interactive-setup-and-connect-closure` |
| `58910cf`, `401be3a`, `340922b`, `31565ed`, `c8966fd`, `f38e73b`, `66d4876`, `07bd814`, `63cb862` | Prompt baseline, package, migrations, policy guards, and stream tests | Port selectively | `prompt-and-tty-contract-closure` |
| `f459a74` | `migrate-nested` confirmation/refusal | Port only the original satellite confirmation contract | `interactive-setup-and-connect-closure`; reject unrelated behavior growth |
| `485c32f`, `8bfe053`, `4b1163b` | Connect writer/help and setup prompt adoption | Port selectively | `interactive-setup-and-connect-closure` |
| `1f213e2` | PROMPT/SELECTOR/NEVER-PROMPT classification | Port as governing design evidence | Parent initiative plus prompt-policy tests; do not import stale counts verbatim |
| `9840e8d` | Original corpus selector implementation | Port then repair the >25 hard cap | `corpus-selector-closure` |
| `d10eae2`, `77ccdfa` | Unrelated spec-document corrections | Evidence only | Re-evaluate independently; no successor code dependency |
| `c191f6d` | Shared test binary build/cache isolation | Extract after current-main comparison | New or existing `cli-test-binary-build-isolation` follow-up |
| `c0536bf` | `node_index` repartition migration | Extract, but do not reuse as-is: cold audit found the migration empties search metadata until reprojection | `node-index-repartition-preserves-search` bug |
| `ed75db0` | Root instruction managed-block removal | Extract | `uninstall-never-cleans-root-instruction-file` |
| `d53acdf` | Connect removal key-space repair | Extract | `connect-remove-key-space-mismatch` |
| `a8c9554`, `d5914f3` | Codex config block line-welding diagnosis/fix | Extract | `codex-config-block-removal-welds-lines` |
| `fea8b86`, `7fb96a5`, `f6ea5a8`, `008df82`, `53c4c13`, `a7bd937` | Guided `hero init`, target help, and target-enumeration tests | Extract as a product feature; not part of the original initiative | `init-first-run-setup` plus its target-enumeration follow-up |
| `184ee09`, `bed7a50` | Shared `AGENTS.md` uninstall ownership bug | Extract | The donor bug spec filed by `184ee09`; verify slug/current-main status before promotion |
| `d3cd80d`, `7b57e59` | Broad command-surface triage and remediation specs | Evidence only / extract individual accepted items | `cli-surface-consolidation` and its explicitly designed children |
| `b93f6ef` | Alias flag parity | Extract | `alias-flag-parity`; also retain planned `alias-pair-derivation` and `alias-parity-message-assertion` |
| `7e0f129`, `e6221d7`, `b28cb20`, `c01ecc1`, `05c2653` | Alias argument parity, extra-argument correction, and gate repair | Extract | `alias-args-parity`; revalidate the restored build/vet gate |
| `afdf930`, `ac0c4c0`, `238e3a0` | Repository-wide timeout policy and its audit repairs | Do not port as-is: the cold review found no 10-minute reproduction and a weaker hang signal | `cli-test-package-headroom`; retain useful pipefail assertions separately if still missing on `main` |
| `ecbbe4a` | Generated handoff projection | Replaced | Clean-branch projections only |
| `e725245` | Global Go invocation guard plus 40+ string corrections | Extract findings, not implementation: the guard intentionally misses mid-sentence dead commands | `generated-command-refs-validated` / `cli-invocation-strings-guard`; include the demonstrated dead `hero verify` phrase |
| `6f2f524` | Recomposition planning commit on donor branch | Replaced | Clean successor commit `d5e7cc8` and subsequent design commits |

## Salvage rules

1. No **Extract** row is satisfied merely because its donor spec says
   `completed`; compare the donor diff and tests against current `main` first.
2. If current `main` already contains an equivalent fix, record the containing
   commit and mark the row **already present**.
3. If a donor fix fails its own cold-review evidence, preserve the diagnosis and
   reproduction in the destination spec, not the flawed patch.
4. Extracted work gets its own branch/PR or an already-scoped initiative. It is
   never appended to this successor branch.
5. The donor branch must not be deleted until the final merge gate resolves
   every placeholder destination (notably the shared-`AGENTS.md` slug) and
   records the final status of every row.
