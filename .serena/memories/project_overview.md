# Project overview

## Tech stack

- Language: Go (see `go.mod`; toolchain pinned via `.mise.toml`).
- Core deps: Cosmos SDK + CometBFT (ABCI++), drand v2, gRPC (+ grpc-gateway), Prometheus client.
- Protobuf: buf-based generation/linting (see `buf.yaml`, `scripts/buf/*`, `make proto-*`).

## High-level components

- **Sidecar (`sidecar`)**: runs alongside a validator; supervises and/or queries a local drand daemon and exposes a simple VRF/randomness API.
- **ABCI integration**: vote extensions + verification + finalize/preblock logic to incorporate randomness into consensus deterministically.
- **On-chain module (`x/vrf`)**: stores beacons/params and exposes gRPC queries (and related client wiring).

## Repo structure (selected)

- `cmd/sidecar/`: `sidecar` binary entrypoint.
- `sidecar/`: sidecar implementation (config, drand service, errors, types).
- `x/vrf/abci/`: ABCI++ vote extension + proposal/preblock plumbing.
- `x/vrf/`: Cosmos SDK module implementation.
- `x/vrf/config/`: Cosmos SDK app config schema.
- `proto/`: protobuf definitions.
