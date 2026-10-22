#!/bin/sh
# Reproducible cloud Go setup. Does not edit source manifests or lockfiles.
set -eu
cd "$(dirname "$0")/.."
TOOLS=${DOTAMAN_TOOLS:-/workspace/toolchains}
mkdir -p "$TOOLS"
if [ ! -x "$TOOLS/go/bin/go" ] || [ "$("$TOOLS/go/bin/go" version)" != 'go version go1.27.3 linux/amd64' ]; then
 archive=$(mktemp /tmp/dotaman-go.XXXXXX.tar.gz)
 trap 'rm -f "$archive"' EXIT HUP INT TERM
 curl --proto '=https' --proto-redir '=https' -fL https://go.dev/dl/go1.27.3.linux-amd64.tar.gz -o "$archive"
 printf '%s  %s\n' '65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40' "$archive" | sha256sum -c -
 tar -xzf "$archive" -C "$TOOLS"
fi
export PATH="$TOOLS/go/bin:$PATH"
export GOPATH=${DOTAMAN_GOPATH:-/workspace/go}
export GOMODCACHE="$GOPATH/pkg/mod" GOCACHE="$GOPATH/cache"
export CGO_ENABLED=1 GOTOOLCHAIN=local
command -v cc >/dev/null
go mod download
go mod verify
mkdir -p bin
go build -tags sqlite_fts5 -trimpath -o bin/dotamanai-codex ./cmd/dotamanai-codex
