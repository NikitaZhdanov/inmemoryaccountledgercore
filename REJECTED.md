# Acceptance-criterion review

All eight supplied claims are assessed independently. Refuse 2, 7, and 8;
accept 1 and 6 under the documented interpretations; accept 3 and 4;
accept 5 conditionally. TASK.md remains unchanged.

| # | Verdict | Reason and supporting record |
| --- | --- | --- |
| 1 | Correct under the agreed interpretation | A fee-free prefix calculation is distinct from a posted snapshot; see [observation rationale](AMBIGUITIES.md#a8--observations-and-report-interpretation) and [arithmetic](NUMBERS.md#checkpoint-calculations). |
| 2 | Incorrect; refuse | Several assessed days are negative. Charging only one day violates the daily-fee rule; see [chronological calculations](NUMBERS.md#checkpoint-calculations). Refunds do not erase assessments. |
| 3 | Correct | The agreed final-settlement interpretation accepts E5; see [hold decision](AMBIGUITIES.md#a7--hold-lifecycle). |
| 4 | Correct | Unknown-authorization rejection conflicts with no non-negotiable rule; see [the corrected review](AMBIGUITIES.md#a3--unmatched-settlement). |
| 5 | Correct conditional statement | The hold invariant is required, although Auth-B's actual decline leaves the premise untriggered; see [authorization behavior](docs/SPEC.md#authorizations-and-settlements). |
| 6 | Correct under the agreed interpretation | Financial restoration preserves record history and original authorization decisions; see [reversal rationale](AMBIGUITIES.md#a4--backdating-and-reversal). |
| 7 | Incorrect; refuse | The proposed equal rounded-up credits exceed the supplied total; see [conservation arithmetic](NUMBERS.md#instalment-conservation). |
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
  workflows. See the [package-boundary decision](AMBIGUITIES.md#a16--ledger-package-boundaries).
- Replaced technical `ErrUnsupportedInput` handling for a different attempt to
  reverse an already reversed debit with a recorded business rejection, following
  the [explicit clarification](AMBIGUITIES.md#a3--backdating-and-reversal).
- Replaced handwritten assertion/error checks with Testify `assert`/`require`
  to adopt the project's selected assertion library. Retained Go's test runner,
  fixture tables, and exact expectations; see [A13](AMBIGUITIES.md#a13--testify-unit-test-assertions).
- Replaced the implemented AED/BHD currency constants and scale switch with
  validated code/scale descriptors so currency definitions can be supplied at
  composition; see [A14](AMBIGUITIES.md#a14--configured-currency-identity-and-initialization).

Earlier specification and design alternatives are recorded in
[A2](AMBIGUITIES.md#a2--storage-boundary),
[A3](AMBIGUITIES.md#a3--unmatched-settlement),
[A10](AMBIGUITIES.md#a10--duplicate-ids-and-errors).
The superseded runtime undefined-BHD-policy proposal is recorded in
[A5](AMBIGUITIES.md#a5--fee-bases-and-missing-bhd-tariff).
