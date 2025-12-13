# Task completion checklist

Before handing off a change:

- Run formatting: `make format`
- Run lint: `make lint`
- Run unit tests: `make test`
- If protobuf/API changes: `make proto-all`
- If sidecar/consensus integration changes: consider running relevant integration tests (`make test-integration`, `make test-petri-integ`).
- Ensure new behavior is deterministic for consensus-critical code paths (ABCI vote extensions / preblock).
- Update docs (`PRD.md` / `PLAN.md`) if the implemented behavior changes requirements or rollout steps.
