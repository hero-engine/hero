---
title: "Local Project Model — Hero-Bundled Tiny Model Continuously Trained on Project Corpus"
slug: local-project-model
type: feature
status: draft
priority: low
horizon: someday
tags: [research, local-inference, fine-tuning, model-training, long-horizon]
relations:
  - target: next-compact-handoff
    kind: related
  - target: knowledge-flywheel
    kind: related
---

# Local Project Model — Hero-Bundled Tiny Model Continuously Trained on Project Corpus

## Problem

Hero accumulates an enormous amount of project-specific signal over time: every spec written, every decision made, every convention enforced, every bug diagnosed, every peer-call answer, every code change linked back to its motivating spec. Today that corpus is used in two ways: humans browse it, and Hero stuffs relevant slices into prompts sent to a frontier LLM via API.

The frontier-LLM path has real costs:

1. **Latency.** Every "smart" Hero feature (compact handoff summarization, `/resume` curation, peer-call answer drafting, `/why` synthesis) requires a network round trip. Sub-second-feeling features become 1–5s features.
2. **Cost per call.** Cheap with Haiku, but it adds up across a team running thousands of compactions, resumes, and peer-calls per week.
3. **Privacy / sovereignty.** Code and decisions leave the machine. Acceptable for most projects, blocking for regulated or air-gapped environments.
4. **Generic understanding.** A frontier model is brilliant at general code but has no built-in knowledge of *this* project's vocabulary, conventions, decision history, naming patterns, or in-flight initiatives. Every call has to rebuild that context from the prompt.
5. **Re-explaining the project every turn.** RAG mitigates this but every retrieval-augmented call still pays the prompt tokens.

A **tiny model trained on the project's own corpus** — continuously, as the corpus grows — would change all five characteristics. Sub-100ms inference. Effectively zero marginal cost. Fully local. And it would *know the project* without having to be re-told what `outpost` means, who decided to drop the old auth middleware, which conventions are load-bearing vs. aspirational.

The shape of the idea: Hero bundles or installs a small base model (single-digit billions of params), maintains a project-specific LoRA adapter or full fine-tune, retrains incrementally as new specs/events/knowledge land, and routes selected high-volume tasks to the local model — falling back to frontier APIs when local quality is insufficient.

## Goal

Establish whether a project-trained tiny model is feasible, useful, and worth the infrastructure investment. This spec is **exploratory** — the deliverable is an honest answer (with evidence) about whether to build it, not the system itself.

Success at the exploration stage:

- A clear-eyed assessment of quality ceiling vs. Haiku-class API calls for the specific tasks Hero would route locally
- A working prototype on one task (compact handoff summarization is the natural candidate) trained on a real Hero project's corpus, with evaluations against the same task via Haiku
- A cost model: training compute, storage per project, inference compute, ongoing retraining cadence, and what user hardware tier this assumes
- A go/no-go recommendation with the conditions under which it changes

If the answer is "no, the quality gap is too wide and the maintenance burden too high," that's a valid outcome — the analysis prevents future revisits from re-litigating the same questions.

## Design

### Phase 0 — Pre-prototype research (no code)

Survey the current state of:

- **Small-model fine-tuning for code/text comprehension.** What's the floor model size where summarization, decision-extraction, and "what's next" inference become reliable? Likely Qwen2.5-3B, Llama-3.2-3B, Phi-4-mini class as starting points.
- **Adapter strategies.** Full fine-tune vs. LoRA vs. QLoRA. Storage and training cost per option.
- **Continuous learning patterns.** How others handle catastrophic forgetting in long-running fine-tunes (replay buffers, scheduled retraining vs. continual, eval gating).
- **Hardware assumptions.** What user hardware is actually available — MacBook M-series unified memory, RTX-class GPU, cloud GPU. Hero shouldn't assume any one of these.

Output: a written assessment (no code) identifying the most promising base model family and adapter strategy for prototyping.

### Phase 1 — Single-task prototype

Pick the highest-leverage task: **compact handoff summarization** (see [next-compact-handoff](../next-compact-handoff/spec.md)). It already has clear inputs (transcript) and outputs (4 labeled sections), and it's the spec's "Part 2" that the local model would replace.

Build:

1. Curated training corpus from this Hero project itself — every spec, knowledge file, session-arc event, decision, peer-call. Tens of thousands of items.
2. Fine-tune chosen base + adapter on the corpus, focused on the summarization task pattern.
3. Run the prototype on 50 historical compactions (real transcripts from Hero project sessions) and compare against Haiku output by:
   - Blind preference rating from the project's actual users (chet-bellows + a handful of others)
   - Retention of named entities (spec slugs, file paths, decisions) — graded mechanically
   - Latency and cost per call

Output: side-by-side evaluation report. Local model is either "competitive" (≥80% Haiku preference), "promising" (60–80%, with a clear path to closing the gap), or "not worth pursuing."

### Phase 2 — Continuous retraining loop

Only triggered if Phase 1 lands on "competitive" or "promising."

Design questions answered in this phase, with a working implementation:

- **Retraining cadence.** Every spec landed? Every N events? Nightly? On `hero check`?
- **Eval gating.** A fixed regression suite per project — the model can't ship an update unless it scores ≥ previous version on a held-out set. Without this, drift is silent.
- **Adapter storage.** Where does the LoRA live — `.hero/models/`? Gitignored? Per-user / shared? Versioned alongside the project?
- **Drift detection.** Some way to know "the model used to be good and isn't anymore" — e.g. periodic blind comparisons against the frontier API as a quality canary.

### Phase 3 — Routing layer

A `hero.json` config that names which tasks route locally vs. frontier, with sensible defaults and per-task fallback to frontier when local confidence is low:

```json
{
  "model_routing": {
    "compact_handoff_summary": { "local": true, "fallback": "anthropic:haiku" },
    "peer_call_draft": { "local": false },
    "spec_synthesis": { "local": false }
  }
}
```

Default: most things stay on frontier API. Local model is opt-in per task as quality is demonstrated.

### Phase 4 — Distribution

Hero binary stays slim; the model is **fetched on demand** by a separate `hero models install` command. Base model downloaded once (~2–8 GB depending on choice). Per-project adapter is small (~tens to hundreds of MB) and either lives in the repo (committed) or in `~/.hero/models/<project-id>/` (per-machine).

## Out of scope

- Replacing the frontier API entirely. The local model handles a curated set of high-volume tasks; everything else continues to route to Anthropic / OpenAI.
- Distributing Hero with a pre-trained model baked in. The base is fetched at install time, not bundled in the binary.
- A multi-project foundation model that learns across all Hero users. That's a separate (and much harder) product question about data sharing, consent, and centralization.
- Replacing RAG. A local model and a retrieval index are **complementary** — see [embeddings-index](../embeddings-index/spec.md) (if written). RAG buys "knows the project" without training; the local model buys "knows the project AND is fast and free." Either alone is useful; together they're stronger.

## Risks

- **Quality gap too wide.** Realistic risk: 3B-class models are noticeably worse at comprehension than Haiku for tasks like "extract decisions from a long transcript." If the Phase 1 prototype lands at 40% preference, the answer is no and we move on.
- **Training infrastructure is a real product.** "Continuously retraining" sounds elegant but it's a service: scheduling, eval harness, version management, rollback, drift detection. Mis-scoping this as a side project leads to a half-built training system that quietly degrades.
- **User hardware variance.** Inference on a low-end MacBook may already be sluggish for a 3B model; training on it is impractical. Either we assume a hardware floor (excluding some users) or we add a "Hero training service" (centralized infra, a different business model).
- **Continuous learning is unsolved.** Catastrophic forgetting, training instability, and "when to retrain" are open research problems. We'd be doing applied research, not engineering — fine if budgeted that way, fatal if confused for a normal feature.
- **The bar moves.** Frontier-model prices keep falling and quality keeps rising. The Phase 1 evaluation might land "competitive" today and "obsolete" 12 months later. Need a clear re-evaluation trigger.
- **Maintenance burden on Hero core.** Even after shipping, the training pipeline is a long-term liability — base models get deprecated, adapter formats change, fine-tuning libraries break.
- **Privacy promise reversed by accident.** If training data ever leaks (e.g. a hosted training service), the privacy story collapses harder than a frontier-API story does, because users believed it was local.

## Acceptance criteria (for Phase 0 exploration deliverable)

- [ ] Written assessment of base model candidates with reasoned recommendation
- [ ] Hardware assumptions documented and validated against realistic Hero user profiles
- [ ] Continuous-learning strategy survey with named techniques and citations
- [ ] Cost model (training + inference + storage) under three scenarios: solo dev, 5-person team, 50-person team
- [ ] Explicit go/no-go criteria for Phase 1, written before Phase 1 runs

## Kickoff

This is an exploratory spec. The deliverable is **a decision**, not a feature.

Start with Phase 0 only. Do not build anything. Read [next-compact-handoff/spec.md](../next-compact-handoff/spec.md) for the task this is meant to accelerate. Survey current state-of-the-art for small-model fine-tuning on code-and-text-comprehension tasks (Qwen2.5-3B, Llama-3.2-3B, Phi-4-mini class), continuous-learning strategies, and realistic hardware floors for Hero's user base.

Output a written assessment with a clear recommendation: proceed to Phase 1 prototype, or close as "not worth it" with the analysis preserved so future revisits don't re-litigate.

Time budget: ~2 days of focused research, no implementation. If the assessment can't be reached in that budget, the answer is probably no.
