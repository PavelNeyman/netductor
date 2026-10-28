#!/usr/bin/env bash
set -euo pipefail
# Local release helper.
# Usage: GITHUB_TOKEN=ghp_… ./scripts/release.sh
# Requires: clean git tree; VERSION already matches internal/version.Release.
# Operator is macOS-only (darwin-arm64 / darwin-amd64).
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
if [ "$(tr -d ' \n' < VERSION)" != "$VER" ]; then
  echo "FAIL: VERSION file != $VER" >&2
  exit 1
fi
rel=$(grep -E '(var|const) Release[[:space:]]*=' internal/version/version.go | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
if [ "$rel" != "$VER" ]; then
  echo "FAIL: internal/version.Release ($rel) != $VER — bump and commit first" >&2
  exit 1
fi
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

for arch in amd64 arm64; do
  CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags="$LD" \
    -o "dist/netductor-linux-$arch" ./cmd/netductor
done
for arch in amd64 arm64; do
  CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags="$LD" \
    -o "dist/netductor-tg-linux-$arch" ./cmd/netductor-tg
done
for arch in amd64 arm64 arm mipsle riscv64; do
  extra=""
  case $arch in mipsle) extra="GOMIPS=softfloat";; arm) extra="GOARM=7";; esac
  env $extra CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags="$LD" \
    -o "dist/netductor-agent-linux-$arch" ./cmd/netductor-agent
done
# Operator — macOS only (explicit GOOS/GOARCH)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="$LD" \
  -o dist/netductor-op-darwin-arm64 ./cmd/netductor-op
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="$LD" \
  -o dist/netductor-op-darwin-amd64 ./cmd/netductor-op

(cd dist && sha256sum netductor-* > SHA256SUMS)
./scripts/verify-release-binaries.sh

SHA_ARM=$(sha256sum dist/netductor-op-darwin-arm64 | awk '{print $1}')
SHA_AMD=$(sha256sum dist/netductor-op-darwin-amd64 | awk '{print $1}')
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
  exit 0
fi

API="https://api.github.com/repos/PavelNeyman/netductor"
AUTH=(-H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github+json")

resolve_tag_commit() {
  local tag="$1"
  local ref json sha typ
  if ! ref=$(curl -fsSL "${AUTH[@]}" "$API/git/ref/tags/$tag" 2>/dev/null); then
    echo ""
    return 0
  fi
  sha=$(printf '%s' "$ref" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d["object"]["sha"])')
  typ=$(printf '%s' "$ref" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d["object"]["type"])')
  if [ "$typ" = "tag" ]; then
    sha=$(curl -fsSL "${AUTH[@]}" "$API/git/tags/$sha" | python3 -c 'import json,sys; print(json.load(sys.stdin)["object"]["sha"])')
  fi
  printf '%s' "$sha"
}

REMOTE_TAG_SHA=$(resolve_tag_commit "$TAG" || true)
if [ -n "$REMOTE_TAG_SHA" ]; then
  if [ "$REMOTE_TAG_SHA" != "$HEAD_SHA" ]; then
    echo "FAIL: remote tag $TAG points to $REMOTE_TAG_SHA, HEAD is $HEAD_SHA" >&2
    exit 1
  fi
  echo "remote tag $TAG already at HEAD_SHA"
else
  if git rev-parse "$TAG" >/dev/null 2>&1; then
    existing=$(git rev-parse "$TAG^{commit}")
    if [ "$existing" != "$HEAD_SHA" ]; then
      echo "FAIL: local tag $TAG at $existing != HEAD $HEAD_SHA" >&2
      exit 1
    fi
  else
    git tag -a "$TAG" -m "release $TAG" "$HEAD_SHA"
  fi
fi

if curl -fsSL "${AUTH[@]}" "$API/releases/tags/$TAG" > /tmp/nd-rel.json 2>/dev/null; then
  echo "using existing GitHub release $TAG"
else
  curl -fsSL -X POST "${AUTH[@]}" "$API/releases" \
    -d "{\"tag_name\":\"$TAG\",\"name\":\"$TAG\",\"target_commitish\":\"$HEAD_SHA\",\"body\":\"commit $HEAD_SHA\\nOperator: macOS only (darwin-arm64, darwin-amd64).\"}" \
    > /tmp/nd-rel.json
fi

UP=$(python3 -c 'import json;print(json.load(open("/tmp/nd-rel.json")).get("upload_url","").split("{")[0])')
if [ -z "$UP" ]; then
  echo "FAIL: no upload_url for release $TAG" >&2
  cat /tmp/nd-rel.json >&2
  exit 1
fi

REQUIRED_REMOTE=(
  netductor-linux-amd64 netductor-linux-arm64
  netductor-tg-linux-amd64 netductor-tg-linux-arm64
  netductor-agent-linux-amd64 netductor-agent-linux-arm64
  netductor-agent-linux-arm netductor-agent-linux-mipsle netductor-agent-linux-riscv64
  netductor-op-darwin-arm64 netductor-op-darwin-amd64
  SHA256SUMS
)

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
    curl -fsSL "${AUTH[@]}" "$API/releases/tags/$TAG" > /tmp/nd-rel.json
    asset_id=$(python3 -c '
import json,sys
rel=json.load(open("/tmp/nd-rel.json"))
name=sys.argv[1]
for a in rel.get("assets",[]):
  if a.get("name")==name:
    print(a.get("id",""))
    break
' "$name")
    if [ -z "$asset_id" ]; then
      echo "FAIL: $name HTTP 422 but asset id not found" >&2
      upload_fail=1
      continue
    fi
    curl -fsSL "${AUTH[@]}" -H "Accept: application/octet-stream" \
      -o "/tmp/nd-asset-$name" "$API/releases/assets/$asset_id"
    remote_sha=$(sha256sum "/tmp/nd-asset-$name" | awk '{print $1}')
    rm -f "/tmp/nd-asset-$name"
    if [ "$remote_sha" != "$local_sha" ]; then
      echo "FAIL: $name already on release but SHA mismatch local=$local_sha remote=$remote_sha" >&2
      upload_fail=1
    else
      echo "$name already exists and matches SHA256"
    fi
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

curl -fsSL "${AUTH[@]}" "$API/releases/tags/$TAG" > /tmp/nd-rel.json
python3 - <<'PY'
import json, sys
rel = json.load(open("/tmp/nd-rel.json"))
assets = {a["name"]: a for a in rel.get("assets", [])}
required = [
  "netductor-linux-amd64", "netductor-linux-arm64",
  "netductor-tg-linux-amd64", "netductor-tg-linux-arm64",
  "netductor-agent-linux-amd64", "netductor-agent-linux-arm64",
  "netductor-agent-linux-arm", "netductor-agent-linux-mipsle", "netductor-agent-linux-riscv64",
  "netductor-op-darwin-arm64", "netductor-op-darwin-amd64",
  "SHA256SUMS",
]
missing = [n for n in required if n not in assets]
if missing:
  print("FAIL: remote release missing:", ", ".join(missing), file=sys.stderr)
  sys.exit(1)
open("/tmp/nd-asset-ids.json","w").write(json.dumps({n: assets[n]["id"] for n in required}))
print("remote asset names complete:", len(required))
PY

mkdir -p /tmp/nd-rel-dl
dl_fail=0
for name in "${REQUIRED_REMOTE[@]}"; do
  id=$(python3 -c 'import json,sys; print(json.load(open("/tmp/nd-asset-ids.json"))[sys.argv[1]])' "$name")
  curl -fsSL "${AUTH[@]}" -H "Accept: application/octet-stream" \
    -o "/tmp/nd-rel-dl/$name" "$API/releases/assets/$id"
  local_sha=$(sha256sum "dist/$name" | awk '{print $1}')
  remote_sha=$(sha256sum "/tmp/nd-rel-dl/$name" | awk '{print $1}')
  if [ "$local_sha" != "$remote_sha" ]; then
    echo "FAIL: post-upload SHA mismatch $name local=$local_sha remote=$remote_sha" >&2
    dl_fail=1
  else
    echo "OK  remote $name sha256 matches"
  fi
done
rm -rf /tmp/nd-rel-dl

if [ "$dl_fail" -ne 0 ]; then
  echo "FAIL: published assets do not match local dist/" >&2
  exit 1
fi

FINAL_SHA=$(resolve_tag_commit "$TAG")
if [ -z "$FINAL_SHA" ] || [ "$FINAL_SHA" != "$HEAD_SHA" ]; then
  echo "FAIL: after publish, tag $TAG is '${FINAL_SHA:-missing}' != HEAD $HEAD_SHA" >&2
  exit 1
fi

echo "==> uploaded and SHA-verified $TAG (commit $HEAD_SHA)"
echo "Checklist: VERSION + Release + full assets + remote SHA256 match + tag==HEAD"
