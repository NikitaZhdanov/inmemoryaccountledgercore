.DEFAULT_GOAL := check

GO ?= go
GOFMT ?= "$(shell $(GO) env GOROOT)/bin/gofmt"
STATICCHECK_VERSION := v0.8.2
GOVULNCHECK_VERSION := v1.9.0

.PHONY: help fmt fmt-check build test test-race coverage vet lint vuln check ci

help:
	@printf '%s\n' \
	  'make check      Format check, build, vet, lint, and normal tests (default)' \
	  'make ci         Format check, build, vet, lint, race/coverage tests, vulnerabilities' \
	  'make fmt        Format all Go files' \
	  'make fmt-check  Reject unformatted Go files without changing them' \
	  'make build      Build all packages' \
	  'make test       Run normal regression tests' \
	  'make test-race  Run uncached race tests and write coverage.out' \
	  'make coverage   Run race tests and print per-function coverage' \
	  'make vet        Run go vet' \
	  'make lint       Run pinned Staticcheck' \
	  'make vuln       Run pinned govulncheck (requires network)'

fmt:
	$(GOFMT) -w .

fmt-check:
	@unformatted="$$($(GOFMT) -l .)" || exit $$?; \
	if [ -n "$$unformatted" ]; then \
	  printf '%s\n' "$$unformatted" 'Run make fmt to format Go sources.'; \
	  exit 1; \
	fi

build:
	$(GO) build ./...

test:
	$(GO) test ./...

test-race:
	$(GO) test -race -count=1 -coverprofile=coverage.out ./...

coverage: test-race
	$(GO) tool cover -func=coverage.out

vet:
	$(GO) vet ./...

lint:
	$(GO) run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) ./...

vuln:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) -test ./...

check: fmt-check build vet lint test

ci: fmt-check build vet lint test-race vuln
