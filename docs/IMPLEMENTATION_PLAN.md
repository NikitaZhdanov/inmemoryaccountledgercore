# Implementation plan

Status: P01 implemented at the human's request; P02–P14 remain planned. This
document does not itself authorize further implementation. Money unit tests
exist; no ledger replay or observed design-test failure exists yet.

[TASK](TASK.md) preserves the assignment; [SPEC](SPEC.md) defines agreed behavior;
[DESIGN](DESIGN.md) proposes its Go implementation; [USE_CASES](USE_CASES.md)
defines verification scenarios. This plan schedules that work without changing
business decisions, numerical oracles, or [criterion verdicts](../REJECTED.md).

## Commit and verification discipline

Execute the tasks below in order when implementation is authorized. Each task
is intended as a focused commit containing its implementation, meaningful tests,
and relevant record updates. Dependencies identify required capabilities; use
direct fixtures to review lower layers before the complete runner exists.
Partial use-case coverage is called out where later integration is necessary.
Preserve intact commit history without squashing.

For each task, keep [WORKLOG](../WORKLOG.md) timestamped with actual milestones
and check results. Inventory new constants and test parameters, including their
units, rationale, halving justification, and derived formulas in
[NUMBERS](../NUMBERS.md) in the same change. Update [AMBIGUITIES](../AMBIGUITIES.md)
if a real new decision arises and [REJECTED](../REJECTED.md) only for actual
abandoned approaches or corrections. Ask about genuinely unresolved business
behavior before dependent work; do not reopen agreed decisions or implement
deferred authorization extensions. Keep [README](../README.md) accurate about
which commands and features exist at each stage.

Use independently derived exact expectations from NUMBERS and literal source
fields from TASK. Production calculations must not generate their own test
oracles. Once Go files exist, format changed files with `gofmt`, run focused
tests plus `go test ./...` and `go vet ./...` for each completed code task, and
review its diff. The separately tagged failing test is excluded from normal
checks. Completion checks below define requirements; actual outcomes belong in
[WORKLOG](../WORKLOG.md).

The human subsequently requested Testify assertions for unit tests. Use Go's
`testing` runner with Testify `require` for prerequisites and `assert` for
independent results; preserve exact, type-sensitive money expectations. This
supersedes the original standard-library-only test dependency choice without
adding production dependencies; see [A13](../AMBIGUITIES.md#a13--testify-unit-test-assertions).

Follow the [package/import map](DESIGN.md#small-code-layout): `internal/domain`
owns shared models and money; `internal/ledger` owns business logic and the small
repository interface; `internal/repository/memory` owns storage; `internal/report`
owns presentation; `internal/scenario` owns canonical inputs and close sequencing.
`cmd/replay/main.go` composes them. Ledger and memory depend on domain; report and
scenario may use ledger and domain. No reverse imports or report/scenario coupling
are needed. Keep repository signatures in shared domain types so memory need not
import ledger. Preserve `History()` / `Append(batch)` as the entire repository
boundary, with no storage-specific queries added to support reporting.

Place unit tests beside each package, using a test-local repository double for
ledger unit tests. Put tests using multiple real components in the separate
`tests/replay` external test package, even before the scenario runner exists.
They compose components through the same boundaries as `main`. Once present,
run unit tests with `go test ./internal/...` and end-to-end tests with
`go test ./tests/replay`; `go test ./...` includes both. Review import direction
as packages are introduced and verify the eventual graph with `go list -deps ./...`.

## Ordered tasks

### P01 — Bootstrap Go and exact money representation

- **Status:** Implemented in [`internal/domain`](../internal/domain/money.go),
  with adjacent [unit tests](../internal/domain/money_test.go). Go 1.23.1 module
  baseline; standard library for production and Testify for unit tests following
  the human's later request. Submission diagnostics and raw-input retention remain
  in their later tasks.
- **Scope:** Create the module and `internal/domain` package using the standard
  library for production and Testify for unit tests; record the actual Go toolchain
  and module baseline. Add currency
  scales, `int64` minor-unit money, exact decimal parsing/formatting, currency
  checks, and checked addition, subtraction, negation, and accumulation. Retain
  raw amount text at the later input-record boundary. Distinguish input
  representability errors from derived arithmetic failures.
- **Dependencies:** None.
- **Use cases:** U03, U06, U07, X04, X10 (money primitives; submission behavior
  follows in P05).
- **Completion checks:** Independently tabulated AED/BHD precision and trailing
  zero cases pass. Both signed endpoints parse/format exactly; inputs just
  outside the endpoints reject. Every checked operation detects overflow,
  including minimum-value negation and intermediate accumulation. Cross-currency
  arithmetic fails explicitly. No monetary calculation uses binary floating point.

### P02 — Exact interest rounding and conserving allocation

- **Scope:** Introduce `internal/ledger` with pure policy functions using domain
  money operations. Add positive-base daily interest calculation using the exact
  rate denominator and quotient/remainder HALF_EVEN rounding. Add the integer
  instalment allocation needed by E10, assigning its remainder to the last
  instalment. Keep these calculations independent of ledger state.
- **Dependencies:** P01.
- **Use cases:** H06, X03, X09, X10 (calculation coverage).
- **Completion checks:** Zero/negative bases accrue zero; both midpoint tie
  directions pass in both currencies. Maximum-positive-base interest is exact
  without an overflowing rate intermediate. E10 allocates BHD 3.333, 3.333,
  3.334 and conserves BHD 10.000 exactly. Calculated expectations and fixture
  rationale match NUMBERS; no principal ingress rounding is introduced.

### P03 — Immutable records and atomic in-memory repository

- **Scope:** Define account, original input, attempt/outcome, monetary entry,
  authorization transition, diagnostic, batch, and history records in
  `internal/domain`. Define the small `History()` / `Append(batch)` interface in
  `internal/ledger` using domain types and implement it in
  `internal/repository/memory` with ordered detached snapshots,
  defensive copies, and all-or-nothing record-key validation. Separate internal
  record keys and attempt identities from supplied event IDs; storage owns no
  business policy.
- **Dependencies:** P01.
- **Use cases:** H01, U04, U07, X11 (storage guarantees).
- **Completion checks:** Append order survives read-back. Mutating submitted or
  returned containers cannot change stored records. A conflicting key, including
  one inside a proposed batch, appends none of that batch. Distinct rejected
  attempts can reference an existing event ID without replacing it. Repository
  unit tests sit beside memory storage and do not mistake key uniqueness for the
  eventual duplicate-input policy. Ledger imports no memory adapter; memory
  imports only domain within this module.

### P04 — Rebuild balances and recorded state from history

- **Scope:** In `internal/ledger`, add checked value-date balance queries over
  detached history and disposable projections for authorization lifecycle/active
  holds, fee net effects, unpaid daily accruals, and closed-day tracking. Support
  explicit observation prefixes and fee exclusions needed by later reconciliation and
  criterion 1. Rebuild financial state without rerunning authorization decisions.
- **Dependencies:** P01, P03.
- **Use cases:** H01, H02, H05, X01, X11 (direct history fixtures).
- **Completion checks:** Literal record fixtures distinguish booking order from
  value-date inclusion and different observation prefixes. Fee-free and posted
  queries remain distinct. Rebuilding leaves history and original decisions
  intact; holds affect availability only. Derived overflow is surfaced before
  any dependent batch append. Do not create a second printed snapshot report.

### P05 — Ordinary input processing, validation, and duplicate rejection

- **Scope:** In `internal/ledger`, implement submission handling for credits/debits,
  exact monetary validation, original input retention, unique attempt identities,
  outcomes, and diagnostics. Reject every reused event ID before any
  event-triggered effects, regardless of original outcome or payload equality. Compute a batch
  completely before append; continue after covered business rejections and
  propagate technical failures.
- **Dependencies:** P01, P03, P04.
- **Use cases:** H01, U03, U04, U06, U07, X04, X11 (ordinary input paths;
  authorization and reversal duplicate cases follow in P06/P07).
- **Completion checks:** E1/E2 yield AED 250.00 and E4 later yields AED 650.00
  in isolated posting tests. Exact source fields/raw text survive. Invalid
  precision/out-of-range inputs retain diagnostics without financial effects;
  a subsequent fresh valid input succeeds. Identical/changed duplicate payloads
  preserve the original outcome and add no posting. Derived failure appends no
  partial failing batch and is not relabeled as a domain rejection.

### P06 — Authorization decisions and final settlement

- **Scope:** In `internal/ledger`, add event-checkpoint availability,
  approval/decline transitions, active holds, E5 final settlement, full hold
  release, and E6 unknown-ID rejection. Use posted fees in availability and
  exclude unpaid interest.
  Do not introduce partial capture, expiry, or inactive-ID settlement policies.
- **Dependencies:** P04, P05.
- **Use cases:** H02, H03, U01, U02, U04, X02 (U02 full E7 integration in P09).
- **Completion checks:** Auth-A leaves ledger AED 250.00 and availability AED
  50.00; exactly zero proposed availability approves, negative availability
  declines without a hold. E5 debits only AED 185.00, releases the entire hold,
  and leaves ledger/availability AED 465.00 with no unused-reservation credit.
  E6 records rejection without debit and replay can continue. Duplicate approved,
  declined, and rejected authorization/settlement inputs add no effects.

### P07 — Linked reversal and instalment posting

- **Scope:** In `internal/ledger`, add E9's principal reversal linked to E7,
  retaining the original record and using booking Day 6/value Day 2. Add E10's
  ordered instalment entries with its original Day 5 dates. Separate
  recorded-input duplication from idempotent internal posting/correction causes.
- **Dependencies:** P02, P05.
- **Use cases:** H05, H06, U04, X06 (posting layer; refunds/accrual restoration
  follow in P08/P09).
- **Completion checks:** Direct fixtures prove E7 remains intact and its inverse
  has the right amount, links, and dates. Repeating internal principal correction
  appends nothing; resubmitting E9 rejects as a duplicate. E10's entries preserve
  order, currency, dates, and exact total without affecting AED. Do not claim
  full financial restoration until fee and accrual integration passes.

### P08 — Daily fee reconciliation and unpaid accruals

- **Scope:** In `internal/ledger`, implement explicit close/reconciliation for
  Days 1–5, using chronological closed-day processing. Exclude each day's own
  net fee only from eligibility; carry earlier net fees into later days. Compare
  required and recorded net charges under one account/day obligation and append linked
  assessments, refunds, or renewed adjustments. Replace derived post-fee daily
  accruals; never post daily accrual ledger entries. Add the missing-BHD-policy
  diagnostic keyed by account/day/cause with unchanged-repeat suppression.
- **Dependencies:** P02, P03, P04, P05.
- **Use cases:** H04, H05, U05, X03, X05, X06, X07, X09 (controlled close fixtures).
- **Completion checks:** NUMBERS' E7 chronological fee/accrual oracle passes.
  A fee cannot sustain its own eligibility; refund and renewed eligibility share
  the original assessment identity. Repeated unchanged reconciliation/close adds
  no records or diagnostics. Negative BHD preserves balance, reports undefined
  policy, adds no invented charge/conversion, and permits continuation. Rebuilt
  accruals replace old values and never increase ledger or availability.

### P09 — Integrate backdated changes with closed-day processing

- **Scope:** In `internal/ledger`, connect financial submissions to affected
  closed-day reconciliation and explicit close calls. Reconcile E7's Days 2–4
  before returning control for E8; close Day 5 only after E8. Process E9's
  principal correction and Days 2–5 refunds/accrual rebuild before E10; E10 then
  corrects BHD Day 5. Preserve all
  original outcomes and distinguish per-batch atomicity from whole-operation
  rollback, which DESIGN does not promise.
- **Dependencies:** P06, P07, P08.
- **Use cases:** H04, H05, H06, U02, U04, U07, X01, X05, X06, X09, X11.
- **Completion checks:** Direct checkpoint integration tests in `tests/replay`,
  composing ledger with real memory storage, show Day 2 balances
  AED −370.00 fee-free, −395.00 posted at Day 5 close, and 250.00 after
  restoration. E8's proposed availability is AED −295.00 and Auth-B stays
  declined after E9. Fee assessments for Days 2, 4, 5 are booked Day 5; linked
  refunds are booked Day 6 with the same historical value dates. E9 restores
  AED 465.00 before capitalization and corrected accruals; E10 restores BHD
  Day 5 accrual to 0.004. Repeated internal work adds nothing; duplicate input
  rejection triggers no additional reconciliation effects.

### P10 — Consistent Day 6 finalization

- **Scope:** In `internal/ledger`, implement the SPEC Day 6 equations after
  historical corrections. Determine fee, rounded accrual, and exact capitalization
  before posting; include capitalization in fee eligibility and exclude it from its own interest
  base. Use the direct branch in DESIGN, with no provisional fee or iterative
  solver. Capitalization identity is per account/window.
- **Dependencies:** P08, P09.
- **Use cases:** H07, U05, U07, X03, X06, X08, X09.
- **Completion checks:** Independent rounded daily sums equal the single credits
  AED 1.03 and BHD 0.008; final closes are AED 466.03 and BHD 10.008.
  NUMBERS' Day 6 boundary fixtures close at AED 0.01, 0.00, and −25.01.
  Negative BHD eligibility uses the diagnostic fallback after capitalization.
  No capitalization earns same-day interest; repeat finalization adds nothing.
  Overflow produces no partial finalization batch or misleading business rejection.

### P11 — Canonical scenario runner and end-to-end regression

- **Scope:** Add `internal/scenario` with literal domain inputs and explicit
  close checkpoints; its runner receives an already constructed ledger core.
  Put scenario unit tests beside the package and complete replay tests in
  `tests/replay`, which constructs fresh memory/ledger components. Preserve
  E1–E10 source order, including E10 after E9; never infer closing from
  booking-date changes.
  Integrate all prior layers with checkpoint assertions before the printed report.
- **Dependencies:** P09, P10.
- **Use cases:** H01–H07, U01, U02, U04, X01, X06, X09, X11.
- **Completion checks:** Compare every canonical ID, kind, amount, currency,
  booking/value date, authorization/reversal reference, and instalment count
  against TASK, independently of the runner. Close Days 1–5 after E2/E3/E4/E6/E8
  respectively, and Day 6 after E10/corrections. Assert every final daily balance
  and accrual, retained fee/refund history, and checkpoint outcomes from NUMBERS.
  Fresh runs yield identical history/state; resubmitting to a populated repository
  rejects duplicates without changing financial or hold state. Scenario imports
  neither memory storage nor report and constructs no hidden dependencies.

### P12 — Deterministic report and executable replay

- **Scope:** Add pure projection/formatting in `internal/report`, with unit tests
  beside the package. In `cmd/replay/main.go`, construct a fresh memory repository,
  inject it into ledger, invoke scenario with that core, pass detached history
  and read-only projections to report, and print the agreed final daily table.
  Extend `tests/replay` with end-to-end report assertions.
  Include every day/account, closing ledger balance, linked fee history with
  booking/value days, recorded authorization states, and errors/decision reasons.
  Publish the now-runnable replay command and output interpretation in README.
- **Dependencies:** P11.
- **Use cases:** H08, X01, X04, X06; report outcomes from H01–H07, U01, U02.
- **Completion checks:** `go run ./cmd/replay` succeeds. Repeated fresh runs print
  identical output in supplied day/account order at each currency's precision.
  Label the observation as restated after E1–E10 and Day 6 finalization and
  distinguish restated finances from recorded decisions. Auth-A is active
  Days 2–3 and settled from Day 4; Auth-B is declined from Day 5. E6/E8 reasons
  appear only on Days 4/5. Group charges/refunds by assessed day even though net
  final fees are zero. No extra original-snapshot table is printed. `main` owns
  composition; report does not mutate ledger state or construct dependencies.
  `go list -deps ./...` succeeds and its import graph matches DESIGN without cycles.

### P13 — Investigate and preserve a genuine failing design test

- **Scope:** Examine the implemented design for an evidenced limitation, using
  the [USE_CASES candidates](USE_CASES.md#design-test-candidates) as investigation
  leads only. Select a useful property, explain the actual trade-off and evidence
  in AMBIGUITIES A11, and add a real test with an inline explanation under the
  `designlimitation` build tag beside the affected package, or in `tests/replay`
  if it exposes a cross-component limitation. Document its exact isolated command
  and why it satisfies the assignment. Record any actually abandoned investigation
  honestly.
- **Dependencies:** P11, P12; actual implementation evidence is required.
- **Use cases:** No preselected failing scenario. U07, X06, X10, X11 may inform
  investigation; their specified regression expectations must still pass.
- **Completion checks:** `go test -tags designlimitation ./...` executes the
  annotated assertion and genuinely exits nonzero for the explained limitation;
  record the observed failure. `go test ./...` still passes. No skip, expected
  failure wrapper, unconditional failure, invented balance, weakened assertion,
  or preserved regression bug substitutes for this deliverable. Expected overflow
  alone is a passing U07 test, not evidence of the required failing test. If a
  candidate proves artificial or reveals no meaningful limitation, continue the
  investigation and leave this task explicitly incomplete; do not claim selection
  or completion without evidence or introduce a new business promise to force it.

### P14 — Final documentation, coverage, and deliverable audit

- **Scope:** Reconcile README, NUMBERS, AMBIGUITIES, REJECTED, WORKLOG, and the
  specification/design/use-case/plan statuses with the delivered implementation.
  Verify each TASK deliverable and every scenario's actual test coverage. Keep
  genuine limitations/deferred extensions explicit and record actual results.
- **Dependencies:** P01–P13, including the observed genuine failure.
- **Use cases:** H01–H08, U01–U07, X01–X11; separate P13 design-test evidence.
- **Completion checks:** Run `go test ./...`, `go vet ./...`, and
  `go run ./cmd/replay`; check formatting and `go list -deps ./...` for the
  specified acyclic package graph. Confirm unit tests are beside each package
  and end-to-end tests are separate in `tests/replay`; document their respective
  `go test ./internal/...` and `go test ./tests/replay` commands. Run
  `go test -tags designlimitation ./...` separately and report its real nonzero
  outcome. README gives actual runtime/dependencies, exact setup/run commands,
  report interpretation, and the isolated failing-test command. Check local
  Markdown links/anchors, source-event fidelity, numerical inventory, diff, and
  intact unsquashed history. Reassess all eight criteria without changing their
  agreed verdicts: refuse 2/7/8; qualify 1/6 and the conditional 5; accept 3/4.
  Ensure abandoned-build claims reflect real attempts and every worklog result
  is timestamped honestly. Declare completion only with all deliverables checked;
  otherwise report exactly what remains open.
