# Acceptance-criterion review

All eight supplied claims are assessed independently. Refuse 2, 7, and 8;
accept 1 and 6 under the documented interpretations; accept 3 and 4;
accept 5 conditionally. TASK.md remains unchanged.

| # | Verdict | Reason and supporting record |
| --- | --- | --- |
| 1 | Correct under the agreed interpretation | Fee-free E1–E8 gives 250.00 − 620.00 = AED −370.00; posted Day 2 at Day 5 close is −395.00. See [observation rationale](AMBIGUITIES.md#a7--observations-and-report-interpretation) and [arithmetic](NUMBERS.md#checkpoint-calculations). |
| 2 | Incorrect; refuse | E7 causes AED 25.00 assessments for Days 2, 4, and 5, totaling 75.00. Charging only Day 2 violates the daily-fee rule; see [chronological calculations](NUMBERS.md#checkpoint-calculations). Later refunds do not erase assessments. |
| 3 | Correct | E5 debits AED 185.00 and releases the full AED 200.00 hold under the agreed final-settlement interpretation; see [hold decision](AMBIGUITIES.md#a6--hold-lifecycle). |
| 4 | Correct | E6 records an unknown-authorization rejection without debit; see [unmatched settlement](AMBIGUITIES.md#a2--unmatched-settlement). |
| 5 | Correct conditional statement | The hold invariant is required, although Auth-B's actual decline leaves the premise untriggered; see [authorization behavior](docs/SPEC.md#authorizations-and-settlements). |
| 6 | Correct under the agreed interpretation | E9/corrections restore financial balances, net fees, and accruals before capitalization. They preserve E7, assessments, and the original Auth-B decline; a literal state reset is incorrect. See [reversal rationale](AMBIGUITIES.md#a3--backdating-and-reversal). |
| 7 | Incorrect; refuse | Three BHD 3.334 credits total 10.002, exceeding 10.000. Use 3.333, 3.333, 3.334; see [conservation arithmetic](NUMBERS.md#instalment-conservation). |
| 8 | Incorrect; refuse | Discarding a remainder violates exact reconciliation of rounded daily accruals to capitalization; see [finalization](docs/SPEC.md#day-6-finalization). |

## Abandoned approaches

- Rejected P13's initial expectation that a duplicate/retried `Submit` would
  complete missing financial work. This directly violated
  [A9](AMBIGUITIES.md#a9--duplicate-ids-and-errors), which requires duplicate
  rejection with zero financial side effects. Replaced that test with a single
  submission's post-failure atomicity assertion against the
  [repository boundary](docs/DESIGN.md#repository-boundary). Production behavior
  is unchanged.
- Replaced the broad ledger business-logic package with focused posting,
  projection, and interest packages; ledger retains configuration and commit
  workflows. See the [package boundaries](docs/DESIGN.md#packages-and-dependencies).
- Replaced technical `ErrUnsupportedInput` handling for a different attempt to
  reverse an already reversed debit with a recorded business rejection, following
  the [explicit clarification](AMBIGUITIES.md#a3--backdating-and-reversal).
- Removed the implemented Day 5 cutoff from ordinary closing/reconciliation
  because the caller owns its schedule. Alternate Day 6/7 regressions verify the
  replacement; see [processing](docs/DESIGN.md#processing-and-reconciliation).
- Replaced handwritten assertion/error checks with Testify `assert`/`require`
  to adopt the project's selected assertion library. Retained Go's test runner,
  fixture tables, and exact expectations; see [assertion convention](docs/DEVELOPMENT.md#assertion-convention).
- Replaced the implemented AED/BHD currency constants and scale switch with
  validated code/scale descriptors so currency definitions can be supplied at
  composition; see [configuration design](docs/DESIGN.md#configuration-validation).

Earlier specification and design alternatives are recorded in
[money representation](docs/DESIGN.md#money-representation),
[A2](AMBIGUITIES.md#a2--unmatched-settlement),
[A9](AMBIGUITIES.md#a9--duplicate-ids-and-errors).
The superseded runtime undefined-BHD-policy proposal is recorded in
[A4](AMBIGUITIES.md#a4--fee-bases-and-missing-bhd-tariff).
