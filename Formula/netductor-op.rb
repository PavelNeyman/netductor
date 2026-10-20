class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.442"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.442/netductor-op-darwin-arm64"
      sha256 "87dfc6fab269ad9ad232729acefe9ac2c441bce8f50fd25c47b112d6fea32c07"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.442/netductor-op-darwin-amd64"
      sha256 "08d9b867b0a745d8746779a7d3dfeeb8b70bc67a55e69ab306bf8fbcd7731fbb"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.442/netductor-op-linux-amd64"
      sha256 "23a410e340a374be561463e11aec9159bdf871db671545ff5d766d2e15e2d67d"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
