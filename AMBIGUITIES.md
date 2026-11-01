# Ambiguities and decision rationale

This record owns alternatives, resolutions, and rationale for task ambiguities
and consequential project choices. Entries label that distinction; explicit task
requirements remain closed. [TASK](docs/TASK.md) owns the assignment,
[SPEC](docs/SPEC.md) behavior, [DESIGN](docs/DESIGN.md) architecture,
[NUMBERS](NUMBERS.md) numerical evidence, and [DEVELOPMENT](docs/DEVELOPMENT.md)
tooling and API references. Status belongs to the [plan](docs/IMPLEMENTATION_PLAN.md)
and observed changes/checks to [WORKLOG](WORKLOG.md).

## A1 — Replay order and projections

**Task requirement, closed.** Supplied order takes precedence over booking- or
value-date sorting because TASK explicitly orders the stream. See
[replay](docs/SPEC.md#history-and-replay).

**Project choice.** Rebuildable projections support historical correction while
retaining immutable history; their design is owned by
[DESIGN](docs/DESIGN.md#domain-models-and-ownership).

## A2 — Storage boundary

**Project choices.** Use a small in-memory repository rather than combined
storage/business logic or general infrastructure to keep the workload simple.
Choose checked `int64` value-type money over the initial `big.Int` proposal to
reduce representation and ownership complexity, accepting the documented
[range/intermediate limits](NUMBERS.md#int64-representation).

Use separate internal packages instead of the root-package proposal to make
ownership/import direction explicit and avoid cycles. Contracts belong to
[DESIGN](docs/DESIGN.md#packages-and-dependencies).

## A3 — Unmatched settlement

**Task requirement, closed.** Reject rather than force-post the unmatched
settlement: criterion 4 requires rejection and TASK supplies no force-post
indicator. This supersedes the earlier acceptance proposal; the behavior is in
[SPEC](docs/SPEC.md#authorizations-and-settlements).

## A4 — Backdating and reversal

**Task ambiguities.** Reconcile closed days immediately rather than defer
correction, so subsequent inputs see corrected history without closing the open
day early. Restore affected financial effects, including unjustified fees,
rather than reverse principal alone or retain those fees. Interpret criterion 6
as financial restoration rather than a replay-state reset, preserving original
records and authorization decisions. Timing and correction contracts belong to
[replay](docs/SPEC.md#history-and-replay) and
[restoration](docs/SPEC.md#interest-and-financial-restoration).

## A5 — Fee bases and missing BHD tariff

**Task ambiguity — eligibility.** Test before the assessed day's own fee rather
than let its existing charge affect eligibility, preventing a fee from sustaining
itself after its underlying overdraft is corrected. See [fees](docs/SPEC.md#fees).

**Task ambiguity — missing tariff.** Configure explicit no-fee treatment for BHD
rather than the earlier runtime undefined-policy diagnostic proposal. TASK gives
no BHD tariff or exchange rate; no-fee is an assumption completing explicit
configuration, not a task-supplied tariff. The superseded proposal was never
implemented. See [configuration](docs/SPEC.md#currency-and-policy-configuration).

## A6 — Interest rounding, representation, and finalization

**Task ambiguity — rounding.** Choose HALF_EVEN over HALF_UP to reduce midpoint
tie bias. See [interest](docs/SPEC.md#interest-and-financial-restoration).

**Project choice — accrual representation.** Use rebuildable daily state rather
than immutable accrual/adjustment records for simplicity; see
[models](docs/DESIGN.md#domain-models-and-ownership).

**Task ambiguity — finalization bases.** Include capitalization in fee eligibility
rather than test only the pre-capitalization balance, because the brief refers to
closing ledger balance. Exclude capitalization from its own interest base and
finalize after the complete stream so the late credit is included. See
[finalization](docs/SPEC.md#day-6-finalization).

## A7 — Hold lifecycle

**Task ambiguity.** Treat E5 as final settlement rather than partial capture
retaining a residual hold: TASK provides neither a partial-settlement indicator
nor further captures. Broader hold workflows remain unspecified; see
[settlement](docs/SPEC.md#authorizations-and-settlements) and
[deferred scope](docs/SPEC.md#verification-and-deferred-work).

## A8 — Observations and report interpretation

**Project choice — report.** Print a final restated table instead of original
snapshots plus a restated table, keeping output compact while retaining
correction history and original decisions. See
[reporting](docs/SPEC.md#reporting-and-observations).

**Task ambiguity — criterion 1.** Use a fee-free prefix calculation rather than
a literal posted snapshot to give its qualification precise meaning without
suppressing actual fees. Observation definitions belong to SPEC; calculations
belong to [NUMBERS](NUMBERS.md#checkpoint-calculations).

## A9 — Currency precision and instalments

**Task ambiguity — allocation.** Put the residual in the last instalment rather
than distribute it earlier, using final-instalment balancing as the convention. See
[instalments](docs/SPEC.md#instalments).

**Task ambiguity — input precision.** Reject nonzero excess digits rather than
round principal, preserving submitted monetary meaning. Redundant trailing zeros
remain acceptable because they do not change representability. See
[validation](docs/SPEC.md#monetary-validation-and-errors).

**Project choice — decimal interface.** Use simple ASCII fixed-point text rather
than separators, exponents, whitespace, or incomplete decimal forms, avoiding
implicit normalization. Grammar and value/error contracts belong to the
[currency](internal/domain/currency.go) and [money](internal/domain/money.go) code;
arithmetic mechanisms belong to [DESIGN](docs/DESIGN.md#money-representation).

## A10 — Duplicate IDs and errors

**Project choice.** Reject every repeated input ID rather than provide an
identical-payload retry protocol with conflict rejection. This favors a small,
predictable implementation; retry support is outside the assignment. Internal
correction idempotence is separate from input retries. See
[validation/errors](docs/SPEC.md#monetary-validation-and-errors).

## A11 — Required failing test

**Project choice.** Select the required failing assertion from implementation
evidence rather than preselect an aggregate-rounding expectation. The earlier
proposal contradicted specified output and was withdrawn.

**Unresolved:** which evidenced limitation and numerical expectation to select.
The [assignment deliverable](docs/TASK.md#deliverable-1--repository) remains required;
its contract is in [SPEC](docs/SPEC.md#verification-and-deferred-work), investigation
leads in [USE_CASES](docs/USE_CASES.md#design-test-candidates), and work sequencing
in [P13](docs/IMPLEMENTATION_PLAN.md#p13--investigate-and-preserve-a-genuine-failing-design-test).

## A12 — Lightweight development checks

**Project choice.** Use Make, Go tooling, standalone Staticcheck/govulncheck, and
standard GitHub Actions rather than a multi-linter framework or service
integration, keeping the toolset small. Run security checks on current stable Go
while retaining the module baseline for normal tests, since the selected
compiler's standard library can itself have vulnerabilities. Tool/CI operation
belongs to [DEVELOPMENT](docs/DEVELOPMENT.md); pins and operational parameters
belong to [NUMBERS](NUMBERS.md#development-automation).

## A13 — Testify unit-test assertions

**Project choice.** Use pinned, test-only Testify instead of handwritten
standard-library assertion/error checks. This adopts the project's selected
assertion-library convention; no further comparative rationale is recorded.
Retain Go's runner and exact, type-sensitive expectations. Assertion usage belongs
to [DEVELOPMENT](docs/DEVELOPMENT.md#assertion-convention), dependency versions to
[NUMBERS](NUMBERS.md#test-assertion-dependencies), and the implemented replacement
to [REJECTED](REJECTED.md#abandoned-approaches).

## A14 — Configured currency identity and initialization

**Project choice.** Use immutable validated code/scale descriptors instead of a
built-in currency enumeration, allowing currency definitions at composition.
The selected syntax convention and precision bound are justified in
[NUMBERS](NUMBERS.md#configurable-currency-verification); descriptor contracts
belong to [SPEC](docs/SPEC.md#monetary-validation-and-errors) and the
[currency code](internal/domain/currency.go).

Require explicit policies at initialization rather than infer defaults or defer
missing-policy errors to replay, establishing complete configuration before any
financial effects. See [configuration](docs/SPEC.md#currency-and-policy-configuration)
and [ownership](docs/DESIGN.md#configuration-validation); A5 owns the BHD assumption.
