# Ledger architecture

This document defines package boundaries, data ownership, and processing
mechanisms for the Go ledger. [SPEC](SPEC.md) owns business behavior;
[AMBIGUITIES](../AMBIGUITIES.md) records decisions and rationale.
Implementation status and sequencing belong to the
[implementation plan](IMPLEMENTATION_PLAN.md); tooling and test organization
belong to [DEVELOPMENT](DEVELOPMENT.md).

## Packages and dependencies

Production code uses the Go standard library. The table lists allowed imports
within this module. Report, scenario, and executable composition remain planned.
The split follows [A16](../AMBIGUITIES.md#a16--ledger-package-boundaries).

| Package | Responsibility | Allowed internal imports |
| --- | --- | --- |
| `internal/domain` | Shared records, currency/money values, exact parsing/formatting, checked arithmetic, ratio rounding. | None. |
| `internal/interest` | Positive-base daily interest policy and HALF_EVEN accrual calculation. | `domain` |
| `internal/projection` | Read-only history rebuilds, balances, holds, fee nets, accruals, and proposed-batch arithmetic checks. | `domain`, `interest` |
| `internal/posting` | Pure input validation/decisions and batch construction, including authorizations, settlements, reversals, and instalment allocation. | `domain`, `projection` |
| `internal/ledger` | Validated configuration, submission commits, chronological closing/reconciliation, and the consumer-owned repository interface. | `domain`, `posting`, `projection` |
| `internal/repository/memory` | Append-only storage, detached history, record-key validation, atomic batches. | `domain` |
| `internal/report` (planned) | Pure report projection and formatting from detached history and read-only projections. | `domain`, `projection` |
| `internal/scenario` (planned) | Canonical inputs and explicit close sequencing against an injected core. | `domain`, `ledger` |
| `cmd/replay` (planned) | Construct storage/configuration/core, run scenario, and supply finalized history/projections to report. | `domain`, `ledger`, `projection`, `repository/memory`, `report`, `scenario` |

Imports point toward `domain`; the graph is acyclic. Ledger owns all workflow
commits and does not import its storage adapter. Posting, projection, and interest
have no repository dependency and cannot append history. Memory implements the
ledger-owned interface structurally through domain types without importing ledger;
composition checks conformance. Scenario constructs no repository or report.
Report constructs no core or repository and changes no financial state. Ordinary
functions and small structs provide these boundaries without new interfaces or
dependency injection tooling.

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
business projections belong to `projection`; presentation rows belong to `report`.
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

[History projections](../internal/projection/projection.go) own a cloned record prefix
and private derived indexes. Balance scans retain append order while selecting
by value date; authorization indexes retain recorded transitions. Closed-day
accruals are recalculated from posted balances rather than persisted as entries.
Rebuilding returns no partial projection on arithmetic failure; individual
balance/availability queries also propagate checked arithmetic errors. Projection
APIs and exclusion options are documented in code. Projections have no repository
dependency and cannot append records; ledger submission and reconciliation own appends.
[Proposed-batch validation](../internal/projection/validation.go) checks a copied
history plus the candidate records at each affected value-date checkpoint. It
returns only an error and preserves the caller's history and backing slice.

## Money representation

Money pairs signed `int64` minor units with an immutable currency descriptor.
Currency identity includes code and decimal scale. Money and currency have invalid
Go zero values. Descriptor validation and compatibility follow
[monetary validation](SPEC.md#monetary-validation-and-errors), with precision
bounds justified in [NUMBERS](../NUMBERS.md#configurable-currency-verification).
The scale limit bounds decimal unit factors and formatting padding while retaining
the full signed minor-unit range.
Domain and business packages contain no currency-code constants, switches, or defaults.

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
[Daily interest](../internal/interest/interest.go) owns the rate and positive-base
policy. The rounding primitive is independent of currencies and can serve other calculated amounts.
[Posting allocation](../internal/posting/instalments.go) is private to its caller
and uses integer quotient and residual under the [allocation rule](SPEC.md#instalments).
No binary floating point, decimal context, or tolerance is used.

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
account identity and policy lookup for closing, while history owns financial state.
Posting receives only the selected configured account value from the core;
it receives no configuration map or storage handle.
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
not trigger closes. Posting validates identity and amounts and computes the
outcome; ledger appends the corresponding input, outcome, monetary, and
authorization records according to [SPEC](SPEC.md).

[Posting preparation](../internal/posting/submission.go) scans stored attempts to
reject reused event IDs before dispatch. Committed attempt count supplies fresh
sequential identities independent of supplied IDs/order. It validates accounts,
currencies, and amounts, then builds an entire batch without modifying history.
Proposed-history checks catch overflow at later value dates as well as the input's
own day. Early business rejections return their diagnostic batch; technical
failures return no batch or outcome.

[Ledger submission](../internal/ledger/submission.go) supplies committed history and the configured
account value to posting preparation and appends the returned batch once. A
financial batch then triggers closed-day reconciliation for its account from its
posted value day, using the posting's booking day and the committed attempt key
as correction cause. Supported submissions post on one account/value day, so
instalment batches trigger one reconciliation. Batches without monetary entries,
including approvals, declines, and rejections, trigger no reconciliation. Open
days remain open until an explicit close. A successful outcome returns only after
all triggered corrections succeed; technical errors return a zero outcome even
when the input or earlier correction batches have committed. Error sentinels and
diagnostic codes for input processing belong to posting; observation errors and
query types belong to projection. Callers import those owners directly.

[Authorization handling](../internal/posting/authorization.go) computes decisions
from the committed projection and builds transitions without changing it.
Settlement batches contain one debit and a final transition linked to the
original approval. Declines retain the requested amount and a linked diagnostic,
while only approved transitions contribute active holds. Unsupported extensions
return technical errors without appending; this includes missing/reused
authorization IDs, authorizations with differing dates, inactive-ID settlements,
and over-captures. Shared submission validation enforces the amount-sign rule in
[SPEC](SPEC.md#monetary-validation-and-errors) before building transitions or postings.

[Principal correction](../internal/posting/corrections.go) resolves the first
attempt for the referenced event to its stored debit. The inverse uses checked
negation, retains the original value day, and links both the original entry and
the reversal attempt. Its key derives from the original entry; repeating the
same correction cause returns no records before any append. Input resubmission
still rejects through the shared duplicate check. Supported reversal inputs
reference a single ordinary debit on the same account/currency/value day;
missing or other targets return ErrUnsupportedInput without records. A different
cause for an already reversed debit yields a business error that the reversal
batch builder converts to the shared attempt/outcome/diagnostic rejection batch.

[Instalment posting](../internal/posting/instalments.go) uses its private allocation
function and constructs ordered credits keyed by source event and one-based
position. All parts retain the input dates, currency, count, and attempt link,
and share the submission's single validated atomic batch. Positive credit counts
request allocation; zero retains ordinary posting. Negative counts and instalment
requests for other kinds remain unsupported. Reversal account/currency validation
is shared with amount parsing; its principal comes from the original entry rather
than a separately parsed input amount.

[Explicit closing/reconciliation](../internal/ledger/closing.go) builds one batch
per account/day, combining a new close checkpoint with any fee effect. It reads
committed history before each day so earlier fee corrections carry forward.
An original assessment key derives from account/day; later refund/adjustment keys
derive from that obligation and correction cause and link to the original
assessment. Required-net equality skips append entirely. Proposed batches use
the same projection and affected-value-date checks as posting preparation.
Closed-day membership uses the projection's public query; closing has no access
to its private indexes. Projection rebuilding derives corrected daily accruals without stored accrual records.
API parameters, supported days, and validation are documented in code.

The replay owns its close schedule and capitalization checkpoint. Ordinary
closing accepts a supplied positive day without a fixed replay-window cutoff;
the caller chooses ordinary closing or capitalization-aware finalization.
The canonical schedule belongs to [SPEC](SPEC.md#history-and-replay) and will be
supplied by scenario composition, not encoded as a day limit in the core.

Financial changes affecting closed history require chronological reconciliation
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

After finalization, main supplies detached history and read-only history
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
