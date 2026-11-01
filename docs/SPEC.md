# Ledger specification

Status: business behavior resolved for the supplied replay and covered validation
cases. The human authorized P01 money primitives; ledger/replay implementation
remains pending later plan tasks.

[TASK.md](TASK.md) preserves the original assignment and canonical input stream.
This file defines behavior; [AMBIGUITIES.md](../AMBIGUITIES.md) records decision
rationale; [NUMBERS.md](../NUMBERS.md) owns constants, calculations, and expected
results; [REJECTED.md](../REJECTED.md) assesses the acceptance criteria.

## Scope and storage

Build an in-memory ledger core with a runnable replay/report, without a web
layer, persistence, UI, or database. Preserve every supplied event ID, amount,
account currency, booking day, value date, authorization reference, and the
six-day window. Both accounts start at zero in their specified currencies.

Separate business logic from storage through a small repository interface with
an in-memory implementation. The repository supplies read/append access to
immutable history; business logic computes decisions and projections. Exact API
names, language/runtime, and dependencies are implementation choices to record
when selected. Keep the design simple for this workload.

## History and replay

Replay E1–E10 in supplied order independently of their dates, including E10
after E9. Input events and monetary records are append-only; corrections append
records and originals survive. Keep rejected inputs and original authorization
outcomes separately from posted monetary records. Rebuild balances and unpaid
interest from an affected date without rerunning past approval decisions.

| Day | Initial close checkpoint |
| --- | --- |
| 1 | After E2 |
| 2 | After E3 |
| 3 | After E4 |
| 4 | After E6 and its rejection |
| 5 | After E8, before E9 |
| 6 | After E10, historical reconciliation, and capitalization |

After E7, reconcile only already-closed Days 2–4 chronologically before E8.
Append required fee corrections booked Day 5 with the affected historical
value dates; rebuild affected interest. Assess Day 5's own fee and interest
only at closing after E8.

E9 reconciles closed Days 2–5 before E10. E10 retains booking/value Day 5
and corrects the closed BHD Day 5. Finalize Day 6 after the entire stream and
all historical corrections. Repeating a reconciliation or close is idempotent.

## Authorizations and settlements

Available balance is posted ledger balance minus active holds. Approve a hold
only when availability after the proposed hold remains nonnegative. Holds
change availability, never ledger balance. For E3/E8, booking and value days
coincide: use that day's balance known at the event's replay checkpoint. E8
includes E7's posted corrections and excludes uncapitalized interest.

E3 approves Auth-A. Accept E5 as its final settlement: debit only the settlement
amount, release the entire original hold, and mark Auth-A settled. The unused
hold becomes available without a separate credit.

Reject E6 without debit because Auth-Z has no authorization. Retain its error
and continue replay. Auth-B declines at E8, creates no hold, and stays declined
after E9 restores financial balances. It has no hold to expire or release.
Past authorization decisions remain unchanged during historical rebuilding.

## Monetary validation and errors

Use exact monetary/rate arithmetic, never binary floating point. Store and
format AED to two places and BHD to three. Reject nonzero digits beyond currency
precision; accept redundant trailing zeros when the value is exactly representable.
Preserve the original input when stored monetary values use the required scale.
The exact validation condition is amount × 10^currency_scale being an integer.
No tolerance or rounding may make an invalid principal amount pass validation.
HALF_EVEN is reserved for calculated interest.

Reject every duplicate input ID, regardless of payload equality or original
success, decline, or rejection. Preserve original input/outcome; record the
rejection and add no monetary, hold, or event-triggered correction effects.
No successful-retry or original-outcome retry protocol is provided.

Covered input/domain rejections retain diagnostics, leave financial/hold state
unaffected by the rejected operation, and continue replay. Unexpected implementation
failures must not be suppressed as business rejections. Internal correction
idempotence remains distinct from input rejection: a second E9 input rejects;
repeating correction processing for recorded E9 adds nothing. Fresh-repository
replay is deterministic; resubmission to a populated repository rejects duplicates.

## Fees

The specified daily charge is AED 25.00 per account/assessed day. Reconcile
closed days chronologically. For Days 1–5:

1. Calculate the close including ordinary value-dated monetary entries and
   earlier days' net fees/refunds, excluding the assessed day's own fee/refund
   records from its eligibility test.
2. Reconcile that day's fee to the required charge when this balance is negative,
   otherwise refund an existing fee no longer due.
3. Calculate interest on the resulting positive post-fee balance, zero otherwise.

A day's own fee cannot make itself eligible after correction of its underlying
overdraft. Fee exclusions apply only to eligibility; reported ledger balances
include posted fees/refunds. These records carry into later fee/interest bases
and authorization availability.

Use one assessment identity per account/assessed day. Compare required net fee
with recorded net effects; subsequent corrections stay linked to that obligation,
even if eligibility changes again. Never create a second independent assessment.
An unchanged reconciliation appends nothing. Reversal identity derives from the
original entry, refunds from assessment/correction cause, capitalization from
account/window. Serialized identifiers are implementation details.

### Missing BHD fee policy

No BHD tariff or exchange rate is supplied. A negative BHD eligibility balance
produces an undefined fee-policy error, no fee, a preserved balance, and continued
processing. Do not invent a currency amount/conversion or treat the missing tariff
as a zero-fee policy. Day 6 uses the capitalization-inclusive eligibility rule
below. Identify errors by account/assessed day/reconciliation cause; an unchanged
repeated close must not fabricate further occurrences.

## Interest and financial restoration

Daily interest is 0.04% of the eligible positive post-fee closing balance,
rounded HALF_EVEN at currency precision; exact ties choose an even minor-unit
count. Nonpositive balances accrue zero. Keep unpaid daily accruals as rebuildable
derived state per account/day, without daily accrual or adjustment ledger records.
They affect neither ledger nor available balance. Rebuilding replaces values
rather than accumulating adjustments.

E9 appends the linked inverse of E7, booked Day 6 and value-dated Day 2. Append
linked refunds of unjustified fees booked Day 6 with the original fees' value
dates Days 2, 4, 5. Preserve E7 and original assessments. Rebuild affected daily
accruals; principal/refund corrections and projections are idempotent.

Criterion 6 means restoration of E7's financial effects after E9/corrections,
before capitalization. Compare corresponding historical balances and accruals
without E7's financial effects while retaining other events. It does not reset
replay state or change past authorizations. The original criterion is not rewritten.

## Day 6 finalization

After E10 and its Day 5 corrections, capitalize once per account the exact sum
of the six corrected, individually rounded daily accruals. Do not round a sum
of unrounded interest instead or discard any remainder.

Fee eligibility includes capitalization and excludes Day 6's own net fee.
Interest uses the positive post-fee balance excluding capitalization, so the
capitalized credit earns no interest that day. Determine consistent final amounts
before posting; no provisional fee is based solely on a negative pre-capitalization
balance. Zero is not negative.

Let B exclude Day 6's own net fee and capitalization, S be corrected rounded
interest for Days 1–5, F the Day 6 fee, I its accrual, and K its capitalization:

- I = HALF_EVEN(max(B − F, 0) × daily rate, currency scale).
- K = S + I.
- For AED, F is the specified charge if B + K < 0, otherwise zero.
- Reported Day 6 close = B + K − F.

For negative BHD eligibility use the missing-policy fallback. Consistency and
boundary calculations are in [NUMBERS.md](../NUMBERS.md#day-6-boundaries).

## Instalments

Post E10 as three ordered credits, retaining its input and booking/value dates.
Allocate the integer minor-unit quotient to each instalment and add the residual
to the last. Use the [conserving allocation](../NUMBERS.md#instalment-conservation);
do not independently round all three amounts upward.

## Reporting and observations

Print only a final daily table for both accounts, explicitly labeled **restated
after the entire event stream E1–E10 and Day 6 finalization**. Include closing
ledger balances at currency precision, fee assessment/refund history with links
and booking/value days, recorded authorization states, and errors. Day 6 includes
capitalization. Do not print an additional original-snapshot table.

Each row shows recorded authorization lifecycle through its booking day: Auth-A
active Days 2–3, settled from Day 4; Auth-B declined from Day 5. Errors/decision
reasons appear on their original observation day: E6 Day 4, E8 Day 5. Do not
repeat them as new later errors. Group fee history by assessed/value day. Recorded
decisions remain distinct from hypothetical decisions based on restated balances.

Criterion 1 is a fee-free calculation using input prefix E1–E8, excluding all
fee entries and selecting ordinary monetary entries value-dated on/before Day 2.
It is not the posted Day 2 balance known at Day 5 closing and does not suppress
actual fee records. The [checkpoint calculations](../NUMBERS.md#checkpoint-calculations)
distinguish this observation from the posted and final-restated balances.
The [expected replay results](../NUMBERS.md#expected-replay-results) are the numerical
oracle; authorization/error behavior is specified above.

## Verification and deferred work

Normal verification must cover the behaviors above, source-field fidelity, and
all numerical oracles/boundary cases in NUMBERS.md. Include fee self-eligibility,
repeat correction/closing, preserved records/outcomes, full hold release, the
approved-hold invariant even though Auth-B declines, error continuation, duplicate
rejection, representability, instalment conservation, and interest reconciliation.

The assignment still requires a real isolated failing test. Choose it during
implementation, when it can expose a genuine design limitation; no assertion or
expected result is preselected. Explain what it reveals inline, preserve its
failure, isolate it from the passing normal suite, document its exact command,
and report the observed result honestly. Money unit tests exist under P01;
ledger/replay tests and the required failing test remain pending. README records
current runtime/dependencies and test commands; add replay instructions when built.
Preserve intact commit history without squashing.

Extra workflows—authorizations with differing booking/value dates, explicit
partial/over-captures, expiry/cancellation, and inactive-ID settlements—remain
unspecified extensions outside this supplied scenario. Do not invent or implement
their business policies as extra features. The missing BHD tariff remains explicit
with its operational handling resolved.
