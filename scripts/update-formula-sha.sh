#!/usr/bin/env bash
# Update Formula/netductor-op.rb from GitHub release assets (or local dist/).
# Usage:
#   ./scripts/update-formula-sha.sh v0.9.309
#   ./scripts/update-formula-sha.sh v0.9.309 ./dist
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TAG="${1:?tag e.g. v0.9.309}"
VER="${TAG#v}"
DIST="${2:-}"
tmpdir=
cleanup() { [[ -n "${tmpdir:-}" && -d "$tmpdir" ]] && rm -rf "$tmpdir"; }
trap cleanup EXIT
if [[ -z "$DIST" ]]; then
  tmpdir=$(mktemp -d)
  DIST=$tmpdir
  for f in netductor-op-darwin-arm64 netductor-op-darwin-amd64 netductor-op-linux-amd64; do
    url="https://github.com/PavelNeyman/netductor/releases/download/${TAG}/${f}"
    echo "fetch $url"
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
      curl -fsSL -H "Authorization: token $GITHUB_TOKEN" -L "$url" -o "$DIST/$f"
    else
      curl -fsSL -L "$url" -o "$DIST/$f"
    fi
  done
fi
SHA_ARM=$(sha256sum "$DIST/netductor-op-darwin-arm64" | awk '{print $1}')
SHA_AMD=$(sha256sum "$DIST/netductor-op-darwin-amd64" | awk '{print $1}')
SHA_LIN=$(sha256sum "$DIST/netductor-op-linux-amd64" | awk '{print $1}')
cat > "$ROOT/Formula/netductor-op.rb" << FORM
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
echo "updated Formula/netductor-op.rb → $VER"
echo "  arm64 $SHA_ARM"
echo "  amd64 $SHA_AMD"
echo "  linux $SHA_LIN"
