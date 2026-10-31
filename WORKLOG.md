# Worklog

Earlier entries are shortened actual milestones; current decisions are linked
rather than repeated. No ledger implementation or ledger tests have run.

- 2026-10-31T13:56:42Z — Added repository instructions and preserved [TASK](docs/TASK.md); linked the documentation from README.
- 2026-10-31T13:58:17Z — Initial source-fidelity, link, event-order, and whitespace checks passed.
- 2026-10-31T14:11:19Z — Revised repository instructions after human review and condensed the log without changing facts.
- 2026-10-31T14:24:36Z — Created SPEC and supporting decision/number records; reviewed all criteria provisionally.
- 2026-10-31T14:25:53Z — Documentation links, task order, criterion coverage, and whitespace checks passed.
- 2026-10-31T15:00:43Z — Corrected the unsupported unmatched-settlement alternative after human review; see [A3](AMBIGUITIES.md#a3--unmatched-settlement).
- 2026-10-31T15:11:10Z — Recorded the human's reversal/restoration interpretation; see [A4](AMBIGUITIES.md#a4--backdating-and-reversal). Documentation checks passed.
- 2026-10-31T15:14:25Z — Recorded final-settlement treatment; see [A7](AMBIGUITIES.md#a7--hold-lifecycle). Independent arithmetic passed.
- 2026-10-31T15:22:00Z — Recorded historical-correction timing and its availability consequences under [A4](AMBIGUITIES.md#a4--backdating-and-reversal); checked arithmetic.
- 2026-10-31T15:38:53Z — Recorded the rounding decision under [A6](AMBIGUITIES.md#a6--interest-rounding-representation-and-finalization); verified midpoint examples.
- 2026-10-31T15:40:47Z — Recorded residual placement under [A9](AMBIGUITIES.md#a9--currency-precision-and-instalments); checked conservation.
- 2026-10-31T15:45:05Z — Recorded the report selection and preserved authorization outcomes; see [A8](AMBIGUITIES.md#a8--observations-and-report-interpretation).
- 2026-10-31T15:57:28Z — Recorded daily calculation order under [A5](AMBIGUITIES.md#a5--fee-bases-and-missing-bhd-tariff); independently checked financial checkpoints.
- 2026-10-31T16:12:22Z — Recorded finalization bases under [A6](AMBIGUITIES.md#a6--interest-rounding-representation-and-finalization); checked final totals and boundaries.
- 2026-10-31T16:24:58Z — Recorded criterion 1's observation interpretation under [A8](AMBIGUITIES.md#a8--observations-and-report-interpretation); completed all [verdicts](REJECTED.md).
- 2026-10-31T16:27:42Z — Recorded missing-policy handling under [A5](AMBIGUITIES.md#a5--fee-bases-and-missing-bhd-tariff); documentation checks passed.
- 2026-10-31T16:31:03Z — Recorded the accrual representation choice under [A6](AMBIGUITIES.md#a6--interest-rounding-representation-and-finalization); removed stale questions.
- 2026-10-31T16:36:50Z — Consolidated routine consequences and independently rechecked the expected report and retained outcomes.
- 2026-10-31T16:43:49Z — Recorded duplicate rejection under [A10](AMBIGUITIES.md#a10--duplicate-ids-and-errors); checked policy consistency.
- 2026-10-31T16:47:54Z — Completed the supplied-replay specification and input-validation decision, documented a test proposal, and checked numerical oracles/source fidelity. No ledger test was built or run.
- 2026-10-31T17:24:29Z — Removed the artificial failing-test proposal and deferred selection to implementation; see [A11](AMBIGUITIES.md#a11--required-failing-test). Deduplicated document roles and shortened milestones. Links/anchors, source fidelity, preserved event/result/fee tables and timestamps, exact arithmetic, and whitespace checks passed. No implementation or ledger tests run.
- 2026-10-31T17:31:21Z — Drafted the [minimal technical design](docs/DESIGN.md) using the human's Go preference; recorded proposed storage/arithmetic choices and linked the design from README. Business decisions remain unchanged; no implementation added.
- 2026-10-31T17:31:40Z — Reviewed the design against SPEC and all criterion verdicts; 92 local links/anchors and source-fidelity checks passed. Reviewed tracked/new-file diffs and whitespace. No ledger replay or executable tests run.
- 2026-10-31T17:37:14Z — Revised the [money design](docs/DESIGN.md#language-and-tooling) following the human's `int64` preference; recorded checked arithmetic, representation bounds, error handling, and boundary verification. Replaced the earlier `big.Int` proposal; no implementation added.
- 2026-10-31T17:38:12Z — Documentation checks passed: 96 local links/anchors, exact integer bounds, whitespace, and unchanged TASK/SPEC/criterion verdicts. No ledger tests run.
- 2026-10-31T17:48:11Z — Added [USE_CASES](docs/USE_CASES.md) with 26 happy/unhappy/edge scenarios, coverage review, and evidence-dependent design-test candidates; linked it from README. Checked all scenario fields/spec references, 157 local links/anchors, unchanged TASK/SPEC/DESIGN/decision records, and tracked/new-file diffs and whitespace. No implementation or ledger tests run; no failure manufactured or selected.
