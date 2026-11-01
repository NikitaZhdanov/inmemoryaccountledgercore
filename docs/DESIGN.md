# Minimal technical design

Status: P01 money representation implemented with unit tests; the remaining
ledger design is proposed for later implementation.
The human selected Go. [SPEC.md](SPEC.md) defines business behavior;
[TASK.md](TASK.md) remains the original assignment. This design selects technical
mechanisms without changing agreed decisions, numerical oracles, or
[criterion verdicts](../REJECTED.md).

## Language and tooling

Use Go, a Go module, and the standard library only: `strconv` for integer parsing,
`testing` for tests, and `fmt`/`strings` for text. P01 uses module
`inmemoryaccountledgercore`, baseline Go 1.23.1, observed toolchain
`go1.23.1 darwin/amd64`, and `gofmt`, `go test`, and `go vet`.
No services, dependency framework, or storage setup.

The human's subsequent automation request adds a [Makefile](../Makefile) and
GitHub Actions sharing formatting, build, vet, lint, test, and vulnerability
commands. Standalone Staticcheck and govulncheck are pinned development tools,
invoked outside application module dependencies. CI uses current stable Go;
the core module baseline remains Go 1.23.1. See
[A12](../AMBIGUITIES.md#a12--lightweight-development-checks) for tooling choices
and README for exact commands and runtime/network requirements.

Store money as `int64` minor units paired with currency, in a small value type.
The human preferred this simpler representation over the initial `big.Int`
proposal. Its range comfortably covers this workload and copying money needs no
reference ownership rules. Cross-currency arithmetic is invalid; there is no
conversion facility.

Check parsing, scaling, addition, subtraction, negation, and accumulation for
overflow before using their results. Out-of-range input retains its diagnostic
and is rejected without financial effects; derived arithmetic overflow stops
replay as a technical error rather than silently wrapping or becoming a business
decline. Compute each batch fully before appending it. This is an explicit
representation limit, not a business balance cap; see the
[bounds and checks](../NUMBERS.md#int64-representation).

Parse decimal amount text exactly into minor units, rejecting a nonzero excess
fraction and accepting redundant trailing zeros. Preserve the original text in
the input. For interest, divide positive minor units by the exact rate denominator
from [NUMBERS.md](../NUMBERS.md#constants-and-sources); use quotient/remainder
comparison to implement HALF_EVEN, including an even-quotient tie check. Format
money directly from integers at the currency's scale, including the minimum
signed value without negating it. Division by the rate denominator avoids a
potentially overflowing balance-times-rate intermediate. No decimal context,
tolerance, or intermediate floating point is needed.

## Small code layout

The human selected separate internal packages with composition in `main`.
The table lists allowed production imports within this module; standard-library
imports are omitted.

| Location | Responsibility | Allowed internal imports |
| --- | --- | --- |
| `internal/domain` | Models shared across boundaries, currency and money, exact parsing/formatting, and checked monetary arithmetic. | None. |
| `internal/ledger` | Business logic: input decisions, balance/state projections, interest/allocation policy, reconciliation, closing, and the repository interface. | `domain` |
| `internal/repository/memory` | Append-only in-memory storage, detached history, record-key validation, and atomic batches. | `domain` |
| `internal/report` | Pure report projection and text formatting from detached history and read-only ledger projections/queries. | `domain`, `ledger` |
| `internal/scenario` | Literal canonical inputs and explicit close checkpoints from SPEC; submit/close orchestration against an injected ledger core. | `domain`, `ledger` |
| `cmd/replay` | `main` constructs memory storage, injects it into the ledger, runs the scenario, supplies finalized data to the report, and prints output. | `domain`, `ledger`, `repository/memory`, `report`, `scenario` as needed. |
| `tests/replay` | Separate end-to-end test package composing real components and checking checkpoints, complete replay, and final report. | The internal packages needed by each test; no production package imports tests. |

Imports point toward `domain`; no internal package imports `cmd/replay`.
`ledger` never imports its memory adapter, scenario, or report. Memory storage
implements the ledger-owned interface structurally through shared domain types
without importing `ledger`; composition checks conformance. Scenario creates no
repository and neither builds nor prints reports. Report creates no ledger or
repository and does not run scenarios or change financial state. This direction
keeps the graph acyclic without a service locator or dependency injection tooling.
Keep each package as ordinary functions and small structs.

Put unit tests beside the package they exercise (`*_test.go`); ledger unit tests
can use a small test-local repository double. Keep tests composing ledger and
real storage, including replay/report integration, in `tests/replay` with an
external test package. They wire their own fresh components through the same
boundaries as `main`, rather than importing the executable. When selected, the
genuine failing test belongs beside its package for a local limitation or in
`tests/replay` for a cross-component limitation, isolated by its build tag.

## Domain records and projections

| Model | Contents and purpose |
| --- | --- |
| Account / currency | Supplied account identity, currency, opening balance, and formatting scale. |
| Input event | Original ID, kind, account, raw amount, booking/value days, and authorization/reversal reference; preserve supplied order and fields. |
| Input attempt / outcome | Each submission and its accepted, declined, or rejected result. A duplicate attempt retains its payload/diagnostic without replacing the original event or outcome. |
| Monetary entry | Signed amount, account, booking/value days, kind, source/correction cause, and links to the original entry or fee obligation. Instalment entries also retain their position. |
| Authorization transition | Recorded approval, decline, or final settlement, linked to the input and original hold. Retain original decisions and transitions. |
| Diagnostic | Domain reason, account, observation day, input/correction cause, and assessed day where applicable. Decline reasons remain tied to their outcome. |
| Rebuildable state | Balances, active holds/current authorization lookup, fee net effects, rounded unpaid accrual per account/day, and closed-day tracking. |

Shared history records, batches, and snapshot types live in `domain`; disposable
business projections live in `ledger`, and presentation rows belong to `report`.
History is authoritative. Mutable maps are disposable projections; rebuilding
financial projections never reevaluates authorization decisions. Go structs are
not inherently immutable: keep stored slices/maps private and expose
defensive copies so callers cannot alter recorded history.

## Repository boundary

Define a single small interface in `internal/ledger` with `History()` and
`Append(batch)` operations, using `domain` types for both signatures. Inject the
implementation into the ledger core in `main`; keep construction out of business
logic. History returns an ordered, detached snapshot; append validates record-key
uniqueness before adding the entire batch. `internal/repository/memory` uses an
append-only slice plus a record-key index. It knows no fee, interest, or approval
rules. Scanning history is sufficient for this workload; avoid query interfaces
and persisted aggregates.

The core computes a batch before append, then refreshes affected projections.
Business rejections append the attempt/outcome/diagnostic but no monetary or hold
effects. Do not catch unexpected failures and label them business rejections.
The runner is sequential; distributed transactions and concurrent writers are
outside this design.

Distinguish internal record keys from supplied event IDs: repeated IDs are stored
as separate rejected attempts, never as replacements or successful retries.
Use deterministic keys from domain causes: source event/instalment position for
postings, original entry for reversal, account/assessed day for a fee obligation,
obligation/correction cause for its adjustments, and account/window for
capitalization. Input attempts have distinct sequence identities. Required-net
comparison prevents repeated reconciliation from generating another adjustment;
repository uniqueness is a final structural guard, not the business policy.

## Processing and reconciliation

`internal/ledger` owns the processing and reconciliation rules below, including
daily interest and instalment policy built on `domain` money operations.
The runner in `internal/scenario` uses the injected core to submit E1–E10 exactly
as supplied and invokes the explicit [close checkpoints](SPEC.md#history-and-replay).
Never infer closing from a booking-date change: E10 arrives after E9 while
retaining its earlier dates.

For each input, check duplicate identity, validate exact amounts, and compute
its outcome. Credits/debits append monetary entries. Authorization uses the
event-day ledger known at that checkpoint minus active holds. E5 appends its
settlement debit and a transition releasing the full hold; E6 appends a rejection
without debit. E9 appends E7's linked inverse, retaining E7. E10 appends the
conserving quotient allocation with the residual in its last instalment.

After a financial change affecting closed history, reconcile affected closed
days chronologically. For each account/day, sum known value-dated entries,
exclude that day's own net fee from eligibility, compare required charge with
recorded net fee, and append any linked difference. Earlier days' corrections
carry forward. Negative BHD eligibility produces the agreed missing-policy
diagnostic and no fee, not a guessed tariff. Identify these diagnostics by
account/day/cause so unchanged repeated reconciliation adds nothing.

Then replace that day's unpaid accrual using its post-fee positive balance.
Unpaid interest is not a monetary entry and affects neither ledger nor holds.
E7 reconciles closed Days 2–4 before E8; Day 5 closes after E8. E9 reconciles
Days 2–5, refunding unjustified charges; E10 rebuilds BHD Day 5. Original
authorization outcomes survive each rebuild, including Auth-B's decline.

Day 6 waits until the complete stream. Compute the consistent fee, rounded daily
accrual, and capitalization before appending final postings, using the
[specified equations](SPEC.md#day-6-finalization). A direct branch suffices:
when the pre-fee/pre-capitalization base is negative, current-day accrual is
zero and prior accruals determine fee eligibility; when nonnegative, no fee is
due and interest uses that base. Apply the BHD fallback to negative eligibility.
Exclude capitalization from its own interest base. Post the exact sum of corrected
daily accruals as one credit per account/window; repeated close/finalization
must append nothing. No iterative solver or provisional fee is necessary.

## Reporting

In `internal/report`, build a pure report projection after finalization from
data supplied by `main`, reusing read-only ledger queries where needed. Label it
**restated after the entire event stream E1–E10 and Day 6 finalization**.
Iterate days and accounts in supplied order. Use all final monetary records with
value day through the row's day; show fee charges/refunds grouped by assessed
day, including source links and
booking/value days. Reconstruct recorded authorization lifecycle through each
row's booking day. Place errors and decision reasons on their original observation
day, without inventing later occurrences. Format at the account currency's scale.

This deliberately combines restated financial balances with recorded decisions;
label that distinction. Print only the agreed final table. Prefix observations,
including criterion 1's fee-free calculation, belong in tests rather than a
second snapshot report.

## Verification and implementation handoff

Use table-driven tests with exact expectations independently derived in
[NUMBERS.md](../NUMBERS.md), never expectations calculated by the production
closing functions. Cover:

- Money representability, both HALF_EVEN tie directions/currencies, exact
  formatting, and conserving instalments with the final residual. Exercise
  integer bounds and checked arithmetic, including minimum-value negation and
  formatting, overflowing accumulation, and maximum-base interest calculation.
- Checkpoint arithmetic, full settlement/hold release, approval at the inclusive
  availability boundary, unknown-authorization rejection and continuation, and
  preserved decisions after restatement.
- Backdating, linked reversal/refunds, fee self-eligibility and chronological
  carry-forward, missing BHD policy, and unchanged repeat reconciliation/closing.
- Duplicate IDs across original success/decline/rejection and changed payloads;
  preserved source fields/history and defensive-copy isolation.
- All final daily balances/accruals, fee history, Day 6 boundaries, capitalization
  equality, deterministic report placement, and identical fresh-repository replay.

Planned commands once built: `go run ./cmd/replay`, `go test ./...`, and
`go vet ./...`. Unit tests can run with `go test ./internal/...`; separate
end-to-end tests run with `go test ./tests/replay`. Run the genuine failing test
separately with `go test -tags designlimitation ./...`.
The tagged test must expose an evidenced limitation of the implemented design,
include its inline explanation, and genuinely fail. Its assertion remains deferred
under [A11](../AMBIGUITIES.md#a11--required-failing-test); do not manufacture an
incorrect expected balance or mark it as an expected failure.

At implementation time, publish exact setup/run/output instructions in README,
inventory any additional constants/test parameters, and log actual check results.
Unspecified authorization extensions remain outside scope; no new business policy
is introduced here. Preserve intact commit history without squashing.
