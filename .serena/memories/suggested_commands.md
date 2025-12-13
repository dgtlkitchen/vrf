# Suggested commands

## Tooling

- Install toolchain (via mise): `mise install`

## Build

- Build everything: `make build`
- Install main binary (if applicable): `make install`

## Tests

- Unit tests: `make test`
- Integration tests: `make test-integration`
- Petri integration: `make test-petri-integ`
- Coverage: `make test-cover`

## Lint/format

- Lint: `make lint`
- Lint with fixes: `make lint-fix`
- Format: `make format`
- Markdown lint: `make lint-markdown`
- Vulnerability scan: `make govulncheck`

## Protobuf

- Proto format+lint+gen: `make proto-all`
- Proto checks only: `make proto-check`
- Proto generation only: `make proto-gen`

## Dev environment

- Start dev stack: `make start-all-dev`
- Stop dev stack: `make stop-all-dev`
- Start sidecar-only: `make start-sidecar-dev`
- Stop sidecar-only: `make stop-sidecar-dev`
