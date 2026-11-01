# Repository instructions

## Workflow and sources of truth

- Read [docs/TASK.md](docs/TASK.md) before planning or changing implementation.
  Preserve the supplied task; do not rewrite it to fit the implementation.
  Evaluate acceptance criteria independently against its non-negotiable rules.
- Follow agreed behavior in [docs/SPEC.md](docs/SPEC.md), technical boundaries in
  [docs/DESIGN.md](docs/DESIGN.md), and the work sequence in
  [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md). Use
  [docs/USE_CASES.md](docs/USE_CASES.md) and [NUMBERS.md](NUMBERS.md) for coverage
  and independent expected results.
- Distinguish explicit requirements, agreed decisions, proposals, and unresolved
  questions. Request clarification before assuming unclear requirements or
  business behavior. Record the resolution in [AMBIGUITIES.md](AMBIGUITIES.md)
  before implementing dependent behavior; continue independent work while waiting.
  Keep decisions resolved by the brief or explicit clarification closed.

## Documentation maintenance

Give each document a clear purpose and ownership of its content; other documents
reference that owner instead of repeating it. Keep DESIGN focused on architecture.
Use standalone wording and avoid conversational phrasing or repetitive assurances.
Update documentation proportionally to the change, including affected references.

Update owning records alongside code and tests, as work happens:

- [WORKLOG.md](WORKLOG.md): keep short entries for meaningful completed changes,
  decisions, and observed verification outcomes. Use observed ISO 8601 timestamps
  with a timezone, preferably UTC. Link to decision records; omit command
  transcripts, routine inspection, repeated plans, and duplicated rationale.
- [NUMBERS.md](NUMBERS.md): inventory every chosen constant and test parameter
  with meaning, units, source or rationale, and why that value rather than half.
  Explain when halving is meaningless; distinguish mandated values from choices,
  and include derivation formulas and rounding rules. Avoid unexplained magic
  numbers in code and tests.
- [AMBIGUITIES.md](AMBIGUITIES.md) is a decision record, not a changelog or catch-all.
  Add or revise entries only to resolve a genuine ambiguity or consequential
  trade-off; routine implementation changes belong in their owning records.
  Distinguish task ambiguities from project design choices. Keep alternatives,
  resolution, and rationale concise, retain unresolved questions explicitly, and
  link to behavior/design/test owners instead of repeating consequences across
  documents.
- [REJECTED.md](REJECTED.md): assess every supplied acceptance criterion, explicitly
  refuse incorrect claims with reasons, and distinguish correct and conditional
  interpretations. Record actual superseded implementations with replacement
  reasons and decision links; keep earlier design alternatives in their decision
  records.
- [README.md](README.md): keep a concise entry point with a readable title,
  purpose, current status, minimal requirements, working quick-start commands,
  main Make targets, and grouped documentation links. Keep detailed tooling, CI,
  compatibility, troubleshooting, output interpretation, and the isolated failing
  test command in [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md); keep API details in
  code documentation. Omit milestone narratives and commands for features
  that do not exist.
- Keep product behavior in SPEC, architecture in DESIGN, API details in code,
  decisions in AMBIGUITIES, and task status in IMPLEMENTATION_PLAN. Create records
  when their first relevant decision is made and keep them consistent in the same
  change.
- Update AGENTS.md only for durable changes to how contributors work. Keep
  product behavior, design decisions, configuration values, history, and progress
  in their owning documents. Avoid duplicating requirements across files; link
  to the owning document instead.

## Documentation and comments

- Write standalone engineering records: state the requirement or decision, its
  recorded reason, and observable consequences without requiring conversation
  context. Describe project choices directly and distinguish requirements,
  conventions, proposals, and unresolved questions.
- Preserve facts, uncertainties, historical timestamps, and check outcomes.
  Use only recorded rationale; identify a convention or explicit requirement
  when no technical rationale is recorded. Summarize without changing facts and
  correct factual errors explicitly.
- State actual changes, limitations, and observed checks. Express verification
  requirements as concrete assertions, evidence, and completion conditions; omit
  hypothetical assurances.

## Verification and completion

- Run checks appropriate to each change using [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)
  and the [Makefile](Makefile). Use meaningful tests with expected values derived
  independently of the implementation; follow the coverage requirements in
  [docs/SPEC.md](docs/SPEC.md#verification-and-deferred-work) and
  [docs/USE_CASES.md](docs/USE_CASES.md).
- For documentation-only changes, verify local links/anchors, source fidelity,
  consistency, and diffs. Verify that README quick-start commands and listed
  Make targets exist and match their descriptions.
- Follow the [failing-test requirements](docs/SPEC.md#verification-and-deferred-work)
  and [isolated command guidance](docs/DEVELOPMENT.md#planned-replay-and-isolated-failing-test).
  Preserve the annotated failing assertion; run it separately from the normal
  suite and record its observed nonzero result.
- Record only checks actually run in WORKLOG; distinguish pending verification.
  Mark deliverables complete only when their contents and required checks are
  complete. Before calling the ledger complete, audit every TASK deliverable,
  review all criterion verdicts, run replay and normal tests, and run the
  intentional failing test separately. Report actual results and limitations.

## Commit history

- Preserve intact commit history without squashing, as required by TASK.
  The assignment does not impose a blanket ban on amend or rebase; assess any
  history edit against history preservation and explicit project instructions.
