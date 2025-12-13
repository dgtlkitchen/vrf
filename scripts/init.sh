#!/bin/sh
set -eu
(set -o pipefail) 2>/dev/null || true

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

TARGET=${1:-all}
if [ "$#" -gt 0 ]; then
    shift
fi

case "$TARGET" in
chain)
    exec sh "$SCRIPT_DIR/chain/init.sh" "$@"
    ;;
sidecar)
    exec sh "$SCRIPT_DIR/sidecar/init.sh" "$@"
    ;;
all)
    sh "$SCRIPT_DIR/chain/init.sh" "$@"
    exec sh "$SCRIPT_DIR/sidecar/init.sh"
    ;;
help|-h|--help)
    echo "Usage: scripts/init.sh [chain|sidecar|all]"
    ;;
*)
    echo "Unknown init target: $TARGET" >&2
    exit 2
    ;;
esac
