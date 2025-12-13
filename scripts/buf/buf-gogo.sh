#!/usr/bin/env sh
set -eo pipefail

go tool buf dep update
go tool buf generate --template ./proto/buf.gen.gogo.yaml

cp -r ./github.com/vexxvakan/vrf/* ./
rm -rf ./github.com
