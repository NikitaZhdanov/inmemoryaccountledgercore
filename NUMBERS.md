# Number inventory — specification stage

Task inputs, human choices, derived results, and chosen verification examples
are distinguished below. No implementation limits/runtime choices are selected.
Verification uses exact equality without numerical tolerance. Monetary/rate
calculations require exact arithmetic; formatting separators are cosmetic.

| Value | Meaning, units, source/rationale | Why this value rather than half |
| --- | --- | --- |
| Days 1–6; six days | TASK.md reporting/accrual window | Half changes the supplied window; day labels are discrete identifiers. |
| ACC-001, ACC-002; two accounts | TASK.md AED/BHD accounts | Identifiers cannot be halved; omitting an account changes scope. |
| E1–E10; ten events | TASK.md immutable replay order | Identifiers cannot be halved; a shorter stream omits required events. |
| AED 0.00; BHD 0.000 | TASK.md opening balances | Zero is its own half; a nonzero opening changes the brief. |
| AED 25.00 | TASK.md daily overdraft fee | Half violates the specified charge. No BHD tariff/FX is selected: report undefined policy for a negative BHD close, post no fee, retain balance, continue, per human decision. |
| Once per account/assessed day | TASK.md maximum assessment frequency | Half an occurrence is meaningless; corrections retain one daily assessment identity. |
| 0.04% = 0.0004 = 1/2,500 daily | TASK.md exact positive-balance interest rate | Half changes the contractual rate. Daily interest is eligible base / 2,500 rounded HALF_EVEN to currency precision. |
| AED two places; BHD three | TASK.md storage/rounding scales | Discrete currency conventions, not adjustable magnitudes. |
| AED 0.01; BHD 0.001 | Derived minor units, 10^−scale | Half is not representable at the mandated scale. |
| Factors 100 (10^2), 1,000 (10^3) | Derived minor-unit factors for AED/BHD | Half changes the unit and invalidates the representability check. Input × factor must be an integer exactly. |
| Zero available balance | TASK.md inclusive approval boundary | Zero is its own half; resulting negative availability prohibits approval. |
| Three instalments | TASK.md E10 split count | A fractional count is meaningless and changing count alters the input. |
| HALF_EVEN | Human choice for daily calculated interest | Discrete rounding convention; no meaningful half. At an exact half-minor-unit midpoint choose an even minor-unit count. Capitalize the exact rounded daily sum. |

## Event constants preserved from TASK.md

| Event | Booked day | Value day | Account | Amount/reference |
| --- | --- | --- | --- | --- |
| E1 | 1 | 1 | ACC-001 | Credit AED 1,200.00 |
| E2 | 1 | 1 | ACC-001 | Debit AED 950.00 |
| E3 | 2 | 2 | ACC-001 | Auth-A hold AED 200.00 |
| E4 | 3 | 3 | ACC-001 | Credit AED 400.00 |
| E5 | 4 | 4 | ACC-001 | Auth-A settlement AED 185.00 |
| E6 | 4 | 4 | ACC-001 | Auth-Z settlement AED 180.00 |
| E7 | 5 | 2 | ACC-001 | Debit AED 620.00 |
| E8 | 5 | 5 | ACC-001 | Auth-B hold AED 90.00 |
| E9 | 6 | 2 | ACC-001 | Reverse E7; derived inverse credit AED 620.00 |
| E10 | 5 | 5 | ACC-002 | Credit BHD 10.000 in three instalments |

Every event amount/date is task-mandated; half changes the input. Event/auth/
account references are identifiers, not quantities. E9's inverse is derived
from E7, not a newly selected amount.

## Independently derived AED state and fee calculations

These are results, not adjustable constants. Halving breaks the displayed
event arithmetic, mandatory charge, or human-agreed ordering.

- After E2: 1,200.00 − 950.00 = 250.00. E3 approval leaves 250.00 −
  200.00 = 50.00 available, unchanged ledger 250.00.
- Before E5: 250.00 + 400.00 = 650.00 ledger; 650.00 − 200.00 = 450.00
  available. After E5: 650.00 − 185.00 = 465.00 ledger/available. Full
  200.00 hold released, zero remains. Unused 200.00 − 185.00 = 15.00
  raises availability without a separate credit.
- Criterion 1: fee-free Day 2 using E1–E8 = 250.00 − 620.00 = −370.00.
  Actual posted Day 2 known at Day 5 close = −370.00 − 25.00 = −395.00.
  Final restated Day 2 = −395.00 + 620.00 + 25.00 = 250.00. Distinct
  checkpoints, not conflicting expectations or mutable input.
- E7 reconciles three closed days, Days 2–4 ascending, before E8; Day 5's
  own assessment waits until closing. This human-selected boundary cannot
  be halved without omitting affected days or including the open day early.
- After E7, principal before fees: Day 3 = 250.00 − 620.00 + 400.00 =
  30.00; Days 4/5 = 30.00 − 185.00 = −155.00. E6 has no debit; E8
  has no ledger debit.
- Including prior fees: Day 2 closes −395.00; Day 3 = −395.00 + 400.00
  = 5.00. Day 4 pre-own-fee = 5.00 − 185.00 = −180.00, close −205.00
  after 25.00 fee. E8's proposed availability = −205.00 − 90.00 = −295.00;
  decline without a hold. Day 5 closes −205.00 − 25.00 = −230.00.
- Three assessments on Days 2, 4, 5 total 3 × 25.00 = 75.00, booked Day 5.
  Three linked refunds total 75.00, booked Day 6 with values Days 2, 4, 5.
  Date labels are discrete; halving cannot preserve the agreed chronology.
- E9 debit inverse nets exactly zero with E7. Each 25.00 refund nets zero
  with its assessment at the same value date. Current balance before
  capitalization = −230.00 + 620.00 + 75.00 = 465.00. Half a correction
  would fail financial restoration.

## Interest and final balances

Derived from balances / 2,500 and HALF_EVEN at currency precision; halve
these values only by changing the required inputs/rate or breaking reconciliation.

- After E7 Day 3: 5.00 / 2,500 = 0.002, rounded AED 0.00. Days 2, 4,
  5 accrue zero while negative.
- Restored AED Days 1–5: bases 250.00, 250.00, 650.00, 465.00, 465.00;
  daily accruals 0.10, 0.10, 0.26, 0.19, 0.19, sum 0.84. Each 465.00
  base yields exact 0.186, rounded to 0.19. Days 2–5 correct from zero
  total to 0.74. That is derived unpaid accrual, not a 0.74 ledger credit.
- Day 6 also earns AED 0.19 on 465.00, excluding capitalization. Total
  capitalization = 0.10 + 0.10 + 0.26 + 0.19 + 0.19 + 0.19 = 1.03.
  Final ledger/fee eligibility base = 465.00 + 1.03 = 466.03; no fee.
- BHD Days 1–4 balances/accruals are zero. E10 produces 10.000 on Day 5;
  Day 5 and Day 6 each earn 10.000 / 2,500 = 0.004. Capitalization =
  0.008; final close/fee eligibility base = 10.008; interest base = 10.000.
  No negative BHD close occurs; no BHD fee or exchange rate is inferred.

## Instalment conservation

Derived 10,000 BHD minor units ÷ 3 = quotient 3,333, residual 1. Human
choice: final instalment receives that residual; amounts 3.333, 3.333, 3.334
sum to 10.000. Position is discrete and cannot be halved; halving amounts
breaks conservation. Requiring 3.334 each gives 10.002, excess 0.002.
These are arithmetic results, not independently selected constants.

## Chosen verification examples and parameters

These are illustrative cases, not modifications to the ten-event stream.

- HALF_EVEN midpoint bases AED 12.50/37.50 yield exact 0.005/0.015 and
  rounded 0.00/0.02. BHD 1.250/3.750 yield 0.0005/0.0015 and rounded
  0.000/0.002. Bases are chosen to exercise ties in both directions/currencies;
  half a base removes its exact tie. Outputs follow the formula, not choices.
- Day 6 boundary cases: prior accrual S = AED 0.02; pre-capitalization/
  pre-own-fee B = −0.01, −0.02, −0.03. Results after capitalization:
  0.01/no fee, 0.00/no fee, −25.01 after 25.00 fee. Day 6 interest zero.
  Two minor units of credit allow all starting balances to be negative while
  spanning one unit above, at, and below zero afterwards. Half loses that
  set of adjacent representable cases. Outputs are derived from B + S − F.
- Duplicate verification uses two submissions of an ID: the smallest chosen
  count exercising rejection. Half gives one submission and never duplicates.
  Check identical/changed payloads and no extra financial effects; this is not
  a retry limit or tolerance.
- AED 1.000 is the human's trailing-zero example, equal to stored 1.00.
  Half also represents exactly but is not the supplied example. Chosen AED
  1.005 shows an exact midpoint must reject rather than normalize to 1.00;
  half no longer demonstrates that midpoint trap. BHD 1.0000/1.0001 isolate
  zero/nonzero fourth digits, accepted as 1.000 versus rejected. Halving adds
  no useful coverage and is less direct. Validation uses no rounding/tolerance.
- Planned isolated diagnostic: exact six-day AED interest = 0.100 + 0.100 +
  0.260 + 0.186 + 0.186 + 0.186 = 1.018. Round once to 1.02, versus
  sum daily rounded amounts 1.03, difference 0.01. All follow from the task
  and chosen policy; half breaks the arithmetic. The failing diagnostic
  demands the stronger aggregate-rounding property; correct production credit
  remains 1.03. It has not been implemented or observed failing yet.
