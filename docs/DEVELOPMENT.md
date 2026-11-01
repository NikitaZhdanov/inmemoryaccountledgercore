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
go test ./...
go vet ./...
```

The focused command runs the money unit tests; `go test ./...` runs all
implemented tests. Success prints `ok`; vet is silent on success. A failing
assertion or check returns a nonzero exit status. The money tests use literal
expected values for configurable precision, trailing zeros, signed endpoints,
code/scale compatibility, and overflow, including intermediate accumulation.

For the full local checks, using a current Go toolchain:

```sh
make ci
```

## Test organization

Place unit tests beside each package in `*_test.go`; ledger unit tests can use a
test-local repository double. Tests composing multiple real components belong in
the `tests/replay` external test package, wiring fresh components through the same
boundaries as main rather than importing the executable. Once those packages exist,
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

## Planned replay and isolated failing test

The replay runner and end-to-end test package remain planned. Their intended
composition is in [DESIGN](DESIGN.md#packages-and-dependencies), and the future report's
observation checkpoint and output semantics are in [SPEC](SPEC.md#reporting-and-observations).

The required failing design test remains pending selection under
[A11](../AMBIGUITIES.md#a11--required-failing-test). Its planned isolated command is:

```sh
go test -tags designlimitation ./...
```

No tagged failing test exists yet. Once added, it must execute the annotated
assertion and exit nonzero for the documented limitation. Normal Make/CI targets
keep that build tag disabled.

Place the selected test beside the affected package for a local limitation or in
`tests/replay` for a cross-component limitation. Selection and completion conditions
are in [P13](IMPLEMENTATION_PLAN.md#p13--investigate-and-preserve-a-genuine-failing-design-test).

Currency and money API details are in the [currency](../internal/domain/currency.go)
and [money](../internal/domain/money.go) code.
