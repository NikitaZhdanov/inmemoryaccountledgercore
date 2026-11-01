# Acceptance-criterion review

Refuse criteria 2, 7, and 8. Criteria 1 and 6 require the qualifications below;
6's literal state-reset interpretation is refused. Accept 3 and 4 under the
recorded settlement policy; 5 is a valid conditional. [TASK](docs/TASK.md) is
preserved unchanged.

| # | Verdict | Reason and supporting record |
| --- | --- | --- |
| 1 | Correct as a fee-free calculation | E1–E8 gives 250.00 − 620.00 = AED −370.00 before fees. The actual posted Day 2 balance at Day 5 close is −395.00. See [observation](AMBIGUITIES.md#a7--observations-and-report-interpretation). |
| 2 | Incorrect; refuse | E7 causes AED 25.00 assessments for Days 2, 4, and 5, totaling 75.00. Charging only Day 2 violates the daily-fee rule; see [chronological calculations](NUMBERS.md#checkpoint-calculations). Later refunds do not erase assessments. |
| 3 | Correct | E5 debits AED 185.00 and releases the full AED 200.00 hold under the agreed final-settlement interpretation; see [hold decision](AMBIGUITIES.md#a6--hold-lifecycle). |
| 4 | Correct under the selected settlement policy | E6 is rejected without debit because Auth-Z has no recorded approval. The task supplies no force-posting rule. See [policy](AMBIGUITIES.md#a2--unmatched-settlement). |
| 5 | Correct conditional statement | The hold invariant is required, although Auth-B's actual decline leaves the premise untriggered; see [authorization behavior](docs/SPEC.md#authorizations-and-settlements). |
| 6 | Conditional; literal reset refused | E9/corrections restore E7's financial effects: AED 465.00 before capitalization, zero net fees, and corrected accruals. They do not erase assessments or change Auth-B's decline. See [restoration](AMBIGUITIES.md#a3--backdating-and-reversal). |
| 7 | Incorrect; refuse | Three BHD 3.334 credits total 10.002, exceeding 10.000. Use 3.333, 3.333, 3.334; see [conservation arithmetic](NUMBERS.md#instalment-conservation). |
| 8 | Incorrect; refuse | Discarding a remainder violates exact reconciliation of rounded daily accruals to capitalization; see [finalization](docs/SPEC.md#day-6-finalization). |

## Abandoned approaches

These are superseded implementations or tests, not additional task requirements.

| Superseded approach | Replacement and reason |
| --- | --- |
| One broad ledger business-logic package | Split posting, projection, and interest calculations from commit workflows; see [boundaries](docs/DESIGN.md#packages-and-dependencies). |
| Technical error for a fresh reversal of an already reversed debit | Recorded business rejection following [explicit clarification](AMBIGUITIES.md#a3--backdating-and-reversal). |
| Temporary technical error for nonpositive authorization/settlement amounts | Shared nonnegative validation following [A15](AMBIGUITIES.md#a15--authorization-and-settlement-amount-signs): zero is allowed; negatives produce business rejections. |
| Fixed Day 5 cutoff in ordinary closing | Caller-supplied schedule; the six-day scenario does not set a core API limit. See [processing](docs/DESIGN.md#processing-and-reconciliation). |
| Handwritten assertion/error checks | Testify under the selected [test convention](docs/DEVELOPMENT.md#assertion-convention), retaining exact expectations and Go's runner. |
| Built-in AED/BHD constants and scale switch | Injected currency descriptors under [configuration design](docs/DESIGN.md#configuration-validation). |

Unimplemented alternatives remain with their decisions: [big.Int representation](docs/DESIGN.md#money-representation),
[force-posting](AMBIGUITIES.md#a2--unmatched-settlement),
[retry protocol](AMBIGUITIES.md#a9--duplicate-ids-and-errors), and
[undefined BHD tariff](AMBIGUITIES.md#a4--fee-bases-and-missing-bhd-tariff).
The withdrawn aggregate-rounding test proposal is recorded in
[USE_CASES](docs/USE_CASES.md#design-test-candidates).
