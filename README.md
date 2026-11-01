# In-Memory Account Ledger Core

An in-memory ledger for value-dated account balances, authorizations, fees, and
daily interest, using exact monetary arithmetic.

Configurable currencies, exact money values, checked arithmetic, daily interest
rounding, conserving instalment allocation, and append-only in-memory storage
are implemented. Ledger processing and replay reporting remain planned.

## Requirements

Go 1.23.1 or newer, plus Make for the shortcuts below. Tests download their
dependencies on first use. See [Development](docs/DEVELOPMENT.md#requirements)
for full-check requirements.

## Quick start

From the repository root:

```sh
go test ./...
go vet ./...
```

Tests print `ok` on success; vet is silent when it finds no issues.

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
