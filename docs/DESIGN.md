# Ledger architecture

This document defines package boundaries, data ownership, and processing
mechanisms for the Go ledger. [SPEC](SPEC.md) owns business behavior;
[AMBIGUITIES](../AMBIGUITIES.md) records decisions and rationale.
Implementation status and sequencing belong to the
[implementation plan](IMPLEMENTATION_PLAN.md); tooling and test organization
belong to [DEVELOPMENT](DEVELOPMENT.md).

## Packages and dependencies

Production code uses the Go standard library. Composition is in `cmd/replay/main.go`.
The table lists allowed imports within this module.

| Package | Responsibility | Allowed internal imports |
| --- | --- | --- |
| `internal/domain` | Shared models, currency and money values, exact parsing/formatting, checked arithmetic, ratio rounding. | None. |
| `internal/ledger` | Input decisions, balance/state projections, interest/allocation policy, reconciliation, closing, and the repository interface. | `domain` |
| `internal/repository/memory` | Append-only storage, detached history, record-key validation, atomic batches. | `domain` |
| `internal/report` | Pure report projection and formatting from detached history and read-only ledger queries/projections. | `domain`, `ledger` |
| `internal/scenario` | Canonical inputs and explicit close sequencing against an injected core. | `domain`, `ledger` |
| `cmd/replay` | Construct storage, currencies/accounts/policies, and core; run scenario; supply finalized data to report and print output. | `domain`, `ledger`, `repository/memory`, `report`, `scenario` |

Imports point toward `domain`; no internal package imports the executable.
Ledger does not import its storage adapter, scenario, or report. Memory
implements the ledger-owned interface structurally through shared domain types
without importing ledger; composition checks conformance. Scenario constructs no
repository and produces no report. Report constructs no core or repository,
runs no scenario, and changes no financial state. Ordinary functions and small
structs keep this graph acyclic without dependency injection tooling.

## Domain models and ownership

| Model | Contents and purpose |
| --- | --- |
| Account / currency | Identity, validated code/scale descriptor, opening balance; concrete definitions are injected at composition. |
| Input event | Original ID, kind, account, raw amount text, booking/value days, authorization/reversal reference, and supplied order. |
| Input attempt / outcome | Submission payload, accepted/declined/rejected result, and diagnostic; duplicate attempts remain separate from the original event/outcome. |
| Monetary entry | Signed amount, account, booking/value days, kind, source/correction cause, original-entry or fee-obligation links, and instalment position where applicable. |
| Authorization transition | Recorded approval, decline, or final settlement, linked to the input and original hold. |
| Diagnostic | Reason, account, observation day, input/correction cause, and assessed day where applicable; decline reasons link to their outcome. |
| Day close | Account/day checkpoint identifying historical days eligible for reconciliation. |
| Rebuildable state | Balances, active holds/current authorization lookup, fee net effects, rounded unpaid accrual per account/day, closed-day tracking. |

Shared history records, batches, and snapshots belong to `domain`; disposable
business projections belong to `ledger`; presentation rows belong to `report`.
History is authoritative. Projection rebuilds retain recorded authorization
decisions rather than reevaluating them. Stored slices/maps remain private;
defensive copies on submission and read-back, including nested mutable data,
prevent caller mutation of history. Money and currency have private value fields
and require no reference ownership management.

[Domain records](../internal/domain/records.go) use a keyed envelope with exactly
one payload pointer. Payload structs contain only value fields; `Record.Clone`
detaches their pointers. Adding mutable payload fields requires extending that
copy operation. Batch and history containers retain a single total record order
across payload types. API fields and link semantics are documented in code.

## History projections

[Ledger projections](../internal/ledger/projection.go) own a cloned record prefix
and private derived indexes. Balance scans retain append order while selecting
by value date; authorization indexes retain recorded transitions. Closed-day
accruals are recalculated from posted balances rather than persisted as entries.
Rebuilding returns no partial projection on arithmetic failure; individual
balance/availability queries also propagate checked arithmetic errors. Projection
APIs and exclusion options are documented in code. Projections have no repository
dependency and cannot append records; submission and reconciliation own appends.

## Money representation

Money pairs signed `int64` minor units with an immutable currency descriptor.
Currency identity includes code and decimal scale. Money and currency have invalid
Go zero values. Descriptor validation and compatibility follow
[monetary validation](SPEC.md#monetary-validation-and-errors), with precision
bounds justified in [NUMBERS](../NUMBERS.md#configurable-currency-verification).
The scale limit bounds decimal unit factors and formatting padding while retaining
the full signed minor-unit range.
Domain and ledger contain no currency-code constants, switches, or defaults.

Parse decimal text exactly into minor units, retaining the original text in the
input record. Format directly from integers at the descriptor's scale, including
the minimum signed value without negating it. Parsing, scaling, addition,
subtraction, negation, and accumulation check overflow; subtraction checks its
operands directly instead of negating the subtrahend. Accumulation follows caller
order and requires every intermediate to fit.

Interest uses integer quotient/remainder division by the
[exact rate denominator](../NUMBERS.md#constants-and-sources); remainder comparison
and quotient parity implement HALF_EVEN without a potentially overflowing
balance-times-rate intermediate. Shared signed integer ratio rounding belongs to
the standalone [domain rounding function](../internal/domain/rounding.go);
ledger daily interest owns the rate and positive-base policy. The rounding
primitive is independent of currencies and can serve other calculated amounts.
Instalment allocation uses integer quotient and
residual under the [allocation rule](SPEC.md#instalments). No binary floating
point, decimal context, or tolerance is used.

The [currency](../internal/domain/currency.go) and
[money](../internal/domain/money.go) code own API signatures and error sentinels;
[A9](../AMBIGUITIES.md#a9--currency-precision-and-instalments) records the parsing
trade-off. Callers use `errors.Is` to distinguish input
range/precision errors from derived arithmetic overflow.

## Configuration validation

The [core constructor](../internal/ledger/core.go) receives a repository, accounts,
and explicit per-currency charges as `map[domain.Currency]domain.Money`. It validates
descriptors, one consistent scale per code, policy coverage for every account currency, and
charge currency/nonnegativity before processing inputs or appending history.
Membership is checked separately from value so an absent entry cannot become an
implicit zero charge. Accounts and policies are copied into detached, core-owned
configuration; callers cannot change replay behavior through retained containers.
Account records are appended in supplied order in one initialization batch after
validation. Initialization uses a fresh repository; core configuration owns
account identity and policy lookup, while history owns financial state.
The [configuration helpers](../internal/ledger/configuration.go) first validate
distinct currency descriptors collected from accounts and policy keys, including
code/scale consistency. Separate account and policy passes then check and copy
their configuration. Money owns descriptor compatibility; policy coverage and
nonnegative charges remain ledger rules.

`cmd/replay` supplies concrete descriptors, accounts, and policies, then passes
configured descriptors to scenario. Fixtures supply their own configuration.
Canonical values and the explicit no-fee assumption are owned by
[SPEC configuration](SPEC.md#currency-and-policy-configuration) and
[NUMBERS](../NUMBERS.md#constants-and-sources).

## Repository boundary

Ledger owns a small [interface](../internal/ledger/repository.go) with `History()`
and `Append(batch)` using domain types. The
[memory adapter](../internal/repository/memory/repository.go) stores an append-only
slice and a record-key index. History scans suffice without storage-specific
queries or persisted aggregates.
Storage enforces structural integrity without fee, interest, or approval policy.

`History()` returns an ordered, detached snapshot. `Append` copies submitted
records and validates nonempty, globally unique record keys, including conflicts
within the proposed batch, and single-payload envelopes before committing the
entire batch or none of it. Empty batches append nothing. Storage does not
validate business fields, reference targets, or supplied event-ID uniqueness.
The core computes the complete batch before append and refreshes affected
projections only after a
successful append. Business rejections record attempts/outcomes/diagnostics
without monetary or hold effects; unexpected failures propagate as technical
errors under [SPEC](SPEC.md#monetary-validation-and-errors).

Internal record keys differ from supplied event IDs. Each attempt has a distinct
sequence identity, allowing duplicate-ID rejections to retain their payloads.
Deterministic keys derive from source event/instalment position for postings,
original entry for reversal, account/assessed day for a fee obligation,
obligation/correction cause for adjustments, and account/window for
capitalization. Required-net comparison supplies reconciliation idempotence;
key uniqueness is a final structural guard.

Atomicity applies to each appended batch. A submission and its historical
reconciliation may span multiple batches; a later technical failure does not
roll back earlier commits. Whole-operation rollback and recovery are outside
the contract.

## Processing and reconciliation

Scenario submits inputs in supplied order and invokes explicit closes from
[SPEC history and replay](SPEC.md#history-and-replay). Booking-date changes do
not trigger closes. Ledger validates identity and amounts, computes the outcome,
and appends the corresponding input, outcome, monetary, and authorization
records according to [SPEC](SPEC.md).

[Ordinary submission](../internal/ledger/submission.go) scans stored attempts to
reject reused event IDs before dispatch. Committed attempt count supplies fresh
sequential identities independent of supplied IDs/order. A proposed posting
history is rebuilt and checked at every affected value-date checkpoint before
append; this catches overflow at later postings as well as the input's own day.
No candidate projection becomes authoritative before storage commits it.

Financial changes affecting closed history trigger chronological reconciliation
of affected account/days before the next input. Ledger projects the value-dated
balance, excludes the assessed day's own net fee from eligibility, compares the
configured required charge with recorded net effects, and appends a linked
difference. Earlier corrections carry into later days. It then replaces unpaid
daily accrual using the corrected post-fee balance. Unpaid accrual remains
derived state without monetary entries or hold effects. Close checkpoints and
event-specific correction ranges belong to SPEC; numerical oracles belong to
[NUMBERS](../NUMBERS.md#checkpoint-calculations).

Finalization follows the full stream and historical reconciliation. Compute
consistent fee, rounded current-day accrual, and capitalization before appending
final postings under the [Day 6 equations](SPEC.md#day-6-finalization). A direct
branch suffices: a negative pre-fee/pre-capitalization base yields zero current-day
accrual, with prior accruals determining fee eligibility; a nonnegative base
requires no fee and supplies the interest base. Capitalization is excluded from
its own interest base. The corrected daily accrual sum becomes one credit per
account/window. Unchanged repeated reconciliation or finalization appends nothing;
no iterative solver or provisional fee is needed.

After finalization, main supplies detached history and read-only ledger
projections to report. Report reconstructs recorded authorization lifecycle and
formats rows in supplied day/account order under
[reporting and observations](SPEC.md#reporting-and-observations).

## Technical limitations

The runner is sequential and storage is in memory, without persistence,
concurrent writers, or distributed transactions. Fixed-width amounts and every
intermediate must fit the [int64 bounds](../NUMBERS.md#int64-representation), even
when later terms could cancel an overflowing sum. Out-of-range inputs reject
without financial effects; derived overflow stops replay before the dependent
batch is appended. Earlier committed batches remain subject to the atomicity
boundary above. There is no currency conversion or input retry protocol;
duplicate-input and deferred business behavior are defined in
[SPEC](SPEC.md#monetary-validation-and-errors) and its
[scope boundary](SPEC.md#verification-and-deferred-work).

Verification coverage and limitation investigations belong to
[USE_CASES](USE_CASES.md), with independent expectations in
[NUMBERS](../NUMBERS.md). The required failing-test contract is in
[SPEC](SPEC.md#verification-and-deferred-work), its selection decision in
[A11](../AMBIGUITIES.md#a11--required-failing-test), and commands in
[DEVELOPMENT](DEVELOPMENT.md#planned-replay-and-isolated-failing-test).
