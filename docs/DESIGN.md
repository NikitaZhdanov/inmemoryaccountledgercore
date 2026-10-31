# Minimal technical design

Status: proposal for implementation; no ledger code or executable tests yet.
The human selected Go. [SPEC.md](SPEC.md) defines business behavior;
[TASK.md](TASK.md) remains the original assignment. This design selects technical
mechanisms without changing agreed decisions, numerical oracles, or
[criterion verdicts](../REJECTED.md).

## Language and tooling

Use Go, a Go module, and the standard library only: `strconv` for integer parsing,
`testing` for tests, and `fmt`/`strings` for the text report. Use
`gofmt`, `go test`, and `go vet`; record the actual Go version and module baseline
when implementation begins. No services, dependency framework, or storage setup.

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

| Location | Responsibility |
| --- | --- |
| Root package `ledger` | Money, domain records, repository interface/in-memory implementation, processing, closing, report projection/formatting; split into files by responsibility. |
| `internal/scenario` | Literal canonical inputs and explicit close checkpoints from SPEC; used by replay and integration tests. |
| `cmd/replay` | Construct a fresh repository, run the supplied scenario, print the final report. |
| Tests beside their packages | Unit and integration tests; a build-tagged file isolates the required genuine failing test when selected. |

Keep these as ordinary functions and small structs. Separate packages for every
domain concept, a generic event bus, and dependency injection tooling would add
little to this in-memory workload.

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

History is authoritative. Mutable maps are disposable projections; rebuilding
financial projections never reevaluates authorization decisions. Go structs are
not inherently immutable: keep stored slices/maps private and expose
defensive copies so callers cannot alter recorded history.

## Repository boundary

Use a single small interface with `History()` and `Append(batch)` operations.
History returns an ordered, detached snapshot; append validates record-key
uniqueness before adding the entire batch. The in-memory implementation uses an
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

The scenario runner submits E1–E10 exactly as supplied and invokes the explicit
[close checkpoints](SPEC.md#history-and-replay). Never infer closing from a
booking-date change: E10 arrives after E9 while retaining its earlier dates.

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

Build a pure report projection after finalization, labeled **restated after the
entire event stream E1–E10 and Day 6 finalization**. Iterate days and accounts in
supplied order. Use all final monetary records with value day through the row's
day; show fee charges/refunds grouped by assessed day, including source links and
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

Planned commands once built: `go run ./cmd/replay`, `go test ./...`,
`go vet ./...`, and separately `go test -tags designlimitation ./...`.
The tagged test must expose an evidenced limitation of the implemented design,
include its inline explanation, and genuinely fail. Its assertion remains deferred
under [A11](../AMBIGUITIES.md#a11--required-failing-test); do not manufacture an
incorrect expected balance or mark it as an expected failure.

At implementation time, publish exact setup/run/output instructions in README,
inventory any additional constants/test parameters, and log actual check results.
Unspecified authorization extensions remain outside scope; no new business policy
is introduced here. Preserve intact commit history without squashing.
