# In-Memory Account Ledger Core

A Go ledger core with append-only history and exact monetary arithmetic. It
handles value-dated balances, authorization holds, settlements, backdated
corrections, overdraft fees, instalments, and daily interest.

The supplied six-day AED/BHD replay, daily report, and normal regression suite
are implemented; see [delivery status](docs/IMPLEMENTATION_PLAN.md). Storage is
in memory only. No web layer, database, services, or configuration are needed.

## Requirements

Go 1.23.1 or newer, plus Make for the shortcuts below. Tests download their
dependencies on first use. See [Development](docs/DEVELOPMENT.md#requirements)
for full-check requirements.

## Quick start

From the repository root:

```sh
go run ./cmd/replay
go test ./...
go vet ./...
```

Replay prints one row per day and account: closing ledger balance, daily
interest, fee history, recorded authorization states, and errors/decision reasons.
Balances and interest reflect all backdated corrections; Day 6 balances include
capitalized interest. Fee charges and refunds remain visible, and past
authorization decisions stay unchanged. Business rejections appear in the report;
technical failures exit nonzero. See the [output guide](docs/DEVELOPMENT.md#replay-and-isolated-failing-test)
and [expected balances](NUMBERS.md#expected-replay-results).
Tests print `ok` on success; vet is silent when it finds no issues.

## Intentional failing test

Run separately from the normal suite:

```sh
go test -tags designlimitation ./tests/replay/...
```

This must fail at the annotated balance assertion: expected AED 0.00, actual
AED −25.01. It exposes a limitation: a submission interrupted during fee
reconciliation can leave committed financial changes. Commits are atomic per
batch; recovery requires explicit reconciliation. See the
[test and recovery explanation](docs/DEVELOPMENT.md#replay-and-isolated-failing-test).
Normal tests and Make/CI targets exclude this probe.

## Main Make targets

| Command | Purpose |
| --- | --- |
| `make test` | Run normal tests |
| `make check` (default) | Check formatting, build, vet, lint, and test |
| `make ci` | Run full checks, including race tests, coverage, and vulnerabilities |
| `make fmt` | Format Go files |
| `make coverage` | Run race tests and print per-function coverage |
| `make help` | List all targets |

## Documentation

- **Development:** [Setup, tooling, and CI](docs/DEVELOPMENT.md),
  [Architecture](docs/DESIGN.md), [Money API](internal/domain/money.go),
  [Implementation plan](docs/IMPLEMENTATION_PLAN.md),
  [Repository guidelines](AGENTS.md).
- **Behavior:** [Assignment](docs/TASK.md), [Specification](docs/SPEC.md),
  [Use cases](docs/USE_CASES.md).
- **Decision records:** [Ambiguities and resolutions](AMBIGUITIES.md),
  [Numerical inventory](NUMBERS.md), [Acceptance-criterion review](REJECTED.md),
  [Worklog](WORKLOG.md).
