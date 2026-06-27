#!/usr/bin/env bash
#
# Build LANLink desktop release binaries.
#
# Produces pure-Go (CGO disabled) cross-platform executables under ./release:
#   - lanlink[.exe]                  the terminal build: receive/send/pair
#                                    entirely in the terminal (no web UI), so it
#                                    runs directly as ./lanlink receive. arm64
#                                    builds get an -arm64 suffix.
#   - lanlink-<os>-<arch>[.exe]      the web build: runs a receiver AND serves
#                                    the dashboard, opening the browser on start
#
# The terminal build (./cmd/lanlink) has zero dependency on agent-web; the web
# build (./agent) embeds the dashboard and carries the app icon on Windows.
#
# Usage:
#   scripts/build-release.sh            # linux/amd64 + windows/amd64
#   LANLINK_ARM64=1 scripts/build-release.sh   # also build arm64 targets
#   LANLINK_VERSION=0.5.0 scripts/build-release.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

OUT="$ROOT/release"
mkdir -p "$OUT"

VERSION="${LANLINK_VERSION:-0.5.0-dev}"
export CGO_ENABLED=0

build() {
  local goos="$1" goarch="$2" ext="$3"

  # Terminal build: name it so a downloaded binary runs as ./lanlink directly.
  # amd64 (the primary target) gets the bare name; other arches are suffixed so
  # they don't collide (e.g. lanlink + lanlink-arm64).
  local term="lanlink"
  [ "$goarch" = "amd64" ] || term="lanlink-$goarch"
  echo "  lanlink (terminal) $goos/$goarch -> $term$ext"
  GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags "-s -w" \
    -o "$OUT/$term$ext" ./cmd/lanlink

  # Web/dashboard build keeps the descriptive <os>-<arch> name.
  echo "  lanlink (web)      $goos/$goarch -> lanlink-$goos-$goarch$ext"
  GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags "-s -w" \
    -o "$OUT/lanlink-$goos-$goarch$ext" ./agent
}

echo "Building LANLink $VERSION release binaries…"

build linux   amd64 ""
build windows amd64 ".exe"

if [ "${LANLINK_ARM64:-0}" = "1" ]; then
  build linux   arm64 ""
  build windows arm64 ".exe"
fi

echo
echo "Release artifacts in $OUT:"
ls -1sh "$OUT"
