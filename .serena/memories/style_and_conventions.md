# Style and conventions

## Go formatting/imports

- Prefer `gofumpt` formatting.
- Imports are ordered via `gci` (see `.golangci.yml` formatter settings).
- Repo includes a `make format` target that runs `gofumpt`, `misspell`, and `goimports` across the tree (excluding generated files).

## Linting

- Lint via `golangci-lint` (see `.golangci.yml`).
- Lint is configured with a fairly strict set of linters (e.g. `err113`, `errorlint`, `revive`, `staticcheck`, `gosec`, etc.).

## Protobuf

- Protobuf formatting/linting/generation is done via `buf` inside a protobuilder Docker image (see `make proto-*` targets).

## General conventions

- Follow standard Go/Cosmos SDK conventions: small packages, explicit error handling, deterministic logic in consensus-critical paths.
- Avoid editing generated protobuf files (`*.pb.go`, `*.pulsar.go`, `*.pb.gw.go`).
