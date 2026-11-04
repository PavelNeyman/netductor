class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.662"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.662/netductor-op-darwin-arm64"
      sha256 "279b02f8ab800a8b5485771175c85ff6a39032ed6fd35efab4f785fcdaa04e15"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.662/netductor-op-darwin-amd64"
      sha256 "0d790f46351383d87025bc8fbfd8a2a3d883b608e23ed81303e19cb40819e535"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
