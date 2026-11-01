# Ledger specification

Business behavior is resolved for the supplied replay and covered validation
cases. Delivery status belongs to the [implementation plan](IMPLEMENTATION_PLAN.md);
test evidence is mapped in [USE_CASES](USE_CASES.md#coverage-review-and-scope-boundary).

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

Use exact monetary/rate arithmetic, never binary floating point. Currency is a
validated code/decimal-scale value; the core has no built-in currency list or
code-specific switches. Codes contain three uppercase ASCII letters and scales
range from zero through 18. Compatibility requires equal codes and scales.
The application configures AED at two places and BHD at three. Reject nonzero
digits beyond currency precision; accept redundant trailing zeros when the value
is exactly representable.
Preserve the original input when stored monetary values use the required scale.
The exact validation condition is amount × 10^currency_scale being an integer.
No tolerance or rounding may make an invalid principal amount pass validation.
HALF_EVEN is reserved for calculated interest.

CREDIT, DEBIT, AUTHORIZATION, and SETTLEMENT inputs require nonnegative parsed
amounts, including zero. Negative amounts reject with `DiagnosticNegativeAmount`
without financial or hold effects. `InputKind` alone determines direction:
CREDIT posts the amount, DEBIT and SETTLEMENT post its checked negation, and
AUTHORIZATION reserves the amount without a monetary posting. Zero passes amount
validation but remains subject to availability and authorization-reference rules.
See [A10](../AMBIGUITIES.md#a10--ordinary-input-amount-signs) and
[resolved A15](../AMBIGUITIES.md#a15--authorization-and-settlement-amount-signs).

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

A new attempt to reverse a debit already reversed by a different attempt records
a rejected outcome with `DiagnosticAlreadyReversed` and no financial or hold
effects. It returns no technical error on successful rejection recording. Reusing
an event ID still takes duplicate-rejection priority; repeating the same internal
correction cause remains an idempotent no-op. See
[A3](../AMBIGUITIES.md#a3--backdating-and-reversal).

## Fees

Use each account currency's explicitly configured daily overdraft policy. The
replay supplies AED 25.00 and explicitly no BHD fee. Reconcile closed days
chronologically. For Days 1–5:

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

### Currency and policy configuration

The cmd/replay executable supplies the concrete currencies, accounts, and
overdraft policies to the core. Every account currency must have a policy before
initialization succeeds; missing configuration rejects initialization before
processing inputs or appending records. There is no implicit currency or fee
default. Invalid currency descriptors, inconsistent scales for one code, negative
charges, and charge-currency mismatches also reject initialization.

A policy contains a nonnegative monetary charge in the same currency descriptor
as its key/account. A present zero charge explicitly means no fee; an absent
entry is an error. Negative balances under a no-fee policy retain their value
and produce neither a fee entry nor an undefined-policy diagnostic. Initialization
configuration is copied into core-owned state so callers cannot change policies
during replay.

The canonical composition uses AED at scale two with charge 25.00, and BHD at
scale three with an explicit zero charge. BHD no-fee is an assumption replacing
the earlier runtime undefined-policy proposal, since TASK supplies no BHD tariff
or exchange rate. TASK remains unchanged. The canonical BHD balance never becomes
negative, so this assumption preserves every supplied replay result.

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

The supported interpretation of criterion 6 is restoration of E7's financial
effects after E9/corrections, before capitalization. Compare corresponding
historical balances and accruals without E7's financial effects while retaining
other events. A literal reset of record history or past authorizations is refused
in [REJECTED](../REJECTED.md). The original criterion is not rewritten.

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
- F is the configured currency charge if B + K < 0, otherwise zero.
- Reported Day 6 close = B + K − F.

An explicit no-fee policy makes F zero, including for negative BHD eligibility.
Missing policies have already rejected initialization. Consistency and
boundary checks are in [USE_CASES](USE_CASES.md#x08--day-6-capitalization-crosses-the-fee-boundary).

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
rejection, configurable precision/currency compatibility, initialization rejection
for missing policies, explicit no-fee behavior, instalment conservation, and
interest reconciliation.

The assignment requires a real isolated failing test against an evidenced design
limitation. Explain what it reveals inline, preserve its failure, isolate it from
the passing normal suite, document its exact command, and record its observed
nonzero assertion result. Compilation/setup failures, skips, unconditional
failures, and expected-error wrappers do not satisfy this deliverable.
The selected [probe](../tests/replay/designlimitation_test.go) exposes the absence
of whole-submission financial atomicity on technical failure, as described in the
[repository boundary](DESIGN.md#repository-boundary). Its stronger atomicity
expectation is outside the behavior promised above. Coverage and investigation
evidence belong to [USE_CASES](USE_CASES.md); observed results belong to
[WORKLOG](../WORKLOG.md). [DEVELOPMENT](DEVELOPMENT.md) owns runtime/dependency
details, replay, and normal/isolated test commands.
Preserve intact commit history without squashing.

Extra workflows—authorizations with differing booking/value dates, explicit
partial/over-captures, expiry/cancellation, and inactive-ID settlements—remain
unspecified extensions outside this supplied scenario and remain deferred.
The BHD no-fee assumption is explicit application configuration.
