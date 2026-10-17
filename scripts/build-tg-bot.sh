#!/usr/bin/env bash
# Build netductor-tg for Linux (amd64/arm64).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${ROOT}/dist"
SRC="${ROOT}/cmd/netductor-tg"
command -v go >/dev/null 2>&1 || { echo "Go required" >&2; exit 1; }
mkdir -p "${OUT}"
arches=("$@")
[[ ${#arches[@]} -eq 0 ]] && arches=(amd64 arm64)
for arch in "${arches[@]}"; do
  dest="${OUT}/netductor-tg-linux-${arch}"
  echo "Building linux/${arch} → ${dest}"
  CGO_ENABLED=0 GOOS=linux GOARCH="${arch}" go build -trimpath -ldflags='-s -w' -o "${dest}" "${SRC}"
done
echo "Done."
