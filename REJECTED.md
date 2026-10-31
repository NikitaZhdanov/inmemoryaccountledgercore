# Acceptance-criterion review

All eight supplied claims are assessed independently. Refuse 2, 7, and 8;
accept 1 and 6 under the documented human interpretations; accept 3 and 4;
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

No build approach was abandoned: implementation has not begun. Discarded
specification proposals, rather than invented build attempts, are recorded with
their replacements and reasons in [A3](AMBIGUITIES.md#a3--unmatched-settlement),
[A10](AMBIGUITIES.md#a10--duplicate-ids-and-errors), and
[A11](AMBIGUITIES.md#a11--required-failing-test).

The initial technical design proposed `math/big.Int` money to avoid a fixed-width
range. Human review favored simpler `int64` value semantics; the revised
[design](docs/DESIGN.md#language-and-tooling) replaces that proposal with checked
integer arithmetic and explicit representation-error handling. This was a design
revision, not an implemented approach that failed a test.
