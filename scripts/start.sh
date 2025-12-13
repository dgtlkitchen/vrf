#!/bin/sh
set -eu
(set -o pipefail) 2>/dev/null || true

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

TARGET=${1:-chain}
if [ "$#" -gt 0 ]; then
    shift
fi

case "$TARGET" in
chain)
    exec sh "$SCRIPT_DIR/chain/start.sh" "$@"
    ;;
sidecar)
    exec sh "$SCRIPT_DIR/sidecar/start.sh" "$@"
    ;;
all)
    sh "$SCRIPT_DIR/chain/start.sh" "$@" &
    CHAIN_PID=$!

    cleanup() {
        kill "$CHAIN_PID" >/dev/null 2>&1 || true
        wait "$CHAIN_PID" >/dev/null 2>&1 || true
    }
    trap cleanup INT TERM EXIT

    sh "$SCRIPT_DIR/sidecar/start.sh" "$@"
    ;;
help|-h|--help)
    echo "Usage: scripts/start.sh [chain|sidecar|all]"
    ;;
*)
    echo "Unknown start target: $TARGET" >&2
    exit 2
    ;;
esac
