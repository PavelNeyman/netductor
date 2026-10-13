class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.376"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.376/netductor-op-darwin-arm64"
      sha256 "67539a56cbe09357531f94b7ca8b7a9a33a4b2458121c5f733dc1af575944244"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.376/netductor-op-darwin-amd64"
      sha256 "0763c5fbe73c6a1ce9577187a2ad5e4cc5e7dc57985ccc40da04b13b39a5ad00"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.376/netductor-op-linux-amd64"
      sha256 "d89d96efbca94de63cb2c6399135a5945887ee4ec06c66cd7c984e2ba1ed61e8"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
