# Ambiguities and Decision Rationale

This record owns alternatives, resolutions, and rationale for business-level task ambiguities and rule interpretations. Explicit task requirements remain closed. [TASK](docs/TASK.md) owns the assignment, [SPEC](docs/SPEC.md) behavior, [DESIGN](docs/DESIGN.md) architecture, [NUMBERS](NUMBERS.md) numerical evidence, and [DEVELOPMENT](docs/DEVELOPMENT.md) tooling and API references.

## A1 — Replay order and projections

**Task requirement, closed.** Supplied order takes precedence over booking- or value-date sorting because TASK explicitly orders the stream. See [replay](docs/SPEC.md#history-and-replay).

## A2 — Unmatched settlement

**Task requirement, closed.** Reject rather than force-post the unmatched settlement: criterion 4 requires rejection and TASK supplies no force-post indicator. The behavior is defined in [SPEC](docs/SPEC.md#authorizations-and-settlements).

## A3 — Backdating and reversal

**Task ambiguities.** Reconcile closed days immediately rather than defer correction, so subsequent inputs see corrected history without closing the open day early. Restore affected financial effects, including unjustified fees, rather than reverse principal alone or retain those fees. Interpret criterion 6 as financial restoration rather than a replay-state reset, preserving original records and authorization decisions. Timing and correction contracts belong to [replay](docs/SPEC.md#history-and-replay) and [restoration](docs/SPEC.md#interest-and-financial-restoration).

## A4 — Fee bases and missing BHD tariff

**Task ambiguity — eligibility.** Test before the assessed day's own fee rather than let its existing charge affect eligibility, preventing a fee from sustaining itself after its underlying overdraft is corrected. See [fees](docs/SPEC.md#fees).

**Task ambiguity — missing tariff.** Configure explicit no-fee treatment for BHD rather than the earlier runtime undefined-policy diagnostic proposal. TASK gives no BHD tariff or exchange rate; no-fee is an assumption completing explicit configuration, not a task-supplied tariff. See [configuration](docs/SPEC.md#currency-and-policy-configuration).

## A5 — Interest rounding, representation, and finalization

**Task ambiguity — rounding.** Choose `HALF_EVEN` over `HALF_UP` to reduce midpoint tie bias. See [interest](docs/SPEC.md#interest-and-financial-restoration).

**Task ambiguity — finalization bases.** Include capitalization in fee eligibility rather than test only the pre-capitalization balance, because the brief refers to closing ledger balance. Exclude capitalization from its own interest base and finalize after the complete stream so the late credit is included. See [finalization](docs/SPEC.md#day-6-finalization).

## A6 — Hold lifecycle

**Task ambiguity.** Treat E5 as final settlement rather than partial capture retaining a residual hold: TASK provides neither a partial-settlement indicator nor further captures. Broader hold workflows remain unspecified; see [settlement](docs/SPEC.md#authorizations-and-settlements) and [deferred scope](docs/SPEC.md#verification-and-deferred-work).

## A7 — Observations and report interpretation

**Task ambiguity — criterion 1.** Use a fee-free prefix calculation rather than a literal posted snapshot to give its qualification precise meaning without suppressing actual fees. Observation definitions belong to SPEC; calculations belong to [NUMBERS](NUMBERS.md#checkpoint-calculations).

## A8 — Currency precision and instalments

**Task ambiguity — allocation.** Put the residual in the last instalment rather than distribute it earlier, using final-instalment balancing as the convention. See [instalments](docs/SPEC.md#instalments).

**Task ambiguity — input precision.** Reject nonzero excess digits rather than round principal, preserving submitted monetary meaning. Redundant trailing zeros remain acceptable because they do not change representability. See [validation](docs/SPEC.md#monetary-validation-and-errors).

## A9 — Duplicate IDs and errors

**Task ambiguity — input handling.** Reject every repeated input ID rather than provide an identical-payload retry protocol with conflict rejection. This favors a small, predictable implementation; retry support is outside the assignment. See [validation/errors](docs/SPEC.md#monetary-validation-and-errors).
