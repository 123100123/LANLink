#!/usr/bin/env bash
#
# Build LANLink desktop release binaries.
#
# Produces pure-Go (CGO disabled) cross-platform executables under ./release:
#   - lanlink-<os>-<arch>[.exe]        the terminal binary (receive/send/scan)
#   - lanlink-agent-<os>-<arch>[.exe]  the dashboard build (serves the web UI)
#
# The terminal `lanlink` binary has zero dependency on agent-web; the
# `lanlink-agent` binary embeds the dashboard. Both come from the same source.
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
  echo "  lanlink        $goos/$goarch"
  GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags "-s -w" \
    -o "$OUT/lanlink-$goos-$goarch$ext" ./cmd/lanlink
  echo "  lanlink-agent  $goos/$goarch"
  GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags "-s -w" \
    -o "$OUT/lanlink-agent-$goos-$goarch$ext" ./agent
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
