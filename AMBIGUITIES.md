# Ambiguities and decision rationale

Supplied-replay decisions are resolved. Behavioral rules live in
[docs/SPEC.md](docs/SPEC.md); numerical evidence and verification examples live
in [NUMBERS.md](NUMBERS.md). Each entry records alternatives, the resolution,
its reason, and the relevant observable checks without restating the rules.

## A1 — Replay order and projections

Alternatives were booking-date sorting, value-date sorting, or supplied order.
Use supplied order and rebuildable derived balances, retaining
immutable history. This respects the provided sequence while permitting historical
correction. Verify order/record preservation under [history and replay](docs/SPEC.md#history-and-replay).

## A2 — Storage boundary

Alternatives were combined storage/business logic, a small repository, or general
infrastructure. Use a small repository with an in-memory implementation
to keep this workload simple. See [scope and storage](docs/SPEC.md#scope-and-storage)
for the boundary exercised by future tests.

Go is the selected language. Use `int64` minor units instead of the initial
`math/big.Int` proposal to reduce representation and ownership complexity.
The [revised proposal](docs/DESIGN.md#language-and-tooling) uses standard-library
tooling and value-type money with checked arithmetic. The technical resolution
is to reject unrepresentable input and surface derived overflow as a technical
failure, never silently wrap it or treat it as a business decline. This adds an
explicit representation limit without changing the supplied scenario's results;
see [bounds and verification](NUMBERS.md#int64-representation). Planned
defensive-copy tests cover stored history's slices/maps.
The [repository proposal](docs/DESIGN.md#repository-boundary) keeps the small
read/append boundary; record types and exact signatures remain implementation
work. P01's money API and toolchain baseline are recorded in
[DESIGN](docs/DESIGN.md#language-and-tooling).

Use separate `internal/domain`, `internal/ledger`,
`internal/repository/memory`, `internal/report`, and `internal/scenario` packages
with wiring in `cmd/replay/main.go`, replacing the root-package layout proposal.
The separation makes ownership and import direction explicit while preserving
the small repository boundary. The [package/import map](docs/DESIGN.md#small-code-layout)
keeps shared types below ledger and storage to avoid cycles. Unit tests stay
beside each package; end-to-end composition tests live separately in
`tests/replay`. Verify import direction and test placement during implementation;
this is a structural decision, with no change to business behavior.

## A3 — Unmatched settlement

Reject E6 under criterion 4: Auth-Z has no authorization, and the brief provides
no force-post indicator. This corrects the earlier specification proposal to
accept an unmatched external payment.
Observe retained input/error, no debit, and continuation under
[authorizations and settlements](docs/SPEC.md#authorizations-and-settlements).

## A4 — Backdating and reversal

Alternatives were deferred versus immediate closed-day reconciliation, retaining
fees versus refunding them, and reversing principal alone versus all affected
financial effects. Use immediate correction of closed history and
financial restoration by appended corrections, preserving original records.
This lets subsequent inputs see corrected history without assessing the open day
early. Refund booking follows the correction-causing event.

Interpret criterion 6 as financial restoration while retaining replay state.
Dates, links, chronology, and identity rules are specified in
[history and replay](docs/SPEC.md#history-and-replay), [fees](docs/SPEC.md#fees),
and [financial restoration](docs/SPEC.md#interest-and-financial-restoration).
Use the [checkpoint calculations](NUMBERS.md#checkpoint-calculations) to verify
restoration and repeated reconciliation without extra effects.

## A5 — Fee bases and missing BHD tariff

Eligibility alternatives included testing before a day's own fee or letting its
existing charge influence eligibility. Use the pre-own-fee test
with earlier net fees carried forward, specifically to prevent a fee sustaining
itself after the underlying overdraft is corrected. See [fees](docs/SPEC.md#fees)
and the associated chronological oracle in NUMBERS.md.

The brief supplies no BHD tariff or conversion rule. The application configuration
now explicitly selects no overdraft fee for BHD. This is an assumption added to
complete the configuration, not a tariff specified by TASK. It replaces the
earlier proposed runtime undefined-policy diagnostic. No fee implementation had
been built under that proposal. An absent policy is a configuration error during
initialization; an explicit zero fee is a valid policy. See
[currency and policy configuration](docs/SPEC.md#currency-and-policy-configuration).

## A6 — Interest rounding, representation, and finalization

Use HALF_EVEN rather than HALF_UP to reduce midpoint tie bias, and use
rebuildable daily state rather than immutable accrual/adjustment records for
simplicity.
These choices preserve monetary history without additional daily ledger entries.
Rounding examples and reconciliation evidence are in
[NUMBERS.md](NUMBERS.md#verification-parameters).

The closing sequence includes capitalization in Day 6 fee eligibility because
the brief refers to closing ledger
balance, but not in its own interest base. Finalization waits for the full stream
so the late credit is included. The [finalization rules](docs/SPEC.md#day-6-finalization)
and [boundary cases](NUMBERS.md#day-6-boundaries) verify this distinction and one
capitalized credit on repeat finalization.

## A7 — Hold lifecycle

E5 could have been interpreted as final settlement or a partial capture leaving
a residual hold. Use final settlement because the task provides no
partial-settlement indicator or further captures. Releasing unused reservation
is not a monetary credit. See [authorizations and settlements](docs/SPEC.md#authorizations-and-settlements)
and [checkpoint calculations](NUMBERS.md#checkpoint-calculations).

Past decisions remain fixed during rebuilding, including
Auth-B's original decline after reversal. Being unsettled does not imply
approval. Differing authorization dates, explicit partial/over-captures, expiry,
cancellation, and inactive-ID settlements remain unspecified outside the supplied
scenario; no broader policy is inferred. Verify approval/release boundaries and
recorded outcomes rather than adding those features.

## A8 — Observations and report interpretation

Use a final restated table instead of original snapshots plus a restated
table to keep output compact while preserving correction records and original
decisions. The close checkpoints and row placement are routine consequences of
the agreed sequence, not new financial policies. See
[reporting and observations](docs/SPEC.md#reporting-and-observations).

For criterion 1, alternatives were a fee-free prefix calculation or a literal
posted snapshot. Use the fee-free calculation to give the qualification precise
meaning while retaining actual fee records. The original criterion remains unchanged.
Verify all distinct observations in [checkpoint calculations](NUMBERS.md#checkpoint-calculations).

## A9 — Currency precision and instalments

Assign the residual to the last instalment, using final-instalment balancing
among conserving alternatives. See
[instalments](docs/SPEC.md#instalments) and [conservation](NUMBERS.md#instalment-conservation).

For input precision, reject excess nonzero digits instead of rounding to preserve
submitted monetary meaning. Trailing zeros are acceptable because they do not
change representability. This is distinct from rounding calculated interest.
Verify the [validation rules](docs/SPEC.md#monetary-validation-and-errors) against
[input examples](NUMBERS.md#verification-parameters).

P01 finalizes the technical decimal-text interface: an optional ASCII `+`/`-`,
one or more integer digits, and an optional decimal point followed by one or more
digits. Leading zeros and signed zero are accepted; formatting normalizes signs
and leading zeros and always prints the currency scale. Alternatives included
accepting separators, exponent notation, whitespace, or incomplete decimal forms;
the simple fixed-point grammar avoids implicit normalization. This is an API
choice, not a new event-validation business rule. Nonzero excess fractional digits
reject; excess zeros are removed without rounding, even at the signed endpoints.
Raw input text belongs to the later immutable input record (P03/P05), not `Money`.

Money has private value fields and a validating constructor; its zero Go value
has no currency and operations reject it. `Format` returns an error for invalid
currency rather than silently assuming AED. Checked subtraction operates directly
without negating its operand, so representable `minimum − minimum` succeeds.
`Sum` accumulates in caller order from currency-specific zero, rejecting an
overflowing intermediate even if later terms could cancel it. Parsing range and
precision errors have distinct sentinel errors from derived arithmetic overflow,
so later submission handling can distinguish rejections from technical failures.
The adjacent [money tests](internal/domain/money_test.go) exercise these choices
with literal expectations catalogued in [NUMBERS](NUMBERS.md#p01-money-verification).

## A10 — Duplicate IDs and errors

Alternatives were an identical-payload retry protocol with conflict rejection,
or rejecting all repeated IDs. Reject all repeated IDs as a simplicity
trade-off: small, predictable implementation; retry support is outside the
assignment. Internal reconciliation idempotence is a separate property. The
[validation/error rules](docs/SPEC.md#monetary-validation-and-errors) define the
observable distinction and preserve original outcomes.

Continuation for E6 follows from the full-stream requirement. BHD's explicit
no-fee configuration is recorded in A5. Covered rejections preserve diagnostics
without effects
from the rejected operation. Unexpected implementation failures are not business
rejections to suppress. Additional unexercised validation categories remain outside
the selected policy.

## A11 — Required failing test

The aggregate-rounding test proposal was withdrawn because its expected result
contradicted the specified output. Select the required failing test from an
evidenced limitation of the implementation. Test selection and its numerical
expectation remain unresolved; the deliverable remains required
under [verification and deferred work](docs/SPEC.md#verification-and-deferred-work).

## A12 — Lightweight development checks

Use a Makefile and standard Go GitHub Actions for lightweight development checks:
Go's formatting/build/vet/test tools plus standalone Staticcheck and govulncheck.
These keep the toolset smaller than a multi-linter framework or service integration.
Tool versions are pinned in the Makefile and invoked with `go run package@version`,
keeping development-tool dependencies outside the application module. Network
access is needed for tool downloads, module metadata lookups (even on some cached
invocations), and vulnerability database queries.

CI runs one Linux job on current stable Go, with race detection and coverage;
normal tests remain available on the P01 baseline. An old compiler's standard
library can have vulnerabilities even without application dependencies. Therefore
security checks use the selected Go toolchain and report its findings.
Full checks require a current Go installation (the pinned analyzers require at
least Go 1.26). The module's minimum language/runtime baseline remains Go 1.23.1.
Coverage is reported; no minimum percentage is enforced. CI does not enable the
deferred `designlimitation` tag.

The workflow reuses `make ci` on pushes, pull requests, manual dispatch, and a
weekly schedule so new vulnerability records can be checked without code changes.
Actions are pinned to release commit SHAs with read-only repository permissions;
weekly Dependabot updates maintain those action pins. Analyzer versions remain
explicit Makefile pins to review when upgrading. Verification includes executing
the Make targets and checking formatting rejection, workflow syntax, and source
fidelity; hosted CI results can only be observed after a push.

## A13 — Testify unit-test assertions

Testify is the project's unit-test assertion library, replacing the earlier
standard-library-only test convention. Keep Go's `testing` runner and existing
table-driven cases, use `require` for prerequisites and expected error identities,
and `assert` for independent result checks. Equality stays exact and type-sensitive;
no approximate assertions or expected values change. Raw sentinel comparisons
become Testify's `ErrorIs`/`NotErrorIs` checks so wrapped-error behavior is retained.

Testify is a pinned test-only module dependency; production code remains standard
library only. Record its resolved transitive dependencies in NUMBERS and go.sum,
and include go.sum in the CI dependency cache key. Verify normal tests on the
Go 1.23.1 baseline and full checks on the automation toolchain, preserving all
existing literal oracles and coverage.
The existing vulnerability target now includes test files with `-test`, so the
new test-only dependency is part of the scan.

## A14 — Configured currency identity and initialization

Represent currency as an immutable validated code/decimal-scale value rather
than a built-in currency enumeration. Code syntax is exactly three uppercase
ASCII letters; this is syntactic validation, not a registry lookup. Scale is an
integer from zero through 18 inclusive, the largest exponent whose decimal unit
factor fits signed int64. This technical precision limit also bounds formatting
padding; it does not reduce the signed minor-unit amount range. Money operations
require equal codes and scales. Separately constructed equal descriptors are
compatible; matching codes with different scales are incompatible.

Concrete currencies, accounts, and overdraft policies belong in cmd/replay's
composition when implemented. The core contains no currency-code constants,
currency-specific switches, default currencies, or default fee policies. Initialize
with a policy for every account currency, rejecting absent entries, invalid
descriptors, inconsistent scales for a code, negative charges, or charge-currency
mismatches before processing or appending history. A configured zero monetary
charge explicitly means no fee; a missing entry never means zero.

The replay configuration will supply AED at scale two with a 25.00 charge, and
BHD at scale three with an explicit zero charge under A5's assumption. Canonical
inputs and numerical results are unchanged because the supplied BHD account
never becomes negative. Currency primitives and compatibility are tested now.
Account descriptor records follow in P03, initialization checks in P05, configured
fee behavior in P08/P10, and application wiring in P12.
