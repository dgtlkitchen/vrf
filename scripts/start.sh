#!/bin/sh

BINARY=${BINARY:-chaind}
$BINARY start --minimum-gas-prices 0uchain --trace