#!/bin/sh
set -eu
(set -o pipefail) 2>/dev/null || true

VRF_HOME=${VRF_HOME:-}
if [ -z "$VRF_HOME" ]; then
    if [ -n "${HOME:-}" ]; then
        VRF_HOME="$HOME/.vrf"
    else
        VRF_HOME="$(pwd)/.vrf"
    fi
fi

SIDECAR_LISTEN_ADDR=${SIDECAR_LISTEN_ADDR:-127.0.0.1:8090}
DRAND_DATA_DIR=${DRAND_DATA_DIR:-$VRF_HOME/drand}

if ! mkdir -p "$DRAND_DATA_DIR" 2>/dev/null; then
    echo "Failed to create DRAND_DATA_DIR=$DRAND_DATA_DIR" >&2
    echo "Set DRAND_DATA_DIR (and optionally VRF_HOME) to a writable location." >&2
    exit 1
fi

case "$SIDECAR_LISTEN_ADDR" in
unix://*)
    SOCK_PATH=${SIDECAR_LISTEN_ADDR#unix://}
    if ! mkdir -p "$(dirname "$SOCK_PATH")" 2>/dev/null; then
        echo "Failed to create socket directory for SIDECAR_LISTEN_ADDR=$SIDECAR_LISTEN_ADDR" >&2
        echo "Set SIDECAR_LISTEN_ADDR (or VRF_HOME) to a writable location." >&2
        exit 1
    fi
    ;;
esac

echo "✅ Sidecar directories ready (DRAND_DATA_DIR=$DRAND_DATA_DIR)"
