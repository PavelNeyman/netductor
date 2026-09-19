#!/usr/bin/env bash
# Build release-shaped artifacts locally (no GitHub required).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VER="${1:-$(tr -d '\n' < "$ROOT/VERSION")}"
OUT="${2:-$ROOT/dist}"
mkdir -p "$OUT"
export CGO_ENABLED=0
build() {
  local goos=$1 goarch=$2 out=$3 pkg=$4
  GOOS=$goos GOARCH=$goarch go build -trimpath -ldflags "-s -w -X main.version=${VER}" -o "$OUT/$out" "$pkg"
}
build linux amd64 netductor-linux-amd64 ./cmd/netductor
build linux arm64 netductor-linux-arm64 ./cmd/netductor
build linux amd64 netductor-tg-linux-amd64 ./cmd/netductor-tg
build linux arm64 netductor-tg-linux-arm64 ./cmd/netductor-tg
build linux amd64 netductor-agent-linux-amd64 ./cmd/netductor-agent
build linux arm64 netductor-agent-linux-arm64 ./cmd/netductor-agent || true
( cd "$OUT" && sha256sum * > SHA256SUMS )
echo "Artifacts in $OUT (version=$VER)"
