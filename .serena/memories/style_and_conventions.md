# Style and conventions

## Task flow (Test driven development, follow extremely closely)

1. Research task information and requirements very in-depth. Use "go doc" actively.
2. Always implement unit tests FIRST.
3. Double-check implemented unit tests in great detail.
4. Create or update corresponding README.md carefully, keep it tight.
5. Implement thorough and highly detailed code to fulfill test requirements.
6. Run respective unit tests and make ALL of them pass without manipulating test files.
7. Finish task by confirming all created or touched unit tests are entirely green without any skips or disabled tests.

## General conventions

- Follow standard Go/Cosmos SDK conventions: small packages, explicit error handling, deterministic logic in consensus-critical paths.
- Avoid editing generated protobuf files (`*.pb.go`, `*.pulsar.go`, `*.pb.gw.go`).
- Use Cosmos SDK Collections for on chain state management
