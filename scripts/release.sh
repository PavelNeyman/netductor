#!/usr/bin/env bash
set -euo pipefail
# Local release helper.
# Usage: VERSION=0.9.594 GITHUB_TOKEN=ghp_… ./scripts/release.sh
# Requires: clean git tree, VERSION already matches internal/version.Release.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

./scripts/check-versions.sh
./scripts/check-web-ui.sh

VER="${VERSION:-$(tr -d '[:space:]' < VERSION)}"
VER="${VER#v}"
TAG="v${VER}"
if [[ ! "$VER" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "FAIL: invalid VERSION='$VER'" >&2
  exit 1
fi

# Pin must already match — do not silently rewrite sources mid-release.
if [ "$(tr -d ' \n' < VERSION)" != "$VER" ]; then
  echo "FAIL: VERSION file != $VER" >&2
  exit 1
fi
rel=$(grep -E '(var|const) Release[[:space:]]*=' internal/version/version.go | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
if [ "$rel" != "$VER" ]; then
  echo "FAIL: internal/version.Release ($rel) != $VER — bump and commit first" >&2
  exit 1
fi

# Build must come from a clean, committed tree (tag will point at HEAD).
if [ -n "$(git status --porcelain)" ]; then
  echo "FAIL: working tree is dirty — commit or stash before release" >&2
  git status --porcelain >&2
  exit 1
fi
HEAD_SHA=$(git rev-parse HEAD)
echo "==> release $TAG from commit $HEAD_SHA"

export GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}"
mkdir -p dist
rm -f dist/netductor-* dist/SHA256SUMS
LD="-s -w -X github.com/PavelNeyman/netductor/internal/version.Release=${VER}"
build() { CGO_ENABLED=0 go build -trimpath -ldflags="$LD" -o "dist/$1" "$2"; }

# Node — linux amd64+arm64
for arch in amd64 arm64; do
  CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags="$LD" \
    -o "dist/netductor-linux-$arch" ./cmd/netductor
done
# Telegram — linux amd64+arm64
for arch in amd64 arm64; do
  CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags="$LD" \
    -o "dist/netductor-tg-linux-$arch" ./cmd/netductor-tg
done
# Agent — multi-arch
for arch in amd64 arm64 arm mipsle riscv64; do
  extra=""
  case $arch in mipsle) extra="GOMIPS=softfloat";; arm) extra="GOARM=7";; esac
  env $extra CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags="$LD" \
    -o "dist/netductor-agent-linux-$arch" ./cmd/netductor-agent
done
# Operator
build netductor-op-linux-amd64 ./cmd/netductor-op
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="$LD" -o dist/netductor-op-darwin-arm64 ./cmd/netductor-op
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="$LD" -o dist/netductor-op-darwin-amd64 ./cmd/netductor-op

(cd dist && sha256sum netductor-* > SHA256SUMS)
./scripts/verify-release-binaries.sh

# Formula
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

echo "==> built assets in dist/ (commit $HEAD_SHA)"
ls -la dist/

if [ -z "${GITHUB_TOKEN:-}" ]; then
  echo "Set GITHUB_TOKEN to upload release assets (dist/ is ready)"
  echo "Checklist: VERSION + Release + full assets + SHA256SUMS verified"
  exit 0
fi

API="https://api.github.com/repos/PavelNeyman/netductor"
AUTH=(-H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github+json")

# Ensure tag points at this HEAD (create lightweight tag locally + push via release API target).
if git rev-parse "$TAG" >/dev/null 2>&1; then
  existing=$(git rev-parse "$TAG^{commit}")
  if [ "$existing" != "$HEAD_SHA" ]; then
    echo "FAIL: tag $TAG exists at $existing, HEAD is $HEAD_SHA" >&2
    exit 1
  fi
else
  git tag -a "$TAG" -m "release $TAG" "$HEAD_SHA"
fi

# Create or fetch release bound to HEAD_SHA
if ! curl -fsSL "${AUTH[@]}" "$API/releases/tags/$TAG" > /tmp/nd-rel.json 2>/dev/null; then
  curl -fsSL -X POST "${AUTH[@]}" "$API/releases" \
    -d "{\"tag_name\":\"$TAG\",\"name\":\"$TAG\",\"target_commitish\":\"$HEAD_SHA\",\"body\":\"commit $HEAD_SHA\"}" \
    > /tmp/nd-rel.json
fi
REL_SHA=$(python3 -c 'import json;print(json.load(open("/tmp/nd-rel.json")).get("target_commitish",""))' 2>/dev/null || true)
UP=$(python3 -c 'import json;print(json.load(open("/tmp/nd-rel.json")).get("upload_url","").split("{")[0])')
if [ -z "$UP" ]; then
  echo "FAIL: no upload_url for release $TAG" >&2
  cat /tmp/nd-rel.json >&2
  exit 1
fi

upload_fail=0
for f in dist/*; do
  name=$(basename "$f")
  local_sha=$(sha256sum "$f" | awk '{print $1}')
  code=$(curl -sS -o /tmp/nd-asset-up.json -w "%{http_code}" -X POST \
    "${AUTH[@]}" -H "Content-Type: application/octet-stream" \
    "${UP}?name=${name}" --data-binary @"$f" || echo 000)
  if [ "$code" = "201" ] || [ "$code" = "200" ]; then
    echo "$name uploaded ($code) sha256=$local_sha"
  elif [ "$code" = "422" ]; then
    # Existing asset: download and compare SHA-256
    DL=$(python3 - << PY
import json, urllib.request
rel=json.load(open("/tmp/nd-rel.json"))
# refresh release for assets list
print("")
PY
)
    # Re-fetch release JSON for assets
    curl -fsSL "${AUTH[@]}" "$API/releases/tags/$TAG" > /tmp/nd-rel.json
    asset_url=$(python3 -c '
import json,sys
rel=json.load(open("/tmp/nd-rel.json"))
name=sys.argv[1]
for a in rel.get("assets",[]):
  if a.get("name")==name:
    print(a.get("browser_download_url",""))
    break
' "$name")
    if [ -z "$asset_url" ]; then
      echo "FAIL: $name HTTP 422 but asset URL not found" >&2
      upload_fail=1
      continue
    fi
    curl -fsSL -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/octet-stream" \
      -o "/tmp/nd-asset-$name" "$asset_url" || {
      # browser_download_url may need asset API id
      asset_id=$(python3 -c '
import json,sys
rel=json.load(open("/tmp/nd-rel.json"))
name=sys.argv[1]
for a in rel.get("assets",[]):
  if a.get("name")==name:
    print(a.get("id",""))
    break
' "$name")
      curl -fsSL "${AUTH[@]}" -H "Accept: application/octet-stream" \
        -o "/tmp/nd-asset-$name" "$API/releases/assets/$asset_id"
    }
    remote_sha=$(sha256sum "/tmp/nd-asset-$name" | awk '{print $1}')
    if [ "$remote_sha" != "$local_sha" ]; then
      echo "FAIL: $name already on release but SHA mismatch local=$local_sha remote=$remote_sha" >&2
      upload_fail=1
    else
      echo "$name already exists and matches SHA256"
    fi
    rm -f "/tmp/nd-asset-$name"
  else
    echo "FAIL: $name upload HTTP $code" >&2
    head -c 400 /tmp/nd-asset-up.json 2>/dev/null; echo >&2
    upload_fail=1
  fi
done

if [ "$upload_fail" -ne 0 ]; then
  echo "FAIL: release $TAG upload incomplete" >&2
  exit 1
fi

# Confirm every required name is present on the remote release
curl -fsSL "${AUTH[@]}" "$API/releases/tags/$TAG" > /tmp/nd-rel.json
python3 - << PY
import json, sys
rel = json.load(open("/tmp/nd-rel.json"))
assets = {a["name"] for a in rel.get("assets", [])}
required = [
  "netductor-linux-amd64", "netductor-linux-arm64",
  "netductor-tg-linux-amd64", "netductor-tg-linux-arm64",
  "netductor-agent-linux-amd64", "netductor-agent-linux-arm64",
  "netductor-agent-linux-arm", "netductor-agent-linux-mipsle", "netductor-agent-linux-riscv64",
  "netductor-op-linux-amd64", "netductor-op-darwin-arm64", "netductor-op-darwin-amd64",
  "SHA256SUMS",
]
missing = [n for n in required if n not in assets]
if missing:
  print("FAIL: remote release missing:", ", ".join(missing), file=sys.stderr)
  sys.exit(1)
print("remote assets complete:", len(required))
PY

echo "==> uploaded and verified $TAG (commit $HEAD_SHA)"
echo "Checklist: VERSION + Release + full assets + SHA256SUMS + remote set OK"
