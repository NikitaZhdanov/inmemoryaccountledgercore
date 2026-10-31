# Repository instructions

## Scope and source of truth

- Read [docs/TASK.md](docs/TASK.md) before planning or changing the implementation. It preserves the supplied task; do not silently rewrite it to fit the implementation.
- Build an in-memory account ledger core with a runnable replay suite or script. No web layer, persistence, UI, or database. Any language is allowed; record the language and dependency choices.
- Treat the non-negotiable rules as constraints. The acceptance criteria deliberately contain errors: evaluate each independently instead of implementing every criterion literally.
- Keep changes within the requested scope. Documentation setup does not authorize starting the ledger implementation.

## Required working records

Maintain these repository-root Markdown files as work happens, not retrospectively at the end:

- `WORKLOG.md`: append real, timestamped entries in ISO 8601 with a timezone, preferably UTC. Record actions, decisions, actual commands/checks and outcomes, failures, and next steps where useful. Distinguish plans from completed work. Never invent timestamps, test results, or abandoned approaches. Preserve earlier entries; explicitly correct errors in a later entry.
- `NUMBERS.md`: inventory every chosen constant, including monetary values, rates, currency scales, window sizes, tolerances, limits, and test parameters. For each, state its meaning, units, source or rationale, and why that value rather than half of it. Distinguish task-mandated values from implementation choices. If halving is meaningless (for example, an identifier or discrete currency scale), explain why. Include formulas and rounding rules where numbers are derived. Avoid unexplained magic numbers in code and tests.
- `AMBIGUITIES.md`: record each ambiguity, plausible interpretations, the chosen resolution, rationale, and observable consequences with relevant tests. Keep unresolved questions explicit. A near-empty file is unacceptable. Resolve routine choices using the brief and document them; ask the user only when a material choice cannot reasonably be resolved.
- `REJECTED.md`: assess all eight acceptance criteria, identify every incorrect one, explicitly refuse it, and explain the conflict with the non-negotiable rules or arithmetic. Distinguish correct criteria, incorrect criteria, and conditional interpretations. Also record approaches actually abandoned during the build, why they failed, and what replaced them; do not invent an abandoned approach to populate the file.
- `README.md`: keep setup, exact run commands, output interpretation, and the intentional failing-test command current. Explain any required runtime or dependencies.

Create the decision records when their first relevant decision is made. Keep them consistent with code and tests in the same change. Do not create empty deliverables and describe them as complete.

## Decisions that must be explicit

Before relying on a behavior, record its interpretation in `AMBIGUITIES.md` and exercise it where material:

- Booking/replay order versus value-date order, including E10 being listed after E9 despite being booked on Day 5. Preserve the supplied stream order; explain how day closing and reporting work with that order.
- Historical balances as known at different replay checkpoints, including the meaning of “evaluated at end of Day 5” and “before any fee is assessed.”
- Whether a late backdated entry triggers historical fee assessments, when those assessments are appended, their value dates, and how assessment is made idempotent.
- Whether fees influence the fee assessment base, interest base, subsequent daily balances, and authorization decisions.
- How the AED-denominated overdraft fee applies to the BHD account if it becomes negative; do not invent an exchange rate or silently treat AED 25.00 as BHD 25.000.
- What reversal means for the original entry, historical fee assessments, and interest. Preserve the original event and use new records for any correction.
- Which balance date an authorization uses, hold lifecycle and release, partial settlement, unknown or inactive authorization IDs, and Auth-B remaining open at the end of the window.
- Currency precision, rounding mode, equal-instalment residual allocation, and which instalment receives a remainder.
- Daily interest rounding and summation, any retrospective accrual corrections, and the ordering of Day 6 fees, accrual, and capitalization, including avoiding interest on the capitalization itself.
- Error handling and replay continuation, duplicate event handling, and the distinction between immutable input events and derived ledger/state projections.

Do not present one plausible interpretation as if the task explicitly specified it. Where a criterion's verdict depends on interpretation, document that dependency.

## Implementation and verification constraints

- Use exact monetary arithmetic, such as integer minor units or an exact decimal type. Never use binary floating point for money or rate calculations. Store and round AED to 2 places and BHD to 3 places.
- Keep ledger records append-only. Reversals and corrections append new records; original events are neither mutated nor deleted. Derived projections may be rebuilt, provided the immutable record history is preserved.
- Approve a hold only when the resulting available balance is nonnegative. An active hold affects available balance, not ledger balance. Unknown-authorization settlements must produce an error without a debit.
- Assess the specified overdraft fee no more than once per account per assessed day. Explicitly define how this interacts with historical replay and reversals.
- Accrue the specified daily interest only on eligible positive closing balances. The rounded daily accruals must sum exactly to the single end-of-Day-6 capitalized credit; never discard a remainder.
- Preserve all ten event identifiers, amounts, account currencies, booking days, value dates, authorization references, and the six-day window from the brief. Do not sort away the supplied replay order or silently repair input events.
- Produce a deterministic report for every day and both accounts showing closing ledger balance, fee assessments, authorization states, and errors. Make booking time versus value-date reporting and the report's observation checkpoint clear. Format amounts at their currency precision.
- Verify arithmetic and state transitions with meaningful tests, including backdating, reversal, fee idempotence, hold release, unknown authorization rejection without debit, residual allocation, and interest reconciliation. Derive expected values independently of the code being tested.
- Include one real failing test against the chosen design, with an inline annotation explaining what it reveals. Preserve the failure rather than weakening its assertion. Isolate it under a clearly documented command so the normal regression suite can pass; document why this satisfies the failing-test deliverable and report its observed failure honestly.
- Run checks appropriate to each change. For documentation-only changes, verify links, source fidelity, and diffs; do not claim the ledger replay was tested before it exists. Record commands and actual outcomes in the worklog.

## History and completion

- Preserve intact commit history. Do not squash, amend, rebase, reset, or force-push existing history as part of this task. Make new commits only when requested or otherwise authorized.
- Before calling the ledger complete, verify every deliverable in `docs/TASK.md`, review all acceptance-criterion verdicts, run the replay and normal tests, and run the intentional failing test separately. Report actual results and any remaining limitations.
