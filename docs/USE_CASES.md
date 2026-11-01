# Ledger use cases

This catalog owns verification scenarios and their implemented test coverage.
[SPEC](SPEC.md) owns business expectations, [DESIGN](DESIGN.md) technical
boundaries, and [TASK](TASK.md) the canonical input. Scenario IDs are labels;
amounts and boundary fixtures come from [NUMBERS](../NUMBERS.md). The
[coverage matrix](#coverage-review-and-scope-boundary) maps all 28 scenarios to
executable assertions; execution outcomes belong to [WORKLOG](../WORKLOG.md).

Each scenario is independent unless its preconditions name a replay checkpoint.
Validation fixtures supplement, rather than modify, E1–E10. Expected rejection,
decline, initialization error, or technical failure is a **passing regression test**
when observed as specified. None by itself satisfies the assignment's required
genuinely failing design test; see [design-test candidates](#design-test-candidates).

## Happy flows

### H01 — Initialize and post ordinary entries

- **Preconditions:** Fresh in-memory repository with the supplied accounts.
- **Action:** Submit E1, E2, and later E4 at their supplied checkpoints.
- **Expected outcome:** AED closes after E2 at 250.00 and reaches 650.00 after
  E4; BHD remains unchanged. Original fields and ordered monetary postings survive.
- **Preserved invariants:** Account/currency isolation and exact value-dated
  balances; [scope and storage](SPEC.md#scope-and-storage),
  [history and replay](SPEC.md#history-and-replay).

### H02 — Approve Auth-A

- **Preconditions:** E1–E2 posted; no active hold.
- **Action:** Submit E3.
- **Expected outcome:** Auth-A is active, ledger remains AED 250.00, availability
  becomes AED 50.00; the original approval is recorded.
- **Preserved invariants:** An approved hold leaves nonnegative availability and
  affects no monetary posting; [authorizations and settlements](SPEC.md#authorizations-and-settlements).

### H03 — Final settlement releases the entire reservation

- **Preconditions:** Replay through E4 with Auth-A active.
- **Action:** Submit E5.
- **Expected outcome:** Ledger and availability become AED 465.00, Auth-A is
  settled, and its hold is zero. The unused AED 15.00 reservation produces no credit.
- **Preserved invariants:** Only the settlement amount leaves the ledger; full
  hold release and retained lifecycle; [authorizations and settlements](SPEC.md#authorizations-and-settlements).

### H04 — Reconcile the late backdated debit

- **Preconditions:** Days 1–4 closed at the specified checkpoints, including E6's rejection.
- **Action:** Submit E7 and finish its reconciliation before E8; later close Day 5 after E8.
- **Expected outcome:** Before E8, Days 2–4 have the fee/interest results in the
  [checkpoint oracle](../NUMBERS.md#checkpoint-calculations). Day 5's own fee
  appears only at its close. Assessments for Days 2, 4, 5 are booked Day 5 with
  their historical value dates; Day 3 has no fee.
- **Preserved invariants:** Closed-day chronology, one obligation per account/day,
  earlier-fee carry-forward, and no premature open-day assessment;
  [history and replay](SPEC.md#history-and-replay), [fees](SPEC.md#fees).

### H05 — Reverse financial effects without erasing decisions

- **Preconditions:** Day 5 closed after E8; E7 and its assessments are recorded.
- **Action:** Submit E9 and reconcile Days 2–5 before E10.
- **Expected outcome:** Append E7's inverse and linked refunds, booked Day 6
  with the specified historical value dates. AED returns to 465.00 before
  capitalization; corrected accruals match the oracle. E7, assessments, E6's
  rejection, and Auth-B's decline remain recorded.
- **Preserved invariants:** Financial restoration through appended corrections,
  with original authorization outcomes unchanged;
  [interest and financial restoration](SPEC.md#interest-and-financial-restoration).

### H06 — Post the late BHD instalments

- **Preconditions:** E9 processed; BHD Day 5 was already closed.
- **Action:** Submit E10 in its supplied position after E9.
- **Expected outcome:** Ordered BHD credits are 3.333, 3.333, 3.334, total
  10.000; booking/value Day 5 is retained. BHD Day 5 accrual becomes 0.004;
  AED is unaffected. See the [allocation oracle](../NUMBERS.md#instalment-conservation).
- **Preserved invariants:** Principal conservation, last-instalment residual,
  source order, and independent currencies; [instalments](SPEC.md#instalments),
  [history and replay](SPEC.md#history-and-replay).

### H07 — Finalize and capitalize

- **Preconditions:** All supplied inputs and historical corrections processed.
- **Action:** Finalize Day 6 for both accounts.
- **Expected outcome:** Single capitalization credits of AED 1.03 and BHD 0.008;
  closing balances AED 466.03 and BHD 10.008. Every daily accrual matches the
  [final oracle](../NUMBERS.md#expected-replay-results).
- **Preserved invariants:** Capitalization equals the exact sum of corrected
  rounded accruals and earns no same-day interest;
  [Day 6 finalization](SPEC.md#day-6-finalization).

### H08 — Produce the deterministic final report

- **Coverage:** [Canonical report integration](../tests/replay/report_test.go),
  [report unit tests](../internal/report/report_test.go), and
  [composition tests](../cmd/replay/main_test.go).
- **Preconditions:** H07 complete in a fresh repository.
- **Action:** Generate the report; repeat the whole replay in another fresh repository.
- **Expected outcome:** Identical output, one final restated table for every day
  and both accounts, labeled with its final observation checkpoint. Balances and
  fee records match NUMBERS. Auth-A is active Days 2–3 and settled from Day 4;
  Auth-B is declined from Day 5. E6 and E8 reasons appear on Days 4 and 5 only.
  Charges/refunds retain links and booking/value days, grouped by assessed day.
- **Preserved invariants:** Currency formatting and recorded decisions remain
  distinct from restated finances; [reporting and observations](SPEC.md#reporting-and-observations).

### H09 — Negative BHD balance with explicit no-fee policy

- **Preconditions:** A valid initialized core with BHD's policy explicitly set
  to zero and a negative BHD closing base. For the Day 6 fixture, prior accrual is zero.
- **Action:** Close/reconcile the day, repeat unchanged processing, and separately
  finalize the Day 6 fixture.
- **Expected outcome:** No fee entry, conversion, or undefined-policy diagnostic;
  the negative balance is retained and its interest is zero. Repetition adds no effects.
- **Preserved invariants:** Explicit zero is a configured policy; the BHD no-fee
  assumption is separate from TASK's AED tariff. See
  [currency and policy configuration](SPEC.md#currency-and-policy-configuration).

## Unhappy flows

### U01 — Unknown authorization settlement

- **Preconditions:** Replay through E5; Auth-Z has never been authorized.
- **Action:** Submit E6, then continue the supplied replay.
- **Expected outcome:** Retain the input and rejection diagnostic without its
  debit; AED remains 465.00 at this checkpoint. Subsequent events still process.
- **Preserved invariants:** Rejection has no monetary/hold effects;
  [authorizations and settlements](SPEC.md#authorizations-and-settlements),
  [monetary validation and errors](SPEC.md#monetary-validation-and-errors).

### U02 — Insufficient availability and the unsettled Auth-B

- **Preconditions:** E7's closed-day corrections posted; Day 5 still open.
- **Action:** Submit E8, then process E9 and finish the window.
- **Expected outcome:** Proposed availability is AED −295.00. E8 declines and
  creates no hold. The decline survives restoration; there is no active Auth-B
  hold at window end and no inferred expiry or release.
- **Preserved invariants:** Approval uses the event checkpoint, includes posted
  fees, excludes unpaid interest, and is never retroactively rerun;
  [authorizations and settlements](SPEC.md#authorizations-and-settlements).

### U03 — Reject principal with excess precision

- **Preconditions:** Valid account and otherwise covered input; isolated fixtures
  from [verification parameters](../NUMBERS.md#verification-parameters).
- **Action:** Submit AED 1.005 or BHD 1.0001, then a valid input with a fresh ID.
- **Expected outcome:** Reject each invalid amount with its diagnostic and no
  financial/hold effects; the subsequent valid input processes.
- **Preserved invariants:** No ingress rounding or tolerance changes principal;
  [monetary validation and errors](SPEC.md#monetary-validation-and-errors).
  Ordinary CREDIT/DEBIT validation also rejects negative amounts and accepts
  zero under [A10](../AMBIGUITIES.md#a10--ordinary-input-amount-signs).

### U04 — Reject duplicate IDs across original outcomes

- **Preconditions:** An original input succeeded, declined, or was rejected.
- **Action:** Resubmit its ID with identical and changed payloads in independent
  cases, including monetary inputs, authorizations, and E9.
- **Expected outcome:** Each duplicate attempt is retained as a rejection;
  original payload/outcome remains intact. No new debit, credit, hold, or
  event-triggered correction follows. Resubmitting the full stream to a populated
  repository rejects duplicates rather than reproducing a fresh replay.
- **Preserved invariants:** Input rejection is distinct from internal correction
  idempotence; [monetary validation and errors](SPEC.md#monetary-validation-and-errors).

### U05 — Reject missing or incompatible initialization policy

- **Preconditions:** Account configuration with a missing overdraft-policy entry;
  independent fixtures have invalid descriptors, a code assigned conflicting
  scales, a negative charge, or money incompatible with its policy currency.
- **Action:** Initialize the core, then try a separate valid initialization with
  an explicit currency-specific zero charge.
- **Expected outcome:** Invalid configurations reject before processing/append;
  history stays unchanged. The valid no-fee configuration initializes successfully.
  An absent policy is never treated as a zero-charge policy or a replay diagnostic.
- **Preserved invariants:** Complete configuration before replay;
  [currency and policy configuration](SPEC.md#currency-and-policy-configuration).

### U06 — Reject unrepresentable input

- **Preconditions:** Exact monetary text one minor unit outside either `int64`
  endpoint, using the [boundary inventory](../NUMBERS.md#int64-representation).
- **Action:** Parse/submit the input, then process a representable input.
- **Expected outcome:** Out-of-range diagnostic and rejection without financial
  effects; continuation remains possible. Scaling/parsing never wraps or clamps.
- **Preserved invariants:** Exact representability and covered rejection handling;
  [monetary validation and errors](SPEC.md#monetary-validation-and-errors),
  technical bounds in [DESIGN](DESIGN.md#money-representation).

### U07 — Surface a derived arithmetic failure

- **Preconditions:** Representable operands whose addition, subtraction,
  accumulation, or minimum-value negation cannot fit `int64`.
- **Action:** Execute the affected calculation/batch.
- **Expected outcome:** Explicit technical failure stops replay; no wrapped or
  clamped result and no partial append of the failing batch. Do not relabel the
  failure a business decline/rejection. Earlier committed batches are separate;
  whole-replay rollback/recovery is not promised.
- **Preserved invariants:** Unexpected failures remain visible per
  [monetary validation and errors](SPEC.md#monetary-validation-and-errors);
  checked arithmetic/batch boundary per [DESIGN](DESIGN.md#repository-boundary).

## Edge cases

### X01 — Distinct historical observations

- **Preconditions:** Retain the E1–E8 prefix observation and the completed replay.
- **Action:** Query Day 2 fee-free at that prefix, posted at Day 5 close, and
  finally restated after finalization.
- **Expected outcome:** Respectively AED −370.00, −395.00, and 250.00. The first
  is a calculation, not a stored snapshot or a reason to suppress actual fees.
- **Preserved invariants:** Value-date selection and explicit observation checkpoint;
  [reporting and observations](SPEC.md#reporting-and-observations).

### X02 — Approval exactly at the availability boundary

- **Preconditions:** Independent approved-hold fixture with proposed availability
  exactly zero, accounting for all active holds; no unpaid accrual is spendable.
- **Action:** Request the hold and inspect ledger and availability.
- **Expected outcome:** Approve, leave ledger unchanged, and report zero available.
  A request producing negative availability declines as in U02.
- **Preserved invariants:** Inclusive approval boundary and hold-only reservation;
  [authorizations and settlements](SPEC.md#authorizations-and-settlements).

### X03 — Interest at zero, negative balances, and exact ties

- **Preconditions:** Separate zero/negative bases and the AED/BHD positive midpoint
  fixtures in [NUMBERS](../NUMBERS.md#verification-parameters).
- **Action:** Calculate each daily accrual and inspect unpaid state before capitalization.
- **Expected outcome:** Zero for nonpositive bases; both even-tie directions match
  the tabulated exact results. Positive accrued amounts remain unpaid state,
  without accrual ledger records or increased availability.
- **Preserved invariants:** Currency-scale HALF_EVEN only for calculated interest;
  [interest and financial restoration](SPEC.md#interest-and-financial-restoration).

### X04 — Harmless trailing zeros and exact endpoint formatting

- **Preconditions:** AED 1.000, BHD 1.0000, and representable `int64` endpoints.
- **Action:** Parse/store/format each fixture.
- **Expected outcome:** Accept the redundant zeros, retain raw input text, and
  format stored values at their currency scale. Both endpoints format exactly,
  including the minimum without unsafe negation.
- **Preserved invariants:** Principal value and source fidelity;
  [monetary validation and errors](SPEC.md#monetary-validation-and-errors),
  endpoint mechanism in [DESIGN](DESIGN.md#money-representation).

### X05 — Fee cannot sustain its own eligibility

- **Preconditions:** A historical assessment exists and a correction removes its
  underlying negative eligibility balance, as in E9.
- **Action:** Reconcile the affected day and subsequent closed days chronologically.
- **Expected outcome:** Refund the now-unjustified fee; its own old charge cannot
  keep it due. Earlier days' net corrections flow into subsequent balances,
  accruals, and subsequent authorization availability.
- **Preserved invariants:** Own-fee exclusion applies to eligibility, not reported
  ledger balances; [fees](SPEC.md#fees).

### X06 — Repeat internal processing without new effects

- **Preconditions:** Recorded E7 reconciliation, E9 correction, or completed day close.
- **Action:** Repeat the internal processing for the same cause; also repeat Day 6 finalization.
- **Expected outcome:** Unchanged history, net fees, daily accruals, and report;
  no extra reversal, refund, diagnostic, or capitalization. Rebuilt accruals replace
  previous derived values rather than accumulating adjustments.
- **Preserved invariants:** Idempotent causes/obligations and unchanged original
  outcomes; [history and replay](SPEC.md#history-and-replay),
  [fees](SPEC.md#fees), [Day 6 finalization](SPEC.md#day-6-finalization).

### X07 — Fee eligibility changes again

- **Preconditions:** Isolated historical fee obligation already assessed and
  refunded; another valid ordinary posting makes that closed day eligible again.
- **Action:** Reconcile the day and repeat unchanged reconciliation.
- **Expected outcome:** A linked adjustment restores the required net fee under
  the existing obligation; no second independent assessment. The repeat adds nothing.
- **Preserved invariants:** One account/day assessment identity across eligibility
  changes; [fees](SPEC.md#fees). No additional reversal workflow is assumed.

### X08 — Day 6 capitalization crosses the fee boundary

- **Preconditions:** The separate AED fixtures in
  [Day 6 boundaries](../NUMBERS.md#day-6-boundaries), with prior accrual 0.02.
- **Action:** Finalize bases −0.01, −0.02, and −0.03 independently.
- **Expected outcome:** Final closes 0.01, 0.00, and −25.01, respectively;
  no fee in the first two cases and the specified charge in the last. Current-day
  accrual is zero; no provisional fee is posted merely because the base is negative.
- **Preserved invariants:** Capitalization-inclusive eligibility, zero boundary,
  and capitalization-exclusive interest; [Day 6 finalization](SPEC.md#day-6-finalization).

### X09 — Rebuild daily accruals and reconcile capitalization exactly

- **Preconditions:** Daily accrual state exists before E7, after E7/Day 5 close,
  and after E9/E10 corrections.
- **Action:** Inspect replacements at each checkpoint, then sum corrected rounded
  daily values independently and compare to each capitalization posting.
- **Expected outcome:** Values match the [checkpoint](../NUMBERS.md#checkpoint-calculations)
  and [final](../NUMBERS.md#expected-replay-results) oracles, including restored
  AED accruals and late BHD Day 5 interest. Exact equality holds without a discarded
  remainder or a separately rounded aggregate of unrounded interest.
- **Preserved invariants:** No double accrual or daily monetary adjustments;
  [interest and financial restoration](SPEC.md#interest-and-financial-restoration),
  [Day 6 finalization](SPEC.md#day-6-finalization).

### X10 — Exact maximum-base interest and currency isolation

- **Preconditions:** Maximum positive representable minor-unit base, and money
  operands carrying different codes or different scales, in isolated arithmetic checks.
- **Action:** Calculate the base's daily interest; separately attempt cross-currency arithmetic.
- **Expected outcome:** Interest is exact and safely rounded without an overflowing
  rate intermediate. Cross-currency arithmetic is invalid; no conversion is inferred.
  Exact expectations derive independently from [NUMBERS](../NUMBERS.md#int64-representation).
- **Preserved invariants:** Exact money/rate calculations per
  [monetary validation and errors](SPEC.md#monetary-validation-and-errors);
  representation checks per [DESIGN](DESIGN.md#money-representation).

### X11 — Protect stored history and atomic repository batches

- **Preconditions:** Stored input, outcome, monetary, and authorization records;
  a detached history snapshot and a proposed batch with a conflicting record key.
- **Action:** Attempt to mutate submitted/read-back containers; independently
  attempt the conflicting append and rebuild disposable projections.
- **Expected outcome:** Stored originals are unchanged; the conflicting batch
  appends no records. Rebuilding retains monetary history and original decisions.
- **Preserved invariants:** Append-only history per [history and replay](SPEC.md#history-and-replay);
  defensive copies, key uniqueness, and batch atomicity per
  [DESIGN](DESIGN.md#repository-boundary). This is separate from duplicate input rejection.

### X12 — Configurable currency precision and identity

- **Preconditions:** Independently constructed code/scale descriptors, including
  scales zero, four, and 18, plus invalid code/scale fixtures in
  [NUMBERS](../NUMBERS.md#configurable-currency-verification).
- **Action:** Validate descriptors; parse/format exact amounts at each scale;
  add/subtract/sum values with matching or conflicting descriptors.
- **Expected outcome:** Invalid descriptors reject. Fixed-point formatting and
  int64 endpoints remain exact at configurable scales. Equal code/scale values
  interoperate across instances; different codes or scales reject arithmetic.
- **Preserved invariants:** No built-in currency enumeration or implicit conversion;
  [monetary validation](SPEC.md#monetary-validation-and-errors). P05 also covers
  account/policy descriptor consistency during initialization and exact
  ordinary submissions at these scales.

## Coverage review and scope boundary

Every scenario has normal regression coverage. The table names representative
assertions; the linked files contain additional boundary and failure cases.
Unit tests sit beside their owning packages; integration tests compose real
components under `tests/replay`. Commands and placement conventions belong to
[DEVELOPMENT](DEVELOPMENT.md#test-organization).

| Scenario | Test file and representative assertion |
| --- | --- |
| H01 | [Ordinary integration](../tests/replay/ordinary_test.go): `TestOrdinaryPostings` |
| H02 | [Authorization integration](../tests/replay/authorization_test.go): `TestAuthorizationAndFinalSettlement` |
| H03 | [Authorization unit tests](../internal/ledger/authorization_test.go): `TestFinalSettlementReleasesOnlyItsEntireHold` |
| H04 | [Checkpoint integration](../tests/replay/backdating_test.go): `TestBackdatedSubmissionsReconcileBeforeNextInputAndFinalize` |
| H05 | [Checkpoint integration](../tests/replay/backdating_test.go): same test, E9 refunds/accruals and retained decisions |
| H06 | [Correction integration](../tests/replay/corrections_test.go): `TestLinkedReversalAndLateInstalments`; checkpoint integration verifies late Day 5 interest |
| H07 | [Canonical replay](../tests/replay/scenario_test.go): `TestCanonicalScenario`, exact daily sums and one credit per account |
| H08 | [Report integration](../tests/replay/report_test.go): `TestCanonicalReport`; [composition](../cmd/replay/main_test.go): `TestRunFreshReplayAndOutputFailure` |
| H09 | [Closing](../internal/ledger/closing_test.go): `TestClosingConfiguredCurrenciesAndInterestTies`; [finalization](../internal/ledger/finalization_test.go): `TestFinalizationUsesConfiguredChargeIncludingZero` |
| U01 | [Authorization integration](../tests/replay/authorization_test.go): `TestAuthorizationAndFinalSettlement`, E6 rejection and continuation |
| U02 | [Checkpoint integration](../tests/replay/backdating_test.go): E8 proposed availability and decline retained after E9/finalization |
| U03 | [Submission](../internal/ledger/submission_test.go): `TestOrdinaryInputRejectionsAndContinuation`; [instalments](../internal/ledger/instalments_test.go): `TestInstalmentRejectionsAndContinuation` |
| U04 | [Submission](../internal/ledger/submission_test.go): `TestDuplicateInputsPreserveOriginalOutcomes`; [authorization](../internal/ledger/authorization_test.go): `TestAuthorizationAndSettlementDuplicateOutcomes`; [canonical replay](../tests/replay/scenario_test.go): full resubmission |
| U05 | [Configuration](../internal/ledger/configuration_test.go): `TestConfigurationValidationBeforeAppend`; [composition](../cmd/replay/main_test.go): `TestInvalidCompositionStopsBeforeReplayAndReport` |
| U06 | [Money](../internal/domain/money_test.go): `TestParseMoneyRepresentabilityErrors`; [submission](../internal/ledger/submission_test.go): `TestOrdinaryInputRejectionsAndContinuation` |
| U07 | [Money](../internal/domain/money_test.go): `TestCheckedArithmetic`, `TestSum`; [submission](../internal/ledger/submission_test.go): `TestDerivedPostingFailuresAreAtomic`; [backdating](../internal/ledger/backdating_test.go): `TestSubmissionFeeOverflowRetainsCommittedBatches`; [finalization](../internal/ledger/finalization_test.go): `TestFinalizationFailuresAppendNothing`, `TestFinalizationAccrualSumOverflow` |
| X01 | [Canonical replay](../tests/replay/scenario_test.go): Day 2 prefix fee-free/posted/final balances; [projection](../internal/projection/projection_test.go): `TestBalanceObservations` |
| X02 | [Authorization](../internal/ledger/authorization_test.go): `TestAuthorizationAvailabilityBoundaries`, `TestAuthorizationIncludesPostedFeesExcludesUnpaidInterestAndFutureValues` |
| X03 | [Interest](../internal/interest/interest_test.go): `TestDailyInterest`; [closing](../internal/ledger/closing_test.go): `TestClosingConfiguredCurrenciesAndInterestTies` |
| X04 | [Money](../internal/domain/money_test.go): `TestParseMoney`, `TestMoneyFormat`; [submission](../internal/ledger/submission_test.go): `TestOrdinaryPostingAndSourceRetention`; [report](../internal/report/report_test.go): `TestFormatUsesSuppliedOrderAndPrecision` |
| X05 | [Closing](../internal/ledger/closing_test.go): `TestFeeSelfEligibilityRenewalAndEarlierFeeCarry`; checkpoint integration verifies automatic E9 refunds |
| X06 | [Correction builder](../internal/posting/corrections_test.go): `TestPrincipalCorrectionIdempotenceAndConflictingCause`; checkpoint integration repeats reconciliation, closing, and finalization without records |
| X07 | [Closing](../internal/ledger/closing_test.go): `TestFeeSelfEligibilityRenewalAndEarlierFeeCarry`, two refund/renewal cycles |
| X08 | [Finalization](../internal/ledger/finalization_test.go): `TestFinalizationCapitalizationFeeBoundaries`, `TestFinalizationReconcilesExistingOwnFee` |
| X09 | [Checkpoint integration](../tests/replay/backdating_test.go): replacement accruals before/after E7/E9/E10 and exact final capitalization; [finalization](../internal/ledger/finalization_test.go): `TestFinalizationExcludesCapitalizationFromSameDayInterest` |
| X10 | [Interest](../internal/interest/interest_test.go): maximum-base case in `TestDailyInterest`; [money](../internal/domain/money_test.go): `TestCurrencyChecksOnOperations` |
| X11 | [Repository](../internal/repository/memory/repository_test.go): `TestHistoryOwnership`, `TestAppendAtomicValidation`; [projection](../internal/projection/projection_test.go): `TestProjectionOwnsHistoryAndReturnedContainers`; [canonical replay](../tests/replay/scenario_test.go): original records/outcomes survive corrections |
| X12 | [Currency](../internal/domain/currency_test.go): `TestCurrencyValidation`, `TestMoneyConfigurablePrecision`, `TestCurrencyCompatibility`; [submission](../internal/ledger/submission_test.go): `TestOrdinaryConfiguredPrecision`; [configuration](../internal/ledger/configuration_test.go): `TestConfigurationAcceptsCustomCurrencies` |

All eight criterion verdicts remain those in [REJECTED](../REJECTED.md): H/X/U
scenarios cover criterion 1 via X01; 2 via H04; 3 via H03; 4 via U01; 5 via
H02/X02 (its approved premise does not occur for Auth-B); 6 via H05; 7 via H06;
and 8 via X09. Refused claims do not become expected outcomes.

The coverage review found no additional supplied-replay policy gap. Execution
outcomes are recorded in [WORKLOG](../WORKLOG.md). E5's final settlement does not
define a general partial-capture policy. Auth-B remains declined, so there is no
active hold or expiry to report.

Differing authorization booking/value dates, explicit partial/over-captures,
expiry/cancellation, inactive-ID settlements, and additional unexercised input
validation categories stay outside this catalog's asserted behavior; see
[deferred scope](SPEC.md#verification-and-deferred-work) and
[validation scope](SPEC.md#monetary-validation-and-errors). Alternate BHD tariffs,
exchange rates, retry/recovery protocols, performance targets, and concurrent writers
remain outside scope. The design remains sequential and in memory.

## Design-test candidates

The [assignment](TASK.md#deliverable-1--repository) requires a real failing test
against the chosen design. [SPEC](SPEC.md#verification-and-deferred-work) defines
its contract; [DESIGN](DESIGN.md#repository-boundary) owns the atomicity boundary
and trade-off. The scenarios above must pass their
regression assertions, including U07's expected technical failure. A regression
bug should be fixed rather than preserved merely to fill this deliverable.

Credible investigation candidates arising from the current technical choices are:

| Candidate | Evidence to seek before selecting a failing assertion | Boundary on the claim |
| --- | --- | --- |
| Interruption during multi-batch historical reconciliation | Exercise a real technical failure between committed batches; inspect whether retained history and rebuilt projections expose incomplete financial correction and whether it can be completed using the existing interface. | Per-batch atomicity does not establish whole-operation rollback or automatic recovery. The explicit recovery rule belongs to DESIGN's repository boundary. |
| Fixed-width accumulation sensitivity | Use independently derived, individually representable postings whose mathematical aggregate fits but an intermediate sum does not; inspect whether the implemented history scan cannot produce that representable projection. | DESIGN explicitly requires intermediates to fit. An expected overflow is a passing U07 check, not automatically a failing design test. A separate probe would expose a documented representation limitation, not a violated business requirement or a new promise of unbounded money. |

P13 selected the interruption candidate in the tagged
[integration probe](../tests/replay/designlimitation_test.go). Its useful property
is financial atomicity of a single failed submission: callers could handle a
technical error without observing partially reconciled financial effects. The
test interrupts the second fee append and compares the immediate balance with
the unchanged pre-submission zero oracle in
[NUMBERS](../NUMBERS.md#interrupted-reconciliation-probe). Its annotated assertion
fails because prior batches survive. The normal interruption regressions verify
explicit recovery separately. No duplicate input is submitted. This documents
the atomicity boundary without changing U07, U04, or X06.
The fixed-width lead was not selected: its existing `MaxInt64 + 1 − 1`
projection fixture expects intermediate overflow under U07, which alone supplies
no additional failing property. The earlier aggregate-rounding proposal was
withdrawn at explicit request before implementation (WORKLOG,
2026-10-31T17:24:29Z): an artificial assertion contradicting agreed output did
not establish a genuine design limitation. Superseded implementations and their
replacement reasons belong to [REJECTED](../REJECTED.md#abandoned-approaches). See
[DEVELOPMENT](DEVELOPMENT.md#replay-and-isolated-failing-test) for isolated commands
and the required nonzero assertion result.
