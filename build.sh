#!/usr/bin/env bash

set -euo pipefail

project_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
cd "$project_root"

# Build the settings page before embedding assets in release binaries.
HUSKY=0 pnpm --dir webs install --frozen-lockfile
pnpm --dir webs build
test -s webs/dist/index.html
# Embed only this build's assets, excluding obsolete bundles from older releases.
rm -rf -- "$project_root/static"
cp -R webs/dist "$project_root/static"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o sublink_amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-w -s" -o sublink_arm64 .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o sublink.exe .
