# Implementation plan

Status: P01–P14 complete for the agreed scope. The canonical scenario,
deterministic final report, and executable replay have unit and real-component
integration coverage. The isolated failing design test exposes the absence of
whole-submission financial atomicity; it remains intentionally failing. 
Deferred scope and technical limitations
remain explicit in SPEC and DESIGN.

[TASK](TASK.md) preserves the assignment; [SPEC](SPEC.md) defines agreed behavior;
[DESIGN](DESIGN.md) defines its Go architecture; [USE_CASES](USE_CASES.md)
defines verification scenarios. This plan schedules that work without changing
business decisions, numerical oracles, or [criterion verdicts](../REJECTED.md).

## Commit and verification discipline

Execute the tasks below in order within the agreed implementation scope. Each task
is intended as a focused commit containing its implementation, meaningful tests,
and relevant record updates. Dependencies identify required capabilities; use
direct fixtures to review lower layers independently of the complete runner.
Partial use-case coverage is called out where later integration is necessary.
Preserve intact commit history without squashing.

For each task, keep [WORKLOG](../WORKLOG.md) timestamped with actual milestones
and check results. Inventory new constants and test parameters, including their
units, rationale, halving justification, and derived formulas in
[NUMBERS](../NUMBERS.md) in the same change. Follow
[AGENTS documentation guidance](../AGENTS.md#documentation-maintenance) for decision
records; update [REJECTED](../REJECTED.md) only for actual abandoned approaches or
corrections. Ask about genuinely unresolved business
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

Use the [DEVELOPMENT assertion convention](DEVELOPMENT.md#assertion-convention)
for unit tests.

Follow DESIGN's [package/import map](DESIGN.md#packages-and-dependencies) and
[repository contract](DESIGN.md#repository-boundary) as packages are introduced.

Follow [DEVELOPMENT test organization](DEVELOPMENT.md#test-organization) for test
placement, component wiring, and package-specific commands. Integration tests can
compose real components independently of the scenario runner.

## Ordered tasks

### P01 — Bootstrap Go and exact money representation

- **Status:** Implemented in [`internal/domain`](../internal/domain/money.go),
  with adjacent [unit tests](../internal/domain/money_test.go). Go 1.23.1 module
  baseline; standard library for production and Testify for unit tests.
  Currency descriptors are now validated code/scale values; concrete currencies
  are test fixtures or application composition, not core constants.
  Submission diagnostics and raw-input retention are implemented under P03/P05.
- **Scope:** Create the module and `internal/domain` package using the standard
  library for production and Testify for unit tests; record the actual Go toolchain
  and module baseline. Add currency
  scales, `int64` minor-unit money, exact decimal parsing/formatting, currency
  checks, and checked addition, subtraction, negation, and accumulation. Retain
  raw amount text at the later input-record boundary. Distinguish input
  representability errors from derived arithmetic failures.
- **Dependencies:** None.
- **Use cases:** U03, U06, U07, X04, X10, X12 (money primitives; submission behavior
  follows in P05).
- **Completion checks:** Independently tabulated AED/BHD precision and trailing
  zero cases pass. Both signed endpoints parse/format exactly; inputs just
  outside the endpoints reject. Every checked operation detects overflow,
  including minimum-value negation and intermediate accumulation. Cross-currency
  arithmetic fails explicitly. Configurable scales zero/four/18 and same-code
  scale conflicts are exercised; equal independently constructed descriptors
  are compatible. No monetary calculation uses binary floating point.

### P02 — Exact interest rounding and conserving allocation

- **Status:** Implemented in [daily interest](../internal/interest/interest.go)
  and [private posting allocation](../internal/posting/instalments.go), with
  adjacent [interest](../internal/interest/interest_test.go) and
  [allocation tests](../internal/posting/allocation_test.go), using shared
  [domain rounding](../internal/domain/rounding.go). Posting, accrual state,
  and capitalization are implemented under P07/P08/P10.
- **Scope:** Add pure interest and posting-allocation functions using domain
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

- **Status:** Implemented in [domain records](../internal/domain/records.go), the
  [ledger interface](../internal/ledger/repository.go), and the
  [memory adapter](../internal/repository/memory/repository.go), with adjacent
  [repository tests](../internal/repository/memory/repository_test.go). Storage
  checks envelope shape and global record keys, preserves order, and detaches
  containers and payloads on write/read. Duplicate-input decisions, configuration
  validation, and projection rebuilding are implemented under P04/P05; P03
  supplies storage-level coverage for U04, U07, and X11.
- **Scope:** Define account, original input, attempt/outcome, monetary entry,
  authorization transition, diagnostic, batch, and history records in
  `internal/domain`. Define the small `History()` / `Append(batch)` interface in
  `internal/ledger` using domain types and implement it in
  `internal/repository/memory` with ordered detached snapshots,
  defensive copies, and all-or-nothing record-key validation. Separate internal
  record keys and attempt identities from supplied event IDs; storage owns no
  business policy. Account records carry validated code/scale descriptors and
  opening money; no account, currency, or policy defaults are supplied here.
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

- **Status:** Implemented in [history projections](../internal/projection/projection.go)
  with adjacent [direct-fixture tests](../internal/projection/projection_test.go).
  Explicit record prefixes, checked value-date balances and fee exclusions,
  recorded authorization state/holds, signed fee nets, closed days, and derived
  daily/unpaid accruals are covered. Rebuilds own their history and preserve
  original decisions. Submission/append integration and chronological fee
  reconciliation are implemented under P05–P09; P04 tests use direct history.
- **Scope:** In `internal/projection`, add checked value-date balance queries over
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

- **Status:** Implemented in [core initialization](../internal/ledger/core.go)
  and [submission commits](../internal/ledger/submission.go), with
  [posting preparation](../internal/posting/submission.go) and adjacent
  configuration/submission tests and [posting integration](../tests/replay/ordinary_test.go).
  Configuration is validated and copied before account records are appended.
  Ordinary attempts retain source fields and append outcomes, postings, or
  diagnostics atomically; duplicate checks precede dispatch and validation.
  Proposed-history checks catch derived overflow before append, including at
  later value dates. Amount signs follow [A10](../AMBIGUITIES.md#a10--ordinary-input-amount-signs).
  Authorization and settlement are implemented under P06; reversal, instalments, and
  reconciliation are implemented under P07–P09.
- **Scope:** In `internal/posting`, prepare credit/debit batches for ledger commits,
  exact monetary validation, original input retention, unique attempt identities,
  outcomes, and diagnostics. Reject every reused event ID before any
  event-triggered effects, regardless of original outcome or payload equality. Compute a batch
  completely before append; continue after covered business rejections and
  propagate technical failures. Add core initialization with injected accounts
  and explicit per-currency charges (zero means no fee). Before any processing or
  append, reject invalid descriptors, conflicting scales for a code, missing
  account-currency policies, negative charges, and incompatible policy money.
  Copy configuration into core-owned state.
- **Dependencies:** P01, P03, P04.
- **Use cases:** H01, U03, U04, U05, U06, U07, X04, X11, X12 (ordinary input paths;
  authorization and reversal duplicate cases follow in P06/P07).
- **Completion checks:** E1/E2 yield AED 250.00 and E4 later yields AED 650.00
  in isolated posting tests. Exact source fields/raw text survive. Invalid
  precision/out-of-range inputs retain diagnostics without financial effects;
  a subsequent fresh valid input succeeds. Identical/changed duplicate payloads
  preserve the original outcome and add no posting. Derived failure appends no
  partial failing batch and is not relabeled as a domain rejection. Initialization
  tests distinguish absent policies from an explicit currency-specific zero;
  reject malformed configurations without records, accept a subsequent valid
  configuration, and prove caller mutation cannot alter initialized policy.

### P06 — Authorization decisions and final settlement

- **Status:** Implemented in [authorization handling](../internal/posting/authorization.go)
  with [builder tests](../internal/posting/authorization_test.go),
  [ledger workflow tests](../internal/ledger/authorization_test.go) and
  [E1–E6 integration](../tests/replay/authorization_test.go). Decisions use posted
  availability; final settlement atomically debits the settlement amount and
  releases the linked full hold. Unknown-ID rejection, duplicate outcomes,
  fee/unpaid-interest boundaries, and technical-failure atomicity are covered.
  U02 uses direct correction fixtures; P09 adds full E7/E9 checkpoint integration.
  Zero amounts pass validation and negative amounts reject under
  [resolved A15](../AMBIGUITIES.md#a15--authorization-and-settlement-amount-signs).
  Deferred workflows return a technical error without an attempt.
- **Scope:** In `internal/posting`, add event-checkpoint availability,
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

- **Status:** Implemented in [principal corrections](../internal/posting/corrections.go)
  and [instalment posting](../internal/posting/instalments.go), with adjacent
  builder tests, ledger workflow tests, and [real-component integration](../tests/replay/corrections_test.go).
  E9 derives E7's inverse from the stored debit, preserving its original record
  and value day. Internal correction repeats produce no records; duplicate E9/E10
  inputs reject before effects. Fresh attempts to reverse an already reversed
  debit record business rejections. E10 posts all three ordered BHD credits in one
  batch. Tests cover retained source fields, currency isolation, exact totals,
  and technical-failure atomicity. P08 covers fee refunds and closed-day accrual
  restoration with controlled fixtures; P09 supplies full E7/E8/E9 checkpoint integration.
- **Scope:** In `internal/posting`, add E9's principal reversal linked to E7,
  retaining the original record and using booking Day 6/value Day 2. Add E10's
  ordered instalment entries with its original Day 5 dates. Separate
  recorded-input duplication from idempotent internal posting/correction causes.
- **Dependencies:** P02, P05.
- **Use cases:** H05, H06, U04, X06 (posting layer; refunds/accrual restoration
  follow in P08/P09).
- **Completion checks:** Direct fixtures prove E7 remains intact and its inverse
  has the right amount, links, and dates. Repeating internal principal correction
  appends nothing; resubmitting E9 rejects as a duplicate. E10's entries preserve
  order, currency, dates, and exact total without affecting AED. Full financial
  restoration requires the fee and accrual integration checks in P08/P09.

### P08 — Daily fee reconciliation and unpaid accruals

- **Status:** Implemented in [explicit closing/reconciliation](../internal/ledger/closing.go),
  with adjacent [controlled-fixture tests](../internal/ledger/closing_test.go)
  and [real-storage account isolation](../tests/replay/closing_test.go).
  Chronological fee comparison appends assessments, linked refunds, or renewed
  adjustments under one account/day obligation. New closes and their fee effects
  commit together; unchanged work makes no append. Rebuilt unpaid accruals remain
  projections. Tests cover the E7/E9 numerical oracle, own-fee exclusion,
  earlier-fee carry, repeated eligibility changes, zero/custom currency policies,
  interest ties, and per-batch technical failures. P09 supplies automatic
  submission-triggered reconciliation; P10 supplies Day 6 finalization.
- **Schedule boundary:** The caller supplies ordinary close days; the core has
  no fixed Day 5 cutoff. Days 1–5 below identify canonical replay coverage.
  [Alternate-schedule tests](../internal/ledger/closing_test.go) exercise ordinary
  closes and fee corrections on Days 6 and 7.
- **Scope:** In `internal/ledger`, implement explicit close/reconciliation for
  Days 1–5, using chronological closed-day processing. Exclude each day's own
  net fee only from eligibility; carry earlier net fees into later days. Compare
  required and recorded net charges under one account/day obligation and append linked
  assessments, refunds, or renewed adjustments. Replace derived post-fee daily
  accruals; never post daily accrual ledger entries. Obtain charges from validated
  initialization configuration for every currency; an explicit zero policy posts
  no fee entry or undefined-policy diagnostic.
- **Dependencies:** P02, P03, P04, P05.
- **Use cases:** H04, H05, H09, X03, X05, X06, X07, X09 (controlled close fixtures;
  missing-policy initialization is covered in P05).
- **Completion checks:** NUMBERS' E7 chronological fee/accrual oracle passes.
  A fee cannot sustain its own eligibility; refund and renewed eligibility share
  the original assessment identity. Repeated unchanged reconciliation/close adds
  no records or diagnostics. Negative BHD under explicit no-fee configuration
  preserves balance without a charge or undefined-policy error. A custom-code
  currency with a configured nonzero charge exercises the same fee logic. Rebuilt
  accruals replace old values and never increase ledger or availability.

### P09 — Integrate backdated changes with closed-day processing

- **Status:** Implemented in [submission orchestration](../internal/ledger/submission.go),
  with [batch-boundary tests](../internal/ledger/backdating_test.go) and
  [real-component checkpoint integration](../tests/replay/backdating_test.go).
  Financial submissions reconcile affected closed days before returning without
  closing open days. Tests cover E7/E8/E9 timing, linked fee/refund dates,
  preserved outcomes, corrected AED/BHD accruals, repeated internal work,
  duplicate rejection without correction effects, and append/arithmetic failures
  after committed input or earlier fee batches. Day 6 capitalization is implemented
  under P10.
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

- **Status:** Implemented in [finalization](../internal/ledger/finalization.go),
  with [unit tests](../internal/ledger/finalization_test.go) and
  [corrected-stream integration](../tests/replay/backdating_test.go).
  The direct branch determines the final fee before projecting current-day
  interest and summing rounded accruals. Fee effects, capitalization, and a new
  close share one validated batch per account/window. Tests cover exact AED/BHD
  totals, fee boundaries, zero/custom charges, own-fee refunds, same-day interest
  exclusion, repeat calls, and append/arithmetic failures without partial writes.
  P11 supplies the reusable runner; P12 supplies executable reporting.
- **Scope:** In `internal/ledger`, implement the SPEC Day 6 equations after
  historical corrections. Determine fee, rounded accrual, and exact capitalization
  before posting; include capitalization in fee eligibility and exclude it from its own interest
  base. Use the direct branch in DESIGN, with no provisional fee or iterative
  solver. Capitalization identity is per account/window.
- **Dependencies:** P08, P09.
- **Use cases:** H07, H09, U07, X03, X06, X08, X09.
- **Completion checks:** Independent rounded daily sums equal the single credits
  AED 1.03 and BHD 0.008; final closes are AED 466.03 and BHD 10.008.
  NUMBERS' Day 6 boundary fixtures close at AED 0.01, 0.00, and −25.01.
  Negative BHD eligibility uses its configured zero charge after capitalization;
  no runtime missing-policy fallback exists.
  No capitalization earns same-day interest; repeat finalization adds nothing.
  Overflow produces no partial finalization batch or misleading business rejection.

### P11 — Canonical scenario runner and end-to-end regression

- **Status:** Implemented in [scenario](../internal/scenario/scenario.go), with
  adjacent [unit tests](../internal/scenario/scenario_test.go) and real-component
  [canonical replay tests](../tests/replay/scenario_test.go). Tests independently
  compare every source field, explicit close positions, checkpoint and final
  numerical oracles, original decisions, linked fee/refund history, deterministic
  fresh runs, duplicate resubmission, and stopping on technical failures.
  Accounts/descriptors and the initialized core are injected; scenario constructs
  no repository, policy, or report. P12 supplies executable reporting.
- **Scope:** Add `internal/scenario` with literal domain inputs and explicit
  close checkpoints; its runner receives an already constructed ledger core.
  Put scenario unit tests beside the package and complete replay tests in
  `tests/replay`, which constructs fresh memory/ledger components. Preserve
  E1–E10 source order, including E10 after E9; never infer closing from
  booking-date changes.
  Receive configured account/currency descriptors from composition rather than
  constructing defaults. Integrate all prior layers with checkpoint assertions
  before the printed report.
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

- **Status:** Implemented in [report](../internal/report/report.go) and
  [executable composition](../cmd/replay/main.go), with adjacent unit/composition
  tests and [report integration](../tests/replay/report_test.go) against a literal
  full-table oracle. Tests cover every day/account, exact precision, linked fee
  history, recorded lifecycle/reason placement, fresh-run determinism, read-only
  reporting, and constructor rejection before replay/output. Replay and output
  interpretation are documented in README and DEVELOPMENT. The required failing
  design test and final audit are completed under P13–P14.
- **Scope:** Add pure projection/formatting in `internal/report`, with unit tests
  beside the package. In `cmd/replay/main.go`, construct a fresh memory repository,
  inject it into ledger, invoke scenario with that core, pass detached history
  and read-only history projections to report, and print the agreed final daily table.
  Compose AED/2 and BHD/3 descriptors, zero opening accounts, AED 25.00 policy,
  and explicit BHD no-fee policy in main. Pass configuration to initialization
  and descriptors to scenario; propagate initialization errors before replay/report.
  Extend `tests/replay` with end-to-end report assertions.
  Include every day/account, closing ledger balance, linked fee history with
  booking/value days, recorded authorization states, and errors/decision reasons.
  Publish the runnable replay command in README and detailed output interpretation
  in [DEVELOPMENT](DEVELOPMENT.md).
- **Dependencies:** P11.
- **Use cases:** H08, X01, X04, X06; report outcomes from H01–H07, U01, U02.
- **Configuration checks:** U05 and X12 at composition, using the P05 constructor.
- **Completion checks:** `go run ./cmd/replay` succeeds. Repeated fresh runs print
  identical output in supplied day/account order at each currency's precision.
  Label the observation as restated after E1–E10 and Day 6 finalization and
  distinguish restated finances from recorded decisions. Auth-A is active
  Days 2–3 and settled from Day 4; Auth-B is declined from Day 5. E6/E8 reasons
  appear only on Days 4/5. Group charges/refunds by assessed day even though net
  final fees are zero. No extra original-snapshot table is printed. `main` owns
  composition; report does not mutate ledger state or construct dependencies.
  Composition checks confirm exact descriptor/charge values and required policy
  coverage; core constructor failure prevents replay/report. No concrete currency
  constants or switches appear in domain or business packages, and scenario defines no defaults.
  `go list -deps ./...` succeeds and its import graph matches DESIGN without cycles.

### P13 — Investigate and preserve a genuine failing design test

- **Status:** Implemented in the tagged
  [integration probe](../tests/replay/designlimitation_test.go), exposing the
  [repository atomicity boundary](DESIGN.md#repository-boundary). A transient
  failure between fee batches leaves the input and first fee committed. The
  annotated financial atomicity assertion executes and fails under the isolated
  command; normal interruption regressions verify idempotent explicit recovery.
  Normal tests pass. Observed checks are recorded in [WORKLOG](../WORKLOG.md).
- **Scope:** Examine the implemented design for an evidenced limitation, using
  the [USE_CASES candidates](USE_CASES.md#design-test-candidates) as investigation
  leads only. Select a useful property, document the architectural trade-off in
  DESIGN and verification evidence in USE_CASES, and add a real test with an
  inline explanation under the `designlimitation` build tag beside the affected
  package, or in `tests/replay` if it exposes a cross-component limitation.
  Document its exact isolated command
  and why it satisfies the assignment. Record any superseded investigation and
  its outcome.
- **Dependencies:** P11, P12; actual implementation evidence is required.
- **Use cases:** No preselected failing scenario. U07, X06, X10, X11 may inform
  investigation; their specified regression expectations must still pass.
- **Completion checks:** `go test -tags designlimitation ./...` executes the
  annotated assertion and genuinely exits nonzero for the explained limitation;
  record the observed failure. `go test ./...` still passes. The selected
  assertion must execute, use a justified expectation, and expose a design
  limitation. Skips, expected-failure wrappers, unconditional failures, weakened
  assertions, and retained regression bugs do not meet those conditions.
  Expected overflow alone is a passing U07 test, not evidence of the required
  failing test. If a candidate reveals no meaningful limitation, continue the investigation and
  leave this task incomplete. Selection requires implementation evidence and
  must distinguish a design limitation from an unspecified business requirement.

### P14 — Final documentation, coverage, and deliverable audit

- **Status:** Complete. Reconciled documentation with implementation/history,
  restored removed decision and numerical records, repaired local references,
  and mapped all 28 scenarios to actual tests in
  [USE_CASES](USE_CASES.md#coverage-review-and-scope-boundary). TASK and the
  annotated design assertion are unchanged. Observed checks and compiler versions
  are recorded in [WORKLOG](../WORKLOG.md).

| TASK deliverable | Audited evidence |
| --- | --- |
| In-memory core, no web/persistence/UI/database | [Package graph](DESIGN.md#packages-and-dependencies), [memory adapter](../internal/repository/memory/repository.go), standard-library-only production dependencies |
| Ordered six-day replay and per-day output | [Source-field tests](../internal/scenario/scenario_test.go), [checkpoint/final assertions](../tests/replay/scenario_test.go), [literal report oracle](../tests/replay/testdata/canonical_report.txt), executable replay |
| Intact history without squashing | All existing milestone commits retained; no history changes in P14 |
| README run/output guidance | [Quick start](../README.md#quick-start), [DEVELOPMENT commands and interpretation](DEVELOPMENT.md#replay-and-isolated-failing-test), existing Make targets |
| Every chosen constant and rationale | [NUMBERS](../NUMBERS.md): source values, exact formulas/bounds, rounding/allocation, test fixtures, tool/configuration parameters, and halving explanations |
| Ambiguities and resolutions | [AMBIGUITIES](../AMBIGUITIES.md): task interpretations, explicit clarifications, and links to deferred scope |
| All eight criteria and abandoned implementations | [REJECTED](../REJECTED.md): refuse 2/7/8; qualify 1/6 and conditional 5; accept 3/4; replacements checked against commits and recorded investigations |
| One annotated genuinely failing design test | [Tagged probe](../tests/replay/designlimitation_test.go): isolated full command exits 1 at expected 0 versus actual −2,501 minor units; normal recovery tests pass |
| Timestamped real worklog | [WORKLOG](../WORKLOG.md): historical timestamps/outcomes preserved; historical claims reviewed against available records; P14 reports fresh observed checks separately |

The audit found documentation drift without a missing supplied-scenario behavior.
Per-batch atomicity, intermediate int64 bounds, sequential in-memory execution,
and deferred authorization extensions remain the documented limitations. Hosted
CI execution is not claimed; local checks and earlier CI-equivalent runs are
identified in WORKLOG.

- **Scope:** Reconcile README, NUMBERS, AMBIGUITIES, REJECTED, WORKLOG, and the
  specification/design/use-case/plan statuses with the delivered implementation.
  Verify each TASK deliverable and every scenario's actual test coverage. Keep
  genuine limitations/deferred extensions explicit and record actual results.
- **Dependencies:** P01–P13, including the observed genuine failure.
- **Use cases:** H01–H09, U01–U07, X01–X12; separate P13 design-test evidence.
- **Completion checks:** Run `go test ./...`, `go vet ./...`, and
  `go run ./cmd/replay`; check formatting and `go list -deps ./...` for the
  specified acyclic package graph. Confirm unit tests are beside each package
  and end-to-end tests are separate in `tests/replay`; document their respective
  `go test ./internal/...` and `go test ./tests/replay` commands. Run
  `go test -tags designlimitation ./...` separately and report its real nonzero
  outcome. README provides a concise quick start and links to
  [DEVELOPMENT](DEVELOPMENT.md) for runtime/dependency details, setup/run commands,
  report interpretation, and the isolated failing-test command. Check local
  Markdown links/anchors, source-event fidelity, numerical inventory, diff, and
  intact unsquashed history. Reassess all eight criteria without changing their
  agreed verdicts: refuse 2/7/8; qualify 1/6 and the conditional 5; accept 3/4.
  Review replacement records against implementation history and verify each
  worklog timestamp and recorded outcome. Declare completion only with all
  deliverables checked; otherwise report exactly what remains open.
