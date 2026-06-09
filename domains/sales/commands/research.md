---
description: Competitive intel, buyer background, company context, or market research. Prepares you to walk into any meeting fully informed.
---
Route this research request to the `buyer-researcher` agent. For competitive
research, also involve the `competitive-intel` agent.

**Determine the research type** from the argument:

1. **Company research** — `hero research "Acme Corp"` or `hero research acme-corp`
2. **Person research** — `hero research "Jane Smith VP Engineering Acme"`
3. **Competitive research** — `hero research "vs. Competitor X"` or `hero research "battlecard Competitor X"`
4. **Market / segment research** — `hero research "Series B fintech payments"` or `hero research "healthcare CIOs"`

**Check for existing intel** before researching:
```
hero search "<company or topic>"         # existing deal specs, notes, battlecards
hero search --type battlecard "<competitor>" # existing competitive intel
hero search --type knowledge "<company>"    # ingested call transcripts, notes
```
Surface what's already known before starting fresh research. Do not
re-research what Hero already knows.

**Delegate to appropriate agent(s)**:

- Company or person research → `buyer-researcher`
- Competitive intel → `competitive-intel`
- Both → both agents, synthesized into one brief

**For company research**, the agent will produce:

### Company Brief

1. **Company overview** — size, industry, revenue (public/estimated), growth
   stage, HQ, key products
2. **Technology stack** — known tools, vendors, platforms (from job postings,
   public data, integrations)
3. **Recent news** — funding, acquisitions, leadership changes, product
   launches, layoffs (last 90 days)
4. **Pain indicators** — signals this company has the problem Hero solves
   (job postings mentioning pain areas, public blog posts, conference talks)
5. **Buying triggers** — events that make them ready to buy now (new funding,
   new leadership, compliance deadline, competitor win)
6. **Org chart** — key stakeholders and their likely roles in a purchase:
   Economic Buyer, Champion candidates, Technical Evaluator, Procurement
7. **Mutual connections** — shared customers, partner network, LinkedIn
   connections (if available)

**For competitive research**, the agent will produce or update a battlecard:

### Battlecard: Hero vs. [Competitor]

1. **Competitor overview** — positioning, pricing tier, customer base
2. **Where we win** — specific advantages with proof points
3. **Where they win** — honest assessment; don't dismiss
4. **Their objections about us** — what they tell prospects; our responses
5. **Our objections about them** — what we can legitimately say; our proof
6. **Deal signals** — how to know you're in a competitive deal against them
7. **Trap questions** — discovery questions that expose their weaknesses
8. **References** — customers who switched from them to us

**Save the research** to the appropriate spec:
- Company research → append to deal spec at `.hero/planning/deals/<slug>/spec.md`
  under `## Research` section, or create a prospect spec
- Battlecard → write to `.hero/knowledge/battlecards/<competitor>.md`

**Auto-capture** the research into the knowledge base so it benefits
future deals with the same company or competitor.

---

## Flags

- `--company <name>` — explicitly research a company
- `--person <name>` — research a specific contact
- `--vs <competitor>` — competitive battlecard research
- `--segment <description>` — market segment research
- `--depth quick` — 10-minute brief (headlines only)
- `--depth full` — comprehensive report (default)

---

## Session Title

Set the session title to: `research: <target>`

---

Target: $ARGUMENTS
