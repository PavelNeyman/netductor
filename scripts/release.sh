#!/usr/bin/env bash
set -euo pipefail
# Local release helper (GitHub Actions disabled for this repo).
# Usage: VERSION=0.9.106 GITHUB_TOKEN=ghp_… ./scripts/release.sh
# Preflight (web/version) — must run under set -e
./scripts/check-versions.sh
./scripts/check-web-ui.sh
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
VER="${VERSION:-$(cat VERSION | tr -d '[:space:]')}"
VER="${VER#v}"
TAG="v${VER}"
echo "==> release $TAG"
# pin version files (var or const Release)
echo "$VER" > VERSION
if grep -qE '(var|const) Release' internal/version/version.go 2>/dev/null; then
  sed -i -E "s/(var|const) Release = \"[^\"]*\"/\1 Release = \"$VER\"/" internal/version/version.go
fi
# Re-run preflight after pin
./scripts/check-versions.sh
export GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}"
mkdir -p dist
LD="-s -w -X github.com/PavelNeyman/netductor/internal/version.Release=${VER}"
build() { CGO_ENABLED=0 go build -trimpath -ldflags="$LD" -o "dist/$1" "$2"; }
build netductor-linux-amd64 ./cmd/netductor
build netductor-tg-linux-amd64 ./cmd/netductor-tg
build netductor-agent-linux-amd64 ./cmd/netductor-agent
for arch in arm64 arm mipsle riscv64; do
  extra=""
  case $arch in mipsle) extra="GOMIPS=softfloat";; arm) extra="GOARM=7";; esac
  env $extra CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags="$LD" \
    -o "dist/netductor-agent-linux-$arch" ./cmd/netductor-agent
done
build netductor-op-linux-amd64 ./cmd/netductor-op
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="$LD" -o dist/netductor-op-darwin-arm64 ./cmd/netductor-op
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="$LD" -o dist/netductor-op-darwin-amd64 ./cmd/netductor-op
(cd dist && sha256sum netductor-* > SHA256SUMS)
# Smoke: version string embedded via -X must match VER for amd64 node/tg/agent/op
for bin in netductor-linux-amd64 netductor-tg-linux-amd64 netductor-agent-linux-amd64 netductor-op-linux-amd64; do
  if [ -x "dist/$bin" ] && command -v strings >/dev/null; then
    if ! strings "dist/$bin" | grep -qF "$VER"; then
      echo "FAIL: $bin does not embed version $VER" >&2
      exit 1
    fi
  fi
done
echo "binary version smoke: OK ($VER)"
# Formula sha
SHA_ARM=$(sha256sum dist/netductor-op-darwin-arm64 | awk '{print $1}')
SHA_AMD=$(sha256sum dist/netductor-op-darwin-amd64 | awk '{print $1}')
SHA_LIN=$(sha256sum dist/netductor-op-linux-amd64 | awk '{print $1}')
rm -f Formula/netductor.rb
cat > Formula/netductor-op.rb << FORM
class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "$VER"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/$TAG/netductor-op-darwin-arm64"
      sha256 "$SHA_ARM"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/$TAG/netductor-op-darwin-amd64"
      sha256 "$SHA_AMD"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/$TAG/netductor-op-linux-amd64"
      sha256 "$SHA_LIN"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
FORM
echo "==> built assets in dist/"
ls -la dist/
if [ -n "${GITHUB_TOKEN:-}" ]; then
  API="https://api.github.com/repos/PavelNeyman/netductor"
  curl -fsSL -X POST -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github+json" \
    "$API/releases" -d "{\"tag_name\":\"$TAG\",\"name\":\"$TAG\",\"target_commitish\":\"main\"}" > /tmp/nd-rel.json || \
  curl -fsSL -H "Authorization: token $GITHUB_TOKEN" "$API/releases/tags/$TAG" > /tmp/nd-rel.json
  UP=$(python3 -c 'import json;print(json.load(open("/tmp/nd-rel.json")).get("upload_url","").split("{")[0])')
  for f in dist/*; do
    name=$(basename "$f")
    code=$(curl -sS -o /tmp/nd-asset-up.json -w "%{http_code}" -X POST \
      -H "Authorization: token $GITHUB_TOKEN" -H "Content-Type: application/octet-stream" \
      "${UP}?name=${name}" --data-binary @"$f" || echo 000)
    if [ "$code" = "201" ] || [ "$code" = "200" ]; then
      python3 -c 'import json;d=json.load(open("/tmp/nd-asset-up.json"));print(d.get("name"), d.get("state","ok"))' 2>/dev/null || echo "$name ok"
    elif [ "$code" = "422" ]; then
      echo "$name already exists (skip)"
    else
      echo "$name upload HTTP $code" >&2
      head -c 240 /tmp/nd-asset-up.json 2>/dev/null; echo >&2
    fi
  done
  echo "==> uploaded $TAG"
else
  echo "Set GITHUB_TOKEN to upload release assets"
fi
echo "Checklist: VERSION + internal/version.Release + all assets (node,tg,agent,op) + Formula"
