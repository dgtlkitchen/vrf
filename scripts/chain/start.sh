#!/bin/sh
set -eu
(set -o pipefail) 2>/dev/null || true

BINARY=${BINARY:-chaind}
MIN_GAS_PRICES=${MIN_GAS_PRICES:-0uchain}
CHAIN_DIR=${CHAIN_DIR:-"$HOME/.chaind"}
TRACE=${TRACE:-true}

if [ ! -f "$CHAIN_DIR/config/genesis.json" ]; then
    echo "genesis not found at $CHAIN_DIR; run scripts/chain/init.sh first" >&2
    exit 1
fi

if [ "$TRACE" = "true" ]; then
    exec "$BINARY" start --home "$CHAIN_DIR" --minimum-gas-prices "$MIN_GAS_PRICES" --trace
fi

exec "$BINARY" start --home "$CHAIN_DIR" --minimum-gas-prices "$MIN_GAS_PRICES"
