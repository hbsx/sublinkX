#!/usr/bin/env bash

set -euo pipefail

# Build the settings page before embedding assets in release binaries.
HUSKY=0 pnpm --dir webs install --frozen-lockfile
pnpm --dir webs build
cp -R webs/dist/. static/

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o sublink_amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-w -s" -o sublink_arm64 .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o sublink.exe .
