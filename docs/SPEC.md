# Ledger specification

Status: business specification resolved for the supplied replay and the
validation cases defined here. No implementation is authorized in this phase.
The source brief is [TASK.md](TASK.md), which remains unchanged. Decisions and
their rationale are in [AMBIGUITIES.md](../AMBIGUITIES.md); criterion verdicts
are in [REJECTED.md](../REJECTED.md); constants are in
[NUMBERS.md](../NUMBERS.md).

## Scope and sources of authority

Build an in-memory ledger core with a runnable replay/report, without a web
layer, persistence, UI, or database. Preserve every supplied event identifier,
amount, account currency, booking day, value date, authorization reference, and
the six-day window. ACC-001 starts at AED 0.00; ACC-002 at BHD 0.000.
Assess acceptance criteria independently against the non-negotiable rules and
arithmetic: some incorrect criteria do not invalidate the correct ones.

Use exact monetary/rate arithmetic, never binary floating point. Store and
format AED to two places and BHD to three. The specified overdraft charge is
AED 25.00, once per account per assessed day; positive closing balances earn
0.04% daily interest. The detailed agreed interpretations follow.

## Immutable history, projections, and storage

Replay E1–E10 in the supplied order independently of their dates. E10 stays
after E9 despite booking Day 5. All input events and monetary ledger records
are append-only. Reversals/corrections append records; originals survive.
Keep all ten inputs, including rejected E6, and retain original authorization
outcomes separately. Rebuild balance and uncapitalized-interest projections
from the affected date without changing immutable history or rerunning old
authorization approval decisions.

Separate business logic from storage through a small repository interface
with an in-memory implementation. It needs read/append access to immutable
history; business logic computes decisions and projections. Keep the design
simple for this workload. Exact API names, language/runtime, and dependencies
are implementation choices to record later, not new business questions.

## Replay and closing checkpoints

These checkpoints consolidate the agreed stream order, booking boundaries,
immediate closed-day correction policy, and delayed Day 6 finalization:

| Day | Initial close checkpoint |
| --- | --- |
| 1 | After E2 |
| 2 | After E3 |
| 3 | After E4 |
| 4 | After E6, retaining its rejection and continuing |
| 5 | After E8, before E9 |
| 6 | After E10, historical reconciliation, and capitalization |

After E7, reconcile only already-closed affected Days 2–4, chronologically,
before E8. Append missing fees/refunds as necessary and rebuild affected
interest. These monetary fee corrections are booked Day 5 and use the
affected days as value dates. E8 sees posted historical corrections;
uncapitalized interest does not affect available balance. Assess Day 5's own
fee and interest only at its closing after E8.

E9 reconciles closed Days 2–5 chronologically before proceeding to E10.
E10 retains booking/value Day 5 and corrects the already-closed BHD Day 5.
Do not reopen earlier authorization decisions or reorder inputs. Finalize
Day 6 only after the entire stream and all historical corrections. Repeating
reconciliation/closing is idempotent rather than another fee or capitalization.

## Authorizations and settlements

Available balance is posted ledger balance minus active holds. Approve a hold
only if availability after the proposed hold is nonnegative. A hold changes
available balance, not ledger balance. For E3/E8, booking day equals value
day: use that day's posted balance as known at the replay checkpoint. E8
includes E7's completed historical corrections and excludes unpaid interest.

E3 approves Auth-A: AED 250.00 − AED 200.00 = AED 50.00 available.
Accept E5 as its final settlement: append only the AED 185.00 Day 4 debit,
release the entire AED 200.00 hold, and mark Auth-A settled. The unused
AED 15.00 becomes available through hold release, without a separate credit.
The human's rationale: the task has no partial-settlement indicator or further
captures. Before E5, ledger/available balances are AED 650.00/450.00; after
E5 both are AED 465.00. Auth-A's remaining hold is zero.

Reject E6 without debiting: Auth-Z has no authorization. Criterion 4 is
consistent with the non-negotiable rules; no force-post exception is specified.
Retain its input/error and continue replay. E8 declines Auth-B; it remains
declined after E9 restores financial balances. It never creates an active
hold, so being unsettled does not require expiry or release. Past approval
and decline decisions remain unchanged after historical rebuilding.

## Daily fees and interest bases

Reconcile chronologically so later days use corrected earlier days. For
Days 1–5:

1. Determine the close including ordinary value-dated monetary entries and
   earlier days' net fees/refunds, but excluding the assessed day's own
   fee/refund records from its fee-eligibility calculation.
2. If that pre-own-fee balance is negative, reconcile the daily fee to the
   required charge; otherwise refund a previously charged fee no longer due.
   Append corrections linked to the original assessment; preserve originals.
3. Calculate interest on the positive post-fee close, zero when nonpositive.

A day's own fee must not make itself eligible after correction of its
underlying overdraft. Excluding it from the trigger does not exclude it from
the reported ledger balance. Prior fees/refunds affect later daily balances,
fee triggers, interest bases, and authorization availability.

Use one assessment identity per account/assessed day. Reconcile required net
fee against net monetary effects already recorded; subsequent corrections
refer to that same identity, even if eligibility changes again. Do not create
a second independent assessment. Repeating an unchanged reconciliation appends
nothing. Reversal identity derives from the reversed entry, refunds from
assessment/correction cause, capitalization from account/window. Exact string
identifiers are implementation details, distinct from input duplicate policy.

## Missing BHD fee policy

The brief supplies no BHD fee amount or exchange rate. For a negative BHD
closing balance under the agreed eligibility calculation, report an undefined
fee-policy error, post no fee, preserve the balance, and continue processing.
Do not invent a BHD amount, convert the AED charge, or silently omit the error.
No charge is not a configured zero-fee tariff; the missing tariff remains an
explicit source ambiguity. The supplied stream never triggers this case.

For Day 6 include capitalization in eligibility: a credit curing the overdraft
prevents the negative-close condition. Identify a policy error by account,
assessed day, and reconciliation cause; repeated unchanged closing must not
fabricate more occurrences. Exact display labels are implementation details.

## Input validation and duplicates

Reject monetary inputs with nonzero digits beyond their currency precision.
Accept redundant trailing zeros when exactly representable: AED 1.000 is
valid AED 1.00; AED 1.005 rejects rather than rounding. BHD 1.0000 is valid
BHD 1.000; BHD 1.0001 rejects. Preserve the original input even when stored
monetary values use the required scale. Exact criterion: amount multiplied
by 10^currency_scale is an integer. Use no tolerance or rounding to pass
validation. HALF_EVEN applies to calculated interest, not invalid principal.

Reject every duplicate input event ID, identical or different payload,
regardless of whether its original outcome succeeded, declined, or rejected.
Retain the original record/outcome and record a duplicate-ID rejection. Do
not add monetary effects, change holds, or rerun triggered corrections.
Continue the replay's normal rejection handling. The human deliberately chose
this simplicity trade-off: small, predictable implementation; retry handling
is outside the assignment's stated requirements. No payload comparison or
successful retry/original-outcome protocol is needed.

Input rejection is distinct from internal idempotence: a second E9 submission
rejects, while repeating correction processing for already-recorded E9 adds
nothing. Fresh-repository replay is deterministic; resubmission to an existing
repository produces duplicate rejections without repeated financial effects.

Covered input/domain rejections retain inputs and diagnostics, leave monetary/
hold state unaffected by the rejected operation, and continue replay. The
BHD policy error preserves the already-valid balance. This is not an instruction
to suppress unexpected implementation failures.

## Daily interest and E9 restoration

For eligible positive post-fee balance C, calculate exact interest C / 2,500
and round HALF_EVEN at the currency scale. At a midpoint choose the even
minor-unit count. Nonpositive bases accrue zero. Keep daily unpaid accruals
as rebuildable derived state per account/day: no daily accrual or adjustment
ledger records. They affect neither monetary ledger nor available balance.
Rebuilding replaces the derived amounts rather than accumulating corrections.

E9 appends a linked inverse of E7: AED 620.00 credit, booked Day 6, value
Day 2. Refund fees no longer justified with three linked AED 25.00 credits
booked Day 6 and value-dated Days 2, 4, 5. Preserve E7 and original assessments.
Rebuild affected daily accruals using restored balances. Principal, refund,
and projection corrections are idempotent.

Interpret criterion 6 as restoration of E7's financial effects after E9 and
its corrections, before Day 6 capitalization. Compare corresponding historical
balances/accruals without E7's financial effects, retaining other events.
This is not a rollback of all replay state. Past authorization decisions
remain unchanged; no declined request becomes approved retrospectively.
TASK.md's original criterion is not rewritten.

## Day 6 capitalization and distinct bases

After E10 and its Day 5 corrections, capitalize once per account the exact
sum of all six corrected, individually rounded daily accruals. Do not instead
round a sum of unrounded interest or discard a remainder.

- Fee eligibility includes capitalization but excludes Day 6's own net fee.
  A capitalization credit curing an overdraft prevents a fee; zero is not negative.
- Interest uses the positive post-fee balance excluding capitalization.
  The credit earns no interest on Day 6.

Derivation: B is the balance before Day 6's own fee and capitalization; S is
corrected rounded interest for Days 1–5; F is Day 6's fee; I is its accrual;
K is its capitalization. Then:

- I = HALF_EVEN(max(B − F, 0) / 2,500, currency scale).
- K = S + I.
- For AED, F = 25.00 if B + K < 0, otherwise 0.00.
- Reported Day 6 closing balance = B + K − F.

Determine consistent final amounts before posting, without a provisional fee
based solely on negative B. There is no circular calculation: S and K are
nonnegative. If B < 0, I = 0 with or without a fee, so K = S and eligibility
uses B + S. If B ≥ 0, no fee can apply and I uses B. For negative BHD
eligibility, use the agreed error/no-charge fallback, not an invented F.

## Reporting and criterion 1 checkpoints

Print only a final restated daily table for both accounts, with amounts at
currency precision. Label balances **restated after processing the entire
event stream E1–E10 and Day 6 finalization**. The Day 6 row includes
capitalization. These are not originally observed closing balances.

Include fee assessments and linked refunds even when net fees are zero;
group by assessed/value day and show booking days and links. As a presentation
convention, show recorded authorization lifecycle through each row's booking
day: Auth-A active Days 2–3 and settled from Day 4; Auth-B declined from Day 5.
Errors/decision reasons appear on their original observation/booking day:
E6 rejection Day 4; E8 decline Day 5. Do not invent renewed errors on later
days. Rebuilding projections uses recorded outcomes, not hypothetical decisions
based on restated balances. Do not print a second original-snapshot table.

Criterion 1 uses input prefix E1–E8, excludes every fee entry, and selects
ordinary monetary entries value-dated on/before Day 2. This human-agreed
fee-free calculation is AED 1,200.00 − 950.00 − 620.00 = AED −370.00.
It does not remove or suppress any actual fee record. Keep three observations
distinct; the original criterion in TASK.md remains unchanged:

| Day 2 observation | AED balance |
| --- | --- |
| Fee-free calculation using inputs through E8 | −370.00 |
| Actual posted balance as known at Day 5 closing | −395.00 |
| Final restated balance after the entire stream and corrections | 250.00 |

The latter two reconcile as −370.00 − 25.00 = −395.00 and
−395.00 + 620.00 + 25.00 = 250.00.

## Scope boundaries and implementation verification

No material business choice remains open for the supplied stream and covered
validation cases. Additional workflows—authorizations with differing booking/
value dates, explicit partial/over-captures, expiry, cancellation, and inactive-ID
settlements—are not exercised and remain unspecified extensions. Do not invent
their business policies or expand the task to implement them. Runtime/dependency
choices, executable tests, and exact commands belong to implementation.

The normal suite must independently verify source fidelity and every expected
row below; chronological backdating/reversal; preserved immutable history and
authorization outcomes; refund links/dates; fee identity/idempotence and a fee
not justifying itself; repeated closing/capitalization; approval boundary and
full hold release; unknown authorization rejection/continuation; BHD policy
errors/continuation; exact input validation and duplicates; residual conservation;
HALF_EVEN ties; and daily interest/capitalization reconciliation. Test the
approved-hold invariant of criterion 5 even though Auth-B declines here.

Planned isolated failing diagnostic against the real capitalization path:
assert equality with the single rounding of the exact six-day interest sum.
For AED, the unrounded sum 1.018 rounds once to 1.02, while the agreed sum
of rounded daily amounts is 1.03. An inline annotation must explain this
stronger aggregate-rounding property fails by AED 0.01 because rounding and
summation do not commute. It reveals a consequence of the chosen design;
it is not a production requirement to replace daily rounding. Exercise the
actual result, not a fabricated failure; preserve the assertion and isolate
it outside normal discovery under a documented exact command. Run it and
honestly report its observed failure after implementation. This is a plan,
not a claim that a failing test exists or has run. Normal tests must pass.
Update README with real setup/replay/test/failing-test commands and record
language/dependencies when built. Preserve intact commit history; no squashing.

## Independent calculations and expected results

These are arithmetic/state-transition expectations, not executed ledger tests.
Temporary observations below are test checkpoints, not extra printed reports.

### After E7 and before E9

| Assessed day | Pre-own-fee balance (AED) | Fee (AED) | Post-fee close (AED) | Rounded interest (AED) |
| --- | --- | --- | --- | --- |
| 2 | −370.00 | 25.00 | −395.00 | 0.00 |
| 3 | 5.00 | 0.00 | 5.00 | 0.00 |
| 4 | −180.00 | 25.00 | −205.00 | 0.00 |
| 5 | −205.00 | 25.00 | −230.00 | 0.00 |

Days 2/4 fees post before E8, booked Day 5; Day 5's fee posts at closing
after E8. Day 3 = 250.00 − 620.00 − 25.00 + 400.00 = 5.00; interest
5.00 / 2,500 = 0.002 rounds to 0.00. E8 sees AED −205.00 available;
after its proposed AED 90.00 hold the result would be AED −295.00, so it
declines without a hold. Assessments on Days 2, 4, 5 total AED 75.00.

E9 and refunds restore −230.00 + 620.00 + 75.00 = AED 465.00 before
capitalization. Restored Days 1–5 balances are AED 250.00, 250.00, 650.00,
465.00, 465.00. Corrected accruals are 0.10, 0.10, 0.26, 0.19, 0.19,
summing to AED 0.84. Days 2–5 move from zero to AED 0.74 accrued interest;
this changes derived state, not a separate AED 0.74 credit.

### Independently calculated final monetary results

Historical balances **restated after E1–E10, all corrections, and Day 6
finalization**. ACC-001 is AED; ACC-002 is BHD. Net fees are zero on every
restated row. AED assessments remain visible with their full linked refunds.

| Day | ACC-001 closing ledger | ACC-001 daily interest | ACC-002 closing ledger | ACC-002 daily interest | ACC-001 authorization states | Recorded error/decision reason |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 250.00 | 0.10 | 0.000 | 0.000 | None | None |
| 2 | 250.00 | 0.10 | 0.000 | 0.000 | Auth-A active, hold 200.00 | None |
| 3 | 650.00 | 0.26 | 0.000 | 0.000 | Auth-A active, hold 200.00 | None |
| 4 | 465.00 | 0.19 | 0.000 | 0.000 | Auth-A settled | E6 rejected: unknown Auth-Z |
| 5 | 465.00 | 0.19 | 10.000 | 0.004 | Auth-A settled; Auth-B declined | E8: insufficient available balance at decision |
| 6 | 466.03 | 0.19 | 10.008 | 0.004 | Auth-A settled; Auth-B declined | None |

ACC-002 has no authorization states or errors on any day; no account has an
active hold at window end. Retain this visible AED assessment/refund history:

| Assessed/value day | Original assessment | Assessment booking day | Linked refund | Refund booking day |
| --- | --- | --- | --- | --- |
| 2 | AED 25.00 | 5 | AED 25.00 | 6 |
| 4 | AED 25.00 | 5 | AED 25.00 | 6 |
| 5 | AED 25.00 | 5 | AED 25.00 | 6 |

Each pair represents two distinct immutable records, not a zero-valued
replacement. BHD has no fee assessment/refund records in this stream.

- AED capitalization = 0.10 + 0.10 + 0.26 + 0.19 + 0.19 + 0.19 = 1.03;
  closing ledger = 465.00 + 1.03 = 466.03. Interest base is 465.00, not
  466.03; fee eligibility base is 466.03, so no Day 6 fee.
- BHD instalments = 3.333 + 3.333 + 3.334 = 10.000. E10 corrects Day 5
  interest from 0.000 to 10.000 / 2,500 = 0.004. Day 6 also earns 0.004;
  capitalization = 0.008; closing ledger/fee eligibility base = 10.008,
  interest base = 10.000. No missing BHD policy error is triggered.
- Three BHD 3.334 instalments would total 10.002, an excess of 0.002.

### Independent boundary expectations

HALF_EVEN midpoint examples, not additional replay events:

| Currency | Positive base | Exact daily interest | Rounded accrual |
| --- | --- | --- | --- |
| AED | 12.50 | 0.005 | 0.00 |
| AED | 37.50 | 0.015 | 0.02 |
| BHD | 1.250 | 0.0005 | 0.000 |
| BHD | 3.750 | 0.0015 | 0.002 |

For illustrative AED Day 6 cases with S = 0.02: B = −0.01 yields no fee
and close 0.01; B = −0.02 yields no fee and close zero; B = −0.03 yields
a 25.00 fee and close −25.01. Day 6 interest is zero in all three because
capitalization is excluded from its interest base.
