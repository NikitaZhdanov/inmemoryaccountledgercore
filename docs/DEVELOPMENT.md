# Development

## Requirements

- The module baseline is Go 1.23.1; production code uses only the standard
  library. Unit tests use pinned [Testify](https://github.com/stretchr/testify)
  `assert`/`require` with Go's `testing`
  runner. Dependencies and checksums are recorded in [go.mod](../go.mod) and
  [go.sum](../go.sum).
- Full checks use current stable Go. The pinned analyzers require at least
  Go 1.26; Go may automatically download a newer toolchain to build them.
- Make provides the shared local/CI commands. Race detection requires a C
  compiler and CGO enabled.
- Test dependencies, analyzer/toolchain downloads, and module metadata lookups
  need network access, including some cached analyzer invocations. Vulnerability
  scanning also queries the Go vulnerability database.

No services or configuration are needed. Toolchain versions used in earlier
checks are recorded in [WORKLOG](../WORKLOG.md) and [NUMBERS](../NUMBERS.md#development-automation).

## Setup and tests

Run commands from the repository root. Go fetches test dependencies on demand;
to download them ahead of time:

```sh
go mod download
```

```sh
go test ./internal/domain
go test ./internal/interest
go test ./internal/projection
go test ./internal/posting
go test ./internal/ledger
go test ./internal/repository/memory
go test ./internal/scenario
go test ./internal/report
go test ./cmd/replay
go test ./tests/replay
go test ./...
go vet ./...
```

The focused commands run domain, policy, projection, posting, ledger, repository,
scenario, report, and executable composition tests plus real-component replay integration;
`go test ./...` runs all implemented tests.
Success prints `ok`; vet is silent on success. A failing
assertion or check returns a nonzero exit status. The money tests use literal
expected values for configurable precision, trailing zeros, signed endpoints,
code/scale compatibility, and overflow, including intermediate accumulation.
Calculation tests cover HALF_EVEN ties, maximum-base interest, and exact
instalment conservation using the [numerical inventory](../NUMBERS.md).
Repository tests cover ordered snapshots, detached containers and payloads,
atomic key/envelope validation, and retained attempts sharing an event ID.
They do not execute the later duplicate-input or other ledger policies.
Projection tests use literal history records to check observation prefixes,
value-date balances, fee exclusions/nets, retained authorization decisions and
holds, corrected accruals, capitalization exclusion, detached state, and derived
overflow. Configuration/submission tests cover explicit policy coverage,
configuration ownership, exact principal validation, source retention, duplicates,
and atomic technical failures. Authorization tests cover exact-zero/negative
proposed availability, recorded declines, posted fees, unpaid-interest exclusion,
final settlement/full hold release, duplicate outcomes, and atomic failures.
Reversal/instalment tests cover original-entry and correction-cause links,
internal correction idempotence versus duplicate input rejection, ordered exact
allocation, retained source fields, and atomic append/overflow failures.
`tests/replay` wires the real core and memory adapter for E1–E6, including unknown
settlement rejection and continuation, and for E7/E9/E10 principal posting and
currency isolation. Closing tests cover explicit Days 1–5 fee reconciliation,
linked refunds/renewed adjustments, replaced unpaid accruals, policy/currency
isolation, and append/arithmetic failures, including an interruption between
committed days. Real-storage integration also checks independent account/day
obligations. Backdating integration submits E1–E10 with explicit Days 1–5 closes
and checks financial restoration, Auth-B's retained decline, late BHD accrual,
and duplicate/correction idempotence before capitalization. Ledger tests inject
append failures at the input and successive fee batches and arithmetic failure
during automatic reconciliation, verifying retained commits and no partial
failing batch. Finalization tests cover exact rounded capitalization sums,
capitalization-inclusive fee boundaries, zero/custom charges, same-day interest
exclusion, repeat calls, and atomic append/overflow failures. The checkpoint
integration also verifies the corrected six-day accruals and final balances.
Scenario unit tests independently compare every TASK source field and verify
injected account/currency values and stopping on submission, correction, close,
or finalization failures. Canonical replay tests invoke the runner against fresh
real components and check explicit close positions, checkpoint/final numerical
oracles, retained decisions and fee/refund links, deterministic fresh runs, and
duplicate resubmission without financial or hold effects. Report tests use direct
history fixtures for booking-day lifecycle, assessed-day fee grouping, precision,
supplied ordering, read-only behavior, and errors without partial output. Replay
integration compares the full table with a literal oracle and checks retained
validation reasons. Executable tests check exact configuration, fresh-run
determinism, output failures, and constructor rejection before replay/report.

For the full local checks, using a current Go toolchain:

```sh
make ci
```

## Test organization

Place unit tests beside each package in `*_test.go`. Interest and allocation
tests live with their calculations; projection tests use literal history; posting
builder tests have no repository. Ledger workflow tests use a test-local
repository double to verify commits, failures, and orchestration. Tests composing
multiple real components belong in the `tests/replay` external test package, wiring fresh components through the same
boundaries as main rather than importing the executable.
`go test ./internal/...` runs unit tests and `go test ./tests/replay` runs integration
tests; `go test ./...` includes both. Use table-driven tests with independent exact
expectations from [NUMBERS](../NUMBERS.md) and coverage from
[USE_CASES](USE_CASES.md). Review import direction as packages are introduced;
`go list -deps ./...` must match the [package map](DESIGN.md#packages-and-dependencies).

### Assertion convention

Use Go's `testing` runner with Testify `require` for prerequisites and expected
error identities, and `assert` for independent results. Keep equality exact and
type-sensitive; use `ErrorIs`/`NotErrorIs` for wrapped sentinel errors rather than
message text. Preserve fixture tables and literal oracles when changing assertion
code. Verify assertion-dependency changes with normal tests on the module baseline
and full checks on the automation toolchain. The choice is recorded in
[A13](../AMBIGUITIES.md#a13--testify-unit-test-assertions).

## Make targets and analyzers

The [Makefile](../Makefile) is the command source of truth.

| Target | Checks or output |
| --- | --- |
| `make` / `make check` | Formatting check, build, vet, Staticcheck lint, normal tests |
| `make ci` | Formatting check, build, vet, lint, uncached race tests/coverage, vulnerability scan |
| `make fmt` / `make fmt-check` | Format all Go files / reject unformatted files without modifying them |
| `make test` / `make test-race` | Normal tests / uncached race tests writing ignored `coverage.out` |
| `make coverage` | Run race tests and print per-function coverage; no minimum percentage enforced |
| `make build` / `make vet` | Build packages / run Go's static analyzer |
| `make lint` | Run pinned [Staticcheck](https://staticcheck.dev/docs/getting-started/) |
| `make vuln` | Run pinned [govulncheck](https://go.dev/doc/security/vuln/), including test dependencies |
| `make help` | List targets |

Analyzer versions are pinned in Makefile and invoked with
`go run package@version`; their dependencies stay outside the application module.
Test dependencies remain in the module. Testify and analyzer choices are recorded
in [A13](../AMBIGUITIES.md#a13--testify-unit-test-assertions) and
[A12](../AMBIGUITIES.md#a12--lightweight-development-checks).

Formatting targets use `gofmt` from the toolchain selected by `GO` and
`GOTOOLCHAIN`, resolved through `go env GOROOT`. Setting `GOTOOLCHAIN` alone
does not change the standalone `gofmt` on `PATH`. Override `GOFMT` explicitly
only when a different formatter is intended.

## Continuous integration

The [Go workflow](../.github/workflows/go.yml) runs one Linux job on current
stable Go for pushes, pull requests, manual dispatch, and weekly Monday checks.
It calls `make ci`, caches Go modules/builds using `go.mod` and `go.sum`, and uses
read-only repository permissions. The two official GitHub Actions are pinned to
release commit SHAs; weekly [Dependabot](../.github/dependabot.yml) updates
maintain those pins. Analyzer pins are updated explicitly in Makefile.

## Compatibility and troubleshooting

- **Older Go versions:** Go 1.23.1 supports the normal tests. Use current stable
  Go for full checks, whose analyzers require at least Go 1.26. Prior automation
  checks used `GOTOOLCHAIN=go1.27.3`; see [WORKLOG](../WORKLOG.md) for results.
- **Download or metadata failures:** Check access to the Go module proxy and
  vulnerability database. A cached analyzer can still need a metadata lookup.
- **Toolchain or cache access errors:** Inspect `go env GOROOT GOCACHE` and the
  installation/cache permissions. Restricted environments must permit the
  toolchain and build-cache access needed by Go.
- **Race-test failures to start:** Check that a C compiler is available and
  CGO is enabled; inspect `go env CGO_ENABLED CC`.
- **Formatting failures:** `make fmt-check` lists unformatted files without
  modifying them. Run `make fmt` and repeat the check.
- **Vulnerability findings:** The scanner includes the selected compiler's
  standard library and test dependencies. Upgrade the toolchain or affected
  dependency indicated by the finding.

## Replay and isolated failing test

Run the canonical stream in fresh in-memory storage:

```sh
go run ./cmd/replay
```

The executable initializes the two accounts and explicit currency policies,
replays E1–E10 in supplied order, finalizes Day 6, and prints one table labeled
restated after the entire stream and finalization. Each day lists ACC-001 then
ACC-002, with two and three decimal places respectively. Repeated fresh runs
produce identical output. Initialization or technical failures exit nonzero;
recorded business rejections remain in the report and allow replay to continue.

- **Closing ledger / daily interest:** Final value-dated balances and corrected,
  individually rounded daily accruals. Day 6 balances include capitalization;
  its interest excludes that credit. Daily interest is informational, already
  paid by the final capitalization. Expected balances are in
  [NUMBERS](../NUMBERS.md#expected-replay-results).
- **Fee history:** Signed assessments, refunds, and adjustments grouped by the
  assessed/value day. `key` identifies the entry, `obligation` links its account/day
  assessment, `original` links a correction to its first assessment, and `cause`
  identifies the triggering input or close. `booked` and `value` show both dates.
  Days 2/4/5 retain charges booked Day 5 and refunds booked Day 6 despite final
  net fees of zero.
- **Recorded authorizations:** Lifecycle through each row's booking day. `ACTIVE`
  means a recorded approval with the displayed hold; settled holds are released.
  Auth-B stays declined on Days 5/6 even though the restated finances are positive.
- **Errors/decision reasons:** Original event ID, recorded outcome, diagnostic
  code, and reason on the observation day only: E6 on Day 4 and E8 on Day 5.
  A dash denotes no applicable record. There is no original-snapshot table.

The canonical runner and report are exercised by `go test ./tests/replay`, with
unit tests under `go test ./internal/scenario` and `go test ./internal/report`;
`go test ./cmd/replay` checks executable composition. Composition is in
[DESIGN](DESIGN.md#packages-and-dependencies); report semantics are owned by
[SPEC](SPEC.md#reporting-and-observations).

Run the required failing design test separately:

```sh
go test -tags designlimitation ./tests/replay/...
```

The tagged [integration test](../tests/replay/designlimitation_test.go),
`TestDesignLimitationFailedSubmissionIsFinanciallyAtomic`, exits nonzero
at its annotated equality assertion: expected zero minor units, actual −2,501.
It interrupts the Day 3 fee append after a backdated debit and Day 2 fee commit.
The test deliberately fails at the post-failure assertion to expose the single-call
submission atomicity boundary. It submits once and queries immediately after the
technical error, before recovery, testing whether the prior financial balance is
unchanged. The normal
[interruption regressions](../internal/ledger/backdating_test.go) separately verify
that explicit reconciliation completes the missing work idempotently. The probe
does not require financial effects from duplicate inputs or introduce a business
policy. The atomicity boundary, trade-off, and recovery rule are owned by
[DESIGN](DESIGN.md#repository-boundary), with fixture derivations in
[NUMBERS](../NUMBERS.md#interrupted-reconciliation-probe).

To run only that probe:

```sh
go test -tags designlimitation ./tests/replay -run '^TestDesignLimitationFailedSubmissionIsFinanciallyAtomic$' -count=1
```

The expected nonzero result must include that assertion failure; a compilation,
dependency, or cache-access error does not demonstrate the limitation. Normal
Make/CI targets omit the `designlimitation` tag and keep the passing regression
suite separate.

Currency and money API details are in the [currency](../internal/domain/currency.go)
and [money](../internal/domain/money.go) code.
