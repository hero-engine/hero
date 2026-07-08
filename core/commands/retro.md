---
description: Run a post-delivery retrospective comparing a completed spec against what was actually implemented.
---
Delegate to the install's delivery-lead agent if one is installed, otherwise
run the retrospective directly as the session agent.

Engineering installs route product work to `feature-delivery-lead` and
platform work to `platform-delivery-lead`; pm installs ship
`pm-delivery-lead`.

Whoever runs this will:
1. Read the completed spec from the provided path
2. Review the git history to see what was actually implemented (diff against the spec's timeframe)
3. Compare spec intent vs delivered reality:
   - What matched the plan
   - What deviated and why
   - What was harder than expected
   - What was easier or unnecessary
4. Identify learnings:
   - Convention updates worth capturing (suggest `/convention` if appropriate)
   - Decision updates or new decisions needed (suggest `/decide` if appropriate)
   - Estimation calibration insights
5. Produce a retrospective summary
6. **Knowledge capture**: If `knowledge.auto_capture` is enabled in `hero.json` (default: on), don't just suggest conventions and decisions — actually write them. Silently create knowledge entries in `.hero/knowledge/` for any conventions, decisions, or rules that emerged from the retro. Run `hero index` when done. Briefly mention what was captured. See the `auto-knowledge-capture` skill for details.

If a spec path is provided, use it directly. If a slug or description is provided, look for the matching spec in the hero specs folder.

Spec to retrospect: $ARGUMENTS
