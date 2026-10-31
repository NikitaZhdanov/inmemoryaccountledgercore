# Ambiguities and decisions

Status: supplied-replay business decisions resolved. Implementation work and
unspecified extensions remain distinguished below. The consolidated behavior
and independent expectations are in [docs/SPEC.md](docs/SPEC.md).

## A1 — Replay order and derived projections — agreed

Alternatives: sort by booking date, sort by value date, preserve supplied order.
Human resolution: preserve E1–E10, including E10 after E9; dates do not reorder
input. Events and monetary records remain append-only. Derived balances may
rebuild from an affected value date without erasing immutable history.
Rationale: explicit initial human agreement. Verify source field fidelity,
E9-before-E10 ordering, original record preservation, and restated balances.
Detailed closing/correction policies were subsequently resolved under A4–A8.

## A2 — Storage boundary — agreed

Alternatives: combine storage/business logic, introduce a small repository
boundary, or build general infrastructure. Human resolution: a small repository
interface with an in-memory implementation, simple for this workload. Read/
append immutable history through it; business logic owns decisions/projections.
Rationale: explicit initial agreement. Later tests exercise behavior through
that boundary. Exact method names, language/runtime, and dependencies are
implementation choices to record when selected, not further business questions.

## A3 — Settlement without authorization — criterion 4 and human confirmation

Resolution: reject unknown-authorization settlements without a debit. E6's
Auth-Z is absent; criterion 4 is correct and conflicts with no non-negotiable
rule. The task supplies no force-post indicator. The earlier specification
proposal accepting E6 as an unmatched external payment was unsupported and
withdrawn after human correction. General banking capabilities do not create
an ambiguity in this task. Verify E6's rejection/no debit, retained original
input/error, and continuation to E7–E10. Known inactive-ID settlement behavior
is outside the supplied scenario and is not silently inferred from this rule.

## A4 — Backdated fees, reversal, and correction identity — agreed

Alternatives considered: retain fees after E9 or append refunds; restore only
principal or also fees/interest; defer E7 correction to Day 5 closing or perform
it before E8. Human resolutions:

- E7 immediately reconciles only closed affected Days 2–4, chronologically,
  before E8. Append missing fees/refunds as needed and correct daily interest.
  Fee corrections book Day 5 and use affected historical value dates.
- E8 sees posted corrections, excluding uncapitalized interest. Day 5's own
  fee/interest wait until closing after E8.
- Interpret criterion 6 as restoring E7's financial effects: append the
  AED 620.00 inverse, restore historical balances, refund fees no longer due,
  and correct affected interest. Preserve E7 and original assessments.
- Append linked refunds booked Day 6, with original fee value dates Days
  2, 4, 5. The human confirmed this follows the existing correction policy.
- Evaluate restoration after E9/corrections and before Day 6 capitalization,
  retaining other events. Past authorization decisions remain unchanged.
  This is not rollback to an earlier input checkpoint or deletion of history.
- Corrections are idempotent for principal, refunds, and derived accruals.

Rationale: the human selected financial correction with preserved audit history
and immediate reconciliation of closed history, without premature assessment
of the open day. Verify original records, links, booking/value dates, repeated
corrections, independent historical balances/accruals, and unchanged approvals/
declines. Original criterion 6 in TASK.md remains unchanged.

Routine consequences consolidated at human request: E9 reconciles closed Days
2–5 chronologically before E10; one fee assessment identity per account/day;
linked corrections adjust that same daily obligation if eligibility changes
again. Repeating unchanged reconciliation adds nothing. Reversal identity
derives from the original entry, refund from assessment/correction cause,
capitalization from account/window. Exact serialized IDs are implementation
details. Verify one assessment identity across changes and repeated closing.

## A5 — Fee bases and missing BHD denomination — agreed handling

Trigger alternatives: before a day's own net fee or including its existing
fee. Human resolution: include earlier days' net fees/refunds, exclude the
assessed day's own fee/refund records from its trigger, reconcile its fee,
then accrue interest on the resulting positive post-fee close. Reconcile
chronologically. Day 6 has the additional capitalization-specific rule in A6.
Posted fees/refunds carry into later balances and authorization availability;
uncapitalized interest does not.

Human rationale: a fee must not make itself eligible after the underlying
overdraft is corrected. Verify a corrected positive pre-fee balance whose
existing charge alone would make it negative; refund it rather than retaining
a self-justifying fee. Independently verify Day 3's AED 5.00 close and 0.00
interest after E7, and E8's AED −205.00 availability.

Missing source policy: the charge is AED-denominated, but an account is BHD.
Alternatives: undefined-policy error without charge, or supply a separate BHD
tariff. Human resolution: for a negative BHD close, report an undefined
fee-policy error, post no fee, preserve the balance, and continue. No amount
or exchange rate may be invented. The tariff remains unspecified; operational
handling of its absence is resolved, not a zero-fee tariff. This preserves
source fidelity. Day 6 eligibility includes capitalization; a curing credit
prevents the condition. Verify error, preserved balance, no fee/conversion,
and subsequent processing. This stream never triggers it. Repeated unchanged
closing does not fabricate further errors; identity uses account/assessed day/
reconciliation cause. Exact error labels are implementation details.

## A6 — Interest correction, rounding, and capitalization — agreed

Rounding alternatives: HALF_EVEN versus HALF_UP. Human resolution: HALF_EVEN
for each daily accrual at currency precision; capitalize the exact sum of
corrected rounded daily amounts, not rounding the unrounded sum. No remainder
is discarded. The human selected this after the tie-bias trade-off. Verify
midpoint ties in both directions and currencies and exact reconciliation.

Representation alternatives: rebuildable daily state versus immutable accrual/
adjustment records. Human resolution: rebuildable derived state, consistent
with the simpler projection approach. Recompute affected values from immutable
monetary history; no daily accrual or adjustment records. Only capitalization
credits interest to the ledger. Unpaid interest never affects availability.
Verify repeated rebuild yields the same daily amounts rather than accumulated
adjustments, zero monetary interest before capitalization, and one final credit.

Day 6 human resolution: finalize only after all events including E10 and its
Day 5 corrections. Capitalize once per account using corrected, individually
rounded daily accruals. Exclude the capitalization credit from Day 6 interest,
but include it in fee eligibility while excluding that day's own net fee.
This refines the earlier proposed generic fee-before-capitalization sequence.
Rationale supplied by the human: the fee rule refers to closing ledger balance;
interest should not earn on its own credit.

SPEC derives consistent equations without iteration: negative pre-capitalization/
pre-own-fee B makes Day 6 interest zero regardless of fee, so eligibility uses
B plus prior accruals; nonnegative B prevents a fee and interest uses B. Test
capitalization curing an overdraft, reaching zero, or leaving it negative,
alongside no self-interest and repeat-finalization idempotence. Independent
final balances: AED 466.03 and BHD 10.008.

## A7 — Hold lifecycle and balance checkpoints — supplied scenario resolved

E5 alternatives: final settlement releasing full hold, or partial capture with
a residual hold. Human resolution: accept as final, debit only AED 185.00,
release AED 200.00, mark Auth-A settled. Unused AED 15.00 becomes available
without credit. Rationale: no partial-settlement indicator or further captures.
Criterion 3 is correct. Verify settlement debit, settled state, zero residual
hold, and no credit for the difference; independent arithmetic is in SPEC.

E3/E8 booking and value days coincide: use that day's posted balance known
at the replay checkpoint minus active holds; E8 includes historical corrections.
This consolidates agreed consequences, not a policy for differing authorization
booking/value dates. Historical rebuilds never change past decisions. The human
explicitly confirmed Auth-B declines at E8 and remains declined after E9;
it has no active hold to expire/release. “Never settled” does not imply approved.

Explicit partial captures, over-captures, expiry/cancellation, differing dates,
and inactive-ID settlements are unexercised extensions with unspecified broader
business policies. Do not invent them as extra features or reopen supplied
scenario decisions. Unknown IDs are resolved under A3. Tests cover approval
boundary, hold-only availability effect, full release, and recorded outcomes.

## A8 — Closing, checkpoints, and reporting — resolved

Human agreements: Days 2–4 are closed when E7 arrives; reconcile before E8;
Day 5 closes after E8; evaluate E9 restoration before capitalization; finalize
Day 6 only after E10 and its Day 5 corrections. Routine checkpoint consequences
are after E2, E3, E4, E6, E8 for Days 1–5 and after E10/corrections/capitalization
for Day 6. E10 corrects closed BHD Day 5 without reordering. Tests may observe
these original checkpoints without printing them as a second report.

Reporting alternatives: original snapshots plus restated table, or final
restated table only. Human resolution: final restated table only, retaining
fee assessment/refund records, authorization states, and errors. Explicitly
label balances as restated after the entire stream and Day 6 finalization.
Auth-B stays declined despite restored positive balances. Rationale: compact
output retaining provenance and original decisions. Day 6 includes capitalization.

Criterion 1 alternatives: fee-free E1–E8 calculation or literal posted Day 5
closing snapshot. Human resolution: fee-free Day 2 calculation excluding all
fee entries, AED −370.00. Distinguish actual posted Day 2 known at Day 5 close,
AED −395.00, and final restated Day 2, AED 250.00. This gives “before any
fee” precise meaning without denying posted fees or altering TASK.md. Verify
all three checkpoint meanings and preserve fee records.

Routine reporting convention: each row shows recorded authorization lifecycle
through its booking day (Auth-A active Days 2–3, settled from Day 4; Auth-B
declined from Day 5). Show errors/decision reasons on observation days (E6
Day 4, E8 Day 5), not repeatedly as new later errors. Group fee history by
assessed/value day with booking dates/links. These are presentation consequences,
not new financial policies. Verify all daily/account rows, correction dates,
retained decisions/errors, E10's late arrival, and repeated closing.

## A9 — Precision and instalments — agreed

Exact arithmetic/currency scales are mandated; calculated interest uses A6.
Residual alternatives: first or last instalment receives indivisible BHD 0.001.
Human resolution: last, producing BHD 3.333, 3.333, 3.334. Rationale: the
human chose final-instalment balancing from two conserving alternatives. Three
3.334 credits would total 10.002 and are invalid. Verify ordered allocation,
10.000 conservation, precision, and unchanged E10 fields/order.

Input alternatives: reject nonrepresentable values or round on entry. Human
resolution: reject nonzero digits beyond currency precision; accept trailing
zeros such as AED 1.000 because the value is exactly representable. HALF_EVEN
is for calculated interest, not normalization of invalid amounts. Rationale:
preserve submitted monetary meaning rather than silently changing principal.
Verify representability in both currencies, accepted trailing zeros, rejected
inputs without monetary/hold effects, and unchanged original input. SPEC gives
the exact integer-minor-unit criterion and independent examples.

## A10 — Errors, duplicates, and record types — resolved for covered cases

E6 continues by the full-stream requirement; BHD policy-error continuation was
explicitly selected. Preserve all ten immutable inputs, including rejected E6.
Distinguish inputs, posted monetary records, recorded authorization outcomes,
and rebuildable balance/accrual projections; no rebuild rewrites input/outcomes.

Duplicate alternatives: identical-payload idempotent retries with conflict
rejection, or reject all repeated input IDs. Human resolution: reject duplicates
without additional financial effects. Deliberate human-supplied simplicity
trade-off: small/predictable implementation; retry handling is outside the
assignment's stated requirements. Apply to identical/changed payloads and
originally rejected/declined IDs; preserve original input/outcome, produce a
rejection diagnostic, continue, and add no monetary/hold/correction effects.

Internal reconciliation remains idempotent: another E9 input rejects, but
recomputing its already-recorded correction adds nothing. Fresh-repository replay
is deterministic; existing-repository resubmission rejects duplicates. Verify
identical/conflicting duplicates and monetary/authorization/originally rejected
inputs with unchanged original financial effects and diagnostic output.

Invalid precision rejects under A9. Routine rejection handling records the
diagnostic and continues without effects from that rejected operation. BHD
missing-policy handling preserves an already-valid balance. Do not suppress
unexpected implementation failures as business errors. Additional unexercised
validation categories/banking workflows do not acquire an invented policy.

## A11 — Verification and implementation work — deferred

Record language/runtime/dependencies during implementation. SPEC proposes an
isolated diagnostic against real capitalization: demand aggregate rounding
AED 1.02 although the daily-rounding design produces AED 1.03. It exposes the
independent AED 0.01 difference from noncommuting rounding/summation. Explain
the stronger property/trade-off inline, isolate its command, preserve its
assertion, and honestly report the actual failure once built. Normal tests
must pass with the agreed 1.03 credit. No executable test exists or has run.
No alternative failing-test implementation was built or abandoned.

Final monetary/state expectations and normal verification are in SPEC. Further
business-policy gaps are explicitly unexercised extensions under A7; the BHD
tariff remains unspecified under A5 with its handling fully resolved.
