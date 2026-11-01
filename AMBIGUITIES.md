# Task ambiguities and resolutions

This record distinguishes task requirements, resolved interpretations, and
assumptions. No question blocks the supplied replay; extensions remain
[deferred](docs/SPEC.md#verification-and-deferred-work). [SPEC](docs/SPEC.md) owns
behavior, [DESIGN](docs/DESIGN.md) implementation choices, and
[NUMBERS](NUMBERS.md) calculations. Decision IDs are retained for existing links.

## A1 — Replay order and projections

**Explicit requirement, closed.** Replay E1–E10 in supplied order, including E10
after E9. Sorting by booking or value date would change the input stream. See
[replay](docs/SPEC.md#history-and-replay).

## A2 — Unmatched settlement

**Resolved policy.** Require a recorded approval before settlement. Reject E6
without a debit because Auth-Z is absent. Force-posting was considered; the task
provides no force-post indicator or separate clearing rule. Criterion 4 is
accepted under this policy; the availability rule alone does not establish it.
See [settlements](docs/SPEC.md#authorizations-and-settlements).

## A3 — Backdating and reversal

**Resolved interpretation.** Reconcile affected closed days before the next input;
deferred correction could give that input stale availability. E9 appends the
principal inverse and refunds fees no longer due, then rebuilds unpaid interest.
Criterion 6 supports financial restoration before capitalization, not deletion of
history or reconsideration of Auth-B. See [replay](docs/SPEC.md#history-and-replay)
and [restoration](docs/SPEC.md#interest-and-financial-restoration).

**Explicit clarification.** A fresh attempt to reverse an already reversed debit
is a business rejection. Repeating internal correction work is idempotent;
duplicate input IDs follow [A9](#a9--duplicate-ids-and-errors).

## A4 — Fee bases and missing BHD tariff

**Resolved interpretation.** Exclude the assessed day's own net fee from its
eligibility test; otherwise the charge could sustain itself after correction.
Earlier days' fees carry forward. See [fees](docs/SPEC.md#fees).

**Assumption — missing BHD tariff.** Configure BHD with zero fee, replacing the
earlier undefined-policy diagnostic proposal. The task supplies neither a BHD
tariff nor an exchange rate. BHD never becomes negative in this replay, so this
assumption does not affect its results. See
[configuration](docs/SPEC.md#currency-and-policy-configuration).

## A5 — Interest rounding, representation, and finalization

**Convention — rounding.** Use `HALF_EVEN` to reduce midpoint tie bias. The task
does not select a rounding mode. See
[interest](docs/SPEC.md#interest-and-financial-restoration).

**Resolved interpretation.** Finalize after E10. Include capitalization in Day 6
fee eligibility because fees use the closing ledger balance; exclude that credit
from its own interest base because it is paid at day's end. See
[finalization](docs/SPEC.md#day-6-finalization).

## A6 — Hold lifecycle

**Resolved interpretation.** E5 is a final settlement: release the full hold and
debit only AED 185.00. The task gives no partial-capture indicator or further
captures. Other hold workflows remain [deferred](docs/SPEC.md#verification-and-deferred-work).

## A7 — Observations and report interpretation

**Resolved interpretation — criterion 1.** Its “before any fee” qualification
means the fee-free E1–E8 calculation. Actual posted history includes the Day 2
fee. See [checkpoint calculations](NUMBERS.md#checkpoint-calculations).

**Report choice.** Print one final restated daily table, retaining fee/refund
history and original authorization decisions. This makes the observation point
explicit without a second snapshot table. See
[reporting](docs/SPEC.md#reporting-and-observations).

## A8 — Currency precision and instalments

**Convention — allocation.** Put the indivisible residual in the last instalment.
Exact equality is impossible at BHD precision; total conservation is required.
See [instalments](docs/SPEC.md#instalments).

**Resolved interpretation — input precision.** Reject nonzero excess digits
instead of rounding principal. Accept redundant trailing zeros because the amount
is exactly representable. See [validation](docs/SPEC.md#monetary-validation-and-errors).

## A9 — Duplicate IDs and errors

**Convention.** Reject every repeated input ID, including identical payloads,
instead of implementing a retry protocol. This keeps input handling small and
predictable. Internal correction idempotence is separate. See
[validation](docs/SPEC.md#monetary-validation-and-errors).

## A10 — Ordinary input amount signs

**Explicit clarification.** CREDIT and DEBIT take nonnegative amounts, including
zero. The event kind determines direction, excluding signed operational amounts.
See [monetary validation](docs/SPEC.md#monetary-validation-and-errors).

## A11 — Submission-Level Atomicity Limitation

This is an implementation limitation, documented with its recovery rule in
[DESIGN](docs/DESIGN.md#repository-boundary).

## A15 — Authorization and settlement amount signs

**Explicit clarification.** AUTHORIZATION and SETTLEMENT follow
[A10](#a10--ordinary-input-amount-signs). This excludes both signed amounts and a
strictly positive requirement. Zero still undergoes availability/reference checks;
a zero final settlement releases the hold. See
[validation](docs/SPEC.md#monetary-validation-and-errors).
