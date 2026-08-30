#!/bin/sh
set -eu

root=$(CDPATH= cd "$(dirname "$0")/.." && pwd)
out=${1:-"$root/dist"}

rm -rf "$out"
mkdir -p "$out/plugins/codex/bin" "$out/plugins/codex/libexec/x"
cp -R "$root/plugins/codex/." "$out/plugins/codex/"
cp "$root/bin/x" "$out/plugins/codex/bin/x"
cp "$root/bin/x.cmd" "$out/plugins/codex/bin/x.cmd"
chmod 0755 "$out/plugins/codex/bin/x"

build() {
    goos=$1
    goarch=$2
    target=$3
    name=$4

    mkdir -p "$out/plugins/codex/libexec/x/$target"
    (
        cd "$root"
        CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch \
            go build -trimpath -ldflags="-s -w" \
            -o "$out/plugins/codex/libexec/x/$target/$name" ./cmd/x
    )
}

build darwin arm64 darwin-arm64 x
build darwin amd64 darwin-amd64 x
build linux arm64 linux-arm64 x
build linux amd64 linux-amd64 x
build windows amd64 windows-amd64 x.exe
