# inmemoryaccountledgercore
In Memory Account Ledger Core

The full task description is preserved in [docs/TASK.md](docs/TASK.md).
Repository working instructions are in [AGENTS.md](AGENTS.md), and actual work is
recorded in [WORKLOG.md](WORKLOG.md).

The supplied replay's business specification is defined in
[docs/SPEC.md](docs/SPEC.md), with decisions and explicitly unspecified extensions
in [AMBIGUITIES.md](AMBIGUITIES.md), acceptance-criterion review in
[REJECTED.md](REJECTED.md), and constants in [NUMBERS.md](NUMBERS.md).

The repository currently contains documentation only. The ledger implementation
and runnable replay suite have not yet been built.

The specification includes independently calculated expected results. Planned
output is a final daily table restated after the entire stream and Day 6
finalization, with original authorization decisions, errors, and linked fee
assessment/refund history. It is not a table of originally observed balances.
Runtime setup, replay/test commands, and the isolated intentional-failure
command will be added when those programs exist.
