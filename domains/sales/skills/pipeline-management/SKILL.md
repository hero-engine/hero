---
name: pipeline-management
description: Stage definitions, exit criteria, deal hygiene rules, and the signals that indicate a deal belongs where it says it does. Loaded by forecast-analyst and the /pipeline command.
metadata:
  audience: forecast-analyst, /pipeline command
  purpose: pipeline-hygiene
---

## Stage Definitions and Exit Criteria

Each stage requires specific evidence before a deal can advance. Moving a
deal forward without meeting exit criteria is the primary cause of forecast
inaccuracy.

### Prospect / Prospecting (10% probability)

**What it means:** Lead identified; outreach planned or in progress.
We believe this company fits our ICP.

**What must be true:**
- Company identified and added as a deal spec
- Initial contact identified (who to reach out to)
- ICP fit assessment completed
- Outreach sequence started or planned

**Exit criteria to advance to Qualifying:**
- Initial response from a contact at the company
- At least one substantive conversation (call or email exchange)
- Confirmed there is a potential problem we can solve

**Red flags at this stage:**
- Deal added with no outreach activity for 14+ days
- No contact identified

---

### Qualifying (20% probability)

**What it means:** In active conversation; assessing whether this deal
is real and worth pursuing. MEDDPICC qualification in progress.

**What must be true:**
- At least one discovery conversation completed
- Pain has been articulated (Identify Pain dimension started)
- Champion candidate identified (even if not yet confirmed as champion)

**Exit criteria to advance to Demo / Evaluation:**
- MEDDPICC score ≥ 40 (or equivalent on configured framework)
- Pain confirmed and quantified (or reasonably estimated)
- Economic Buyer identified by name and title (even if not yet met)
- Timeline established (even if tentative)
- Decision criteria understood at high level

**Red flags at this stage:**
- No discovery call completed after 2+ weeks of activity
- MEDDPICC score < 25 after 3 conversations
- Rep cannot identify a champion

---

### Demo / Evaluation (40% probability)

**What it means:** Active technical and business evaluation. Buyer is
comparing vendors, running a formal evaluation, or in proof-of-concept.

**What must be true:**
- Demo or trial access has been provided
- Technical stakeholders are engaged
- Decision criteria documented in more detail

**Exit criteria to advance to Proposal:**
- Demo or POC completed and feedback received
- Technical evaluation is positive (no outstanding blockers)
- Economic Buyer has been engaged (meeting held or scheduled)
- Decision criteria and decision process fully documented
- Timeline confirmed and realistic
- MEDDPICC score ≥ 55

**Red flags at this stage:**
- Demo completed but no next step scheduled
- Technical evaluation stalled with no clear blocker surfaced
- Economic Buyer not yet engaged
- Competitive situation unknown

---

### Proposal (60% probability)

**What it means:** We have submitted (or are preparing to submit) a
formal proposal, SOW, or pricing document.

**What must be true:**
- Economic Buyer has been engaged in at least one meeting
- Business case has been presented or is ready
- Decision criteria confirmed and our proposal addresses them

**Exit criteria to advance to Negotiation:**
- Proposal submitted and acknowledged
- Buyer has confirmed they have budget (or a clear path to budget)
- Verbal intent to move forward (buying signal from EB or champion)
- Legal/procurement review process started

**Red flags at this stage:**
- Proposal submitted without EB engagement
- No response to proposal after 5+ business days
- Buyer has not confirmed budget availability

---

### Negotiation (80% probability)

**What it means:** Buyer has indicated intent to purchase. We are
working through commercial and legal terms.

**What must be true:**
- Verbal or written intent to purchase from an EB-level contact
- Legal review in progress or complete
- Specific commercial terms being negotiated (discounts, term length, payment)

**Exit criteria to advance to Closed Won:**
- Signed agreement / MSA / Order Form
- Purchase Order received (if required)
- Payment terms agreed

**Red flags at this stage:**
- "In negotiation" for 30+ days with no movement on legal
- Buyer unresponsive to commercial terms discussions
- New stakeholders appearing late with unknown concerns

---

### Closed Won (100%) / Closed Lost (0%)

**Closed Won:** Signed agreement in hand. Record the actual close date.

**Closed Lost:** Deal will not close. Record the actual outcome and reason.
Mark for debrief immediately.

---

## Pipeline Hygiene Rules

These rules apply at all times. Violations should be surfaced in the
`/pipeline` command output.

### 1. Every deal must have a next action

No deal should exist without a specific next action, an owner, and a date.
"Follow up" is not a next action. "Call Jane to schedule the CFO meeting
— Rep — Friday" is a next action.

### 2. Close dates must be realistic

A close date is not a wish. It should reflect the actual decision timeline
of the buyer given their stated process. If the close date requires legal
review in 3 days when legal typically takes 3 weeks, the close date is wrong.

Rule: if a deal's close date has passed without closing, it must be updated
within 48 hours with a revised date and a note on what happened.

### 3. Stale deal thresholds

This skill is the single owner of all numeric staleness and risk thresholds.
Sibling skills reference these values rather than restating them.

| Stage | Stale threshold | Action |
|---|---|---|
| Prospect | 14 days no activity | Re-engage or qualify out |
| Qualifying | 14 days no activity | Manager check-in |
| Demo/Evaluation | 21 days no activity | Escalate to manager |
| Proposal | 10 days no response | Follow-up call required |
| Negotiation | 7 days no movement | Escalate to manager + exec |

### 4. No stage advancement without evidence

A deal moves forward in stage when the exit criteria are met — not when
the rep feels optimistic. Stage advancement without meeting exit criteria
is the leading cause of forecast inaccuracy.

### 5. Single-threaded deals must be flagged

Any deal above $50K ARR with only one active contact is single-threaded
and must be flagged. Not an error — a risk that the deal plan must address.

### 6. Missing fields trigger hygiene alerts

Deals missing these fields are flagged in the `/pipeline` command:
- `close_date`
- `arr`
- `owner`
- A next action in the spec body

---

## Pipeline Review Cadence

**Weekly (rep-level):** Review all deals in Proposal and Negotiation.
Verify exit criteria met, next actions set, stale threshold not exceeded.

**Bi-weekly (manager-level):** Review all deals in Qualifying and Evaluation.
Spot-check exit criteria and MEDDPICC scores. Surface stale deals.

**Monthly (pipeline review):** Full pipeline health review with `forecast-analyst`.
Compare against coverage ratio target (3x). Identify pipeline gaps early enough
to course-correct.
