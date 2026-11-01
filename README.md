# inmemoryaccountledgercore

P01 is implemented: exact AED/BHD money values and checked arithmetic in
[`internal/domain`](internal/domain/money.go), with adjacent unit tests. The ledger
and replay/report are still pending P02–P14.

- [Original assignment](docs/TASK.md)
- [Behavioral specification and output semantics](docs/SPEC.md)
- [Proposed Go technical design](docs/DESIGN.md)
- [Use cases and scenario coverage](docs/USE_CASES.md)
- [Ordered implementation tasks and completion checks](docs/IMPLEMENTATION_PLAN.md)
- [Decision rationale and unspecified extensions](AMBIGUITIES.md)
- [Numerical justifications and expected results](NUMBERS.md)
- [Acceptance-criterion review](REJECTED.md)
- [Actual milestones](WORKLOG.md)
- [Repository instructions](AGENTS.md)

Use Go 1.23.1 or newer; implementation was checked with `go1.23.1 darwin/amd64`.
The module baseline is `go 1.23.1`. The core uses only the Go standard library,
so its tests need no dependency installation, service, or configuration. From
the repository root, run:

```sh
go test ./internal/domain
go test ./...
go vet ./...
```

The first command runs the money unit tests; the second runs all currently
implemented tests. Successful tests print `ok`; `go vet` is silent on success.
The tests use literal expected amounts and exercise currency scales, exact
parsing/formatting, signed endpoints, cross-currency rejection, and overflow of
each arithmetic operation, including intermediate accumulation.

The [Makefile](Makefile) provides the same commands locally and in
[GitHub Actions](.github/workflows/go.yml). For the complete development checks,
use current stable Go, Make, and a C compiler with CGO enabled for race detection.
The pinned analyzers require Go 1.26 or newer; Go may download a newer toolchain
automatically when building them. Their versions are recorded in Makefile and
their dependencies stay outside the application module. Analyzer/toolchain downloads
and module metadata lookups need network access, including some cached invocations;
vulnerability scans also query the Go vulnerability database. Existing Go 1.23.1
remains sufficient for the ordinary core tests above. Local `make check` and
`make ci` were verified with `GOTOOLCHAIN=go1.27.3`.

| Command | Checks or output |
| --- | --- |
| `make` / `make check` | Formatting check, build, vet, Staticcheck lint, normal tests |
| `make ci` | Formatting check, build, vet, lint, uncached race tests/coverage, vulnerability scan |
| `make fmt` / `make fmt-check` | Format all Go files / reject unformatted files without changing them |
| `make test` / `make test-race` | Normal tests / uncached race tests writing ignored `coverage.out` |
| `make coverage` | Run race tests, then print per-function coverage; no arbitrary threshold |
| `make build` / `make vet` | Build packages / run Go's static analyzer |
| `make lint` | Run pinned [Staticcheck](https://staticcheck.dev/docs/getting-started/) |
| `make vuln` | Run pinned [govulncheck](https://go.dev/doc/security/vuln/) against the selected Go toolchain and dependencies |
| `make help` | List targets |

The workflow runs one Linux job on current stable Go for pushes, pull requests,
manual dispatch, and weekly Monday checks. It calls `make ci`, caches Go modules
and builds using `go.mod`, and uses read-only repository permissions. Weekly
[Dependabot](.github/dependabot.yml) updates maintain release-SHA pins for the
two official GitHub Actions; analyzer pins are updated explicitly in Makefile.
An old toolchain can produce real standard-library vulnerability findings; upgrade
the selected compiler rather than suppressing those findings. See
[A12](AMBIGUITIES.md#a12--lightweight-development-checks) for these technical choices.

`domain.ParseMoney` accepts ASCII fixed-point text with an optional sign. It
accepts redundant fractional zeros and rejects nonzero excess precision without
rounding. `NewMoney` takes signed `int64` minor units. `Format` prints the exact
currency scale. `Add`, `Sub`, `Neg`, and `Sum` check overflow; `errors.Is` can
distinguish input range/precision errors from derived arithmetic overflow. See
[A9](AMBIGUITIES.md#a9--currency-precision-and-instalments) for the API choices.

`go run ./cmd/replay` and `go test ./tests/replay` are future commands; those
packages do not exist yet. The future report semantics are in [SPEC](docs/SPEC.md#reporting-and-observations).
The required intentional failing test is deferred to P13, where a genuine design
limitation will be investigated. Its planned isolated command is
`go test -tags designlimitation ./...`; no tagged failing test exists yet, so
running that command now does not demonstrate the failing-test deliverable.
Normal Make/CI targets keep that build tag disabled when the test is added later.
